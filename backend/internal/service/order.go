package service

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/AboAuther/RGPerp/backend/internal/model"
	apperr "github.com/AboAuther/RGPerp/backend/internal/pkg/errors"
)

type OrderService struct {
	db *gorm.DB
}

func NewOrderService(db *gorm.DB) *OrderService {
	return &OrderService{db: db}
}

type CreateOrderInput struct {
	UserID        uint64
	ClientOrderID string
	Symbol        string
	Side          string
	Type          string
	Size          decimal.Decimal
	Leverage      uint32
	Margin        decimal.Decimal
	ReduceOnly    bool
}

type OrderExecutionOutput struct {
	Order    model.Order         `json:"order"`
	Position *PositionListItem   `json:"position,omitempty"`
	Account  AccountBalanceBrief `json:"account"`
}

type AccountBalanceBrief struct {
	AvailableBalance decimal.Decimal `json:"available_balance"`
	LockedBalance    decimal.Decimal `json:"locked_balance"`
}

type PositionListItem struct {
	ID               uint64          `json:"id"`
	Symbol           string          `json:"symbol"`
	Side             string          `json:"side"`
	Size             decimal.Decimal `json:"size"`
	EntryPrice       decimal.Decimal `json:"entry_price"`
	MarkPrice        decimal.Decimal `json:"mark_price"`
	LiquidationPrice decimal.Decimal `json:"liquidation_price"`
	Margin           decimal.Decimal `json:"margin"`
	Leverage         uint32          `json:"leverage"`
	UnrealizedPnL    decimal.Decimal `json:"unrealized_pnl"`
	RealizedPnL      decimal.Decimal `json:"realized_pnl"`
	Status           string          `json:"status"`
}

func (s *OrderService) Create(input CreateOrderInput) (*OrderExecutionOutput, error) {
	input.Symbol = strings.ToUpper(strings.TrimSpace(input.Symbol))
	input.Side = strings.ToLower(strings.TrimSpace(input.Side))
	input.Type = strings.ToLower(strings.TrimSpace(input.Type))

	if input.Type != "market" {
		return nil, apperr.ErrOrderRejected
	}
	if input.Side != "long" && input.Side != "short" {
		return nil, apperr.ErrOrderRejected
	}
	if input.Size.LessThanOrEqual(decimal.Zero) {
		return nil, apperr.ErrInvalidOrderSize
	}

	var symbol model.Symbol
	if err := s.db.Where("name = ? AND status = ?", input.Symbol, "trading").First(&symbol).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrSymbolNotFound
		}
		return nil, err
	}

	if input.Leverage == 0 || input.Leverage > symbol.MaxLeverage {
		return nil, apperr.ErrInvalidLeverage
	}
	if input.Size.LessThan(symbol.MinOrderSize) {
		return nil, apperr.ErrInvalidOrderSize
	}

	var tick model.PriceTick
	if err := s.db.Where("symbol = ?", input.Symbol).Order("created_at desc").First(&tick).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrPriceUnavailable
		}
		return nil, err
	}

	price := tick.MarkPrice
	notional := price.Mul(input.Size)
	if notional.GreaterThan(symbol.MaxPositionNotional) {
		return nil, apperr.ErrMaxPositionExceeded
	}

	requiredMargin := input.Margin
	if requiredMargin.LessThanOrEqual(decimal.Zero) {
		requiredMargin = notional.Div(decimal.NewFromInt(int64(input.Leverage)))
	}
	requiredMargin = requiredMargin.Round(18)
	fee := notional.Mul(symbol.TakerFeeRate).Round(18)

	if input.ClientOrderID != "" {
		var dup int64
		if err := s.db.Model(&model.Order{}).
			Where("user_id = ? AND client_order_id = ?", input.UserID, input.ClientOrderID).
			Count(&dup).Error; err != nil {
			return nil, err
		}
		if dup > 0 {
			return nil, apperr.ErrDuplicateClientOrder
		}
	}

	var response *OrderExecutionOutput
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var account model.Account
		if err := tx.Where("user_id = ?", input.UserID).First(&account).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperr.ErrAccountNotFound
			}
			return err
		}

		var existing model.Position
		existingErr := tx.Where("user_id = ? AND symbol = ? AND status = ?", input.UserID, input.Symbol, "open").
			First(&existing).Error
		hasPosition := existingErr == nil
		if existingErr != nil && existingErr != gorm.ErrRecordNotFound {
			return existingErr
		}

		order := model.Order{
			ClientOrderID: input.ClientOrderID,
			UserID:        input.UserID,
			Symbol:        input.Symbol,
			Side:          input.Side,
			Type:          input.Type,
			Size:          input.Size,
			Price:         price,
			Leverage:      input.Leverage,
			Margin:        requiredMargin,
			ReduceOnly:    input.ReduceOnly,
			Status:        "filled",
			FilledSize:    input.Size,
			ExecPrice:     price,
			Fee:           fee,
		}

		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		realizedPnL := decimal.Zero
		marginConsumed := decimal.Zero
		marginReleased := decimal.Zero
		var positionOut *PositionListItem

		if !hasPosition {
			if input.ReduceOnly {
				return apperr.ErrNoOpenPosition
			}
			totalNeed := requiredMargin.Add(fee)
			if account.AvailableBalance.LessThan(totalNeed) {
				return apperr.ErrInsufficientMargin
			}

			account.AvailableBalance = account.AvailableBalance.Sub(totalNeed)
			account.LockedBalance = account.LockedBalance.Add(requiredMargin)
			marginConsumed = requiredMargin

			position := model.Position{
				UserID:           input.UserID,
				Symbol:           input.Symbol,
				Side:             input.Side,
				Size:             input.Size,
				EntryPrice:       price,
				MarkPrice:        price,
				LiquidationPrice: calculateLiquidationPrice(input.Side, price, input.Size, requiredMargin, symbol.MaintenanceMarginRate),
				Margin:           requiredMargin,
				Leverage:         input.Leverage,
				Status:           "open",
			}
			if err := tx.Create(&position).Error; err != nil {
				return err
			}
			positionOut = buildPositionOutput(position, price)

			if err := tx.Create(&model.Trade{
				OrderID:     order.ID,
				UserID:      input.UserID,
				Symbol:      input.Symbol,
				Side:        input.Side,
				Size:        input.Size,
				Price:       price,
				Margin:      requiredMargin,
				Fee:         fee,
				RealizedPnL: realizedPnL,
			}).Error; err != nil {
				return err
			}
		} else {
			beforeAvailable := account.AvailableBalance
			beforeLocked := account.LockedBalance

			if existing.Side == input.Side && !input.ReduceOnly {
				totalNeed := requiredMargin.Add(fee)
				if account.AvailableBalance.LessThan(totalNeed) {
					return apperr.ErrInsufficientMargin
				}

				newSize := existing.Size.Add(input.Size)
				newMargin := existing.Margin.Add(requiredMargin)
				newEntry := existing.EntryPrice.Mul(existing.Size).Add(price.Mul(input.Size)).Div(newSize)

				account.AvailableBalance = account.AvailableBalance.Sub(totalNeed)
				account.LockedBalance = account.LockedBalance.Add(requiredMargin)
				marginConsumed = requiredMargin

				existing.Size = newSize
				existing.Margin = newMargin
				existing.EntryPrice = newEntry
				existing.MarkPrice = price
				existing.Leverage = input.Leverage
				existing.LiquidationPrice = calculateLiquidationPrice(existing.Side, existing.EntryPrice, existing.Size, existing.Margin, symbol.MaintenanceMarginRate)
				if err := tx.Save(&existing).Error; err != nil {
					return err
				}
			} else {
				closeSize := decimal.Min(existing.Size, input.Size)
				marginReleased = existing.Margin.Mul(closeSize).Div(existing.Size).Round(18)
				realizedPnL = calculatePnL(existing.Side, existing.EntryPrice, price, closeSize).Round(18)

				account.AvailableBalance = account.AvailableBalance.Add(marginReleased).Add(realizedPnL).Sub(fee)
				account.LockedBalance = account.LockedBalance.Sub(marginReleased)

				remainingSize := existing.Size.Sub(closeSize)
				remainingMargin := existing.Margin.Sub(marginReleased)

				if remainingSize.LessThanOrEqual(decimal.Zero) {
					existing.Size = decimal.Zero
					existing.Margin = decimal.Zero
					existing.MarkPrice = price
					existing.UnrealizedPnL = decimal.Zero
					existing.Status = "closed"
					existing.LiquidationPrice = decimal.Zero
					if err := tx.Save(&existing).Error; err != nil {
						return err
					}
				} else {
					existing.Size = remainingSize
					existing.Margin = remainingMargin
					existing.MarkPrice = price
					existing.LiquidationPrice = calculateLiquidationPrice(existing.Side, existing.EntryPrice, existing.Size, existing.Margin, symbol.MaintenanceMarginRate)
					if err := tx.Save(&existing).Error; err != nil {
						return err
					}
					positionOut = buildPositionOutput(existing, price)
				}

				flipSize := input.Size.Sub(closeSize)
				if flipSize.GreaterThan(decimal.Zero) {
					if input.ReduceOnly {
						return apperr.ErrOrderRejected
					}
					flipNotional := price.Mul(flipSize)
					flipMargin := flipNotional.Div(decimal.NewFromInt(int64(input.Leverage))).Round(18)
					totalNeed := flipMargin
					if account.AvailableBalance.LessThan(totalNeed) {
						return apperr.ErrInsufficientMargin
					}

					account.AvailableBalance = account.AvailableBalance.Sub(flipMargin)
					account.LockedBalance = account.LockedBalance.Add(flipMargin)
					marginConsumed = marginConsumed.Add(flipMargin)

					newPos := model.Position{
						UserID:           input.UserID,
						Symbol:           input.Symbol,
						Side:             input.Side,
						Size:             flipSize,
						EntryPrice:       price,
						MarkPrice:        price,
						LiquidationPrice: calculateLiquidationPrice(input.Side, price, flipSize, flipMargin, symbol.MaintenanceMarginRate),
						Margin:           flipMargin,
						Leverage:         input.Leverage,
						Status:           "open",
					}
					if err := tx.Create(&newPos).Error; err != nil {
						return err
					}
					positionOut = buildPositionOutput(newPos, price)
				}
			}

			order.RealizedPnL = realizedPnL
			if err := tx.Model(&model.Order{}).Where("id = ?", order.ID).
				Update("RealizedPnL", order.RealizedPnL).Error; err != nil {
				return err
			}

			if err := tx.Create(&model.Trade{
				OrderID:     order.ID,
				UserID:      input.UserID,
				Symbol:      input.Symbol,
				Side:        input.Side,
				Size:        input.Size,
				Price:       price,
				Margin:      marginConsumed,
				Fee:         fee,
				RealizedPnL: realizedPnL,
			}).Error; err != nil {
				return err
			}

			if marginConsumed.GreaterThan(decimal.Zero) {
				if err := tx.Create(&model.LedgerEntry{
					UserID:        input.UserID,
					Type:          "margin_lock",
					Amount:        marginConsumed,
					BalanceBefore: beforeAvailable,
					BalanceAfter:  account.AvailableBalance,
					ReferenceType: "order",
					ReferenceID:   order.ID,
					Description:   "margin locked for order",
				}).Error; err != nil {
					return err
				}
			}
			if marginReleased.GreaterThan(decimal.Zero) || !realizedPnL.Equal(decimal.Zero) {
				if err := tx.Create(&model.LedgerEntry{
					UserID:        input.UserID,
					Type:          "position_settle",
					Amount:        marginReleased.Add(realizedPnL),
					BalanceBefore: beforeAvailable,
					BalanceAfter:  account.AvailableBalance,
					ReferenceType: "order",
					ReferenceID:   order.ID,
					Description:   "margin released and pnl settled",
				}).Error; err != nil {
					return err
				}
			}
			_ = beforeLocked
		}

		beforeFeeBalance := account.AvailableBalance
		if !hasPosition {
			if err := tx.Create(&model.LedgerEntry{
				UserID:        input.UserID,
				Type:          "margin_lock",
				Amount:        marginConsumed,
				BalanceBefore: account.AvailableBalance.Add(fee).Add(marginConsumed),
				BalanceAfter:  account.AvailableBalance.Add(fee),
				ReferenceType: "order",
				ReferenceID:   order.ID,
				Description:   "margin locked for new position",
			}).Error; err != nil {
				return err
			}
		}

		if fee.GreaterThan(decimal.Zero) {
			if err := tx.Create(&model.LedgerEntry{
				UserID:        input.UserID,
				Type:          "trading_fee",
				Amount:        fee,
				BalanceBefore: beforeFeeBalance.Add(fee),
				BalanceAfter:  beforeFeeBalance,
				ReferenceType: "order",
				ReferenceID:   order.ID,
				Description:   fmt.Sprintf("taker fee for %s", input.Symbol),
			}).Error; err != nil {
				return err
			}
		}

		if err := tx.Save(&account).Error; err != nil {
			return err
		}

		response = &OrderExecutionOutput{
			Order:    order,
			Position: positionOut,
			Account: AccountBalanceBrief{
				AvailableBalance: account.AvailableBalance,
				LockedBalance:    account.LockedBalance,
			},
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *OrderService) ListOpenPositions(userID uint64) ([]PositionListItem, error) {
	var positions []model.Position
	if err := s.db.Where("user_id = ? AND status = ?", userID, "open").Order("updated_at desc").Find(&positions).Error; err != nil {
		return nil, err
	}

	items := make([]PositionListItem, 0, len(positions))
	for _, pos := range positions {
		items = append(items, buildPositionOutputValue(pos, pos.MarkPrice))
	}
	return items, nil
}

func buildPositionOutput(pos model.Position, markPrice decimal.Decimal) *PositionListItem {
	item := buildPositionOutputValue(pos, markPrice)
	return &item
}

func buildPositionOutputValue(pos model.Position, markPrice decimal.Decimal) PositionListItem {
	unrealized := calculatePnL(pos.Side, pos.EntryPrice, markPrice, pos.Size).Round(18)
	return PositionListItem{
		ID:               pos.ID,
		Symbol:           pos.Symbol,
		Side:             pos.Side,
		Size:             pos.Size,
		EntryPrice:       pos.EntryPrice,
		MarkPrice:        markPrice,
		LiquidationPrice: pos.LiquidationPrice,
		Margin:           pos.Margin,
		Leverage:         pos.Leverage,
		UnrealizedPnL:    unrealized,
		RealizedPnL:      pos.RealizedPnL,
		Status:           pos.Status,
	}
}

func calculatePnL(side string, entryPrice, exitPrice, size decimal.Decimal) decimal.Decimal {
	switch side {
	case "long":
		return exitPrice.Sub(entryPrice).Mul(size)
	case "short":
		return entryPrice.Sub(exitPrice).Mul(size)
	default:
		return decimal.Zero
	}
}

func calculateLiquidationPrice(side string, entryPrice, size, margin, maintenanceRate decimal.Decimal) decimal.Decimal {
	if size.LessThanOrEqual(decimal.Zero) || entryPrice.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}

	switch side {
	case "long":
		denominator := size.Mul(decimal.NewFromInt(1).Sub(maintenanceRate))
		if denominator.IsZero() {
			return decimal.Zero
		}
		return entryPrice.Mul(size).Sub(margin).Div(denominator).Round(18)
	case "short":
		denominator := size.Mul(decimal.NewFromInt(1).Add(maintenanceRate))
		if denominator.IsZero() {
			return decimal.Zero
		}
		return entryPrice.Mul(size).Add(margin).Div(denominator).Round(18)
	default:
		return decimal.Zero
	}
}

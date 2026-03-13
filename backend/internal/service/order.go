package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AboAuther/RGPerp/backend/internal/model"
	apperr "github.com/AboAuther/RGPerp/backend/internal/pkg/errors"
)

const priceMaxAge = 60 * time.Second
const testLeverageLimit uint32 = 1000

type OrderService struct {
	db *gorm.DB
}

func NewOrderService(db *gorm.DB) *OrderService {
	return &OrderService{db: db}
}

type CreateOrderInput struct {
	UserID          uint64
	ClientOrderID   string
	Symbol          string
	Side            string
	Type            string
	MarginMode      string
	Size            decimal.Decimal
	LimitPrice      decimal.Decimal
	TimeInForce     string
	Leverage        uint32
	Margin          decimal.Decimal
	ReduceOnly      bool
	TestMode        bool
	ParentOrderID   *uint64
	ExecutionSource string
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

type limitReservation struct {
	Margin decimal.Decimal
	Fee    decimal.Decimal
}

func hasReservation(res limitReservation) bool {
	return res.Margin.GreaterThan(decimal.Zero) || res.Fee.GreaterThan(decimal.Zero)
}

func settleReservation(account *model.Account, reserved limitReservation, actualMargin, actualFee decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	if !hasReservation(reserved) {
		total := actualMargin.Add(actualFee)
		account.AvailableBalance = account.AvailableBalance.Sub(total)
		account.LockedBalance = account.LockedBalance.Add(actualMargin)
		return total, actualMargin
	}

	account.LockedBalance = account.LockedBalance.Sub(reserved.Margin)
	if actualMargin.GreaterThan(decimal.Zero) {
		account.LockedBalance = account.LockedBalance.Add(actualMargin)
	}

	totalReserved := reserved.Margin.Add(reserved.Fee)
	totalActual := actualMargin.Add(actualFee)
	delta := totalActual.Sub(totalReserved)
	account.AvailableBalance = account.AvailableBalance.Sub(delta)
	return delta, actualMargin.Sub(reserved.Margin)
}

type PositionListItem struct {
	ID                uint64          `json:"id"`
	Symbol            string          `json:"symbol"`
	Side              string          `json:"side"`
	MarginMode        string          `json:"margin_mode"`
	Size              decimal.Decimal `json:"size"`
	EntryPrice        decimal.Decimal `json:"entry_price"`
	MarkPrice         decimal.Decimal `json:"mark_price"`
	LiquidationPrice  decimal.Decimal `json:"liquidation_price"`
	Margin            decimal.Decimal `json:"margin"`
	Notional          decimal.Decimal `json:"notional"`
	MaintenanceMargin decimal.Decimal `json:"maintenance_margin"`
	RiskRatio         decimal.Decimal `json:"risk_ratio"`
	Leverage          uint32          `json:"leverage"`
	UnrealizedPnL     decimal.Decimal `json:"unrealized_pnl"`
	RealizedPnL       decimal.Decimal `json:"realized_pnl"`
	Status            string          `json:"status"`
}

type OrderListItem struct {
	ID            uint64          `json:"id"`
	ClientOrderID string          `json:"client_order_id"`
	Symbol        string          `json:"symbol"`
	Side          string          `json:"side"`
	Type          string          `json:"type"`
	MarginMode    string          `json:"margin_mode"`
	Size          decimal.Decimal `json:"size"`
	LimitPrice    decimal.Decimal `json:"limit_price"`
	TimeInForce   string          `json:"time_in_force"`
	ExecPrice     decimal.Decimal `json:"exec_price"`
	Leverage      uint32          `json:"leverage"`
	Margin        decimal.Decimal `json:"margin"`
	ReduceOnly    bool            `json:"reduce_only"`
	Status        string          `json:"status"`
	Fee           decimal.Decimal `json:"fee"`
	RealizedPnL   decimal.Decimal `json:"realized_pnl"`
	CreatedAt     string          `json:"created_at"`
	TriggeredAt   string          `json:"triggered_at,omitempty"`
	CancelReason  string          `json:"cancel_reason,omitempty"`
}

type TradeListItem struct {
	ID            uint64          `json:"id"`
	OrderID       uint64          `json:"order_id"`
	Symbol        string          `json:"symbol"`
	Side          string          `json:"side"`
	Size          decimal.Decimal `json:"size"`
	Price         decimal.Decimal `json:"price"`
	Fee           decimal.Decimal `json:"fee"`
	RealizedPnL   decimal.Decimal `json:"realized_pnl"`
	IsLiquidation bool            `json:"is_liquidation"`
	CreatedAt     string          `json:"created_at"`
}

func (s *OrderService) Create(input CreateOrderInput) (*OrderExecutionOutput, error) {
	input.Symbol = strings.ToUpper(strings.TrimSpace(input.Symbol))
	input.Side = strings.ToLower(strings.TrimSpace(input.Side))
	input.Type = strings.ToLower(strings.TrimSpace(input.Type))
	input.MarginMode = normalizeMarginMode(input.MarginMode)
	input.TimeInForce = normalizeTimeInForce(input.TimeInForce)
	if input.ExecutionSource == "" {
		input.ExecutionSource = "direct"
	}

	if input.Type != "market" && input.Type != "limit" {
		return nil, apperr.ErrOrderRejected
	}
	if input.Side != "long" && input.Side != "short" {
		return nil, apperr.ErrOrderRejected
	}
	if input.MarginMode == "" {
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

	maxLeverage := symbol.MaxLeverage
	if input.TestMode && testLeverageLimit > maxLeverage {
		maxLeverage = testLeverageLimit
	}
	if input.Leverage == 0 || input.Leverage > maxLeverage {
		return nil, apperr.ErrInvalidLeverage
	}
	if input.Size.LessThan(symbol.MinOrderSize) {
		return nil, apperr.ErrInvalidOrderSize
	}
	if input.Type == "limit" {
		if input.LimitPrice.LessThanOrEqual(decimal.Zero) {
			return nil, apperr.ErrBadRequest
		}
		if input.TimeInForce != "gtc" {
			return nil, apperr.ErrBadRequest
		}
	}

	var tick model.PriceTick
	if err := s.db.Where("symbol = ?", input.Symbol).Order("created_at desc").First(&tick).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrPriceUnavailable
		}
		return nil, err
	}
	if time.Since(tick.CreatedAt) > priceMaxAge {
		return nil, apperr.ErrPriceStale
	}

	price := tick.MarkPrice
	if input.Type == "limit" {
		price = input.LimitPrice
	}
	notional := price.Mul(input.Size)
	minRequiredMargin := minimumRequiredMargin(notional, input.Leverage)
	requiredMargin := input.Margin
	if requiredMargin.LessThanOrEqual(decimal.Zero) {
		requiredMargin = minRequiredMargin
	} else if requiredMargin.LessThan(minRequiredMargin) {
		return nil, apperr.ErrInsufficientMargin
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
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", input.UserID).First(&account).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperr.ErrAccountNotFound
			}
			return err
		}

		var openPositions []model.Position
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND symbol = ? AND status = ?", input.UserID, input.Symbol, "open").
			Find(&openPositions).Error; err != nil {
			return err
		}

		var sameSidePosition *model.Position
		var oppositeSidePosition *model.Position
		for i := range openPositions {
			pos := &openPositions[i]
			if pos.Side == input.Side {
				sameSidePosition = pos
				continue
			}
			oppositeSidePosition = pos
		}

		riskState, err := syncUserRiskStatusTx(tx, input.UserID)
		if err != nil {
			return err
		}
		if riskState.RiskLevel == "frozen" {
			return apperr.ErrAccountFrozen
		}
		if riskState.RiskLevel == "liquidating" {
			return apperr.ErrAccountLiquidating
		}
		var exposurePosition *model.Position
		if !input.ReduceOnly {
			exposurePosition = sameSidePosition
		} else {
			exposurePosition = oppositeSidePosition
		}
		if riskState.RiskLevel == "reduce_only" && orderIncreasesExposure(exposurePosition, input.Side, input.Size, input.ReduceOnly) {
			return apperr.ErrReduceOnlyMode
		}

		realizedPnL := decimal.Zero
		marginConsumed := decimal.Zero
		marginReleased := decimal.Zero
		var positionOut *PositionListItem
		activeMarginMode := input.MarginMode
		if input.ReduceOnly {
			if oppositeSidePosition == nil {
				return apperr.ErrNoOpenPosition
			}
			activeMarginMode = oppositeSidePosition.MarginMode
			if oppositeSidePosition.MarginMode != input.MarginMode {
				return apperr.ErrOrderRejected
			}
		} else {
			if sameSidePosition != nil {
				activeMarginMode = sameSidePosition.MarginMode
				if sameSidePosition.MarginMode != input.MarginMode {
					return apperr.ErrOrderRejected
				}
			}
			if oppositeSidePosition != nil && oppositeSidePosition.MarginMode != input.MarginMode {
				return apperr.ErrOrderRejected
			}
		}

		snapshotAvailable := account.AvailableBalance
		reservation := limitReservation{}
		var parentOrder *model.Order
		if input.ParentOrderID != nil {
			parentOrder = &model.Order{}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ? AND user_id = ? AND type = ? AND parent_order_id IS NULL", *input.ParentOrderID, input.UserID, "limit").
				First(parentOrder).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return apperr.ErrOrderRejected
				}
				return err
			}
			if parentOrder.Status != "triggered" {
				return apperr.ErrOrderRejected
			}
			reservation.Margin = parentOrder.ReservedMargin
			reservation.Fee = parentOrder.ReservedFee
		}

		order := model.Order{
			ClientOrderID:   input.ClientOrderID,
			UserID:          input.UserID,
			Symbol:          input.Symbol,
			Side:            input.Side,
			Type:            input.Type,
			MarginMode:      activeMarginMode,
			Size:            input.Size,
			Price:           price,
			LimitPrice:      input.LimitPrice,
			TimeInForce:     input.TimeInForce,
			Leverage:        input.Leverage,
			Margin:          requiredMargin,
			ReduceOnly:      input.ReduceOnly,
			Status:          "filled",
			FilledSize:      input.Size,
			ExecPrice:       price,
			Fee:             fee,
			ParentOrderID:   input.ParentOrderID,
			ExecutionSource: input.ExecutionSource,
		}
		if input.Type == "limit" {
			order.Status = "open"
			order.FilledSize = decimal.Zero
			order.ExecPrice = decimal.Zero
			order.Fee = decimal.Zero
		}

		if input.Type == "limit" && !input.ReduceOnly {
			order.ReservedMargin = requiredMargin
			order.ReservedFee = fee
		}

		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		if input.Type == "limit" {
			if !input.ReduceOnly {
				totalNeed := requiredMargin.Add(fee)
				if account.AvailableBalance.LessThan(totalNeed) {
					return apperr.ErrInsufficientMargin
				}
				if orderIncreasesExposure(exposurePosition, input.Side, input.Size, input.ReduceOnly) && riskState.FreeCollateral.LessThan(totalNeed) {
					return apperr.ErrOrderRejected
				}
				account.AvailableBalance = account.AvailableBalance.Sub(totalNeed)
				account.LockedBalance = account.LockedBalance.Add(requiredMargin)
				if err := tx.Create(&model.LedgerEntry{
					UserID:        input.UserID,
					Type:          "limit_reserve",
					Amount:        totalNeed,
					BalanceBefore: snapshotAvailable,
					BalanceAfter:  snapshotAvailable.Sub(totalNeed),
					ReferenceType: "order",
					ReferenceID:   order.ID,
					Description:   "reserved margin and fee budget for limit order",
				}).Error; err != nil {
					return err
				}
				if err := tx.Save(&account).Error; err != nil {
					return err
				}
			}
			response = &OrderExecutionOutput{
				Order: order,
				Account: AccountBalanceBrief{
					AvailableBalance: account.AvailableBalance,
					LockedBalance:    account.LockedBalance,
				},
			}
			return nil
		}

		if sameSidePosition == nil && (!input.ReduceOnly || oppositeSidePosition == nil) {
			if input.ReduceOnly {
				return apperr.ErrNoOpenPosition
			}
			if notional.GreaterThan(symbol.MaxPositionNotional) {
				return apperr.ErrMaxPositionExceeded
			}
			totalNeed := requiredMargin.Add(fee)
			availableNeed := totalNeed
			if hasReservation(reservation) {
				delta := totalNeed.Sub(reservation.Margin.Add(reservation.Fee))
				if delta.GreaterThan(decimal.Zero) {
					availableNeed = delta
				} else {
					availableNeed = decimal.Zero
				}
			}
			if account.AvailableBalance.LessThan(availableNeed) {
				return apperr.ErrInsufficientMargin
			}
			if orderIncreasesExposure(exposurePosition, input.Side, input.Size, input.ReduceOnly) && riskState.FreeCollateral.LessThan(availableNeed) {
				return apperr.ErrOrderRejected
			}

			settleReservation(&account, reservation, requiredMargin, fee)
			marginConsumed = requiredMargin

			positionLeverage := effectivePositionLeverage(price, input.Size, requiredMargin, input.Leverage)
			effectiveMaintenance := effectiveMaintenanceRate(symbol.InitialMarginRate, symbol.MaintenanceMarginRate, positionLeverage)
			position := model.Position{
				UserID:           input.UserID,
				Symbol:           input.Symbol,
				Side:             input.Side,
				MarginMode:       input.MarginMode,
				Size:             input.Size,
				EntryPrice:       price,
				MarkPrice:        price,
				LiquidationPrice: calculateLiquidationPrice(input.Side, input.MarginMode, price, input.Size, requiredMargin, effectiveMaintenance),
				Margin:           requiredMargin,
				Leverage:         positionLeverage,
				Status:           "open",
			}
			if err := tx.Create(&position).Error; err != nil {
				return err
			}
			builtPosition, buildErr := s.buildPositionOutput(tx, account, position, price, effectiveMaintenance)
			if buildErr != nil {
				return buildErr
			}
			positionOut = builtPosition

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
			if sameSidePosition != nil && !input.ReduceOnly {
				existing := *sameSidePosition
				positionVersion := existing.Version
				totalNeed := requiredMargin.Add(fee)

				newSize := existing.Size.Add(input.Size)
				newMargin := existing.Margin.Add(requiredMargin)
				newEntry := existing.EntryPrice.Mul(existing.Size).Add(price.Mul(input.Size)).Div(newSize)
				newNotional := price.Mul(newSize)
				if newNotional.GreaterThan(symbol.MaxPositionNotional) {
					return apperr.ErrMaxPositionExceeded
				}

				availableNeed := totalNeed
				if hasReservation(reservation) {
					delta := totalNeed.Sub(reservation.Margin.Add(reservation.Fee))
					if delta.GreaterThan(decimal.Zero) {
						availableNeed = delta
					} else {
						availableNeed = decimal.Zero
					}
				}
				if account.AvailableBalance.LessThan(availableNeed) {
					return apperr.ErrInsufficientMargin
				}
				if riskState.FreeCollateral.LessThan(availableNeed) {
					return apperr.ErrOrderRejected
				}

				settleReservation(&account, reservation, requiredMargin, fee)
				marginConsumed = requiredMargin

				positionLeverage := effectivePositionLeverage(newEntry, newSize, newMargin, existing.Leverage)
				effectiveMaintenance := effectiveMaintenanceRate(symbol.InitialMarginRate, symbol.MaintenanceMarginRate, positionLeverage)
				result := tx.Model(&model.Position{}).
					Where("id = ? AND version = ?", existing.ID, positionVersion).
					Updates(map[string]interface{}{
						"size":              newSize,
						"margin":            newMargin,
						"entry_price":       newEntry,
						"mark_price":        price,
						"leverage":          positionLeverage,
						"margin_mode":       activeMarginMode,
						"liquidation_price": calculateLiquidationPrice(existing.Side, activeMarginMode, newEntry, newSize, newMargin, effectiveMaintenance),
						"version":           gorm.Expr("version + 1"),
					})
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected == 0 {
					return apperr.ErrConcurrentUpdate
				}
			} else {
				existing := *oppositeSidePosition
				positionVersion := existing.Version
				closeSize := decimal.Min(existing.Size, input.Size)
				marginReleased = existing.Margin.Mul(closeSize).Div(existing.Size).Round(18)
				realizedPnL = calculatePnL(existing.Side, existing.EntryPrice, price, closeSize).Round(18)

				account.AvailableBalance = account.AvailableBalance.Add(marginReleased).Add(realizedPnL).Sub(fee)
				account.LockedBalance = account.LockedBalance.Sub(marginReleased)

				remainingSize := existing.Size.Sub(closeSize)
				remainingMargin := existing.Margin.Sub(marginReleased)

				if remainingSize.LessThanOrEqual(decimal.Zero) {
					result := tx.Model(&model.Position{}).
						Where("id = ? AND version = ?", existing.ID, positionVersion).
						Updates(map[string]interface{}{
							"size":              decimal.Zero,
							"margin":            decimal.Zero,
							"mark_price":        price,
							"unrealized_pn_l":   decimal.Zero,
							"status":            "closed",
							"liquidation_price": decimal.Zero,
							"version":           gorm.Expr("version + 1"),
						})
					if result.Error != nil {
						return result.Error
					}
					if result.RowsAffected == 0 {
						return apperr.ErrConcurrentUpdate
					}
				} else {
					effectiveMaintenance := effectiveMaintenanceRate(symbol.InitialMarginRate, symbol.MaintenanceMarginRate, existing.Leverage)
					result := tx.Model(&model.Position{}).
						Where("id = ? AND version = ?", existing.ID, positionVersion).
						Updates(map[string]interface{}{
							"size":              remainingSize,
							"margin":            remainingMargin,
							"mark_price":        price,
							"liquidation_price": calculateLiquidationPrice(existing.Side, activeMarginMode, existing.EntryPrice, remainingSize, remainingMargin, effectiveMaintenance),
							"version":           gorm.Expr("version + 1"),
						})
					if result.Error != nil {
						return result.Error
					}
					if result.RowsAffected == 0 {
						return apperr.ErrConcurrentUpdate
					}
					existing.Size = remainingSize
					existing.Margin = remainingMargin
					builtPosition, buildErr := s.buildPositionOutput(tx, account, existing, price, effectiveMaintenance)
					if buildErr != nil {
						return buildErr
					}
					positionOut = builtPosition
				}

				flipSize := input.Size.Sub(closeSize)
				if flipSize.GreaterThan(decimal.Zero) {
					return apperr.ErrOppositePositionMode
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

			ledgerCursor := snapshotAvailable

			if marginReleased.GreaterThan(decimal.Zero) || !realizedPnL.Equal(decimal.Zero) {
				afterSettle := ledgerCursor.Add(marginReleased).Add(realizedPnL).Sub(fee)
				if err := tx.Create(&model.LedgerEntry{
					UserID:        input.UserID,
					Type:          "position_settle",
					Amount:        marginReleased.Add(realizedPnL).Sub(fee),
					BalanceBefore: ledgerCursor,
					BalanceAfter:  afterSettle,
					ReferenceType: "order",
					ReferenceID:   order.ID,
					Description:   "margin released, pnl settled, and fee deducted",
				}).Error; err != nil {
					return err
				}
				ledgerCursor = afterSettle
			}
			if marginConsumed.GreaterThan(decimal.Zero) {
				afterLock := ledgerCursor.Sub(marginConsumed)
				if err := tx.Create(&model.LedgerEntry{
					UserID:        input.UserID,
					Type:          "margin_lock",
					Amount:        marginConsumed,
					BalanceBefore: ledgerCursor,
					BalanceAfter:  afterLock,
					ReferenceType: "order",
					ReferenceID:   order.ID,
					Description:   "margin locked for order",
				}).Error; err != nil {
					return err
				}
			}
		}

		if sameSidePosition == nil && (!input.ReduceOnly || oppositeSidePosition == nil) {
			afterMarginLock := snapshotAvailable.Sub(requiredMargin)
			if reservation.Margin.GreaterThan(decimal.Zero) || reservation.Fee.GreaterThan(decimal.Zero) {
				afterMarginLock = account.AvailableBalance.Add(fee)
			}
			if err := tx.Create(&model.LedgerEntry{
				UserID:        input.UserID,
				Type:          "margin_lock",
				Amount:        marginConsumed,
				BalanceBefore: snapshotAvailable,
				BalanceAfter:  afterMarginLock,
				ReferenceType: "order",
				ReferenceID:   order.ID,
				Description:   "margin locked for new position",
			}).Error; err != nil {
				return err
			}

			if fee.GreaterThan(decimal.Zero) {
				balanceBeforeFee := afterMarginLock
				if reservation.Margin.GreaterThan(decimal.Zero) || reservation.Fee.GreaterThan(decimal.Zero) {
					balanceBeforeFee = account.AvailableBalance.Add(fee)
				}
				if err := tx.Create(&model.LedgerEntry{
					UserID:        input.UserID,
					Type:          "trading_fee",
					Amount:        fee,
					BalanceBefore: balanceBeforeFee,
					BalanceAfter:  balanceBeforeFee.Sub(fee),
					ReferenceType: "order",
					ReferenceID:   order.ID,
					Description:   fmt.Sprintf("taker fee for %s", input.Symbol),
				}).Error; err != nil {
					return err
				}
			}
		}

		if parentOrder != nil {
			if err := tx.Model(parentOrder).Updates(map[string]any{
				"reserved_margin": decimal.Zero,
				"reserved_fee":    decimal.Zero,
			}).Error; err != nil {
				return err
			}
		}

		if err := tx.Save(&account).Error; err != nil {
			return err
		}
		if err := s.createMockHedgeTask(tx, input.Symbol, "trade", price); err != nil {
			return err
		}
		if positionOut != nil {
			refreshedPosition, refreshErr := s.refreshPositionOutput(tx, account, positionOut.ID)
			if refreshErr != nil {
				return refreshErr
			}
			positionOut = refreshedPosition
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
	var account model.Account
	if err := s.db.Where("user_id = ?", userID).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return []PositionListItem{}, nil
		}
		return nil, err
	}

	var positions []model.Position
	if err := s.db.Where("user_id = ? AND status = ?", userID, "open").Order("updated_at desc").Find(&positions).Error; err != nil {
		return nil, err
	}
	symbols, err := s.loadSymbolMap(positions)
	if err != nil {
		return nil, err
	}
	markPrices, err := loadLatestMarkPrices(s.db, positions)
	if err != nil {
		return nil, err
	}

	items := make([]PositionListItem, 0, len(positions))
	for _, pos := range positions {
		maintenanceRate := decimal.Zero
		if sym, ok := symbols[pos.Symbol]; ok {
			maintenanceRate = effectiveMaintenanceRate(sym.InitialMarginRate, sym.MaintenanceMarginRate, pos.Leverage)
		}
		item, err := s.buildPositionOutput(s.db, account, pos, latestMarkForSymbol(markPrices, pos.Symbol, pos.MarkPrice), maintenanceRate)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, nil
}

func (s *OrderService) ListOrders(userID uint64, limit int) ([]OrderListItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var orders []model.Order
	if err := s.db.Where("user_id = ? AND parent_order_id IS NULL", userID).Order("created_at desc").Limit(limit).Find(&orders).Error; err != nil {
		return nil, err
	}

	items := make([]OrderListItem, 0, len(orders))
	for _, order := range orders {
		items = append(items, OrderListItem{
			ID:            order.ID,
			ClientOrderID: order.ClientOrderID,
			Symbol:        order.Symbol,
			Side:          order.Side,
			Type:          order.Type,
			MarginMode:    order.MarginMode,
			Size:          order.Size,
			LimitPrice:    order.LimitPrice,
			TimeInForce:   order.TimeInForce,
			ExecPrice:     order.ExecPrice,
			Leverage:      order.Leverage,
			Margin:        order.Margin,
			ReduceOnly:    order.ReduceOnly,
			Status:        order.Status,
			Fee:           order.Fee,
			RealizedPnL:   order.RealizedPnL,
			CreatedAt:     order.CreatedAt.UTC().Format(time.RFC3339),
			CancelReason:  order.CancelReason,
		})
		if order.TriggeredAt != nil {
			items[len(items)-1].TriggeredAt = order.TriggeredAt.UTC().Format(time.RFC3339)
		}
	}
	return items, nil
}

func (s *OrderService) ListOpenOrders(userID uint64, limit int) ([]OrderListItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var orders []model.Order
	if err := s.db.Where("user_id = ? AND parent_order_id IS NULL AND status IN ?", userID, []string{"open", "pending", "partially_filled", "triggered"}).
		Order("created_at desc").
		Limit(limit).
		Find(&orders).Error; err != nil {
		return nil, err
	}

	items := make([]OrderListItem, 0, len(orders))
	for _, order := range orders {
		items = append(items, OrderListItem{
			ID:            order.ID,
			ClientOrderID: order.ClientOrderID,
			Symbol:        order.Symbol,
			Side:          order.Side,
			Type:          order.Type,
			MarginMode:    order.MarginMode,
			Size:          order.Size,
			LimitPrice:    order.LimitPrice,
			TimeInForce:   order.TimeInForce,
			ExecPrice:     order.ExecPrice,
			Leverage:      order.Leverage,
			Margin:        order.Margin,
			ReduceOnly:    order.ReduceOnly,
			Status:        order.Status,
			Fee:           order.Fee,
			RealizedPnL:   order.RealizedPnL,
			CreatedAt:     order.CreatedAt.UTC().Format(time.RFC3339),
			CancelReason:  order.CancelReason,
		})
		if order.TriggeredAt != nil {
			items[len(items)-1].TriggeredAt = order.TriggeredAt.UTC().Format(time.RFC3339)
		}
	}
	return items, nil
}

func (s *OrderService) CancelOrder(userID, orderID uint64) (*OrderListItem, error) {
	var out *OrderListItem
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ? AND parent_order_id IS NULL", orderID, userID).
			First(&order).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperr.ErrOrderRejected
			}
			return err
		}
		if order.Type != "limit" || order.Status != "open" {
			return apperr.ErrOrderRejected
		}
		if order.ReservedMargin.GreaterThan(decimal.Zero) || order.ReservedFee.GreaterThan(decimal.Zero) {
			var account model.Account
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ?", userID).
				First(&account).Error; err != nil {
				return err
			}
			totalRefund := order.ReservedMargin.Add(order.ReservedFee)
			balanceBefore := account.AvailableBalance
			account.AvailableBalance = account.AvailableBalance.Add(totalRefund)
			account.LockedBalance = account.LockedBalance.Sub(order.ReservedMargin)
			if err := tx.Save(&account).Error; err != nil {
				return err
			}
			if err := tx.Create(&model.LedgerEntry{
				UserID:        userID,
				Type:          "limit_release",
				Amount:        totalRefund,
				BalanceBefore: balanceBefore,
				BalanceAfter:  account.AvailableBalance,
				ReferenceType: "order",
				ReferenceID:   order.ID,
				Description:   "released reserved margin and fee budget for canceled limit order",
			}).Error; err != nil {
				return err
			}
		}
		cancelReason := "user_cancelled"
		if err := tx.Model(&order).Updates(map[string]any{
			"status":          "canceled",
			"cancel_reason":   cancelReason,
			"reserved_margin": decimal.Zero,
			"reserved_fee":    decimal.Zero,
		}).Error; err != nil {
			return err
		}
		out = &OrderListItem{
			ID:            order.ID,
			ClientOrderID: order.ClientOrderID,
			Symbol:        order.Symbol,
			Side:          order.Side,
			Type:          order.Type,
			MarginMode:    order.MarginMode,
			Size:          order.Size,
			LimitPrice:    order.LimitPrice,
			TimeInForce:   order.TimeInForce,
			ExecPrice:     order.ExecPrice,
			Leverage:      order.Leverage,
			Margin:        order.Margin,
			ReduceOnly:    order.ReduceOnly,
			Status:        "canceled",
			Fee:           order.Fee,
			RealizedPnL:   order.RealizedPnL,
			CreatedAt:     order.CreatedAt.UTC().Format(time.RFC3339),
			CancelReason:  cancelReason,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *OrderService) ListTrades(userID uint64, limit int) ([]TradeListItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var trades []model.Trade
	if err := s.db.Where("user_id = ?", userID).Order("created_at desc").Limit(limit).Find(&trades).Error; err != nil {
		return nil, err
	}

	items := make([]TradeListItem, 0, len(trades))
	for _, trade := range trades {
		items = append(items, TradeListItem{
			ID:            trade.ID,
			OrderID:       trade.OrderID,
			Symbol:        trade.Symbol,
			Side:          trade.Side,
			Size:          trade.Size,
			Price:         trade.Price,
			Fee:           trade.Fee,
			RealizedPnL:   trade.RealizedPnL,
			IsLiquidation: trade.IsLiquidation,
			CreatedAt:     trade.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return items, nil
}

func (s *OrderService) refreshPositionOutput(tx *gorm.DB, account model.Account, positionID uint64) (*PositionListItem, error) {
	var pos model.Position
	if err := tx.Where("id = ?", positionID).First(&pos).Error; err != nil {
		return nil, err
	}
	var symbol model.Symbol
	if err := tx.Where("name = ?", pos.Symbol).First(&symbol).Error; err != nil {
		return nil, err
	}
	markPrices, err := loadLatestMarkPrices(tx, []model.Position{pos})
	if err != nil {
		return nil, err
	}
	return s.buildPositionOutput(
		tx,
		account,
		pos,
		latestMarkForSymbol(markPrices, pos.Symbol, pos.MarkPrice),
		effectiveMaintenanceRate(symbol.InitialMarginRate, symbol.MaintenanceMarginRate, pos.Leverage),
	)
}

func (s *OrderService) buildPositionOutput(tx *gorm.DB, account model.Account, pos model.Position, markPrice, maintenanceRate decimal.Decimal) (*PositionListItem, error) {
	markPrices, err := loadLatestMarkPrices(tx, []model.Position{pos})
	if err != nil {
		return nil, err
	}
	currentMark := latestMarkForSymbol(markPrices, pos.Symbol, markPrice)
	effectiveMargin, err := s.positionEffectiveMargin(tx, account, pos, currentMark)
	if err != nil {
		return nil, err
	}
	unrealized := calculatePnL(pos.Side, pos.EntryPrice, currentMark, pos.Size).Round(18)
	notional := currentMark.Mul(pos.Size).Round(18)
	maintenanceMargin := notional.Mul(maintenanceRate).Round(18)
	equity := effectiveMargin.Add(unrealized)
	riskRatio := decimal.Zero
	if equity.GreaterThan(decimal.Zero) {
		riskRatio = maintenanceMargin.Div(equity).Mul(decimal.NewFromInt(100)).Round(6)
	}

	return &PositionListItem{
		ID:                pos.ID,
		Symbol:            pos.Symbol,
		Side:              pos.Side,
		MarginMode:        pos.MarginMode,
		Size:              pos.Size,
		EntryPrice:        pos.EntryPrice,
		MarkPrice:         currentMark,
		LiquidationPrice:  calculateLiquidationPrice(pos.Side, pos.MarginMode, pos.EntryPrice, pos.Size, effectiveMargin, maintenanceRate),
		Margin:            pos.Margin,
		Notional:          notional,
		MaintenanceMargin: maintenanceMargin,
		RiskRatio:         riskRatio,
		Leverage:          pos.Leverage,
		UnrealizedPnL:     unrealized,
		RealizedPnL:       pos.RealizedPnL,
		Status:            pos.Status,
	}, nil
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

func minimumRequiredMargin(notional decimal.Decimal, leverage uint32) decimal.Decimal {
	if leverage == 0 {
		return decimal.Zero
	}
	return notional.Div(decimal.NewFromInt(int64(leverage))).Round(18)
}

func effectivePositionLeverage(entryPrice, size, margin decimal.Decimal, fallback uint32) uint32 {
	if entryPrice.LessThanOrEqual(decimal.Zero) || size.LessThanOrEqual(decimal.Zero) || margin.LessThanOrEqual(decimal.Zero) {
		if fallback == 0 {
			return 1
		}
		return fallback
	}

	leverage := entryPrice.Mul(size).Div(margin).Ceil().IntPart()
	if leverage <= 0 {
		if fallback == 0 {
			return 1
		}
		return fallback
	}
	return uint32(leverage)
}

func effectiveMaintenanceRate(initialRate, maintenanceRate decimal.Decimal, leverage uint32) decimal.Decimal {
	if leverage == 0 {
		return maintenanceRate
	}
	derivedInitialRate := decimal.NewFromInt(1).Div(decimal.NewFromInt(int64(leverage)))
	if initialRate.GreaterThan(decimal.Zero) && derivedInitialRate.GreaterThan(initialRate) {
		derivedInitialRate = initialRate
	}
	derivedMaintenanceCap := derivedInitialRate.Div(decimal.NewFromInt(2))
	if derivedMaintenanceCap.LessThanOrEqual(decimal.Zero) {
		return maintenanceRate
	}
	if maintenanceRate.LessThanOrEqual(decimal.Zero) {
		return derivedMaintenanceCap
	}
	if derivedMaintenanceCap.LessThan(maintenanceRate) {
		return derivedMaintenanceCap
	}
	return maintenanceRate
}

func calculateLiquidationPrice(side, marginMode string, entryPrice, size, margin, maintenanceRate decimal.Decimal) decimal.Decimal {
	if size.LessThanOrEqual(decimal.Zero) || entryPrice.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}

	switch side {
	case "long":
		denominator := size.Mul(decimal.NewFromInt(1).Sub(maintenanceRate))
		if denominator.IsZero() {
			return decimal.Zero
		}
		return decimal.Max(entryPrice.Mul(size).Sub(margin).Div(denominator).Round(18), decimal.Zero)
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

func normalizeMarginMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "", "isolated":
		return "isolated"
	case "cross":
		return "cross"
	default:
		return ""
	}
}

func normalizeTimeInForce(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "gtc":
		return "gtc"
	default:
		return ""
	}
}

func (s *OrderService) loadSymbolMap(positions []model.Position) (map[string]model.Symbol, error) {
	if len(positions) == 0 {
		return map[string]model.Symbol{}, nil
	}
	names := make([]string, 0, len(positions))
	seen := make(map[string]struct{}, len(positions))
	for _, pos := range positions {
		if _, ok := seen[pos.Symbol]; ok {
			continue
		}
		seen[pos.Symbol] = struct{}{}
		names = append(names, pos.Symbol)
	}
	var symbols []model.Symbol
	if err := s.db.Where("name IN ?", names).Find(&symbols).Error; err != nil {
		return nil, err
	}
	out := make(map[string]model.Symbol, len(symbols))
	for _, sym := range symbols {
		out[sym.Name] = sym
	}
	return out, nil
}

func (s *OrderService) positionEffectiveMargin(tx *gorm.DB, account model.Account, pos model.Position, markPrice decimal.Decimal) (decimal.Decimal, error) {
	if pos.MarginMode != "cross" {
		return pos.Margin, nil
	}

	var crossPositions []model.Position
	if err := tx.Where("user_id = ? AND status = ? AND margin_mode = ?", pos.UserID, "open", "cross").Find(&crossPositions).Error; err != nil {
		return decimal.Zero, err
	}
	if len(crossPositions) == 0 {
		return pos.Margin, nil
	}

	markPrices, err := loadLatestMarkPrices(tx, crossPositions)
	if err != nil {
		return decimal.Zero, err
	}

	totalCrossNotional := decimal.Zero
	positionNotional := markPrice.Mul(pos.Size)
	for _, item := range crossPositions {
		itemPrice := latestMarkForSymbol(markPrices, item.Symbol, item.MarkPrice)
		if item.ID == pos.ID {
			itemPrice = markPrice
		}
		notional := itemPrice.Mul(item.Size)
		totalCrossNotional = totalCrossNotional.Add(notional)
		if item.ID == pos.ID {
			positionNotional = notional
		}
	}
	if totalCrossNotional.LessThanOrEqual(decimal.Zero) {
		return pos.Margin, nil
	}

	sharedCollateral := account.AvailableBalance
	allocation := sharedCollateral.Mul(positionNotional).Div(totalCrossNotional).Round(18)
	return pos.Margin.Add(allocation), nil
}

func (s *OrderService) createMockHedgeTask(tx *gorm.DB, symbol, triggerType string, markPrice decimal.Decimal) error {
	var positions []model.Position
	if err := tx.Where("symbol = ? AND status = ?", symbol, "open").Find(&positions).Error; err != nil {
		return err
	}

	internalNet := decimal.Zero
	for _, pos := range positions {
		if pos.Side == "long" {
			internalNet = internalNet.Add(pos.Size)
		} else {
			internalNet = internalNet.Sub(pos.Size)
		}
	}
	target := internalNet.Round(18)
	currentExternal, err := loadProjectedExternalPosition(tx, symbol)
	if err != nil {
		return err
	}
	drift := target.Sub(currentExternal).Round(18)
	status := "noop"
	side := ""
	size := drift.Abs()
	if !drift.IsZero() {
		status = "pending"
		if drift.GreaterThan(decimal.Zero) {
			side = "long"
		} else {
			side = "short"
		}
	}

	task := model.HedgeTask{
		Symbol:               symbol,
		TriggerType:          triggerType,
		InternalNetPosition:  internalNet,
		TargetHedgePosition:  target,
		CurrentHedgePosition: currentExternal,
		Drift:                drift,
		Status:               status,
	}
	if err := tx.Create(&task).Error; err != nil {
		return err
	}
	if err := supersedeOlderHedgeTasks(tx, symbol, task.ID); err != nil {
		return err
	}
	if status == "noop" {
		return nil
	}
	return tx.Create(&model.HedgeOrder{
		HedgeTaskID:     task.ID,
		Symbol:          symbol,
		Side:            side,
		Size:            size,
		Price:           markPrice,
		ExternalOrderID: "",
		Status:          "pending",
	}).Error
}

func supersedeOlderHedgeTasks(tx *gorm.DB, symbol string, keepTaskID uint64) error {
	activeStatuses := []string{"pending", "retrying", "buffered", "failed"}
	if err := tx.Model(&model.HedgeTask{}).
		Where("symbol = ? AND id <> ? AND status IN ?", symbol, keepTaskID, activeStatuses).
		Updates(map[string]any{
			"status":        "superseded",
			"error_message": "superseded by a newer hedge task",
		}).Error; err != nil {
		return err
	}

	return tx.Model(&model.HedgeOrder{}).
		Where("symbol = ? AND hedge_task_id <> ? AND status IN ?", symbol, keepTaskID, activeStatuses).
		Updates(map[string]any{
			"status":        "superseded",
			"error_message": "superseded by a newer hedge task",
		}).Error
}

func loadProjectedExternalPosition(tx *gorm.DB, symbol string) (decimal.Decimal, error) {
	var lastTask model.HedgeTask
	if err := tx.Where("symbol = ?", symbol).Order("id desc").First(&lastTask).Error; err == nil {
		switch lastTask.Status {
		case "pending", "retrying", "completed", "noop":
			return lastTask.TargetHedgePosition, nil
		default:
			return lastTask.CurrentHedgePosition, nil
		}
	} else if err != gorm.ErrRecordNotFound {
		return decimal.Zero, err
	}

	var lastSnapshot model.SystemRiskSnapshot
	if err := tx.Where("symbol = ?", symbol).Order("id desc").First(&lastSnapshot).Error; err == nil {
		return lastSnapshot.ExternalHedgePosition, nil
	} else if err != gorm.ErrRecordNotFound {
		return decimal.Zero, err
	}

	return decimal.Zero, nil
}

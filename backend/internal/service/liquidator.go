package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AboAuther/RGPerp/backend/internal/model"
)

type LiquidatorService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewLiquidatorService(db *gorm.DB, logger *zap.Logger) *LiquidatorService {
	return &LiquidatorService{db: db, logger: logger}
}

func (s *LiquidatorService) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.scanAndLiquidate(); err != nil {
				s.logger.Warn("liquidator scan failed", zap.Error(err))
			}
		}
	}
}

func (s *LiquidatorService) scanAndLiquidate() error {
	var userIDs []uint64
	if err := s.db.Model(&model.Position{}).
		Distinct("user_id").
		Where("status = ?", "open").
		Pluck("user_id", &userIDs).Error; err != nil {
		return err
	}

	for _, userID := range userIDs {
		riskState, err := syncUserRiskStatus(s.db, userID)
		if err != nil {
			return err
		}
		if riskState.RiskLevel != "liquidating" {
			continue
		}
		if err := s.liquidateUser(userID); err != nil {
			s.logger.Warn("liquidate user failed", zap.Uint64("user_id", userID), zap.Error(err))
		}
	}
	return nil
}

func (s *LiquidatorService) liquidateUser(userID uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		orderService := NewOrderService(tx)

		var account model.Account
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).First(&account).Error; err != nil {
			return err
		}

		var positions []model.Position
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND status = ?", userID, "open").Order("updated_at asc").Find(&positions).Error; err != nil {
			return err
		}
		if len(positions) == 0 {
			return nil
		}

		riskState, err := syncUserRiskStatusTx(tx, userID)
		if err != nil {
			return err
		}
		if riskState.RiskLevel != "liquidating" {
			return nil
		}

		symbolMap, err := orderService.loadSymbolMap(positions)
		if err != nil {
			return err
		}
		markPrices, err := loadLatestMarkPrices(tx, positions)
		if err != nil {
			return err
		}

		targets := make([]model.Position, 0, len(positions))
		for _, pos := range positions {
			pos.MarkPrice = latestMarkForSymbol(markPrices, pos.Symbol, pos.MarkPrice)
			sym := symbolMap[pos.Symbol]
			maintenanceRate := effectiveMaintenanceRate(sym.InitialMarginRate, sym.MaintenanceMarginRate, pos.Leverage)
			if normalizeMarginMode(pos.MarginMode) == "cross" {
				if riskState.HasCrossLiquidation {
					targets = append(targets, pos)
				}
				continue
			}
			if isolatedPositionShouldLiquidate(pos, pos.MarkPrice, maintenanceRate) {
				targets = append(targets, pos)
			}
		}
		if len(targets) == 0 {
			return nil
		}

		for _, pos := range targets {
			sym := symbolMap[pos.Symbol]
			originalMarkPrice := pos.MarkPrice
			originalLiquidationPrice := pos.LiquidationPrice
			portion := decimal.RequireFromString("0.5")
			liquidationType := "partial"
			if normalizeMarginMode(pos.MarginMode) != "cross" || riskState.Equity.LessThanOrEqual(decimal.Zero) || len(targets) == 1 {
				portion = decimal.NewFromInt(1)
				liquidationType = "full"
			}

			closeSize := pos.Size.Mul(portion).Round(18)
			if closeSize.LessThanOrEqual(decimal.Zero) || closeSize.GreaterThanOrEqual(pos.Size) {
				closeSize = pos.Size
				liquidationType = "full"
			}

			execPrice := liquidationExecutionPrice(pos.Side, pos.MarkPrice)
			marginReleased := pos.Margin.Mul(closeSize).Div(pos.Size).Round(18)
			realizedPnL := calculatePnL(pos.Side, pos.EntryPrice, execPrice, closeSize).Round(18)
			liquidationFee := execPrice.Mul(closeSize).Mul(decimal.RequireFromString("0.001")).Round(18)

			beforeBalance := account.AvailableBalance
			account.AvailableBalance = account.AvailableBalance.Add(marginReleased).Add(realizedPnL).Sub(liquidationFee)
			account.LockedBalance = account.LockedBalance.Sub(marginReleased)

			remainingSize := pos.Size.Sub(closeSize).Round(18)
			remainingMargin := pos.Margin.Sub(marginReleased).Round(18)
			if remainingSize.LessThanOrEqual(decimal.Zero) {
				pos.Size = decimal.Zero
				pos.Margin = decimal.Zero
				pos.Status = "closed"
				pos.UnrealizedPnL = decimal.Zero
				pos.MarkPrice = execPrice
				pos.LiquidationPrice = decimal.Zero
			} else {
				pos.Size = remainingSize
				pos.Margin = remainingMargin
				pos.MarkPrice = execPrice
				pos.LiquidationPrice = calculateLiquidationPrice(
					pos.Side,
					pos.MarginMode,
					pos.EntryPrice,
					pos.Size,
					pos.Margin,
					effectiveMaintenanceRate(sym.InitialMarginRate, sym.MaintenanceMarginRate, pos.Leverage),
				)
			}
			if err := tx.Save(&pos).Error; err != nil {
				return err
			}

			insurance := decimal.Zero
			if account.AvailableBalance.LessThan(decimal.Zero) {
				insurance = account.AvailableBalance.Abs().Round(18)
				account.AvailableBalance = decimal.Zero
			}

			order := model.Order{
				ClientOrderID: fmt.Sprintf("liq-%d-%d", userID, pos.ID),
				UserID:        userID,
				Symbol:        pos.Symbol,
				Side:          oppositeSide(pos.Side),
				Type:          "market",
				MarginMode:    pos.MarginMode,
				Size:          closeSize,
				Price:         execPrice,
				Leverage:      pos.Leverage,
				Margin:        marginReleased,
				ReduceOnly:    true,
				Status:        "filled",
				FilledSize:    closeSize,
				ExecPrice:     execPrice,
				Fee:           liquidationFee,
				RealizedPnL:   realizedPnL,
			}
			if err := tx.Create(&order).Error; err != nil {
				return err
			}

			if err := tx.Create(&model.Trade{
				OrderID:       order.ID,
				UserID:        userID,
				Symbol:        pos.Symbol,
				Side:          order.Side,
				Size:          closeSize,
				Price:         execPrice,
				Margin:        marginReleased,
				Fee:           liquidationFee,
				RealizedPnL:   realizedPnL,
				IsLiquidation: true,
			}).Error; err != nil {
				return err
			}

			if err := tx.Create(&model.Liquidation{
				UserID:           userID,
				PositionID:       pos.ID,
				Symbol:           pos.Symbol,
				Side:             pos.Side,
				Size:             closeSize,
				EntryPrice:       pos.EntryPrice,
				MarkPrice:        originalMarkPrice,
				LiquidationPrice: originalLiquidationPrice,
				ExecutionPrice:   execPrice,
				MarginReleased:   marginReleased,
				RealizedPnL:      realizedPnL,
				InsuranceFund:    insurance,
				Type:             liquidationType,
				Status:           "completed",
			}).Error; err != nil {
				return err
			}

			if err := tx.Create(&model.LedgerEntry{
				UserID:        userID,
				Type:          "liquidation_settle",
				Amount:        marginReleased.Add(realizedPnL).Sub(liquidationFee),
				BalanceBefore: beforeBalance,
				BalanceAfter:  account.AvailableBalance,
				ReferenceType: "liquidation",
				ReferenceID:   pos.ID,
				Description:   "liquidation settle and fee deduction",
			}).Error; err != nil {
				return err
			}

			if insurance.GreaterThan(decimal.Zero) {
				var fund model.InsuranceFund
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("symbol = ?", pos.Symbol).First(&fund).Error; err != nil {
					s.logger.Warn("insurance fund not found", zap.String("symbol", pos.Symbol), zap.Error(err))
				} else {
					deduct := decimal.Min(insurance, fund.Balance)
					if deduct.GreaterThan(decimal.Zero) {
						if err := tx.Model(&model.InsuranceFund{}).
							Where("id = ?", fund.ID).
							Update("balance", gorm.Expr("balance - ?", deduct)).Error; err != nil {
							return err
						}
					}
					shortfall := insurance.Sub(deduct)
					if shortfall.GreaterThan(decimal.Zero) {
						s.logger.Warn("insurance fund shortfall",
							zap.String("symbol", pos.Symbol),
							zap.String("shortfall", shortfall.String()),
						)
					}
				}
			}

			if err := orderService.createMockHedgeTask(tx, pos.Symbol, "liquidation", execPrice); err != nil {
				return err
			}

			if err := tx.Save(&account).Error; err != nil {
				return err
			}

			if normalizeMarginMode(pos.MarginMode) == "cross" {
				riskState, err = syncUserRiskStatusTx(tx, userID)
				if err != nil {
					return err
				}
				if !riskState.HasCrossLiquidation {
					break
				}
			}
		}

		if _, err := syncUserRiskStatusTx(tx, userID); err != nil {
			return err
		}
		return nil
	})
}

func liquidationExecutionPrice(side string, markPrice decimal.Decimal) decimal.Decimal {
	penalty := decimal.RequireFromString("0.0025")
	if side == "long" {
		return markPrice.Mul(decimal.NewFromInt(1).Sub(penalty)).Round(18)
	}
	return markPrice.Mul(decimal.NewFromInt(1).Add(penalty)).Round(18)
}

func oppositeSide(side string) string {
	if side == "long" {
		return "short"
	}
	return "long"
}

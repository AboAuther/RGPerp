package hedge

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AboAuther/RGPerp/backend/internal/model"
	"github.com/shopspring/decimal"
)

type Service struct {
	db      *gorm.DB
	logger  *zap.Logger
	adapter Adapter
}

func NewService(db *gorm.DB, logger *zap.Logger, adapter Adapter) *Service {
	return &Service{
		db:      db,
		logger:  logger,
		adapter: adapter,
	}
}

func (s *Service) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.processPending(ctx); err != nil {
				s.logger.Warn("process hedge tasks failed", zap.Error(err))
			}
			if err := s.captureRiskSnapshots(); err != nil {
				s.logger.Warn("capture hedge snapshots failed", zap.Error(err))
			}
		}
	}
}

func (s *Service) processPending(ctx context.Context) error {
	var tasks []model.HedgeTask
	if err := s.db.Where("status IN ?", []string{"pending", "retrying", "buffered"}).Order("id asc").Limit(20).Find(&tasks).Error; err != nil {
		return err
	}

	for _, task := range tasks {
		if err := s.processTask(ctx, task.ID); err != nil {
			s.logger.Warn("process hedge task failed", zap.Uint64("task_id", task.ID), zap.Error(err))
		}
	}
	return nil
}

func (s *Service) processTask(ctx context.Context, taskID uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var task model.HedgeTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", taskID).First(&task).Error; err != nil {
			return err
		}
		if task.Status != "pending" && task.Status != "retrying" && task.Status != "buffered" {
			return nil
		}

		var order model.HedgeOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("hedge_task_id = ?", task.ID).Order("id asc").First(&order).Error; err != nil {
			return err
		}

		currentExternal, err := s.adapter.GetPosition(ctx, task.Symbol)
		if err != nil {
			return err
		}
		task.CurrentHedgePosition = currentExternal
		delta := task.TargetHedgePosition.Sub(currentExternal).Round(18)
		task.Drift = delta
		if delta.Abs().LessThanOrEqual(decimal.RequireFromString("0.0001")) {
			task.Status = "completed"
			task.ErrorMessage = ""
			order.Status = "filled"
			order.FilledSize = decimal.Zero
			order.FilledPrice = order.Price
			if err := tx.Save(&order).Error; err != nil {
				return err
			}
			return tx.Save(&task).Error
		}
		order.Side = "short"
		if delta.GreaterThan(decimal.Zero) {
			order.Side = "long"
		}
		order.Size = delta.Abs()
		markPrice, err := loadLatestMarkPriceForSymbol(tx, task.Symbol, order.Price)
		if err != nil {
			return err
		}
		order.Price = markPrice
		minNotional := s.adapter.MinOrderNotional(task.Symbol)
		orderNotional := markPrice.Mul(order.Size)
		if minNotional.GreaterThan(decimal.Zero) && orderNotional.LessThan(minNotional) {
			task.Status = "buffered"
			task.ErrorMessage = fmt.Sprintf("buffered until hedge notional reaches %s USDC (current %s USDC)", minNotional.StringFixed(2), orderNotional.Round(4).String())
			order.Status = "buffered"
			order.ErrorMessage = task.ErrorMessage
			order.Price = markPrice
			if err := tx.Save(&order).Error; err != nil {
				return err
			}
			return tx.Save(&task).Error
		}
		reduceOnly := !currentExternal.IsZero() &&
			currentExternal.Sign() != delta.Sign() &&
			delta.Abs().LessThanOrEqual(currentExternal.Abs())

		result, err := s.adapter.PlaceOrder(ctx, OrderRequest{
			Symbol:     task.Symbol,
			Side:       order.Side,
			Size:       order.Size,
			Price:      order.Price,
			ReduceOnly: reduceOnly,
		})
		if err != nil {
			order.RetryCount++
			task.Status = "failed"
			order.Status = "failed"
			if order.RetryCount < 3 {
				task.Status = "retrying"
				order.Status = "retrying"
			}
			task.ErrorMessage = err.Error()
			order.ErrorMessage = err.Error()
			if saveErr := tx.Save(&order).Error; saveErr != nil {
				return saveErr
			}
			return tx.Save(&task).Error
		}

		order.ExternalOrderID = result.ExternalOrderID
		order.Status = normalizeOrderStatus(result.Status)
		order.FilledSize = result.FilledSize
		order.FilledPrice = result.FilledPrice
		order.ErrorMessage = ""
		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		actualFilled := result.FilledSize
		if order.Side == "short" {
			actualFilled = actualFilled.Neg()
		}
		task.CurrentHedgePosition = currentExternal.Add(actualFilled)
		task.Drift = task.TargetHedgePosition.Sub(task.CurrentHedgePosition).Round(18)

		if task.Drift.Abs().LessThanOrEqual(decimal.RequireFromString("0.0001")) {
			task.Status = "completed"
		} else {
			task.Status = "retrying"
			s.logger.Warn("hedge partially filled, will retry",
				zap.String("symbol", task.Symbol),
				zap.String("filled", result.FilledSize.String()),
				zap.String("remaining_drift", task.Drift.String()),
			)
		}
		task.ErrorMessage = ""
		return tx.Save(&task).Error
	})
}

func (s *Service) captureRiskSnapshots() error {
	var symbols []model.Symbol
	if err := s.db.Where("status = ?", "trading").Find(&symbols).Error; err != nil {
		return err
	}

	for _, sym := range symbols {
		if err := s.captureSymbolRisk(sym); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) captureSymbolRisk(sym model.Symbol) error {
	var positions []model.Position
	if err := s.db.Where("symbol = ? AND status = ?", sym.Name, "open").Find(&positions).Error; err != nil {
		return err
	}

	totalLong := decimal.Zero
	totalShort := decimal.Zero
	for _, pos := range positions {
		if pos.Side == "long" {
			totalLong = totalLong.Add(pos.Size)
		} else {
			totalShort = totalShort.Add(pos.Size)
		}
	}
	internalNet := totalLong.Sub(totalShort).Round(18)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	externalPos, err := s.adapter.GetPosition(ctx, sym.Name)
	if err != nil {
		if _, ok := s.adapter.(*MockAdapter); !ok {
			return fmt.Errorf("get external position for %s: %w", sym.Name, err)
		}
	}

	drift := externalPos.Sub(internalNet).Round(18)
	threshold := decimal.Max(sym.LotSize, decimal.RequireFromString("0.01"))
	healthy := drift.Abs().LessThanOrEqual(threshold)

	openInterest := decimal.Zero
	for _, pos := range positions {
		openInterest = openInterest.Add(pos.MarkPrice.Mul(pos.Size))
	}

	if err := s.db.Create(&model.SystemRiskSnapshot{
		Symbol:                sym.Name,
		TotalLongPosition:     totalLong,
		TotalShortPosition:    totalShort,
		NetPosition:           internalNet,
		ExternalHedgePosition: externalPos,
		Drift:                 drift,
		HedgeHealthy:          healthy,
		TotalOpenInterest:     openInterest.Round(18),
	}).Error; err != nil {
		return err
	}

	if !healthy {
		s.logger.Warn("hedge drift exceeds threshold",
			zap.String("symbol", sym.Name),
			zap.String("drift", drift.String()),
			zap.String("threshold", threshold.String()),
		)
	}
	return nil
}

func normalizeOrderStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "filled", "mock_filled":
		return "filled"
	case "buffered":
		return "buffered"
	case "retrying":
		return "retrying"
	case "failed":
		return "failed"
	case "submitted":
		return "submitted"
	default:
		return "filled"
	}
}

func loadLatestMarkPriceForSymbol(db *gorm.DB, symbol string, fallback decimal.Decimal) (decimal.Decimal, error) {
	var tick model.PriceTick
	if err := db.Where("symbol = ?", strings.ToUpper(strings.TrimSpace(symbol))).Order("created_at desc, id desc").First(&tick).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fallback, nil
		}
		return decimal.Zero, err
	}
	if tick.MarkPrice.GreaterThan(decimal.Zero) {
		return tick.MarkPrice, nil
	}
	return fallback, nil
}

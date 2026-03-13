package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AboAuther/RGPerp/backend/internal/model"
	apperr "github.com/AboAuther/RGPerp/backend/internal/pkg/errors"
)

type LimitMatcherService struct {
	db           *gorm.DB
	orderService *OrderService
	logger       *zap.Logger
}

func NewLimitMatcherService(db *gorm.DB, logger *zap.Logger) *LimitMatcherService {
	return &LimitMatcherService{
		db:           db,
		orderService: NewOrderService(db),
		logger:       logger,
	}
}

func (s *LimitMatcherService) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.runOnce()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce()
		}
	}
}

func (s *LimitMatcherService) runOnce() {
	var orders []model.Order
	if err := s.db.
		Where("type = ? AND status = ? AND parent_order_id IS NULL", "limit", "open").
		Order("id asc").
		Limit(100).
		Find(&orders).Error; err != nil {
		s.logger.Warn("failed to load limit orders", zap.Error(err))
		return
	}

	for _, order := range orders {
		if err := s.processOrder(order.ID); err != nil {
			s.logger.Warn("failed to process limit order", zap.Uint64("order_id", order.ID), zap.Error(err))
		}
	}
}

func (s *LimitMatcherService) processOrder(orderID uint64) error {
	var (
		order         model.Order
		now           = time.Now().UTC()
		shouldExecute bool
	)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND type = ? AND parent_order_id IS NULL", orderID, "limit").
			First(&order).Error; err != nil {
			return err
		}
		if order.Status != "open" {
			return nil
		}

		var tick model.PriceTick
		if err := tx.Where("symbol = ?", order.Symbol).Order("created_at desc").First(&tick).Error; err != nil {
			return err
		}
		if !limitOrderShouldTrigger(order, tick) {
			return nil
		}
		shouldExecute = true

		return tx.Model(&order).Updates(map[string]any{
			"status":       "triggered",
			"triggered_at": now,
		}).Error
	})
	if err != nil {
		return err
	}
	if !shouldExecute || order.ID == 0 {
		return nil
	}

	childID := fmt.Sprintf("limit-fill-%d-%d", order.ID, now.UnixNano())
	result, err := s.orderService.Create(CreateOrderInput{
		UserID:          order.UserID,
		ClientOrderID:   childID,
		Symbol:          order.Symbol,
		Side:            order.Side,
		Type:            "market",
		MarginMode:      order.MarginMode,
		Size:            order.Size,
		Leverage:        order.Leverage,
		Margin:          order.Margin,
		ReduceOnly:      order.ReduceOnly,
		ParentOrderID:   &order.ID,
		ExecutionSource: "matcher",
	})
	if err != nil {
		if isRetryableLimitError(err) {
			return s.db.Model(&model.Order{}).
				Where("id = ?", order.ID).
				Updates(map[string]any{
					"status":        "open",
					"error_message": "",
					"triggered_at":  nil,
				}).Error
		}
		return s.db.Transaction(func(tx *gorm.DB) error {
			var parent model.Order
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ?", order.ID).
				First(&parent).Error; err != nil {
				return err
			}
			if parent.ReservedMargin.GreaterThan(decimal.Zero) || parent.ReservedFee.GreaterThan(decimal.Zero) {
				var account model.Account
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("user_id = ?", parent.UserID).
					First(&account).Error; err != nil {
					return err
				}
				totalRefund := parent.ReservedMargin.Add(parent.ReservedFee)
				balanceBefore := account.AvailableBalance
				account.AvailableBalance = account.AvailableBalance.Add(totalRefund)
				account.LockedBalance = account.LockedBalance.Sub(parent.ReservedMargin)
				if err := tx.Save(&account).Error; err != nil {
					return err
				}
				if err := tx.Create(&model.LedgerEntry{
					UserID:        parent.UserID,
					Type:          "limit_release",
					Amount:        totalRefund,
					BalanceBefore: balanceBefore,
					BalanceAfter:  account.AvailableBalance,
					ReferenceType: "order",
					ReferenceID:   parent.ID,
					Description:   "released reserved margin and fee budget after limit trigger rejection",
				}).Error; err != nil {
					return err
				}
			}
			return tx.Model(&parent).
				Updates(map[string]any{
					"status":          "rejected",
					"error_message":   err.Error(),
					"triggered_at":    now,
					"reserved_margin": decimal.Zero,
					"reserved_fee":    decimal.Zero,
				}).Error
		})
	}

	return s.db.Model(&model.Order{}).
		Where("id = ?", order.ID).
		Updates(map[string]any{
			"status":        "filled",
			"filled_size":   order.Size,
			"exec_price":    result.Order.ExecPrice,
			"realized_pn_l": result.Order.RealizedPnL,
			"fee":           result.Order.Fee,
			"triggered_at":  now,
			"error_message": "",
		}).Error
}

func limitOrderShouldTrigger(order model.Order, tick model.PriceTick) bool {
	if order.Side == "long" {
		return tick.BestAsk.LessThanOrEqual(order.LimitPrice)
	}
	if order.Side == "short" {
		return tick.BestBid.GreaterThanOrEqual(order.LimitPrice)
	}
	return false
}

func isRetryableLimitError(err error) bool {
	return errors.Is(err, apperr.ErrPriceUnavailable) || errors.Is(err, apperr.ErrPriceStale) || errors.Is(err, apperr.ErrConcurrentUpdate)
}

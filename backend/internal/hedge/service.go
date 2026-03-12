package hedge

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/AboAuther/RGPerp/backend/internal/model"
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
		}
	}
}

func (s *Service) processPending(ctx context.Context) error {
	var tasks []model.HedgeTask
	if err := s.db.Where("status = ?", "pending").Order("id asc").Limit(20).Find(&tasks).Error; err != nil {
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
		if err := tx.Where("id = ?", taskID).First(&task).Error; err != nil {
			return err
		}
		if task.Status != "pending" {
			return nil
		}

		var order model.HedgeOrder
		if err := tx.Where("hedge_task_id = ?", task.ID).Order("id asc").First(&order).Error; err != nil {
			return err
		}

		result, err := s.adapter.PlaceOrder(ctx, OrderRequest{
			Symbol: task.Symbol,
			Side:   order.Side,
			Size:   order.Size,
			Price:  order.Price,
		})
		if err != nil {
			task.Status = "failed"
			task.ErrorMessage = err.Error()
			order.Status = "mock_failed"
			order.ErrorMessage = err.Error()
			if saveErr := tx.Save(&order).Error; saveErr != nil {
				return saveErr
			}
			return tx.Save(&task).Error
		}

		order.ExternalOrderID = result.ExternalOrderID
		order.Status = "mock_filled"
		order.FilledSize = result.FilledSize
		order.FilledPrice = result.FilledPrice
		if err := tx.Save(&order).Error; err != nil {
			return err
		}

		task.Status = "completed"
		task.CurrentHedgePosition = task.TargetHedgePosition
		task.Drift = task.TargetHedgePosition.Sub(task.CurrentHedgePosition)
		task.ErrorMessage = ""
		return tx.Save(&task).Error
	})
}

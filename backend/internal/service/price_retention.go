package service

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PriceRetentionCleaner struct {
	db       *gorm.DB
	logger   *zap.Logger
	retention time.Duration
}

func NewPriceRetentionCleaner(db *gorm.DB, logger *zap.Logger, retention time.Duration) *PriceRetentionCleaner {
	return &PriceRetentionCleaner{
		db:        db,
		logger:    logger,
		retention: retention,
	}
}

func (c *PriceRetentionCleaner) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.Cleanup(); err != nil {
				c.logger.Warn("price tick retention cleanup failed", zap.Error(err))
			}
		}
	}
}

func (c *PriceRetentionCleaner) Cleanup() error {
	cutoff := time.Now().UTC().Add(-c.retention)
	result := c.db.Exec("DELETE FROM price_ticks WHERE created_at < ?", cutoff)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		c.logger.Info("price tick retention cleanup completed",
			zap.Duration("retention", c.retention),
			zap.Int64("deleted_rows", result.RowsAffected),
		)
	}
	return nil
}

package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/AboAuther/RGPerp/backend/internal/model"
)

type MockPriceFeeder struct {
	db            *gorm.DB
	logger        *zap.Logger
	marketService *MarketService
	interval      time.Duration
}

func NewMockPriceFeeder(db *gorm.DB, logger *zap.Logger, interval time.Duration) *MockPriceFeeder {
	return &MockPriceFeeder{
		db:            db,
		logger:        logger,
		marketService: NewMarketService(db),
		interval:      interval,
	}
}

func (f *MockPriceFeeder) Run(ctx context.Context) {
	ticker := time.NewTicker(f.interval)
	defer ticker.Stop()

	lastPrice := map[string]decimal.Decimal{}
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			var symbols []model.Symbol
			if err := f.db.Where("status = ?", "trading").Find(&symbols).Error; err != nil {
				f.logger.Warn("mock price feeder load symbols failed", zap.Error(err))
				continue
			}

			for _, sym := range symbols {
				base := lastPrice[sym.Name]
				if base.IsZero() {
					base = defaultMockPrice(sym.Name)
				}
				// Random walk drift in [-0.2%, +0.2%], enough for local integration.
				drift := (rand.Float64() - 0.5) * 0.004
				next := base.Mul(decimal.NewFromFloat(1 + drift))
				if next.LessThan(decimal.NewFromInt(1)) {
					next = decimal.NewFromInt(1)
				}

				lastPrice[sym.Name] = next
				if err := f.marketService.UpsertMockTick(sym.Name, next, now); err != nil {
					f.logger.Warn("mock price feeder write tick failed",
						zap.String("symbol", sym.Name),
						zap.Error(err),
					)
				}
			}
		}
	}
}

func defaultMockPrice(symbol string) decimal.Decimal {
	switch symbol {
	case "BTC-PERP":
		return decimal.NewFromInt(85000)
	case "ETH-PERP":
		return decimal.NewFromInt(3000)
	default:
		return decimal.NewFromInt(100)
	}
}

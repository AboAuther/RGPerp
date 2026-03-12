package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/AboAuther/RGPerp/backend/internal/config"
	"github.com/AboAuther/RGPerp/backend/internal/model"
)

type PriceFeeder struct {
	db            *gorm.DB
	logger        *zap.Logger
	cfg           *config.Config
	marketService *MarketService
	httpClient    *http.Client
	oracle        *OracleReader
}

func NewPriceFeeder(db *gorm.DB, logger *zap.Logger, cfg *config.Config) *PriceFeeder {
	return &PriceFeeder{
		db:            db,
		logger:        logger,
		cfg:           cfg,
		marketService: NewMarketService(db),
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
		oracle: NewOracleReader(cfg.Blockchain.RPCURL),
	}
}

func (f *PriceFeeder) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	lastPrice := map[string]decimal.Decimal{}
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			var symbols []model.Symbol
			if err := f.db.Where("status = ?", "trading").Find(&symbols).Error; err != nil {
				f.logger.Warn("price feeder load symbols failed", zap.Error(err))
				continue
			}

			switch strings.ToLower(strings.TrimSpace(f.cfg.Price.Source)) {
			case "", "mock":
				f.writeMockTicks(now, symbols, lastPrice)
			case "hyperliquid":
				f.writeHyperliquidTicks(now, symbols)
			case "oracle":
				f.writeOracleTicks(now, symbols)
			default:
				f.logger.Warn("unknown price source, fallback to mock", zap.String("source", f.cfg.Price.Source))
				f.writeMockTicks(now, symbols, lastPrice)
			}
		}
	}
}

func (f *PriceFeeder) writeMockTicks(now time.Time, symbols []model.Symbol, lastPrice map[string]decimal.Decimal) {
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
		if err := f.marketService.UpsertTick(sym.Name, next, now, "mock"); err != nil {
			f.logger.Warn("mock write tick failed", zap.String("symbol", sym.Name), zap.Error(err))
		}
	}
}

func (f *PriceFeeder) writeHyperliquidTicks(now time.Time, symbols []model.Symbol) {
	mids, err := f.fetchHyperliquidAllMids()
	if err != nil {
		f.logger.Warn("fetch hyperliquid mids failed", zap.Error(err))
		return
	}

	for _, sym := range symbols {
		mid, ok := mids[strings.ToUpper(sym.BaseAsset)]
		if !ok {
			continue
		}
		price, err := decimal.NewFromString(mid)
		if err != nil || price.LessThanOrEqual(decimal.Zero) {
			continue
		}
		if err := f.marketService.UpsertTick(sym.Name, price, now, "hyperliquid"); err != nil {
			f.logger.Warn("hyperliquid write tick failed", zap.String("symbol", sym.Name), zap.Error(err))
		}
	}
}

func (f *PriceFeeder) fetchHyperliquidAllMids() (map[string]string, error) {
	endpoint := strings.TrimRight(f.cfg.Hyperliquid.APIURL, "/") + "/info"
	payload := map[string]string{"type": "allMids"}
	raw, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var data map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func (f *PriceFeeder) writeOracleTicks(now time.Time, symbols []model.Symbol) {
	for _, sym := range symbols {
		feedAddr := f.oracleFeedForSymbol(sym.Name)
		if feedAddr == "" {
			continue
		}
		price, err := f.oracle.GetPrice(context.Background(), feedAddr)
		if err != nil || price.LessThanOrEqual(decimal.Zero) {
			f.logger.Warn("oracle read failed", zap.String("symbol", sym.Name), zap.Error(err))
			continue
		}
		if err := f.marketService.UpsertTick(sym.Name, price, now, "oracle"); err != nil {
			f.logger.Warn("oracle write tick failed", zap.String("symbol", sym.Name), zap.Error(err))
		}
	}
}

func (f *PriceFeeder) oracleFeedForSymbol(symbol string) string {
	switch strings.ToUpper(symbol) {
	case "BTC-PERP":
		return f.cfg.Oracle.BTCUSDFeed
	case "ETH-PERP":
		return f.cfg.Oracle.ETHUSDFeed
	default:
		return ""
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

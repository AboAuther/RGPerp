package service

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/AboAuther/RGPerp/backend/internal/model"
	apperr "github.com/AboAuther/RGPerp/backend/internal/pkg/errors"
)

type MarketService struct {
	db *gorm.DB
}

func NewMarketService(db *gorm.DB) *MarketService {
	return &MarketService{db: db}
}

type MarketOutput struct {
	Name                  string          `json:"name"`
	BaseAsset             string          `json:"base_asset"`
	QuoteAsset            string          `json:"quote_asset"`
	Status                string          `json:"status"`
	MaxLeverage           uint32          `json:"max_leverage"`
	MinOrderSize          decimal.Decimal `json:"min_order_size"`
	TickSize              decimal.Decimal `json:"tick_size"`
	LotSize               decimal.Decimal `json:"lot_size"`
	InitialMarginRate     decimal.Decimal `json:"initial_margin_rate"`
	MaintenanceMarginRate decimal.Decimal `json:"maintenance_margin_rate"`
	MakerFeeRate          decimal.Decimal `json:"maker_fee_rate"`
	TakerFeeRate          decimal.Decimal `json:"taker_fee_rate"`
}

type TickerOutput struct {
	Symbol     string          `json:"symbol"`
	MarkPrice  decimal.Decimal `json:"mark_price"`
	IndexPrice decimal.Decimal `json:"index_price"`
	BestBid    decimal.Decimal `json:"best_bid"`
	BestAsk    decimal.Decimal `json:"best_ask"`
	Source     string          `json:"source"`
	Timestamp  int64           `json:"timestamp"`
}

func (s *MarketService) ListMarkets() ([]MarketOutput, error) {
	var symbols []model.Symbol
	if err := s.db.Order("id asc").Find(&symbols).Error; err != nil {
		return nil, err
	}

	out := make([]MarketOutput, 0, len(symbols))
	for _, item := range symbols {
		out = append(out, MarketOutput{
			Name:                  item.Name,
			BaseAsset:             item.BaseAsset,
			QuoteAsset:            item.QuoteAsset,
			Status:                item.Status,
			MaxLeverage:           item.MaxLeverage,
			MinOrderSize:          item.MinOrderSize,
			TickSize:              item.TickSize,
			LotSize:               item.LotSize,
			InitialMarginRate:     item.InitialMarginRate,
			MaintenanceMarginRate: item.MaintenanceMarginRate,
			MakerFeeRate:          item.MakerFeeRate,
			TakerFeeRate:          item.TakerFeeRate,
		})
	}
	return out, nil
}

func (s *MarketService) GetTicker(symbol string) (*TickerOutput, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, apperr.ErrBadRequest
	}

	var symbolRow model.Symbol
	if err := s.db.Where("name = ? AND status = ?", symbol, "trading").First(&symbolRow).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrSymbolNotFound
		}
		return nil, err
	}

	var tick model.PriceTick
	if err := s.db.Where("symbol = ?", symbol).Order("created_at desc").First(&tick).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrPriceUnavailable
		}
		return nil, err
	}

	indexPrice := tick.IndexPrice
	if indexPrice.IsZero() {
		indexPrice = tick.MarkPrice
	}

	return &TickerOutput{
		Symbol:     symbol,
		MarkPrice:  tick.MarkPrice,
		IndexPrice: indexPrice,
		BestBid:    tick.BestBid,
		BestAsk:    tick.BestAsk,
		Source:     tick.Source,
		Timestamp:  tick.CreatedAt.UTC().Unix(),
	}, nil
}

func (s *MarketService) UpsertMockTick(symbol string, markPrice decimal.Decimal, ts time.Time) error {
	indexPrice := markPrice
	halfSpread := markPrice.Mul(decimal.NewFromFloat(0.0002))
	bestBid := markPrice.Sub(halfSpread)
	bestAsk := markPrice.Add(halfSpread)

	return s.db.Create(&model.PriceTick{
		Symbol:     symbol,
		IndexPrice: indexPrice,
		MarkPrice:  markPrice,
		BestBid:    bestBid,
		BestAsk:    bestAsk,
		Source:     "mock",
		CreatedAt:  ts.UTC(),
	}).Error
}

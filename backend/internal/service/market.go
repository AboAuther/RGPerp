package service

import (
	"fmt"
	"sort"
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
	Symbol        string          `json:"symbol"`
	MarkPrice     decimal.Decimal `json:"mark_price"`
	IndexPrice    decimal.Decimal `json:"index_price"`
	BestBid       decimal.Decimal `json:"best_bid"`
	BestAsk       decimal.Decimal `json:"best_ask"`
	Source        string          `json:"source"`
	Timestamp     int64           `json:"timestamp"`
	Change24H     decimal.Decimal `json:"change_24h"`
	Change24HPct  decimal.Decimal `json:"change_24h_pct"`
	Volume24H     decimal.Decimal `json:"volume_24h"`
	OpenInterest  decimal.Decimal `json:"open_interest"`
	FundingRate   decimal.Decimal `json:"funding_rate"`
	FundingNextAt int64           `json:"funding_next_at"`
	MaxLeverage   uint32          `json:"max_leverage"`
	DisplayPair   string          `json:"display_pair"`
	BaseAsset     string          `json:"base_asset"`
	QuoteAsset    string          `json:"quote_asset"`
}

type KlineOutput struct {
	Time   int64           `json:"time"`
	Open   decimal.Decimal `json:"open"`
	High   decimal.Decimal `json:"high"`
	Low    decimal.Decimal `json:"low"`
	Close  decimal.Decimal `json:"close"`
	Volume decimal.Decimal `json:"volume"`
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

	stats, err := s.buildTickerStats(symbol, tick.MarkPrice, indexPrice)
	if err != nil {
		return nil, err
	}

	return &TickerOutput{
		Symbol:        symbol,
		MarkPrice:     tick.MarkPrice,
		IndexPrice:    indexPrice,
		BestBid:       tick.BestBid,
		BestAsk:       tick.BestAsk,
		Source:        tick.Source,
		Timestamp:     tick.CreatedAt.UTC().Unix(),
		Change24H:     stats.Change24H,
		Change24HPct:  stats.Change24HPct,
		Volume24H:     stats.Volume24H,
		OpenInterest:  stats.OpenInterest,
		FundingRate:   stats.FundingRate,
		FundingNextAt: stats.FundingNextAt.UTC().Unix(),
		MaxLeverage:   symbolRow.MaxLeverage,
		DisplayPair:   fmt.Sprintf("%s/%s", symbolRow.BaseAsset, symbolRow.QuoteAsset),
		BaseAsset:     symbolRow.BaseAsset,
		QuoteAsset:    symbolRow.QuoteAsset,
	}, nil
}

type tickerStats struct {
	Change24H     decimal.Decimal
	Change24HPct  decimal.Decimal
	Volume24H     decimal.Decimal
	OpenInterest  decimal.Decimal
	FundingRate   decimal.Decimal
	FundingNextAt time.Time
}

func (s *MarketService) buildTickerStats(symbol string, markPrice, indexPrice decimal.Decimal) (*tickerStats, error) {
	windowStart := time.Now().UTC().Add(-24 * time.Hour)

	var firstTick model.PriceTick
	if err := s.db.Where("symbol = ? AND created_at >= ?", symbol, windowStart).Order("created_at asc").First(&firstTick).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		if err := s.db.Where("symbol = ?", symbol).Order("created_at asc").First(&firstTick).Error; err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}

	change24H := decimal.Zero
	change24HPct := decimal.Zero
	if firstTick.MarkPrice.GreaterThan(decimal.Zero) {
		change24H = markPrice.Sub(firstTick.MarkPrice).Round(6)
		change24HPct = change24H.Div(firstTick.MarkPrice).Mul(decimal.NewFromInt(100)).Round(4)
	}

	var trades []model.Trade
	if err := s.db.Select("size", "price").Where("symbol = ? AND created_at >= ?", symbol, windowStart).Find(&trades).Error; err != nil {
		return nil, err
	}
	volume24H := decimal.Zero
	for _, trade := range trades {
		volume24H = volume24H.Add(trade.Size.Mul(trade.Price))
	}

	var positions []model.Position
	if err := s.db.Select("size").Where("symbol = ? AND status = ?", symbol, "open").Find(&positions).Error; err != nil {
		return nil, err
	}
	openInterest := decimal.Zero
	for _, position := range positions {
		openInterest = openInterest.Add(position.Size.Abs().Mul(markPrice))
	}

	now := time.Now().UTC()
	fundingNextAt := now.Truncate(time.Hour).Add(time.Hour)
	fundingRate := decimal.Zero

	var latestTick model.PriceTick
	if err := s.db.Select("funding_rate", "funding_next_at").
		Where("symbol = ?", symbol).
		Order("created_at desc").
		First(&latestTick).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if latestTick.FundingRate.GreaterThan(decimal.Zero) || latestTick.FundingRate.LessThan(decimal.Zero) {
		fundingRate = latestTick.FundingRate.Round(6)
	} else if indexPrice.GreaterThan(decimal.Zero) {
		fundingRate = markPrice.Sub(indexPrice).Div(indexPrice).Mul(decimal.RequireFromString("0.1")).Round(6)
	}
	if latestTick.FundingNextAt != nil && !latestTick.FundingNextAt.IsZero() {
		fundingNextAt = latestTick.FundingNextAt.UTC()
	}

	return &tickerStats{
		Change24H:     change24H,
		Change24HPct:  change24HPct,
		Volume24H:     volume24H.Round(2),
		OpenInterest:  openInterest.Round(2),
		FundingRate:   fundingRate,
		FundingNextAt: fundingNextAt,
	}, nil
}

func (s *MarketService) UpsertTick(symbol string, markPrice decimal.Decimal, ts time.Time, source string) error {
	return s.UpsertTickWithDetails(symbol, markPrice, markPrice, decimal.Zero, decimal.Zero, decimal.Zero, nil, ts, source)
}

func (s *MarketService) UpsertTickWithDetails(
	symbol string,
	markPrice decimal.Decimal,
	indexPrice decimal.Decimal,
	bestBid decimal.Decimal,
	bestAsk decimal.Decimal,
	fundingRate decimal.Decimal,
	fundingNextAt *time.Time,
	ts time.Time,
	source string,
) error {
	var symbolRow model.Symbol
	tickSize := decimal.RequireFromString("0.000001")
	if err := s.db.Select("tick_size").Where("name = ?", symbol).First(&symbolRow).Error; err == nil && symbolRow.TickSize.GreaterThan(decimal.Zero) {
		tickSize = symbolRow.TickSize
	}

	markPrice = quantizeToStep(markPrice, tickSize)
	if indexPrice.LessThanOrEqual(decimal.Zero) {
		indexPrice = markPrice
	}
	indexPrice = quantizeToStep(indexPrice, tickSize)
	if bestBid.LessThanOrEqual(decimal.Zero) || bestAsk.LessThanOrEqual(decimal.Zero) {
		halfSpread := quantizeToStep(markPrice.Mul(decimal.RequireFromString("0.0002")), tickSize)
		bestBid = markPrice.Sub(halfSpread)
		bestAsk = markPrice.Add(halfSpread)
	} else {
		bestBid = quantizeToStep(bestBid, tickSize)
		bestAsk = quantizeToStep(bestAsk, tickSize)
	}

	return s.db.Create(&model.PriceTick{
		Symbol:        symbol,
		IndexPrice:    indexPrice,
		MarkPrice:     markPrice,
		BestBid:       bestBid,
		BestAsk:       bestAsk,
		FundingRate:   fundingRate,
		FundingNextAt: fundingNextAt,
		Source:        source,
		CreatedAt:     ts.UTC(),
	}).Error
}

func (s *MarketService) GetKlines(symbol, interval string, startTime, endTime time.Time, limit int) ([]KlineOutput, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, apperr.ErrBadRequest
	}

	bucketSize, err := parseInterval(interval)
	if err != nil {
		return nil, apperr.ErrBadRequest
	}
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}

	query := s.db.Model(&model.PriceTick{}).Where("symbol = ?", symbol)
	if !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime.UTC())
	}
	if !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime.UTC())
	}

	var ticks []model.PriceTick
	if err := query.Order("created_at asc").Limit(limit * 50).Find(&ticks).Error; err != nil {
		return nil, err
	}
	if len(ticks) == 0 {
		return []KlineOutput{}, nil
	}

	type klineBucket struct {
		time   int64
		open   decimal.Decimal
		high   decimal.Decimal
		low    decimal.Decimal
		close  decimal.Decimal
		volume decimal.Decimal
	}

	buckets := make(map[int64]*klineBucket)
	order := make([]int64, 0, len(ticks))
	for _, tick := range ticks {
		ts := tick.CreatedAt.UTC().Truncate(bucketSize).Unix()
		price := tick.MarkPrice
		b, exists := buckets[ts]
		if !exists {
			b = &klineBucket{
				time:   ts,
				open:   price,
				high:   price,
				low:    price,
				close:  price,
				volume: decimal.NewFromInt(1),
			}
			buckets[ts] = b
			order = append(order, ts)
			continue
		}
		if price.GreaterThan(b.high) {
			b.high = price
		}
		if price.LessThan(b.low) {
			b.low = price
		}
		b.close = price
		b.volume = b.volume.Add(decimal.NewFromInt(1))
	}

	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	if len(order) > limit {
		order = order[len(order)-limit:]
	}

	result := make([]KlineOutput, 0, len(order))
	for _, ts := range order {
		b := buckets[ts]
		result = append(result, KlineOutput{
			Time:   b.time,
			Open:   b.open,
			High:   b.high,
			Low:    b.low,
			Close:  b.close,
			Volume: b.volume,
		})
	}
	return result, nil
}

func quantizeToStep(value, step decimal.Decimal) decimal.Decimal {
	if step.LessThanOrEqual(decimal.Zero) {
		return value.Round(6)
	}
	return value.Div(step).Round(0).Mul(step).Round(6)
}

func parseInterval(interval string) (time.Duration, error) {
	switch strings.ToLower(strings.TrimSpace(interval)) {
	case "", "1m":
		return time.Minute, nil
	case "5m":
		return 5 * time.Minute, nil
	case "15m":
		return 15 * time.Minute, nil
	case "1h":
		return time.Hour, nil
	case "4h":
		return 4 * time.Hour, nil
	case "1d":
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported interval")
	}
}

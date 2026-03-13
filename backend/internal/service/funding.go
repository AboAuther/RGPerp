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

type FundingService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewFundingService(db *gorm.DB, logger *zap.Logger) *FundingService {
	return &FundingService{db: db, logger: logger}
}

type FundingHistoryItem struct {
	Symbol       string          `json:"symbol"`
	Side         string          `json:"side"`
	FundingRate  decimal.Decimal `json:"funding_rate"`
	MarkPrice    decimal.Decimal `json:"mark_price"`
	Notional     decimal.Decimal `json:"notional"`
	Amount       decimal.Decimal `json:"amount"`
	SettlementAt time.Time       `json:"settlement_at"`
	CreatedAt    time.Time       `json:"created_at"`
}

func (s *FundingService) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.ScanAndSettle(time.Now().UTC()); err != nil {
				s.logger.Warn("funding settlement scan failed", zap.Error(err))
			}
		}
	}
}

func (s *FundingService) ScanAndSettle(now time.Time) error {
	symbols, err := s.loadDueSymbols(now)
	if err != nil {
		return err
	}
	for _, item := range symbols {
		if err := s.settleSymbol(item.Symbol, item.FundingRate, item.MarkPrice, item.FundingNextAt.UTC()); err != nil {
			s.logger.Warn("funding settlement failed",
				zap.String("symbol", item.Symbol),
				zap.Time("settlement_at", item.FundingNextAt.UTC()),
				zap.Error(err),
			)
		}
	}
	return nil
}

func (s *FundingService) ListFundingHistory(userID uint64, limit int) ([]FundingHistoryItem, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var events []model.FundingEvent
	if err := s.db.Where("user_id = ?", userID).
		Order("settlement_at desc, id desc").
		Limit(limit).
		Find(&events).Error; err != nil {
		return nil, err
	}

	items := make([]FundingHistoryItem, 0, len(events))
	for _, event := range events {
		items = append(items, FundingHistoryItem{
			Symbol:       event.Symbol,
			Side:         event.Side,
			FundingRate:  event.FundingRate,
			MarkPrice:    event.MarkPrice,
			Notional:     event.Notional,
			Amount:       event.Amount,
			SettlementAt: event.SettlementAt,
			CreatedAt:    event.CreatedAt,
		})
	}
	return items, nil
}

type dueFundingSymbol struct {
	Symbol        string
	FundingRate   decimal.Decimal
	MarkPrice     decimal.Decimal
	FundingNextAt time.Time
}

func (s *FundingService) loadDueSymbols(now time.Time) ([]dueFundingSymbol, error) {
	var symbols []model.Symbol
	if err := s.db.Where("status = ?", "trading").Find(&symbols).Error; err != nil {
		return nil, err
	}

	items := make([]dueFundingSymbol, 0, len(symbols))
	for _, sym := range symbols {
		var ticks []model.PriceTick
		if err := s.db.Where("symbol = ? AND funding_next_at IS NOT NULL AND funding_next_at <= ?", sym.Name, now).
			Order("funding_next_at desc, created_at desc, id desc").
			Limit(1).
			Find(&ticks).Error; err != nil {
			return nil, err
		}
		if len(ticks) == 0 {
			continue
		}
		tick := ticks[0]
		items = append(items, dueFundingSymbol{
			Symbol:        sym.Name,
			FundingRate:   tick.FundingRate,
			MarkPrice:     tick.MarkPrice,
			FundingNextAt: tick.FundingNextAt.UTC(),
		})
	}
	return items, nil
}

func (s *FundingService) settleSymbol(symbol string, fundingRate, markPrice decimal.Decimal, settlementAt time.Time) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var positions []model.Position
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("symbol = ? AND status = ? AND created_at <= ?", symbol, "open", settlementAt).
			Find(&positions).Error; err != nil {
			return err
		}
		if len(positions) == 0 {
			return nil
		}

		impactedUsers := make(map[uint64]struct{})
		for _, pos := range positions {
			var exists int64
			if err := tx.Model(&model.FundingEvent{}).
				Where("position_id = ? AND settlement_at = ?", pos.ID, settlementAt).
				Count(&exists).Error; err != nil {
				return err
			}
			if exists > 0 {
				continue
			}

			var account model.Account
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ?", pos.UserID).
				First(&account).Error; err != nil {
				return err
			}

			notional := markPrice.Mul(pos.Size).Round(18)
			amount := calculateFundingAmount(pos.Side, fundingRate, notional).Round(18)
			before := account.AvailableBalance
			account.AvailableBalance = account.AvailableBalance.Add(amount).Round(18)
			if err := settleRealizedDelta(tx, &account, amount); err != nil {
				return err
			}

			event := model.FundingEvent{
				UserID:       pos.UserID,
				PositionID:   pos.ID,
				Symbol:       pos.Symbol,
				Side:         pos.Side,
				FundingRate:  fundingRate.Round(10),
				MarkPrice:    markPrice.Round(18),
				Notional:     notional,
				Amount:       amount,
				SettlementAt: settlementAt,
			}
			if err := tx.Create(&event).Error; err != nil {
				return err
			}

			description := fmt.Sprintf("funding settlement at %s", settlementAt.Format(time.RFC3339))
			if err := tx.Create(&model.LedgerEntry{
				UserID:        pos.UserID,
				Type:          "funding",
				Amount:        amount,
				BalanceBefore: before,
				BalanceAfter:  account.AvailableBalance,
				ReferenceType: "funding",
				ReferenceID:   event.ID,
				Description:   description,
			}).Error; err != nil {
				return err
			}

			impactedUsers[pos.UserID] = struct{}{}
		}

		for userID := range impactedUsers {
			if _, err := syncUserRiskStatusTx(tx, userID); err != nil {
				return err
			}
		}
		return nil
	})
}

func calculateFundingAmount(side string, fundingRate, notional decimal.Decimal) decimal.Decimal {
	switch side {
	case "long":
		return fundingRate.Mul(notional).Neg()
	case "short":
		return fundingRate.Mul(notional)
	default:
		return decimal.Zero
	}
}

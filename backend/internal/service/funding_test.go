package service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/AboAuther/RGPerp/backend/internal/model"
)

func TestCalculateFundingAmount(t *testing.T) {
	notional := decimal.RequireFromString("1000")
	rate := decimal.RequireFromString("0.0001")

	longAmount := calculateFundingAmount("long", rate, notional)
	if !longAmount.Equal(decimal.RequireFromString("-0.1")) {
		t.Fatalf("unexpected long funding amount: %s", longAmount)
	}

	shortAmount := calculateFundingAmount("short", rate, notional)
	if !shortAmount.Equal(decimal.RequireFromString("0.1")) {
		t.Fatalf("unexpected short funding amount: %s", shortAmount)
	}
}

func TestFundingService_SettlesDuePositionsAndIsIdempotent(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewFundingService(db, nil)

	settlementAt := time.Now().UTC().Truncate(time.Second)
	if err := db.Model(&model.Account{}).Where("user_id = ?", 1).Updates(map[string]any{
		"available_balance": decimal.RequireFromString("100"),
		"locked_balance":    decimal.RequireFromString("2"),
	}).Error; err != nil {
		t.Fatalf("update account: %v", err)
	}
	position := model.Position{
		UserID:           1,
		Symbol:           "BTC-PERP",
		Side:             "long",
		MarginMode:       "isolated",
		Size:             decimal.RequireFromString("0.01"),
		EntryPrice:       decimal.RequireFromString("86000"),
		MarkPrice:        decimal.RequireFromString("86000"),
		LiquidationPrice: decimal.RequireFromString("70000"),
		Margin:           decimal.RequireFromString("2"),
		Leverage:         10,
		Status:           "open",
	}
	if err := db.Create(&position).Error; err != nil {
		t.Fatalf("create position: %v", err)
	}
	if err := db.Model(&position).Update("created_at", settlementAt.Add(-time.Minute)).Error; err != nil {
		t.Fatalf("update position created_at: %v", err)
	}
	if err := db.Create(&model.PriceTick{
		Symbol:        "BTC-PERP",
		IndexPrice:    decimal.RequireFromString("86050"),
		MarkPrice:     decimal.RequireFromString("86100"),
		BestBid:       decimal.RequireFromString("86099"),
		BestAsk:       decimal.RequireFromString("86101"),
		FundingRate:   decimal.RequireFromString("0.0001"),
		FundingNextAt: &settlementAt,
		Source:        "binance",
		CreatedAt:     settlementAt,
	}).Error; err != nil {
		t.Fatalf("create due tick: %v", err)
	}

	if err := svc.ScanAndSettle(settlementAt.Add(time.Second)); err != nil {
		t.Fatalf("first settlement: %v", err)
	}
	if err := svc.ScanAndSettle(settlementAt.Add(2 * time.Second)); err != nil {
		t.Fatalf("second settlement: %v", err)
	}

	var events []model.FundingEvent
	if err := db.Order("id asc").Find(&events).Error; err != nil {
		t.Fatalf("load funding events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 funding event, got %d", len(events))
	}
	if !events[0].Amount.Equal(decimal.RequireFromString("-0.0861")) {
		t.Fatalf("unexpected funding amount: %s", events[0].Amount)
	}

	var ledgerEntries []model.LedgerEntry
	if err := db.Where("type = ?", "funding").Find(&ledgerEntries).Error; err != nil {
		t.Fatalf("load ledger entries: %v", err)
	}
	if len(ledgerEntries) != 1 {
		t.Fatalf("expected 1 funding ledger entry, got %d", len(ledgerEntries))
	}

	var account model.Account
	if err := db.Where("user_id = ?", 1).First(&account).Error; err != nil {
		t.Fatalf("reload account: %v", err)
	}
	if !account.AvailableBalance.Equal(decimal.RequireFromString("99.9139")) {
		t.Fatalf("unexpected available balance: %s", account.AvailableBalance)
	}
}

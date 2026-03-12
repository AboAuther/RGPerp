package service

import (
	"testing"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/AboAuther/RGPerp/backend/internal/model"
)

func TestLiquidatorService_LiquidatesDangerPosition(t *testing.T) {
	db := mustNewOrderTestDB(t)

	if err := db.Model(&model.Account{}).Where("user_id = ?", 1).Updates(map[string]any{
		"available_balance": decimal.Zero,
		"locked_balance":    decimal.RequireFromString("0.2"),
	}).Error; err != nil {
		t.Fatalf("update account: %v", err)
	}
	position := model.Position{
		UserID:           1,
		Symbol:           "BTC-PERP",
		Side:             "long",
		MarginMode:       "cross",
		Size:             decimal.RequireFromString("0.001"),
		EntryPrice:       decimal.RequireFromString("86000"),
		MarkPrice:        decimal.RequireFromString("86000"),
		LiquidationPrice: decimal.RequireFromString("85500"),
		Margin:           decimal.RequireFromString("0.2"),
		Leverage:         10,
		Status:           "open",
	}
	if err := db.Create(&position).Error; err != nil {
		t.Fatalf("create position: %v", err)
	}

	svc := NewLiquidatorService(db, zap.NewNop())
	if err := svc.scanAndLiquidate(); err != nil {
		t.Fatalf("scan and liquidate failed: %v", err)
	}

	var liquidations int64
	if err := db.Model(&model.Liquidation{}).Where("user_id = ?", 1).Count(&liquidations).Error; err != nil {
		t.Fatalf("count liquidations: %v", err)
	}
	if liquidations == 0 {
		t.Fatal("expected liquidation record")
	}

	var updated model.Position
	if err := db.First(&updated, position.ID).Error; err != nil {
		t.Fatalf("reload position: %v", err)
	}
	if updated.Status != "closed" {
		t.Fatalf("expected position closed, got %s", updated.Status)
	}

	var hedgeTask model.HedgeTask
	if err := db.Where("symbol = ? AND trigger_type = ?", "BTC-PERP", "liquidation").Order("id desc").First(&hedgeTask).Error; err != nil {
		t.Fatalf("load liquidation hedge task: %v", err)
	}
}

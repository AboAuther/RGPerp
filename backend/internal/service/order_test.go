package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/AboAuther/RGPerp/backend/internal/model"
)

func TestOrderService_ReduceOnlyClose(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	if _, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "open-long",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      10,
	}); err != nil {
		t.Fatalf("open long failed: %v", err)
	}

	result, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "close-long",
		Symbol:        "BTC-PERP",
		Side:          "short",
		Type:          "market",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      10,
		ReduceOnly:    true,
	})
	if err != nil {
		t.Fatalf("reduce-only close failed: %v", err)
	}
	if result.Order.Status != "filled" {
		t.Fatalf("unexpected order status: %s", result.Order.Status)
	}

	var openPositions int64
	if err := db.Model(&model.Position{}).Where("user_id = ? AND status = ?", 1, "open").Count(&openPositions).Error; err != nil {
		t.Fatalf("count positions failed: %v", err)
	}
	if openPositions != 0 {
		t.Fatalf("expected no open positions, got %d", openPositions)
	}
}

func TestOrderService_CrossMarginAndMockHedge(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	result, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "cross-open",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		MarginMode:    "cross",
		Size:          decimal.RequireFromString("0.002"),
		Leverage:      10,
	})
	if err != nil {
		t.Fatalf("cross open failed: %v", err)
	}
	if result.Position == nil || result.Position.MarginMode != "cross" {
		t.Fatalf("expected cross position, got %#v", result.Position)
	}

	var hedgeTask model.HedgeTask
	if err := db.Order("id desc").First(&hedgeTask).Error; err != nil {
		t.Fatalf("load hedge task: %v", err)
	}
	if hedgeTask.Symbol != "BTC-PERP" || hedgeTask.Status != "pending" {
		t.Fatalf("unexpected hedge task: %#v", hedgeTask)
	}

	var hedgeOrder model.HedgeOrder
	if err := db.Where("hedge_task_id = ?", hedgeTask.ID).First(&hedgeOrder).Error; err != nil {
		t.Fatalf("load hedge order: %v", err)
	}
	if hedgeOrder.Status != "mock_pending" {
		t.Fatalf("unexpected hedge order status: %s", hedgeOrder.Status)
	}
}

func mustNewOrderTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	user := model.User{WalletAddress: "0x70997970C51812dc3A010C7d01b50e0d17dc79C8", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	account := model.Account{
		UserID:           user.ID,
		AvailableBalance: decimal.RequireFromString("500"),
		LockedBalance:    decimal.Zero,
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create account: %v", err)
	}
	symbol := model.Symbol{
		Name:                  "BTC-PERP",
		BaseAsset:             "BTC",
		QuoteAsset:            "USDC",
		Status:                "trading",
		MaxLeverage:           50,
		MinOrderSize:          decimal.RequireFromString("0.001"),
		MaxPositionNotional:   decimal.RequireFromString("1000000"),
		TickSize:              decimal.RequireFromString("0.1"),
		LotSize:               decimal.RequireFromString("0.001"),
		InitialMarginRate:     decimal.RequireFromString("0.02"),
		MaintenanceMarginRate: decimal.RequireFromString("0.005"),
		MakerFeeRate:          decimal.Zero,
		TakerFeeRate:          decimal.RequireFromString("0.0005"),
		HyperliquidAssetIndex: 0,
	}
	if err := db.Create(&symbol).Error; err != nil {
		t.Fatalf("create symbol: %v", err)
	}
	tick := model.PriceTick{
		Symbol:     "BTC-PERP",
		IndexPrice: decimal.RequireFromString("86000"),
		MarkPrice:  decimal.RequireFromString("86000"),
		BestBid:    decimal.RequireFromString("85999"),
		BestAsk:    decimal.RequireFromString("86001"),
		Source:     "mock",
		CreatedAt:  time.Now().UTC(),
	}
	if err := db.Create(&tick).Error; err != nil {
		t.Fatalf("create tick: %v", err)
	}

	return db
}

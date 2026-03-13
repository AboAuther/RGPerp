package hedge

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/AboAuther/RGPerp/backend/internal/model"
)

type bufferOnlyAdapter struct{}

func (bufferOnlyAdapter) PlaceOrder(context.Context, OrderRequest) (*OrderResult, error) {
	return &OrderResult{
		ExternalOrderID: "buffer-stub",
		Status:          "filled",
		FilledSize:      decimal.Zero,
		FilledPrice:     decimal.Zero,
	}, nil
}

func (bufferOnlyAdapter) GetPosition(context.Context, string) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (bufferOnlyAdapter) MinOrderNotional(string) decimal.Decimal {
	return decimal.NewFromInt(10)
}

func TestService_ProcessPending(t *testing.T) {
	db := mustNewHedgeTestDB(t)

	// Users net long 1 BTC → target hedge = long 1 BTC externally
	task := model.HedgeTask{
		Symbol:               "BTC-PERP",
		TriggerType:          "trade",
		InternalNetPosition:  decimal.RequireFromString("1"),
		TargetHedgePosition:  decimal.RequireFromString("1"),
		CurrentHedgePosition: decimal.Zero,
		Drift:                decimal.RequireFromString("1"),
		Status:               "pending",
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	order := model.HedgeOrder{
		HedgeTaskID: task.ID,
		Symbol:      "BTC-PERP",
		Side:        "long",
		Size:        decimal.RequireFromString("1"),
		Price:       decimal.RequireFromString("85000"),
		Status:      "pending",
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	mock := &MockAdapter{positions: make(map[string]decimal.Decimal)}
	svc := NewService(db, zap.NewNop(), mock)
	if err := svc.processPending(context.Background()); err != nil {
		t.Fatalf("process pending: %v", err)
	}

	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if task.Status != "completed" {
		t.Fatalf("unexpected task status: %s", task.Status)
	}
	if !task.CurrentHedgePosition.Equal(task.TargetHedgePosition) {
		t.Fatalf("expected hedge position %s, got %s",
			task.TargetHedgePosition.String(), task.CurrentHedgePosition.String())
	}

	// Verify MockAdapter tracked the position
	pos, _ := mock.GetPosition(context.Background(), "BTC-PERP")
	if !pos.Equal(decimal.RequireFromString("1")) {
		t.Fatalf("mock adapter should track long 1 BTC, got %s", pos.String())
	}
}

func TestService_ProcessPending_ShortHedge(t *testing.T) {
	db := mustNewHedgeTestDB(t)

	// Users net short 0.5 BTC → target hedge = short 0.5 BTC externally
	task := model.HedgeTask{
		Symbol:               "BTC-PERP",
		TriggerType:          "trade",
		InternalNetPosition:  decimal.RequireFromString("-0.5"),
		TargetHedgePosition:  decimal.RequireFromString("-0.5"),
		CurrentHedgePosition: decimal.Zero,
		Drift:                decimal.RequireFromString("-0.5"),
		Status:               "pending",
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	order := model.HedgeOrder{
		HedgeTaskID: task.ID,
		Symbol:      "BTC-PERP",
		Side:        "short",
		Size:        decimal.RequireFromString("0.5"),
		Price:       decimal.RequireFromString("85000"),
		Status:      "pending",
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	mock := &MockAdapter{positions: make(map[string]decimal.Decimal)}
	svc := NewService(db, zap.NewNop(), mock)
	if err := svc.processPending(context.Background()); err != nil {
		t.Fatalf("process pending: %v", err)
	}

	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if task.Status != "completed" {
		t.Fatalf("unexpected task status: %s", task.Status)
	}

	pos, _ := mock.GetPosition(context.Background(), "BTC-PERP")
	if !pos.Equal(decimal.RequireFromString("-0.5")) {
		t.Fatalf("mock adapter should track short 0.5 BTC, got %s", pos.String())
	}
}

func TestService_ProcessPending_BuffersSmallNotionalForHyperliquid(t *testing.T) {
	db := mustNewHedgeTestDB(t)
	if err := db.Create(&model.PriceTick{
		Symbol:     "SOL-PERP",
		IndexPrice: decimal.RequireFromString("89"),
		MarkPrice:  decimal.RequireFromString("89"),
		BestBid:    decimal.RequireFromString("88.9"),
		BestAsk:    decimal.RequireFromString("89.1"),
		Source:     "mock",
	}).Error; err != nil {
		t.Fatalf("create tick: %v", err)
	}

	task := model.HedgeTask{
		Symbol:               "SOL-PERP",
		TriggerType:          "trade",
		InternalNetPosition:  decimal.RequireFromString("0.1"),
		TargetHedgePosition:  decimal.RequireFromString("0.1"),
		CurrentHedgePosition: decimal.Zero,
		Drift:                decimal.RequireFromString("0.1"),
		Status:               "pending",
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
	order := model.HedgeOrder{
		HedgeTaskID: task.ID,
		Symbol:      "SOL-PERP",
		Side:        "long",
		Size:        decimal.RequireFromString("0.1"),
		Price:       decimal.RequireFromString("89"),
		Status:      "pending",
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	adapter := bufferOnlyAdapter{}
	svc := NewService(db, zap.NewNop(), adapter)
	if err := svc.processPending(context.Background()); err != nil {
		t.Fatalf("process pending: %v", err)
	}

	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if task.Status != "buffered" {
		t.Fatalf("expected buffered task status, got %s", task.Status)
	}
	if err := db.First(&order, order.ID).Error; err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if order.Status != "buffered" {
		t.Fatalf("expected buffered order status, got %s", order.Status)
	}
}

func mustNewHedgeTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := db.Create(&model.PriceTick{
		Symbol:     "BTC-PERP",
		IndexPrice: decimal.RequireFromString("85000"),
		MarkPrice:  decimal.RequireFromString("85000"),
		BestBid:    decimal.RequireFromString("84999"),
		BestAsk:    decimal.RequireFromString("85001"),
		Source:     "mock",
	}).Error; err != nil {
		t.Fatalf("create tick: %v", err)
	}

	return db
}

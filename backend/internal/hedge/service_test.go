package hedge

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/AboAuther/RGPerp/backend/internal/model"
)

func TestService_ProcessPending(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:hedge-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

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
		Status:      "mock_pending",
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
	db, err := gorm.Open(sqlite.Open("file:hedge-short-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

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
		Status:      "mock_pending",
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

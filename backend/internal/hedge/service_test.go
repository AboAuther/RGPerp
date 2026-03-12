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

	svc := NewService(db, zap.NewNop(), &MockAdapter{})
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
		t.Fatalf("expected hedge position synced")
	}
}

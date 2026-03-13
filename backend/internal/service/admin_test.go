package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/AboAuther/RGPerp/backend/internal/model"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAdminService_OverviewAndLists(t *testing.T) {
	db := mustNewAdminTestDB(t)
	svc := NewAdminService(db)

	overview, err := svc.GetOverview()
	if err != nil {
		t.Fatalf("get overview: %v", err)
	}
	if overview.TradingSymbols != 1 {
		t.Fatalf("expected 1 trading symbol, got %d", overview.TradingSymbols)
	}
	if overview.PendingHedges != 0 {
		t.Fatalf("expected 0 pending hedges after current-task aggregation, got %d", overview.PendingHedges)
	}
	if overview.BufferedHedges != 1 {
		t.Fatalf("expected 1 buffered hedge, got %d", overview.BufferedHedges)
	}
	if overview.RecentLiquidations != 1 {
		t.Fatalf("expected 1 recent liquidation, got %d", overview.RecentLiquidations)
	}

	hedges, err := svc.ListHedgeTasks(10)
	if err != nil {
		t.Fatalf("list hedge tasks: %v", err)
	}
	if len(hedges) != 2 {
		t.Fatalf("expected 2 hedge task history rows, got %d", len(hedges))
	}
	if hedges[0].Status != "buffered" {
		t.Fatalf("expected latest hedge task to be buffered, got %s", hedges[0].Status)
	}

	snapshots, err := svc.ListRiskSnapshots(10)
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected 1 risk snapshot, got %d", len(snapshots))
	}

	liquidations, err := svc.ListLiquidations(10)
	if err != nil {
		t.Fatalf("list liquidations: %v", err)
	}
	if len(liquidations) != 1 {
		t.Fatalf("expected 1 liquidation, got %d", len(liquidations))
	}

	alerts, err := svc.ListAlerts(10)
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}
	if len(alerts) == 0 {
		t.Fatal("expected alerts")
	}
}

func TestAdminService_RetryHedgeTask(t *testing.T) {
	db := mustNewAdminTestDB(t)
	svc := NewAdminService(db)

	var task model.HedgeTask
	if err := db.Where("status = ?", "buffered").Order("id desc").First(&task).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}

	if err := svc.RetryHedgeTask(task.ID); err != nil {
		t.Fatalf("retry hedge task: %v", err)
	}

	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if task.Status != "noop" {
		t.Fatalf("expected task noop after retry rebases on latest managed state, got %s", task.Status)
	}
	if !task.Drift.IsZero() {
		t.Fatalf("expected drift zero after retry, got %s", task.Drift.String())
	}

	var order model.HedgeOrder
	if err := db.Where("hedge_task_id = ?", task.ID).Order("id desc").First(&order).Error; err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if order.Status != "filled" {
		t.Fatalf("expected order filled after noop retry, got %s", order.Status)
	}
}

func mustNewAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	user := model.User{WalletAddress: "0xadmin", Status: "liquidating"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.Account{
		UserID:           user.ID,
		AvailableBalance: decimal.RequireFromString("10"),
		LockedBalance:    decimal.RequireFromString("2"),
	}).Error; err != nil {
		t.Fatalf("create account: %v", err)
	}
	if err := db.Create(&model.Symbol{
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
	}).Error; err != nil {
		t.Fatalf("create symbol: %v", err)
	}
	if err := db.Create(&model.Position{
		UserID:           user.ID,
		Symbol:           "BTC-PERP",
		Side:             "long",
		MarginMode:       "isolated",
		Size:             decimal.RequireFromString("0.01"),
		EntryPrice:       decimal.RequireFromString("70000"),
		MarkPrice:        decimal.RequireFromString("69900"),
		LiquidationPrice: decimal.RequireFromString("69000"),
		Margin:           decimal.RequireFromString("14"),
		Leverage:         50,
		Status:           "open",
	}).Error; err != nil {
		t.Fatalf("create position: %v", err)
	}
	pendingTask := model.HedgeTask{
		Symbol:               "BTC-PERP",
		TriggerType:          "trade",
		InternalNetPosition:  decimal.RequireFromString("0.01"),
		TargetHedgePosition:  decimal.RequireFromString("0.01"),
		CurrentHedgePosition: decimal.Zero,
		Drift:                decimal.RequireFromString("0.01"),
		Status:               "pending",
	}
	if err := db.Create(&pendingTask).Error; err != nil {
		t.Fatalf("create pending task: %v", err)
	}
	if err := db.Create(&model.HedgeOrder{
		HedgeTaskID: pendingTask.ID,
		Symbol:      "BTC-PERP",
		Side:        "long",
		Size:        decimal.RequireFromString("0.01"),
		Price:       decimal.RequireFromString("69900"),
		Status:      "pending",
	}).Error; err != nil {
		t.Fatalf("create pending order: %v", err)
	}
	bufferedTask := model.HedgeTask{
		Symbol:               "BTC-PERP",
		TriggerType:          "trade",
		InternalNetPosition:  decimal.RequireFromString("0.005"),
		TargetHedgePosition:  decimal.RequireFromString("0.005"),
		CurrentHedgePosition: decimal.Zero,
		Drift:                decimal.RequireFromString("0.005"),
		Status:               "buffered",
		ErrorMessage:         "buffered until hedge notional reaches 10 USDC",
	}
	if err := db.Create(&bufferedTask).Error; err != nil {
		t.Fatalf("create buffered task: %v", err)
	}
	if err := db.Create(&model.HedgeOrder{
		HedgeTaskID: bufferedTask.ID,
		Symbol:      "BTC-PERP",
		Side:        "long",
		Size:        decimal.RequireFromString("0.005"),
		Price:       decimal.RequireFromString("69900"),
		Status:      "buffered",
	}).Error; err != nil {
		t.Fatalf("create buffered order: %v", err)
	}
	if err := db.Create(&model.SystemRiskSnapshot{
		Symbol:                "BTC-PERP",
		TotalLongPosition:     decimal.RequireFromString("0.01"),
		TotalShortPosition:    decimal.Zero,
		NetPosition:           decimal.RequireFromString("0.01"),
		ExternalHedgePosition: decimal.Zero,
		Drift:                 decimal.RequireFromString("-0.01"),
		HedgeHealthy:          false,
		TotalOpenInterest:     decimal.RequireFromString("699"),
	}).Error; err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if err := db.Create(&model.Liquidation{
		UserID:           user.ID,
		PositionID:       1,
		Symbol:           "BTC-PERP",
		Side:             "long",
		Size:             decimal.RequireFromString("0.01"),
		EntryPrice:       decimal.RequireFromString("70000"),
		MarkPrice:        decimal.RequireFromString("69900"),
		LiquidationPrice: decimal.RequireFromString("69000"),
		ExecutionPrice:   decimal.RequireFromString("69850"),
		MarginReleased:   decimal.RequireFromString("14"),
		RealizedPnL:      decimal.RequireFromString("-1.5"),
		Type:             "full",
		Status:           "completed",
	}).Error; err != nil {
		t.Fatalf("create liquidation: %v", err)
	}

	return db
}

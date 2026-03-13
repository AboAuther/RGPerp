package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/AboAuther/RGPerp/backend/internal/model"
	apperr "github.com/AboAuther/RGPerp/backend/internal/pkg/errors"
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
	if hedgeOrder.Status != "pending" {
		t.Fatalf("unexpected hedge order status: %s", hedgeOrder.Status)
	}
}

func TestOrderService_ReduceOnlyRiskBlocksIncreasingExposure(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	if err := db.Model(&model.Account{}).Where("user_id = ?", 1).Updates(map[string]any{
		"available_balance": decimal.Zero,
		"locked_balance":    decimal.RequireFromString("0.5"),
	}).Error; err != nil {
		t.Fatalf("update account: %v", err)
	}
	if err := db.Create(&model.Position{
		UserID:           1,
		Symbol:           "BTC-PERP",
		Side:             "long",
		MarginMode:       "cross",
		Size:             decimal.RequireFromString("0.001"),
		EntryPrice:       decimal.RequireFromString("86000"),
		MarkPrice:        decimal.RequireFromString("86000"),
		LiquidationPrice: decimal.Zero,
		Margin:           decimal.RequireFromString("0.5"),
		Leverage:         10,
		Status:           "open",
	}).Error; err != nil {
		t.Fatalf("create position: %v", err)
	}

	_, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "blocked-open",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		MarginMode:    "cross",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      10,
	})
	if err == nil {
		t.Fatal("expected reduce-only rejection")
	}
	if err != apperr.ErrReduceOnlyMode {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOrderService_TestModeAllowsHighLeverage(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	if _, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "test-mode-1000x",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      1000,
		TestMode:      true,
	}); err != nil {
		t.Fatalf("expected test mode order to pass, got %v", err)
	}
}

func TestOrderService_HighLeverageRejectedWithoutTestMode(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	_, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "normal-1000x",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      1000,
	})
	if err == nil {
		t.Fatal("expected invalid leverage error")
	}
	if err != apperr.ErrInvalidLeverage {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOrderService_OppositeSideOrderCreatesSeparatePosition(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	if _, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "open-long",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      10,
	}); err != nil {
		t.Fatalf("open long failed: %v", err)
	}

	result, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "open-short-separately",
		Symbol:        "BTC-PERP",
		Side:          "short",
		Type:          "market",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      10,
	})
	if err != nil {
		t.Fatalf("expected opposite-side order to open independently, got %v", err)
	}
	if result.Position == nil || result.Position.Side != "short" {
		t.Fatalf("expected short position output, got %#v", result.Position)
	}

	var positions []model.Position
	if err := db.Where("user_id = ? AND symbol = ? AND status = ?", 1, "BTC-PERP", "open").Order("id asc").Find(&positions).Error; err != nil {
		t.Fatalf("load positions failed: %v", err)
	}
	if len(positions) != 2 {
		t.Fatalf("expected 2 open positions, got %d", len(positions))
	}
	if positions[0].Side != "long" || positions[1].Side != "short" {
		t.Fatalf("expected long + short positions, got %#v", positions)
	}
}

func TestOrderService_ReduceOnlyMustUseOppositeSide(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	if _, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "open-long",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      10,
	}); err != nil {
		t.Fatalf("open long failed: %v", err)
	}

	_, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "same-side-reduce-only",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      10,
		ReduceOnly:    true,
	})
	if err == nil {
		t.Fatal("expected same-side reduce-only to be rejected")
	}
	if err != apperr.ErrNoOpenPosition {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOrderService_ReduceOnlyCannotExceedOppositeSidePosition(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	if _, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "open-long",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      10,
	}); err != nil {
		t.Fatalf("open long failed: %v", err)
	}

	_, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "oversized-reduce-only",
		Symbol:        "BTC-PERP",
		Side:          "short",
		Type:          "market",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.002"),
		Leverage:      10,
		ReduceOnly:    true,
	})
	if err == nil {
		t.Fatal("expected oversized reduce-only order to be rejected")
	}
	if err != apperr.ErrOppositePositionMode {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOrderService_ReduceOnlyClosesOppositeSideWhenBothDirectionsExist(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	if _, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "open-long",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      10,
	}); err != nil {
		t.Fatalf("open long failed: %v", err)
	}

	if _, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "open-short",
		Symbol:        "BTC-PERP",
		Side:          "short",
		Type:          "market",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      10,
	}); err != nil {
		t.Fatalf("open short failed: %v", err)
	}

	result, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "close-long-via-short-reduce",
		Symbol:        "BTC-PERP",
		Side:          "short",
		Type:          "market",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      10,
		ReduceOnly:    true,
	})
	if err != nil {
		t.Fatalf("reduce-only close failed: %v", err)
	}
	if result.Position != nil && result.Position.Side != "long" {
		t.Fatalf("expected closed target to be long or nil output, got %#v", result.Position)
	}

	var positions []model.Position
	if err := db.Where("user_id = ? AND symbol = ? AND status = ?", 1, "BTC-PERP", "open").Order("id asc").Find(&positions).Error; err != nil {
		t.Fatalf("load positions failed: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("expected 1 open position after closing long, got %d", len(positions))
	}
	if positions[0].Side != "short" {
		t.Fatalf("expected remaining short position, got %#v", positions[0])
	}
}

func TestEffectiveMaintenanceRate_NormalLeverageUnchanged(t *testing.T) {
	initialRate := decimal.RequireFromString("0.04")
	maintenanceRate := decimal.RequireFromString("0.01")

	got := effectiveMaintenanceRate(initialRate, maintenanceRate, 10)
	if !got.Equal(maintenanceRate) {
		t.Fatalf("expected maintenance rate %s, got %s", maintenanceRate, got)
	}
}

func TestEffectiveMaintenanceRate_TestLeverageScalesDown(t *testing.T) {
	initialRate := decimal.RequireFromString("0.04")
	maintenanceRate := decimal.RequireFromString("0.01")

	got := effectiveMaintenanceRate(initialRate, maintenanceRate, 1000)
	want := decimal.RequireFromString("0.0005")
	if !got.Equal(want) {
		t.Fatalf("expected maintenance rate %s, got %s", want, got)
	}
}

func TestCalculateLiquidationPrice_LongTestLeverageStaysBelowEntry(t *testing.T) {
	entry := decimal.RequireFromString("89.46")
	size := decimal.RequireFromString("0.2")
	margin := decimal.RequireFromString("0.017892")
	maintenanceRate := effectiveMaintenanceRate(
		decimal.RequireFromString("0.04"),
		decimal.RequireFromString("0.01"),
		1000,
	)

	liq := calculateLiquidationPrice("long", "isolated", entry, size, margin, maintenanceRate)
	if !liq.LessThan(entry) {
		t.Fatalf("expected liquidation price below entry, entry=%s liq=%s", entry, liq)
	}
}

func TestOrderService_CreateHedgeTaskUsesLatestTargetWhenSnapshotLags(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	if err := db.Create(&model.HedgeTask{
		Symbol:               "BTC-PERP",
		TriggerType:          "trade",
		InternalNetPosition:  decimal.RequireFromString("0.001"),
		TargetHedgePosition:  decimal.RequireFromString("0.001"),
		CurrentHedgePosition: decimal.RequireFromString("0.001"),
		Drift:                decimal.Zero,
		Status:               "completed",
	}).Error; err != nil {
		t.Fatalf("create hedge task: %v", err)
	}
	if err := db.Create(&model.SystemRiskSnapshot{
		Symbol:                "BTC-PERP",
		TotalLongPosition:     decimal.Zero,
		TotalShortPosition:    decimal.Zero,
		NetPosition:           decimal.Zero,
		ExternalHedgePosition: decimal.Zero,
		Drift:                 decimal.Zero,
		HedgeHealthy:          true,
		TotalOpenInterest:     decimal.Zero,
	}).Error; err != nil {
		t.Fatalf("create risk snapshot: %v", err)
	}

	if err := svc.createMockHedgeTask(db, "BTC-PERP", "liquidation", decimal.RequireFromString("86000")); err != nil {
		t.Fatalf("create mock hedge task: %v", err)
	}

	var task model.HedgeTask
	if err := db.Order("id desc").First(&task).Error; err != nil {
		t.Fatalf("load latest task: %v", err)
	}
	if task.TriggerType != "liquidation" {
		t.Fatalf("unexpected trigger type: %s", task.TriggerType)
	}
	if task.Status != "pending" {
		t.Fatalf("expected pending hedge task, got %s", task.Status)
	}
	if !task.Drift.Equal(decimal.RequireFromString("-0.001")) {
		t.Fatalf("unexpected drift: %s", task.Drift.String())
	}
}

func TestOrderService_CreateHedgeTaskSupersedesOlderBufferedTask(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	oldTask := model.HedgeTask{
		Symbol:               "SOL-PERP",
		TriggerType:          "trade",
		InternalNetPosition:  decimal.RequireFromString("0.1"),
		TargetHedgePosition:  decimal.RequireFromString("0.1"),
		CurrentHedgePosition: decimal.Zero,
		Drift:                decimal.RequireFromString("0.1"),
		Status:               "buffered",
		ErrorMessage:         "buffered until hedge notional reaches 10 USDC",
	}
	if err := db.Create(&oldTask).Error; err != nil {
		t.Fatalf("create old task: %v", err)
	}
	oldOrder := model.HedgeOrder{
		HedgeTaskID: oldTask.ID,
		Symbol:      "SOL-PERP",
		Side:        "long",
		Size:        decimal.RequireFromString("0.1"),
		Price:       decimal.RequireFromString("89"),
		Status:      "buffered",
	}
	if err := db.Create(&oldOrder).Error; err != nil {
		t.Fatalf("create old order: %v", err)
	}
	if err := db.Create(&model.PriceTick{
		Symbol:     "SOL-PERP",
		IndexPrice: decimal.RequireFromString("100"),
		MarkPrice:  decimal.RequireFromString("100"),
		BestBid:    decimal.RequireFromString("99.9"),
		BestAsk:    decimal.RequireFromString("100.1"),
		Source:     "mock",
	}).Error; err != nil {
		t.Fatalf("create tick: %v", err)
	}
	if err := db.Create(&model.Position{
		UserID:           1,
		Symbol:           "SOL-PERP",
		Side:             "long",
		MarginMode:       "isolated",
		Size:             decimal.RequireFromString("0.2"),
		EntryPrice:       decimal.RequireFromString("100"),
		MarkPrice:        decimal.RequireFromString("100"),
		LiquidationPrice: decimal.Zero,
		Margin:           decimal.RequireFromString("1"),
		Leverage:         10,
		Status:           "open",
	}).Error; err != nil {
		t.Fatalf("create position: %v", err)
	}

	if err := svc.createMockHedgeTask(db, "SOL-PERP", "trade", decimal.RequireFromString("100")); err != nil {
		t.Fatalf("create new hedge task: %v", err)
	}

	var reloadedTask model.HedgeTask
	if err := db.First(&reloadedTask, oldTask.ID).Error; err != nil {
		t.Fatalf("reload old task: %v", err)
	}
	if reloadedTask.Status != "superseded" {
		t.Fatalf("expected old task to be superseded, got %s", reloadedTask.Status)
	}

	var reloadedOrder model.HedgeOrder
	if err := db.First(&reloadedOrder, oldOrder.ID).Error; err != nil {
		t.Fatalf("reload old order: %v", err)
	}
	if reloadedOrder.Status != "superseded" {
		t.Fatalf("expected old order to be superseded, got %s", reloadedOrder.Status)
	}
}

func TestOrderService_RejectsCustomMarginBelowMinimum(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	_, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "low-custom-margin",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      10,
		Margin:        decimal.RequireFromString("0.1"),
	})
	if err == nil {
		t.Fatal("expected insufficient margin rejection")
	}
	if err != apperr.ErrInsufficientMargin {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOrderService_AddingPositionRecomputesEffectiveLeverage(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	if _, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "base-position",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		Size:          decimal.RequireFromString("0.01"),
		Leverage:      10,
	}); err != nil {
		t.Fatalf("open base position failed: %v", err)
	}

	if _, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "small-high-leverage-add",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      50,
	}); err != nil {
		t.Fatalf("add position failed: %v", err)
	}

	var pos model.Position
	if err := db.Where("user_id = ? AND symbol = ? AND status = ?", 1, "BTC-PERP", "open").First(&pos).Error; err != nil {
		t.Fatalf("load position failed: %v", err)
	}
	if pos.Leverage != 11 {
		t.Fatalf("expected effective leverage 11 after add, got %d", pos.Leverage)
	}
}

func TestOrderService_RejectsAggregatePositionNotionalExceedingLimit(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	if err := db.Model(&model.Symbol{}).
		Where("name = ?", "BTC-PERP").
		Update("max_position_notional", decimal.RequireFromString("900")).Error; err != nil {
		t.Fatalf("update max position notional failed: %v", err)
	}

	if _, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "limit-first-open",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		Size:          decimal.RequireFromString("0.01"),
		Leverage:      10,
	}); err != nil {
		t.Fatalf("first order failed: %v", err)
	}

	_, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "limit-second-open",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      10,
	})
	if err == nil {
		t.Fatal("expected aggregate max position rejection")
	}
	if err != apperr.ErrMaxPositionExceeded {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRiskState_UsesActualLockedMarginAsTotalInitial(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	if _, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "custom-margin-open",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      10,
		Margin:        decimal.RequireFromString("10"),
	}); err != nil {
		t.Fatalf("custom margin order failed: %v", err)
	}

	riskState, err := buildRiskState(db, 1)
	if err != nil {
		t.Fatalf("build risk state failed: %v", err)
	}
	if !riskState.TotalInitial.Equal(decimal.RequireFromString("10")) {
		t.Fatalf("expected total initial 10, got %s", riskState.TotalInitial)
	}
	if !riskState.TotalInitial.Equal(riskState.LockedBalance) {
		t.Fatalf("expected total initial %s to match locked balance %s", riskState.TotalInitial, riskState.LockedBalance)
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

package service

import (
	"testing"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/AboAuther/RGPerp/backend/internal/model"
)

func TestOrderService_CreateLimitOrderStoresOpenOrder(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	result, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "limit-open-long",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "limit",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.001"),
		LimitPrice:    decimal.RequireFromString("85000"),
		TimeInForce:   "gtc",
		Leverage:      10,
	})
	if err != nil {
		t.Fatalf("create limit order failed: %v", err)
	}
	if result.Order.Status != "open" {
		t.Fatalf("expected open limit order, got %s", result.Order.Status)
	}
	if !result.Order.ExecPrice.Equal(decimal.Zero) {
		t.Fatalf("expected zero exec price before trigger, got %s", result.Order.ExecPrice)
	}

	var openPositions int64
	if err := db.Model(&model.Position{}).Where("user_id = ? AND status = ?", 1, "open").Count(&openPositions).Error; err != nil {
		t.Fatalf("count positions failed: %v", err)
	}
	if openPositions != 0 {
		t.Fatalf("expected no open positions before trigger, got %d", openPositions)
	}

	var account model.Account
	if err := db.Where("user_id = ?", 1).First(&account).Error; err != nil {
		t.Fatalf("load account failed: %v", err)
	}
	if !account.LockedBalance.Equal(result.Order.ReservedMargin) {
		t.Fatalf("expected locked balance %s to equal reserved margin %s", account.LockedBalance, result.Order.ReservedMargin)
	}
	if !account.AvailableBalance.Equal(decimal.RequireFromString("500").Sub(result.Order.ReservedMargin).Sub(result.Order.ReservedFee)) {
		t.Fatalf("unexpected available balance after reserve: %s", account.AvailableBalance)
	}
}

func TestLimitMatcher_TriggersAndExecutesOpenOrder(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)
	matcher := NewLimitMatcherService(db, zap.NewNop())

	result, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "limit-trigger-long",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "limit",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.001"),
		LimitPrice:    decimal.RequireFromString("85000"),
		TimeInForce:   "gtc",
		Leverage:      10,
	})
	if err != nil {
		t.Fatalf("create limit order failed: %v", err)
	}

	if err := db.Create(&model.PriceTick{
		Symbol:     "BTC-PERP",
		IndexPrice: decimal.RequireFromString("84990"),
		MarkPrice:  decimal.RequireFromString("84995"),
		BestBid:    decimal.RequireFromString("84994"),
		BestAsk:    decimal.RequireFromString("85000"),
		Source:     "test",
	}).Error; err != nil {
		t.Fatalf("insert trigger tick failed: %v", err)
	}

	if err := matcher.processOrder(result.Order.ID); err != nil {
		t.Fatalf("process limit order failed: %v", err)
	}

	var parent model.Order
	if err := db.First(&parent, result.Order.ID).Error; err != nil {
		t.Fatalf("load parent order failed: %v", err)
	}
	if parent.Status != "filled" {
		t.Fatalf("expected parent order filled, got %s", parent.Status)
	}
	if parent.TriggeredAt == nil {
		t.Fatal("expected triggered_at to be set")
	}

	var child model.Order
	if err := db.Where("parent_order_id = ?", parent.ID).First(&child).Error; err != nil {
		t.Fatalf("load child order failed: %v", err)
	}
	if child.Type != "market" || child.ExecutionSource != "matcher" {
		t.Fatalf("unexpected child order: %#v", child)
	}

	var positions []model.Position
	if err := db.Where("user_id = ? AND symbol = ? AND status = ?", 1, "BTC-PERP", "open").Find(&positions).Error; err != nil {
		t.Fatalf("load positions failed: %v", err)
	}
	if len(positions) != 1 || positions[0].Side != "long" {
		t.Fatalf("expected one long position after trigger, got %#v", positions)
	}

	var account model.Account
	if err := db.Where("user_id = ?", 1).First(&account).Error; err != nil {
		t.Fatalf("load account failed: %v", err)
	}
	if !account.LockedBalance.Equal(child.Margin) {
		t.Fatalf("expected locked balance %s to equal child margin %s", account.LockedBalance, child.Margin)
	}
	if !parent.ReservedMargin.Equal(decimal.Zero) || !parent.ReservedFee.Equal(decimal.Zero) {
		t.Fatalf("expected parent reservation to be cleared, got margin=%s fee=%s", parent.ReservedMargin, parent.ReservedFee)
	}
}

func TestOrderService_CancelLimitOrderReleasesReservation(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)

	result, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "limit-cancel-long",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "limit",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.001"),
		LimitPrice:    decimal.RequireFromString("85000"),
		TimeInForce:   "gtc",
		Leverage:      10,
	})
	if err != nil {
		t.Fatalf("create limit order failed: %v", err)
	}

	if _, err := svc.CancelOrder(1, result.Order.ID); err != nil {
		t.Fatalf("cancel limit order failed: %v", err)
	}

	var account model.Account
	if err := db.Where("user_id = ?", 1).First(&account).Error; err != nil {
		t.Fatalf("load account failed: %v", err)
	}
	if !account.AvailableBalance.Equal(decimal.RequireFromString("500")) {
		t.Fatalf("expected full refund to available balance, got %s", account.AvailableBalance)
	}
	if !account.LockedBalance.Equal(decimal.Zero) {
		t.Fatalf("expected zero locked balance after cancel, got %s", account.LockedBalance)
	}

	var order model.Order
	if err := db.First(&order, result.Order.ID).Error; err != nil {
		t.Fatalf("load order failed: %v", err)
	}
	if order.Status != "canceled" {
		t.Fatalf("expected canceled order, got %s", order.Status)
	}
	if !order.ReservedMargin.Equal(decimal.Zero) || !order.ReservedFee.Equal(decimal.Zero) {
		t.Fatalf("expected reservations cleared, got margin=%s fee=%s", order.ReservedMargin, order.ReservedFee)
	}
}

func TestLimitMatcher_TriggersReduceOnlyLimitWithoutReservation(t *testing.T) {
	db := mustNewOrderTestDB(t)
	svc := NewOrderService(db)
	matcher := NewLimitMatcherService(db, zap.NewNop())

	if _, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "open-long-before-reduce-limit",
		Symbol:        "BTC-PERP",
		Side:          "long",
		Type:          "market",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.001"),
		Leverage:      10,
	}); err != nil {
		t.Fatalf("open base position failed: %v", err)
	}

	result, err := svc.Create(CreateOrderInput{
		UserID:        1,
		ClientOrderID: "reduce-limit-short",
		Symbol:        "BTC-PERP",
		Side:          "short",
		Type:          "limit",
		MarginMode:    "isolated",
		Size:          decimal.RequireFromString("0.001"),
		LimitPrice:    decimal.RequireFromString("86000"),
		TimeInForce:   "gtc",
		Leverage:      10,
		ReduceOnly:    true,
	})
	if err != nil {
		t.Fatalf("create reduce-only limit failed: %v", err)
	}
	if !result.Order.ReservedMargin.Equal(decimal.Zero) || !result.Order.ReservedFee.Equal(decimal.Zero) {
		t.Fatalf("expected no reservation for reduce-only limit, got margin=%s fee=%s", result.Order.ReservedMargin, result.Order.ReservedFee)
	}

	if err := db.Create(&model.PriceTick{
		Symbol:     "BTC-PERP",
		IndexPrice: decimal.RequireFromString("86000"),
		MarkPrice:  decimal.RequireFromString("86000"),
		BestBid:    decimal.RequireFromString("86000"),
		BestAsk:    decimal.RequireFromString("86002"),
		Source:     "test",
	}).Error; err != nil {
		t.Fatalf("insert trigger tick failed: %v", err)
	}

	if err := matcher.processOrder(result.Order.ID); err != nil {
		t.Fatalf("process reduce-only limit failed: %v", err)
	}

	var positions int64
	if err := db.Model(&model.Position{}).Where("user_id = ? AND status = ?", 1, "open").Count(&positions).Error; err != nil {
		t.Fatalf("count positions failed: %v", err)
	}
	if positions != 0 {
		t.Fatalf("expected position to be closed by reduce-only limit, got %d open positions", positions)
	}

	var parent model.Order
	if err := db.First(&parent, result.Order.ID).Error; err != nil {
		t.Fatalf("load parent order failed: %v", err)
	}
	if parent.Status != "filled" {
		t.Fatalf("expected parent order filled, got %s", parent.Status)
	}
}

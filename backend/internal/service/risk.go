package service

import (
	"strings"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/AboAuther/RGPerp/backend/internal/model"
)

type RiskState struct {
	User                model.User
	Account             model.Account
	PendingWithdrawal   decimal.Decimal
	OpenOrderReserved   decimal.Decimal
	AvailableBalance    decimal.Decimal
	LockedBalance       decimal.Decimal
	UnrealizedPnL       decimal.Decimal
	Equity              decimal.Decimal
	TotalInitial        decimal.Decimal
	TotalMaintenance    decimal.Decimal
	FreeCollateral      decimal.Decimal
	WithdrawableBalance decimal.Decimal
	MarginRatio         decimal.Decimal
	RiskLevel           string
}

func buildRiskState(db *gorm.DB, userID uint64) (*RiskState, error) {
	var user model.User
	if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}

	var account model.Account
	if err := db.Where("user_id = ?", userID).First(&account).Error; err != nil {
		return nil, err
	}

	pending, err := pendingWithdrawalAmount(db, userID)
	if err != nil {
		return nil, err
	}
	openOrderReserved, err := pendingOpenOrderReserved(db, userID)
	if err != nil {
		return nil, err
	}

	var positions []model.Position
	if err := db.Where("user_id = ? AND status = ?", userID, "open").Find(&positions).Error; err != nil {
		return nil, err
	}

	symbolMap, err := loadRiskSymbols(db, positions)
	if err != nil {
		return nil, err
	}
	markPrices, err := loadLatestMarkPrices(db, positions)
	if err != nil {
		return nil, err
	}

	unrealized := decimal.Zero
	totalInitial := decimal.Zero
	totalMaintenance := decimal.Zero
	for _, pos := range positions {
		markPrice := latestMarkForSymbol(markPrices, pos.Symbol, pos.MarkPrice)
		unrealized = unrealized.Add(calculatePnL(pos.Side, pos.EntryPrice, markPrice, pos.Size))
		maintenanceRate := decimal.Zero
		if sym, ok := symbolMap[pos.Symbol]; ok {
			maintenanceRate = effectiveMaintenanceRate(sym.InitialMarginRate, sym.MaintenanceMarginRate, pos.Leverage)
		}
		notional := markPrice.Mul(pos.Size)
		initialMargin := pos.Margin.Round(18)
		totalInitial = totalInitial.Add(initialMargin)
		totalMaintenance = totalMaintenance.Add(notional.Mul(maintenanceRate))
	}

	equity := account.AvailableBalance.Add(account.LockedBalance).Add(unrealized).Round(18)
	freeCollateral := equity.Sub(totalInitial).Sub(pending).Sub(openOrderReserved)
	marginRatio := decimal.Zero
	if equity.GreaterThan(decimal.Zero) {
		marginRatio = totalMaintenance.Div(equity).Mul(decimal.NewFromInt(100)).Round(6)
	}

	riskBuffer := equity.Sub(totalMaintenance).Sub(pending).Sub(openOrderReserved)
	maxWithdrawByAvailable := account.AvailableBalance.Sub(pending)
	withdrawable := decimal.Min(maxWithdrawByAvailable, riskBuffer)
	if withdrawable.IsNegative() {
		withdrawable = decimal.Zero
	}

	riskLevel := "normal"
	switch {
	case user.Status == "frozen":
		riskLevel = "frozen"
	case len(positions) > 0 && (equity.LessThanOrEqual(totalMaintenance) || marginRatio.GreaterThanOrEqual(decimal.NewFromInt(100))):
		riskLevel = "liquidating"
	case len(positions) > 0 && (marginRatio.GreaterThanOrEqual(decimal.NewFromInt(80)) || freeCollateral.LessThanOrEqual(decimal.Zero)):
		riskLevel = "reduce_only"
	case len(positions) > 0 && marginRatio.GreaterThanOrEqual(decimal.NewFromInt(60)):
		riskLevel = "at_risk"
	}

	return &RiskState{
		User:                user,
		Account:             account,
		PendingWithdrawal:   pending,
		OpenOrderReserved:   openOrderReserved.Round(18),
		AvailableBalance:    account.AvailableBalance,
		LockedBalance:       account.LockedBalance,
		UnrealizedPnL:       unrealized.Round(18),
		Equity:              equity,
		TotalInitial:        totalInitial.Round(18),
		TotalMaintenance:    totalMaintenance.Round(18),
		FreeCollateral:      freeCollateral.Round(18),
		WithdrawableBalance: withdrawable.Round(18),
		MarginRatio:         marginRatio,
		RiskLevel:           riskLevel,
	}, nil
}

func pendingOpenOrderReserved(db *gorm.DB, userID uint64) (decimal.Decimal, error) {
	var orders []model.Order
	if err := db.Where(
		"user_id = ? AND type = ? AND parent_order_id IS NULL AND reduce_only = ? AND status IN ?",
		userID,
		"limit",
		false,
		[]string{"open", "triggered"},
	).Find(&orders).Error; err != nil {
		return decimal.Zero, err
	}

	total := decimal.Zero
	for _, order := range orders {
		total = total.Add(order.Margin).Add(order.Fee)
	}
	return total, nil
}

func syncUserRiskStatus(db *gorm.DB, userID uint64) (*RiskState, error) {
	return syncUserRiskStatusTx(db, userID)
}

func syncUserRiskStatusTx(tx *gorm.DB, userID uint64) (*RiskState, error) {
	riskState, err := buildRiskState(tx, userID)
	if err != nil {
		return nil, err
	}

	targetStatus := riskState.RiskLevel
	if targetStatus == "normal" {
		targetStatus = "active"
	}
	if riskState.User.Status == "frozen" {
		targetStatus = "frozen"
	}

	if riskState.User.Status != targetStatus {
		if err := tx.Model(&model.User{}).Where("id = ?", userID).Update("status", targetStatus).Error; err != nil {
			return nil, err
		}
		riskState.User.Status = targetStatus
	}
	return riskState, nil
}

func orderIncreasesExposure(existing *model.Position, side string, size decimal.Decimal, reduceOnly bool) bool {
	if reduceOnly {
		return false
	}
	if existing == nil {
		return true
	}
	if existing.Side == side {
		return true
	}
	return size.GreaterThan(existing.Size)
}

func loadRiskSymbols(db *gorm.DB, positions []model.Position) (map[string]model.Symbol, error) {
	if len(positions) == 0 {
		return map[string]model.Symbol{}, nil
	}
	names := make([]string, 0, len(positions))
	seen := make(map[string]struct{}, len(positions))
	for _, pos := range positions {
		name := strings.ToUpper(strings.TrimSpace(pos.Symbol))
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	var symbols []model.Symbol
	if err := db.Where("name IN ?", names).Find(&symbols).Error; err != nil {
		return nil, err
	}
	out := make(map[string]model.Symbol, len(symbols))
	for _, sym := range symbols {
		out[sym.Name] = sym
	}
	return out, nil
}

func loadLatestMarkPrices(db *gorm.DB, positions []model.Position) (map[string]decimal.Decimal, error) {
	if len(positions) == 0 {
		return map[string]decimal.Decimal{}, nil
	}

	names := make([]string, 0, len(positions))
	seen := make(map[string]struct{}, len(positions))
	for _, pos := range positions {
		name := strings.ToUpper(strings.TrimSpace(pos.Symbol))
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}

	var ticks []model.PriceTick
	if err := db.Where("symbol IN ?", names).Order("symbol asc, created_at desc, id desc").Find(&ticks).Error; err != nil {
		return nil, err
	}

	out := make(map[string]decimal.Decimal, len(names))
	for _, tick := range ticks {
		if _, ok := out[tick.Symbol]; ok {
			continue
		}
		out[tick.Symbol] = tick.MarkPrice
	}
	return out, nil
}

func latestMarkForSymbol(markPrices map[string]decimal.Decimal, symbol string, fallback decimal.Decimal) decimal.Decimal {
	if price, ok := markPrices[strings.ToUpper(strings.TrimSpace(symbol))]; ok && price.GreaterThan(decimal.Zero) {
		return price
	}
	return fallback
}

package service

import (
	"strings"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/AboAuther/RGPerp/backend/internal/model"
)

type RiskState struct {
	Account             model.Account
	PendingWithdrawal   decimal.Decimal
	AvailableBalance    decimal.Decimal
	LockedBalance       decimal.Decimal
	UnrealizedPnL       decimal.Decimal
	Equity              decimal.Decimal
	TotalMaintenance    decimal.Decimal
	WithdrawableBalance decimal.Decimal
	MarginRatio         decimal.Decimal
	RiskLevel           string
}

func buildRiskState(db *gorm.DB, userID uint64) (*RiskState, error) {
	var account model.Account
	if err := db.Where("user_id = ?", userID).First(&account).Error; err != nil {
		return nil, err
	}

	pending, err := pendingWithdrawalAmount(db, userID)
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

	unrealized := decimal.Zero
	totalMaintenance := decimal.Zero
	for _, pos := range positions {
		markPrice := pos.MarkPrice
		unrealized = unrealized.Add(calculatePnL(pos.Side, pos.EntryPrice, markPrice, pos.Size))
		maintenanceRate := decimal.Zero
		if sym, ok := symbolMap[pos.Symbol]; ok {
			maintenanceRate = sym.MaintenanceMarginRate
		}
		totalMaintenance = totalMaintenance.Add(markPrice.Mul(pos.Size).Mul(maintenanceRate))
	}

	equity := account.AvailableBalance.Add(account.LockedBalance).Add(unrealized).Round(18)
	marginRatio := decimal.Zero
	if equity.GreaterThan(decimal.Zero) {
		marginRatio = totalMaintenance.Div(equity).Mul(decimal.NewFromInt(100)).Round(6)
	}

	riskBuffer := equity.Sub(totalMaintenance).Sub(pending)
	maxWithdrawByAvailable := account.AvailableBalance.Sub(pending)
	withdrawable := decimal.Min(maxWithdrawByAvailable, riskBuffer)
	if withdrawable.IsNegative() {
		withdrawable = decimal.Zero
	}

	riskLevel := "normal"
	switch {
	case equity.LessThanOrEqual(totalMaintenance) || marginRatio.GreaterThanOrEqual(decimal.NewFromInt(100)):
		riskLevel = "danger"
	case marginRatio.GreaterThanOrEqual(decimal.NewFromInt(80)):
		riskLevel = "warning"
	}

	return &RiskState{
		Account:             account,
		PendingWithdrawal:   pending,
		AvailableBalance:    account.AvailableBalance,
		LockedBalance:       account.LockedBalance,
		UnrealizedPnL:       unrealized.Round(18),
		Equity:              equity,
		TotalMaintenance:    totalMaintenance.Round(18),
		WithdrawableBalance: withdrawable.Round(18),
		MarginRatio:         marginRatio,
		RiskLevel:           riskLevel,
	}, nil
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

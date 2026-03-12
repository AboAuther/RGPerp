package service

import (
	"time"

	"github.com/AboAuther/RGPerp/backend/internal/model"
	apperr "github.com/AboAuther/RGPerp/backend/internal/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type AccountService struct {
	db *gorm.DB
}

func NewAccountService(db *gorm.DB) *AccountService {
	return &AccountService{db: db}
}

type AccountOutput struct {
	Asset               string          `json:"asset"`
	AvailableBalance    decimal.Decimal `json:"available_balance"`
	LockedBalance       decimal.Decimal `json:"locked_balance"`
	PendingWithdrawal   decimal.Decimal `json:"pending_withdrawal"`
	WithdrawableBalance decimal.Decimal `json:"withdrawable_balance"`
	UnrealizedPnL       decimal.Decimal `json:"unrealized_pnl"`
	Equity              decimal.Decimal `json:"equity"`
	MaintenanceMargin   decimal.Decimal `json:"maintenance_margin"`
	MarginRatio         decimal.Decimal `json:"margin_ratio"`
	RiskLevel           string          `json:"risk_level"`
}

type DepositRecordOutput struct {
	TxHash      string          `json:"tx_hash"`
	LogIndex    uint64          `json:"log_index"`
	BlockNumber uint64          `json:"block_number"`
	Amount      decimal.Decimal `json:"amount"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
}

func (s *AccountService) GetAccount(userID uint64) (*AccountOutput, error) {
	if err := expireDueWithdrawals(s.db, userID); err != nil {
		return nil, err
	}

	riskState, err := syncUserRiskStatus(s.db, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrAccountNotFound
		}
		return nil, err
	}

	return &AccountOutput{
		Asset:               "USDC",
		AvailableBalance:    riskState.AvailableBalance,
		LockedBalance:       riskState.LockedBalance,
		PendingWithdrawal:   riskState.PendingWithdrawal,
		WithdrawableBalance: riskState.WithdrawableBalance,
		UnrealizedPnL:       riskState.UnrealizedPnL,
		Equity:              riskState.Equity,
		MaintenanceMargin:   riskState.TotalMaintenance,
		MarginRatio:         riskState.MarginRatio,
		RiskLevel:           riskState.RiskLevel,
	}, nil
}

func (s *AccountService) ListDeposits(userID uint64) ([]DepositRecordOutput, error) {
	var user model.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrUnauthorized
		}
		return nil, err
	}

	var events []model.VaultEvent
	if err := s.db.Where("user_address = ? AND event_type = ?", user.WalletAddress, "deposit").
		Order("created_at desc").
		Limit(200).
		Find(&events).Error; err != nil {
		return nil, err
	}

	items := make([]DepositRecordOutput, 0, len(events))
	for _, event := range events {
		items = append(items, DepositRecordOutput{
			TxHash:      event.TxHash,
			LogIndex:    event.LogIndex,
			BlockNumber: event.BlockNumber,
			Amount:      event.Amount,
			Status:      event.Status,
			CreatedAt:   event.CreatedAt,
		})
	}
	return items, nil
}

func pendingWithdrawalAmount(db *gorm.DB, userID uint64) (decimal.Decimal, error) {
	var requests []model.WithdrawalRequest
	if err := db.Where("user_id = ? AND status IN ?", userID, []string{"signed", "submitted"}).Find(&requests).Error; err != nil {
		return decimal.Zero, err
	}

	total := decimal.Zero
	for _, item := range requests {
		total = total.Add(item.Amount)
	}
	return total, nil
}

func expireDueWithdrawals(db *gorm.DB, userID uint64) error {
	return db.Model(&model.WithdrawalRequest{}).
		Where("user_id = ? AND status = ? AND deadline < ?", userID, "signed", time.Now().UTC()).
		Update("status", "expired").Error
}

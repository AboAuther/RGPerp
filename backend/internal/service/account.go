package service

import (
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
	Equity              decimal.Decimal `json:"equity"`
}

func (s *AccountService) GetAccount(userID uint64) (*AccountOutput, error) {
	var account model.Account
	if err := s.db.Where("user_id = ?", userID).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrAccountNotFound
		}
		return nil, err
	}

	pending, err := pendingWithdrawalAmount(s.db, userID)
	if err != nil {
		return nil, err
	}

	withdrawable := account.AvailableBalance.Sub(pending)
	if withdrawable.IsNegative() {
		withdrawable = decimal.Zero
	}

	return &AccountOutput{
		Asset:               "USDC",
		AvailableBalance:    account.AvailableBalance,
		LockedBalance:       account.LockedBalance,
		PendingWithdrawal:   pending,
		WithdrawableBalance: withdrawable,
		Equity:              account.AvailableBalance.Add(account.LockedBalance),
	}, nil
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

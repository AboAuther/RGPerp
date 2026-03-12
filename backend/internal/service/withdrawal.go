package service

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/AboAuther/RGPerp/backend/internal/config"
	"github.com/AboAuther/RGPerp/backend/internal/model"
	apperr "github.com/AboAuther/RGPerp/backend/internal/pkg/errors"
	"github.com/AboAuther/RGPerp/backend/internal/pkg/web3"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type WithdrawalService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewWithdrawalService(db *gorm.DB, cfg *config.Config) *WithdrawalService {
	return &WithdrawalService{db: db, cfg: cfg}
}

type CreateWithdrawalInput struct {
	UserID         uint64
	WalletAddress  string
	Amount         decimal.Decimal
	IdempotencyKey string
}

type WithdrawalOutput struct {
	RequestID    string          `json:"request_id"`
	Asset        string          `json:"asset"`
	Amount       decimal.Decimal `json:"amount"`
	Nonce        uint64          `json:"nonce"`
	Deadline     time.Time       `json:"deadline"`
	Signature    string          `json:"signature"`
	VaultAddress string          `json:"vault_address"`
	ChainID      int64           `json:"chain_id"`
	Status       string          `json:"status"`
}

func (s *WithdrawalService) Create(input CreateWithdrawalInput) (*WithdrawalOutput, error) {
	if err := expireDueWithdrawals(s.db, input.UserID); err != nil {
		return nil, err
	}

	if input.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, apperr.ErrMinWithdrawalAmount
	}

	var account model.Account
	if err := s.db.Where("user_id = ?", input.UserID).First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrAccountNotFound
		}
		return nil, err
	}

	pending, err := pendingWithdrawalAmount(s.db, input.UserID)
	if err != nil {
		return nil, err
	}
	withdrawable := account.AvailableBalance.Sub(pending)
	if withdrawable.LessThan(input.Amount) {
		return nil, apperr.ErrInsufficientBalance
	}

	if input.IdempotencyKey != "" {
		var existing model.WithdrawalRequest
		err := s.db.Where("user_id = ? AND idempotency_key = ? AND status IN ?", input.UserID, input.IdempotencyKey, []string{"signed", "submitted"}).First(&existing).Error
		if err == nil {
			return &WithdrawalOutput{
				RequestID:    existing.RequestID,
				Asset:        "USDC",
				Amount:       existing.Amount,
				Nonce:        existing.Nonce,
				Deadline:     existing.Deadline,
				Signature:    existing.Signature,
				VaultAddress: s.cfg.Blockchain.VaultAddress,
				ChainID:      s.cfg.Blockchain.ChainID,
				Status:       existing.Status,
			}, nil
		}
	}

	if s.cfg.Blockchain.OperatorPrivateKey == "" || s.cfg.Blockchain.VaultAddress == "" {
		return nil, apperr.ErrInternal
	}

	amountUnits := input.Amount.Mul(decimal.NewFromInt(1_000_000))
	if !amountUnits.Equal(amountUnits.Truncate(0)) {
		return nil, apperr.ErrBadRequest
	}

	nonce, err := randomUint64()
	if err != nil {
		return nil, err
	}
	deadline := time.Now().UTC().Add(15 * time.Minute)
	signature, err := web3.SignWithdrawal(
		s.cfg.Blockchain.OperatorPrivateKey,
		input.WalletAddress,
		amountUnits.BigInt().Uint64(),
		nonce,
		uint64(deadline.Unix()),
		s.cfg.Blockchain.ChainID,
		s.cfg.Blockchain.VaultAddress,
	)
	if err != nil {
		return nil, err
	}

	requestID := fmt.Sprintf("wd_%d", time.Now().UTC().UnixNano())
	record := &model.WithdrawalRequest{
		RequestID:      requestID,
		IdempotencyKey: input.IdempotencyKey,
		UserID:         input.UserID,
		Amount:         input.Amount,
		Status:         "signed",
		Nonce:          nonce,
		Signature:      signature,
		Deadline:       deadline,
	}
	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}

	return &WithdrawalOutput{
		RequestID:    requestID,
		Asset:        "USDC",
		Amount:       input.Amount,
		Nonce:        nonce,
		Deadline:     deadline,
		Signature:    signature,
		VaultAddress: s.cfg.Blockchain.VaultAddress,
		ChainID:      s.cfg.Blockchain.ChainID,
		Status:       record.Status,
	}, nil
}

func (s *WithdrawalService) List(userID uint64) ([]model.WithdrawalRequest, error) {
	if err := expireDueWithdrawals(s.db, userID); err != nil {
		return nil, err
	}

	var requests []model.WithdrawalRequest
	if err := s.db.Where("user_id = ?", userID).Order("created_at desc").Find(&requests).Error; err != nil {
		return nil, err
	}
	return requests, nil
}

func randomUint64() (uint64, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(buf[:]), nil
}

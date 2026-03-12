package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/AboAuther/RGPerp/backend/internal/config"
	"github.com/AboAuther/RGPerp/backend/internal/model"
	"github.com/AboAuther/RGPerp/backend/internal/pkg/authjwt"
	apperr "github.com/AboAuther/RGPerp/backend/internal/pkg/errors"
	"github.com/AboAuther/RGPerp/backend/internal/pkg/web3"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type AuthService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewAuthService(db *gorm.DB, cfg *config.Config) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

type ChallengeInput struct {
	WalletAddress string
	Domain        string
	ChainID       uint64
}

type ChallengeOutput struct {
	Nonce     string    `json:"nonce"`
	Message   string    `json:"message"`
	ExpiresAt time.Time `json:"expires_at"`
}

type LoginInput struct {
	WalletAddress string
	Nonce         string
	Signature     string
}

type LoginOutput struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      ginUser   `json:"user"`
}

type ginUser struct {
	ID            uint64 `json:"id"`
	WalletAddress string `json:"wallet_address"`
	Status        string `json:"status"`
}

func (s *AuthService) CreateChallenge(input ChallengeInput) (*ChallengeOutput, error) {
	walletAddress, err := web3.NormalizeAddress(input.WalletAddress)
	if err != nil {
		return nil, apperr.ErrInvalidAddress
	}

	nonce, err := randomHex(16)
	if err != nil {
		return nil, apperr.ErrInternal
	}

	domain := input.Domain
	if domain == "" {
		domain = "localhost"
	}
	chainID := input.ChainID
	if chainID == 0 {
		chainID = uint64(s.cfg.Blockchain.ChainID)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(5 * time.Minute)
	message := fmt.Sprintf(
		"Welcome to RGPerp!\n\nWallet: %s\nNonce: %s\nDomain: %s\nChain ID: %d\nTimestamp: %s",
		walletAddress,
		nonce,
		domain,
		chainID,
		now.Format(time.RFC3339),
	)

	record := &model.AuthNonce{
		WalletAddress: walletAddress,
		Nonce:         nonce,
		Message:       message,
		Domain:        domain,
		ChainID:       chainID,
		ExpiresAt:     expiresAt,
		Used:          false,
	}
	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}

	return &ChallengeOutput{
		Nonce:     nonce,
		Message:   message,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *AuthService) Login(input LoginInput) (*LoginOutput, error) {
	walletAddress, err := web3.NormalizeAddress(input.WalletAddress)
	if err != nil {
		return nil, apperr.ErrInvalidAddress
	}

	var nonce model.AuthNonce
	err = s.db.Where("nonce = ? AND wallet_address = ?", input.Nonce, walletAddress).First(&nonce).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNonceExpired
		}
		return nil, err
	}
	if nonce.Used || time.Now().UTC().After(nonce.ExpiresAt) {
		return nil, apperr.ErrNonceExpired
	}

	if err := web3.VerifyMessageSignature(nonce.Message, input.Signature, walletAddress); err != nil {
		return nil, apperr.ErrSignatureInvalid
	}

	var user model.User
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.AuthNonce{}).
			Where("id = ? AND used = ?", nonce.ID, false).
			Update("used", true).Error; err != nil {
			return err
		}

		if err := tx.Where("wallet_address = ?", walletAddress).First(&user).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			user = model.User{
				WalletAddress: walletAddress,
				Status:        "active",
			}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
		}

		var account model.Account
		if err := tx.Where("user_id = ?", user.ID).First(&account).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
			account = model.Account{
				UserID:           user.ID,
				AvailableBalance: decimal.Zero,
				LockedBalance:    decimal.Zero,
			}
			if err := tx.Create(&account).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	expiresAt := time.Now().UTC().Add(time.Duration(s.cfg.JWT.ExpireHours) * time.Hour)
	token, err := authjwt.Generate(
		s.cfg.JWT.Secret,
		user.ID,
		user.WalletAddress,
		time.Duration(s.cfg.JWT.ExpireHours)*time.Hour,
	)
	if err != nil {
		return nil, err
	}

	return &LoginOutput{
		Token:     token,
		ExpiresAt: expiresAt,
		User: ginUser{
			ID:            user.ID,
			WalletAddress: user.WalletAddress,
			Status:        user.Status,
		},
	}, nil
}

func randomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

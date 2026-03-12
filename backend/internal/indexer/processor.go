package indexer

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/AboAuther/RGPerp/backend/internal/model"
)

const vaultABI = `[
  {"anonymous":false,"inputs":[
    {"indexed":true,"internalType":"address","name":"user","type":"address"},
    {"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},
    {"indexed":false,"internalType":"uint256","name":"timestamp","type":"uint256"}
  ],"name":"Deposit","type":"event"},
  {"anonymous":false,"inputs":[
    {"indexed":true,"internalType":"address","name":"user","type":"address"},
    {"indexed":false,"internalType":"uint256","name":"amount","type":"uint256"},
    {"indexed":false,"internalType":"uint256","name":"nonce","type":"uint256"},
    {"indexed":false,"internalType":"uint256","name":"timestamp","type":"uint256"}
  ],"name":"Withdraw","type":"event"}
]`

type Processor struct {
	db            *gorm.DB
	parsedABI     abi.ABI
	depositTopic  common.Hash
	withdrawTopic common.Hash
}

func NewProcessor(db *gorm.DB) (*Processor, error) {
	parsedABI, err := abi.JSON(strings.NewReader(vaultABI))
	if err != nil {
		return nil, fmt.Errorf("parse vault abi: %w", err)
	}

	return &Processor{
		db:            db,
		parsedABI:     parsedABI,
		depositTopic:  parsedABI.Events["Deposit"].ID,
		withdrawTopic: parsedABI.Events["Withdraw"].ID,
	}, nil
}

func (p *Processor) ProcessLog(ctx context.Context, vLog gethtypes.Log) error {
	switch vLog.Topics[0] {
	case p.depositTopic:
		return p.processDeposit(ctx, vLog)
	case p.withdrawTopic:
		return p.processWithdraw(ctx, vLog)
	default:
		return nil
	}
}

func (p *Processor) processDeposit(ctx context.Context, vLog gethtypes.Log) error {
	var event struct {
		Amount    *big.Int
		Timestamp *big.Int
	}
	if err := p.parsedABI.UnpackIntoInterface(&event, "Deposit", vLog.Data); err != nil {
		return err
	}
	userAddress := common.HexToAddress(vLog.Topics[1].Hex()).Hex()
	amount := usdcUnitsToDecimal(event.Amount)

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if exists, err := vaultEventExists(tx, vLog.TxHash.Hex(), vLog.Index); err != nil || exists {
			return err
		}

		user, account, err := ensureUserAccount(tx, userAddress)
		if err != nil {
			return err
		}

		before := account.AvailableBalance
		account.AvailableBalance = account.AvailableBalance.Add(amount)
		if err := tx.Save(&account).Error; err != nil {
			return err
		}

		now := time.Now().UTC()
		if err := tx.Create(&model.VaultEvent{
			TxHash:      vLog.TxHash.Hex(),
			LogIndex:    uint64(vLog.Index),
			BlockNumber: vLog.BlockNumber,
			EventType:   "deposit",
			UserAddress: userAddress,
			Amount:      amount,
			Status:      "confirmed",
			Processed:   true,
			ProcessedAt: &now,
		}).Error; err != nil {
			return err
		}

		return tx.Create(&model.LedgerEntry{
			UserID:        user.ID,
			Type:          "deposit",
			Amount:        amount,
			BalanceBefore: before,
			BalanceAfter:  account.AvailableBalance,
			ReferenceType: "vault_event",
			Description:   "vault deposit confirmed",
		}).Error
	})
}

func (p *Processor) processWithdraw(ctx context.Context, vLog gethtypes.Log) error {
	var event struct {
		Amount    *big.Int
		Nonce     *big.Int
		Timestamp *big.Int
	}
	if err := p.parsedABI.UnpackIntoInterface(&event, "Withdraw", vLog.Data); err != nil {
		return err
	}
	userAddress := common.HexToAddress(vLog.Topics[1].Hex()).Hex()
	amount := usdcUnitsToDecimal(event.Amount)
	nonce := event.Nonce.Uint64()

	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if exists, err := vaultEventExists(tx, vLog.TxHash.Hex(), vLog.Index); err != nil || exists {
			return err
		}

		user, account, err := ensureUserAccount(tx, userAddress)
		if err != nil {
			return err
		}

		before := account.AvailableBalance
		account.AvailableBalance = account.AvailableBalance.Sub(amount)
		if err := tx.Save(&account).Error; err != nil {
			return err
		}

		now := time.Now().UTC()
		if err := tx.Create(&model.VaultEvent{
			TxHash:      vLog.TxHash.Hex(),
			LogIndex:    uint64(vLog.Index),
			BlockNumber: vLog.BlockNumber,
			EventType:   "withdraw",
			UserAddress: userAddress,
			Amount:      amount,
			Status:      "confirmed",
			Processed:   true,
			ProcessedAt: &now,
		}).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.WithdrawalRequest{}).
			Where("user_id = ? AND nonce = ?", user.ID, nonce).
			Updates(map[string]interface{}{
				"status":  "confirmed",
				"tx_hash": vLog.TxHash.Hex(),
			}).Error; err != nil {
			return err
		}

		return tx.Create(&model.LedgerEntry{
			UserID:        user.ID,
			Type:          "withdraw",
			Amount:        amount,
			BalanceBefore: before,
			BalanceAfter:  account.AvailableBalance,
			ReferenceType: "vault_event",
			Description:   "vault withdraw confirmed",
		}).Error
	})
}

func vaultEventExists(tx *gorm.DB, txHash string, logIndex uint) (bool, error) {
	var count int64
	if err := tx.Model(&model.VaultEvent{}).Where("tx_hash = ? AND log_index = ?", txHash, logIndex).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func ensureUserAccount(tx *gorm.DB, walletAddress string) (*model.User, *model.Account, error) {
	var user model.User
	if err := tx.Where("wallet_address = ?", walletAddress).First(&user).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, nil, err
		}
		user = model.User{
			WalletAddress: walletAddress,
			Status:        "active",
		}
		if err := tx.Create(&user).Error; err != nil {
			return nil, nil, err
		}
	}

	var account model.Account
	if err := tx.Where("user_id = ?", user.ID).First(&account).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, nil, err
		}
		account = model.Account{
			UserID: user.ID,
		}
		if err := tx.Create(&account).Error; err != nil {
			return nil, nil, err
		}
	}

	return &user, &account, nil
}

func usdcUnitsToDecimal(v *big.Int) decimal.Decimal {
	if v == nil {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(v, -6)
}

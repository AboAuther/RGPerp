package service

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AboAuther/RGPerp/backend/internal/model"
)

const settlementAsset = "USDC"

func ensureSettlementPool(tx *gorm.DB, asset string) (*model.SettlementPool, error) {
	var pool model.SettlementPool
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("asset = ?", asset).
		First(&pool).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			pool = model.SettlementPool{Asset: asset, Balance: decimal.Zero}
			if createErr := tx.Create(&pool).Error; createErr != nil {
				return nil, createErr
			}
			return &pool, nil
		}
		return nil, err
	}
	return &pool, nil
}

func settleRealizedDelta(tx *gorm.DB, account *model.Account, delta decimal.Decimal) error {
	if delta.IsZero() {
		return nil
	}

	pool, err := ensureSettlementPool(tx, settlementAsset)
	if err != nil {
		return err
	}

	if delta.GreaterThan(decimal.Zero) {
		account.UnsettledPnlBalance = account.UnsettledPnlBalance.Add(delta).Round(18)
		payout := decimal.Min(account.UnsettledPnlBalance, pool.Balance)
		if payout.GreaterThan(decimal.Zero) {
			account.UnsettledPnlBalance = account.UnsettledPnlBalance.Sub(payout).Round(18)
			account.SettledPnlBalance = account.SettledPnlBalance.Add(payout).Round(18)
			pool.Balance = pool.Balance.Sub(payout).Round(18)
		}
	} else {
		loss := delta.Abs()
		unsettledOffset := decimal.Min(account.UnsettledPnlBalance, loss)
		if unsettledOffset.GreaterThan(decimal.Zero) {
			account.UnsettledPnlBalance = account.UnsettledPnlBalance.Sub(unsettledOffset).Round(18)
			loss = loss.Sub(unsettledOffset)
		}

		settledOffset := decimal.Min(account.SettledPnlBalance, loss)
		if settledOffset.GreaterThan(decimal.Zero) {
			account.SettledPnlBalance = account.SettledPnlBalance.Sub(settledOffset).Round(18)
			pool.Balance = pool.Balance.Add(settledOffset).Round(18)
			loss = loss.Sub(settledOffset)
		}

		if loss.GreaterThan(decimal.Zero) {
			// Remaining loss is absorbed by the user's principal-backed trading equity.
		}
	}

	if err := tx.Save(pool).Error; err != nil {
		return err
	}
	return tx.Save(account).Error
}

func ConsumeWithdrawalBacking(account *model.Account, amount decimal.Decimal) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return
	}

	fromDeposits := decimal.Min(account.NetDeposits, amount)
	account.NetDeposits = account.NetDeposits.Sub(fromDeposits).Round(18)
	remaining := amount.Sub(fromDeposits)
	if remaining.LessThanOrEqual(decimal.Zero) {
		return
	}

	fromSettled := decimal.Min(account.SettledPnlBalance, remaining)
	account.SettledPnlBalance = account.SettledPnlBalance.Sub(fromSettled).Round(18)
}

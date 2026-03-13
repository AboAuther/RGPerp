package service

import (
	"strings"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/AboAuther/RGPerp/backend/internal/model"
)

func BackfillAccountSettlementState(db *gorm.DB) error {
	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		return err
	}

	for _, user := range users {
		if err := db.Transaction(func(tx *gorm.DB) error {
			var account model.Account
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ?", user.ID).
				First(&account).Error; err != nil {
				return nil
			}

			deposits, err := vaultEventSum(tx, strings.ToLower(user.WalletAddress), "deposit")
			if err != nil {
				return err
			}
			withdrawals, err := vaultEventSum(tx, strings.ToLower(user.WalletAddress), "withdraw")
			if err != nil {
				return err
			}

			grossNetDeposits := deposits.Sub(withdrawals)
			if grossNetDeposits.IsNegative() {
				grossNetDeposits = decimal.Zero
			}

			backedBalance := account.AvailableBalance.Add(account.LockedBalance)
			if backedBalance.IsNegative() {
				backedBalance = decimal.Zero
			}

			backedPrincipal := decimal.Min(grossNetDeposits, backedBalance)
			settledProfit := backedBalance.Sub(backedPrincipal)
			if settledProfit.IsNegative() {
				settledProfit = decimal.Zero
			}

			updates := map[string]any{
				"net_deposits":          backedPrincipal.Round(18),
				"settled_pnl_balance":   settledProfit.Round(18),
				"unsettled_pnl_balance": decimal.Zero,
			}
			return tx.Model(&model.Account{}).Where("id = ?", account.ID).Updates(updates).Error
		}); err != nil {
			return err
		}
	}

	return nil
}

func vaultEventSum(tx *gorm.DB, walletAddress, eventType string) (decimal.Decimal, error) {
	var events []model.VaultEvent
	if err := tx.Where("LOWER(user_address) = ? AND event_type = ? AND status = ?", walletAddress, eventType, "confirmed").
		Find(&events).Error; err != nil {
		return decimal.Zero, err
	}

	total := decimal.Zero
	for _, event := range events {
		total = total.Add(event.Amount)
	}
	return total.Round(18), nil
}

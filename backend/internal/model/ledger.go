package model

import "github.com/shopspring/decimal"

type LedgerEntry struct {
	BaseModel
	UserID        uint64          `gorm:"index;not null" json:"user_id"`
	Type          string          `gorm:"type:varchar(30);index;not null" json:"type"`
	Amount        decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"amount"`
	BalanceBefore decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"balance_before"`
	BalanceAfter  decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"balance_after"`
	ReferenceType string          `gorm:"type:varchar(30)" json:"reference_type"`
	ReferenceID   uint64          `gorm:"index" json:"reference_id"`
	Description   string          `gorm:"type:varchar(255)" json:"description"`
}

func (LedgerEntry) TableName() string { return "ledger_entries" }

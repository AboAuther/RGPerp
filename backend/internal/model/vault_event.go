package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type VaultEvent struct {
	BaseModel
	TxHash      string          `gorm:"type:varchar(66);not null" json:"tx_hash"`
	LogIndex    uint64          `gorm:"not null" json:"log_index"`
	BlockNumber uint64          `gorm:"index;not null" json:"block_number"`
	EventType   string          `gorm:"type:varchar(20);not null" json:"event_type"`
	UserAddress string          `gorm:"type:varchar(42);index;not null" json:"user_address"`
	Amount      decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"amount"`
	Status      string          `gorm:"type:varchar(20);not null" json:"status"`
	Processed   bool            `gorm:"default:false;not null;index" json:"processed"`
	ProcessedAt *time.Time      `json:"processed_at"`
	ErrorMsg    string          `gorm:"type:varchar(500)" json:"error_msg"`
}

func (VaultEvent) TableName() string { return "vault_events" }

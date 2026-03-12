package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type WithdrawalRequest struct {
	BaseModel
	UserID          uint64          `gorm:"index;not null" json:"user_id"`
	Amount          decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"amount"`
	Status          string          `gorm:"type:varchar(20);index;not null" json:"status"`
	TxHash          string          `gorm:"type:varchar(66)" json:"tx_hash"`
	Nonce           uint64          `gorm:"not null" json:"nonce"`
	Signature       string          `gorm:"type:varchar(132)" json:"signature"`
	Deadline        time.Time       `json:"deadline"`
	RejectionReason string          `gorm:"type:varchar(500)" json:"rejection_reason"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (WithdrawalRequest) TableName() string { return "withdrawal_requests" }

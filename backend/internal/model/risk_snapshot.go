package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type SystemRiskSnapshot struct {
	ID                    uint64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Symbol                string          `gorm:"type:varchar(20);index;not null" json:"symbol"`
	TotalLongPosition     decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"total_long_position"`
	TotalShortPosition    decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"total_short_position"`
	NetPosition           decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"net_position"`
	ExternalHedgePosition decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"external_hedge_position"`
	Drift                 decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"drift"`
	HedgeHealthy          bool            `gorm:"not null" json:"hedge_healthy"`
	TotalOpenInterest     decimal.Decimal `gorm:"type:decimal(36,18)" json:"total_open_interest"`
	CreatedAt             time.Time       `gorm:"autoCreateTime;index" json:"created_at"`
}

func (SystemRiskSnapshot) TableName() string { return "system_risk_snapshots" }

package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type FundingEvent struct {
	BaseModel
	UserID       uint64          `gorm:"index;not null" json:"user_id"`
	PositionID   uint64          `gorm:"index;not null" json:"position_id"`
	Symbol       string          `gorm:"type:varchar(20);index;not null" json:"symbol"`
	Side         string          `gorm:"type:varchar(10);not null" json:"side"`
	FundingRate  decimal.Decimal `gorm:"type:decimal(18,10);not null" json:"funding_rate"`
	MarkPrice    decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"mark_price"`
	Notional     decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"notional"`
	Amount       decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"amount"`
	SettlementAt time.Time       `gorm:"index;not null" json:"settlement_at"`

	User     User     `gorm:"foreignKey:UserID" json:"-"`
	Position Position `gorm:"foreignKey:PositionID" json:"-"`
}

func (FundingEvent) TableName() string { return "funding_events" }

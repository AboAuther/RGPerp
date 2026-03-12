package model

import "github.com/shopspring/decimal"

type Position struct {
	BaseModel
	UserID           uint64          `gorm:"index;not null" json:"user_id"`
	Symbol           string          `gorm:"type:varchar(20);index;not null" json:"symbol"`
	Side             string          `gorm:"type:varchar(10);not null" json:"side"`
	Size             decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"size"`
	EntryPrice       decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"entry_price"`
	MarkPrice        decimal.Decimal `gorm:"type:decimal(36,18)" json:"mark_price"`
	LiquidationPrice decimal.Decimal `gorm:"type:decimal(36,18)" json:"liquidation_price"`
	Margin           decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"margin"`
	Leverage         uint32          `gorm:"not null" json:"leverage"`
	UnrealizedPnL    decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"unrealized_pnl"`
	RealizedPnL      decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"realized_pnl"`
	Status           string          `gorm:"type:varchar(20);index;not null" json:"status"`
	Version          uint64          `gorm:"default:0;not null" json:"version"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (Position) TableName() string { return "positions" }

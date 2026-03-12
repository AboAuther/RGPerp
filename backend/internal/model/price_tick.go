package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type PriceTick struct {
	ID         uint64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Symbol     string          `gorm:"type:varchar(20);index;not null" json:"symbol"`
	IndexPrice decimal.Decimal `gorm:"type:decimal(36,18)" json:"index_price"`
	MarkPrice  decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"mark_price"`
	BestBid    decimal.Decimal `gorm:"type:decimal(36,18)" json:"best_bid"`
	BestAsk    decimal.Decimal `gorm:"type:decimal(36,18)" json:"best_ask"`
	Source     string          `gorm:"type:varchar(30)" json:"source"`
	CreatedAt  time.Time       `gorm:"autoCreateTime;index" json:"created_at"`
}

func (PriceTick) TableName() string { return "price_ticks" }

package model

import "github.com/shopspring/decimal"

type SettlementPool struct {
	BaseModel
	Asset   string          `gorm:"type:varchar(20);uniqueIndex;not null" json:"asset"`
	Balance decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null" json:"balance"`
	Version uint64          `gorm:"default:0;not null" json:"version"`
}

func (SettlementPool) TableName() string { return "settlement_pools" }

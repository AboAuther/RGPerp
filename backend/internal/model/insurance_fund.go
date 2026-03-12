package model

import "github.com/shopspring/decimal"

type InsuranceFund struct {
	BaseModel
	Symbol  string          `gorm:"type:varchar(20);uniqueIndex;not null" json:"symbol"`
	Balance decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null" json:"balance"`
	Version uint64          `gorm:"default:0;not null" json:"version"`
}

func (InsuranceFund) TableName() string { return "insurance_fund" }

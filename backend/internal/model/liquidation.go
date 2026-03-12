package model

import "github.com/shopspring/decimal"

type Liquidation struct {
	BaseModel
	UserID           uint64          `gorm:"index;not null" json:"user_id"`
	PositionID       uint64          `gorm:"index;not null" json:"position_id"`
	Symbol           string          `gorm:"type:varchar(20);not null" json:"symbol"`
	Side             string          `gorm:"type:varchar(10);not null" json:"side"`
	Size             decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"size"`
	EntryPrice       decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"entry_price"`
	MarkPrice        decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"mark_price"`
	LiquidationPrice decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"liquidation_price"`
	ExecutionPrice   decimal.Decimal `gorm:"type:decimal(36,18)" json:"execution_price"`
	MarginReleased   decimal.Decimal `gorm:"type:decimal(36,18)" json:"margin_released"`
	RealizedPnL      decimal.Decimal `gorm:"type:decimal(36,18)" json:"realized_pnl"`
	InsuranceFund    decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"insurance_fund"`
	Type             string          `gorm:"type:varchar(20);not null" json:"type"`
	Status           string          `gorm:"type:varchar(20);index;not null" json:"status"`

	User     User     `gorm:"foreignKey:UserID" json:"-"`
	Position Position `gorm:"foreignKey:PositionID" json:"-"`
}

func (Liquidation) TableName() string { return "liquidations" }

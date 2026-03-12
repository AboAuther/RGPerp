package model

import "github.com/shopspring/decimal"

type Trade struct {
	BaseModel
	OrderID       uint64          `gorm:"index;not null" json:"order_id"`
	UserID        uint64          `gorm:"index;not null" json:"user_id"`
	Symbol        string          `gorm:"type:varchar(20);index;not null" json:"symbol"`
	Side          string          `gorm:"type:varchar(10);not null" json:"side"`
	Size          decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"size"`
	Price         decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"price"`
	Margin        decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"margin"`
	Fee           decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"fee"`
	RealizedPnL   decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"realized_pnl"`
	IsLiquidation bool            `gorm:"default:false;not null" json:"is_liquidation"`

	Order Order `gorm:"foreignKey:OrderID" json:"-"`
	User  User  `gorm:"foreignKey:UserID" json:"-"`
}

func (Trade) TableName() string { return "trades" }

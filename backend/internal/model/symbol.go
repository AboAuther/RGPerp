package model

import "github.com/shopspring/decimal"

type Symbol struct {
	BaseModel
	Name                  string          `gorm:"type:varchar(20);uniqueIndex;not null" json:"name"`
	BaseAsset             string          `gorm:"type:varchar(10);not null" json:"base_asset"`
	QuoteAsset            string          `gorm:"type:varchar(10);not null" json:"quote_asset"`
	Status                string          `gorm:"type:varchar(20);default:'trading';not null" json:"status"`
	MaxLeverage           uint32          `gorm:"not null" json:"max_leverage"`
	MinOrderSize          decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"min_order_size"`
	MaxPositionNotional   decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"max_position_notional"`
	TickSize              decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"tick_size"`
	LotSize               decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"lot_size"`
	InitialMarginRate     decimal.Decimal `gorm:"type:decimal(10,6);not null" json:"initial_margin_rate"`
	MaintenanceMarginRate decimal.Decimal `gorm:"type:decimal(10,6);not null" json:"maintenance_margin_rate"`
	MakerFeeRate          decimal.Decimal `gorm:"type:decimal(10,6);default:0;not null" json:"maker_fee_rate"`
	TakerFeeRate          decimal.Decimal `gorm:"type:decimal(10,6);default:0;not null" json:"taker_fee_rate"`
	HyperliquidAssetIndex int             `gorm:"not null" json:"hyperliquid_asset_index"`
}

func (Symbol) TableName() string { return "symbols" }

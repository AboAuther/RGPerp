package model

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func SeedSymbols(db *gorm.DB) error {
	btcPerp := Symbol{
		Name:                  "BTC-PERP",
		BaseAsset:             "BTC",
		QuoteAsset:            "USDC",
		Status:                "trading",
		MaxLeverage:           50,
		MinOrderSize:          decimal.NewFromFloat(0.001),
		MaxPositionNotional:   decimal.NewFromFloat(1000000),
		TickSize:              decimal.NewFromFloat(0.1),
		LotSize:               decimal.NewFromFloat(0.001),
		InitialMarginRate:     decimal.NewFromFloat(0.02),
		MaintenanceMarginRate: decimal.NewFromFloat(0.005),
		MakerFeeRate:          decimal.NewFromFloat(0),
		TakerFeeRate:          decimal.NewFromFloat(0.0005),
		HyperliquidAssetIndex: 0,
	}

	var count int64
	db.Model(&Symbol{}).Where("name = ?", btcPerp.Name).Count(&count)
	if count == 0 {
		return db.Create(&btcPerp).Error
	}
	return nil
}

func SeedInsuranceFund(db *gorm.DB) error {
	fund := InsuranceFund{
		Symbol:  "BTC-PERP",
		Balance: decimal.Zero,
	}

	var count int64
	db.Model(&InsuranceFund{}).Where("symbol = ?", fund.Symbol).Count(&count)
	if count == 0 {
		return db.Create(&fund).Error
	}
	return nil
}

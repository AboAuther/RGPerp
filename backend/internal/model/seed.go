package model

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func SeedSymbols(db *gorm.DB) error {
	symbols := []Symbol{
		{
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
		},
		{
			Name:                  "ETH-PERP",
			BaseAsset:             "ETH",
			QuoteAsset:            "USDC",
			Status:                "trading",
			MaxLeverage:           40,
			MinOrderSize:          decimal.NewFromFloat(0.01),
			MaxPositionNotional:   decimal.NewFromFloat(500000),
			TickSize:              decimal.NewFromFloat(0.01),
			LotSize:               decimal.NewFromFloat(0.01),
			InitialMarginRate:     decimal.NewFromFloat(0.025),
			MaintenanceMarginRate: decimal.NewFromFloat(0.006),
			MakerFeeRate:          decimal.NewFromFloat(0),
			TakerFeeRate:          decimal.NewFromFloat(0.0005),
			HyperliquidAssetIndex: 1,
		},
		{
			Name:                  "SOL-PERP",
			BaseAsset:             "SOL",
			QuoteAsset:            "USDC",
			Status:                "trading",
			MaxLeverage:           25,
			MinOrderSize:          decimal.NewFromFloat(0.1),
			MaxPositionNotional:   decimal.NewFromFloat(250000),
			TickSize:              decimal.NewFromFloat(0.01),
			LotSize:               decimal.NewFromFloat(0.1),
			InitialMarginRate:     decimal.NewFromFloat(0.04),
			MaintenanceMarginRate: decimal.NewFromFloat(0.01),
			MakerFeeRate:          decimal.NewFromFloat(0),
			TakerFeeRate:          decimal.NewFromFloat(0.0005),
			HyperliquidAssetIndex: 5,
		},
	}

	for _, item := range symbols {
		var count int64
		db.Model(&Symbol{}).Where("name = ?", item.Name).Count(&count)
		if count > 0 {
			continue
		}
		if err := db.Create(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

func SeedInsuranceFund(db *gorm.DB) error {
	funds := []InsuranceFund{
		{Symbol: "BTC-PERP", Balance: decimal.Zero},
		{Symbol: "ETH-PERP", Balance: decimal.Zero},
		{Symbol: "SOL-PERP", Balance: decimal.Zero},
	}

	for _, item := range funds {
		var count int64
		db.Model(&InsuranceFund{}).Where("symbol = ?", item.Symbol).Count(&count)
		if count > 0 {
			continue
		}
		if err := db.Create(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

func SeedSettlementPools(db *gorm.DB) error {
	pools := []SettlementPool{
		{Asset: "USDC", Balance: decimal.RequireFromString("100000")},
	}

	for _, item := range pools {
		var count int64
		db.Model(&SettlementPool{}).Where("asset = ?", item.Asset).Count(&count)
		if count > 0 {
			continue
		}
		if err := db.Create(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

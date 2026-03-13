package model

import "github.com/shopspring/decimal"

type Account struct {
	BaseModel
	UserID              uint64          `gorm:"uniqueIndex;not null" json:"user_id"`
	AvailableBalance    decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null" json:"available_balance"`
	LockedBalance       decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null" json:"locked_balance"`
	NetDeposits         decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null" json:"net_deposits"`
	SettledPnlBalance   decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null" json:"settled_pnl_balance"`
	UnsettledPnlBalance decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null" json:"unsettled_pnl_balance"`
	Version             uint64          `gorm:"default:0;not null" json:"version"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (Account) TableName() string { return "accounts" }

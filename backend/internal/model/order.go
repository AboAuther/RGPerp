package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type Order struct {
	BaseModel
	ClientOrderID   string          `gorm:"type:varchar(64);index" json:"client_order_id"`
	UserID          uint64          `gorm:"index;not null" json:"user_id"`
	Symbol          string          `gorm:"type:varchar(20);index;not null" json:"symbol"`
	Side            string          `gorm:"type:varchar(10);not null" json:"side"`
	Type            string          `gorm:"type:varchar(20);not null" json:"type"`
	MarginMode      string          `gorm:"type:varchar(20);default:'isolated';not null" json:"margin_mode"`
	Size            decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"size"`
	Price           decimal.Decimal `gorm:"type:decimal(36,18)" json:"price"`
	LimitPrice      decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"limit_price"`
	TimeInForce     string          `gorm:"type:varchar(20);default:'gtc'" json:"time_in_force"`
	Leverage        uint32          `gorm:"not null" json:"leverage"`
	Margin          decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"margin"`
	ReduceOnly      bool            `gorm:"default:false;not null" json:"reduce_only"`
	ReservedMargin  decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null" json:"reserved_margin"`
	ReservedFee     decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null" json:"reserved_fee"`
	Status          string          `gorm:"type:varchar(20);index;not null" json:"status"`
	FilledSize      decimal.Decimal `gorm:"type:decimal(36,18);default:0;not null" json:"filled_size"`
	ExecPrice       decimal.Decimal `gorm:"type:decimal(36,18)" json:"exec_price"`
	RealizedPnL     decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"realized_pnl"`
	Fee             decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"fee"`
	ErrorMessage    string          `gorm:"type:varchar(500)" json:"error_message"`
	CancelReason    string          `gorm:"type:varchar(255)" json:"cancel_reason"`
	TriggeredAt     *time.Time      `json:"triggered_at"`
	ExpiresAt       *time.Time      `json:"expires_at"`
	ParentOrderID   *uint64         `gorm:"index" json:"parent_order_id"`
	ExecutionSource string          `gorm:"type:varchar(20);default:'direct'" json:"execution_source"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (Order) TableName() string { return "orders" }

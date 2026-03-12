package model

import "github.com/shopspring/decimal"

type HedgeTask struct {
	BaseModel
	Symbol               string          `gorm:"type:varchar(20);index;not null" json:"symbol"`
	TriggerType          string          `gorm:"type:varchar(20);not null" json:"trigger_type"`
	InternalNetPosition  decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"internal_net_position"`
	TargetHedgePosition  decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"target_hedge_position"`
	CurrentHedgePosition decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"current_hedge_position"`
	Drift                decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"drift"`
	Status               string          `gorm:"type:varchar(20);index;not null" json:"status"`
	ErrorMessage         string          `gorm:"type:varchar(500)" json:"error_message"`
}

func (HedgeTask) TableName() string { return "hedge_tasks" }

type HedgeOrder struct {
	BaseModel
	HedgeTaskID     uint64          `gorm:"index;not null" json:"hedge_task_id"`
	Symbol          string          `gorm:"type:varchar(20);not null" json:"symbol"`
	Side            string          `gorm:"type:varchar(10);not null" json:"side"`
	Size            decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"size"`
	Price           decimal.Decimal `gorm:"type:decimal(36,18)" json:"price"`
	ExternalOrderID string          `gorm:"type:varchar(128)" json:"external_order_id"`
	Status          string          `gorm:"type:varchar(20);index;not null" json:"status"`
	FilledSize      decimal.Decimal `gorm:"type:decimal(36,18);default:0" json:"filled_size"`
	FilledPrice     decimal.Decimal `gorm:"type:decimal(36,18)" json:"filled_price"`
	RetryCount      uint32          `gorm:"default:0" json:"retry_count"`
	ErrorMessage    string          `gorm:"type:varchar(500)" json:"error_message"`

	HedgeTask HedgeTask `gorm:"foreignKey:HedgeTaskID" json:"-"`
}

func (HedgeOrder) TableName() string { return "hedge_orders" }

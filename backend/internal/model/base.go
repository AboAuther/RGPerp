package model

import "time"

type BaseModel struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

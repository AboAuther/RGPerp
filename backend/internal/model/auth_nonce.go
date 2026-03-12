package model

import "time"

type AuthNonce struct {
	BaseModel
	WalletAddress string    `gorm:"type:varchar(42);index;not null" json:"wallet_address"`
	Nonce         string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"nonce"`
	Message       string    `gorm:"type:text;not null" json:"message"`
	Domain        string    `gorm:"type:varchar(255);not null" json:"domain"`
	ChainID       uint64    `gorm:"not null" json:"chain_id"`
	ExpiresAt     time.Time `gorm:"index;not null" json:"expires_at"`
	Used          bool      `gorm:"default:false;not null" json:"used"`
}

func (AuthNonce) TableName() string { return "auth_nonces" }

package model

type User struct {
	BaseModel
	WalletAddress string `gorm:"type:varchar(42);uniqueIndex;not null" json:"wallet_address"`
	Status        string `gorm:"type:varchar(20);default:'active';not null" json:"status"`
}

func (User) TableName() string { return "users" }

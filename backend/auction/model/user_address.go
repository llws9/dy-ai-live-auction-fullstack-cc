package model

import "time"

// UserAddress 收货地址（spec A / F-A3）。
//
// 同 user_id 至多一条 is_default=true，由 DAO.SetDefault 事务保证。
type UserAddress struct {
	ID            int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        int64     `json:"user_id" gorm:"index:idx_user;index:idx_user_default,priority:1;not null"`
	RecipientName string    `json:"recipient_name" gorm:"type:varchar(32);not null"`
	Phone         string    `json:"phone" gorm:"type:varchar(20);not null"`
	Province      string    `json:"province" gorm:"type:varchar(32);not null"`
	City          string    `json:"city" gorm:"type:varchar(32);not null"`
	District      string    `json:"district" gorm:"type:varchar(32);not null"`
	Detail        string    `json:"detail" gorm:"type:varchar(128);not null"`
	IsDefault     bool      `json:"is_default" gorm:"index:idx_user_default,priority:2;not null;default:false"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (UserAddress) TableName() string {
	return "user_addresses"
}

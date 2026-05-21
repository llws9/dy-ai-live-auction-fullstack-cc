package model

import "time"

// User 用户信息模型
type User struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"type:varchar(64);not null"`
	Avatar    string    `json:"avatar" gorm:"type:varchar(256)"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

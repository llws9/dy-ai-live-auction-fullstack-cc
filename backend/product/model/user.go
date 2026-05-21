package model

import "time"

// User 用户信息模型
type User struct {
	ID          int64      `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string     `json:"name" gorm:"type:varchar(64);not null"`
	Avatar      string     `json:"avatar" gorm:"type:varchar(256)"`
	Email       *string    `json:"email,omitempty" gorm:"type:varchar(128);uniqueIndex"`
	Phone       *string    `json:"phone,omitempty" gorm:"type:varchar(20);uniqueIndex"`
	Password    string     `json:"-" gorm:"type:varchar(256);not null"` // 不序列化到JSON
	Role        int        `json:"role" gorm:"default:0"`              // 0=普通用户, 1=管理员
	Status      int        `json:"status" gorm:"default:1"`            // 0=禁用, 1=正常
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
}

// IsAdmin 判断是否为管理员
func (u *User) IsAdmin() bool {
	return u.Role == 1
}

// IsActive 判断用户是否激活
func (u *User) IsActive() bool {
	return u.Status == 1
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

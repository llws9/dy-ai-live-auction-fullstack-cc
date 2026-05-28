package model

import "time"

// SkyLampStatus 天灯订阅状态
type SkyLampStatus int

const (
	SkyLampStatusActive   SkyLampStatus = 1 // 活跃状态
	SkyLampStatusStopped  SkyLampStatus = 2 // 已停止
	SkyLampStatusExpired  SkyLampStatus = 3 // 已过期
)

// SkyLampSubscription 天灯订阅模型
type SkyLampSubscription struct {
	ID        int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	AuctionID int64          `json:"auction_id" gorm:"index;not null"`
	UserID    int64          `json:"user_id" gorm:"index;not null"`
	Status    SkyLampStatus  `json:"status" gorm:"type:tinyint;default:1"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (SkyLampSubscription) TableName() string {
	return "sky_lamp_subscriptions"
}

// IsActive 检查订阅是否活跃
func (s *SkyLampSubscription) IsActive() bool {
	return s.Status == SkyLampStatusActive
}
package model

import "time"

// LiveStreamStatus 直播间状态
type LiveStreamStatus int

const (
	LiveStreamStatusDisabled LiveStreamStatus = 0 // 禁用
	LiveStreamStatusActive   LiveStreamStatus = 1 // 正常
)

// LiveStream 直播间实体
type LiveStream struct {
	ID          int64            `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatorID   int64            `json:"creator_id" gorm:"uniqueIndex;not null"`         // 商家ID（用户ID），一对一关系
	Name        string           `json:"name" gorm:"type:varchar(128);not null"`          // 直播间名称
	Description string           `json:"description" gorm:"type:text"`                    // 直播间描述
	CoverImage  string           `json:"cover_image" gorm:"type:varchar(256)"`            // 封面图URL
	VideoURL    string           `json:"video_url" gorm:"type:varchar(512)"`              // 直播流URL（HLS/FLV，本期由后台手动配置）
	Status      LiveStreamStatus `json:"status" gorm:"type:tinyint;default:1"`            // 状态：0=禁用，1=正常
	CreatedAt   time.Time        `json:"created_at" gorm:"autoCreateTime"`                // 创建时间
	UpdatedAt   time.Time        `json:"updated_at" gorm:"autoUpdateTime"`                // 更新时间
}

// TableName 指定表名
func (LiveStream) TableName() string {
	return "live_streams"
}

// IsActive 判断直播间是否正常
func (ls *LiveStream) IsActive() bool {
	return ls.Status == LiveStreamStatusActive
}

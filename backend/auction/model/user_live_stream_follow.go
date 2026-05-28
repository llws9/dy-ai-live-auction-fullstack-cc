package model

import "time"

// UserLiveStreamFollow 用户关注直播间关系
type UserLiveStreamFollow struct {
	ID                  int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID              int64     `json:"user_id" gorm:"uniqueIndex:uk_user_live_stream;not null;index"`        // 用户ID
	LiveStreamID        int64     `json:"live_stream_id" gorm:"uniqueIndex:uk_user_live_stream;not null;index"` // 直播间ID
	NotificationEnabled bool      `json:"notification_enabled" gorm:"default:true"`                              // 是否接收通知
	CreatedAt           time.Time `json:"created_at" gorm:"autoCreateTime"`                                      // 关注时间
}

// TableName 指定表名
func (UserLiveStreamFollow) TableName() string {
	return "user_live_stream_follows"
}

// EnableNotification 开启通知
func (f *UserLiveStreamFollow) EnableNotification() {
	f.NotificationEnabled = true
}

// DisableNotification 关闭通知
func (f *UserLiveStreamFollow) DisableNotification() {
	f.NotificationEnabled = false
}

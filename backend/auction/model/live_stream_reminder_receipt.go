package model

import "time"

type LiveStreamReminderReceipt struct {
	ID            int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        int64     `json:"user_id" gorm:"not null;uniqueIndex:uk_user_stream_started,priority:1;index"`
	LiveStreamID  int64     `json:"live_stream_id" gorm:"not null;uniqueIndex:uk_user_stream_started,priority:2"`
	LiveStartedAt int64     `json:"live_started_at" gorm:"not null;uniqueIndex:uk_user_stream_started,priority:3"`
	RemindedAt    time.Time `json:"reminded_at" gorm:"not null"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (LiveStreamReminderReceipt) TableName() string {
	return "live_stream_reminder_receipts"
}

type PendingLiveReminderResponse struct {
	HasReminder bool        `json:"hasReminder"`
	Stream      *StreamInfo `json:"stream"`
}

type StreamInfo struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatarUrl"`
	StatusText string `json:"statusText"`
	LiveRoomID int64  `json:"liveRoomId"`
	StartedAt  int64  `json:"startedAt"`
}

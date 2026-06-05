package model

import "time"

type BusinessEvent struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        int64     `gorm:"index;not null" json:"user_id"`
	EventType     string    `gorm:"size:64;index;not null" json:"event_type"`
	Source        string    `gorm:"size:64;index;not null;default:unknown" json:"source"`
	LiveStreamID  int64     `gorm:"index" json:"live_stream_id"`
	AuctionID     int64     `gorm:"index" json:"auction_id"`
	ProductID     int64     `gorm:"index" json:"product_id"`
	ClientEventID string    `gorm:"size:128;uniqueIndex" json:"client_event_id"`
	Metadata      string    `gorm:"type:json" json:"metadata"`
	CreatedAt     time.Time `gorm:"index" json:"created_at"`
}

func (BusinessEvent) TableName() string {
	return "business_events"
}

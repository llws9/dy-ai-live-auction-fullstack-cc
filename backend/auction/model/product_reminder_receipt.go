package model

import "time"

type ProductReminderReceipt struct {
	ID         int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID     int64     `json:"user_id" gorm:"not null;uniqueIndex:uk_user_auction_reminder,priority:1;index"`
	AuctionID  int64     `json:"auction_id" gorm:"not null;uniqueIndex:uk_user_auction_reminder,priority:2;index"`
	RemindedAt time.Time `json:"reminded_at" gorm:"not null"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (ProductReminderReceipt) TableName() string {
	return "product_reminder_receipts"
}

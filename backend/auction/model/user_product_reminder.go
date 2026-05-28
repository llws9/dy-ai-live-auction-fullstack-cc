package model

import "time"

// UserProductReminder 用户订阅商品竞拍提醒
type UserProductReminder struct {
	ID                  int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID              int64     `json:"user_id" gorm:"not null;index:idx_user_product"` // 用户ID
	ProductID           int64     `json:"product_id" gorm:"not null;index:idx_user_product;index:idx_product"` // 商品ID
	AuctionID           int64     `json:"auction_id" gorm:"index"` // 关联的竞拍ID
	NotificationEnabled bool      `json:"notification_enabled" gorm:"default:true"` // 是否接收通知
	CreatedAt           time.Time `json:"created_at" gorm:"autoCreateTime"` // 订阅时间
}

// TableName 指定表名
func (UserProductReminder) TableName() string {
	return "user_product_reminders"
}

// EnableNotification 开启通知
func (r *UserProductReminder) EnableNotification() {
	r.NotificationEnabled = true
}

// DisableNotification 关闭通知
func (r *UserProductReminder) DisableNotification() {
	r.NotificationEnabled = false
}
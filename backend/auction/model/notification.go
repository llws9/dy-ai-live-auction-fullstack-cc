package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// NotificationType 通知类型
type NotificationType string

const (
	NotificationTypeBidOutbid      NotificationType = "bid_outbid"       // 出价被超越
	NotificationTypeAuctionWon     NotificationType = "auction_won"      // 竞拍中标
	NotificationTypeAuctionLost    NotificationType = "auction_lost"     // 竞拍未中标
	NotificationTypeOrderPaid      NotificationType = "order_paid"       // 订单已支付
	NotificationTypeOrderShipped   NotificationType = "order_shipped"    // 订单已发货
	NotificationTypeOrderCompleted NotificationType = "order_completed"  // 订单已完成
	// 新增：直播间竞拍相关通知
	NotificationTypeNewProduct       NotificationType = "new_product"        // 新商品发布
	NotificationTypeAuctionStarting  NotificationType = "auction_starting"   // 竞拍即将开始
	NotificationTypeProductUnpublished NotificationType = "product_unpublished" // 商品已下架
	NotificationTypeAuctionEnded     NotificationType = "auction_ended"      // 竞拍已结束
)

// Notification 用户通知实体
type Notification struct {
	ID        int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64            `gorm:"not null;index:idx_user_id_created_at,priority:1;index:idx_user_id_read_at,priority:1" json:"user_id"`
	Type      NotificationType `gorm:"type:varchar(32);not null" json:"type"`
	Title     string           `gorm:"type:varchar(128);not null" json:"title"`
	Content   string           `gorm:"type:text;not null" json:"content"`
	Data      JSONMap          `gorm:"type:json" json:"data"`
	ReadAt    *time.Time       `gorm:"index:idx_user_id_read_at,priority:2" json:"read_at"`
	CreatedAt time.Time        `gorm:"autoCreateTime;index:idx_user_id_created_at,priority:2" json:"created_at"`
}

// TableName 指定表名
func (Notification) TableName() string {
	return "notifications"
}

// JSONMap 用于存储JSON数据的map类型
type JSONMap map[string]interface{}

// Value 实现driver.Valuer接口
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现sql.Scanner接口
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}

// NotificationRequest 通知发送请求
type NotificationRequest struct {
	UserID      int64
	Type        NotificationType
	Title       string
	Content     string
	Data        map[string]interface{}
	Immediately bool // 是否立即推送，默认true
}

// NotificationListResponse 通知列表响应
type NotificationListResponse struct {
	Items    []Notification `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// UnreadCountResponse 未读数量响应
type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

// OrderEvent 订单事件（二期实现）
type OrderEvent struct {
	OrderID   int64
	EventType OrderEventType
	OldStatus int
	NewStatus int
	UserID    int64
	Timestamp time.Time
	Extra     map[string]interface{}
}

// OrderEventType 订单事件类型
type OrderEventType string

const (
	OrderEventPaid      OrderEventType = "paid"
	OrderEventShipped   OrderEventType = "shipped"
	OrderEventCompleted OrderEventType = "completed"
	OrderEventCancelled OrderEventType = "cancelled"
)

package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// OrderStatus 订单状态
type OrderStatus int

const (
	OrderStatusPending   OrderStatus = 0 // 待支付
	OrderStatusPaid      OrderStatus = 1 // 已支付
	OrderStatusShipped   OrderStatus = 2 // 已发货
	OrderStatusCompleted OrderStatus = 3 // 已完成
)

// Order 订单模型
type Order struct {
	ID          int64           `json:"id" gorm:"primaryKey;autoIncrement"`
	AuctionID   int64           `json:"auction_id" gorm:"not null;uniqueIndex"`
	ProductID   int64           `json:"product_id" gorm:"not null;index"`
	WinnerID    int64           `json:"winner_id" gorm:"not null;index"`
	FinalPrice  decimal.Decimal `json:"final_price" gorm:"type:decimal(10,2);not null"`
	Status      OrderStatus     `json:"status" gorm:"type:tinyint;default:0"`
	PaidAt      *time.Time      `json:"paid_at,omitempty"`
	ShippedAt   *time.Time      `json:"shipped_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (Order) TableName() string {
	return "orders"
}

// OrderSummaryResponse 订单触点汇总
type OrderSummaryResponse struct {
	PendingPayment int64 `json:"pendingPayment"`
	WonNotPaid     int64 `json:"wonNotPaid"`
}

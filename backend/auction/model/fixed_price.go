package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// FixedPriceStatus 一口价商品状态
type FixedPriceStatus int8

const (
	FixedPriceStatusOnSale  FixedPriceStatus = 1 // 在售
	FixedPriceStatusSoldOut FixedPriceStatus = 2 // 已售罄
	FixedPriceStatusOffline FixedPriceStatus = 3 // 已下架
)

// FixedPriceItem 一口价商品模型
type FixedPriceItem struct {
	ID             int64            `json:"id" gorm:"primaryKey;autoIncrement"`
	LiveStreamID   int64            `json:"live_stream_id" gorm:"index;not null"`
	ProductID      int64            `json:"product_id" gorm:"not null"`
	CreatorID      int64            `json:"creator_id" gorm:"index;not null"`
	Price          decimal.Decimal  `json:"price" gorm:"type:decimal(10,2);not null"`
	TotalStock     int              `json:"total_stock" gorm:"not null"`
	RemainingStock int              `json:"remaining_stock" gorm:"not null"`
	MaxPerUser     int              `json:"max_per_user" gorm:"not null;default:1"`
	Status         FixedPriceStatus `json:"status" gorm:"type:tinyint;not null;default:1"`
	Version        int              `json:"version" gorm:"not null;default:0"`
	CreatedAt      time.Time        `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time        `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (FixedPriceItem) TableName() string { return "fixed_price_items" }

// FixedPricePurchase 一口价购买记录模型
type FixedPricePurchase struct {
	ID        int64           `json:"id" gorm:"primaryKey;autoIncrement"`
	ItemID    int64           `json:"item_id" gorm:"uniqueIndex:uniq_item_user;not null"`
	UserID    int64           `json:"user_id" gorm:"uniqueIndex:uniq_item_user;not null"`
	Price     decimal.Decimal `json:"price" gorm:"type:decimal(10,2);not null"`
	CreatedAt time.Time       `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (FixedPricePurchase) TableName() string { return "fixed_price_purchases" }

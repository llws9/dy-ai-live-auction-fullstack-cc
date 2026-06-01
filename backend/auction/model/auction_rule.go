package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// AuctionRule 竞拍规则模型
type AuctionRule struct {
	ID                 int64            `json:"id" gorm:"primaryKey;autoIncrement"`
	ProductID          int64            `json:"product_id" gorm:"not null;index"`
	StartPrice         decimal.Decimal  `json:"start_price" gorm:"type:decimal(10,2);default:0"`
	Increment          decimal.Decimal  `json:"increment" gorm:"type:decimal(10,2);not null"`
	CapPrice           *decimal.Decimal `json:"cap_price" gorm:"type:decimal(10,2)"`
	Duration           int              `json:"duration" gorm:"not null"`
	DelayDuration      int              `json:"delay_duration" gorm:"default:30"`
	MaxDelayTime       int              `json:"max_delay_time" gorm:"default:180"`
	TriggerDelayBefore int              `json:"trigger_delay_before" gorm:"default:30"`
	CreatedAt          time.Time        `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (AuctionRule) TableName() string {
	return "auction_rules"
}

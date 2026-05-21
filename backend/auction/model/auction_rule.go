package model

import "time"

// AuctionRule 竞拍规则模型
type AuctionRule struct {
	ID                int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ProductID         int64     `json:"product_id" gorm:"not null;index"`
	StartPrice        float64   `json:"start_price" gorm:"type:decimal(10,2);default:0"`
	Increment         float64   `json:"increment" gorm:"type:decimal(10,2);not null"`
	CapPrice         *float64   `json:"cap_price" gorm:"type:decimal(10,2)"`
	Duration          int       `json:"duration" gorm:"not null"`
	DelayDuration     int       `json:"delay_duration" gorm:"default:30"`
	MaxDelayTime      int       `json:"max_delay_time" gorm:"default:180"`
	TriggerDelayBefore int      `json:"trigger_delay_before" gorm:"default:30"`
	CreatedAt         time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (AuctionRule) TableName() string {
	return "auction_rules"
}

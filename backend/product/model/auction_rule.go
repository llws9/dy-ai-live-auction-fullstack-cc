package model

import "time"

// AuctionRule 竞拍规则模型
type AuctionRule struct {
	ID                int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	ProductID         int64     `json:"product_id" gorm:"not null;index"`
	StartPrice        float64   `json:"start_price" gorm:"type:decimal(10,2);default:0"`      // 起拍价，默认0元
	Increment         float64   `json:"increment" gorm:"type:decimal(10,2);not null"`         // 加价幅度
	CapPrice         *float64   `json:"cap_price" gorm:"type:decimal(10,2)"`                  // 封顶价
	Duration          int       `json:"duration" gorm:"not null"`                             // 竞拍时长（秒）
	DelayDuration     int       `json:"delay_duration" gorm:"default:30"`                     // 单次延时时长（秒）
	MaxDelayTime      int       `json:"max_delay_time" gorm:"default:180"`                    // 最大延时时长（秒）
	TriggerDelayBefore int      `json:"trigger_delay_before" gorm:"default:30"`               // 延时触发时间（秒）
	CreatedAt         time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (AuctionRule) TableName() string {
	return "auction_rules"
}

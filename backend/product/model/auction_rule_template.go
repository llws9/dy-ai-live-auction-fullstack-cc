package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type AuctionRuleTemplate struct {
	ID                 int64            `json:"id" gorm:"primaryKey;autoIncrement"`
	OwnerID            int64            `json:"owner_id" gorm:"not null;index;uniqueIndex:uniq_rule_templates_owner_name,priority:1"`
	Name               string           `json:"name" gorm:"type:varchar(128);not null;uniqueIndex:uniq_rule_templates_owner_name,priority:2"`
	StartPrice         decimal.Decimal  `json:"start_price" gorm:"type:decimal(10,2);not null;default:0"`
	Increment          decimal.Decimal  `json:"increment" gorm:"type:decimal(10,2);not null"`
	CapPrice           *decimal.Decimal `json:"cap_price,omitempty" gorm:"type:decimal(10,2)"`
	Duration           int              `json:"duration" gorm:"not null"`
	DelayDuration      int              `json:"delay_duration" gorm:"not null;default:30"`
	MaxDelayTime       int              `json:"max_delay_time" gorm:"not null;default:180"`
	TriggerDelayBefore int              `json:"trigger_delay_before" gorm:"not null;default:30"`
	IsDefault          bool             `json:"is_default" gorm:"not null;default:false"`
	CreatedAt          time.Time        `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt          time.Time        `json:"updated_at" gorm:"autoUpdateTime"`
}

func (AuctionRuleTemplate) TableName() string {
	return "auction_rule_templates"
}

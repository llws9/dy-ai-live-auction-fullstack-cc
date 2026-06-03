package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// Bid 出价记录模型
type Bid struct {
	ID        int64           `json:"id" gorm:"primaryKey;autoIncrement"`
	AuctionID int64           `json:"auction_id" gorm:"index;not null"`
	UserID    int64           `json:"user_id" gorm:"index;not null"`
	Amount    decimal.Decimal `json:"amount" gorm:"type:decimal(10,2);not null"`
	CreatedAt time.Time       `json:"created_at" gorm:"autoCreateTime;index:idx_auction_created,priority:2,sort:desc"`
}

// TableName 指定表名
func (Bid) TableName() string {
	return "bids"
}

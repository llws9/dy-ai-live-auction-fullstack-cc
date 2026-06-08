package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// AuctionStatus 竞拍状态
type AuctionStatus int

const (
	AuctionStatusPending   AuctionStatus = 0 // 待开始
	AuctionStatusOngoing   AuctionStatus = 1 // 进行中
	AuctionStatusDelayed   AuctionStatus = 2 // 延时中
	AuctionStatusEnded     AuctionStatus = 3 // 已结束
	AuctionStatusCancelled AuctionStatus = 4 // 已取消
)

// Auction 竞拍场次模型
type Auction struct {
	ID           int64           `json:"id" gorm:"primaryKey;autoIncrement"`
	ProductID    int64           `json:"product_id" gorm:"index;not null"`
	LiveStreamID *int64          `json:"live_stream_id" gorm:"index"` // 直播间ID（新增字段）
	CreatorID    *int64          `json:"creator_id" gorm:"index"`     // 竞拍创建者ID（主播）
	Status       AuctionStatus   `json:"status" gorm:"type:tinyint;default:0"`
	CurrentPrice decimal.Decimal `json:"current_price" gorm:"type:decimal(10,2);default:0"`
	WinnerID     *int64          `json:"winner_id"`
	StartTime    time.Time       `json:"start_time" gorm:"index;not null"`
	EndTime      time.Time       `json:"end_time" gorm:"not null"`
	DelayUsed    int             `json:"delay_used" gorm:"default:0"` // 已延时秒数
	Version      int             `json:"version" gorm:"default:0"`    // 乐观锁版本号
	CreatedAt    time.Time       `json:"created_at" gorm:"autoCreateTime"`
	// BidCount 由聚合查询回填，不参与建表或写入。
	BidCount int `json:"bid_count" gorm:"->;-:migration"`
}

type AuctionOrderRequest struct {
	AuctionID  int64           `json:"auction_id"`
	ProductID  int64           `json:"product_id"`
	WinnerID   int64           `json:"winner_id"`
	FinalPrice decimal.Decimal `json:"final_price"`
}

// TableName 指定表名
func (Auction) TableName() string {
	return "auctions"
}

// IsEnded 检查竞拍是否已结束
func (a *Auction) IsEnded() bool {
	return a.Status == AuctionStatusEnded || a.Status == AuctionStatusCancelled
}

// CanBid 检查是否可以出价
func (a *Auction) CanBid() bool {
	if !(a.Status == AuctionStatusOngoing || a.Status == AuctionStatusDelayed) {
		return false
	}
	return time.Now().Before(a.EndTime)
}

// IsInDelayWindow 检查是否在延时窗口内（结束前30秒）
func (a *Auction) IsInDelayWindow(triggerDelayBefore int) bool {
	remaining := time.Until(a.EndTime)
	return remaining.Seconds() <= float64(triggerDelayBefore) && remaining.Seconds() > 0
}

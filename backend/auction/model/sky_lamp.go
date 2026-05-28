package model

import "time"

// SkyLampStatus 点天灯状态
type SkyLampStatus int

const (
	SkyLampStatusActive    SkyLampStatus = 1 // 活跃，正在跟价
	SkyLampStatusStopped   SkyLampStatus = 2 // 已停止（达到上限）
	SkyLampStatusCancelled SkyLampStatus = 3 // 已取消（用户主动）
	SkyLampStatusEnded     SkyLampStatus = 4 // 竞拍结束
)

// SkyLampSubscription 点天灯订阅
type SkyLampSubscription struct {
	ID                  int64         `json:"id" gorm:"primaryKey;autoIncrement"`
	AuctionID           int64         `json:"auction_id" gorm:"not null;index:idx_auction_user;index:idx_auction_status"`
	UserID              int64         `json:"user_id" gorm:"not null;index:idx_auction_user;index:idx_user_status"`
	Status              SkyLampStatus `json:"status" gorm:"type:tinyint;default:1"`
	InitialPrice        float64       `json:"initial_price" gorm:"type:decimal(10,2);not null"`       // 开启时的当前价格
	InitialBidAmount    float64       `json:"initial_bid_amount" gorm:"type:decimal(10,2);not null"`   // 首次出价金额
	MaxPriceLimit       float64       `json:"max_price_limit" gorm:"type:decimal(10,2);not null"`      // 天灯上限金额
	CurrentAutoBidCount int           `json:"current_auto_bid_count" gorm:"default:0"`                 // 已自动跟价次数
	TotalBidAmount      float64       `json:"total_bid_amount" gorm:"type:decimal(10,2);default:0"`    // 累计出价金额
	CreatedAt           time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time     `json:"updated_at" gorm:"autoUpdateTime"`
	StoppedAt           *time.Time    `json:"stopped_at"` // 停止时间
}

// TableName 指定表名
func (SkyLampSubscription) TableName() string {
	return "sky_lamp_subscriptions"
}

// IsActive 检查点天灯是否活跃
func (s *SkyLampSubscription) IsActive() bool {
	return s.Status == SkyLampStatusActive
}

// CanAutoBid 检查是否可以自动跟价
func (s *SkyLampSubscription) CanAutoBid(currentPrice float64) bool {
	if !s.IsActive() {
		return false
	}
	// 当前价格不能超过天灯上限
	return currentPrice <= s.MaxPriceLimit
}

// GetNextBidAmount 计算下次出价金额（按最低加价幅度）
func (s *SkyLampSubscription) GetNextBidAmount(currentPrice, increment float64) float64 {
	return currentPrice + increment
}
package model

import "time"

type AuctionSettlementTaskStatus string

const (
	AuctionSettlementTaskStatusPending   AuctionSettlementTaskStatus = "pending"
	AuctionSettlementTaskStatusOrderDone AuctionSettlementTaskStatus = "order_done"
	AuctionSettlementTaskStatusDone      AuctionSettlementTaskStatus = "done"
)

// AuctionSettlementTask records post-auction settlement progress independently
// from the auction lifecycle state.
type AuctionSettlementTask struct {
	AuctionID int64                       `json:"auction_id" gorm:"primaryKey"`
	Status    AuctionSettlementTaskStatus `json:"status" gorm:"type:varchar(20);not null;index"`
	LastError string                      `json:"last_error" gorm:"type:text"`
	CreatedAt time.Time                   `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time                   `json:"updated_at" gorm:"autoUpdateTime"`
}

func (AuctionSettlementTask) TableName() string {
	return "auction_settlement_tasks"
}

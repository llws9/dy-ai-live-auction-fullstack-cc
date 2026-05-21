package service

import (
	"errors"
	"time"

	"auction-service/model"
)

// 状态转换错误
var (
	ErrInvalidTransition = errors.New("无效的状态转换")
	ErrAuctionEnded      = errors.New("竞拍已结束")
	ErrAuctionCancelled  = errors.New("竞拍已取消")
)

// StateMachine 竞拍状态机
type StateMachine struct {
	auction *model.Auction
}

// NewStateMachine 创建状态机
func NewStateMachine(auction *model.Auction) *StateMachine {
	return &StateMachine{auction: auction}
}

// CanTransition 检查是否可以进行状态转换
func (sm *StateMachine) CanTransition(targetStatus model.AuctionStatus) bool {
	current := sm.auction.Status

	switch current {
	case model.AuctionStatusPending:
		// 待开始可以转换为：进行中、已取消
		return targetStatus == model.AuctionStatusOngoing || targetStatus == model.AuctionStatusCancelled

	case model.AuctionStatusOngoing:
		// 进行中可以转换为：延时中、已结束、已取消
		return targetStatus == model.AuctionStatusDelayed || targetStatus == model.AuctionStatusEnded || targetStatus == model.AuctionStatusCancelled

	case model.AuctionStatusDelayed:
		// 延时中可以转换为：已结束、已取消
		return targetStatus == model.AuctionStatusEnded || targetStatus == model.AuctionStatusCancelled

	case model.AuctionStatusEnded, model.AuctionStatusCancelled:
		// 已结束或已取消不能转换
		return false

	default:
		return false
	}
}

// Transition 执行状态转换
func (sm *StateMachine) Transition(targetStatus model.AuctionStatus) error {
	if !sm.CanTransition(targetStatus) {
		return ErrInvalidTransition
	}

	sm.auction.Status = targetStatus
	return nil
}

// CanBid 检查是否可以出价
func (sm *StateMachine) CanBid() bool {
	return sm.auction.Status == model.AuctionStatusOngoing || sm.auction.Status == model.AuctionStatusDelayed
}

// CanCancel 检查是否可以取消
func (sm *StateMachine) CanCancel() bool {
	return sm.auction.Status == model.AuctionStatusPending || sm.auction.Status == model.AuctionStatusOngoing
}

// ShouldTriggerDelay 检查是否应该触发延时
func (sm *StateMachine) ShouldTriggerDelay(triggerDelayBefore int) bool {
	if !sm.CanBid() {
		return false
	}

	remaining := time.Until(sm.auction.EndTime)
	return remaining.Seconds() <= float64(triggerDelayBefore) && remaining.Seconds() > 0
}

// CanDelay 检查是否可以继续延时
func (sm *StateMachine) CanDelay(maxDelayTime int) bool {
	return sm.auction.DelayUsed < maxDelayTime
}

// GetRemainingDelayTime 获取剩余可延时时长
func (sm *StateMachine) GetRemainingDelayTime(maxDelayTime int) int {
	remaining := maxDelayTime - sm.auction.DelayUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ShouldEnd 检查是否应该结束
func (sm *StateMachine) ShouldEnd() bool {
	if sm.auction.Status == model.AuctionStatusEnded || sm.auction.Status == model.AuctionStatusCancelled {
		return false
	}

	// 检查是否到达结束时间
	return time.Now().After(sm.auction.EndTime) || time.Now().Equal(sm.auction.EndTime)
}

// IsExpired 检查竞拍是否已过期（用于定时任务检查）
func (sm *StateMachine) IsExpired() bool {
	return time.Now().After(sm.auction.EndTime)
}

// CanStart 检查是否应该开始
func (sm *StateMachine) CanStart() bool {
	return sm.auction.Status == model.AuctionStatusPending && time.Now().After(sm.auction.StartTime)
}

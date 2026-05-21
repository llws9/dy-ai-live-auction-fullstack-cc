package service

import (
	"context"
	"time"

	"auction-service/dao"
	"auction-service/model"
)

// DelayService 延时服务
type DelayService struct {
	auctionDAO *dao.AuctionDAO
}

// NewDelayService 创建延时服务
func NewDelayService(auctionDAO *dao.AuctionDAO) *DelayService {
	return &DelayService{
		auctionDAO: auctionDAO,
	}
}

// CheckDelayRequest 延时检查请求
type CheckDelayRequest struct {
	AuctionID int64
	Rule      *model.AuctionRule
}

// CheckDelayResult 延时检查结果
type CheckDelayResult struct {
	ShouldDelay    bool          `json:"should_delay"`
	DelayDuration  int           `json:"delay_duration"`
	NewEndTime     time.Time     `json:"new_end_time"`
	RemainingDelay int           `json:"remaining_delay"`
	MaxDelay       int           `json:"max_delay"`
}

// CheckAndTriggerDelay 检查并触发延时
func (s *DelayService) CheckAndTriggerDelay(ctx context.Context, req *CheckDelayRequest) (*CheckDelayResult, error) {
	// 获取竞拍信息
	auction, err := s.auctionDAO.GetByID(ctx, req.AuctionID)
	if err != nil {
		return nil, err
	}

	// 检查是否在延时窗口内
	sm := NewStateMachine(auction)
	if !sm.ShouldTriggerDelay(req.Rule.TriggerDelayBefore) {
		return &CheckDelayResult{
			ShouldDelay: false,
		}, nil
	}

	// 检查是否可以继续延时
	if !sm.CanDelay(req.Rule.MaxDelayTime) {
		return &CheckDelayResult{
			ShouldDelay:    false,
			RemainingDelay: 0,
			MaxDelay:       req.Rule.MaxDelayTime,
		}, nil
	}

	// 计算可延时时长
	remainingDelay := sm.GetRemainingDelayTime(req.Rule.MaxDelayTime)
	actualDelay := req.Rule.DelayDuration
	if actualDelay > remainingDelay {
		actualDelay = remainingDelay
	}

	// 执行延时
	if err := s.auctionDAO.ExtendEndTime(ctx, req.AuctionID, actualDelay); err != nil {
		return nil, err
	}

	// 获取更新后的竞拍信息
	auction, err = s.auctionDAO.GetByID(ctx, req.AuctionID)
	if err != nil {
		return nil, err
	}

	// 更新状态为延时中
	if auction.Status == model.AuctionStatusOngoing {
		if err := s.auctionDAO.UpdateStatus(ctx, req.AuctionID, model.AuctionStatusDelayed); err != nil {
			return nil, err
		}
	}

	return &CheckDelayResult{
		ShouldDelay:    true,
		DelayDuration:  actualDelay,
		NewEndTime:     auction.EndTime,
		RemainingDelay: remainingDelay - actualDelay,
		MaxDelay:       req.Rule.MaxDelayTime,
	}, nil
}

// IsInDelayWindow 检查是否在延时窗口内
func (s *DelayService) IsInDelayWindow(ctx context.Context, auctionID int64, triggerDelayBefore int) (bool, error) {
	auction, err := s.auctionDAO.GetByID(ctx, auctionID)
	if err != nil {
		return false, err
	}

	remaining := time.Until(auction.EndTime)
	return remaining.Seconds() <= float64(triggerDelayBefore) && remaining.Seconds() > 0, nil
}

// GetRemainingDelayTime 获取剩余可延时时长
func (s *DelayService) GetRemainingDelayTime(ctx context.Context, auctionID int64, maxDelayTime int) (int, error) {
	auction, err := s.auctionDAO.GetByID(ctx, auctionID)
	if err != nil {
		return 0, err
	}

	remaining := maxDelayTime - auction.DelayUsed
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}

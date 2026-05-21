package service

import (
	"context"
	"fmt"

	"auction-service/dao"
	"auction-service/lock"
	"auction-service/model"
)

// BidService 出价服务
type BidService struct {
	auctionDAO *dao.AuctionDAO
	bidDAO     *dao.BidDAO
	ruleDAO    *dao.AuctionRuleDAO
	userDAO    *dao.UserDAO
}

// NewBidService 创建出价服务
func NewBidService(auctionDAO *dao.AuctionDAO, bidDAO *dao.BidDAO, ruleDAO *dao.AuctionRuleDAO, userDAO *dao.UserDAO) *BidService {
	return &BidService{
		auctionDAO: auctionDAO,
		bidDAO:     bidDAO,
		ruleDAO:    ruleDAO,
		userDAO:    userDAO,
	}
}

// PlaceBidRequest 出价请求
type PlaceBidRequest struct {
	AuctionID int64
	UserID    int64
	Amount    float64
}

// PlaceBidResult 出价结果
type PlaceBidResult struct {
	Success      bool    `json:"success"`
	Message      string  `json:"message"`
	CurrentPrice float64 `json:"current_price"`
	Rank         int     `json:"rank"`
	WinnerID     int64   `json:"winner_id"`
}

// PlaceBid 出价
func (s *BidService) PlaceBid(ctx context.Context, req *PlaceBidRequest) (*PlaceBidResult, error) {
	// 1. 校验用户是否存在（逻辑外键校验）
	if s.userDAO != nil {
		exists, err := s.userDAO.Exists(ctx, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("校验用户失败: %w", err)
		}
		if !exists {
			return &PlaceBidResult{
				Success: false,
				Message: fmt.Sprintf("用户 %d 不存在，请先创建用户", req.UserID),
			}, nil
		}
	}

	// 2. 获取竞拍信息
	auction, err := s.auctionDAO.GetByID(ctx, req.AuctionID)
	if err != nil {
		return nil, fmt.Errorf("竞拍不存在: %w", err)
	}

	// 3. 检查竞拍状态
	sm := NewStateMachine(auction)
	if !sm.CanBid() {
		return &PlaceBidResult{
			Success: false,
			Message: "竞拍已结束或已取消，无法出价",
		}, nil
	}

	// 4. 获取竞拍规则
	rule, err := s.ruleDAO.GetByProductID(ctx, auction.ProductID)
	if err != nil {
		return nil, fmt.Errorf("获取竞拍规则失败: %w", err)
	}

	// 5. 校验出价金额
	minBidAmount := auction.CurrentPrice + rule.Increment
	if req.Amount < minBidAmount {
		return &PlaceBidResult{
			Success: false,
			Message: fmt.Sprintf("出价金额不足，最低出价为 %.2f 元", minBidAmount),
		}, nil
	}

	// 6. 检查封顶价
	if rule.CapPrice != nil && *rule.CapPrice > 0 && req.Amount >= *rule.CapPrice {
		// 达到封顶价，直接成交
		return s.handleCapPriceBid(ctx, auction, req, *rule.CapPrice)
	}

	// 7. 获取分布式锁
	bidLock := lock.NewAuctionBidLock(dao.GetRedis(), req.AuctionID)
	if err := bidLock.Acquire(ctx); err != nil {
		return &PlaceBidResult{
			Success: false,
			Message: "出价过于频繁，请稍后再试",
		}, nil
	}
	defer bidLock.Release(ctx)

	// 8. 重新获取竞拍信息（防止并发问题）
	auction, err = s.auctionDAO.GetByID(ctx, req.AuctionID)
	if err != nil {
		return nil, err
	}

	// 9. 再次校验（双重检查）
	minBidAmount = auction.CurrentPrice + rule.Increment
	if req.Amount < minBidAmount {
		return &PlaceBidResult{
			Success: false,
			Message: fmt.Sprintf("已被超越，当前最低出价为 %.2f 元", minBidAmount),
		}, nil
	}

	// 10. 创建出价记录
	bid := &model.Bid{
		AuctionID: req.AuctionID,
		UserID:    req.UserID,
		Amount:    req.Amount,
	}

	if err := s.bidDAO.Create(ctx, bid); err != nil {
		return nil, fmt.Errorf("创建出价记录失败: %w", err)
	}

	// 11. 更新竞拍当前价格和中标者
	if err := s.auctionDAO.UpdatePrice(ctx, req.AuctionID, req.Amount, req.UserID); err != nil {
		return nil, fmt.Errorf("更新竞拍价格失败: %w", err)
	}

	// 12. 获取用户排名
	rank, err := s.getUserRank(ctx, req.AuctionID, req.UserID)
	if err != nil {
		rank = 1 // 默认第一名
	}

	// 13. 返回成功结果
	return &PlaceBidResult{
		Success:      true,
		Message:      "出价成功",
		CurrentPrice: req.Amount,
		Rank:         rank,
		WinnerID:     req.UserID,
	}, nil
}

// handleCapPriceBid 处理封顶价出价
func (s *BidService) handleCapPriceBid(ctx context.Context, auction *model.Auction, req *PlaceBidRequest, capPrice float64) (*PlaceBidResult, error) {
	// 创建出价记录
	bid := &model.Bid{
		AuctionID: req.AuctionID,
		UserID:    req.UserID,
		Amount:    capPrice,
	}

	if err := s.bidDAO.Create(ctx, bid); err != nil {
		return nil, fmt.Errorf("创建出价记录失败: %w", err)
	}

	// 更新竞拍价格和状态
	if err := s.auctionDAO.UpdatePrice(ctx, req.AuctionID, capPrice, req.UserID); err != nil {
		return nil, fmt.Errorf("更新竞拍价格失败: %w", err)
	}

	// 更新竞拍状态为已结束
	if err := s.auctionDAO.UpdateStatus(ctx, req.AuctionID, model.AuctionStatusEnded); err != nil {
		return nil, fmt.Errorf("更新竞拍状态失败: %w", err)
	}

	return &PlaceBidResult{
		Success:      true,
		Message:      "达到封顶价，竞拍成交！",
		CurrentPrice: capPrice,
		Rank:         1,
		WinnerID:     req.UserID,
	}, nil
}

// getUserRank 获取用户排名
func (s *BidService) getUserRank(ctx context.Context, auctionID, userID int64) (int, error) {
	bids, err := s.bidDAO.GetRanking(ctx, auctionID, 100)
	if err != nil {
		return 0, err
	}

	for i, bid := range bids {
		if bid.UserID == userID {
			return i + 1, nil
		}
	}

	return 0, nil
}

// GetRanking 获取出价排名
func (s *BidService) GetRanking(ctx context.Context, auctionID int64, limit int) ([]model.Bid, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	return s.bidDAO.GetRanking(ctx, auctionID, limit)
}

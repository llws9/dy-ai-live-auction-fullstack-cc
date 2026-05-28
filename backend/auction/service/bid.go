package service

import (
	"context"
	"fmt"
	"time"

	"auction-service/dao"
	"auction-service/lock"
	"auction-service/model"
	"auction-service/pkg/metrics"
	"auction-service/websocket"
)

// BidService 出价服务
type BidService struct {
	auctionDAO         *dao.AuctionDAO
	bidDAO             *dao.BidDAO
	ruleDAO            *dao.AuctionRuleDAO
	userDAO            *dao.UserDAO
	hub                *websocket.Hub
	rankThrottle       *RankingThrottle
	notificationSender NotificationSender       // 通知发送接口
	metrics            *metrics.AuctionMetrics  // 新增：指标收集器
	skyLampTrigger     SkyLampTrigger           // 点天灯触发接口
}

// SkyLampTrigger 点天灯触发接口
type SkyLampTrigger interface {
	TriggerAutoBid(ctx context.Context, auctionID int64, currentPrice float64, increment float64) error
}

// NewBidService 创建出价服务
func NewBidService(auctionDAO *dao.AuctionDAO, bidDAO *dao.BidDAO, ruleDAO *dao.AuctionRuleDAO, userDAO *dao.UserDAO) *BidService {
	return &BidService{
		auctionDAO:   auctionDAO,
		bidDAO:       bidDAO,
		ruleDAO:      ruleDAO,
		userDAO:      userDAO,
		rankThrottle: NewRankingThrottle(),
	}
}

// SetMetrics 设置指标收集器
func (s *BidService) SetMetrics(m *metrics.AuctionMetrics) {
	s.metrics = m
}

// SetHub 设置 WebSocket Hub
func (s *BidService) SetHub(hub *websocket.Hub) {
	s.hub = hub
}

// SetNotificationSender 设置通知发送服务
func (s *BidService) SetNotificationSender(sender NotificationSender) {
	s.notificationSender = sender
}

// SetSkyLampTrigger 设置点天灯触发服务
func (s *BidService) SetSkyLampTrigger(trigger SkyLampTrigger) {
	s.skyLampTrigger = trigger
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
	start := time.Now()

	// 记录并发出价计数（新增）
	if s.metrics != nil {
		s.metrics.IncConcurrentBids()
	}
	var concurrentDecreased bool
	defer func() {
		if !concurrentDecreased && s.metrics != nil {
			s.metrics.DecConcurrentBids()
		}
	}()

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
	// 保存前中标者信息，用于发送出价超越通知
	previousWinnerID := auction.WinnerID
	previousPrice := auction.CurrentPrice

	if err := s.auctionDAO.UpdatePrice(ctx, req.AuctionID, req.Amount, req.UserID); err != nil {
		return nil, fmt.Errorf("更新竞拍价格失败: %w", err)
	}

	// 11.1 发送出价超越通知
	if s.notificationSender != nil && previousWinnerID != nil && *previousWinnerID > 0 && *previousWinnerID != req.UserID {
		notifyUserID := *previousWinnerID
		go func() {
			// 异步发送通知，不阻塞主流程
			_ = s.notificationSender.SendNotification(ctx, &model.NotificationRequest{
				UserID:  notifyUserID,
				Type:    model.NotificationTypeBidOutbid,
				Title:   "出价被超越",
				Content: fmt.Sprintf("您在竞拍中的出价 %.2f 元已被超越，当前最高价为 %.2f 元", previousPrice, req.Amount),
				Data: map[string]interface{}{
					"auction_id": req.AuctionID,
					"old_bid":    previousPrice,
					"new_bid":    req.Amount,
				},
			})
		}()
	}

	// 11.2 触发点天灯自动跟价
	if s.skyLampTrigger != nil {
		go func() {
			// 异步触发，不阻塞主流程
			_ = s.skyLampTrigger.TriggerAutoBid(ctx, req.AuctionID, req.Amount, rule.Increment)
		}()
	}

	// 11.5 检查并触发延时
	if rule.TriggerDelayBefore > 0 {
		// 重新获取竞拍信息
		auction, err = s.auctionDAO.GetByID(ctx, req.AuctionID)
		if err == nil {
			sm := NewStateMachine(auction)
			if sm.ShouldTriggerDelay(rule.TriggerDelayBefore) && sm.CanDelay(rule.MaxDelayTime) {
				// 计算可延时时长
				remainingDelay := sm.GetRemainingDelayTime(rule.MaxDelayTime)
				actualDelay := rule.DelayDuration
				if actualDelay > remainingDelay {
					actualDelay = remainingDelay
				}

				// 执行延时
				if err := s.auctionDAO.ExtendEndTime(ctx, req.AuctionID, actualDelay); err == nil {
					// 更新状态为延时中
					if auction.Status == model.AuctionStatusOngoing {
						s.auctionDAO.UpdateStatus(ctx, req.AuctionID, model.AuctionStatusDelayed)
					}
					fmt.Printf("Auction %d delayed by %d seconds\n", req.AuctionID, actualDelay)
					// 记录延时触发指标（新增）
					if s.metrics != nil {
						s.metrics.RecordDelayTriggered(req.AuctionID)
					}
				}
			}
		}
	}

	// 12. 获取用户排名
	rank, err := s.getUserRank(ctx, req.AuctionID, req.UserID)
	if err != nil {
		rank = 1 // 默认第一名
	}

	// 13. 广播排名更新
	s.broadcastRanking(ctx, req.AuctionID)

	// 14. 返回成功结果
	// 记录出价成功指标（新增）
	if s.metrics != nil {
		s.metrics.RecordBidLatency(req.AuctionID, start, true)
		s.metrics.RecordBid(req.AuctionID, req.Amount, true)
		s.metrics.RecordBidUser(req.AuctionID)
	}
	concurrentDecreased = true // 标记已处理

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

// broadcastRanking 广播排名更新
func (s *BidService) broadcastRanking(ctx context.Context, auctionID int64) {
	if s.hub == nil {
		return
	}

	// 节流检查：每 200ms 最多推送一次
	if !s.rankThrottle.ShouldSend(auctionID) {
		return
	}

	// 获取排名列表
	bids, err := s.bidDAO.GetRanking(ctx, auctionID, 10)
	if err != nil {
		return
	}

	// 转换为排名项
	rankItems := make([]websocket.RankItem, len(bids))
	for i, bid := range bids {
		rankItems[i] = websocket.RankItem{
			Rank:   i + 1,
			UserID: bid.UserID,
			Amount: bid.Amount,
		}

		// 获取用户名
		if s.userDAO != nil {
			user, err := s.userDAO.GetByID(ctx, bid.UserID)
			if err == nil {
				rankItems[i].UserName = user.Name
			}
		}
	}

	// 广播排名更新消息
	message := websocket.NewRankUpdateMessage(auctionID, rankItems)
	s.hub.BroadcastToRoom(auctionID, message)
}
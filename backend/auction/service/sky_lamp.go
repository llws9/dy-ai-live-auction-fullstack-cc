package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"auction-service/config"
	"auction-service/dao"
	"auction-service/model"
	"auction-service/pkg/metrics"
	"auction-service/websocket"

	"github.com/shopspring/decimal"
)

// SkyLampService 点天灯服务
type SkyLampService struct {
	skyLampDAO *dao.SkyLampDAO
	bidService *BidService
	metrics    *metrics.SkyLampMetrics
	cfg        config.SkyLampConfig
	lockSvc    *DistributedLockService
	hub        *websocket.Hub
}

// NewSkyLampService 创建点天灯服务
func NewSkyLampService(skyLampDAO *dao.SkyLampDAO, bidService *BidService, cfg config.SkyLampConfig, lockSvc *DistributedLockService) *SkyLampService {
	return &SkyLampService{
		skyLampDAO: skyLampDAO,
		bidService: bidService,
		cfg:        cfg,
		lockSvc:    lockSvc,
	}
}

// SetMetrics 设置指标收集器
func (s *SkyLampService) SetMetrics(m *metrics.SkyLampMetrics) {
	if m == nil {
		log.Println("WARNING: SkyLampService metrics not initialized")
	}
	s.metrics = m
}

func (s *SkyLampService) SetHub(hub *websocket.Hub) {
	s.hub = hub
}

// StartSubscription 开启点天灯订阅
func (s *SkyLampService) StartSubscription(ctx context.Context, userID, auctionID int64) (*model.SkyLampSubscription, error) {
	startTime := time.Now()

	if !s.cfg.Enabled {
		return nil, errors.New("点天灯功能未开启")
	}

	// 获取分布式锁，防止并发创建订阅
	if s.lockSvc != nil {
		lockKey := fmt.Sprintf("skylamp:subscribe:%d:%d", userID, auctionID)
		acquired, err := s.lockSvc.AcquireLock(ctx, lockKey, 5*time.Second)
		if err != nil {
			log.Printf("SkyLamp订阅锁获取失败: user=%d, auction=%d, err=%v", userID, auctionID, err)
		}
		if !acquired {
			return nil, errors.New("请稍后重试")
		}
		defer s.lockSvc.ReleaseLock(ctx, lockKey)
	}

	auction, err := s.bidService.auctionDAO.GetByID(ctx, auctionID)
	if err != nil {
		return nil, fmt.Errorf("竞拍不存在: %w", err)
	}
	if !NewStateMachine(auction).CanBid() {
		return nil, errors.New("当前竞拍状态不可开启点天灯")
	}

	// 检查是否已有活跃订阅
	existing, err := s.skyLampDAO.GetActiveByUser(ctx, auctionID, userID)
	if err != nil {
		return nil, fmt.Errorf("检查订阅失败: %w", err)
	}
	if existing != nil {
		return nil, errors.New("已有活跃的点天灯订阅")
	}

	rule, err := s.bidService.ruleDAO.GetByProductID(ctx, auction.ProductID)
	if err != nil {
		return nil, fmt.Errorf("获取竞拍规则失败: %w", err)
	}
	if rule == nil {
		return nil, errors.New("竞拍规则不存在")
	}

	initialPrice := auction.CurrentPrice
	initialBidAmount := minimumBidAmount(auction.CurrentPrice, rule.StartPrice, rule.Increment)
	if rule.CapPrice != nil && rule.CapPrice.GreaterThan(decimal.Zero) && initialBidAmount.GreaterThan(*rule.CapPrice) {
		return nil, errors.New("当前竞拍已达到封顶价，无法开启点天灯")
	}

	maxPriceLimit := initialPrice.Add(decimal.NewFromInt(int64(s.cfg.MaxPriceOffset)))
	if maxPriceLimit.LessThan(initialBidAmount) {
		maxPriceLimit = initialBidAmount
	}
	if rule.CapPrice != nil && rule.CapPrice.GreaterThan(decimal.Zero) && maxPriceLimit.GreaterThan(*rule.CapPrice) {
		maxPriceLimit = *rule.CapPrice
	}

	// 创建订阅
	subscription := &model.SkyLampSubscription{
		AuctionID:           auctionID,
		UserID:              userID,
		Status:              model.SkyLampStatusActive,
		InitialPrice:        decimalToFloat(initialPrice),
		InitialBidAmount:    decimalToFloat(initialBidAmount),
		MaxPriceLimit:       decimalToFloat(maxPriceLimit),
		CurrentAutoBidCount: 0,
		TotalBidAmount:      0,
	}

	// 拆分长事务：避免在事务中调用 PlaceBid（PlaceBid 包含独立事务、Redis 锁、WebSocket 推送）
	// 改为补偿模式：先短事务建订阅 → 调用 PlaceBid → 失败则删除订阅、成功则更新统计
	// Step 1: 短事务创建订阅
	if err := s.skyLampDAO.Create(ctx, subscription); err != nil {
		if s.metrics != nil {
			s.metrics.RecordSubscriptionFailed(auctionID, "create_failed")
		}
		return nil, fmt.Errorf("创建订阅失败: %w", err)
	}

	// Step 2: 在事务外执行首次出价
	result, bidErr := s.bidService.PlaceBid(ctx, &PlaceBidRequest{
		AuctionID:          auctionID,
		UserID:             userID,
		Amount:             initialBidAmount,
		SkipSkyLampTrigger: true, // 避免递归触发
	})
	if bidErr != nil || result == nil || !result.Success {
		// 首次出价失败：补偿删除订阅
		// 使用独立 context 防止主 ctx 已取消时无法清理
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if delErr := s.skyLampDAO.Delete(cleanupCtx, subscription.ID); delErr != nil {
			log.Printf("SkyLamp首次出价失败后删除订阅失败: id=%d, err=%v", subscription.ID, delErr)
		}
		cancel()

		if s.metrics != nil {
			s.metrics.RecordSubscriptionFailed(auctionID, "first_bid_failed")
		}
		if bidErr != nil {
			return nil, fmt.Errorf("首次出价失败: %w", bidErr)
		}
		return nil, fmt.Errorf("首次出价失败: %s", result.Message)
	}

	// Step 3: 短事务更新订阅统计（失败仅记录日志，订阅本体已可用）
	subscription.TotalBidAmount = decimalToFloat(initialBidAmount)
	if err := s.skyLampDAO.Update(ctx, subscription); err != nil {
		log.Printf("SkyLamp订阅统计更新失败（非致命）: id=%d, err=%v", subscription.ID, err)
	}

	// 记录订阅创建成功指标
	if s.metrics != nil {
		s.metrics.RecordSubscriptionCreated(auctionID, userID, decimalToFloat(maxPriceLimit))
	}

	log.Printf("SkyLamp订阅创建成功: user=%d, auction=%d, max=%s, latency=%v", userID, auctionID, maxPriceLimit.StringFixed(2), time.Since(startTime))
	return subscription, nil
}

// StopSubscription 停止点天灯订阅（用户主动）
func (s *SkyLampService) StopSubscription(ctx context.Context, userID, subscriptionID int64) error {
	subscription, err := s.skyLampDAO.GetByID(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("获取订阅失败: %w", err)
	}

	if subscription.UserID != userID {
		return errors.New("无权操作此订阅")
	}

	if !subscription.IsActive() {
		return errors.New("订阅已不活跃")
	}

	// 计算订阅时长
	durationSeconds := time.Since(subscription.CreatedAt).Seconds()

	if err := s.skyLampDAO.UpdateStatusWithStoppedAt(ctx, subscriptionID, model.SkyLampStatusCancelled); err != nil {
		return fmt.Errorf("停止订阅失败: %w", err)
	}

	// 记录订阅停止指标
	if s.metrics != nil {
		s.metrics.RecordSubscriptionStopped(subscription.AuctionID, userID, durationSeconds, "user_action")
	}

	log.Printf("SkyLamp订阅已停止: id=%d, user=%d, duration=%v", subscriptionID, userID, durationSeconds)
	return nil
}

// TriggerAutoBid 触发自动跟价（被其他出价超越时调用）
// 改进：将 per-auction 大锁拆为 per-subscription 细粒度锁，并发处理多个订阅
func (s *SkyLampService) TriggerAutoBid(ctx context.Context, auctionID int64, currentPrice float64, increment float64) error {
	_ = currentPrice
	_ = increment

	auction, err := s.bidService.auctionDAO.GetByID(ctx, auctionID)
	if err != nil {
		return fmt.Errorf("获取竞拍失败: %w", err)
	}
	if !NewStateMachine(auction).CanBid() {
		return nil
	}

	rule, err := s.bidService.ruleDAO.GetByProductID(ctx, auction.ProductID)
	if err != nil {
		return fmt.Errorf("获取竞拍规则失败: %w", err)
	}
	if rule == nil {
		return errors.New("竞拍规则不存在")
	}

	subscriptions, err := s.skyLampDAO.GetActiveByAuction(ctx, auctionID)
	if err != nil {
		return fmt.Errorf("获取订阅失败: %w", err)
	}
	if len(subscriptions) == 0 {
		return nil
	}

	// 并发处理订阅，使用信号量限制并发数避免压垮 DB
	// 信号量获取放到 goroutine 内，避免阻塞主 goroutine 派发循环
	const maxConcurrency = 8
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i := range subscriptions {
		sub := &subscriptions[i]

		wg.Add(1)
		go func(sub *model.SkyLampSubscription) {
			defer wg.Done()

			// 在 goroutine 内获取信号量，主 goroutine 不被阻塞
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			s.processOneSubscription(ctx, auctionID, sub, decimalToFloat(rule.Increment))
		}(sub)
	}

	wg.Wait()
	return nil
}

// processOneSubscription 处理单个订阅的自动跟价（带 per-subscription 锁）
func (s *SkyLampService) processOneSubscription(ctx context.Context, auctionID int64, sub *model.SkyLampSubscription, increment float64) {
	autoBidStartTime := time.Now()

	// 细粒度锁：仅锁定当前订阅，不阻塞其他订阅处理
	if s.lockSvc != nil {
		lockKey := fmt.Sprintf("skylamp:sub:%d", sub.ID)
		acquired, err := s.lockSvc.AcquireLock(ctx, lockKey, 3*time.Second)
		if err != nil {
			log.Printf("SkyLamp订阅锁获取失败: sub=%d, err=%v", sub.ID, err)
			return
		}
		if !acquired {
			// 其他 goroutine 正在处理同一订阅，跳过
			return
		}
		defer s.lockSvc.ReleaseLock(ctx, lockKey)
	}

	// 重新拉取最新订阅与竞拍状态（避免使用陈旧数据）
	latestSub, err := s.skyLampDAO.GetByID(ctx, sub.ID)
	if err != nil || latestSub == nil || !latestSub.IsActive() {
		return
	}
	auction, err := s.bidService.auctionDAO.GetByID(ctx, auctionID)
	if err != nil || !NewStateMachine(auction).CanBid() {
		return
	}

	// 自身已是最高价，无需跟价
	if auction.WinnerID != nil && *auction.WinnerID == latestSub.UserID {
		return
	}

	// 达到次数上限：停止订阅
	if latestSub.CurrentAutoBidCount >= s.cfg.MaxAutoBidCount {
		durationSeconds := time.Since(latestSub.CreatedAt).Seconds()
		if err := s.skyLampDAO.UpdateStatusWithStoppedAt(ctx, latestSub.ID, model.SkyLampStatusStopped); err != nil {
			log.Printf("达到最大自动跟价次数后停止订阅失败: id=%d, err=%v", latestSub.ID, err)
		} else if s.metrics != nil {
			s.metrics.RecordSubscriptionLimitReached(auctionID, latestSub.UserID, durationSeconds)
		}
		return
	}

	// 达到价格上限：停止订阅
	currentPrice := decimalToFloat(auction.CurrentPrice)
	if !latestSub.CanAutoBid(currentPrice) {
		durationSeconds := time.Since(latestSub.CreatedAt).Seconds()
		if err := s.skyLampDAO.UpdateStatusWithStoppedAt(ctx, latestSub.ID, model.SkyLampStatusStopped); err != nil {
			log.Printf("停止订阅失败: id=%d, err=%v", latestSub.ID, err)
		} else if s.metrics != nil {
			s.metrics.RecordSubscriptionLimitReached(auctionID, latestSub.UserID, durationSeconds)
		}
		return
	}

	// 计算下次出价金额
	nextBid := latestSub.GetNextBidAmount(currentPrice, increment)
	if nextBid > latestSub.MaxPriceLimit {
		nextBid = latestSub.MaxPriceLimit
	}
	if nextBid <= currentPrice {
		return
	}

	// 执行出价
	result, err := s.bidService.PlaceBid(ctx, &PlaceBidRequest{
		AuctionID:          auctionID,
		UserID:             latestSub.UserID,
		Amount:             decimal.NewFromFloat(nextBid),
		SkipSkyLampTrigger: true,
	})
	if err != nil || result == nil || !result.Success {
		if s.metrics != nil {
			errorType := "bid_failed"
			if err != nil {
				errorType = "bid_error"
			}
			s.metrics.RecordAutoBidFailed(auctionID, latestSub.ID, errorType)
		}
		if err != nil {
			log.Printf("SkyLamp自动出价失败: sub=%d, user=%d, amount=%f, err=%v", latestSub.ID, latestSub.UserID, nextBid, err)
		}
		return
	}

	// 更新订阅统计
	latestSub.CurrentAutoBidCount++
	latestSub.TotalBidAmount += nextBid
	if err := s.skyLampDAO.Update(ctx, latestSub); err != nil {
		log.Printf("更新订阅失败: id=%d, err=%v", latestSub.ID, err)
	}

	if s.metrics != nil {
		s.metrics.RecordAutoBidSuccess(auctionID, latestSub.UserID, latestSub.ID, nextBid, time.Since(autoBidStartTime))
	}
	s.broadcastAutoBid(auctionID, latestSub.UserID, nextBid, latestSub.MaxPriceLimit, latestSub.CurrentAutoBidCount)

	log.Printf("SkyLamp自动出价成功: sub=%d, user=%d, amount=%f, count=%d", latestSub.ID, latestSub.UserID, nextBid, latestSub.CurrentAutoBidCount)
}

func (s *SkyLampService) broadcastAutoBid(auctionID, userID int64, amount, maxPriceLimit float64, autoBidCount int) {
	if s.hub == nil {
		return
	}

	amountDecimal := decimal.NewFromFloat(amount)
	remainingBudget := decimal.NewFromFloat(maxPriceLimit).Sub(amountDecimal)
	if remainingBudget.IsNegative() {
		remainingBudget = decimal.Zero
	}

	msg := websocket.NewSkyLampAutoBidMessage(auctionID, userID, amountDecimal, remainingBudget, autoBidCount)
	_ = s.hub.TryBroadcastToRoom(auctionID, msg)
}

// GetUserSubscriptions 获取用户的点天灯订阅列表
func (s *SkyLampService) GetUserSubscriptions(ctx context.Context, userID int64, status model.SkyLampStatus, page, pageSize int) ([]model.SkyLampSubscription, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	return s.skyLampDAO.GetByUserStatus(ctx, userID, status, page, pageSize)
}

// GetSubscriptionDetail 获取订阅详情
func (s *SkyLampService) GetSubscriptionDetail(ctx context.Context, subscriptionID int64, userID int64) (*model.SkyLampSubscription, error) {
	subscription, err := s.skyLampDAO.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("获取订阅失败: %w", err)
	}

	if subscription.UserID != userID {
		return nil, errors.New("无权查看此订阅")
	}

	return subscription, nil
}

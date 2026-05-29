package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"auction-service/config"
	"auction-service/dao"
	"auction-service/model"
)

// SkyLampService 点天灯服务
type SkyLampService struct {
	skyLampDAO *dao.SkyLampDAO
	bidService *BidService
	cfg        config.SkyLampConfig
}

// NewSkyLampService 创建点天灯服务
func NewSkyLampService(skyLampDAO *dao.SkyLampDAO, bidService *BidService, cfg config.SkyLampConfig) *SkyLampService {
	return &SkyLampService{
		skyLampDAO: skyLampDAO,
		bidService: bidService,
		cfg:        cfg,
	}
}

// StartSubscription 开启点天灯订阅
func (s *SkyLampService) StartSubscription(ctx context.Context, userID, auctionID int64) (*model.SkyLampSubscription, error) {
	startTime := time.Now()

	if !s.cfg.Enabled {
		return nil, errors.New("点天灯功能未开启")
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

	initialPrice := auction.CurrentPrice
	initialBidAmount := initialPrice + rule.Increment
	if rule.CapPrice != nil && *rule.CapPrice > 0 && initialBidAmount > *rule.CapPrice {
		return nil, errors.New("当前竞拍已达到封顶价，无法开启点天灯")
	}

	maxPriceLimit := initialPrice + float64(s.cfg.MaxPriceOffset)
	if maxPriceLimit < initialBidAmount {
		maxPriceLimit = initialBidAmount
	}
	if rule.CapPrice != nil && *rule.CapPrice > 0 && maxPriceLimit > *rule.CapPrice {
		maxPriceLimit = *rule.CapPrice
	}

	// 创建订阅
	subscription := &model.SkyLampSubscription{
		AuctionID:           auctionID,
		UserID:              userID,
		Status:              model.SkyLampStatusActive,
		InitialPrice:        initialPrice,
		InitialBidAmount:    initialBidAmount,
		MaxPriceLimit:       maxPriceLimit,
		CurrentAutoBidCount: 0,
		TotalBidAmount:      0,
	}

	if err := s.skyLampDAO.Create(ctx, subscription); err != nil {
		return nil, fmt.Errorf("创建订阅失败: %w", err)
	}

	// 立即执行首次出价
	result, err := s.bidService.PlaceBid(ctx, &PlaceBidRequest{
		AuctionID:          auctionID,
		UserID:             userID,
		Amount:             initialBidAmount,
		SkipSkyLampTrigger: true, // 避免递归触发
	})
	if err != nil || result == nil || !result.Success {
		// 首次出价失败，删除订阅避免状态不一致
		if delErr := s.skyLampDAO.Delete(ctx, subscription.ID); delErr != nil {
			log.Printf("SkyLamp订阅回滚失败: sub=%d, err=%v", subscription.ID, delErr)
			_ = s.skyLampDAO.UpdateStatusWithStoppedAt(ctx, subscription.ID, model.SkyLampStatusCancelled)
		}


		if err != nil {
			return nil, fmt.Errorf("首次出价失败: %w", err)
		}
		return nil, fmt.Errorf("首次出价失败: %s", result.Message)
	}

	subscription.TotalBidAmount = initialBidAmount
	if err := s.skyLampDAO.Update(ctx, subscription); err != nil {
		return nil, fmt.Errorf("更新订阅统计失败: %w", err)
	}


	log.Printf("SkyLamp订阅创建成功: user=%d, auction=%d, max=%f, latency=%v", userID, auctionID, maxPriceLimit, time.Since(startTime))
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


	log.Printf("SkyLamp订阅已停止: id=%d, user=%d, duration=%v", subscriptionID, userID, durationSeconds)
	return nil
}

// TriggerAutoBid 触发自动跟价（被其他出价超越时调用）
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

	subscriptions, err := s.skyLampDAO.GetActiveByAuction(ctx, auctionID)
	if err != nil {
		return fmt.Errorf("获取订阅失败: %w", err)
	}
	if len(subscriptions) == 0 {
		return nil
	}

	for _, sub := range subscriptions {
		auction, err = s.bidService.auctionDAO.GetByID(ctx, auctionID)
		if err != nil {
			log.Printf("SkyLamp自动跟价读取竞拍失败: auction=%d, err=%v", auctionID, err)
			continue
		}
		if !NewStateMachine(auction).CanBid() {
			return nil
		}

		if auction.WinnerID != nil && *auction.WinnerID == sub.UserID {
			continue
		}

		if sub.CurrentAutoBidCount >= s.cfg.MaxAutoBidCount {
			if err := s.skyLampDAO.UpdateStatusWithStoppedAt(ctx, sub.ID, model.SkyLampStatusStopped); err != nil {
				log.Printf("达到最大自动跟价次数后停止订阅失败: id=%d, err=%v", sub.ID, err)
			}
			continue
		}

		if !sub.CanAutoBid(auction.CurrentPrice) {
			if err := s.skyLampDAO.UpdateStatusWithStoppedAt(ctx, sub.ID, model.SkyLampStatusStopped); err != nil {
				log.Printf("停止订阅失败: id=%d, err=%v", sub.ID, err)
			}
			continue
		}

		nextBid := sub.GetNextBidAmount(auction.CurrentPrice, rule.Increment)
		if nextBid > sub.MaxPriceLimit {
			nextBid = sub.MaxPriceLimit
		}
		if nextBid <= auction.CurrentPrice {
			continue
		}

		result, err := s.bidService.PlaceBid(ctx, &PlaceBidRequest{
			AuctionID:          auctionID,
			UserID:             sub.UserID,
			Amount:             nextBid,
			SkipSkyLampTrigger: true,
		})
		if err != nil || result == nil || !result.Success {
			if err != nil {
				log.Printf("SkyLamp自动出价失败: sub=%d, user=%d, amount=%f, err=%v", sub.ID, sub.UserID, nextBid, err)
			}
			continue
		}

		sub.CurrentAutoBidCount++
		sub.TotalBidAmount += nextBid
		if err := s.skyLampDAO.Update(ctx, &sub); err != nil {
			log.Printf("更新订阅失败: id=%d, err=%v", sub.ID, err)
		}

		if s.cfg.MinFollowInterval > 0 {
			time.Sleep(time.Duration(s.cfg.MinFollowInterval) * time.Millisecond)
		}

		log.Printf("SkyLamp自动出价成功: sub=%d, user=%d, amount=%f, count=%d", sub.ID, sub.UserID, nextBid, sub.CurrentAutoBidCount)
	}

	return nil
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

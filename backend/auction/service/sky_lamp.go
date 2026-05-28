package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"auction-service/dao"
	"auction-service/model"
)

// SkyLampService 点天灯服务
type SkyLampService struct {
	skyLampDAO *dao.SkyLampDAO
	bidService *BidService
}

// NewSkyLampService 创建点天灯服务
func NewSkyLampService(skyLampDAO *dao.SkyLampDAO, bidService *BidService) *SkyLampService {
	return &SkyLampService{
		skyLampDAO: skyLampDAO,
		bidService: bidService,
	}
}

// StartSubscription 开启点天灯订阅
func (s *SkyLampService) StartSubscription(ctx context.Context, userID, auctionID int64, initialPrice, initialBidAmount, maxPriceLimit float64) (*model.SkyLampSubscription, error) {
	// 检查是否已有活跃订阅
	existing, err := s.skyLampDAO.GetActiveByUser(ctx, auctionID, userID)
	if err != nil {
		return nil, fmt.Errorf("检查订阅失败: %w", err)
	}
	if existing != nil {
		return nil, errors.New("已有活跃的点天灯订阅")
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
	if _, err := s.bidService.PlaceBid(ctx, &PlaceBidRequest{
		AuctionID: auctionID,
		UserID:    userID,
		Amount:    initialBidAmount,
	}); err != nil {
		log.Printf("SkyLamp首次出价失败: %v", err)
		// 不返回错误，订阅已创建成功，后续可继续自动跟价
	}

	subscription.TotalBidAmount = initialBidAmount

	log.Printf("SkyLamp订阅创建成功: user=%d, auction=%d, max=%f", userID, auctionID, maxPriceLimit)
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

	if err := s.skyLampDAO.UpdateStatus(ctx, subscriptionID, model.SkyLampStatusCancelled); err != nil {
		return fmt.Errorf("停止订阅失败: %w", err)
	}

	log.Printf("SkyLamp订阅已停止: id=%d, user=%d", subscriptionID, userID)
	return nil
}

// TriggerAutoBid 触发自动跟价（被其他出价超越时调用）
func (s *SkyLampService) TriggerAutoBid(ctx context.Context, auctionID int64, currentPrice float64, increment float64) error {
	// 获取所有活跃订阅
	subscriptions, err := s.skyLampDAO.GetActiveByAuction(ctx, auctionID)
	if err != nil {
		return fmt.Errorf("获取订阅失败: %w", err)
	}

	if len(subscriptions) == 0 {
		return nil // 无活跃订阅
	}

	for _, sub := range subscriptions {
		if !sub.CanAutoBid(currentPrice) {
			// 超过上限，停止订阅
			if err := s.skyLampDAO.UpdateStatus(ctx, sub.ID, model.SkyLampStatusStopped); err != nil {
				log.Printf("停止订阅失败: id=%d, err=%v", sub.ID, err)
			}
			continue
		}

		nextBid := sub.GetNextBidAmount(currentPrice, increment)
		if nextBid > sub.MaxPriceLimit {
			nextBid = sub.MaxPriceLimit // 最后一次出价，达到上限
		}

		// 执行自动出价
		if _, err := s.bidService.PlaceBid(ctx, &PlaceBidRequest{
			AuctionID: auctionID,
			UserID:    sub.UserID,
			Amount:    nextBid,
		}); err != nil {
			log.Printf("SkyLamp自动出价失败: sub=%d, user=%d, amount=%f, err=%v", sub.ID, sub.UserID, nextBid, err)
			continue
		}

		// 更新订阅状态
		sub.CurrentAutoBidCount++
		sub.TotalBidAmount += nextBid
		if err := s.skyLampDAO.Update(ctx, &sub); err != nil {
			log.Printf("更新订阅失败: id=%d, err=%v", sub.ID, err)
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
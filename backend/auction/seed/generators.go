package main

import (
	"fmt"
	"math/rand"
	"time"

	"auction-service/model"
)

// SeedConfig 种子数据配置
type SeedConfig struct {
	Size string

	// 数据数量配置
	AuctionsCount        int
	BidsPerAuction      int
	NotificationsCount  int
	SkyLampCount        int
	FollowCount         int
	ReminderCount       int

	// 比例配置
	PendingRatio  float64
	OngoingRatio  float64
	DelayedRatio  float64
	EndedRatio    float64
	CancelledRatio float64
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig(size string) *SeedConfig {
	cfg := &SeedConfig{Size: size}

	switch size {
	case "small":
		cfg.AuctionsCount = 10
		cfg.BidsPerAuction = 5
		cfg.NotificationsCount = 20
		cfg.SkyLampCount = 5
		cfg.FollowCount = 15
		cfg.ReminderCount = 10
	case "large":
		cfg.AuctionsCount = 50
		cfg.BidsPerAuction = 20
		cfg.NotificationsCount = 100
		cfg.SkyLampCount = 30
		cfg.FollowCount = 100
		cfg.ReminderCount = 50
	default: // medium
		cfg.AuctionsCount = 20
		cfg.BidsPerAuction = 10
		cfg.NotificationsCount = 40
		cfg.SkyLampCount = 10
		cfg.FollowCount = 30
		cfg.ReminderCount = 20
	}

	cfg.PendingRatio = 0.2
	cfg.OngoingRatio = 0.3
	cfg.DelayedRatio = 0.1
	cfg.EndedRatio = 0.3
	cfg.CancelledRatio = 0.1

	return cfg
}

// GenerateAuctions 生成竞拍数据
func GenerateAuctions(cfg *SeedConfig, productIDs, liveStreamIDs, creatorIDs []int64) []model.Auction {
	auctions := make([]model.Auction, 0, cfg.AuctionsCount)
	now := time.Now()
	r := rand.New(rand.NewSource(now.UnixNano()))

	pendingCount := int(float64(cfg.AuctionsCount) * cfg.PendingRatio)
	ongoingCount := int(float64(cfg.AuctionsCount) * cfg.OngoingRatio)
	delayedCount := int(float64(cfg.AuctionsCount) * cfg.DelayedRatio)
	endedCount := int(float64(cfg.AuctionsCount) * cfg.EndedRatio)
	cancelledCount := cfg.AuctionsCount - pendingCount - ongoingCount - delayedCount - endedCount

	for i := 0; i < pendingCount; i++ {
		startTime := now.Add(time.Hour * time.Duration(1+r.Intn(24)))
		endTime := startTime.Add(time.Minute * time.Duration(5+r.Intn(10)))
		creatorID := creatorIDs[r.Intn(len(creatorIDs))]
		liveStreamID := liveStreamIDs[r.Intn(len(liveStreamIDs))]
		auctions = append(auctions, model.Auction{
			ProductID:    productIDs[r.Intn(len(productIDs))],
			LiveStreamID: &liveStreamID,
			CreatorID:    &creatorID,
			Status:       model.AuctionStatusPending,
			CurrentPrice: 0,
			StartTime:    startTime,
			EndTime:      endTime,
			DelayUsed:    0,
		})
	}

	for i := 0; i < ongoingCount; i++ {
		startTime := now.Add(-time.Minute * time.Duration(5+r.Intn(30)))
		endTime := now.Add(time.Minute * time.Duration(5+r.Intn(10)))
		currentPrice := float64(50 + r.Intn(500))
		winnerID := creatorIDs[r.Intn(len(creatorIDs))]
		creatorID := creatorIDs[r.Intn(len(creatorIDs))]
		liveStreamID := liveStreamIDs[r.Intn(len(liveStreamIDs))]
		auctions = append(auctions, model.Auction{
			ProductID:    productIDs[r.Intn(len(productIDs))],
			LiveStreamID: &liveStreamID,
			CreatorID:    &creatorID,
			Status:       model.AuctionStatusOngoing,
			CurrentPrice: currentPrice,
			WinnerID:     &winnerID,
			StartTime:    startTime,
			EndTime:      endTime,
			DelayUsed:    r.Intn(2),
		})
	}

	for i := 0; i < delayedCount; i++ {
		startTime := now.Add(-time.Minute * 30)
		endTime := now.Add(time.Minute * time.Duration(r.Intn(5)))
		currentPrice := float64(100 + r.Intn(400))
		winnerID := creatorIDs[r.Intn(len(creatorIDs))]
		creatorID := creatorIDs[r.Intn(len(creatorIDs))]
		liveStreamID := liveStreamIDs[r.Intn(len(liveStreamIDs))]
		auctions = append(auctions, model.Auction{
			ProductID:    productIDs[r.Intn(len(productIDs))],
			LiveStreamID: &liveStreamID,
			CreatorID:    &creatorID,
			Status:       model.AuctionStatusDelayed,
			CurrentPrice: currentPrice,
			WinnerID:     &winnerID,
			StartTime:    startTime,
			EndTime:      endTime,
			DelayUsed:    2,
		})
	}

	for i := 0; i < endedCount; i++ {
		startTime := now.Add(-time.Hour * time.Duration(1+r.Intn(24)))
		endTime := startTime.Add(time.Minute * 15)
		currentPrice := float64(100 + r.Intn(500))
		winnerID := creatorIDs[r.Intn(len(creatorIDs))]
		creatorID := creatorIDs[r.Intn(len(creatorIDs))]
		liveStreamID := liveStreamIDs[r.Intn(len(liveStreamIDs))]
		auctions = append(auctions, model.Auction{
			ProductID:    productIDs[r.Intn(len(productIDs))],
			LiveStreamID: &liveStreamID,
			CreatorID:    &creatorID,
			Status:       model.AuctionStatusEnded,
			CurrentPrice: currentPrice,
			WinnerID:     &winnerID,
			StartTime:    startTime,
			EndTime:      endTime,
			DelayUsed:    r.Intn(3),
		})
	}

	for i := 0; i < cancelledCount; i++ {
		startTime := now.Add(-time.Hour * 24)
		endTime := startTime.Add(time.Minute * 15)
		creatorID := creatorIDs[r.Intn(len(creatorIDs))]
		liveStreamID := liveStreamIDs[r.Intn(len(liveStreamIDs))]
		auctions = append(auctions, model.Auction{
			ProductID:    productIDs[r.Intn(len(productIDs))],
			LiveStreamID: &liveStreamID,
			CreatorID:    &creatorID,
			Status:       model.AuctionStatusCancelled,
			CurrentPrice: 0,
			StartTime:    startTime,
			EndTime:      endTime,
			DelayUsed:    0,
		})
	}

	return auctions
}

// GenerateBids 生成出价记录
func GenerateBids(cfg *SeedConfig, auctions []model.Auction, userIDs []int64) []model.Bid {
	bids := make([]model.Bid, 0)
	now := time.Now()
	r := rand.New(rand.NewSource(now.UnixNano() + 1))

	for _, auction := range auctions {
		if auction.Status != model.AuctionStatusEnded && auction.Status != model.AuctionStatusOngoing && auction.Status != model.AuctionStatusDelayed {
			continue
		}

		bidCount := cfg.BidsPerAuction + r.Intn(5) - 2
		if bidCount < 3 {
			bidCount = 3
		}

		basePrice := auction.CurrentPrice - float64(bidCount*10)
		if basePrice < 50 {
			basePrice = 50
		}

		for j := 0; j < bidCount; j++ {
			amount := basePrice + float64(j*10) + float64(r.Intn(5))
			bids = append(bids, model.Bid{
				AuctionID: auction.ID,
				UserID:    userIDs[r.Intn(len(userIDs))],
				Amount:    amount,
			})
		}
	}

	return bids
}

// GenerateNotifications 生成通知数据
func GenerateNotifications(cfg *SeedConfig, userIDs []int64, auctionIDs []int64) []model.Notification {
	notifications := make([]model.Notification, 0, cfg.NotificationsCount)
	now := time.Now()
	r := rand.New(rand.NewSource(now.UnixNano() + 2))

	types := []model.NotificationType{
		model.NotificationTypeAuctionWon,
		model.NotificationTypeAuctionLost,
		model.NotificationTypeBidOutbid,
		model.NotificationTypeAuctionStarting,
		model.NotificationTypeLiveStreamStartingSoon,
		model.NotificationTypeLiveStreamNowLive,
	}

	for i := 0; i < cfg.NotificationsCount; i++ {
		userID := userIDs[r.Intn(len(userIDs))]
		notifType := types[r.Intn(len(types))]
		auctionID := auctionIDs[r.Intn(len(auctionIDs))]

		var title, content string
		switch notifType {
		case model.NotificationTypeAuctionWon:
			title = "竞拍中标"
			content = fmt.Sprintf("恭喜！您在竞拍 #%d 中中标", auctionID)
		case model.NotificationTypeAuctionLost:
			title = "竞拍未中标"
			content = fmt.Sprintf("很遗憾，您在竞拍 #%d 中未能中标", auctionID)
		case model.NotificationTypeBidOutbid:
			title = "出价被超越"
			content = fmt.Sprintf("您的出价已被超越，当前竞拍 #%d", auctionID)
		case model.NotificationTypeAuctionStarting:
			title = "竞拍即将开始"
			content = fmt.Sprintf("竞拍 #%d 将在5分钟后开始，请做好准备", auctionID)
		case model.NotificationTypeLiveStreamStartingSoon:
			title = "即将开播"
			content = "您关注的直播间将在10分钟后开播"
		case model.NotificationTypeLiveStreamNowLive:
			title = "正在直播"
			content = "您关注的直播间已开始直播，快来参与竞拍吧！"
		}

		notification := model.Notification{
			UserID:  userID,
			Type:    notifType,
			Title:   title,
			Content: content,
			Data:    model.JSONMap{"auction_id": auctionID},
		}

		if r.Intn(2) == 0 {
			readAt := now.Add(-time.Hour * time.Duration(r.Intn(24)))
			notification.ReadAt = &readAt
		}

		notifications = append(notifications, notification)
	}

	return notifications
}

// GenerateSkyLampSubscriptions 生成点天灯订阅数据
func GenerateSkyLampSubscriptions(cfg *SeedConfig, auctionIDs []int64, userIDs []int64) []model.SkyLampSubscription {
	subscriptions := make([]model.SkyLampSubscription, 0, cfg.SkyLampCount)
	now := time.Now()
	r := rand.New(rand.NewSource(now.UnixNano() + 3))

	statuses := []model.SkyLampStatus{
		model.SkyLampStatusActive,
		model.SkyLampStatusStopped,
		model.SkyLampStatusCancelled,
		model.SkyLampStatusEnded,
	}

	for i := 0; i < cfg.SkyLampCount; i++ {
		auctionID := auctionIDs[r.Intn(len(auctionIDs))]
		userID := userIDs[r.Intn(len(userIDs))]
		status := statuses[r.Intn(len(statuses))]

		initialPrice := float64(50 + r.Intn(100))
		initialBidAmount := initialPrice + 10
		maxPriceLimit := initialPrice * 3

		subscription := model.SkyLampSubscription{
			AuctionID:           auctionID,
			UserID:              userID,
			Status:              status,
			InitialPrice:        initialPrice,
			InitialBidAmount:    initialBidAmount,
			MaxPriceLimit:       maxPriceLimit,
			CurrentAutoBidCount: r.Intn(5),
			TotalBidAmount:      initialBidAmount + float64(r.Intn(50)),
		}

		if status != model.SkyLampStatusActive {
			stoppedAt := now.Add(-time.Hour * time.Duration(r.Intn(24)))
			subscription.StoppedAt = &stoppedAt
		}

		subscriptions = append(subscriptions, subscription)
	}

	return subscriptions
}

// GenerateUserLiveStreamFollows 生成用户关注直播间数据
func GenerateUserLiveStreamFollows(cfg *SeedConfig, userIDs []int64, liveStreamIDs []int64) []model.UserLiveStreamFollow {
	follows := make([]model.UserLiveStreamFollow, 0, cfg.FollowCount)
	r := rand.New(rand.NewSource(time.Now().UnixNano() + 4))

	for i := 0; i < cfg.FollowCount; i++ {
		follows = append(follows, model.UserLiveStreamFollow{
			UserID:              userIDs[r.Intn(len(userIDs))],
			LiveStreamID:        liveStreamIDs[r.Intn(len(liveStreamIDs))],
			NotificationEnabled: r.Intn(2) == 0,
		})
	}

	return follows
}

// GenerateUserProductReminders 生成商品提醒订阅数据
func GenerateUserProductReminders(cfg *SeedConfig, userIDs []int64, productIDs []int64, auctionIDs []int64) []model.UserProductReminder {
	reminders := make([]model.UserProductReminder, 0, cfg.ReminderCount)
	r := rand.New(rand.NewSource(time.Now().UnixNano() + 5))

	for i := 0; i < cfg.ReminderCount; i++ {
		reminders = append(reminders, model.UserProductReminder{
			UserID:              userIDs[r.Intn(len(userIDs))],
			ProductID:           productIDs[r.Intn(len(productIDs))],
			AuctionID:           auctionIDs[r.Intn(len(auctionIDs))],
			NotificationEnabled: r.Intn(2) == 0,
		})
	}

	return reminders
}

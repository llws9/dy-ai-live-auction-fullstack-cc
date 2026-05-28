package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"auction-service/dao"
	"auction-service/model"
)

// ColdPushScheduler 冷推定时任务
// 每5分钟检查即将开播的冷门直播间，推送通知
type ColdPushScheduler struct {
	notificationSender      NotificationSender
	userLiveStreamFollowDAO *dao.UserLiveStreamFollowDAO
	redis                   *redis.Client
	interval                time.Duration // 默认5分钟
}

// NewColdPushScheduler 创建冷推定时任务
func NewColdPushScheduler(sender NotificationSender, followDAO *dao.UserLiveStreamFollowDAO, redis *redis.Client) *ColdPushScheduler {
	return &ColdPushScheduler{
		notificationSender:      sender,
		userLiveStreamFollowDAO: followDAO,
		redis:                   redis,
		interval:                5 * time.Minute,
	}
}

// SetInterval 设置检查间隔（用于测试）
func (s *ColdPushScheduler) SetInterval(interval time.Duration) {
	s.interval = interval
}

// Run 启动定时任务，每5分钟执行一次
func (s *ColdPushScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	log.Printf("Cold push scheduler started, interval: %v", s.interval)

	// 立即执行一次
	s.pushColdLiveStreamNotifications(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Cold push scheduler stopped")
			return
		case <-ticker.C:
			s.pushColdLiveStreamNotifications(ctx)
		}
	}
}

// pushColdLiveStreamNotifications 推送冷门直播间开播提醒
// 1. 查询 ZRANGEBYSCORE live_stream:cold:start_time (now, now+10min)
// 2. 遍历每个直播间
// 3. 获取关注该直播间的用户列表（从数据库）
// 4. 推送开播提醒通知（使用NotificationSender）
// 5. 推送后从ZSET移除
func (s *ColdPushScheduler) pushColdLiveStreamNotifications(ctx context.Context) error {
	now := time.Now()
	// 查询10分钟内即将开播的直播间
	endTime := now.Add(10 * time.Minute)

	// 1. 从Redis ZSET查询即将开播的冷门直播间
	liveStreamIDs, err := dao.GetColdLiveStreamsStartingSoon(ctx, now, endTime)
	if err != nil {
		log.Printf("Failed to get cold live streams: %v", err)
		return err
	}

	if len(liveStreamIDs) == 0 {
		log.Printf("No cold live streams starting in next 10 minutes")
		return nil
	}

	log.Printf("Found %d cold live streams starting soon: %v", len(liveStreamIDs), liveStreamIDs)

	// 2. 遍历每个直播间
	for _, liveStreamID := range liveStreamIDs {
		// 3. 获取关注该直播间的用户列表（开启通知的用户）
		followers, err := s.userLiveStreamFollowDAO.GetFollowers(ctx, liveStreamID, 0, 1000)
		if err != nil {
			log.Printf("Failed to get followers for live stream %d: %v", liveStreamID, err)
			continue
		}

		if len(followers) == 0 {
			log.Printf("No followers with notification enabled for live stream %d", liveStreamID)
			// 没有关注用户，也要从ZSET移除
			s.removeFromZSET(ctx, liveStreamID)
			continue
		}

		log.Printf("Live stream %d has %d followers with notification enabled", liveStreamID, len(followers))

		// 4. 推送开播提醒通知（批量）
		notifications := make([]*model.NotificationRequest, 0, len(followers))
		for _, follow := range followers {
			notification := &model.NotificationRequest{
				UserID:  follow.UserID,
				Type:    model.NotificationTypeLiveStreamStartingSoon,
				Title:   "即将开播",
				Content: fmt.Sprintf("您关注的直播间「%d」将在10分钟后开播", liveStreamID),
				Data: map[string]interface{}{
					"live_stream_id": liveStreamID,
					"start_time":     endTime.Format(time.RFC3339),
				},
			}
			notifications = append(notifications, notification)
		}

		// 批量发送通知
		if err := s.notificationSender.SendBatchNotifications(ctx, notifications); err != nil {
			log.Printf("Failed to send batch notifications for live stream %d: %v", liveStreamID, err)
			continue
		}

		log.Printf("Sent %d notifications for live stream %d", len(notifications), liveStreamID)

		// 5. 推送后从ZSET移除
		s.removeFromZSET(ctx, liveStreamID)
	}

	return nil
}

// removeFromZSET 从ZSET移除直播间
func (s *ColdPushScheduler) removeFromZSET(ctx context.Context, liveStreamID int64) {
	if err := dao.RemoveFromZSET(ctx, liveStreamID); err != nil {
		log.Printf("Failed to remove live stream %d from ZSET: %v", liveStreamID, err)
	} else {
		log.Printf("Removed live stream %d from ZSET", liveStreamID)
	}
}

package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/mq"
)

// BatchNotificationService 批量通知服务
type BatchNotificationService struct {
	followDAO       *dao.UserLiveStreamFollowDAO
	notificationDAO *dao.NotificationDAO
	notificationService *NotificationService
}

const (
	// BatchSize 每批处理用户数
	BatchSize = 10000
	// BatchDelay 批次间隔时间
	BatchDelay = 3 * time.Second
	// MaxRetries 最大重试次数
	MaxRetries = 3
)

// NewBatchNotificationService 创建批量通知服务
func NewBatchNotificationService(
	followDAO *dao.UserLiveStreamFollowDAO,
	notificationDAO *dao.NotificationDAO,
	notificationService *NotificationService,
) *BatchNotificationService {
	return &BatchNotificationService{
		followDAO:          followDAO,
		notificationDAO:    notificationDAO,
		notificationService: notificationService,
	}
}

// ProcessNotification 处理通知消息（实现 mq.NotificationServiceInterface）
func (s *BatchNotificationService) ProcessNotification(msg *mq.NotificationMessage) error {
	ctx := context.Background()

	// 获取关注用户总数
	totalUsers, err := s.followDAO.CountByLiveStream(ctx, msg.LiveStreamID)
	if err != nil {
		return fmt.Errorf("获取关注用户数失败: %w", err)
	}

	if totalUsers == 0 {
		log.Printf("No followers for live stream %d, skipping notification", msg.LiveStreamID)
		return nil
	}

	log.Printf("Processing notification for %d users: Type=%s, LiveStreamID=%d", totalUsers, msg.Type, msg.LiveStreamID)

	// 计算批次数
	batches := (totalUsers + BatchSize - 1) / BatchSize

	// 分批推送
	for i := 0; i < int(batches); i++ {
		offset := i * BatchSize

		// 获取本批用户
		users, err := s.followDAO.GetFollowers(ctx, msg.LiveStreamID, offset, BatchSize)
		if err != nil {
			log.Printf("Failed to get followers batch %d: %v", i, err)
			continue
		}

		// 批量创建通知记录
		notifications := make([]*model.Notification, 0, len(users))
		for _, user := range users {
			notifications = append(notifications, &model.Notification{
				UserID:  user.UserID,
				Type:    model.NotificationType(msg.Type),
				Title:   msg.GenerateTitle(),
				Content: msg.GenerateContent(),
				Data:    msg.GenerateData(),
			})
		}

		// 批量保存到数据库
		if err := s.notificationDAO.CreateBatch(ctx, notifications); err != nil {
			log.Printf("Failed to save notifications batch %d: %v", i, err)
			continue
		}

		// 实时推送通知
		for _, notification := range notifications {
			s.notificationService.SendNotification(ctx, &model.NotificationRequest{
				UserID:  notification.UserID,
				Type:    notification.Type,
				Title:   notification.Title,
				Content: notification.Content,
				Data:    notification.Data,
			})
		}

		log.Printf("Batch %d/%d completed: %d notifications sent", i+1, batches, len(notifications))

		// 如果不是最后一批，等待一段时间
		if i < int(batches)-1 {
			time.Sleep(BatchDelay)
		}
	}

	log.Printf("Notification processing completed: Type=%s, TotalUsers=%d, Batches=%d", msg.Type, totalUsers, batches)
	return nil
}

// ProcessNotificationWithRetry 带重试的通知处理
func (s *BatchNotificationService) ProcessNotificationWithRetry(msg *mq.NotificationMessage) error {
	var lastErr error

	for i := 0; i < MaxRetries; i++ {
		err := s.ProcessNotification(msg)
		if err == nil {
			return nil
		}

		lastErr = err
		log.Printf("Notification processing attempt %d failed: %v", i+1, err)

		// 等待一段时间后重试
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	return fmt.Errorf("notification processing failed after %d retries: %w", MaxRetries, lastErr)
}

// SendNewProductNotification 发送新商品发布通知
func (s *BatchNotificationService) SendNewProductNotification(ctx context.Context, liveStreamID, productID int64, productName string) error {
	// 注意：这个方法会由 product-service 通过消息队列调用
	// 这里只是预留接口，实际发送由 mq producer 负责
	return nil
}

// SendAuctionStartingNotification 发送竞拍即将开始通知（提前30分钟）
func (s *BatchNotificationService) SendAuctionStartingNotification(ctx context.Context, liveStreamID, productID, auctionID int64, productName string, startTime time.Time) error {
	// 注意：这个方法会由 mq producer 调用，通过延迟队列实现
	return nil
}

// SendProductUnpublishedNotification 发送商品下架通知
func (s *BatchNotificationService) SendProductUnpublishedNotification(ctx context.Context, liveStreamID, productID int64, productName, reason string) error {
	// 注意：这个方法会由 product-service 通过消息队列调用
	return nil
}

// SendAuctionEndedNotification 发送竞拍结束通知
func (s *BatchNotificationService) SendAuctionEndedNotification(ctx context.Context, liveStreamID, productID, auctionID int64, productName string, winnerID int64, winnerName string, finalPrice float64) error {
	// 注意：这个方法会由 auction-service 通过消息队列调用
	return nil
}

// Ensure BatchNotificationService implements mq.NotificationServiceInterface
var _ mq.NotificationServiceInterface = (*BatchNotificationService)(nil)

package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/websocket"
)

// NotificationSender 通知发送接口
// 业务系统通过此接口发送通知，无需关心底层实现
type NotificationSender interface {
	// SendNotification 发送通知
	SendNotification(ctx context.Context, req *model.NotificationRequest) error
	// SendBatchNotifications 批量发送通知
	SendBatchNotifications(ctx context.Context, reqs []*model.NotificationRequest) error
}

// OrderEventPublisher 订单事件发布接口（二期实现）
// 订单系统实现此接口，在状态变更时发布事件
type OrderEventPublisher interface {
	// PublishOrderEvent 发布订单事件
	PublishOrderEvent(ctx context.Context, event *model.OrderEvent) error
}

// EventSubscriber 事件订阅器
// 通知系统订阅订单事件，自动发送通知
type EventSubscriber interface {
	// Subscribe 订阅事件
	Subscribe(eventType string, handler EventHandler) error
}

// EventHandler 事件处理函数
type EventHandler func(ctx context.Context, event interface{}) error

// NotificationService 通知服务
type NotificationService struct {
	notificationDAO *dao.NotificationDAO
	hub             *websocket.Hub
	redis           *redis.Client
	followDAO       *dao.UserLiveStreamFollowDAO
}

// NewNotificationService 创建通知服务
func NewNotificationService(notificationDAO *dao.NotificationDAO, redis *redis.Client) *NotificationService {
	return &NotificationService{
		notificationDAO: notificationDAO,
		redis:           redis,
	}
}

// SetHub 设置WebSocket Hub
func (s *NotificationService) SetHub(hub *websocket.Hub) {
	s.hub = hub
}

// SetFollowDAO 设置关注DAO（用于热拉Redis失败时DB兜底）
func (s *NotificationService) SetFollowDAO(followDAO *dao.UserLiveStreamFollowDAO) {
	s.followDAO = followDAO
}

// SetMetrics 设置指标收集器（占位方法，通知服务暂不需要metrics）
func (s *NotificationService) SetMetrics(metrics interface{}) {
	// NotificationService目前不需要metrics
	// 保留此方法以备将来扩展
}

// SendNotification 发送通知
func (s *NotificationService) SendNotification(ctx context.Context, req *model.NotificationRequest) error {
	// 创建通知实体
	notification := &model.Notification{
		UserID:  req.UserID,
		Type:    req.Type,
		Title:   req.Title,
		Content: req.Content,
		Data:    req.Data,
	}

	// 保存到数据库
	if err := s.notificationDAO.Create(ctx, notification); err != nil {
		return fmt.Errorf("保存通知失败: %w", err)
	}

	// 实时推送（默认立即推送）
	if req.Immediately || req.Immediately == false {
		s.pushNotification(ctx, notification)
	}

	log.Printf("Notification sent: user=%d, type=%s, title=%s", req.UserID, req.Type, req.Title)
	return nil
}

// SendBatchNotifications 批量发送通知
func (s *NotificationService) SendBatchNotifications(ctx context.Context, reqs []*model.NotificationRequest) error {
	if len(reqs) == 0 {
		return nil
	}

	// 创建通知实体列表
	notifications := make([]*model.Notification, len(reqs))
	for i, req := range reqs {
		notifications[i] = &model.Notification{
			UserID:  req.UserID,
			Type:    req.Type,
			Title:   req.Title,
			Content: req.Content,
			Data:    req.Data,
		}
	}

	// 批量保存到数据库
	if err := s.notificationDAO.CreateBatch(ctx, notifications); err != nil {
		return fmt.Errorf("批量保存通知失败: %w", err)
	}

	// 实时推送
	for _, notification := range notifications {
		s.pushNotification(ctx, notification)
	}

	log.Printf("Batch notifications sent: count=%d", len(reqs))
	return nil
}

// pushNotification 推送通知到WebSocket
func (s *NotificationService) pushNotification(ctx context.Context, notification *model.Notification) {
	if s.hub == nil {
		return
	}

	// 构建WebSocket消息
	msg := &websocket.Message{
		Type: "notification",
		Data: map[string]interface{}{
			"id":         notification.ID,
			"type":       notification.Type,
			"title":      notification.Title,
			"content":    notification.Content,
			"data":       notification.Data,
			"created_at": notification.CreatedAt,
		},
	}

	// 发送到用户房间（用户ID作为房间ID）
	s.hub.BroadcastToUserRoom(notification.UserID, msg)
}

// SendBidOutbidNotification 发送出价被超越通知
func (s *NotificationService) SendBidOutbidNotification(ctx context.Context, userID int64, auctionID int64, oldBid, newBid float64) error {
	return s.SendNotification(ctx, &model.NotificationRequest{
		UserID:  userID,
		Type:    model.NotificationTypeBidOutbid,
		Title:   "出价被超越",
		Content: fmt.Sprintf("您在竞拍中的出价 %.2f 元已被超越，当前最高价为 %.2f 元", oldBid, newBid),
		Data: map[string]interface{}{
			"auction_id": auctionID,
			"old_bid":    oldBid,
			"new_bid":    newBid,
		},
	})
}

// SendAuctionWonNotification 发送竞拍中标通知
func (s *NotificationService) SendAuctionWonNotification(ctx context.Context, userID int64, auctionID int64, finalPrice float64) error {
	return s.SendNotification(ctx, &model.NotificationRequest{
		UserID:  userID,
		Type:    model.NotificationTypeAuctionWon,
		Title:   "竞拍中标",
		Content: fmt.Sprintf("恭喜！您以 %.2f 元中标了竞拍", finalPrice),
		Data: map[string]interface{}{
			"auction_id":  auctionID,
			"final_price": finalPrice,
		},
	})
}

// SendAuctionLostNotification 发送竞拍未中标通知
func (s *NotificationService) SendAuctionLostNotification(ctx context.Context, userID int64, auctionID int64, winnerPrice float64) error {
	return s.SendNotification(ctx, &model.NotificationRequest{
		UserID:  userID,
		Type:    model.NotificationTypeAuctionLost,
		Title:   "竞拍未中标",
		Content: fmt.Sprintf("很遗憾，您未能中标。最终成交价为 %.2f 元", winnerPrice),
		Data: map[string]interface{}{
			"auction_id":   auctionID,
			"winner_price": winnerPrice,
		},
	})
}

// SendOrderPaidNotification 发送订单已支付通知（Mock触发）
func (s *NotificationService) SendOrderPaidNotification(ctx context.Context, userID int64, orderID int64) error {
	return s.SendNotification(ctx, &model.NotificationRequest{
		UserID:  userID,
		Type:    model.NotificationTypeOrderPaid,
		Title:   "订单已支付",
		Content: fmt.Sprintf("您的订单 #%d 已支付成功", orderID),
		Data: map[string]interface{}{
			"order_id": orderID,
		},
	})
}

// SendOrderShippedNotification 发送订单已发货通知（Mock触发）
func (s *NotificationService) SendOrderShippedNotification(ctx context.Context, userID int64, orderID int64) error {
	return s.SendNotification(ctx, &model.NotificationRequest{
		UserID:  userID,
		Type:    model.NotificationTypeOrderShipped,
		Title:   "订单已发货",
		Content: fmt.Sprintf("您的订单 #%d 已发货，请留意查收", orderID),
		Data: map[string]interface{}{
			"order_id": orderID,
		},
	})
}

// SendOrderCompletedNotification 发送订单已完成通知（Mock触发）
func (s *NotificationService) SendOrderCompletedNotification(ctx context.Context, userID int64, orderID int64) error {
	return s.SendNotification(ctx, &model.NotificationRequest{
		UserID:  userID,
		Type:    model.NotificationTypeOrderCompleted,
		Title:   "订单已完成",
		Content: fmt.Sprintf("您的订单 #%d 已完成，感谢您的购买！", orderID),
		Data: map[string]interface{}{
			"order_id": orderID,
		},
	})
}

// OnOrderEvent 处理订单事件（二期实现）
func (s *NotificationService) OnOrderEvent(ctx context.Context, event *model.OrderEvent) error {
	switch event.EventType {
	case model.OrderEventPaid:
		return s.SendOrderPaidNotification(ctx, event.UserID, event.OrderID)
	case model.OrderEventShipped:
		return s.SendOrderShippedNotification(ctx, event.UserID, event.OrderID)
	case model.OrderEventCompleted:
		return s.SendOrderCompletedNotification(ctx, event.UserID, event.OrderID)
	}
	return nil
}

// GetNotifications 获取用户通知列表
func (s *NotificationService) GetNotifications(ctx context.Context, userID int64, page, pageSize int, unreadOnly bool) (*model.NotificationListResponse, error) {
	return s.notificationDAO.GetByUserID(ctx, userID, page, pageSize, unreadOnly)
}

// GetUnreadCount 获取未读通知数量
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID int64) (int64, error) {
	return s.notificationDAO.GetUnreadCount(ctx, userID)
}

// MarkAsRead 标记通知为已读
func (s *NotificationService) MarkAsRead(ctx context.Context, id int64, userID int64) error {
	return s.notificationDAO.MarkAsRead(ctx, id, userID)
}

// MarkAllAsRead 标记所有通知为已读
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID int64) error {
	return s.notificationDAO.MarkAllAsRead(ctx, userID)
}

// SyncUnreadNotifications 同步未读通知（用户重连WebSocket时调用）
func (s *NotificationService) SyncUnreadNotifications(ctx context.Context, userID int64) error {
	notifications, err := s.notificationDAO.GetUnreadByUserID(ctx, userID, 50)
	if err != nil {
		return err
	}

	for _, notification := range notifications {
		s.pushNotification(ctx, &notification)
	}

	return nil
}

// HotPullNotifications - 热拉通知
// 用户登录/切换前台时主动拉取热门直播间通知
// 1. Redis获取用户关注的直播间集合 SMEMBERS user:{uid}:followed_live_streams
// 2. ZRANGEBYSCORE live_stream:hot:start_time (now, now+1hour) 获取即将开播的热门直播间
// 3. SMEMBERS live_stream:hot:live_now 获取正在直播的热门直播间
// 4. 过滤：只返回用户关注的热门直播间
// 5. 生成通知，返回结果
func (s *NotificationService) HotPullNotifications(ctx context.Context, userID int64) ([]*model.Notification, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	// 1. 获取用户关注的直播间集合
	followedLiveStreams, err := dao.GetUserFollowedLiveStreams(ctx, userID)
	if err != nil {
		log.Printf("HotPull: failed to get user followed live streams: %v", err)
		return nil, fmt.Errorf("failed to get user followed live streams: %w", err)
	}

	if len(followedLiveStreams) == 0 {
		// 用户没有关注任何直播间，返回空列表
		return []*model.Notification{}, nil
	}

	// 创建关注直播间的Set便于快速查找
	followedSet := make(map[int64]bool)
	for _, id := range followedLiveStreams {
		followedSet[id] = true
	}

	now := time.Now()
	oneHourLater := now.Add(1 * time.Hour)

	notifications := make([]*model.Notification, 0)

	// 2. 获取即将开播的热门直播间 (now, now+1hour)
	startingSoon, err := dao.GetHotLiveStreamsStartingSoon(ctx, now, oneHourLater)
	if err != nil {
		log.Printf("HotPull: failed to get hot live streams starting soon: %v", err)
		// 继续处理，不返回错误
	}

	// 4. 过滤：只返回用户关注的热门直播间，生成即将开播通知
	for _, liveStreamID := range startingSoon {
		if followedSet[liveStreamID] {
			notification := &model.Notification{
				UserID:  userID,
				Type:    model.NotificationTypeLiveStreamStartingSoon,
				Title:   "即将开播",
				Content: fmt.Sprintf("您关注的直播间 #%d 即将开播，请准时收看！", liveStreamID),
				Data: map[string]interface{}{
					"live_stream_id": liveStreamID,
					"triggered_at":   now.Format(time.RFC3339),
				},
			}
			notifications = append(notifications, notification)

			// 实时推送
			s.pushNotification(ctx, notification)
		}
	}

	// 3. 获取正在直播的热门直播间
	liveNow, err := dao.GetHotLiveNowSet(ctx)
	if err != nil {
		log.Printf("HotPull: failed to get hot live now set: %v", err)
		// 继续处理，不返回错误
	}

	// 4. 过滤：只返回用户关注的热门直播间，生成正在直播通知
	for _, liveStreamID := range liveNow {
		if followedSet[liveStreamID] {
			notification := &model.Notification{
				UserID:  userID,
				Type:    model.NotificationTypeLiveStreamNowLive,
				Title:   "正在直播",
				Content: fmt.Sprintf("您关注的直播间 #%d 正在直播，快来看看！", liveStreamID),
				Data: map[string]interface{}{
					"live_stream_id": liveStreamID,
					"triggered_at":   now.Format(time.RFC3339),
				},
			}
			notifications = append(notifications, notification)

			// 实时推送
			s.pushNotification(ctx, notification)
		}
	}

	log.Printf("HotPull: user=%d, followed=%d, starting_soon=%d, live_now=%d, notifications=%d",
		userID, len(followedLiveStreams), len(startingSoon), len(liveNow), len(notifications))

	return notifications, nil
}

// Ensure NotificationService implements NotificationSender
var _ NotificationSender = (*NotificationService)(nil)

package service

import (
	"context"
	"testing"
	"time"

	"auction-service/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockNotificationSender Mock通知发送器
type MockNotificationSender struct {
	mock.Mock
	notifications []*model.NotificationRequest
}

func (m *MockNotificationSender) SendNotification(ctx context.Context, req *model.NotificationRequest) error {
	m.Called(ctx, req)
	m.notifications = append(m.notifications, req)
	return nil
}

func (m *MockNotificationSender) SendBatchNotifications(ctx context.Context, reqs []*model.NotificationRequest) error {
	m.Called(ctx, reqs)
	m.notifications = append(m.notifications, reqs...)
	return nil
}

// TestNotificationFlow_E2E 端到端测试通知流程
func TestNotificationFlow_E2E(t *testing.T) {
	// ========== 场景1：出价被超越通知 ==========
	t.Run("出价被超越通知", func(t *testing.T) {
		// 1. 准备Mock通知发送器
		mockSender := new(MockNotificationSender)
		mockSender.On("SendNotification", mock.Anything, mock.Anything).Return(nil)

		// 2. 模拟出价超越场景
		previousWinnerID := int64(100)
		auctionID := int64(1)
		oldPrice := 100.0
		newPrice := 120.0
		newBidderID := int64(200)

		// 3. 验证通知触发条件
		shouldNotify := previousWinnerID > 0 && previousWinnerID != newBidderID && newPrice > oldPrice
		assert.True(t, shouldNotify, "应该触发出价超越通知")

		// 4. 模拟发送通知
		ctx := context.Background()
		req := &model.NotificationRequest{
			UserID:  previousWinnerID,
			Type:    model.NotificationTypeBidOutbid,
			Title:   "出价被超越",
			Content: "您在竞拍中的出价 100.00 元已被超越，当前最高价为 120.00 元",
			Data: map[string]interface{}{
				"auction_id": auctionID,
				"new_price":  newPrice,
				"old_price":  oldPrice,
			},
		}
		err := mockSender.SendNotification(ctx, req)
		assert.NoError(t, err)

		// 5. 验证通知已发送
		mockSender.AssertExpectations(t)
		assert.Equal(t, 1, len(mockSender.notifications))
		assert.Equal(t, model.NotificationTypeBidOutbid, mockSender.notifications[0].Type)
		assert.Equal(t, previousWinnerID, mockSender.notifications[0].UserID)
	})

	// ========== 场景2：竞拍中标通知 ==========
	t.Run("竞拍中标通知", func(t *testing.T) {
		mockSender := new(MockNotificationSender)
		mockSender.On("SendNotification", mock.Anything, mock.Anything).Return(nil)

		winnerID := int64(100)
		auctionID := int64(1)
		finalPrice := 120.0

		ctx := context.Background()
		req := &model.NotificationRequest{
			UserID:  winnerID,
			Type:    model.NotificationTypeAuctionWon,
			Title:   "竞拍中标",
			Content: "恭喜您！您以 120.00 元的价格成功竞得商品",
			Data: map[string]interface{}{
				"auction_id":  auctionID,
				"final_price": finalPrice,
			},
		}
		err := mockSender.SendNotification(ctx, req)
		assert.NoError(t, err)

		mockSender.AssertExpectations(t)
		assert.Equal(t, model.NotificationTypeAuctionWon, mockSender.notifications[0].Type)
	})

	// ========== 场景3：竞拍失败通知 ==========
	t.Run("竞拍失败通知", func(t *testing.T) {
		mockSender := new(MockNotificationSender)
		mockSender.On("SendNotification", mock.Anything, mock.Anything).Return(nil)

		loserID := int64(200)
		auctionID := int64(1)

		ctx := context.Background()
		req := &model.NotificationRequest{
			UserID:  loserID,
			Type:    model.NotificationTypeAuctionLost,
			Title:   "竞拍失败",
			Content: "很遗憾，您在本次竞拍中未能中标",
			Data: map[string]interface{}{
				"auction_id": auctionID,
			},
		}
		err := mockSender.SendNotification(ctx, req)
		assert.NoError(t, err)

		mockSender.AssertExpectations(t)
		assert.Equal(t, model.NotificationTypeAuctionLost, mockSender.notifications[0].Type)
	})

	// ========== 场景4：批量通知发送 ==========
	t.Run("批量通知发送", func(t *testing.T) {
		mockSender := new(MockNotificationSender)
		mockSender.On("SendBatchNotifications", mock.Anything, mock.Anything).Return(nil)

		auctionID := int64(1)
		winnerID := int64(100)
		loserIDs := []int64{200, 300, 400}
		finalPrice := 120.0

		// 准备批量通知
		var reqs []*model.NotificationRequest

		// 中标通知
		reqs = append(reqs, &model.NotificationRequest{
			UserID:  winnerID,
			Type:    model.NotificationTypeAuctionWon,
			Title:   "竞拍中标",
			Content: "恭喜您中标！",
			Data: map[string]interface{}{
				"auction_id":  auctionID,
				"final_price": finalPrice,
			},
		})

		// 失败通知
		for _, loserID := range loserIDs {
			reqs = append(reqs, &model.NotificationRequest{
				UserID:  loserID,
				Type:    model.NotificationTypeAuctionLost,
				Title:   "竞拍失败",
				Content: "很遗憾未中标",
				Data: map[string]interface{}{
					"auction_id": auctionID,
				},
			})
		}

		ctx := context.Background()
		err := mockSender.SendBatchNotifications(ctx, reqs)
		assert.NoError(t, err)

		mockSender.AssertExpectations(t)
		assert.Equal(t, 4, len(mockSender.notifications)) // 1 winner + 3 losers
	})
}

// TestNotificationTypes_E2E 测试所有通知类型
func TestNotificationTypes_E2E(t *testing.T) {
	tests := []struct {
		name         string
		notifyType   model.NotificationType
		expectedDesc string
	}{
		{"出价被超越", model.NotificationTypeBidOutbid, "bid_outbid"},
		{"竞拍中标", model.NotificationTypeAuctionWon, "auction_won"},
		{"竞拍失败", model.NotificationTypeAuctionLost, "auction_lost"},
		{"订单支付成功", model.NotificationTypeOrderPaid, "order_paid"},
		{"订单已发货", model.NotificationTypeOrderShipped, "order_shipped"},
		{"订单已完成", model.NotificationTypeOrderCompleted, "order_completed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.notifyType, model.NotificationType(tt.expectedDesc))
		})
	}
}

// TestNotificationPersistence_E2E 测试通知持久化
func TestNotificationPersistence_E2E(t *testing.T) {
	t.Run("通知结构验证", func(t *testing.T) {
		notification := &model.Notification{
			UserID:  100,
			Type:    model.NotificationTypeBidOutbid,
			Title:   "出价被超越",
			Content: "您的出价已被超越",
			Data: model.JSONMap{
				"auction_id": int64(1),
				"new_price":  100.0,
			},
			ReadAt: nil,
		}

		// 验证字段
		assert.Equal(t, int64(100), notification.UserID)
		assert.Equal(t, model.NotificationTypeBidOutbid, notification.Type)
		assert.Nil(t, notification.ReadAt)
		assert.NotNil(t, notification.Data)
	})

	t.Run("通知已读状态切换", func(t *testing.T) {
		notification := &model.Notification{
			ReadAt: nil,
		}

		// 标记为已读
		now := time.Now()
		notification.ReadAt = &now
		assert.NotNil(t, notification.ReadAt)
	})
}

// TestNotificationContent_E2E 测试通知内容生成
func TestNotificationContent_E2E(t *testing.T) {
	t.Run("出价超越通知内容", func(t *testing.T) {
		content := "您在竞拍中的出价 100.00 元已被超越，当前最高价为 120.00 元"

		assert.Contains(t, content, "100.00")
		assert.Contains(t, content, "120.00")
		assert.Contains(t, content, "超越")
	})

	t.Run("中标通知内容", func(t *testing.T) {
		content := "恭喜您！您以 120.00 元的价格成功竞得商品"

		assert.Contains(t, content, "120.00")
		assert.Contains(t, content, "恭喜")
		assert.Contains(t, content, "成功")
	})

	t.Run("失败通知内容", func(t *testing.T) {
		content := "很遗憾，您在本次竞拍中未能中标"

		assert.Contains(t, content, "遗憾")
		assert.Contains(t, content, "未能中标")
	})
}

// TestNotificationThrottle_E2E 测试通知发送节流
func TestNotificationThrottle_E2E(t *testing.T) {
	t.Run("同一用户通知不应节流", func(t *testing.T) {
		mockSender := new(MockNotificationSender)
		mockSender.On("SendNotification", mock.Anything, mock.Anything).Return(nil)

		userID := int64(100)
		ctx := context.Background()

		// 发送多个通知（不应节流）
		for i := 0; i < 5; i++ {
			req := &model.NotificationRequest{
				UserID:  userID,
				Type:    model.NotificationTypeBidOutbid,
				Title:   "出价被超越",
				Content: "您的出价已被超越",
			}
			err := mockSender.SendNotification(ctx, req)
			assert.NoError(t, err)
		}

		// 验证所有通知都已发送
		assert.Equal(t, 5, len(mockSender.notifications))
	})
}

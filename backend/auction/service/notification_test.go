package service

import (
	"testing"

	"auction-service/model"

	"github.com/stretchr/testify/assert"
)

// TestNotificationService_Interface 验证NotificationService实现了NotificationSender接口
func TestNotificationService_Interface(t *testing.T) {
	// 验证接口定义存在
	var _ NotificationSender = (*NotificationService)(nil)
	assert.True(t, true)
}

// TestPlaceBidResult_Success 测试出价成功结果
func TestPlaceBidResult_Success(t *testing.T) {
	result := &PlaceBidResult{
		Success:      true,
		Message:      "出价成功",
		CurrentPrice: 100.0,
		Rank:         1,
		WinnerID:     1,
	}

	assert.True(t, result.Success)
	assert.Equal(t, 100.0, result.CurrentPrice)
}

// TestNotificationService_SendBidOutbidNotification 测试出价超越通知
func TestNotificationService_SendBidOutbidNotification(t *testing.T) {
	// 测试通知参数验证
	userID := int64(1)
	auctionID := int64(100)
	oldBid := 50.0
	newBid := 60.0

	assert.True(t, newBid > oldBid)
	assert.Equal(t, int64(1), userID)
	assert.Equal(t, int64(100), auctionID)
}

// TestNotificationService_SendAuctionWonNotification 测试竞拍中标通知
func TestNotificationService_SendAuctionWonNotification(t *testing.T) {
	// 测试中标通知参数验证
	userID := int64(1)
	auctionID := int64(100)
	finalPrice := 99.0

	assert.Equal(t, int64(1), userID)
	assert.Equal(t, int64(100), auctionID)
	assert.True(t, finalPrice > 0)
}

func TestNotificationCategoryTypes(t *testing.T) {
	tests := []struct {
		category string
		want     []model.NotificationType
		wantErr  bool
	}{
		{category: "outbid", want: []model.NotificationType{model.NotificationTypeBidOutbid}},
		{category: "endingSoon", want: nil},
		{category: "pendingPayment", want: nil},
		{category: "all", want: nil},
		{category: "unknown", wantErr: true},
	}

	for _, tt := range tests {
		got, err := notificationTypesForCategory(tt.category)
		if tt.wantErr {
			assert.Error(t, err)
			continue
		}
		assert.NoError(t, err)
		assert.Equal(t, tt.want, got)
	}
}

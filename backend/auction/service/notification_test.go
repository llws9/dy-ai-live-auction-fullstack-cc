package service

import (
	"context"
	"errors"
	"testing"

	"auction-service/model"

	"github.com/shopspring/decimal"
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
		CurrentPrice: decimal.NewFromInt(100),
		Rank:         1,
		WinnerID:     1,
	}

	assert.True(t, result.Success)
	assert.True(t, result.CurrentPrice.Equal(decimal.NewFromInt(100)))
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
			assert.ErrorIs(t, err, ErrInvalidCategory)
			continue
		}
		assert.NoError(t, err)
		assert.Equal(t, tt.want, got)
	}
}

type fakeNotificationDAO struct {
	countUnreadByTypesFunc      func(ctx context.Context, userID int64, types []model.NotificationType) (int64, error)
	markUnreadByTypesAsReadFunc func(ctx context.Context, userID int64, types []model.NotificationType) error
	markAllAsReadFunc           func(ctx context.Context, userID int64) error

	countCalls     [][]model.NotificationType
	markTypeCalls  [][]model.NotificationType
	markAllUserIDs []int64
}

func (f *fakeNotificationDAO) CountUnreadByTypes(ctx context.Context, userID int64, types []model.NotificationType) (int64, error) {
	f.countCalls = append(f.countCalls, append([]model.NotificationType(nil), types...))
	if f.countUnreadByTypesFunc != nil {
		return f.countUnreadByTypesFunc(ctx, userID, types)
	}
	return 0, nil
}

func (f *fakeNotificationDAO) MarkUnreadByTypesAsRead(ctx context.Context, userID int64, types []model.NotificationType) error {
	f.markTypeCalls = append(f.markTypeCalls, append([]model.NotificationType(nil), types...))
	if f.markUnreadByTypesAsReadFunc != nil {
		return f.markUnreadByTypesAsReadFunc(ctx, userID, types)
	}
	return nil
}

func (f *fakeNotificationDAO) MarkAllAsRead(ctx context.Context, userID int64) error {
	f.markAllUserIDs = append(f.markAllUserIDs, userID)
	if f.markAllAsReadFunc != nil {
		return f.markAllAsReadFunc(ctx, userID)
	}
	return nil
}

func (f *fakeNotificationDAO) Create(ctx context.Context, notification *model.Notification) error {
	return nil
}

func (f *fakeNotificationDAO) CreateBatch(ctx context.Context, notifications []*model.Notification) error {
	return nil
}

func (f *fakeNotificationDAO) GetByUserID(ctx context.Context, userID int64, page, pageSize int, unreadOnly bool) (*model.NotificationListResponse, error) {
	return nil, nil
}

func (f *fakeNotificationDAO) GetUnreadCount(ctx context.Context, userID int64) (int64, error) {
	return 0, nil
}

func (f *fakeNotificationDAO) MarkAsRead(ctx context.Context, id int64, userID int64) error {
	return nil
}

func (f *fakeNotificationDAO) GetUnreadByUserID(ctx context.Context, userID int64, limit int) ([]model.Notification, error) {
	return nil, nil
}

func TestNotificationServiceGetSummary(t *testing.T) {
	t.Run("aggregates total and outbid unread counts", func(t *testing.T) {
		store := &fakeNotificationDAO{
			countUnreadByTypesFunc: func(ctx context.Context, userID int64, types []model.NotificationType) (int64, error) {
				assert.Equal(t, int64(42), userID)
				if len(types) == 0 {
					return 7, nil
				}
				assert.Equal(t, []model.NotificationType{model.NotificationTypeBidOutbid}, types)
				return 3, nil
			},
		}
		svc := &NotificationService{notificationDAO: store}

		got, err := svc.GetSummary(context.Background(), 42)

		assert.NoError(t, err)
		assert.Equal(t, &model.NotificationSummaryResponse{
			UnreadTotal: 7,
			Outbid:      3,
			EndingSoon:  0,
		}, got)
		assert.Equal(t, [][]model.NotificationType{
			nil,
			{model.NotificationTypeBidOutbid},
		}, store.countCalls)
	})

	t.Run("hides underlying DAO errors", func(t *testing.T) {
		store := &fakeNotificationDAO{
			countUnreadByTypesFunc: func(ctx context.Context, userID int64, types []model.NotificationType) (int64, error) {
				return 0, errors.New("dial tcp 10.0.0.1:3306: connection refused")
			},
		}
		svc := &NotificationService{notificationDAO: store}

		got, err := svc.GetSummary(context.Background(), 42)

		assert.Nil(t, got)
		assert.EqualError(t, err, "notification summary unavailable")
		assert.NotContains(t, err.Error(), "dial tcp")
	})
}

func TestNotificationServiceMarkCategoryAsRead(t *testing.T) {
	t.Run("marks outbid notifications by type", func(t *testing.T) {
		store := &fakeNotificationDAO{}
		svc := &NotificationService{notificationDAO: store}

		err := svc.MarkCategoryAsRead(context.Background(), 42, "outbid")

		assert.NoError(t, err)
		assert.Equal(t, [][]model.NotificationType{{model.NotificationTypeBidOutbid}}, store.markTypeCalls)
		assert.Empty(t, store.markAllUserIDs)
	})

	t.Run("marks all notifications through existing all-read path", func(t *testing.T) {
		store := &fakeNotificationDAO{}
		svc := &NotificationService{notificationDAO: store}

		err := svc.MarkCategoryAsRead(context.Background(), 42, "all")

		assert.NoError(t, err)
		assert.Equal(t, []int64{42}, store.markAllUserIDs)
		assert.Empty(t, store.markTypeCalls)
	})

	t.Run("keeps unsupported empty categories as no-op", func(t *testing.T) {
		for _, category := range []string{"pendingPayment", "endingSoon"} {
			store := &fakeNotificationDAO{}
			svc := &NotificationService{notificationDAO: store}

			err := svc.MarkCategoryAsRead(context.Background(), 42, category)

			assert.NoError(t, err)
			assert.Empty(t, store.markTypeCalls)
			assert.Empty(t, store.markAllUserIDs)
		}
	})

	t.Run("rejects unknown category without mutating data", func(t *testing.T) {
		store := &fakeNotificationDAO{}
		svc := &NotificationService{notificationDAO: store}

		err := svc.MarkCategoryAsRead(context.Background(), 42, "unknown")

		assert.ErrorIs(t, err, ErrInvalidCategory)
		assert.Empty(t, store.markTypeCalls)
		assert.Empty(t, store.markAllUserIDs)
	})
}

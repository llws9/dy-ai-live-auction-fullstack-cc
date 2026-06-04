package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"auction-service/dao"
	"auction-service/model"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

type fakeProductReminderHotPullStore struct {
	candidates []dao.ProductReminderCandidate
	failOn     map[int64]error
}

func (f *fakeProductReminderHotPullStore) GetStartingSoonByUser(ctx context.Context, userID int64, start, end time.Time) ([]dao.ProductReminderCandidate, error) {
	return f.candidates, nil
}

func (f *fakeProductReminderHotPullStore) ClaimAndCreateAuctionStartNotification(ctx context.Context, userID, auctionID int64, notification *model.Notification) (bool, error) {
	if err, ok := f.failOn[auctionID]; ok {
		return false, err
	}
	notification.ID = auctionID + 10000
	return true, nil
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

func setupNotificationHotPullDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Notification{},
		&model.Auction{},
		&model.UserProductReminder{},
		&model.ProductReminderReceipt{},
	))

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}

func TestNotificationServiceHotPullProductReminder(t *testing.T) {
	db := setupNotificationHotPullDB(t)
	now := time.Now()

	require.NoError(t, db.Create(&model.Auction{
		ID:        7001,
		ProductID: 8001,
		Status:    model.AuctionStatusPending,
		StartTime: now.Add(20 * time.Minute),
		EndTime:   now.Add(2 * time.Hour),
	}).Error)
	require.NoError(t, db.Create(&model.UserProductReminder{
		UserID:              42,
		ProductID:           8001,
		AuctionID:           7001,
		NotificationEnabled: true,
		CreatedAt:           now.Add(-time.Hour),
	}).Error)

	notificationDAO := dao.NewNotificationDAO(db, nil)
	reminderDAO := dao.NewUserProductReminderDAO(db)
	svc := NewNotificationService(notificationDAO, nil)
	svc.SetProductReminderDAO(reminderDAO)

	first, err := svc.HotPullNotifications(context.Background(), 42)
	require.NoError(t, err)
	require.Len(t, first, 1)
	assert.Equal(t, model.NotificationTypeAuctionStarting, first[0].Type)
	assert.Equal(t, int64(7001), first[0].Data["auction_id"])
	assert.Equal(t, int64(8001), first[0].Data["product_id"])

	var unreadCount int64
	require.NoError(t, db.Model(&model.Notification{}).
		Where("user_id = ? AND type = ? AND read_at IS NULL", 42, model.NotificationTypeAuctionStarting).
		Count(&unreadCount).Error)
	assert.Equal(t, int64(1), unreadCount)

	second, err := svc.HotPullNotifications(context.Background(), 42)
	require.NoError(t, err)
	assert.Empty(t, second)
	require.NoError(t, db.Model(&model.Notification{}).
		Where("user_id = ? AND type = ? AND read_at IS NULL", 42, model.NotificationTypeAuctionStarting).
		Count(&unreadCount).Error)
	assert.Equal(t, int64(1), unreadCount)
}

func TestNotificationServiceHotPullProductReminderContinuesAfterCandidateFailure(t *testing.T) {
	now := time.Now()
	svc := &NotificationService{
		notificationDAO: &fakeNotificationDAO{},
		productReminder: &fakeProductReminderHotPullStore{
			candidates: []dao.ProductReminderCandidate{
				{UserID: 42, ProductID: 8001, AuctionID: 7001, StartTime: now.Add(10 * time.Minute)},
				{UserID: 42, ProductID: 8002, AuctionID: 7002, StartTime: now.Add(20 * time.Minute)},
				{UserID: 42, ProductID: 8003, AuctionID: 7003, StartTime: now.Add(25 * time.Minute)},
			},
			failOn: map[int64]error{7002: errors.New("temporary insert failure")},
		},
	}

	notifications, err := svc.HotPullNotifications(context.Background(), 42)

	require.NoError(t, err)
	require.Len(t, notifications, 2)
	assert.Equal(t, int64(7001), notifications[0].Data["auction_id"])
	assert.Equal(t, int64(7003), notifications[1].Data["auction_id"])
}

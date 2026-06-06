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
)

// TestStateMachine_CanBid 测试出价权限判断
func TestStateMachine_CanBid(t *testing.T) {
	tests := []struct {
		name     string
		status   model.AuctionStatus
		expected bool
	}{
		{"待开始不可出价", model.AuctionStatusPending, false},
		{"进行中可以出价", model.AuctionStatusOngoing, true},
		{"延时中可以出价", model.AuctionStatusDelayed, true},
		{"已结束不可出价", model.AuctionStatusEnded, false},
		{"已取消不可出价", model.AuctionStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auction := &model.Auction{Status: tt.status, EndTime: time.Now().Add(time.Hour)}
			sm := NewStateMachine(auction)
			assert.Equal(t, tt.expected, sm.CanBid())
		})
	}
}

// TestStateMachine_ShouldTriggerDelay 测试延时触发判断
func TestStateMachine_ShouldTriggerDelay(t *testing.T) {
	tests := []struct {
		name            string
		endTime         time.Time
		status          model.AuctionStatus
		triggerBefore   int
		expectedTrigger bool
	}{
		{
			name:            "进行中且在延时窗口内",
			endTime:         time.Now().Add(20 * time.Second),
			status:          model.AuctionStatusOngoing,
			triggerBefore:   30,
			expectedTrigger: true,
		},
		{
			name:            "延时中且在延时窗口内",
			endTime:         time.Now().Add(20 * time.Second),
			status:          model.AuctionStatusDelayed,
			triggerBefore:   30,
			expectedTrigger: true,
		},
		{
			name:            "不在延时窗口内",
			endTime:         time.Now().Add(2 * time.Minute),
			status:          model.AuctionStatusOngoing,
			triggerBefore:   30,
			expectedTrigger: false,
		},
		{
			name:            "已结束不触发延时",
			endTime:         time.Now().Add(-1 * time.Second),
			status:          model.AuctionStatusEnded,
			triggerBefore:   30,
			expectedTrigger: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auction := &model.Auction{
				EndTime: tt.endTime,
				Status:  tt.status,
			}
			sm := NewStateMachine(auction)
			result := sm.ShouldTriggerDelay(tt.triggerBefore)
			assert.Equal(t, tt.expectedTrigger, result)
		})
	}
}

// TestStateMachine_CanDelay 测试是否可以继续延时
func TestStateMachine_CanDelay(t *testing.T) {
	tests := []struct {
		name        string
		delayUsed   int
		maxDelay    int
		expectedCan bool
	}{
		{
			name:        "未延时可以继续",
			delayUsed:   0,
			maxDelay:    180,
			expectedCan: true,
		},
		{
			name:        "部分延时可以继续",
			delayUsed:   100,
			maxDelay:    180,
			expectedCan: true,
		},
		{
			name:        "已达最大延时不可继续",
			delayUsed:   180,
			maxDelay:    180,
			expectedCan: false,
		},
		{
			name:        "超过最大延时不可继续",
			delayUsed:   200,
			maxDelay:    180,
			expectedCan: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auction := &model.Auction{
				DelayUsed: tt.delayUsed,
			}
			sm := NewStateMachine(auction)
			result := sm.CanDelay(tt.maxDelay)
			assert.Equal(t, tt.expectedCan, result)
		})
	}
}

// TestStateMachine_GetRemainingDelayTime 测试剩余延时时长计算
func TestStateMachine_GetRemainingDelayTime(t *testing.T) {
	tests := []struct {
		name      string
		delayUsed int
		maxDelay  int
		expected  int
	}{
		{"未延时", 0, 180, 180},
		{"已延时部分", 100, 180, 80},
		{"已达最大", 180, 180, 0},
		{"超过最大", 200, 180, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auction := &model.Auction{DelayUsed: tt.delayUsed}
			sm := NewStateMachine(auction)
			result := sm.GetRemainingDelayTime(tt.maxDelay)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAuction_IsInDelayWindow 测试延时窗口判断
func TestAuction_IsInDelayWindow(t *testing.T) {
	tests := []struct {
		name             string
		endTime          time.Time
		triggerBefore    int
		expectedInWindow bool
	}{
		{
			name:             "在延时窗口内（剩余20秒）",
			endTime:          time.Now().Add(20 * time.Second),
			triggerBefore:    30,
			expectedInWindow: true,
		},
		{
			name:             "不在延时窗口内（剩余40秒）",
			endTime:          time.Now().Add(40 * time.Second),
			triggerBefore:    30,
			expectedInWindow: false,
		},
		{
			name:             "刚好在边界（剩余30秒）",
			endTime:          time.Now().Add(30 * time.Second),
			triggerBefore:    30,
			expectedInWindow: true,
		},
		{
			name:             "竞拍已结束",
			endTime:          time.Now().Add(-1 * time.Second),
			triggerBefore:    30,
			expectedInWindow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auction := &model.Auction{EndTime: tt.endTime}
			result := auction.IsInDelayWindow(tt.triggerBefore)
			assert.Equal(t, tt.expectedInWindow, result)
		})
	}
}

// TestAuction_CanBid 测试竞拍模型方法
func TestAuction_CanBid_Method(t *testing.T) {
	tests := []struct {
		name     string
		status   model.AuctionStatus
		expected bool
	}{
		{"进行中可以出价", model.AuctionStatusOngoing, true},
		{"延时中可以出价", model.AuctionStatusDelayed, true},
		{"待开始不可出价", model.AuctionStatusPending, false},
		{"已结束不可出价", model.AuctionStatusEnded, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auction := &model.Auction{Status: tt.status, EndTime: time.Now().Add(time.Hour)}
			assert.Equal(t, tt.expected, auction.CanBid())
		})
	}
}

type recordingOrderCreator struct {
	err   error
	calls []model.AuctionOrderRequest
}

func (r *recordingOrderCreator) CreateOrderFromAuctionResult(_ context.Context, req model.AuctionOrderRequest) error {
	r.calls = append(r.calls, req)
	return r.err
}

type recordingNotificationSender struct {
	err       error
	batchErr  error
	sent      []model.NotificationRequest
	batchSent []model.NotificationRequest
}

func (r *recordingNotificationSender) SendNotification(_ context.Context, req *model.NotificationRequest) error {
	r.sent = append(r.sent, *req)
	return r.err
}

func (r *recordingNotificationSender) SendBatchNotifications(_ context.Context, reqs []*model.NotificationRequest) error {
	for _, req := range reqs {
		r.batchSent = append(r.batchSent, *req)
	}
	return r.batchErr
}

func newAuctionServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Auction{}, &model.Bid{}, &model.AuctionSettlementTask{}))
	return db
}

func TestEndAuctionCreatesPendingOrderBeforeWinnerNotification(t *testing.T) {
	db := newAuctionServiceTestDB(t)
	winnerID := int64(2001)
	auction := &model.Auction{
		ID:           101,
		ProductID:    11,
		Status:       model.AuctionStatusOngoing,
		CurrentPrice: decimal.NewFromInt(110),
		WinnerID:     &winnerID,
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      time.Now().Add(-time.Second),
	}
	require.NoError(t, db.Create(auction).Error)
	require.NoError(t, db.Create(&model.Bid{AuctionID: auction.ID, UserID: winnerID, Amount: decimal.NewFromInt(110)}).Error)

	orderCreator := &recordingOrderCreator{}
	notifications := &recordingNotificationSender{}
	svc := NewAuctionService(dao.NewAuctionDAO(db))
	svc.SetBidDAO(dao.NewBidDAO(db))
	svc.SetOrderCreator(orderCreator)
	svc.SetNotificationSender(notifications)

	err := svc.EndAuction(context.Background(), auction.ID)

	require.NoError(t, err)
	require.Len(t, orderCreator.calls, 1)
	assert.Equal(t, int64(101), orderCreator.calls[0].AuctionID)
	assert.Equal(t, int64(11), orderCreator.calls[0].ProductID)
	assert.Equal(t, int64(2001), orderCreator.calls[0].WinnerID)
	assert.True(t, orderCreator.calls[0].FinalPrice.Equal(decimal.NewFromInt(110)))
	require.Len(t, notifications.sent, 1)
	assert.Equal(t, model.NotificationTypeAuctionWon, notifications.sent[0].Type)

	var task model.AuctionSettlementTask
	require.NoError(t, db.First(&task, "auction_id = ?", auction.ID).Error)
	assert.Equal(t, model.AuctionSettlementTaskStatusDone, task.Status)
}

func TestEndAuctionKeepsSettlementTaskRetryableWhenOrderCreationFails(t *testing.T) {
	db := newAuctionServiceTestDB(t)
	winnerID := int64(2001)
	auction := &model.Auction{
		ID:           102,
		ProductID:    11,
		Status:       model.AuctionStatusOngoing,
		CurrentPrice: decimal.NewFromInt(110),
		WinnerID:     &winnerID,
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      time.Now().Add(-time.Second),
	}
	require.NoError(t, db.Create(auction).Error)
	require.NoError(t, db.Create(&model.Bid{AuctionID: auction.ID, UserID: winnerID, Amount: decimal.NewFromInt(110)}).Error)

	orderCreator := &recordingOrderCreator{err: errors.New("product-service unavailable")}
	notifications := &recordingNotificationSender{}
	svc := NewAuctionService(dao.NewAuctionDAO(db))
	svc.SetBidDAO(dao.NewBidDAO(db))
	svc.SetOrderCreator(orderCreator)
	svc.SetNotificationSender(notifications)

	err := svc.EndAuction(context.Background(), auction.ID)

	require.Error(t, err)
	require.Len(t, orderCreator.calls, 1)
	assert.Empty(t, notifications.sent)

	var saved model.Auction
	require.NoError(t, db.First(&saved, auction.ID).Error)
	assert.Equal(t, model.AuctionStatusEnded, saved.Status)

	var task model.AuctionSettlementTask
	require.NoError(t, db.First(&task, "auction_id = ?", auction.ID).Error)
	assert.Equal(t, model.AuctionSettlementTaskStatusPending, task.Status)
	assert.Contains(t, task.LastError, "product-service unavailable")
}

// TestAuction_IsEnded 测试竞拍是否已结束
func TestAuction_IsEnded(t *testing.T) {
	tests := []struct {
		name     string
		status   model.AuctionStatus
		expected bool
	}{
		{"已结束", model.AuctionStatusEnded, true},
		{"已取消", model.AuctionStatusCancelled, true},
		{"进行中", model.AuctionStatusOngoing, false},
		{"延时中", model.AuctionStatusDelayed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auction := &model.Auction{Status: tt.status}
			assert.Equal(t, tt.expected, auction.IsEnded())
		})
	}
}

// TestAuctionService_Creation 测试竞拍服务创建
func TestAuctionService_Creation(t *testing.T) {
	// 测试竞拍创建请求验证
	req := &CreateAuctionRequest{
		ProductID: 1,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(1 * time.Hour),
	}

	assert.Equal(t, int64(1), req.ProductID)
	assert.True(t, req.EndTime.After(req.StartTime))
}

// TestNotificationType_Constants 测试通知类型常量
func TestNotificationType_Constants(t *testing.T) {
	assert.Equal(t, model.NotificationType("bid_outbid"), model.NotificationTypeBidOutbid)
	assert.Equal(t, model.NotificationType("auction_won"), model.NotificationTypeAuctionWon)
	assert.Equal(t, model.NotificationType("auction_lost"), model.NotificationTypeAuctionLost)
}

// TestStateMachine_Transitions 测试状态机转换
func TestStateMachine_Transitions(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus model.AuctionStatus
		targetStatus  model.AuctionStatus
		canTransition bool
	}{
		{"Pending to Ongoing", model.AuctionStatusPending, model.AuctionStatusOngoing, true},
		{"Ongoing to Ended", model.AuctionStatusOngoing, model.AuctionStatusEnded, true},
		{"Ongoing to Cancelled", model.AuctionStatusOngoing, model.AuctionStatusCancelled, true},
		{"Pending to Ended", model.AuctionStatusPending, model.AuctionStatusEnded, false},
		{"Ended to Ongoing", model.AuctionStatusEnded, model.AuctionStatusOngoing, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auction := &model.Auction{Status: tt.currentStatus}
			sm := NewStateMachine(auction)
			err := sm.Transition(tt.targetStatus)

			if tt.canTransition {
				assert.NoError(t, err)
				assert.Equal(t, tt.targetStatus, auction.Status)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

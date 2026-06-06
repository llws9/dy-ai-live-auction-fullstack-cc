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

func newBidSettlementTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Auction{},
		&model.AuctionRule{},
		&model.Bid{},
		&model.AuctionSettlementTask{},
	))
	return db
}

func TestPlaceBidAtCapPriceFinalizesAuctionResult(t *testing.T) {
	db := newBidSettlementTestDB(t)
	rdb := setupTestRedis(t)
	previousRedis := dao.GetRedis()
	dao.RedisClient = rdb
	t.Cleanup(func() { dao.RedisClient = previousRedis })

	capPrice := decimal.NewFromInt(100)
	require.NoError(t, db.Create(&model.User{
		ID:       2001,
		Name:     "buyer",
		Password: "password",
		Status:   1,
	}).Error)
	require.NoError(t, db.Create(&model.Auction{
		ID:           301,
		ProductID:    401,
		Status:       model.AuctionStatusOngoing,
		CurrentPrice: decimal.NewFromInt(90),
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      time.Now().Add(time.Hour),
	}).Error)
	require.NoError(t, db.Create(&model.AuctionRule{
		ProductID:          401,
		StartPrice:         decimal.NewFromInt(80),
		Increment:          decimal.NewFromInt(10),
		CapPrice:           &capPrice,
		Duration:           60,
		DelayDuration:      30,
		MaxDelayTime:       180,
		TriggerDelayBefore: 30,
	}).Error)

	auctionDAO := dao.NewAuctionDAO(db)
	bidDAO := dao.NewBidDAO(db)
	orderCreator := &recordingOrderCreator{}
	notifications := &recordingNotificationSender{}
	settlement := NewAuctionSettlementService(auctionDAO, bidDAO)
	settlement.SetOrderCreator(orderCreator)
	settlement.SetNotificationSender(notifications)

	bidSvc := NewBidService(
		auctionDAO,
		bidDAO,
		dao.NewAuctionRuleDAO(db),
		dao.NewUserDAO(db),
	)
	bidSvc.SetSettlementService(settlement)

	result, err := bidSvc.PlaceBid(context.Background(), &PlaceBidRequest{
		AuctionID: 301,
		UserID:    2001,
		Amount:    decimal.NewFromInt(120),
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Success)
	assert.Equal(t, int64(2001), result.WinnerID)
	assert.True(t, result.CurrentPrice.Equal(capPrice))

	require.Len(t, orderCreator.calls, 1)
	assert.Equal(t, int64(301), orderCreator.calls[0].AuctionID)
	assert.Equal(t, int64(401), orderCreator.calls[0].ProductID)
	assert.Equal(t, int64(2001), orderCreator.calls[0].WinnerID)
	assert.True(t, orderCreator.calls[0].FinalPrice.Equal(capPrice))

	require.Len(t, notifications.sent, 1)
	assert.Equal(t, model.NotificationTypeAuctionWon, notifications.sent[0].Type)
	assert.Equal(t, int64(2001), notifications.sent[0].UserID)

	var saved model.Auction
	require.NoError(t, db.First(&saved, int64(301)).Error)
	assert.Equal(t, model.AuctionStatusEnded, saved.Status)

	var task model.AuctionSettlementTask
	require.NoError(t, db.First(&task, "auction_id = ?", int64(301)).Error)
	assert.Equal(t, model.AuctionSettlementTaskStatusDone, task.Status)
}

func TestCapPriceSettlementRetryCompletesPendingTask(t *testing.T) {
	db := newBidSettlementTestDB(t)
	rdb := setupTestRedis(t)
	previousRedis := dao.GetRedis()
	dao.RedisClient = rdb
	t.Cleanup(func() { dao.RedisClient = previousRedis })

	capPrice := decimal.NewFromInt(100)
	require.NoError(t, db.Create(&model.User{
		ID:       2001,
		Name:     "buyer",
		Password: "password",
		Status:   1,
	}).Error)
	require.NoError(t, db.Create(&model.Auction{
		ID:           302,
		ProductID:    402,
		Status:       model.AuctionStatusOngoing,
		CurrentPrice: decimal.NewFromInt(90),
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      time.Now().Add(time.Hour),
	}).Error)
	require.NoError(t, db.Create(&model.AuctionRule{
		ProductID:          402,
		StartPrice:         decimal.NewFromInt(80),
		Increment:          decimal.NewFromInt(10),
		CapPrice:           &capPrice,
		Duration:           60,
		DelayDuration:      30,
		MaxDelayTime:       180,
		TriggerDelayBefore: 30,
	}).Error)

	auctionDAO := dao.NewAuctionDAO(db)
	bidDAO := dao.NewBidDAO(db)
	orderCreator := &recordingOrderCreator{err: errors.New("product-service unavailable")}
	notifications := &recordingNotificationSender{}
	settlement := NewAuctionSettlementService(auctionDAO, bidDAO)
	settlement.SetOrderCreator(orderCreator)
	settlement.SetNotificationSender(notifications)

	bidSvc := NewBidService(
		auctionDAO,
		bidDAO,
		dao.NewAuctionRuleDAO(db),
		dao.NewUserDAO(db),
	)
	bidSvc.SetSettlementService(settlement)

	result, err := bidSvc.PlaceBid(context.Background(), &PlaceBidRequest{
		AuctionID: 302,
		UserID:    2001,
		Amount:    decimal.NewFromInt(120),
	})

	require.Error(t, err)
	require.Nil(t, result)
	require.Len(t, orderCreator.calls, 1)
	assert.Empty(t, notifications.sent)

	var saved model.Auction
	require.NoError(t, db.First(&saved, int64(302)).Error)
	assert.Equal(t, model.AuctionStatusEnded, saved.Status)

	var task model.AuctionSettlementTask
	require.NoError(t, db.First(&task, "auction_id = ?", int64(302)).Error)
	assert.Equal(t, model.AuctionSettlementTaskStatusPending, task.Status)

	orderCreator.err = nil
	require.NoError(t, settlement.RetryUnfinished(context.Background(), 10))

	require.Len(t, orderCreator.calls, 2)
	require.Len(t, notifications.sent, 1)
	assert.Equal(t, model.NotificationTypeAuctionWon, notifications.sent[0].Type)

	require.NoError(t, db.First(&task, "auction_id = ?", int64(302)).Error)
	assert.Equal(t, model.AuctionSettlementTaskStatusDone, task.Status)
}

func TestFinalizeEndedAuctionKeepsTaskRetryableWhenNotificationFails(t *testing.T) {
	db := newBidSettlementTestDB(t)
	winnerID := int64(2001)
	require.NoError(t, db.Create(&model.Auction{
		ID:           303,
		ProductID:    403,
		Status:       model.AuctionStatusEnded,
		CurrentPrice: decimal.NewFromInt(100),
		WinnerID:     &winnerID,
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      time.Now().Add(-time.Second),
	}).Error)
	require.NoError(t, db.Create(&model.Bid{AuctionID: 303, UserID: 2001, Amount: decimal.NewFromInt(100)}).Error)
	require.NoError(t, db.Create(&model.AuctionSettlementTask{
		AuctionID: 303,
		Status:    model.AuctionSettlementTaskStatusPending,
	}).Error)

	auctionDAO := dao.NewAuctionDAO(db)
	settlement := NewAuctionSettlementService(auctionDAO, dao.NewBidDAO(db))
	settlement.SetOrderCreator(&recordingOrderCreator{})
	settlement.SetNotificationSender(&recordingNotificationSender{err: errors.New("notification db unavailable")})

	err := settlement.FinalizeEndedAuction(context.Background(), 303)

	require.Error(t, err)
	var task model.AuctionSettlementTask
	require.NoError(t, db.First(&task, "auction_id = ?", int64(303)).Error)
	assert.Equal(t, model.AuctionSettlementTaskStatusOrderDone, task.Status)
	assert.Contains(t, task.LastError, "notification db unavailable")
}

func TestRetryUnfinishedRecordsOrderFailure(t *testing.T) {
	db := newBidSettlementTestDB(t)
	winnerID := int64(2001)
	require.NoError(t, db.Create(&model.Auction{
		ID:           304,
		ProductID:    404,
		Status:       model.AuctionStatusEnded,
		CurrentPrice: decimal.NewFromInt(100),
		WinnerID:     &winnerID,
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      time.Now().Add(-time.Second),
	}).Error)
	require.NoError(t, db.Create(&model.Bid{AuctionID: 304, UserID: 2001, Amount: decimal.NewFromInt(100)}).Error)
	require.NoError(t, db.Create(&model.AuctionSettlementTask{
		AuctionID: 304,
		Status:    model.AuctionSettlementTaskStatusPending,
	}).Error)

	auctionDAO := dao.NewAuctionDAO(db)
	settlement := NewAuctionSettlementService(auctionDAO, dao.NewBidDAO(db))
	settlement.SetOrderCreator(&recordingOrderCreator{err: errors.New("product-service unavailable")})
	settlement.SetNotificationSender(&recordingNotificationSender{})

	require.NoError(t, settlement.RetryUnfinished(context.Background(), 10))

	var task model.AuctionSettlementTask
	require.NoError(t, db.First(&task, "auction_id = ?", int64(304)).Error)
	assert.Equal(t, model.AuctionSettlementTaskStatusPending, task.Status)
	assert.Contains(t, task.LastError, "product-service unavailable")
}

func TestRetryUnfinishedResumesNotifyingTask(t *testing.T) {
	db := newBidSettlementTestDB(t)
	winnerID := int64(2001)
	require.NoError(t, db.Create(&model.Auction{
		ID:           305,
		ProductID:    405,
		Status:       model.AuctionStatusEnded,
		CurrentPrice: decimal.NewFromInt(100),
		WinnerID:     &winnerID,
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      time.Now().Add(-time.Second),
	}).Error)
	require.NoError(t, db.Create(&model.Bid{AuctionID: 305, UserID: 2001, Amount: decimal.NewFromInt(100)}).Error)
	require.NoError(t, db.Create(&model.AuctionSettlementTask{
		AuctionID: 305,
		Status:    model.AuctionSettlementTaskStatusNotifying,
	}).Error)

	notifications := &recordingNotificationSender{}
	settlement := NewAuctionSettlementService(dao.NewAuctionDAO(db), dao.NewBidDAO(db))
	settlement.SetOrderCreator(&recordingOrderCreator{})
	settlement.SetNotificationSender(notifications)

	require.NoError(t, settlement.RetryUnfinished(context.Background(), 10))

	require.Len(t, notifications.sent, 1)
	var task model.AuctionSettlementTask
	require.NoError(t, db.First(&task, "auction_id = ?", int64(305)).Error)
	assert.Equal(t, model.AuctionSettlementTaskStatusDone, task.Status)
}

func TestRecordFailureDoesNotOverwriteUnexpectedTaskStatus(t *testing.T) {
	db := newBidSettlementTestDB(t)
	taskDAO := dao.NewAuctionSettlementTaskDAO(db)
	require.NoError(t, db.Create(&model.AuctionSettlementTask{
		AuctionID: 306,
		Status:    model.AuctionSettlementTaskStatusDone,
	}).Error)

	updated, err := taskDAO.RecordFailureIfStatus(
		context.Background(),
		306,
		model.AuctionSettlementTaskStatusNotifying,
		model.AuctionSettlementTaskStatusOrderDone,
		errors.New("notification db unavailable"),
	)

	require.NoError(t, err)
	assert.False(t, updated)

	var task model.AuctionSettlementTask
	require.NoError(t, db.First(&task, "auction_id = ?", int64(306)).Error)
	assert.Equal(t, model.AuctionSettlementTaskStatusDone, task.Status)
	assert.Empty(t, task.LastError)
}

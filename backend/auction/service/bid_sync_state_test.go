package service

import (
	"context"
	"testing"
	"time"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/websocket"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBidServicePlaceBidSavesSyncState(t *testing.T) {
	db := setupServiceDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Auction{}, &model.Bid{}, &model.AuctionRule{}))
	rdb := setupTestRedis(t)
	previousRedis := dao.GetRedis()
	dao.RedisClient = rdb
	t.Cleanup(func() { dao.RedisClient = previousRedis })

	ctx := context.Background()
	endTime := time.Now().Add(time.Hour).Truncate(time.Millisecond)
	require.NoError(t, db.Create(&model.User{ID: 88, Name: "buyer", Password: "password", Status: 1}).Error)
	require.NoError(t, db.Create(&model.Auction{
		ID:           7,
		ProductID:    70,
		Status:       model.AuctionStatusOngoing,
		CurrentPrice: decimal.NewFromInt(100),
		StartTime:    time.Now().Add(-time.Hour),
		EndTime:      endTime,
	}).Error)
	require.NoError(t, db.Create(&model.AuctionRule{
		ProductID:  70,
		StartPrice: decimal.NewFromInt(100),
		Increment:  decimal.NewFromInt(10),
		Duration:   3600,
	}).Error)

	stateManager := websocket.NewStateManager(rdb)
	svc := NewBidService(dao.NewAuctionDAO(db), dao.NewBidDAO(db), dao.NewAuctionRuleDAO(db), dao.NewUserDAO(db))
	svc.SetStateManager(stateManager)

	result, err := svc.PlaceBid(ctx, &PlaceBidRequest{
		AuctionID: 7,
		UserID:    88,
		Amount:    decimal.NewFromInt(120),
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Success)
	state, err := stateManager.GetSyncState(ctx, 7)
	require.NoError(t, err)
	assert.Equal(t, int64(7), state.AuctionID)
	assert.True(t, decimal.NewFromInt(120).Equal(state.CurrentPrice))
	assert.Equal(t, int64(88), state.WinnerID)
	assert.Equal(t, int(model.AuctionStatusOngoing), state.Status)
	assert.True(t, state.EndTime.Equal(endTime))
}

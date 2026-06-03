package service

import (
	"context"
	"testing"
	"time"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/websocket"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuctionService_StartAuctionSavesSyncState(t *testing.T) {
	db := setupServiceDB(t)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()
	endTime := time.Now().Add(time.Hour).Truncate(time.Millisecond)
	auctionDAO := dao.NewAuctionDAO(db)
	require.NoError(t, auctionDAO.Create(ctx, &model.Auction{
		ID:           10,
		ProductID:    100,
		Status:       model.AuctionStatusPending,
		CurrentPrice: decimal.NewFromInt(500),
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      endTime,
	}))

	stateManager := websocket.NewStateManager(rdb)
	svc := NewAuctionService(auctionDAO)
	svc.SetStateManager(stateManager)

	require.NoError(t, svc.StartAuction(ctx, 10))

	state, err := stateManager.GetSyncState(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(10), state.AuctionID)
	assert.True(t, decimal.NewFromInt(500).Equal(state.CurrentPrice))
	assert.Equal(t, int(model.AuctionStatusOngoing), state.Status)
	assert.True(t, state.EndTime.Equal(endTime))
}

func TestAuctionService_EndAuctionSavesSyncState(t *testing.T) {
	db := setupServiceDB(t)
	require.NoError(t, db.AutoMigrate(&model.Auction{}, &model.Bid{}))

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()
	winnerID := int64(88)
	endTime := time.Now().Add(time.Hour).Truncate(time.Millisecond)
	auctionDAO := dao.NewAuctionDAO(db)
	require.NoError(t, auctionDAO.Create(ctx, &model.Auction{
		ID:           11,
		ProductID:    101,
		Status:       model.AuctionStatusOngoing,
		CurrentPrice: decimal.NewFromInt(900),
		WinnerID:     &winnerID,
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      endTime,
	}))

	stateManager := websocket.NewStateManager(rdb)
	svc := NewAuctionService(auctionDAO)
	svc.SetStateManager(stateManager)

	require.NoError(t, svc.EndAuction(ctx, 11))

	state, err := stateManager.GetSyncState(ctx, 11)
	require.NoError(t, err)
	assert.Equal(t, int64(11), state.AuctionID)
	assert.True(t, decimal.NewFromInt(900).Equal(state.CurrentPrice))
	assert.Equal(t, winnerID, state.WinnerID)
	assert.Equal(t, int(model.AuctionStatusEnded), state.Status)
}

func TestAuctionService_StartAuctionIgnoresSyncStateCacheWriteFailure(t *testing.T) {
	db := setupServiceDB(t)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))

	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	require.NoError(t, rdb.Close())

	ctx := context.Background()
	auctionDAO := dao.NewAuctionDAO(db)
	require.NoError(t, auctionDAO.Create(ctx, &model.Auction{
		ID:           12,
		ProductID:    102,
		Status:       model.AuctionStatusPending,
		CurrentPrice: decimal.NewFromInt(500),
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      time.Now().Add(time.Hour),
	}))

	svc := NewAuctionService(auctionDAO)
	svc.SetStateManager(websocket.NewStateManager(rdb))

	require.NoError(t, svc.StartAuction(ctx, 12))
	updated, err := auctionDAO.GetByID(ctx, 12)
	require.NoError(t, err)
	assert.Equal(t, model.AuctionStatusOngoing, updated.Status)
}

package service

import (
	"context"
	"testing"
	"time"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/websocket"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuctionSyncStateLoaderLoadsFromAuctionTable(t *testing.T) {
	db := setupServiceDB(t)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))

	ctx := context.Background()
	winnerID := int64(88)
	endTime := time.Now().Add(time.Hour).Truncate(time.Millisecond)
	auctionDAO := dao.NewAuctionDAO(db)
	require.NoError(t, auctionDAO.Create(ctx, &model.Auction{
		ID:           10,
		ProductID:    100,
		Status:       model.AuctionStatusOngoing,
		CurrentPrice: decimal.NewFromInt(500),
		WinnerID:     &winnerID,
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      endTime,
	}))

	loader := NewAuctionSyncStateLoader(auctionDAO)
	state, err := loader.LoadSyncState(ctx, 10)

	require.NoError(t, err)
	assert.Equal(t, int64(10), state.AuctionID)
	assert.True(t, decimal.NewFromInt(500).Equal(state.CurrentPrice))
	assert.Equal(t, winnerID, state.WinnerID)
	assert.Equal(t, int(model.AuctionStatusOngoing), state.Status)
	assert.True(t, state.EndTime.Equal(endTime))
}

func TestAuctionServiceStartAndEndAuctionSaveSyncState(t *testing.T) {
	db := setupServiceDB(t)
	require.NoError(t, db.AutoMigrate(&model.Auction{}, &model.Bid{}, &model.AuctionSettlementTask{}))
	rdb := setupTestRedis(t)

	ctx := context.Background()
	endTime := time.Now().Add(time.Hour).Truncate(time.Millisecond)
	auctionDAO := dao.NewAuctionDAO(db)
	require.NoError(t, auctionDAO.Create(ctx, &model.Auction{
		ID:           11,
		ProductID:    101,
		Status:       model.AuctionStatusPending,
		CurrentPrice: decimal.NewFromInt(900),
		StartTime:    time.Now().Add(-time.Minute),
		EndTime:      endTime,
	}))

	stateManager := websocket.NewStateManager(rdb)
	svc := NewAuctionService(auctionDAO)
	svc.SetBidDAO(dao.NewBidDAO(db))
	svc.SetStateManager(stateManager)

	require.NoError(t, svc.StartAuction(ctx, 11))
	state, err := stateManager.GetSyncState(ctx, 11)
	require.NoError(t, err)
	assert.Equal(t, int(model.AuctionStatusOngoing), state.Status)
	assert.True(t, decimal.NewFromInt(900).Equal(state.CurrentPrice))

	require.NoError(t, svc.EndAuction(ctx, 11))
	state, err = stateManager.GetSyncState(ctx, 11)
	require.NoError(t, err)
	assert.Equal(t, int(model.AuctionStatusEnded), state.Status)
}

func TestSaveAuctionSyncStateIgnoresNilInputsAndPropagatesWriteFailure(t *testing.T) {
	require.NoError(t, SaveAuctionSyncState(context.Background(), nil, nil))

	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	require.NoError(t, rdb.Close())
	stateManager := websocket.NewStateManager(rdb)
	err := SaveAuctionSyncState(context.Background(), stateManager, &model.Auction{ID: 99})
	require.Error(t, err)
}

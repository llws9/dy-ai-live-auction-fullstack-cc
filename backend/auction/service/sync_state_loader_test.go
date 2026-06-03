package service

import (
	"context"
	"testing"
	"time"

	"auction-service/dao"
	"auction-service/model"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuctionSyncStateLoader_LoadSyncStateMapsAuctionSnapshot(t *testing.T) {
	db := setupServiceDB(t)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))

	winnerID := int64(42)
	endTime := time.Now().Add(30 * time.Minute).Truncate(time.Millisecond)
	auction := &model.Auction{
		ID:           6,
		ProductID:    10,
		Status:       model.AuctionStatusOngoing,
		CurrentPrice: decimal.NewFromInt(8800),
		WinnerID:     &winnerID,
		StartTime:    time.Now().Add(-30 * time.Minute),
		EndTime:      endTime,
	}
	require.NoError(t, dao.NewAuctionDAO(db).Create(context.Background(), auction))

	loader := NewAuctionSyncStateLoader(dao.NewAuctionDAO(db))

	state, err := loader.LoadSyncState(context.Background(), 6)

	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, int64(6), state.AuctionID)
	assert.True(t, decimal.NewFromInt(8800).Equal(state.CurrentPrice))
	assert.Equal(t, winnerID, state.WinnerID)
	assert.Equal(t, int(model.AuctionStatusOngoing), state.Status)
	assert.True(t, state.EndTime.Equal(endTime))
}

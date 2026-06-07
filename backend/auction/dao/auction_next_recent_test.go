package dao

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"auction-service/model"
)

func ptrInt64(v int64) *int64 { return &v }

func TestGetNextByLiveStreamIDs(t *testing.T) {
	db := newCurrentTestDB(t)
	d := NewAuctionDAO(db)
	now := time.Now()

	require.NoError(t, db.Create(&model.Auction{ID: 11, ProductID: 101, LiveStreamID: ptrInt64(1), Status: model.AuctionStatusPending, CurrentPrice: decimal.NewFromInt(100), StartTime: now.Add(10 * time.Minute), EndTime: now.Add(40 * time.Minute)}).Error)
	require.NoError(t, db.Create(&model.Auction{ID: 12, ProductID: 102, LiveStreamID: ptrInt64(1), Status: model.AuctionStatusPending, CurrentPrice: decimal.NewFromInt(200), StartTime: now.Add(30 * time.Minute), EndTime: now.Add(60 * time.Minute)}).Error)
	require.NoError(t, db.Create(&model.Auction{ID: 13, ProductID: 103, LiveStreamID: ptrInt64(1), Status: model.AuctionStatusOngoing, CurrentPrice: decimal.NewFromInt(300), StartTime: now.Add(-5 * time.Minute), EndTime: now.Add(20 * time.Minute)}).Error)
	require.NoError(t, db.Create(&model.Auction{ID: 21, ProductID: 201, LiveStreamID: ptrInt64(2), Status: model.AuctionStatusEnded, CurrentPrice: decimal.NewFromInt(500), StartTime: now.Add(-60 * time.Minute), EndTime: now.Add(-30 * time.Minute)}).Error)

	got, err := d.GetNextByLiveStreamIDs(context.Background(), []int64{1, 2})
	require.NoError(t, err)
	require.Contains(t, got, int64(1))
	require.NotContains(t, got, int64(2))
	require.Equal(t, int64(11), got[1].ID)
	require.Equal(t, int64(101), got[1].ProductID)
}

func TestGetNextByLiveStreamIDsEmpty(t *testing.T) {
	db := newCurrentTestDB(t)
	d := NewAuctionDAO(db)
	got, err := d.GetNextByLiveStreamIDs(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, got)
}

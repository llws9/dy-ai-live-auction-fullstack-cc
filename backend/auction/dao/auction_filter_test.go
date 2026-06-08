package dao

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"auction-service/model"
)

func newFilterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Auction{}, &model.Bid{}))
	return db
}

func TestListWithFiltersPriceRange(t *testing.T) {
	db := newFilterTestDB(t)
	dao := NewAuctionDAO(db)
	ctx := context.Background()

	now := time.Now()
	rows := []model.Auction{
		{ID: 1, ProductID: 1, Status: model.AuctionStatusOngoing, CurrentPrice: decimal.NewFromInt(500), StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)},
		{ID: 2, ProductID: 2, Status: model.AuctionStatusOngoing, CurrentPrice: decimal.NewFromInt(2000), StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)},
		{ID: 3, ProductID: 3, Status: model.AuctionStatusOngoing, CurrentPrice: decimal.NewFromInt(8000), StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)},
	}
	for i := range rows {
		require.NoError(t, db.Create(&rows[i]).Error)
	}

	min := decimal.NewFromInt(1000)
	max := decimal.NewFromInt(5000)
	got, total, err := dao.ListWithFilters(ctx, &AuctionFilters{PriceMin: &min, PriceMax: &max}, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, got, 1)
	require.Equal(t, int64(2), got[0].ID)
}

func TestListWithFiltersSortByHot(t *testing.T) {
	db := newFilterTestDB(t)
	dao := NewAuctionDAO(db)
	ctx := context.Background()

	now := time.Now()
	auctions := []model.Auction{
		{ID: 1, ProductID: 1, Status: model.AuctionStatusOngoing, CurrentPrice: decimal.NewFromInt(100), StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)},
		{ID: 2, ProductID: 2, Status: model.AuctionStatusOngoing, CurrentPrice: decimal.NewFromInt(100), StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)},
	}
	for i := range auctions {
		require.NoError(t, db.Create(&auctions[i]).Error)
	}
	// auction 2 有 3 次出价，auction 1 有 1 次出价。
	bids := []model.Bid{
		{ID: 1, AuctionID: 1, UserID: 10, Amount: decimal.NewFromInt(110), CreatedAt: now},
		{ID: 2, AuctionID: 2, UserID: 11, Amount: decimal.NewFromInt(110), CreatedAt: now},
		{ID: 3, AuctionID: 2, UserID: 12, Amount: decimal.NewFromInt(120), CreatedAt: now},
		{ID: 4, AuctionID: 2, UserID: 13, Amount: decimal.NewFromInt(130), CreatedAt: now},
	}
	for i := range bids {
		require.NoError(t, db.Create(&bids[i]).Error)
	}

	got, total, err := dao.ListWithFilters(ctx, &AuctionFilters{SortByHot: true}, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, got, 2)
	require.Equal(t, int64(2), got[0].ID, "出价多的应排最前")
	require.Equal(t, 3, got[0].BidCount, "应回填 bid_count")
	require.Equal(t, int64(1), got[1].ID)
	require.Equal(t, 1, got[1].BidCount)
}

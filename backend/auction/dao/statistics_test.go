package dao

import (
	"context"
	"testing"
	"time"

	"auction-service/model"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestStatisticsDAOListAuctionDailyStats(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Auction{}, &model.Bid{}))
	dao := NewStatisticsDAO(db)
	ctx := context.Background()
	day1 := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	winnerID := int64(2001)
	merchantID := int64(9)
	otherMerchantID := int64(10)

	auctions := []model.Auction{
		{ID: 101, CreatorID: &merchantID, ProductID: 1, Status: model.AuctionStatusEnded, WinnerID: &winnerID, CurrentPrice: decimal.NewFromInt(120), StartTime: day1, EndTime: day1.Add(time.Hour)},
		{ID: 102, CreatorID: &merchantID, ProductID: 2, Status: model.AuctionStatusEnded, WinnerID: nil, CurrentPrice: decimal.NewFromInt(80), StartTime: day1, EndTime: day1.Add(time.Hour)},
		{ID: 103, CreatorID: &merchantID, ProductID: 3, Status: model.AuctionStatusOngoing, WinnerID: nil, CurrentPrice: decimal.NewFromInt(50), StartTime: day2, EndTime: day2.Add(time.Hour)},
		{ID: 104, CreatorID: &otherMerchantID, ProductID: 4, Status: model.AuctionStatusEnded, WinnerID: &winnerID, CurrentPrice: decimal.NewFromInt(500), StartTime: day1, EndTime: day1.Add(time.Hour)},
	}
	require.NoError(t, db.Create(&auctions).Error)
	bids := []model.Bid{
		{AuctionID: 101, UserID: 1, Amount: decimal.NewFromInt(100), CreatedAt: day1},
		{AuctionID: 101, UserID: 2, Amount: decimal.NewFromInt(120), CreatedAt: day1},
		{AuctionID: 102, UserID: 3, Amount: decimal.NewFromInt(80), CreatedAt: day1},
		{AuctionID: 104, UserID: 4, Amount: decimal.NewFromInt(500), CreatedAt: day1},
	}
	require.NoError(t, db.Create(&bids).Error)

	rows, err := dao.ListAuctionDailyStats(ctx, day1.Truncate(24*time.Hour), day2.AddDate(0, 0, 1), &merchantID)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	require.Equal(t, "2026-06-01", rows[0].Date)
	require.Equal(t, int64(2), rows[0].AuctionCount)
	require.Equal(t, int64(3), rows[0].BidCount)
	require.Equal(t, int64(1), rows[0].SuccessCount)
	require.Equal(t, 120.0, rows[0].AvgPrice)

	require.Equal(t, "2026-06-02", rows[1].Date)
	require.Equal(t, int64(1), rows[1].AuctionCount)
	require.Equal(t, int64(0), rows[1].BidCount)
	require.Equal(t, int64(0), rows[1].SuccessCount)
	require.Equal(t, 0.0, rows[1].AvgPrice)
}

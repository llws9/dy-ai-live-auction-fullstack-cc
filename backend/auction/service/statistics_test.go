package service

import (
	"testing"
	"time"

	"auction-service/dao"

	"github.com/stretchr/testify/require"
)

func TestAuctionStatisticsBuildSeries(t *testing.T) {
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)
	rows := []dao.AuctionDailyStatsRow{
		{Date: "2026-06-01", AuctionCount: 2, BidCount: 3, SuccessCount: 1, AvgPrice: 120},
		{Date: "2026-06-03", AuctionCount: 1, BidCount: 0, SuccessCount: 0, AvgPrice: 0},
	}

	stats := buildAuctionDailySeries(start, end, rows)

	require.Len(t, stats, 3)
	require.Equal(t, "2026-06-01", stats[0].Date)
	require.Equal(t, int64(2), stats[0].AuctionCount)
	require.Equal(t, int64(3), stats[0].BidCount)
	require.Equal(t, 50.0, stats[0].SuccessRate)
	require.Equal(t, 120.0, stats[0].AvgPrice)
	require.Equal(t, "2026-06-02", stats[1].Date)
	require.Equal(t, int64(0), stats[1].AuctionCount)
	require.Equal(t, 0.0, stats[1].SuccessRate)
	require.Equal(t, "2026-06-03", stats[2].Date)
}

func TestValidateAuctionStatisticsRange(t *testing.T) {
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	require.NoError(t, validateAuctionStatisticsRange(start, end))

	require.Error(t, validateAuctionStatisticsRange(end, start))
	require.Error(t, validateAuctionStatisticsRange(start, start.AddDate(0, 0, 91)))
}

package service

import (
	"context"
	"errors"
	"math"
	"time"

	"auction-service/dao"
)

var ErrInvalidStatisticsRange = errors.New("invalid statistics date range")

type AuctionDailyStat struct {
	Date         string  `json:"date"`
	AuctionCount int64   `json:"auction_count"`
	BidCount     int64   `json:"bid_count"`
	AvgPrice     float64 `json:"avg_price"`
	SuccessRate  float64 `json:"success_rate"`
}

type StatisticsService struct {
	statisticsDAO *dao.StatisticsDAO
}

func NewStatisticsService(statisticsDAO *dao.StatisticsDAO) *StatisticsService {
	return &StatisticsService{statisticsDAO: statisticsDAO}
}

func (s *StatisticsService) GetAuctionDailyStats(ctx context.Context, startDate, endDate time.Time, creatorID *int64) ([]AuctionDailyStat, error) {
	if err := validateAuctionStatisticsRange(startDate, endDate); err != nil {
		return nil, err
	}
	endExclusive := endDate.AddDate(0, 0, 1)
	rows, err := s.statisticsDAO.ListAuctionDailyStats(ctx, startDate, endExclusive, creatorID)
	if err != nil {
		return nil, err
	}
	return buildAuctionDailySeries(startDate, endDate, rows), nil
}

func validateAuctionStatisticsRange(startDate, endDate time.Time) error {
	if startDate.After(endDate) {
		return ErrInvalidStatisticsRange
	}
	if int(endDate.Sub(startDate).Hours()/24) > 90 {
		return ErrInvalidStatisticsRange
	}
	return nil
}

func buildAuctionDailySeries(startDate, endDate time.Time, rows []dao.AuctionDailyStatsRow) []AuctionDailyStat {
	byDate := make(map[string]dao.AuctionDailyStatsRow, len(rows))
	for _, row := range rows {
		byDate[row.Date] = row
	}

	stats := make([]AuctionDailyStat, 0, int(endDate.Sub(startDate).Hours()/24)+1)
	for day := startDate; !day.After(endDate); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		row := byDate[date]
		successRate := 0.0
		if row.AuctionCount > 0 {
			successRate = round1(float64(row.SuccessCount) / float64(row.AuctionCount) * 100)
		}
		stats = append(stats, AuctionDailyStat{
			Date:         date,
			AuctionCount: row.AuctionCount,
			BidCount:     row.BidCount,
			AvgPrice:     row.AvgPrice,
			SuccessRate:  successRate,
		})
	}
	return stats
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

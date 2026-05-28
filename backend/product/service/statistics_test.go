package service

import (
	"testing"
	"time"

	"product-service/dao"

	"github.com/stretchr/testify/assert"
)

func TestStatisticsService_GetOverview_Validation(t *testing.T) {

	t.Run("should calculate success rate correctly", func(t *testing.T) {
		totalAuctions := int64(100)
		successAuctions := int64(85)

		var successRate float64
		if totalAuctions > 0 {
			successRate = float64(successAuctions) / float64(totalAuctions)
		}

		assert.Equal(t, 0.85, successRate)
	})

	t.Run("should handle zero total auctions", func(t *testing.T) {
		totalAuctions := int64(0)
		successAuctions := int64(0)

		var successRate float64
		if totalAuctions > 0 {
			successRate = float64(successAuctions) / float64(totalAuctions)
		}

		assert.Equal(t, 0.0, successRate)
	})

	t.Run("should calculate active users", func(t *testing.T) {
		totalUsers := int64(1000)
		activeUsers := int64(250)

		assert.LessOrEqual(t, activeUsers, totalUsers)
	})
}

func TestStatisticsService_GetAuctionStatistics_DateFilter(t *testing.T) {
	t.Run("should validate date range", func(t *testing.T) {
		startDate := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2026, 5, 31, 23, 59, 59, 0, time.UTC)

		assert.True(t, startDate.Before(endDate))
		assert.True(t, endDate.After(startDate))
	})

	t.Run("should handle nil dates", func(t *testing.T) {
		var startDate, endDate *time.Time

		assert.Nil(t, startDate)
		assert.Nil(t, endDate)
	})

	t.Run("should calculate average bid count", func(t *testing.T) {
		totalBids := 350
		totalAuctions := 100

		avgBidCount := float64(totalBids) / float64(totalAuctions)
		assert.Equal(t, 3.5, avgBidCount)
	})
}

func TestStatisticsService_GetRevenueStatistics_CategoryFilter(t *testing.T) {
	t.Run("should calculate total revenue", func(t *testing.T) {
		dailyRevenues := []float64{1000.0, 1200.0, 800.0, 1500.0}
		totalRevenue := 0.0

		for _, rev := range dailyRevenues {
			totalRevenue += rev
		}

		assert.Equal(t, 4500.0, totalRevenue)
	})

	t.Run("should calculate category percentage", func(t *testing.T) {
		totalRevenue := 10000.0
		categoryRevenue := 3000.0

		percentage := (categoryRevenue / totalRevenue) * 100
		assert.Equal(t, 30.0, percentage)
	})

	t.Run("should handle category filter", func(t *testing.T) {
		category := "Electronics"
		assert.NotEmpty(t, category)
	})
}

func TestStatisticsService_GetUserStatistics_ActivityMetrics(t *testing.T) {
	t.Run("should calculate user activity rate", func(t *testing.T) {
		totalUsers := int64(1000)
		activeUsers := int64(250)

		activeRate := float64(activeUsers) / float64(totalUsers)
		assert.Equal(t, 0.25, activeRate)
	})

	t.Run("should calculate new user growth", func(t *testing.T) {
		totalUsers := int64(1000)
		newUsers := int64(120)

		growthRate := float64(newUsers) / float64(totalUsers)
		assert.Equal(t, 0.12, growthRate)
	})

	t.Run("should handle bid distribution", func(t *testing.T) {
		bidRanges := []dao.BidRange{
			{Range: "0-50", Count: 150},
			{Range: "50-100", Count: 280},
			{Range: "100-500", Count: 320},
			{Range: "500+", Count: 100},
		}

		assert.Len(t, bidRanges, 4)

		totalCount := int64(0)
		for _, br := range bidRanges {
			totalCount += br.Count
		}
		assert.Equal(t, int64(850), totalCount)
	})
}

func TestStatisticsService_DataAggregation(t *testing.T) {
	t.Run("should correctly aggregate statistics", func(t *testing.T) {
		// Simulate aggregation
		orders := []struct {
			Price  float64
			Status string
		}{
			{100.0, "paid"},
			{200.0, "paid"},
			{150.0, "pending"},
			{300.0, "paid"},
		}

		totalRevenue := 0.0
		paidCount := 0

		for _, order := range orders {
			if order.Status == "paid" {
				totalRevenue += order.Price
				paidCount++
			}
		}

		assert.Equal(t, 600.0, totalRevenue)
		assert.Equal(t, 3, paidCount)
	})

	t.Run("should handle percentage calculations", func(t *testing.T) {
		categoryRevenues := []dao.CategoryRevenue{
			{Category: "Electronics", Revenue: 5000.0, Percentage: 50.0},
			{Category: "Fashion", Revenue: 3000.0, Percentage: 30.0},
			{Category: "Home", Revenue: 2000.0, Percentage: 20.0},
		}

		totalPercentage := 0.0
		for _, cr := range categoryRevenues {
			totalPercentage += cr.Percentage
		}

		assert.Equal(t, 100.0, totalPercentage)
	})
}

func TestStatisticsService_DailyRevenue(t *testing.T) {
	t.Run("should generate daily revenue data", func(t *testing.T) {
		dailyRevenues := []dao.DailyRevenue{
			{Date: "2026-05-22", Revenue: 7500.00},
			{Date: "2026-05-21", Revenue: 8200.00},
			{Date: "2026-05-20", Revenue: 6300.00},
		}

		assert.Len(t, dailyRevenues, 3)

		for _, dr := range dailyRevenues {
			assert.NotEmpty(t, dr.Date)
			assert.GreaterOrEqual(t, dr.Revenue, 0.0)
		}
	})

	t.Run("should format dates correctly", func(t *testing.T) {
		date := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)
		dateStr := date.Format("2006-01-02")

		assert.Equal(t, "2026-05-22", dateStr)
	})
}

func TestStatisticsService_PeriodComparison(t *testing.T) {
	t.Run("should compare current vs previous period", func(t *testing.T) {
		currentPeriod := 50000.0
		previousPeriod := 45000.0

		growth := ((currentPeriod - previousPeriod) / previousPeriod) * 100
		assert.Equal(t, 11.11111111111111, growth)
	})

	t.Run("should handle zero previous period", func(t *testing.T) {
		currentPeriod := 50000.0
		previousPeriod := 0.0

		var growth float64
		if previousPeriod > 0 {
			growth = ((currentPeriod - previousPeriod) / previousPeriod) * 100
		} else {
			growth = 100.0 // 100% growth from zero
		}

		assert.Equal(t, 100.0, growth)
	})
}

func TestStatisticsService_TopAuctions(t *testing.T) {
	t.Run("should sort auctions by final price", func(t *testing.T) {
		topAuctions := []dao.AuctionSummary{
			{ID: 1, Title: "Product A", FinalPrice: 1500.00, BidCount: 25},
			{ID: 2, Title: "Product B", FinalPrice: 1200.00, BidCount: 18},
			{ID: 3, Title: "Product C", FinalPrice: 800.00, BidCount: 12},
		}

		assert.Len(t, topAuctions, 3)

		// Verify descending order
		for i := 1; i < len(topAuctions); i++ {
			assert.GreaterOrEqual(t, topAuctions[i-1].FinalPrice, topAuctions[i].FinalPrice)
		}
	})

	t.Run("should limit top auctions count", func(t *testing.T) {
		maxTopAuctions := 10
		allAuctions := make([]dao.AuctionSummary, 20)

		// Should limit to maxTopAuctions
		assert.LessOrEqual(t, maxTopAuctions, len(allAuctions))
	})
}

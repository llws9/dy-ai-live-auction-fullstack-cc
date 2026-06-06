package handler

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/service"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var statisticsHandlerDBCounter int64

func setupStatisticsHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:statistics_handler_test_%d?mode=memory&cache=shared", atomic.AddInt64(&statisticsHandlerDBCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Auction{}, &model.Bid{}))
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestStatisticsHandlerGetAuctionStatistics(t *testing.T) {
	db := setupStatisticsHandlerTestDB(t)
	merchantID := int64(9)
	winnerID := int64(2001)
	start := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	require.NoError(t, db.Create(&model.Auction{
		ID: 101, CreatorID: &merchantID, ProductID: 1, Status: model.AuctionStatusEnded,
		WinnerID: &winnerID, CurrentPrice: decimal.NewFromInt(120), StartTime: start, EndTime: start.Add(time.Hour),
	}).Error)
	require.NoError(t, db.Create(&model.Bid{AuctionID: 101, UserID: 1, Amount: decimal.NewFromInt(120), CreatedAt: start}).Error)

	statisticsDAO := dao.NewStatisticsDAO(db)
	statisticsService := service.NewStatisticsService(statisticsDAO)
	statisticsHandler := NewStatisticsHandler(statisticsService)
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	h.GET("/api/v1/statistics/auctions", statisticsHandler.GetAuctionStatistics)

	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/statistics/auctions?start_date=2026-06-01&end_date=2026-06-01", nil,
		ut.Header{Key: "X-User-Role", Value: "merchant"},
		ut.Header{Key: "X-User-ID", Value: "9"})

	require.Equal(t, http.StatusOK, w.Result().StatusCode())
	require.Contains(t, string(w.Body.Bytes()), `"date":"2026-06-01"`)
	require.Contains(t, string(w.Body.Bytes()), `"auction_count":1`)
	require.Contains(t, string(w.Body.Bytes()), `"bid_count":1`)
	require.Contains(t, string(w.Body.Bytes()), `"success_rate":100`)
}

func TestStatisticsHandlerRejectsInvalidRoleAndRange(t *testing.T) {
	db := setupStatisticsHandlerTestDB(t)
	statisticsHandler := NewStatisticsHandler(service.NewStatisticsService(dao.NewStatisticsDAO(db)))
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	h.GET("/api/v1/statistics/auctions", statisticsHandler.GetAuctionStatistics)

	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/statistics/auctions", nil,
		ut.Header{Key: "X-User-Role", Value: "user"},
		ut.Header{Key: "X-User-ID", Value: "1"})
	require.Equal(t, http.StatusForbidden, w.Result().StatusCode())

	w = ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/statistics/auctions?start_date=2026-06-07&end_date=2026-06-01", nil,
		ut.Header{Key: "X-User-Role", Value: "admin"},
		ut.Header{Key: "X-User-ID", Value: "99"})
	require.Equal(t, http.StatusBadRequest, w.Result().StatusCode())

	w = ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/statistics/auctions?group_by=month", nil,
		ut.Header{Key: "X-User-Role", Value: "admin"},
		ut.Header{Key: "X-User-ID", Value: "99"})
	require.Equal(t, http.StatusBadRequest, w.Result().StatusCode())
}

func TestDefaultAuctionStatisticsRange(t *testing.T) {
	now := time.Date(2026, 6, 7, 15, 30, 0, 0, time.UTC)
	start, end := defaultAuctionStatisticsRange(now)
	require.Equal(t, "2026-06-01", start.Format("2006-01-02"))
	require.Equal(t, "2026-06-07", end.Format("2006-01-02"))
}

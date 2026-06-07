package handler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/dao"
	"product-service/model"
	"product-service/service"
)

func newStatisticsHandlerWithSeed(t *testing.T, seed func(db *gorm.DB)) *StatisticsHandler {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Order{}, &model.User{}))
	db.Exec("DELETE FROM orders")
	db.Exec("DELETE FROM users")
	if seed != nil {
		seed(db)
	}

	statsSvc := service.NewStatisticsService(dao.NewStatisticsDAO(db))
	return NewStatisticsHandler(statsSvc)
}

func TestStatisticsHandlerGetOverviewAllowsAdminHeader(t *testing.T) {
	h := newStatisticsHandlerWithSeed(t, func(db *gorm.DB) {
		email := "admin@example.com"
		require.NoError(t, db.Create(&model.User{
			ID:       1,
			Name:     "admin",
			Email:    &email,
			Password: "hashed",
			Role:     int(model.RoleAdmin),
			Status:   1,
		}).Error)
		require.NoError(t, db.Create(&model.Order{
			ID:         101,
			AuctionID:  201,
			ProductID:  301,
			WinnerID:   1,
			FinalPrice: decimal.NewFromInt(188),
			Status:     model.OrderStatusPaid,
		}).Error)
	})

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/statistics/overview")
	c.Request.Header.Set("X-User-Role", "admin")

	h.GetOverview(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.EqualValues(t, 1, body["total_auctions"])
	assert.EqualValues(t, 1, body["total_users"])
}

func TestStatisticsHandlerGetOverviewMerchantOnlyOwnSellerOrders(t *testing.T) {
	h := newStatisticsHandlerWithSeed(t, func(db *gorm.DB) {
		sellerA := int64(1001)
		sellerB := int64(1002)
		now := time.Now()
		yesterday := now.AddDate(0, 0, -1)
		require.NoError(t, db.Create(&model.Order{
			ID:         101,
			AuctionID:  201,
			ProductID:  301,
			SellerID:   &sellerA,
			WinnerID:   501,
			FinalPrice: decimal.NewFromInt(188),
			Status:     model.OrderStatusPaid,
			CreatedAt:  now,
		}).Error)
		require.NoError(t, db.Create(&model.Order{
			ID:         102,
			AuctionID:  202,
			ProductID:  302,
			SellerID:   &sellerB,
			WinnerID:   502,
			FinalPrice: decimal.NewFromInt(288),
			Status:     model.OrderStatusPaid,
			CreatedAt:  now,
		}).Error)
		require.NoError(t, db.Create(&model.Order{
			ID:         103,
			AuctionID:  203,
			ProductID:  303,
			SellerID:   &sellerA,
			WinnerID:   503,
			FinalPrice: decimal.NewFromInt(99),
			Status:     model.OrderStatusPaid,
			CreatedAt:  yesterday,
		}).Error)
	})

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/statistics/overview")
	c.Request.Header.Set("X-User-Role", "merchant")
	c.Request.Header.Set("X-User-ID", "1001")

	h.GetOverview(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.EqualValues(t, 2, body["total_auctions"])
	assert.EqualValues(t, 2, body["total_orders"])
	assert.EqualValues(t, 287, body["total_revenue"])
	assert.EqualValues(t, 188, body["today_revenue"])
}

func TestStatisticsHandlerGetUserStatisticsMerchantRejected(t *testing.T) {
	h := newStatisticsHandlerWithSeed(t, nil)

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/statistics/users")
	c.Request.Header.Set("X-User-Role", "merchant")
	c.Request.Header.Set("X-User-ID", "1001")

	h.GetUserStatistics(context.Background(), c)

	assert.Equal(t, 403, c.Response.StatusCode())
}

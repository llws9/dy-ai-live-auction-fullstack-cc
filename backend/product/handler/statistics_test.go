package handler

import (
	"context"
	"encoding/json"
	"testing"

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

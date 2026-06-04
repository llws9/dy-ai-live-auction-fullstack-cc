package dao

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/model"
)

func setupStatisticsDAOTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Order{}, &model.User{}))
	require.NoError(t, db.Exec("DELETE FROM orders").Error)
	require.NoError(t, db.Exec("DELETE FROM users").Error)
	return db
}

func TestStatisticsDAOGetOverviewScopedMerchantOnlyOwnSellerOrders(t *testing.T) {
	db := setupStatisticsDAOTestDB(t)
	statsDAO := NewStatisticsDAO(db)
	ctx := context.Background()
	sellerA := int64(1001)
	sellerB := int64(1002)

	require.NoError(t, db.Create(&model.Order{
		ID:         1,
		AuctionID:  11,
		ProductID:  21,
		SellerID:   &sellerA,
		WinnerID:   501,
		FinalPrice: decimal.NewFromInt(100),
		Status:     model.OrderStatusPaid,
	}).Error)
	require.NoError(t, db.Create(&model.Order{
		ID:         2,
		AuctionID:  12,
		ProductID:  22,
		SellerID:   &sellerB,
		WinnerID:   502,
		FinalPrice: decimal.NewFromInt(200),
		Status:     model.OrderStatusPaid,
	}).Error)

	overview, err := statsDAO.GetOverviewScoped(ctx, &sellerA)

	require.NoError(t, err)
	require.Equal(t, int64(1), overview.TotalAuctions)
	require.Equal(t, float64(100), overview.TotalRevenue)
	require.Equal(t, int64(1), overview.ActiveUsers)
}

func TestStatisticsDAOGetRevenueStatisticsScopedMerchantOnlyOwnSellerOrders(t *testing.T) {
	db := setupStatisticsDAOTestDB(t)
	statsDAO := NewStatisticsDAO(db)
	ctx := context.Background()
	sellerA := int64(1001)
	sellerB := int64(1002)
	now := time.Now()

	require.NoError(t, db.Create(&model.Order{
		ID:         1,
		AuctionID:  11,
		ProductID:  21,
		SellerID:   &sellerA,
		WinnerID:   501,
		FinalPrice: decimal.NewFromInt(100),
		Status:     model.OrderStatusPaid,
		CreatedAt:  now,
	}).Error)
	require.NoError(t, db.Create(&model.Order{
		ID:         2,
		AuctionID:  12,
		ProductID:  22,
		SellerID:   &sellerB,
		WinnerID:   502,
		FinalPrice: decimal.NewFromInt(200),
		Status:     model.OrderStatusPaid,
		CreatedAt:  now,
	}).Error)

	stats, err := statsDAO.GetRevenueStatisticsScoped(ctx, nil, nil, "", &sellerA)

	require.NoError(t, err)
	require.Equal(t, float64(100), stats.TotalRevenue)
}

package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"auction-service/config"
	"auction-service/dao"
	"auction-service/model"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupRuleGuardDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Auction{},
		&model.AuctionRule{},
		&model.Bid{},
		&model.SkyLampSubscription{},
	))

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db
}

func seedRuleGuardAuction(t *testing.T, db *gorm.DB) {
	t.Helper()

	require.NoError(t, db.Create(&model.User{
		ID:       1001,
		Name:     "buyer",
		Password: "password",
		Status:   1,
	}).Error)
	require.NoError(t, db.Create(&model.Auction{
		ID:           993305,
		ProductID:    993205,
		Status:       model.AuctionStatusOngoing,
		CurrentPrice: decimal.NewFromInt(1888),
		StartTime:    time.Now().Add(-time.Hour),
		EndTime:      time.Now().Add(time.Hour),
	}).Error)
}

func TestSkyLampStartSubscription_MissingRuleReturnsBusinessError(t *testing.T) {
	db := setupRuleGuardDB(t)
	seedRuleGuardAuction(t, db)

	bidSvc := NewBidService(
		dao.NewAuctionDAO(db),
		dao.NewBidDAO(db),
		dao.NewAuctionRuleDAO(db),
		dao.NewUserDAO(db),
	)
	skyLampSvc := NewSkyLampService(
		dao.NewSkyLampDAO(db),
		bidSvc,
		config.DefaultSkyLampConfig(),
		nil,
	)

	subscription, err := skyLampSvc.StartSubscription(context.Background(), 1001, 993305)

	require.Nil(t, subscription)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "竞拍规则不存在"), "unexpected error: %v", err)
}

func TestBidServicePlaceBid_MissingRuleReturnsBusinessError(t *testing.T) {
	db := setupRuleGuardDB(t)
	seedRuleGuardAuction(t, db)

	bidSvc := NewBidService(
		dao.NewAuctionDAO(db),
		dao.NewBidDAO(db),
		dao.NewAuctionRuleDAO(db),
		dao.NewUserDAO(db),
	)

	result, err := bidSvc.PlaceBid(context.Background(), &PlaceBidRequest{
		AuctionID: 993305,
		UserID:    1001,
		Amount:    decimal.NewFromInt(1988),
	})

	require.Nil(t, result)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "竞拍规则不存在"), "unexpected error: %v", err)
}

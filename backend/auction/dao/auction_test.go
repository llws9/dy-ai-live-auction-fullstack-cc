package dao

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"auction-service/model"
)

func newAuctionDAOTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))
	return db
}

func TestAuctionDAO_GetActiveAndLatestTerminalByProductID(t *testing.T) {
	db := newAuctionDAOTestDB(t)
	d := NewAuctionDAO(db)
	now := time.Now()
	winnerID := int64(2001)
	rows := []model.Auction{
		{ID: 1, ProductID: 11, Status: model.AuctionStatusEnded, WinnerID: nil, StartTime: now.Add(-4 * time.Hour), EndTime: now.Add(-3 * time.Hour)},
		{ID: 2, ProductID: 11, Status: model.AuctionStatusEnded, WinnerID: &winnerID, StartTime: now.Add(-2 * time.Hour), EndTime: now.Add(-time.Hour)},
		{ID: 3, ProductID: 11, Status: model.AuctionStatusPending, StartTime: now, EndTime: now.Add(time.Hour)},
	}
	require.NoError(t, db.Create(&rows).Error)

	active, err := d.GetActiveByProductID(context.Background(), 11)
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, int64(3), active.ID)

	terminal, err := d.GetLatestTerminalByProductID(context.Background(), 11)
	require.NoError(t, err)
	require.NotNil(t, terminal)
	assert.Equal(t, int64(2), terminal.ID)
	assert.NotNil(t, terminal.WinnerID)
}

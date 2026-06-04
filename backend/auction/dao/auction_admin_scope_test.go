package dao

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"auction-service/model"
)

func setupAuctionAdminScopeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))
	return db
}

func TestAuctionDAOListAdminScopedMerchantOnlyOwnAuctions(t *testing.T) {
	db := setupAuctionAdminScopeDB(t)
	dao := NewAuctionDAO(db)
	ctx := context.Background()
	now := time.Now()
	ownerA := int64(1001)
	ownerB := int64(1002)
	require.NoError(t, dao.Create(ctx, &model.Auction{ProductID: 1, CreatorID: &ownerA, StartTime: now, EndTime: now.Add(time.Hour)}))
	require.NoError(t, dao.Create(ctx, &model.Auction{ProductID: 2, CreatorID: &ownerB, StartTime: now, EndTime: now.Add(time.Hour)}))

	items, total, err := dao.ListAdminScoped(ctx, nil, 1, 20, &ownerA)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, int64(1), items[0].ProductID)
}

func TestAuctionDAOGetByIDAndCreatorIDRejectsOtherOwner(t *testing.T) {
	db := setupAuctionAdminScopeDB(t)
	dao := NewAuctionDAO(db)
	ctx := context.Background()
	now := time.Now()
	owner := int64(1001)
	other := int64(1002)
	item := &model.Auction{ProductID: 1, CreatorID: &owner, StartTime: now, EndTime: now.Add(time.Hour)}
	require.NoError(t, dao.Create(ctx, item))

	got, err := dao.GetByIDAndCreatorID(ctx, item.ID, other)

	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	require.Nil(t, got)
}

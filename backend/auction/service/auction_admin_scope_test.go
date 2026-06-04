package service

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"auction-service/dao"
	"auction-service/model"
)

func setupAuctionAdminScopeService(t *testing.T) *AuctionService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))
	return NewAuctionService(dao.NewAuctionDAO(db))
}

func TestAuctionServiceCreateAuctionStoresCreatorID(t *testing.T) {
	svc := setupAuctionAdminScopeService(t)
	creatorID := int64(1001)

	auction, err := svc.CreateAuction(context.Background(), &CreateAuctionRequest{
		ProductID: 1,
		CreatorID: &creatorID,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
	})

	require.NoError(t, err)
	require.NotNil(t, auction.CreatorID)
	require.Equal(t, creatorID, *auction.CreatorID)
}

func TestAuctionServiceListAdminScopedMerchantOnlyOwnAuctions(t *testing.T) {
	svc := setupAuctionAdminScopeService(t)
	ctx := context.Background()
	ownerA := int64(1001)
	ownerB := int64(1002)
	_, err := svc.CreateAuction(ctx, &CreateAuctionRequest{ProductID: 1, CreatorID: &ownerA, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)})
	require.NoError(t, err)
	_, err = svc.CreateAuction(ctx, &CreateAuctionRequest{ProductID: 2, CreatorID: &ownerB, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)})
	require.NoError(t, err)

	items, total, err := svc.ListAdminAuctions(ctx, nil, 1, 20, &ownerA)

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, int64(1), items[0].ProductID)
}

func TestAuctionServiceCancelAuctionByCreatorRejectsOtherOwner(t *testing.T) {
	svc := setupAuctionAdminScopeService(t)
	ctx := context.Background()
	ownerA := int64(1001)
	ownerB := int64(1002)
	auction, err := svc.CreateAuction(ctx, &CreateAuctionRequest{
		ProductID: 1,
		CreatorID: &ownerA,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
	})
	require.NoError(t, err)

	err = svc.CancelAuctionByCreator(ctx, auction.ID, ownerB)

	require.Error(t, err)
	reloaded, err := svc.GetAuction(ctx, auction.ID)
	require.NoError(t, err)
	require.Equal(t, model.AuctionStatusPending, reloaded.Status)
}

package handler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/service"
)

func setupAuctionAdminScopeHandler(t *testing.T) *AuctionHandler {
	t.Helper()
	h, _ := setupAuctionAdminScopeHandlerWithDB(t)
	return h
}

func setupAuctionAdminScopeHandlerWithDB(t *testing.T) (*AuctionHandler, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))
	svc := service.NewAuctionService(dao.NewAuctionDAO(db))
	return NewAuctionHandler(svc), db
}

func createAdminScopeAuction(t *testing.T, ctx context.Context, h *AuctionHandler, productID, ownerID int64) *model.Auction {
	t.Helper()
	auction, err := h.auctionService.CreateAuction(ctx, &service.CreateAuctionRequest{
		ProductID:      productID,
		CreatorID:      &ownerID,
		Duration:       3600,
		ProductOwnerID: ownerID,
		ProductStatus:  1,
		RuleBound:      true,
		LiveStreamID:   ownerID,
	})
	require.NoError(t, err)
	return auction
}

func TestAuctionHandlerAdminListMerchantOnlyOwnAuctions(t *testing.T) {
	h := setupAuctionAdminScopeHandler(t)
	ctx := context.Background()
	ownerA := int64(1001)
	ownerB := int64(1002)
	createAdminScopeAuction(t, ctx, h, 1, ownerA)
	createAdminScopeAuction(t, ctx, h, 2, ownerB)

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/admin/auctions")
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "merchant")

	h.AdminList(ctx, c)

	require.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	data := body["data"].(map[string]interface{})
	list := data["list"].([]interface{})
	require.Len(t, list, 1)
	item := list[0].(map[string]interface{})
	require.EqualValues(t, 1, item["product_id"])
}

func TestAuctionHandlerAdminGetMerchantRejectsOtherOwner(t *testing.T) {
	h := setupAuctionAdminScopeHandler(t)
	ctx := context.Background()
	owner := int64(1001)
	auction := createAdminScopeAuction(t, ctx, h, 1, owner)

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/admin/auctions/1")
	c.Request.Header.Set("X-User-ID", "1002")
	c.Request.Header.Set("X-User-Role", "merchant")
	c.Params = append(c.Params, param.Param{Key: "id", Value: "1"})

	h.AdminGet(ctx, c)

	require.Equal(t, 404, c.Response.StatusCode(), "auction id %d must not be visible to other merchant", auction.ID)
}

func TestAuctionHandlerCancelMerchantRejectsOtherOwner(t *testing.T) {
	h := setupAuctionAdminScopeHandler(t)
	ctx := context.Background()
	owner := int64(1001)
	auction := createAdminScopeAuction(t, ctx, h, 1, owner)

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/auctions/1/cancel")
	c.Request.Header.Set("X-User-ID", "1002")
	c.Request.Header.Set("X-User-Role", "merchant")
	c.Params = append(c.Params, param.Param{Key: "id", Value: "1"})

	h.Cancel(ctx, c)

	require.Equal(t, 404, c.Response.StatusCode(), "auction id %d must not be cancellable by other merchant", auction.ID)
	reloaded, err := h.auctionService.GetAuction(ctx, auction.ID)
	require.NoError(t, err)
	require.Equal(t, model.AuctionStatusPending, reloaded.Status)
}

func TestAuctionHandlerCancelEndedAuctionReturnsConflict(t *testing.T) {
	h, db := setupAuctionAdminScopeHandlerWithDB(t)
	ctx := context.Background()
	owner := int64(1001)
	auction := createAdminScopeAuction(t, ctx, h, 1, owner)
	require.NoError(t, db.Model(&model.Auction{}).Where("id = ?", auction.ID).Update("status", model.AuctionStatusEnded).Error)

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/auctions/1/cancel")
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "merchant")
	c.Params = append(c.Params, param.Param{Key: "id", Value: "1"})

	h.Cancel(ctx, c)

	require.Equal(t, 409, c.Response.StatusCode(), "auction id %d exists but cannot be cancelled in ended state", auction.ID)
}

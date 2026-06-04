package handler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))
	svc := service.NewAuctionService(dao.NewAuctionDAO(db))
	return NewAuctionHandler(svc)
}

func TestAuctionHandlerAdminListMerchantOnlyOwnAuctions(t *testing.T) {
	h := setupAuctionAdminScopeHandler(t)
	ctx := context.Background()
	ownerA := int64(1001)
	ownerB := int64(1002)
	_, err := h.auctionService.CreateAuction(ctx, &service.CreateAuctionRequest{ProductID: 1, CreatorID: &ownerA, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)})
	require.NoError(t, err)
	_, err = h.auctionService.CreateAuction(ctx, &service.CreateAuctionRequest{ProductID: 2, CreatorID: &ownerB, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)})
	require.NoError(t, err)

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
	auction, err := h.auctionService.CreateAuction(ctx, &service.CreateAuctionRequest{ProductID: 1, CreatorID: &owner, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)})
	require.NoError(t, err)

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/admin/auctions/1")
	c.Request.Header.Set("X-User-ID", "1002")
	c.Request.Header.Set("X-User-Role", "merchant")
	c.Params = append(c.Params, param.Param{Key: "id", Value: "1"})

	h.AdminGet(ctx, c)

	require.Equal(t, 404, c.Response.StatusCode(), "auction id %d must not be visible to other merchant", auction.ID)
}

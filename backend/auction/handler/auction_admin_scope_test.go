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

	"auction-service/client"
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

func TestAuctionHandlerCreateAcceptsStartTimeAndLiveStreamID(t *testing.T) {
	h, db := setupAuctionAdminScopeHandlerWithDB(t)
	h.SetLiveStreamClient(fakeAuctionLiveStreamClient{
		streams: map[int64]client.LiveStreamSummary{
			880301: {ID: 880301, CreatorID: 1001},
		},
	})
	ctx := context.Background()
	startTime := time.Now().Add(time.Minute).Truncate(time.Second)

	c := app.NewContext(0)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.SetBodyString(`{"product_id":42,"duration":180,"start_time":"` + startTime.Format(time.RFC3339) + `","live_stream_id":880301}`)

	h.Create(ctx, c)

	require.Equal(t, 201, c.Response.StatusCode())
	var auction model.Auction
	require.NoError(t, db.First(&auction).Error)
	require.NotNil(t, auction.LiveStreamID)
	require.Equal(t, int64(880301), *auction.LiveStreamID)
	require.WithinDuration(t, startTime, auction.StartTime, time.Second)
	require.WithinDuration(t, startTime.Add(180*time.Second), auction.EndTime, time.Second)
}

func TestAuctionHandlerCreateRejectsLiveStreamOwnedByOtherMerchant(t *testing.T) {
	h, _ := setupAuctionAdminScopeHandlerWithDB(t)
	h.SetLiveStreamClient(fakeAuctionLiveStreamClient{
		streams: map[int64]client.LiveStreamSummary{
			880301: {ID: 880301, CreatorID: 2002},
		},
	})
	ctx := context.Background()

	c := app.NewContext(0)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.SetBodyString(`{"product_id":42,"duration":180,"live_stream_id":880301}`)

	h.Create(ctx, c)

	require.Equal(t, 403, c.Response.StatusCode())
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

func TestAuctionHandlerCancelMerchantRejectsOtherOwner(t *testing.T) {
	h := setupAuctionAdminScopeHandler(t)
	ctx := context.Background()
	owner := int64(1001)
	auction, err := h.auctionService.CreateAuction(ctx, &service.CreateAuctionRequest{ProductID: 1, CreatorID: &owner, StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)})
	require.NoError(t, err)

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

type fakeAuctionLiveStreamClient struct {
	streams map[int64]client.LiveStreamSummary
	err     error
}

func (f fakeAuctionLiveStreamClient) BatchGetLiveStreams(_ context.Context, ids []int64) (map[int64]client.LiveStreamSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[int64]client.LiveStreamSummary)
	for _, id := range ids {
		if stream, ok := f.streams[id]; ok {
			out[id] = stream
		}
	}
	return out, nil
}

func TestAuctionHandlerCancelEndedAuctionReturnsConflict(t *testing.T) {
	h, db := setupAuctionAdminScopeHandlerWithDB(t)
	ctx := context.Background()
	owner := int64(1001)
	auction, err := h.auctionService.CreateAuction(ctx, &service.CreateAuctionRequest{ProductID: 1, CreatorID: &owner, StartTime: time.Now().Add(-2 * time.Hour), EndTime: time.Now().Add(-time.Hour)})
	require.NoError(t, err)
	require.NoError(t, db.Model(&model.Auction{}).Where("id = ?", auction.ID).Update("status", model.AuctionStatusEnded).Error)

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/auctions/1/cancel")
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "merchant")
	c.Params = append(c.Params, param.Param{Key: "id", Value: "1"})

	h.Cancel(ctx, c)

	require.Equal(t, 409, c.Response.StatusCode(), "auction id %d exists but cannot be cancelled in ended state", auction.ID)
}

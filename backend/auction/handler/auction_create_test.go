package handler

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"auction-service/client"
	"auction-service/dao"
	"auction-service/model"
	"auction-service/service"
)

func newAuctionHandlerCreateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))
	return db
}

func TestAuctionHandler_CreateRequiresMerchantAndWritesLiveStream(t *testing.T) {
	db := newAuctionHandlerCreateTestDB(t)
	auctionDAO := dao.NewAuctionDAO(db)
	svc := service.NewAuctionService(auctionDAO)
	h := NewAuctionHandler(svc)
	pc := &fakeCreateProductClient{
		info: &client.AuctionProductInfo{ID: 11, OwnerID: 1001, Status: 1, RuleBound: true},
		live: &client.LiveStreamInfo{ID: 77, CreatorID: 1001, Status: 1},
	}
	h.SetProductClient(pc)

	app := server.Default(server.WithHostPorts("127.0.0.1:0"))
	app.POST("/api/v1/auctions", h.Create)
	body := []byte(`{"product_id":11,"duration":3600}`)
	resp := ut.PerformRequest(app.Engine, http.MethodPost, "/api/v1/auctions", &ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "X-User-ID", Value: "1001"},
		ut.Header{Key: "X-User-Role", Value: "merchant"})

	require.Equal(t, http.StatusCreated, resp.Result().StatusCode())
	assert.Equal(t, int64(11), pc.gotProductID)
	assert.Equal(t, int64(1001), pc.gotCreatorID)
	assert.Equal(t, "merchant_1001", pc.gotCreatorName)

	var auction model.Auction
	require.NoError(t, db.First(&auction, "product_id = ?", 11).Error)
	require.NotNil(t, auction.LiveStreamID)
	assert.Equal(t, int64(77), *auction.LiveStreamID)
}

func TestAuctionHandler_CreateRejectsUserRole(t *testing.T) {
	db := newAuctionHandlerCreateTestDB(t)
	h := NewAuctionHandler(service.NewAuctionService(dao.NewAuctionDAO(db)))
	app := server.Default(server.WithHostPorts("127.0.0.1:0"))
	app.POST("/api/v1/auctions", h.Create)
	body := []byte(`{"product_id":11,"duration":3600}`)
	resp := ut.PerformRequest(app.Engine, http.MethodPost, "/api/v1/auctions", &ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "X-User-ID", Value: "1001"},
		ut.Header{Key: "X-User-Role", Value: "user"})

	assert.Equal(t, http.StatusForbidden, resp.Result().StatusCode())
}

type fakeCreateProductClient struct {
	info *client.AuctionProductInfo
	live *client.LiveStreamInfo
	err  error

	gotProductID   int64
	gotCreatorID   int64
	gotCreatorName string
}

func (f *fakeCreateProductClient) ListProductIDsByCategory(context.Context, int64) ([]int64, error) {
	return nil, nil
}

func (f *fakeCreateProductClient) BatchGetSummaries(context.Context, []int64) (map[int64]client.ProductSummary, error) {
	return map[int64]client.ProductSummary{}, nil
}

func (f *fakeCreateProductClient) GetAuctionProductInfo(_ context.Context, productID int64) (*client.AuctionProductInfo, error) {
	f.gotProductID = productID
	return f.info, f.err
}

func (f *fakeCreateProductClient) GetOrCreateActiveLiveStream(_ context.Context, creatorID int64, creatorName string) (*client.LiveStreamInfo, error) {
	f.gotCreatorID = creatorID
	f.gotCreatorName = creatorName
	return f.live, f.err
}

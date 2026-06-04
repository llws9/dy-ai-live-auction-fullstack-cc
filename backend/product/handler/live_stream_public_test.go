package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/client"
	"product-service/dao"
	"product-service/model"
	"product-service/service"
)

func TestListPublic_OnlyLiveAndCurrentAuction(t *testing.T) {
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/auctions/current-by-live-streams", r.URL.Path)
		// 只有直播中的 501 应被查询当前竞拍
		_, _ = w.Write([]byte(`{"code":200,"data":{"items":[{"live_stream_id":501,"auction_id":11,"product_id":8,"current_price":"1200.00","status":1}]}}`))
	}))
	t.Cleanup(auctionMock.Close)

	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.Category{}, &model.AuctionRule{}, &model.LiveStream{}))
	db.Exec("DELETE FROM live_streams")
	require.NoError(t, db.Create(&model.LiveStream{
		ID:             501,
		CreatorID:      9001,
		Name:           "直播中A",
		CoverImage:     "https://cdn/a.png",
		Status:         model.LiveStreamStatusLive,
		StreamerName:   "主播A",
		StreamerAvatar: "https://cdn/avatar-a.png",
	}).Error)
	require.NoError(t, db.Create(&model.LiveStream{
		ID:        502,
		CreatorID: 9002,
		Name:      "直播中B-无竞拍",
		Status:    model.LiveStreamStatusLive,
	}).Error)
	require.NoError(t, db.Create(&model.LiveStream{
		ID:        503,
		CreatorID: 9003,
		Name:      "已结束",
		Status:    model.LiveStreamStatusEnded,
	}).Error)

	viewers := service.StaticLiveViewerCounter{501: 42}
	svc := service.NewLiveStreamServiceWithMetrics(dao.NewLiveStreamDAO(db), viewers)
	h := NewLiveStreamHandler(svc)
	h.SetAuctionClient(client.NewAuctionClient(auctionMock.URL, 0))

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/live-streams")
	h.ListPublic(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	data := body["data"].(map[string]interface{})

	list := data["list"].([]interface{})
	require.Len(t, list, 2) // 仅 status=1 的两个

	byID := map[int64]map[string]interface{}{}
	for _, raw := range list {
		item := raw.(map[string]interface{})
		id := int64(item["id"].(float64))
		byID[id] = item
		assert.EqualValues(t, 1, item["status"])
	}

	a := byID[501]
	require.NotNil(t, a)
	assert.Equal(t, "主播A", a["host_name"])
	assert.Equal(t, "https://cdn/avatar-a.png", a["host_avatar"])
	assert.EqualValues(t, 42, a["viewer_count"])
	assert.EqualValues(t, 11, a["current_auction_id"])
	assert.EqualValues(t, 8, a["current_product_id"])
	assert.Equal(t, "1200.00", a["current_price"])

	b := byID[502]
	require.NotNil(t, b)
	require.Contains(t, b, "current_auction_id")
	assert.Nil(t, b["current_auction_id"])
	assert.Nil(t, b["current_product_id"])
	assert.Nil(t, b["current_price"])

	assert.EqualValues(t, 2, data["total"])
	assert.EqualValues(t, 1, data["page"])
	assert.EqualValues(t, 20, data["page_size"])
}

func TestListPublic_ClampsPageSize(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.Category{}, &model.AuctionRule{}, &model.LiveStream{}))
	db.Exec("DELETE FROM live_streams")

	svc := service.NewLiveStreamService(dao.NewLiveStreamDAO(db))
	h := NewLiveStreamHandler(svc)

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/live-streams?page_size=1000")
	h.ListPublic(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	data := body["data"].(map[string]interface{})
	assert.EqualValues(t, 50, data["page_size"])
}

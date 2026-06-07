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
		switch r.URL.Path {
		case "/internal/auctions/current-by-live-streams":
			// 直播中的 501 有当前竞拍
			_, _ = w.Write([]byte(`{"code":200,"data":{"items":[{"live_stream_id":501,"auction_id":11,"product_id":8,"current_price":"1200.00","status":1}]}}`))
		case "/internal/auctions/next-by-live-streams":
			// 502 无当前竞拍，但有下一场，避免被空壳过滤丢弃
			_, _ = w.Write([]byte(`{"code":200,"data":{"items":[{"live_stream_id":502,"auction_id":22,"product_id":8,"start_price":"100.00","start_time":"2026-06-08T12:00:00Z"}]}}`))
		case "/internal/auctions/recent-deals-by-live-streams":
			_, _ = w.Write([]byte(`{"code":200,"data":{"items":[]}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(auctionMock.Close)

	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.Category{}, &model.AuctionRule{}, &model.LiveStream{}))
	db.Exec("DELETE FROM live_streams")
	db.Exec("DELETE FROM products")
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
		Name:      "直播中B-仅下一场",
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
	h.SetProductNameResolver(dao.NewProductDAO(db))

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/live-streams")
	h.ListPublic(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	data := body["data"].(map[string]interface{})

	list := data["list"].([]interface{})
	require.Len(t, list, 2) // 501(有current) + 502(有next)；503 已结束非候选，无任何场次的空壳被丢弃

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
	// 502 凭借 next_auction 存活
	bNext := b["next_auction"].(map[string]interface{})
	assert.EqualValues(t, 22, bNext["auction_id"])

	assert.EqualValues(t, 2, data["total"])
	assert.EqualValues(t, 1, data["page"])
	assert.EqualValues(t, 20, data["page_size"])
}

func TestListPublic_BackfillsNextAndRecentDeals(t *testing.T) {
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/auctions/current-by-live-streams":
			_, _ = w.Write([]byte(`{"code":200,"data":{"items":[{"live_stream_id":601,"auction_id":11,"product_id":8,"current_price":"1200.00","status":1}]}}`))
		case "/internal/auctions/next-by-live-streams":
			_, _ = w.Write([]byte(`{"code":200,"data":{"items":[{"live_stream_id":602,"auction_id":21,"product_id":9,"start_price":"300.00","start_time":"2026-06-08T10:00:00Z"}]}}`))
		case "/internal/auctions/recent-deals-by-live-streams":
			_, _ = w.Write([]byte(`{"code":200,"data":{"items":[{"live_stream_id":601,"deals":[{"auction_id":31,"product_id":7,"final_price":"500.00","end_time":"2026-06-08T09:00:00Z"}]}]}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(auctionMock.Close)

	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.Category{}, &model.AuctionRule{}, &model.LiveStream{}))
	db.Exec("DELETE FROM live_streams")
	db.Exec("DELETE FROM products")
	require.NoError(t, db.Create(&model.LiveStream{ID: 601, CreatorID: 1, Name: "A", Status: model.LiveStreamStatusLive}).Error)
	require.NoError(t, db.Create(&model.LiveStream{ID: 602, CreatorID: 2, Name: "B", Status: model.LiveStreamStatusNotStarted}).Error)
	require.NoError(t, db.Create(&model.LiveStream{ID: 603, CreatorID: 3, Name: "空壳", Status: model.LiveStreamStatusLive}).Error)
	require.NoError(t, db.Create(&model.Product{ID: 9, Name: "翡翠手镯", Status: model.ProductStatusPublished}).Error)
	require.NoError(t, db.Create(&model.Product{ID: 7, Name: "和田玉牌", Status: model.ProductStatusPublished}).Error)

	svc := service.NewLiveStreamService(dao.NewLiveStreamDAO(db))
	h := NewLiveStreamHandler(svc)
	h.SetAuctionClient(client.NewAuctionClient(auctionMock.URL, 0))
	h.SetProductNameResolver(dao.NewProductDAO(db))

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/live-streams")
	h.ListPublic(context.Background(), c)

	require.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	data := body["data"].(map[string]interface{})
	list := data["list"].([]interface{})
	require.Len(t, list, 2)

	byID := map[int64]map[string]interface{}{}
	for _, raw := range list {
		it := raw.(map[string]interface{})
		byID[int64(it["id"].(float64))] = it
	}

	next := byID[602]["next_auction"].(map[string]interface{})
	assert.EqualValues(t, 21, next["auction_id"])
	assert.EqualValues(t, 9, next["product_id"])
	assert.Equal(t, "翡翠手镯", next["product_name"])
	assert.Equal(t, "300.00", next["start_price"])

	deals := byID[601]["recent_deals"].([]interface{})
	require.Len(t, deals, 1)
	d0 := deals[0].(map[string]interface{})
	assert.Equal(t, "和田玉牌", d0["product_name"])
	assert.Equal(t, "500.00", d0["final_price"])
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

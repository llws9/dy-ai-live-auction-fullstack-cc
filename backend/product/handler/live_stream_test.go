package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/client"
	"product-service/dao"
	"product-service/model"
	"product-service/service"
)

// newLiveStreamHandlerWithSeed 与 internal_test.go 用同一套 sqlite in-memory，
// 但聚焦 live_streams 表的 detail 测试。
func newLiveStreamHandlerWithSeed(t *testing.T, seed func(db *gorm.DB)) *LiveStreamHandler {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.AuctionRule{}, &model.LiveStream{}))
	// Clean slate（共享 :memory:）
	db.Exec("DELETE FROM live_streams")
	if seed != nil {
		seed(db)
	}
	svc := service.NewLiveStreamService(dao.NewLiveStreamDAO(db))
	return NewLiveStreamHandler(svc)
}

// TestGetDetail_ResponseShape 验证 spec B §2.1 F-B1 字段扩展（MVP 占位版本）：
//   - 已有字段保留：id/name/description/cover_image/status/creator_id/created_at
//   - 新增字段（占位）：host_name="", host_avatar="", viewer_count=0,
//     video_url=null, is_following=false
//
// 跨服务 host 信息回填、is_following 真实查询留给后续 task；本期保证字段稳定存在。
func TestGetDetail_ResponseShape(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.LiveStream{
			ID:          101,
			CreatorID:   9001,
			Name:        "顶流主播·古玩专场",
			Description: "古玩鉴赏直播",
			CoverImage:  "https://cdn/.../cover.jpg",
			Status:      model.LiveStreamStatusActive,
		})
	})

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/live-streams/101")
	c.Params = append(c.Params, param.Param{Key: "id", Value: "101"})

	h.GetDetail(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	data := body["data"].(map[string]interface{})

	// 已有字段
	assert.EqualValues(t, 101, data["id"])
	assert.Equal(t, "顶流主播·古玩专场", data["name"])
	assert.Equal(t, "古玩鉴赏直播", data["description"])
	assert.Equal(t, "https://cdn/.../cover.jpg", data["cover_image"])
	assert.EqualValues(t, 9001, data["creator_id"])

	// 新增字段（MVP 占位）
	hostName, ok := data["host_name"]
	require.True(t, ok, "host_name field must exist")
	assert.Equal(t, "", hostName)

	hostAvatar, ok := data["host_avatar"]
	require.True(t, ok, "host_avatar field must exist")
	assert.Equal(t, "", hostAvatar)

	viewerCount, ok := data["viewer_count"]
	require.True(t, ok, "viewer_count field must exist")
	assert.EqualValues(t, 0, viewerCount)

	videoURL, ok := data["video_url"]
	require.True(t, ok, "video_url field must exist")
	assert.Nil(t, videoURL, "video_url should be null when not set")

	isFollowing, ok := data["is_following"]
	require.True(t, ok, "is_following field must exist")
	assert.Equal(t, false, isFollowing)
}

// TestGetDetail_VideoURL_FromDB 验证 video_url 从 DB 字段读取，
// 当 LiveStream.VideoURL 非空时返回字符串。
func TestGetDetail_VideoURL_FromDB(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.LiveStream{
			ID:        102,
			CreatorID: 9002,
			Name:      "test",
			Status:    model.LiveStreamStatusActive,
			VideoURL:  "https://cdn/.../live.m3u8",
		})
	})

	c := app.NewContext(0)
	c.Params = append(c.Params, param.Param{Key: "id", Value: "102"})
	h.GetDetail(context.Background(), c)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "https://cdn/.../live.m3u8", data["video_url"])
}

func TestListAdmin_T4FieldsAndStatusFilter(t *testing.T) {
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/auctions", r.URL.Path)
		assert.Equal(t, "101", r.URL.Query().Get("live_stream_id"))
		_, _ = w.Write([]byte(`{"code":200,"data":{"list":[],"total":3}}`))
	}))
	t.Cleanup(auctionMock.Close)

	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.AuctionRule{}, &model.LiveStream{}))
	db.Exec("DELETE FROM live_streams")
	require.NoError(t, db.Create(&model.LiveStream{
		ID:             101,
		CreatorID:      9001,
		Name:           "直播中",
		Status:         model.LiveStreamStatusLive,
		StreamerName:   "主播A",
		StreamerAvatar: "https://cdn/a.png",
	}).Error)
	require.NoError(t, db.Create(&model.LiveStream{
		ID:        102,
		CreatorID: 9002,
		Name:      "已结束",
		Status:    model.LiveStreamStatusEnded,
	}).Error)

	viewers := service.StaticLiveViewerCounter{101: 42}
	svc := service.NewLiveStreamServiceWithMetrics(dao.NewLiveStreamDAO(db), viewers)
	h := NewLiveStreamHandler(svc)
	h.SetAuctionClient(client.NewAuctionClient(auctionMock.URL, 0))

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/admin/live-streams?status=1")
	h.ListAdmin(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	data := body["data"].(map[string]interface{})
	list := data["list"].([]interface{})
	require.Len(t, list, 1)
	item := list[0].(map[string]interface{})
	assert.EqualValues(t, 101, item["id"])
	assert.EqualValues(t, 9001, item["streamer_id"])
	assert.Equal(t, "主播A", item["streamer_name"])
	assert.Equal(t, "https://cdn/a.png", item["streamer_avatar"])
	assert.EqualValues(t, 42, item["viewer_count"])
	assert.EqualValues(t, 3, item["auction_count"])
	assert.EqualValues(t, 1, item["status"])
}

func TestEndAndBanAdminLiveStream(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.LiveStream{ID: 201, CreatorID: 9001, Name: "待控制", Status: model.LiveStreamStatusLive})
	})

	endCtx := app.NewContext(0)
	endCtx.Params = append(endCtx.Params, param.Param{Key: "id", Value: "201"})
	h.EndAdmin(context.Background(), endCtx)
	assert.Equal(t, 200, endCtx.Response.StatusCode())
	var endBody map[string]interface{}
	require.NoError(t, json.Unmarshal(endCtx.Response.Body(), &endBody))
	endData := endBody["data"].(map[string]interface{})
	assert.EqualValues(t, model.LiveStreamStatusEnded, endData["status"])
	assert.Equal(t, "live_stream_ended", endData["event"])

	banCtx := app.NewContext(0)
	banCtx.Request.SetBody([]byte(`{"reason":"违规内容"}`))
	banCtx.Params = append(banCtx.Params, param.Param{Key: "id", Value: "201"})
	h.BanAdmin(context.Background(), banCtx)
	assert.Equal(t, 200, banCtx.Response.StatusCode())
	var banBody map[string]interface{}
	require.NoError(t, json.Unmarshal(banCtx.Response.Body(), &banBody))
	banData := banBody["data"].(map[string]interface{})
	assert.EqualValues(t, model.LiveStreamStatusBanned, banData["status"])
	assert.Equal(t, "违规内容", banData["ban_reason"])
}

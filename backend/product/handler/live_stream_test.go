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
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.Category{}, &model.AuctionRule{}, &model.LiveStream{}))
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
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/internal/auctions/count-by-live-streams", r.URL.Path)
		var req struct {
			LiveStreamIDs []int64 `json:"live_stream_ids"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, []int64{101}, req.LiveStreamIDs)
		_, _ = w.Write([]byte(`{"code":200,"message":"success","data":{"counts":{"101":3}}}`))
	}))
	t.Cleanup(auctionMock.Close)

	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.Category{}, &model.AuctionRule{}, &model.LiveStream{}))
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
	c.Request.Header.Set("X-User-Role", "admin")
	c.Request.Header.Set("X-User-ID", "2001")
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

func TestAdminGetReturnsAuctionCountAndViewerFallback(t *testing.T) {
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/internal/auctions/count-by-live-streams", r.URL.Path)
		var req struct {
			LiveStreamIDs []int64 `json:"live_stream_ids"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, []int64{201}, req.LiveStreamIDs)
		_, _ = w.Write([]byte(`{"code":200,"message":"success","data":{"counts":{"201":8}}}`))
	}))
	t.Cleanup(auctionMock.Close)

	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.Category{}, &model.AuctionRule{}, &model.LiveStream{}))
	require.NoError(t, db.Exec("DELETE FROM live_streams").Error)
	require.NoError(t, db.Create(&model.LiveStream{
		ID:          201,
		CreatorID:   9001,
		Name:        "带兜底人数",
		Status:      model.LiveStreamStatusLive,
		ViewerCount: 19,
	}).Error)

	svc := service.NewLiveStreamServiceWithMetrics(dao.NewLiveStreamDAO(db), service.StaticLiveViewerCounter{})
	h := NewLiveStreamHandler(svc)
	h.SetAuctionClient(client.NewAuctionClient(auctionMock.URL, 0))

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/admin/live-streams/201")
	c.Request.Header.Set("X-User-ID", "2001")
	c.Request.Header.Set("X-User-Role", "admin")
	c.Params = append(c.Params, param.Param{Key: "id", Value: "201"})

	h.AdminGet(context.Background(), c)

	require.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	data := body["data"].(map[string]interface{})
	assert.EqualValues(t, 8, data["auction_count"])
	assert.EqualValues(t, 19, data["viewer_count"])
}

func TestListAdminLiveStreamMerchantOnlyOwnStreams(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.LiveStream{ID: 301, CreatorID: 1001, Name: "A", Status: model.LiveStreamStatusLive})
		db.Create(&model.LiveStream{ID: 302, CreatorID: 1002, Name: "B", Status: model.LiveStreamStatusLive})
	})

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/admin/live-streams")
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "merchant")
	h.ListAdmin(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	data := body["data"].(map[string]interface{})
	list := data["list"].([]interface{})
	require.Len(t, list, 1)
	item := list[0].(map[string]interface{})
	assert.Equal(t, "A", item["name"])
}

func TestListAdminLiveStreamRequiresManagementRole(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.LiveStream{ID: 301, CreatorID: 9001, Name: "直播中", Status: model.LiveStreamStatusLive})
	})

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/admin/live-streams")
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "user")
	h.ListAdmin(context.Background(), c)

	assert.Equal(t, 403, c.Response.StatusCode())
}

func TestAdminCreateLiveStreamRejectsAdminActor(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, nil)
	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/admin/live-streams")
	c.Request.Header.Set("X-User-ID", "2001")
	c.Request.Header.Set("X-User-Role", "admin")
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.SetBodyString(`{"name":"平台代运营直播间"}`)

	h.AdminCreate(context.Background(), c)

	assert.Equal(t, 403, c.Response.StatusCode())
}

func TestAdminCreateLiveStreamMerchantSetsCreatorID(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, nil)
	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/admin/live-streams")
	c.Request.Header.Set("X-User-ID", "1001")
	c.Request.Header.Set("X-User-Role", "merchant")
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.SetBodyString(`{"name":"商家直播间","description":"说明","cover_image":"https://cdn/cover.png","video_url":"https://cdn/live.m3u8","streamer_name":"商家主播","streamer_avatar":"https://cdn/avatar.png"}`)

	h.AdminCreate(context.Background(), c)

	assert.Equal(t, 201, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	data := body["data"].(map[string]interface{})
	assert.EqualValues(t, 1001, data["creator_id"])
	assert.Equal(t, "商家直播间", data["name"])
	assert.EqualValues(t, model.LiveStreamStatusNotStarted, data["status"])
}

func TestMerchantStartAndEndOwnLiveStream(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.LiveStream{ID: 601, CreatorID: 1001, Name: "待开播", Status: model.LiveStreamStatusNotStarted})
	})

	startCtx := app.NewContext(0)
	startCtx.Params = append(startCtx.Params, param.Param{Key: "id", Value: "601"})
	startCtx.Request.Header.Set("X-User-Role", "merchant")
	startCtx.Request.Header.Set("X-User-ID", "1001")
	h.StartMerchant(context.Background(), startCtx)

	require.Equal(t, 200, startCtx.Response.StatusCode())
	var startBody map[string]interface{}
	require.NoError(t, json.Unmarshal(startCtx.Response.Body(), &startBody))
	startData := startBody["data"].(map[string]interface{})
	assert.EqualValues(t, model.LiveStreamStatusLive, startData["status"])
	assert.Equal(t, "live_stream_started", startData["event"])

	endCtx := app.NewContext(0)
	endCtx.Params = append(endCtx.Params, param.Param{Key: "id", Value: "601"})
	endCtx.Request.Header.Set("X-User-Role", "merchant")
	endCtx.Request.Header.Set("X-User-ID", "1001")
	h.EndMerchant(context.Background(), endCtx)

	require.Equal(t, 200, endCtx.Response.StatusCode())
	var endBody map[string]interface{}
	require.NoError(t, json.Unmarshal(endCtx.Response.Body(), &endBody))
	endData := endBody["data"].(map[string]interface{})
	assert.EqualValues(t, model.LiveStreamStatusEnded, endData["status"])
	assert.Equal(t, "live_stream_ended", endData["event"])
}

func TestMerchantCannotStartOtherLiveStream(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.LiveStream{ID: 602, CreatorID: 1002, Name: "其他商家", Status: model.LiveStreamStatusNotStarted})
	})

	c := app.NewContext(0)
	c.Params = append(c.Params, param.Param{Key: "id", Value: "602"})
	c.Request.Header.Set("X-User-Role", "merchant")
	c.Request.Header.Set("X-User-ID", "1001")
	h.StartMerchant(context.Background(), c)

	assert.Equal(t, 404, c.Response.StatusCode())
}

func TestMerchantCannotEndBannedLiveStream(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.LiveStream{ID: 603, CreatorID: 1001, Name: "已封禁", Status: model.LiveStreamStatusBanned})
	})

	c := app.NewContext(0)
	c.Params = append(c.Params, param.Param{Key: "id", Value: "603"})
	c.Request.Header.Set("X-User-Role", "merchant")
	c.Request.Header.Set("X-User-ID", "1001")
	h.EndMerchant(context.Background(), c)

	assert.Equal(t, 409, c.Response.StatusCode())

	check := app.NewContext(0)
	check.Params = append(check.Params, param.Param{Key: "id", Value: "603"})
	check.Request.Header.Set("X-User-Role", "merchant")
	check.Request.Header.Set("X-User-ID", "1001")
	h.AdminGet(context.Background(), check)
	require.Equal(t, 200, check.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(check.Response.Body(), &body))
	data := body["data"].(map[string]interface{})
	assert.EqualValues(t, model.LiveStreamStatusBanned, data["status"])
}

func TestAdminCreateLiveStreamReturnsOKForExistingStream(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, nil)

	first := app.NewContext(0)
	first.Request.SetRequestURI("/api/v1/admin/live-streams")
	first.Request.Header.Set("X-User-ID", "1001")
	first.Request.Header.Set("X-User-Role", "merchant")
	first.Request.Header.Set("Content-Type", "application/json")
	first.Request.SetBodyString(`{"name":"已有直播间"}`)
	h.AdminCreate(context.Background(), first)
	require.Equal(t, 201, first.Response.StatusCode())

	second := app.NewContext(0)
	second.Request.SetRequestURI("/api/v1/admin/live-streams")
	second.Request.Header.Set("X-User-ID", "1001")
	second.Request.Header.Set("X-User-Role", "merchant")
	second.Request.Header.Set("Content-Type", "application/json")
	second.Request.SetBodyString(`{"name":"重复创建直播间"}`)
	h.AdminCreate(context.Background(), second)

	assert.Equal(t, 200, second.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(second.Response.Body(), &body))
	assert.EqualValues(t, 200, body["code"])
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "已有直播间", data["name"])
}

func TestListAdminLiveStreamRejectsInvalidStatus(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, nil)

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/admin/live-streams?status=bad")
	c.Request.Header.Set("X-User-Role", "admin")
	c.Request.Header.Set("X-User-ID", "2001")
	h.ListAdmin(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
}

func TestListAdminLiveStreamClampsPageSize(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, nil)

	c := app.NewContext(0)
	c.Request.SetRequestURI("/api/v1/admin/live-streams?page_size=1000")
	c.Request.Header.Set("X-User-Role", "admin")
	c.Request.Header.Set("X-User-ID", "2001")
	h.ListAdmin(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	data := body["data"].(map[string]interface{})
	assert.EqualValues(t, 100, data["page_size"])
}

func TestEndAndBanAdminLiveStream(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.LiveStream{ID: 201, CreatorID: 9001, Name: "待控制", Status: model.LiveStreamStatusLive})
	})

	endCtx := app.NewContext(0)
	endCtx.Params = append(endCtx.Params, param.Param{Key: "id", Value: "201"})
	endCtx.Request.Header.Set("X-User-Role", "admin")
	endCtx.Request.Header.Set("X-User-ID", "2001")
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
	banCtx.Request.Header.Set("X-User-Role", "admin")
	banCtx.Request.Header.Set("X-User-ID", "2001")
	h.BanAdmin(context.Background(), banCtx)
	assert.Equal(t, 200, banCtx.Response.StatusCode())
	var banBody map[string]interface{}
	require.NoError(t, json.Unmarshal(banCtx.Response.Body(), &banBody))
	banData := banBody["data"].(map[string]interface{})
	assert.EqualValues(t, model.LiveStreamStatusBanned, banData["status"])
	assert.Equal(t, "违规内容", banData["ban_reason"])
}

func TestAdminEndDoesNotOverwriteBannedLiveStream(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.LiveStream{ID: 701, CreatorID: 9001, Name: "已封禁", Status: model.LiveStreamStatusBanned})
	})

	endCtx := app.NewContext(0)
	endCtx.Params = append(endCtx.Params, param.Param{Key: "id", Value: "701"})
	endCtx.Request.Header.Set("X-User-Role", "admin")
	endCtx.Request.Header.Set("X-User-ID", "2001")
	h.EndAdmin(context.Background(), endCtx)

	assert.Equal(t, 409, endCtx.Response.StatusCode())

	getCtx := app.NewContext(0)
	getCtx.Params = append(getCtx.Params, param.Param{Key: "id", Value: "701"})
	getCtx.Request.Header.Set("X-User-Role", "admin")
	getCtx.Request.Header.Set("X-User-ID", "2001")
	h.AdminGet(context.Background(), getCtx)
	require.Equal(t, 200, getCtx.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(getCtx.Response.Body(), &body))
	data := body["data"].(map[string]interface{})
	assert.EqualValues(t, model.LiveStreamStatusBanned, data["status"])
}

func TestBanAdminLiveStreamRejectsBlankReason(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.LiveStream{ID: 501, CreatorID: 9001, Name: "待封禁", Status: model.LiveStreamStatusLive})
	})

	banCtx := app.NewContext(0)
	banCtx.Request.SetBody([]byte(`{"reason":"   "}`))
	banCtx.Params = append(banCtx.Params, param.Param{Key: "id", Value: "501"})
	banCtx.Request.Header.Set("X-User-Role", "admin")
	banCtx.Request.Header.Set("X-User-ID", "2001")
	h.BanAdmin(context.Background(), banCtx)

	assert.Equal(t, 400, banCtx.Response.StatusCode())
}

func TestEndAndBanAdminLiveStreamRequireAdminRole(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.LiveStream{ID: 401, CreatorID: 9001, Name: "待控制", Status: model.LiveStreamStatusLive})
	})

	endCtx := app.NewContext(0)
	endCtx.Params = append(endCtx.Params, param.Param{Key: "id", Value: "401"})
	h.EndAdmin(context.Background(), endCtx)
	assert.Equal(t, 403, endCtx.Response.StatusCode())

	banCtx := app.NewContext(0)
	banCtx.Request.SetBody([]byte(`{"reason":"违规内容"}`))
	banCtx.Params = append(banCtx.Params, param.Param{Key: "id", Value: "401"})
	h.BanAdmin(context.Background(), banCtx)
	assert.Equal(t, 403, banCtx.Response.StatusCode())
}

func TestEndAndBanAdminLiveStreamReturnNotFound(t *testing.T) {
	h := newLiveStreamHandlerWithSeed(t, nil)

	endCtx := app.NewContext(0)
	endCtx.Params = append(endCtx.Params, param.Param{Key: "id", Value: "404"})
	endCtx.Request.Header.Set("X-User-Role", "admin")
	endCtx.Request.Header.Set("X-User-ID", "2001")
	h.EndAdmin(context.Background(), endCtx)
	assert.Equal(t, 404, endCtx.Response.StatusCode())

	banCtx := app.NewContext(0)
	banCtx.Request.SetBody([]byte(`{"reason":"违规内容"}`))
	banCtx.Params = append(banCtx.Params, param.Param{Key: "id", Value: "404"})
	banCtx.Request.Header.Set("X-User-Role", "admin")
	banCtx.Request.Header.Set("X-User-ID", "2001")
	h.BanAdmin(context.Background(), banCtx)
	assert.Equal(t, 404, banCtx.Response.StatusCode())
}

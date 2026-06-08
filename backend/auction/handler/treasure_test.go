package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"auction-service/client"
	"auction-service/dao"
	"auction-service/model"
	"auction-service/service"

	"github.com/alicebob/miniredis/v2"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var treasureHandlerDBCounter int64

type fakeTreasureLiveStreamLookup struct {
	items map[int64]client.LiveStreamSummary
	err   error
}

func (f fakeTreasureLiveStreamLookup) BatchGetLiveStreams(ctx context.Context, ids []int64) (map[int64]client.LiveStreamSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[int64]client.LiveStreamSummary, len(ids))
	for _, id := range ids {
		if item, ok := f.items[id]; ok {
			out[id] = item
		}
	}
	return out, nil
}

type treasureHandlerTestEnv struct {
	server *server.Hertz
	dao    *dao.TreasureDAO
}

func newTreasureHandlerTestEnv(t *testing.T) *treasureHandlerTestEnv {
	t.Helper()

	dsn := fmt.Sprintf("file:treasure_handler_test_%d?mode=memory&cache=shared", atomic.AddInt64(&treasureHandlerDBCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.UserCoin{},
		&model.UserWatchDuration{},
		&model.TreasureClaim{},
	))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	treasureDAO := dao.NewTreasureDAO(db)
	treasureService := service.NewTreasureService(treasureDAO, rdb)
	treasureService.SetLiveStreamLookup(fakeTreasureLiveStreamLookup{items: map[int64]client.LiveStreamSummary{
		200: {ID: 200, Status: 1},
	}})
	treasureHandler := NewTreasureHandler(treasureService)

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		if v := string(c.GetHeader("X-User-ID")); v != "" {
			if id, err := strconv.ParseInt(v, 10, 64); err == nil {
				c.Set("user_id", id)
			}
		}
		c.Next(ctx)
	})
	v1 := h.Group("/api/v1")
	v1.GET("/treasure/status", treasureHandler.GetStatus)
	v1.POST("/watch/heartbeat", treasureHandler.Heartbeat)
	v1.POST("/treasure/claim", treasureHandler.Claim)

	return &treasureHandlerTestEnv{server: h, dao: treasureDAO}
}

func treasureHandlerStatDate(t *testing.T) string {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	return time.Now().In(loc).Format("2006-01-02")
}

func performTreasureRequest(h *server.Hertz, method, path, userID, body string) *ut.ResponseRecorder {
	var reqBody *ut.Body
	if body != "" {
		reqBody = &ut.Body{Body: bytes.NewReader([]byte(body)), Len: len(body)}
	}
	headers := []ut.Header{{Key: "Content-Type", Value: "application/json"}}
	if userID != "" {
		headers = append(headers, ut.Header{Key: "X-User-ID", Value: userID})
	}
	return ut.PerformRequest(h.Engine, method, path, reqBody, headers...)
}

func TestTreasureHandlerGetStatusRequiresLogin(t *testing.T) {
	env := newTreasureHandlerTestEnv(t)

	w := performTreasureRequest(env.server, http.MethodGet, "/api/v1/treasure/status", "", "")

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "未登录")
}

func TestTreasureHandlerGetStatusResponseShape(t *testing.T) {
	env := newTreasureHandlerTestEnv(t)
	ctx := context.Background()
	_, err := env.dao.AddWatchSeconds(ctx, 100, treasureHandlerStatDate(t), 640)
	require.NoError(t, err)

	w := performTreasureRequest(env.server, http.MethodGet, "/api/v1/treasure/status", "100", "")

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			StatDate       string `json:"stat_date"`
			WatchedSeconds int    `json:"watched_seconds"`
			CoinBalance    int64  `json:"coin_balance"`
			Tiers          []struct {
				Tier             int8   `json:"tier"`
				ThresholdSeconds int    `json:"threshold_seconds"`
				Coins            int64  `json:"coins"`
				State            string `json:"state"`
			} `json:"tiers"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 200, resp.Code)
	assert.NotEmpty(t, resp.Data.StatDate)
	assert.Equal(t, 640, resp.Data.WatchedSeconds)
	assert.Equal(t, int64(0), resp.Data.CoinBalance)
	require.Len(t, resp.Data.Tiers, 3)
	assert.Equal(t, int8(0), resp.Data.Tiers[0].Tier)
	assert.Equal(t, 180, resp.Data.Tiers[0].ThresholdSeconds)
	assert.Equal(t, int64(100), resp.Data.Tiers[0].Coins)
	assert.Equal(t, "unlockable", resp.Data.Tiers[0].State)
}

func TestTreasureHandlerHeartbeatResponseShape(t *testing.T) {
	env := newTreasureHandlerTestEnv(t)

	w := performTreasureRequest(env.server, http.MethodPost, "/api/v1/watch/heartbeat", "100", `{"live_stream_id":200}`)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			WatchedSeconds int `json:"watched_seconds"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 200, resp.Code)
	assert.Equal(t, 30, resp.Data.WatchedSeconds)
}

func TestTreasureHandlerHeartbeatRejectsMissingLiveStreamID(t *testing.T) {
	env := newTreasureHandlerTestEnv(t)

	w := performTreasureRequest(env.server, http.MethodPost, "/api/v1/watch/heartbeat", "100", `{}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "live_stream_id")
}

func TestTreasureHandlerHeartbeatRejectsInvalidJSON(t *testing.T) {
	env := newTreasureHandlerTestEnv(t)

	w := performTreasureRequest(env.server, http.MethodPost, "/api/v1/watch/heartbeat", "100", `{"live_stream_id":`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid json")
}

func TestTreasureHandlerClaimRejectsThresholdNotMet(t *testing.T) {
	env := newTreasureHandlerTestEnv(t)

	w := performTreasureRequest(env.server, http.MethodPost, "/api/v1/treasure/claim", "100", `{"tier":0}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 400, resp.Code)
	assert.NotEmpty(t, resp.Message)
}

func TestTreasureHandlerClaimSuccessResponseShape(t *testing.T) {
	env := newTreasureHandlerTestEnv(t)
	ctx := context.Background()
	_, err := env.dao.AddWatchSeconds(ctx, 100, treasureHandlerStatDate(t), 200)
	require.NoError(t, err)

	w := performTreasureRequest(env.server, http.MethodPost, "/api/v1/treasure/claim", "100", `{"tier":0}`)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Coins       int64 `json:"coins"`
			CoinBalance int64 `json:"coin_balance"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 200, resp.Code)
	assert.Equal(t, int64(100), resp.Data.Coins)
	assert.Equal(t, int64(100), resp.Data.CoinBalance)
}

func TestTreasureHandlerClaimRejectsInvalidJSON(t *testing.T) {
	env := newTreasureHandlerTestEnv(t)

	w := performTreasureRequest(env.server, http.MethodPost, "/api/v1/treasure/claim", "100", `{"tier":`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid json")
}

func TestTreasureHandlerClaimRejectsTierOverflow(t *testing.T) {
	env := newTreasureHandlerTestEnv(t)
	ctx := context.Background()
	_, err := env.dao.AddWatchSeconds(ctx, 100, treasureHandlerStatDate(t), 200)
	require.NoError(t, err)

	w := performTreasureRequest(env.server, http.MethodPost, "/api/v1/treasure/claim", "100", `{"tier":256}`)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "无效的宝箱档位")
}

func TestTreasureHandlerClaimDuplicateReturnsConflict(t *testing.T) {
	env := newTreasureHandlerTestEnv(t)
	ctx := context.Background()
	_, err := env.dao.AddWatchSeconds(ctx, 100, treasureHandlerStatDate(t), 200)
	require.NoError(t, err)

	w := performTreasureRequest(env.server, http.MethodPost, "/api/v1/treasure/claim", "100", `{"tier":0}`)
	require.Equal(t, http.StatusOK, w.Code)
	w = performTreasureRequest(env.server, http.MethodPost, "/api/v1/treasure/claim", "100", `{"tier":0}`)

	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "已领取")
}

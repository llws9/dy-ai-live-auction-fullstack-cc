package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/assert"

	"gateway-service/config"
	"gateway-service/middleware"
)

// TestLiveStreamDetailRoute_OptionalJWT 验证 /live-streams/:id 详情路由：
//   - 无 token → 下游不应收到 X-User-ID（路由仍可访问，公开）
//   - 带合法 token → 下游必须收到 X-User-ID（透传给 product-service 用于查询 is_following）
//
// 对应 spec：docs/superpowers/specs/2026-05-30-h5-missing-b-livestream.md §5.3 / tasks T2.5。
//
// 当前路由 router.go#L106 在公开 v1 上注册，未应用 OptionalJWTAuth，
// 即便带合法 token，下游也收不到 X-User-ID — 故此测试在挂载中间件前必失败。
func TestLiveStreamDetailRoute_OptionalJWT(t *testing.T) {
	var lastUserIDHeader atomic.Value
	lastUserIDHeader.Store("")

	// Mock product-service：捕获请求 header。
	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastUserIDHeader.Store(r.Header.Get("X-User-ID"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"data":{"id":1}}`))
	}))
	defer productMock.Close()

	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer auctionMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			ProductURL: productMock.URL,
			AuctionURL: auctionMock.URL,
			TestURL:    "http://127.0.0.1:0",
			TestWSURL:  "ws://127.0.0.1:0",
		},
		JWT: config.JWTConfig{Secret: "live-stream-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	t.Run("no token → downstream receives no X-User-ID", func(t *testing.T) {
		lastUserIDHeader.Store("")
		w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/live-streams/123", nil)
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Equal(t, "", lastUserIDHeader.Load().(string))
	})

	t.Run("valid token → downstream receives X-User-ID", func(t *testing.T) {
		lastUserIDHeader.Store("")
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 4242, "alice", 0, 24)
		assert.NoError(t, err)

		w := ut.PerformRequest(
			h.Engine,
			http.MethodGet,
			"/api/v1/live-streams/123",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Equal(t, "4242", lastUserIDHeader.Load().(string),
			"带合法 token 时 gateway 必须把 X-User-ID 透传给 product-service")
	})

	t.Run("invalid token → still public, no X-User-ID, no 401", func(t *testing.T) {
		lastUserIDHeader.Store("")
		w := ut.PerformRequest(
			h.Engine,
			http.MethodGet,
			"/api/v1/live-streams/123",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer not-a-valid-jwt"},
		)
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode(), "非法 token 不应触发 401")
		assert.Equal(t, "", lastUserIDHeader.Load().(string))
	})

	// 静态分析也使用 context，避免 unused import。
	_ = context.Background
}

func TestTouchpointRoutes(t *testing.T) {
	tokenSecret := "touchpoint-route-secret"
	token, err := middleware.GenerateToken(tokenSecret, 123, "buyer", 0, 24)
	assert.NoError(t, err)
	authHeader := "Bearer " + token

	var auctionRequests atomic.Value
	auctionRequests.Store([]string{})
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, authHeader, r.Header.Get("Authorization"))
		assert.Equal(t, "123", r.Header.Get("X-User-ID"))
		requests := auctionRequests.Load().([]string)
		auctionRequests.Store(append(requests, r.Method+" "+r.URL.Path))
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/notifications/summary":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"unreadTotal":2,"outbid":1,"endingSoon":0}}`))
		case "/api/v1/notifications/read-category":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"success":true}}`))
		case "/api/v1/live/pending-reminder":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"hasReminder":false,"stream":null}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer auctionMock.Close()

	var productRequests atomic.Value
	productRequests.Store([]string{})
	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, authHeader, r.Header.Get("Authorization"))
		assert.Equal(t, "123", r.Header.Get("X-User-ID"))
		requests := productRequests.Load().([]string)
		productRequests.Store(append(requests, r.Method+" "+r.URL.Path))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"pendingPayment":1,"wonNotPaid":1}}`))
	}))
	defer productMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			ProductURL: productMock.URL,
			AuctionURL: auctionMock.URL,
			TestURL:    "http://127.0.0.1:0",
			TestWSURL:  "ws://127.0.0.1:0",
		},
		JWT: config.JWTConfig{Secret: tokenSecret},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	summary := ut.PerformRequest(
		h.Engine,
		http.MethodGet,
		"/api/v1/notifications/summary",
		nil,
		ut.Header{Key: "Authorization", Value: authHeader},
	).Result()
	assert.Equal(t, http.StatusOK, summary.StatusCode())
	assert.Contains(t, string(summary.Body()), `"unreadTotal":2`)
	assert.Contains(t, string(summary.Body()), `"pendingPayment":1`)

	readCategory := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/v1/notifications/read-category",
		&ut.Body{Body: strings.NewReader(`{"category":"outbid"}`), Len: len(`{"category":"outbid"}`)},
		ut.Header{Key: "Authorization", Value: authHeader},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	).Result()
	assert.Equal(t, http.StatusOK, readCategory.StatusCode())

	pendingReminder := ut.PerformRequest(
		h.Engine,
		http.MethodGet,
		"/api/v1/live/pending-reminder",
		nil,
		ut.Header{Key: "Authorization", Value: authHeader},
	).Result()
	assert.Equal(t, http.StatusOK, pendingReminder.StatusCode())

	assert.ElementsMatch(t, []string{
		"GET /api/v1/notifications/summary",
		"POST /api/v1/notifications/read-category",
		"GET /api/v1/live/pending-reminder",
	}, auctionRequests.Load().([]string))
	assert.ElementsMatch(t, []string{"GET /api/v1/orders/summary"}, productRequests.Load().([]string))
}

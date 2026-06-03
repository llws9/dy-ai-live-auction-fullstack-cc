package router

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/assert"

	"gateway-service/config"
)

// TestLiveStreamListRoute 验证公开列表路由 GET /api/v1/live-streams：
//   - 无需 JWT（公开），转发到 product-service。
//   - 转发路径为 /api/v1/live-streams，query 参数原样保留。
//
// 对应 H5 直播 feed（T3 已在 product 侧实现该接口）。
func TestLiveStreamListRoute(t *testing.T) {
	var lastPath atomic.Value
	var lastRawQuery atomic.Value
	lastPath.Store("")
	lastRawQuery.Store("")

	// Mock product-service：捕获请求 path 与 query。
	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath.Store(r.URL.Path)
		lastRawQuery.Store(r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
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
		JWT: config.JWTConfig{Secret: "live-stream-list-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	// 公开访问，不带 Authorization header。
	w := ut.PerformRequest(
		h.Engine,
		http.MethodGet,
		"/api/v1/live-streams?status=1&page=1&page_size=20",
		nil,
	)
	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode(), "公开列表路由应返回 200")
	assert.Equal(t, "/api/v1/live-streams", lastPath.Load().(string),
		"gateway 必须把请求转发到 product-service 的 /api/v1/live-streams")
	assert.Equal(t, "status=1&page=1&page_size=20", lastRawQuery.Load().(string),
		"query 参数必须原样保留")
}

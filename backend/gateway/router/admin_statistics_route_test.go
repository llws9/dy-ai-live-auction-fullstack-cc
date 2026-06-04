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
	"gateway-service/middleware"
)

func TestStatisticsRoutesRoleScope(t *testing.T) {
	var calls atomic.Int64
	var lastPath atomic.Value
	var lastRole atomic.Value
	lastPath.Store("")
	lastRole.Store("")

	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		lastPath.Store(r.URL.Path)
		lastRole.Store(r.Header.Get("X-User-Role"))
		w.WriteHeader(http.StatusOK)
	}))
	defer productMock.Close()
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		JWT: config.JWTConfig{Secret: "statistics-route-secret"},
	}
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)
	adminToken, err := middleware.GenerateToken(cfg.JWT.Secret, 99, "admin", 2, 24)
	assert.NoError(t, err)
	merchantToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9, "merchant", 1, 24)
	assert.NoError(t, err)

	calls.Store(0)
	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/statistics/overview", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + merchantToken})
	assert.Equal(t, http.StatusOK, w.Result().StatusCode())
	assert.Equal(t, int64(1), calls.Load())
	assert.Equal(t, "/api/v1/statistics/overview", lastPath.Load().(string))
	assert.Equal(t, "merchant", lastRole.Load().(string))

	calls.Store(0)
	w = ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/statistics/users", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + merchantToken})
	assert.Equal(t, http.StatusForbidden, w.Result().StatusCode())
	assert.Equal(t, int64(0), calls.Load())

	w = ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/statistics/users", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + adminToken})
	assert.Equal(t, http.StatusOK, w.Result().StatusCode())
	assert.Equal(t, int64(1), calls.Load())
	assert.Equal(t, "admin", lastRole.Load().(string))
}

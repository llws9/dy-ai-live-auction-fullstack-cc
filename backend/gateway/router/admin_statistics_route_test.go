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

func TestStatisticsAuctionRouteUsesAuctionService(t *testing.T) {
	var productCalls atomic.Int64
	var auctionCalls atomic.Int64
	var auctionPath atomic.Value
	var auctionQuery atomic.Value
	var auctionRole atomic.Value
	var auctionUserID atomic.Value
	var auctionInternalToken atomic.Value
	auctionPath.Store("")
	auctionQuery.Store("")
	auctionRole.Store("")
	auctionUserID.Store("")
	auctionInternalToken.Store("")

	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		productCalls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer productMock.Close()
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auctionCalls.Add(1)
		auctionPath.Store(r.URL.Path)
		auctionQuery.Store(r.URL.RawQuery)
		auctionRole.Store(r.Header.Get("X-User-Role"))
		auctionUserID.Store(r.Header.Get("X-User-ID"))
		auctionInternalToken.Store(r.Header.Get("X-Internal-Token"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer auctionMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			ProductURL:    productMock.URL,
			AuctionURL:    auctionMock.URL,
			TestURL:       "http://127.0.0.1:0",
			TestWSURL:     "ws://127.0.0.1:0",
			InternalToken: "internal-secret",
		},
		JWT: config.JWTConfig{Secret: "statistics-route-secret"},
	}
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)
	merchantToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9, "merchant", 1, 24)
	assert.NoError(t, err)

	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/statistics/auctions?start_date=2026-06-01&end_date=2026-06-07&group_by=day", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + merchantToken})

	assert.Equal(t, http.StatusOK, w.Result().StatusCode())
	assert.Equal(t, int64(0), productCalls.Load())
	assert.Equal(t, int64(1), auctionCalls.Load())
	assert.Equal(t, "/api/v1/statistics/auctions", auctionPath.Load().(string))
	assert.Equal(t, "start_date=2026-06-01&end_date=2026-06-07&group_by=day", auctionQuery.Load().(string))
	assert.Equal(t, "merchant", auctionRole.Load().(string))
	assert.Equal(t, "9", auctionUserID.Load().(string))
	assert.Equal(t, "internal-secret", auctionInternalToken.Load().(string))
}

func TestStatisticsNonAuctionRoutesStillUseProductService(t *testing.T) {
	var productCalls atomic.Int64
	var auctionCalls atomic.Int64
	var lastProductPath atomic.Value
	lastProductPath.Store("")

	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		productCalls.Add(1)
		lastProductPath.Store(r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer productMock.Close()
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auctionCalls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer auctionMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			ProductURL:    productMock.URL,
			AuctionURL:    auctionMock.URL,
			TestURL:       "http://127.0.0.1:0",
			TestWSURL:     "ws://127.0.0.1:0",
			InternalToken: "internal-secret",
		},
		JWT: config.JWTConfig{Secret: "statistics-route-secret"},
	}
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)
	adminToken, err := middleware.GenerateToken(cfg.JWT.Secret, 99, "admin", 2, 24)
	assert.NoError(t, err)
	merchantToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9, "merchant", 1, 24)
	assert.NoError(t, err)

	for _, path := range []string{"/api/v1/statistics/overview", "/api/v1/statistics/revenue"} {
		productCalls.Store(0)
		auctionCalls.Store(0)
		w := ut.PerformRequest(h.Engine, http.MethodGet, path, nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + merchantToken})
		assert.Equal(t, http.StatusOK, w.Result().StatusCode(), path)
		assert.Equal(t, int64(1), productCalls.Load(), path)
		assert.Equal(t, int64(0), auctionCalls.Load(), path)
		assert.Equal(t, path, lastProductPath.Load().(string))
	}

	productCalls.Store(0)
	auctionCalls.Store(0)
	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/statistics/users", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + merchantToken})
	assert.Equal(t, http.StatusForbidden, w.Result().StatusCode())
	assert.Equal(t, int64(0), productCalls.Load())
	assert.Equal(t, int64(0), auctionCalls.Load())

	productCalls.Store(0)
	auctionCalls.Store(0)
	w = ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/statistics/users", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + adminToken})
	assert.Equal(t, http.StatusOK, w.Result().StatusCode())
	assert.Equal(t, int64(1), productCalls.Load())
	assert.Equal(t, int64(0), auctionCalls.Load())
	assert.Equal(t, "/api/v1/statistics/users", lastProductPath.Load().(string))
}

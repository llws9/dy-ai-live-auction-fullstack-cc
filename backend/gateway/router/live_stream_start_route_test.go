package router

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gateway-service/config"
	"gateway-service/middleware"
)

func TestStartLiveRouteAllowsMerchantAndForwardsInternalHeaders(t *testing.T) {
	var productCalled atomic.Int32
	var auctionCalled atomic.Int32
	var productPath atomic.Value
	var auctionPath atomic.Value
	var capturedToken atomic.Value
	var capturedUserID atomic.Value
	var capturedRole atomic.Value
	var order atomic.Value
	productPath.Store("")
	auctionPath.Store("")
	capturedToken.Store("")
	capturedUserID.Store("")
	capturedRole.Store("")
	order.Store("")

	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		productCalled.Add(1)
		productPath.Store(r.URL.Path)
		order.Store(order.Load().(string) + "product>")
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "internal-secret", r.Header.Get("X-Internal-Token"))
		assert.Equal(t, "9002", r.Header.Get("X-User-ID"))
		assert.Equal(t, "merchant", r.Header.Get("X-User-Role"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"message":"success","data":{"id":123,"status":1,"event":"live_stream_started"}}`))
	}))
	defer productMock.Close()

	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auctionCalled.Add(1)
		auctionPath.Store(r.URL.Path)
		order.Store(order.Load().(string) + "auction>")
		capturedToken.Store(r.Header.Get("X-Internal-Token"))
		capturedUserID.Store(r.Header.Get("X-User-ID"))
		capturedRole.Store(r.Header.Get("X-User-Role"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"success":true}}`))
	}))
	defer auctionMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			AuctionURL:    auctionMock.URL,
			ProductURL:    productMock.URL,
			TestURL:       "http://127.0.0.1:0",
			TestWSURL:     "ws://127.0.0.1:0",
			InternalToken: "internal-secret",
		},
		JWT: config.JWTConfig{Secret: "start-live-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	merchantToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9002, "merchant", 1, 24)
	require.NoError(t, err)
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/v1/live-streams/123/start",
		nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + merchantToken},
	)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode())
	assert.Equal(t, int32(1), productCalled.Load())
	assert.Equal(t, int32(1), auctionCalled.Load())
	assert.Equal(t, "/api/v1/admin/live-streams/123/start", productPath.Load().(string))
	assert.Equal(t, "/internal/live-streams/123/start", auctionPath.Load().(string))
	assert.Equal(t, "auction>product>", order.Load().(string))
	assert.Equal(t, "internal-secret", capturedToken.Load().(string))
	assert.Equal(t, "9002", capturedUserID.Load().(string))
	assert.Equal(t, "merchant", capturedRole.Load().(string))
}

func TestStartLiveRouteDoesNotMarkProductLiveWhenAuctionProjectionFails(t *testing.T) {
	var productCalled atomic.Int32
	var auctionCalled atomic.Int32
	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		productCalled.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"data":{"status":1}}`))
	}))
	defer productMock.Close()
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auctionCalled.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"code":502,"message":"redis unavailable"}`))
	}))
	defer auctionMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			AuctionURL:    auctionMock.URL,
			ProductURL:    productMock.URL,
			TestURL:       "http://127.0.0.1:0",
			TestWSURL:     "ws://127.0.0.1:0",
			InternalToken: "internal-secret",
		},
		JWT: config.JWTConfig{Secret: "start-live-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	merchantToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9002, "merchant", 1, 24)
	require.NoError(t, err)
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/v1/live-streams/123/start",
		nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + merchantToken},
	)

	assert.Equal(t, http.StatusBadGateway, w.Result().StatusCode())
	assert.Equal(t, int32(1), auctionCalled.Load())
	assert.Equal(t, int32(0), productCalled.Load())
}

func TestStartLiveRouteStopsBeforeProductWhenAuctionRejectsOwner(t *testing.T) {
	var productCalled atomic.Int32
	var auctionCalled atomic.Int32
	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		productCalled.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	defer productMock.Close()
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auctionCalled.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":403,"message":"无权限操作直播间"}`))
	}))
	defer auctionMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			AuctionURL:    auctionMock.URL,
			ProductURL:    productMock.URL,
			TestURL:       "http://127.0.0.1:0",
			TestWSURL:     "ws://127.0.0.1:0",
			InternalToken: "internal-secret",
		},
		JWT: config.JWTConfig{Secret: "start-live-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	merchantToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9002, "merchant", 1, 24)
	require.NoError(t, err)
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/v1/live-streams/123/start",
		nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + merchantToken},
	)

	assert.Equal(t, http.StatusForbidden, w.Result().StatusCode())
	assert.Contains(t, string(w.Result().Body()), "无权限操作直播间")
	assert.Equal(t, int32(0), productCalled.Load())
	assert.Equal(t, int32(1), auctionCalled.Load())
}

func TestEndLiveRouteAllowsMerchantAndClearsAuctionProjection(t *testing.T) {
	var productCalled atomic.Int32
	var auctionCalled atomic.Int32
	var productPath atomic.Value
	var auctionPath atomic.Value
	productPath.Store("")
	auctionPath.Store("")

	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		productCalled.Add(1)
		productPath.Store(r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "internal-secret", r.Header.Get("X-Internal-Token"))
		assert.Equal(t, "9002", r.Header.Get("X-User-ID"))
		assert.Equal(t, "merchant", r.Header.Get("X-User-Role"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"message":"success","data":{"id":123,"status":2,"event":"live_stream_ended"}}`))
	}))
	defer productMock.Close()

	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auctionCalled.Add(1)
		auctionPath.Store(r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "internal-secret", r.Header.Get("X-Internal-Token"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"success":true}}`))
	}))
	defer auctionMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			AuctionURL:    auctionMock.URL,
			ProductURL:    productMock.URL,
			TestURL:       "http://127.0.0.1:0",
			TestWSURL:     "ws://127.0.0.1:0",
			InternalToken: "internal-secret",
		},
		JWT: config.JWTConfig{Secret: "end-live-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	merchantToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9002, "merchant", 1, 24)
	require.NoError(t, err)
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPut,
		"/api/v1/live-streams/123/end",
		nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + merchantToken},
	)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode())
	assert.Equal(t, int32(1), productCalled.Load())
	assert.Equal(t, int32(1), auctionCalled.Load())
	assert.Equal(t, "/api/v1/admin/live-streams/123/end", productPath.Load().(string))
	assert.Equal(t, "/internal/live-streams/123/end", auctionPath.Load().(string))
}

func TestStartLiveRouteRejectsAdminBeforeAuction(t *testing.T) {
	var called atomic.Int32
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer auctionMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			AuctionURL:    auctionMock.URL,
			ProductURL:    "http://127.0.0.1:0",
			TestURL:       "http://127.0.0.1:0",
			TestWSURL:     "ws://127.0.0.1:0",
			InternalToken: "internal-secret",
		},
		JWT: config.JWTConfig{Secret: "start-live-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	adminToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9001, "admin", 2, 24)
	require.NoError(t, err)
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/v1/live-streams/123/start",
		nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + adminToken},
	)

	assert.Equal(t, http.StatusForbidden, w.Result().StatusCode())
	assert.Equal(t, int32(0), called.Load())
}

func TestStartLiveRouteRejectsUserBeforeAuction(t *testing.T) {
	var called atomic.Int32
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer auctionMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			AuctionURL:    auctionMock.URL,
			ProductURL:    "http://127.0.0.1:0",
			TestURL:       "http://127.0.0.1:0",
			TestWSURL:     "ws://127.0.0.1:0",
			InternalToken: "internal-secret",
		},
		JWT: config.JWTConfig{Secret: "start-live-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	userToken, err := middleware.GenerateToken(cfg.JWT.Secret, 7001, "buyer", 0, 24)
	require.NoError(t, err)
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/v1/live-streams/123/start",
		nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + userToken},
	)

	assert.Equal(t, http.StatusForbidden, w.Result().StatusCode())
	assert.Equal(t, int32(0), called.Load())
}

func TestPendingReminderRouteForwardsToInternalWithToken(t *testing.T) {
	var called atomic.Int32
	var capturedPath atomic.Value
	var capturedToken atomic.Value
	var capturedUserID atomic.Value
	capturedPath.Store("")
	capturedToken.Store("")
	capturedUserID.Store("")

	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		capturedPath.Store(r.URL.Path)
		capturedToken.Store(r.Header.Get("X-Internal-Token"))
		capturedUserID.Store(r.Header.Get("X-User-ID"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"hasReminder":false}}`))
	}))
	defer auctionMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			AuctionURL:    auctionMock.URL,
			ProductURL:    "http://127.0.0.1:0",
			TestURL:       "http://127.0.0.1:0",
			TestWSURL:     "ws://127.0.0.1:0",
			InternalToken: "internal-secret",
		},
		JWT: config.JWTConfig{Secret: "pending-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	userToken, err := middleware.GenerateToken(cfg.JWT.Secret, 7001, "buyer", 0, 24)
	require.NoError(t, err)
	w := ut.PerformRequest(
		h.Engine,
		http.MethodGet,
		"/api/v1/live/pending-reminder",
		nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + userToken},
	)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode())
	assert.Equal(t, int32(1), called.Load())
	assert.Equal(t, "/internal/live/pending-reminder", capturedPath.Load().(string))
	assert.Equal(t, "internal-secret", capturedToken.Load().(string))
	assert.Equal(t, "7001", capturedUserID.Load().(string))
}

func TestAdminLiveStreamControlRoutesRequireAdminAndForwardInternalToken(t *testing.T) {
	var capturedPath atomic.Value
	var capturedToken atomic.Value
	var capturedAuctionPath atomic.Value
	capturedPath.Store("")
	capturedToken.Store("")
	capturedAuctionPath.Store("")

	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath.Store(r.URL.Path)
		capturedToken.Store(r.Header.Get("X-Internal-Token"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"data":{"ok":true}}`))
	}))
	defer productMock.Close()
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuctionPath.Store(r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"ok":true}}`))
	}))
	defer auctionMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			AuctionURL:    auctionMock.URL,
			ProductURL:    productMock.URL,
			TestURL:       "http://127.0.0.1:0",
			TestWSURL:     "ws://127.0.0.1:0",
			InternalToken: "internal-secret",
		},
		JWT: config.JWTConfig{Secret: "admin-live-control-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	adminToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9001, "admin", 2, 24)
	require.NoError(t, err)

	endResp := ut.PerformRequest(
		h.Engine,
		http.MethodPut,
		"/api/v1/admin/live-streams/123/end",
		nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + adminToken},
	)
	assert.Equal(t, http.StatusOK, endResp.Result().StatusCode())
	assert.Equal(t, "/api/v1/admin/live-streams/123/end", capturedPath.Load().(string))
	assert.Equal(t, "internal-secret", capturedToken.Load().(string))
	assert.Equal(t, "/internal/live-streams/123/end", capturedAuctionPath.Load().(string))

	banResp := ut.PerformRequest(
		h.Engine,
		http.MethodPut,
		"/api/v1/admin/live-streams/123/ban",
		nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + adminToken},
	)
	assert.Equal(t, http.StatusOK, banResp.Result().StatusCode())
	assert.Equal(t, "/api/v1/admin/live-streams/123/ban", capturedPath.Load().(string))

	streamerToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9002, "streamer", 1, 24)
	require.NoError(t, err)
	rejected := ut.PerformRequest(
		h.Engine,
		http.MethodPut,
		"/api/v1/admin/live-streams/123/end",
		nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + streamerToken},
	)
	assert.Equal(t, http.StatusForbidden, rejected.Result().StatusCode())
}

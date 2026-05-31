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

func TestStartLiveRouteRequiresAdminAndForwardsInternalToken(t *testing.T) {
	var called atomic.Int32
	var capturedPath atomic.Value
	var capturedToken atomic.Value
	capturedPath.Store("")
	capturedToken.Store("")

	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Add(1)
		capturedPath.Store(r.URL.Path)
		capturedToken.Store(r.Header.Get("X-Internal-Token"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"success":true}}`))
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

	assert.Equal(t, http.StatusOK, w.Result().StatusCode())
	assert.Equal(t, int32(1), called.Load())
	assert.Equal(t, "/internal/live-streams/123/start", capturedPath.Load().(string))
	assert.Equal(t, "internal-secret", capturedToken.Load().(string))
}

func TestStartLiveRouteRejectsNonAdminBeforeAuction(t *testing.T) {
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

	streamerToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9002, "streamer", 1, 24)
	require.NoError(t, err)
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/v1/live-streams/123/start",
		nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + streamerToken},
	)

	assert.Equal(t, http.StatusForbidden, w.Result().StatusCode())
	assert.Equal(t, int32(0), called.Load())
}

package router

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/require"

	"gateway-service/config"
	"gateway-service/middleware"
)

func TestBusinessWriteRoutesRejectAdminDelegatedOperations(t *testing.T) {
	var productCalls atomic.Int64
	var auctionCalls atomic.Int64
	var lastUserID atomic.Value
	var lastRole atomic.Value
	var lastInternalToken atomic.Value
	lastUserID.Store("")
	lastRole.Store("")
	lastInternalToken.Store("")
	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		productCalls.Add(1)
		lastUserID.Store(r.Header.Get("X-User-ID"))
		lastRole.Store(r.Header.Get("X-User-Role"))
		lastInternalToken.Store(r.Header.Get("X-Internal-Token"))
		w.WriteHeader(http.StatusOK)
	}))
	defer productMock.Close()
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auctionCalls.Add(1)
		lastUserID.Store(r.Header.Get("X-User-ID"))
		lastRole.Store(r.Header.Get("X-User-Role"))
		lastInternalToken.Store(r.Header.Get("X-Internal-Token"))
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
		JWT: config.JWTConfig{Secret: "business-write-secret"},
	}
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)
	adminToken, err := middleware.GenerateToken(cfg.JWT.Secret, 99, "admin", 2, 24)
	require.NoError(t, err)
	merchantToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9, "merchant", 1, 24)
	require.NoError(t, err)

	for _, tc := range []struct {
		method string
		path   string
		calls  *atomic.Int64
	}{
		{http.MethodPost, "/api/v1/products/ai/copywriting", &productCalls},
		{http.MethodPost, "/api/v1/products/1/publish", &productCalls},
		{http.MethodPost, "/api/v1/products/1/unpublish", &productCalls},
		{http.MethodPost, "/api/v1/auctions", &auctionCalls},
		{http.MethodPut, "/api/v1/auctions/1/cancel", &auctionCalls},
	} {
		tc.calls.Store(0)
		w := ut.PerformRequest(h.Engine, tc.method, tc.path, nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + adminToken})
		require.Equal(t, http.StatusForbidden, w.Result().StatusCode(), tc.path)
		require.Equal(t, int64(0), tc.calls.Load(), tc.path)

		w = ut.PerformRequest(h.Engine, tc.method, tc.path, nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + merchantToken})
		require.Equal(t, http.StatusOK, w.Result().StatusCode(), tc.path)
		require.Equal(t, "9", lastUserID.Load().(string), tc.path)
		require.Equal(t, "merchant", lastRole.Load().(string), tc.path)
		require.Equal(t, "", lastInternalToken.Load().(string), tc.path)
	}
}

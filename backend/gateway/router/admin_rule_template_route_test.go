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

func TestAdminRuleTemplateRoutesMerchantOnly(t *testing.T) {
	var calls atomic.Int64
	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.Equal(t, "internal-secret", r.Header.Get("X-Internal-Token"))
		require.Equal(t, "merchant", r.Header.Get("X-User-Role"))
		w.WriteHeader(http.StatusOK)
	}))
	defer productMock.Close()
	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		JWT: config.JWTConfig{Secret: "rule-template-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)
	merchantToken, err := middleware.GenerateToken(cfg.JWT.Secret, 1001, "seller", 1, 24)
	require.NoError(t, err)
	adminToken, err := middleware.GenerateToken(cfg.JWT.Secret, 2001, "admin", 2, 24)
	require.NoError(t, err)

	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/admin/auction-rule-templates", nil, ut.Header{Key: "Authorization", Value: "Bearer " + merchantToken})
	require.Equal(t, http.StatusOK, w.Result().StatusCode())
	require.Equal(t, int64(1), calls.Load())

	calls.Store(0)
	w = ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/admin/auction-rule-templates", nil, ut.Header{Key: "Authorization", Value: "Bearer " + adminToken})
	require.Equal(t, http.StatusForbidden, w.Result().StatusCode())
	require.Equal(t, int64(0), calls.Load())
}

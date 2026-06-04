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

func TestCopywritingRoute_RequireMerchantAndForwardIdentity(t *testing.T) {
	var calls atomic.Int64
	var lastUserID atomic.Value
	var lastUserRole atomic.Value
	lastUserID.Store("")
	lastUserRole.Store("")

	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		lastUserID.Store(r.Header.Get("X-User-ID"))
		lastUserRole.Store(r.Header.Get("X-User-Role"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"x","description":"y","selling_points":["a"],"suggested_start_price":"1"}`))
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
		JWT: config.JWTConfig{Secret: "copywriting-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	t.Run("without token is rejected at gateway", func(t *testing.T) {
		calls.Store(0)
		w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/products/ai/copywriting", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode())
		assert.Equal(t, int64(0), calls.Load())
	})

	t.Run("normal user is rejected before product service", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 42, "buyer", 0, 24)
		assert.NoError(t, err)

		w := ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/v1/products/ai/copywriting",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)

		assert.Equal(t, http.StatusForbidden, w.Result().StatusCode())
		assert.Equal(t, int64(0), calls.Load())
	})

	t.Run("merchant request forwards user identity", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 77, "merchant", 1, 24)
		assert.NoError(t, err)

		w := ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/v1/products/ai/copywriting",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode())
		assert.Equal(t, int64(1), calls.Load())
		assert.Equal(t, "77", lastUserID.Load().(string))
		assert.Equal(t, "merchant", lastUserRole.Load().(string))
	})

	t.Run("admin request forwards admin role", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 7, "admin", 2, 24)
		assert.NoError(t, err)

		w := ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/v1/products/ai/copywriting",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode())
		assert.Equal(t, int64(1), calls.Load())
		assert.Equal(t, "7", lastUserID.Load().(string))
		assert.Equal(t, "admin", lastUserRole.Load().(string))
	})
}

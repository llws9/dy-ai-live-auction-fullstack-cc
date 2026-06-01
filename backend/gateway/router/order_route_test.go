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

func TestOrderRoutes_RequireJWTAndForwardUserID(t *testing.T) {
	var calls atomic.Int64
	var lastMethod atomic.Value
	var lastUserID atomic.Value
	lastMethod.Store("")
	lastUserID.Store("")

	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		lastMethod.Store(r.Method)
		lastUserID.Store(r.Header.Get("X-User-ID"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"data":{"id":123}}`))
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
		JWT: config.JWTConfig{Secret: "order-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	t.Run("order detail without token is rejected at gateway", func(t *testing.T) {
		calls.Store(0)
		w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/orders/123", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode())
		assert.Equal(t, int64(0), calls.Load())
	})

	t.Run("order pay uses POST and forwards authenticated user", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 42, "buyer", 0, 24)
		assert.NoError(t, err)

		w := ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/v1/orders/123/pay",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode())
		assert.Equal(t, int64(1), calls.Load())
		assert.Equal(t, http.MethodPost, lastMethod.Load().(string))
		assert.Equal(t, "42", lastUserID.Load().(string))
	})
}

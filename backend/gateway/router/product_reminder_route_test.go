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

func TestProductReminderRoutes_RequireAuthAndForwardToAuction(t *testing.T) {
	var calls atomic.Int64
	var lastPath atomic.Value
	lastPath.Store("")

	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		lastPath.Store(r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"message":"订阅成功","data":{"product_id":88}}`))
	}))
	defer auctionMock.Close()

	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer productMock.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			ProductURL: productMock.URL,
			AuctionURL: auctionMock.URL,
			TestURL:    "http://127.0.0.1:0",
			TestWSURL:  "ws://127.0.0.1:0",
		},
		JWT: config.JWTConfig{Secret: "product-reminder-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	t.Run("without token is rejected at gateway", func(t *testing.T) {
		calls.Store(0)
		w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/products/88/remind", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode())
		assert.Equal(t, int64(0), calls.Load())
	})

	t.Run("authenticated subscribe is forwarded to auction service", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 42, "buyer", 0, 24)
		assert.NoError(t, err)

		w := ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/v1/products/88/remind",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode())
		assert.Equal(t, int64(1), calls.Load())
		assert.Equal(t, "/api/v1/products/88/remind", lastPath.Load().(string))
	})
}

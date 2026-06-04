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

func TestAdminProductRoutes_RoleScopeAndInternalToken(t *testing.T) {
	var calls atomic.Int64
	var lastMethod atomic.Value
	var lastInternalToken atomic.Value
	var lastUserRole atomic.Value
	lastMethod.Store("")
	lastInternalToken.Store("")
	lastUserRole.Store("")

	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		lastMethod.Store(r.Method)
		lastInternalToken.Store(r.Header.Get("X-Internal-Token"))
		lastUserRole.Store(r.Header.Get("X-User-Role"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"message":"success"}`))
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
		JWT: config.JWTConfig{Secret: "admin-product-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)

	t.Run("admin can read product list through internal proxy", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 7, "admin", 2, 24)
		assert.NoError(t, err)

		w := ut.PerformRequest(
			h.Engine,
			http.MethodGet,
			"/api/v1/admin/products",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode())
		assert.Equal(t, int64(1), calls.Load())
		assert.Equal(t, http.MethodGet, lastMethod.Load().(string))
		assert.Equal(t, "internal-secret", lastInternalToken.Load().(string))
		assert.Equal(t, "admin", lastUserRole.Load().(string))
	})

	t.Run("admin cannot create product", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 7, "admin", 2, 24)
		assert.NoError(t, err)

		w := ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/v1/admin/products",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)

		assert.Equal(t, http.StatusForbidden, w.Result().StatusCode())
		assert.Equal(t, int64(0), calls.Load())
	})

	t.Run("merchant can create product through internal proxy", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 1001, "seller", 1, 24)
		assert.NoError(t, err)

		w := ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/v1/admin/products",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode())
		assert.Equal(t, int64(1), calls.Load())
		assert.Equal(t, http.MethodPost, lastMethod.Load().(string))
		assert.Equal(t, "merchant", lastUserRole.Load().(string))
	})
}

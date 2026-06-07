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

// newFixedPriceTestGateway 构建一个把 /api/v1/fixed-price/* 转发到 auction mock 的网关。
// auctionMock 记录最近一次请求的 method / path / X-User-ID / X-Idempotency-Key。
func newFixedPriceTestGateway(t *testing.T) (*server.Hertz, *config.Config, *atomic.Int64, *atomic.Value, *atomic.Value, *atomic.Value, *atomic.Value) {
	t.Helper()

	var calls atomic.Int64
	var lastMethod, lastPath, lastUserID, lastIdemKey atomic.Value
	lastMethod.Store("")
	lastPath.Store("")
	lastUserID.Store("")
	lastIdemKey.Store("")

	auctionMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		lastMethod.Store(r.Method)
		lastPath.Store(r.URL.Path)
		lastUserID.Store(r.Header.Get("X-User-ID"))
		lastIdemKey.Store(r.Header.Get("X-Idempotency-Key"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"order_id":88001}}`))
	}))
	t.Cleanup(auctionMock.Close)

	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(productMock.Close)

	cfg := &config.Config{
		Services: config.ServicesConfig{
			ProductURL: productMock.URL,
			AuctionURL: auctionMock.URL,
			TestURL:    "http://127.0.0.1:0",
			TestWSURL:  "ws://127.0.0.1:0",
		},
		JWT: config.JWTConfig{Secret: "fixed-price-route-secret"},
	}

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)
	return h, cfg, &calls, &lastMethod, &lastPath, &lastUserID, &lastIdemKey
}

func TestFixedPricePurchaseRoute_RequireJWTAndForwardHeaders(t *testing.T) {
	h, cfg, calls, lastMethod, lastPath, lastUserID, lastIdemKey := newFixedPriceTestGateway(t)

	t.Run("purchase without token is rejected at gateway", func(t *testing.T) {
		calls.Store(0)
		w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/fixed-price/items/77/purchase", nil)
		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode())
		assert.Equal(t, int64(0), calls.Load())
	})

	t.Run("purchase with token forwards user id and idempotency key", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 42, "buyer", 0, 24)
		assert.NoError(t, err)

		w := ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/v1/fixed-price/items/77/purchase",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
			ut.Header{Key: "X-Idempotency-Key", Value: "550e8400-e29b-41d4-a716-446655440000"},
		)

		assert.Equal(t, http.StatusOK, w.Result().StatusCode())
		assert.Equal(t, int64(1), calls.Load())
		assert.Equal(t, http.MethodPost, lastMethod.Load().(string))
		assert.Equal(t, "/api/v1/fixed-price/items/77/purchase", lastPath.Load().(string))
		assert.Equal(t, "42", lastUserID.Load().(string))
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", lastIdemKey.Load().(string))
	})
}

func TestFixedPriceMyPurchaseRoute_RequireJWT(t *testing.T) {
	h, cfg, calls, _, _, lastUserID, _ := newFixedPriceTestGateway(t)

	t.Run("my-purchase without token is rejected", func(t *testing.T) {
		calls.Store(0)
		w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/fixed-price/items/77/my-purchase", nil)
		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode())
		assert.Equal(t, int64(0), calls.Load())
	})

	t.Run("my-purchase with token forwards user id", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 7, "buyer", 0, 24)
		assert.NoError(t, err)
		w := ut.PerformRequest(
			h.Engine,
			http.MethodGet,
			"/api/v1/fixed-price/items/77/my-purchase",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode())
		assert.Equal(t, int64(1), calls.Load())
		assert.Equal(t, "7", lastUserID.Load().(string))
	})
}

func TestFixedPriceCreateAndOfflineRoutes_RequireMerchantOnly(t *testing.T) {
	h, cfg, calls, lastMethod, _, _, _ := newFixedPriceTestGateway(t)

	t.Run("create as buyer is rejected before auction service", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 42, "buyer", 0, 24)
		assert.NoError(t, err)
		w := ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/v1/fixed-price/items",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)
		assert.Equal(t, http.StatusForbidden, w.Result().StatusCode())
		assert.Equal(t, int64(0), calls.Load())
	})

	t.Run("create as streamer is forwarded", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 9, "streamer", 1, 24)
		assert.NoError(t, err)
		w := ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/v1/fixed-price/items",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode())
		assert.Equal(t, int64(1), calls.Load())
		assert.Equal(t, http.MethodPost, lastMethod.Load().(string))
	})

	t.Run("create as admin is rejected before auction service", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 99, "admin", 2, 24)
		assert.NoError(t, err)
		w := ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/v1/fixed-price/items",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)
		assert.Equal(t, http.StatusForbidden, w.Result().StatusCode())
		assert.Equal(t, int64(0), calls.Load())
	})

	t.Run("offline as buyer is rejected before auction service", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 42, "buyer", 0, 24)
		assert.NoError(t, err)
		w := ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/v1/fixed-price/items/77/offline",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)
		assert.Equal(t, http.StatusForbidden, w.Result().StatusCode())
		assert.Equal(t, int64(0), calls.Load())
	})

	t.Run("offline as streamer is forwarded", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 9, "streamer", 1, 24)
		assert.NoError(t, err)
		w := ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/v1/fixed-price/items/77/offline",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode())
		assert.Equal(t, int64(1), calls.Load())
	})

	t.Run("offline as admin is rejected before auction service", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 99, "admin", 2, 24)
		assert.NoError(t, err)
		w := ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/v1/fixed-price/items/77/offline",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)
		assert.Equal(t, http.StatusForbidden, w.Result().StatusCode())
		assert.Equal(t, int64(0), calls.Load())
	})
}

func TestFixedPriceDetailRoute_PublicForward(t *testing.T) {
	h, _, calls, lastMethod, lastPath, _, _ := newFixedPriceTestGateway(t)

	calls.Store(0)
	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/fixed-price/items/77", nil)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode())
	assert.Equal(t, int64(1), calls.Load())
	assert.Equal(t, http.MethodGet, lastMethod.Load().(string))
	assert.Equal(t, "/api/v1/fixed-price/items/77", lastPath.Load().(string))
}

func TestFixedPriceLiveStreamListRoute_PublicForward(t *testing.T) {
	h, _, calls, lastMethod, lastPath, _, _ := newFixedPriceTestGateway(t)

	calls.Store(0)
	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/live-streams/1001/fixed-price/items", nil)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode())
	assert.Equal(t, int64(1), calls.Load())
	assert.Equal(t, http.MethodGet, lastMethod.Load().(string))
	assert.Equal(t, "/api/v1/live-streams/1001/fixed-price/items", lastPath.Load().(string))
}

func TestFixedPriceAuctionListRoute_PublicForward(t *testing.T) {
	h, _, calls, lastMethod, lastPath, _, _ := newFixedPriceTestGateway(t)

	calls.Store(0)
	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/auctions/8002/fixed-price/items", nil)
	assert.Equal(t, http.StatusOK, w.Result().StatusCode())
	assert.Equal(t, int64(1), calls.Load())
	assert.Equal(t, http.MethodGet, lastMethod.Load().(string))
	assert.Equal(t, "/api/v1/auctions/8002/fixed-price/items", lastPath.Load().(string))
}

func TestFixedPriceAdminLiveStreamListRoute_RequireStreamer(t *testing.T) {
	h, cfg, calls, lastMethod, lastPath, lastUserID, _ := newFixedPriceTestGateway(t)

	t.Run("admin list as buyer is rejected before auction service", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 42, "buyer", 0, 24)
		assert.NoError(t, err)
		w := ut.PerformRequest(
			h.Engine,
			http.MethodGet,
			"/api/v1/admin/live-streams/1001/fixed-price/items",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)
		assert.Equal(t, http.StatusForbidden, w.Result().StatusCode())
		assert.Equal(t, int64(0), calls.Load())
	})

	t.Run("admin list as streamer is forwarded", func(t *testing.T) {
		calls.Store(0)
		token, err := middleware.GenerateToken(cfg.JWT.Secret, 9, "streamer", 1, 24)
		assert.NoError(t, err)
		w := ut.PerformRequest(
			h.Engine,
			http.MethodGet,
			"/api/v1/admin/live-streams/1001/fixed-price/items",
			nil,
			ut.Header{Key: "Authorization", Value: "Bearer " + token},
		)
		assert.Equal(t, http.StatusOK, w.Result().StatusCode())
		assert.Equal(t, int64(1), calls.Load())
		assert.Equal(t, http.MethodGet, lastMethod.Load().(string))
		assert.Equal(t, "/api/v1/admin/live-streams/1001/fixed-price/items", lastPath.Load().(string))
		assert.Equal(t, "9", lastUserID.Load().(string))
	})
}

func TestOrderAdminRoutesAllowMerchantReadAndShipMerchantOnly(t *testing.T) {
	var calls atomic.Int64
	var lastMethod, lastPath, lastRole atomic.Value
	lastMethod.Store("")
	lastPath.Store("")
	lastRole.Store("")
	productMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		lastMethod.Store(r.Method)
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
			ProductURL:    productMock.URL,
			AuctionURL:    auctionMock.URL,
			TestURL:       "http://127.0.0.1:0",
			TestWSURL:     "ws://127.0.0.1:0",
			InternalToken: "internal-secret",
		},
		JWT: config.JWTConfig{Secret: "order-route-secret"},
	}
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	RegisterRoutes(h, cfg, nil)
	adminToken, err := middleware.GenerateToken(cfg.JWT.Secret, 99, "admin", 2, 24)
	assert.NoError(t, err)
	merchantToken, err := middleware.GenerateToken(cfg.JWT.Secret, 9, "merchant", 1, 24)
	assert.NoError(t, err)

	calls.Store(0)
	w := ut.PerformRequest(h.Engine, http.MethodGet, "/api/v1/admin/orders", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + merchantToken})
	assert.Equal(t, http.StatusOK, w.Result().StatusCode())
	assert.Equal(t, int64(1), calls.Load())
	assert.Equal(t, "/api/v1/admin/orders", lastPath.Load().(string))
	assert.Equal(t, "merchant", lastRole.Load().(string))

	calls.Store(0)
	w = ut.PerformRequest(h.Engine, http.MethodPut, "/api/v1/orders/77/ship", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + adminToken})
	assert.Equal(t, http.StatusForbidden, w.Result().StatusCode())
	assert.Equal(t, int64(0), calls.Load())

	w = ut.PerformRequest(h.Engine, http.MethodPut, "/api/v1/orders/77/ship", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + merchantToken})
	assert.Equal(t, http.StatusOK, w.Result().StatusCode())
	assert.Equal(t, http.MethodPut, lastMethod.Load().(string))
	assert.Equal(t, "merchant", lastRole.Load().(string))
}

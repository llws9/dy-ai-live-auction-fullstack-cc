package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"auction-service/client"
	"auction-service/model"
	"auction-service/service"
)

// fakeFixedPriceUsecase 实现 FixedPriceUsecase，按需返回可配置结果/错误，
// 用于在 Hertz handler 单测中隔离 service 编排。
type fakeFixedPriceUsecase struct {
	purchaseResult *service.PurchaseResult
	purchaseErr    error
	purchaseReqs   []service.PurchaseReq

	listResult *model.FixedPriceItem
	listErr    error
	listReqs   []service.ListItemReq

	liveItems    []*service.LiveFixedPriceItem
	liveItemsErr error
	liveItemReqs []service.ListLiveItemsReq

	adminItems    []*service.LiveFixedPriceItem
	adminItemsErr error
	adminItemReqs []service.ListLiveItemsReq

	offlineErr   error
	offlineCalls []int64

	item    *model.FixedPriceItem
	itemErr error

	remaining    int
	remainingErr error

	myPurchase    *model.FixedPricePurchase
	myPurchaseErr error
}

func (f *fakeFixedPriceUsecase) Purchase(_ context.Context, r service.PurchaseReq) (*service.PurchaseResult, error) {
	f.purchaseReqs = append(f.purchaseReqs, r)
	return f.purchaseResult, f.purchaseErr
}

func (f *fakeFixedPriceUsecase) ListItem(_ context.Context, r service.ListItemReq) (*model.FixedPriceItem, error) {
	f.listReqs = append(f.listReqs, r)
	return f.listResult, f.listErr
}

func (f *fakeFixedPriceUsecase) ListByLiveStream(_ context.Context, r service.ListLiveItemsReq) ([]*service.LiveFixedPriceItem, error) {
	f.liveItemReqs = append(f.liveItemReqs, r)
	return f.liveItems, f.liveItemsErr
}

func (f *fakeFixedPriceUsecase) ListAllByLiveStream(_ context.Context, r service.ListLiveItemsReq) ([]*service.LiveFixedPriceItem, error) {
	f.adminItemReqs = append(f.adminItemReqs, r)
	return f.adminItems, f.adminItemsErr
}

func (f *fakeFixedPriceUsecase) Offline(_ context.Context, itemID, _ int64) error {
	f.offlineCalls = append(f.offlineCalls, itemID)
	return f.offlineErr
}

func (f *fakeFixedPriceUsecase) GetItem(_ context.Context, _ int64) (*model.FixedPriceItem, error) {
	if f.itemErr != nil {
		return nil, f.itemErr
	}
	if f.item == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return f.item, nil
}

func (f *fakeFixedPriceUsecase) RemainingStock(_ context.Context, _ int64) (int, error) {
	return f.remaining, f.remainingErr
}

func (f *fakeFixedPriceUsecase) GetMyPurchase(_ context.Context, _, _ int64) (*model.FixedPricePurchase, error) {
	if f.myPurchaseErr != nil {
		return nil, f.myPurchaseErr
	}
	if f.myPurchase == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return f.myPurchase, nil
}

// fakeFPBalanceProvider 实现 handler.BalanceProvider，返回固定余额。
type fakeFPBalanceProvider struct {
	available decimal.Decimal
	hit       bool
	err       error
}

func (f *fakeFPBalanceProvider) GetByUserID(_ context.Context, _ int64) (decimal.Decimal, decimal.Decimal, string, bool, error) {
	return f.available, decimal.Zero, "CNY", f.hit, f.err
}

// newFixedPriceTestServer 构建挂载一口价路由的 Hertz engine。
// 通过轻量中间件把 X-User-ID 翻译为 c.Set("user_id")，对齐生产 gatewayIdentityMiddleware。
func newFixedPriceTestServer(uc FixedPriceUsecase, bp BalanceProvider) *route.Engine {
	return newFixedPriceTestServerWithProductClient(uc, bp, nil)
}

func newFixedPriceTestServerWithProductClient(uc FixedPriceUsecase, bp BalanceProvider, pc client.ProductClient) *route.Engine {
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		if v := string(c.GetHeader("X-User-ID")); v != "" {
			if id, err := strconv.ParseInt(v, 10, 64); err == nil {
				c.Set("user_id", id)
			}
		}
		c.Next(ctx)
	})
	fph := NewFixedPriceHandler(uc, bp)
	fph.SetProductClient(pc)
	v1 := h.Group("/api/v1")
	RegisterFixedPriceRoutes(v1, fph)
	return h.Engine
}

const validIdemKey = "550e8400-e29b-41d4-a716-446655440000"

func decodeFPErr(t *testing.T, body []byte) fpErrResp {
	t.Helper()
	var r fpErrResp
	require.NoError(t, json.Unmarshal(body, &r))
	return r
}

// --- T9 抢购 handler 错误码映射 ---

func TestPurchaseHandler_MissingIdemKey_400(t *testing.T) {
	eng := newFixedPriceTestServer(&fakeFixedPriceUsecase{}, &fakeFPBalanceProvider{})
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items/7001/purchase", nil,
		ut.Header{Key: "X-User-ID", Value: "100"})
	resp := w.Result()
	assert.Equal(t, 400, resp.StatusCode())
	assert.Equal(t, "FP_INVALID_PARAM", decodeFPErr(t, resp.Body()).Code)
}

func TestPurchaseHandler_MissingUser_401(t *testing.T) {
	eng := newFixedPriceTestServer(&fakeFixedPriceUsecase{}, &fakeFPBalanceProvider{})
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items/7001/purchase", nil,
		ut.Header{Key: "X-Idempotency-Key", Value: validIdemKey})
	resp := w.Result()
	assert.Equal(t, 401, resp.StatusCode())
	assert.Equal(t, "FP_NOT_AUTHENTICATED", decodeFPErr(t, resp.Body()).Code)
}

func TestPurchaseHandler_InvalidIdemFormat_400(t *testing.T) {
	uc := &fakeFixedPriceUsecase{purchaseErr: service.ErrInvalidParam}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items/7001/purchase", nil,
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-Idempotency-Key", Value: "not-a-uuid"})
	resp := w.Result()
	assert.Equal(t, 400, resp.StatusCode())
	assert.Equal(t, "FP_INVALID_PARAM", decodeFPErr(t, resp.Body()).Code)
}

func TestPurchaseHandler_SoldOut_409(t *testing.T) {
	uc := &fakeFixedPriceUsecase{purchaseErr: service.ErrSoldOut}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items/7001/purchase", nil,
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-Idempotency-Key", Value: validIdemKey})
	resp := w.Result()
	assert.Equal(t, 409, resp.StatusCode())
	assert.Equal(t, "FP_SOLD_OUT", decodeFPErr(t, resp.Body()).Code)
}

func TestPurchaseHandler_AlreadyBought_409(t *testing.T) {
	uc := &fakeFixedPriceUsecase{purchaseErr: service.ErrAlreadyBought}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items/7001/purchase", nil,
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-Idempotency-Key", Value: validIdemKey})
	resp := w.Result()
	assert.Equal(t, 409, resp.StatusCode())
	assert.Equal(t, "FP_ALREADY_BOUGHT", decodeFPErr(t, resp.Body()).Code)
}

func TestPurchaseHandler_NotOnSale_409(t *testing.T) {
	uc := &fakeFixedPriceUsecase{purchaseErr: service.ErrNotOnSale}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items/7001/purchase", nil,
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-Idempotency-Key", Value: validIdemKey})
	resp := w.Result()
	assert.Equal(t, 409, resp.StatusCode())
	assert.Equal(t, "FP_NOT_ON_SALE", decodeFPErr(t, resp.Body()).Code)
}

func TestPurchaseHandler_InsufficientBalance_402_WithDetails(t *testing.T) {
	uc := &fakeFixedPriceUsecase{
		purchaseErr: service.ErrInsufficient,
		item:        &model.FixedPriceItem{ID: 7001, Price: decimal.NewFromInt(99)},
	}
	bp := &fakeFPBalanceProvider{available: decimal.NewFromInt(50), hit: true}
	eng := newFixedPriceTestServer(uc, bp)
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items/7001/purchase", nil,
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-Idempotency-Key", Value: validIdemKey})
	resp := w.Result()
	assert.Equal(t, 402, resp.StatusCode())
	body := decodeFPErr(t, resp.Body())
	assert.Equal(t, "FP_INSUFFICIENT_BALANCE", body.Code)
	assert.Equal(t, "99.00", body.Details["required"])
	assert.Equal(t, "50.00", body.Details["available"])
	assert.Equal(t, "49.00", body.Details["shortage"])
}

func TestPurchaseHandler_Success_PriceAsString(t *testing.T) {
	uc := &fakeFixedPriceUsecase{purchaseResult: &service.PurchaseResult{
		PurchaseID: 88001, ItemID: 7001, Price: decimal.NewFromInt(99), RemainingStock: 4,
	}}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items/7001/purchase", nil,
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-Idempotency-Key", Value: validIdemKey})
	resp := w.Result()
	require.Equal(t, 200, resp.StatusCode())
	var body map[string]any
	require.NoError(t, json.Unmarshal(resp.Body(), &body))
	assert.Equal(t, "99.00", body["price"])
	assert.Equal(t, float64(4), body["remaining_stock"])
	assert.Equal(t, float64(88001), body["order_id"])
	assert.Equal(t, "success", body["status"])
	// 透传给 service 的 idem key 与 item id 正确。
	require.Len(t, uc.purchaseReqs, 1)
	assert.Equal(t, int64(7001), uc.purchaseReqs[0].ItemID)
	assert.Equal(t, validIdemKey, uc.purchaseReqs[0].IdemKey)
}

// --- T10 上架/下架/详情/my-purchase handler ---

func TestListItemHandler_Success(t *testing.T) {
	uc := &fakeFixedPriceUsecase{listResult: &model.FixedPriceItem{
		ID: 7001, Status: model.FixedPriceStatusOnSale, RemainingStock: 100,
	}}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})
	body := `{"live_stream_id":1001,"product_id":5001,"price":"99.00","total_stock":100,"max_per_user":1}`
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items", &ut.Body{Body: bytes.NewReader([]byte(body)), Len: len(body)},
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-User-Role", Value: "merchant"},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	resp := w.Result()
	require.Equal(t, 200, resp.StatusCode())
	var out map[string]any
	require.NoError(t, json.Unmarshal(resp.Body(), &out))
	assert.Equal(t, "on_sale", out["status"])
	assert.Equal(t, float64(7001), out["id"])
	// CreatorID 取登录用户，不信任请求体。
	require.Len(t, uc.listReqs, 1)
	assert.Equal(t, int64(100), uc.listReqs[0].CreatorID)
	assert.Equal(t, "99", uc.listReqs[0].Price.String())
}

func TestListItemHandler_NonOwner_403(t *testing.T) {
	uc := &fakeFixedPriceUsecase{listErr: service.ErrNotStreamOwner}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})
	body := `{"live_stream_id":1001,"product_id":5001,"price":"99.00","total_stock":100}`
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items", &ut.Body{Body: bytes.NewReader([]byte(body)), Len: len(body)},
		ut.Header{Key: "X-User-ID", Value: "9999"},
		ut.Header{Key: "X-User-Role", Value: "merchant"},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	resp := w.Result()
	assert.Equal(t, 403, resp.StatusCode())
	assert.Equal(t, "FP_NOT_STREAM_OWNER", decodeFPErr(t, resp.Body()).Code)
}

func TestListItemHandler_AdminRoleRejected(t *testing.T) {
	uc := &fakeFixedPriceUsecase{listResult: &model.FixedPriceItem{ID: 7001}}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})
	body := `{"live_stream_id":1001,"product_id":5001,"price":"99.00","total_stock":100}`
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items", &ut.Body{Body: bytes.NewReader([]byte(body)), Len: len(body)},
		ut.Header{Key: "X-User-ID", Value: "99"},
		ut.Header{Key: "X-User-Role", Value: "admin"},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	resp := w.Result()
	assert.Equal(t, 403, resp.StatusCode())
	assert.Equal(t, "FP_FORBIDDEN_ROLE", decodeFPErr(t, resp.Body()).Code)
	assert.Empty(t, uc.listReqs)
}

func TestListItemHandler_MissingUser_401(t *testing.T) {
	eng := newFixedPriceTestServer(&fakeFixedPriceUsecase{}, &fakeFPBalanceProvider{})
	body := `{"live_stream_id":1001,"product_id":5001,"price":"99.00","total_stock":100}`
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items", &ut.Body{Body: bytes.NewReader([]byte(body)), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	resp := w.Result()
	assert.Equal(t, 401, resp.StatusCode())
}

func TestOfflineHandler_Success(t *testing.T) {
	uc := &fakeFixedPriceUsecase{}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items/7001/offline", nil,
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-User-Role", Value: "merchant"})
	resp := w.Result()
	require.Equal(t, 200, resp.StatusCode())
	assert.Equal(t, []int64{7001}, uc.offlineCalls)
}

func TestOfflineHandler_NonOwner_403(t *testing.T) {
	uc := &fakeFixedPriceUsecase{offlineErr: service.ErrNotStreamOwner}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items/7001/offline", nil,
		ut.Header{Key: "X-User-ID", Value: "9999"},
		ut.Header{Key: "X-User-Role", Value: "merchant"})
	resp := w.Result()
	assert.Equal(t, 403, resp.StatusCode())
	assert.Equal(t, "FP_NOT_STREAM_OWNER", decodeFPErr(t, resp.Body()).Code)
}

func TestOfflineHandler_AdminRoleRejected(t *testing.T) {
	uc := &fakeFixedPriceUsecase{}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items/7001/offline", nil,
		ut.Header{Key: "X-User-ID", Value: "99"},
		ut.Header{Key: "X-User-Role", Value: "admin"})
	resp := w.Result()
	assert.Equal(t, 403, resp.StatusCode())
	assert.Equal(t, "FP_FORBIDDEN_ROLE", decodeFPErr(t, resp.Body()).Code)
	assert.Empty(t, uc.offlineCalls)
}

func TestDetailHandler_Success(t *testing.T) {
	uc := &fakeFixedPriceUsecase{
		item: &model.FixedPriceItem{
			ID: 7001, LiveStreamID: 1001, ProductID: 5001,
			Price: decimal.NewFromInt(99), TotalStock: 100, RemainingStock: 100,
			MaxPerUser: 1, Status: model.FixedPriceStatusOnSale,
		},
		remaining: 87,
	}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})
	w := ut.PerformRequest(eng, http.MethodGet, "/api/v1/fixed-price/items/7001", nil)
	resp := w.Result()
	require.Equal(t, 200, resp.StatusCode())
	var out map[string]any
	require.NoError(t, json.Unmarshal(resp.Body(), &out))
	assert.Equal(t, "99.00", out["price"])
	assert.Equal(t, float64(87), out["remaining_stock"]) // Redis 权威优先于 DB
	assert.Equal(t, "on_sale", out["status"])
}

func TestDetailHandler_NotFound_404(t *testing.T) {
	uc := &fakeFixedPriceUsecase{itemErr: gorm.ErrRecordNotFound}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})
	w := ut.PerformRequest(eng, http.MethodGet, "/api/v1/fixed-price/items/9999", nil)
	resp := w.Result()
	assert.Equal(t, 404, resp.StatusCode())
}

func TestLiveStreamFixedPriceListHandler_Public(t *testing.T) {
	uc := &fakeFixedPriceUsecase{liveItems: []*service.LiveFixedPriceItem{{
		Item: &model.FixedPriceItem{
			ID: 7001, LiveStreamID: 1001, ProductID: 5001,
			Price: decimal.NewFromInt(99), TotalStock: 100, RemainingStock: 87,
			MaxPerUser: 1, Status: model.FixedPriceStatusOnSale,
		},
		RemainingStock: 87,
	}}}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})

	w := ut.PerformRequest(eng, http.MethodGet, "/api/v1/live-streams/1001/fixed-price/items", nil)
	resp := w.Result()
	require.Equal(t, 200, resp.StatusCode())
	var out map[string][]map[string]any
	require.NoError(t, json.Unmarshal(resp.Body(), &out))
	require.Len(t, out["items"], 1)
	assert.Equal(t, float64(7001), out["items"][0]["id"])
	assert.Equal(t, "99.00", out["items"][0]["price"])
	assert.Equal(t, float64(87), out["items"][0]["remaining_stock"])
	require.Len(t, uc.liveItemReqs, 1)
	assert.Equal(t, int64(1001), uc.liveItemReqs[0].LiveStreamID)
}

func TestAdminLiveStreamFixedPriceListHandler_ReturnsAllStatuses(t *testing.T) {
	now := time.Now()
	uc := &fakeFixedPriceUsecase{adminItems: []*service.LiveFixedPriceItem{
		{
			Item: &model.FixedPriceItem{
				ID: 7001, LiveStreamID: 1001, ProductID: 5001,
				Price: decimal.NewFromInt(99), TotalStock: 100, RemainingStock: 87,
				MaxPerUser: 1, Status: model.FixedPriceStatusOnSale, CreatedAt: now,
			},
			RemainingStock: 87,
		},
		{
			Item: &model.FixedPriceItem{
				ID: 7002, LiveStreamID: 1001, ProductID: 5002,
				Price: decimal.NewFromInt(88), TotalStock: 1, RemainingStock: 0,
				MaxPerUser: 1, Status: model.FixedPriceStatusSoldOut, CreatedAt: now,
			},
			RemainingStock: 0,
		},
		{
			Item: &model.FixedPriceItem{
				ID: 7003, LiveStreamID: 1001, ProductID: 5003,
				Price: decimal.NewFromInt(77), TotalStock: 5, RemainingStock: 2,
				MaxPerUser: 1, Status: model.FixedPriceStatusOffline, CreatedAt: now,
			},
			RemainingStock: 2,
		},
	}}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})

	w := ut.PerformRequest(eng, http.MethodGet, "/api/v1/admin/live-streams/1001/fixed-price/items", nil,
		ut.Header{Key: "X-User-ID", Value: "100"})
	resp := w.Result()
	require.Equal(t, 200, resp.StatusCode())
	var out map[string]any
	require.NoError(t, json.Unmarshal(resp.Body(), &out))
	items := out["items"].([]any)
	require.Len(t, items, 3)
	assert.Equal(t, "on_sale", items[0].(map[string]any)["status"])
	assert.Equal(t, "sold_out", items[1].(map[string]any)["status"])
	assert.Equal(t, "offline", items[2].(map[string]any)["status"])
	// total 字段存在
	assert.Equal(t, float64(3), out["total"])
	// created_at 字段存在且非空
	assert.NotEmpty(t, items[0].(map[string]any)["created_at"])
	require.Len(t, uc.adminItemReqs, 1)
	assert.Equal(t, int64(1001), uc.adminItemReqs[0].LiveStreamID)
}

func TestAdminLiveStreamFixedPriceListHandler_IncludesProductTitle(t *testing.T) {
	uc := &fakeFixedPriceUsecase{adminItems: []*service.LiveFixedPriceItem{
		{
			Item: &model.FixedPriceItem{
				ID: 7001, LiveStreamID: 1001, ProductID: 5001,
				Price: decimal.NewFromInt(99), TotalStock: 100, RemainingStock: 87,
				MaxPerUser: 1, Status: model.FixedPriceStatusOnSale,
			},
			RemainingStock: 87,
		},
	}}
	pc := &fakeProductClient{batchOut: map[int64]client.ProductSummary{
		5001: {ID: 5001, Name: "翡翠手镯"},
	}}
	eng := newFixedPriceTestServerWithProductClient(uc, &fakeFPBalanceProvider{}, pc)

	w := ut.PerformRequest(eng, http.MethodGet, "/api/v1/admin/live-streams/1001/fixed-price/items", nil,
		ut.Header{Key: "X-User-ID", Value: "100"})
	resp := w.Result()
	require.Equal(t, 200, resp.StatusCode())
	var out map[string]any
	require.NoError(t, json.Unmarshal(resp.Body(), &out))
	items := out["items"].([]any)
	require.Len(t, items, 1)
	assert.Equal(t, "翡翠手镯", items[0].(map[string]any)["product_title"])
}

func TestListHandler_ReturnsFullFields(t *testing.T) {
	uc := &fakeFixedPriceUsecase{listResult: &model.FixedPriceItem{
		ID: 7001, LiveStreamID: 1001, ProductID: 5001,
		Price: decimal.NewFromInt(99), TotalStock: 100, RemainingStock: 100,
		MaxPerUser: 1, Status: model.FixedPriceStatusOnSale,
	}}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})

	body, _ := json.Marshal(map[string]any{
		"live_stream_id": 1001,
		"product_id":     5001,
		"price":          "99.00",
		"total_stock":    100,
	})
	w := ut.PerformRequest(eng, http.MethodPost, "/api/v1/fixed-price/items", &ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "X-User-ID", Value: "100"},
		ut.Header{Key: "X-User-Role", Value: "merchant"})
	resp := w.Result()
	require.Equal(t, 200, resp.StatusCode())
	var out map[string]any
	require.NoError(t, json.Unmarshal(resp.Body(), &out))
	assert.Equal(t, float64(7001), out["id"])
	assert.Equal(t, "on_sale", out["status"])
	assert.Equal(t, float64(100), out["remaining_stock"])
	assert.Equal(t, float64(5001), out["product_id"])
	assert.Equal(t, "99.00", out["price"])
	assert.Equal(t, float64(100), out["total_stock"])
}

func TestMyPurchaseHandler_Bought(t *testing.T) {
	uc := &fakeFixedPriceUsecase{myPurchase: &model.FixedPricePurchase{
		ID: 88001, ItemID: 7001, UserID: 100, Price: decimal.NewFromInt(99),
	}}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})
	w := ut.PerformRequest(eng, http.MethodGet, "/api/v1/fixed-price/items/7001/my-purchase", nil,
		ut.Header{Key: "X-User-ID", Value: "100"})
	resp := w.Result()
	require.Equal(t, 200, resp.StatusCode())
	var out map[string]any
	require.NoError(t, json.Unmarshal(resp.Body(), &out))
	assert.Equal(t, true, out["i_bought"])
	assert.Equal(t, float64(88001), out["order_id"])
}

func TestMyPurchaseHandler_NotBought(t *testing.T) {
	uc := &fakeFixedPriceUsecase{myPurchaseErr: gorm.ErrRecordNotFound}
	eng := newFixedPriceTestServer(uc, &fakeFPBalanceProvider{})
	w := ut.PerformRequest(eng, http.MethodGet, "/api/v1/fixed-price/items/7001/my-purchase", nil,
		ut.Header{Key: "X-User-ID", Value: "100"})
	resp := w.Result()
	require.Equal(t, 200, resp.StatusCode())
	var out map[string]any
	require.NoError(t, json.Unmarshal(resp.Body(), &out))
	assert.Equal(t, false, out["i_bought"])
}

// 编译期校验：真实 *service.FixedPriceService 满足 FixedPriceUsecase。
var _ FixedPriceUsecase = (*service.FixedPriceService)(nil)

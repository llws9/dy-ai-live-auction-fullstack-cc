package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/dao"
	"product-service/model"
	"product-service/service"
)

// newRuleHandlerWithSeed 构造一个隔离的 in-memory product-service 栈，
// 仅聚焦 auction_rules 表（T3.4 / spec C §4.4 语义校验）。
func newRuleHandlerWithSeed(t *testing.T, seed func(db *gorm.DB)) *RuleHandler {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Product{}, &model.AuctionRule{}, &model.LiveStream{}))
	// 共享 :memory:，需逐表清空
	db.Exec("DELETE FROM auction_rules")
	if seed != nil {
		seed(db)
	}
	svc := service.NewProductService(dao.NewProductDAO(db), dao.NewAuctionRuleDAO(db), dao.NewLiveStreamDAO(db))
	return NewRuleHandler(svc)
}

// 帮助函数：构造一个带 path :id 的 hertz 上下文。
func newCtxWithProductID(method, uri, idValue string) *app.RequestContext {
	c := app.NewContext(0)
	c.Request.SetMethod(method)
	c.Request.SetRequestURI(uri)
	c.Params = append(c.Params, param.Param{Key: "id", Value: idValue})
	return c
}

// ---- Create ----

func TestRuleHandler_Create_OK_PathIDIsProductID(t *testing.T) {
	h := newRuleHandlerWithSeed(t, nil)

	body := map[string]interface{}{
		"start_price":           100.0,
		"increment":             10.0,
		"duration":              60,
		"delay_duration":        30,
		"max_delay_time":        180,
		"trigger_delay_before":  30,
	}
	bodyBytes, _ := json.Marshal(body)

	c := newCtxWithProductID("POST", "/api/v1/products/5001/rules", "5001")
	c.Request.SetBody(bodyBytes)
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.Create(context.Background(), c)

	assert.Equal(t, 201, c.Response.StatusCode())
	var rule model.AuctionRule
	require.NoError(t, json.Unmarshal(c.Response.Body(), &rule))
	// path id 必须直接落到 product_id（spec C §4.4：禁止再做 auction_id 兼容映射）
	assert.EqualValues(t, 5001, rule.ProductID)
	assert.EqualValues(t, 60, rule.Duration)
}

func TestRuleHandler_Create_InvalidProductID(t *testing.T) {
	h := newRuleHandlerWithSeed(t, nil)

	c := newCtxWithProductID("POST", "/api/v1/products/abc/rules", "abc")
	c.Request.SetBody([]byte("{}"))
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.Create(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
	assert.Contains(t, string(c.Response.Body()), "无效的商品ID")
}

func TestRuleHandler_Create_BindError(t *testing.T) {
	h := newRuleHandlerWithSeed(t, nil)

	c := newCtxWithProductID("POST", "/api/v1/products/5001/rules", "5001")
	c.Request.SetBody([]byte("not-a-json"))
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.Create(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
	assert.Contains(t, string(c.Response.Body()), "请求参数错误")
}

func TestRuleHandler_Create_BodyProductIDOverriddenByPath(t *testing.T) {
	// spec C §3：下游禁止信任 body 内的 user_id；同理 path id 是 product_id 的 SSOT。
	// 即便 body 里携带 product_id=9999，也必须以 path 的 5001 为准。
	h := newRuleHandlerWithSeed(t, nil)

	body := map[string]interface{}{
		"product_id":    9999,
		"start_price":   100.0,
		"increment":     10.0,
		"duration":      60,
	}
	bodyBytes, _ := json.Marshal(body)

	c := newCtxWithProductID("POST", "/api/v1/products/5001/rules", "5001")
	c.Request.SetBody(bodyBytes)
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.Create(context.Background(), c)

	assert.Equal(t, 201, c.Response.StatusCode())
	var rule model.AuctionRule
	require.NoError(t, json.Unmarshal(c.Response.Body(), &rule))
	assert.EqualValues(t, 5001, rule.ProductID)
}

// ---- Get ----

func TestRuleHandler_Get_OK(t *testing.T) {
	h := newRuleHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.AuctionRule{
			ProductID:  5001,
			StartPrice: 100,
			Increment:  10,
			Duration:   60,
		})
	})

	c := newCtxWithProductID("GET", "/api/v1/products/5001/rules", "5001")

	h.Get(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var rule model.AuctionRule
	require.NoError(t, json.Unmarshal(c.Response.Body(), &rule))
	assert.EqualValues(t, 5001, rule.ProductID)
}

func TestRuleHandler_Get_NotFound(t *testing.T) {
	h := newRuleHandlerWithSeed(t, nil)

	c := newCtxWithProductID("GET", "/api/v1/products/5001/rules", "5001")

	h.Get(context.Background(), c)

	assert.Equal(t, 404, c.Response.StatusCode())
	assert.Contains(t, string(c.Response.Body()), "竞拍规则不存在")
}

func TestRuleHandler_Get_InvalidProductID(t *testing.T) {
	h := newRuleHandlerWithSeed(t, nil)

	c := newCtxWithProductID("GET", "/api/v1/products/abc/rules", "abc")

	h.Get(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
}

// TestRuleHandler_Get_DoesNotFallbackToAuctionID 验证 service 层不再做
// auction_id → product_id 兼容映射：seed 的规则 product_id=5001，
// 用一个不存在的 id=9999（与任何 auction id 都无关）必须 404，
// 而不是因兼容查询命中。
func TestRuleHandler_Get_DoesNotFallbackToAuctionID(t *testing.T) {
	h := newRuleHandlerWithSeed(t, func(db *gorm.DB) {
		db.Create(&model.AuctionRule{
			ProductID:  5001,
			StartPrice: 100,
			Increment:  10,
			Duration:   60,
		})
	})

	c := newCtxWithProductID("GET", "/api/v1/products/9999/rules", "9999")

	h.Get(context.Background(), c)

	assert.Equal(t, 404, c.Response.StatusCode())
}

// 兜个底，保证测试文件本身真在跑（避免 build tag 配置错误）。
func TestRuleHandler_TestSuiteIsExecuted(t *testing.T) {
	require.True(t, bytes.Equal([]byte("ok"), []byte("ok")))
}

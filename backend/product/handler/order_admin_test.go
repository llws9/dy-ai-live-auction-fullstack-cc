package handler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/dao"
	"product-service/model"
	"product-service/service"
)

// newAdminOrderHandlerWithSeed 在独立的 sqlite 内存库上构造完整 product-service 栈，
// 用于覆盖 admin 订单 handler 的真实 DAO/JOIN 行为。
func newAdminOrderHandlerWithSeed(t *testing.T, seed func(db *gorm.DB)) *OrderHandler {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Order{}, &model.Product{}))
	// shared cache 在进程内复用，先清表再 seed。
	db.Exec("DELETE FROM orders")
	db.Exec("DELETE FROM products")
	if seed != nil {
		seed(db)
	}
	svc := service.NewOrderService(dao.NewOrderDAO(db), nil)
	svc.SetAdminOrderDAO(dao.NewOrderAdminDAO(db))
	return NewOrderHandler(svc)
}

// TestOrderHandler_AdminList_NoXUserID admin 列表不再依赖 X-User-ID。
// 这是 T2 的核心断言：管理员能拿到全量订单（不被 winner_id 过滤）。
func TestOrderHandler_AdminList_NoXUserID(t *testing.T) {
	h := newAdminOrderHandlerWithSeed(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.Product{ID: 11, Name: "茶杯", Images: model.JSONArray{"https://cdn/a.jpg", "https://cdn/b.jpg"}}).Error)
		require.NoError(t, db.Create(&model.Product{ID: 12, Name: "茶壶", Images: model.JSONArray{"https://cdn/c.jpg"}}).Error)
		require.NoError(t, db.Create(&model.Order{ID: 101, AuctionID: 201, ProductID: 11, WinnerID: 901, FinalPrice: decimal.NewFromInt(100), Status: model.OrderStatusPending}).Error)
		require.NoError(t, db.Create(&model.Order{ID: 102, AuctionID: 202, ProductID: 12, WinnerID: 902, FinalPrice: decimal.NewFromInt(200), Status: model.OrderStatusPaid}).Error)
		require.NoError(t, db.Create(&model.Order{ID: 103, AuctionID: 203, ProductID: 11, WinnerID: 901, FinalPrice: decimal.NewFromInt(300), Status: model.OrderStatusShipped}).Error)
	})

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/admin/orders?page=1&page_size=20")
	c.Request.Header.Set("X-User-Role", "admin")
	// 不带 X-User-ID

	h.AdminList(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			List     []map[string]interface{} `json:"list"`
			Total    int64                    `json:"total"`
			Page     int                      `json:"page"`
			PageSize int                      `json:"page_size"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.Equal(t, 200, body.Code)
	assert.Equal(t, "success", body.Message)
	assert.EqualValues(t, 3, body.Data.Total)
	assert.Len(t, body.Data.List, 3)
}

// TestOrderHandler_AdminList_StatusFilter 验证 status 筛选生效。
func TestOrderHandler_AdminList_StatusFilter(t *testing.T) {
	h := newAdminOrderHandlerWithSeed(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.Product{ID: 11, Name: "茶杯", Images: model.JSONArray{"https://cdn/a.jpg"}}).Error)
		require.NoError(t, db.Create(&model.Order{ID: 101, AuctionID: 201, ProductID: 11, WinnerID: 901, FinalPrice: decimal.NewFromInt(100), Status: model.OrderStatusPending}).Error)
		require.NoError(t, db.Create(&model.Order{ID: 102, AuctionID: 202, ProductID: 11, WinnerID: 902, FinalPrice: decimal.NewFromInt(200), Status: model.OrderStatusPaid}).Error)
		require.NoError(t, db.Create(&model.Order{ID: 103, AuctionID: 203, ProductID: 11, WinnerID: 901, FinalPrice: decimal.NewFromInt(300), Status: model.OrderStatusPaid}).Error)
	})

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/admin/orders?status=1")
	c.Request.Header.Set("X-User-Role", "admin")
	h.AdminList(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body struct {
		Data struct {
			List  []map[string]interface{} `json:"list"`
			Total int64                    `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.EqualValues(t, 2, body.Data.Total)
	for _, o := range body.Data.List {
		assert.EqualValues(t, 1, o["status"])
	}
}

// TestOrderHandler_AdminList_UserIDFilter admin 可按 user_id（=winner_id）筛某用户订单。
func TestOrderHandler_AdminList_UserIDFilter(t *testing.T) {
	h := newAdminOrderHandlerWithSeed(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.Product{ID: 11, Name: "茶杯", Images: model.JSONArray{"https://cdn/a.jpg"}}).Error)
		require.NoError(t, db.Create(&model.Order{ID: 101, AuctionID: 201, ProductID: 11, WinnerID: 901, FinalPrice: decimal.NewFromInt(100), Status: model.OrderStatusPending}).Error)
		require.NoError(t, db.Create(&model.Order{ID: 102, AuctionID: 202, ProductID: 11, WinnerID: 902, FinalPrice: decimal.NewFromInt(200), Status: model.OrderStatusPaid}).Error)
		require.NoError(t, db.Create(&model.Order{ID: 103, AuctionID: 203, ProductID: 11, WinnerID: 901, FinalPrice: decimal.NewFromInt(300), Status: model.OrderStatusShipped}).Error)
	})

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/admin/orders?user_id=901")
	c.Request.Header.Set("X-User-Role", "admin")
	h.AdminList(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body struct {
		Data struct {
			List  []map[string]interface{} `json:"list"`
			Total int64                    `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.EqualValues(t, 2, body.Data.Total)
	for _, o := range body.Data.List {
		assert.EqualValues(t, 901, o["winner_id"])
		assert.EqualValues(t, 901, o["user_id"]) // 前端 fallback 兼容字段
	}
}

func TestOrderHandler_AdminList_RequiresAdminRole(t *testing.T) {
	h := newAdminOrderHandlerWithSeed(t, nil)

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/admin/orders")

	h.AdminList(context.Background(), c)

	assert.Equal(t, 403, c.Response.StatusCode())
}

func TestOrderHandler_AdminList_ClampsPageSize(t *testing.T) {
	h := newAdminOrderHandlerWithSeed(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.Product{ID: 11, Name: "茶杯"}).Error)
		for i := int64(1); i <= 3; i++ {
			require.NoError(t, db.Create(&model.Order{ID: 100 + i, AuctionID: 200 + i, ProductID: 11, WinnerID: 900 + i, FinalPrice: decimal.NewFromInt(100)}).Error)
		}
	})

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/admin/orders?page=1&page_size=10000")
	c.Request.Header.Set("X-User-Role", "admin")

	h.AdminList(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body struct {
		Data struct {
			PageSize int `json:"page_size"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.Equal(t, 100, body.Data.PageSize)
}

func TestOrderHandler_AdminList_RejectsInvalidFilters(t *testing.T) {
	h := newAdminOrderHandlerWithSeed(t, nil)

	tests := []string{
		"/api/v1/admin/orders?status=abc",
		"/api/v1/admin/orders?status=99",
		"/api/v1/admin/orders?user_id=abc",
		"/api/v1/admin/orders?user_id=-1",
	}
	for _, uri := range tests {
		t.Run(uri, func(t *testing.T) {
			c := app.NewContext(0)
			c.Request.SetMethod("GET")
			c.Request.SetRequestURI(uri)
			c.Request.Header.Set("X-User-Role", "admin")

			h.AdminList(context.Background(), c)

			assert.Equal(t, 400, c.Response.StatusCode())
		})
	}
}

func TestOrderHandler_AdminList_DoesNotLeakInternalError(t *testing.T) {
	h := newAdminOrderHandlerWithSeed(t, nil)
	h.orderService.SetAdminOrderDAO(nil)

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/admin/orders")
	c.Request.Header.Set("X-User-Role", "admin")

	h.AdminList(context.Background(), c)

	assert.Equal(t, 500, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.Equal(t, float64(500), body["code"])
	assert.Equal(t, "获取订单列表失败", body["message"])
	assert.NotContains(t, body["message"], "admin order DAO")
}

// TestOrderHandler_AdminGet 单条返回 product_name 与首图。
func TestOrderHandler_AdminGet(t *testing.T) {
	h := newAdminOrderHandlerWithSeed(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.Product{ID: 11, Name: "茶杯", Images: model.JSONArray{"https://cdn/first.jpg", "https://cdn/second.jpg"}}).Error)
		require.NoError(t, db.Create(&model.Order{ID: 101, AuctionID: 201, ProductID: 11, WinnerID: 901, FinalPrice: decimal.NewFromInt(100), Status: model.OrderStatusPending}).Error)
	})

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/admin/orders/101")
	c.Request.Header.Set("X-User-Role", "admin")
	c.Params = append(c.Params, param.Param{Key: "id", Value: "101"})

	h.AdminGet(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	var body struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.Equal(t, 200, body.Code)
	assert.EqualValues(t, 101, body.Data["id"])
	assert.EqualValues(t, 901, body.Data["winner_id"])
	assert.EqualValues(t, 901, body.Data["user_id"])
	assert.Equal(t, "茶杯", body.Data["product_name"])
	assert.Equal(t, "https://cdn/first.jpg", body.Data["product_image"])
}

func TestOrderHandler_AdminGet_RequiresAdminRole(t *testing.T) {
	h := newAdminOrderHandlerWithSeed(t, nil)

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/admin/orders/101")
	c.Params = append(c.Params, param.Param{Key: "id", Value: "101"})

	h.AdminGet(context.Background(), c)

	assert.Equal(t, 403, c.Response.StatusCode())
}

func TestOrderHandler_AdminGet_DoesNotLeakInternalError(t *testing.T) {
	h := newAdminOrderHandlerWithSeed(t, nil)
	h.orderService.SetAdminOrderDAO(nil)

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/admin/orders/101")
	c.Request.Header.Set("X-User-Role", "admin")
	c.Params = append(c.Params, param.Param{Key: "id", Value: "101"})

	h.AdminGet(context.Background(), c)

	assert.Equal(t, 500, c.Response.StatusCode())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.Equal(t, "获取订单详情失败", body["message"])
	assert.NotContains(t, body["message"], "admin order DAO")
}

// TestOrderHandler_AdminGet_NotFound 不存在订单返回 404。
func TestOrderHandler_AdminGet_NotFound(t *testing.T) {
	h := newAdminOrderHandlerWithSeed(t, nil)

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/admin/orders/999")
	c.Request.Header.Set("X-User-Role", "admin")
	c.Params = append(c.Params, param.Param{Key: "id", Value: "999"})

	h.AdminGet(context.Background(), c)

	assert.Equal(t, 404, c.Response.StatusCode())
}

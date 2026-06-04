package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/dao"
	"product-service/model"
	"product-service/service"
)

// TestOrderHandler_GetUserHistory_AuthContract 验证 spec C / F-C3 安全契约：
//   - 未携带 X-User-ID（即 Gateway JWT 中间件未放行） → 401；
//   - X-User-ID 解析失败 / 非正整数 → 401；
//   - X-User-ID 合法 → 200，且仅以 header 用户身份查询，query user_id 不再生效。
func TestOrderHandler_GetUserHistory_AuthContract(t *testing.T) {
	// historyDAO=nil 时 OrderService.GetUserHistory 返回空集合，足够覆盖 handler 鉴权分支。
	h := NewOrderHandler(service.NewOrderService(nil, nil))

	t.Run("missing X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/history")

		h.GetUserHistory(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
		var body map[string]interface{}
		assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
		assert.EqualValues(t, 401, body["code"])
	})

	t.Run("non-numeric X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/history")
		c.Request.Header.Set("X-User-ID", "not-a-number")

		h.GetUserHistory(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("non-positive X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/history")
		c.Request.Header.Set("X-User-ID", "0")

		h.GetUserHistory(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("valid X-User-ID returns 200 and query user_id is ignored", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		// query 中故意塞一个不同的 user_id：必须被忽略，不能借此越权。
		c.Request.SetRequestURI("/api/v1/orders/history?user_id=999&page=1&page_size=20")
		c.Request.Header.Set("X-User-ID", "123")

		h.GetUserHistory(context.Background(), c)

		assert.Equal(t, 200, c.Response.StatusCode())
		var body map[string]interface{}
		assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
		assert.EqualValues(t, 0, body["total"])
		assert.EqualValues(t, 1, body["page"])
		assert.EqualValues(t, 20, body["page_size"])
		assert.NotNil(t, body["list"])
	})
}

func TestOrderHandler_Summary_XUserIDContract(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&model.Order{}))

	h := NewOrderHandler(service.NewOrderService(dao.NewOrderDAO(db), nil))

	t.Run("missing X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/summary")

		h.Summary(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
		var body map[string]interface{}
		assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
		assert.EqualValues(t, 401, body["code"])
	})

	t.Run("non-numeric X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/summary")
		c.Request.Header.Set("X-User-ID", "not-a-number")

		h.Summary(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("non-positive X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/summary")
		c.Request.Header.Set("X-User-ID", "0")

		h.Summary(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("valid X-User-ID returns header user's summary and ignores query user_id", func(t *testing.T) {
		assert.NoError(t, db.Exec("DELETE FROM orders").Error)
		ctx := context.Background()
		svc := service.NewOrderService(dao.NewOrderDAO(db), nil)
		_, err := svc.CreateOrder(ctx, 1, 1, 123, decimal.NewFromInt(100))
		assert.NoError(t, err)
		_, err = svc.CreateOrder(ctx, 2, 1, 999, decimal.NewFromInt(200))
		assert.NoError(t, err)
		_, err = svc.CreateOrder(ctx, 3, 1, 999, decimal.NewFromInt(300))
		assert.NoError(t, err)

		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		// query 中伪造不同 user_id：Summary 必须忽略 query，只信 Gateway 透传的 X-User-ID。
		c.Request.SetRequestURI("/api/v1/orders/summary?user_id=999")
		c.Request.Header.Set("X-User-ID", "123")

		h.Summary(context.Background(), c)

		assert.Equal(t, 200, c.Response.StatusCode())
		var body struct {
			Code int `json:"code"`
			Data struct {
				PendingPayment int64 `json:"pendingPayment"`
				WonNotPaid     int64 `json:"wonNotPaid"`
			} `json:"data"`
		}
		assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
		assert.Equal(t, 0, body.Code)
		assert.Equal(t, int64(1), body.Data.PendingPayment)
		assert.Equal(t, int64(1), body.Data.WonNotPaid)
	})
}

// TestOrderHandler_List_AuthContract 验证 GET /orders 的安全契约（与 GetUserHistory 对齐）：
//   - 未携带 X-User-ID → 401；
//   - X-User-ID 非法 → 401；
//   - 合法 X-User-ID → 200，query user_id 被忽略。
func TestOrderHandler_List_AuthContract(t *testing.T) {
	h := NewOrderHandler(service.NewOrderService(nil, nil))

	t.Run("missing X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders")

		h.List(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("non-numeric X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders")
		c.Request.Header.Set("X-User-ID", "abc")

		h.List(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("valid X-User-ID returns 200", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders?page=1&page_size=20")
		c.Request.Header.Set("X-User-ID", "42")

		h.List(context.Background(), c)

		assert.Equal(t, 200, c.Response.StatusCode())
	})
}

type stubSummaryGetter struct {
	err error
}

func (s *stubSummaryGetter) GetSummary(_ context.Context, _ int64) (*model.OrderSummaryResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &model.OrderSummaryResponse{PendingPayment: 0, WonNotPaid: 0}, nil
}

func TestOrderHandler_Summary_ErrorNoLeak(t *testing.T) {
	internalErr := errors.New("db connection refused: password=secret")
	h := &OrderHandler{
		orderService:   service.NewOrderService(nil, nil),
		summaryService: &stubSummaryGetter{err: internalErr},
	}

	writer := &testLogWriter{}
	log.SetOutput(writer)
	defer log.SetOutput(os.Stderr)

	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/orders/summary")
	c.Request.Header.Set("X-User-ID", "1")

	h.Summary(context.Background(), c)

	assert.Equal(t, 500, c.Response.StatusCode())
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.EqualValues(t, 500, body["code"])
	assert.Equal(t, "获取订单汇总失败", body["message"])
	assert.NotContains(t, body["message"], "secret")
	assert.NotContains(t, body["message"], "db connection")
	assert.Contains(t, writer.String(), "db connection refused")
}

type testLogWriter struct {
	data []byte
}

func (w *testLogWriter) Write(p []byte) (n int, err error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

func (w *testLogWriter) String() string {
	return string(w.data)
}

func TestOrderHandler_Get_AuthContract(t *testing.T) {
	h := NewOrderHandler(service.NewOrderService(nil, nil))

	t.Run("missing X-User-ID returns 401 before reading order", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/123")

		h.Get(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("non-numeric X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/orders/123")
		c.Request.Header.Set("X-User-ID", "abc")

		h.Get(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})
}

func TestOrderHandler_Pay_AuthContract(t *testing.T) {
	h := NewOrderHandler(service.NewOrderService(nil, nil))

	t.Run("missing X-User-ID returns 401 before paying order", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("POST")
		c.Request.SetRequestURI("/api/v1/orders/123/pay")

		h.Pay(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})

	t.Run("non-positive X-User-ID returns 401", func(t *testing.T) {
		c := app.NewContext(0)
		c.Request.SetMethod("POST")
		c.Request.SetRequestURI("/api/v1/orders/123/pay")
		c.Request.Header.Set("X-User-ID", "0")

		h.Pay(context.Background(), c)

		assert.Equal(t, 401, c.Response.StatusCode())
	})
}

func TestOrderHandler_Ship_AdminRoleRejected(t *testing.T) {
	h := NewOrderHandler(service.NewOrderService(nil, nil))
	c := app.NewContext(0)
	c.Request.SetMethod("PUT")
	c.Request.SetRequestURI("/api/v1/orders/123/ship")
	c.Request.Header.Set("X-User-ID", "2001")
	c.Request.Header.Set("X-User-Role", "admin")

	h.Ship(context.Background(), c)

	assert.Equal(t, 403, c.Response.StatusCode())
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.EqualValues(t, 403, body["code"])
}

func TestOrderHandler_Ship_MerchantOtherSellerRejected(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&model.Order{}))
	assert.NoError(t, db.Exec("DELETE FROM orders").Error)
	seller := int64(1001)
	order := &model.Order{ID: 123, AuctionID: 1, ProductID: 1, SellerID: &seller, WinnerID: 3001, FinalPrice: decimal.NewFromInt(100), Status: model.OrderStatusPaid}
	assert.NoError(t, db.Create(order).Error)
	h := NewOrderHandler(service.NewOrderService(dao.NewOrderDAO(db), nil))

	c := app.NewContext(0)
	c.Request.SetMethod("PUT")
	c.Request.SetRequestURI("/api/v1/orders/123/ship")
	c.Request.Header.Set("X-User-ID", "1002")
	c.Request.Header.Set("X-User-Role", "merchant")

	h.Ship(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
	var saved model.Order
	assert.NoError(t, db.First(&saved, 123).Error)
	assert.Equal(t, model.OrderStatusPaid, saved.Status)
}

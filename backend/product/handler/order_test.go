package handler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
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
		// 期望 items 是空数组，total=0；page/page_size 透传。
		assert.EqualValues(t, 0, body["total"])
		assert.EqualValues(t, 1, body["page"])
		assert.EqualValues(t, 20, body["page_size"])
		assert.NotNil(t, body["items"])
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
		_, err := svc.CreateOrder(ctx, 1, 1, 123, 100.0)
		assert.NoError(t, err)
		_, err = svc.CreateOrder(ctx, 2, 1, 999, 200.0)
		assert.NoError(t, err)
		_, err = svc.CreateOrder(ctx, 3, 1, 999, 300.0)
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

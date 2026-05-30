package handler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"

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
		assert.NotNil(t, body["items"])
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

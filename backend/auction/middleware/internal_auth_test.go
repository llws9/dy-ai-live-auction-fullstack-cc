package middleware

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

// runInternalAuth 在受控的 RequestContext 上跑中间件，返回 status 与是否被 abort。
func runInternalAuth(expected, clientToken string) (status int, aborted bool) {
	ctx := context.Background()
	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/internal/ping")
	if clientToken != "" {
		c.Request.Header.Set("X-Internal-Token", clientToken)
	}
	mw := InternalAuthMiddleware(expected)
	c.SetHandlers([]app.HandlerFunc{
		mw,
		func(ctx context.Context, c *app.RequestContext) {
			c.JSON(200, map[string]string{"ok": "yes"})
		},
	})
	c.Next(ctx)
	return c.Response.StatusCode(), c.IsAborted()
}

// TestInternalAuthMiddleware 验证 X-Internal-Token 鉴权（T3.3 / spec B §4.1）：
//   - 未配置 token（空）→ 服务端拒绝（500），避免裸奔
//   - token 不匹配 → 401
//   - 缺少 client header → 401
//   - token 匹配 → 放行（不 abort），下游写 200
func TestInternalAuthMiddleware(t *testing.T) {
	t.Run("rejects when server token is empty", func(t *testing.T) {
		status, aborted := runInternalAuth("", "anything")
		assert.Equal(t, 500, status)
		assert.True(t, aborted)
	})

	t.Run("rejects when token mismatch", func(t *testing.T) {
		status, aborted := runInternalAuth("expected", "wrong")
		assert.Equal(t, 401, status)
		assert.True(t, aborted)
	})

	t.Run("rejects when client header missing", func(t *testing.T) {
		status, aborted := runInternalAuth("expected", "")
		assert.Equal(t, 401, status)
		assert.True(t, aborted)
	})

	t.Run("passes when token matches", func(t *testing.T) {
		status, aborted := runInternalAuth("expected", "expected")
		assert.Equal(t, 200, status)
		assert.False(t, aborted)
	})
}

package middleware

import (
	"context"
	"crypto/subtle"

	"github.com/cloudwego/hertz/pkg/app"
)

// InternalAuthMiddleware 校验 X-Internal-Token Header 与服务端预置的 expected token 一致，
// 用于 /internal/* 内部接口（T3.3 / spec B §4.1）。
//
// 安全约束：
//   - 服务端 expected 为空时直接 500：避免环境变量缺失造成内部接口裸奔。
//   - 使用 constant-time 比较，防止 timing oracle。
func InternalAuthMiddleware(expected string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if expected == "" {
			c.JSON(500, map[string]interface{}{
				"code":    500,
				"message": "internal auth not configured",
			})
			c.Abort()
			return
		}
		got := string(c.GetHeader("X-Internal-Token"))
		if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
			c.JSON(401, map[string]interface{}{
				"code":    401,
				"message": "unauthorized internal call",
			})
			c.Abort()
			return
		}
		c.Next(ctx)
	}
}

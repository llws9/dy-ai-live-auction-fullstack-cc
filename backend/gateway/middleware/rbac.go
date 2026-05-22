package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// RBACMiddleware RBAC权限中间件
func RBACMiddleware(requiredRole int) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		userRole := c.GetInt("user_role") // 从 JWT 解析

		if userRole < requiredRole {
			c.JSON(403, map[string]interface{}{
				"code":    403,
				"message": "权限不足",
			})
			c.Abort()
			return
		}

		c.Next(ctx)
	}
}

// RequireRole 要求特定角色
func RequireRole(role int) app.HandlerFunc {
	return RBACMiddleware(role)
}

// RequireStreamer 要求主播权限
func RequireStreamer() app.HandlerFunc {
	return RBACMiddleware(1) // 1 = 主播
}

// RequireAdmin 要求管理员权限
func RequireAdmin() app.HandlerFunc {
	return RBACMiddleware(2) // 2 = 管理员
}

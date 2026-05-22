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

// CheckOwnership 检查资源所有权
func CheckOwnership(getOwnerID func(ctx context.Context, c *app.RequestContext) (int64, error)) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		userRole := c.GetInt("user_role")
		userID := c.GetInt64("user_id")

		// 管理员可以操作所有资源
		if userRole >= 2 {
			c.Next(ctx)
			return
		}

		// 主播只能操作自己的资源
		if userRole == 1 {
			ownerID, err := getOwnerID(ctx, c)
			if err != nil {
				c.JSON(500, map[string]interface{}{
					"code":    500,
					"message": "获取资源归属失败",
				})
				c.Abort()
				return
			}

			if ownerID != userID {
				c.JSON(403, map[string]interface{}{
					"code":    403,
					"message": "无权操作他人资源",
				})
				c.Abort()
				return
			}
		}

		c.Next(ctx)
	}
}

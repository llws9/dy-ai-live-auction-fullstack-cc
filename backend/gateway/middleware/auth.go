package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// RequireMerchant 商家权限中间件
func RequireMerchant() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		userRole := c.GetInt("user_role")
		if userRole != 1 && userRole != 2 { // 商家或管理员
			c.JSON(403, map[string]interface{}{
				"code":    403,
				"message": "权限不足：需要商家或管理员权限",
			})
			c.Abort()
			return
		}
		c.Next(ctx)
	}
}

// RequireOwnership 数据所有权验证中间件
func RequireOwnership() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		userID := c.GetInt64("user_id")
		userRole := c.GetInt("user_role")

		// 管理员可以访问所有数据
		if userRole == 2 {
			c.Next(ctx)
			return
		}

		// 商家只能访问自己的数据
		// 这里需要在具体的handler中验证资源所有权
		c.Set("require_ownership_check", true)
		c.Set("current_user_id", userID)

		c.Next(ctx)
	}
}

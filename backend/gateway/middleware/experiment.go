package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"gateway-service/pkg/growthbook"
)

// ExperimentMiddleware 实验中间件
// 在 JWT 认证后注入 GrowthBook 用户属性
func ExperimentMiddleware(gbClient *growthbook.Client) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// 从 JWT 中间件获取用户信息
		userID, exists := c.Get("user_id")
		if !exists {
			// 未认证用户，使用匿名属性
			c.Set("gb_attributes", &growthbook.Attributes{
				ID: "anonymous",
			})
			c.Next(ctx)
			return
		}

		userRole, _ := c.Get("user_role")

		// 构建用户属性
		attrs := growthbook.NewBuilder(userID.(int64), userRole.(int)).Build()

		// 存储到 context
		c.Set("gb_attributes", attrs)

		c.Next(ctx)
	}
}

// GetExperimentAttributes 从 context 获取实验属性
func GetExperimentAttributes(c *app.RequestContext) *growthbook.Attributes {
	if attrs, exists := c.Get("gb_attributes"); exists {
		return attrs.(*growthbook.Attributes)
	}
	return nil
}

// RequireFeatureFlag 特性开关中间件
// 检查用户是否在某个实验的 treatment 组
func RequireFeatureFlag(gbClient *growthbook.Client, featureKey string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		attrs := GetExperimentAttributes(c)

		if attrs == nil {
			c.JSON(403, map[string]interface{}{
				"code":    403,
				"message": "无法获取用户属性",
			})
			c.Abort()
			return
		}

		// 检查特性是否开启
		if !gbClient.IsOn(featureKey, attrs) {
			c.JSON(403, map[string]interface{}{
				"code":    403,
				"message": "功能未开启或您不在此功能的测试组中",
			})
			c.Abort()
			return
		}

		c.Next(ctx)
	}
}

// FeatureValueMiddleware 特性值中间件
// 将特性值注入到 context 中供后续 handler 使用
func FeatureValueMiddleware(gbClient *growthbook.Client, featureKey string, contextKey string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		attrs := GetExperimentAttributes(c)

		if attrs != nil {
			value := gbClient.GetValue(featureKey, attrs)
			c.Set(contextKey, value)
		}

		c.Next(ctx)
	}
}
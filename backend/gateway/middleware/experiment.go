package middleware

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"gateway-service/pkg/growthbook"
)

const (
	ctxKeyGBAttrs        = "gb_attributes"
	ctxKeyGBFeatures     = "gb_features"
	ctxKeyGBFeaturesJSON = "gb_features_json"
)

// ExperimentMiddleware 在 JWT 之后运行,做两件事:
//  1. 根据 user_id/user_role 构造 GrowthBook Attributes;
//  2. 预先批量评估 KnownFeatureKeys 并缓存到 c.Set,
//     避免后续 handler/proxy 重复调用 SDK,也方便代理层注入 X-Experiment-Context。
//
// 未登录用户使用 anonymous id,仍会进行评估 (官方 SDK 会按 anonymous 哈希分桶)。
func ExperimentMiddleware(gbClient *growthbook.Client) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		attrs := buildAttrs(c)
		c.Set(ctxKeyGBAttrs, attrs)

		if gbClient != nil && gbClient.Enabled() {
			features := gbClient.EvalFeatures(ctx, attrs, growthbook.KnownFeatureKeys)
			c.Set(ctxKeyGBFeatures, features)
			if json := growthbook.SerializeFeatures(features); json != "" {
				c.Set(ctxKeyGBFeaturesJSON, json)
			}
		}

		c.Next(ctx)
	}
}

func buildAttrs(c *app.RequestContext) *growthbook.Attributes {
	if userID, exists := c.Get("user_id"); exists {
		role, _ := c.Get("user_role")
		return &growthbook.Attributes{
			UserID: toIDString(userID),
			Role:   toRoleInt(role),
		}
	}
	return &growthbook.Attributes{UserID: "anonymous"}
}

// GetExperimentAttributes 从 context 读取实验属性 (handler 使用)。
func GetExperimentAttributes(c *app.RequestContext) *growthbook.Attributes {
	if v, ok := c.Get(ctxKeyGBAttrs); ok {
		if attrs, ok := v.(*growthbook.Attributes); ok {
			return attrs
		}
	}
	return nil
}

// GetExperimentFeatures 读取中间件预评估的 feature 结果。
func GetExperimentFeatures(c *app.RequestContext) map[string]growthbook.FeatureSnapshot {
	if v, ok := c.Get(ctxKeyGBFeatures); ok {
		if f, ok := v.(map[string]growthbook.FeatureSnapshot); ok {
			return f
		}
	}
	return nil
}

// GetExperimentContextHeader 读取已序列化的 X-Experiment-Context 字符串 (proxy 使用)。
func GetExperimentContextHeader(c *app.RequestContext) string {
	if v, ok := c.Get(ctxKeyGBFeaturesJSON); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func toIDString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case int64:
		return strconv.FormatInt(x, 10)
	case int:
		return strconv.Itoa(x)
	default:
		return ""
	}
}

func toRoleInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	default:
		return 0
	}
}

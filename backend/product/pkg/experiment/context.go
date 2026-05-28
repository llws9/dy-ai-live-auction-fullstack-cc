package experiment

import (
	"encoding/json"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

// Context 实验上下文
type Context struct {
	UserID      string
	Role        int
	Features    map[string]interface{}
}

// FromHeaders 从请求头解析实验上下文
// Gateway 通过 X-Experiment-Context 头转发实验结果
func FromHeaders(c *app.RequestContext) *Context {
	ctx := &Context{
		Features: make(map[string]interface{}),
	}

	// 从 X-User-ID 获取用户ID
	userID := string(c.GetHeader("X-User-ID"))
	if userID != "" {
		ctx.UserID = userID
	}

	// 从 X-User-Role 获取用户角色
	userRole := string(c.GetHeader("X-User-Role"))
	if userRole != "" {
		role := 0
		if strings.Contains(userRole, "admin") {
			role = 2
		} else if strings.Contains(userRole, "merchant") || strings.Contains(userRole, "streamer") {
			role = 1
		}
		ctx.Role = role
	}

	// 从 X-Experiment-Context 获取实验结果
	experimentContext := string(c.GetHeader("X-Experiment-Context"))
	if experimentContext != "" {
		var features map[string]interface{}
		if err := json.Unmarshal([]byte(experimentContext), &features); err == nil {
			ctx.Features = features
		}
	}

	return ctx
}

// GetFeature 从上下文获取特性值
func (ctx *Context) GetFeature(key string) interface{} {
	return ctx.Features[key]
}

// IsFeatureOn 检查特性是否开启
func (ctx *Context) IsFeatureOn(key string) bool {
	value := ctx.GetFeature(key)
	if value == nil {
		return false
	}

	// 尝试解析为布尔值
	if v, ok := value.(bool); ok {
		return v
	}

	// 尝试解析为 map 并检查 on 字段
	if v, ok := value.(map[string]interface{}); ok {
		if on, ok := v["on"].(bool); ok {
			return on
		}
	}

	return false
}

// GetFeatureValue 获取特性值
func (ctx *Context) GetFeatureValue(key string) string {
	value := ctx.GetFeature(key)
	if value == nil {
		return ""
	}

	// 尝试解析为字符串
	if v, ok := value.(string); ok {
		return v
	}

	// 尝试解析为 map 并获取 value 字段
	if v, ok := value.(map[string]interface{}); ok {
		if val, ok := v["value"].(string); ok {
			return val
		}
	}

	return ""
}
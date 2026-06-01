package handler

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"gateway-service/middleware"
)

// ProxyHandler 代理处理器
type ProxyHandler struct {
	targetURL     string
	internalToken string
	client        *http.Client
}

// NewProxyHandler 创建代理处理器
func NewProxyHandler(targetURL string) *ProxyHandler {
	return &ProxyHandler{
		targetURL: targetURL,
		client: &http.Client{
			Timeout: 0, // 不设置超时，由客户端控制
		},
	}
}

func NewProxyHandlerWithInternalToken(targetURL, internalToken string) *ProxyHandler {
	p := NewProxyHandler(targetURL)
	p.internalToken = internalToken
	return p
}

// Forward 转发请求到后端服务
func (p *ProxyHandler) Forward(ctx context.Context, c *app.RequestContext) {
	// 构建目标 URL
	targetPath := string(c.URI().Path())
	query := string(c.URI().QueryString())
	targetURL := p.targetURL + targetPath
	if query != "" {
		targetURL += "?" + query
	}

	// 读取请求体
	var body io.Reader
	if c.Request.Body() != nil {
		body = bytes.NewReader(c.Request.Body())
	}

	// 创建新请求
	req, err := http.NewRequestWithContext(ctx, string(c.Method()), targetURL, body)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "内部服务器错误",
		})
		return
	}

	// 复制请求头
	c.Request.Header.VisitAll(func(key, value []byte) {
		keyStr := string(key)
		// 跳过某些头部
		if strings.EqualFold(keyStr, "Host") {
			return
		}
		if strings.EqualFold(keyStr, "X-Internal-Token") {
			return
		}
		req.Header.Set(keyStr, string(value))
	})

	// 添加用户信息头部（如果存在）
	if userID, exists := c.Get("user_id"); exists {
		req.Header.Set("X-User-ID", toString(userID))
	}
	if username, exists := c.Get("username"); exists {
		req.Header.Set("X-Username", toString(username))
	}
	// X-User-Role: 下游服务用其判断角色 (admin/streamer/merchant)
	if role, exists := c.Get("user_role"); exists {
		req.Header.Set("X-User-Role", toRoleString(role))
	}
	if p.internalToken != "" {
		req.Header.Set("X-Internal-Token", p.internalToken)
	}
	// X-Experiment-Context: 透传 gateway 评估的 A/B feature 结果给下游
	if expCtx := middleware.GetExperimentContextHeader(c); expCtx != "" {
		req.Header.Set("X-Experiment-Context", expCtx)
	}

	// 发送请求
	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("Failed to forward request: %v", err)
		c.JSON(502, map[string]interface{}{
			"code":    502,
			"message": "上游服务不可用",
		})
		return
	}
	defer resp.Body.Close()

	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// 复制响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "读取响应失败",
		})
		return
	}

	// 设置响应状态码和内容
	c.Response.SetStatusCode(resp.StatusCode)
	c.Response.SetBody(respBody)
}

// toString 将 interface{} 转换为字符串
func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int64:
		return strconv.FormatInt(val, 10)
	case int:
		return strconv.Itoa(val)
	case []byte:
		return string(val)
	default:
		return ""
	}
}

// toRoleString 将 user_role (int) 转为下游可理解的字符串。
// 与 auction/pkg/experiment.FromHeaders 的解析约定保持一致:
//
//	admin > merchant/streamer > 普通用户。
func toRoleString(v interface{}) string {
	role := 0
	switch x := v.(type) {
	case int:
		role = x
	case int64:
		role = int(x)
	}
	switch role {
	case 2:
		return "admin"
	case 1:
		return "streamer"
	default:
		return "user"
	}
}

// HandleWebSocket 处理 WebSocket 连接
func HandleWebSocket(ctx context.Context, c *app.RequestContext, auctionServiceURL string) {
	// WebSocket 升级处理
	// 由于 Hertz 的特殊性，这里需要特殊处理 WebSocket

	// 获取 auction_id 参数
	auctionID := c.DefaultQuery("auction_id", "0")

	// 构建目标 WebSocket URL
	wsURL := strings.Replace(auctionServiceURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL += "/api/v1/ws?auction_id=" + auctionID

	// 返回 WebSocket 连接信息
	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "WebSocket 端点",
		"data": map[string]interface{}{
			"ws_url":      wsURL,
			"auction_id":  auctionID,
			"instruction": "请直接连接后端服务的 WebSocket 端点",
		},
	})
}

// HandleTestWebSocket 处理测试平台 WS endpoint discovery
// 与 HandleWebSocket 同样模式：返回真实 WS 直连地址，由前端建连。
func HandleTestWebSocket(ctx context.Context, c *app.RequestContext, testWSBase string) {
	testID := c.Query("test_id")
	wsURL := strings.TrimRight(testWSBase, "/") + "/ws/test/progress?test_id=" + testID
	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "Test WebSocket 端点",
		"data": map[string]interface{}{
			"ws_url":      wsURL,
			"test_id":     testID,
			"instruction": "请直接连接 test-service 的 WebSocket 端点",
		},
	})
}

// ProxyWebSocket WebSocket 代理
func ProxyWebSocket(targetURL string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// 获取查询参数
		query := string(c.URI().QueryString())

		// 构建 WebSocket 目标地址
		wsURL, _ := url.Parse(targetURL)
		if query != "" {
			wsURL.RawQuery = query
		}

		// 返回连接信息
		c.JSON(200, map[string]interface{}{
			"code":    200,
			"message": "WebSocket 代理",
			"data": map[string]string{
				"target": wsURL.String(),
			},
		})
	}
}

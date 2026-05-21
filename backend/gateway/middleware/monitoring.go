package middleware

import (
	"context"
	"log"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// Logger 结构化日志中间件
func Logger() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()

		c.Next(ctx)

		latency := time.Since(start)
		method := string(c.Method())
		path := string(c.URI().Path())
		status := c.Response.StatusCode()
		clientIP := c.ClientIP()

		log.Printf("[REQUEST] method=%s path=%s status=%d latency=%s client_ip=%s",
			method, path, status, latency.String(), clientIP)
	}
}

// RequestID 请求 ID 中间件
func RequestID() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// 检查请求头中是否有 Request ID
		requestID := string(c.GetHeader("X-Request-ID"))
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// 设置响应头
		c.Header("X-Request-ID", requestID)

		// 存储到上下文
		c.Set("request_id", requestID)

		c.Next(ctx)
	}
}

// Metrics 指标收集中间件
func Metrics() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()

		c.Next(ctx)

		// 记录请求指标
		latency := time.Since(start)
		method := string(c.Method())
		path := string(c.URI().Path())
		status := c.Response.StatusCode()

		// 这里可以集成 Prometheus 客户端
		_ = latency
		_ = method
		_ = path
		_ = status
	}
}

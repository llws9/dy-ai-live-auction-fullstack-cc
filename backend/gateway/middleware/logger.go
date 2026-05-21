package middleware

import (
	"context"
	"log"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

// RequestLogger 请求日志中间件
func RequestLogger() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()

		// 处理请求
		c.Next(ctx)

		// 记录日志
		latency := time.Since(start)
		method := string(c.Method())
		path := string(c.URI().Path())
		status := c.Response.StatusCode()
		clientIP := c.ClientIP()

		log.Printf("[%s] %s %s %d %v %s",
			method,
			path,
			clientIP,
			status,
			latency,
			c.Response.Body(),
		)
	}
}

package middleware

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"gateway-service/pkg/metrics"
)

// MetricsMiddleware 指标中间件
func MetricsMiddleware(serviceName string, m *metrics.Metrics) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()

		// 处理请求
		c.Next(ctx)

		// 记录请求指标
		duration := time.Since(start).Seconds()
		method := string(c.Method())
		path := string(c.URI().Path())
		status := c.Response.StatusCode()

		m.RequestsTotal.WithLabelValues(serviceName, method, path, strconv.Itoa(status)).Inc()
		m.RequestDuration.WithLabelValues(serviceName, method, path).Observe(duration)
	}
}

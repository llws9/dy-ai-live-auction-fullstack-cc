package middleware

import (
	"context"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gateway-service/pkg/metrics"
	"github.com/cloudwego/hertz/pkg/app"
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
		path := normalizeMetricPath(string(c.URI().Path()))
		status := c.Response.StatusCode()

		m.RequestsTotal.WithLabelValues(serviceName, method, path, strconv.Itoa(status)).Inc()
		m.RequestDuration.WithLabelValues(serviceName, method, path).Observe(duration)
	}
}

func normalizeMetricPath(path string) string {
	if path == "" || path == "/" {
		return path
	}

	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if isDynamicPathSegment(segment) {
			segments[i] = ":id"
		}
	}
	return strings.Join(segments, "/")
}

func isDynamicPathSegment(segment string) bool {
	if segment == "" {
		return false
	}

	allDigits := true
	for _, r := range segment {
		if !unicode.IsDigit(r) {
			allDigits = false
			break
		}
	}
	if allDigits {
		return true
	}

	if len(segment) == 36 && strings.Count(segment, "-") == 4 {
		for _, r := range segment {
			if r == '-' || unicode.IsDigit(r) || ('a' <= r && r <= 'f') || ('A' <= r && r <= 'F') {
				continue
			}
			return false
		}
		return true
	}

	return false
}

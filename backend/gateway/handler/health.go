package handler

import (
	"context"
	"runtime"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Timestamp int64             `json:"timestamp"`
	Uptime    int64             `json:"uptime"`
	Checks    map[string]string `json:"checks,omitempty"`
}

var startTime = time.Now()

// Health 健康检查处理器
func Health(serviceName string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		c.JSON(200, HealthResponse{
			Status:    "ok",
			Service:   serviceName,
			Timestamp: time.Now().Unix(),
			Uptime:    int64(time.Since(startTime).Seconds()),
		})
	}
}

// Ready 就绪检查处理器
func Ready(serviceName string, checks map[string]func() bool) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		checkResults := make(map[string]string)
		allHealthy := true

		for name, check := range checks {
			if check() {
				checkResults[name] = "healthy"
			} else {
				checkResults[name] = "unhealthy"
				allHealthy = false
			}
		}

		status := "ok"
		statusCode := 200
		if !allHealthy {
			status = "degraded"
			statusCode = 503
		}

		c.JSON(statusCode, HealthResponse{
			Status:    status,
			Service:   serviceName,
			Timestamp: time.Now().Unix(),
			Uptime:    int64(time.Since(startTime).Seconds()),
			Checks:    checkResults,
		})
	}
}

// Metrics Prometheus 指标处理器
func Metrics(serviceName string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// 返回基本的运行时指标
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		metrics := map[string]interface{}{
			"service":           serviceName,
			"uptime_seconds":    int64(time.Since(startTime).Seconds()),
			"goroutines":        runtime.NumGoroutine(),
			"alloc_bytes":       m.Alloc,
			"total_alloc_bytes": m.TotalAlloc,
			"sys_bytes":         m.Sys,
			"num_gc":            m.NumGC,
		}

		c.JSON(200, metrics)
	}
}

package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// Health 健康检查
func Health(_ context.Context, c *app.RequestContext) {
	c.JSON(200, map[string]any{
		"status":  "ok",
		"service": "test-service",
	})
}

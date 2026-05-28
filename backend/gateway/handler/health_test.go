package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestHealthHandler_Check(t *testing.T) {
	t.Run("should return healthy status", func(t *testing.T) {
		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")

		handler := Health("gateway-service")
		handler(ctx, c)

		// Verify response
		assert.Equal(t, http.StatusOK, c.Response.StatusCode())
		assert.Contains(t, string(c.Response.Body()), "ok")
		assert.Contains(t, string(c.Response.Body()), "gateway-service")
	})

	t.Run("should return JSON response with timestamp", func(t *testing.T) {
		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")

		handler := Health("gateway-service")
		handler(ctx, c)

		// Check content type
		contentType := string(c.Response.Header.Peek("Content-Type"))
		assert.Contains(t, contentType, "application/json")

		// Check response structure
		body := string(c.Response.Body())
		assert.Contains(t, body, "status")
		assert.Contains(t, body, "service")
		assert.Contains(t, body, "timestamp")
		assert.Contains(t, body, "uptime")
	})
}

func TestHealthHandler_Ready(t *testing.T) {
	t.Run("should return ready status when all checks pass", func(t *testing.T) {
		checks := map[string]func() bool{
			"database": func() bool { return true },
			"redis":    func() bool { return true },
		}

		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")

		handler := Ready("gateway-service", checks)
		handler(ctx, c)

		// Verify response
		assert.Equal(t, http.StatusOK, c.Response.StatusCode())
		assert.Contains(t, string(c.Response.Body()), "ok")
	})

	t.Run("should return degraded status when checks fail", func(t *testing.T) {
		checks := map[string]func() bool{
			"database": func() bool { return true },
			"redis":    func() bool { return false },
		}

		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")

		handler := Ready("gateway-service", checks)
		handler(ctx, c)

		// Should return 503 Service Unavailable
		assert.Equal(t, http.StatusServiceUnavailable, c.Response.StatusCode())
		assert.Contains(t, string(c.Response.Body()), "degraded")
		assert.Contains(t, string(c.Response.Body()), "unhealthy")
	})

	t.Run("should check dependencies", func(t *testing.T) {
		checks := map[string]func() bool{
			"product-service": func() bool { return true },
			"auction-service": func() bool { return true },
		}

		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")

		handler := Ready("gateway-service", checks)
		handler(ctx, c)

		// Should include dependency status
		body := string(c.Response.Body())
		assert.Contains(t, body, "checks")
		assert.Contains(t, body, "product-service")
		assert.Contains(t, body, "auction-service")
	})
}

func TestHealthHandler_Metrics(t *testing.T) {
	t.Run("should return runtime metrics", func(t *testing.T) {
		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")

		handler := Metrics("gateway-service")
		handler(ctx, c)

		// Verify response
		assert.Equal(t, http.StatusOK, c.Response.StatusCode())

		body := string(c.Response.Body())
		assert.Contains(t, body, "service")
		assert.Contains(t, body, "uptime_seconds")
		assert.Contains(t, body, "goroutines")
		assert.Contains(t, body, "alloc_bytes")
	})
}

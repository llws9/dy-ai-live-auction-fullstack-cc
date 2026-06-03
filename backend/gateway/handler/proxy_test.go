package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestProxyHandler_Forward(t *testing.T) {
	// Create a mock backend server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for custom headers
		if userID := r.Header.Get("X-User-ID"); userID != "" {
			w.Header().Set("X-User-ID", userID)
		}
		if username := r.Header.Get("X-Username"); username != "" {
			w.Header().Set("X-Username", username)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"success"}`))
	}))
	defer mockServer.Close()

	t.Run("should forward GET request successfully", func(t *testing.T) {
		proxy := NewProxyHandler(mockServer.URL)

		// Create test context
		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/test?param=value")

		// Forward request
		proxy.Forward(ctx, c)

		// Verify response
		assert.Equal(t, http.StatusOK, c.Response.StatusCode())
		assert.Contains(t, string(c.Response.Body()), "success")
	})

	t.Run("should forward POST request with body", func(t *testing.T) {
		proxy := NewProxyHandler(mockServer.URL)

		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("POST")
		c.Request.SetRequestURI("/api/v1/test")
		c.Request.SetBody([]byte(`{"data":"test"}`))

		proxy.Forward(ctx, c)

		assert.Equal(t, http.StatusOK, c.Response.StatusCode())
	})

	t.Run("should forward user context headers", func(t *testing.T) {
		proxy := NewProxyHandler(mockServer.URL)

		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/test")

		// Set user context
		c.Set("user_id", int64(123))
		c.Set("username", "testuser")

		proxy.Forward(ctx, c)

		// Verify user headers are forwarded
		assert.Equal(t, http.StatusOK, c.Response.StatusCode())
	})

	t.Run("should handle backend error", func(t *testing.T) {
		proxy := NewProxyHandler("http://invalid-backend:9999")

		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/test")

		proxy.Forward(ctx, c)

		// Should return 502 Bad Gateway
		assert.Equal(t, http.StatusBadGateway, c.Response.StatusCode())
	})

	t.Run("should copy query parameters", func(t *testing.T) {
		proxy := NewProxyHandler(mockServer.URL)

		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/test?param1=value1&param2=value2")

		proxy.Forward(ctx, c)

		assert.Equal(t, http.StatusOK, c.Response.StatusCode())
	})
}

func TestProxyHandler_DoesNotForwardClientInternalToken(t *testing.T) {
	var seenInternalToken atomic.Value
	seenInternalToken.Store("")
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenInternalToken.Store(r.Header.Get("X-Internal-Token"))
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	proxy := NewProxyHandler(mockServer.URL)
	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/products")
	c.Request.Header.Set("X-Internal-Token", "client-supplied")

	proxy.Forward(context.Background(), c)

	assert.Equal(t, http.StatusOK, c.Response.StatusCode())
	assert.Empty(t, seenInternalToken.Load().(string))
}

func TestProxyHandler_WebSocket(t *testing.T) {
	t.Run("should return WebSocket connection info", func(t *testing.T) {
		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/ws?auction_id=123")

		HandleWebSocket(ctx, c, "http://localhost:8082")

		// Should return WebSocket info
		assert.Equal(t, http.StatusOK, c.Response.StatusCode())
		assert.Contains(t, string(c.Response.Body()), "ws://")
		assert.Contains(t, string(c.Response.Body()), "auction_id")
	})

	t.Run("should handle missing auction_id", func(t *testing.T) {
		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/ws")

		HandleWebSocket(ctx, c, "http://localhost:8082")

		// Should still return info with default auction_id=0
		assert.Equal(t, http.StatusOK, c.Response.StatusCode())
	})
}

func TestProxyWebSocket(t *testing.T) {
	t.Run("should create WebSocket proxy handler", func(t *testing.T) {
		handler := ProxyWebSocket("ws://localhost:8082")

		assert.NotNil(t, handler)

		// Test the handler
		ctx := context.Background()
		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.SetRequestURI("/api/v1/ws?auction_id=456")

		handler(ctx, c)

		assert.Equal(t, http.StatusOK, c.Response.StatusCode())
	})
}

func TestToString(t *testing.T) {
	t.Run("should convert string", func(t *testing.T) {
		result := toString("test")
		assert.Equal(t, "test", result)
	})

	t.Run("should convert int64", func(t *testing.T) {
		result := toString(int64(123))
		assert.NotEmpty(t, result)
	})

	t.Run("should convert byte slice", func(t *testing.T) {
		result := toString([]byte("test"))
		assert.Equal(t, "test", result)
	})

	t.Run("should handle unknown type", func(t *testing.T) {
		result := toString(123.45)
		assert.Empty(t, result)
	})
}

func TestNewProxyHandler(t *testing.T) {
	t.Run("should create proxy with target URL", func(t *testing.T) {
		proxy := NewProxyHandler("http://localhost:8081")

		assert.NotNil(t, proxy)
		assert.Equal(t, "http://localhost:8081", proxy.targetURL)
		assert.NotNil(t, proxy.client)
	})

	t.Run("should have HTTP client configured", func(t *testing.T) {
		proxy := NewProxyHandler("http://localhost:8081")

		assert.NotNil(t, proxy.client)
	})
}

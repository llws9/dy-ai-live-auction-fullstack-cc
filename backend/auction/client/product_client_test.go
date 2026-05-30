package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestListProductIDsByCategory 验证 client 真实发起 GET /internal/products?category_id=
// 并把返回的 items[].id 抽成 []int64。
func TestListProductIDsByCategory(t *testing.T) {
	t.Run("returns ids when product-service responds 200", func(t *testing.T) {
		var capturedQuery string
		var capturedPath string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedPath = r.URL.Path
			capturedQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code":    200,
				"message": "success",
				"data": map[string]interface{}{
					"items": []map[string]interface{}{
						{"id": 11, "name": "p1", "images": []string{"u1"}, "category_id": 7},
						{"id": 22, "name": "p2", "images": []string{}, "category_id": 7},
						{"id": 33, "name": "p3", "images": []string{"u3"}, "category_id": 7},
					},
					"total":     3,
					"page":      1,
					"page_size": 500,
				},
			})
		}))
		defer srv.Close()

		c := NewHTTPProductClient(srv.URL, 2*time.Second)
		ids, err := c.ListProductIDsByCategory(context.Background(), 7)
		require.NoError(t, err)
		assert.Equal(t, []int64{11, 22, 33}, ids)
		assert.Equal(t, "/internal/products", capturedPath)
		assert.Contains(t, capturedQuery, "category_id=7")
	})

	t.Run("returns error when product-service responds 5xx", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"code":500,"message":"db down"}`))
		}))
		defer srv.Close()

		c := NewHTTPProductClient(srv.URL, 2*time.Second)
		_, err := c.ListProductIDsByCategory(context.Background(), 7)
		require.Error(t, err)
	})

	t.Run("returns empty slice when items is empty", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{"items": []interface{}{}, "total": 0},
			})
		}))
		defer srv.Close()

		c := NewHTTPProductClient(srv.URL, 2*time.Second)
		ids, err := c.ListProductIDsByCategory(context.Background(), 7)
		require.NoError(t, err)
		assert.Empty(t, ids)
	})
}

// TestBatchGetSummaries 验证 client 真实发起 POST /internal/products/batch
// 并把返回的 items[] 索引到 map[id]Summary。
func TestBatchGetSummaries(t *testing.T) {
	t.Run("returns summaries indexed by id", func(t *testing.T) {
		var capturedBody map[string]interface{}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/internal/products/batch", r.URL.Path)
			_ = json.NewDecoder(r.Body).Decode(&capturedBody)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{
					"items": []map[string]interface{}{
						{"id": 11, "name": "p1", "images": []string{"u1"}, "category_id": 7},
						{"id": 22, "name": "p2", "images": []string{}, "category_id": 7},
					},
				},
			})
		}))
		defer srv.Close()

		c := NewHTTPProductClient(srv.URL, 2*time.Second)
		got, err := c.BatchGetSummaries(context.Background(), []int64{11, 22})
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "p1", got[11].Name)
		assert.Equal(t, []string{"u1"}, got[11].Images)
		assert.Equal(t, "p2", got[22].Name)

		// 验证请求体携带了 ids
		ids, _ := capturedBody["ids"].([]interface{})
		require.Len(t, ids, 2)
	})

	t.Run("returns empty map without HTTP call when ids empty", func(t *testing.T) {
		called := false
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))
		defer srv.Close()

		c := NewHTTPProductClient(srv.URL, 2*time.Second)
		got, err := c.BatchGetSummaries(context.Background(), []int64{})
		require.NoError(t, err)
		assert.Empty(t, got)
		assert.False(t, called, "should not call product-service for empty ids")
	})

	t.Run("returns error when product-service responds 5xx", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer srv.Close()

		c := NewHTTPProductClient(srv.URL, 2*time.Second)
		_, err := c.BatchGetSummaries(context.Background(), []int64{11})
		require.Error(t, err)
	})
}

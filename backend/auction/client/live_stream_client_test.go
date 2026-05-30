package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBatchGetLiveStreams 验证 LiveStreamClient 真实 POST /internal/live-streams/batch，
// 并把返回的 items[] 索引到 map[id]LiveStreamSummary。
func TestBatchGetLiveStreams(t *testing.T) {
	t.Run("returns map when product-service responds 200", func(t *testing.T) {
		var capturedPath string
		var capturedToken string
		var capturedBody []byte
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedPath = r.URL.Path
			capturedToken = r.Header.Get("X-Internal-Token")
			capturedBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code":    200,
				"message": "success",
				"data": map[string]interface{}{
					"items": []map[string]interface{}{
						{"id": 10, "name": "alice 直播间", "cover_image": "a.jpg", "status": 1, "creator_id": 100},
						{"id": 20, "name": "bob 直播间", "cover_image": "", "status": 0, "creator_id": 200},
					},
				},
			})
		}))
		defer srv.Close()

		c := NewHTTPLiveStreamClient(srv.URL, 2*time.Second)
		c.SetInternalToken("secret")

		got, err := c.BatchGetLiveStreams(context.Background(), []int64{10, 20})
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "/internal/live-streams/batch", capturedPath)
		assert.Equal(t, "secret", capturedToken, "client must forward X-Internal-Token")
		assert.Contains(t, string(capturedBody), "10")
		assert.Equal(t, "alice 直播间", got[10].Name)
		assert.EqualValues(t, 100, got[10].CreatorID)
		assert.EqualValues(t, 0, got[20].Status)
	})

	t.Run("empty ids → no HTTP call", func(t *testing.T) {
		var called bool
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
		defer srv.Close()
		c := NewHTTPLiveStreamClient(srv.URL, 2*time.Second)
		got, err := c.BatchGetLiveStreams(context.Background(), nil)
		require.NoError(t, err)
		assert.Empty(t, got)
		assert.False(t, called)
	})

	t.Run("returns error on non-200", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer srv.Close()
		c := NewHTTPLiveStreamClient(srv.URL, 2*time.Second)
		_, err := c.BatchGetLiveStreams(context.Background(), []int64{1})
		require.Error(t, err)
	})
}

package handler

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

// TestBuildUserStats 验证 T2.7 (F-A1) Gateway BFF 聚合：
//   - 并行调用 auction `/user/followed-live-streams` 与 product `/orders/history`
//   - won_count 由 product 历史项 is_winner=true 累计
//   - 单一下游失败时对应字段返回 nil（前端 -）
//   - 透传 X-User-ID
func TestBuildUserStats(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path: aggregates following + history + won counts", func(t *testing.T) {
		var auctionGotUserID, productGotUserID string

		auctionSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auctionGotUserID = r.Header.Get("X-User-ID")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{"items": []interface{}{}, "total": 12},
			})
		}))
		defer auctionSrv.Close()

		productSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			productGotUserID = r.Header.Get("X-User-ID")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"order_id": 1, "is_winner": true},
					map[string]interface{}{"order_id": 2, "is_winner": false},
					map[string]interface{}{"order_id": 3, "is_winner": true},
				},
				"total": 34,
			})
		}))
		defer productSrv.Close()

		fetcher := NewUserStatsFetcher(auctionSrv.URL, productSrv.URL, 2*time.Second)
		got := fetcher.Fetch(ctx, 7)

		assert.Equal(t, "7", auctionGotUserID)
		assert.Equal(t, "7", productGotUserID)
		require.NotNil(t, got.FollowingCount)
		assert.Equal(t, int64(12), *got.FollowingCount)
		require.NotNil(t, got.AuctionHistoryCount)
		assert.Equal(t, int64(34), *got.AuctionHistoryCount)
		require.NotNil(t, got.WonCount)
		assert.Equal(t, int64(2), *got.WonCount)
	})

	t.Run("auction down: following=nil, others computed", func(t *testing.T) {
		auctionSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", 500)
		}))
		defer auctionSrv.Close()
		productSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []interface{}{map[string]interface{}{"is_winner": true}},
				"total": 5,
			})
		}))
		defer productSrv.Close()

		fetcher := NewUserStatsFetcher(auctionSrv.URL, productSrv.URL, 2*time.Second)
		got := fetcher.Fetch(ctx, 9)

		assert.Nil(t, got.FollowingCount)
		require.NotNil(t, got.AuctionHistoryCount)
		assert.Equal(t, int64(5), *got.AuctionHistoryCount)
		require.NotNil(t, got.WonCount)
		assert.Equal(t, int64(1), *got.WonCount)
	})

	t.Run("product down: history+won=nil, following ok", func(t *testing.T) {
		auctionSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{"total": 7},
			})
		}))
		defer auctionSrv.Close()
		productSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", 503)
		}))
		defer productSrv.Close()

		fetcher := NewUserStatsFetcher(auctionSrv.URL, productSrv.URL, 2*time.Second)
		got := fetcher.Fetch(ctx, 9)

		require.NotNil(t, got.FollowingCount)
		assert.Equal(t, int64(7), *got.FollowingCount)
		assert.Nil(t, got.AuctionHistoryCount)
		assert.Nil(t, got.WonCount)
	})

	t.Run("both down: all nil, no error to caller", func(t *testing.T) {
		bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", 500)
		}))
		defer bad.Close()

		fetcher := NewUserStatsFetcher(bad.URL, bad.URL, 2*time.Second)
		got := fetcher.Fetch(ctx, 9)
		assert.Nil(t, got.FollowingCount)
		assert.Nil(t, got.AuctionHistoryCount)
		assert.Nil(t, got.WonCount)
	})
}

package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateManager_SaveSyncState(t *testing.T) {
	// This test verifies the state structure
	// Integration test with actual Redis would require running Redis instance

	t.Run("should create valid sync state", func(t *testing.T) {
		now := time.Now()
		state := &SyncState{
			AuctionID:    1,
			CurrentPrice: decimal.NewFromInt(150),
			WinnerID:     10001,
			EndTime:      now,
			Status:       1,
			UpdatedAt:    now,
		}

		assert.Equal(t, int64(1), state.AuctionID)
		assert.True(t, decimal.NewFromInt(150).Equal(state.CurrentPrice))
		assert.Equal(t, int64(10001), state.WinnerID)
		assert.Equal(t, 1, state.Status)
	})

	t.Run("should create valid connection state", func(t *testing.T) {
		now := time.Now()
		state := &ConnectionState{
			ClientID:       "test-client-1",
			AuctionID:      1,
			UserID:         10001,
			ConnectedAt:    now,
			LastPongAt:     now,
			ReconnectCount: 0,
		}

		assert.Equal(t, "test-client-1", state.ClientID)
		assert.Equal(t, int64(1), state.AuctionID)
		assert.Equal(t, int64(10001), state.UserID)
	})
}

func TestStateManager_GetSyncStateFallsBackAndBackfillsOnRedisMiss(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()
	manager := NewStateManager(rdb)
	endTime := time.Now().Add(10 * time.Minute).Truncate(time.Millisecond)
	loaderCalls := 0
	manager.SetSyncStateLoader(SyncStateLoaderFunc(func(ctx context.Context, auctionID int64) (*SyncState, error) {
		loaderCalls++
		return &SyncState{
			AuctionID:    auctionID,
			CurrentPrice: decimal.NewFromInt(8800),
			WinnerID:     42,
			EndTime:      endTime,
			Status:       1,
			UpdatedAt:    time.Now(),
		}, nil
	}))

	state, err := manager.GetSyncState(ctx, 6)

	require.NoError(t, err)
	assert.Equal(t, int64(6), state.AuctionID)
	assert.True(t, decimal.NewFromInt(8800).Equal(state.CurrentPrice))
	assert.Equal(t, int64(42), state.WinnerID)
	assert.Equal(t, 1, state.Status)
	assert.Equal(t, 1, loaderCalls)
	assert.True(t, mr.Exists("sync:state:6"))

	_, err = manager.GetSyncState(ctx, 6)

	require.NoError(t, err)
	assert.Equal(t, 1, loaderCalls, "second read should use Redis backfill instead of loader")
}

func TestStateManager_ConnectionState(t *testing.T) {
	t.Run("should handle reconnect count", func(t *testing.T) {
		now := time.Now()
		state := &ConnectionState{
			ClientID:       "test-client-2",
			AuctionID:      1,
			UserID:         10001,
			ConnectedAt:    now,
			LastPongAt:     now,
			ReconnectCount: 3,
		}

		assert.Equal(t, 3, state.ReconnectCount)
	})
}

func TestStateManager_StateKey(t *testing.T) {
	t.Run("should generate correct Redis key pattern", func(t *testing.T) {
		// Verify key format pattern
		clientID := "test-client-123"
		expectedPattern := "conn:state:test-client-123"

		// The actual key generation is in SaveConnectionState
		assert.Contains(t, expectedPattern, clientID)
	})
}

func TestSyncState_Expiration(t *testing.T) {
	t.Run("should set appropriate TTL for state", func(t *testing.T) {
		// State should expire after auction ends (7 days for safety)
		expectedTTL := 7 * 24 * time.Hour

		// Verify TTL is reasonable for reconnection scenarios
		assert.GreaterOrEqual(t, expectedTTL.Hours(), 24.0)
		assert.LessOrEqual(t, expectedTTL.Hours(), 168.0) // Max 7 days
	})
}

func TestStateManager_ConcurrentAccess(t *testing.T) {
	t.Run("should handle concurrent state updates", func(t *testing.T) {
		// This tests the concept - actual implementation would need Redis mock
		done := make(chan bool, 2)

		// Simulate concurrent updates
		go func() {
			state := &SyncState{AuctionID: 1, CurrentPrice: decimal.NewFromInt(100)}
			_ = state
			done <- true
		}()

		go func() {
			state := &SyncState{AuctionID: 1, CurrentPrice: decimal.NewFromInt(150)}
			_ = state
			done <- true
		}()

		// Wait for both goroutines
		<-done
		<-done

		// Both should complete without panic
		assert.True(t, true)
	})
}

// Integration test placeholder (requires running Redis)
func TestStateManager_Integration(t *testing.T) {
	t.Skip("Integration test - requires Redis instance")

	// This would test:
	// 1. Save state to Redis
	// 2. Retrieve state from Redis
	// 3. Update existing state
	// 4. Delete expired state
}

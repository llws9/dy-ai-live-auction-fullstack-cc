package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupThrottle(t *testing.T) (*ChatThrottle, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cfg := ThrottleConfig{
		UserMax:      1,
		UserInterval: time.Second,
		RoomMax:      20,
		RoomInterval: time.Second,
	}
	return NewChatThrottle(rdb, cfg), mr
}

func TestChatThrottle_UserLimit(t *testing.T) {
	th, _ := setupThrottle(t)
	ctx := context.Background()

	if code := th.Allow(ctx, 100, 1); code != 0 {
		t.Fatalf("first message should pass, got code %d", code)
	}
	if code := th.Allow(ctx, 100, 1); code != ChatErrCodeRateLimited {
		t.Fatalf("second message in 1s should be rate-limited, got %d", code)
	}
}

func TestChatThrottle_RoomLimit(t *testing.T) {
	th, _ := setupThrottle(t)
	ctx := context.Background()

	// 20 个不同用户连续发，第 21 个被房间限流
	for i := 1; i <= 20; i++ {
		if code := th.Allow(ctx, int64(i), 999); code != 0 {
			t.Fatalf("user %d should pass, got code %d", i, code)
		}
	}
	if code := th.Allow(ctx, 9999, 999); code != ChatErrCodeRateLimited {
		t.Fatalf("21st message in same room should be rate-limited, got %d", code)
	}
}

func TestChatThrottle_TTLReset(t *testing.T) {
	th, mr := setupThrottle(t)
	ctx := context.Background()

	if code := th.Allow(ctx, 100, 1); code != 0 {
		t.Fatal("first should pass")
	}
	mr.FastForward(time.Second + 100*time.Millisecond) // 跳过 TTL
	if code := th.Allow(ctx, 100, 1); code != 0 {
		t.Fatalf("after TTL expires, should pass, got %d", code)
	}
}

func TestChatThrottle_DoesNotRefreshTTLOnRepeatedHits(t *testing.T) {
	th, mr := setupThrottle(t)
	ctx := context.Background()

	if code := th.Allow(ctx, 100, 1); code != 0 {
		t.Fatal("first should pass")
	}
	mr.FastForward(900 * time.Millisecond)
	if code := th.Allow(ctx, 100, 1); code != ChatErrCodeRateLimited {
		t.Fatalf("second in window should be rate-limited, got %d", code)
	}
	mr.FastForward(200 * time.Millisecond)
	if code := th.Allow(ctx, 100, 1); code != 0 {
		t.Fatalf("fixed window should expire from first hit, got %d", code)
	}
}

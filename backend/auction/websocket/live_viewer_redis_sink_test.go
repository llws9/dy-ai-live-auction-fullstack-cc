package websocket

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisLiveViewerCountSinkWritesProductViewerKey(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	sink := NewRedisLiveViewerCountSink(rdb)
	if err := sink.SetLiveViewerCount(993112, 3); err != nil {
		t.Fatalf("SetLiveViewerCount returned error: %v", err)
	}

	got, err := rdb.Get(context.Background(), "live:viewer:993112").Int()
	if err != nil {
		t.Fatalf("redis get live:viewer key: %v", err)
	}
	if got != 3 {
		t.Fatalf("live:viewer:993112 = %d, want 3", got)
	}
}

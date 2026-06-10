package websocket

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisLiveViewerCountSink struct {
	client *redis.Client
}

func NewRedisLiveViewerCountSink(client *redis.Client) *RedisLiveViewerCountSink {
	return &RedisLiveViewerCountSink{client: client}
}

func (s *RedisLiveViewerCountSink) SetLiveViewerCount(liveStreamID int64, count int) error {
	if s == nil || s.client == nil || liveStreamID <= 0 || count < 0 {
		return nil
	}
	key := fmt.Sprintf("live:viewer:%d", liveStreamID)
	return s.client.Set(context.Background(), key, count, 0).Err()
}

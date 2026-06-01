package service

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisLiveViewerCounter struct {
	client *redis.Client
}

func NewRedisLiveViewerCounter(client *redis.Client) *RedisLiveViewerCounter {
	return &RedisLiveViewerCounter{client: client}
}

func (c *RedisLiveViewerCounter) Count(ctx context.Context, liveStreamID int64) (int64, error) {
	if c == nil || c.client == nil {
		return 0, nil
	}
	key := fmt.Sprintf("live:viewer:%d", liveStreamID)
	count, err := c.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

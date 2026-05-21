package lock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// 锁相关错误
var (
	ErrLockNotAcquired = errors.New("锁获取失败")
	ErrLockNotHeld     = errors.New("锁未持有")
)

// RedisLock Redis 分布式锁
type RedisLock struct {
	client *redis.Client
	key    string
	value  string
	ttl    time.Duration
}

// RedisLockConfig 锁配置
type RedisLockConfig struct {
	Client *redis.Client
	Key    string
	Value  string
	TTL    time.Duration
}

// NewRedisLock 创建 Redis 分布式锁
func NewRedisLock(client *redis.Client, key string, ttl time.Duration) *RedisLock {
	return &RedisLock{
		client: client,
		key:    key,
		value:  fmt.Sprintf("%d:%d", time.Now().UnixNano(), time.Now().Nanosecond()),
		ttl:    ttl,
	}
}

// Acquire 获取锁（非阻塞）
func (l *RedisLock) Acquire(ctx context.Context) error {
	// 使用 SETNX 原子操作
	success, err := l.client.SetNX(ctx, l.key, l.value, l.ttl).Result()
	if err != nil {
		return fmt.Errorf("redis setnx error: %w", err)
	}

	if !success {
		return ErrLockNotAcquired
	}

	return nil
}

// AcquireWithRetry 获取锁（带重试）
func (l *RedisLock) AcquireWithRetry(ctx context.Context, maxRetries int, retryInterval time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		err := l.Acquire(ctx)
		if err == nil {
			return nil
		}

		if err != ErrLockNotAcquired {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
			continue
		}
	}

	return ErrLockNotAcquired
}

// Release 释放锁
func (l *RedisLock) Release(ctx context.Context) error {
	// 使用 Lua 脚本确保原子性：只有持有锁的客户端才能释放锁
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script, []string{l.key}, l.value).Int()
	if err != nil {
		return fmt.Errorf("redis eval error: %w", err)
	}

	if result == 0 {
		return ErrLockNotHeld
	}

	return nil
}

// Extend 延长锁的过期时间
func (l *RedisLock) Extend(ctx context.Context, additionalTTL time.Duration) error {
	// 使用 Lua 脚本确保原子性：只有持有锁的客户端才能延长
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("PEXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := l.client.Eval(ctx, script, []string{l.key}, l.value, additionalTTL.Milliseconds()).Int()
	if err != nil {
		return fmt.Errorf("redis eval error: %w", err)
	}

	if result == 0 {
		return ErrLockNotHeld
	}

	return nil
}

// IsHeld 检查是否持有锁
func (l *RedisLock) IsHeld(ctx context.Context) (bool, error) {
	value, err := l.client.Get(ctx, l.key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return value == l.value, nil
}

// GetTTL 获取锁的剩余过期时间
func (l *RedisLock) GetTTL(ctx context.Context) (time.Duration, error) {
	return l.client.TTL(ctx, l.key).Result()
}

// AuctionBidLock 竞拍出价专用锁
type AuctionBidLock struct {
	lock *RedisLock
}

// NewAuctionBidLock 创建竞拍出价锁
func NewAuctionBidLock(client *redis.Client, auctionID int64) *AuctionBidLock {
	key := fmt.Sprintf("auction:bid:%d:lock", auctionID)
	return &AuctionBidLock{
		lock: NewRedisLock(client, key, 5*time.Second),
	}
}

// Acquire 获取出价锁
func (l *AuctionBidLock) Acquire(ctx context.Context) error {
	// 尝试最多 10 次，每次间隔 50ms
	return l.lock.AcquireWithRetry(ctx, 10, 50*time.Millisecond)
}

// Release 释放出价锁
func (l *AuctionBidLock) Release(ctx context.Context) error {
	return l.lock.Release(ctx)
}

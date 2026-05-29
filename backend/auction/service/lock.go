package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// DistributedLockService 分布式锁服务
type DistributedLockService struct {
	redis       *redis.Client
	localLocks  sync.Map // 本地锁降级
	defaultTTL  time.Duration
}

// NewDistributedLockService 创建分布式锁服务
func NewDistributedLockService(redisClient *redis.Client) *DistributedLockService {
	return &DistributedLockService{
		redis:      redisClient,
		defaultTTL: 5 * time.Second,
	}
}

// AcquireLock 获取分布式锁
func (s *DistributedLockService) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if ttl == 0 {
		ttl = s.defaultTTL
	}

	// 尝试使用 Redis 分布式锁
	if s.redis != nil {
		acquired, err := s.redis.SetNX(ctx, key, "locked", ttl).Result()
		if err != nil {
			log.Printf("Redis lock acquire failed, fallback to local lock: %v", err)
			// 降级为本地锁
			return s.acquireLocalLock(key, ttl), nil
		}
		return acquired, nil
	}

	// Redis 不可用，使用本地锁
	return s.acquireLocalLock(key, ttl), nil
}

// ReleaseLock 释放分布式锁
func (s *DistributedLockService) ReleaseLock(ctx context.Context, key string) error {
	if s.redis != nil {
		err := s.redis.Del(ctx, key).Err()
		if err != nil {
			log.Printf("Redis lock release failed: %v", err)
		}
	}

	// 始终释放本地锁
	s.releaseLocalLock(key)
	return nil
}

// acquireLocalLock 获取本地锁（降级方案）
// 使用 LoadOrStore 保证 check-and-set 原子性，避免并发场景下的 TOCTOU 竞态
func (s *DistributedLockService) acquireLocalLock(key string, ttl time.Duration) bool {
	// LoadOrStore: 如果 key 已存在则返回已有值（loaded=true），否则存储新值（loaded=false）
	if _, loaded := s.localLocks.LoadOrStore(key, time.Now()); loaded {
		return false
	}

	// 设置过期删除
	go func() {
		time.Sleep(ttl)
		s.localLocks.Delete(key)
	}()

	return true
}

// releaseLocalLock 释放本地锁
func (s *DistributedLockService) releaseLocalLock(key string) {
	s.localLocks.Delete(key)
}

// WithLock 使用分布式锁执行函数
func (s *DistributedLockService) WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	acquired, err := s.AcquireLock(ctx, key, ttl)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !acquired {
		return fmt.Errorf("lock not acquired: %s", key)
	}

	defer s.ReleaseLock(ctx, key)
	return fn()
}

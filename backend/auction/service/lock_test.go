package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDistributedLockService_AcquireLock(t *testing.T) {
	// 创建服务（无 Redis，使用本地锁）
	svc := NewDistributedLockService(nil)

	ctx := context.Background()
	key := "test:lock:1"

	// 测试获取锁
	acquired, err := svc.AcquireLock(ctx, key, time.Second)
	assert.NoError(t, err)
	assert.True(t, acquired)

	// 测试重复获取锁（应失败）
	acquired2, err := svc.AcquireLock(ctx, key, time.Second)
	assert.NoError(t, err)
	assert.False(t, acquired2)

	// 释放锁
	err = svc.ReleaseLock(ctx, key)
	assert.NoError(t, err)

	// 再次获取锁（应成功）
	acquired3, err := svc.AcquireLock(ctx, key, time.Second)
	assert.NoError(t, err)
	assert.True(t, acquired3)
}

func TestDistributedLockService_WithLock(t *testing.T) {
	svc := NewDistributedLockService(nil)

	ctx := context.Background()
	key := "test:lock:with"

	executed := false
	err := svc.WithLock(ctx, key, time.Second, func() error {
		executed = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestDistributedLockService_ConcurrentLock(t *testing.T) {
	svc := NewDistributedLockService(nil)

	ctx := context.Background()
	key := "test:lock:concurrent"

	successCount := 0
	var mu sync.Mutex

	// 并发获取锁
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			acquired, _ := svc.AcquireLock(ctx, key, 100*time.Millisecond)
			if acquired {
				mu.Lock()
				successCount++
				mu.Unlock()
				time.Sleep(50 * time.Millisecond)
				svc.ReleaseLock(ctx, key)
			}
		}()
	}
	wg.Wait()

	// 只有一个应该成功
	assert.Equal(t, 1, successCount)
}

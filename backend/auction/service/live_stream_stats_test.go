package service

import (
	"context"
	"testing"
	"time"

	"auction-service/dao"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLiveStreamStatsService_NewService 测试服务创建
func TestLiveStreamStatsService_NewService(t *testing.T) {
	service := NewLiveStreamStatsService()
	assert.NotNil(t, service)
}

// TestLiveStreamStats_GetStatsKey 测试Redis key生成
func TestLiveStreamStats_GetStatsKey(t *testing.T) {
	service := NewLiveStreamStatsService()
	key := service.getStatsKey(123)
	assert.Contains(t, key, "123")
	assert.Contains(t, key, "live_stream")
}

// TestLiveStreamStats_HotnessThreshold 测试热门阈值
func TestLiveStreamStats_HotnessThreshold(t *testing.T) {
	// 冷门直播间：关注人数 < 200
	assert.True(t, 150 < HotnessThreshold)

	// 热门直播间：关注人数 >= 200
	assert.False(t, 250 < HotnessThreshold)
}

// TestLiveStreamStatsService_SetScheduledStartTime 测试设定开播时间
func TestLiveStreamStatsService_SetScheduledStartTime(t *testing.T) {
	// 此测试需要Redis连接，在无Redis环境下跳过
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	service := NewLiveStreamStatsService()
	if service.redis == nil {
		t.Skip("redis client not initialized")
	}
	ctx := context.Background()

	startTime := time.Now().Add(2 * time.Hour)

	// 测试冷门直播间（关注人数 < 200）
	err := service.SetScheduledStartTime(ctx, 1001, startTime, 100)
	assert.NoError(t, err)

	// 验证状态
	stats, err := service.GetStats(ctx, 1001)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(1001), stats.LiveStreamID)
	assert.Equal(t, 100, stats.FollowerCount)
	assert.False(t, stats.IsHot)
	assert.Equal(t, "pending", stats.Status)

	// 测试热门直播间（关注人数 >= 200）
	err = service.SetScheduledStartTime(ctx, 1002, startTime, 250)
	assert.NoError(t, err)

	stats, err = service.GetStats(ctx, 1002)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.True(t, stats.IsHot)
}

// TestLiveStreamStatsService_UpdateHotness 测试热度更新和迁移
func TestLiveStreamStatsService_UpdateHotness(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	service := NewLiveStreamStatsService()
	if service.redis == nil {
		t.Skip("redis client not initialized")
	}
	ctx := context.Background()

	startTime := time.Now().Add(1 * time.Hour)

	// 初始状态：冷门直播间
	err := service.SetScheduledStartTime(ctx, 2001, startTime, 50)
	assert.NoError(t, err)

	stats, err := service.GetStats(ctx, 2001)
	assert.NoError(t, err)
	assert.False(t, stats.IsHot)

	// 更新热度：跨越阈值成为热门
	err = service.UpdateHotness(ctx, 2001, 250)
	assert.NoError(t, err)

	stats, err = service.GetStats(ctx, 2001)
	assert.NoError(t, err)
	assert.True(t, stats.IsHot)
	assert.Equal(t, 250, stats.FollowerCount)

	// 更新热度：回落到冷门
	err = service.UpdateHotness(ctx, 2001, 150)
	assert.NoError(t, err)

	stats, err = service.GetStats(ctx, 2001)
	assert.NoError(t, err)
	assert.False(t, stats.IsHot)
	assert.Equal(t, 150, stats.FollowerCount)
}

// TestLiveStreamStatsService_StartLive 测试开始直播
func TestLiveStreamStatsService_StartLive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	service := NewLiveStreamStatsService()
	if service.redis == nil {
		t.Skip("redis client not initialized")
	}
	ctx := context.Background()

	startTime := time.Now().Add(30 * time.Minute)

	// 设置热门直播间
	err := service.SetScheduledStartTime(ctx, 3001, startTime, 300)
	assert.NoError(t, err)

	stats, err := service.GetStats(ctx, 3001)
	assert.NoError(t, err)
	assert.True(t, stats.IsHot)

	// 开始直播
	err = service.StartLive(ctx, 3001)
	assert.NoError(t, err)

	stats, err = service.GetStats(ctx, 3001)
	assert.NoError(t, err)
	assert.Equal(t, "live", stats.Status)
	assert.Nil(t, stats.ScheduledStart)
	assert.NotNil(t, stats.StartedAt)
	assert.WithinDuration(t, time.Now(), *stats.StartedAt, 2*time.Second)

	// 验证在live_now集合中
	liveNowList, err := service.GetLiveNowHotStreams(ctx)
	assert.NoError(t, err)
	assert.Contains(t, liveNowList, int64(3001))
}

func TestLiveStreamStatsService_StartLiveIsIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_, err := dao.InitRedis("localhost:6379", "")
	require.NoError(t, err)
	service := NewLiveStreamStatsService()
	if service.redis == nil {
		t.Skip("redis client not initialized")
	}
	ctx := context.Background()

	liveStreamID := time.Now().UnixNano()%1_000_000_000 + 2_000_000_000
	err = service.SetScheduledStartTime(ctx, liveStreamID, time.Now().Add(30*time.Minute), 300)
	assert.NoError(t, err)

	err = service.StartLive(ctx, liveStreamID)
	assert.NoError(t, err)
	first, err := service.GetStats(ctx, liveStreamID)
	assert.NoError(t, err)
	assert.NotNil(t, first.StartedAt)

	time.Sleep(10 * time.Millisecond)
	err = service.StartLive(ctx, liveStreamID)
	assert.NoError(t, err)
	second, err := service.GetStats(ctx, liveStreamID)
	assert.NoError(t, err)
	assert.NotNil(t, second.StartedAt)
	assert.Equal(t, first.StartedAt.UnixNano(), second.StartedAt.UnixNano())
}

// TestLiveStreamStatsService_EndLive 测试结束直播
func TestLiveStreamStatsService_EndLive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	service := NewLiveStreamStatsService()
	if service.redis == nil {
		t.Skip("redis client not initialized")
	}
	ctx := context.Background()

	startTime := time.Now().Add(30 * time.Minute)

	// 设置并开始直播
	err := service.SetScheduledStartTime(ctx, 4001, startTime, 300)
	assert.NoError(t, err)

	err = service.StartLive(ctx, 4001)
	assert.NoError(t, err)

	// 结束直播
	err = service.EndLive(ctx, 4001)
	assert.NoError(t, err)

	// 验证缓存已清除
	stats, err := service.GetStats(ctx, 4001)
	assert.NoError(t, err)
	assert.Nil(t, stats)

	// 验证不在live_now集合中
	liveNowList, err := service.GetLiveNowHotStreams(ctx)
	assert.NoError(t, err)
	assert.NotContains(t, liveNowList, int64(4001))
}

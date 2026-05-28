package cron

import (
	"context"
	"testing"
	"time"

	"auction-service/dao"
	"auction-service/service"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// TestStatsCron_NewStatsCron 测试创建 StatsCron
func TestStatsCron_NewStatsCron(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 创建服务
	followDAO := dao.NewUserLiveStreamFollowDAO(nil)
	statsService := service.NewLiveStreamStatsService()

	// 创建 StatsCron
	cron := NewStatsCron(followDAO, statsService)
	assert.NotNil(t, cron)
	assert.Equal(t, 5*time.Minute, cron.interval)
}

// TestStatsCron_SetInterval 测试设置间隔
func TestStatsCron_SetInterval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	followDAO := dao.NewUserLiveStreamFollowDAO(nil)
	statsService := service.NewLiveStreamStatsService()

	cron := NewStatsCron(followDAO, statsService)
	cron.SetInterval(1 * time.Minute)
	assert.Equal(t, 1*time.Minute, cron.interval)
}

// TestStatsCron_StartStop 测试启动和停止
func TestStatsCron_StartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	followDAO := dao.NewUserLiveStreamFollowDAO(nil)
	statsService := service.NewLiveStreamStatsService()

	cron := NewStatsCron(followDAO, statsService)
	cron.SetInterval(1 * time.Second) // 测试用较短间隔

	ctx := context.Background()
	cron.Start(ctx)

	// 等待一小段时间
	time.Sleep(100 * time.Millisecond)

	// 停止
	cron.Stop()

	// 确认已停止
	assert.False(t, cron.running)
}

// TestStatsCron_GetActiveLiveStreams 测试获取活跃直播间
func TestStatsCron_GetActiveLiveStreams(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	followDAO := dao.NewUserLiveStreamFollowDAO(nil)
	statsService := service.NewLiveStreamStatsService()

	cron := NewStatsCron(followDAO, statsService)

	ctx := context.Background()

	// 添加一些测试数据到 Redis
	redis := dao.GetRedis()

	// 添加正在直播的热门直播间
	redis.SAdd(ctx, dao.HotLiveNowSet, 1, 2, 3)

	// 添加计划开播的冷门直播间
	now := time.Now()
	score := float64(now.Add(30 * time.Minute).Unix())
	redis.ZAdd(ctx, dao.ColdLiveStreamZSET, goredis.Z{Score: score, Member: int64(4)})
	redis.ZAdd(ctx, dao.ColdLiveStreamZSET, goredis.Z{Score: score, Member: int64(5)})

	// 添加计划开播的热门直播间
	redis.ZAdd(ctx, dao.HotLiveStreamZSET, goredis.Z{Score: score, Member: int64(6)})

	// 获取活跃直播间
	activeStreams, err := cron.getActiveLiveStreams(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(activeStreams), 3) // 至少有3个正在直播的热门

	// 清理测试数据
	redis.SRem(ctx, dao.HotLiveNowSet, 1, 2, 3)
	redis.ZRem(ctx, dao.ColdLiveStreamZSET, 4, 5)
	redis.ZRem(ctx, dao.HotLiveStreamZSET, 6)
}

// TestStatsCron_UpdateAllLiveStreamHotness 测试更新所有直播间热度
func TestStatsCron_UpdateAllLiveStreamHotness(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	followDAO := dao.NewUserLiveStreamFollowDAO(nil)
	statsService := service.NewLiveStreamStatsService()

	cron := NewStatsCron(followDAO, statsService)

	ctx := context.Background()

	// 添加测试数据
	redis := dao.GetRedis()

	// 添加正在直播的热门直播间
	redis.SAdd(ctx, dao.HotLiveNowSet, 9991, 9992)

	// 执行更新
	cron.updateAllLiveStreamHotness(ctx)

	// 清理测试数据
	redis.SRem(ctx, dao.HotLiveNowSet, 9991, 9992)
}

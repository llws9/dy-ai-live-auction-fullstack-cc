package cron

import (
	"context"
	"log"
	"sync"
	"time"

	"auction-service/dao"
	"auction-service/service"
)

// StatsCron 热度自动更新定时任务
// 每5分钟查询所有活跃直播间，根据关注数更新热度状态
type StatsCron struct {
	followDAO              *dao.UserLiveStreamFollowDAO
	liveStreamStatsService *service.LiveStreamStatsService
	interval               time.Duration
	stopChan               chan struct{}
	stopOnce               sync.Once
	wg                     sync.WaitGroup
	mu                     sync.Mutex
	running                bool
}

// NewStatsCron 创建热度自动更新定时任务
func NewStatsCron(followDAO *dao.UserLiveStreamFollowDAO, statsService *service.LiveStreamStatsService) *StatsCron {
	return &StatsCron{
		followDAO:              followDAO,
		liveStreamStatsService: statsService,
		interval:               5 * time.Minute,
		stopChan:               make(chan struct{}),
		running:                false,
	}
}

// SetInterval 设置检查间隔（用于测试）
func (c *StatsCron) SetInterval(interval time.Duration) {
	c.interval = interval
}

// Start 启动定时任务
func (c *StatsCron) Start(ctx context.Context) {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	// Reset stopChan and stopOnce for a fresh start
	c.stopChan = make(chan struct{})
	c.stopOnce = sync.Once{}
	c.running = true
	c.mu.Unlock()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		log.Printf("[StatsCron] Started, interval: %v", c.interval)

		// 立即执行一次
		c.updateAllLiveStreamHotness(ctx)

		for {
			select {
			case <-ctx.Done():
				log.Printf("[StatsCron] Context cancelled, stopping")
				return
			case <-c.stopChan:
				log.Printf("[StatsCron] Stop signal received, stopping")
				return
			case <-ticker.C:
				c.updateAllLiveStreamHotness(ctx)
			}
		}
	}()
}

// Stop 停止定时任务
func (c *StatsCron) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	// Use sync.Once to ensure stopChan is closed only once
	c.stopOnce.Do(func() {
		close(c.stopChan)
	})
	c.wg.Wait()

	c.mu.Lock()
	c.running = false
	c.mu.Unlock()

	log.Printf("[StatsCron] Stopped")
}

// updateAllLiveStreamHotness 更新所有活跃直播间的热度状态
func (c *StatsCron) updateAllLiveStreamHotness(ctx context.Context) {
	startTime := time.Now()
	log.Printf("[StatsCron] Starting hotness update cycle")

	// 1. 获取所有活跃直播间ID
	// 从 Redis 的 HotLiveNowSet 和 ZSET 获取活跃直播间
	activeLiveStreamIDs, err := c.getActiveLiveStreams(ctx)
	if err != nil {
		log.Printf("[StatsCron] Failed to get active live streams: %v", err)
		return
	}

	if len(activeLiveStreamIDs) == 0 {
		log.Printf("[StatsCron] No active live streams found")
		return
	}

	log.Printf("[StatsCron] Found %d active live streams", len(activeLiveStreamIDs))

	// 2. 遍历每个直播间，获取关注数并更新热度
	successCount := 0
	failCount := 0

	for _, liveStreamID := range activeLiveStreamIDs {
		// 获取关注数
		followerCount, err := c.followDAO.CountByLiveStream(ctx, liveStreamID)
		if err != nil {
			log.Printf("[StatsCron] Failed to count followers for live stream %d: %v", liveStreamID, err)
			failCount++
			continue
		}

		// 更新热度状态
		if err := c.liveStreamStatsService.UpdateHotness(ctx, liveStreamID, int(followerCount)); err != nil {
			log.Printf("[StatsCron] Failed to update hotness for live stream %d: %v", liveStreamID, err)
			failCount++
			continue
		}

		successCount++
		// Only log at debug level for individual updates - summary is logged at end
	}

	elapsed := time.Since(startTime)
	log.Printf("[StatsCron] Update cycle completed: success=%d, fail=%d, elapsed=%v", successCount, failCount, elapsed)
}

// getActiveLiveStreams 获取所有活跃直播间ID
// 包括：正在直播的热门直播间 + 计划开播的直播间（冷门+热门）
func (c *StatsCron) getActiveLiveStreams(ctx context.Context) ([]int64, error) {
	// 使用 map 去重
	liveStreamMap := make(map[int64]bool)

	// 1. 获取正在直播的热门直播间
	hotLiveNow, err := c.liveStreamStatsService.GetLiveNowHotStreams(ctx)
	if err != nil {
		log.Printf("[StatsCron] Failed to get hot live now streams: %v", err)
		// 不返回错误，继续尝试获取其他直播间
	} else {
		for _, id := range hotLiveNow {
			liveStreamMap[id] = true
		}
	}

	// 2. 获取计划开播的冷门直播间（未来1小时内）
	now := time.Now()
	endTime := now.Add(1 * time.Hour)
	coldScheduled, err := c.liveStreamStatsService.GetScheduledColdLiveStreams(ctx, now.Unix(), endTime.Unix())
	if err != nil {
		log.Printf("[StatsCron] Failed to get scheduled cold live streams: %v", err)
	} else {
		for _, id := range coldScheduled {
			liveStreamMap[id] = true
		}
	}

	// 3. 获取计划开播的热门直播间（未来1小时内）
	hotScheduled, err := c.liveStreamStatsService.GetScheduledHotLiveStreams(ctx, now.Unix(), endTime.Unix())
	if err != nil {
		log.Printf("[StatsCron] Failed to get scheduled hot live streams: %v", err)
	} else {
		for _, id := range hotScheduled {
			liveStreamMap[id] = true
		}
	}

	// 转换为切片
	result := make([]int64, 0, len(liveStreamMap))
	for id := range liveStreamMap {
		result = append(result, id)
	}

	return result, nil
}

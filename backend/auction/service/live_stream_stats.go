package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"auction-service/dao"

	"github.com/redis/go-redis/v9"
)

const HotnessThreshold = 200 // 热门直播间阈值

// Note: Redis key constants are defined in dao/redis_live_stream.go
// We reuse those keys: ColdLiveStreamZSET, HotLiveStreamZSET, HotLiveNowSet, LiveStreamStatsKey

// LiveStreamStats 直播间热度状态缓存
type LiveStreamStats struct {
	LiveStreamID   int64      `json:"live_stream_id"`
	FollowerCount  int        `json:"follower_count"`
	IsHot          bool       `json:"is_hot"`
	ScheduledStart *time.Time `json:"scheduled_start,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	Status         string     `json:"status"` // "pending", "live", "ended"
}

// LiveStreamStatsService 直播间热度状态服务
type LiveStreamStatsService struct {
	redis *redis.Client
}

// NewLiveStreamStatsService 创建直播间热度状态服务
func NewLiveStreamStatsService() *LiveStreamStatsService {
	return &LiveStreamStatsService{
		redis: dao.GetRedis(),
	}
}

// getStatsKey 获取直播间统计缓存的Redis key
func (s *LiveStreamStatsService) getStatsKey(liveStreamID int64) string {
	return fmt.Sprintf(dao.LiveStreamStatsKey, liveStreamID)
}

// UpdateHotness 更新热度状态，跨阈值时迁移ZSET
// 1. 获取当前热度状态
// 2. 判断是否跨阈值（冷->热 或 热->冷）
// 3. 如果跨阈值：从旧ZSET移除，加入新ZSET
// 4. 更新Redis缓存 live_stream:{id}:stats
func (s *LiveStreamStatsService) UpdateHotness(ctx context.Context, liveStreamID int64, followerCount int) error {
	// 获取当前状态
	currentStats, err := s.GetStats(ctx, liveStreamID)
	if err != nil && err != redis.Nil {
		return fmt.Errorf("获取当前状态失败: %w", err)
	}

	// 如果不存在，创建新状态
	if currentStats == nil {
		currentStats = &LiveStreamStats{
			LiveStreamID:  liveStreamID,
			FollowerCount: 0,
			IsHot:         false,
			Status:        "pending",
		}
	}

	// 判断是否跨阈值
	wasHot := currentStats.IsHot
	nowHot := followerCount >= HotnessThreshold

	// 如果状态发生变化且有待开播时间
	if wasHot != nowHot && currentStats.ScheduledStart != nil && currentStats.Status == "pending" {
		// 需要迁移ZSET
		if err := s.migrateZSET(ctx, liveStreamID, wasHot, nowHot, *currentStats.ScheduledStart); err != nil {
			return fmt.Errorf("迁移ZSET失败: %w", err)
		}
	}

	// 更新缓存
	currentStats.FollowerCount = followerCount
	currentStats.IsHot = nowHot

	if err := s.saveStats(ctx, currentStats); err != nil {
		return fmt.Errorf("保存状态失败: %w", err)
	}

	return nil
}

// GetStats 获取直播间热度缓存
func (s *LiveStreamStatsService) GetStats(ctx context.Context, liveStreamID int64) (*LiveStreamStats, error) {
	key := s.getStatsKey(liveStreamID)

	data, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 不存在返回nil
		}
		return nil, err
	}

	var stats LiveStreamStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, fmt.Errorf("解析缓存数据失败: %w", err)
	}

	return &stats, nil
}

// SetScheduledStartTime 设定开播时间
// 同时根据热度加入对应ZSET（冷门或热门）
func (s *LiveStreamStatsService) SetScheduledStartTime(ctx context.Context, liveStreamID int64, startTime time.Time, followerCount int) error {
	// 获取或创建状态
	stats, err := s.GetStats(ctx, liveStreamID)
	if err != nil {
		return fmt.Errorf("获取状态失败: %w", err)
	}

	if stats == nil {
		stats = &LiveStreamStats{
			LiveStreamID:  liveStreamID,
			FollowerCount: followerCount,
			Status:        "pending",
		}
	}

	// 判断热度
	isHot := followerCount >= HotnessThreshold
	stats.IsHot = isHot
	stats.ScheduledStart = &startTime
	stats.FollowerCount = followerCount

	// 加入对应ZSET
	zsetKey := dao.ColdLiveStreamZSET
	if isHot {
		zsetKey = dao.HotLiveStreamZSET
	}

	// 使用时间戳作为score
	score := float64(startTime.Unix())

	if err := s.redis.ZAdd(ctx, zsetKey, redis.Z{
		Score:  score,
		Member: liveStreamID,
	}).Err(); err != nil {
		return fmt.Errorf("加入ZSET失败: %w", err)
	}

	// 保存状态
	if err := s.saveStats(ctx, stats); err != nil {
		// 回滚：从ZSET中移除
		s.redis.ZRem(ctx, zsetKey, liveStreamID)
		return fmt.Errorf("保存状态失败: %w", err)
	}

	return nil
}

// StartLive 开始直播
// 从ZSET移除，如果是热门直播间则加入live_now集合
func (s *LiveStreamStatsService) StartLive(ctx context.Context, liveStreamID int64) error {
	// 获取当前状态
	stats, err := s.GetStats(ctx, liveStreamID)
	if err != nil {
		return fmt.Errorf("获取状态失败: %w", err)
	}

	if stats == nil {
		return fmt.Errorf("直播间状态不存在")
	}

	// 从冷门和热门ZSET中都尝试移除（确保清理干净）
	s.redis.ZRem(ctx, dao.ColdLiveStreamZSET, liveStreamID)
	s.redis.ZRem(ctx, dao.HotLiveStreamZSET, liveStreamID)

	// 如果是热门直播间，加入live_now集合
	if stats.IsHot {
		if err := s.redis.SAdd(ctx, dao.HotLiveNowSet, liveStreamID).Err(); err != nil {
			return fmt.Errorf("加入live_now集合失败: %w", err)
		}
	}

	// 更新状态为live；StartedAt 是 pending-reminder 的真实 session key。
	now := time.Now()
	stats.Status = "live"
	stats.StartedAt = &now
	stats.ScheduledStart = nil // 清空计划开播时间

	if err := s.saveStats(ctx, stats); err != nil {
		// 回滚：从live_now移除
		s.redis.SRem(ctx, dao.HotLiveNowSet, liveStreamID)
		return fmt.Errorf("保存状态失败: %w", err)
	}

	return nil
}

// EndLive 结束直播
// 清理缓存，从live_now集合移除
func (s *LiveStreamStatsService) EndLive(ctx context.Context, liveStreamID int64) error {
	// 从live_now集合移除
	s.redis.SRem(ctx, dao.HotLiveNowSet, liveStreamID)

	// 从ZSET中移除（如果还存在）
	s.redis.ZRem(ctx, dao.ColdLiveStreamZSET, liveStreamID)
	s.redis.ZRem(ctx, dao.HotLiveStreamZSET, liveStreamID)

	if stats, err := s.GetStats(ctx, liveStreamID); err == nil && stats != nil {
		stats.Status = "ended"
		stats.StartedAt = nil
		_ = s.saveStats(ctx, stats)
	}

	// 删除缓存
	key := s.getStatsKey(liveStreamID)
	if err := s.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("删除缓存失败: %w", err)
	}

	return nil
}

// migrateZSET 迁移ZSET（从冷->热 或 热->冷）
func (s *LiveStreamStatsService) migrateZSET(ctx context.Context, liveStreamID int64, wasHot, nowHot bool, scheduledStart time.Time) error {
	var oldZSET, newZSET string

	if wasHot {
		oldZSET = dao.HotLiveStreamZSET
		newZSET = dao.ColdLiveStreamZSET
	} else {
		oldZSET = dao.ColdLiveStreamZSET
		newZSET = dao.HotLiveStreamZSET
	}

	score := float64(scheduledStart.Unix())

	// 使用事务确保原子性
	pipe := s.redis.TxPipeline()

	// 从旧ZSET移除
	pipe.ZRem(ctx, oldZSET, liveStreamID)

	// 加入新ZSET
	pipe.ZAdd(ctx, newZSET, redis.Z{
		Score:  score,
		Member: liveStreamID,
	})

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("ZSET迁移事务失败: %w", err)
	}

	return nil
}

// saveStats 保存状态到Redis
func (s *LiveStreamStatsService) saveStats(ctx context.Context, stats *LiveStreamStats) error {
	key := s.getStatsKey(stats.LiveStreamID)

	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("序列化状态失败: %w", err)
	}

	// 缓存24小时
	return s.redis.Set(ctx, key, data, 24*time.Hour).Err()
}

// GetScheduledColdLiveStreams 获取计划开播的冷门直播间（按时间排序）
func (s *LiveStreamStatsService) GetScheduledColdLiveStreams(ctx context.Context, start, end int64) ([]int64, error) {
	startTime := time.Unix(start, 0)
	endTime := time.Unix(end, 0)
	return dao.GetColdLiveStreamsStartingSoon(ctx, startTime, endTime)
}

// GetScheduledHotLiveStreams 获取计划开播的热门直播间（按时间排序）
func (s *LiveStreamStatsService) GetScheduledHotLiveStreams(ctx context.Context, start, end int64) ([]int64, error) {
	startTime := time.Unix(start, 0)
	endTime := time.Unix(end, 0)
	return dao.GetHotLiveStreamsStartingSoon(ctx, startTime, endTime)
}

// GetLiveNowHotStreams 获取正在直播的热门直播间列表
func (s *LiveStreamStatsService) GetLiveNowHotStreams(ctx context.Context) ([]int64, error) {
	return dao.GetHotLiveNowSet(ctx)
}

// RemoveFromScheduledZSET 从计划开播ZSET中移除直播间
func (s *LiveStreamStatsService) RemoveFromScheduledZSET(ctx context.Context, liveStreamID int64) error {
	return dao.RemoveFromZSET(ctx, liveStreamID)
}

package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis ZSET key 常量
const (
	ColdLiveStreamZSET = "live_stream:cold:start_time" // 冷门直播间开播时间索引
	HotLiveStreamZSET  = "live_stream:hot:start_time"   // 热门直播间开播时间索引
	HotLiveNowSet      = "live_stream:hot:live_now"     // 正在直播的热门直播间
	LiveStreamStatsKey = "live_stream:%d:stats"         // 直播间热度状态缓存
	UserFollowedKey    = "user:%d:followed_live_streams" // 用户关注的直播间
)

// AddToColdZSET - 加入冷门开播时间索引
func AddToColdZSET(ctx context.Context, liveStreamID int64, startTime time.Time) error {
	client := GetRedis()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	score := float64(startTime.Unix())
	member := liveStreamID

	return client.ZAdd(ctx, ColdLiveStreamZSET, redis.Z{
		Score:  score,
		Member: member,
	}).Err()
}

// AddToHotZSET - 加入热门开播时间索引
func AddToHotZSET(ctx context.Context, liveStreamID int64, startTime time.Time) error {
	client := GetRedis()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	score := float64(startTime.Unix())
	member := liveStreamID

	return client.ZAdd(ctx, HotLiveStreamZSET, redis.Z{
		Score:  score,
		Member: member,
	}).Err()
}

// RemoveFromZSET - 从ZSET移除（同时从冷门和热门ZSET移除）
func RemoveFromZSET(ctx context.Context, liveStreamID int64) error {
	client := GetRedis()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	// 使用pipeline批量执行移除操作
	pipe := client.Pipeline()
	pipe.ZRem(ctx, ColdLiveStreamZSET, liveStreamID)
	pipe.ZRem(ctx, HotLiveStreamZSET, liveStreamID)
	pipe.SRem(ctx, HotLiveNowSet, liveStreamID)

	_, err := pipe.Exec(ctx)
	return err
}

// GetColdLiveStreamsStartingSoon - 查询即将开播的冷门直播间 (start, end)
func GetColdLiveStreamsStartingSoon(ctx context.Context, start, end time.Time) ([]int64, error) {
	client := GetRedis()
	if client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	min := float64(start.Unix())
	max := float64(end.Unix())

	result, err := client.ZRangeByScore(ctx, ColdLiveStreamZSET, &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", min),
		Max: fmt.Sprintf("%f", max),
	}).Result()
	if err != nil {
		return nil, err
	}

	// 转换string到int64
	liveStreamIDs := make([]int64, 0, len(result))
	for _, s := range result {
		var id int64
		if _, err := fmt.Sscanf(s, "%d", &id); err == nil {
			liveStreamIDs = append(liveStreamIDs, id)
		}
	}

	return liveStreamIDs, nil
}

// GetHotLiveStreamsStartingSoon - 查询即将开播的热门直播间
func GetHotLiveStreamsStartingSoon(ctx context.Context, start, end time.Time) ([]int64, error) {
	client := GetRedis()
	if client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	min := float64(start.Unix())
	max := float64(end.Unix())

	result, err := client.ZRangeByScore(ctx, HotLiveStreamZSET, &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", min),
		Max: fmt.Sprintf("%f", max),
	}).Result()
	if err != nil {
		return nil, err
	}

	// 转换string到int64
	liveStreamIDs := make([]int64, 0, len(result))
	for _, s := range result {
		var id int64
		if _, err := fmt.Sscanf(s, "%d", &id); err == nil {
			liveStreamIDs = append(liveStreamIDs, id)
		}
	}

	return liveStreamIDs, nil
}

// AddToHotLiveNowSet - 加入正在直播的热门集合
func AddToHotLiveNowSet(ctx context.Context, liveStreamID int64) error {
	client := GetRedis()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	return client.SAdd(ctx, HotLiveNowSet, liveStreamID).Err()
}

// RemoveFromHotLiveNowSet - 从正在直播集合移除
func RemoveFromHotLiveNowSet(ctx context.Context, liveStreamID int64) error {
	client := GetRedis()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	return client.SRem(ctx, HotLiveNowSet, liveStreamID).Err()
}

// GetHotLiveNowSet - 获取正在直播的热门直播间
func GetHotLiveNowSet(ctx context.Context) ([]int64, error) {
	client := GetRedis()
	if client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	result, err := client.SMembers(ctx, HotLiveNowSet).Result()
	if err != nil {
		return nil, err
	}

	// 转换string到int64
	liveStreamIDs := make([]int64, 0, len(result))
	for _, s := range result {
		var id int64
		if _, err := fmt.Sscanf(s, "%d", &id); err == nil {
			liveStreamIDs = append(liveStreamIDs, id)
		}
	}

	return liveStreamIDs, nil
}

// GetUserFollowedLiveStreams - 获取用户关注的直播间
func GetUserFollowedLiveStreams(ctx context.Context, userID int64) ([]int64, error) {
	client := GetRedis()
	if client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	key := fmt.Sprintf(UserFollowedKey, userID)
	result, err := client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	// 转换string到int64
	liveStreamIDs := make([]int64, 0, len(result))
	for _, s := range result {
		var id int64
		if _, err := fmt.Sscanf(s, "%d", &id); err == nil {
			liveStreamIDs = append(liveStreamIDs, id)
		}
	}

	return liveStreamIDs, nil
}

// AddUserFollowedLiveStream - 用户关注直播间
func AddUserFollowedLiveStream(ctx context.Context, userID, liveStreamID int64) error {
	client := GetRedis()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	key := fmt.Sprintf(UserFollowedKey, userID)
	return client.SAdd(ctx, key, liveStreamID).Err()
}

// RemoveUserFollowedLiveStream - 用户取消关注
func RemoveUserFollowedLiveStream(ctx context.Context, userID, liveStreamID int64) error {
	client := GetRedis()
	if client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	key := fmt.Sprintf(UserFollowedKey, userID)
	return client.SRem(ctx, key, liveStreamID).Err()
}
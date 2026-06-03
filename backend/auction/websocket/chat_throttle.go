package websocket

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ThrottleConfig 频控配置
type ThrottleConfig struct {
	UserMax      int
	UserInterval time.Duration
	RoomMax      int
	RoomInterval time.Duration
}

// ChatThrottle 基于 Redis 的双层频控
type ChatThrottle struct {
	rdb *redis.Client
	cfg ThrottleConfig
}

// NewChatThrottle 创建频控器
func NewChatThrottle(rdb *redis.Client, cfg ThrottleConfig) *ChatThrottle {
	return &ChatThrottle{rdb: rdb, cfg: cfg}
}

// Allow 校验当前用户在指定直播间是否可发送
// 返回 0 表示通过，ChatErrCodeRateLimited 表示被拒
func (t *ChatThrottle) Allow(ctx context.Context, userID, liveStreamID int64) int {
	// 用户级
	userKey := fmt.Sprintf("chat:rate:user:%d", userID)
	if !t.incrAndCheck(ctx, userKey, t.cfg.UserMax, t.cfg.UserInterval) {
		return ChatErrCodeRateLimited
	}
	// 房间级
	roomKey := fmt.Sprintf("chat:rate:room:%d", liveStreamID)
	if !t.incrAndCheck(ctx, roomKey, t.cfg.RoomMax, t.cfg.RoomInterval) {
		return ChatErrCodeRateLimited
	}
	return 0
}

// incrAndCheck 原子递增并比较；首次写入时设置 TTL
func (t *ChatThrottle) incrAndCheck(ctx context.Context, key string, max int, ttl time.Duration) bool {
	const script = `
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return current
`
	current, err := t.rdb.Eval(ctx, script, []string{key}, ttl.Milliseconds()).Int64()
	if err != nil {
		// Redis 故障时降级放行，避免直播间静音
		return true
	}
	return current <= int64(max)
}

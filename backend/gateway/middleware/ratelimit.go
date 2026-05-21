package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/redis/go-redis/v9"
)

// RateLimiter 限流器
type RateLimiter struct {
	redis   *redis.Client
	limit   int           // 限制次数
	window  time.Duration // 时间窗口
	keyFunc func(ctx context.Context, c *app.RequestContext) string
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Redis  *redis.Client
	Limit  int           // 时间窗口内最大请求数
	Window time.Duration // 时间窗口
}

// NewRateLimiter 创建限流器
func NewRateLimiter(cfg *RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		redis:  cfg.Redis,
		limit:  cfg.Limit,
		window: cfg.Window,
		keyFunc: func(ctx context.Context, c *app.RequestContext) string {
			// 默认使用 IP 作为限流 key
			return fmt.Sprintf("ratelimit:%s", c.ClientIP())
		},
	}
}

// Middleware 返回限流中间件
func (rl *RateLimiter) Middleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		key := rl.keyFunc(ctx, c)

		// 使用滑动窗口算法实现限流
		count, err := rl.redis.Incr(ctx, key).Result()
		if err != nil {
			// Redis 错误时不阻塞请求
			c.Next(ctx)
			return
		}

		// 第一次请求时设置过期时间
		if count == 1 {
			rl.redis.Expire(ctx, key, rl.window)
		}

		// 超过限制返回 429
		if count > int64(rl.limit) {
			c.JSON(429, map[string]interface{}{
				"code":    429,
				"message": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}

		c.Next(ctx)
	}
}

// IPRateLimit IP 级别限流中间件
func IPRateLimit(redis *redis.Client, limit int, window time.Duration) app.HandlerFunc {
	limiter := NewRateLimiter(&RateLimitConfig{
		Redis:  redis,
		Limit:  limit,
		Window: window,
	})
	return limiter.Middleware()
}

// PathRateLimit 路径级别限流中间件
func PathRateLimit(redis *redis.Client, limit int, window time.Duration) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		key := fmt.Sprintf("ratelimit:%s:%s", c.ClientIP(), string(c.URI().Path()))

		count, err := redis.Incr(ctx, key).Result()
		if err != nil {
			c.Next(ctx)
			return
		}

		if count == 1 {
			redis.Expire(ctx, key, window)
		}

		if count > int64(limit) {
			c.JSON(429, map[string]interface{}{
				"code":    429,
				"message": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}

		c.Next(ctx)
	}
}

// TokenBucketRateLimit 令牌桶限流（适用于出价接口）
func TokenBucketRateLimit(redis *redis.Client, rate int, burst int) app.HandlerFunc {
	// rate: 每秒生成的令牌数
	// burst: 桶容量

	return func(ctx context.Context, c *app.RequestContext) {
		key := fmt.Sprintf("tokenbucket:%s", c.ClientIP())

		// 简化的令牌桶实现：使用 Redis 的 DECR 和 TTL
		// 实际生产环境建议使用更完善的令牌桶算法
		tokens, err := redis.Get(ctx, key).Int()
		if err == redis.Nil {
			// 首次请求，初始化令牌桶
			redis.Set(ctx, key, burst-1, time.Second)
			c.Next(ctx)
			return
		}

		if tokens <= 0 {
			c.JSON(429, map[string]interface{}{
				"code":    429,
				"message": "出价过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}

		redis.Decr(ctx, key)
		c.Next(ctx)
	}
}

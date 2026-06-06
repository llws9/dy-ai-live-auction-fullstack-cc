package dao

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr           string
	Password       string
	DB             int
	PoolSize       int
	MinIdleConns   int
	MaxIdleConns   int
	MaxActiveConns int
}

// InitRedis 初始化 Redis 连接（从参数）
func InitRedis(addr, password string) (*redis.Client, error) {
	client := redis.NewClient(redisOptions(RedisConfig{Addr: addr, Password: password}))

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect redis: %w", err)
	}

	RedisClient = client
	log.Printf("Redis connected successfully: %s", addr)
	return client, nil
}

// InitRedisWithConfig 初始化 Redis 连接（从配置结构体）
func InitRedisWithConfig(cfg *RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(redisOptions(*cfg))

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect redis: %w", err)
	}

	RedisClient = client
	log.Printf("Redis connected successfully: %s", cfg.Addr)
	return client, nil
}

func redisOptions(cfg RedisConfig) *redis.Options {
	poolSize := cfg.PoolSize
	if poolSize <= 0 {
		poolSize = 128
	}
	minIdle := cfg.MinIdleConns
	if minIdle < 0 {
		minIdle = 0
	}
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = poolSize
	}
	maxActive := cfg.MaxActiveConns
	if maxActive <= 0 {
		maxActive = poolSize
	}
	if minIdle > poolSize {
		minIdle = poolSize
	}
	if maxIdle > maxActive {
		maxIdle = maxActive
	}

	return &redis.Options{
		Addr:               cfg.Addr,
		Password:           cfg.Password,
		DB:                 cfg.DB,
		PoolSize:           poolSize,
		MinIdleConns:       minIdle,
		MaxIdleConns:       maxIdle,
		MaxActiveConns:     maxActive,
		MaxConcurrentDials: min(16, poolSize),
		PoolTimeout:        time.Second,
		DialTimeout:        time.Second,
		ReadTimeout:        2 * time.Second,
		WriteTimeout:       2 * time.Second,
	}
}

// GetRedis 获取 Redis 客户端
func GetRedis() *redis.Client {
	return RedisClient
}

// InitRedisFromEnv 从环境变量初始化 Redis 连接
func InitRedisFromEnv() (*redis.Client, error) {
	addr := getEnvOrDefault("REDIS_ADDR", "localhost:6379")
	password := getEnvOrDefault("REDIS_PASSWORD", "")
	return InitRedis(addr, password)
}

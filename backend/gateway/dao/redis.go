package dao

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

// InitRedisFromEnv 从环境变量初始化 Redis 连接
func InitRedisFromEnv() (*redis.Client, error) {
	addr := getEnvOrDefault("REDIS_ADDR", "localhost:6379")
	password := getEnvOrDefault("REDIS_PASSWORD", "")

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect redis: %w", err)
	}

	RedisClient = client
	log.Println("Redis connected successfully")
	return client, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

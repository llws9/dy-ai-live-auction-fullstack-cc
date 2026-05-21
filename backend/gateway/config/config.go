package config

import "os"

// Config 网关配置
type Config struct {
	Server    ServerConfig
	Services  ServicesConfig
	RateLimit RateLimitConfig
	JWT       JWTConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string
}

// ServicesConfig 后端服务配置
type ServicesConfig struct {
	ProductURL string
	AuctionURL string
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	RequestsPerSecond int
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret     string
	ExpireTime int // 小时
}

// Load 加载配置
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnvOrDefault("GATEWAY_PORT", ":8080"),
		},
		Services: ServicesConfig{
			ProductURL: getEnvOrDefault("PRODUCT_SERVICE_URL", "http://localhost:8081"),
			AuctionURL: getEnvOrDefault("AUCTION_SERVICE_URL", "http://localhost:8082"),
		},
		RateLimit: RateLimitConfig{
			RequestsPerSecond: 1000,
		},
		JWT: JWTConfig{
			Secret:     getEnvOrDefault("JWT_SECRET", "your-secret-key-change-in-production"),
			ExpireTime: 24,
		},
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

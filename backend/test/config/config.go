package config

import (
	"os"
	"strconv"
)

// Config test-service 应用配置
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Target   TargetConfig
	Security SecurityConfig
	Mock     MockConfig
	Chaos    ChaosConfig
	Cleanup  CleanupConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	HTTPPort string // test-service 主端口（默认 :18090）
	WSPort   string // WebSocket 独立端口（默认 :18092）
}

// DatabaseConfig 数据库配置（复用 auction 库即可，建独立表）
type DatabaseConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	Name         string
	MaxIdleConns int
	MaxOpenConns int
}

// TargetConfig 被测目标地址
type TargetConfig struct {
	GatewayURL string // 默认走 gateway
	AuctionURL string // 直连 auction，自检用
}

// SecurityConfig 测试服务调用被测系统所需的鉴权配置。
type SecurityConfig struct {
	JWTSecret     string
	InternalToken string
}

// MockConfig Mock Partner 配置
type MockConfig struct {
	PartnerPort string // Mock Partner 端口（默认 :18091）
}

// ChaosConfig 混沌测试配置
type ChaosConfig struct {
	ToxiproxyURL string // toxiproxy admin URL
}

// CleanupConfig 清理策略
type CleanupConfig struct {
	RetentionDays int // test_results 保留天数
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{HTTPPort: ":18090", WSPort: ":18092"},
		Database: DatabaseConfig{
			Host:         "localhost",
			Port:         3306,
			User:         "root",
			Password:     "",
			Name:         "auction",
			MaxIdleConns: 5,
			MaxOpenConns: 20,
		},
		Target: TargetConfig{
			GatewayURL: "http://localhost:8080",
			AuctionURL: "http://localhost:8082",
		},
		Security: SecurityConfig{
			JWTSecret:     "your-secret-key-change-in-production",
			InternalToken: "",
		},
		Mock: MockConfig{PartnerPort: ":18091"},
		Chaos: ChaosConfig{
			ToxiproxyURL: "http://localhost:8474",
		},
		Cleanup: CleanupConfig{RetentionDays: 7},
	}
}

// LoadFromEnv 从环境变量加载配置（可被 Nacos 之后接管）
func LoadFromEnv() *Config {
	cfg := DefaultConfig()

	cfg.Server.HTTPPort = getEnvOrDefault("TEST_HTTP_PORT", cfg.Server.HTTPPort)
	cfg.Server.WSPort = getEnvOrDefault("TEST_WS_PORT", cfg.Server.WSPort)

	cfg.Database.Host = getEnvOrDefault("DB_HOST", cfg.Database.Host)
	if v := getEnvInt("DB_PORT"); v != nil {
		cfg.Database.Port = *v
	}
	cfg.Database.User = getEnvOrDefault("DB_USER", cfg.Database.User)
	cfg.Database.Password = getEnvOrDefault("DB_PASSWORD", cfg.Database.Password)
	cfg.Database.Name = getEnvOrDefault("DB_NAME", cfg.Database.Name)

	cfg.Target.GatewayURL = getEnvOrDefault("TEST_GATEWAY_URL", cfg.Target.GatewayURL)
	cfg.Target.AuctionURL = getEnvOrDefault("TEST_AUCTION_URL", cfg.Target.AuctionURL)
	cfg.Security.JWTSecret = getEnvOrDefault("JWT_SECRET", cfg.Security.JWTSecret)
	cfg.Security.InternalToken = getEnvOrDefault("INTERNAL_API_TOKEN", cfg.Security.InternalToken)

	cfg.Mock.PartnerPort = getEnvOrDefault("TEST_MOCK_PARTNER_PORT", cfg.Mock.PartnerPort)
	cfg.Chaos.ToxiproxyURL = getEnvOrDefault("TEST_TOXIPROXY_URL", cfg.Chaos.ToxiproxyURL)

	if v := getEnvInt("TEST_RETENTION_DAYS"); v != nil {
		cfg.Cleanup.RetentionDays = *v
	}
	return cfg
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string) *int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return &n
		}
	}
	return nil
}

package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"

	nacospkg "gateway-service/pkg/nacos"
)

// Config 网关配置
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Services   ServicesConfig   `yaml:"services"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
	JWT        JWTConfig        `yaml:"jwt"`
	Redis      RedisConfig      `yaml:"redis"`
	Database   DatabaseConfig   `yaml:"database"`
	GrowthBook GrowthBookConfig `yaml:"growthbook"`
}

// GrowthBookConfig A/B测试配置
type GrowthBookConfig struct {
	APIHost   string `yaml:"api_host"`
	ClientKey string `yaml:"client_key"`
	SecretKey string `yaml:"secret_key"`
	Enabled   bool   `yaml:"enabled"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `yaml:"port"`
}

// ServicesConfig 后端服务配置
type ServicesConfig struct {
	ProductURL    string `yaml:"product_url"`
	AuctionURL    string `yaml:"auction_url"`
	TestURL       string `yaml:"test_url"`       // test-service HTTP 入口
	TestWSURL     string `yaml:"test_ws_url"`    // test-service WS 入口（前端直连）
	InternalToken string `yaml:"internal_token"` // service-to-service internal auth token
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	RequestsPerSecond int `yaml:"requests_per_second"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret     string `yaml:"secret"`
	ExpireTime string `yaml:"expire_time"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	PoolSize int    `yaml:"pool_size"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
}

// Load 从环境变量加载配置（本地开发）
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnvOrDefault("GATEWAY_PORT", ":8080"),
		},
		Services: ServicesConfig{
			ProductURL:    getEnvOrDefault("PRODUCT_SERVICE_URL", "http://localhost:8081"),
			AuctionURL:    getEnvOrDefault("AUCTION_SERVICE_URL", "http://localhost:8082"),
			TestURL:       getEnvOrDefault("TEST_SERVICE_URL", "http://localhost:18090"),
			TestWSURL:     getEnvOrDefault("TEST_SERVICE_WS_URL", "ws://localhost:18092"),
			InternalToken: getEnvOrDefault("INTERNAL_API_TOKEN", ""),
		},
		RateLimit: RateLimitConfig{
			RequestsPerSecond: 1000,
		},
		JWT: JWTConfig{
			Secret:     getEnvOrDefault("JWT_SECRET", "your-secret-key-change-in-production"),
			ExpireTime: "24h",
		},
		Redis: RedisConfig{
			Addr:     getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
			Password: getEnvOrDefault("REDIS_PASSWORD", ""),
			PoolSize: 100,
		},
		Database: DatabaseConfig{
			Host:     getEnvOrDefault("DB_HOST", "localhost"),
			Port:     getEnvOrDefault("DB_PORT", "3306"),
			User:     getEnvOrDefault("DB_USER", "root"),
			Password: getEnvOrDefault("DB_PASSWORD", ""),
			Name:     getEnvOrDefault("DB_NAME", "auction"),
		},
		GrowthBook: GrowthBookConfig{
			APIHost:   getEnvOrDefault("GROWTHBOOK_API_HOST", "http://localhost:3200"),
			ClientKey: getEnvOrDefault("GROWTHBOOK_CLIENT_KEY", "dev-client-key"),
			SecretKey: getEnvOrDefault("GROWTHBOOK_SECRET_KEY", "dev-secret-key"),
			Enabled:   getEnvOrDefault("GROWTHBOOK_ENABLED", "true") == "true",
		},
	}
}

// LoadFromNacos 从 Nacos 配置中心加载配置
func LoadFromNacos() (*Config, *nacospkg.ConfigLoader, error) {
	// 从环境变量获取 Nacos 连接信息
	nacosCfg := nacospkg.GetConfigFromEnv()
	group, dataId := nacospkg.GetServiceConfigInfo()

	// 创建 Nacos 客户端
	client, err := nacospkg.NewNacosClient(nacosCfg)
	if err != nil {
		log.Printf("Failed to connect Nacos, falling back to env config: %v", err)
		return Load(), nil, err
	}

	// 创建配置加载器
	loader := nacospkg.NewConfigLoader(client, group, dataId)

	// 加载配置
	cfg := &Config{}
	if err := loader.Load(cfg); err != nil {
		log.Printf("Failed to load config from Nacos, falling back to env config: %v", err)
		return Load(), nil, err
	}

	log.Printf("Config loaded from Nacos: [group=%s, dataId=%s]", group, dataId)
	injectRuntimeSecrets(cfg)
	return cfg, loader, nil
}

// LoadFromNacosWithFallback 从 Nacos 加载配置，失败时使用环境变量
func LoadFromNacosWithFallback() (*Config, *nacospkg.ConfigLoader) {
	cfg, loader, err := LoadFromNacos()
	if err != nil {
		log.Printf("Using fallback config from environment variables")
		return Load(), nil
	}
	return cfg, loader
}

// LoadFromYAML 从 YAML 内容加载配置
func LoadFromYAML(content string) (*Config, error) {
	cfg := &Config{}
	if err := yaml.Unmarshal([]byte(content), cfg); err != nil {
		return nil, err
	}
	injectRuntimeSecrets(cfg)
	return cfg, nil
}

func injectRuntimeSecrets(cfg *Config) {
	cfg.Server.Port = getEnvOrDefault("GATEWAY_PORT", cfg.Server.Port)
	cfg.Services.ProductURL = getEnvOrDefault("PRODUCT_SERVICE_URL", cfg.Services.ProductURL)
	cfg.Services.AuctionURL = getEnvOrDefault("AUCTION_SERVICE_URL", cfg.Services.AuctionURL)
	cfg.Services.TestURL = getEnvOrDefault("TEST_SERVICE_URL", cfg.Services.TestURL)
	cfg.Services.TestWSURL = getEnvOrDefault("TEST_SERVICE_WS_URL", cfg.Services.TestWSURL)
	if token := os.Getenv("INTERNAL_API_TOKEN"); token != "" {
		cfg.Services.InternalToken = token
	}
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.JWT.Secret = secret
	}
	cfg.Database.Host = getEnvOrDefault("DB_HOST", cfg.Database.Host)
	cfg.Database.Port = getEnvOrDefault("DB_PORT", cfg.Database.Port)
	cfg.Database.User = getEnvOrDefault("DB_USER", cfg.Database.User)
	cfg.Database.Password = getEnvOrDefault("DB_PASSWORD", cfg.Database.Password)
	cfg.Database.Name = getEnvOrDefault("DB_NAME", cfg.Database.Name)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

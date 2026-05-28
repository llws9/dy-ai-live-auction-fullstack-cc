package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"

	nacospkg "product-service/pkg/nacos"
)

// Config Product 服务配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Services ServicesConfig `yaml:"services"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `yaml:"port"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	Name         string `yaml:"name"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	PoolSize int    `yaml:"pool_size"`
}

// ServicesConfig 外部服务配置
type ServicesConfig struct {
	AuctionServiceURL string `yaml:"auction_service_url"`
}

// Load 从环境变量加载配置（本地开发）
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnvOrDefault("PRODUCT_SERVICE_PORT", ":8081"),
		},
		Database: DatabaseConfig{
			Host:         getEnvOrDefault("DB_HOST", "localhost"),
			Port:         3306,
			User:         getEnvOrDefault("DB_USER", "root"),
			Password:     getEnvOrDefault("DB_PASSWORD", ""),
			Name:         getEnvOrDefault("DB_NAME", "auction"),
			MaxIdleConns: 10,
			MaxOpenConns: 100,
		},
		Redis: RedisConfig{
			Addr:     getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
			Password: getEnvOrDefault("REDIS_PASSWORD", ""),
			PoolSize: 50,
		},
		Services: ServicesConfig{
			AuctionServiceURL: getEnvOrDefault("AUCTION_SERVICE_URL", "http://localhost:8082"),
		},
	}
}

// LoadFromNacos 从 Nacos 配置中心加载配置
func LoadFromNacos() (*Config, *nacospkg.ConfigLoader, error) {
	nacosCfg := nacospkg.GetConfigFromEnv()
	group, dataId := nacospkg.GetServiceConfigInfo()

	client, err := nacospkg.NewNacosClient(nacosCfg)
	if err != nil {
		log.Printf("Failed to connect Nacos, falling back to env config: %v", err)
		return Load(), nil, err
	}

	loader := nacospkg.NewConfigLoader(client, group, dataId)

	cfg := &Config{}
	if err := loader.Load(cfg); err != nil {
		log.Printf("Failed to load config from Nacos: %v", err)
		return Load(), nil, err
	}

	log.Printf("Config loaded from Nacos: [group=%s, dataId=%s]", group, dataId)
	return cfg, loader, nil
}

// LoadFromNacosWithFallback 从 Nacos 加载配置，失败时使用环境变量
func LoadFromNacosWithFallback() (*Config, *nacospkg.ConfigLoader) {
	cfg, loader, err := LoadFromNacos()
	if err != nil {
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
	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
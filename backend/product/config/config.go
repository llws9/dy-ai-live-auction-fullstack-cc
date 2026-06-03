package config

import (
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	nacospkg "product-service/pkg/nacos"
)

const (
	defaultLLMProvider  = "doubao"
	defaultLLMTimeoutMs = 60000
	defaultArkBaseURL   = "https://ark.cn-beijing.volces.com/api/v3"
	defaultArkModel     = "doubao-seed-1-6-vision-250815"
)

// Config Product 服务配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Services ServicesConfig `yaml:"services"`
	LLM      LLMConfig      `yaml:"llm"`
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

// LLMConfig LLM 总配置。
type LLMConfig struct {
	Provider  string       `yaml:"provider"`
	TimeoutMs int          `yaml:"timeout_ms"`
	Doubao    DoubaoConfig `yaml:"doubao"`
}

// DoubaoConfig 豆包/方舟配置。
type DoubaoConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
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
		LLM: LLMConfig{
			Provider:  getEnvOrDefault("LLM_PROVIDER", defaultLLMProvider),
			TimeoutMs: defaultLLMTimeoutMs,
			Doubao: DoubaoConfig{
				BaseURL: getEnvOrDefault("ARK_BASE_URL", defaultArkBaseURL),
				APIKey:  os.Getenv("ARK_API_KEY"),
				Model:   getEnvOrDefault("ARK_MODEL", defaultArkModel),
			},
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
	ApplyDefaults(cfg)

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
	ApplyDefaults(cfg)
	return cfg, nil
}

// ApplyDefaults fills missing optional config values after YAML/Nacos loading.
func ApplyDefaults(cfg *Config) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.LLM.Provider) == "" {
		cfg.LLM.Provider = defaultLLMProvider
	}
	if cfg.LLM.TimeoutMs <= 0 {
		cfg.LLM.TimeoutMs = defaultLLMTimeoutMs
	}
	if strings.TrimSpace(cfg.LLM.Doubao.BaseURL) == "" {
		cfg.LLM.Doubao.BaseURL = defaultArkBaseURL
	}
	if strings.TrimSpace(cfg.LLM.Doubao.Model) == "" {
		cfg.LLM.Doubao.Model = defaultArkModel
	}
}

// ResolveLLMSecrets 把 yaml 中 ${ARK_API_KEY} 占位符或空 key 用环境变量替换。
// Nacos/yaml 配置不写明文 key，由 K8s secret 通过环境变量注入容器。
func ResolveLLMSecrets(cfg *Config) {
	k := strings.TrimSpace(cfg.LLM.Doubao.APIKey)
	if k == "" || (strings.HasPrefix(k, "${") && strings.HasSuffix(k, "}")) {
		cfg.LLM.Doubao.APIKey = os.Getenv("ARK_API_KEY")
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

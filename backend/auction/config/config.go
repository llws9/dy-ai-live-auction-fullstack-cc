package config

import (
	"os"
	"strconv"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
	JWT      JWTConfig
	SkyLamp  SkyLampConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	HTTPPort    string
	WSPort      string
	ReadTimeout int
	WriteTimeout int
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// RabbitMQConfig RabbitMQ配置
type RabbitMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	VHost    string
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret     string
	ExpireHours int
}

// SkyLampConfig 点天灯配置
type SkyLampConfig struct {
	Enabled           bool // 是否启用点天灯功能
	MaxPriceOffset    int  // 上限偏移量X（相对于开启时的价格）
	MinFollowInterval int  // 自动跟价最小间隔（毫秒）
	MaxAutoBidCount   int  // 单次点天灯最大自动跟价次数
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			HTTPPort:     "8082",
			WSPort:       "8083",
			ReadTimeout:  10,
			WriteTimeout: 10,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     "3306",
			User:     "root",
			Password: "",
			Database: "auction",
		},
		Redis: RedisConfig{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		RabbitMQ: RabbitMQConfig{
			Host:     "localhost",
			Port:     "5672",
			User:     "guest",
			Password: "guest",
			VHost:    "/",
		},
		JWT: JWTConfig{
			Secret:      "your-secret-key-change-in-production",
			ExpireHours: 24,
		},
		SkyLamp: DefaultSkyLampConfig(),
	}
}

// DefaultSkyLampConfig 返回默认的点天灯配置
func DefaultSkyLampConfig() SkyLampConfig {
	return SkyLampConfig{
		Enabled:           true,
		MaxPriceOffset:    10000,
		MinFollowInterval: 500,
		MaxAutoBidCount:   100,
	}
}

// LoadFromEnv 从环境变量加载配置
func LoadFromEnv() *Config {
	cfg := DefaultConfig()

	// 服务器配置
	cfg.Server.HTTPPort = getEnvOrDefault("HTTP_PORT", cfg.Server.HTTPPort)
	cfg.Server.WSPort = getEnvOrDefault("WS_PORT", cfg.Server.WSPort)
	if v := getEnvInt("HTTP_READ_TIMEOUT"); v != nil {
		cfg.Server.ReadTimeout = *v
	}
	if v := getEnvInt("HTTP_WRITE_TIMEOUT"); v != nil {
		cfg.Server.WriteTimeout = *v
	}

	// 数据库配置
	cfg.Database.Host = getEnvOrDefault("DB_HOST", cfg.Database.Host)
	cfg.Database.Port = getEnvOrDefault("DB_PORT", cfg.Database.Port)
	cfg.Database.User = getEnvOrDefault("DB_USER", cfg.Database.User)
	cfg.Database.Password = getEnvOrDefault("DB_PASSWORD", cfg.Database.Password)
	cfg.Database.Database = getEnvOrDefault("DB_NAME", cfg.Database.Database)

	// Redis配置
	cfg.Redis.Host = getEnvOrDefault("REDIS_HOST", cfg.Redis.Host)
	cfg.Redis.Port = getEnvOrDefault("REDIS_PORT", cfg.Redis.Port)
	cfg.Redis.Password = getEnvOrDefault("REDIS_PASSWORD", cfg.Redis.Password)
	if v := getEnvInt("REDIS_DB"); v != nil {
		cfg.Redis.DB = *v
	}

	// RabbitMQ配置
	cfg.RabbitMQ.Host = getEnvOrDefault("RABBITMQ_HOST", cfg.RabbitMQ.Host)
	cfg.RabbitMQ.Port = getEnvOrDefault("RABBITMQ_PORT", cfg.RabbitMQ.Port)
	cfg.RabbitMQ.User = getEnvOrDefault("RABBITMQ_USER", cfg.RabbitMQ.User)
	cfg.RabbitMQ.Password = getEnvOrDefault("RABBITMQ_PASSWORD", cfg.RabbitMQ.Password)
	cfg.RabbitMQ.VHost = getEnvOrDefault("RABBITMQ_VHOST", cfg.RabbitMQ.VHost)

	// JWT配置
	cfg.JWT.Secret = getEnvOrDefault("JWT_SECRET", cfg.JWT.Secret)
	if v := getEnvInt("JWT_EXPIRE_HOURS"); v != nil {
		cfg.JWT.ExpireHours = *v
	}

	// 点天灯配置
	if v := getEnvBool("SKYLAMP_ENABLED"); v != nil {
		cfg.SkyLamp.Enabled = *v
	}
	if v := getEnvInt("SKYLAMP_MAX_PRICE_OFFSET"); v != nil {
		cfg.SkyLamp.MaxPriceOffset = *v
	}
	if v := getEnvInt("SKYLAMP_MIN_FOLLOW_INTERVAL"); v != nil {
		cfg.SkyLamp.MinFollowInterval = *v
	}
	if v := getEnvInt("SKYLAMP_MAX_AUTO_BID_COUNT"); v != nil {
		cfg.SkyLamp.MaxAutoBidCount = *v
	}

	return cfg
}

// getEnvOrDefault 获取环境变量或返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取整数类型的环境变量
func getEnvInt(key string) *int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return &intVal
		}
	}
	return nil
}

// getEnvBool 获取布尔类型的环境变量
func getEnvBool(key string) *bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return &boolVal
		}
	}
	return nil
}
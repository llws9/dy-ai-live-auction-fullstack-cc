package dao

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"product-service/pkg/dbmetrics"
)

var DB *gorm.DB

// Config 数据库配置
type Config struct {
	Host         string
	Port         string
	User         string
	Password     string
	Database     string
	MaxIdleConns int
	MaxOpenConns int
}

// InitDB 初始化数据库连接（从参数）
func InitDB(cfg *Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// 设置连接池参数
	maxIdle := cfg.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 10
	}
	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 100
	}
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetMaxOpenConns(maxOpen)

	// 注册 SQL 查询耗时监控插件
	sqlMetrics := dbmetrics.InitSQLMetrics("product-service")
	if err := db.Use(dbmetrics.NewGormMetricsPlugin("product-service", sqlMetrics)); err != nil {
		log.Printf("Warning: Failed to register GORM metrics plugin: %v", err)
	} else {
		log.Println("GORM metrics plugin registered successfully")
	}

	DB = db
	log.Printf("Database connected successfully: %s@%s:%s/%s", cfg.User, cfg.Host, cfg.Port, cfg.Database)
	return db, nil
}

// GetDB 获取数据库连接
func GetDB() *gorm.DB {
	return DB
}

// InitDBFromEnv 从环境变量初始化数据库连接
func InitDBFromEnv() (*gorm.DB, error) {
	cfg := &Config{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		Port:     getEnvOrDefault("DB_PORT", "3306"),
		User:     getEnvOrDefault("DB_USER", "root"),
		Password: getEnvOrDefault("DB_PASSWORD", ""),
		Database: getEnvOrDefault("DB_NAME", "auction"),
	}
	return InitDB(cfg)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

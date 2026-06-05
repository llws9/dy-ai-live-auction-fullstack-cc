package dao

import (
	"fmt"
	"log"
	"os"
	"strings"

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

// EnsureAuctionRuleProductScopeSchema aligns legacy MySQL databases with the
// current product-scoped auction_rules contract.
func EnsureAuctionRuleProductScopeSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if db.Dialector.Name() != "mysql" {
		return nil
	}
	if !db.Migrator().HasTable("auction_rules") {
		return nil
	}

	type foreignKey struct {
		ConstraintName string `gorm:"column:CONSTRAINT_NAME"`
	}
	var fks []foreignKey
	if err := db.Raw(`
		SELECT CONSTRAINT_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'auction_rules'
		  AND COLUMN_NAME = 'auction_id'
		  AND REFERENCED_TABLE_NAME IS NOT NULL
	`).Scan(&fks).Error; err != nil {
		return fmt.Errorf("inspect auction_rules.auction_id foreign keys: %w", err)
	}
	for _, fk := range fks {
		if fk.ConstraintName == "" {
			continue
		}
		if err := db.Exec("ALTER TABLE auction_rules DROP FOREIGN KEY " + quoteMySQLIdentifier(fk.ConstraintName)).Error; err != nil {
			return fmt.Errorf("drop auction_rules foreign key %s: %w", fk.ConstraintName, err)
		}
	}

	type columnInfo struct {
		IsNullable string `gorm:"column:IS_NULLABLE"`
	}
	var col columnInfo
	tx := db.Raw(`
		SELECT IS_NULLABLE
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'auction_rules'
		  AND COLUMN_NAME = 'auction_id'
	`).Scan(&col)
	if tx.Error != nil {
		return fmt.Errorf("inspect auction_rules.auction_id column: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return nil
	}
	if strings.EqualFold(col.IsNullable, "NO") {
		if err := db.Exec("ALTER TABLE auction_rules MODIFY COLUMN auction_id BIGINT NULL COMMENT 'legacy auction id; product_id is source of truth'").Error; err != nil {
			return fmt.Errorf("make auction_rules.auction_id nullable: %w", err)
		}
	}
	return nil
}

func quoteMySQLIdentifier(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
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

package dao

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"test-service/config"
	"test-service/model"
)

// InitDB 用 MySQL 创建主 DB 句柄；启动时自动迁移 test-service 的表
func InitDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)

	// AutoMigrate：保证 test-service 自身的表在共享库（auction）中存在
	// 仅会创建/补列，不会破坏 auction 业务表
	if err := db.AutoMigrate(&model.TestResult{}, &model.TestSeedData{}); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}
	return db, nil
}

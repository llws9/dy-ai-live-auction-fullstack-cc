package dao

import (
	"fmt"
	"sync/atomic"
	"testing"

	"auction-service/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var dbCounter int64

// setupTestDB 返回一个隔离的内存 sqlite GORM 实例，并迁移一口价相关表。
// 每次调用使用独立的命名内存库，避免并行用例间数据串扰。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:dao_test_%d?mode=memory&cache=shared", atomic.AddInt64(&dbCounter, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.FixedPriceItem{},
		&model.FixedPricePurchase{},
		&model.UserBalance{},
	))
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

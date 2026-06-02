package service

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"auction-service/dao"
	"auction-service/model"
)

// setupFailFastService 构造一个可直接操控底层 miniredis 的 service，
// 用于验证 Redis 宕机时 Purchase 的 fail-fast 语义（T12 零依赖等价验证）。
func setupFailFastService(t *testing.T) (*FixedPriceService, *miniredis.Miniredis, *gorm.DB) {
	t.Helper()

	svcDBCounter++
	dsn := "file:failfast_test_" + itoa(svcDBCounter) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.FixedPriceItem{}, &model.FixedPricePurchase{}, &model.UserBalance{}))
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	svc := NewFixedPriceService(
		db,
		dao.NewFixedPriceItemDAO(db),
		dao.NewFixedPricePurchaseDAO(db),
		dao.NewUserBalanceDAO(db),
		NewStockGuard(rdb),
		NewIdemStore(rdb),
		&fakeStreamOwner{owners: nil},
		&fakeProductChecker{},
		nil,
		nil,
	)
	return svc, mr, db
}

// TestPurchase_RedisDown_FailFast 验证：Redis 宕机时 Purchase 立即报错，
// 且不触碰余额/库存（fail-fast 优于静默降级）。
func TestPurchase_RedisDown_FailFast(t *testing.T) {
	svc, mr, db := setupFailFastService(t)
	ctx := context.Background()

	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	setBalance(t, svc, 100, decimal.NewFromInt(1000))

	// 模拟 Redis 宕机。
	mr.Close()

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	require.Error(t, err, "Redis 宕机时 Purchase 必须报错，而非静默降级")

	// 余额未被扣减：仍为 1000。
	var bal model.UserBalance
	require.NoError(t, db.WithContext(ctx).Where("user_id = ?", int64(100)).First(&bal).Error)
	assert.True(t, bal.AvailableAmount.Equal(decimal.NewFromInt(1000)),
		"fail-fast 路径不得扣余额，实际=%s", bal.AvailableAmount.String())

	// 未写入任何购买记录。
	var cnt int64
	require.NoError(t, db.WithContext(ctx).Model(&model.FixedPricePurchase{}).
		Where("item_id = ?", item.ID).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt, "fail-fast 路径不得写购买记录")
}

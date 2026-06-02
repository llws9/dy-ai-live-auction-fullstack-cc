package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStockGuard_Success(t *testing.T) {
	rdb := setupTestRedis(t)
	ctx := context.Background()
	sg := NewStockGuard(rdb)

	require.NoError(t, sg.Init(ctx, 7001, 100))

	res, err := sg.TryAcquire(ctx, 7001, 100)
	require.NoError(t, err)
	assert.Equal(t, StockResultSuccess, res)
}

func TestStockGuard_AlreadyBought(t *testing.T) {
	rdb := setupTestRedis(t)
	ctx := context.Background()
	sg := NewStockGuard(rdb)

	require.NoError(t, sg.Init(ctx, 7001, 100))

	res1, err := sg.TryAcquire(ctx, 7001, 100)
	require.NoError(t, err)
	assert.Equal(t, StockResultSuccess, res1)

	res2, err := sg.TryAcquire(ctx, 7001, 100)
	require.NoError(t, err)
	assert.Equal(t, StockResultAlreadyBought, res2)
}

func TestStockGuard_SoldOut(t *testing.T) {
	rdb := setupTestRedis(t)
	ctx := context.Background()
	sg := NewStockGuard(rdb)

	require.NoError(t, sg.Init(ctx, 7001, 1))

	res1, err := sg.TryAcquire(ctx, 7001, 100)
	require.NoError(t, err)
	assert.Equal(t, StockResultSuccess, res1)

	res2, err := sg.TryAcquire(ctx, 7001, 200)
	require.NoError(t, err)
	assert.Equal(t, StockResultSoldOut, res2)
}

func TestStockGuard_Uninitialized(t *testing.T) {
	rdb := setupTestRedis(t)
	ctx := context.Background()
	sg := NewStockGuard(rdb)

	res, err := sg.TryAcquire(ctx, 9999, 100)
	require.NoError(t, err)
	assert.Equal(t, StockResultUninitialized, res)
}

func TestStockGuard_Compensate(t *testing.T) {
	rdb := setupTestRedis(t)
	ctx := context.Background()
	sg := NewStockGuard(rdb)

	require.NoError(t, sg.Init(ctx, 7001, 5))

	res, err := sg.TryAcquire(ctx, 7001, 100)
	require.NoError(t, err)
	assert.Equal(t, StockResultSuccess, res)

	require.NoError(t, sg.Compensate(ctx, 7001, 100))

	stock, err := rdb.Get(ctx, "fp:stock:7001").Int()
	require.NoError(t, err)
	assert.Equal(t, 5, stock)

	isMember, err := rdb.SIsMember(ctx, "fp:bought:7001", "100").Result()
	require.NoError(t, err)
	assert.False(t, isMember)
}

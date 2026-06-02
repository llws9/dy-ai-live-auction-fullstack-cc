package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdemStore_FirstInsertReturnsZero(t *testing.T) {
	rdb := setupTestRedis(t)
	store := NewIdemStore(rdb)

	id, ok, err := store.GetOrInit(context.Background(), 100, 7001, "key-first", 0)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, int64(0), id)
}

func TestIdemStore_SecondCallReturnsStoredID(t *testing.T) {
	rdb := setupTestRedis(t)
	store := NewIdemStore(rdb)
	ctx := context.Background()

	_, ok, err := store.GetOrInit(ctx, 100, 7001, "key-second", 0)
	require.NoError(t, err)
	require.False(t, ok)

	require.NoError(t, store.Persist(ctx, 100, 7001, "key-second", 88001))

	id, ok, err := store.GetOrInit(ctx, 100, 7001, "key-second", 0)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, int64(88001), id)
}

func TestIdemStore_ValidateUUID(t *testing.T) {
	s := &IdemStore{}
	assert.False(t, s.IsValidKey("not-a-uuid"))
	assert.True(t, s.IsValidKey("550e8400-e29b-41d4-a716-446655440000"))
}

func TestIdemStore_TTL(t *testing.T) {
	rdb := setupTestRedis(t)
	store := NewIdemStore(rdb)
	ctx := context.Background()

	require.NoError(t, store.Persist(ctx, 100, 7001, "key-ttl", 99001))

	ttl, err := rdb.TTL(ctx, "fp:idem:100:7001:key-ttl").Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, 9*time.Minute)
	assert.LessOrEqual(t, ttl, 10*time.Minute)
}

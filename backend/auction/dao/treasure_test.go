package dao

import (
	"context"
	"testing"

	"auction-service/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreasureDAO_AddWatchSeconds_AccumulatesPerDate(t *testing.T) {
	db := setupTestDB(t)
	d := NewTreasureDAO(db)
	ctx := context.Background()

	total, err := d.AddWatchSeconds(ctx, 100, "2026-06-09", 30)
	require.NoError(t, err)
	assert.Equal(t, 30, total)

	total, err = d.AddWatchSeconds(ctx, 100, "2026-06-09", 30)
	require.NoError(t, err)
	assert.Equal(t, 60, total)

	total, err = d.AddWatchSeconds(ctx, 100, "2026-06-10", 30)
	require.NoError(t, err)
	assert.Equal(t, 30, total)
}

func TestTreasureDAO_GetWatchSeconds_NoRecordReturnsZero(t *testing.T) {
	db := setupTestDB(t)
	d := NewTreasureDAO(db)
	ctx := context.Background()

	secs, err := d.GetWatchSeconds(ctx, 999, "2026-06-09")
	require.NoError(t, err)
	assert.Equal(t, 0, secs)
}

func TestTreasureDAO_GetCoinBalance_NoRecordReturnsZero(t *testing.T) {
	db := setupTestDB(t)
	d := NewTreasureDAO(db)
	ctx := context.Background()

	bal, err := d.GetCoinBalance(ctx, 999)
	require.NoError(t, err)
	assert.Equal(t, int64(0), bal)
}

func TestTreasureDAO_ListClaimedTiers(t *testing.T) {
	db := setupTestDB(t)
	d := NewTreasureDAO(db)
	ctx := context.Background()

	require.NoError(t, db.Create(&model.TreasureClaim{
		UserID: 100, StatDate: "2026-06-09", Tier: 0, Coins: 100,
	}).Error)

	tiers, err := d.ListClaimedTiers(ctx, 100, "2026-06-09")
	require.NoError(t, err)
	assert.Equal(t, map[int8]bool{0: true}, tiers)
}

func TestTreasureDAO_ClaimTx_Success(t *testing.T) {
	db := setupTestDB(t)
	d := NewTreasureDAO(db)
	ctx := context.Background()

	newBalance, err := d.ClaimTx(ctx, 100, "2026-06-09", 1, 300)
	require.NoError(t, err)
	assert.Equal(t, int64(300), newBalance)

	bal, err := d.GetCoinBalance(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(300), bal)
}

func TestTreasureDAO_ClaimTx_DuplicateIsIdempotent(t *testing.T) {
	db := setupTestDB(t)
	d := NewTreasureDAO(db)
	ctx := context.Background()

	_, err := d.ClaimTx(ctx, 100, "2026-06-09", 1, 300)
	require.NoError(t, err)

	_, err = d.ClaimTx(ctx, 100, "2026-06-09", 1, 300)
	assert.ErrorIs(t, err, ErrAlreadyClaimed)

	bal, err := d.GetCoinBalance(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(300), bal)
}

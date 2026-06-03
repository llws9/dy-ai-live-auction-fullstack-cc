package dao

import (
	"context"
	"testing"

	"auction-service/model"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserBalanceDAO_DeductWithTx_Success(t *testing.T) {
	db := setupTestDB(t)
	d := NewUserBalanceDAO(db)
	ctx := context.Background()

	require.NoError(t, db.Create(&model.UserBalance{
		UserID:          100,
		AvailableAmount: decimal.NewFromInt(200),
		Currency:        "CNY",
	}).Error)

	var affected int64
	err := db.Transaction(func(tx *gorm.DB) error {
		var e error
		affected, e = d.DeductWithTx(ctx, tx, 100, decimal.NewFromInt(99))
		return e
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	avail, _, _, hit, err := d.GetByUserID(ctx, 100)
	require.NoError(t, err)
	require.True(t, hit)
	assert.Equal(t, "101.00", avail.StringFixed(2))
}

func TestUserBalanceDAO_DeductWithTx_Insufficient(t *testing.T) {
	db := setupTestDB(t)
	d := NewUserBalanceDAO(db)
	ctx := context.Background()

	require.NoError(t, db.Create(&model.UserBalance{
		UserID:          100,
		AvailableAmount: decimal.NewFromInt(50),
		Currency:        "CNY",
	}).Error)

	var affected int64
	err := db.Transaction(func(tx *gorm.DB) error {
		var e error
		affected, e = d.DeductWithTx(ctx, tx, 100, decimal.NewFromInt(99))
		return e
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), affected, "余额不足时不应扣减")

	avail, _, _, hit, err := d.GetByUserID(ctx, 100)
	require.NoError(t, err)
	require.True(t, hit)
	assert.Equal(t, "50.00", avail.StringFixed(2), "余额不足时金额不变")
}

func TestUserBalanceDAO_DeductWithTx_ExactBalance(t *testing.T) {
	db := setupTestDB(t)
	d := NewUserBalanceDAO(db)
	ctx := context.Background()

	require.NoError(t, db.Create(&model.UserBalance{
		UserID:          100,
		AvailableAmount: decimal.NewFromInt(99),
		Currency:        "CNY",
	}).Error)

	var affected int64
	err := db.Transaction(func(tx *gorm.DB) error {
		var e error
		affected, e = d.DeductWithTx(ctx, tx, 100, decimal.NewFromInt(99))
		return e
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected, "余额恰好相等应允许扣减")

	avail, _, _, _, err := d.GetByUserID(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, "0.00", avail.StringFixed(2))
}

func TestUserBalanceDAO_DeductWithTx_NoRecord(t *testing.T) {
	db := setupTestDB(t)
	d := NewUserBalanceDAO(db)
	ctx := context.Background()

	var affected int64
	err := db.Transaction(func(tx *gorm.DB) error {
		var e error
		affected, e = d.DeductWithTx(ctx, tx, 999, decimal.NewFromInt(1))
		return e
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), affected, "无余额记录视为余额不足")
}

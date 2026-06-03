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

func TestFixedPricePurchaseDAO_Insert_UniqueViolation(t *testing.T) {
	db := setupTestDB(t)
	d := NewFixedPricePurchaseDAO(db)
	ctx := context.Background()

	p1 := &model.FixedPricePurchase{ItemID: 7001, UserID: 100, Price: decimal.NewFromInt(99)}
	require.NoError(t, d.Insert(ctx, p1))

	p2 := &model.FixedPricePurchase{ItemID: 7001, UserID: 100, Price: decimal.NewFromInt(199)}
	err := d.Insert(ctx, p2)
	assert.ErrorIs(t, err, ErrAlreadyBought)
}

func TestFixedPricePurchaseDAO_GetByItemAndUser(t *testing.T) {
	db := setupTestDB(t)
	d := NewFixedPricePurchaseDAO(db)
	ctx := context.Background()

	require.NoError(t, d.Insert(ctx, &model.FixedPricePurchase{ItemID: 7001, UserID: 100, Price: decimal.NewFromInt(99)}))

	got, err := d.GetByItemAndUser(ctx, 7001, 100)
	require.NoError(t, err)
	assert.Equal(t, "99.00", got.Price.StringFixed(2))
	assert.Equal(t, int64(100), got.UserID)

	_, err = d.GetByItemAndUser(ctx, 7001, 999)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

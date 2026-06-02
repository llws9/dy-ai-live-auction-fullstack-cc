package service

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixedPriceService_List_ValidatesAndCreates(t *testing.T) {
	svc := setupFixedPriceService(t)
	ctx := context.Background()
	item, err := svc.ListItem(ctx, ListItemReq{
		LiveStreamID: 1001, ProductID: 5001, CreatorID: 100,
		Price: decimal.NewFromFloat(99), TotalStock: 50, MaxPerUser: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, 50, item.RemainingStock)
	remain, err := svc.stock.Remaining(ctx, item.ID)
	require.NoError(t, err)
	assert.Equal(t, 50, remain)
}

func TestFixedPriceService_List_RejectsInvalidPrice(t *testing.T) {
	svc := setupFixedPriceService(t)
	_, err := svc.ListItem(context.Background(), ListItemReq{
		LiveStreamID: 1, ProductID: 1, CreatorID: 1,
		Price: decimal.Zero, TotalStock: 10,
	})
	assert.ErrorIs(t, err, ErrInvalidParam)
}

func TestFixedPriceService_List_RejectsExcessiveStock(t *testing.T) {
	svc := setupFixedPriceService(t)
	_, err := svc.ListItem(context.Background(), ListItemReq{
		LiveStreamID: 1, ProductID: 1, CreatorID: 1,
		Price: decimal.NewFromInt(10), TotalStock: 10001,
	})
	assert.ErrorIs(t, err, ErrInvalidParam)
}

func TestFixedPriceService_List_RejectsNonOwner(t *testing.T) {
	svc := setupFixedPriceServiceWithStream(t, 1001, 100) // owner=100
	_, err := svc.ListItem(context.Background(), ListItemReq{
		LiveStreamID: 1001, ProductID: 5001, CreatorID: 999, // not owner
		Price: decimal.NewFromInt(99), TotalStock: 10,
	})
	assert.ErrorIs(t, err, ErrNotStreamOwner)
}

func TestFixedPriceService_List_RejectsMissingProduct(t *testing.T) {
	db := setupServiceDB(t)
	rdb := setupTestRedis(t)
	svc := NewFixedPriceService(
		db,
		newItemDAO(db), newPurchaseDAO(db), newBalanceDAO(db),
		NewStockGuard(rdb), NewIdemStore(rdb),
		&fakeStreamOwner{owners: nil},
		&fakeProductChecker{missing: map[int64]bool{5001: true}},
		nil,
	)
	_, err := svc.ListItem(context.Background(), ListItemReq{
		LiveStreamID: 1001, ProductID: 5001, CreatorID: 100,
		Price: decimal.NewFromInt(99), TotalStock: 10,
	})
	assert.ErrorIs(t, err, ErrProductNotFound)
}

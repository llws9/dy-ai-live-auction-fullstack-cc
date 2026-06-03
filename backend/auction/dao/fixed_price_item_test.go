package dao

import (
	"context"
	"testing"

	"auction-service/model"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixedPriceItemDAO_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	d := NewFixedPriceItemDAO(db)
	ctx := context.Background()

	item := &model.FixedPriceItem{
		LiveStreamID:   2001,
		ProductID:      1,
		CreatorID:      10,
		Price:          decimal.NewFromInt(99),
		TotalStock:     100,
		RemainingStock: 100,
		MaxPerUser:     1,
		Status:         model.FixedPriceStatusOnSale,
	}
	require.NoError(t, d.Create(ctx, item))
	require.NotZero(t, item.ID)

	got, err := d.GetByID(ctx, item.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "99.00", got.Price.StringFixed(2))
	assert.Equal(t, model.FixedPriceStatusOnSale, got.Status)
}

func TestFixedPriceItemDAO_UpdateStatus_LegalTransitions(t *testing.T) {
	db := setupTestDB(t)
	d := NewFixedPriceItemDAO(db)
	ctx := context.Background()

	item := &model.FixedPriceItem{
		LiveStreamID:   2001,
		ProductID:      1,
		CreatorID:      10,
		Price:          decimal.NewFromInt(99),
		TotalStock:     100,
		RemainingStock: 100,
		MaxPerUser:     1,
		Status:         model.FixedPriceStatusOnSale,
	}
	require.NoError(t, d.Create(ctx, item))

	// on_sale -> sold_out 合法
	require.NoError(t, d.UpdateStatus(ctx, item.ID, model.FixedPriceStatusSoldOut))
	got, err := d.GetByID(ctx, item.ID)
	require.NoError(t, err)
	assert.Equal(t, model.FixedPriceStatusSoldOut, got.Status)

	// sold_out -> on_sale 非法
	err = d.UpdateStatus(ctx, item.ID, model.FixedPriceStatusOnSale)
	assert.ErrorIs(t, err, ErrIllegalStatusTransition)
}

func TestFixedPriceItemDAO_ListByLiveStreamID(t *testing.T) {
	db := setupTestDB(t)
	d := NewFixedPriceItemDAO(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		item := &model.FixedPriceItem{
			LiveStreamID:   2002,
			ProductID:      int64(i + 1),
			CreatorID:      10,
			Price:          decimal.NewFromInt(99),
			TotalStock:     100,
			RemainingStock: 100,
			MaxPerUser:     1,
			Status:         model.FixedPriceStatusOnSale,
		}
		require.NoError(t, d.Create(ctx, item))
	}

	items, err := d.ListByLiveStreamID(ctx, 2002, []model.FixedPriceStatus{model.FixedPriceStatusOnSale})
	require.NoError(t, err)
	assert.Len(t, items, 3)
}

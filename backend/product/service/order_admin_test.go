package service

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"product-service/dao"
	"product-service/model"
)

type fakeUserSummaryProvider struct {
	items map[int64]UserSummary
	err   error
}

func (f fakeUserSummaryProvider) BatchGetUserSummaries(_ context.Context, ids []int64) (map[int64]UserSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	result := make(map[int64]UserSummary, len(ids))
	for _, id := range ids {
		if item, ok := f.items[id]; ok {
			result[id] = item
		}
	}
	return result, nil
}

func newAdminOrderServiceWithSeed(t *testing.T, seed func(db *gorm.DB)) *OrderService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Order{}, &model.Product{}))
	require.NoError(t, db.Exec("DELETE FROM orders").Error)
	require.NoError(t, db.Exec("DELETE FROM products").Error)
	if seed != nil {
		seed(db)
	}
	svc := NewOrderService(dao.NewOrderDAO(db), nil)
	svc.SetAdminOrderDAO(dao.NewOrderAdminDAO(db))
	return svc
}

func TestOrderServiceListAdminOrdersEnrichesBuyerSummaries(t *testing.T) {
	svc := newAdminOrderServiceWithSeed(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.Product{ID: 11, Name: "茶杯"}).Error)
		require.NoError(t, db.Create(&model.Order{ID: 101, AuctionID: 201, ProductID: 11, WinnerID: 901, FinalPrice: decimal.NewFromInt(100), Status: model.OrderStatusPaid}).Error)
	})
	svc.SetUserSummaryProvider(fakeUserSummaryProvider{
		items: map[int64]UserSummary{
			901: {ID: 901, Username: "张三", Avatar: "https://cdn/u901.png"},
		},
	})

	got, err := svc.ListAdminOrdersScoped(context.Background(), nil, nil, nil, "", 1, 20)

	require.NoError(t, err)
	require.Len(t, got.Items, 1)
	assert.Equal(t, "张三", got.Items[0].UserName)
	assert.Equal(t, "https://cdn/u901.png", got.Items[0].UserAvatar)
}

func TestOrderServiceListAdminOrdersKeepsOrdersWhenBuyerSummaryFails(t *testing.T) {
	svc := newAdminOrderServiceWithSeed(t, func(db *gorm.DB) {
		require.NoError(t, db.Create(&model.Product{ID: 11, Name: "茶杯"}).Error)
		require.NoError(t, db.Create(&model.Order{ID: 101, AuctionID: 201, ProductID: 11, WinnerID: 901, FinalPrice: decimal.NewFromInt(100), Status: model.OrderStatusPaid}).Error)
	})
	svc.SetUserSummaryProvider(fakeUserSummaryProvider{err: errors.New("auction unavailable")})

	got, err := svc.ListAdminOrdersScoped(context.Background(), nil, nil, nil, "", 1, 20)

	require.NoError(t, err)
	require.Len(t, got.Items, 1)
	assert.Empty(t, got.Items[0].UserName)
	assert.Empty(t, got.Items[0].UserAvatar)
}

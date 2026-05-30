package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBalanceProvider struct {
	calledUser int64
	available  decimal.Decimal
	frozen     decimal.Decimal
	currency   string
	hit        bool
	err        error
}

func (f *fakeBalanceProvider) GetByUserID(_ context.Context, userID int64) (available, frozen decimal.Decimal, currency string, hit bool, err error) {
	f.calledUser = userID
	return f.available, f.frozen, f.currency, f.hit, f.err
}

func TestBuildUserBalanceResponse(t *testing.T) {
	ctx := context.Background()

	t.Run("returns zero balance when no record exists", func(t *testing.T) {
		fp := &fakeBalanceProvider{hit: false}
		got, err := BuildUserBalanceResponse(ctx, fp, 42)
		require.NoError(t, err)
		assert.Equal(t, int64(42), fp.calledUser)
		assert.True(t, got.AvailableAmount.IsZero())
		assert.True(t, got.FrozenAmount.IsZero())
		assert.Equal(t, "CNY", got.Currency)
	})

	t.Run("returns real values when record exists", func(t *testing.T) {
		fp := &fakeBalanceProvider{
			available: decimal.NewFromFloat(1234.56),
			frozen:    decimal.NewFromFloat(78.90),
			currency:  "CNY",
			hit:       true,
		}
		got, err := BuildUserBalanceResponse(ctx, fp, 42)
		require.NoError(t, err)
		assert.True(t, got.AvailableAmount.Equal(decimal.NewFromFloat(1234.56)))
		assert.True(t, got.FrozenAmount.Equal(decimal.NewFromFloat(78.90)))
		assert.Equal(t, "CNY", got.Currency)
	})

	t.Run("falls back to CNY when currency is empty", func(t *testing.T) {
		fp := &fakeBalanceProvider{available: decimal.NewFromInt(100), hit: true, currency: ""}
		got, err := BuildUserBalanceResponse(ctx, fp, 42)
		require.NoError(t, err)
		assert.Equal(t, "CNY", got.Currency)
	})

	t.Run("propagates dao error", func(t *testing.T) {
		fp := &fakeBalanceProvider{err: errors.New("db down")}
		_, err := BuildUserBalanceResponse(ctx, fp, 42)
		require.Error(t, err)
	})

	t.Run("rejects zero/negative user id", func(t *testing.T) {
		fp := &fakeBalanceProvider{}
		_, err := BuildUserBalanceResponse(ctx, fp, 0)
		require.Error(t, err)
		_, err = BuildUserBalanceResponse(ctx, fp, -1)
		require.Error(t, err)
	})
}

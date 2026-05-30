package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeBalanceProvider 模拟 BalanceDAO.GetByUserID 行为，覆盖三类返回路径：
//   - 命中 (hit=true)
//   - 未命中 (hit=false, err=nil)
//   - DB 故障 (err!=nil)
type fakeBalanceProvider struct {
	calledUser int64
	available  float64
	frozen     float64
	currency   string
	hit        bool
	err        error
}

func (f *fakeBalanceProvider) GetByUserID(_ context.Context, userID int64) (available, frozen float64, currency string, hit bool, err error) {
	f.calledUser = userID
	return f.available, f.frozen, f.currency, f.hit, f.err
}

// TestBuildUserBalanceResponse 验证 T3.1 余额查询编排逻辑：
//   - 无记录 → 返回零余额 + 默认 currency=CNY（spec A / F-A2）
//   - 有记录 → 透传 dao 字段
//   - dao 故障 → 错误冒泡（handler 转 5xx）
//   - 非法 user_id → 拒绝
func TestBuildUserBalanceResponse(t *testing.T) {
	ctx := context.Background()

	t.Run("returns zero balance when no record exists", func(t *testing.T) {
		fp := &fakeBalanceProvider{hit: false}
		got, err := BuildUserBalanceResponse(ctx, fp, 42)
		require.NoError(t, err)
		assert.Equal(t, int64(42), fp.calledUser)
		assert.Equal(t, 0.0, got.AvailableAmount)
		assert.Equal(t, 0.0, got.FrozenAmount)
		assert.Equal(t, "CNY", got.Currency)
	})

	t.Run("returns real values when record exists", func(t *testing.T) {
		fp := &fakeBalanceProvider{
			available: 1234.56,
			frozen:    78.90,
			currency:  "CNY",
			hit:       true,
		}
		got, err := BuildUserBalanceResponse(ctx, fp, 42)
		require.NoError(t, err)
		assert.Equal(t, 1234.56, got.AvailableAmount)
		assert.Equal(t, 78.90, got.FrozenAmount)
		assert.Equal(t, "CNY", got.Currency)
	})

	t.Run("falls back to CNY when currency is empty", func(t *testing.T) {
		fp := &fakeBalanceProvider{available: 100, hit: true, currency: ""}
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

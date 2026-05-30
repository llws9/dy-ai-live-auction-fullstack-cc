package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAddressStore 模拟 AddressStore，用于覆盖 T3.2 编排逻辑的所有路径：
//   - 命中 / 未命中（hit=false → handler 转 404）
//   - 数量超限
//   - 默认互斥事务调用
//   - 越权（id 不属于当前 user_id → hit=false）
type fakeAddressStore struct {
	listResult  []AddressView
	listErr     error
	count       int64
	countErr    error
	getResult   *AddressView
	getHit      bool
	getErr      error
	createErr   error
	createCalls []AddressMutation
	updateErr   error
	updateHit   bool
	updateCalls []AddressMutation
	deleteErr   error
	deleteHit   bool
	deleteCalls []int64
	setDefaultErr error
	setDefaultHit bool
	setDefaultCalls []int64
}

func (f *fakeAddressStore) List(_ context.Context, _ int64) ([]AddressView, error) {
	return f.listResult, f.listErr
}
func (f *fakeAddressStore) Count(_ context.Context, _ int64) (int64, error) {
	return f.count, f.countErr
}
func (f *fakeAddressStore) Get(_ context.Context, _ int64, _ int64) (*AddressView, bool, error) {
	return f.getResult, f.getHit, f.getErr
}
func (f *fakeAddressStore) Create(_ context.Context, m AddressMutation) (*AddressView, error) {
	f.createCalls = append(f.createCalls, m)
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &AddressView{
		ID:            1001,
		RecipientName: m.RecipientName,
		Phone:         m.Phone,
		Province:      m.Province,
		City:          m.City,
		District:      m.District,
		Detail:        m.Detail,
		IsDefault:     m.IsDefault,
	}, nil
}
func (f *fakeAddressStore) Update(_ context.Context, id, _ int64, m AddressMutation) (bool, error) {
	m.ID = id
	f.updateCalls = append(f.updateCalls, m)
	return f.updateHit, f.updateErr
}
func (f *fakeAddressStore) Delete(_ context.Context, id, _ int64) (bool, error) {
	f.deleteCalls = append(f.deleteCalls, id)
	return f.deleteHit, f.deleteErr
}
func (f *fakeAddressStore) SetDefault(_ context.Context, id, _ int64) (bool, error) {
	f.setDefaultCalls = append(f.setDefaultCalls, id)
	return f.setDefaultHit, f.setDefaultErr
}

// TestValidateAddressInput 锁定字段约束（spec A / F-A3 §字段约束）。
func TestValidateAddressInput(t *testing.T) {
	base := AddressInput{
		RecipientName: "张三",
		Phone:         "13800000000",
		Province:      "北京市",
		City:          "北京市",
		District:      "海淀区",
		Detail:        "中关村大街 1 号",
	}

	t.Run("accepts valid input", func(t *testing.T) {
		require.NoError(t, ValidateAddressInput(base))
	})

	t.Run("rejects empty recipient_name", func(t *testing.T) {
		in := base
		in.RecipientName = ""
		require.Error(t, ValidateAddressInput(in))
	})

	t.Run("rejects invalid phone", func(t *testing.T) {
		in := base
		in.Phone = "12345"
		require.Error(t, ValidateAddressInput(in))
	})

	t.Run("rejects oversize detail", func(t *testing.T) {
		in := base
		in.Detail = string(make([]byte, 200))
		require.Error(t, ValidateAddressInput(in))
	})
}

// TestBuildListAddresses 验证 GET /users/me/addresses 编排：
//   - 透传 store 列表
//   - 非法 user_id 拒绝
func TestBuildListAddresses(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("returns store result", func(t *testing.T) {
		fs := &fakeAddressStore{listResult: []AddressView{{ID: 1, IsDefault: true, CreatedAt: now}}}
		got, err := BuildListAddresses(ctx, fs, 42)
		require.NoError(t, err)
		assert.Len(t, got.Items, 1)
		assert.Equal(t, int64(1), got.Total)
	})

	t.Run("rejects zero user id", func(t *testing.T) {
		fs := &fakeAddressStore{}
		_, err := BuildListAddresses(ctx, fs, 0)
		require.Error(t, err)
	})
}

// TestBuildCreateAddress 验证 POST /users/me/addresses 编排：
//   - 校验失败 → ErrAddressInvalid
//   - 超限 (>=20) → ErrAddressLimitExceeded
//   - 首条强制 is_default=true（即使请求体传 false）
//   - is_default=true 透传给 store
func TestBuildCreateAddress(t *testing.T) {
	ctx := context.Background()
	good := AddressInput{
		RecipientName: "张三",
		Phone:         "13800000000",
		Province:      "北京市",
		City:          "北京市",
		District:      "海淀区",
		Detail:        "中关村大街 1 号",
		IsDefault:     false,
	}

	t.Run("rejects invalid input", func(t *testing.T) {
		fs := &fakeAddressStore{}
		bad := good
		bad.Phone = "abc"
		_, err := BuildCreateAddress(ctx, fs, 42, bad)
		require.ErrorIs(t, err, ErrAddressInvalid)
	})

	t.Run("rejects when already 20 addresses", func(t *testing.T) {
		fs := &fakeAddressStore{count: 20}
		_, err := BuildCreateAddress(ctx, fs, 42, good)
		require.ErrorIs(t, err, ErrAddressLimitExceeded)
	})

	t.Run("forces is_default=true on first address", func(t *testing.T) {
		fs := &fakeAddressStore{count: 0}
		_, err := BuildCreateAddress(ctx, fs, 42, good)
		require.NoError(t, err)
		require.Len(t, fs.createCalls, 1)
		assert.True(t, fs.createCalls[0].IsDefault, "首条地址必须强制 is_default=true")
	})

	t.Run("preserves explicit is_default=true", func(t *testing.T) {
		in := good
		in.IsDefault = true
		fs := &fakeAddressStore{count: 5}
		_, err := BuildCreateAddress(ctx, fs, 42, in)
		require.NoError(t, err)
		require.Len(t, fs.createCalls, 1)
		assert.True(t, fs.createCalls[0].IsDefault)
	})
}

// TestBuildUpdateAddress 验证 PUT /users/me/addresses/:id 编排：
//   - 越权 / 不存在（store hit=false）→ ErrAddressNotFound
//   - 校验失败 → ErrAddressInvalid
func TestBuildUpdateAddress(t *testing.T) {
	ctx := context.Background()
	good := AddressInput{
		RecipientName: "张三",
		Phone:         "13800000000",
		Province:      "北京市",
		City:          "北京市",
		District:      "海淀区",
		Detail:        "中关村大街 1 号",
	}

	t.Run("returns 404 semantic when store reports miss", func(t *testing.T) {
		fs := &fakeAddressStore{updateHit: false}
		err := BuildUpdateAddress(ctx, fs, 42, 1001, good)
		require.ErrorIs(t, err, ErrAddressNotFound)
	})

	t.Run("rejects invalid input before store call", func(t *testing.T) {
		fs := &fakeAddressStore{updateHit: true}
		bad := good
		bad.RecipientName = ""
		err := BuildUpdateAddress(ctx, fs, 42, 1001, bad)
		require.ErrorIs(t, err, ErrAddressInvalid)
		assert.Empty(t, fs.updateCalls, "校验失败不应调用 store")
	})

	t.Run("ok when store hits", func(t *testing.T) {
		fs := &fakeAddressStore{updateHit: true}
		err := BuildUpdateAddress(ctx, fs, 42, 1001, good)
		require.NoError(t, err)
		require.Len(t, fs.updateCalls, 1)
		assert.Equal(t, int64(1001), fs.updateCalls[0].ID)
	})
}

// TestBuildDeleteAddress 验证 DELETE /users/me/addresses/:id 编排：
//   - 越权 / 不存在 → ErrAddressNotFound
func TestBuildDeleteAddress(t *testing.T) {
	ctx := context.Background()

	t.Run("returns 404 when store miss", func(t *testing.T) {
		fs := &fakeAddressStore{deleteHit: false}
		err := BuildDeleteAddress(ctx, fs, 42, 1001)
		require.ErrorIs(t, err, ErrAddressNotFound)
	})

	t.Run("ok when store hits", func(t *testing.T) {
		fs := &fakeAddressStore{deleteHit: true}
		err := BuildDeleteAddress(ctx, fs, 42, 1001)
		require.NoError(t, err)
		require.Equal(t, []int64{1001}, fs.deleteCalls)
	})

	t.Run("propagates dao error", func(t *testing.T) {
		fs := &fakeAddressStore{deleteErr: errors.New("db down")}
		err := BuildDeleteAddress(ctx, fs, 42, 1001)
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrAddressNotFound)
	})
}

// TestBuildSetDefaultAddress 验证 POST /users/me/addresses/:id/default 编排：
//   - 越权 / 不存在 → ErrAddressNotFound
//   - 命中触发 store.SetDefault（内部事务清零 + 置位）
func TestBuildSetDefaultAddress(t *testing.T) {
	ctx := context.Background()

	t.Run("returns 404 when store miss", func(t *testing.T) {
		fs := &fakeAddressStore{setDefaultHit: false}
		err := BuildSetDefaultAddress(ctx, fs, 42, 1001)
		require.ErrorIs(t, err, ErrAddressNotFound)
	})

	t.Run("invokes store SetDefault when hit", func(t *testing.T) {
		fs := &fakeAddressStore{setDefaultHit: true}
		err := BuildSetDefaultAddress(ctx, fs, 42, 1001)
		require.NoError(t, err)
		require.Equal(t, []int64{1001}, fs.setDefaultCalls)
	})
}

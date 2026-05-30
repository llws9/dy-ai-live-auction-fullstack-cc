package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"auction-service/model"
)

// fakeUserBatchProvider 模拟 UserDAO.GetByIDs，记录入参便于断言。
type fakeUserBatchProvider struct {
	calledIDs []int64
	users     map[int64]*model.User
	err       error
}

func (f *fakeUserBatchProvider) GetByIDs(_ context.Context, ids []int64) (map[int64]*model.User, error) {
	f.calledIDs = ids
	if f.err != nil {
		return nil, f.err
	}
	return f.users, nil
}

// TestBuildUserSummaries 验证 T3.3 / spec B §4.1 内部接口编排：
//   - 空 ids → 错误
//   - 超长 ids → 错误
//   - 正常返回按入参顺序、缺失 id 跳过、id 重复去重
//   - dao 故障冒泡
func TestBuildUserSummaries(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects empty ids", func(t *testing.T) {
		_, err := BuildUserSummaries(ctx, &fakeUserBatchProvider{}, nil)
		require.Error(t, err)
		_, err = BuildUserSummaries(ctx, &fakeUserBatchProvider{}, []int64{})
		require.Error(t, err)
	})

	t.Run("rejects oversize ids", func(t *testing.T) {
		ids := make([]int64, internalUserBatchMaxIDs+1)
		for i := range ids {
			ids[i] = int64(i + 1)
		}
		_, err := BuildUserSummaries(ctx, &fakeUserBatchProvider{}, ids)
		require.Error(t, err)
	})

	t.Run("returns ordered summaries, skips missing, dedupes", func(t *testing.T) {
		fp := &fakeUserBatchProvider{
			users: map[int64]*model.User{
				1: {ID: 1, Name: "alice", Avatar: "a.png"},
				3: {ID: 3, Name: "carol", Avatar: ""},
			},
		}
		got, err := BuildUserSummaries(ctx, fp, []int64{1, 2, 3, 1})
		require.NoError(t, err)
		require.Len(t, got.Items, 2)
		assert.Equal(t, int64(1), got.Items[0].ID)
		assert.Equal(t, "alice", got.Items[0].Username)
		assert.Equal(t, "a.png", got.Items[0].Avatar)
		assert.Equal(t, int64(3), got.Items[1].ID)
		assert.Equal(t, "carol", got.Items[1].Username)
	})

	t.Run("propagates dao error", func(t *testing.T) {
		fp := &fakeUserBatchProvider{err: errors.New("db down")}
		_, err := BuildUserSummaries(ctx, fp, []int64{1})
		require.Error(t, err)
	})
}

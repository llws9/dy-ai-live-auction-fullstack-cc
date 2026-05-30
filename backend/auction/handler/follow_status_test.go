package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeFollowChecker 模拟 FollowService.IsFollowing。
type fakeFollowChecker struct {
	calledUser   int64
	calledStream int64
	out          bool
	err          error
}

func (f *fakeFollowChecker) IsFollowing(_ context.Context, userID, liveStreamID int64) (bool, error) {
	f.calledUser = userID
	f.calledStream = liveStreamID
	return f.out, f.err
}

// TestBuildFollowStatusResponse 验证 F-B2 follow-status 编排逻辑：
//   - 透传 user_id / live_stream_id 给 checker
//   - service 错误冒泡到 handler（由 handler 转 5xx）
//   - 业务正确路径返回稳定 shape
func TestBuildFollowStatusResponse(t *testing.T) {
	ctx := context.Background()

	t.Run("forwards user/stream to checker and returns is_following=true", func(t *testing.T) {
		fc := &fakeFollowChecker{out: true}
		got, err := BuildFollowStatusResponse(ctx, fc, 42, 101)
		require.NoError(t, err)
		assert.Equal(t, int64(42), fc.calledUser)
		assert.Equal(t, int64(101), fc.calledStream)
		assert.True(t, got.IsFollowing)
	})

	t.Run("returns is_following=false when not followed", func(t *testing.T) {
		fc := &fakeFollowChecker{out: false}
		got, err := BuildFollowStatusResponse(ctx, fc, 42, 999)
		require.NoError(t, err)
		assert.False(t, got.IsFollowing)
	})

	t.Run("propagates checker error", func(t *testing.T) {
		fc := &fakeFollowChecker{err: errors.New("db down")}
		_, err := BuildFollowStatusResponse(ctx, fc, 42, 101)
		require.Error(t, err)
	})

	t.Run("rejects zero/negative ids", func(t *testing.T) {
		fc := &fakeFollowChecker{}
		_, err := BuildFollowStatusResponse(ctx, fc, 0, 101)
		require.Error(t, err)
		_, err = BuildFollowStatusResponse(ctx, fc, 42, 0)
		require.Error(t, err)
	})
}

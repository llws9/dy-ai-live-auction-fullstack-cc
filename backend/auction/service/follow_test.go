package service

import (
	"context"
	"errors"
	"testing"

	"auction-service/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// MockUserLiveStreamFollowDAO 模拟 DAO
type MockUserLiveStreamFollowDAO struct {
	mock.Mock
}

func (m *MockUserLiveStreamFollowDAO) Create(ctx context.Context, follow *model.UserLiveStreamFollow) error {
	args := m.Called(ctx, follow)
	return args.Error(0)
}

func (m *MockUserLiveStreamFollowDAO) Delete(ctx context.Context, userID, liveStreamID int64) error {
	args := m.Called(ctx, userID, liveStreamID)
	return args.Error(0)
}

func (m *MockUserLiveStreamFollowDAO) GetByUserAndLiveStream(ctx context.Context, userID, liveStreamID int64) (*model.UserLiveStreamFollow, error) {
	args := m.Called(ctx, userID, liveStreamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserLiveStreamFollow), args.Error(1)
}

func (m *MockUserLiveStreamFollowDAO) GetUserFollows(ctx context.Context, userID int64, offset, limit int) ([]model.UserLiveStreamFollow, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]model.UserLiveStreamFollow), args.Error(1)
}

func (m *MockUserLiveStreamFollowDAO) CountUserFollows(ctx context.Context, userID int64) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserLiveStreamFollowDAO) UpdateNotificationEnabled(ctx context.Context, userID, liveStreamID int64, enabled bool) error {
	args := m.Called(ctx, userID, liveStreamID, enabled)
	return args.Error(0)
}

func (m *MockUserLiveStreamFollowDAO) GetFollowStats(ctx context.Context, liveStreamID int64) (map[string]int64, error) {
	args := m.Called(ctx, liveStreamID)
	return args.Get(0).(map[string]int64), args.Error(1)
}

// TestFollowService_Follow 测试关注功能
func TestFollowService_Follow(t *testing.T) {
	mockDAO := new(MockUserLiveStreamFollowDAO)
	service := NewFollowService(mockDAO)

	ctx := context.Background()
	userID := int64(1)
	liveStreamID := int64(10)

	t.Run("成功关注", func(t *testing.T) {
		// 模拟未关注
		mockDAO.On("GetByUserAndLiveStream", ctx, userID, liveStreamID).
			Return(nil, gorm.ErrRecordNotFound)

		// 模拟创建成功
		mockDAO.On("Create", ctx, mock.AnythingOfType("*model.UserLiveStreamFollow")).
			Return(nil)

		follow, err := service.Follow(ctx, userID, liveStreamID)

		assert.NoError(t, err)
		assert.NotNil(t, follow)
		assert.Equal(t, userID, follow.UserID)
		assert.Equal(t, liveStreamID, follow.LiveStreamID)
		assert.True(t, follow.NotificationEnabled)

		mockDAO.AssertExpectations(t)
	})

	t.Run("重复关注", func(t *testing.T) {
		mockDAO := new(MockUserLiveStreamFollowDAO)
		service := NewFollowService(mockDAO)

		// 模拟已关注
		existingFollow := &model.UserLiveStreamFollow{
			UserID:              userID,
			LiveStreamID:        liveStreamID,
			NotificationEnabled: true,
		}
		mockDAO.On("GetByUserAndLiveStream", ctx, userID, liveStreamID).
			Return(existingFollow, nil)

		follow, err := service.Follow(ctx, userID, liveStreamID)

		assert.Error(t, err)
		assert.Nil(t, follow)
		assert.Contains(t, err.Error(), "已经关注")

		mockDAO.AssertExpectations(t)
	})
}

// TestFollowService_Unfollow 测试取消关注功能
func TestFollowService_Unfollow(t *testing.T) {
	mockDAO := new(MockUserLiveStreamFollowDAO)
	service := NewFollowService(mockDAO)

	ctx := context.Background()
	userID := int64(1)
	liveStreamID := int64(10)

	t.Run("成功取消关注", func(t *testing.T) {
		mockDAO.On("Delete", ctx, userID, liveStreamID).
			Return(nil)

		err := service.Unfollow(ctx, userID, liveStreamID)

		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("取消关注失败", func(t *testing.T) {
		mockDAO := new(MockUserLiveStreamFollowDAO)
		service := NewFollowService(mockDAO)

		mockDAO.On("Delete", ctx, userID, liveStreamID).
			Return(errors.New("数据库错误"))

		err := service.Unfollow(ctx, userID, liveStreamID)

		assert.Error(t, err)
		mockDAO.AssertExpectations(t)
	})
}

// TestFollowService_GetUserFollows 测试获取用户关注列表
func TestFollowService_GetUserFollows(t *testing.T) {
	mockDAO := new(MockUserLiveStreamFollowDAO)
	service := NewFollowService(mockDAO)

	ctx := context.Background()
	userID := int64(1)

	t.Run("成功获取列表", func(t *testing.T) {
		expectedFollows := []model.UserLiveStreamFollow{
			{UserID: userID, LiveStreamID: 10, NotificationEnabled: true},
			{UserID: userID, LiveStreamID: 11, NotificationEnabled: true},
		}

		mockDAO.On("GetUserFollows", ctx, userID, 0, 20).
			Return(expectedFollows, nil)
		mockDAO.On("CountUserFollows", ctx, userID).
			Return(int64(2), nil)

		follows, total, err := service.GetUserFollows(ctx, userID, 1, 20)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, follows, 2)

		mockDAO.AssertExpectations(t)
	})
}

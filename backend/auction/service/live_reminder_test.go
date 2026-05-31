package service

import (
	"context"
	"testing"

	"auction-service/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLiveSessionResolver struct {
	sessions map[int64]*model.StreamInfo
}

func (r *fakeLiveSessionResolver) GetActiveSession(ctx context.Context, liveStreamID int64) (*model.StreamInfo, error) {
	return r.sessions[liveStreamID], nil
}

type fakeReminderClaimer struct {
	claimableByStream map[int64]bool
}

func (c *fakeReminderClaimer) Claim(ctx context.Context, userID, liveStreamID, startedAt int64) (bool, error) {
	return c.claimableByStream[liveStreamID], nil
}

func TestLiveReminderServiceReturnsFirstClaimableLiveCandidate(t *testing.T) {
	ctx := context.Background()
	followDAO := new(MockUserLiveStreamFollowDAO)
	followDAO.On("GetUserFollows", ctx, int64(100), 0, 50).Return([]model.UserLiveStreamFollow{
		{UserID: 100, LiveStreamID: 1, NotificationEnabled: false},
		{UserID: 100, LiveStreamID: 2, NotificationEnabled: true},
		{UserID: 100, LiveStreamID: 3, NotificationEnabled: true},
	}, nil)

	service := NewLiveReminderService(
		followDAO,
		&fakeLiveSessionResolver{sessions: map[int64]*model.StreamInfo{
			3: {ID: 3, LiveRoomID: 3, Name: "直播间 3", StatusText: "正在直播", StartedAt: 1717000000000},
		}},
		&fakeReminderClaimer{claimableByStream: map[int64]bool{3: true}},
	)

	result, err := service.GetPendingReminder(ctx, 100)

	assert.NoError(t, err)
	assert.True(t, result.HasReminder)
	assert.Equal(t, int64(3), result.Stream.ID)
	followDAO.AssertExpectations(t)
}

func TestLiveReminderServiceContinuesAfterClaimConflict(t *testing.T) {
	ctx := context.Background()
	followDAO := new(MockUserLiveStreamFollowDAO)
	followDAO.On("GetUserFollows", ctx, int64(100), 0, 50).Return([]model.UserLiveStreamFollow{
		{UserID: 100, LiveStreamID: 2, NotificationEnabled: true},
		{UserID: 100, LiveStreamID: 3, NotificationEnabled: true},
	}, nil)

	service := NewLiveReminderService(
		followDAO,
		&fakeLiveSessionResolver{sessions: map[int64]*model.StreamInfo{
			2: {ID: 2, LiveRoomID: 2, Name: "直播间 2", StatusText: "正在直播", StartedAt: 1717000000000},
			3: {ID: 3, LiveRoomID: 3, Name: "直播间 3", StatusText: "正在直播", StartedAt: 1717000010000},
		}},
		&fakeReminderClaimer{claimableByStream: map[int64]bool{2: false, 3: true}},
	)

	result, err := service.GetPendingReminder(ctx, 100)

	assert.NoError(t, err)
	assert.True(t, result.HasReminder)
	assert.Equal(t, int64(3), result.Stream.ID)
	followDAO.AssertExpectations(t)
}

func TestLiveReminderServiceReturnsEmptyWhenNoCandidateCanBeClaimed(t *testing.T) {
	ctx := context.Background()
	followDAO := new(MockUserLiveStreamFollowDAO)
	followDAO.On("GetUserFollows", ctx, int64(100), 0, 50).Return([]model.UserLiveStreamFollow{
		{UserID: 100, LiveStreamID: 2, NotificationEnabled: true},
	}, nil)

	service := NewLiveReminderService(
		followDAO,
		&fakeLiveSessionResolver{sessions: map[int64]*model.StreamInfo{
			2: {ID: 2, LiveRoomID: 2, Name: "直播间 2", StatusText: "正在直播", StartedAt: 1717000000000},
		}},
		&fakeReminderClaimer{claimableByStream: map[int64]bool{2: false}},
	)

	result, err := service.GetPendingReminder(ctx, 100)

	assert.NoError(t, err)
	assert.False(t, result.HasReminder)
	assert.Nil(t, result.Stream)
	followDAO.AssertExpectations(t)
}

func TestLiveReminderServiceScansNextPageWhenFirstPageHasNoLiveCandidate(t *testing.T) {
	ctx := context.Background()
	userID := int64(100)
	firstPage := make([]model.UserLiveStreamFollow, 50)
	for i := range firstPage {
		firstPage[i] = model.UserLiveStreamFollow{
			UserID:              userID,
			LiveStreamID:        int64(i + 1),
			NotificationEnabled: true,
		}
	}
	secondPage := []model.UserLiveStreamFollow{
		{UserID: userID, LiveStreamID: 51, NotificationEnabled: true},
	}

	followDAO := new(MockUserLiveStreamFollowDAO)
	followDAO.On("GetUserFollows", ctx, userID, 0, 50).Return(firstPage, nil).Once()
	followDAO.On("GetUserFollows", ctx, userID, 50, 50).Return(secondPage, nil).Once()

	service := NewLiveReminderService(
		followDAO,
		&fakeLiveSessionResolver{sessions: map[int64]*model.StreamInfo{
			51: {ID: 51, LiveRoomID: 51, Name: "直播间 51", StatusText: "正在直播", StartedAt: 1717000051000},
		}},
		&fakeReminderClaimer{claimableByStream: map[int64]bool{51: true}},
	)

	result, err := service.GetPendingReminder(ctx, userID)

	assert.NoError(t, err)
	require.True(t, result.HasReminder)
	require.NotNil(t, result.Stream)
	assert.Equal(t, int64(51), result.Stream.ID)
	followDAO.AssertExpectations(t)
}

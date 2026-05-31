package service

import (
	"context"
	"testing"

	"auction-service/model"
	"github.com/stretchr/testify/assert"
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

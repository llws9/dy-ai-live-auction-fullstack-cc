package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/service"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/stretchr/testify/require"
)

type flowFollowDAO struct {
	follows []model.UserLiveStreamFollow
}

func (d *flowFollowDAO) Create(ctx context.Context, follow *model.UserLiveStreamFollow) error {
	return nil
}

func (d *flowFollowDAO) Delete(ctx context.Context, userID, liveStreamID int64) error {
	return nil
}

func (d *flowFollowDAO) GetByUserAndLiveStream(ctx context.Context, userID, liveStreamID int64) (*model.UserLiveStreamFollow, error) {
	return nil, nil
}

func (d *flowFollowDAO) GetUserFollows(ctx context.Context, userID int64, offset, limit int) ([]model.UserLiveStreamFollow, error) {
	return d.follows, nil
}

func (d *flowFollowDAO) CountUserFollows(ctx context.Context, userID int64) (int64, error) {
	return int64(len(d.follows)), nil
}

func (d *flowFollowDAO) UpdateNotificationEnabled(ctx context.Context, userID, liveStreamID int64, enabled bool) error {
	return nil
}

func (d *flowFollowDAO) GetFollowStats(ctx context.Context, liveStreamID int64) (map[string]int64, error) {
	return nil, nil
}

type memoryReminderClaimer struct {
	seen map[string]bool
}

func (c *memoryReminderClaimer) Claim(ctx context.Context, userID, liveStreamID, startedAt int64) (bool, error) {
	key := fmt.Sprintf("%d:%d:%d", userID, liveStreamID, startedAt)
	if c.seen[key] {
		return false, nil
	}
	c.seen[key] = true
	return true, nil
}

func TestProductionStartLiveTransitionFeedsPendingReminderOnce(t *testing.T) {
	ctx := context.Background()
	_, err := dao.InitRedis("localhost:6379", "")
	require.NoError(t, err)

	userID := int64(991)
	liveStreamID := time.Now().UnixNano() % 1_000_000_000
	statsService := service.NewLiveStreamStatsService()
	require.NoError(t, statsService.SetScheduledStartTime(ctx, liveStreamID, time.Now().Add(time.Hour), 80))

	startHandler := NewLiveStreamStatsHandler(statsService)
	c := app.NewContext(1)
	c.Params = append(c.Params, param.Param{Key: "id", Value: strconv.FormatInt(liveStreamID, 10)})
	c.Set("user_id", int64(10001))
	c.Set("user_role", 1)

	startHandler.StartLive(ctx, c)

	require.Equal(t, http.StatusOK, c.Response.StatusCode())
	stats, err := statsService.GetStats(ctx, liveStreamID)
	require.NoError(t, err)
	require.NotNil(t, stats.StartedAt)

	reminderService := service.NewLiveReminderService(
		&flowFollowDAO{follows: []model.UserLiveStreamFollow{
			{UserID: userID, LiveStreamID: liveStreamID, NotificationEnabled: true},
		}},
		service.NewLiveStatsSessionResolver(statsService),
		&memoryReminderClaimer{seen: map[string]bool{}},
	)

	first, err := reminderService.GetPendingReminder(ctx, userID)
	require.NoError(t, err)
	require.True(t, first.HasReminder)
	require.NotNil(t, first.Stream)
	require.Equal(t, liveStreamID, first.Stream.ID)
	require.Equal(t, stats.StartedAt.UnixMilli(), first.Stream.StartedAt)

	second, err := reminderService.GetPendingReminder(ctx, userID)
	require.NoError(t, err)
	require.False(t, second.HasReminder)
	require.Nil(t, second.Stream)
}

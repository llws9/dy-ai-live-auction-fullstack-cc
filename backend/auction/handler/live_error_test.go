package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"auction-service/model"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/stretchr/testify/assert"
)

type failingLiveReminderService struct{}

func (s *failingLiveReminderService) GetPendingReminder(ctx context.Context, userID int64) (*model.PendingLiveReminderResponse, error) {
	return nil, errors.New("mysql password leaked in driver error")
}

type failingLiveStarter struct{}

func (s *failingLiveStarter) StartLive(ctx context.Context, liveStreamID int64) error {
	return errors.New("redis password leaked in driver error")
}

func TestLiveReminderHandlerHidesInternalErrorDetails(t *testing.T) {
	h := NewLiveReminderHandler(&failingLiveReminderService{})
	c := app.NewContext(0)
	c.Set("user_id", int64(100))

	h.GetPendingReminder(context.Background(), c)

	body := string(c.Response.Body())
	assert.Equal(t, http.StatusInternalServerError, c.Response.StatusCode())
	assert.Contains(t, body, "获取开播提醒失败")
	assert.NotContains(t, body, "mysql password")
	assert.False(t, strings.Contains(body, "driver error"))
}

func TestLiveStreamStatsHandlerHidesInternalStartErrorDetails(t *testing.T) {
	h := NewLiveStreamStatsHandler(&failingLiveStarter{})
	c := app.NewContext(1)
	c.Params = append(c.Params, param.Param{Key: "id", Value: "123"})
	c.Set("user_id", int64(9001))
	c.Set("user_role", 2)

	h.StartLive(context.Background(), c)

	body := string(c.Response.Body())
	assert.Equal(t, http.StatusInternalServerError, c.Response.StatusCode())
	assert.Contains(t, body, "开始直播失败")
	assert.NotContains(t, body, "redis password")
	assert.False(t, strings.Contains(body, "driver error"))
}

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"reflect"
	"testing"
	"unsafe"

	"auction-service/model"
	"auction-service/service"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route/param"
	"github.com/stretchr/testify/assert"
)

type fakeNotificationStoreForHandler struct {
	getByUserIDErr    error
	getUnreadCountErr error
	markAsReadErr     error
	markAllAsReadErr  error
}

func (f *fakeNotificationStoreForHandler) Create(context.Context, *model.Notification) error {
	return nil
}

func (f *fakeNotificationStoreForHandler) CreateBatch(context.Context, []*model.Notification) error {
	return nil
}

func (f *fakeNotificationStoreForHandler) GetByUserID(context.Context, int64, int, int, bool) (*model.NotificationListResponse, error) {
	if f.getByUserIDErr != nil {
		return nil, f.getByUserIDErr
	}
	return &model.NotificationListResponse{}, nil
}

func (f *fakeNotificationStoreForHandler) GetUnreadCount(context.Context, int64) (int64, error) {
	return 0, f.getUnreadCountErr
}

func (f *fakeNotificationStoreForHandler) CountUnreadByTypes(context.Context, int64, []model.NotificationType) (int64, error) {
	return 0, nil
}

func (f *fakeNotificationStoreForHandler) MarkUnreadByTypesAsRead(context.Context, int64, []model.NotificationType) error {
	return nil
}

func (f *fakeNotificationStoreForHandler) MarkAsRead(context.Context, int64, int64) error {
	return f.markAsReadErr
}

func (f *fakeNotificationStoreForHandler) MarkAllAsRead(context.Context, int64) error {
	return f.markAllAsReadErr
}

func (f *fakeNotificationStoreForHandler) GetUnreadByUserID(context.Context, int64, int) ([]model.Notification, error) {
	return nil, nil
}

func setUnexportedField(target interface{}, fieldName string, value interface{}) {
	field := reflect.ValueOf(target).Elem().FieldByName(fieldName)
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

func newNotificationHandlerWithStore(store *fakeNotificationStoreForHandler) *NotificationHandler {
	svc := new(service.NotificationService)
	setUnexportedField(svc, "notificationDAO", store)
	return NewNotificationHandler(svc)
}

func TestNotificationHandlerErrorResponsesDoNotLeakInternalDetails(t *testing.T) {
	tests := []struct {
		name           string
		buildContext   func() *app.RequestContext
		invoke         func(*app.RequestContext)
		wantMessage    string
		wantLogSnippet string
		wantErrSnippet string
	}{
		{
			name: "list hides dao error details",
			buildContext: func() *app.RequestContext {
				c := app.NewContext(0)
				c.Request.SetMethod(http.MethodGet)
				c.Request.SetRequestURI("/api/v1/notifications?page=1&page_size=20")
				c.Set("user_id", int64(101))
				return c
			},
			invoke: func(c *app.RequestContext) {
				newNotificationHandlerWithStore(&fakeNotificationStoreForHandler{
					getByUserIDErr: errors.New("sql: database is closed"),
				}).List(context.Background(), c)
			},
			wantMessage:    "获取通知列表失败",
			wantLogSnippet: "获取通知列表失败 failed",
			wantErrSnippet: "sql: database is closed",
		},
		{
			name: "unread count hides dao error details",
			buildContext: func() *app.RequestContext {
				c := app.NewContext(0)
				c.Request.SetMethod(http.MethodGet)
				c.Request.SetRequestURI("/api/v1/notifications/unread-count")
				c.Set("user_id", int64(101))
				return c
			},
			invoke: func(c *app.RequestContext) {
				newNotificationHandlerWithStore(&fakeNotificationStoreForHandler{
					getUnreadCountErr: errors.New("sql: database is closed"),
				}).GetUnreadCount(context.Background(), c)
			},
			wantMessage:    "获取未读数量失败",
			wantLogSnippet: "获取未读数量失败 failed",
			wantErrSnippet: "sql: database is closed",
		},
		{
			name: "mark as read hides dao error details",
			buildContext: func() *app.RequestContext {
				c := app.NewContext(1)
				c.Request.SetMethod(http.MethodPut)
				c.Request.SetRequestURI("/api/v1/notifications/1/read")
				c.Params = append(c.Params, param.Param{Key: "id", Value: "1"})
				c.Set("user_id", int64(101))
				return c
			},
			invoke: func(c *app.RequestContext) {
				newNotificationHandlerWithStore(&fakeNotificationStoreForHandler{
					markAsReadErr: errors.New("sql: database is closed"),
				}).MarkAsRead(context.Background(), c)
			},
			wantMessage:    "标记已读失败",
			wantLogSnippet: "标记已读失败 failed",
			wantErrSnippet: "sql: database is closed",
		},
		{
			name: "mark all as read hides dao error details",
			buildContext: func() *app.RequestContext {
				c := app.NewContext(0)
				c.Request.SetMethod(http.MethodPut)
				c.Request.SetRequestURI("/api/v1/notifications/read-all")
				c.Set("user_id", int64(101))
				return c
			},
			invoke: func(c *app.RequestContext) {
				newNotificationHandlerWithStore(&fakeNotificationStoreForHandler{
					markAllAsReadErr: errors.New("sql: database is closed"),
				}).MarkAllAsRead(context.Background(), c)
			},
			wantMessage:    "标记已读失败",
			wantLogSnippet: "标记已读失败 failed",
			wantErrSnippet: "sql: database is closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			originalLogOutput := log.Writer()
			log.SetOutput(&logs)
			defer log.SetOutput(originalLogOutput)

			c := tt.buildContext()
			tt.invoke(c)

			assert.Equal(t, http.StatusInternalServerError, c.Response.StatusCode())
			var body map[string]interface{}
			assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
			assert.EqualValues(t, 500, body["code"])
			assert.Equal(t, tt.wantMessage, body["message"])
			assert.NotContains(t, body["message"], "database is closed")
			assert.Contains(t, logs.String(), tt.wantLogSnippet)
			assert.Contains(t, logs.String(), tt.wantErrSnippet)
		})
	}
}

func TestNotificationHandlerHotPullReturnsEmptyListWhenLiveReminderSourceUnavailable(t *testing.T) {
	var logs bytes.Buffer
	originalLogOutput := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(originalLogOutput)

	handler := NewNotificationHandler(new(service.NotificationService))
	c := app.NewContext(0)
	c.Request.SetMethod(http.MethodPost)
	c.Request.SetRequestURI("/api/v1/notifications/hot-pull")
	c.Set("user_id", int64(101))

	handler.HotPullNotifications(context.Background(), c)

	assert.Equal(t, http.StatusOK, c.Response.StatusCode())
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.EqualValues(t, 0, body["code"])
	assert.Equal(t, "success", body["message"])
	assert.NotContains(t, body["message"], "redis client not initialized")

	data, ok := body["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.EqualValues(t, 0, data["count"])
	notifications, ok := data["notifications"].([]interface{})
	assert.True(t, ok)
	assert.Empty(t, notifications)

	assert.Contains(t, logs.String(), "HotPull: redis unavailable user=101 follow_dao_configured=false")
	assert.Contains(t, logs.String(), "HotPull: completed without live reminder source user=101 total=0")
}

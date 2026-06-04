package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/service"
)

func newProductReminderTestServer(t *testing.T) *server.Hertz {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.UserProductReminder{}))

	reminderDAO := dao.NewUserProductReminderDAO(db)
	reminderService := service.NewProductReminderService(reminderDAO)
	reminderHandler := NewProductReminderHandler(reminderService)

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		if v := string(c.GetHeader("X-User-ID")); v != "" {
			if id, err := strconv.ParseInt(v, 10, 64); err == nil {
				c.Set("user_id", id)
			}
		}
		c.Next(ctx)
	})
	v1 := h.Group("/api/v1")
	v1.POST("/products/:id/remind", reminderHandler.SubscribeProductReminder)
	return h
}

func TestProductReminderSubscribeAcceptsGatewayInjectedInt64UserID(t *testing.T) {
	h := newProductReminderTestServer(t)

	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/api/v1/products/88/remind",
		nil,
		ut.Header{Key: "X-User-ID", Value: "100"},
	)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "订阅成功", body["message"])
}

func TestProductReminderSubscribeIsIdempotentForExistingReminder(t *testing.T) {
	h := newProductReminderTestServer(t)

	for range 2 {
		w := ut.PerformRequest(
			h.Engine,
			http.MethodPost,
			"/api/v1/products/88/remind",
			nil,
			ut.Header{Key: "X-User-ID", Value: "100"},
		)

		require.Equal(t, http.StatusOK, w.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "订阅成功", body["message"])
	}
}

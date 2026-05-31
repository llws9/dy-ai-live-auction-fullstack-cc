package handler

import (
	"context"

	"auction-service/service"
	"github.com/cloudwego/hertz/pkg/app"
)

type LiveReminderHandler struct {
	service *service.LiveReminderService
}

func NewLiveReminderHandler(service *service.LiveReminderService) *LiveReminderHandler {
	return &LiveReminderHandler{service: service}
}

func (h *LiveReminderHandler) GetPendingReminder(ctx context.Context, c *app.RequestContext) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
		return
	}
	userID := userIDInterface.(int64)

	result, err := h.service.GetPendingReminder(ctx, userID)
	if err != nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": "获取开播提醒失败: " + err.Error()})
		return
	}
	c.JSON(200, map[string]interface{}{"code": 0, "message": "success", "data": result})
}

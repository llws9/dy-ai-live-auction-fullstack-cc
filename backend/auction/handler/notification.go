package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"auction-service/service"
)

// NotificationHandler 通知 Handler
type NotificationHandler struct {
	notificationService *service.NotificationService
}

// NewNotificationHandler 创建通知 Handler
func NewNotificationHandler(notificationService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// List 获取通知列表
// @Summary 获取通知列表
// @Description 获取用户的通知列表，支持分页和筛选
// @Tags notification
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param unread_only query bool false "仅未读通知" default(false)
// @Success 200 {object} model.NotificationListResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /notifications [get]
func (h *NotificationHandler) List(ctx context.Context, c *app.RequestContext) {
	// 从JWT获取用户ID
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, map[string]interface{}{
			"code":    401,
			"message": "未认证，请先登录",
		})
		return
	}
	userID := userIDInterface.(int64)

	// 解析查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	unreadOnly := c.Query("unread_only") == "true"

	// 获取通知列表
	result, err := h.notificationService.GetNotifications(ctx, userID, page, pageSize, unreadOnly)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取通知列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, result)
}

// GetUnreadCount 获取未读通知数量
// @Summary 获取未读通知数量
// @Description 获取用户的未读通知总数
// @Tags notification
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.UnreadCountResponse
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /notifications/unread-count [get]
func (h *NotificationHandler) GetUnreadCount(ctx context.Context, c *app.RequestContext) {
	// 从JWT获取用户ID
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, map[string]interface{}{
			"code":    401,
			"message": "未认证，请先登录",
		})
		return
	}
	userID := userIDInterface.(int64)

	// 获取未读数量
	count, err := h.notificationService.GetUnreadCount(ctx, userID)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取未读数量失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    0,
		"message": "success",
		"data": map[string]interface{}{
			"count": count,
		},
	})
}

// MarkAsRead 标记通知为已读
// @Summary 标记通知为已读
// @Description 将指定通知标记为已读
// @Tags notification
// @Produce json
// @Security BearerAuth
// @Param id path int true "通知ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /notifications/{id}/read [put]
func (h *NotificationHandler) MarkAsRead(ctx context.Context, c *app.RequestContext) {
	// 从JWT获取用户ID
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, map[string]interface{}{
			"code":    401,
			"message": "未认证，请先登录",
		})
		return
	}
	userID := userIDInterface.(int64)

	// 解析通知ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的通知ID",
		})
		return
	}

	// 标记已读
	if err := h.notificationService.MarkAsRead(ctx, id, userID); err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "标记已读失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    0,
		"message": "success",
	})
}

// MarkAllAsRead 标记所有通知为已读
// @Summary 标记所有通知为已读
// @Description 将用户的所有未读通知标记为已读
// @Tags notification
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /notifications/read-all [put]
func (h *NotificationHandler) MarkAllAsRead(ctx context.Context, c *app.RequestContext) {
	// 从JWT获取用户ID
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, map[string]interface{}{
			"code":    401,
			"message": "未认证，请先登录",
		})
		return
	}
	userID := userIDInterface.(int64)

	// 标记所有已读
	if err := h.notificationService.MarkAllAsRead(ctx, userID); err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "标记已读失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    0,
		"message": "success",
	})
}

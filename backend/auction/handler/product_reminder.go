package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"auction-service/service"
)

func getProductReminderUserID(c *app.RequestContext) (int64, bool) {
	userID := c.GetInt64("user_id")
	if userID > 0 {
		return userID, true
	}

	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		return 0, false
	}
	parsed, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

// ProductReminderHandler 商品提醒订阅Handler
type ProductReminderHandler struct {
	reminderService *service.ProductReminderService
}

// NewProductReminderHandler 创建商品提醒订阅Handler
func NewProductReminderHandler(reminderService *service.ProductReminderService) *ProductReminderHandler {
	return &ProductReminderHandler{
		reminderService: reminderService,
	}
}

// SubscribeProductReminder 订阅商品提醒
// @Summary 订阅商品提醒
// @Description 用户订阅商品的竞拍提醒，当商品开始竞拍时会收到通知
// @Tags product-reminder
// @Produce json
// @Security BearerAuth
// @Param id path int true "商品ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /products/{id}/remind [post]
func (h *ProductReminderHandler) SubscribeProductReminder(ctx context.Context, c *app.RequestContext) {
	// 获取商品ID
	productIDStr := c.Param("id")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的商品ID",
		})
		return
	}

	// 获取用户ID（由 gateway 通过 X-User-ID 注入到上下文）
	userID, ok := getProductReminderUserID(c)
	if !ok {
		c.JSON(401, map[string]interface{}{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	// 调用服务订阅
	if err := h.reminderService.Subscribe(ctx, userID, productID); err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "订阅失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "订阅成功",
		"data": map[string]interface{}{
			"product_id": productID,
			"user_id":    userID,
		},
	})
}

// UnsubscribeProductReminder 取消订阅商品提醒
// @Summary 取消订阅商品提醒
// @Description 用户取消订阅商品的竞拍提醒
// @Tags product-reminder
// @Produce json
// @Security BearerAuth
// @Param id path int true "商品ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /products/{id}/remind [delete]
func (h *ProductReminderHandler) UnsubscribeProductReminder(ctx context.Context, c *app.RequestContext) {
	// 获取商品ID
	productIDStr := c.Param("id")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的商品ID",
		})
		return
	}

	// 获取用户ID（由 gateway 通过 X-User-ID 注入到上下文）
	userID, ok := getProductReminderUserID(c)
	if !ok {
		c.JSON(401, map[string]interface{}{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	// 调用服务取消订阅
	if err := h.reminderService.Unsubscribe(ctx, userID, productID); err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "取消订阅失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "取消订阅成功",
		"data": map[string]interface{}{
			"product_id": productID,
			"user_id":    userID,
		},
	})
}

// GetUserReminders 获取用户订阅列表
// @Summary 获取我的订阅列表
// @Description 获取用户订阅的所有商品提醒列表
// @Tags product-reminder
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /users/me/reminders [get]
func (h *ProductReminderHandler) GetUserReminders(ctx context.Context, c *app.RequestContext) {
	// 获取用户ID（由 gateway 通过 X-User-ID 注入到上下文）
	userID, ok := getProductReminderUserID(c)
	if !ok {
		c.JSON(401, map[string]interface{}{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	// 调用服务获取订阅列表
	reminders, err := h.reminderService.GetUserReminders(ctx, userID)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取订阅列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data": map[string]interface{}{
			"items": reminders,
			"total": len(reminders),
		},
	})
}

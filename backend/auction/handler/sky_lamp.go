package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"auction-service/model"
	"auction-service/service"
)

// SkyLampHandler 点天灯处理器
type SkyLampHandler struct {
	skyLampService *service.SkyLampService
}

// NewSkyLampHandler 创建点天灯处理器
func NewSkyLampHandler(skyLampService *service.SkyLampService) *SkyLampHandler {
	return &SkyLampHandler{
		skyLampService: skyLampService,
	}
}

// StartSubscriptionRequest 开启订阅请求
type StartSubscriptionRequest struct {
	AuctionID int64 `json:"auction_id" binding:"required"`
}

// StartSubscription 开启点天灯订阅
func (h *SkyLampHandler) StartSubscription(ctx context.Context, c *app.RequestContext) {
	userID, ok := extractUserID(c)
	if !ok {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未登录"})
		return
	}

	var req StartSubscriptionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}

	subscription, err := h.skyLampService.StartSubscription(ctx, userID, req.AuctionID)
	if err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":         200,
		"message":      "点天灯订阅已开启",
		"subscription": subscription,
	})
}

// StopSubscription 停止点天灯订阅
func (h *SkyLampHandler) StopSubscription(ctx context.Context, c *app.RequestContext) {
	userID, ok := extractUserID(c)
	if !ok {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未登录"})
		return
	}

	subscriptionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的订阅ID"})
		return
	}

	if err := h.skyLampService.StopSubscription(ctx, userID, subscriptionID); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{"code": 200, "message": "订阅已停止"})
}

// GetUserSubscriptions 获取用户订阅列表
func (h *SkyLampHandler) GetUserSubscriptions(ctx context.Context, c *app.RequestContext) {
	userID, ok := extractUserID(c)
	if !ok {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未登录"})
		return
	}

	status := model.SkyLampStatusActive
	if s := c.Query("status"); s != "" {
		statusInt, err := strconv.Atoi(s)
		if err == nil {
			status = model.SkyLampStatus(statusInt)
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	subscriptions, total, err := h.skyLampService.GetUserSubscriptions(
		ctx,
		userID,
		status,
		page,
		pageSize,
	)
	if err != nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":          200,
		"subscriptions": subscriptions,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
	})
}

// GetSubscriptionDetail 获取订阅详情
func (h *SkyLampHandler) GetSubscriptionDetail(ctx context.Context, c *app.RequestContext) {
	userID, ok := extractUserID(c)
	if !ok {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未登录"})
		return
	}

	subscriptionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的订阅ID"})
		return
	}

	subscription, err := h.skyLampService.GetSubscriptionDetail(
		ctx,
		subscriptionID,
		userID,
	)
	if err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{"code": 200, "data": subscription})
}

func extractUserID(c *app.RequestContext) (int64, bool) {
	if v, ok := c.Get("userID"); ok {
		switch t := v.(type) {
		case int64:
			return t, true
		case int:
			return int64(t), true
		case float64:
			return int64(t), true
		case string:
			id, err := strconv.ParseInt(t, 10, 64)
			if err == nil {
				return id, true
			}
		}
	}

	if v, ok := c.Get("user_id"); ok {
		switch t := v.(type) {
		case int64:
			return t, true
		case int:
			return int64(t), true
		case float64:
			return int64(t), true
		case string:
			id, err := strconv.ParseInt(t, 10, 64)
			if err == nil {
				return id, true
			}
		}
	}

	if s := c.GetString("user_id"); s != "" {
		id, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			return id, true
		}
	}

	return 0, false
}
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
	AuctionID        int64   `json:"auction_id" binding:"required"`
	InitialPrice     float64 `json:"initial_price" binding:"required"`
	InitialBidAmount float64 `json:"initial_bid_amount" binding:"required"`
	MaxPriceLimit    float64 `json:"max_price_limit" binding:"required"`
}

// StartSubscription 开启点天灯订阅
func (h *SkyLampHandler) StartSubscription(ctx context.Context, c *app.RequestContext) {
	userID, exists := c.Get("userID")
	if !exists {
		c.String(401, "未登录")
		return
	}

	var req StartSubscriptionRequest
	if err := c.BindJSON(&req); err != nil {
		c.String(400, err.Error())
		return
	}

	// 验证金额合理性
	if req.InitialBidAmount < req.InitialPrice {
		c.String(400, "首次出价金额不能低于当前价格")
		return
	}
	if req.MaxPriceLimit < req.InitialBidAmount {
		c.String(400, "天灯上限不能低于首次出价")
		return
	}

	subscription, err := h.skyLampService.StartSubscription(
		ctx,
		userID.(int64),
		req.AuctionID,
		req.InitialPrice,
		req.InitialBidAmount,
		req.MaxPriceLimit,
	)
	if err != nil {
		c.String(400, err.Error())
		return
	}

	c.JSON(200, map[string]interface{}{
		"message":      "点天灯订阅已开启",
		"subscription": subscription,
	})
}

// StopSubscription 停止点天灯订阅
func (h *SkyLampHandler) StopSubscription(ctx context.Context, c *app.RequestContext) {
	userID, exists := c.Get("userID")
	if !exists {
		c.String(401, "未登录")
		return
	}

	subscriptionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(400, "无效的订阅ID")
		return
	}

	if err := h.skyLampService.StopSubscription(ctx, userID.(int64), subscriptionID); err != nil {
		c.String(400, err.Error())
		return
	}

	c.JSON(200, map[string]string{"message": "订阅已停止"})
}

// GetUserSubscriptions 获取用户订阅列表
func (h *SkyLampHandler) GetUserSubscriptions(ctx context.Context, c *app.RequestContext) {
	userID, exists := c.Get("userID")
	if !exists {
		c.String(401, "未登录")
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
		userID.(int64),
		status,
		page,
		pageSize,
	)
	if err != nil {
		c.String(500, err.Error())
		return
	}

	c.JSON(200, map[string]interface{}{
		"subscriptions": subscriptions,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
	})
}

// GetSubscriptionDetail 获取订阅详情
func (h *SkyLampHandler) GetSubscriptionDetail(ctx context.Context, c *app.RequestContext) {
	userID, exists := c.Get("userID")
	if !exists {
		c.String(401, "未登录")
		return
	}

	subscriptionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(400, "无效的订阅ID")
		return
	}

	subscription, err := h.skyLampService.GetSubscriptionDetail(
		ctx,
		subscriptionID,
		userID.(int64),
	)
	if err != nil {
		c.String(400, err.Error())
		return
	}

	c.JSON(200, subscription)
}
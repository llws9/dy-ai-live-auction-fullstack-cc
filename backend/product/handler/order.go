package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"product-service/service"
)

// OrderHandler 订单 Handler
type OrderHandler struct {
	orderService *service.OrderService
}

// NewOrderHandler 创建订单 Handler
func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

// List 获取订单列表
// @Summary 获取订单列表
// @Description 获取订单列表，支持按用户筛选和分页
// @Tags order
// @Produce json
// @Param user_id query int false "用户ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /orders [get]
func (h *OrderHandler) List(ctx context.Context, c *app.RequestContext) {
	userIDStr := c.Query("user_id")
	var userID *int64
	if userIDStr != "" {
		id, err := strconv.ParseInt(userIDStr, 10, 64)
		if err == nil {
			userID = &id
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	orders, total, err := h.orderService.ListOrders(ctx, userID, page, pageSize)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取订单列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"items": orders,
		"total": total,
	})
}

// Get 获取订单详情
// @Summary 获取订单详情
// @Description 获取指定订单的详细信息
// @Tags order
// @Produce json
// @Param id path int true "订单ID"
// @Success 200 {object} model.Order
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /orders/{id} [get]
func (h *OrderHandler) Get(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的订单ID",
		})
		return
	}

	order, err := h.orderService.GetOrder(ctx, id)
	if err != nil {
		c.JSON(404, map[string]interface{}{
			"code":    404,
			"message": "订单不存在",
		})
		return
	}

	c.JSON(200, order)
}

// Pay 支付订单
// @Summary 支付订单
// @Description 支付指定订单（Mock实现）
// @Tags order
// @Produce json
// @Security BearerAuth
// @Param id path int true "订单ID"
// @Success 200 {object} model.Order
// @Failure 400 {object} map[string]interface{}
// @Router /orders/{id}/pay [post]
func (h *OrderHandler) Pay(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的订单ID",
		})
		return
	}

	order, err := h.orderService.PayOrder(ctx, id)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "支付失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, order)
}

// Ship 发货
// @Summary 发货
// @Description 发货指定订单（Mock实现）
// @Tags order
// @Produce json
// @Security BearerAuth
// @Param id path int true "订单ID"
// @Success 200 {object} model.Order
// @Failure 400 {object} map[string]interface{}
// @Router /orders/{id}/ship [post]
func (h *OrderHandler) Ship(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的订单ID",
		})
		return
	}

	order, err := h.orderService.ShipOrder(ctx, id)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "发货失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, order)
}

// Update 更新订单状态
func (h *OrderHandler) Update(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的订单ID",
		})
		return
	}

	var req struct {
		Status int `json:"status"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误",
		})
		return
	}

	order, err := h.orderService.GetOrder(ctx, id)
	if err != nil {
		c.JSON(404, map[string]interface{}{
			"code":    404,
			"message": "订单不存在",
		})
		return
	}

	// 根据状态执行相应操作
	switch req.Status {
	case 1: // 已支付
		order, err = h.orderService.PayOrder(ctx, id)
	case 2: // 已发货
		order, err = h.orderService.ShipOrder(ctx, id)
	default:
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的订单状态",
		})
		return
	}

	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "更新订单失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, order)
}

// GetUserHistory 获取用户历史记录
func (h *OrderHandler) GetUserHistory(ctx context.Context, c *app.RequestContext) {
	// 从JWT中获取用户ID（简化实现）
	userIDStr := c.Query("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的用户ID",
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	items, total, err := h.orderService.GetUserHistory(ctx, userID, page, pageSize)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取历史记录失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"items": items,
		"total": total,
		"page":  page,
		"page_size": pageSize,
	})
}

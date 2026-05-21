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

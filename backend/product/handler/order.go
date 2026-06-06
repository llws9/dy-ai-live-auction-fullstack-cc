package handler

import (
	"context"
	"log"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/shopspring/decimal"

	"product-service/model"
	"product-service/service"
)

type orderSummaryGetter interface {
	GetSummary(ctx context.Context, userID int64) (*model.OrderSummaryResponse, error)
}

// OrderHandler 订单 Handler
type OrderHandler struct {
	orderService   *service.OrderService
	summaryService orderSummaryGetter
}

// NewOrderHandler 创建订单 Handler
func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService:   orderService,
		summaryService: orderService,
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
	userID, ok := readHeaderUserID(c)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	orders, total, err := h.orderService.ListOrderViews(ctx, &userID, page, pageSize)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取订单列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"list":  orders,
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
	userID, ok := readHeaderUserID(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的订单ID",
		})
		return
	}

	order, err := h.orderService.GetOrderForUser(ctx, id, userID)
	if err != nil {
		c.JSON(404, map[string]interface{}{
			"code":    404,
			"message": "订单不存在",
		})
		return
	}

	c.JSON(200, order)
}

// Summary 获取当前登录用户订单触点汇总。
func (h *OrderHandler) Summary(ctx context.Context, c *app.RequestContext) {
	userIDStr := string(c.Request.Header.Peek("X-User-ID"))
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
		return
	}

	summary, err := h.summaryService.GetSummary(ctx, userID)
	if err != nil {
		log.Printf("Summary failed: userID=%d err=%v", userID, err)
		c.JSON(500, map[string]interface{}{"code": 500, "message": "获取订单汇总失败"})
		return
	}

	c.JSON(200, map[string]interface{}{"code": 0, "message": "success", "data": summary})
}

type createOrderFromAuctionResultRequest struct {
	AuctionID  int64  `json:"auction_id"`
	ProductID  int64  `json:"product_id"`
	WinnerID   int64  `json:"winner_id"`
	FinalPrice string `json:"final_price"`
}

// CreateFromAuctionResult creates the pending order for a finalized auction.
// This handler is mounted behind /internal and protected by InternalAuthMiddleware.
func (h *OrderHandler) CreateFromAuctionResult(ctx context.Context, c *app.RequestContext) {
	var req createOrderFromAuctionResultRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误"})
		return
	}
	if req.AuctionID <= 0 || req.ProductID <= 0 || req.WinnerID <= 0 || req.FinalPrice == "" {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误"})
		return
	}
	finalPrice, err := decimal.NewFromString(req.FinalPrice)
	if err != nil || !finalPrice.GreaterThan(decimal.Zero) {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的成交价"})
		return
	}

	order, err := h.orderService.CreateOrderFromAuctionResult(ctx, req.AuctionID, req.ProductID, req.WinnerID, finalPrice)
	if err != nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": "创建订单失败"})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    0,
		"message": "success",
		"data": map[string]interface{}{
			"id":          order.ID,
			"auction_id":  order.AuctionID,
			"product_id":  order.ProductID,
			"winner_id":   order.WinnerID,
			"final_price": order.FinalPrice.StringFixed(2),
			"status":      order.Status,
		},
	})
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
	userID, ok := readHeaderUserID(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的订单ID",
		})
		return
	}

	order, err := h.orderService.PayOrderForUser(ctx, id, userID)
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
	actor, ok := requireMerchantActor(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的订单ID",
		})
		return
	}

	order, err := h.orderService.ShipOrderForSeller(ctx, id, actor.UserID)
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

// GetUserHistory 获取当前登录用户的订单历史。
//
// 安全契约（spec C / F-C3, M1 P0）：
//   - 用户身份只来自 Gateway 透传的 X-User-ID header，不接受 query/body user_id；
//     这样调用方无法通过篡改请求参数读取他人订单。
//   - 缺失 X-User-ID（即未经 Gateway JWT 中间件）时返回 401。
func (h *OrderHandler) GetUserHistory(ctx context.Context, c *app.RequestContext) {
	userID, ok := readHeaderUserID(c)
	if !ok {
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
		"list":      items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func readHeaderUserID(c *app.RequestContext) (int64, bool) {
	userIDStr := string(c.GetHeader("X-User-ID"))
	if userIDStr == "" {
		c.JSON(401, map[string]interface{}{
			"code":    401,
			"message": "未认证",
		})
		return 0, false
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(401, map[string]interface{}{
			"code":    401,
			"message": "无效的用户身份",
		})
		return 0, false
	}
	return userID, true
}

package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"product-service/service"
)

// RuleHandler 竞拍规则 Handler
type RuleHandler struct {
	productService *service.ProductService
}

// NewRuleHandler 创建规则 Handler
func NewRuleHandler(productService *service.ProductService) *RuleHandler {
	return &RuleHandler{
		productService: productService,
	}
}

// Create 配置竞拍规则
func (h *RuleHandler) Create(ctx context.Context, c *app.RequestContext) {
	// 从路径参数获取商品 ID
	productIDStr := c.Param("id")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的商品ID",
		})
		return
	}

	var req service.CreateAuctionRuleRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 设置 product_id
	req.ProductID = productID

	rule, err := h.productService.CreateAuctionRule(ctx, &req)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "创建竞拍规则失败: " + err.Error(),
		})
		return
	}

	c.JSON(201, rule)
}

// Get 获取竞拍规则
func (h *RuleHandler) Get(ctx context.Context, c *app.RequestContext) {
	productIDStr := c.Param("id")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的商品ID",
		})
		return
	}

	// 临时使用 product_id 作为 auction_id
	rule, err := h.productService.GetAuctionRule(ctx, productID)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取竞拍规则失败: " + err.Error(),
		})
		return
	}

	if rule == nil {
		c.JSON(404, map[string]interface{}{
			"code":    404,
			"message": "竞拍规则不存在",
		})
		return
	}

	c.JSON(200, rule)
}

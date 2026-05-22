package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"product-service/model"
	"product-service/service"
)

// ProductHandler 商品 Handler
type ProductHandler struct {
	productService *service.ProductService
}

// NewProductHandler 创建商品 Handler
func NewProductHandler(productService *service.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
	}
}

// Create 创建商品
// @Summary 创建商品
// @Description 创建新商品
// @Tags product
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body service.CreateProductRequest true "商品信息"
// @Success 201 {object} model.Product
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /products [post]
func (h *ProductHandler) Create(ctx context.Context, c *app.RequestContext) {
	var req service.CreateProductRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	product, err := h.productService.CreateProduct(ctx, &req)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "创建商品失败: " + err.Error(),
		})
		return
	}

	c.JSON(201, product)
}

// Get 获取商品详情
// @Summary 获取商品详情
// @Description 获取指定商品的详细信息
// @Tags product
// @Produce json
// @Param id path int true "商品ID"
// @Success 200 {object} model.Product
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /products/{id} [get]
func (h *ProductHandler) Get(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的商品ID",
		})
		return
	}

	product, err := h.productService.GetProduct(ctx, id)
	if err != nil {
		c.JSON(404, map[string]interface{}{
			"code":    404,
			"message": "商品不存在",
		})
		return
	}

	c.JSON(200, product)
}

// List 获取商品列表
// @Summary 获取商品列表
// @Description 获取商品列表，支持按状态筛选和分页
// @Tags product
// @Produce json
// @Param status query int false "商品状态：0-下架, 1-上架"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /products [get]
func (h *ProductHandler) List(ctx context.Context, c *app.RequestContext) {
	// 解析查询参数
	statusStr := c.Query("status")
	var status *model.ProductStatus
	if statusStr != "" {
		s, err := strconv.Atoi(statusStr)
		if err == nil {
			st := model.ProductStatus(s)
			status = &st
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	products, total, err := h.productService.ListProducts(ctx, status, page, pageSize)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取商品列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"items":     products,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Update 更新商品
// @Summary 更新商品
// @Description 更新指定商品的信息
// @Tags product
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "商品ID"
// @Param body body service.UpdateProductRequest true "商品更新信息"
// @Success 200 {object} model.Product
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /products/{id} [put]
func (h *ProductHandler) Update(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的商品ID",
		})
		return
	}

	var req service.UpdateProductRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	product, err := h.productService.UpdateProduct(ctx, id, &req)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "更新商品失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, product)
}

// Delete 删除商品
// @Summary 删除商品
// @Description 删除指定商品
// @Tags product
// @Produce json
// @Security BearerAuth
// @Param id path int true "商品ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /products/{id} [delete]
func (h *ProductHandler) Delete(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的商品ID",
		})
		return
	}

	if err := h.productService.DeleteProduct(ctx, id); err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "删除商品失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "删除成功",
	})
}

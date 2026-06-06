package handler

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"

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
		if errors.Is(err, service.ErrInvalidCategory) {
			c.JSON(400, map[string]interface{}{
				"code":    400,
				"message": "非法或未启用的 category_id",
			})
			return
		}
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
	if err != nil || product.Status != model.ProductStatusPublished {
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
	publishedStatus := model.ProductStatusPublished
	status := &publishedStatus
	if statusStr != "" {
		s, err := strconv.Atoi(statusStr)
		if err == nil && model.ProductStatus(s) != model.ProductStatusPublished {
			page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
			pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
			c.JSON(200, map[string]interface{}{
				"code":    200,
				"message": "success",
				"data": map[string]interface{}{
					"list":      []model.Product{},
					"total":     int64(0),
					"page":      page,
					"page_size": pageSize,
				},
			})
			return
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
		"code":    200,
		"message": "success",
		"data": map[string]interface{}{
			"list":      products,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func (h *ProductHandler) AdminList(ctx context.Context, c *app.RequestContext) {
	actor, ok := readAdminActor(c)
	if !ok {
		return
	}
	status := parseProductStatusQuery(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	products, total, err := h.productService.ListAdminProducts(ctx, actor.Role, actor.UserID, status, page, pageSize)
	if err != nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": "获取商品列表失败: " + err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data": map[string]interface{}{
			"list":      products,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func (h *ProductHandler) AdminGet(ctx context.Context, c *app.RequestContext) {
	actor, ok := readAdminActor(c)
	if !ok {
		return
	}
	id, ok := parseProductIDParam(c)
	if !ok {
		return
	}

	product, err := h.productService.GetAdminProduct(ctx, actor.Role, actor.UserID, id)
	if err != nil {
		writeProductError(c, err, "商品不存在")
		return
	}

	c.JSON(200, map[string]interface{}{"code": 200, "message": "success", "data": product})
}

func (h *ProductHandler) AdminCreate(ctx context.Context, c *app.RequestContext) {
	actor, ok := requireMerchantActor(c)
	if !ok {
		return
	}
	var req service.CreateProductRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}

	product, err := h.productService.CreateProductForOwner(ctx, actor.UserID, &req)
	if err != nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": "创建商品失败: " + err.Error()})
		return
	}

	c.JSON(201, map[string]interface{}{"code": 201, "message": "success", "data": product})
}

func (h *ProductHandler) AdminUpdate(ctx context.Context, c *app.RequestContext) {
	actor, ok := requireMerchantActor(c)
	if !ok {
		return
	}
	id, ok := parseProductIDParam(c)
	if !ok {
		return
	}
	var req service.UpdateProductRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}

	product, err := h.productService.UpdateAdminProduct(ctx, actor.UserID, id, &req)
	if err != nil {
		writeProductError(c, err, "更新商品失败")
		return
	}

	c.JSON(200, map[string]interface{}{"code": 200, "message": "success", "data": product})
}

func (h *ProductHandler) AdminDelete(ctx context.Context, c *app.RequestContext) {
	actor, ok := requireMerchantActor(c)
	if !ok {
		return
	}
	id, ok := parseProductIDParam(c)
	if !ok {
		return
	}

	if err := h.productService.DeleteAdminProduct(ctx, actor.UserID, id); err != nil {
		writeProductError(c, err, "删除商品失败")
		return
	}

	c.JSON(200, map[string]interface{}{"code": 200, "message": "删除成功"})
}

func parseProductStatusQuery(c *app.RequestContext) *model.ProductStatus {
	statusStr := c.Query("status")
	if statusStr == "" {
		return nil
	}
	s, err := strconv.Atoi(statusStr)
	if err != nil {
		return nil
	}
	status := model.ProductStatus(s)
	return &status
}

func parseProductIDParam(c *app.RequestContext) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的商品ID"})
		return 0, false
	}
	return id, true
}

func writeProductError(c *app.RequestContext, err error, fallback string) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(404, map[string]interface{}{"code": 404, "message": "商品不存在"})
		return
	}
	c.JSON(500, map[string]interface{}{"code": 500, "message": fallback + ": " + err.Error()})
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
		if errors.Is(err, service.ErrInvalidCategory) {
			c.JSON(400, map[string]interface{}{
				"code":    400,
				"message": "非法或未启用的 category_id",
			})
			return
		}
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

// PublishRequest 发布商品请求
type PublishRequest struct {
	StartTime *string `json:"start_time"` // ISO 8601 格式
	RuleID    *int64  `json:"rule_id"`
}

// PublishHandler 发布商品到直播间
func (h *ProductHandler) PublishHandler(ctx context.Context, c *app.RequestContext) {
	productIDStr := c.Param("id")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的商品ID",
		})
		return
	}

	var req PublishRequest
	if err := c.BindJSON(&req); err != nil {
		req.StartTime = nil
		req.RuleID = nil
	}

	userID := c.GetInt64("user_id")
	userRole := c.GetInt("user_role")

	if userRole != 1 && userRole != 2 {
		c.JSON(403, map[string]interface{}{
			"code":    403,
			"message": "权限不足：需要商家或管理员权限",
		})
		return
	}

	var startTime *time.Time
	if req.StartTime != nil {
		t, err := time.Parse(time.RFC3339, *req.StartTime)
		if err != nil {
			c.JSON(400, map[string]interface{}{
				"code":    400,
				"message": "时间格式错误，请使用ISO 8601格式",
			})
			return
		}
		startTime = &t
	}

	product, liveStream, err := h.productService.PublishProduct(ctx, productID, userID, startTime)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "发布成功",
		"data": map[string]interface{}{
			"product": product,
			"live_stream": map[string]interface{}{
				"id":   liveStream.ID,
				"name": liveStream.Name,
			},
		},
	})
}

// UnpublishRequest 下架商品请求
type UnpublishRequest struct {
	Reason string `json:"reason"`
}

// UnpublishHandler 下架商品
func (h *ProductHandler) UnpublishHandler(ctx context.Context, c *app.RequestContext) {
	productIDStr := c.Param("id")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的商品ID",
		})
		return
	}

	var req UnpublishRequest
	_ = c.BindJSON(&req)

	userID := c.GetInt64("user_id")
	userRole := c.GetInt("user_role")

	if userRole != 1 && userRole != 2 {
		c.JSON(403, map[string]interface{}{
			"code":    403,
			"message": "权限不足：需要商家或管理员权限",
		})
		return
	}

	product, err := h.productService.UnpublishProduct(ctx, productID, userID, req.Reason)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "下架成功",
		"data": map[string]interface{}{
			"product_id":  product.ID,
			"status":      product.Status,
			"unpublished": true,
		},
	})
}

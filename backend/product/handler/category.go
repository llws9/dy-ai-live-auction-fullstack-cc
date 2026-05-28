package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"product-service/model"
	"product-service/service"
)

// CategoryHandler 类别Handler
type CategoryHandler struct {
	categoryService *service.CategoryService
}

// NewCategoryHandler 创建类别Handler
func NewCategoryHandler(categoryService *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
	}
}

// List 类别列表
func (h *CategoryHandler) List(ctx context.Context, c *app.RequestContext) {
	statusStr := c.Query("status")

	var statusFilter *int
	if statusStr != "" {
		status, _ := strconv.Atoi(statusStr)
		statusFilter = &status
	}

	categories, total, err := h.categoryService.List(ctx, statusFilter)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取类别列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"list":  categories,
			"total": total,
		},
	})
}

// Create 创建类别
func (h *CategoryHandler) Create(ctx context.Context, c *app.RequestContext) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}

	if err := c.Bind(&req); err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	category := &model.Category{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		Status:      model.CategoryStatusActive,
	}

	err := h.categoryService.Create(ctx, category)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(201, map[string]interface{}{
		"code": 201,
		"data": map[string]interface{}{
			"id":   category.ID,
			"name": category.Name,
			"code": category.Code,
		},
		"message": "创建成功",
	})
}

// Update 更新类别
func (h *CategoryHandler) Update(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的类别ID",
		})
		return
	}

	var req map[string]interface{}
	if err := c.Bind(&req); err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	category, err := h.categoryService.Update(ctx, id, req)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"id":   category.ID,
			"name": category.Name,
			"code": category.Code,
		},
		"message": "更新成功",
	})
}

// Delete 删除类别
func (h *CategoryHandler) Delete(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的类别ID",
		})
		return
	}

	err = h.categoryService.Delete(ctx, id)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "删除成功",
	})
}
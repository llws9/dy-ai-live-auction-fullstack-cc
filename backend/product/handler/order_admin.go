package handler

import (
	"context"
	"errors"
	"log"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"

	"product-service/model"
)

const maxAdminOrderPageSize = 100

// AdminList 处理 GET /api/v1/admin/orders。
//
// 与 C 端 List 的核心区别：
//   - 不读 X-User-ID，admin 端语义即看全量；
//   - 入参 status / user_id / page / page_size 均可选；user_id 用作筛选某用户（=winner_id）。
//
// Gateway 仍是入口 SSOT；这里保留角色头校验，防止直连 product-service 读取全量订单。
func (h *OrderHandler) AdminList(ctx context.Context, c *app.RequestContext) {
	if !requireAdminRole(c) {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > maxAdminOrderPageSize {
		pageSize = maxAdminOrderPageSize
	}

	var statusPtr *model.OrderStatus
	if statusStr := string(c.Query("status")); statusStr != "" {
		if v, err := strconv.Atoi(statusStr); err == nil {
			st := model.OrderStatus(v)
			statusPtr = &st
		}
	}

	var userIDPtr *int64
	if userIDStr := string(c.Query("user_id")); userIDStr != "" {
		if v, err := strconv.ParseInt(userIDStr, 10, 64); err == nil && v > 0 {
			userIDPtr = &v
		}
	}

	items, total, err := h.orderService.ListAdminOrders(ctx, statusPtr, userIDPtr, page, pageSize)
	if err != nil {
		log.Printf("AdminList failed: status=%v userID=%v page=%d pageSize=%d err=%v", statusPtr, userIDPtr, page, pageSize, err)
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取订单列表失败",
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data": map[string]interface{}{
			"list":      items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// AdminGet 处理 GET /api/v1/admin/orders/:id。
// admin 视角不按 winner_id 过滤；返回 product_name 与首图便于前端直接展示。
func (h *OrderHandler) AdminGet(ctx context.Context, c *app.RequestContext) {
	if !requireAdminRole(c) {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的订单ID",
		})
		return
	}

	vo, err := h.orderService.GetAdminOrder(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, map[string]interface{}{
				"code":    404,
				"message": "订单不存在",
			})
			return
		}
		log.Printf("AdminGet failed: orderID=%d err=%v", id, err)
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取订单详情失败",
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data":    vo,
	})
}

func requireAdminRole(c *app.RequestContext) bool {
	if string(c.GetHeader("X-User-Role")) == "admin" {
		return true
	}
	c.JSON(403, map[string]interface{}{
		"code":    403,
		"message": "权限不足：需要管理员权限",
	})
	return false
}

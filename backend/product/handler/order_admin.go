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
// Gateway 仍是入口 SSOT；这里保留角色头校验，防止直连 product-service 越权读取订单。
func (h *OrderHandler) AdminList(ctx context.Context, c *app.RequestContext) {
	actor, ok := readOrderManagementActor(c)
	if !ok {
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
		v, err := strconv.Atoi(statusStr)
		if err != nil {
			c.JSON(400, map[string]interface{}{
				"code":    400,
				"message": "无效的订单状态",
			})
			return
		}
		st := model.OrderStatus(v)
		if !isValidOrderStatus(st) {
			c.JSON(400, map[string]interface{}{
				"code":    400,
				"message": "无效的订单状态",
			})
			return
		}
		statusPtr = &st
	}

	var userIDPtr *int64
	if userIDStr := string(c.Query("user_id")); userIDStr != "" {
		v, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil || v <= 0 {
			c.JSON(400, map[string]interface{}{
				"code":    400,
				"message": "无效的用户ID",
			})
			return
		}
		userIDPtr = &v
	}

	var sellerIDPtr *int64
	if actor.Role == "merchant" {
		sellerIDPtr = &actor.UserID
	}
	search := string(c.Query("search"))
	result, err := h.orderService.ListAdminOrdersScoped(ctx, statusPtr, userIDPtr, sellerIDPtr, search, page, pageSize)
	if err != nil {
		log.Printf("AdminList failed: status=%v userID=%v search=%q page=%d pageSize=%d err=%v", statusPtr, userIDPtr, search, page, pageSize, err)
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
			"list":      result.Items,
			"total":     result.Total,
			"page":      page,
			"page_size": pageSize,
			"summary":   result.Summary,
		},
	})
}

// AdminGet 处理 GET /api/v1/admin/orders/:id。
// admin 视角不按 winner_id 过滤；返回 product_name 与首图便于前端直接展示。
func (h *OrderHandler) AdminGet(ctx context.Context, c *app.RequestContext) {
	actor, ok := readOrderManagementActor(c)
	if !ok {
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

	var sellerIDPtr *int64
	if actor.Role == "merchant" {
		sellerIDPtr = &actor.UserID
	}
	vo, err := h.orderService.GetAdminOrderScoped(ctx, id, sellerIDPtr)
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

func isValidOrderStatus(status model.OrderStatus) bool {
	switch status {
	case model.OrderStatusPending, model.OrderStatusPaid, model.OrderStatusShipped, model.OrderStatusCompleted:
		return true
	default:
		return false
	}
}

func readOrderManagementActor(c *app.RequestContext) (AdminActor, bool) {
	role := string(c.GetHeader("X-User-Role"))
	switch role {
	case "admin":
		return AdminActor{Role: "admin"}, true
	case "merchant":
		userID, ok := readHeaderUserID(c)
		if !ok {
			return AdminActor{}, false
		}
		return AdminActor{UserID: userID, Role: "merchant"}, true
	default:
		c.JSON(403, map[string]interface{}{
			"code":    403,
			"message": "权限不足",
		})
		return AdminActor{}, false
	}
}

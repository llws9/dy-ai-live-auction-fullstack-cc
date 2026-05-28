package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"product-service/service"
)

// LiveStreamHandler 直播间Handler
type LiveStreamHandler struct {
	liveStreamService *service.LiveStreamService
}

// NewLiveStreamHandler 创建直播间Handler
func NewLiveStreamHandler(liveStreamService *service.LiveStreamService) *LiveStreamHandler {
	return &LiveStreamHandler{
		liveStreamService: liveStreamService,
	}
}

// ListAdmin 管理端直播间列表 (T011)
func (h *LiveStreamHandler) ListAdmin(ctx context.Context, c *app.RequestContext) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	statusStr := c.Query("status")

	var statusFilter *int
	if statusStr != "" {
		status, _ := strconv.Atoi(statusStr)
		statusFilter = &status
	}

	liveStreams, total, err := h.liveStreamService.ListAdmin(ctx, page, pageSize, statusFilter)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取直播间列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"list":  liveStreams,
			"total": total,
		},
	})
}

// GetDetail 直播间详情 (T012)
func (h *LiveStreamHandler) GetDetail(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的直播间ID",
		})
		return
	}

	detail, err := h.liveStreamService.GetDetail(ctx, id)
	if err != nil {
		c.JSON(404, map[string]interface{}{
			"code":    404,
			"message": "直播间不存在",
		})
		return
	}

	// 构建响应
	result := map[string]interface{}{
		"id":          detail.ID,
		"name":        detail.Name,
		"description": detail.Description,
		"cover_image": detail.CoverImage,
		"status":      detail.Status,
		"creator_id":  detail.CreatorID,
		"created_at":  detail.CreatedAt,
	}

	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": result,
	})
}
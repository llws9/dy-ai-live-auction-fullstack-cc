package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"auction-service/service"
)

// FollowHandler 关注处理器
type FollowHandler struct {
	followService *service.FollowService
}

// NewFollowHandler 创建关注处理器
func NewFollowHandler(followService *service.FollowService) *FollowHandler {
	return &FollowHandler{
		followService: followService,
	}
}

// FollowHandler 关注直播间
func (h *FollowHandler) FollowHandler(ctx context.Context, c *app.RequestContext) {
	// 解析路径参数
	liveStreamIDStr := c.Param("id")
	liveStreamID, err := strconv.ParseInt(liveStreamIDStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的直播间ID",
		})
		return
	}

	// 获取用户ID
	userID := c.GetInt64("user_id")

	// 执行关注
	follow, err := h.followService.Follow(ctx, userID, liveStreamID)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "关注成功",
		"data": map[string]interface{}{
			"follow": follow,
		},
	})
}

// UnfollowHandler 取消关注直播间
func (h *FollowHandler) UnfollowHandler(ctx context.Context, c *app.RequestContext) {
	// 解析路径参数
	liveStreamIDStr := c.Param("id")
	liveStreamID, err := strconv.ParseInt(liveStreamIDStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的直播间ID",
		})
		return
	}

	// 获取用户ID
	userID := c.GetInt64("user_id")

	// 执行取消关注
	if err := h.followService.Unfollow(ctx, userID, liveStreamID); err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "取消关注成功",
	})
}

// GetUserFollowsHandler 获取用户关注的直播间列表
func (h *FollowHandler) GetUserFollowsHandler(ctx context.Context, c *app.RequestContext) {
	// 获取用户ID
	userID := c.GetInt64("user_id")

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 获取关注列表
	follows, total, err := h.followService.GetUserFollows(ctx, userID, page, pageSize)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取关注列表失败",
		})
		return
	}

	// 返回响应
	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"items":     follows,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// ToggleNotificationHandler 切换通知状态
func (h *FollowHandler) ToggleNotificationHandler(ctx context.Context, c *app.RequestContext) {
	// 解析路径参数
	liveStreamIDStr := c.Param("id")
	liveStreamID, err := strconv.ParseInt(liveStreamIDStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的直播间ID",
		})
		return
	}

	// 获取用户ID
	userID := c.GetInt64("user_id")

	// 解析请求体
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误",
		})
		return
	}

	// 切换通知状态
	if err := h.followService.ToggleNotification(ctx, userID, liveStreamID, req.Enabled); err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "通知状态更新成功",
	})
}

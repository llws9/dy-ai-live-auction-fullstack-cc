package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"auction-service/service"
)

// FollowHandler 关注处理器
type FollowHandler struct {
	followService  *service.FollowService
	lsFetcher      LiveStreamBatchFetcher
	userFetcher    UserAvatarFetcher
	auctionFetcher AuctionCountFetcher
}

// NewFollowHandler 创建关注处理器
func NewFollowHandler(followService *service.FollowService) *FollowHandler {
	return &FollowHandler{
		followService: followService,
	}
}

// SetFollowedListFetchers 注入 GetUserFollowsHandler 编排所需的批量取数依赖（T3.3 / spec B §2.3）。
func (h *FollowHandler) SetFollowedListFetchers(
	ls LiveStreamBatchFetcher,
	user UserAvatarFetcher,
	auction AuctionCountFetcher,
) {
	h.lsFetcher = ls
	h.userFetcher = user
	h.auctionFetcher = auction
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

// GetUserFollowsHandler 获取用户关注的直播间列表（T3.3 / spec B §2.3 / F-B3）
//
// 通过 BuildFollowedLiveStreams 编排：拉关注列表 → 跨服务批量取直播间 →
// 批量取主播头像 → 批量取进行中竞拍数。返回字段固定全量（含 viewer_count=0）。
func (h *FollowHandler) GetUserFollowsHandler(ctx context.Context, c *app.RequestContext) {
	userID := c.GetInt64("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if h.lsFetcher == nil || h.userFetcher == nil || h.auctionFetcher == nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "服务未就绪",
		})
		return
	}

	resp, err := BuildFollowedLiveStreams(ctx, h.followService,
		h.lsFetcher, h.userFetcher, h.auctionFetcher,
		userID, page, pageSize)
	if err != nil {
		if err.Error() == "invalid user_id" {
			c.JSON(400, map[string]interface{}{
				"code":    400,
				"message": err.Error(),
			})
			return
		}
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取关注列表失败",
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": resp,
	})
}

// GetFollowStatusHandler 查询当前用户对指定直播间的关注状态（F-B2）
//
// 路径：GET /api/v1/live-streams/:id/follow-status（authGroup）
// 响应：{ "code":200, "data":{ "is_following": bool } }
//
// user_id 来源同 POST/DELETE follow：由 gateway JWTAuth 注入到 c.Set("user_id", ...)。
func (h *FollowHandler) GetFollowStatusHandler(ctx context.Context, c *app.RequestContext) {
	liveStreamIDStr := c.Param("id")
	liveStreamID, err := strconv.ParseInt(liveStreamIDStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的直播间ID",
		})
		return
	}

	userID := c.GetInt64("user_id")

	resp, err := BuildFollowStatusResponse(ctx, h.followService, userID, liveStreamID)
	if err != nil {
		if err.Error() == "invalid user_id" || err.Error() == "invalid live_stream_id" {
			c.JSON(400, map[string]interface{}{"code": 400, "message": err.Error()})
			return
		}
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "查询关注状态失败",
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": resp,
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

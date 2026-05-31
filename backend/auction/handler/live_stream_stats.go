package handler

import (
	"context"
	"strconv"

	"auction-service/service"
	"github.com/cloudwego/hertz/pkg/app"
)

type LiveStreamStatsHandler struct {
	service *service.LiveStreamStatsService
}

func NewLiveStreamStatsHandler(service *service.LiveStreamStatsService) *LiveStreamStatsHandler {
	return &LiveStreamStatsHandler{service: service}
}

func (h *LiveStreamStatsHandler) StartLive(ctx context.Context, c *app.RequestContext) {
	if _, exists := c.Get("user_id"); !exists {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
		return
	}
	if role := c.GetInt("user_role"); role < 1 {
		c.JSON(403, map[string]interface{}{"code": 403, "message": "无权限操作直播间"})
		return
	}

	liveStreamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || liveStreamID <= 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的直播间ID"})
		return
	}

	if err := h.service.StartLive(ctx, liveStreamID); err != nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": "开始直播失败: " + err.Error()})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    0,
		"message": "success",
		"data":    map[string]interface{}{"success": true},
	})
}

package handler

import (
	"context"
	"log"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
)

type LiveStreamStatsHandler struct {
	service      LiveStarter
	ownerChecker LiveStreamOwnerChecker
}

type LiveStarter interface {
	StartLive(ctx context.Context, liveStreamID int64) error
}

type LiveStreamOwnerChecker interface {
	OwnerID(ctx context.Context, liveStreamID int64) (int64, error)
}

func NewLiveStreamStatsHandler(service LiveStarter) *LiveStreamStatsHandler {
	return &LiveStreamStatsHandler{service: service}
}

func (h *LiveStreamStatsHandler) SetOwnerChecker(checker LiveStreamOwnerChecker) {
	h.ownerChecker = checker
}

func (h *LiveStreamStatsHandler) StartLive(ctx context.Context, c *app.RequestContext) {
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
		return
	}
	userID, ok := userIDRaw.(int64)
	if !ok || userID <= 0 {
		c.JSON(401, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
		return
	}
	if role := c.GetInt("user_role"); role != 1 {
		c.JSON(403, map[string]interface{}{"code": 403, "message": "无权限操作直播间"})
		return
	}

	liveStreamID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || liveStreamID <= 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的直播间ID"})
		return
	}

	if h.ownerChecker == nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": "直播间归属校验未配置"})
		return
	}
	ownerID, err := h.ownerChecker.OwnerID(ctx, liveStreamID)
	if err != nil {
		log.Printf("StartLive owner check failed: liveStreamID=%d userID=%d err=%v", liveStreamID, userID, err)
		c.JSON(500, map[string]interface{}{"code": 500, "message": "开始直播失败"})
		return
	}
	if ownerID == 0 || ownerID != userID {
		c.JSON(403, map[string]interface{}{"code": 403, "message": "无权限操作直播间"})
		return
	}

	if err := h.service.StartLive(ctx, liveStreamID); err != nil {
		log.Printf("StartLive failed: liveStreamID=%d err=%v", liveStreamID, err)
		c.JSON(500, map[string]interface{}{"code": 500, "message": "开始直播失败"})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    0,
		"message": "success",
		"data":    map[string]interface{}{"success": true},
	})
}

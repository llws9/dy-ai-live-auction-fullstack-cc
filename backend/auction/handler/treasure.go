package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"auction-service/dao"
	"auction-service/service"

	"github.com/cloudwego/hertz/pkg/app"
)

type TreasureHandler struct {
	svc *service.TreasureService
}

func NewTreasureHandler(svc *service.TreasureService) *TreasureHandler {
	return &TreasureHandler{svc: svc}
}

func (h *TreasureHandler) GetStatus(ctx context.Context, c *app.RequestContext) {
	userID, ok := requireTreasureUser(c)
	if !ok {
		return
	}

	status, err := h.svc.GetStatus(ctx, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]any{
			"code":    500,
			"message": "获取宝箱状态失败",
		})
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"code": 200,
		"data": map[string]any{
			"stat_date":       status.StatDate,
			"watched_seconds": status.WatchedSeconds,
			"coin_balance":    status.CoinBalance,
			"tiers":           status.Tiers,
		},
	})
}

func (h *TreasureHandler) Heartbeat(ctx context.Context, c *app.RequestContext) {
	userID, ok := requireTreasureUser(c)
	if !ok {
		return
	}

	var req struct {
		LiveStreamID int64 `json:"live_stream_id"`
	}
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{
			"code":    400,
			"message": "invalid json",
		})
		return
	}
	if req.LiveStreamID <= 0 {
		c.JSON(http.StatusBadRequest, map[string]any{
			"code":    400,
			"message": "live_stream_id 必须为正整数",
		})
		return
	}

	watchedSeconds, err := h.svc.Heartbeat(ctx, userID, req.LiveStreamID)
	if err != nil {
		writeTreasureHeartbeatError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"code": 200,
		"data": map[string]any{
			"watched_seconds": watchedSeconds,
		},
	})
}

func (h *TreasureHandler) Claim(ctx context.Context, c *app.RequestContext) {
	userID, ok := requireTreasureUser(c)
	if !ok {
		return
	}

	var req struct {
		Tier int `json:"tier"`
	}
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{
			"code":    400,
			"message": "invalid json",
		})
		return
	}
	if req.Tier != 0 && req.Tier != 1 && req.Tier != 2 {
		writeTreasureClaimError(c, service.ErrInvalidTier)
		return
	}

	coins, balance, err := h.svc.Claim(ctx, userID, int8(req.Tier))
	if err != nil {
		writeTreasureClaimError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"code": 200,
		"data": map[string]any{
			"coins":        coins,
			"coin_balance": balance,
		},
	})
}

func requireTreasureUser(c *app.RequestContext) (int64, bool) {
	userID := c.GetInt64("user_id")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, map[string]any{
			"code":    401,
			"message": "未登录或无效用户",
		})
		return 0, false
	}
	return userID, true
}

func writeTreasureHeartbeatError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidLiveStreamID):
		c.JSON(http.StatusBadRequest, map[string]any{
			"code":    400,
			"message": "live_stream_id 必须为正整数",
		})
	case errors.Is(err, service.ErrLiveStreamNotLive):
		c.JSON(http.StatusBadRequest, map[string]any{
			"code":    400,
			"message": "直播间不存在或未开播，观看时长未累计",
		})
	case errors.Is(err, service.ErrLiveStreamLookupUnavailable):
		c.JSON(http.StatusServiceUnavailable, map[string]any{
			"code":    503,
			"message": "直播间状态校验暂不可用，观看时长未累计",
		})
	default:
		c.JSON(http.StatusInternalServerError, map[string]any{
			"code":    500,
			"message": "记录观看时长失败",
		})
	}
}

func writeTreasureClaimError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, service.ErrThresholdNotMet):
		c.JSON(http.StatusBadRequest, map[string]any{
			"code":    400,
			"message": "观看时长未达到领取门槛",
		})
	case errors.Is(err, service.ErrInvalidTier):
		c.JSON(http.StatusBadRequest, map[string]any{
			"code":    400,
			"message": "无效的宝箱档位",
		})
	case errors.Is(err, dao.ErrAlreadyClaimed):
		c.JSON(http.StatusConflict, map[string]any{
			"code":    409,
			"message": "宝箱已领取",
		})
	default:
		c.JSON(http.StatusInternalServerError, map[string]any{
			"code":    500,
			"message": "领取宝箱失败",
		})
	}
}

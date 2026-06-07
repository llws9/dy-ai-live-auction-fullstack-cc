package handler

import (
	"context"
	"log"

	"github.com/cloudwego/hertz/pkg/app"
)

type AuctionCountByLiveStreamProvider interface {
	CountByLiveStreamIDs(ctx context.Context, liveStreamIDs []int64) (map[int64]int64, error)
}

type InternalAuctionCountHandler struct {
	provider AuctionCountByLiveStreamProvider
}

func NewInternalAuctionCountHandler(provider AuctionCountByLiveStreamProvider) *InternalAuctionCountHandler {
	return &InternalAuctionCountHandler{provider: provider}
}

type internalAuctionCountRequest struct {
	LiveStreamIDs []int64 `json:"live_stream_ids"`
}

func (h *InternalAuctionCountHandler) Handle(ctx context.Context, c *app.RequestContext) {
	var req internalAuctionCountRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}

	counts, err := h.provider.CountByLiveStreamIDs(ctx, req.LiveStreamIDs)
	if err != nil {
		log.Printf("internal count-by-live-streams failed: live_stream_ids=%v err=%v", req.LiveStreamIDs, err)
		c.JSON(500, map[string]interface{}{"code": 500, "message": "internal error"})
		return
	}
	if counts == nil {
		counts = map[int64]int64{}
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data":    map[string]interface{}{"counts": counts},
	})
}

package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// AuctionCountByLiveStreamProvider 抽象按 live_stream_id 批量取竞拍总数，
// 由 auction DAO 实现，供内部接口 /internal/auctions/count-by-live-streams 使用。
type AuctionCountByLiveStreamProvider interface {
	CountByLiveStreamIDs(ctx context.Context, liveStreamIDs []int64) (map[int64]int64, error)
}

// InternalAuctionCountHandler 暴露 /internal/auctions/count-by-live-streams 内部接口，
// 仅供同 VPC 的其它服务（product-service）调用，由 InternalAuthMiddleware 鉴权，
// 用于一次性批量获取多个直播间的竞拍数量，避免逐个直播间发起 HTTP 请求（N+1）。
type InternalAuctionCountHandler struct {
	provider AuctionCountByLiveStreamProvider
}

func NewInternalAuctionCountHandler(provider AuctionCountByLiveStreamProvider) *InternalAuctionCountHandler {
	return &InternalAuctionCountHandler{provider: provider}
}

type internalAuctionCountRequest struct {
	LiveStreamIDs []int64 `json:"live_stream_ids"`
}

// CountByLiveStreams 处理 POST /internal/auctions/count-by-live-streams。
// 返回 data.counts: { "<live_stream_id>": <count> }，无记录的 id 不出现在 map 中。
func (h *InternalAuctionCountHandler) CountByLiveStreams(ctx context.Context, c *app.RequestContext) {
	var req internalAuctionCountRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误"})
		return
	}
	counts, err := h.provider.CountByLiveStreamIDs(ctx, req.LiveStreamIDs)
	if err != nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": "获取竞拍数量失败"})
		return
	}
	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data":    map[string]interface{}{"counts": counts},
	})
}

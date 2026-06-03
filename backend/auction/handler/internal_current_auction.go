package handler

import (
	"context"
	"log"

	"github.com/cloudwego/hertz/pkg/app"

	"auction-service/dao"
	"auction-service/model"
)

// CurrentAuctionItem 是 /internal/auctions/current-by-live-streams 单条返回。
type CurrentAuctionItem struct {
	AuctionID    int64  `json:"auction_id"`
	ProductID    int64  `json:"product_id"`
	CurrentPrice string `json:"current_price"`
	Status       int    `json:"status"`
}

// CurrentAuctionFetcher 抽象按 live_stream_id 拉取当前竞品，便于单测注入 fake。
// 返回 map 的 key 为 live_stream_id；无候选的 id 不在 map 中。
type CurrentAuctionFetcher interface {
	Fetch(ctx context.Context, liveStreamIDs []int64) (map[int64]CurrentAuctionItem, error)
}

// InternalCurrentAuctionHandler 暴露内部接口，仅供同 VPC 服务调用，由 InternalAuthMiddleware 鉴权。
type InternalCurrentAuctionHandler struct {
	fetcher CurrentAuctionFetcher
}

func NewInternalCurrentAuctionHandler(fetcher CurrentAuctionFetcher) *InternalCurrentAuctionHandler {
	return &InternalCurrentAuctionHandler{fetcher: fetcher}
}

// internalCurrentAuctionRequest 是请求体。
type internalCurrentAuctionRequest struct {
	LiveStreamIDs []int64 `json:"live_stream_ids"`
}

// Handle 处理 POST /internal/auctions/current-by-live-streams。
func (h *InternalCurrentAuctionHandler) Handle(ctx context.Context, c *app.RequestContext) {
	var req internalCurrentAuctionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}

	current, err := h.fetcher.Fetch(ctx, req.LiveStreamIDs)
	if err != nil {
		log.Printf("internal current-by-live-streams fetch failed: live_stream_ids=%v err=%v", req.LiveStreamIDs, err)
		c.JSON(500, map[string]interface{}{"code": 500, "message": "internal error"})
		return
	}

	items := make([]map[string]interface{}, 0, len(current))
	for liveStreamID, item := range current {
		items = append(items, map[string]interface{}{
			"live_stream_id": liveStreamID,
			"auction_id":     item.AuctionID,
			"product_id":     item.ProductID,
			"current_price":  item.CurrentPrice,
			"status":         item.Status,
		})
	}

	c.JSON(200, map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{"items": items},
	})
}

// CurrentAuctionDAOFetcher 把 AuctionDAO.GetCurrentByLiveStreamIDs 适配成 CurrentAuctionFetcher。
type CurrentAuctionDAOFetcher struct {
	dao *dao.AuctionDAO
}

func NewCurrentAuctionDAOFetcher(d *dao.AuctionDAO) *CurrentAuctionDAOFetcher {
	return &CurrentAuctionDAOFetcher{dao: d}
}

func (f *CurrentAuctionDAOFetcher) Fetch(ctx context.Context, liveStreamIDs []int64) (map[int64]CurrentAuctionItem, error) {
	auctions, err := f.dao.GetCurrentByLiveStreamIDs(ctx, liveStreamIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]CurrentAuctionItem, len(auctions))
	for liveStreamID, a := range auctions {
		out[liveStreamID] = currentAuctionItemFromModel(a)
	}
	return out, nil
}

func currentAuctionItemFromModel(a *model.Auction) CurrentAuctionItem {
	return CurrentAuctionItem{
		AuctionID:    a.ID,
		ProductID:    a.ProductID,
		CurrentPrice: a.CurrentPrice.String(),
		Status:       int(a.Status),
	}
}

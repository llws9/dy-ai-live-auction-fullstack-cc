package handler

import (
	"context"
	"log"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"auction-service/dao"
)

// ---------- next ----------

// NextAuctionItem 是 /internal/auctions/next-by-live-streams 单条返回。
type NextAuctionItem struct {
	AuctionID  int64
	ProductID  int64
	StartPrice string
	StartTime  string
}

// NextAuctionFetcher 抽象按 live_stream_id 拉取下一场竞品，便于单测注入 fake。
type NextAuctionFetcher interface {
	Fetch(ctx context.Context, liveStreamIDs []int64) (map[int64]NextAuctionItem, error)
}

// InternalNextAuctionHandler 暴露内部接口，仅供同 VPC 服务调用，由 InternalAuthMiddleware 鉴权。
type InternalNextAuctionHandler struct {
	fetcher NextAuctionFetcher
}

func NewInternalNextAuctionHandler(f NextAuctionFetcher) *InternalNextAuctionHandler {
	return &InternalNextAuctionHandler{fetcher: f}
}

// internalNextRecentRequest 是请求体，复用于 next 与 recent-deals 两个接口。
type internalNextRecentRequest struct {
	LiveStreamIDs []int64 `json:"live_stream_ids"`
	Limit         int     `json:"limit"`
}

func (h *InternalNextAuctionHandler) Handle(ctx context.Context, c *app.RequestContext) {
	var req internalNextRecentRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}
	got, err := h.fetcher.Fetch(ctx, req.LiveStreamIDs)
	if err != nil {
		log.Printf("internal next-by-live-streams failed: ids=%v err=%v", req.LiveStreamIDs, err)
		c.JSON(500, map[string]interface{}{"code": 500, "message": "internal error"})
		return
	}
	items := make([]map[string]interface{}, 0, len(got))
	for lsID, it := range got {
		items = append(items, map[string]interface{}{
			"live_stream_id": lsID,
			"auction_id":     it.AuctionID,
			"product_id":     it.ProductID,
			"start_price":    it.StartPrice,
			"start_time":     it.StartTime,
		})
	}
	c.JSON(200, map[string]interface{}{"code": 200, "data": map[string]interface{}{"items": items}})
}

// NextAuctionDAOFetcher 把 AuctionDAO.GetNextByLiveStreamIDs 适配成 NextAuctionFetcher。
type NextAuctionDAOFetcher struct{ dao *dao.AuctionDAO }

func NewNextAuctionDAOFetcher(d *dao.AuctionDAO) *NextAuctionDAOFetcher {
	return &NextAuctionDAOFetcher{dao: d}
}

func (f *NextAuctionDAOFetcher) Fetch(ctx context.Context, ids []int64) (map[int64]NextAuctionItem, error) {
	rows, err := f.dao.GetNextByLiveStreamIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]NextAuctionItem, len(rows))
	for lsID, a := range rows {
		out[lsID] = NextAuctionItem{
			AuctionID:  a.ID,
			ProductID:  a.ProductID,
			StartPrice: a.CurrentPrice.String(),
			StartTime:  a.StartTime.Format(time.RFC3339),
		}
	}
	return out, nil
}

// ---------- recent deals ----------

// DealAuctionItem 是 /internal/auctions/recent-deals-by-live-streams 单条成交返回。
type DealAuctionItem struct {
	AuctionID  int64
	ProductID  int64
	FinalPrice string
	EndTime    string
}

// RecentDealsFetcher 抽象按 live_stream_id 拉取最近成交，便于单测注入 fake。
type RecentDealsFetcher interface {
	Fetch(ctx context.Context, liveStreamIDs []int64, n int) (map[int64][]DealAuctionItem, error)
}

// InternalRecentDealsHandler 暴露内部接口，仅供同 VPC 服务调用，由 InternalAuthMiddleware 鉴权。
type InternalRecentDealsHandler struct {
	fetcher RecentDealsFetcher
	limit   int
}

func NewInternalRecentDealsHandler(f RecentDealsFetcher, limit int) *InternalRecentDealsHandler {
	if limit <= 0 {
		limit = 3
	}
	return &InternalRecentDealsHandler{fetcher: f, limit: limit}
}

func (h *InternalRecentDealsHandler) Handle(ctx context.Context, c *app.RequestContext) {
	var req internalNextRecentRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}
	got, err := h.fetcher.Fetch(ctx, req.LiveStreamIDs, h.limit)
	if err != nil {
		log.Printf("internal recent-deals-by-live-streams failed: ids=%v err=%v", req.LiveStreamIDs, err)
		c.JSON(500, map[string]interface{}{"code": 500, "message": "internal error"})
		return
	}
	items := make([]map[string]interface{}, 0, len(got))
	for lsID, deals := range got {
		ds := make([]map[string]interface{}, 0, len(deals))
		for _, d := range deals {
			ds = append(ds, map[string]interface{}{
				"auction_id":  d.AuctionID,
				"product_id":  d.ProductID,
				"final_price": d.FinalPrice,
				"end_time":    d.EndTime,
			})
		}
		items = append(items, map[string]interface{}{"live_stream_id": lsID, "deals": ds})
	}
	c.JSON(200, map[string]interface{}{"code": 200, "data": map[string]interface{}{"items": items}})
}

// RecentDealsDAOFetcher 把 AuctionDAO.GetRecentDealsByLiveStreamIDs 适配成 RecentDealsFetcher。
type RecentDealsDAOFetcher struct{ dao *dao.AuctionDAO }

func NewRecentDealsDAOFetcher(d *dao.AuctionDAO) *RecentDealsDAOFetcher {
	return &RecentDealsDAOFetcher{dao: d}
}

func (f *RecentDealsDAOFetcher) Fetch(ctx context.Context, ids []int64, n int) (map[int64][]DealAuctionItem, error) {
	rows, err := f.dao.GetRecentDealsByLiveStreamIDs(ctx, ids, n)
	if err != nil {
		return nil, err
	}
	out := make(map[int64][]DealAuctionItem, len(rows))
	for lsID, deals := range rows {
		list := make([]DealAuctionItem, 0, len(deals))
		for _, a := range deals {
			list = append(list, DealAuctionItem{
				AuctionID:  a.ID,
				ProductID:  a.ProductID,
				FinalPrice: a.CurrentPrice.String(),
				EndTime:    a.EndTime.Format(time.RFC3339),
			})
		}
		out[lsID] = list
	}
	return out, nil
}

package handler

import (
	"context"
	"fmt"
	"time"

	"auction-service/client"
	"auction-service/model"
)

// auctionFetcher 是 BuildAuctionResultResponse 依赖的最小接口，
// 由 service.AuctionService.GetAuction 提供。
type auctionFetcher func(ctx context.Context, id int64) (*model.Auction, error)

// AuctionResultProduct 是 result 接口内嵌的 product 摘要（spec C §4.2）。
// 与 list 接口不同：result 给完整 images 数组，便于详情页画廊展示。
type AuctionResultProduct struct {
	ID     int64    `json:"id"`
	Name   string   `json:"name"`
	Images []string `json:"images"`
}

// AuctionResultResponse 是 GET /auctions/:id/result 的响应 data 部分。
// 已有字段全部保留，含义不变；新增 won_bid / product。
type AuctionResultResponse struct {
	AuctionID  int64                 `json:"auction_id"`
	ProductID  int64                 `json:"product_id"`
	Status     model.AuctionStatus   `json:"status"`
	FinalPrice float64               `json:"final_price"`
	WinnerID   *int64                `json:"winner_id"`
	StartedAt  time.Time             `json:"started_at"`
	EndedAt    time.Time             `json:"ended_at"`
	DelayUsed  int                   `json:"delay_used"`
	WonBid     float64               `json:"won_bid"`           // 中标价别名，本期等同 final_price
	Product    *AuctionResultProduct `json:"product,omitempty"` // null 表示降级或未找到
}

// BuildAuctionResultResponse 编排 GET /auctions/:id/result：
//  1. 拉 auction（失败上抛 → handler 返回 404）。
//  2. 用 productClient 拉商品摘要；失败或未命中时 product=nil（软降级，用户决策）。
//
// product=nil 软降级（与 list 强失败不同）：result 是单资源详情页，
// 即使 product-service 抖动，winner_id/final_price 等核心字段仍对用户有价值。
func BuildAuctionResultResponse(
	ctx context.Context,
	pc client.ProductClient,
	fetch auctionFetcher,
	id int64,
) (*AuctionResultResponse, error) {
	auction, err := fetch(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get auction: %w", err)
	}

	currentPrice, _ := auction.CurrentPrice.Float64()
	resp := &AuctionResultResponse{
		AuctionID:  auction.ID,
		ProductID:  auction.ProductID,
		Status:     auction.Status,
		FinalPrice: currentPrice,
		WinnerID:   auction.WinnerID,
		StartedAt:  auction.StartTime,
		EndedAt:    auction.EndTime,
		DelayUsed:  auction.DelayUsed,
		WonBid:     currentPrice, // 别名，恒等于 FinalPrice
	}

	if pc == nil {
		return resp, nil
	}

	summaries, err := pc.BatchGetSummaries(ctx, []int64{auction.ProductID})
	if err != nil {
		// 软降级：保留 product=nil，不阻塞 result 主体返回
		return resp, nil
	}
	if s, ok := summaries[auction.ProductID]; ok {
		images := s.Images
		if images == nil {
			images = []string{}
		}
		resp.Product = &AuctionResultProduct{
			ID:     s.ID,
			Name:   s.Name,
			Images: images,
		}
	}
	return resp, nil
}

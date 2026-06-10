package handler

import (
	"context"
	"fmt"
	"log"

	"auction-service/client"
	"auction-service/dao"
	"auction-service/model"

	"github.com/shopspring/decimal"
)

// auctionLister 是 BuildAuctionListResponse 依赖的最小接口，
// 在生产环境由 service.AuctionService.ListAuctionsWithFilters 提供。
type auctionLister func(ctx context.Context, filters *dao.AuctionFilters, page, pageSize int) ([]model.Auction, int64, error)

type auctionRuleBatchFetcher interface {
	GetByProductIDs(ctx context.Context, productIDs []int64) (map[int64]*model.AuctionRule, error)
}

// ListParams 是 GET /auctions 的归一化查询参数。
type ListParams struct {
	Status         *model.AuctionStatus
	LiveStreamID   *int64
	LiveStreamName string
	Search         string
	CategoryID     *int64
	Upcoming       bool
	Page           int
	PageSize       int
	SortByHot      bool
	PriceMin       *decimal.Decimal
	PriceMax       *decimal.Decimal
}

// AuctionProductSummary 是 list 响应里内嵌的商品摘要（spec C §4.1）。
// 与 result 接口不同：list 只回首图（image），不回 images[]，避免放大响应体积。
type AuctionProductSummary struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Image      string `json:"image"`
	CategoryID *int64 `json:"category_id"`
}

// AuctionListItem 是 list 响应中的单条记录：原 auction 字段 + 内嵌 product 摘要。
type AuctionListItem struct {
	model.Auction
	StartPrice  *decimal.Decimal      `json:"start_price,omitempty"`
	Product     AuctionProductSummary `json:"product"`
	ViewerCount int64                 `json:"viewer_count"`
}

// BuildAuctionListResponse 编排 GET /auctions 的核心逻辑：
//  1. 若带 category_id：先调 product-service 拿全分类 product_id 列表，作为 dao IN 过滤；
//     无命中时直接短路返回空结果。
//  2. 调 lister 拿 auction 分页（dao 层只过滤 product_id 集合，不连表 product）。
//  3. 收集本页 product_id 调 batch 拿摘要，回填到每条 auction。
//
// 失败语义（用户决策）：任意下游 client 调用失败 → 整个 list 5xx，无静默降级。
func BuildAuctionListResponse(
	ctx context.Context,
	pc client.ProductClient,
	lsc client.LiveStreamClient,
	lister auctionLister,
	ruleFetcher auctionRuleBatchFetcher,
	p ListParams,
) ([]AuctionListItem, int64, error) {
	filters := &dao.AuctionFilters{
		Status:         p.Status,
		LiveStreamID:   p.LiveStreamID,
		LiveStreamName: p.LiveStreamName,
		Search:         p.Search,
		Upcoming:       p.Upcoming,
		SortByHot:      p.SortByHot,
		PriceMin:       p.PriceMin,
		PriceMax:       p.PriceMax,
	}

	// Step 1: category 过滤
	if p.CategoryID != nil {
		ids, err := pc.ListProductIDsByCategory(ctx, *p.CategoryID)
		if err != nil {
			return nil, 0, fmt.Errorf("list products by category: %w", err)
		}
		if len(ids) == 0 {
			return []AuctionListItem{}, 0, nil
		}
		filters.ProductIDs = ids
	}

	// Step 2: 取 auction 分页
	auctions, total, err := lister(ctx, filters, p.Page, p.PageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list auctions: %w", err)
	}
	if len(auctions) == 0 {
		return []AuctionListItem{}, total, nil
	}

	// Step 3: 收集本页 product_id 调 batch
	productIDs := make([]int64, 0, len(auctions))
	seen := make(map[int64]struct{}, len(auctions))
	for _, a := range auctions {
		if _, ok := seen[a.ProductID]; ok {
			continue
		}
		seen[a.ProductID] = struct{}{}
		productIDs = append(productIDs, a.ProductID)
	}

	summaries, err := pc.BatchGetSummaries(ctx, productIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("batch get product summaries: %w", err)
	}
	rules := map[int64]*model.AuctionRule{}
	if ruleFetcher != nil {
		rules, err = ruleFetcher.GetByProductIDs(ctx, productIDs)
		if err != nil {
			return nil, 0, fmt.Errorf("batch get auction rules: %w", err)
		}
	}

	// Step 3.5: 批量取直播间观看人数（仅 viewer_count）。
	// 降级语义：失败不阻断整页，缺省为 0（装饰性信息不让整页挂）。
	viewerByStream := map[int64]int64{}
	if lsc != nil {
		streamIDs := make([]int64, 0, len(auctions))
		seenStream := make(map[int64]struct{}, len(auctions))
		for _, a := range auctions {
			if a.LiveStreamID == nil || *a.LiveStreamID <= 0 {
				continue
			}
			if _, ok := seenStream[*a.LiveStreamID]; ok {
				continue
			}
			seenStream[*a.LiveStreamID] = struct{}{}
			streamIDs = append(streamIDs, *a.LiveStreamID)
		}
		if len(streamIDs) > 0 {
			if streams, lerr := lsc.BatchGetLiveStreams(ctx, streamIDs); lerr != nil {
				log.Printf("[WARN] auction list: batch live streams for viewer_count failed (degraded): %v", lerr)
			} else {
				for id, s := range streams {
					viewerByStream[id] = s.ViewerCount
				}
			}
		}
	}

	// Step 4: 回填
	out := make([]AuctionListItem, 0, len(auctions))
	hidden := int64(0)
	for _, a := range auctions {
		if s, ok := summaries[a.ProductID]; ok {
			item := AuctionListItem{Auction: a}
			item.Product = AuctionProductSummary{
				ID:         s.ID,
				Name:       s.Name,
				Image:      firstImage(s.Images),
				CategoryID: s.CategoryID,
			}
			if rule, ok := rules[a.ProductID]; ok && rule != nil {
				item.StartPrice = &rule.StartPrice
			}
			if a.LiveStreamID != nil {
				item.ViewerCount = viewerByStream[*a.LiveStreamID]
			}
			out = append(out, item)
			continue
		}
		hidden++
	}
	if hidden > 0 {
		total -= hidden
		if total < 0 {
			total = 0
		}
	}
	return out, total, nil
}

// firstImage 安全提取第一张图片 URL；空时返回 ""。
func firstImage(imgs []string) string {
	if len(imgs) == 0 {
		return ""
	}
	return imgs[0]
}

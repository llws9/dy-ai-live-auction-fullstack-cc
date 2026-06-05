package handler

import (
	"context"
	"fmt"

	"auction-service/client"
	"auction-service/dao"
	"auction-service/model"
)

// auctionLister 是 BuildAuctionListResponse 依赖的最小接口，
// 在生产环境由 service.AuctionService.ListAuctionsWithFilters 提供。
type auctionLister func(ctx context.Context, filters *dao.AuctionFilters, page, pageSize int) ([]model.Auction, int64, error)

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
	Product AuctionProductSummary `json:"product"`
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
	lister auctionLister,
	p ListParams,
) ([]AuctionListItem, int64, error) {
	filters := &dao.AuctionFilters{
		Status:         p.Status,
		LiveStreamID:   p.LiveStreamID,
		LiveStreamName: p.LiveStreamName,
		Search:         p.Search,
		Upcoming:       p.Upcoming,
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

	// Step 4: 回填
	out := make([]AuctionListItem, 0, len(auctions))
	for _, a := range auctions {
		item := AuctionListItem{Auction: a}
		if s, ok := summaries[a.ProductID]; ok {
			item.Product = AuctionProductSummary{
				ID:         s.ID,
				Name:       s.Name,
				Image:      firstImage(s.Images),
				CategoryID: s.CategoryID,
			}
		}
		out = append(out, item)
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

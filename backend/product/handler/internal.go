package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"product-service/model"
	"product-service/service"
)

type InternalHandler struct {
	productService  *service.ProductService
	liveStreamDAO   liveStreamBatchProvider
	userAvatarDAO   userAvatarProvider
	auctionCountDAO auctionCountProvider
}

type liveStreamBatchProvider interface {
	GetByIDs(ctx context.Context, ids []int64) (map[int64]*model.LiveStream, error)
}

type userAvatarProvider interface {
	GetAvatarsByIDs(ctx context.Context, ids []int64) (map[int64]string, error)
}

type auctionCountProvider interface {
	CountActiveByLiveStreamIDs(ctx context.Context, ids []int64) (map[int64]int, error)
}

func NewInternalHandler(productService *service.ProductService, liveStreamDAO liveStreamBatchProvider, userAvatarDAO userAvatarProvider, auctionCountDAO auctionCountProvider) *InternalHandler {
	return &InternalHandler{
		productService:  productService,
		liveStreamDAO:   liveStreamDAO,
		userAvatarDAO:   userAvatarDAO,
		auctionCountDAO: auctionCountDAO,
	}
}

type productSummary struct {
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	Images     []string `json:"images"`
	CategoryID *int64   `json:"category_id"`
}

func toSummary(p model.Product) productSummary {
	imgs := []string(p.Images)
	if imgs == nil {
		imgs = []string{}
	}
	return productSummary{
		ID:         p.ID,
		Name:       p.Name,
		Images:     imgs,
		CategoryID: p.CategoryID,
	}
}

func (h *InternalHandler) ListByCategory(ctx context.Context, c *app.RequestContext) {
	categoryIDStr := string(c.Query("category_id"))
	if categoryIDStr == "" {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "category_id 必填"})
		return
	}
	categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
	if err != nil || categoryID <= 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "category_id 非法"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "500"))

	items, total, err := h.productService.ListProductsByCategory(ctx, categoryID, page, pageSize)
	if err != nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": "查询失败: " + err.Error()})
		return
	}

	summaries := make([]productSummary, 0, len(items))
	for _, p := range items {
		summaries = append(summaries, toSummary(p))
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data": map[string]interface{}{
			"items":     summaries,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

type BatchByIDsRequest struct {
	IDs []int64 `json:"ids"`
}

func (h *InternalHandler) BatchByIDs(ctx context.Context, c *app.RequestContext) {
	var req BatchByIDsRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "ids 不能为空"})
		return
	}

	items, err := h.productService.GetProductsByIDs(ctx, req.IDs)
	if err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": err.Error()})
		return
	}

	summaries := make([]productSummary, 0, len(items))
	for _, p := range items {
		summaries = append(summaries, toSummary(p))
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data": map[string]interface{}{
			"items": summaries,
		},
	})
}

type liveStreamSummary struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	CoverImage   string  `json:"cover_image"`
	Status       int     `json:"status"`
	CreatorID    int64   `json:"creator_id"`
	HostAvatar   *string `json:"host_avatar"`
	AuctionCount *int    `json:"auction_count"`
}

const internalLiveStreamBatchMaxIDs = 200

func (h *InternalHandler) BatchLiveStreams(ctx context.Context, c *app.RequestContext) {
	var req BatchByIDsRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "ids 不能为空"})
		return
	}
	if len(req.IDs) > internalLiveStreamBatchMaxIDs {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "ids 超出上限"})
		return
	}

	items, err := h.liveStreamDAO.GetByIDs(ctx, req.IDs)
	if err != nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": "查询失败: " + err.Error()})
		return
	}

	creatorIDs := make([]int64, 0, len(items))
	seenCreator := make(map[int64]struct{}, len(items))
	for _, ls := range items {
		if ls.CreatorID == 0 {
			continue
		}
		if _, ok := seenCreator[ls.CreatorID]; ok {
			continue
		}
		seenCreator[ls.CreatorID] = struct{}{}
		creatorIDs = append(creatorIDs, ls.CreatorID)
	}

	avatars := map[int64]string{}
	if h.userAvatarDAO != nil && len(creatorIDs) > 0 {
		avatars, _ = h.userAvatarDAO.GetAvatarsByIDs(ctx, creatorIDs)
	}

	counts := map[int64]int{}
	if h.auctionCountDAO != nil {
		counts, _ = h.auctionCountDAO.CountActiveByLiveStreamIDs(ctx, req.IDs)
	}

	summaries := make([]liveStreamSummary, 0, len(items))
	seen := make(map[int64]struct{}, len(req.IDs))
	for _, id := range req.IDs {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ls, ok := items[id]
		if !ok {
			continue
		}
		s := liveStreamSummary{
			ID:         ls.ID,
			Name:       ls.Name,
			CoverImage: ls.CoverImage,
			Status:     int(ls.Status),
			CreatorID:  ls.CreatorID,
		}
		if avatar, hit := avatars[ls.CreatorID]; hit && avatar != "" {
			s.HostAvatar = &avatar
		}
		if cnt, hit := counts[ls.ID]; hit {
			s.AuctionCount = &cnt
		}
		summaries = append(summaries, s)
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data": map[string]interface{}{
			"items": summaries,
		},
	})
}

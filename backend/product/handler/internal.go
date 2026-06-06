package handler

import (
	"context"
	"errors"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"

	"product-service/model"
	"product-service/service"
)

// InternalHandler 暴露 /internal/* 内部接口，仅供同 VPC 的其它服务调用，
// 禁止注册到 Gateway（spec C §5.3 / §6.3）。
type InternalHandler struct {
	productService *service.ProductService
	liveStreamDAO  liveStreamBatchProvider
}

// liveStreamBatchProvider 抽象 LiveStreamDAO.GetByIDs，便于 handler 单测注入 fake。
type liveStreamBatchProvider interface {
	GetByIDs(ctx context.Context, ids []int64) (map[int64]*model.LiveStream, error)
}

// NewInternalHandler 创建内部接口 Handler。
func NewInternalHandler(productService *service.ProductService, liveStreamDAO liveStreamBatchProvider) *InternalHandler {
	return &InternalHandler{
		productService: productService,
		liveStreamDAO:  liveStreamDAO,
	}
}

// productSummary 是 list 场景下回给调用方的精简摘要。
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

// ListByCategory 处理 GET /internal/products?category_id=&page=&page_size=
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

// BatchByIDsRequest 是 POST /internal/products/batch 的请求体。
type BatchByIDsRequest struct {
	IDs []int64 `json:"ids"`
}

type getOrCreateLiveStreamRequest struct {
	CreatorID   int64  `json:"creator_id"`
	CreatorName string `json:"creator_name"`
}

// BatchByIDs 处理 POST /internal/products/batch，按 id 列表批量取摘要。
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
		// service 仅在超长时返回错误（见 spec §5.1.1），按 400 回。
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

func (h *InternalHandler) GetAuctionProductInfo(ctx context.Context, c *app.RequestContext) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "无效的商品ID"})
		return
	}
	info, err := h.productService.GetProductAuctionInfo(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, map[string]interface{}{"code": 404, "message": err.Error()})
			return
		}
		c.JSON(500, map[string]interface{}{"code": 500, "message": "查询失败: " + err.Error()})
		return
	}
	c.JSON(200, map[string]interface{}{"code": 200, "message": "success", "data": info})
}

// liveStreamSummary 是 /internal/live-streams/batch 的返回单元（spec B §4.1 契约）。
type liveStreamSummary struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	CoverImage string `json:"cover_image"`
	Status     int    `json:"status"`
	CreatorID  int64  `json:"creator_id"`
}

const internalLiveStreamBatchMaxIDs = 200

// BatchLiveStreams 处理 POST /internal/live-streams/batch（T3.3 / spec B §4.1）。
// 入参 ids（非空、长度 ≤ 200），返回按 id 命中的直播间摘要列表，缺失 id 跳过。
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
		summaries = append(summaries, liveStreamSummary{
			ID:         ls.ID,
			Name:       ls.Name,
			CoverImage: ls.CoverImage,
			Status:     int(ls.Status),
			CreatorID:  ls.CreatorID,
		})
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data": map[string]interface{}{
			"items": summaries,
		},
	})
}

func (h *InternalHandler) GetOrCreateActiveLiveStream(ctx context.Context, c *app.RequestContext) {
	var req getOrCreateLiveStreamRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "请求参数错误: " + err.Error()})
		return
	}
	if req.CreatorID <= 0 {
		c.JSON(400, map[string]interface{}{"code": 400, "message": "creator_id 必填"})
		return
	}

	liveStream, err := h.productService.GetOrCreateLiveStream(ctx, req.CreatorID, req.CreatorName)
	if err != nil {
		c.JSON(500, map[string]interface{}{"code": 500, "message": "获取直播间失败: " + err.Error()})
		return
	}
	if !liveStream.IsActive() {
		c.JSON(409, map[string]interface{}{"code": 409, "message": "直播间已被禁用，无法创建竞拍"})
		return
	}

	c.JSON(200, map[string]interface{}{"code": 200, "message": "success", "data": liveStreamSummary{
		ID:         liveStream.ID,
		Name:       liveStream.Name,
		CoverImage: liveStream.CoverImage,
		Status:     int(liveStream.Status),
		CreatorID:  liveStream.CreatorID,
	}})
}

package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"auction-service/client"
	"auction-service/dao"
	"auction-service/model"
	"auction-service/service"
)

// AuctionHandler 竞拍 Handler
type AuctionHandler struct {
	auctionService *service.AuctionService
	// productClient 用于在 List 中按 category_id 过滤、批量回填商品摘要。
	// 可为 nil（向后兼容），但若客户端传 category_id 会得到 503。
	productClient client.ProductClient
}

// NewAuctionHandler 创建竞拍 Handler
func NewAuctionHandler(auctionService *service.AuctionService) *AuctionHandler {
	return &AuctionHandler{
		auctionService: auctionService,
	}
}

// SetProductClient 注入 product-service 内部接口客户端。
func (h *AuctionHandler) SetProductClient(pc client.ProductClient) {
	h.productClient = pc
}

// CreateAuctionRequest 创建竞拍请求
type CreateAuctionRequest struct {
	ProductID  int64   `json:"product_id" binding:"required"`
	StartPrice float64 `json:"start_price"`
	Increment  float64 `json:"increment"`
	Duration   int     `json:"duration" binding:"required"`
}

// Create 创建竞拍场次
// @Summary 创建竞拍
// @Description 创建新的竞拍场次
// @Tags auction
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateAuctionRequest true "竞拍信息"
// @Success 201 {object} model.Auction
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auctions [post]
func (h *AuctionHandler) Create(ctx context.Context, c *app.RequestContext) {
	var req CreateAuctionRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 创建竞拍场次请求
	auctionReq := &service.CreateAuctionRequest{
		ProductID: req.ProductID,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Duration(req.Duration) * time.Second),
	}

	auction, err := h.auctionService.CreateAuction(ctx, auctionReq)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "创建竞拍失败: " + err.Error(),
		})
		return
	}

	c.JSON(201, auction)
}

// Cancel 取消竞拍
// @Summary 取消竞拍
// @Description 取消指定的竞拍场次
// @Tags auction
// @Produce json
// @Security BearerAuth
// @Param id path int true "竞拍ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auctions/{id}/cancel [put]
func (h *AuctionHandler) Cancel(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的竞拍ID",
		})
		return
	}

	if err := h.auctionService.CancelAuction(ctx, id); err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "取消竞拍失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "竞拍已取消",
	})
}

// GetResult 获取竞拍结果
// @Summary 获取竞拍结果
// @Description 获取指定竞拍的最终结果
// @Tags auction
// @Produce json
// @Param id path int true "竞拍ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /auctions/{id}/result [get]
func (h *AuctionHandler) GetResult(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的竞拍ID",
		})
		return
	}

	auction, err := h.auctionService.GetAuction(ctx, id)
	if err != nil {
		c.JSON(404, map[string]interface{}{
			"code":    404,
			"message": "竞拍不存在",
		})
		return
	}

	// 构建结果响应
	result := map[string]interface{}{
		"auction_id":    auction.ID,
		"product_id":    auction.ProductID,
		"status":        auction.Status,
		"final_price":   auction.CurrentPrice,
		"winner_id":     auction.WinnerID,
		"started_at":    auction.StartTime,
		"ended_at":      auction.EndTime,
		"delay_used":    auction.DelayUsed,
	}

	c.JSON(200, result)
}

// Get 获取竞拍详情
// @Summary 获取竞拍详情
// @Description 获取指定竞拍的详细信息
// @Tags auction
// @Produce json
// @Param id path int true "竞拍ID"
// @Success 200 {object} model.Auction
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /auctions/{id} [get]
func (h *AuctionHandler) Get(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的竞拍ID",
		})
		return
	}

	auction, err := h.auctionService.GetAuction(ctx, id)
	if err != nil {
		c.JSON(404, map[string]interface{}{
			"code":    404,
			"message": "竞拍不存在",
		})
		return
	}

	c.JSON(200, auction)
}

// List 获取竞拍列表
// @Summary 获取竞拍列表
// @Description 获取竞拍列表，支持按状态筛选和分页
// @Tags auction
// @Produce json
// @Param status query int false "竞拍状态：0-待开始, 1-进行中, 2-已结束, 3-已取消"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auctions [get]
func (h *AuctionHandler) List(ctx context.Context, c *app.RequestContext) {
	// 解析查询参数
	statusStr := c.Query("status")
	liveStreamIDStr := c.Query("live_stream_id")
	liveStreamName := c.Query("live_stream_name")
	search := c.Query("search")
	categoryIDStr := c.Query("category_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 归一化为 ListParams
	params := ListParams{
		LiveStreamName: liveStreamName,
		Search:         search,
		Page:           page,
		PageSize:       pageSize,
	}
	if statusStr != "" {
		if s, err := strconv.Atoi(statusStr); err == nil {
			st := model.AuctionStatus(s)
			params.Status = &st
		}
	}
	if liveStreamIDStr != "" {
		if id, err := strconv.ParseInt(liveStreamIDStr, 10, 64); err == nil {
			params.LiveStreamID = &id
		}
	}
	if categoryIDStr != "" {
		if cid, err := strconv.ParseInt(categoryIDStr, 10, 64); err == nil && cid > 0 {
			params.CategoryID = &cid
		}
	}

	// 走带 product 摘要回填的编排路径（spec C §5.2）。
	if h.productClient != nil {
		items, total, err := BuildAuctionListResponse(ctx, h.productClient, h.auctionService.ListAuctionsWithFilters, params)
		if err != nil {
			c.JSON(500, map[string]interface{}{
				"code":    500,
				"message": "获取竞拍列表失败: " + err.Error(),
			})
			return
		}
		c.JSON(200, map[string]interface{}{
			"code":    200,
			"message": "success",
			"data": map[string]interface{}{
				"items":     items,
				"total":     total,
				"page":      page,
				"page_size": pageSize,
			},
		})
		return
	}

	// 旧路径：未注入 productClient 时走原有逻辑（保持向后兼容，单元测试用）
	var filters *dao.AuctionFilters
	if statusStr != "" || liveStreamIDStr != "" || liveStreamName != "" || search != "" {
		filters = &dao.AuctionFilters{
			Status:         params.Status,
			LiveStreamID:   params.LiveStreamID,
			LiveStreamName: liveStreamName,
			Search:         search,
		}
	}

	var auctions []model.Auction
	var total int64
	var err error
	if filters != nil {
		auctions, total, err = h.auctionService.ListAuctionsWithFilters(ctx, filters, page, pageSize)
	} else {
		auctions, total, err = h.auctionService.ListAuctions(ctx, params.Status, page, pageSize)
	}
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取竞拍列表失败: " + err.Error(),
		})
		return
	}
	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "success",
		"data": map[string]interface{}{
			"items":     auctions,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetBids 获取竞拍出价记录
// @Summary 获取竞拍出价记录
// @Description 获取指定竞拍的所有出价记录
// @Tags auction
// @Produce json
// @Param id path int true "竞拍ID"
// @Param limit query int false "返回数量限制" default(100)
// @Success 200 {array} model.Bid
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auctions/{id}/bids [get]
func (h *AuctionHandler) GetBids(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的竞拍ID",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	bids, err := h.auctionService.GetAuctionBids(ctx, id, limit)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取出价记录失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, bids)
}

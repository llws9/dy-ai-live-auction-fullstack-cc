package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"auction-service/model"
	"auction-service/service"
)

// AuctionHandler 竞拍 Handler
type AuctionHandler struct {
	auctionService *service.AuctionService
}

// NewAuctionHandler 创建竞拍 Handler
func NewAuctionHandler(auctionService *service.AuctionService) *AuctionHandler {
	return &AuctionHandler{
		auctionService: auctionService,
	}
}

// CreateAuctionRequest 创建竞拍请求
type CreateAuctionRequest struct {
	ProductID  int64 `json:"product_id" binding:"required"`
	StartPrice float64 `json:"start_price"`
	Increment  float64 `json:"increment"`
	Duration   int     `json:"duration" binding:"required"`
}

// Create 创建竞拍场次
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
func (h *AuctionHandler) List(ctx context.Context, c *app.RequestContext) {
	// 解析查询参数
	statusStr := c.Query("status")
	var status *model.AuctionStatus
	if statusStr != "" {
		s, err := strconv.Atoi(statusStr)
		if err == nil {
			st := model.AuctionStatus(s)
			status = &st
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 获取竞拍列表
	auctions, total, err := h.auctionService.ListAuctions(ctx, status, page, pageSize)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取竞拍列表失败: " + err.Error(),
		})
		return
	}

	// 构建响应
	response := map[string]interface{}{
		"auctions":   auctions,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	}

	c.JSON(200, response)
}

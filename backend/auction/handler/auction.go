package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

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

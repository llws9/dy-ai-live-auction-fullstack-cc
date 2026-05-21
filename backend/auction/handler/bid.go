package handler

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"auction-service/service"
)

// BidHandler 出价 Handler
type BidHandler struct {
	bidService *service.BidService
}

// NewBidHandler 创建出价 Handler
func NewBidHandler(bidService *service.BidService) *BidHandler {
	return &BidHandler{
		bidService: bidService,
	}
}

// PlaceBidRequest 出价请求
type PlaceBidRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

// PlaceBid 出价
func (h *BidHandler) PlaceBid(ctx context.Context, c *app.RequestContext) {
	// 解析竞拍 ID
	idStr := c.Param("id")
	auctionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的竞拍ID",
		})
		return
	}

	// 解析请求体
	var req PlaceBidRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 从JWT上下文获取用户ID
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, map[string]interface{}{
			"code":    401,
			"message": "未认证，请先登录",
		})
		return
	}
	userID := userIDInterface.(int64)

	// 调用服务层出价
	result, err := h.bidService.PlaceBid(ctx, &service.PlaceBidRequest{
		AuctionID: auctionID,
		UserID:    userID,
		Amount:    req.Amount,
	})

	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "出价失败: " + err.Error(),
		})
		return
	}

	// 根据成功或失败返回不同状态码
	if result.Success {
		c.JSON(200, result)
	} else {
		c.JSON(400, result)
	}
}

// GetRanking 获取当前排名
func (h *BidHandler) GetRanking(ctx context.Context, c *app.RequestContext) {
	idStr := c.Param("id")
	auctionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的竞拍ID",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	bids, err := h.bidService.GetRanking(ctx, auctionID, limit)
	if err != nil {
		c.JSON(500, map[string]interface{}{
			"code":    500,
			"message": "获取排名失败: " + err.Error(),
		})
		return
	}

	// 构建排名列表响应
	ranking := make([]map[string]interface{}, len(bids))
	for i, bid := range bids {
		ranking[i] = map[string]interface{}{
			"rank":      i + 1,
			"user_id":   bid.UserID,
			"amount":    bid.Amount,
			"bid_time":  bid.CreatedAt,
		}
	}

	c.JSON(200, map[string]interface{}{
		"items": ranking,
	})
}

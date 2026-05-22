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
	UserID int64   `json:"user_id,omitempty"` // 用于测试，生产环境应删除
}

// PlaceBid 出价
// @Summary 出价竞拍
// @Description 在指定竞拍中出价
// @Tags bid
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "竞拍ID"
// @Param body body PlaceBidRequest true "出价信息"
// @Success 200 {object} service.PlaceBidResult
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auctions/{id}/bids [post]
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

	// 从JWT上下文获取用户ID，如果没有则从请求体获取（测试模式）
	var userID int64
	userIDInterface, exists := c.Get("user_id")
	if exists {
		userID = userIDInterface.(int64)
	} else if req.UserID > 0 {
		// 测试模式：允许从请求体获取user_id
		userID = req.UserID
	} else {
		c.JSON(401, map[string]interface{}{
			"code":    401,
			"message": "未认证，请先登录",
		})
		return
	}

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
// @Summary 获取竞拍排名
// @Description 获取指定竞拍的出价排名列表
// @Tags bid
// @Produce json
// @Param id path int true "竞拍ID"
// @Param limit query int false "返回数量" default(10)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auctions/{id}/ranking [get]
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

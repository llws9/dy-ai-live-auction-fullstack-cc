package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/gorilla/websocket"

	ws "auction-service/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应限制
	},
}

// WSHandler WebSocket Handler
type WSHandler struct {
	hub *ws.Hub
}

// NewWSHandler 创建 WebSocket Handler
func NewWSHandler(hub *ws.Hub) *WSHandler {
	return &WSHandler{
		hub: hub,
	}
}

// Handle 处理 WebSocket 连接
func (h *WSHandler) Handle(ctx context.Context, c *app.RequestContext) {
	// 获取竞拍 ID
	auctionIDStr := c.Query("auction_id")
	if auctionIDStr == "" {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "缺少 auction_id 参数",
		})
		return
	}

	auctionID, err := strconv.ParseInt(auctionIDStr, 10, 64)
	if err != nil {
		c.JSON(400, map[string]interface{}{
			"code":    400,
			"message": "无效的 auction_id",
		})
		return
	}

	// 获取用户 ID（简化实现）
	userIDStr := c.GetHeader("X-User-ID")
	userID, _ := strconv.ParseInt(string(userIDStr), 10, 64)
	if userID == 0 {
		userID = 1 // 默认用户
	}

	// 获取底层 HTTP 连接（Hertz 特殊处理）
	// 注意：Hertz 的 WebSocket 支持需要特殊处理
	// 这里简化实现，直接返回提示
	c.JSON(200, map[string]interface{}{
		"code":    200,
		"message": "WebSocket 连接端点",
		"hint":    "请使用 WebSocket 客户端连接",
		"auction_id": auctionID,
		"user_id": userID,
	})
}

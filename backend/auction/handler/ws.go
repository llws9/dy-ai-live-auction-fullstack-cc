package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	ws "auction-service/websocket"
)

// WebSocket 连接管理
var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许所有来源，生产环境应限制
		},
	}
)

// WSHandler WebSocket Handler
type WSHandler struct {
	hub *ws.Hub
}

// NewWSHandler 创建 WebSocket Handler
func NewWSHandler() *WSHandler {
	return &WSHandler{}
}

// SetHub 设置 WebSocket Hub
func (h *WSHandler) SetHub(hub *ws.Hub) {
	h.hub = hub
}

// HandleWebSocket 处理 WebSocket 连接
func (h *WSHandler) HandleWebSocket(hub *ws.Hub, auctionID int64, w http.ResponseWriter, r *http.Request) {
	// 升级 HTTP 连接为 WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// 获取用户 ID
	userIDStr := r.URL.Query().Get("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || userID == 0 {
		userID = time.Now().UnixNano() % 10000 // 生成临时用户ID
	}

	// 创建客户端
	client := ws.NewClientSimple(conn, auctionID, userID)

	// 注册到 Hub
	if hub != nil {
		hub.Register <- client
	} else if h.hub != nil {
		h.hub.Register <- client
	}

	log.Printf("WebSocket connected: auction=%d, user=%d", auctionID, userID)

	// 发送欢迎消息
	welcomeMsg := &ws.Message{
		Type:      "system",
		Timestamp: time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"message":    "连接成功",
			"user_id":    userID,
			"auction_id": auctionID,
		},
	}
	data, _ := json.Marshal(welcomeMsg)
	conn.WriteMessage(websocket.TextMessage, data)

	// 读取消息循环
	go func() {
		defer func() {
			if hub != nil {
				hub.Unregister <- client
			} else if h.hub != nil {
				h.hub.Unregister <- client
			}
			conn.Close()
			log.Printf("WebSocket disconnected: auction=%d, user=%d", auctionID, userID)
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				break
			}

			// 处理客户端消息
			var clientMsg map[string]interface{}
			if err := json.Unmarshal(message, &clientMsg); err != nil {
				continue
			}

			msgType, ok := clientMsg["type"].(string)
			if !ok {
				continue
			}

			switch msgType {
			case "ping":
				pingMsg := &ws.Message{
					Type:      "pong",
					Timestamp: time.Now().UnixMilli(),
					Data:      "pong",
				}
				data, _ := json.Marshal(pingMsg)
				conn.WriteMessage(websocket.TextMessage, data)
			}
		}
	}()
}

// BroadcastPriceUpdate 广播价格更新
func BroadcastPriceUpdate(hub *ws.Hub, auctionID int64, userID int64, price float64, rank int) {
	msg := &ws.Message{
		Type:      "price_update",
		Timestamp: time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"user_id":    userID,
			"price":      price,
			"rank":       rank,
			"auction_id": auctionID,
		},
	}
	hub.BroadcastToRoom(auctionID, msg)
}

// BroadcastCountdown 广播倒计时
func BroadcastCountdown(hub *ws.Hub, auctionID int64, remainingMs int64) {
	msg := &ws.Message{
		Type:      "countdown",
		Timestamp: time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"remaining_ms": remainingMs,
			"auction_id":   auctionID,
		},
	}
	hub.BroadcastToRoom(auctionID, msg)
}

// BroadcastAuctionEnd 广播竞拍结束
func BroadcastAuctionEnd(hub *ws.Hub, auctionID int64, winnerID int64, finalPrice float64) {
	msg := &ws.Message{
		Type:      "auction_end",
		Timestamp: time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"winner_id":   winnerID,
			"final_price": finalPrice,
			"auction_id":  auctionID,
		},
	}
	hub.BroadcastToRoom(auctionID, msg)
}
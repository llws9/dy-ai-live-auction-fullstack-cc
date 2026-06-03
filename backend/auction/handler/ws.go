package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	ws "auction-service/websocket"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
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
	hub         *ws.Hub
	jwtSecret   string
	chatHandler *ws.ChatHandler
}

// NewWSHandler 创建 WebSocket Handler
func NewWSHandler() *WSHandler {
	return &WSHandler{}
}

// SetHub 设置 WebSocket Hub
func (h *WSHandler) SetHub(hub *ws.Hub) {
	h.hub = hub
}

// SetJWTSecret 设置 JWT 密钥
func (h *WSHandler) SetJWTSecret(secret string) {
	h.jwtSecret = secret
}

// SetChatHandler 注入弹幕 handler
func (h *WSHandler) SetChatHandler(ch *ws.ChatHandler) {
	h.chatHandler = ch
}

// HandleWebSocket 处理 WebSocket 连接
func (h *WSHandler) HandleWebSocket(hub *ws.Hub, auctionID int64, w http.ResponseWriter, r *http.Request) {
	// 获取用户ID（优先从token验证）
	var userID int64
	var userName string
	authenticated := false

	// 尝试从token参数验证
	tokenStr := r.URL.Query().Get("token")
	if tokenStr != "" && h.jwtSecret != "" {
		claims, err := h.validateToken(tokenStr)
		if err == nil && claims != nil {
			userID = claims.UserID
			userName = claims.Username
			authenticated = true
		} else {
			log.Printf("WebSocket auth failed: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	} else {
		// 兼容旧方式（user_id参数），但不推荐
		userIDStr := r.URL.Query().Get("user_id")
		var err error
		userID, err = strconv.ParseInt(userIDStr, 10, 64)
		if err != nil || userID == 0 {
			// 如果没有token也没有user_id，拒绝连接
			http.Error(w, "Missing authentication", http.StatusUnauthorized)
			return
		}
	}

	// 直播间订阅（可选）：握手 URL 携带 live_stream_id 才订阅弹幕
	liveStreamID, _ := strconv.ParseInt(r.URL.Query().Get("live_stream_id"), 10, 64)

	// 升级 HTTP 连接为 WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// 选择有效的 Hub（优先入参，其次实例字段）
	activeHub := hub
	if activeHub == nil {
		activeHub = h.hub
	}

	// 创建客户端并装配 Hub / ChatHandler
	clientID := fmt.Sprintf("%d-%d-%d", auctionID, userID, time.Now().UnixNano())
	client := ws.NewClient(clientID, auctionID, userID, liveStreamID, userName, authenticated, conn, activeHub)
	if h.chatHandler != nil {
		client.SetChatHandler(h.chatHandler)
	}

	// 注册到 Hub（registerClient 内部会按 live_stream_id 双注册弹幕房间）
	if activeHub != nil {
		activeHub.Register <- client
	}

	log.Printf("WebSocket connected: auction=%d, user=%d, live_stream=%d", auctionID, userID, liveStreamID)

	// 发送欢迎消息（必须在 WritePump 启动前直接写，避免与 WritePump 并发写同一连接）
	welcomeMsg := &ws.Message{
		Type:      "system",
		Timestamp: time.Now().UnixMilli(),
		Data: map[string]interface{}{
			"message":    "连接成功",
			"user_id":    userID,
			"auction_id": auctionID,
		},
	}
	client.Send <- welcomeMsg

	// 启动标准读写泵：ReadPump 处理 ping/sync_request/chat_send，WritePump 统一写出
	go client.ReadPump()
	go client.WritePump()
}

// BroadcastPriceUpdate 广播价格更新
func BroadcastPriceUpdate(hub *ws.Hub, auctionID int64, userID int64, price decimal.Decimal, rank int) {
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
func BroadcastAuctionEnd(hub *ws.Hub, auctionID int64, winnerID int64, finalPrice decimal.Decimal) {
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

// JWTClaims JWT声明
type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// validateToken 验证JWT Token
func (h *WSHandler) validateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	return claims, nil
}

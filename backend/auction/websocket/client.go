package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 写超时
	writeWait = 10 * time.Second

	// Pong 超时
	pongWait = 60 * time.Second

	// Ping 发送间隔
	pingPeriod = 30 * time.Second

	// 发送缓冲区大小
	sendBufferSize = 256
)

// Client WebSocket 客户端
type Client struct {
	ID            string
	AuctionID     int64
	LiveStreamID  int64 // 直播间 ID（0 表示未订阅弹幕）
	UserID        int64
	UserName      string // 发弹幕时回填到广播
	Authenticated bool   // true 表示身份来自服务端验证过的 JWT
	ConnectedAt   time.Time

	conn *websocket.Conn
	Send chan *Message

	hub          *Hub
	stateManager *StateManager
	chatHandler  *ChatHandler

	closeOnce sync.Once
	closed    bool
}

// NewClient 创建客户端
func NewClient(id string, auctionID, userID, liveStreamID int64, userName string, authenticated bool, conn *websocket.Conn, hub *Hub) *Client {
	now := time.Now()
	client := &Client{
		ID:            id,
		AuctionID:     auctionID,
		LiveStreamID:  liveStreamID,
		UserID:        userID,
		UserName:      userName,
		Authenticated: authenticated,
		ConnectedAt:   now,
		conn:          conn,
		Send:          make(chan *Message, sendBufferSize),
		hub:           hub,
	}

	// 保存连接状态到 Redis
	if hub != nil && hub.GetStateManager() != nil {
		sm := hub.GetStateManager()
		client.SetStateManager(sm)
		state := &ConnectionState{
			ClientID:       client.ID,
			AuctionID:      client.AuctionID,
			UserID:         client.UserID,
			ConnectedAt:    client.ConnectedAt,
			LastPongAt:     client.ConnectedAt,
			ReconnectCount: 0,
		}

		ctx := context.Background()
		// 检查是否为重连
		existingState, err := sm.GetConnectionState(ctx, client.ID)
		if err == nil && existingState != nil {
			state.ReconnectCount = existingState.ReconnectCount + 1
		}

		if err := sm.SaveConnectionState(ctx, state); err != nil {
			log.Printf("Failed to save connection state: %v", err)
		}
	}

	return client
}

// SetStateManager 设置状态管理器
func (c *Client) SetStateManager(sm *StateManager) {
	c.stateManager = sm
}

// SetChatHandler 注入弹幕处理器
func (c *Client) SetChatHandler(h *ChatHandler) {
	c.chatHandler = h
}

// SetHub 设置所属 Hub（ReadPump 注销时需要）
func (c *Client) SetHub(hub *Hub) {
	c.hub = hub
}

// NewClientSimple 创建客户端（简化版，自动生成ID）
func NewClientSimple(conn *websocket.Conn, auctionID, userID, liveStreamID int64, userName string, authenticated bool) *Client {
	id := fmt.Sprintf("%d-%d-%d", auctionID, userID, time.Now().UnixNano())
	return &Client{
		ID:            id,
		AuctionID:     auctionID,
		LiveStreamID:  liveStreamID,
		UserID:        userID,
		UserName:      userName,
		Authenticated: authenticated,
		conn:          conn,
		Send:          make(chan *Message, sendBufferSize),
	}
}

// ReadPump 读取消息循环
func (c *Client) ReadPump() {
	defer func() {
		// 删除连接状态
		if c.stateManager != nil {
			ctx := context.Background()
			if err := c.stateManager.DeleteConnectionState(ctx, c.ID); err != nil {
				log.Printf("Failed to delete connection state: %v", err)
			}
		}
		c.hub.Unregister <- c
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		// 解析消息
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		// 处理消息
		c.handleMessage(&msg)
	}
}

// WritePump 发送消息循环
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 填充时间戳
			message.Timestamp = time.Now().UnixMilli()

			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("Failed to write message: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 处理客户端消息
func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case MessageTypePing:
		// 响应 pong
		c.Send <- NewPongMessage()

	case MessageTypeSyncRequest:
		// 处理状态同步请求（重连后）
		c.handleSyncRequest(msg)

	case MessageTypeChatSend:
		c.handleChatSend(msg)

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// handleChatSend 解析 ChatSendData 并交给 ChatHandler
func (c *Client) handleChatSend(msg *Message) {
	if c.chatHandler == nil {
		return
	}

	raw, err := json.Marshal(msg.Data)
	if err != nil {
		c.Send <- NewErrorMessage(ChatErrCodeLengthExceeded, "invalid chat payload")
		return
	}
	var data ChatSendData
	if err := json.Unmarshal(raw, &data); err != nil {
		c.Send <- NewErrorMessage(ChatErrCodeLengthExceeded, "invalid chat payload")
		return
	}
	c.chatHandler.Handle(context.Background(), c, &data)
}

// handleSyncRequest 处理状态同步请求
func (c *Client) handleSyncRequest(msg *Message) {
	if c.stateManager == nil {
		return
	}

	ctx := context.Background()

	// 获取同步状态
	syncState, err := c.stateManager.GetSyncState(ctx, c.AuctionID)
	if err != nil {
		log.Printf("Failed to get sync state: %v", err)
		// 返回错误消息
		c.Send <- NewErrorMessage(500, "Failed to sync state")
		return
	}

	// 发送同步响应
	response := &SyncResponseData{
		AuctionID:    syncState.AuctionID,
		CurrentPrice: syncState.CurrentPrice,
		WinnerID:     syncState.WinnerID,
		EndTime:      syncState.EndTime.UnixMilli(),
		Status:       syncState.Status,
	}

	c.Send <- NewSyncResponseMessage(response)
}

// Close 关闭客户端连接
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.closed = true
		close(c.Send)
		c.conn.Close()
	})
}

// IsClosed 检查是否已关闭
func (c *Client) IsClosed() bool {
	return c.closed
}

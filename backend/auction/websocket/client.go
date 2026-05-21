package websocket

import (
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
	ID        string
	AuctionID int64
	UserID    int64

	conn *websocket.Conn
	Send chan *Message

	hub *Hub

	closeOnce sync.Once
	closed    bool
}

// NewClient 创建客户端
func NewClient(id string, auctionID, userID int64, conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:        id,
		AuctionID: auctionID,
		UserID:    userID,
		conn:      conn,
		Send:      make(chan *Message, sendBufferSize),
		hub:       hub,
	}
}

// NewClientSimple 创建客户端（简化版，自动生成ID）
func NewClientSimple(conn *websocket.Conn, auctionID, userID int64) *Client {
	id := fmt.Sprintf("%d-%d-%d", auctionID, userID, time.Now().UnixNano())
	return &Client{
		ID:        id,
		AuctionID: auctionID,
		UserID:    userID,
		conn:      conn,
		Send:      make(chan *Message, sendBufferSize),
	}
}

// ReadPump 读取消息循环
func (c *Client) ReadPump() {
	defer func() {
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

	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
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

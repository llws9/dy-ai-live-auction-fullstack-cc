package websocket

import (
	"log"
	"sync"
)

// Hub WebSocket Hub，管理所有房间和连接
type Hub struct {
	rooms     map[int64]*Room
	roomsLock sync.RWMutex

	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage

	done chan struct{}
}

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	AuctionID int64
	Message   *Message
}

// NewHub 创建 Hub
func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[int64]*Room),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		broadcast:  make(chan *BroadcastMessage, 1024),
		done:       make(chan struct{}),
	}
}

// Run 运行 Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case msg := <-h.broadcast:
			h.broadcastMessage(msg)

		case <-h.done:
			return
		}
	}
}

// Stop 停止 Hub
func (h *Hub) Stop() {
	close(h.done)

	h.roomsLock.Lock()
	for _, room := range h.rooms {
		room.Close()
	}
	h.roomsLock.Unlock()
}

// registerClient 注册客户端
func (h *Hub) registerClient(client *Client) {
	h.roomsLock.Lock()
	defer h.roomsLock.Unlock()

	room, exists := h.rooms[client.AuctionID]
	if !exists {
		room = NewRoom(client.AuctionID)
		h.rooms[client.AuctionID] = room
		go room.Run()
	}

	room.Register <- client
	log.Printf("Client registered: auction_id=%d, client_id=%s", client.AuctionID, client.ID)
}

// unregisterClient 注销客户端
func (h *Hub) unregisterClient(client *Client) {
	h.roomsLock.RLock()
	room, exists := h.rooms[client.AuctionID]
	h.roomsLock.RUnlock()

	if exists {
		room.Unregister <- client
		log.Printf("Client unregistered: auction_id=%d, client_id=%s", client.AuctionID, client.ID)
	}
}

// broadcastMessage 广播消息
func (h *Hub) broadcastMessage(msg *BroadcastMessage) {
	h.roomsLock.RLock()
	room, exists := h.rooms[msg.AuctionID]
	h.roomsLock.RUnlock()

	if exists {
		room.Broadcast <- msg.Message
	}
}

// BroadcastToRoom 向指定房间广播消息
func (h *Hub) BroadcastToRoom(auctionID int64, message *Message) {
	h.broadcast <- &BroadcastMessage{
		AuctionID: auctionID,
		Message:   message,
	}
}

// GetRoomCount 获取房间数量
func (h *Hub) GetRoomCount() int {
	h.roomsLock.RLock()
	defer h.roomsLock.RUnlock()
	return len(h.rooms)
}

// GetClientCount 获取客户端总数
func (h *Hub) GetClientCount() int {
	h.roomsLock.RLock()
	defer h.roomsLock.RUnlock()

	count := 0
	for _, room := range h.rooms {
		count += room.GetClientCount()
	}
	return count
}

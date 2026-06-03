package websocket

import (
	"log"
	"sync"
)

// Room WebSocket 房间，管理单个竞拍的所有连接
type Room struct {
	AuctionID int64

	clients     map[string]*Client
	clientsLock sync.RWMutex

	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *Message

	done chan struct{}
}

// NewRoom 创建房间
func NewRoom(auctionID int64) *Room {
	return &Room{
		AuctionID:  auctionID,
		clients:    make(map[string]*Client),
		Register:   make(chan *Client, 64),
		Unregister: make(chan *Client, 64),
		Broadcast:  make(chan *Message, 256),
		done:       make(chan struct{}),
	}
}

// Run 运行房间
func (r *Room) Run() {
	for {
		select {
		case client := <-r.Register:
			r.registerClient(client)

		case client := <-r.Unregister:
			r.unregisterClient(client)

		case message := <-r.Broadcast:
			r.broadcastMessage(message)

		case <-r.done:
			return
		}
	}
}

// Close 关闭房间
func (r *Room) Close() {
	close(r.done)

	r.clientsLock.Lock()
	for _, client := range r.clients {
		client.Close()
	}
	r.clients = make(map[string]*Client)
	r.clientsLock.Unlock()
}

// registerClient 注册客户端
func (r *Room) registerClient(client *Client) {
	r.clientsLock.Lock()
	r.clients[client.ID] = client
	r.clientsLock.Unlock()
}

// unregisterClient 注销客户端
func (r *Room) unregisterClient(client *Client) {
	r.clientsLock.Lock()
	delete(r.clients, client.ID)
	r.clientsLock.Unlock()

	client.Close()
}

// broadcastMessage 广播消息到房间内所有客户端
func (r *Room) broadcastMessage(message *Message) {
	r.clientsLock.RLock()
	defer r.clientsLock.RUnlock()

	for _, client := range r.clients {
		if client.IsClosed() {
			continue
		}
		select {
		case client.Send <- message:
		default:
			log.Printf("Client send buffer full, skipping: client_id=%s", client.ID)
		}
	}
}

// GetClientCount 获取客户端数量
func (r *Room) GetClientCount() int {
	r.clientsLock.RLock()
	defer r.clientsLock.RUnlock()
	return len(r.clients)
}

// GetClientIDs 获取所有客户端 ID
func (r *Room) GetClientIDs() []string {
	r.clientsLock.RLock()
	defer r.clientsLock.RUnlock()

	ids := make([]string, 0, len(r.clients))
	for id := range r.clients {
		ids = append(ids, id)
	}
	return ids
}

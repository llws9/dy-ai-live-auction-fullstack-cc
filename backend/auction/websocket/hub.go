package websocket

import (
	"log"
	"sync"
)

// Hub WebSocket Hub，管理所有房间和连接
type Hub struct {
	rooms     map[int64]*Room
	roomsLock sync.RWMutex

	// 直播间房间管理 - 按 live_stream_id 隔离的弹幕房间
	liveStreamRooms     map[int64]*LiveStreamRoom
	liveStreamRoomsLock sync.RWMutex

	// 用户房间管理 - 支持按用户ID推送通知
	UserRooms   map[int64]map[*Client]bool // userID -> clients
	userRoomsMu sync.RWMutex

	Register   chan *Client
	Unregister chan *Client
	broadcast  chan *BroadcastMessage

	stateManager *StateManager // 状态管理器

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
		rooms:           make(map[int64]*Room),
		liveStreamRooms: make(map[int64]*LiveStreamRoom),
		UserRooms:       make(map[int64]map[*Client]bool),
		Register:        make(chan *Client, 256),
		Unregister:      make(chan *Client, 256),
		broadcast:       make(chan *BroadcastMessage, 1024),
		done:            make(chan struct{}),
	}
}

// SetStateManager 设置状态管理器
func (h *Hub) SetStateManager(sm *StateManager) {
	h.stateManager = sm
}

// GetStateManager 获取状态管理器
func (h *Hub) GetStateManager() *StateManager {
	return h.stateManager
}

// Run 运行 Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
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

	h.liveStreamRoomsLock.Lock()
	for _, r := range h.liveStreamRooms {
		r.Close()
	}
	h.liveStreamRoomsLock.Unlock()
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

	// 用户连接时，如果有userID，自动加入用户房间
	if client.UserID > 0 {
		h.JoinUserRoom(client.UserID, client)
	}

	// 订阅了直播间弹幕时，自动加入直播间房间
	if client.LiveStreamID > 0 {
		h.RegisterToLiveStream(client)
	}
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

	// 用户断开时，从用户房间移除
	if client.UserID > 0 {
		h.LeaveUserRoom(client.UserID, client)
	}

	// 订阅了直播间弹幕时，从直播间房间移除
	if client.LiveStreamID > 0 {
		h.UnregisterFromLiveStream(client)
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

// TryBroadcastToRoom 非阻塞广播；队列满时返回 false，由调用方决定是否丢弃。
func (h *Hub) TryBroadcastToRoom(auctionID int64, message *Message) bool {
	select {
	case h.broadcast <- &BroadcastMessage{AuctionID: auctionID, Message: message}:
		return true
	default:
		return false
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

// JoinUserRoom - 用户加入个人房间
func (h *Hub) JoinUserRoom(userID int64, client *Client) {
	h.userRoomsMu.Lock()
	defer h.userRoomsMu.Unlock()

	if h.UserRooms[userID] == nil {
		h.UserRooms[userID] = make(map[*Client]bool)
	}
	h.UserRooms[userID][client] = true
	log.Printf("Client joined user room: user_id=%d, client_id=%s", userID, client.ID)
}

// LeaveUserRoom - 用户离开个人房间
func (h *Hub) LeaveUserRoom(userID int64, client *Client) {
	h.userRoomsMu.Lock()
	defer h.userRoomsMu.Unlock()

	if h.UserRooms[userID] != nil {
		delete(h.UserRooms[userID], client)
		if len(h.UserRooms[userID]) == 0 {
			delete(h.UserRooms, userID)
		}
		log.Printf("Client left user room: user_id=%d, client_id=%s", userID, client.ID)
	}
}

// BroadcastToUserRoom - 向用户房间广播消息（使用读锁优先，仅在需要修改时升级为写锁）
func (h *Hub) BroadcastToUserRoom(userID int64, message *Message) {
	// 先用读锁尝试发送
	h.userRoomsMu.RLock()
	clients := h.UserRooms[userID]
	if clients == nil {
		h.userRoomsMu.RUnlock()
		return
	}

	// 尝试发送到所有客户端，记录发送失败的客户端
	var blockedClients []*Client
	for client := range clients {
		if client.IsClosed() {
			blockedClients = append(blockedClients, client)
			continue
		}
		select {
		case client.Send <- message:
			// 发送成功
		default:
			// 发送缓冲区满，记录需要移除的客户端
			blockedClients = append(blockedClients, client)
		}
	}
	h.userRoomsMu.RUnlock()

	// 如果没有阻塞的客户端，直接返回
	if len(blockedClients) == 0 {
		return
	}

	// 需要移除阻塞的客户端，获取写锁
	h.userRoomsMu.Lock()
	for _, client := range blockedClients {
		if h.UserRooms[userID] != nil {
			client.Close()
			delete(h.UserRooms[userID], client)
		}
	}
	h.userRoomsMu.Unlock()
}

// GetUserRoomClientCount 获取用户房间客户端数量
func (h *Hub) GetUserRoomClientCount(userID int64) int {
	h.userRoomsMu.RLock()
	defer h.userRoomsMu.RUnlock()

	if h.UserRooms[userID] == nil {
		return 0
	}
	return len(h.UserRooms[userID])
}

// RegisterToLiveStream 将客户端加入直播间弹幕房间（房间不存在则创建）
func (h *Hub) RegisterToLiveStream(client *Client) {
	if client.LiveStreamID <= 0 {
		return
	}
	h.liveStreamRoomsLock.Lock()
	room, ok := h.liveStreamRooms[client.LiveStreamID]
	if !ok {
		room = NewLiveStreamRoom(client.LiveStreamID)
		h.liveStreamRooms[client.LiveStreamID] = room
		go room.Run()
	}
	h.liveStreamRoomsLock.Unlock()
	room.registerClient(client)
}

// UnregisterFromLiveStream 将客户端移出直播间弹幕房间
func (h *Hub) UnregisterFromLiveStream(client *Client) {
	if client.LiveStreamID <= 0 {
		return
	}
	h.liveStreamRoomsLock.Lock()
	defer h.liveStreamRoomsLock.Unlock()
	room, ok := h.liveStreamRooms[client.LiveStreamID]
	if ok {
		room.unregisterClient(client)
		if room.GetClientCount() == 0 {
			room.Close()
			delete(h.liveStreamRooms, client.LiveStreamID)
		}
	}
}

// BroadcastToLiveStream 向指定直播间弹幕房间广播消息
func (h *Hub) BroadcastToLiveStream(liveStreamID int64, msg *Message) {
	h.liveStreamRoomsLock.RLock()
	room, ok := h.liveStreamRooms[liveStreamID]
	h.liveStreamRoomsLock.RUnlock()
	if !ok {
		return
	}
	select {
	case room.Broadcast <- msg:
	default:
		log.Printf("[hub] livestream room %d broadcast buffer full", liveStreamID)
	}
}

// GetLiveStreamRoom 获取直播间弹幕房间（不存在返回 nil）
func (h *Hub) GetLiveStreamRoom(liveStreamID int64) *LiveStreamRoom {
	h.liveStreamRoomsLock.RLock()
	defer h.liveStreamRoomsLock.RUnlock()
	return h.liveStreamRooms[liveStreamID]
}

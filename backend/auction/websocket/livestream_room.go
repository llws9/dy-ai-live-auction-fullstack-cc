package websocket

import (
	"log"
	"sync"
)

const chatHistorySize = 100

// LiveStreamRoom 直播间级 WebSocket 房间
type LiveStreamRoom struct {
	LiveStreamID int64

	clients     map[string]*Client
	clientsLock sync.RWMutex

	history     [chatHistorySize]*Message
	historyHead int
	historyLen  int
	historyLock sync.RWMutex

	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *Message

	done chan struct{}
}

// NewLiveStreamRoom 创建直播间房间
func NewLiveStreamRoom(liveStreamID int64) *LiveStreamRoom {
	return &LiveStreamRoom{
		LiveStreamID: liveStreamID,
		clients:      make(map[string]*Client),
		Register:     make(chan *Client, 64),
		Unregister:   make(chan *Client, 64),
		Broadcast:    make(chan *Message, 256),
		done:         make(chan struct{}),
	}
}

// Run 运行房间事件循环
func (r *LiveStreamRoom) Run() {
	for {
		select {
		case c := <-r.Register:
			r.registerClient(c)
		case c := <-r.Unregister:
			r.unregisterClient(c)
		case msg := <-r.Broadcast:
			r.pushHistory(msg)
			r.broadcastMessage(msg)
		case <-r.done:
			return
		}
	}
}

// Close 关闭房间
func (r *LiveStreamRoom) Close() {
	close(r.done)
	r.clientsLock.Lock()
	r.clients = make(map[string]*Client)
	r.clientsLock.Unlock()
}

func (r *LiveStreamRoom) registerClient(c *Client) {
	r.clientsLock.Lock()
	r.clients[c.ID] = c
	r.clientsLock.Unlock()

	// 立即回放历史
	r.historyLock.RLock()
	for i := 0; i < r.historyLen; i++ {
		idx := (r.historyHead - r.historyLen + i + chatHistorySize) % chatHistorySize
		m := r.history[idx]
		if m == nil {
			continue
		}
		select {
		case c.Send <- m:
		default:
			log.Printf("[livestream_room] client %s buffer full during replay", c.ID)
		}
	}
	r.historyLock.RUnlock()
}

func (r *LiveStreamRoom) unregisterClient(c *Client) {
	r.clientsLock.Lock()
	delete(r.clients, c.ID)
	r.clientsLock.Unlock()
}

func (r *LiveStreamRoom) pushHistory(msg *Message) {
	r.historyLock.Lock()
	defer r.historyLock.Unlock()
	r.history[r.historyHead] = msg
	r.historyHead = (r.historyHead + 1) % chatHistorySize
	if r.historyLen < chatHistorySize {
		r.historyLen++
	}
}

func (r *LiveStreamRoom) broadcastMessage(msg *Message) {
	r.clientsLock.RLock()
	defer r.clientsLock.RUnlock()
	for _, c := range r.clients {
		select {
		case c.Send <- msg:
		default:
			log.Printf("[livestream_room] client %s buffer full, dropping msg", c.ID)
		}
	}
}

// GetClientCount 客户端数量
func (r *LiveStreamRoom) GetClientCount() int {
	r.clientsLock.RLock()
	defer r.clientsLock.RUnlock()
	return len(r.clients)
}

// GetHistory 测试用：返回历史快照
func (r *LiveStreamRoom) GetHistory() []*Message {
	r.historyLock.RLock()
	defer r.historyLock.RUnlock()
	out := make([]*Message, 0, r.historyLen)
	for i := 0; i < r.historyLen; i++ {
		idx := (r.historyHead - r.historyLen + i + chatHistorySize) % chatHistorySize
		out = append(out, r.history[idx])
	}
	return out
}

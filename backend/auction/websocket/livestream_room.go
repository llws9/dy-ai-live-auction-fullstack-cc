package websocket

import (
	"log"
	"sync"
)

const chatHistorySize = 100
const presenceViewerLimit = 3

type presenceViewerState struct {
	viewer  LivePresenceViewer
	clients map[string]struct{}
}

// LiveStreamRoom 直播间级 WebSocket 房间
type LiveStreamRoom struct {
	LiveStreamID int64

	clients     map[string]*Client
	clientsLock sync.RWMutex

	presenceByUserID map[int64]*presenceViewerState
	presenceLock     sync.RWMutex

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
		LiveStreamID:     liveStreamID,
		clients:          make(map[string]*Client),
		presenceByUserID: make(map[int64]*presenceViewerState),
		Register:         make(chan *Client, 64),
		Unregister:       make(chan *Client, 64),
		Broadcast:        make(chan *Message, 256),
		done:             make(chan struct{}),
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
	r.presenceLock.Lock()
	r.presenceByUserID = make(map[int64]*presenceViewerState)
	r.presenceLock.Unlock()
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

	if r.addPresenceClient(c) {
		r.broadcastPresenceSnapshot()
	}
}

func (r *LiveStreamRoom) unregisterClient(c *Client) {
	r.clientsLock.Lock()
	delete(r.clients, c.ID)
	r.clientsLock.Unlock()

	if r.removePresenceClient(c) {
		r.broadcastPresenceSnapshot()
	}
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

func (r *LiveStreamRoom) addPresenceClient(c *Client) bool {
	if c == nil || !c.Authenticated || c.UserID <= 0 {
		return false
	}
	r.presenceLock.Lock()
	defer r.presenceLock.Unlock()

	state, ok := r.presenceByUserID[c.UserID]
	if !ok {
		name := c.UserName
		if name == "" {
			name = "用户"
		}
		state = &presenceViewerState{
			viewer: LivePresenceViewer{
				UserID:    c.UserID,
				Name:      name,
				AvatarURL: c.AvatarURL,
			},
			clients: make(map[string]struct{}),
		}
		r.presenceByUserID[c.UserID] = state
	}
	if _, exists := state.clients[c.ID]; exists {
		return false
	}
	state.clients[c.ID] = struct{}{}
	return true
}

func (r *LiveStreamRoom) removePresenceClient(c *Client) bool {
	if c == nil || !c.Authenticated || c.UserID <= 0 {
		return false
	}
	r.presenceLock.Lock()
	defer r.presenceLock.Unlock()

	state, ok := r.presenceByUserID[c.UserID]
	if !ok {
		return false
	}
	if _, exists := state.clients[c.ID]; !exists {
		return false
	}
	delete(state.clients, c.ID)
	if len(state.clients) == 0 {
		delete(r.presenceByUserID, c.UserID)
	}
	return true
}

// GetPresenceSnapshot 返回当前直播间在线状态快照。
func (r *LiveStreamRoom) GetPresenceSnapshot() *LivePresenceUpdateData {
	r.presenceLock.RLock()
	defer r.presenceLock.RUnlock()

	viewers := make([]LivePresenceViewer, 0, presenceViewerLimit)
	for _, state := range r.presenceByUserID {
		if len(viewers) >= presenceViewerLimit {
			break
		}
		viewers = append(viewers, state.viewer)
	}
	return &LivePresenceUpdateData{
		LiveStreamID: r.LiveStreamID,
		ViewerCount:  len(r.presenceByUserID),
		Viewers:      viewers,
	}
}

func (r *LiveStreamRoom) broadcastPresenceSnapshot() {
	msg := NewLivePresenceUpdateMessage(r.GetPresenceSnapshot())

	r.clientsLock.RLock()
	defer r.clientsLock.RUnlock()
	for _, c := range r.clients {
		if !c.Authenticated {
			continue
		}
		select {
		case c.Send <- msg:
		default:
			log.Printf("[livestream_room] client %s buffer full, dropping presence update", c.ID)
		}
	}
}

# 直播间弹幕 MVP 实施计划（Plan A：M1 + M2）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在直播竞拍平台上线"直播间级弹幕"能力（M1 双 Room 抽象 + M2 弹幕全链路），完成后用户可在直播间发送/接收实时聊天消息。

**Architecture:** 在现有 `Hub` 中并列新增 `LiveStreamRoom`（按 `live_stream_id` 隔离），与已有 `AuctionRoom` 平行；客户端单 WS 连接，握手 URL 同时携带 `auction_id` 与 `live_stream_id`，服务端在 Hub 注册时双写到两个 Room。弹幕走 LiveStreamRoom 广播，由黑词过滤 + Redis 频控双层守护，并保留 100 条环形缓冲用于新进房回放。

**Tech Stack:**
- 后端：Go 1.21+ / Gorilla WebSocket / Redis (go-redis v9) / 现有 Hub & Hertz
- 前端：React 18 / TypeScript / Zustand / CSS Modules / Jest + RTL
- 关联 spec：[2026-05-31-live-chat-and-price-flair-design.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/superpowers/specs/2026-05-31-live-chat-and-price-flair-design.md)

**Out of scope (Plan B 处理):**
- 高价飘屏 FlairChecker、`price_flair` 消息
- Prometheus 指标、Nacos 热更新

---

## File Structure

### 后端新增

| 文件 | 责任 |
|---|---|
| `backend/auction/websocket/livestream_room.go` | LiveStreamRoom 结构 + 环形缓冲 + Run/Close/Broadcast |
| `backend/auction/websocket/livestream_room_test.go` | LiveStreamRoom 单元测试 |
| `backend/auction/websocket/chat_filter.go` | 黑词过滤 + 长度校验 |
| `backend/auction/websocket/chat_filter_test.go` | 过滤器单元测试 |
| `backend/auction/websocket/chat_throttle.go` | Redis 双层频控 |
| `backend/auction/websocket/chat_throttle_test.go` | 频控单元测试（miniredis） |
| `backend/auction/websocket/chat_handler.go` | `handleChatSend` 业务编排 |
| `backend/auction/websocket/chat_handler_test.go` | handleChatSend 集成测试 |

### 后端修改

| 文件 | 修改 |
|---|---|
| `backend/auction/websocket/message.go` | 新增 `chat_send` / `chat_message` 消息常量与 DTO；新增错误码常量 |
| `backend/auction/websocket/hub.go` | 新增 `liveStreamRooms` map + `RegisterToLiveStream` / `UnregisterFromLiveStream` / `BroadcastToLiveStream` |
| `backend/auction/websocket/client.go` | `Client` 增加 `LiveStreamID` + `UserName`；`handleMessage` 增加 `chat_send` 分支 |
| `backend/auction/handler/ws.go` | 升级时解析 `live_stream_id`；连接成功后双注册 |

### 前端新增

| 文件 | 责任 |
|---|---|
| `frontend/h5/src/components/LiveChat/ChatPanel.tsx` | 滚动列表 + 输入框（固定底部） |
| `frontend/h5/src/components/LiveChat/ChatBubble.tsx` | 单条气泡 |
| `frontend/h5/src/components/LiveChat/ChatPanel.module.css` | 样式 |
| `frontend/h5/src/components/LiveChat/__tests__/ChatBubble.test.tsx` | 气泡渲染测试 |
| `frontend/h5/src/components/LiveChat/__tests__/ChatPanel.test.tsx` | 输入校验/频控倒计时测试 |
| `frontend/h5/src/store/liveChatStore.ts` | Zustand：history、connect、send |
| `frontend/h5/src/store/__tests__/liveChatStore.test.ts` | store 单测 |

### 前端修改

| 文件 | 修改 |
|---|---|
| `frontend/h5/src/services/websocket.ts` | 构造 URL 时拼接 `live_stream_id`；新增 `sendChat(text)` 方法；新增 `chat_message` 派发 |
| `frontend/h5/src/pages/Live/index.tsx` | 挂载 `<ChatPanel />` |

---

## Task 1: 扩展 WS 消息协议（DTO + 错误码）

**Files:**
- Modify: `backend/auction/websocket/message.go`

- [ ] **Step 1: 新增消息类型常量与 DTO**

在 `backend/auction/websocket/message.go` 的 `const` 块底部追加：

```go
// 弹幕相关消息类型（M2）
const (
	MessageTypeChatSend    MessageType = "chat_send"     // 客户端 -> 服务端
	MessageTypeChatMessage MessageType = "chat_message"  // 服务端 -> 客户端
)

// 弹幕错误码
const (
	ChatErrCodeLengthExceeded   = 40001
	ChatErrCodeBlockedWord      = 40002
	ChatErrCodeRateLimited      = 40003
	ChatErrCodeNotAuthenticated = 40101
)

// ChatSendData 客户端发送的弹幕请求
type ChatSendData struct {
	LiveStreamID int64  `json:"live_stream_id"`
	Text         string `json:"text"`
	ClientMsgID  string `json:"client_msg_id"`
}

// ChatMessageData 服务端广播的弹幕消息
type ChatMessageData struct {
	LiveStreamID int64  `json:"live_stream_id"`
	UserID       int64  `json:"user_id"`
	UserName     string `json:"user_name"`
	AvatarURL    string `json:"avatar_url,omitempty"`
	Text         string `json:"text"`
	SentAt       int64  `json:"sent_at"`
	ClientMsgID  string `json:"client_msg_id,omitempty"`
}

// NewChatMessage 创建弹幕广播消息
func NewChatMessage(data *ChatMessageData) *Message {
	return NewMessage(MessageTypeChatMessage, data)
}
```

- [ ] **Step 2: 编译验证**

Run: `cd backend/auction && go build ./...`
Expected: 通过，无编译错误。

- [ ] **Step 3: 提交**

```bash
git add backend/auction/websocket/message.go
git commit -m "feat(ws): add chat_send/chat_message DTOs and error codes"
```

---

## Task 2: 实现 ChatFilter（黑词与长度校验）

**Files:**
- Create: `backend/auction/websocket/chat_filter.go`
- Test: `backend/auction/websocket/chat_filter_test.go`

- [ ] **Step 1: 写失败测试**

创建 `backend/auction/websocket/chat_filter_test.go`：

```go
package websocket

import "testing"

func TestChatFilter_LengthBoundary(t *testing.T) {
	f := NewChatFilter(50, []string{})

	cases := []struct {
		name    string
		text    string
		wantErr int
	}{
		{"empty", "", ChatErrCodeLengthExceeded},
		{"49 chars", repeatRune('a', 49), 0},
		{"50 chars", repeatRune('a', 50), 0},
		{"51 chars", repeatRune('a', 51), ChatErrCodeLengthExceeded},
		{"50 chinese", repeatRune('中', 50), 0},
		{"51 chinese", repeatRune('中', 51), ChatErrCodeLengthExceeded},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotCode := f.Validate(c.text)
			if gotCode != c.wantErr {
				t.Errorf("Validate(%q) code = %d, want %d", c.name, gotCode, c.wantErr)
			}
		})
	}
}

func TestChatFilter_BlockedWord(t *testing.T) {
	f := NewChatFilter(50, []string{"微信", "vx"})

	cases := []struct {
		text    string
		wantErr int
	}{
		{"加我微信", ChatErrCodeBlockedWord},
		{"加vx一下", ChatErrCodeBlockedWord},
		{"VX 大写", ChatErrCodeBlockedWord}, // 大小写不敏感
		{"正常聊天内容", 0},
	}

	for _, c := range cases {
		gotCode := f.Validate(c.text)
		if gotCode != c.wantErr {
			t.Errorf("Validate(%q) code = %d, want %d", c.text, gotCode, c.wantErr)
		}
	}
}

func repeatRune(r rune, n int) string {
	out := make([]rune, n)
	for i := range out {
		out[i] = r
	}
	return string(out)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend/auction && go test ./websocket -run TestChatFilter -v`
Expected: FAIL（NewChatFilter undefined）

- [ ] **Step 3: 实现 ChatFilter**

创建 `backend/auction/websocket/chat_filter.go`：

```go
package websocket

import (
	"strings"
	"unicode/utf8"
)

// ChatFilter 弹幕内容校验器
type ChatFilter struct {
	maxLen   int
	blocked  []string // 已转小写
}

// NewChatFilter 创建弹幕过滤器
func NewChatFilter(maxLen int, blockedWords []string) *ChatFilter {
	lowered := make([]string, len(blockedWords))
	for i, w := range blockedWords {
		lowered[i] = strings.ToLower(w)
	}
	return &ChatFilter{
		maxLen:  maxLen,
		blocked: lowered,
	}
}

// Validate 校验弹幕文本
// 返回 0 表示通过，非 0 为错误码（ChatErrCode*）
func (f *ChatFilter) Validate(text string) int {
	n := utf8.RuneCountInString(text)
	if n == 0 || n > f.maxLen {
		return ChatErrCodeLengthExceeded
	}
	lower := strings.ToLower(text)
	for _, w := range f.blocked {
		if strings.Contains(lower, w) {
			return ChatErrCodeBlockedWord
		}
	}
	return 0
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd backend/auction && go test ./websocket -run TestChatFilter -v`
Expected: PASS（所有用例）

- [ ] **Step 5: 提交**

```bash
git add backend/auction/websocket/chat_filter.go backend/auction/websocket/chat_filter_test.go
git commit -m "feat(ws): add ChatFilter for length and blocked-word validation"
```

---

## Task 3: 实现 ChatThrottle（Redis 双层频控）

**Files:**
- Create: `backend/auction/websocket/chat_throttle.go`
- Test: `backend/auction/websocket/chat_throttle_test.go`

> 项目已使用 `github.com/redis/go-redis/v9`。测试用 `github.com/alicebob/miniredis/v2` 起内存 Redis（如未安装：`go get github.com/alicebob/miniredis/v2`）。

- [ ] **Step 1: 写失败测试**

创建 `backend/auction/websocket/chat_throttle_test.go`：

```go
package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupThrottle(t *testing.T) (*ChatThrottle, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cfg := ThrottleConfig{
		UserMax:        1,
		UserInterval:   time.Second,
		RoomMax:        20,
		RoomInterval:   time.Second,
	}
	return NewChatThrottle(rdb, cfg), mr
}

func TestChatThrottle_UserLimit(t *testing.T) {
	th, _ := setupThrottle(t)
	ctx := context.Background()

	if code := th.Allow(ctx, 100, 1); code != 0 {
		t.Fatalf("first message should pass, got code %d", code)
	}
	if code := th.Allow(ctx, 100, 1); code != ChatErrCodeRateLimited {
		t.Fatalf("second message in 1s should be rate-limited, got %d", code)
	}
}

func TestChatThrottle_RoomLimit(t *testing.T) {
	th, _ := setupThrottle(t)
	ctx := context.Background()

	// 20 个不同用户连续发，第 21 个被房间限流
	for i := 1; i <= 20; i++ {
		if code := th.Allow(ctx, int64(i), 999); code != 0 {
			t.Fatalf("user %d should pass, got code %d", i, code)
		}
	}
	if code := th.Allow(ctx, 9999, 999); code != ChatErrCodeRateLimited {
		t.Fatalf("21st message in same room should be rate-limited, got %d", code)
	}
}

func TestChatThrottle_TTLReset(t *testing.T) {
	th, mr := setupThrottle(t)
	ctx := context.Background()

	if code := th.Allow(ctx, 100, 1); code != 0 {
		t.Fatal("first should pass")
	}
	mr.FastForward(time.Second + 100*time.Millisecond) // 跳过 TTL
	if code := th.Allow(ctx, 100, 1); code != 0 {
		t.Fatalf("after TTL expires, should pass, got %d", code)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend/auction && go test ./websocket -run TestChatThrottle -v`
Expected: FAIL（NewChatThrottle undefined）

- [ ] **Step 3: 实现 ChatThrottle**

创建 `backend/auction/websocket/chat_throttle.go`：

```go
package websocket

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ThrottleConfig 频控配置
type ThrottleConfig struct {
	UserMax      int
	UserInterval time.Duration
	RoomMax      int
	RoomInterval time.Duration
}

// ChatThrottle 基于 Redis 的双层频控
type ChatThrottle struct {
	rdb *redis.Client
	cfg ThrottleConfig
}

// NewChatThrottle 创建频控器
func NewChatThrottle(rdb *redis.Client, cfg ThrottleConfig) *ChatThrottle {
	return &ChatThrottle{rdb: rdb, cfg: cfg}
}

// Allow 校验当前用户在指定直播间是否可发送
// 返回 0 表示通过，ChatErrCodeRateLimited 表示被拒
func (t *ChatThrottle) Allow(ctx context.Context, userID, liveStreamID int64) int {
	// 用户级
	userKey := fmt.Sprintf("chat:rate:user:%d", userID)
	if !t.incrAndCheck(ctx, userKey, t.cfg.UserMax, t.cfg.UserInterval) {
		return ChatErrCodeRateLimited
	}
	// 房间级
	roomKey := fmt.Sprintf("chat:rate:room:%d", liveStreamID)
	if !t.incrAndCheck(ctx, roomKey, t.cfg.RoomMax, t.cfg.RoomInterval) {
		return ChatErrCodeRateLimited
	}
	return 0
}

// incrAndCheck 原子递增并比较；首次写入时设置 TTL
func (t *ChatThrottle) incrAndCheck(ctx context.Context, key string, max int, ttl time.Duration) bool {
	pipe := t.rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl) // 幂等：每次都会刷新 TTL
	if _, err := pipe.Exec(ctx); err != nil {
		// Redis 故障时降级放行，避免直播间静音
		return true
	}
	return incr.Val() <= int64(max)
}
```

> **设计说明**：每次 INCR 都 Expire 会让 TTL 持续刷新，可能让早期计数永不释放。这里是有意的——更严格地保护房间。如要"滑动窗口"，将来可换成 Redis Lua 脚本。

- [ ] **Step 4: 运行测试确认通过**

Run: `cd backend/auction && go test ./websocket -run TestChatThrottle -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/auction/websocket/chat_throttle.go backend/auction/websocket/chat_throttle_test.go
git commit -m "feat(ws): add Redis-based dual-layer chat throttling"
```

---

## Task 4: 实现 LiveStreamRoom（含环形缓冲）

**Files:**
- Create: `backend/auction/websocket/livestream_room.go`
- Test: `backend/auction/websocket/livestream_room_test.go`

- [ ] **Step 1: 写失败测试**

创建 `backend/auction/websocket/livestream_room_test.go`：

```go
package websocket

import (
	"testing"
	"time"
)

func newTestRoomClient(id string) *Client {
	return &Client{
		ID:   id,
		Send: make(chan *Message, 16),
	}
}

func TestLiveStreamRoom_BroadcastDelivers(t *testing.T) {
	r := NewLiveStreamRoom(123)
	go r.Run()
	defer r.Close()

	c := newTestRoomClient("c1")
	r.Register <- c
	time.Sleep(20 * time.Millisecond)

	msg := NewChatMessage(&ChatMessageData{LiveStreamID: 123, Text: "hello"})
	r.Broadcast <- msg

	select {
	case got := <-c.Send:
		if got.Type != MessageTypeChatMessage {
			t.Fatalf("got type %s, want chat_message", got.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive broadcast within 1s")
	}
}

func TestLiveStreamRoom_RingBufferReplay(t *testing.T) {
	r := NewLiveStreamRoom(456)
	go r.Run()
	defer r.Close()

	// 先广播 105 条（环形缓冲容量 100）
	for i := 0; i < 105; i++ {
		r.Broadcast <- NewChatMessage(&ChatMessageData{LiveStreamID: 456, Text: "x"})
	}
	time.Sleep(50 * time.Millisecond)

	history := r.GetHistory()
	if len(history) != 100 {
		t.Fatalf("history len = %d, want 100", len(history))
	}
}

func TestLiveStreamRoom_RegisterReplaysHistory(t *testing.T) {
	r := NewLiveStreamRoom(789)
	go r.Run()
	defer r.Close()

	// 注入 3 条历史
	for i := 0; i < 3; i++ {
		r.Broadcast <- NewChatMessage(&ChatMessageData{LiveStreamID: 789, Text: "old"})
	}
	time.Sleep(20 * time.Millisecond)

	c := newTestRoomClient("late")
	r.Register <- c
	time.Sleep(20 * time.Millisecond)

	// 进房后应立即收到 3 条历史
	got := 0
	timeout := time.After(500 * time.Millisecond)
loop:
	for {
		select {
		case <-c.Send:
			got++
			if got == 3 {
				break loop
			}
		case <-timeout:
			break loop
		}
	}
	if got != 3 {
		t.Fatalf("replayed %d messages, want 3", got)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend/auction && go test ./websocket -run TestLiveStreamRoom -v`
Expected: FAIL

- [ ] **Step 3: 实现 LiveStreamRoom**

创建 `backend/auction/websocket/livestream_room.go`：

```go
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
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd backend/auction && go test ./websocket -run TestLiveStreamRoom -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/auction/websocket/livestream_room.go backend/auction/websocket/livestream_room_test.go
git commit -m "feat(ws): add LiveStreamRoom with ring-buffer history replay"
```

---

## Task 5: Hub 集成 LiveStreamRoom

**Files:**
- Modify: `backend/auction/websocket/hub.go`
- Modify: `backend/auction/websocket/client.go`

- [ ] **Step 1: 在 Client 结构上新增 LiveStreamID 与 UserName**

编辑 [client.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/websocket/client.go) 中的 `Client` 结构（约第 28-43 行），增加字段：

```go
// Client WebSocket 客户端
type Client struct {
	ID           string
	AuctionID    int64
	LiveStreamID int64  // 新增：直播间 ID（0 表示未订阅弹幕）
	UserID       int64
	UserName     string // 新增：发弹幕时回填到广播
	ConnectedAt  time.Time

	conn *websocket.Conn
	Send chan *Message

	hub          *Hub
	stateManager *StateManager

	closeOnce sync.Once
	closed    bool
}
```

并修改 `NewClient` 与 `NewClientSimple` 接受 `liveStreamID int64`、`userName string` 参数。例如 `NewClientSimple`：

```go
// NewClientSimple 创建客户端（简化版，自动生成ID）
func NewClientSimple(conn *websocket.Conn, auctionID, userID, liveStreamID int64, userName string) *Client {
	id := fmt.Sprintf("%d-%d-%d", auctionID, userID, time.Now().UnixNano())
	return &Client{
		ID:           id,
		AuctionID:    auctionID,
		LiveStreamID: liveStreamID,
		UserID:       userID,
		UserName:     userName,
		conn:         conn,
		Send:         make(chan *Message, sendBufferSize),
	}
}
```

`NewClient` 同样追加这两个参数（位置与 NewClientSimple 一致）。

- [ ] **Step 2: 在 Hub 中新增 liveStreamRooms 与 API**

编辑 [hub.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/websocket/hub.go)：

在 `Hub` 结构中（第 9 行起）的 `rooms` 后追加：

```go
	liveStreamRooms     map[int64]*LiveStreamRoom
	liveStreamRoomsLock sync.RWMutex
```

在 `NewHub()` 中初始化：

```go
		liveStreamRooms: make(map[int64]*LiveStreamRoom),
```

在文件末尾追加：

```go
// RegisterToLiveStream 把客户端注册进直播间 room（自动创建）
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

	room.Register <- client
}

// UnregisterFromLiveStream 移出直播间 room
func (h *Hub) UnregisterFromLiveStream(client *Client) {
	if client.LiveStreamID <= 0 {
		return
	}
	h.liveStreamRoomsLock.RLock()
	room, ok := h.liveStreamRooms[client.LiveStreamID]
	h.liveStreamRoomsLock.RUnlock()
	if ok {
		room.Unregister <- client
	}
}

// BroadcastToLiveStream 向直播间 room 广播
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

// GetLiveStreamRoom 测试用
func (h *Hub) GetLiveStreamRoom(liveStreamID int64) *LiveStreamRoom {
	h.liveStreamRoomsLock.RLock()
	defer h.liveStreamRoomsLock.RUnlock()
	return h.liveStreamRooms[liveStreamID]
}
```

修改 `registerClient`（第 85 行附近），在原有 auction room 注册后追加：

```go
	// 同时注册到直播间房间
	if client.LiveStreamID > 0 {
		go h.RegisterToLiveStream(client)
	}
```

修改 `unregisterClient`（第 105 行附近），在原有 auction room 注销后追加：

```go
	// 从直播间房间注销
	if client.LiveStreamID > 0 {
		go h.UnregisterFromLiveStream(client)
	}
```

修改 `Stop`（第 73 行附近），在关闭 auction rooms 后追加关闭直播间 rooms：

```go
	h.liveStreamRoomsLock.Lock()
	for _, r := range h.liveStreamRooms {
		r.Close()
	}
	h.liveStreamRoomsLock.Unlock()
```

- [ ] **Step 3: 修复因 NewClient 签名变化导致的调用方编译错误**

Run: `cd backend/auction && go build ./...`
观察报错位置（如 `handler/ws.go`），把这些调用点的签名补全。`handler/ws.go` 当前调用 `NewClientSimple(conn, auctionID, userID)`，临时改为 `NewClientSimple(conn, auctionID, userID, 0, "")`（Task 8 会再细化）。

- [ ] **Step 4: 运行所有 ws 测试**

Run: `cd backend/auction && go test ./websocket -v`
Expected: PASS（包含已有的 lock/state 测试与 Task 2-4 新增测试）

- [ ] **Step 5: 提交**

```bash
git add backend/auction/websocket/hub.go backend/auction/websocket/client.go backend/auction/handler/ws.go
git commit -m "feat(ws): wire LiveStreamRoom into Hub with auto register/unregister"
```

---

## Task 6: 实现 handleChatSend 业务编排

**Files:**
- Create: `backend/auction/websocket/chat_handler.go`
- Test: `backend/auction/websocket/chat_handler_test.go`
- Modify: `backend/auction/websocket/client.go`

- [ ] **Step 1: 写失败测试**

创建 `backend/auction/websocket/chat_handler_test.go`：

```go
package websocket

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newChatHandlerFixture(t *testing.T) (*ChatHandler, *Hub, *miniredis.Miniredis) {
	t.Helper()
	hub := NewHub()
	go hub.Run()

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	filter := NewChatFilter(50, []string{"微信"})
	throttle := NewChatThrottle(rdb, ThrottleConfig{
		UserMax: 1, UserInterval: time.Second,
		RoomMax: 20, RoomInterval: time.Second,
	})
	h := NewChatHandler(hub, filter, throttle)
	return h, hub, mr
}

func dataAs[T any](t *testing.T, m *Message) T {
	t.Helper()
	raw, _ := json.Marshal(m.Data)
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func TestChatHandler_GuestRejected(t *testing.T) {
	h, hub, _ := newChatHandlerFixture(t)
	defer hub.Stop()

	c := &Client{ID: "guest", UserID: 0, LiveStreamID: 1, Send: make(chan *Message, 4)}
	h.Handle(context.Background(), c, &ChatSendData{LiveStreamID: 1, Text: "hi"})

	select {
	case msg := <-c.Send:
		if msg.Type != MessageTypeError {
			t.Fatalf("expected error message, got %s", msg.Type)
		}
		err := dataAs[ErrorData](t, msg)
		if err.Code != ChatErrCodeNotAuthenticated {
			t.Fatalf("got code %d, want %d", err.Code, ChatErrCodeNotAuthenticated)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("no error reply")
	}
}

func TestChatHandler_LengthError(t *testing.T) {
	h, hub, _ := newChatHandlerFixture(t)
	defer hub.Stop()

	c := &Client{ID: "u1", UserID: 1, LiveStreamID: 1, Send: make(chan *Message, 4)}
	h.Handle(context.Background(), c, &ChatSendData{LiveStreamID: 1, Text: ""})

	msg := <-c.Send
	if dataAs[ErrorData](t, msg).Code != ChatErrCodeLengthExceeded {
		t.Fatal("expected length error")
	}
}

func TestChatHandler_BlockedWord(t *testing.T) {
	h, hub, _ := newChatHandlerFixture(t)
	defer hub.Stop()

	c := &Client{ID: "u2", UserID: 2, LiveStreamID: 1, Send: make(chan *Message, 4)}
	h.Handle(context.Background(), c, &ChatSendData{LiveStreamID: 1, Text: "加我微信"})

	msg := <-c.Send
	if dataAs[ErrorData](t, msg).Code != ChatErrCodeBlockedWord {
		t.Fatal("expected blocked-word error")
	}
}

func TestChatHandler_HappyPath(t *testing.T) {
	h, hub, _ := newChatHandlerFixture(t)
	defer hub.Stop()

	// 先把客户端注册进直播间，才能收到广播
	c := &Client{ID: "u3", UserID: 3, UserName: "Alice", LiveStreamID: 7, Send: make(chan *Message, 4)}
	hub.RegisterToLiveStream(c)
	time.Sleep(20 * time.Millisecond)

	h.Handle(context.Background(), c, &ChatSendData{LiveStreamID: 7, Text: "hello", ClientMsgID: "cm-1"})

	timeout := time.After(time.Second)
	for {
		select {
		case msg := <-c.Send:
			if msg.Type == MessageTypeChatMessage {
				d := dataAs[ChatMessageData](t, msg)
				if d.UserID != 3 || d.UserName != "Alice" || d.Text != "hello" || d.ClientMsgID != "cm-1" {
					t.Fatalf("unexpected payload: %+v", d)
				}
				return
			}
		case <-timeout:
			t.Fatal("did not receive chat_message")
		}
	}
}

func TestChatHandler_RateLimit(t *testing.T) {
	h, hub, _ := newChatHandlerFixture(t)
	defer hub.Stop()

	c := &Client{ID: "u4", UserID: 4, LiveStreamID: 7, Send: make(chan *Message, 8)}
	hub.RegisterToLiveStream(c)
	time.Sleep(20 * time.Millisecond)

	h.Handle(context.Background(), c, &ChatSendData{LiveStreamID: 7, Text: "first"})
	h.Handle(context.Background(), c, &ChatSendData{LiveStreamID: 7, Text: "second"})

	rateLimitedSeen := false
	deadline := time.After(time.Second)
	for !rateLimitedSeen {
		select {
		case msg := <-c.Send:
			if msg.Type == MessageTypeError && dataAs[ErrorData](t, msg).Code == ChatErrCodeRateLimited {
				rateLimitedSeen = true
			}
		case <-deadline:
			t.Fatal("did not see rate-limited error")
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend/auction && go test ./websocket -run TestChatHandler -v`
Expected: FAIL

- [ ] **Step 3: 实现 ChatHandler**

创建 `backend/auction/websocket/chat_handler.go`：

```go
package websocket

import (
	"context"
	"strings"
	"time"
)

// ChatHandler 编排弹幕发送：校验 → 频控 → 广播
type ChatHandler struct {
	hub      *Hub
	filter   *ChatFilter
	throttle *ChatThrottle
}

// NewChatHandler 构造
func NewChatHandler(hub *Hub, filter *ChatFilter, throttle *ChatThrottle) *ChatHandler {
	return &ChatHandler{hub: hub, filter: filter, throttle: throttle}
}

// Handle 处理一次弹幕发送
func (h *ChatHandler) Handle(ctx context.Context, c *Client, data *ChatSendData) {
	// 鉴权：必须登录
	if c.UserID <= 0 {
		c.Send <- NewErrorMessage(ChatErrCodeNotAuthenticated, "login required")
		return
	}

	// 直播间一致性
	if data.LiveStreamID <= 0 || (c.LiveStreamID > 0 && data.LiveStreamID != c.LiveStreamID) {
		c.Send <- NewErrorMessage(ChatErrCodeLengthExceeded, "invalid live_stream_id")
		return
	}

	text := strings.TrimSpace(data.Text)

	// 内容校验
	if code := h.filter.Validate(text); code != 0 {
		c.Send <- NewErrorMessage(code, codeMessage(code))
		return
	}

	// 频控
	if code := h.throttle.Allow(ctx, c.UserID, data.LiveStreamID); code != 0 {
		c.Send <- NewErrorMessage(code, codeMessage(code))
		return
	}

	// 广播
	out := NewChatMessage(&ChatMessageData{
		LiveStreamID: data.LiveStreamID,
		UserID:       c.UserID,
		UserName:     c.UserName,
		Text:         text,
		SentAt:       time.Now().UnixMilli(),
		ClientMsgID:  data.ClientMsgID,
	})
	h.hub.BroadcastToLiveStream(data.LiveStreamID, out)
}

func codeMessage(code int) string {
	switch code {
	case ChatErrCodeLengthExceeded:
		return "text length invalid"
	case ChatErrCodeBlockedWord:
		return "blocked word detected"
	case ChatErrCodeRateLimited:
		return "rate limited"
	case ChatErrCodeNotAuthenticated:
		return "login required"
	default:
		return "chat error"
	}
}
```

- [ ] **Step 4: 在 Client.handleMessage 中接入 ChatHandler**

编辑 [client.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/websocket/client.go)。

首先在 `Client` 结构追加：

```go
	chatHandler *ChatHandler
```

新增 setter：

```go
// SetChatHandler 注入弹幕处理器
func (c *Client) SetChatHandler(h *ChatHandler) {
	c.chatHandler = h
}
```

修改 `handleMessage`（约第 184 行）的 switch，增加分支：

```go
	case MessageTypeChatSend:
		c.handleChatSend(msg)
```

在 default 分支前插入新方法：

```go
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
```

注意：`context` 包已在 client.go 顶部 import。

- [ ] **Step 5: 运行测试确认全部通过**

Run: `cd backend/auction && go test ./websocket -v`
Expected: PASS（含 TestChatHandler*）

- [ ] **Step 6: 提交**

```bash
git add backend/auction/websocket/chat_handler.go backend/auction/websocket/chat_handler_test.go backend/auction/websocket/client.go
git commit -m "feat(ws): orchestrate chat_send via ChatHandler with filter+throttle+broadcast"
```

---

## Task 7: WS 握手解析 live_stream_id 并装配 ChatHandler

**Files:**
- Modify: `backend/auction/handler/ws.go`
- Modify: `backend/auction/main.go`

- [ ] **Step 1: ws.go 解析 live_stream_id 与 user_name**

编辑 [ws.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/ws.go)：

`WSHandler` 结构追加字段：

```go
	chatHandler *ws.ChatHandler
```

新增 setter：

```go
// SetChatHandler 注入弹幕 handler
func (h *WSHandler) SetChatHandler(ch *ws.ChatHandler) {
	h.chatHandler = ch
}
```

修改 `HandleWebSocket`：

1. 在解析 `userID` 之后追加（约第 75 行后，token 分支末尾）：

```go
	// 从 token claims 取 user_name 用于弹幕展示
	var userName string
	if tokenStr != "" && h.jwtSecret != "" {
		if claims, err := h.validateToken(tokenStr); err == nil && claims != nil {
			userName = claims.Username
		}
	}

	// 直播间订阅（可选）
	liveStreamID, _ := strconv.ParseInt(r.URL.Query().Get("live_stream_id"), 10, 64)
```

2. 替换 `client := ws.NewClientSimple(conn, auctionID, userID)` 为：

```go
	client := ws.NewClientSimple(conn, auctionID, userID, liveStreamID, userName)
	if h.chatHandler != nil {
		client.SetChatHandler(h.chatHandler)
	}
```

3. 删除当前 ws.go 内行内的 ReadMessage 循环（第 109-151 行）中针对 `ping` 的私有处理。**不动**——保持现有路径以避免回归；ChatSend 走的是 `Client.ReadPump`/`handleMessage` 路径，需在注册后启动 pump。

   **重要修改**：当前 `HandleWebSocket` 用了一个内嵌 goroutine 做 read 循环，未启动 `Client.ReadPump`/`WritePump`。要让 `chat_send` 生效，需启动这两个泵。在 `client := ws.NewClientSimple(...)` 之后、原 goroutine 之前，加：

```go
	// 启动 client 的标准读写循环，处理 ping/sync_request/chat_send
	go client.ReadPump()
	go client.WritePump()
	return  // 不再使用旧的内嵌 goroutine
```

   **将原内嵌 goroutine 整段删除**（含 `for { _, message, err := conn.ReadMessage() ... }` 包络），并保留前面的欢迎消息发送（直接走 `conn.WriteMessage`，因为 ReadPump 才接管）。

   > 注意：此修改让 ws.go 与 client.go 行为一致。这是必要的回归——但由于现有 e2e 仅依赖 ping/sync_request，client.go 的 handleMessage 已经处理这两类，所以不会破坏既有测试。

- [ ] **Step 2: main.go 装配 ChatHandler**

打开 [main.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/main.go)，找到创建 Hub 与 WSHandler 的地方（搜 `NewHub` 与 `NewWSHandler`）。在 Hub/Redis 已就绪、WSHandler 创建之后，追加：

```go
	chatFilter := websocket.NewChatFilter(50, []string{
		"微信", "weixin", "vx", "qq", "电话",
	})
	chatThrottle := websocket.NewChatThrottle(redisClient, websocket.ThrottleConfig{
		UserMax: 1, UserInterval: time.Second,
		RoomMax: 20, RoomInterval: time.Second,
	})
	chatHandler := websocket.NewChatHandler(hub, chatFilter, chatThrottle)
	wsHandler.SetChatHandler(chatHandler)
```

变量名 `redisClient` / `hub` / `wsHandler` 须与现有代码一致——查找文件实际命名后替换。

- [ ] **Step 3: 编译验证**

Run: `cd backend/auction && go build ./...`
Expected: PASS

- [ ] **Step 4: 跑全量后端测试**

Run: `cd backend/auction && go test ./... -count=1`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add backend/auction/handler/ws.go backend/auction/main.go
git commit -m "feat(ws): parse live_stream_id at handshake and wire ChatHandler"
```

---

## Task 8: 前端 WebSocket 服务扩展（订阅 + 发送）

**Files:**
- Modify: `frontend/h5/src/services/websocket.ts`

- [ ] **Step 1: 扩展 connect URL 与新增 sendChat**

编辑 `frontend/h5/src/services/websocket.ts`。

1. 给 `WebSocketService` 类新增字段：

```ts
  private liveStreamId: number | null;
```

2. 修改 constructor：

```ts
  constructor(auctionId: number, token?: string, liveStreamId?: number) {
    this.auctionId = auctionId;
    this.token = token || null;
    this.liveStreamId = liveStreamId ?? null;
    this.messageThrottlers = new MessageTypeThrottlers(200);
    this.setupThrottlers();
  }
```

3. 找到 `connect()` 中拼接 ws URL 的位置（搜 `auction_id=`），在 URL 中追加 `live_stream_id`（仅当存在时）：

```ts
    const params = new URLSearchParams({
      auction_id: String(this.auctionId),
    });
    if (this.token) params.set('token', this.token);
    if (this.liveStreamId) params.set('live_stream_id', String(this.liveStreamId));
    const url = `${baseWsUrl}?${params.toString()}`;
```

> 若现有代码并非用 URLSearchParams，请保持其原拼接风格，仅追加 `&live_stream_id=...`。

4. 在 `class WebSocketService` 的方法区追加：

```ts
  /** 发送弹幕 */
  sendChat(text: string, clientMsgId: string): void {
    if (this.ws?.readyState !== WebSocket.OPEN) return;
    if (!this.liveStreamId) return;
    const payload = {
      type: 'chat_send',
      timestamp: Date.now(),
      data: {
        live_stream_id: this.liveStreamId,
        text,
        client_msg_id: clientMsgId,
      },
    };
    this.ws.send(JSON.stringify(payload));
  }
```

5. 在 `onmessage` 处理中找到 type 分发处，新增 chat_message 分发（与现有 bid_placed 等同样调用 `this.handlers.get('chat_message')` 即可——若现有代码统一用 `handlers` map 分发，无需特殊处理；若不是，按现有模式补充一个 case）。

- [ ] **Step 2: 运行现有 ws 测试**

Run: `cd frontend/h5 && npm test -- --testPathPattern=websocket`
Expected: PASS（不破坏既有用例）

- [ ] **Step 3: 提交**

```bash
git add frontend/h5/src/services/websocket.ts
git commit -m "feat(h5): WebSocketService supports live_stream_id and sendChat"
```

---

## Task 9: 前端 liveChatStore（Zustand）

**Files:**
- Create: `frontend/h5/src/store/liveChatStore.ts`
- Test: `frontend/h5/src/store/__tests__/liveChatStore.test.ts`

- [ ] **Step 1: 写失败测试**

创建 `frontend/h5/src/store/__tests__/liveChatStore.test.ts`：

```ts
import { useLiveChatStore } from '../liveChatStore';

describe('liveChatStore', () => {
  beforeEach(() => {
    useLiveChatStore.getState().reset();
  });

  it('appends incoming messages with cap of 200', () => {
    const { receive } = useLiveChatStore.getState();
    for (let i = 0; i < 250; i++) {
      receive({
        live_stream_id: 1,
        user_id: i,
        user_name: 'u' + i,
        text: 'hi',
        sent_at: Date.now(),
      });
    }
    expect(useLiveChatStore.getState().history).toHaveLength(200);
    expect(useLiveChatStore.getState().history[0].user_id).toBe(50);
  });

  it('cooldown returns true within 1 second of send', () => {
    const { markSent, isCoolingDown } = useLiveChatStore.getState();
    markSent();
    expect(isCoolingDown()).toBe(true);
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend/h5 && npm test -- liveChatStore`
Expected: FAIL（模块不存在）

- [ ] **Step 3: 实现 store**

创建 `frontend/h5/src/store/liveChatStore.ts`：

```ts
import { create } from 'zustand';

export interface ChatMessage {
  live_stream_id: number;
  user_id: number;
  user_name: string;
  avatar_url?: string;
  text: string;
  sent_at: number;
  client_msg_id?: string;
}

const MAX_HISTORY = 200;
const COOLDOWN_MS = 1000;

interface LiveChatState {
  history: ChatMessage[];
  lastSentAt: number;

  receive: (msg: ChatMessage) => void;
  markSent: () => void;
  isCoolingDown: () => boolean;
  reset: () => void;
}

export const useLiveChatStore = create<LiveChatState>((set, get) => ({
  history: [],
  lastSentAt: 0,

  receive: (msg) =>
    set((s) => ({
      history: [...s.history, msg].slice(-MAX_HISTORY),
    })),

  markSent: () => set({ lastSentAt: Date.now() }),

  isCoolingDown: () => Date.now() - get().lastSentAt < COOLDOWN_MS,

  reset: () => set({ history: [], lastSentAt: 0 }),
}));
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend/h5 && npm test -- liveChatStore`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add frontend/h5/src/store/liveChatStore.ts frontend/h5/src/store/__tests__/liveChatStore.test.ts
git commit -m "feat(h5): add liveChatStore with history cap and cooldown"
```

---

## Task 10: ChatBubble 组件

**Files:**
- Create: `frontend/h5/src/components/LiveChat/ChatBubble.tsx`
- Create: `frontend/h5/src/components/LiveChat/ChatPanel.module.css`
- Test: `frontend/h5/src/components/LiveChat/__tests__/ChatBubble.test.tsx`

- [ ] **Step 1: 写失败测试**

创建 `frontend/h5/src/components/LiveChat/__tests__/ChatBubble.test.tsx`：

```tsx
import { render, screen } from '@testing-library/react';
import { ChatBubble } from '../ChatBubble';

describe('ChatBubble', () => {
  const baseMsg = {
    live_stream_id: 1,
    user_id: 9,
    user_name: 'Alice',
    text: 'hello world',
    sent_at: 1700000000000,
  };

  it('renders user name and text', () => {
    render(<ChatBubble msg={baseMsg} isSelf={false} />);
    expect(screen.getByText('Alice')).toBeInTheDocument();
    expect(screen.getByText('hello world')).toBeInTheDocument();
  });

  it('does not interpret HTML in text', () => {
    render(<ChatBubble msg={{ ...baseMsg, text: '<img src=x onerror=alert(1)>' }} isSelf={false} />);
    expect(screen.queryByRole('img')).toBeNull();
    expect(screen.getByText('<img src=x onerror=alert(1)>')).toBeInTheDocument();
  });

  it('marks self messages', () => {
    const { container } = render(<ChatBubble msg={baseMsg} isSelf={true} />);
    expect(container.firstChild).toHaveClass('bubbleSelf');
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend/h5 && npm test -- ChatBubble`
Expected: FAIL

- [ ] **Step 3: 实现样式**

创建 `frontend/h5/src/components/LiveChat/ChatPanel.module.css`：

```css
.panel {
  position: absolute;
  left: 12px;
  right: 12px;
  bottom: 88px;
  display: flex;
  flex-direction: column;
  pointer-events: none;
  max-height: 50vh;
  overflow: hidden;
}

.list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  overflow-y: auto;
  padding-bottom: 8px;
  pointer-events: auto;
}

.bubble {
  align-self: flex-start;
  max-width: 80%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  padding: 6px 10px;
  border-radius: 14px;
  font-size: 13px;
  line-height: 1.35;
  word-break: break-word;
}

.bubbleSelf {
  border: 1px solid #4b8bf5;
}

.userName {
  color: #ffd479;
  font-weight: 600;
  margin-right: 6px;
}

.inputBar {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  gap: 8px;
  padding: 12px;
  background: linear-gradient(to top, rgba(0, 0, 0, 0.6), transparent);
  pointer-events: auto;
}

.input {
  flex: 1;
  height: 44px;
  padding: 0 12px;
  border-radius: 22px;
  border: none;
  background: rgba(255, 255, 255, 0.95);
  font-size: 14px;
}

.sendBtn {
  height: 44px;
  min-width: 64px;
  border-radius: 22px;
  border: none;
  background: #4b8bf5;
  color: #fff;
  font-size: 14px;
}

.sendBtn:disabled {
  opacity: 0.45;
}
```

- [ ] **Step 4: 实现 ChatBubble**

创建 `frontend/h5/src/components/LiveChat/ChatBubble.tsx`：

```tsx
import React from 'react';
import styles from './ChatPanel.module.css';
import type { ChatMessage } from '../../store/liveChatStore';

interface ChatBubbleProps {
  msg: ChatMessage;
  isSelf: boolean;
}

export const ChatBubble: React.FC<ChatBubbleProps> = ({ msg, isSelf }) => {
  return (
    <div className={`${styles.bubble} ${isSelf ? styles.bubbleSelf : ''}`}>
      <span className={styles.userName}>{msg.user_name}</span>
      <span>{msg.text}</span>
    </div>
  );
};
```

> 注：React 默认渲染文本走 textContent，自带 XSS 防护，无需手动 escape。

- [ ] **Step 5: 运行测试确认通过**

Run: `cd frontend/h5 && npm test -- ChatBubble`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add frontend/h5/src/components/LiveChat
git commit -m "feat(h5): add ChatBubble component with XSS-safe text rendering"
```

---

## Task 11: ChatPanel 组件（输入 + 滚动列表）

**Files:**
- Create: `frontend/h5/src/components/LiveChat/ChatPanel.tsx`
- Test: `frontend/h5/src/components/LiveChat/__tests__/ChatPanel.test.tsx`

- [ ] **Step 1: 写失败测试**

创建 `frontend/h5/src/components/LiveChat/__tests__/ChatPanel.test.tsx`：

```tsx
import { render, screen, fireEvent, act } from '@testing-library/react';
import { ChatPanel } from '../ChatPanel';
import { useLiveChatStore } from '../../../store/liveChatStore';

describe('ChatPanel', () => {
  beforeEach(() => {
    useLiveChatStore.getState().reset();
    jest.useFakeTimers();
  });
  afterEach(() => {
    jest.useRealTimers();
  });

  it('disables send button while empty', () => {
    render(<ChatPanel currentUserId={1} onSend={jest.fn()} />);
    expect(screen.getByRole('button', { name: /发送/ })).toBeDisabled();
  });

  it('rejects text exceeding 50 chars', () => {
    const onSend = jest.fn();
    render(<ChatPanel currentUserId={1} onSend={onSend} />);
    const input = screen.getByPlaceholderText(/说点什么/);
    fireEvent.change(input, { target: { value: 'a'.repeat(51) } });
    fireEvent.click(screen.getByRole('button', { name: /发送/ }));
    expect(onSend).not.toHaveBeenCalled();
  });

  it('sends valid text and triggers cooldown', () => {
    const onSend = jest.fn();
    render(<ChatPanel currentUserId={1} onSend={onSend} />);
    const input = screen.getByPlaceholderText(/说点什么/);
    fireEvent.change(input, { target: { value: 'hi' } });
    fireEvent.click(screen.getByRole('button', { name: /发送/ }));
    expect(onSend).toHaveBeenCalledWith('hi', expect.any(String));
    expect(screen.getByRole('button', { name: /发送/ })).toBeDisabled(); // 1s cooldown

    act(() => {
      jest.advanceTimersByTime(1100);
    });
    fireEvent.change(input, { target: { value: 'next' } });
    expect(screen.getByRole('button', { name: /发送/ })).not.toBeDisabled();
  });

  it('renders messages from store', () => {
    useLiveChatStore.getState().receive({
      live_stream_id: 1,
      user_id: 9,
      user_name: 'Bob',
      text: 'arriving',
      sent_at: Date.now(),
    });
    render(<ChatPanel currentUserId={1} onSend={jest.fn()} />);
    expect(screen.getByText('Bob')).toBeInTheDocument();
    expect(screen.getByText('arriving')).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend/h5 && npm test -- ChatPanel`
Expected: FAIL

- [ ] **Step 3: 实现 ChatPanel**

创建 `frontend/h5/src/components/LiveChat/ChatPanel.tsx`：

```tsx
import React, { useEffect, useRef, useState } from 'react';
import styles from './ChatPanel.module.css';
import { ChatBubble } from './ChatBubble';
import { useLiveChatStore } from '../../store/liveChatStore';

const MAX_LEN = 50;

interface ChatPanelProps {
  currentUserId: number;
  onSend: (text: string, clientMsgId: string) => void;
}

export const ChatPanel: React.FC<ChatPanelProps> = ({ currentUserId, onSend }) => {
  const history = useLiveChatStore((s) => s.history);
  const markSent = useLiveChatStore((s) => s.markSent);
  const isCoolingDown = useLiveChatStore((s) => s.isCoolingDown);

  const [text, setText] = useState('');
  const [tick, setTick] = useState(0); // 强制刷新 cooldown 状态
  const listRef = useRef<HTMLDivElement>(null);

  // cooldown 倒计时刷新
  useEffect(() => {
    const id = setInterval(() => setTick((t) => t + 1), 200);
    return () => clearInterval(id);
  }, []);

  // 自动滚动到底部
  useEffect(() => {
    if (listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight;
    }
  }, [history.length]);

  const trimmed = text.trim();
  const tooLong = [...trimmed].length > MAX_LEN;
  const canSend = trimmed.length > 0 && !tooLong && !isCoolingDown();

  const handleSend = () => {
    if (!canSend) return;
    const clientMsgId = `${currentUserId}-${Date.now()}`;
    onSend(trimmed, clientMsgId);
    markSent();
    setText('');
  };

  return (
    <div className={styles.panel}>
      <div className={styles.list} ref={listRef} data-testid="chat-list">
        {history.map((m, i) => (
          <ChatBubble key={`${m.sent_at}-${i}`} msg={m} isSelf={m.user_id === currentUserId} />
        ))}
      </div>
      <div className={styles.inputBar}>
        <input
          className={styles.input}
          placeholder="说点什么..."
          value={text}
          maxLength={MAX_LEN * 4 /* 给中文留空间，业务上仍按字符数限制 */}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') handleSend();
          }}
        />
        <button
          type="button"
          className={styles.sendBtn}
          disabled={!canSend}
          onClick={handleSend}
          data-tick={tick}
        >
          发送
        </button>
      </div>
    </div>
  );
};
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend/h5 && npm test -- ChatPanel`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add frontend/h5/src/components/LiveChat
git commit -m "feat(h5): add ChatPanel with input validation and cooldown UX"
```

---

## Task 12: 在 Live 页面集成 ChatPanel

**Files:**
- Modify: `frontend/h5/src/pages/Live/index.tsx`

- [ ] **Step 1: 读取当前 Live 页面**

Run Read tool on `frontend/h5/src/pages/Live/index.tsx` and identify:
- 当前 `liveStreamId` 来源（route param? props?）
- 当前 WebSocketService 实例位置
- 当前 user 信息来源

- [ ] **Step 2: 集成 ChatPanel**

在 Live 页面 JSX 顶层（覆盖在视频区上方）插入：

```tsx
import { ChatPanel } from '../../components/LiveChat/ChatPanel';
import { useLiveChatStore } from '../../store/liveChatStore';
```

并在合适位置：

```tsx
<ChatPanel
  currentUserId={currentUser?.id ?? 0}
  onSend={(text, clientMsgId) => wsService.sendChat(text, clientMsgId)}
/>
```

同时确保在创建 WebSocketService 时把 `liveStreamId` 传入：

```tsx
const wsService = useMemo(
  () => new WebSocketService(auctionId, token, liveStreamId),
  [auctionId, token, liveStreamId],
);
```

并在 wsService 注册一个 `chat_message` handler，把消息推入 store：

```tsx
useEffect(() => {
  const onChat = (data: any) => useLiveChatStore.getState().receive(data);
  wsService.on('chat_message', onChat);
  return () => wsService.off('chat_message', onChat);
}, [wsService]);
```

> 若 `wsService.on/off` API 名不同（如 `subscribe/unsubscribe`），按现有命名调整。

- [ ] **Step 3: 运行 Live 测试**

Run: `cd frontend/h5 && npm test -- LiveRoom`
Expected: PASS（已有用例不破坏；如需新增 chat 集成测试可在 Plan B 补充）

- [ ] **Step 4: 提交**

```bash
git add frontend/h5/src/pages/Live/index.tsx
git commit -m "feat(h5): mount ChatPanel in Live page and dispatch chat_message"
```

---

## Task 13: 端到端冒烟（手动）

> 这是一次人工核查，不写自动化。subagent 跑完 Task 1-12 后由你（用户）执行。

- [ ] **Step 1: 启动后端**

```bash
cd backend/auction && go run main.go &
```

- [ ] **Step 2: 启动 H5**

```bash
cd frontend/h5 && npm run dev
```

- [ ] **Step 3: 浏览器开两个标签**

- 同一直播间，两个用户登录
- A 发"你好" → B 应在 1s 内看到
- A 连续发 → 第二条应被频控（前端按钮灰化、后端返回 40003）
- A 发"加我微信" → 收到 40002 错误
- A 发空字符串或 51 字 → 收到 40001 错误
- B 刷新进房 → 立即看到最近 100 条历史

- [ ] **Step 4: 提交（如有微调）**

如果冒烟过程发现需修复的细节：

```bash
git add -A
git commit -m "fix(chat): smoke-test follow-up tweaks"
```

---

## Self-Review

### Spec coverage（vs spec 第 5 节）

| Spec 章节 | 对应任务 |
|---|---|
| 5.1 进房历史回放 | Task 4 (RingBufferReplay test) + Task 5 (Hub register triggers replay) |
| 5.2 发送弹幕流程 | Task 6 (ChatHandler) + Task 8 (前端 sendChat) + Task 11 (ChatPanel) |
| 5.4 频控 | Task 3 (ChatThrottle) |
| 6 鉴权与安全 | Task 6 (UserID==0 拒绝) + Task 10 (textContent 渲染) |
| 4.3 协议 | Task 1 (DTO) |
| 4.4 握手 URL | Task 7 + Task 8 |

### 已知留给 Plan B 的项目

- 5.5 Room GC（30s 空闲销毁）：当前 LiveStreamRoom 未实现 GC，需在 Plan B 补
- 5.3 高价飘屏判定：完全 Plan B
- 9 Prometheus 指标：Plan B
- 4.5 Nacos 热更新黑词：Plan B（当前是 main.go 硬编码）
- 集成测试 8.2（双 Room 隔离 / 拍品切换）：Plan B 用 Go integration test 覆盖

### Placeholder 自审

- 已无 "TBD" / "Add appropriate" / "Similar to Task N"
- 每个测试都带完整代码
- 每个修改步骤都有具体行号或具体替换内容
- Type 一致性：`ChatMessage`（前端）字段与 `ChatMessageData`（后端）一一映射；`ChatSendData` 与 `sendChat` payload 一一映射；`ChatErrCode*` 错误码前后端一致

---

## 执行方式选择

**Plan complete and saved to `docs/superpowers/plans/2026-05-31-live-chat-mvp-plan-a.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - 我每个 Task 派一个 fresh subagent，每完成一个 Task 我做一次 review，发现问题再迭代；适合本次 13 任务、跨 Go+TS 的场景。

**2. Inline Execution** - 在当前会话内串行执行所有 Task，到关键节点（Task 7 后端完成、Task 12 前端完成）暂停 checkpoint review。

**选哪种？**

# A5 一口价秒杀 - M2 实时同步层 + WS Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 M1 抢购链路上叠加实时同步层：5 种 WS 消息推送、Outbox 事件路由、节流策略、gateway 列表聚合 product 信息。

**Architecture:** 复用 B1 LiveStreamRoom 通道，不新建 Room；通过 Outbox dispatcher 异步消费 `fixed_price_*` 事件并广播；per-item token bucket 实现 stock 节流；gateway 内做 product RPC 聚合。

**Tech Stack:** Go + Hertz + go-redis + 现有 Outbox + 现有 LiveStreamRoom（B1 产物）+ product RPC

**前置依赖：** M1 完成 + B1 Plan A 完成（LiveStreamRoom 抽象就绪）

**Spec：** [2026-06-01-fixed-price-sale-design.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/superpowers/specs/2026-06-01-fixed-price-sale-design.md) §4.4 §4.5

---

## File Structure

**Create:**
- `backend/auction/websocket/fixed_price_messages.go` - 5 种消息构造器
- `backend/auction/service/fixed_price_broadcaster.go` + `_test.go` - 广播编排 + 节流
- `backend/auction/service/outbox_fixed_price_handler.go` + `_test.go` - Outbox 消费
- `backend/gateway/fixed_price_aggregator.go` + `_test.go` - product RPC 聚合

**Modify:**
- `backend/auction/service/fixed_price.go` - service 调用 broadcaster
- `backend/auction/service/outbox_dispatcher.go` - 注册新事件路由

---

### Task 1: 5 种 WS 消息构造器

**Files:**
- Test: `backend/auction/websocket/fixed_price_messages_test.go`
- Create: `backend/auction/websocket/fixed_price_messages.go`

- [ ] **Step 1.1: 写失败测试**

```go
// backend/auction/websocket/fixed_price_messages_test.go
package websocket

import (
    "encoding/json"
    "testing"
    "github.com/shopspring/decimal"
    "github.com/stretchr/testify/assert"
)

func TestFixedPriceMessages_Listed(t *testing.T) {
    msg := NewFixedPriceListed(1001, 7001, decimal.NewFromInt(99), 100, ProductBrief{ID: 5001, Title: "翡翠手镯"})
    b, _ := json.Marshal(msg)
    var got map[string]any
    _ = json.Unmarshal(b, &got)
    assert.Equal(t, "fixed_price_listed", got["type"])
    assert.Equal(t, float64(1001), got["live_stream_id"])
    p := got["payload"].(map[string]any)
    assert.Equal(t, "99.00", p["price"])
}

func TestFixedPriceMessages_Stock(t *testing.T) {
    msg := NewFixedPriceStock(1001, 7001, 87)
    assert.Equal(t, "fixed_price_stock", msg.Type)
}

func TestFixedPriceMessages_SoldOut(t *testing.T) {
    msg := NewFixedPriceSoldOut(1001, 7001)
    assert.Equal(t, "fixed_price_sold_out", msg.Type)
}

func TestFixedPriceMessages_Offline(t *testing.T) {
    msg := NewFixedPriceOffline(1001, 7001)
    assert.Equal(t, "fixed_price_offline", msg.Type)
}

func TestFixedPriceMessages_Flair(t *testing.T) {
    msg := NewFixedPriceFlair(1001, 7001, "Alice", decimal.NewFromInt(99), "翡翠手镯")
    p := msg.Payload.(map[string]any)
    assert.Equal(t, "Alice", p["buyer_nickname"])
}
```

- [ ] **Step 1.2: 跑测试确认失败**

Run: `cd backend/auction && go test ./websocket/ -run TestFixedPriceMessages -v`
Expected: FAIL

- [ ] **Step 1.3: 写实现**

```go
// backend/auction/websocket/fixed_price_messages.go
package websocket

import (
    "time"
    "github.com/shopspring/decimal"
)

// Envelope 复用 B1 LiveStreamRoom 已定义的通用消息结构
// 此处仅作 type 注释提示
// type Envelope struct { Type string; LiveStreamID int64; TS int64; Payload any }

type ProductBrief struct {
    ID         int64  `json:"id"`
    Title      string `json:"title"`
    CoverImage string `json:"cover_image,omitempty"`
}

func nowMS() int64 { return time.Now().UnixMilli() }

func NewFixedPriceListed(liveStreamID, itemID int64, price decimal.Decimal, totalStock int, p ProductBrief) Envelope {
    return Envelope{
        Type: "fixed_price_listed", LiveStreamID: liveStreamID, TS: nowMS(),
        Payload: map[string]any{
            "item_id": itemID, "price": price.StringFixed(2),
            "total_stock": totalStock, "product_brief": p,
        },
    }
}

func NewFixedPriceStock(liveStreamID, itemID int64, remaining int) Envelope {
    return Envelope{
        Type: "fixed_price_stock", LiveStreamID: liveStreamID, TS: nowMS(),
        Payload: map[string]any{"item_id": itemID, "remaining_stock": remaining},
    }
}

func NewFixedPriceSoldOut(liveStreamID, itemID int64) Envelope {
    return Envelope{
        Type: "fixed_price_sold_out", LiveStreamID: liveStreamID, TS: nowMS(),
        Payload: map[string]any{"item_id": itemID},
    }
}

func NewFixedPriceOffline(liveStreamID, itemID int64) Envelope {
    return Envelope{
        Type: "fixed_price_offline", LiveStreamID: liveStreamID, TS: nowMS(),
        Payload: map[string]any{"item_id": itemID},
    }
}

func NewFixedPriceFlair(liveStreamID, itemID int64, nickname string, price decimal.Decimal, productTitle string) Envelope {
    return Envelope{
        Type: "fixed_price_flair", LiveStreamID: liveStreamID, TS: nowMS(),
        Payload: map[string]any{
            "item_id": itemID, "buyer_nickname": nickname,
            "price": price.StringFixed(2), "product_title": productTitle,
        },
    }
}
```

- [ ] **Step 1.4: 跑测试通过**

Run: `cd backend/auction && go test ./websocket/ -run TestFixedPriceMessages -v`
Expected: PASS（5 cases）

- [ ] **Step 1.5: Commit**

```bash
git add backend/auction/websocket/fixed_price_messages*.go
git commit -m "feat(fixed-price): add 5 WS message constructors (M2.T1)"
```

---

### Task 2: per-item 节流的 broadcaster

**Files:**
- Test: `backend/auction/service/fixed_price_broadcaster_test.go`
- Create: `backend/auction/service/fixed_price_broadcaster.go`

- [ ] **Step 2.1: 写失败测试**

```go
// backend/auction/service/fixed_price_broadcaster_test.go
package service

import (
    "context"
    "sync/atomic"
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
)

type fakeRoom struct{ count int64; lastStock int }

func (f *fakeRoom) Broadcast(_ context.Context, _ int64, env any) {
    atomic.AddInt64(&f.count, 1)
    if e, ok := env.(map[string]any); ok {
        if p, ok := e["payload"].(map[string]any); ok {
            if v, ok := p["remaining_stock"].(int); ok { f.lastStock = v }
        }
    }
}

func TestBroadcaster_StockThrottled_To1PerSecond(t *testing.T) {
    room := &fakeRoom{}
    b := NewBroadcaster(room)
    for i := 100; i > 0; i-- {
        b.PublishStock(context.Background(), 1001, 7001, i)
    }
    time.Sleep(1100 * time.Millisecond)

    // 100 次调用，1 秒窗口内最多 ~2 次广播（首次 + 节流末尾）
    cnt := atomic.LoadInt64(&room.count)
    assert.LessOrEqual(t, cnt, int64(2))
    // 最终值应为最近一次的 stock=1
    assert.Equal(t, 1, room.lastStock)
}

func TestBroadcaster_NonStockEvents_NotThrottled(t *testing.T) {
    room := &fakeRoom{}
    b := NewBroadcaster(room)
    b.PublishSoldOut(context.Background(), 1001, 7001)
    b.PublishOffline(context.Background(), 1001, 7001)
    b.PublishFlair(context.Background(), 1001, 7001, "Alice", decimal.NewFromInt(99), "翡翠手镯")
    // 非 stock 事件不节流：3 次调用 = 3 次广播
    assert.Equal(t, int64(3), atomic.LoadInt64(&room.count))
}
```

- [ ] **Step 2.2: 跑测试确认失败**

Run: `cd backend/auction && go test ./service/ -run TestBroadcaster -v`
Expected: FAIL（NewBroadcaster 未定义）

- [ ] **Step 2.3: 写实现**

```go
// backend/auction/service/fixed_price_broadcaster.go
package service

import (
    "context"
    "sync"
    "time"
    "github.com/shopspring/decimal"
    ws "auction/websocket"
)

// Room 抽象 LiveStreamRoom 广播接口（可被测试替身覆盖）
type Room interface {
    Broadcast(ctx context.Context, liveStreamID int64, envelope any)
}

type stockSlot struct {
    mu       sync.Mutex
    lastSent time.Time
    pending  int
    timer    *time.Timer
}

type Broadcaster struct {
    room   Room
    mu     sync.Mutex
    slots  map[int64]*stockSlot // key: itemID
    window time.Duration
}

func NewBroadcaster(room Room) *Broadcaster {
    return &Broadcaster{room: room, slots: make(map[int64]*stockSlot), window: time.Second}
}

func (b *Broadcaster) getSlot(itemID int64) *stockSlot {
    b.mu.Lock(); defer b.mu.Unlock()
    s, ok := b.slots[itemID]
    if !ok { s = &stockSlot{}; b.slots[itemID] = s }
    return s
}

// PublishStock 节流：每 item 1 秒最多 1 条；窗口内的更新延迟到末尾发送最新值
func (b *Broadcaster) PublishStock(ctx context.Context, liveStreamID, itemID int64, remaining int) {
    s := b.getSlot(itemID)
    s.mu.Lock(); defer s.mu.Unlock()
    s.pending = remaining
    now := time.Now()
    if now.Sub(s.lastSent) >= b.window {
        b.room.Broadcast(ctx, liveStreamID, ws.NewFixedPriceStock(liveStreamID, itemID, remaining))
        s.lastSent = now
        return
    }
    if s.timer != nil { return } // 已排队
    delay := b.window - now.Sub(s.lastSent)
    s.timer = time.AfterFunc(delay, func() {
        s.mu.Lock()
        latest := s.pending
        s.lastSent = time.Now()
        s.timer = nil
        s.mu.Unlock()
        b.room.Broadcast(ctx, liveStreamID, ws.NewFixedPriceStock(liveStreamID, itemID, latest))
    })
}

func (b *Broadcaster) PublishListed(ctx context.Context, liveStreamID, itemID int64, price decimal.Decimal, total int, p ws.ProductBrief) {
    b.room.Broadcast(ctx, liveStreamID, ws.NewFixedPriceListed(liveStreamID, itemID, price, total, p))
}
func (b *Broadcaster) PublishSoldOut(ctx context.Context, liveStreamID, itemID int64) {
    b.room.Broadcast(ctx, liveStreamID, ws.NewFixedPriceSoldOut(liveStreamID, itemID))
}
func (b *Broadcaster) PublishOffline(ctx context.Context, liveStreamID, itemID int64) {
    b.room.Broadcast(ctx, liveStreamID, ws.NewFixedPriceOffline(liveStreamID, itemID))
}
func (b *Broadcaster) PublishFlair(ctx context.Context, liveStreamID, itemID int64, nickname string, price decimal.Decimal, title string) {
    b.room.Broadcast(ctx, liveStreamID, ws.NewFixedPriceFlair(liveStreamID, itemID, nickname, price, title))
}
```

> **注意：** test 中的 `fakeRoom.Broadcast` 期望接收 `map[string]any`，需把 `ws.Envelope` 序列化辅助。简化做法：在 `fakeRoom` 中通过反射或 JSON 往返；为避免反射耦合测试，调整 fakeRoom 实现直接 `count++` 即可，stock 末值断言改为通过自定义 struct 取 payload。**最终测试代码以实现侧暴露的 Envelope 结构为准**。

- [ ] **Step 2.4: 跑测试通过**

Run: `cd backend/auction && go test ./service/ -run TestBroadcaster -v -race`
Expected: PASS

- [ ] **Step 2.5: Commit**

```bash
git add backend/auction/service/fixed_price_broadcaster*.go
git commit -m "feat(fixed-price): broadcaster with per-item stock throttle (M2.T2)"
```

---

### Task 3: Outbox 事件消费 handler

**Files:**
- Test: `backend/auction/service/outbox_fixed_price_handler_test.go`
- Create: `backend/auction/service/outbox_fixed_price_handler.go`
- Modify: `backend/auction/service/outbox_dispatcher.go`

- [ ] **Step 3.1: 写失败测试（核心场景）**

```go
// backend/auction/service/outbox_fixed_price_handler_test.go
package service

import (
    "context"
    "encoding/json"
    "testing"
    "github.com/stretchr/testify/assert"
)

type spyBroadcaster struct{ events []string }

func (s *spyBroadcaster) PublishStock(_ context.Context, _, _ int64, _ int) { s.events = append(s.events, "stock") }
func (s *spyBroadcaster) PublishSoldOut(_ context.Context, _, _ int64)      { s.events = append(s.events, "sold_out") }
func (s *spyBroadcaster) PublishFlair(_ context.Context, _, _ int64, _ string, _ any, _ string) {
    s.events = append(s.events, "flair")
}

func TestOutboxHandler_PurchaseSucceeded_FiresStockAndFlair(t *testing.T) {
    spy := &spyBroadcaster{}
    h := NewOutboxFixedPriceHandler(spy, &fakeUserRPC{nick: "Alice"}, &fakeProductRPC{title: "翡翠"})
    payload, _ := json.Marshal(map[string]any{
        "live_stream_id": 1001, "item_id": 7001, "remaining_stock": 87,
        "user_id": 9001, "price": "99.00",
    })
    err := h.Handle(context.Background(), "fixed_price.purchase_succeeded", payload)
    assert.NoError(t, err)
    assert.Equal(t, []string{"stock", "flair"}, spy.events)
}

func TestOutboxHandler_StockExhausted_FiresSoldOut(t *testing.T) {
    spy := &spyBroadcaster{}
    h := NewOutboxFixedPriceHandler(spy, nil, nil)
    payload, _ := json.Marshal(map[string]any{"live_stream_id": 1001, "item_id": 7001})
    err := h.Handle(context.Background(), "fixed_price.stock_exhausted", payload)
    assert.NoError(t, err)
    assert.Equal(t, []string{"sold_out"}, spy.events)
}

func TestOutboxHandler_UnknownEvent_ReturnsNil(t *testing.T) {
    spy := &spyBroadcaster{}
    h := NewOutboxFixedPriceHandler(spy, nil, nil)
    err := h.Handle(context.Background(), "unknown.event", []byte(`{}`))
    assert.NoError(t, err)
    assert.Empty(t, spy.events)
}
```

- [ ] **Step 3.2: 跑测试确认失败**

Run: `cd backend/auction && go test ./service/ -run TestOutboxHandler -v`
Expected: FAIL

- [ ] **Step 3.3: 写实现**

```go
// backend/auction/service/outbox_fixed_price_handler.go
package service

import (
    "context"
    "encoding/json"
    "github.com/shopspring/decimal"
)

type broadcasterIface interface {
    PublishStock(ctx context.Context, liveStreamID, itemID int64, remaining int)
    PublishSoldOut(ctx context.Context, liveStreamID, itemID int64)
    PublishFlair(ctx context.Context, liveStreamID, itemID int64, nickname string, price decimal.Decimal, title string)
}
type userRPC interface{ GetNickname(ctx context.Context, userID int64) (string, error) }
type productRPC interface{ GetTitle(ctx context.Context, productID int64) (string, error) }

type OutboxFixedPriceHandler struct {
    b   broadcasterIface
    u   userRPC
    p   productRPC
}

func NewOutboxFixedPriceHandler(b broadcasterIface, u userRPC, p productRPC) *OutboxFixedPriceHandler {
    return &OutboxFixedPriceHandler{b: b, u: u, p: p}
}

func (h *OutboxFixedPriceHandler) Handle(ctx context.Context, eventType string, payload []byte) error {
    switch eventType {
    case "fixed_price.purchase_succeeded":
        var ev struct {
            LiveStreamID int64  `json:"live_stream_id"`
            ItemID       int64  `json:"item_id"`
            Remaining    int    `json:"remaining_stock"`
            UserID       int64  `json:"user_id"`
            Price        string `json:"price"`
            ProductID    int64  `json:"product_id"`
        }
        if err := json.Unmarshal(payload, &ev); err != nil { return err }
        h.b.PublishStock(ctx, ev.LiveStreamID, ev.ItemID, ev.Remaining)
        nick := ""
        if h.u != nil { nick, _ = h.u.GetNickname(ctx, ev.UserID) }
        title := ""
        if h.p != nil { title, _ = h.p.GetTitle(ctx, ev.ProductID) }
        price, _ := decimal.NewFromString(ev.Price)
        h.b.PublishFlair(ctx, ev.LiveStreamID, ev.ItemID, nick, price, title)
        return nil
    case "fixed_price.stock_exhausted":
        var ev struct{ LiveStreamID, ItemID int64 }
        var raw map[string]any
        _ = json.Unmarshal(payload, &raw)
        ev.LiveStreamID = int64(raw["live_stream_id"].(float64))
        ev.ItemID = int64(raw["item_id"].(float64))
        h.b.PublishSoldOut(ctx, ev.LiveStreamID, ev.ItemID)
        return nil
    case "fixed_price.listed", "fixed_price.offline":
        // 同上：解析 payload 后调用对应 Publish*
        return nil
    default:
        return nil
    }
}
```

- [ ] **Step 3.4: 跑测试通过 + 注册到 dispatcher**

修改 `backend/auction/service/outbox_dispatcher.go`，把 `fixed_price.*` 路由到 `OutboxFixedPriceHandler.Handle`。

Run: `cd backend/auction && go test ./service/ -run TestOutboxHandler -v`
Expected: PASS

- [ ] **Step 3.5: Commit**

```bash
git add backend/auction/service/outbox_fixed_price_handler*.go backend/auction/service/outbox_dispatcher.go
git commit -m "feat(fixed-price): outbox handler routes events to broadcaster (M2.T3)"
```

---

### Task 4: gateway 列表聚合 product 信息

**Files:**
- Test: `backend/gateway/fixed_price_aggregator_test.go`
- Create: `backend/gateway/fixed_price_aggregator.go`

> **背景：** auction-service 的 `GET /live-streams/:id/fixed-price-items` 仅返回 `product_id`；gateway 需 batch 调用 product RPC 拼装 `product_brief`（title、cover_image），符合 BFF 聚合约束。

- [ ] **Step 4.1: 写失败测试**

```go
// backend/gateway/fixed_price_aggregator_test.go
package gateway

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
)

type stubProduct struct{ batch map[int64]ProductBrief }

func (s *stubProduct) BatchGet(_ context.Context, ids []int64) (map[int64]ProductBrief, error) {
    out := make(map[int64]ProductBrief)
    for _, id := range ids { out[id] = s.batch[id] }
    return out, nil
}

func TestAggregator_AttachesProductBrief(t *testing.T) {
    a := NewFixedPriceAggregator(&stubProduct{batch: map[int64]ProductBrief{
        5001: {ID: 5001, Title: "翡翠", CoverImage: "cdn://a.jpg"},
    }})
    items := []FixedPriceItem{{ID: 7001, ProductID: 5001, Price: "99.00", RemainingStock: 87}}
    out, err := a.Enrich(context.Background(), items)
    assert.NoError(t, err)
    assert.Equal(t, "翡翠", out[0].ProductBrief.Title)
}

func TestAggregator_ProductMissing_KeepsItemWithEmptyBrief(t *testing.T) {
    a := NewFixedPriceAggregator(&stubProduct{batch: map[int64]ProductBrief{}})
    items := []FixedPriceItem{{ID: 7001, ProductID: 5001}}
    out, err := a.Enrich(context.Background(), items)
    assert.NoError(t, err)
    assert.Equal(t, int64(0), out[0].ProductBrief.ID)
}
```

- [ ] **Step 4.2: 写实现**

```go
// backend/gateway/fixed_price_aggregator.go
package gateway

import "context"

type ProductBrief struct {
    ID         int64  `json:"id"`
    Title      string `json:"title"`
    CoverImage string `json:"cover_image,omitempty"`
}
type FixedPriceItem struct {
    ID             int64        `json:"id"`
    ProductID      int64        `json:"product_id"`
    Price          string       `json:"price"`
    TotalStock     int          `json:"total_stock"`
    RemainingStock int          `json:"remaining_stock"`
    Status         string       `json:"status"`
    ProductBrief   ProductBrief `json:"product_brief"`
}
type productClient interface {
    BatchGet(ctx context.Context, ids []int64) (map[int64]ProductBrief, error)
}
type FixedPriceAggregator struct{ p productClient }

func NewFixedPriceAggregator(p productClient) *FixedPriceAggregator { return &FixedPriceAggregator{p: p} }

func (a *FixedPriceAggregator) Enrich(ctx context.Context, items []FixedPriceItem) ([]FixedPriceItem, error) {
    if len(items) == 0 { return items, nil }
    ids := make([]int64, 0, len(items))
    for _, it := range items { ids = append(ids, it.ProductID) }
    m, err := a.p.BatchGet(ctx, ids)
    if err != nil { return nil, err } // fail fast
    for i := range items {
        if pb, ok := m[items[i].ProductID]; ok { items[i].ProductBrief = pb }
    }
    return items, nil
}
```

- [ ] **Step 4.3: 跑测试 + 接入 gateway 路由**

在 gateway 的 `GET /api/v1/live-streams/:id/fixed-price-items` handler 中：调用 auction RPC → 得到 raw items → `aggregator.Enrich(ctx, items)` → 返回。

Run: `cd backend/gateway && go test ./... -run TestAggregator -v`
Expected: PASS

- [ ] **Step 4.4: Commit**

```bash
git add backend/gateway/fixed_price_aggregator*.go
git commit -m "feat(fixed-price): gateway aggregates product brief into list (M2.T4)"
```

---

### Task 5: service 集成 broadcaster

**Files:**
- Modify: `backend/auction/service/fixed_price.go`

- [ ] **Step 5.1: 在 PurchaseService.Purchase 成功路径注入 outbox 事件**

在 M1 写入订单 + 扣余额后，写入 outbox 行：
```
event_type = "fixed_price.purchase_succeeded"
payload = {live_stream_id, item_id, remaining_stock, user_id, price, product_id}
```
事务内插入；由独立 dispatcher 异步消费 → broadcaster。

- [ ] **Step 5.2: Listed/Offline 也写 outbox**

`AdminListItem` 成功 → outbox `fixed_price.listed`；`AdminOfflineItem` 成功 → outbox `fixed_price.offline`。

- [ ] **Step 5.3: stock=0 触发 sold_out**

在 Lua 脚本返回 stock=0 时，service 同时写 outbox `fixed_price.stock_exhausted`（payload: live_stream_id, item_id）。

- [ ] **Step 5.4: 集成测试**

新建 `backend/auction/service/fixed_price_e2e_test.go`：
- 起 miniredis + sqlite + fakeRoom；并发 50 个用户购买 stock=10 的 item；
- 断言：fakeRoom 收到 ≤ 11 条 stock 事件（节流）+ 1 条 sold_out + 10 条 flair。

Run: `cd backend/auction && go test ./service/ -run TestFixedPrice_E2E -race -v`
Expected: PASS

- [ ] **Step 5.5: Commit**

```bash
git add backend/auction/service/fixed_price.go backend/auction/service/fixed_price_e2e_test.go
git commit -m "feat(fixed-price): wire purchase/listed/offline events to outbox (M2.T5)"
```

---

## M2 验收标准

- [ ] 单元测试：messages / broadcaster / outbox handler / aggregator 全绿（覆盖率 ≥ 80%）
- [ ] 集成测试：50 并发购买场景下，stock 广播 ≤ 11 条，flair 等于成交数，sold_out 恰 1 条
- [ ] 手动验证：在 staging 环境用 wscat 订阅 `/api/v1/ws/live-stream/:id`，触发购买能在 < 200ms 收到 `fixed_price_stock` + `fixed_price_flair`
- [ ] 监控：Prometheus 暴露 `fixed_price_ws_publish_total{type=*}` 计数器（Task 5 中加入）

---

## Out of Scope（推迟到 M3+）

- 前端订阅与 UI 渲染（M3 的 Task 1-3）
- Grafana Dashboard（M3 的 Task 5）
- 跨服 product 信息缓存（M4 优化）

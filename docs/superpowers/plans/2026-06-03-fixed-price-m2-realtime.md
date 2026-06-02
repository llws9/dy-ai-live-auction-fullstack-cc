# A5 一口价秒杀 M2 实时同步层 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 M1 一口价后端交易链路上增加后端 WebSocket 实时事件广播，让直播间用户实时收到上架、库存、售罄、下架和购买飘屏事件。

**Architecture:** 复用现有 `websocket.Hub.BroadcastToRoom(liveStreamID, msg)`，不新建 Room、不引入 Redis Pub/Sub/Outbox。`FixedPriceService` 只依赖窄接口 `FixedPriceBroadcaster`，生产适配器把业务事件转换成 WebSocket 消息；`fixed_price_stock` 在适配器内按 `item_id` 做 1 秒合并节流。

**Tech Stack:** Go 1.24+, Hertz, GORM, go-redis, `shopspring/decimal`, existing `auction-service/websocket`, `testify`, TDD, `go test -race`。

---

## Scope Check

本 plan 只覆盖 `docs/superpowers/specs/2026-06-03-fixed-price-m2-realtime-design.md` 的后端实时事件范围：

- 做：WebSocket 消息契约、生产 broadcaster、service 成功路径广播、失败路径不广播、节流、main.go 装配、测试。
- 不做：H5 消费、列表接口聚合、跨实例 Pub/Sub、MQ/Outbox、昵称/商品标题跨服务聚合。

---

## File Structure

| File | Action | Responsibility |
| --- | --- | --- |
| `backend/auction/websocket/message.go` | Modify | 新增 5 个 fixed-price message type、payload struct、构造器。金额字段必须为字符串。 |
| `backend/auction/websocket/fixed_price_message_test.go` | Create | 验证消息 type 与 JSON payload 契约。 |
| `backend/auction/service/fixed_price_broadcaster.go` | Create | 定义 `FixedPriceBroadcaster`、`noopFixedPriceBroadcaster`、`FixedPriceWSBroadcaster`、stock 节流逻辑。 |
| `backend/auction/service/fixed_price_broadcaster_test.go` | Create | 验证 broadcaster 的房间广播、stock 合并节流、stock(0) flush、sold_out 顺序。 |
| `backend/auction/service/fixed_price.go` | Modify | `FixedPriceService` 注入 broadcaster，并在 `ListItem`/`Purchase`/`Offline` 成功路径触发事件。 |
| `backend/auction/service/fixed_price_testutil_test.go` | Modify | 测试构造器新增 broadcaster 注入辅助，保持既有测试最小改动。 |
| `backend/auction/service/fixed_price_failfast_test.go` | Modify | `NewFixedPriceService` 调用签名同步更新。 |
| `backend/auction/service/fixed_price_test.go` | Modify | 少量直接构造器调用签名同步更新。 |
| `backend/auction/service/fixed_price_realtime_test.go` | Create | 验证 service 层广播触发和失败路径不广播。 |
| `backend/auction/main.go` | Modify | 用现有 `hub` 创建 `FixedPriceWSBroadcaster` 并注入 fixed-price service。 |
| `docs/superpowers/sdd/runs/2026-06-03-fixed-price-m2-realtime-state.md` | Create | SDD 状态文件，作为执行 SSOT。 |

---

## Task 1: WebSocket Message Contract

**Files:**
- Modify: `backend/auction/websocket/message.go`
- Create: `backend/auction/websocket/fixed_price_message_test.go`

- [ ] **Step 1: Write failing message contract tests**

Create `backend/auction/websocket/fixed_price_message_test.go`:

```go
package websocket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixedPriceListedMessage_JSONContract(t *testing.T) {
	msg := NewFixedPriceListedMessage(&FixedPriceListedData{
		ItemID: 7001, LiveStreamID: 1001, ProductID: 5001,
		Price: "99.00", TotalStock: 100, RemainingStock: 100, Status: "on_sale",
	})

	raw, err := json.Marshal(msg)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "fixed_price_listed", got["type"])
	data := got["data"].(map[string]any)
	assert.Equal(t, float64(7001), data["item_id"])
	assert.Equal(t, float64(1001), data["live_stream_id"])
	assert.Equal(t, float64(5001), data["product_id"])
	assert.Equal(t, "99.00", data["price"])
	assert.Equal(t, float64(100), data["total_stock"])
	assert.Equal(t, float64(100), data["remaining_stock"])
	assert.Equal(t, "on_sale", data["status"])
}

func TestFixedPriceStockMessage_JSONContract(t *testing.T) {
	msg := NewFixedPriceStockMessage(7001, 87)
	raw, err := json.Marshal(msg)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "fixed_price_stock", got["type"])
	data := got["data"].(map[string]any)
	assert.Equal(t, float64(7001), data["item_id"])
	assert.Equal(t, float64(87), data["remaining_stock"])
}

func TestFixedPriceSoldOutMessage_JSONContract(t *testing.T) {
	msg := NewFixedPriceSoldOutMessage(7001)
	raw, err := json.Marshal(msg)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "fixed_price_sold_out", got["type"])
	data := got["data"].(map[string]any)
	assert.Equal(t, float64(7001), data["item_id"])
}

func TestFixedPriceOfflineMessage_JSONContract(t *testing.T) {
	msg := NewFixedPriceOfflineMessage(7001)
	raw, err := json.Marshal(msg)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "fixed_price_offline", got["type"])
	data := got["data"].(map[string]any)
	assert.Equal(t, float64(7001), data["item_id"])
}

func TestFixedPriceFlairMessage_JSONContract(t *testing.T) {
	msg := NewFixedPriceFlairMessage(&FixedPriceFlairData{ItemID: 7001, BuyerID: 42, Price: "99.00"})
	raw, err := json.Marshal(msg)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "fixed_price_flair", got["type"])
	data := got["data"].(map[string]any)
	assert.Equal(t, float64(7001), data["item_id"])
	assert.Equal(t, float64(42), data["buyer_id"])
	assert.Equal(t, "99.00", data["price"])
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd backend/auction && go test ./websocket/ -run TestFixedPrice -v
```

Expected: FAIL with errors like `undefined: NewFixedPriceListedMessage` and `undefined: FixedPriceListedData`.

- [ ] **Step 3: Add message types and constructors**

Modify `backend/auction/websocket/message.go`. Add constants after sky-lamp constants:

```go
	// 一口价秒杀相关消息类型
	MessageTypeFixedPriceListed  MessageType = "fixed_price_listed"
	MessageTypeFixedPriceStock   MessageType = "fixed_price_stock"
	MessageTypeFixedPriceSoldOut MessageType = "fixed_price_sold_out"
	MessageTypeFixedPriceOffline MessageType = "fixed_price_offline"
	MessageTypeFixedPriceFlair   MessageType = "fixed_price_flair"
```

Add data structs after `SkyLampStoppedData`:

```go
// FixedPriceListedData 一口价上架通知。
type FixedPriceListedData struct {
	ItemID         int64  `json:"item_id"`
	LiveStreamID   int64  `json:"live_stream_id"`
	ProductID      int64  `json:"product_id"`
	Price          string `json:"price"`
	TotalStock     int    `json:"total_stock"`
	RemainingStock int    `json:"remaining_stock"`
	Status         string `json:"status"`
}

// FixedPriceStockData 一口价库存变更通知。
type FixedPriceStockData struct {
	ItemID         int64 `json:"item_id"`
	RemainingStock int   `json:"remaining_stock"`
}

// FixedPriceSoldOutData 一口价售罄通知。
type FixedPriceSoldOutData struct {
	ItemID int64 `json:"item_id"`
}

// FixedPriceOfflineData 一口价下架通知。
type FixedPriceOfflineData struct {
	ItemID int64 `json:"item_id"`
}

// FixedPriceFlairData 一口价购买飘屏通知。
type FixedPriceFlairData struct {
	ItemID  int64  `json:"item_id"`
	BuyerID int64  `json:"buyer_id"`
	Price   string `json:"price"`
}
```

Add constructors near other `New...Message` helpers:

```go
func NewFixedPriceListedMessage(data *FixedPriceListedData) *Message {
	return NewMessage(MessageTypeFixedPriceListed, data)
}

func NewFixedPriceStockMessage(itemID int64, remainingStock int) *Message {
	return NewMessage(MessageTypeFixedPriceStock, &FixedPriceStockData{ItemID: itemID, RemainingStock: remainingStock})
}

func NewFixedPriceSoldOutMessage(itemID int64) *Message {
	return NewMessage(MessageTypeFixedPriceSoldOut, &FixedPriceSoldOutData{ItemID: itemID})
}

func NewFixedPriceOfflineMessage(itemID int64) *Message {
	return NewMessage(MessageTypeFixedPriceOffline, &FixedPriceOfflineData{ItemID: itemID})
}

func NewFixedPriceFlairMessage(data *FixedPriceFlairData) *Message {
	return NewMessage(MessageTypeFixedPriceFlair, data)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
cd backend/auction && go test ./websocket/ -run TestFixedPrice -v
```

Expected: PASS for all 5 message contract tests.

- [ ] **Step 5: Commit**

```bash
git add backend/auction/websocket/message.go backend/auction/websocket/fixed_price_message_test.go
git commit -m "feat(fixed-price): add WebSocket message contracts (M2.T1)"
```

---

## Task 2: FixedPrice Broadcaster Adapter + Stock Throttle

**Files:**
- Create: `backend/auction/service/fixed_price_broadcaster.go`
- Create: `backend/auction/service/fixed_price_broadcaster_test.go`

- [ ] **Step 1: Write failing broadcaster tests**

Create `backend/auction/service/fixed_price_broadcaster_test.go`:

```go
package service

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"auction-service/model"
	"auction-service/websocket"
)

func recvMsg(t *testing.T, ch <-chan *websocket.Message) *websocket.Message {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected websocket message")
		return nil
	}
}

func assertNoMsg(t *testing.T, ch <-chan *websocket.Message) {
	t.Helper()
	select {
	case msg := <-ch:
		t.Fatalf("expected no message, got %s", msg.Type)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestFixedPriceWSBroadcaster_BroadcastsImmediateEvents(t *testing.T) {
	hub := websocket.NewHub()
	go hub.Run()
	defer hub.Stop()

	client := &websocket.Client{ID: "c1", AuctionID: 1001, UserID: 42, Send: make(chan *websocket.Message, 16)}
	hub.Register <- client
	t.Cleanup(func() { hub.Unregister <- client })
	time.Sleep(20 * time.Millisecond)

	b := NewFixedPriceWSBroadcaster(hub, newFakeClock())
	ctx := context.Background()

	b.Listed(ctx, &model.FixedPriceItem{
		ID: 7001, LiveStreamID: 1001, ProductID: 5001,
		Price: decimal.NewFromInt(99), TotalStock: 10, RemainingStock: 10, Status: model.FixedPriceStatusOnSale,
	})
	assert.Equal(t, websocket.MessageTypeFixedPriceListed, recvMsg(t, client.Send).Type)

	b.Flair(ctx, 1001, 7001, 42, decimal.NewFromInt(99))
	assert.Equal(t, websocket.MessageTypeFixedPriceFlair, recvMsg(t, client.Send).Type)

	b.Offline(ctx, 1001, 7001)
	assert.Equal(t, websocket.MessageTypeFixedPriceOffline, recvMsg(t, client.Send).Type)

	b.SoldOut(ctx, 1001, 7001)
	assert.Equal(t, websocket.MessageTypeFixedPriceSoldOut, recvMsg(t, client.Send).Type)
}

func TestFixedPriceWSBroadcaster_StockThrottleMergesLatest(t *testing.T) {
	hub := websocket.NewHub()
	go hub.Run()
	defer hub.Stop()

	client := &websocket.Client{ID: "c1", AuctionID: 1001, UserID: 42, Send: make(chan *websocket.Message, 16)}
	hub.Register <- client
	t.Cleanup(func() { hub.Unregister <- client })
	time.Sleep(20 * time.Millisecond)

	clk := newFakeClock()
	b := NewFixedPriceWSBroadcaster(hub, clk)
	ctx := context.Background()

	b.StockChanged(ctx, 1001, 7001, 9)
	b.StockChanged(ctx, 1001, 7001, 8)
	assertNoMsg(t, client.Send)

	clk.Advance(time.Second)
	msg := recvMsg(t, client.Send)
	require.Equal(t, websocket.MessageTypeFixedPriceStock, msg.Type)
	data := msg.Data.(*websocket.FixedPriceStockData)
	assert.Equal(t, int64(7001), data.ItemID)
	assert.Equal(t, 8, data.RemainingStock)
}

func TestFixedPriceWSBroadcaster_StockZeroFlushesBeforeSoldOut(t *testing.T) {
	hub := websocket.NewHub()
	go hub.Run()
	defer hub.Stop()

	client := &websocket.Client{ID: "c1", AuctionID: 1001, UserID: 42, Send: make(chan *websocket.Message, 16)}
	hub.Register <- client
	t.Cleanup(func() { hub.Unregister <- client })
	time.Sleep(20 * time.Millisecond)

	clk := newFakeClock()
	b := NewFixedPriceWSBroadcaster(hub, clk)
	ctx := context.Background()

	b.StockChanged(ctx, 1001, 7001, 0)
	b.SoldOut(ctx, 1001, 7001)

	first := recvMsg(t, client.Send)
	second := recvMsg(t, client.Send)
	assert.Equal(t, websocket.MessageTypeFixedPriceStock, first.Type)
	assert.Equal(t, websocket.MessageTypeFixedPriceSoldOut, second.Type)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd backend/auction && go test ./service/ -run TestFixedPriceWSBroadcaster -v
```

Expected: FAIL with `undefined: NewFixedPriceWSBroadcaster`.

- [ ] **Step 3: Implement broadcaster and throttle**

Create `backend/auction/service/fixed_price_broadcaster.go`:

```go
package service

import (
	"context"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"auction-service/model"
	"auction-service/websocket"
)

const fixedPriceStockThrottleWindow = time.Second

// FixedPriceBroadcaster hides WebSocket details from FixedPriceService.
type FixedPriceBroadcaster interface {
	Listed(ctx context.Context, item *model.FixedPriceItem)
	StockChanged(ctx context.Context, liveStreamID, itemID int64, remaining int)
	SoldOut(ctx context.Context, liveStreamID, itemID int64)
	Offline(ctx context.Context, liveStreamID, itemID int64)
	Flair(ctx context.Context, liveStreamID, itemID, buyerID int64, price decimal.Decimal)
}

type noopFixedPriceBroadcaster struct{}

func (noopFixedPriceBroadcaster) Listed(context.Context, *model.FixedPriceItem) {}
func (noopFixedPriceBroadcaster) StockChanged(context.Context, int64, int64, int) {}
func (noopFixedPriceBroadcaster) SoldOut(context.Context, int64, int64) {}
func (noopFixedPriceBroadcaster) Offline(context.Context, int64, int64) {}
func (noopFixedPriceBroadcaster) Flair(context.Context, int64, int64, int64, decimal.Decimal) {}

type fixedPriceStockPending struct {
	liveStreamID int64
	itemID       int64
	remaining    int
	active       bool
}

// FixedPriceWSBroadcaster converts fixed-price domain events into WebSocket room messages.
type FixedPriceWSBroadcaster struct {
	hub *websocket.Hub
	clk Clock

	mu      sync.Mutex
	pending map[int64]*fixedPriceStockPending
}

func NewFixedPriceWSBroadcaster(hub *websocket.Hub, clk Clock) *FixedPriceWSBroadcaster {
	if clk == nil {
		clk = realClock{}
	}
	return &FixedPriceWSBroadcaster{
		hub:     hub,
		clk:     clk,
		pending: make(map[int64]*fixedPriceStockPending),
	}
}

func (b *FixedPriceWSBroadcaster) Listed(_ context.Context, item *model.FixedPriceItem) {
	if b == nil || b.hub == nil || item == nil {
		return
	}
	b.hub.BroadcastToRoom(item.LiveStreamID, websocket.NewFixedPriceListedMessage(&websocket.FixedPriceListedData{
		ItemID: item.ID, LiveStreamID: item.LiveStreamID, ProductID: item.ProductID,
		Price: item.Price.StringFixed(2), TotalStock: item.TotalStock,
		RemainingStock: item.RemainingStock, Status: fpRealtimeStatusString(item.Status),
	}))
}

func (b *FixedPriceWSBroadcaster) StockChanged(_ context.Context, liveStreamID, itemID int64, remaining int) {
	if b == nil || b.hub == nil {
		return
	}
	if remaining == 0 {
		b.flushStock(liveStreamID, itemID, remaining)
		return
	}

	b.mu.Lock()
	p := b.pending[itemID]
	if p == nil {
		p = &fixedPriceStockPending{active: true}
		b.pending[itemID] = p
		b.clk.AfterFunc(fixedPriceStockThrottleWindow, func() { b.flushPendingStock(itemID) })
	}
	p.liveStreamID = liveStreamID
	p.itemID = itemID
	p.remaining = remaining
	b.mu.Unlock()
}

func (b *FixedPriceWSBroadcaster) SoldOut(_ context.Context, liveStreamID, itemID int64) {
	if b == nil || b.hub == nil {
		return
	}
	b.hub.BroadcastToRoom(liveStreamID, websocket.NewFixedPriceSoldOutMessage(itemID))
}

func (b *FixedPriceWSBroadcaster) Offline(_ context.Context, liveStreamID, itemID int64) {
	if b == nil || b.hub == nil {
		return
	}
	b.hub.BroadcastToRoom(liveStreamID, websocket.NewFixedPriceOfflineMessage(itemID))
}

func (b *FixedPriceWSBroadcaster) Flair(_ context.Context, liveStreamID, itemID, buyerID int64, price decimal.Decimal) {
	if b == nil || b.hub == nil {
		return
	}
	b.hub.BroadcastToRoom(liveStreamID, websocket.NewFixedPriceFlairMessage(&websocket.FixedPriceFlairData{
		ItemID: itemID, BuyerID: buyerID, Price: price.StringFixed(2),
	}))
}

func (b *FixedPriceWSBroadcaster) flushPendingStock(itemID int64) {
	b.mu.Lock()
	p := b.pending[itemID]
	if p == nil {
		b.mu.Unlock()
		return
	}
	delete(b.pending, itemID)
	liveStreamID, remaining := p.liveStreamID, p.remaining
	b.mu.Unlock()
	b.flushStock(liveStreamID, itemID, remaining)
}

func (b *FixedPriceWSBroadcaster) flushStock(liveStreamID, itemID int64, remaining int) {
	b.mu.Lock()
	delete(b.pending, itemID)
	b.mu.Unlock()
	b.hub.BroadcastToRoom(liveStreamID, websocket.NewFixedPriceStockMessage(itemID, remaining))
}

func fpRealtimeStatusString(s model.FixedPriceStatus) string {
	switch s {
	case model.FixedPriceStatusOnSale:
		return "on_sale"
	case model.FixedPriceStatusSoldOut:
		return "sold_out"
	case model.FixedPriceStatusOffline:
		return "offline"
	default:
		return "unknown"
	}
}
```

- [ ] **Step 4: Run broadcaster tests**

Run:

```bash
cd backend/auction && go test ./service/ -run TestFixedPriceWSBroadcaster -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/auction/service/fixed_price_broadcaster.go backend/auction/service/fixed_price_broadcaster_test.go
git commit -m "feat(fixed-price): add WebSocket broadcaster with stock throttle (M2.T2)"
```

---

## Task 3: FixedPriceService Realtime Hooks

**Files:**
- Modify: `backend/auction/service/fixed_price.go`
- Modify: `backend/auction/service/fixed_price_testutil_test.go`
- Modify: `backend/auction/service/fixed_price_test.go`
- Modify: `backend/auction/service/fixed_price_failfast_test.go`
- Create: `backend/auction/service/fixed_price_realtime_test.go`

- [ ] **Step 1: Write failing service realtime tests**

Create `backend/auction/service/fixed_price_realtime_test.go`:

```go
package service

import (
	"context"
	"sync"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"auction-service/model"
)

type fixedPriceEvent struct {
	typeName     string
	liveStreamID int64
	itemID       int64
	buyerID      int64
	remaining    int
	price        string
}

type fakeFixedPriceBroadcaster struct {
	mu     sync.Mutex
	events []fixedPriceEvent
}

func (f *fakeFixedPriceBroadcaster) Listed(_ context.Context, item *model.FixedPriceItem) {
	f.add(fixedPriceEvent{typeName: "listed", liveStreamID: item.LiveStreamID, itemID: item.ID, remaining: item.RemainingStock, price: item.Price.StringFixed(2)})
}
func (f *fakeFixedPriceBroadcaster) StockChanged(_ context.Context, liveStreamID, itemID int64, remaining int) {
	f.add(fixedPriceEvent{typeName: "stock", liveStreamID: liveStreamID, itemID: itemID, remaining: remaining})
}
func (f *fakeFixedPriceBroadcaster) SoldOut(_ context.Context, liveStreamID, itemID int64) {
	f.add(fixedPriceEvent{typeName: "sold_out", liveStreamID: liveStreamID, itemID: itemID})
}
func (f *fakeFixedPriceBroadcaster) Offline(_ context.Context, liveStreamID, itemID int64) {
	f.add(fixedPriceEvent{typeName: "offline", liveStreamID: liveStreamID, itemID: itemID})
}
func (f *fakeFixedPriceBroadcaster) Flair(_ context.Context, liveStreamID, itemID, buyerID int64, price decimal.Decimal) {
	f.add(fixedPriceEvent{typeName: "flair", liveStreamID: liveStreamID, itemID: itemID, buyerID: buyerID, price: price.StringFixed(2)})
}
func (f *fakeFixedPriceBroadcaster) add(e fixedPriceEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
}
func (f *fakeFixedPriceBroadcaster) snapshot() []fixedPriceEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]fixedPriceEvent, len(f.events))
	copy(out, f.events)
	return out
}

func setupFixedPriceServiceWithBroadcaster(t *testing.T, b FixedPriceBroadcaster) *FixedPriceService {
	t.Helper()
	db := setupServiceDB(t)
	rdb := setupTestRedis(t)
	return NewFixedPriceService(
		db,
		newItemDAO(db), newPurchaseDAO(db), newBalanceDAO(db),
		NewStockGuard(rdb), NewIdemStore(rdb),
		&fakeStreamOwner{owners: nil},
		&fakeProductChecker{},
		nil,
		b,
	)
}

func TestFixedPriceServiceRealtime_ListItemBroadcastsListed(t *testing.T) {
	b := &fakeFixedPriceBroadcaster{}
	svc := setupFixedPriceServiceWithBroadcaster(t, b)

	item, err := svc.ListItem(context.Background(), ListItemReq{
		LiveStreamID: 1001, ProductID: 5001, CreatorID: 100,
		Price: decimal.NewFromInt(99), TotalStock: 10, MaxPerUser: 1,
	})
	require.NoError(t, err)

	events := b.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, "listed", events[0].typeName)
	assert.Equal(t, item.ID, events[0].itemID)
	assert.Equal(t, int64(1001), events[0].liveStreamID)
	assert.Equal(t, 10, events[0].remaining)
	assert.Equal(t, "99.00", events[0].price)
}

func TestFixedPriceServiceRealtime_PurchaseBroadcastsStockAndFlair(t *testing.T) {
	b := &fakeFixedPriceBroadcaster{}
	svc := setupFixedPriceServiceWithBroadcaster(t, b)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	setBalance(t, svc, 100, decimal.NewFromInt(1000))
	b.events = nil

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	require.NoError(t, err)

	events := b.snapshot()
	require.Len(t, events, 2)
	assert.Equal(t, "stock", events[0].typeName)
	assert.Equal(t, 4, events[0].remaining)
	assert.Equal(t, "flair", events[1].typeName)
	assert.Equal(t, int64(100), events[1].buyerID)
	assert.Equal(t, "99.00", events[1].price)
}

func TestFixedPriceServiceRealtime_PurchaseReplayDoesNotBroadcast(t *testing.T) {
	b := &fakeFixedPriceBroadcaster{}
	svc := setupFixedPriceServiceWithBroadcaster(t, b)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	setBalance(t, svc, 100, decimal.NewFromInt(1000))
	key := "550e8400-e29b-41d4-a716-446655440099"

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: key})
	require.NoError(t, err)
	b.events = nil

	res, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: key})
	require.NoError(t, err)
	require.True(t, res.Replayed)
	assert.Empty(t, b.snapshot())
}

func TestFixedPriceServiceRealtime_LastUnitBroadcastsStockThenSoldOut(t *testing.T) {
	b := &fakeFixedPriceBroadcaster{}
	svc := setupFixedPriceServiceWithBroadcaster(t, b)
	ctx := context.Background()
	item := setupItem(t, svc, 1, decimal.NewFromInt(99))
	setBalance(t, svc, 100, decimal.NewFromInt(1000))
	b.events = nil

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	require.NoError(t, err)

	events := b.snapshot()
	require.Len(t, events, 3)
	assert.Equal(t, "stock", events[0].typeName)
	assert.Equal(t, 0, events[0].remaining)
	assert.Equal(t, "flair", events[1].typeName)
	assert.Equal(t, "sold_out", events[2].typeName)
}

func TestFixedPriceServiceRealtime_OfflineBroadcastsOffline(t *testing.T) {
	b := &fakeFixedPriceBroadcaster{}
	svc := setupFixedPriceServiceWithBroadcaster(t, b)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	b.events = nil

	require.NoError(t, svc.Offline(ctx, item.ID, item.CreatorID))

	events := b.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, "offline", events[0].typeName)
	assert.Equal(t, item.ID, events[0].itemID)
	assert.Equal(t, int64(1001), events[0].liveStreamID)
}

func TestFixedPriceServiceRealtime_FailurePathsDoNotBroadcast(t *testing.T) {
	b := &fakeFixedPriceBroadcaster{}
	svc := setupFixedPriceServiceWithBroadcaster(t, b)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	b.events = nil

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	assert.ErrorIs(t, err, ErrInsufficient)

	err = svc.Offline(ctx, item.ID, 9999)
	assert.ErrorIs(t, err, ErrNotStreamOwner)

	assert.Empty(t, b.snapshot())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd backend/auction && go test ./service/ -run TestFixedPriceServiceRealtime -v
```

Expected: FAIL with constructor signature mismatch or missing `FixedPriceBroadcaster` wiring.

- [ ] **Step 3: Update `FixedPriceService` constructor and struct**

Modify `backend/auction/service/fixed_price.go`.

Add field:

```go
	broadcaster FixedPriceBroadcaster
```

Change constructor signature:

```go
func NewFixedPriceService(
	db *gorm.DB,
	items *dao.FixedPriceItemDAO,
	purchases *dao.FixedPricePurchaseDAO,
	balance BalanceDeducter,
	stock *StockGuard,
	idem *IdemStore,
	streams StreamOwnerChecker,
	products ProductChecker,
	clk Clock,
	broadcaster FixedPriceBroadcaster,
) *FixedPriceService {
	if clk == nil {
		clk = realClock{}
	}
	if broadcaster == nil {
		broadcaster = noopFixedPriceBroadcaster{}
	}
	return &FixedPriceService{
		db:          db,
		items:       items,
		purchases:   purchases,
		balance:     balance,
		stock:       stock,
		idem:        idem,
		streams:     streams,
		products:    products,
		clk:         clk,
		broadcaster: broadcaster,
	}
}
```

- [ ] **Step 4: Add realtime hooks in success paths**

In `ListItem`, after stock init succeeds and before return:

```go
	s.broadcaster.Listed(ctx, item)
	return item, nil
```

In `Purchase`, replace final status/broadcast block with:

```go
	rem, _ := s.stock.Remaining(ctx, r.ItemID)
	s.broadcaster.StockChanged(ctx, item.LiveStreamID, item.ID, rem)
	s.broadcaster.Flair(ctx, item.LiveStreamID, item.ID, r.UserID, item.Price)
	if rem == 0 {
		if err := s.items.UpdateStatus(ctx, r.ItemID, model.FixedPriceStatusSoldOut); err == nil {
			s.broadcaster.SoldOut(ctx, item.LiveStreamID, item.ID)
		}
	}

	return &PurchaseResult{
		PurchaseID: purchase.ID, ItemID: r.ItemID, Price: item.Price,
		RemainingStock: rem, Replayed: false,
	}, nil
```

Keep the existing replay branch unchanged; it must not broadcast.

In `Offline`, after status update succeeds and before `s.scheduleCleanup(itemID)`:

```go
	s.broadcaster.Offline(ctx, item.LiveStreamID, item.ID)
```

- [ ] **Step 5: Update constructor call sites**

Modify all `NewFixedPriceService(...)` call sites by adding `nil` as the last argument unless a test passes a fake broadcaster.

In `backend/auction/service/fixed_price_testutil_test.go`, both helpers end with:

```go
		nil,
		nil,
	)
```

In `backend/auction/service/fixed_price_test.go` direct constructor for missing product ends with:

```go
		nil,
		nil,
	)
```

In `backend/auction/service/fixed_price_failfast_test.go` direct constructor ends with:

```go
		nil,
		nil,
	)
```

- [ ] **Step 6: Run service realtime tests**

Run:

```bash
cd backend/auction && go test ./service/ -run TestFixedPriceServiceRealtime -v
```

Expected: PASS.

- [ ] **Step 7: Run fixed-price service regression**

Run:

```bash
cd backend/auction && go test ./service/ -run 'TestFixedPriceService|TestPurchase|TestOffline|TestFixedPriceWSBroadcaster' -race
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add backend/auction/service/fixed_price.go backend/auction/service/fixed_price_testutil_test.go backend/auction/service/fixed_price_test.go backend/auction/service/fixed_price_failfast_test.go backend/auction/service/fixed_price_realtime_test.go
git commit -m "feat(fixed-price): broadcast realtime events from service (M2.T3)"
```

---

## Task 4: Production Wiring, SDD State, and Full Regression

**Files:**
- Modify: `backend/auction/main.go`
- Create: `docs/superpowers/sdd/runs/2026-06-03-fixed-price-m2-realtime-state.md`

- [ ] **Step 1: Update production wiring in `main.go`**

In `backend/auction/main.go`, after `hub := websocket.NewHub()` and before fixed-price service construction, add:

```go
	fixedPriceBroadcaster := service.NewFixedPriceWSBroadcaster(hub, nil)
```

Update fixed-price service construction by adding `fixedPriceBroadcaster` as the last argument:

```go
	fixedPriceService := service.NewFixedPriceService(
		db,
		fixedPriceItemDAO,
		fixedPricePurchaseDAO,
		userBalanceDAO,
		fixedPriceStock,
		fixedPriceIdem,
		&liveStreamOwnerChecker{client: liveStreamClient},
		&productExistsChecker{client: productClient},
		nil,
		fixedPriceBroadcaster,
	)
```

- [ ] **Step 2: Run build to catch wiring errors**

Run:

```bash
cd backend/auction && go build ./...
```

Expected: PASS with exit code 0.

- [ ] **Step 3: Create SDD state file**

Create `docs/superpowers/sdd/runs/2026-06-03-fixed-price-m2-realtime-state.md`:

```markdown
# SDD Run State - 2026-06-03-fixed-price-m2-realtime

> SSOT for A5 fixed-price M2 后端实时同步层。

## Run Metadata

| Key | Value |
| --- | --- |
| Run ID | `2026-06-03-fixed-price-m2-realtime` |
| Topic | `fixed-price-m2-realtime` |
| Goal | 新增一口价后端 WebSocket 实时事件广播 |
| Mode | `subagent-driven` |
| Branch | `feat/fixed-price-m1` |
| Worktree | `/Users/bytedance/.config/superpowers/worktrees/dy-ai-live-auction-fullstack-cc/feat-fixed-price-m1` |
| Spec | `docs/superpowers/specs/2026-06-03-fixed-price-m2-realtime-design.md` |
| Plan | `docs/superpowers/plans/2026-06-03-fixed-price-m2-realtime.md` |
| Status | `done` |

## Task Matrix

| Task ID | Title | Status | Owner | Depends On | Evidence |
| --- | --- | --- | --- | --- | --- |
| `M2-T1` | WebSocket message contract | `done` | `main-agent` | spec | `go test ./websocket/ -run TestFixedPrice -v` |
| `M2-T2` | Broadcaster + stock throttle | `done` | `main-agent` | M2-T1 | `go test ./service/ -run TestFixedPriceWSBroadcaster -v` |
| `M2-T3` | FixedPriceService realtime hooks | `done` | `main-agent` | M2-T2 | `go test ./service/ -run TestFixedPriceServiceRealtime -v` |
| `M2-T4` | Production wiring + regression | `done` | `main-agent` | M2-T3 | `go test ./... -race` |

## Final Handoff

当前分支/worktree：feat/fixed-price-m1 @ /Users/bytedance/.config/superpowers/worktrees/dy-ai-live-auction-fullstack-cc/feat-fixed-price-m1

**状态**
- M2 后端实时同步层完成。
- 本轮未实现 H5 消费、列表接口、Redis Pub/Sub、Outbox。
```

This state file is created in Task 4 after M2-T1 through M2-T4 finish; if execution mode uses subagents, record the concrete subagent IDs in task records during execution.

- [ ] **Step 4: Run targeted tests**

Run:

```bash
cd backend/auction && go test ./websocket/ ./service/ -run 'TestFixedPrice' -race
```

Expected: PASS.

- [ ] **Step 5: Run full backend auction regression**

Run:

```bash
cd backend/auction && go test ./... -race
```

Expected: PASS. On macOS, linker may print `malformed LC_DYSYMTAB`; treat as noise only if command exit code is 0 and all packages are `ok`.

- [ ] **Step 6: Commit**

```bash
git add backend/auction/main.go docs/superpowers/sdd/runs/2026-06-03-fixed-price-m2-realtime-state.md
git commit -m "feat(fixed-price): wire realtime broadcaster and mark M2 complete"
```

---

## Final Review Checklist

- [ ] `fixed_price_listed` / `stock` / `sold_out` / `offline` / `flair` message contracts tested.
- [ ] `price` is always `StringFixed(2)` string, never float.
- [ ] `FixedPriceService` depends on `FixedPriceBroadcaster`, not `websocket.Hub`.
- [ ] `nil` broadcaster becomes no-op, so older tests and non-WS contexts are safe.
- [ ] Idempotent replay does not broadcast.
- [ ] Failure paths do not broadcast.
- [ ] `fixed_price_stock(0)` is sent before `fixed_price_sold_out`.
- [ ] `go test ./... -race` passes in `backend/auction`.

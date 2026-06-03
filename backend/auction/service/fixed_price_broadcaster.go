package service

import (
	"context"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"auction-service/model"
	"auction-service/pkg/metrics"
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

func (noopFixedPriceBroadcaster) Listed(context.Context, *model.FixedPriceItem)               {}
func (noopFixedPriceBroadcaster) StockChanged(context.Context, int64, int64, int)             {}
func (noopFixedPriceBroadcaster) SoldOut(context.Context, int64, int64)                       {}
func (noopFixedPriceBroadcaster) Offline(context.Context, int64, int64)                       {}
func (noopFixedPriceBroadcaster) Flair(context.Context, int64, int64, int64, decimal.Decimal) {}

type fixedPriceStockPending struct {
	liveStreamID int64
	remaining    int
}

// FixedPriceWSBroadcaster converts fixed-price domain events into WebSocket room messages.
type FixedPriceWSBroadcaster struct {
	hub *websocket.Hub
	clk Clock

	mu      sync.Mutex
	pending map[int64]*fixedPriceStockPending
	metrics *metrics.FixedPriceMetrics
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

func (b *FixedPriceWSBroadcaster) SetMetrics(m *metrics.FixedPriceMetrics) {
	b.metrics = m
}

func (b *FixedPriceWSBroadcaster) Listed(_ context.Context, item *model.FixedPriceItem) {
	if b == nil || b.hub == nil || item == nil {
		return
	}
	b.tryBroadcast(item.LiveStreamID, websocket.NewFixedPriceListedMessage(&websocket.FixedPriceListedData{
		ItemID:         item.ID,
		LiveStreamID:   item.LiveStreamID,
		ProductID:      item.ProductID,
		Price:          item.Price.StringFixed(2),
		TotalStock:     item.TotalStock,
		RemainingStock: item.RemainingStock,
		Status:         fpRealtimeStatusString(item.Status),
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
	if _, ok := b.pending[itemID]; !ok {
		b.pending[itemID] = &fixedPriceStockPending{}
		b.clk.AfterFunc(fixedPriceStockThrottleWindow, func() { b.flushPendingStock(itemID) })
	}
	b.pending[itemID].liveStreamID = liveStreamID
	b.pending[itemID].remaining = remaining
	b.mu.Unlock()
}

func (b *FixedPriceWSBroadcaster) SoldOut(_ context.Context, liveStreamID, itemID int64) {
	if b == nil || b.hub == nil {
		return
	}
	b.tryBroadcast(liveStreamID, websocket.NewFixedPriceSoldOutMessage(itemID))
}

func (b *FixedPriceWSBroadcaster) Offline(_ context.Context, liveStreamID, itemID int64) {
	if b == nil || b.hub == nil {
		return
	}
	b.tryBroadcast(liveStreamID, websocket.NewFixedPriceOfflineMessage(itemID))
}

func (b *FixedPriceWSBroadcaster) Flair(_ context.Context, liveStreamID, itemID, buyerID int64, price decimal.Decimal) {
	if b == nil || b.hub == nil {
		return
	}
	b.tryBroadcast(liveStreamID, websocket.NewFixedPriceFlairMessage(&websocket.FixedPriceFlairData{
		ItemID:  itemID,
		BuyerID: buyerID,
		Price:   price.StringFixed(2),
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
	liveStreamID := p.liveStreamID
	remaining := p.remaining
	b.mu.Unlock()

	b.flushStock(liveStreamID, itemID, remaining)
}

func (b *FixedPriceWSBroadcaster) flushStock(liveStreamID, itemID int64, remaining int) {
	b.mu.Lock()
	delete(b.pending, itemID)
	b.mu.Unlock()

	b.tryBroadcast(liveStreamID, websocket.NewFixedPriceStockMessage(itemID, remaining))
}

func (b *FixedPriceWSBroadcaster) tryBroadcast(liveStreamID int64, msg *websocket.Message) {
	if b.metrics != nil && msg != nil {
		b.metrics.RecordWSPublish(string(msg.Type))
	}
	_ = b.hub.TryBroadcastToRoom(liveStreamID, msg)
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

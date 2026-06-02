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

func newFixedPriceBroadcasterTestClient(t *testing.T, auctionID int64) (*websocket.Hub, *websocket.Client) {
	t.Helper()

	hub := websocket.NewHub()
	go hub.Run()

	client := &websocket.Client{
		ID:        "fixed-price-test-client",
		AuctionID: auctionID,
		UserID:    42,
		Send:      make(chan *websocket.Message, 16),
	}
	hub.Register <- client
	time.Sleep(20 * time.Millisecond)

	return hub, client
}

func recvFixedPriceMsg(t *testing.T, ch <-chan *websocket.Message) *websocket.Message {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected websocket message")
		return nil
	}
}

func assertNoFixedPriceMsg(t *testing.T, ch <-chan *websocket.Message) {
	t.Helper()
	select {
	case msg := <-ch:
		t.Fatalf("expected no message, got %s", msg.Type)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestFixedPriceWSBroadcaster_BroadcastsImmediateEvents(t *testing.T) {
	hub, client := newFixedPriceBroadcasterTestClient(t, 1001)

	b := NewFixedPriceWSBroadcaster(hub, newFakeClock())
	ctx := context.Background()

	b.Listed(ctx, &model.FixedPriceItem{
		ID:             7001,
		LiveStreamID:   1001,
		ProductID:      5001,
		Price:          decimal.NewFromInt(99),
		TotalStock:     10,
		RemainingStock: 10,
		Status:         model.FixedPriceStatusOnSale,
	})
	listed := recvFixedPriceMsg(t, client.Send)
	require.Equal(t, websocket.MessageTypeFixedPriceListed, listed.Type)
	listedData := listed.Data.(*websocket.FixedPriceListedData)
	assert.Equal(t, int64(7001), listedData.ItemID)
	assert.Equal(t, int64(1001), listedData.LiveStreamID)
	assert.Equal(t, int64(5001), listedData.ProductID)
	assert.Equal(t, "99.00", listedData.Price)
	assert.Equal(t, 10, listedData.TotalStock)
	assert.Equal(t, 10, listedData.RemainingStock)
	assert.Equal(t, "on_sale", listedData.Status)

	b.Flair(ctx, 1001, 7001, 42, decimal.NewFromInt(99))
	flair := recvFixedPriceMsg(t, client.Send)
	require.Equal(t, websocket.MessageTypeFixedPriceFlair, flair.Type)
	flairData := flair.Data.(*websocket.FixedPriceFlairData)
	assert.Equal(t, int64(7001), flairData.ItemID)
	assert.Equal(t, int64(42), flairData.BuyerID)
	assert.Equal(t, "99.00", flairData.Price)

	b.Offline(ctx, 1001, 7001)
	assert.Equal(t, websocket.MessageTypeFixedPriceOffline, recvFixedPriceMsg(t, client.Send).Type)

	b.SoldOut(ctx, 1001, 7001)
	assert.Equal(t, websocket.MessageTypeFixedPriceSoldOut, recvFixedPriceMsg(t, client.Send).Type)
}

func TestFixedPriceWSBroadcaster_StockThrottleMergesLatest(t *testing.T) {
	hub, client := newFixedPriceBroadcasterTestClient(t, 1001)

	clk := newFakeClock()
	b := NewFixedPriceWSBroadcaster(hub, clk)
	ctx := context.Background()

	b.StockChanged(ctx, 1001, 7001, 9)
	b.StockChanged(ctx, 1001, 7001, 8)
	assertNoFixedPriceMsg(t, client.Send)

	clk.Advance(time.Second)
	msg := recvFixedPriceMsg(t, client.Send)
	require.Equal(t, websocket.MessageTypeFixedPriceStock, msg.Type)
	data := msg.Data.(*websocket.FixedPriceStockData)
	assert.Equal(t, int64(7001), data.ItemID)
	assert.Equal(t, 8, data.RemainingStock)
}

func TestFixedPriceWSBroadcaster_StockZeroFlushesBeforeSoldOut(t *testing.T) {
	hub, client := newFixedPriceBroadcasterTestClient(t, 1001)

	clk := newFakeClock()
	b := NewFixedPriceWSBroadcaster(hub, clk)
	ctx := context.Background()

	b.StockChanged(ctx, 1001, 7001, 0)
	b.SoldOut(ctx, 1001, 7001)

	first := recvFixedPriceMsg(t, client.Send)
	second := recvFixedPriceMsg(t, client.Send)
	assert.Equal(t, websocket.MessageTypeFixedPriceStock, first.Type)
	assert.Equal(t, websocket.MessageTypeFixedPriceSoldOut, second.Type)
}

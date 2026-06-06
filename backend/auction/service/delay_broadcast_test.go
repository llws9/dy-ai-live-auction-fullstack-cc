package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"auction-service/websocket"
)

func newDelayBroadcastTestClient(t *testing.T, auctionID int64) (*websocket.Hub, *websocket.Client) {
	t.Helper()

	hub := websocket.NewHub()
	go hub.Run()
	t.Cleanup(hub.Stop)

	client := &websocket.Client{
		ID:        "delay-broadcast-test-client",
		AuctionID: auctionID,
		UserID:    42,
		Send:      make(chan *websocket.Message, 16),
	}
	hub.Register <- client
	require.Eventually(t, func() bool {
		return hub.GetClientCount() == 1
	}, 500*time.Millisecond, 10*time.Millisecond)

	return hub, client
}

func recvDelayBroadcastMsg(t *testing.T, ch <-chan *websocket.Message) *websocket.Message {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected websocket message")
		return nil
	}
}

func assertNoDelayBroadcastMsg(t *testing.T, ch <-chan *websocket.Message) {
	t.Helper()
	select {
	case msg := <-ch:
		t.Fatalf("expected no message, got %s", msg.Type)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestDelayBroadcast_BroadcastsDelayTriggeredToAuctionRoom(t *testing.T) {
	hub, client := newDelayBroadcastTestClient(t, 1001)

	svc := &BidService{}
	svc.SetHub(hub)
	endTime := time.UnixMilli(1780761600000)

	svc.broadcastDelayTriggered(1001, 30, endTime, 60, 90)

	msg := recvDelayBroadcastMsg(t, client.Send)
	require.Equal(t, websocket.MessageTypeDelayTriggered, msg.Type)
	data, ok := msg.Data.(*websocket.DelayTriggeredData)
	require.True(t, ok)
	assert.Equal(t, int64(1001), data.AuctionID)
	assert.Equal(t, 30, data.DelayDuration)
	assert.Equal(t, endTime.UnixMilli(), data.NewEndTime)
	assert.Equal(t, 60, data.RemainingDelay)
	assert.Equal(t, 90, data.MaxDelay)
}

func TestDelayBroadcast_NoHubDoesNotPanic(t *testing.T) {
	svc := &BidService{}

	require.NotPanics(t, func() {
		svc.broadcastDelayTriggered(1001, 30, time.UnixMilli(1780761600000), 60, 90)
	})
}

func TestDelayBroadcast_DoesNotBlockWhenHubBroadcastQueueIsFull(t *testing.T) {
	hub := websocket.NewHub()
	svc := &BidService{}
	svc.SetHub(hub)

	for hub.TryBroadcastToRoom(1001, websocket.NewDelayTriggeredMessage(&websocket.DelayTriggeredData{})) {
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.broadcastDelayTriggered(1001, 30, time.UnixMilli(1780761600000), 60, 90)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("delay triggered broadcast blocked when hub broadcast queue was full")
	}
}

func TestDelayBroadcast_DoesNotLeakAcrossAuctionRooms(t *testing.T) {
	hub, client := newDelayBroadcastTestClient(t, 1001)

	svc := &BidService{}
	svc.SetHub(hub)

	svc.broadcastDelayTriggered(1002, 30, time.UnixMilli(1780761600000), 60, 90)

	assertNoDelayBroadcastMsg(t, client.Send)
}

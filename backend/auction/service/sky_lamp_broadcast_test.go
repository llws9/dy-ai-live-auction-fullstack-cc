package service

import (
	"testing"
	"time"

	"auction-service/websocket"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func newSkyLampBroadcastTestClient(t *testing.T, auctionID int64) (*websocket.Hub, *websocket.Client) {
	t.Helper()

	hub := websocket.NewHub()
	go hub.Run()
	t.Cleanup(hub.Stop)

	client := &websocket.Client{
		ID:        "sky-lamp-broadcast-test-client",
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

func recvSkyLampBroadcastMsg(t *testing.T, ch <-chan *websocket.Message) *websocket.Message {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected websocket message")
		return nil
	}
}

func TestSkyLampBroadcast_AutoBidToAuctionRoom(t *testing.T) {
	hub, client := newSkyLampBroadcastTestClient(t, 9901)
	svc := &SkyLampService{}
	svc.SetHub(hub)

	svc.broadcastAutoBid(9901, 9101, 150, 10000, 2)

	msg := recvSkyLampBroadcastMsg(t, client.Send)
	require.Equal(t, websocket.MessageTypeSkyLampAutoBid, msg.Type)
	data, ok := msg.Data.(websocket.SkyLampAutoBidData)
	require.True(t, ok)
	require.Equal(t, int64(9901), data.AuctionID)
	require.Equal(t, int64(9101), data.UserID)
	require.True(t, data.Amount.Equal(decimal.NewFromInt(150)))
	require.True(t, data.RemainingBudget.Equal(decimal.NewFromInt(9850)))
	require.Equal(t, 2, data.AutoBidCount)
}

func TestSkyLampBroadcast_NoHubDoesNotPanic(t *testing.T) {
	svc := &SkyLampService{}

	require.NotPanics(t, func() {
		svc.broadcastAutoBid(9901, 9101, 150, 10000, 1)
	})
}

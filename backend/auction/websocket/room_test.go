package websocket

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoomBroadcast_SkipsClosedClientWithoutPanic(t *testing.T) {
	room := NewRoom(42)
	client := &Client{
		ID:        "closed-client",
		AuctionID: 42,
		Send:      make(chan *Message, 1),
	}
	room.registerClient(client)
	client.Close()

	require.NotPanics(t, func() {
		room.broadcastMessage(NewPongMessage())
	})
}

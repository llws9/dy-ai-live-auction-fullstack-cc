package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	gorilla "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"auction-service/model"
	auctionws "auction-service/websocket"
)

type fakeWSPresenceUserFetcher struct {
	user *model.User
	err  error
}

func (f fakeWSPresenceUserFetcher) GetByID(context.Context, int64) (*model.User, error) {
	return f.user, f.err
}

func TestHandleWebSocket_DeliversHubBroadcasts(t *testing.T) {
	hub := auctionws.NewHub()
	go hub.Run()
	t.Cleanup(hub.Stop)

	h := NewWSHandler()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.HandleWebSocket(hub, 991101, w, r)
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?user_id=991001"
	conn, _, err := gorilla.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
	_, welcome, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Contains(t, string(welcome), `"type":"system"`)

	hub.BroadcastToRoom(991101, auctionws.NewFixedPriceListedMessage(&auctionws.FixedPriceListedData{
		ItemID:         990003,
		LiveStreamID:   991101,
		ProductID:      991201,
		Price:          "77.00",
		TotalStock:     2,
		RemainingStock: 2,
		Status:         "on_sale",
	}))

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Contains(t, string(msg), `"type":"fixed_price_listed"`)
}

func TestHandleWebSocket_DoesNotWriteConnOutsideWritePump(t *testing.T) {
	src, err := os.ReadFile("ws.go")
	require.NoError(t, err)
	assert.NotContains(t, string(src), "conn.WriteMessage",
		"websocket writes must go through client.Send/WritePump to avoid concurrent writers")
}

func TestWSHandlerResolvePresenceProfileUsesUserAvatar(t *testing.T) {
	h := NewWSHandler()
	h.SetUserFetcher(fakeWSPresenceUserFetcher{
		user: &model.User{ID: 42, Name: "数据库用户", Avatar: "https://cdn/u42.png"},
	})

	name, avatar := h.resolvePresenceProfile(context.Background(), 42, "jwt-name")

	assert.Equal(t, "数据库用户", name)
	assert.Equal(t, "https://cdn/u42.png", avatar)
}

func TestWSHandlerResolvePresenceProfileFallsBackOnUserLookupError(t *testing.T) {
	h := NewWSHandler()
	h.SetUserFetcher(fakeWSPresenceUserFetcher{err: errors.New("db down")})

	name, avatar := h.resolvePresenceProfile(context.Background(), 42, "jwt-name")

	assert.Equal(t, "jwt-name", name)
	assert.Equal(t, "", avatar)
}

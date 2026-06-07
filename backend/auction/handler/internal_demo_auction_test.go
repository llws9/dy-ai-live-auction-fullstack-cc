package handler

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/websocket"
)

type fakeDemoLiveRestarter struct {
	ended   []int64
	started []int64
}

func (f *fakeDemoLiveRestarter) EndLive(_ context.Context, liveStreamID int64) error {
	f.ended = append(f.ended, liveStreamID)
	return nil
}

func (f *fakeDemoLiveRestarter) StartLive(_ context.Context, liveStreamID int64) error {
	f.started = append(f.started, liveStreamID)
	return nil
}

func TestInternalDemoAuctionRestartLiveSessionEndsThenStarts(t *testing.T) {
	restarter := &fakeDemoLiveRestarter{}
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	handler := NewInternalDemoAuctionHandler(nil, websocket.NewHub())
	handler.SetLiveRestarter(restarter)
	h.POST("/internal/test/live-streams/:id/restart", handler.RestartLiveSession)

	w := ut.PerformRequest(h.Engine, http.MethodPost, "/internal/test/live-streams/880301/restart", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []int64{880301}, restarter.ended)
	require.Equal(t, []int64{880301}, restarter.started)
}

func TestInternalDemoAuctionShortenUpdatesEndTimeAndBroadcastsTimeSync(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))

	now := time.Now()
	auction := model.Auction{
		ProductID: 1,
		Status:    model.AuctionStatusOngoing,
		StartTime: now.Add(-time.Minute),
		EndTime:   now.Add(time.Hour),
	}
	require.NoError(t, db.Create(&auction).Error)

	hub, client := newInternalDemoAuctionTestClient(t, auction.ID)
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	handler := NewInternalDemoAuctionHandler(dao.NewAuctionDAO(db), hub)
	h.POST("/internal/test/auctions/shorten", handler.Shorten)

	body := []byte(`{"auction_id":` + strconv.FormatInt(auction.ID, 10) + `,"remaining_seconds":10}`)
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/internal/test/auctions/shorten",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)
	var reloaded model.Auction
	require.NoError(t, db.First(&reloaded, auction.ID).Error)
	assert.WithinDuration(t, time.Now().Add(10*time.Second), reloaded.EndTime, 2*time.Second)

	msg := recvInternalDemoAuctionMessage(t, client.Send)
	require.Equal(t, websocket.MessageTypeTimeSync, msg.Type)
	data, ok := msg.Data.(*websocket.TimeSyncData)
	require.True(t, ok)
	assert.Equal(t, auction.ID, data.AuctionID)
	assert.Equal(t, reloaded.EndTime.UnixMilli(), data.EndTime)
}

func TestInternalDemoAuctionShortenRejectsEndedAuction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))

	now := time.Now()
	auction := model.Auction{
		ProductID: 1,
		Status:    model.AuctionStatusEnded,
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(-time.Minute),
	}
	require.NoError(t, db.Create(&auction).Error)

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	handler := NewInternalDemoAuctionHandler(dao.NewAuctionDAO(db), websocket.NewHub())
	h.POST("/internal/test/auctions/shorten", handler.Shorten)

	body := []byte(`{"auction_id":` + strconv.FormatInt(auction.ID, 10) + `,"remaining_seconds":10}`)
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/internal/test/auctions/shorten",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusConflict, w.Code)
	var reloaded model.Auction
	require.NoError(t, db.First(&reloaded, auction.ID).Error)
	assert.Equal(t, auction.EndTime.Unix(), reloaded.EndTime.Unix())
}

func newInternalDemoAuctionTestClient(t *testing.T, auctionID int64) (*websocket.Hub, *websocket.Client) {
	t.Helper()
	hub := websocket.NewHub()
	go hub.Run()
	t.Cleanup(hub.Stop)

	client := &websocket.Client{
		ID:        "internal-demo-auction-test-client",
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

func recvInternalDemoAuctionMessage(t *testing.T, ch <-chan *websocket.Message) *websocket.Message {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected websocket message")
		return nil
	}
}

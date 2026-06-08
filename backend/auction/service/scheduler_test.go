package service

import (
	"context"
	"testing"
	"time"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/websocket"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchedulerDefaultAuctionCheckIntervalKeepsEndAnimationResponsive(t *testing.T) {
	scheduler := NewScheduler(nil)

	require.Equal(t, 200*time.Millisecond, scheduler.checkInterval)
	require.Equal(t, 5*time.Second, scheduler.timeSyncInterval)
}

func TestSchedulerBroadcastsAuctionEndedForExpiredUnsoldAuction(t *testing.T) {
	db := setupServiceDB(t)
	require.NoError(t, db.AutoMigrate(&model.Auction{}, &model.Bid{}, &model.AuctionSettlementTask{}))

	ctx := context.Background()
	auctionDAO := dao.NewAuctionDAO(db)
	auction := &model.Auction{
		ProductID:    1001,
		Status:       model.AuctionStatusOngoing,
		CurrentPrice: decimal.NewFromInt(800),
		StartTime:    time.Now().Add(-time.Hour),
		EndTime:      time.Now().Add(-time.Second),
	}
	require.NoError(t, auctionDAO.Create(ctx, auction))

	hub := websocket.NewHub()
	go hub.Run()
	t.Cleanup(hub.Stop)

	client := &websocket.Client{
		ID:        "scheduler-ended-test-client",
		AuctionID: auction.ID,
		UserID:    42,
		Send:      make(chan *websocket.Message, 16),
	}
	hub.Register <- client
	require.Eventually(t, func() bool {
		return hub.GetClientCount() == 1
	}, 500*time.Millisecond, 10*time.Millisecond)

	auctionService := NewAuctionService(auctionDAO)
	auctionService.SetBidDAO(dao.NewBidDAO(db))
	scheduler := NewScheduler(auctionService)
	scheduler.SetHub(hub)

	require.NoError(t, scheduler.checkAndEndAuctions(ctx))

	msg := receiveSchedulerMessage(t, client.Send)
	require.Equal(t, websocket.MessageTypeAuctionEnded, msg.Type)
	data, ok := msg.Data.(*websocket.AuctionEndedData)
	require.True(t, ok)
	assert.Equal(t, auction.ID, data.AuctionID)
	assert.Equal(t, int64(0), data.WinnerID)
	assert.True(t, decimal.NewFromInt(800).Equal(data.FinalPrice))
}

func receiveSchedulerMessage(t *testing.T, ch <-chan *websocket.Message) *websocket.Message {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected websocket message")
		return nil
	}
}

package service

import (
	"context"
	"time"

	"auction-service/model"
	"auction-service/websocket"
)

// SaveAuctionSyncState writes an authoritative auction snapshot to WebSocket sync state.
func SaveAuctionSyncState(ctx context.Context, stateManager *websocket.StateManager, auction *model.Auction) error {
	if stateManager == nil || auction == nil {
		return nil
	}

	var winnerID int64
	if auction.WinnerID != nil {
		winnerID = *auction.WinnerID
	}

	return stateManager.SaveSyncState(ctx, &websocket.SyncState{
		AuctionID:    auction.ID,
		CurrentPrice: auction.CurrentPrice,
		WinnerID:     winnerID,
		EndTime:      auction.EndTime,
		Status:       int(auction.Status),
		UpdatedAt:    time.Now(),
	})
}

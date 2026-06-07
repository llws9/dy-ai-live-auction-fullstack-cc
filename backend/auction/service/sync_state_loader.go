package service

import (
	"context"
	"time"

	"auction-service/dao"
	"auction-service/websocket"
)

// AuctionSyncStateLoader builds WebSocket sync state from the authoritative auction table.
type AuctionSyncStateLoader struct {
	auctionDAO *dao.AuctionDAO
}

func NewAuctionSyncStateLoader(auctionDAO *dao.AuctionDAO) *AuctionSyncStateLoader {
	return &AuctionSyncStateLoader{auctionDAO: auctionDAO}
}

func (l *AuctionSyncStateLoader) LoadSyncState(ctx context.Context, auctionID int64) (*websocket.SyncState, error) {
	auction, err := l.auctionDAO.GetByID(ctx, auctionID)
	if err != nil {
		return nil, err
	}

	var winnerID int64
	if auction.WinnerID != nil {
		winnerID = *auction.WinnerID
	}

	return &websocket.SyncState{
		AuctionID:    auction.ID,
		CurrentPrice: auction.CurrentPrice,
		WinnerID:     winnerID,
		EndTime:      auction.EndTime,
		Status:       int(auction.Status),
		UpdatedAt:    time.Now(),
	}, nil
}

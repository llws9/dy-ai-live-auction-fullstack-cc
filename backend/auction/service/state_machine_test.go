package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"auction-service/model"
)

func TestStateMachineCanBidRejectsExpiredActiveAuction(t *testing.T) {
	auction := &model.Auction{
		Status:  model.AuctionStatusOngoing,
		EndTime: time.Now().Add(-time.Second),
	}

	require.False(t, NewStateMachine(auction).CanBid(), "已过 end_time 的进行中竞拍不允许继续出价")
}

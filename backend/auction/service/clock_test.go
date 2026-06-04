package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuctionBusinessNowUsesShanghaiLocation(t *testing.T) {
	now := auctionBusinessNow()

	require.Equal(t, auctionBusinessLocation, now.Location())
	_, offset := now.Zone()
	require.Equal(t, 8*60*60, offset)
}

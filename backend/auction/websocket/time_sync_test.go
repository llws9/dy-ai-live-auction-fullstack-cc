package websocket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTimeSyncMessageIncludesAuctionID(t *testing.T) {
	msg := NewTimeSyncMessage(1001, 1780761600000, 1780761660000)

	require.Equal(t, MessageTypeTimeSync, msg.Type)
	data, ok := msg.Data.(*TimeSyncData)
	require.True(t, ok)
	assert.Equal(t, int64(1001), data.AuctionID)
	assert.Equal(t, int64(1780761600000), data.ServerTime)
	assert.Equal(t, int64(1780761660000), data.EndTime)
}

func TestTimeSyncServiceCreateTimeSyncMessageIncludesAuctionID(t *testing.T) {
	svc := NewTimeSyncService()

	msg := svc.CreateTimeSyncMessage(1001, 1780761660000)

	require.Equal(t, MessageTypeTimeSync, msg.Type)
	data, ok := msg.Data.(*TimeSyncData)
	require.True(t, ok)
	assert.Equal(t, int64(1001), data.AuctionID)
	assert.Equal(t, int64(1780761660000), data.EndTime)
	assert.Greater(t, data.ServerTime, int64(0))
}

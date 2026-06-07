package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextByLiveStreamIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/auctions/next-by-live-streams", r.URL.Path)
		_, _ = w.Write([]byte(`{"code":200,"data":{"items":[{"live_stream_id":1,"auction_id":11,"product_id":101,"start_price":"100.00","start_time":"2026-06-08T10:00:00Z"}]}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewAuctionClient(srv.URL, 0)
	got, err := c.NextByLiveStreamIDs(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Contains(t, got, int64(1))
	assert.EqualValues(t, 11, got[1].AuctionID)
	assert.EqualValues(t, 101, got[1].ProductID)
	assert.Equal(t, "100.00", got[1].StartPrice)
}

func TestRecentDealsByLiveStreamIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/auctions/recent-deals-by-live-streams", r.URL.Path)
		_, _ = w.Write([]byte(`{"code":200,"data":{"items":[{"live_stream_id":1,"deals":[{"auction_id":33,"product_id":303,"final_price":"300.00","end_time":"2026-06-08T09:00:00Z"}]}]}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewAuctionClient(srv.URL, 0)
	got, err := c.RecentDealsByLiveStreamIDs(context.Background(), []int64{1}, 3)
	require.NoError(t, err)
	require.Len(t, got[1], 1)
	assert.EqualValues(t, 303, got[1][0].ProductID)
	assert.Equal(t, "300.00", got[1][0].FinalPrice)
}

package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrentByLiveStreamIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/internal/auctions/current-by-live-streams", r.URL.Path)
		assert.Equal(t, "secret-token", r.Header.Get("X-Internal-Token"))

		raw, _ := io.ReadAll(r.Body)
		var reqBody struct {
			LiveStreamIDs []int64 `json:"live_stream_ids"`
		}
		require.NoError(t, json.Unmarshal(raw, &reqBody))
		assert.Equal(t, []int64{3, 4}, reqBody.LiveStreamIDs)

		_, _ = w.Write([]byte(`{"code":200,"data":{"items":[{"live_stream_id":3,"auction_id":11,"product_id":8,"current_price":"1200.00","status":1}]}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewAuctionClient(srv.URL, 0)
	c.SetInternalToken("secret-token")

	got, err := c.CurrentByLiveStreamIDs(context.Background(), []int64{3, 4})
	require.NoError(t, err)
	require.Len(t, got, 1)
	item, ok := got[3]
	require.True(t, ok)
	assert.EqualValues(t, 11, item.AuctionID)
	assert.EqualValues(t, 8, item.ProductID)
	assert.Equal(t, "1200.00", item.CurrentPrice)
	assert.Equal(t, 1, item.Status)
}

func TestCurrentByLiveStreamIDs_NonOKReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	c := NewAuctionClient(srv.URL, 0)
	_, err := c.CurrentByLiveStreamIDs(context.Background(), []int64{1})
	require.Error(t, err)
}

func TestAuctionClientBatchProductAuctionStates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/internal/auctions/by-products", r.URL.Path)
		assert.Equal(t, "secret-token", r.Header.Get("X-Internal-Token"))

		raw, _ := io.ReadAll(r.Body)
		var reqBody struct {
			ProductIDs []int64 `json:"product_ids"`
		}
		require.NoError(t, json.Unmarshal(raw, &reqBody))
		assert.Equal(t, []int64{101, 102}, reqBody.ProductIDs)

		_, _ = w.Write([]byte(`{"code":200,"message":"success","data":{"items":[{"product_id":101,"active_auction_id":11,"active_status":1},{"product_id":102,"latest_auction_id":12,"latest_auction_status":3,"latest_auction_result":"sold"}]}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewAuctionClient(srv.URL, 0)
	c.SetInternalToken("secret-token")

	got, err := c.BatchProductAuctionStates(context.Background(), []int64{101, 102})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.NotNil(t, got[101].ActiveAuctionID)
	assert.Equal(t, int64(11), *got[101].ActiveAuctionID)
	require.NotNil(t, got[102].LatestAuctionID)
	assert.Equal(t, int64(12), *got[102].LatestAuctionID)
	assert.Equal(t, "sold", got[102].LatestAuctionResult)
}

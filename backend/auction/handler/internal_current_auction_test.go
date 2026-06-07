package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCurrentAuctionFetcher 模拟 CurrentAuctionFetcher，按 live_stream_id 返回当前竞品。
type fakeCurrentAuctionFetcher struct {
	calledIDs []int64
	items     map[int64]CurrentAuctionItem
	err       error
}

func (f *fakeCurrentAuctionFetcher) Fetch(_ context.Context, ids []int64) (map[int64]CurrentAuctionItem, error) {
	f.calledIDs = ids
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func TestCurrentByLiveStreams(t *testing.T) {
	fetcher := &fakeCurrentAuctionFetcher{
		items: map[int64]CurrentAuctionItem{
			3: {AuctionID: 11, ProductID: 8, CurrentPrice: "1200.00", Status: 1},
		},
	}
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	currentAuctionHandler := NewInternalCurrentAuctionHandler(fetcher)
	h.POST("/internal/auctions/current-by-live-streams", currentAuctionHandler.Handle)

	body := `{"live_stream_ids":[3,4]}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/internal/auctions/current-by-live-streams",
		&ut.Body{Body: bytes.NewReader([]byte(body)), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Result().StatusCode())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				LiveStreamID int64  `json:"live_stream_id"`
				AuctionID    int64  `json:"auction_id"`
				ProductID    int64  `json:"product_id"`
				CurrentPrice string `json:"current_price"`
				Status       int    `json:"status"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Result().Body(), &resp))

	assert.Equal(t, 200, resp.Code)
	require.Len(t, resp.Data.Items, 1)
	assert.Equal(t, int64(3), resp.Data.Items[0].LiveStreamID)
	assert.Equal(t, int64(11), resp.Data.Items[0].AuctionID)
	assert.Equal(t, int64(8), resp.Data.Items[0].ProductID)
	assert.Equal(t, "1200.00", resp.Data.Items[0].CurrentPrice)
	assert.Equal(t, 1, resp.Data.Items[0].Status)
	assert.Equal(t, []int64{3, 4}, fetcher.calledIDs)
}

type fakeAuctionCountProvider struct {
	calledIDs []int64
	counts    map[int64]int64
	err       error
}

func (f *fakeAuctionCountProvider) CountByLiveStreamIDs(_ context.Context, ids []int64) (map[int64]int64, error) {
	f.calledIDs = ids
	if f.err != nil {
		return nil, f.err
	}
	return f.counts, nil
}

func TestCountByLiveStreams(t *testing.T) {
	provider := &fakeAuctionCountProvider{
		counts: map[int64]int64{
			3: 2,
			4: 7,
		},
	}
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	countHandler := NewInternalAuctionCountHandler(provider)
	h.POST("/internal/auctions/count-by-live-streams", countHandler.Handle)

	body := `{"live_stream_ids":[3,4]}`
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/internal/auctions/count-by-live-streams",
		&ut.Body{Body: bytes.NewReader([]byte(body)), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Result().StatusCode())

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Counts map[int64]int64 `json:"counts"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Result().Body(), &resp))

	assert.Equal(t, 200, resp.Code)
	assert.EqualValues(t, 2, resp.Data.Counts[3])
	assert.EqualValues(t, 7, resp.Data.Counts[4])
	assert.Equal(t, []int64{3, 4}, provider.calledIDs)
}

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/require"
)

type fakeNextFetcher struct{ m map[int64]NextAuctionItem }

func (f fakeNextFetcher) Fetch(_ context.Context, _ []int64) (map[int64]NextAuctionItem, error) {
	return f.m, nil
}

func TestNextByLiveStreams(t *testing.T) {
	h := NewInternalNextAuctionHandler(fakeNextFetcher{m: map[int64]NextAuctionItem{
		1: {AuctionID: 11, ProductID: 101, StartPrice: "100", StartTime: "2026-06-08T10:00:00Z"},
	}})
	srv := server.Default(server.WithHostPorts("127.0.0.1:0"))
	srv.POST("/internal/auctions/next-by-live-streams", h.Handle)

	body := `{"live_stream_ids":[1,2]}`
	w := ut.PerformRequest(
		srv.Engine,
		http.MethodPost,
		"/internal/auctions/next-by-live-streams",
		&ut.Body{Body: bytes.NewReader([]byte(body)), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode())

	var got struct {
		Data struct {
			Items []map[string]interface{} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body(), &got))
	require.Len(t, got.Data.Items, 1)
	require.EqualValues(t, 1, got.Data.Items[0]["live_stream_id"])
	require.EqualValues(t, 11, got.Data.Items[0]["auction_id"])
	require.EqualValues(t, 101, got.Data.Items[0]["product_id"])
}

type fakeRecentFetcher struct{ m map[int64][]DealAuctionItem }

func (f fakeRecentFetcher) Fetch(_ context.Context, _ []int64, _ int) (map[int64][]DealAuctionItem, error) {
	return f.m, nil
}

func TestRecentDealsByLiveStreams(t *testing.T) {
	h := NewInternalRecentDealsHandler(fakeRecentFetcher{m: map[int64][]DealAuctionItem{
		1: {{AuctionID: 33, ProductID: 303, FinalPrice: "300", EndTime: "2026-06-08T09:00:00Z"}},
	}}, 3)
	srv := server.Default(server.WithHostPorts("127.0.0.1:0"))
	srv.POST("/internal/auctions/recent-deals-by-live-streams", h.Handle)

	body := `{"live_stream_ids":[1],"limit":3}`
	w := ut.PerformRequest(
		srv.Engine,
		http.MethodPost,
		"/internal/auctions/recent-deals-by-live-streams",
		&ut.Body{Body: bytes.NewReader([]byte(body)), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode())

	var got struct {
		Data struct {
			Items []struct {
				LiveStreamID int64                    `json:"live_stream_id"`
				Deals        []map[string]interface{} `json:"deals"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body(), &got))
	require.Len(t, got.Data.Items, 1)
	require.Len(t, got.Data.Items[0].Deals, 1)
	require.EqualValues(t, 303, got.Data.Items[0].Deals[0]["product_id"])
	require.Equal(t, "300", got.Data.Items[0].Deals[0]["final_price"])
}

package service

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"auction-service/model"
	"auction-service/pkg/metrics"
	"auction-service/websocket"
)

func TestFixedPriceWSBroadcaster_EmitsPublishMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	fpMetrics := metrics.NewFixedPriceMetrics(reg)
	hub, client := newFixedPriceBroadcasterTestClient(t, 1001)
	b := NewFixedPriceWSBroadcaster(hub, newFakeClock())
	b.SetMetrics(fpMetrics)
	ctx := context.Background()

	b.Listed(ctx, &model.FixedPriceItem{
		ID: 7001, LiveStreamID: 1001, ProductID: 5001,
		Price: decimal.NewFromInt(99), TotalStock: 10, RemainingStock: 10,
		Status: model.FixedPriceStatusOnSale,
	})
	require.Equal(t, websocket.MessageTypeFixedPriceListed, recvFixedPriceMsg(t, client.Send).Type)
	b.Flair(ctx, 1001, 7001, 42, decimal.NewFromInt(99))
	require.Equal(t, websocket.MessageTypeFixedPriceFlair, recvFixedPriceMsg(t, client.Send).Type)
	b.StockChanged(ctx, 1001, 7001, 0)
	require.Equal(t, websocket.MessageTypeFixedPriceStock, recvFixedPriceMsg(t, client.Send).Type)
	b.Offline(ctx, 1001, 7001)
	require.Equal(t, websocket.MessageTypeFixedPriceOffline, recvFixedPriceMsg(t, client.Send).Type)
	b.SoldOut(ctx, 1001, 7001)
	require.Equal(t, websocket.MessageTypeFixedPriceSoldOut, recvFixedPriceMsg(t, client.Send).Type)

	for _, msgType := range []string{
		string(websocket.MessageTypeFixedPriceFlair),
		string(websocket.MessageTypeFixedPriceListed),
		string(websocket.MessageTypeFixedPriceOffline),
		string(websocket.MessageTypeFixedPriceSoldOut),
		string(websocket.MessageTypeFixedPriceStock),
	} {
		assert.Equal(t, 1.0, gatheredMetricValue(t, reg, "fixed_price_ws_publish_total", map[string]string{"type": msgType}))
	}
}

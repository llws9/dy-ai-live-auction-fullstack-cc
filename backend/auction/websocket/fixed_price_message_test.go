package websocket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixedPriceListedMessage_JSONContract(t *testing.T) {
	msg := NewFixedPriceListedMessage(&FixedPriceListedData{
		ItemID: 7001, LiveStreamID: 1001, ProductID: 5001,
		Price: "99.00", TotalStock: 100, RemainingStock: 100, Status: "on_sale",
	})

	raw, err := json.Marshal(msg)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "fixed_price_listed", got["type"])
	data := got["data"].(map[string]any)
	assert.Equal(t, float64(7001), data["item_id"])
	assert.Equal(t, float64(1001), data["live_stream_id"])
	assert.Equal(t, float64(5001), data["product_id"])
	assert.Equal(t, "99.00", data["price"])
	assert.Equal(t, float64(100), data["total_stock"])
	assert.Equal(t, float64(100), data["remaining_stock"])
	assert.Equal(t, "on_sale", data["status"])
}

func TestFixedPriceStockMessage_JSONContract(t *testing.T) {
	msg := NewFixedPriceStockMessage(7001, 87)
	raw, err := json.Marshal(msg)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "fixed_price_stock", got["type"])
	data := got["data"].(map[string]any)
	assert.Equal(t, float64(7001), data["item_id"])
	assert.Equal(t, float64(87), data["remaining_stock"])
}

func TestFixedPriceSoldOutMessage_JSONContract(t *testing.T) {
	msg := NewFixedPriceSoldOutMessage(7001)
	raw, err := json.Marshal(msg)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "fixed_price_sold_out", got["type"])
	data := got["data"].(map[string]any)
	assert.Equal(t, float64(7001), data["item_id"])
}

func TestFixedPriceOfflineMessage_JSONContract(t *testing.T) {
	msg := NewFixedPriceOfflineMessage(7001)
	raw, err := json.Marshal(msg)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "fixed_price_offline", got["type"])
	data := got["data"].(map[string]any)
	assert.Equal(t, float64(7001), data["item_id"])
}

func TestFixedPriceFlairMessage_JSONContract(t *testing.T) {
	msg := NewFixedPriceFlairMessage(&FixedPriceFlairData{ItemID: 7001, BuyerID: 42, Price: "99.00"})
	raw, err := json.Marshal(msg)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "fixed_price_flair", got["type"])
	data := got["data"].(map[string]any)
	assert.Equal(t, float64(7001), data["item_id"])
	assert.Equal(t, float64(42), data["buyer_id"])
	assert.Equal(t, "99.00", data["price"])
}

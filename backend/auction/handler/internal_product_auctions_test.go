package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"auction-service/dao"
	"auction-service/model"
)

func TestInternalProductAuctionsHandler_ByProducts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Auction{}))

	now := time.Now()
	winnerID := int64(901)
	require.NoError(t, db.Create(&[]model.Auction{
		{ID: 11, ProductID: 101, Status: model.AuctionStatusOngoing, CurrentPrice: decimal.NewFromInt(1200), StartTime: now, EndTime: now.Add(time.Hour)},
		{ID: 12, ProductID: 102, Status: model.AuctionStatusEnded, WinnerID: &winnerID, CurrentPrice: decimal.NewFromInt(1600), StartTime: now.Add(-2 * time.Hour), EndTime: now.Add(-time.Hour)},
		{ID: 13, ProductID: 103, Status: model.AuctionStatusEnded, CurrentPrice: decimal.NewFromInt(900), StartTime: now.Add(-3 * time.Hour), EndTime: now.Add(-2 * time.Hour)},
	}).Error)

	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	internalHandler := NewInternalProductAuctionsHandler(dao.NewAuctionDAO(db))
	h.POST("/internal/auctions/by-products", internalHandler.Handle)

	body := []byte(`{"product_ids":[101,102,103,104]}`)
	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/internal/auctions/by-products",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Result().StatusCode())
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []ProductAuctionState `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Result().Body(), &resp))
	assert.Equal(t, 200, resp.Code)

	byProduct := make(map[int64]ProductAuctionState, len(resp.Data.Items))
	for _, item := range resp.Data.Items {
		byProduct[item.ProductID] = item
	}

	require.Len(t, byProduct, 3)
	require.NotNil(t, byProduct[101].ActiveAuctionID)
	assert.Equal(t, int64(11), *byProduct[101].ActiveAuctionID)
	assert.Equal(t, "sold", byProduct[102].LatestAuctionResult)
	assert.Equal(t, "unsold", byProduct[103].LatestAuctionResult)
	_, ok := byProduct[104]
	assert.False(t, ok, "products without auction facts should be omitted")
}

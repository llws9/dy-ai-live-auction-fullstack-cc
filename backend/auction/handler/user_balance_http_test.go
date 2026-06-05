package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBalanceTopUpper struct {
	lastUserID int64
	lastAmount decimal.Decimal
	balance    decimal.Decimal
	err        error
}

func (f *fakeBalanceTopUpper) GetByUserID(_ context.Context, userID int64) (available, frozen decimal.Decimal, currency string, hit bool, err error) {
	return decimal.Zero, decimal.Zero, "CNY", false, nil
}

func (f *fakeBalanceTopUpper) AddAmount(_ context.Context, userID int64, amount decimal.Decimal) (decimal.Decimal, error) {
	f.lastUserID = userID
	f.lastAmount = amount
	if f.err != nil {
		return decimal.Zero, f.err
	}
	return f.balance, nil
}

func TestTopUpUserBalanceInternalAddsAmount(t *testing.T) {
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	provider := &fakeBalanceTopUpper{balance: decimal.RequireFromString("500.00")}
	handler := NewUserBalanceHandler(provider)
	h.POST("/internal/test/user-balance", handler.TopUpInternal)

	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/internal/test/user-balance",
		&ut.Body{Body: bytes.NewReader([]byte(`{"user_id":1001,"amount":"500.00"}`)), Len: len(`{"user_id":1001,"amount":"500.00"}`)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, provider.lastAmount.Equal(decimal.RequireFromString("500.00")))
	assert.Equal(t, int64(1001), provider.lastUserID)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			UserID  int64  `json:"user_id"`
			Balance string `json:"balance"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, int64(1001), resp.Data.UserID)
	assert.Equal(t, "500.00", resp.Data.Balance)
}

func TestTopUpUserBalanceInternalRejectsInvalidDecimal(t *testing.T) {
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	handler := NewUserBalanceHandler(&fakeBalanceTopUpper{})
	h.POST("/internal/test/user-balance", handler.TopUpInternal)

	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/internal/test/user-balance",
		&ut.Body{Body: bytes.NewReader([]byte(`{"user_id":1001,"amount":"abc"}`)), Len: len(`{"user_id":1001,"amount":"abc"}`)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid amount")
}

func TestTopUpUserBalanceInternalRejectsNonPositiveAmount(t *testing.T) {
	h := server.Default(server.WithHostPorts("127.0.0.1:0"))
	handler := NewUserBalanceHandler(&fakeBalanceTopUpper{})
	h.POST("/internal/test/user-balance", handler.TopUpInternal)

	w := ut.PerformRequest(
		h.Engine,
		http.MethodPost,
		"/internal/test/user-balance",
		&ut.Body{Body: bytes.NewReader([]byte(`{"user_id":1001,"amount":"0.00"}`)), Len: len(`{"user_id":1001,"amount":"0.00"}`)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "amount must be positive")
}

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

func TestAuctionClientBatchGetUserSummaries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/internal/users/batch", r.URL.Path)
		assert.Equal(t, "secret-token", r.Header.Get("X-Internal-Token"))

		raw, _ := io.ReadAll(r.Body)
		var reqBody struct {
			IDs []int64 `json:"ids"`
		}
		require.NoError(t, json.Unmarshal(raw, &reqBody))
		assert.Equal(t, []int64{901, 902}, reqBody.IDs)

		_, _ = w.Write([]byte(`{"code":200,"message":"success","data":{"items":[{"id":901,"username":"张三","avatar":"https://cdn/u901.png"},{"id":902,"username":"李四","avatar":""}]}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewAuctionClient(srv.URL, 0)
	c.SetInternalToken("secret-token")

	got, err := c.BatchGetUserSummaries(context.Background(), []int64{901, 902})
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "张三", got[901].Username)
	assert.Equal(t, "https://cdn/u901.png", got[901].Avatar)
	assert.Equal(t, "李四", got[902].Username)
}

func TestAuctionClientBatchGetUserSummariesNonOKReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	c := NewAuctionClient(srv.URL, 0)
	_, err := c.BatchGetUserSummaries(context.Background(), []int64{901})
	require.Error(t, err)
}

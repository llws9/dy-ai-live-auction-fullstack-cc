package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestTouchpointHandlerSummary(t *testing.T) {
	auctionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/notifications/summary", r.URL.Path)
		assert.Equal(t, "Bearer token-1", r.Header.Get("Authorization"))
		assert.Equal(t, "123", r.Header.Get("X-User-ID"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"unreadTotal":2,"outbid":1,"endingSoon":0}}`))
	}))
	defer auctionServer.Close()

	productServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/orders/summary", r.URL.Path)
		assert.Equal(t, "Bearer token-1", r.Header.Get("Authorization"))
		assert.Equal(t, "123", r.Header.Get("X-User-ID"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"pendingPayment":1,"wonNotPaid":1}}`))
	}))
	defer productServer.Close()

	h := NewTouchpointHandler(auctionServer.URL, productServer.URL)
	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/notifications/summary")
	c.Request.Header.Set("Authorization", "Bearer token-1")
	c.Set("user_id", int64(123))

	h.GetNotificationSummary(context.Background(), c)

	assert.Equal(t, http.StatusOK, c.Response.StatusCode())
	body := string(c.Response.Body())
	assert.Contains(t, body, `"unreadTotal":2`)
	assert.Contains(t, body, `"pendingPayment":1`)
	assert.Contains(t, body, `"wonNotPaid":1`)
	assert.Contains(t, body, `"outbid":1`)
}

func TestTouchpointHandlerSummaryFallsBackForUpstreamFailure(t *testing.T) {
	auctionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer auctionServer.Close()

	productServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"pendingPayment":1,"wonNotPaid":1}}`))
	}))
	defer productServer.Close()

	h := NewTouchpointHandler(auctionServer.URL, productServer.URL)
	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/notifications/summary")
	c.Set("user_id", int64(123))

	h.GetNotificationSummary(context.Background(), c)

	assert.Equal(t, http.StatusOK, c.Response.StatusCode())
	body := string(c.Response.Body())
	assert.Contains(t, body, `"unreadTotal":0`)
	assert.Contains(t, body, `"outbid":0`)
	assert.Contains(t, body, `"endingSoon":0`)
	assert.Contains(t, body, `"pendingPayment":1`)
}

func TestTouchpointHandlerSummaryPropagatesAuthFailure(t *testing.T) {
	auctionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer auctionServer.Close()

	productServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("product upstream must not be called after auth failure")
	}))
	defer productServer.Close()

	h := NewTouchpointHandler(auctionServer.URL, productServer.URL)
	c := app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/notifications/summary")
	c.Set("user_id", int64(123))

	h.GetNotificationSummary(context.Background(), c)

	assert.Equal(t, http.StatusUnauthorized, c.Response.StatusCode())
}

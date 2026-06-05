package handler

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gateway-service/model"
	"gateway-service/pkg/metrics"
)

type fakeBusinessEventStore struct {
	created  *model.BusinessEvent
	inserted bool
	err      error
}

func (s *fakeBusinessEventStore) Create(ctx context.Context, event *model.BusinessEvent) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	cp := *event
	s.created = &cp
	if !s.inserted {
		return false, nil
	}
	return true, nil
}

func TestBusinessEventHandlerCreatesEventWithAuthenticatedUser(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewMetrics("gateway", reg)
	store := &fakeBusinessEventStore{inserted: true}
	h := NewBusinessEventHandler(store, m)

	c := app.NewContext(0)
	c.Request.Header.SetMethod("POST")
	c.Request.SetRequestURI("/api/v1/events")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Set("user_id", int64(42))
	c.Request.SetBody([]byte(`{
		"event_type":"live_room_enter",
		"source":"live_reminder",
		"live_stream_id":1001,
		"auction_id":2002,
		"product_id":3003,
		"metadata":{"client_event_id":"evt-1","entry":"test"}
	}`))

	h.Create(context.Background(), c)

	require.Equal(t, 200, c.Response.StatusCode())
	require.NotNil(t, store.created)
	assert.Equal(t, int64(42), store.created.UserID)
	assert.Equal(t, "live_room_enter", store.created.EventType)
	assert.Equal(t, "live_reminder", store.created.Source)
	assert.Equal(t, int64(1001), store.created.LiveStreamID)
	assert.Equal(t, int64(2002), store.created.AuctionID)
	assert.Equal(t, int64(3003), store.created.ProductID)
	assert.NotEmpty(t, store.created.ClientEventID)
	assert.Equal(t, 1.0, testutil.ToFloat64(m.BusinessFunnelEvent.WithLabelValues(
		"live_room_enter",
		"live_reminder",
		"success",
	)))
}

func TestBusinessEventHandlerRecordsDuplicateSeparately(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewMetrics("gateway", reg)
	store := &fakeBusinessEventStore{inserted: false}
	h := NewBusinessEventHandler(store, m)

	c := app.NewContext(0)
	c.Request.Header.SetMethod("POST")
	c.Request.SetRequestURI("/api/v1/events")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Set("user_id", int64(42))
	c.Request.SetBody([]byte(`{
		"event_type":"reminder_click",
		"source":"live_reminder",
		"metadata":{"client_event_id":"evt-dup-1"}
	}`))

	h.Create(context.Background(), c)

	require.Equal(t, 200, c.Response.StatusCode())
	assert.Equal(t, 0.0, testutil.ToFloat64(m.BusinessFunnelEvent.WithLabelValues(
		"reminder_click",
		"live_reminder",
		"success",
	)))
	assert.Equal(t, 1.0, testutil.ToFloat64(m.BusinessFunnelEvent.WithLabelValues(
		"reminder_click",
		"live_reminder",
		"duplicate",
	)))
}

func TestBusinessEventHandlerRejectsUnknownEventType(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewMetrics("gateway", reg)
	store := &fakeBusinessEventStore{}
	h := NewBusinessEventHandler(store, m)

	c := app.NewContext(0)
	c.Request.Header.SetMethod("POST")
	c.Request.SetRequestURI("/api/v1/events")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Set("user_id", int64(42))
	c.Request.SetBody([]byte(`{"event_type":"user-42-dynamic","source":"home"}`))

	h.Create(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
	assert.Nil(t, store.created)
}

func TestBusinessEventHandlerRejectsOversizedClientEventID(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewMetrics("gateway", reg)
	store := &fakeBusinessEventStore{inserted: true}
	h := NewBusinessEventHandler(store, m)

	c := app.NewContext(0)
	c.Request.Header.SetMethod("POST")
	c.Request.SetRequestURI("/api/v1/events")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Set("user_id", int64(42))
	c.Request.SetBody([]byte(`{
		"event_type":"reminder_click",
		"source":"live_reminder",
		"metadata":{"client_event_id":"` + strings.Repeat("a", 129) + `"}
	}`))

	h.Create(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
	assert.Nil(t, store.created)
}

func TestBusinessEventHandlerRejectsUnsupportedMetadataKey(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewMetrics("gateway", reg)
	store := &fakeBusinessEventStore{inserted: true}
	h := NewBusinessEventHandler(store, m)

	c := app.NewContext(0)
	c.Request.Header.SetMethod("POST")
	c.Request.SetRequestURI("/api/v1/events")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Set("user_id", int64(42))
	c.Request.SetBody([]byte(`{
		"event_type":"fixed_price_click",
		"source":"fixed_price_card",
		"metadata":{"email":"user@example.com"}
	}`))

	h.Create(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
	assert.Nil(t, store.created)
}

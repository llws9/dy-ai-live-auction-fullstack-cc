package metrics

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMetrics(t *testing.T) *Metrics {
	t.Helper()
	reg := prometheus.NewRegistry()
	m := NewMetrics("gateway", reg)
	require.NotNil(t, m)
	return m
}

func TestTrackEventRecordsTouchpointMetric(t *testing.T) {
	m := newTestMetrics(t)
	c := app.NewContext(0)
	c.Request.Header.SetMethod("POST")
	c.Request.SetRequestURI("/api/v1/track")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody([]byte(`{
		"event_type":"touchpoint_event",
		"event_name":"summary_exposed",
		"user_id":"999",
		"params":{
			"source":"bottom_nav",
			"entry":"profile_tab",
			"type":"all",
			"result":"success",
			"notification_id":"123456"
		},
		"timestamp":1780300800000
	}`))

	TrackEvent(m)(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	assert.Equal(t, 1.0, testutil.ToFloat64(m.TouchpointEvent.WithLabelValues(
		"summary_exposed",
		"bottom_nav",
		"profile_tab",
		"all",
		"success",
	)))
}

func TestTrackEventNormalizesUnknownTouchpointLabels(t *testing.T) {
	m := newTestMetrics(t)
	c := app.NewContext(0)
	c.Request.Header.SetMethod("POST")
	c.Request.SetRequestURI("/api/v1/track")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody([]byte(`{
		"event_type":"touchpoint_event",
		"event_name":"not-in-allowlist",
		"params":{
			"source":"user-123456789",
			"entry":"dynamic-entry-123456789",
			"type":"unknown-dynamic-type",
			"result":"unexpected-result"
		}
	}`))

	TrackEvent(m)(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	assert.Equal(t, 1.0, testutil.ToFloat64(m.TouchpointEvent.WithLabelValues(
		"unknown",
		"unknown",
		"unknown",
		"unknown",
		"unknown",
	)))
}

func TestTrackEventRejectsInvalidJSON(t *testing.T) {
	m := newTestMetrics(t)
	c := app.NewContext(0)
	c.Request.Header.SetMethod("POST")
	c.Request.SetRequestURI("/api/v1/track")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody([]byte(`{"event_type":`))

	TrackEvent(m)(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
}

func TestTouchpointMetricDoesNotExposeHighCardinalityLabels(t *testing.T) {
	m := newTestMetrics(t)
	m.RecordTouchpointEvent("summary_exposed", "bottom_nav", "profile_tab", "all", "success")

	output, err := testutil.CollectAndLint(m.TouchpointEvent)
	require.NoError(t, err)
	assert.Empty(t, output)

	require.NoError(t, testutil.CollectAndCompare(m.TouchpointEvent, strings.NewReader(`
# HELP touchpoint_event_total 用户触达曝光和交互事件总数
# TYPE touchpoint_event_total counter
touchpoint_event_total{entry="profile_tab",event="summary_exposed",result="success",source="bottom_nav",type="all"} 1
`)))
}

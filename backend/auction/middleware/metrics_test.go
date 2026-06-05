package middleware

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestMetricsMiddlewareRecordsHTTPRequestMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewHTTPMetrics("auction", reg)
	c := app.NewContext(0)
	c.Request.Header.SetMethod("POST")
	c.Request.SetRequestURI("/api/v1/bids")
	c.Response.SetStatusCode(201)

	MetricsMiddleware("auction", m)(context.Background(), c)

	families, err := reg.Gather()
	require.NoError(t, err)
	requireMetricFamily(t, families, "http_requests_total")
	requireMetricFamily(t, families, "http_request_duration_seconds")
}

func TestMetricsMiddlewareNormalizesDynamicPathSegments(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewHTTPMetrics("auction", reg)
	c := app.NewContext(0)
	c.Request.Header.SetMethod("POST")
	c.Request.SetRequestURI("/api/v1/auctions/993321/bids")
	c.Response.SetStatusCode(200)

	MetricsMiddleware("auction", m)(context.Background(), c)

	families, err := reg.Gather()
	require.NoError(t, err)
	requireMetricLabel(t, families, "http_requests_total", "path", "/api/v1/auctions/:id/bids")
}

func requireMetricFamily(t *testing.T, families []*dto.MetricFamily, name string) {
	t.Helper()
	for _, family := range families {
		if family.GetName() == name && len(family.Metric) > 0 {
			return
		}
	}
	t.Fatalf("metric family %q was not collected", name)
}

func requireMetricLabel(t *testing.T, families []*dto.MetricFamily, familyName, labelName, expected string) {
	t.Helper()
	for _, family := range families {
		if family.GetName() != familyName {
			continue
		}
		for _, metric := range family.Metric {
			for _, label := range metric.Label {
				if label.GetName() == labelName && label.GetValue() == expected {
					return
				}
			}
		}
	}
	t.Fatalf("metric family %q did not include label %s=%q", familyName, labelName, expected)
}

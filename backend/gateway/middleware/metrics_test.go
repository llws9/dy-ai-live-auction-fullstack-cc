package middleware

import (
	"context"
	"testing"

	"gateway-service/pkg/metrics"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestMetricsMiddlewareNormalizesDynamicPathSegments(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewMetrics("gateway", reg)
	c := app.NewContext(0)
	c.Request.Header.SetMethod("GET")
	c.Request.SetRequestURI("/api/v1/auctions/993321/result?debug=true")
	c.Response.SetStatusCode(200)

	MetricsMiddleware("gateway", m)(context.Background(), c)

	families, err := reg.Gather()
	require.NoError(t, err)
	requireMetricLabel(t, families, "http_requests_total", "path", "/api/v1/auctions/:id/result")
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

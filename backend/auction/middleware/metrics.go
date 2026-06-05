package middleware

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

type HTTPMetrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
}

func NewHTTPMetrics(serviceName string, reg prometheus.Registerer) *HTTPMetrics {
	m := &HTTPMetrics{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "HTTP requests total.",
			},
			[]string{"service", "method", "path", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds.",
				Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10},
			},
			[]string{"service", "method", "path"},
		),
	}
	reg.MustRegister(m.RequestsTotal, m.RequestDuration)
	return m
}

func MetricsMiddleware(serviceName string, m *HTTPMetrics) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		c.Next(ctx)

		method := string(c.Method())
		path := normalizeMetricPath(string(c.URI().Path()))
		status := strconv.Itoa(c.Response.StatusCode())
		if path == "/metrics" {
			return
		}
		m.RequestsTotal.WithLabelValues(serviceName, method, path, status).Inc()
		m.RequestDuration.WithLabelValues(serviceName, method, path).Observe(time.Since(start).Seconds())
	}
}

func normalizeMetricPath(path string) string {
	if path == "" || path == "/" {
		return path
	}

	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if isDynamicPathSegment(segment) {
			segments[i] = ":id"
		}
	}
	return strings.Join(segments, "/")
}

func isDynamicPathSegment(segment string) bool {
	if segment == "" {
		return false
	}

	allDigits := true
	for _, r := range segment {
		if !unicode.IsDigit(r) {
			allDigits = false
			break
		}
	}
	if allDigits {
		return true
	}

	if len(segment) == 36 && strings.Count(segment, "-") == 4 {
		for _, r := range segment {
			if r == '-' || unicode.IsDigit(r) || ('a' <= r && r <= 'f') || ('A' <= r && r <= 'F') {
				continue
			}
			return false
		}
		return true
	}

	return false
}

func WriteMetricsResponse(c *app.RequestContext, gatherer prometheus.Gatherer) {
	families, err := gatherer.Gather()
	if err != nil {
		c.String(http.StatusInternalServerError, "Error gathering metrics: %v", err)
		return
	}

	var buf bytes.Buffer
	encoder := expfmt.NewEncoder(&buf, expfmt.NewFormat(expfmt.TypeTextPlain))
	for _, family := range families {
		if err := encoder.Encode(family); err != nil {
			c.String(http.StatusInternalServerError, "Error encoding metrics: %v", err)
			return
		}
	}
	c.Response.Header.Set("Content-Type", string(expfmt.NewFormat(expfmt.TypeTextPlain)))
	c.Response.SetBody(buf.Bytes())
}

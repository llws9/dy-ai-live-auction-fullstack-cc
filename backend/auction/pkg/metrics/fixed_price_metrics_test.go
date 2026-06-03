package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixedPriceMetrics_RecordsBusinessSignals(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewFixedPriceMetrics(reg)

	m.RecordPurchase("success")
	m.RecordPurchaseLatency("total", 25*time.Millisecond)
	m.SetStockRemaining(7001, 4)
	m.RecordWSPublish("fixed_price_stock")
	m.RecordCompensation("insufficient_balance")

	assert.Equal(t, 1.0, counterValue(t, m.purchaseTotal.WithLabelValues("success")))
	assert.Equal(t, 4.0, gaugeValue(t, m.stockRemaining.WithLabelValues("7001")))
	assert.Equal(t, 1.0, counterValue(t, m.wsPublishTotal.WithLabelValues("fixed_price_stock")))
	assert.Equal(t, 1.0, counterValue(t, m.compensationTotal.WithLabelValues("insufficient_balance")))

	histogram, err := m.purchaseLatency.GetMetricWithLabelValues("total")
	require.NoError(t, err)
	histogramMetric, ok := histogram.(prometheus.Metric)
	require.True(t, ok)
	metric := &dto.Metric{}
	require.NoError(t, histogramMetric.Write(metric))
	require.NotNil(t, metric.Histogram)
	assert.Equal(t, uint64(1), metric.Histogram.GetSampleCount())
}

func counterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	metric := &dto.Metric{}
	require.NoError(t, c.Write(metric))
	return metric.Counter.GetValue()
}

func gaugeValue(t *testing.T, g prometheus.Gauge) float64 {
	t.Helper()
	metric := &dto.Metric{}
	require.NoError(t, g.Write(metric))
	return metric.Gauge.GetValue()
}

package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// FixedPriceMetrics collects low-cardinality metrics for fixed-price purchases.
type FixedPriceMetrics struct {
	purchaseTotal     *prometheus.CounterVec
	purchaseLatency   *prometheus.HistogramVec
	stockRemaining    *prometheus.GaugeVec
	wsPublishTotal    *prometheus.CounterVec
	compensationTotal *prometheus.CounterVec
}

var fixedPriceMetrics *FixedPriceMetrics

func NewFixedPriceMetrics(registerer prometheus.Registerer) *FixedPriceMetrics {
	m := &FixedPriceMetrics{
		purchaseTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fixed_price_purchase_total",
				Help: "Fixed-price purchase attempts by result.",
			},
			[]string{"result"},
		),
		purchaseLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "fixed_price_purchase_latency_seconds",
				Help:    "Fixed-price purchase latency by stage.",
				Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
			},
			[]string{"stage"},
		),
		stockRemaining: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "fixed_price_stock_remaining",
				Help: "Remaining stock for fixed-price items.",
			},
			[]string{"item_id"},
		),
		wsPublishTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fixed_price_ws_publish_total",
				Help: "Fixed-price WebSocket publish attempts by message type.",
			},
			[]string{"type"},
		),
		compensationTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "fixed_price_compensation_total",
				Help: "Fixed-price Saga compensation executions by reason.",
			},
			[]string{"reason"},
		),
	}

	registerer.MustRegister(
		m.purchaseTotal,
		m.purchaseLatency,
		m.stockRemaining,
		m.wsPublishTotal,
		m.compensationTotal,
	)
	return m
}

func InitFixedPriceMetrics() *FixedPriceMetrics {
	fixedPriceMetrics = NewFixedPriceMetrics(prometheus.DefaultRegisterer)
	return fixedPriceMetrics
}

func GetFixedPriceMetrics() *FixedPriceMetrics {
	return fixedPriceMetrics
}

func (m *FixedPriceMetrics) RecordPurchase(result string) {
	if m == nil {
		return
	}
	m.purchaseTotal.WithLabelValues(result).Inc()
}

func (m *FixedPriceMetrics) RecordPurchaseLatency(stage string, d time.Duration) {
	if m == nil {
		return
	}
	m.purchaseLatency.WithLabelValues(stage).Observe(d.Seconds())
}

func (m *FixedPriceMetrics) SetStockRemaining(itemID int64, remaining int) {
	if m == nil {
		return
	}
	m.stockRemaining.WithLabelValues(formatInt64(itemID)).Set(float64(remaining))
}

func (m *FixedPriceMetrics) RecordWSPublish(msgType string) {
	if m == nil {
		return
	}
	m.wsPublishTotal.WithLabelValues(msgType).Inc()
}

func (m *FixedPriceMetrics) RecordCompensation(reason string) {
	if m == nil {
		return
	}
	m.compensationTotal.WithLabelValues(reason).Inc()
}

package service

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"auction-service/pkg/metrics"
)

func TestPurchase_EmitsMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	fpMetrics := metrics.NewFixedPriceMetrics(reg)
	svc := setupFixedPriceService(t)
	svc.SetMetrics(fpMetrics)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	setBalance(t, svc, 100, decimal.NewFromInt(1000))

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	require.NoError(t, err)

	assert.Equal(t, 1.0, gatheredMetricValue(t, reg, "fixed_price_purchase_total", map[string]string{"result": "success"}))
	assert.Equal(t, 4.0, gatheredMetricValue(t, reg, "fixed_price_stock_remaining", map[string]string{"item_id": itoa(item.ID)}))
}

func TestPurchase_EmitsFailureAndCompensationMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	fpMetrics := metrics.NewFixedPriceMetrics(reg)
	svc := setupFixedPriceService(t)
	svc.SetMetrics(fpMetrics)
	ctx := context.Background()
	item := setupItem(t, svc, 5, decimal.NewFromInt(99))
	setBalance(t, svc, 100, decimal.NewFromInt(50))

	_, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
	require.ErrorIs(t, err, ErrInsufficient)

	assert.Equal(t, 1.0, gatheredMetricValue(t, reg, "fixed_price_purchase_total", map[string]string{"result": "insufficient_balance"}))
	assert.Equal(t, 1.0, gatheredMetricValue(t, reg, "fixed_price_compensation_total", map[string]string{"reason": "insufficient_balance"}))
}

func gatheredMetricValue(t *testing.T, reg *prometheus.Registry, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			if metricLabelsMatch(metric, labels) {
				switch {
				case metric.Counter != nil:
					return metric.Counter.GetValue()
				case metric.Gauge != nil:
					return metric.Gauge.GetValue()
				}
			}
		}
	}
	t.Fatalf("metric %s with labels %v not found", name, labels)
	return 0
}

func metricLabelsMatch(metric *dto.Metric, expected map[string]string) bool {
	if len(metric.GetLabel()) != len(expected) {
		return false
	}
	for _, label := range metric.GetLabel() {
		if expected[label.GetName()] != label.GetValue() {
			return false
		}
	}
	return true
}

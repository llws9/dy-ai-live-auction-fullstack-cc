package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// SkyLampMetrics 点天灯服务指标收集器
type SkyLampMetrics struct {
	// 订阅相关指标
	subscriptionsCreated   *prometheus.CounterVec
	subscriptionsFailed    *prometheus.CounterVec
	subscriptionsStopped   *prometheus.CounterVec
	subscriptionsLimitReached *prometheus.CounterVec
	subscriptionDuration   *prometheus.HistogramVec
	subscriptionMaxPrice   *prometheus.HistogramVec

	// 自动跟价相关指标
	autoBidsSuccess     *prometheus.CounterVec
	autoBidsFailed      *prometheus.CounterVec
	autoBidLatency      *prometheus.HistogramVec
	autoBidAmount       *prometheus.HistogramVec
	autoBidCountPerSub  *prometheus.HistogramVec

	// 业务统计指标
	activeSubscriptions prometheus.Gauge
	totalAutoBids       prometheus.Counter
	successRate         *prometheus.GaugeVec
}

var skyLampMetrics *SkyLampMetrics

// InitSkyLampMetrics 初始化点天灯指标
func InitSkyLampMetrics() *SkyLampMetrics {
	m := &SkyLampMetrics{
		// 订阅创建总数
		subscriptionsCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "skylamp_subscriptions_created_total",
				Help: "点天灯订阅创建总数",
			},
			[]string{"auction_id", "user_id"},
		),

		// 订阅失败总数
		subscriptionsFailed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "skylamp_subscriptions_failed_total",
				Help: "点天灯订阅失败总数",
			},
			[]string{"auction_id", "error_type"},
		),

		// 订阅停止总数
		subscriptionsStopped: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "skylamp_subscriptions_stopped_total",
				Help: "点天灯订阅停止总数",
			},
			[]string{"auction_id", "user_id", "stop_reason"},
		),

		// 达到上限总数
		subscriptionsLimitReached: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "skylamp_subscriptions_limit_reached_total",
				Help: "点天灯订阅达到上限总数",
			},
			[]string{"auction_id", "user_id"},
		),

		// 订阅持续时间分布
		subscriptionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "skylamp_subscription_duration_seconds",
				Help:    "点天灯订阅持续时间分布",
				Buckets: []float64{10, 30, 60, 120, 300, 600, 1200, 1800, 3600},
			},
			[]string{"auction_id"},
		),

		// 订阅上限价格分布
		subscriptionMaxPrice: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "skylamp_subscription_max_price",
				Help:    "点天灯订阅上限价格分布",
				Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000},
			},
			[]string{"auction_id"},
		),

		// 自动跟价成功总数
		autoBidsSuccess: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "skylamp_auto_bids_success_total",
				Help: "点天灯自动跟价成功总数",
			},
			[]string{"auction_id", "user_id", "subscription_id"},
		),

		// 自动跟价失败总数
		autoBidsFailed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "skylamp_auto_bids_failed_total",
				Help: "点天灯自动跟价失败总数",
			},
			[]string{"auction_id", "subscription_id", "error_type"},
		),

		// 自动跟价延迟
		autoBidLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "skylamp_auto_bid_latency_seconds",
				Help:    "点天灯自动跟价延迟（从触发到完成）",
				Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1, 2, 5},
			},
			[]string{"auction_id"},
		),

		// 自动跟价金额分布
		autoBidAmount: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "skylamp_auto_bid_amount",
				Help:    "点天灯自动跟价金额分布",
				Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000},
			},
			[]string{"auction_id"},
		),

		// 每个订阅的自动跟价次数分布
		autoBidCountPerSub: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "skylamp_auto_bid_count_per_subscription",
				Help:    "每个订阅的自动跟价次数分布",
				Buckets: []float64{1, 2, 3, 5, 10, 20, 50, 100},
			},
			[]string{"auction_id"},
		),

		// 当前活跃订阅数
		activeSubscriptions: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "skylamp_active_subscriptions",
				Help: "当前活跃的点天灯订阅数",
			},
		),

		// 自动跟价总数
		totalAutoBids: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "skylamp_auto_bids_total",
				Help: "点天灯自动跟价总次数",
			},
		),

		// 成功率
		successRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "skylamp_success_rate",
				Help: "点天灯成功率（成功次数/总次数）",
			},
			[]string{"auction_id"},
		),
	}

	// 注册所有指标
	prometheus.MustRegister(
		m.subscriptionsCreated,
		m.subscriptionsFailed,
		m.subscriptionsStopped,
		m.subscriptionsLimitReached,
		m.subscriptionDuration,
		m.subscriptionMaxPrice,
		m.autoBidsSuccess,
		m.autoBidsFailed,
		m.autoBidLatency,
		m.autoBidAmount,
		m.autoBidCountPerSub,
		m.activeSubscriptions,
		m.totalAutoBids,
		m.successRate,
	)

	skyLampMetrics = m
	return m
}

// GetSkyLampMetrics 获取点天灯指标实例
func GetSkyLampMetrics() *SkyLampMetrics {
	return skyLampMetrics
}

// RecordSubscriptionCreated 记录订阅创建成功
func (m *SkyLampMetrics) RecordSubscriptionCreated(auctionID, userID int64, maxPriceLimit float64) {
	m.subscriptionsCreated.WithLabelValues(
		formatInt64(auctionID),
		formatInt64(userID),
	).Inc()

	m.subscriptionMaxPrice.WithLabelValues(
		formatInt64(auctionID),
	).Observe(maxPriceLimit)

	m.activeSubscriptions.Inc()
}

// RecordSubscriptionFailed 记录订阅创建失败
func (m *SkyLampMetrics) RecordSubscriptionFailed(auctionID int64, errorType string) {
	m.subscriptionsFailed.WithLabelValues(
		formatInt64(auctionID),
		errorType,
	).Inc()
}

// RecordSubscriptionStopped 记录订阅停止
func (m *SkyLampMetrics) RecordSubscriptionStopped(auctionID, userID int64, durationSeconds float64, stopReason string) {
	m.subscriptionsStopped.WithLabelValues(
		formatInt64(auctionID),
		formatInt64(userID),
		stopReason,
	).Inc()

	m.subscriptionDuration.WithLabelValues(
		formatInt64(auctionID),
	).Observe(durationSeconds)

	m.activeSubscriptions.Dec()
}

// RecordSubscriptionLimitReached 记录订阅达到上限
func (m *SkyLampMetrics) RecordSubscriptionLimitReached(auctionID, userID int64, durationSeconds float64) {
	m.subscriptionsLimitReached.WithLabelValues(
		formatInt64(auctionID),
		formatInt64(userID),
	).Inc()

	m.subscriptionDuration.WithLabelValues(
		formatInt64(auctionID),
	).Observe(durationSeconds)

	m.activeSubscriptions.Dec()
}

// RecordAutoBidSuccess 记录自动跟价成功
func (m *SkyLampMetrics) RecordAutoBidSuccess(auctionID, userID, subscriptionID int64, amount float64, latency time.Duration) {
	m.autoBidsSuccess.WithLabelValues(
		formatInt64(auctionID),
		formatInt64(userID),
		formatInt64(subscriptionID),
	).Inc()

	m.autoBidLatency.WithLabelValues(
		formatInt64(auctionID),
	).Observe(latency.Seconds())

	m.autoBidAmount.WithLabelValues(
		formatInt64(auctionID),
	).Observe(amount)

	m.totalAutoBids.Inc()
}

// RecordAutoBidFailed 记录自动跟价失败
func (m *SkyLampMetrics) RecordAutoBidFailed(auctionID, subscriptionID int64, errorType string) {
	m.autoBidsFailed.WithLabelValues(
		formatInt64(auctionID),
		formatInt64(subscriptionID),
		errorType,
	).Inc()

	m.totalAutoBids.Inc()
}

// RecordAutoBidCountPerSubscription 记录每个订阅的自动跟价次数
func (m *SkyLampMetrics) RecordAutoBidCountPerSubscription(auctionID int64, count int) {
	m.autoBidCountPerSub.WithLabelValues(
		formatInt64(auctionID),
	).Observe(float64(count))
}

// UpdateSuccessRate 更新成功率
func (m *SkyLampMetrics) UpdateSuccessRate(auctionID int64, successCount, totalCount float64) {
	if totalCount > 0 {
		rate := successCount / totalCount
		m.successRate.WithLabelValues(formatInt64(auctionID)).Set(rate)
	}
}
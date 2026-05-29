package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"auction-service/model"
)

// AuctionMetrics 竞拍指标收集服务
var defaultMetrics *AuctionMetrics

// AuctionMetrics 竞拍指标收集器
type AuctionMetrics struct {
	// 竞拍指标
	bidLatency       *prometheus.HistogramVec
	bidTotal         *prometheus.CounterVec
	bidAmount        *prometheus.HistogramVec
	bidUserCount     *prometheus.CounterVec
	delayCount       *prometheus.CounterVec
	concurrentBids   prometheus.Gauge
	auctionDuration  *prometheus.HistogramVec
	auctionCompleted *prometheus.CounterVec
	premiumRate      *prometheus.GaugeVec
	gmv              *prometheus.GaugeVec

	// WebSocket 指标
	wsConnections    prometheus.Gauge
	wsMessages       *prometheus.CounterVec
	wsMessageLatency *prometheus.HistogramVec
	wsErrors         *prometheus.CounterVec

	// 用户指标
	watchUserCount   prometheus.Gauge
}

// Init 初始化指标
func Init() *AuctionMetrics {
	m := &AuctionMetrics{
		// 出价响应延迟
		bidLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "auction_bid_latency_seconds",
				Help:    "出价响应延迟（从用户出价到系统响应）",
				Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1, 2, 5},
			},
			[]string{"auction_id", "success"},
		),

		// 出价次数统计
		bidTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auction_bid_total",
				Help: "出价次数统计",
			},
			[]string{"auction_id", "status"},
		),

		// 出价金额分布
		bidAmount: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "auction_bid_amount",
				Help:    "出价金额分布",
				Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000},
			},
			[]string{"auction_id"},
		),

		// 出价用户数
		bidUserCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "bid_user_count_total",
				Help: "出价用户数统计",
			},
			[]string{"auction_id"},
		),

		// 延时触发次数
		delayCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auction_delay_triggered_total",
				Help: "竞拍延时触发次数",
			},
			[]string{"auction_id"},
		),

		// 并发出价峰值
		concurrentBids: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "auction_concurrent_bids_peak",
				Help: "当前并发出价峰值",
			},
		),

		// 竞拍时长分布
		auctionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "auction_duration_seconds",
				Help:    "竞拍时长分布（从开始到结束）",
				Buckets: []float64{30, 60, 120, 300, 600, 1200, 1800, 3600},
			},
			[]string{"has_winner"},
		),

		// 竞拍完成总数
		auctionCompleted: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auction_completed_total",
				Help: "竞拍完成总数",
			},
			[]string{"auction_id", "has_winner"},
		),

		// 竞拍溢价率
		premiumRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "auction_premium_rate",
				Help: "竞拍溢价率（成交价/起拍价）",
			},
			[]string{"time_window"},
		),

		// GMV（成交总额）
		gmv: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gmv_total",
				Help: "GMV（成交总额）",
			},
			[]string{"time_window"},
		),

		// WebSocket 连接数
		wsConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "websocket_connections",
				Help: "WebSocket当前连接数",
			},
		),

		// WebSocket 消息总数
		wsMessages: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "websocket_messages_total",
				Help: "WebSocket消息总数",
			},
			[]string{"type", "direction"},
		),

		// WebSocket 消息延迟
		wsMessageLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "websocket_message_latency_seconds",
				Help:    "WebSocket消息推送延迟",
				Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1, 2},
			},
			[]string{"type"},
		),

		// WebSocket 错误总数
		wsErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "websocket_errors_total",
				Help: "WebSocket错误总数",
			},
			[]string{"type"},
		),

		// 观看用户数
		watchUserCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "watch_user_count",
				Help: "当前观看用户数",
			},
		),
	}

	// 注册所有指标
	prometheus.MustRegister(
		m.bidLatency,
		m.bidTotal,
		m.bidAmount,
		m.bidUserCount,
		m.delayCount,
		m.concurrentBids,
		m.auctionDuration,
		m.auctionCompleted,
		m.premiumRate,
		m.gmv,
		m.wsConnections,
		m.wsMessages,
		m.wsMessageLatency,
		m.wsErrors,
		m.watchUserCount,
	)

	defaultMetrics = m
	return m
}

// GetMetrics 获取默认指标实例
func GetMetrics() *AuctionMetrics {
	return defaultMetrics
}

// RecordBidLatency 记录出价响应延迟
func (m *AuctionMetrics) RecordBidLatency(auctionID int64, startTime time.Time, success bool) {
	latency := time.Since(startTime).Seconds()
	successStr := "true"
	if !success {
		successStr = "false"
	}
	m.bidLatency.WithLabelValues(formatInt64(auctionID), successStr).Observe(latency)
}

// RecordBid 记录出价
func (m *AuctionMetrics) RecordBid(auctionID int64, amount float64, success bool) {
	status := "success"
	if !success {
		status = "failed"
	}
	m.bidTotal.WithLabelValues(formatInt64(auctionID), status).Inc()
	if success {
		m.bidAmount.WithLabelValues(formatInt64(auctionID)).Observe(amount)
	}
}

// RecordBidUser 记录出价用户
func (m *AuctionMetrics) RecordBidUser(auctionID int64) {
	m.bidUserCount.WithLabelValues(formatInt64(auctionID)).Inc()
}

// RecordDelayTriggered 记录延时触发
func (m *AuctionMetrics) RecordDelayTriggered(auctionID int64) {
	m.delayCount.WithLabelValues(formatInt64(auctionID)).Inc()
}

// IncConcurrentBids 增加并发出价计数
func (m *AuctionMetrics) IncConcurrentBids() {
	m.concurrentBids.Inc()
}

// DecConcurrentBids 减少并发出价计数
func (m *AuctionMetrics) DecConcurrentBids() {
	m.concurrentBids.Dec()
}

// RecordAuctionCompleted 记录竞拍完成
func (m *AuctionMetrics) RecordAuctionCompleted(auction *model.Auction, startPrice float64) {
	if auction == nil {
		return
	}

	hasWinner := auction.WinnerID != nil && *auction.WinnerID > 0
	hasWinnerStr := "true"
	if !hasWinner {
		hasWinnerStr = "false"
	}

	durationSeconds := auction.EndTime.Sub(auction.StartTime).Seconds()
	m.auctionCompleted.WithLabelValues(formatInt64(auction.ID), hasWinnerStr).Inc()
	m.auctionDuration.WithLabelValues(hasWinnerStr).Observe(durationSeconds)

	if hasWinner && startPrice > 0 {
		premiumRate := (auction.CurrentPrice - startPrice) / startPrice
		m.premiumRate.WithLabelValues("current").Set(premiumRate)
		m.gmv.WithLabelValues("current").Add(auction.CurrentPrice)
	}
}

// SetWatchUserCount 设置观看用户数
func (m *AuctionMetrics) SetWatchUserCount(count float64) {
	m.watchUserCount.Set(count)
}

// IncWSConnection 增加 WebSocket 连接
func (m *AuctionMetrics) IncWSConnection() {
	m.wsConnections.Inc()
}

// DecWSConnection 减少 WebSocket 连接
func (m *AuctionMetrics) DecWSConnection() {
	m.wsConnections.Dec()
}

// RecordWSMessage 记录 WebSocket 消息
func (m *AuctionMetrics) RecordWSMessage(msgType, direction string) {
	m.wsMessages.WithLabelValues(msgType, direction).Inc()
}

// RecordWSMessageLatency 记录 WebSocket 消息延迟
func (m *AuctionMetrics) RecordWSMessageLatency(msgType string, startTime time.Time) {
	latency := time.Since(startTime).Seconds()
	m.wsMessageLatency.WithLabelValues(msgType).Observe(latency)
}

// RecordWSError 记录 WebSocket 错误
func (m *AuctionMetrics) RecordWSError(errorType string) {
	m.wsErrors.WithLabelValues(errorType).Inc()
}

// formatInt64 将 int64 转换为字符串
func formatInt64(n int64) string {
	if n <= 0 {
		return "unknown"
	}
	return strconv.FormatInt(n, 10)
}

// InitRegistry 初始化 Prometheus Registry
func InitRegistry() {
	// 初始化默认 AuctionMetrics
	Init()
	// 初始化 SkyLampMetrics
	InitSkyLampMetrics()
	// 初始化 NotificationMetrics
	InitNotificationMetrics()
}
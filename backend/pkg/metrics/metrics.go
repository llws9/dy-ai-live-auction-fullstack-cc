package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics 全局指标收集器
type Metrics struct {
	// 请求相关
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec

	// 业务指标
	// 直播间
	LiveRoomEnter       *prometheus.CounterVec
	LiveRoomViewers     prometheus.Gauge
	LiveRoomPeakViewers *prometheus.GaugeVec

	// 竞拍
	AuctionCreated        *prometheus.CounterVec
	AuctionBidTotal       *prometheus.CounterVec
	AuctionBidAmount      *prometheus.HistogramVec
	AuctionCompleted      *prometheus.CounterVec
	AuctionSuccessRate    *prometheus.GaugeVec
	AuctionDuration       *prometheus.HistogramVec // 竞拍时长分布
	AuctionDelayCount     *prometheus.CounterVec   // 延时触发次数
	AuctionPremiumRate    *prometheus.GaugeVec     // 竞拍溢价率
	AuctionBidLatency     *prometheus.HistogramVec // 出价响应延迟
	AuctionConcurrentBids prometheus.Gauge         // 并发出价峰值

	// 订单/成交
	OrderCreated    *prometheus.CounterVec
	OrderCompleted  *prometheus.CounterVec
	OrderAmount     *prometheus.HistogramVec
	OrderSuccessRate prometheus.Gauge
	GMV             *prometheus.GaugeVec // GMV（成交总额）
	PaymentRate     *prometheus.GaugeVec // 支付成功率

	// 用户
	UserRegister    *prometheus.CounterVec
	UserLogin       *prometheus.CounterVec
	UserActive      prometheus.Gauge
	BidUserCount    *prometheus.CounterVec // 出价用户数
	WatchUserCount  prometheus.Gauge       // 观看用户数

	// WebSocket
	WSConnections    prometheus.Gauge
	WSMessages       *prometheus.CounterVec
	WSErrors         *prometheus.CounterVec
	WSMessageLatency *prometheus.HistogramVec // 消息推送延迟

	// 支付
	PaymentInitiated *prometheus.CounterVec
	PaymentCompleted *prometheus.CounterVec
	PaymentFailed    *prometheus.CounterVec
	PaymentAmount    *prometheus.HistogramVec
}

var defaultMetrics *Metrics

// Init 初始化指标
func Init(serviceName string) *Metrics {
	m := &Metrics{
		// 请求指标
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "HTTP请求总数",
			},
			[]string{"service", "method", "path", "status"},
		),

		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP请求耗时分布",
				Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10},
			},
			[]string{"service", "method", "path"},
		),

		// 直播间指标
		LiveRoomEnter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "live_room_enter_total",
				Help: "直播间进入次数",
			},
			[]string{"room_id", "user_type"},
		),

		LiveRoomViewers: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "live_room_current_viewers",
				Help: "直播间当前观看人数",
			},
		),

		LiveRoomPeakViewers: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "live_room_peak_viewers",
				Help: "直播间峰值观看人数",
			},
			[]string{"room_id"},
		),

		// 竞拍指标
		AuctionCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auction_created_total",
				Help: "竞拍创建总数",
			},
			[]string{"product_id", "status"},
		),

		AuctionBidTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auction_bid_total",
				Help: "出价次数统计",
			},
			[]string{"auction_id", "status"},
		),

		AuctionBidAmount: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "auction_bid_amount",
				Help:    "出价金额分布",
				Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000},
			},
			[]string{"auction_id"},
		),

		AuctionCompleted: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auction_completed_total",
				Help: "竞拍完成总数",
			},
			[]string{"auction_id", "has_winner"},
		),

		AuctionSuccessRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "auction_success_rate",
				Help: "竞拍成功率",
			},
			[]string{"time_window"},
		),

		// 新增：竞拍时长分布
		AuctionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "auction_duration_seconds",
				Help:    "竞拍时长分布（从开始到结束）",
				Buckets: []float64{30, 60, 120, 300, 600, 1200, 1800, 3600},
			},
			[]string{"has_winner"},
		),

		// 新增：延时触发次数
		AuctionDelayCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "auction_delay_triggered_total",
				Help: "竞拍延时触发次数",
			},
			[]string{"auction_id"},
		),

		// 新增：竞拍溢价率
		AuctionPremiumRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "auction_premium_rate",
				Help: "竞拍溢价率（成交价/起拍价）",
			},
			[]string{"time_window"},
		),

		// 新增：出价响应延迟
		AuctionBidLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "auction_bid_latency_seconds",
				Help:    "出价响应延迟（从用户出价到系统响应）",
				Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1, 2, 5},
			},
			[]string{"auction_id", "success"},
		),

		// 新增：并发出价峰值
		AuctionConcurrentBids: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "auction_concurrent_bids_peak",
				Help: "当前并发出价峰值",
			},
		),

		// 订单指标
		OrderCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "order_created_total",
				Help: "订单创建总数",
			},
			[]string{"auction_id", "product_id"},
		),

		OrderCompleted: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "order_completed_total",
				Help: "订单完成（成交）总数",
			},
			[]string{"auction_id", "product_id"},
		),

		OrderAmount: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "order_amount",
				Help:    "订单金额分布",
				Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000},
			},
			[]string{"status"},
		),

		OrderSuccessRate: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "order_success_rate",
				Help: "订单成交率",
			},
		),

		// 新增：GMV（成交总额）
		GMV: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gmv_total",
				Help: "GMV（成交总额）",
			},
			[]string{"time_window"},
		),

		// 新增：支付成功率
		PaymentRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "payment_success_rate",
				Help: "支付成功率",
			},
			[]string{"method"},
		),

		// 用户指标
		UserRegister: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "user_register_total",
				Help: "用户注册总数",
			},
			[]string{"source"},
		),

		UserLogin: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "user_login_total",
				Help: "用户登录总数",
			},
			[]string{"method"},
		),

		UserActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "user_active_count",
				Help: "活跃用户数",
			},
		),

		// 新增：出价用户数
		BidUserCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "bid_user_count_total",
				Help: "出价用户数统计",
			},
			[]string{"auction_id"},
		),

		// 新增：观看用户数
		WatchUserCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "watch_user_count",
				Help: "当前观看用户数",
			},
		),

		// WebSocket 指标
		WSConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "websocket_connections",
				Help: "WebSocket当前连接数",
			},
		),

		WSMessages: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "websocket_messages_total",
				Help: "WebSocket消息总数",
			},
			[]string{"type", "direction"},
		),

		WSErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "websocket_errors_total",
				Help: "WebSocket错误总数",
			},
			[]string{"type"},
		),

		// 新增：消息推送延迟
		WSMessageLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "websocket_message_latency_seconds",
				Help:    "WebSocket消息推送延迟",
				Buckets: []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1, 2},
			},
			[]string{"type"},
		),

		// 支付指标
		PaymentInitiated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "payment_initiated_total",
				Help: "发起支付次数",
			},
			[]string{"method"},
		),

		PaymentCompleted: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "payment_completed_total",
				Help: "支付完成次数",
			},
			[]string{"method"},
		),

		PaymentFailed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "payment_failed_total",
				Help: "支付失败次数",
			},
			[]string{"method", "error_code"},
		),

		PaymentAmount: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "payment_amount",
				Help:    "支付金额分布",
				Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000},
			},
			[]string{"method"},
		),
	}

	// 注册所有指标
	prometheus.MustRegister(
		m.RequestsTotal,
		m.RequestDuration,
		m.LiveRoomEnter,
		m.LiveRoomViewers,
		m.LiveRoomPeakViewers,
		m.AuctionCreated,
		m.AuctionBidTotal,
		m.AuctionBidAmount,
		m.AuctionCompleted,
		m.AuctionSuccessRate,
		m.AuctionDuration,
		m.AuctionDelayCount,
		m.AuctionPremiumRate,
		m.AuctionBidLatency,
		m.AuctionConcurrentBids,
		m.OrderCreated,
		m.OrderCompleted,
		m.OrderAmount,
		m.OrderSuccessRate,
		m.GMV,
		m.PaymentRate,
		m.UserRegister,
		m.UserLogin,
		m.UserActive,
		m.BidUserCount,
		m.WatchUserCount,
		m.WSConnections,
		m.WSMessages,
		m.WSErrors,
		m.WSMessageLatency,
		m.PaymentInitiated,
		m.PaymentCompleted,
		m.PaymentFailed,
		m.PaymentAmount,
	)

	defaultMetrics = m
	return m
}

// GetMetrics 获取默认指标实例
func GetMetrics() *Metrics {
	return defaultMetrics
}

// Handler 返回 Prometheus 指标处理 Handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// --- 便捷方法 ---

// RecordRequest 记录 HTTP 请求
func (m *Metrics) RecordRequest(service, method, path string, status int, duration float64) {
	m.RequestsTotal.WithLabelValues(service, method, path, string(rune(status))).Inc()
	m.RequestDuration.WithLabelValues(service, method, path).Observe(duration)
}

// RecordLiveRoomEnter 记录直播间进入
func (m *Metrics) RecordLiveRoomEnter(roomID, userType string) {
	m.LiveRoomEnter.WithLabelValues(roomID, userType).Inc()
}

// SetLiveRoomViewers 设置直播间观看人数
func (m *Metrics) SetLiveRoomViewers(count float64) {
	m.LiveRoomViewers.Set(count)
}

// RecordAuctionBid 记录出价
func (m *Metrics) RecordAuctionBid(auctionID string, amount float64, success bool) {
	status := "success"
	if !success {
		status = "failed"
	}
	m.AuctionBidTotal.WithLabelValues(auctionID, status).Inc()
	if success {
		m.AuctionBidAmount.WithLabelValues(auctionID).Observe(amount)
	}
}

// RecordAuctionBidLatency 记录出价延迟
func (m *Metrics) RecordAuctionBidLatency(auctionID string, latency float64, success bool) {
	successStr := "true"
	if !success {
		successStr = "false"
	}
	m.AuctionBidLatency.WithLabelValues(auctionID, successStr).Observe(latency)
}

// RecordAuctionCompleted 记录竞拍完成
func (m *Metrics) RecordAuctionCompleted(auctionID, productID string, hasWinner bool, durationSeconds float64, finalPrice, startPrice float64) {
	hasWinnerStr := "true"
	if !hasWinner {
		hasWinnerStr = "false"
	}
	m.AuctionCompleted.WithLabelValues(auctionID, hasWinnerStr).Inc()
	m.AuctionDuration.WithLabelValues(hasWinnerStr).Observe(durationSeconds)

	if hasWinner && startPrice > 0 {
		premiumRate := (finalPrice - startPrice) / startPrice
		m.AuctionPremiumRate.WithLabelValues("current").Set(premiumRate)
		m.OrderCompleted.WithLabelValues(auctionID, productID).Inc()
		m.OrderAmount.WithLabelValues("completed").Observe(finalPrice)
		m.GMV.WithLabelValues("current").Add(finalPrice)
	}
}

// RecordAuctionDelay 记录延时触发
func (m *Metrics) RecordAuctionDelay(auctionID string) {
	m.AuctionDelayCount.WithLabelValues(auctionID).Inc()
}

// IncConcurrentBids 增加并发出价计数
func (m *Metrics) IncConcurrentBids() {
	m.AuctionConcurrentBids.Inc()
}

// DecConcurrentBids 减少并发出价计数
func (m *Metrics) DecConcurrentBids() {
	m.AuctionConcurrentBids.Dec()
}

// RecordBidUser 记录出价用户
func (m *Metrics) RecordBidUser(auctionID string) {
	m.BidUserCount.WithLabelValues(auctionID).Inc()
}

// SetWatchUserCount 设置观看用户数
func (m *Metrics) SetWatchUserCount(count float64) {
	m.WatchUserCount.Set(count)
}

// RecordOrderCompleted 记录订单完成（成交）
func (m *Metrics) RecordOrderCompleted(auctionID, productID string, amount float64) {
	m.OrderCompleted.WithLabelValues(auctionID, productID).Inc()
	m.OrderAmount.WithLabelValues("completed").Observe(amount)
	m.GMV.WithLabelValues("current").Add(amount)
}

// RecordPayment 记录支付
func (m *Metrics) RecordPayment(method string, amount float64, success bool, errorCode string) {
	m.PaymentInitiated.WithLabelValues(method).Inc()
	if success {
		m.PaymentCompleted.WithLabelValues(method).Inc()
		m.PaymentAmount.WithLabelValues(method).Observe(amount)
	} else {
		m.PaymentFailed.WithLabelValues(method, errorCode).Inc()
	}
}

// IncWSConnections 增加 WebSocket 连接
func (m *Metrics) IncWSConnections() {
	m.WSConnections.Inc()
}

// DecWSConnections 减少 WebSocket 连接
func (m *Metrics) DecWSConnections() {
	m.WSConnections.Dec()
}

// RecordWSMessage 记录 WebSocket 消息
func (m *Metrics) RecordWSMessage(msgType, direction string) {
	m.WSMessages.WithLabelValues(msgType, direction).Inc()
}

// RecordWSMessageLatency 记录 WebSocket 消息延迟
func (m *Metrics) RecordWSMessageLatency(msgType string, latency float64) {
	m.WSMessageLatency.WithLabelValues(msgType).Observe(latency)
}

// RecordWSError 记录 WebSocket 错误
func (m *Metrics) RecordWSError(errorType string) {
	m.WSErrors.WithLabelValues(errorType).Inc()
}

// SetAuctionSuccessRate 设置竞拍成功率
func (m *Metrics) SetAuctionSuccessRate(timeWindow string, rate float64) {
	m.AuctionSuccessRate.WithLabelValues(timeWindow).Set(rate)
}

// SetPaymentSuccessRate 设置支付成功率
func (m *Metrics) SetPaymentSuccessRate(method string, rate float64) {
	m.PaymentRate.WithLabelValues(method).Set(rate)
}
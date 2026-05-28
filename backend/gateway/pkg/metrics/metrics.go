package metrics

import (
	"net/http"
	"strconv"

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
	AuctionCreated      *prometheus.CounterVec
	AuctionBidTotal     *prometheus.CounterVec
	AuctionBidAmount    *prometheus.HistogramVec
	AuctionCompleted    *prometheus.CounterVec
	AuctionSuccessRate  *prometheus.GaugeVec

	// 订单/成交
	OrderCreated   *prometheus.CounterVec
	OrderCompleted *prometheus.CounterVec
	OrderAmount    *prometheus.HistogramVec
	OrderSuccessRate prometheus.Gauge

	// 用户
	UserRegister *prometheus.CounterVec
	UserLogin    *prometheus.CounterVec
	UserActive   prometheus.Gauge

	// WebSocket
	WSConnections   prometheus.Gauge
	WSMessages      *prometheus.CounterVec
	WSErrors        *prometheus.CounterVec

	// 支付
	PaymentInitiated *prometheus.CounterVec
	PaymentCompleted *prometheus.CounterVec
	PaymentFailed    *prometheus.CounterVec
	PaymentAmount    *prometheus.HistogramVec

	// A/B 测试实验
	ExperimentAssigned  *prometheus.CounterVec
	ExperimentViewed    *prometheus.CounterVec
	ExperimentCompleted *prometheus.CounterVec

	// SQL 查询
	SQLQueryDuration *prometheus.HistogramVec
	SQLQueryTotal    *prometheus.CounterVec
	SQLQueryErrors   *prometheus.CounterVec
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

			// A/B 测试实验指标
			ExperimentAssigned: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "experiment_assigned_total",
					Help: "实验分配总数",
				},
				[]string{"experiment", "variation"},
			),

			ExperimentViewed: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "experiment_viewed_total",
					Help: "实验查看总数",
				},
				[]string{"experiment", "variation"},
			),

			ExperimentCompleted: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "experiment_completed_total",
					Help: "实验完成总数",
				},
				[]string{"experiment", "variation"},
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
		m.OrderCreated,
		m.OrderCompleted,
		m.OrderAmount,
		m.OrderSuccessRate,
		m.UserRegister,
		m.UserLogin,
		m.UserActive,
		m.WSConnections,
		m.WSMessages,
		m.WSErrors,
		m.PaymentInitiated,
		m.PaymentCompleted,
		m.PaymentFailed,
		m.PaymentAmount,
		m.ExperimentAssigned,
		m.ExperimentViewed,
		m.ExperimentCompleted,
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
	m.RequestsTotal.WithLabelValues(service, method, path, strconv.Itoa(status)).Inc()
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

// RecordOrderCompleted 记录订单完成（成交）
func (m *Metrics) RecordOrderCompleted(auctionID, productID string, amount float64) {
	m.OrderCompleted.WithLabelValues(auctionID, productID).Inc()
	m.OrderAmount.WithLabelValues("completed").Observe(amount)
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

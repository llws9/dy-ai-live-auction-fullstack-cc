package dbmetrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// SQLMetrics SQL 查询 Prometheus 指标
type SQLMetrics struct {
	QueryDuration *prometheus.HistogramVec
	QueryTotal    *prometheus.CounterVec
	QueryErrors   *prometheus.CounterVec

	once sync.Once
}

var defaultSQLMetrics *SQLMetrics

// InitSQLMetrics 初始化 SQL 指标
func InitSQLMetrics(serviceName string) *SQLMetrics {
	defaultSQLMetrics = &SQLMetrics{
		QueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "sql_query_duration_seconds",
				Help:    "SQL查询耗时分布",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5},
			},
			[]string{"service", "operation", "table"},
		),

		QueryTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "sql_query_total",
				Help: "SQL查询总数",
			},
			[]string{"service", "operation", "table"},
		),

		QueryErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "sql_query_errors_total",
				Help: "SQL查询错误总数",
			},
			[]string{"service", "operation", "table", "error"},
		),
	}

	// 注册指标
	prometheus.MustRegister(
		defaultSQLMetrics.QueryDuration,
		defaultSQLMetrics.QueryTotal,
		defaultSQLMetrics.QueryErrors,
	)

	return defaultSQLMetrics
}

// GetSQLMetrics 获取默认 SQL 指标实例
func GetSQLMetrics() *SQLMetrics {
	return defaultSQLMetrics
}

// RecordSQLQuery 实现 MetricsRecorder 接口
func (m *SQLMetrics) RecordSQLQuery(service, operation, table string, duration float64, err error) {
	m.QueryDuration.WithLabelValues(service, operation, table).Observe(duration)
	m.QueryTotal.WithLabelValues(service, operation, table).Inc()

	if err != nil {
		errorType := "unknown"
		if err.Error() != "" {
			// 简化错误类型
			if len(err.Error()) > 50 {
				errorType = err.Error()[:50]
			} else {
				errorType = err.Error()
			}
		}
		m.QueryErrors.WithLabelValues(service, operation, table, errorType).Inc()
	}
}

// RecordQuery 便捷方法：记录查询
func (m *SQLMetrics) RecordQuery(service, operation, table string, duration float64) {
	m.QueryDuration.WithLabelValues(service, operation, table).Observe(duration)
	m.QueryTotal.WithLabelValues(service, operation, table).Inc()
}

// RecordError 便捷方法：记录错误
func (m *SQLMetrics) RecordError(service, operation, table, errorType string) {
	m.QueryErrors.WithLabelValues(service, operation, table, errorType).Inc()
}
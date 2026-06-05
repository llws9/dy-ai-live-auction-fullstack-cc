package metrics

import "github.com/prometheus/client_golang/prometheus"

type NotificationMetrics struct {
	hotPullTotal *prometheus.CounterVec
}

var notificationMetrics *NotificationMetrics

func NewNotificationMetrics(reg prometheus.Registerer) *NotificationMetrics {
	m := &NotificationMetrics{
		hotPullTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notification_hot_pull_total",
				Help: "热拉通知处理结果总数",
			},
			[]string{"source", "result"},
		),
	}
	reg.MustRegister(m.hotPullTotal)
	return m
}

func InitNotificationMetrics() *NotificationMetrics {
	m := NewNotificationMetrics(prometheus.DefaultRegisterer)
	notificationMetrics = m
	return m
}

// GetNotificationMetrics 获取通知指标实例
func GetNotificationMetrics() *NotificationMetrics {
	if notificationMetrics == nil {
		// 如果未初始化，返回一个空实例避免nil pointer
		return InitNotificationMetrics()
	}
	return notificationMetrics
}

func (m *NotificationMetrics) RecordHotPull(source, result string, count int) {
	if count <= 0 {
		return
	}
	m.hotPullTotal.WithLabelValues(source, result).Add(float64(count))
}

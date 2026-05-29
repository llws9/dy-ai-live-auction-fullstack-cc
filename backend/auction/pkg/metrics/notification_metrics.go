package metrics

// NotificationMetrics 通知服务指标收集器（占位）
// 当前通知服务不需要特定的metrics打点
// 保留此文件以备将来扩展
type NotificationMetrics struct {
	// 未来可根据需要添加：
	// - notificationsSent
	// - notificationsFailed
	// - notificationLatency
	// - batchNotificationSize
}

var notificationMetrics *NotificationMetrics

// InitNotificationMetrics 初始化通知指标（占位）
func InitNotificationMetrics() *NotificationMetrics {
	m := &NotificationMetrics{}
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
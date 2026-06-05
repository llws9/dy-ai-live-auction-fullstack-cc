package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestNotificationMetricsRecordsHotPullResults(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewNotificationMetrics(reg)

	m.RecordHotPull("product_reminder", "candidates", 3)
	m.RecordHotPull("product_reminder", "created", 2)
	m.RecordHotPull("product_reminder", "duplicates", 1)
	m.RecordHotPull("product_reminder", "failures", 1)

	require.NoError(t, testutil.CollectAndCompare(m.hotPullTotal, strings.NewReader(`
# HELP notification_hot_pull_total 热拉通知处理结果总数
# TYPE notification_hot_pull_total counter
notification_hot_pull_total{result="candidates",source="product_reminder"} 3
notification_hot_pull_total{result="created",source="product_reminder"} 2
notification_hot_pull_total{result="duplicates",source="product_reminder"} 1
notification_hot_pull_total{result="failures",source="product_reminder"} 1
`)))
}

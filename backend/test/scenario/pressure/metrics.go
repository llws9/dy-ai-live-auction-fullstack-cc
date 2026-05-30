package pressure

import (
	"sync"
	"sync/atomic"
	"time"
)

// 桶上界（spec §M2.2 定义）
var bucketBounds = []time.Duration{
	1 * time.Millisecond,
	5 * time.Millisecond,
	10 * time.Millisecond,
	50 * time.Millisecond,
	100 * time.Millisecond,
	500 * time.Millisecond,
	1 * time.Second,
	5 * time.Second,
	// 最后一个桶用 +∞ 表示，单独存放在 overflow 槽
}

// Snapshot 一次性快照
type Snapshot struct {
	Total      int64           `json:"total"`
	Success    int64           `json:"success"`
	Failure    int64           `json:"failure"`
	QPS        float64         `json:"qps"`
	Avg        time.Duration   `json:"avg"`
	P50        time.Duration   `json:"p50"`
	P95        time.Duration   `json:"p95"`
	P99        time.Duration   `json:"p99"`
	ErrorCodes map[int]int64   `json:"error_codes"`
	Buckets    []BucketSnap    `json:"buckets"`
	ElapsedMs  int64           `json:"elapsed_ms"`
}

// BucketSnap 单个桶的快照（上界与计数）
type BucketSnap struct {
	UpperMs int64 `json:"upper_ms"` // -1 表示 +∞
	Count   int64 `json:"count"`
}

// Metrics 并发安全的指标采集器
//   - 计数器：原子
//   - 桶式直方图：原子（每桶一个 int64）
//   - 错误码分布：mu 保护的 map
type Metrics struct {
	startedAt time.Time

	total   int64
	success int64
	failure int64
	sumNs   int64

	buckets  []int64 // len == len(bucketBounds)
	overflow int64   // 大于最后一个上界的样本

	mu         sync.Mutex
	errorCodes map[int]int64
}

// NewMetrics 构造
func NewMetrics() *Metrics {
	return &Metrics{
		startedAt:  time.Now(),
		buckets:    make([]int64, len(bucketBounds)),
		errorCodes: make(map[int]int64),
	}
}

// RecordSuccess 记录一次成功
func (m *Metrics) RecordSuccess(latency time.Duration) {
	atomic.AddInt64(&m.total, 1)
	atomic.AddInt64(&m.success, 1)
	atomic.AddInt64(&m.sumNs, int64(latency))
	m.bumpBucket(latency)
}

// RecordFailure 记录一次失败 + HTTP/业务错误码
func (m *Metrics) RecordFailure(latency time.Duration, code int) {
	atomic.AddInt64(&m.total, 1)
	atomic.AddInt64(&m.failure, 1)
	atomic.AddInt64(&m.sumNs, int64(latency))
	m.bumpBucket(latency)

	m.mu.Lock()
	m.errorCodes[code]++
	m.mu.Unlock()
}

// bumpBucket 落桶（线性查找，桶数量固定 8 个，常数时间）
func (m *Metrics) bumpBucket(d time.Duration) {
	for i, ub := range bucketBounds {
		if d <= ub {
			atomic.AddInt64(&m.buckets[i], 1)
			return
		}
	}
	atomic.AddInt64(&m.overflow, 1)
}

// Snapshot 生成一次性快照
func (m *Metrics) Snapshot() Snapshot {
	total := atomic.LoadInt64(&m.total)
	success := atomic.LoadInt64(&m.success)
	failure := atomic.LoadInt64(&m.failure)
	sumNs := atomic.LoadInt64(&m.sumNs)
	elapsed := time.Since(m.startedAt)

	// 桶 + overflow → 累计直方图
	cnts := make([]int64, len(m.buckets)+1)
	var cum int64
	for i := range m.buckets {
		c := atomic.LoadInt64(&m.buckets[i])
		cum += c
		cnts[i] = cum
	}
	overflow := atomic.LoadInt64(&m.overflow)
	cum += overflow
	cnts[len(m.buckets)] = cum

	avg := time.Duration(0)
	if total > 0 {
		avg = time.Duration(sumNs / total)
	}
	qps := float64(0)
	if elapsed > 0 {
		qps = float64(total) / elapsed.Seconds()
	}

	bucketSnaps := make([]BucketSnap, 0, len(bucketBounds)+1)
	for i, ub := range bucketBounds {
		bucketSnaps = append(bucketSnaps, BucketSnap{
			UpperMs: ub.Milliseconds(),
			Count:   atomic.LoadInt64(&m.buckets[i]),
		})
	}
	bucketSnaps = append(bucketSnaps, BucketSnap{UpperMs: -1, Count: overflow})

	m.mu.Lock()
	codes := make(map[int]int64, len(m.errorCodes))
	for k, v := range m.errorCodes {
		codes[k] = v
	}
	m.mu.Unlock()

	return Snapshot{
		Total:      total,
		Success:    success,
		Failure:    failure,
		QPS:        qps,
		Avg:        avg,
		P50:        percentile(cnts, total, 0.50),
		P95:        percentile(cnts, total, 0.95),
		P99:        percentile(cnts, total, 0.99),
		ErrorCodes: codes,
		Buckets:    bucketSnaps,
		ElapsedMs:  elapsed.Milliseconds(),
	}
}

// percentile 在累计直方图上求指定百分位（返回桶上界，保守估计）
//   cumulative[i] = 落在 bucketBounds[0..i] 内的总数
//   cumulative[len-1] = 包含 overflow 的总数
//   返回首个 cumulative[i] >= ceil(total * p) 对应的上界
//   overflow 桶（i == len(bucketBounds)）返回 +∞ 的近似（最后桶的 2x，但更通常做 ub_last）
func percentile(cumulative []int64, total int64, p float64) time.Duration {
	if total == 0 {
		return 0
	}
	target := int64(float64(total)*p + 0.5)
	if target < 1 {
		target = 1
	}
	for i, c := range cumulative {
		if c >= target {
			if i < len(bucketBounds) {
				return bucketBounds[i]
			}
			// overflow：用最后桶的 ×2 作为近似上界（避免无穷大不可序列化）
			return bucketBounds[len(bucketBounds)-1] * 2
		}
	}
	// 不应该到这里
	return bucketBounds[len(bucketBounds)-1] * 2
}

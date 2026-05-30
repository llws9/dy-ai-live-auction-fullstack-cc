package pressure

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"

	"test-service/runner"
)

// Bidder 客户端抽象，便于注入桩
type Bidder interface {
	PlaceBid(ctx context.Context, amount float64, auctionID, userID int64) Result
}

// ClientFactory 提供 Bidder 实例
type ClientFactory interface {
	NewClient() Bidder
}

// Config 压测场景配置
type Config struct {
	ConcurrentUsers int     `json:"concurrent_users"` // 并发用户数
	DurationSec     int     `json:"duration_sec"`     // 持续秒数
	TargetAuctionID int64   `json:"target_auction_id"`
	BidAmount       float64 `json:"bid_amount"`
	EmitIntervalMs  int     `json:"emit_interval_ms"` // 上报间隔，默认 1000ms
}

// Result 场景跑完后的总报告
type Report struct {
	Snapshot Snapshot `json:"snapshot"`
}

// Scenario 压测场景
type Scenario struct {
	cf ClientFactory
}

// New 构造
func New(cf ClientFactory) *Scenario {
	return &Scenario{cf: cf}
}

// Type 实现 runner.Scenario
func (s *Scenario) Type() string { return "pressure" }

// Run 实现 runner.Scenario
//   流程：
//     1. 启动 N 个 worker goroutine（pool）持续打 bid
//     2. ticker 每 EmitIntervalMs 上报一次实时指标
//     3. context 到期或被取消 → 关闭 stop chan → worker 退出 → 等待
//     4. 最后 emit progress=100 + 总报告
func (s *Scenario) Run(ctx context.Context, cfgRaw json.RawMessage, p runner.ProgressEmitter) (any, error) {
	var cfg Config
	if err := json.Unmarshal(cfgRaw, &cfg); err != nil {
		return nil, err
	}
	if cfg.ConcurrentUsers <= 0 {
		cfg.ConcurrentUsers = 10
	}
	if cfg.DurationSec <= 0 {
		cfg.DurationSec = 10
	}
	if cfg.BidAmount <= 0 {
		cfg.BidAmount = 100
	}
	if cfg.EmitIntervalMs <= 0 {
		cfg.EmitIntervalMs = 1000
	}

	hlog.Infof("[pressure] start users=%d duration=%ds auction=%d", cfg.ConcurrentUsers, cfg.DurationSec, cfg.TargetAuctionID)

	metrics := NewMetrics()
	client := s.cf.NewClient()
	totalDuration := time.Duration(cfg.DurationSec) * time.Second

	runCtx, runCancel := context.WithTimeout(ctx, totalDuration)
	defer runCancel()

	// 启动 N 个 worker
	var wg sync.WaitGroup
	for i := 0; i < cfg.ConcurrentUsers; i++ {
		wg.Add(1)
		userID := int64(100000 + i) // 测试用户 ID
		go func() {
			defer wg.Done()
			for {
				select {
				case <-runCtx.Done():
					return
				default:
				}
				res := client.PlaceBid(runCtx, cfg.BidAmount, cfg.TargetAuctionID, userID)
				if res.OK {
					metrics.RecordSuccess(res.Latency)
				} else {
					metrics.RecordFailure(res.Latency, res.StatusCode)
				}
			}
		}()
	}

	// ticker 上报
	tickerDone := make(chan struct{})
	go func() {
		defer close(tickerDone)
		t := time.NewTicker(time.Duration(cfg.EmitIntervalMs) * time.Millisecond)
		defer t.Stop()
		startedAt := time.Now()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-t.C:
				elapsed := time.Since(startedAt)
				prog := int(float64(elapsed) / float64(totalDuration) * 100)
				if prog > 99 {
					prog = 99
				}
				snap := metrics.Snapshot()
				p.Emit(prog, "running", snapToMap(snap))
			}
		}
	}()

	wg.Wait()
	<-tickerDone

	final := metrics.Snapshot()
	p.Emit(100, "done", snapToMap(final))
	hlog.Infof("[pressure] done total=%d success=%d failure=%d qps=%.2f p99=%v",
		final.Total, final.Success, final.Failure, final.QPS, final.P99)

	// 取消是预期路径，不抛错
	if ctx.Err() != nil && ctx.Err() == context.Canceled {
		return Report{Snapshot: final}, nil
	}
	return Report{Snapshot: final}, nil
}

// snapToMap 转 emit metrics map（前端友好的 ms 单位）
func snapToMap(s Snapshot) map[string]any {
	return map[string]any{
		"qps":         s.QPS,
		"avg_ms":      s.Avg.Milliseconds(),
		"p50_ms":      s.P50.Milliseconds(),
		"p95_ms":      s.P95.Milliseconds(),
		"p99_ms":      s.P99.Milliseconds(),
		"total":       s.Total,
		"success":     s.Success,
		"failure":     s.Failure,
		"error_codes": s.ErrorCodes,
		"buckets":     s.Buckets,
		"elapsed_ms":  s.ElapsedMs,
	}
}

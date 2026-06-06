package pressure

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
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

type Fixture struct {
	AuctionID  int64
	AuctionIDs []int64
}

type FixturePreparer interface {
	PrepareFixture(ctx context.Context, cfg Config) (Fixture, error)
}

// Config 压测场景配置
type Config struct {
	ConcurrentUsers int     `json:"concurrent_users"` // 并发用户数
	DurationSec     int     `json:"duration_sec"`     // 持续秒数
	Scenario        string  `json:"scenario"`         // hot_auction | throughput
	TargetAuctionID int64   `json:"target_auction_id"`
	FixtureCount    int     `json:"fixture_count"`
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
//
//	流程：
//	  1. 启动 N 个 worker goroutine（pool）持续打 bid
//	  2. ticker 每 EmitIntervalMs 上报一次实时指标
//	  3. context 到期或被取消 → 关闭 stop chan → worker 退出 → 等待
//	  4. 最后 emit progress=100 + 总报告
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
	if cfg.Scenario == "" {
		cfg.Scenario = "hot_auction"
	}
	if cfg.EmitIntervalMs <= 0 {
		cfg.EmitIntervalMs = 1000
	}

	fixtureCreated := false
	auctionIDs := []int64{cfg.TargetAuctionID}
	if preparer, ok := s.cf.(FixturePreparer); ok {
		fixture, err := preparer.PrepareFixture(ctx, cfg)
		if err != nil {
			return nil, err
		}
		auctionIDs = normalizeAuctionIDs(fixture)
		if len(auctionIDs) == 0 {
			return nil, fmt.Errorf("pressure fixture returned no valid auction ids")
		}
		cfg.TargetAuctionID = auctionIDs[0]
		fixtureCreated = true
	} else if cfg.TargetAuctionID <= 0 {
		return nil, fmt.Errorf("target_auction_id is required when fixture preparer is unavailable")
	} else {
		auctionIDs = []int64{cfg.TargetAuctionID}
	}

	hlog.Infof("[pressure] start scenario=%s users=%d duration=%ds auctions=%d primary_auction=%d",
		cfg.Scenario, cfg.ConcurrentUsers, cfg.DurationSec, len(auctionIDs), cfg.TargetAuctionID)

	metrics := NewMetrics()
	client := s.cf.NewClient()
	totalDuration := time.Duration(cfg.DurationSec) * time.Second

	runCtx, runCancel := context.WithTimeout(ctx, totalDuration)
	defer runCancel()

	// 启动 N 个 worker
	var wg sync.WaitGroup
	var bidSeq int64
	var transportErrSamples int64
	for i := 0; i < cfg.ConcurrentUsers; i++ {
		wg.Add(1)
		userID := int64(100000 + i) // 测试用户 ID
		auctionID := auctionIDs[i%len(auctionIDs)]
		go func() {
			defer wg.Done()
			for {
				select {
				case <-runCtx.Done():
					return
				default:
				}
				amount := cfg.BidAmount + float64(atomic.AddInt64(&bidSeq, 1))
				res := client.PlaceBid(runCtx, amount, auctionID, userID)
				if shouldIgnoreStopResult(runCtx, res) {
					return
				}
				if res.OK {
					metrics.RecordSuccess(res.Latency)
				} else {
					if res.StatusCode == 0 && res.Err != nil && atomic.AddInt64(&transportErrSamples, 1) <= 5 {
						hlog.Warnf("[pressure] transport error sample scenario=%s auction_id=%d user_id=%d latency=%s err=%v",
							cfg.Scenario, auctionID, userID, res.Latency, res.Err)
					}
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
				p.Emit(prog, "running", snapToMap(snap, cfg, auctionIDs, fixtureCreated))
			}
		}
	}()

	wg.Wait()
	<-tickerDone

	final := metrics.Snapshot()
	p.Emit(100, "done", snapToMap(final, cfg, auctionIDs, fixtureCreated))
	hlog.Infof("[pressure] done total=%d success=%d failure=%d qps=%.2f p99=%v",
		final.Total, final.Success, final.Failure, final.QPS, final.P99)

	// 取消是预期路径，不抛错
	if ctx.Err() != nil && ctx.Err() == context.Canceled {
		return Report{Snapshot: final}, nil
	}
	return Report{Snapshot: final}, nil
}

func shouldIgnoreStopResult(ctx context.Context, res Result) bool {
	return ctx.Err() != nil
}

// snapToMap 转 emit metrics map（前端友好的 ms 单位）
func snapToMap(s Snapshot, cfg Config, auctionIDs []int64, fixtureCreated bool) map[string]any {
	return map[string]any{
		"qps":               s.QPS,
		"avg_ms":            s.Avg.Milliseconds(),
		"p50_ms":            s.P50.Milliseconds(),
		"p95_ms":            s.P95.Milliseconds(),
		"p99_ms":            s.P99.Milliseconds(),
		"total":             s.Total,
		"success":           s.Success,
		"failure":           s.Failure,
		"error_codes":       s.ErrorCodes,
		"buckets":           s.Buckets,
		"elapsed_ms":        s.ElapsedMs,
		"scenario":          cfg.Scenario,
		"target_auction_id": cfg.TargetAuctionID,
		"fixture_count":     len(auctionIDs),
		"fixture_created":   fixtureCreated,
	}
}

func normalizeAuctionIDs(fixture Fixture) []int64 {
	seen := map[int64]struct{}{}
	ids := make([]int64, 0, len(fixture.AuctionIDs)+1)
	if fixture.AuctionID > 0 {
		seen[fixture.AuctionID] = struct{}{}
		ids = append(ids, fixture.AuctionID)
	}
	for _, id := range fixture.AuctionIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

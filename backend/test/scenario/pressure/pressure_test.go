package pressure

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// stubEmitter 收集 emit 调用
type stubEmitter struct {
	count    int32
	lastProg int32
	last     map[string]any
}

func (e *stubEmitter) Emit(progress int, step string, metrics map[string]any) {
	atomic.AddInt32(&e.count, 1)
	atomic.StoreInt32(&e.lastProg, int32(progress))
	e.last = metrics
}

// TestScenario_Type 类型必须为 "pressure"
func TestScenario_Type(t *testing.T) {
	s := New(&stubClientFactory{})
	if s.Type() != "pressure" {
		t.Fatalf("Type: want pressure, got %s", s.Type())
	}
}

// TestScenario_RunsAndEmits 跑一段短时间，应该有多次 emit，最后 progress=100
func TestScenario_RunsAndEmits(t *testing.T) {
	cf := stubClientFactory{}
	s := New(&cf)

	cfg := Config{
		ConcurrentUsers: 4,
		DurationSec:     2,
		TargetAuctionID: 1001,
		BidAmount:       100,
		EmitIntervalMs:  500,
	}
	raw, _ := json.Marshal(cfg)

	em := &stubEmitter{}
	res, err := s.Run(context.Background(), raw, em)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if res == nil {
		t.Fatalf("expected result, got nil")
	}
	if cf.created != 1 {
		t.Fatalf("fixture should be created once, got %d", cf.created)
	}
	// 至少 emit 过一次
	if c := atomic.LoadInt32(&em.count); c < 2 {
		t.Fatalf("emit count: want >= 2, got %d", c)
	}
	if p := atomic.LoadInt32(&em.lastProg); p != 100 {
		t.Fatalf("final progress: want 100, got %d", p)
	}
	// 指标字段齐全
	if em.last == nil {
		t.Fatalf("last metrics should not be nil")
	}
	for _, k := range []string{"qps", "p99_ms", "p95_ms", "avg_ms", "total", "success", "failure"} {
		if _, ok := em.last[k]; !ok {
			t.Fatalf("metrics missing key: %s", k)
		}
	}
	if got := em.last["target_auction_id"]; got != int64(2002) {
		t.Fatalf("target_auction_id should come from fixture, got %v", got)
	}
	if got := em.last["fixture_created"]; got != true {
		t.Fatalf("fixture_created should be true, got %v", got)
	}
	if got := em.last["scenario"]; got != "hot_auction" {
		t.Fatalf("scenario should default to hot_auction, got %v", got)
	}
	cf.bidder.mu.Lock()
	defer cf.bidder.mu.Unlock()
	if len(cf.bidder.amounts) == 0 {
		t.Fatalf("expected bids to be sent")
	}
	for _, amount := range cf.bidder.amounts {
		if amount <= cfg.BidAmount {
			t.Fatalf("bid amount should increase above base amount, got %.2f", amount)
		}
	}
}

func TestScenario_ThroughputUsesFixtureShards(t *testing.T) {
	cf := stubClientFactory{}
	s := New(&cf)

	cfg := Config{
		ConcurrentUsers: 4,
		DurationSec:     1,
		Scenario:        "throughput",
		BidAmount:       100,
		EmitIntervalMs:  200,
	}
	raw, _ := json.Marshal(cfg)

	em := &stubEmitter{}
	if _, err := s.Run(context.Background(), raw, em); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if got := em.last["fixture_count"]; got != 2 {
		t.Fatalf("fixture_count should come from fixture shards, got %v", got)
	}
	cf.bidder.mu.Lock()
	defer cf.bidder.mu.Unlock()
	seen := map[int64]bool{}
	for _, auctionID := range cf.bidder.auctionIDs {
		seen[auctionID] = true
	}
	if !seen[3001] || !seen[3002] {
		t.Fatalf("expected bids to be distributed to throughput fixture shards, got %v", seen)
	}
}

func TestScenario_IgnoresExpectedStopCancellation(t *testing.T) {
	cf := &cancelOnStopFactory{}
	s := New(cf)

	cfg := Config{
		ConcurrentUsers: 4,
		DurationSec:     1,
		TargetAuctionID: 2002,
		BidAmount:       100,
		EmitIntervalMs:  200,
	}
	raw, _ := json.Marshal(cfg)

	em := &stubEmitter{}
	res, err := s.Run(context.Background(), raw, em)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	report := res.(Report)
	if report.Snapshot.Failure != 0 {
		t.Fatalf("expected stop cancellation not to be counted as failure, got %d", report.Snapshot.Failure)
	}
	if got := report.Snapshot.ErrorCodes[0]; got != 0 {
		t.Fatalf("expected error code 0 to be omitted for expected stop cancellation, got %d", got)
	}
}

func TestScenario_IgnoresResponsesCompletedAfterStop(t *testing.T) {
	cf := &respondAfterStopFactory{}
	s := New(cf)

	cfg := Config{
		ConcurrentUsers: 4,
		DurationSec:     1,
		TargetAuctionID: 2002,
		BidAmount:       100,
		EmitIntervalMs:  200,
	}
	raw, _ := json.Marshal(cfg)

	res, err := s.Run(context.Background(), raw, &stubEmitter{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	report := res.(Report)
	if report.Snapshot.Failure != 0 {
		t.Fatalf("expected responses after stop not to be counted, got %d", report.Snapshot.Failure)
	}
	if got := report.Snapshot.ErrorCodes[500]; got != 0 {
		t.Fatalf("expected post-stop 500 to be omitted, got %d", got)
	}
}

// TestScenario_Cancellation 提前取消 context，应该尽快退出且 progress 接近 100
func TestScenario_Cancellation(t *testing.T) {
	s := New(&stubClientFactory{})
	cfg := Config{
		ConcurrentUsers: 4,
		DurationSec:     30, // 故意设长
		TargetAuctionID: 1,
		BidAmount:       100,
		EmitIntervalMs:  100,
	}
	raw, _ := json.Marshal(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	em := &stubEmitter{}
	start := time.Now()
	_, err := s.Run(ctx, raw, em)
	cost := time.Since(start)

	// 取消应在 1s 内退出
	if cost > 2*time.Second {
		t.Fatalf("cancellation too slow: %v", cost)
	}
	// 取消是预期路径，不应抛错（场景内部消化）
	if err != nil && err != context.Canceled {
		t.Fatalf("unexpected err: %v", err)
	}
}

// stubClientFactory 注入桩客户端
type stubClientFactory struct {
	created int
	bidder  *stubBidder
}

func (f *stubClientFactory) NewClient() Bidder {
	if f.bidder == nil {
		f.bidder = &stubBidder{}
	}
	return f.bidder
}

func (f *stubClientFactory) PrepareFixture(ctx context.Context, cfg Config) (Fixture, error) {
	f.created++
	if cfg.Scenario == "throughput" {
		return Fixture{AuctionID: 3001, AuctionIDs: []int64{3001, 3002}}, nil
	}
	return Fixture{AuctionID: 2002}, nil
}

type stubBidder struct {
	mu         sync.Mutex
	amounts    []float64
	auctionIDs []int64
}

func (b *stubBidder) PlaceBid(ctx context.Context, amount float64, auctionID, userID int64) Result {
	b.mu.Lock()
	b.amounts = append(b.amounts, amount)
	b.auctionIDs = append(b.auctionIDs, auctionID)
	b.mu.Unlock()
	if auctionID != 2002 && auctionID != 3001 && auctionID != 3002 {
		return Result{OK: false, StatusCode: 404}
	}
	// 模拟 1ms 延迟成功
	time.Sleep(time.Millisecond)
	return Result{OK: true, StatusCode: 200, Latency: time.Millisecond}
}

type cancelOnStopFactory struct{}

func (f *cancelOnStopFactory) NewClient() Bidder {
	return cancelOnStopBidder{}
}

type cancelOnStopBidder struct{}

func (b cancelOnStopBidder) PlaceBid(ctx context.Context, amount float64, auctionID, userID int64) Result {
	start := time.Now()
	<-ctx.Done()
	return Result{OK: false, Latency: time.Since(start), Err: ctx.Err()}
}

type respondAfterStopFactory struct{}

func (f *respondAfterStopFactory) NewClient() Bidder {
	return respondAfterStopBidder{}
}

type respondAfterStopBidder struct{}

func (b respondAfterStopBidder) PlaceBid(ctx context.Context, amount float64, auctionID, userID int64) Result {
	start := time.Now()
	<-ctx.Done()
	return Result{OK: false, StatusCode: 500, Latency: time.Since(start)}
}

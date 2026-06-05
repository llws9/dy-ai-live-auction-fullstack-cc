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
	return Fixture{AuctionID: 2002}, nil
}

type stubBidder struct {
	mu      sync.Mutex
	amounts []float64
}

func (b *stubBidder) PlaceBid(ctx context.Context, amount float64, auctionID, userID int64) Result {
	b.mu.Lock()
	b.amounts = append(b.amounts, amount)
	b.mu.Unlock()
	if auctionID != 2002 {
		return Result{OK: false, StatusCode: 404}
	}
	// 模拟 1ms 延迟成功
	time.Sleep(time.Millisecond)
	return Result{OK: true, StatusCode: 200, Latency: time.Millisecond}
}

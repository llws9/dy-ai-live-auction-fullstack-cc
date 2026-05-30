// Package antisnipe 场景 F：防狙击延时机制测试。
//
// 本文件覆盖 simulator 的纯算法部分（非阻塞、可注入桩客户端）：
//   - 出价节奏控制
//   - 末段窗口判定
//   - delay_used 时间轴采样
//   - 安全期出价不应触发延时
package antisnipe

import (
	"context"
	"sync"
	"testing"
	"time"

	"test-service/client/auction"
)

// ---------- 桩 ----------

// fakeClock 可控时钟
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// fakeAuctionAPI 模拟 auction-service 防狙击行为
type fakeAuctionAPI struct {
	mu sync.Mutex

	// 拍卖快照
	auction auction.Auction

	// 防狙击规则参数
	triggerDelayBefore time.Duration
	delayPerBid        time.Duration
	maxDelay           time.Duration

	// 计数
	bidCalls int
	getCalls int

	// 强制每次出价失败（如已封顶）
	bidFail bool

	// 时钟
	clock *fakeClock

	// 出价时间轴
	bidLog []bidLogEntry
}

type bidLogEntry struct {
	at        time.Time
	userID    int64
	delayUsed int
	endTime   time.Time
	ok        bool
}

func newFakeAPI(clock *fakeClock, startEnd time.Time) *fakeAuctionAPI {
	return &fakeAuctionAPI{
		clock: clock,
		auction: auction.Auction{
			ID:        8001,
			ProductID: 9001,
			Status:    1, // Ongoing
			EndTime:   startEnd,
			StartTime: clock.Now(),
		},
		triggerDelayBefore: 5 * time.Second,
		delayPerBid:        2 * time.Second,
		maxDelay:           10 * time.Second,
	}
}

func (f *fakeAuctionAPI) PlaceBid(ctx context.Context, userID, auctionID int64, amount float64) auction.StepResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bidCalls++

	if f.bidFail {
		f.bidLog = append(f.bidLog, bidLogEntry{
			at: f.clock.Now(), userID: userID,
			delayUsed: f.auction.DelayUsed, endTime: f.auction.EndTime, ok: false,
		})
		return auction.StepResult{Step: "bid", OK: false, RefID: userID, Message: "bid failed"}
	}

	now := f.clock.Now()
	remaining := f.auction.EndTime.Sub(now)

	// 末段窗口内 + 未达上限 → 触发延时
	if remaining > 0 && remaining <= f.triggerDelayBefore {
		usedDur := time.Duration(f.auction.DelayUsed) * time.Second
		if usedDur < f.maxDelay {
			grant := f.delayPerBid
			if usedDur+grant > f.maxDelay {
				grant = f.maxDelay - usedDur
			}
			f.auction.EndTime = f.auction.EndTime.Add(grant)
			f.auction.DelayUsed += int(grant / time.Second)
			f.auction.Status = 2 // Delayed
		}
	}
	f.auction.CurrentPrice = amount
	f.auction.WinnerID = userID

	f.bidLog = append(f.bidLog, bidLogEntry{
		at: now, userID: userID,
		delayUsed: f.auction.DelayUsed, endTime: f.auction.EndTime, ok: true,
	})
	return auction.StepResult{Step: "bid", OK: true, RefID: userID, StatusCode: 200}
}

func (f *fakeAuctionAPI) GetAuction(ctx context.Context, auctionID int64) (auction.Auction, auction.StepResult) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls++
	a := f.auction
	return a, auction.StepResult{Step: "get_auction", OK: true, StatusCode: 200}
}

// ---------- 用例 ----------

// TestSimulator_LastSecondBidsTriggerDelay：末段窗口内出价应触发延时累计
func TestSimulator_LastSecondBidsTriggerDelay(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)}
	originalEnd := clock.Now().Add(30 * time.Second)
	api := newFakeAPI(clock, originalEnd)

	sim := NewSimulator(api, Config{
		AuctionID:       8001,
		BidderIDs:       []int64{1001, 1002, 1003},
		BidIntervalMs:   500,
		EndingWindowSec: 5,
		StartPrice:      100,
		Increment:       10,
		Now:             clock.Now,
		Sleep: func(d time.Duration) {
			clock.advance(d)
		},
	})

	report, err := sim.RunSimulation(context.Background())
	if err != nil {
		t.Fatalf("RunSimulation err: %v", err)
	}

	if report.TriggeredCount == 0 {
		t.Fatalf("expected at least 1 trigger, got 0; bidLog=%+v", api.bidLog)
	}
	if report.DelayUsedMs <= 0 {
		t.Fatalf("expected DelayUsedMs > 0, got %d", report.DelayUsedMs)
	}
	if report.OriginalEndTime.IsZero() {
		t.Fatalf("OriginalEndTime not recorded")
	}
	if !report.ActualEndTime.After(report.OriginalEndTime) {
		t.Fatalf("ActualEndTime should exceed OriginalEndTime: orig=%s actual=%s",
			report.OriginalEndTime, report.ActualEndTime)
	}
	if len(report.Timeline) == 0 {
		t.Fatalf("Timeline should not be empty")
	}
}

// TestSimulator_DelayCappedAtMax：延时累计应受 max_delay 上限保护
func TestSimulator_DelayCappedAtMax(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)}
	originalEnd := clock.Now().Add(6 * time.Second)
	api := newFakeAPI(clock, originalEnd)
	api.maxDelay = 4 * time.Second
	api.delayPerBid = 2 * time.Second
	api.triggerDelayBefore = 6 * time.Second

	// 用足够多的出价确保触顶
	bidders := make([]int64, 0, 20)
	for i := int64(1001); i < 1021; i++ {
		bidders = append(bidders, i)
	}
	sim := NewSimulator(api, Config{
		AuctionID:       8001,
		BidderIDs:       bidders,
		BidIntervalMs:   200,
		EndingWindowSec: 6,
		StartPrice:      100,
		Increment:       10,
		Now:             clock.Now,
		Sleep:           func(d time.Duration) { clock.advance(d) },
	})

	report, err := sim.RunSimulation(context.Background())
	if err != nil {
		t.Fatalf("RunSimulation err: %v", err)
	}

	maxAllowed := int64(api.maxDelay / time.Millisecond)
	if report.DelayUsedMs > maxAllowed {
		t.Fatalf("DelayUsedMs %d exceeded cap %d", report.DelayUsedMs, maxAllowed)
	}
	// 触顶后仍出价但不再延长
	if report.TriggeredCount < 2 {
		t.Fatalf("expected multiple triggers but got %d", report.TriggeredCount)
	}
}

// TestSimulator_SafePeriodNoDelay：安全期内出价不应触发延时
func TestSimulator_SafePeriodNoDelay(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)}
	originalEnd := clock.Now().Add(60 * time.Second)
	api := newFakeAPI(clock, originalEnd)
	api.triggerDelayBefore = 5 * time.Second // 末段只有 5s

	sim := NewSimulator(api, Config{
		AuctionID:       8001,
		BidderIDs:       []int64{1001},
		BidIntervalMs:   500,
		EndingWindowSec: 0, // 不进入末段，开局后立即出 1 笔
		StartPrice:      100,
		Increment:       10,
		Now:             clock.Now,
		Sleep:           func(d time.Duration) { clock.advance(d) },
		SafePeriodOnly:  true, // 让 simulator 在安全期内连发，不进入末段
	})

	report, err := sim.RunSimulation(context.Background())
	if err != nil {
		t.Fatalf("RunSimulation err: %v", err)
	}

	if report.TriggeredCount != 0 {
		t.Fatalf("safe-period bids should not trigger delay; got %d, log=%+v",
			report.TriggeredCount, api.bidLog)
	}
	if report.DelayUsedMs != 0 {
		t.Fatalf("expected DelayUsedMs == 0 in safe period, got %d", report.DelayUsedMs)
	}
}

// TestSimulator_AlreadyCappedNoFurtherDelay：已达延时封顶后再出价不再继续延时
func TestSimulator_AlreadyCappedNoFurtherDelay(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)}
	originalEnd := clock.Now().Add(3 * time.Second)
	api := newFakeAPI(clock, originalEnd)
	api.maxDelay = 2 * time.Second
	api.delayPerBid = 2 * time.Second
	api.triggerDelayBefore = 3 * time.Second

	// 第一次出价已用完延时；第二次出价不应再延
	sim := NewSimulator(api, Config{
		AuctionID:       8001,
		BidderIDs:       []int64{1001, 1002},
		BidIntervalMs:   200,
		EndingWindowSec: 3,
		StartPrice:      100,
		Increment:       10,
		Now:             clock.Now,
		Sleep:           func(d time.Duration) { clock.advance(d) },
	})

	_, err := sim.RunSimulation(context.Background())
	if err != nil {
		t.Fatalf("RunSimulation err: %v", err)
	}

	if api.auction.DelayUsed > int(api.maxDelay/time.Second) {
		t.Fatalf("DelayUsed %d exceeded cap %v", api.auction.DelayUsed, api.maxDelay)
	}
}

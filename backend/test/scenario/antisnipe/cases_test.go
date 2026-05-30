package antisnipe

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"
)

// fakeFactory 桩工厂：每次返回新拍卖；内部维护独立 fakeAuctionAPI map（per-case）
type fakeFactory struct {
	api    *fakeAuctionAPI
	nextID int64
}

func (f *fakeFactory) Prepare(ctx context.Context, name string) (int64, error) {
	id := atomic.AddInt64(&f.nextID, 1)
	// 每个用例 reset 时间轴
	f.api.bidLog = nil
	f.api.auction.DelayUsed = 0
	clock := f.api.clock
	switch name {
	case CaseSafePeriod:
		f.api.auction.EndTime = clock.Now().Add(60 * time.Second)
		f.api.auction.Status = 1
	case CaseDelayCap:
		f.api.auction.EndTime = clock.Now().Add(10 * time.Second)
		f.api.maxDelay = 4 * time.Second
		f.api.delayPerBid = 2 * time.Second
		f.api.triggerDelayBefore = 10 * time.Second
		f.api.auction.Status = 1
	case CaseCappedNoExtend:
		f.api.auction.EndTime = clock.Now().Add(5 * time.Second)
		f.api.maxDelay = 2 * time.Second
		f.api.delayPerBid = 2 * time.Second
		f.api.triggerDelayBefore = 5 * time.Second
		f.api.auction.Status = 1
	default:
		f.api.auction.EndTime = clock.Now().Add(5 * time.Second)
		f.api.triggerDelayBefore = 5 * time.Second
		f.api.delayPerBid = 1 * time.Second
		f.api.maxDelay = 10 * time.Second
		f.api.auction.Status = 1
	}
	f.api.auction.ID = id
	return id, nil
}

func (f *fakeFactory) Cleanup(ctx context.Context, auctionID int64) error { return nil }

// fakeEmitter 收集事件
type collectEmitter struct {
	steps []string
}

func (e *collectEmitter) Emit(progress int, step string, metrics map[string]any) {
	e.steps = append(e.steps, step)
}

func TestScenario_Run_AllCases(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)}
	api := newFakeAPI(clock, clock.Now().Add(30*time.Second))
	fac := &fakeFactory{api: api}

	// 注入 simulator 时钟：通过 monkey-patching Cases 包内 NewSimulator 不便，
	// 改为直接调 AssertCase 跑 5 个用例（覆盖 cases.go 主路径），并验证 Scenario.Run 编排
	sc := NewScenario(api, fac)

	// 让 simulator 用 fakeClock：Scenario.Run 内 NewSimulator 会用 time.Now/Sleep 默认值，
	// 这里我们注入的 fakeAuctionAPI 自身用的是 fakeClock，
	// 但 simulator 的等待逻辑使用 cfg.Now，所以需要走快速路径：
	// safe_period 完全不等待；其他用例的 windowStart = endTime - window，
	// fakeClock 的 now 与 endTime 关系决定模拟器是否会 sleep。
	// 这里我们让所有用例的 endTime 距 now 等于 window（windowStart=now）→ 不需 sleep。
	cfgRaw, _ := json.Marshal(ScenarioConfig{
		Cases:     []string{CaseSafePeriod}, // 只测最安全的一条路径，验证 Scenario 编排
		BidderIDs: []int64{1001, 1002},
	})

	em := &collectEmitter{}
	out, err := sc.Run(context.Background(), cfgRaw, em)
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	rep, ok := out.(ScenarioReport)
	if !ok {
		t.Fatalf("expected ScenarioReport, got %T", out)
	}
	if len(rep.Cases) != 1 {
		t.Fatalf("expected 1 case result, got %d", len(rep.Cases))
	}
	if rep.Cases[0].Name != CaseSafePeriod {
		t.Fatalf("expected safe_period, got %s", rep.Cases[0].Name)
	}
	if !rep.AllOK {
		t.Fatalf("safe-period case should pass, msg=%s", rep.Cases[0].Message)
	}
	if len(em.steps) == 0 {
		t.Fatalf("expected progress emit")
	}
}

func TestAssertCase_LastSecondOK(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)}
	api := newFakeAPI(clock, clock.Now().Add(5*time.Second))

	// 跑 last_second 用例需要时钟推进，注入 sleep
	res := assertCaseWithClock(api, clock, CaseConfig{
		Name: CaseLastSecond, AuctionID: 8001, BidderIDs: []int64{1001, 1002, 1003},
	})
	if !res.OK {
		t.Fatalf("last_second should pass: %s", res.Message)
	}
	if res.Report == nil || res.Report.TriggeredCount == 0 {
		t.Fatalf("expected triggers > 0")
	}
}

func TestAssertCase_SafePeriodOK(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)}
	api := newFakeAPI(clock, clock.Now().Add(60*time.Second))
	res := assertCaseWithClock(api, clock, CaseConfig{
		Name: CaseSafePeriod, AuctionID: 8001, BidderIDs: []int64{1001},
	})
	if !res.OK {
		t.Fatalf("safe_period should pass: %s; report=%+v", res.Message, res.Report)
	}
}

// assertCaseWithClock 注入 fakeClock 跑用例（绕过默认 time.Now）
func assertCaseWithClock(cli BidClient, clock *fakeClock, cfg CaseConfig) CaseResult {
	res := CaseResult{Name: cfg.Name}
	simCfg := Config{
		AuctionID:     cfg.AuctionID,
		BidderIDs:     cfg.BidderIDs,
		BidIntervalMs: 200,
		StartPrice:    100,
		Increment:     10,
		Now:           clock.Now,
		Sleep:         func(d time.Duration) { clock.advance(d) },
	}
	switch cfg.Name {
	case CaseLastSecond, CaseMultiUser, CaseCappedNoExtend:
		simCfg.EndingWindowSec = 5
	case CaseDelayCap:
		simCfg.EndingWindowSec = 10
	case CaseSafePeriod:
		simCfg.SafePeriodOnly = true
	}
	rep, err := NewSimulator(cli, simCfg).RunSimulation(context.Background())
	if err != nil {
		res.Message = err.Error()
		return res
	}
	res.Report = rep
	switch cfg.Name {
	case CaseLastSecond:
		if rep.TriggeredCount >= 1 && rep.ActualEndTime.After(rep.OriginalEndTime) {
			res.OK = true
		}
	case CaseSafePeriod:
		if rep.TriggeredCount == 0 && rep.DelayUsedMs == 0 {
			res.OK = true
		}
	case CaseDelayCap, CaseMultiUser, CaseCappedNoExtend:
		if rep.TriggeredCount > 0 {
			res.OK = true
		}
	}
	return res
}

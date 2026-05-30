package cron

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakePurger 模拟 ResultDAO.DeleteOlderThan
type fakePurger struct {
	mu      sync.Mutex
	calls   int32
	cutoffs []time.Time
	rows    int64
	err     error
}

func (p *fakePurger) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	atomic.AddInt32(&p.calls, 1)
	p.mu.Lock()
	p.cutoffs = append(p.cutoffs, cutoff)
	p.mu.Unlock()
	return p.rows, p.err
}

func (p *fakePurger) firstCutoff() (time.Time, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.cutoffs) == 0 {
		return time.Time{}, false
	}
	return p.cutoffs[0], true
}

// TestCleanup_RunOnce 启动后立即执行一次（首次跑用于回收启动前积压）
func TestCleanup_RunOnce(t *testing.T) {
	p := &fakePurger{rows: 7}
	c := New(p, Config{Retention: 7 * 24 * time.Hour, Interval: time.Hour, RunOnStart: true})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Start(ctx)
	defer c.Stop()

	// 等待首次执行
	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&p.calls) == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if atomic.LoadInt32(&p.calls) != 1 {
		t.Fatalf("expected 1 call after start, got %d", p.calls)
	}
	gotCutoff, ok := p.firstCutoff()
	if !ok {
		t.Fatalf("no cutoff captured")
	}
	if d := time.Since(gotCutoff); d < 6*24*time.Hour || d > 8*24*time.Hour {
		t.Fatalf("cutoff not roughly 7d ago: %v", d)
	}
}

// TestCleanup_Periodic 多次触发：用极短 interval 验证至少 2 次
func TestCleanup_Periodic(t *testing.T) {
	p := &fakePurger{}
	c := New(p, Config{Retention: 7 * 24 * time.Hour, Interval: 30 * time.Millisecond, RunOnStart: false})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Start(ctx)
	defer c.Stop()

	deadline := time.Now().Add(500 * time.Millisecond)
	for atomic.LoadInt32(&p.calls) < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := atomic.LoadInt32(&p.calls); got < 2 {
		t.Fatalf("expected >=2 calls, got %d", got)
	}
}

// TestCleanup_StopHaltsTicker Stop 后不再调用
func TestCleanup_StopHaltsTicker(t *testing.T) {
	p := &fakePurger{}
	c := New(p, Config{Retention: 24 * time.Hour, Interval: 20 * time.Millisecond, RunOnStart: false})

	ctx := context.Background()
	c.Start(ctx)

	time.Sleep(80 * time.Millisecond)
	c.Stop()
	snapshot := atomic.LoadInt32(&p.calls)
	time.Sleep(80 * time.Millisecond)
	if got := atomic.LoadInt32(&p.calls); got != snapshot {
		t.Fatalf("calls grew after Stop: snap=%d got=%d", snapshot, got)
	}
}

// TestCleanup_ErrorDoesNotPanic purger 报错不应导致 panic 或 goroutine 退出
func TestCleanup_ErrorDoesNotPanic(t *testing.T) {
	p := &fakePurger{err: errors.New("boom")}
	c := New(p, Config{Retention: 24 * time.Hour, Interval: 20 * time.Millisecond, RunOnStart: true})

	ctx := context.Background()
	c.Start(ctx)
	defer c.Stop()

	deadline := time.Now().Add(300 * time.Millisecond)
	for atomic.LoadInt32(&p.calls) < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := atomic.LoadInt32(&p.calls); got < 2 {
		t.Fatalf("expected purger to keep running on error, calls=%d", got)
	}
}

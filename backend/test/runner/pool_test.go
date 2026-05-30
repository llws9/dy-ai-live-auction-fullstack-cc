package runner

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestPool_SubmitAndExecute 提交简单任务必须执行
func TestPool_SubmitAndExecute(t *testing.T) {
	p := NewPool(4)
	defer p.Shutdown()

	var n int32
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		if err := p.Submit(func() {
			defer wg.Done()
			atomic.AddInt32(&n, 1)
		}); err != nil {
			t.Fatalf("Submit failed: %v", err)
		}
	}
	wg.Wait()
	if got := atomic.LoadInt32(&n); got != 10 {
		t.Fatalf("expected 10 executions, got %d", got)
	}
}

// TestPool_SubmitAfterShutdown 关闭后 Submit 必须报错
func TestPool_SubmitAfterShutdown(t *testing.T) {
	p := NewPool(2)
	p.Shutdown()
	if err := p.Submit(func() {}); err == nil {
		t.Fatalf("expected error after shutdown, got nil")
	}
}

// TestPool_ShutdownWaitsInflight 关闭必须等待 in-flight 任务完成
func TestPool_ShutdownWaitsInflight(t *testing.T) {
	p := NewPool(2)
	var done int32
	for i := 0; i < 4; i++ {
		_ = p.Submit(func() {
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&done, 1)
		})
	}
	p.Shutdown()
	if got := atomic.LoadInt32(&done); got != 4 {
		t.Fatalf("expected 4 finished tasks, got %d", got)
	}
}

// TestPool_NoGoroutineLeak 1000 并发任务无 goroutine 泄漏
func TestPool_NoGoroutineLeak(t *testing.T) {
	// 给运行时一点时间稳住基线
	runtime.GC()
	time.Sleep(20 * time.Millisecond)
	base := runtime.NumGoroutine()

	p := NewPool(64)
	var wg sync.WaitGroup
	const N = 1000
	wg.Add(N)
	for i := 0; i < N; i++ {
		if err := p.Submit(func() {
			defer wg.Done()
			// 模拟轻量工作
			time.Sleep(time.Millisecond)
		}); err != nil {
			t.Fatalf("Submit %d failed: %v", i, err)
		}
	}
	wg.Wait()
	p.Shutdown()

	// Shutdown 之后 worker goroutine 必须全部退出
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	after := runtime.NumGoroutine()
	if delta := after - base; delta > 5 {
		t.Fatalf("goroutine leak detected: base=%d after=%d delta=%d", base, after, delta)
	}
}

// TestPool_DoubleShutdownSafe 重复 Shutdown 不应 panic
func TestPool_DoubleShutdownSafe(t *testing.T) {
	p := NewPool(2)
	p.Shutdown()
	p.Shutdown() // 再次调用应安全
}

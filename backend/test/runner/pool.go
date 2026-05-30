package runner

import (
	"errors"
	"sync"
)

// ErrPoolClosed pool 关闭后再 Submit 返回此错误
var ErrPoolClosed = errors.New("pool closed")

// Pool 固定大小 worker pool，支持优雅关闭
//
// 设计要点：
//   - 固定 size 个 worker goroutine，避免无界并发
//   - tasks channel 无缓冲：背压由调用方决定（阻塞）
//   - Shutdown 关闭 channel 后 worker 自然退出，wg.Wait 等待 in-flight 完成
//   - closed 标志位用 sync.Once 保护，重复 Shutdown 安全
type Pool struct {
	tasks  chan func()
	wg     sync.WaitGroup
	once   sync.Once
	closed chan struct{}
}

// NewPool 构造 worker pool，size 必须 > 0
func NewPool(size int) *Pool {
	if size <= 0 {
		size = 1
	}
	p := &Pool{
		tasks:  make(chan func()),
		closed: make(chan struct{}),
	}
	p.wg.Add(size)
	for i := 0; i < size; i++ {
		go p.worker()
	}
	return p
}

// worker 循环消费任务，channel 关闭后自然退出
func (p *Pool) worker() {
	defer p.wg.Done()
	for task := range p.tasks {
		// 单独 func 包裹防止 task panic 影响 worker
		func() {
			defer func() {
				_ = recover()
			}()
			task()
		}()
	}
}

// Submit 提交一个任务；pool 关闭后返回 ErrPoolClosed
func (p *Pool) Submit(task func()) error {
	if task == nil {
		return nil
	}
	select {
	case <-p.closed:
		return ErrPoolClosed
	default:
	}
	// 双重检查：在 send 期间可能有并发 Shutdown
	select {
	case <-p.closed:
		return ErrPoolClosed
	case p.tasks <- task:
		return nil
	}
}

// Shutdown 关闭 pool，等待所有 in-flight 任务结束。可重复调用。
func (p *Pool) Shutdown() {
	p.once.Do(func() {
		close(p.closed)
		close(p.tasks)
	})
	p.wg.Wait()
}

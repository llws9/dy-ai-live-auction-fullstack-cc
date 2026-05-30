// Package cron 提供 test-service 内嵌定时清理任务。
//
// 第一性原理：
//   - 测试结果数据无限增长会拖慢 history 查询；不引入额外依赖（robfig/cron）的代价低于其收益。
//   - 用 time.Ticker + 单 goroutine 即可满足"每日一次"语义，可控可测。
package cron

import (
	"context"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

// Purger 抽象出删除操作（便于桩注入）；ResultDAO.DeleteOlderThan 实现
type Purger interface {
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}

// Config 配置
type Config struct {
	Retention  time.Duration // 保留期；早于 (now-Retention) 的记录会被删
	Interval   time.Duration // 触发间隔（默认 24h）
	RunOnStart bool          // 启动后立即跑一次
}

// Cleanup 测试历史清理 cron
type Cleanup struct {
	p   Purger
	cfg Config

	once   sync.Once
	cancel context.CancelFunc
}

// New 构造
func New(p Purger, cfg Config) *Cleanup {
	if cfg.Retention <= 0 {
		cfg.Retention = 7 * 24 * time.Hour
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 24 * time.Hour
	}
	return &Cleanup{p: p, cfg: cfg}
}

// Start 启动后台清理 goroutine（幂等）
func (c *Cleanup) Start(ctx context.Context) {
	c.once.Do(func() {
		runCtx, cancel := context.WithCancel(ctx)
		c.cancel = cancel
		go c.loop(runCtx)
	})
}

// Stop 停止
func (c *Cleanup) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *Cleanup) loop(ctx context.Context) {
	if c.cfg.RunOnStart {
		c.runOnce(ctx)
	}
	t := time.NewTicker(c.cfg.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			hlog.Infof("[cleanup] stopped")
			return
		case <-t.C:
			c.runOnce(ctx)
		}
	}
}

func (c *Cleanup) runOnce(ctx context.Context) {
	cutoff := time.Now().Add(-c.cfg.Retention)
	rows, err := c.p.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		hlog.Errorf("[cleanup] purge failed cutoff=%s err=%v", cutoff.Format(time.RFC3339), err)
		return
	}
	if rows > 0 {
		hlog.Infof("[cleanup] purged rows=%d cutoff=%s", rows, cutoff.Format(time.RFC3339))
	} else {
		hlog.Debugf("[cleanup] no rows purged cutoff=%s", cutoff.Format(time.RFC3339))
	}
}

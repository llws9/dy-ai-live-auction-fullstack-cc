// Package antisnipe 场景 F：防狙击延时机制测试。
//
// 设计思路：
//   - 通过抽象的 BidClient 接口与 auction-service 解耦，便于桩注入；
//   - 通过 Now/Sleep 函数注入时间，便于在测试中加速；
//   - 模拟器负责"等到末段窗口前 N 秒 → 在末段每 200ms 出价 → 直到 EndTime"；
//   - 每次出价后采样 EndTime / DelayUsed，构成时间轴，供前端可视化与断言使用。
package antisnipe

import (
	"context"
	"errors"
	"fmt"
	"time"

	"test-service/client/auction"
)

// BidClient 防狙击场景所需的最小客户端接口
type BidClient interface {
	PlaceBid(ctx context.Context, userID, auctionID int64, amount float64) auction.StepResult
	GetAuction(ctx context.Context, auctionID int64) (auction.Auction, auction.StepResult)
}

// Config 模拟器配置
type Config struct {
	AuctionID       int64   `json:"auction_id"`
	BidderIDs       []int64 `json:"bidder_ids"`
	BidIntervalMs   int     `json:"bid_interval_ms"`   // 末段内出价间隔，建议 200ms
	EndingWindowSec int     `json:"ending_window_sec"` // 模拟器在末段的出价窗口长度（秒）
	StartPrice      float64 `json:"start_price"`
	Increment       float64 `json:"increment"`

	// SafePeriodOnly 仅在安全期内（远离末段）出价；用于 "安全期不触发延时" 用例
	SafePeriodOnly bool `json:"safe_period_only,omitempty"`

	// 钩子：可注入用于测试加速；nil 时使用真实 time
	Now   func() time.Time         `json:"-"`
	Sleep func(d time.Duration)    `json:"-"`
}

// TimelineEvent 时间轴单点（出价后采样）
type TimelineEvent struct {
	At         time.Time `json:"at"`
	UserID     int64     `json:"user_id"`
	BidOK      bool      `json:"bid_ok"`
	DelayUsed  int       `json:"delay_used_sec"`
	EndTime    time.Time `json:"end_time"`
	Triggered  bool      `json:"triggered"` // 此次出价是否使 DelayUsed 增加
}

// Report 模拟器报告
type Report struct {
	AuctionID       int64           `json:"auction_id"`
	OriginalEndTime time.Time       `json:"original_end_time"`
	ActualEndTime   time.Time       `json:"actual_end_time"`
	TriggeredCount  int             `json:"triggered_count"`
	BidCount        int             `json:"bid_count"`
	DelayUsedMs     int64           `json:"delay_used_ms"`
	Timeline        []TimelineEvent `json:"timeline"`
}

// Simulator 末刻出价模拟器
type Simulator struct {
	cli BidClient
	cfg Config
}

// NewSimulator 构造
func NewSimulator(cli BidClient, cfg Config) *Simulator {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Sleep == nil {
		cfg.Sleep = time.Sleep
	}
	if cfg.BidIntervalMs <= 0 {
		cfg.BidIntervalMs = 200
	}
	if cfg.EndingWindowSec <= 0 && !cfg.SafePeriodOnly {
		cfg.EndingWindowSec = 5
	}
	return &Simulator{cli: cli, cfg: cfg}
}

// RunSimulation 跑一轮防狙击模拟。
// 返回报告（含时间轴）。RunSimulation 不写库，由上层 Scenario 决定如何持久化。
func (s *Simulator) RunSimulation(ctx context.Context) (*Report, error) {
	if len(s.cfg.BidderIDs) == 0 {
		return nil, errors.New("bidder_ids is empty")
	}

	// 1) 拍卖快照（取得原计划截拍点）
	a, getStep := s.cli.GetAuction(ctx, s.cfg.AuctionID)
	if !getStep.OK {
		return nil, fmt.Errorf("get auction failed: %s", getStep.Message)
	}

	rep := &Report{
		AuctionID:       s.cfg.AuctionID,
		OriginalEndTime: a.EndTime,
		Timeline:        make([]TimelineEvent, 0, 32),
	}

	// 2) 安全期出价路径：仅出 1 笔后立即返回（不进入末段）
	if s.cfg.SafePeriodOnly {
		s.placeOne(ctx, 0, rep)
		// 重新拉一次拍卖以采集结束态
		final, _ := s.cli.GetAuction(ctx, s.cfg.AuctionID)
		rep.ActualEndTime = final.EndTime
		rep.DelayUsedMs = int64(final.DelayUsed) * 1000
		return rep, nil
	}

	// 3) 普通路径：等到末段窗口前 → 在末段每 BidIntervalMs 出价
	windowStart := a.EndTime.Add(-time.Duration(s.cfg.EndingWindowSec) * time.Second)
	if wait := windowStart.Sub(s.cfg.Now()); wait > 0 {
		s.cfg.Sleep(wait)
	}

	bidIdx := 0
	maxIters := len(s.cfg.BidderIDs) * 8 // 安全围栏，避免极端情况下死循环
	for iter := 0; iter < maxIters; iter++ {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		// 每次出价前重新读取 EndTime，因为可能已被延长
		cur, gs := s.cli.GetAuction(ctx, s.cfg.AuctionID)
		if !gs.OK {
			break
		}
		if !s.cfg.Now().Before(cur.EndTime) {
			break
		}

		s.placeOne(ctx, bidIdx, rep)
		bidIdx++
		if bidIdx >= len(s.cfg.BidderIDs) {
			bidIdx = 0
		}
		s.cfg.Sleep(time.Duration(s.cfg.BidIntervalMs) * time.Millisecond)
	}

	// 4) 收尾：再读一次拍卖，记录 ActualEndTime / DelayUsed
	final, _ := s.cli.GetAuction(ctx, s.cfg.AuctionID)
	rep.ActualEndTime = final.EndTime
	rep.DelayUsedMs = int64(final.DelayUsed) * 1000
	return rep, nil
}

// placeOne 出 1 笔价并把样本写入时间轴
func (s *Simulator) placeOne(ctx context.Context, bidderIdx int, rep *Report) {
	if bidderIdx < 0 || bidderIdx >= len(s.cfg.BidderIDs) {
		bidderIdx = 0
	}
	uid := s.cfg.BidderIDs[bidderIdx]
	rep.BidCount++

	// 读出价前的 DelayUsed，用于判断本次是否触发延时
	beforeAuction, _ := s.cli.GetAuction(ctx, s.cfg.AuctionID)
	beforeDelay := beforeAuction.DelayUsed

	amount := s.cfg.StartPrice + s.cfg.Increment*float64(rep.BidCount)
	step := s.cli.PlaceBid(ctx, uid, s.cfg.AuctionID, amount)

	afterAuction, _ := s.cli.GetAuction(ctx, s.cfg.AuctionID)
	triggered := afterAuction.DelayUsed > beforeDelay
	if triggered {
		rep.TriggeredCount++
	}
	rep.Timeline = append(rep.Timeline, TimelineEvent{
		At:        s.cfg.Now(),
		UserID:    uid,
		BidOK:     step.OK,
		DelayUsed: afterAuction.DelayUsed,
		EndTime:   afterAuction.EndTime,
		Triggered: triggered,
	})
}

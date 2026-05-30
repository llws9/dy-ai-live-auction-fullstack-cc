// Package e2e 实现场景 E：完整业务链路 E2E 编排。
//
// 编排器把 SDK 的 7+ 个原子调用串成 setup→run→verify→cleanup 四阶段，
// 每步通过 ProgressEmitter 实时上报。任何阶段失败都不阻断 cleanup，
// 保证测试副作用可清理。
package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"test-service/client/auction"
	"test-service/runner"
)

// BizClient 业务 SDK 接口（便于桩注入；实现见 client/auction.Client）
type BizClient interface {
	CreateProduct(ctx context.Context, userID int64, req auction.CreateProductReq) auction.StepResult
	CreateAuction(ctx context.Context, userID int64, req auction.CreateAuctionReq) auction.StepResult
	PlaceBid(ctx context.Context, userID, auctionID int64, amount float64) auction.StepResult
	GetAuction(ctx context.Context, auctionID int64) (auction.Auction, auction.StepResult)
	WaitAuctionStarted(ctx context.Context, auctionID int64, interval, timeout time.Duration) auction.StepResult
	WaitAuctionEnded(ctx context.Context, auctionID int64, interval, timeout time.Duration) auction.StepResult
	SubscribeSkyLamp(ctx context.Context, userID, auctionID int64) auction.StepResult
	FindOrdersByAuction(ctx context.Context, winnerID, auctionID int64) ([]auction.Order, auction.StepResult)
}

// SeedRecorder 测试种子数据记录器接口（便于桩注入；实现见 dao.SeedDAO）
type SeedRecorder interface {
	Add(ctx context.Context, testID, kind string, refID int64) error
	DeleteByTestID(ctx context.Context, testID string) error
}

// Config 编排器配置
type Config struct {
	TestID       string        `json:"test_id,omitempty"`       // 关联 runner test_id；用于 SeedRecorder
	SellerID     int64         `json:"seller_id"`               // 创建拍品/拍卖的卖家
	BidderIDs    []int64       `json:"bidder_ids"`              // 出价者列表
	SubscriberID int64         `json:"subscriber_id"`           // 点天灯订阅者
	StartPrice   float64       `json:"start_price"`             // 起拍价
	Increment    float64       `json:"increment"`               // 加价幅度
	Duration     int           `json:"duration"`                // 拍卖持续秒数
	PollInterval time.Duration `json:"poll_interval,omitempty"` // 轮询间隔
	PollTimeout  time.Duration `json:"poll_timeout,omitempty"`  // 轮询超时
}

// Report 测试报告
type Report struct {
	TestID    string               `json:"test_id"`
	AuctionID int64                `json:"auction_id"`
	ProductID int64                `json:"product_id"`
	WinnerID  int64                `json:"winner_id"`
	OrderID   int64                `json:"order_id,omitempty"`
	Steps     []auction.StepResult `json:"steps"`
	AllOK     bool                 `json:"all_ok"`
	Error     string               `json:"error,omitempty"`
}

// Orchestrator E2E 编排器
type Orchestrator struct {
	cli BizClient
	rec SeedRecorder
	cfg Config
}

// New 构造
func New(cli BizClient, rec SeedRecorder, cfg Config) *Orchestrator {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 500 * time.Millisecond
	}
	if cfg.PollTimeout <= 0 {
		cfg.PollTimeout = 60 * time.Second
	}
	return &Orchestrator{cli: cli, rec: rec, cfg: cfg}
}

// Run 串行执行 setup→run→verify→cleanup。
// 返回 error 仅在 setup 阶段失败或 ctx 被取消时；
// verify 失败不算 error，由 Report.AllOK = false 表示。
func (o *Orchestrator) Run(ctx context.Context, p runner.ProgressEmitter) (*Report, error) {
	rep := &Report{TestID: o.cfg.TestID, Steps: make([]auction.StepResult, 0, 16)}

	// cleanup 一定执行
	defer func() {
		o.cleanup(ctx, rep, p)
	}()

	// ---------- setup ----------
	prodStep := o.cli.CreateProduct(ctx, o.cfg.SellerID, auction.CreateProductReq{
		Name:        fmt.Sprintf("E2E 测试拍品 %s", o.cfg.TestID),
		Description: "E2E orchestrator auto-generated",
		Status:      1,
	})
	o.record(rep, p, 5, prodStep)
	if !prodStep.OK {
		return rep, fmt.Errorf("setup create_product failed: %s", prodStep.Message)
	}
	rep.ProductID = prodStep.RefID
	_ = o.rec.Add(ctx, o.cfg.TestID, "product", rep.ProductID)

	if err := ctx.Err(); err != nil {
		return rep, err
	}

	auctionStep := o.cli.CreateAuction(ctx, o.cfg.SellerID, auction.CreateAuctionReq{
		ProductID:  rep.ProductID,
		StartPrice: o.cfg.StartPrice,
		Increment:  o.cfg.Increment,
		Duration:   o.cfg.Duration,
	})
	o.record(rep, p, 10, auctionStep)
	if !auctionStep.OK {
		return rep, fmt.Errorf("setup create_auction failed: %s", auctionStep.Message)
	}
	rep.AuctionID = auctionStep.RefID
	_ = o.rec.Add(ctx, o.cfg.TestID, "auction", rep.AuctionID)

	if err := ctx.Err(); err != nil {
		return rep, err
	}

	// ---------- run ----------
	startStep := o.cli.WaitAuctionStarted(ctx, rep.AuctionID, o.cfg.PollInterval, o.cfg.PollTimeout)
	o.record(rep, p, 20, startStep)
	if !startStep.OK {
		// 拍卖没启动，run 阶段无意义；但不算 error，进入 verify 让其判失败
		return rep, nil
	}

	if o.cfg.SubscriberID > 0 {
		subStep := o.cli.SubscribeSkyLamp(ctx, o.cfg.SubscriberID, rep.AuctionID)
		o.record(rep, p, 30, subStep)
	}

	if err := ctx.Err(); err != nil {
		return rep, err
	}

	// 多轮出价：每个 bidder 出一次，递增价格
	bidPrice := o.cfg.StartPrice + o.cfg.Increment
	bidProgressBase := 30
	bidProgressSpan := 30 // 30 → 60
	for i, uid := range o.cfg.BidderIDs {
		bidStep := o.cli.PlaceBid(ctx, uid, rep.AuctionID, bidPrice)
		progress := bidProgressBase
		if len(o.cfg.BidderIDs) > 0 {
			progress = bidProgressBase + bidProgressSpan*(i+1)/len(o.cfg.BidderIDs)
		}
		o.record(rep, p, progress, bidStep)
		bidPrice += o.cfg.Increment
		if err := ctx.Err(); err != nil {
			return rep, err
		}
	}

	endStep := o.cli.WaitAuctionEnded(ctx, rep.AuctionID, o.cfg.PollInterval, o.cfg.PollTimeout)
	o.record(rep, p, 70, endStep)
	if !endStep.OK {
		return rep, nil
	}

	// ---------- verify ----------
	a, getStep := o.cli.GetAuction(ctx, rep.AuctionID)
	o.record(rep, p, 80, getStep)
	rep.WinnerID = a.WinnerID

	verifyStep := auction.StepResult{Step: "verify_winner"}
	if a.WinnerID == 0 {
		verifyStep.Message = "no winner"
	} else {
		verifyStep.OK = true
		verifyStep.RefID = a.WinnerID
	}
	o.record(rep, p, 85, verifyStep)

	orders, ordersStep := o.cli.FindOrdersByAuction(ctx, a.WinnerID, rep.AuctionID)
	o.record(rep, p, 90, ordersStep)

	orderCheck := auction.StepResult{Step: "verify_order_unique"}
	switch {
	case len(orders) == 1:
		orderCheck.OK = true
		orderCheck.RefID = orders[0].ID
		rep.OrderID = orders[0].ID
	case len(orders) == 0:
		orderCheck.Message = "no order"
	default:
		orderCheck.Message = fmt.Sprintf("expected 1 order, got %d", len(orders))
	}
	o.record(rep, p, 95, orderCheck)

	rep.AllOK = computeAllOK(rep.Steps)
	return rep, nil
}

// record 添加一步并 emit 进度
func (o *Orchestrator) record(rep *Report, p runner.ProgressEmitter, progress int, step auction.StepResult) {
	rep.Steps = append(rep.Steps, step)
	if p != nil {
		p.Emit(progress, step.Step, map[string]any{
			"ok":          step.OK,
			"duration_ms": step.DurationMs,
			"ref_id":      step.RefID,
			"message":     step.Message,
			"status_code": step.StatusCode,
		})
	}
}

// cleanup 反向删除 seed 数据；不阻塞
func (o *Orchestrator) cleanup(ctx context.Context, rep *Report, p runner.ProgressEmitter) {
	// 即使 ctx 取消也要执行 cleanup，用独立 ctx
	cleanCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	step := auction.StepResult{Step: "cleanup", OK: true}
	if err := o.rec.DeleteByTestID(cleanCtx, o.cfg.TestID); err != nil {
		step.OK = false
		step.Message = err.Error()
	}
	rep.Steps = append(rep.Steps, step)
	if p != nil {
		p.Emit(100, "cleanup", map[string]any{"ok": step.OK})
	}
	_ = ctx
}

// computeAllOK 报告整体是否成功（所有步骤 OK 即成功）
func computeAllOK(steps []auction.StepResult) bool {
	if len(steps) == 0 {
		return false
	}
	for _, s := range steps {
		if !s.OK {
			return false
		}
	}
	return true
}

// ---------- runner.Scenario 适配 ----------

// Scenario 适配 runner.Scenario 接口
type Scenario struct {
	cli BizClient
	rec SeedRecorder
}

// NewScenario 构造（runner 注册用）
func NewScenario(cli BizClient, rec SeedRecorder) *Scenario {
	return &Scenario{cli: cli, rec: rec}
}

// Type 场景标识
func (s *Scenario) Type() string { return "e2e" }

// Run 解析配置 → 创建编排器 → 执行
func (s *Scenario) Run(ctx context.Context, raw json.RawMessage, p runner.ProgressEmitter) (any, error) {
	cfg := Config{}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("invalid e2e config: %w", err)
		}
	}
	if cfg.TestID == "" {
		cfg.TestID = runner.TestIDFromContext(ctx)
	}
	applyDefaults(&cfg)
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}
	o := New(s.cli, s.rec, cfg)
	return o.Run(ctx, p)
}

func applyDefaults(c *Config) {
	if c.SellerID == 0 {
		c.SellerID = 9001
	}
	if len(c.BidderIDs) == 0 {
		c.BidderIDs = []int64{2001, 2002, 2003}
	}
	if c.SubscriberID == 0 {
		c.SubscriberID = 3001
	}
	if c.StartPrice == 0 {
		c.StartPrice = 100
	}
	if c.Increment == 0 {
		c.Increment = 10
	}
	if c.Duration == 0 {
		c.Duration = 30
	}
}

func validateConfig(c *Config) error {
	if len(c.BidderIDs) == 0 {
		return errors.New("bidder_ids is empty")
	}
	if c.StartPrice < 0 || c.Increment <= 0 {
		return errors.New("invalid price/increment")
	}
	return nil
}

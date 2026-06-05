package user_journey

import (
	"context"
	"fmt"
	"time"

	"test-service/client/auction"
	"test-service/runner"
)

type BusinessClient interface {
	CreateProductAs(ctx context.Context, actor auction.Actor, req auction.CreateProductReq) auction.StepResult
	CreateLiveStream(ctx context.Context, actor auction.Actor, req auction.CreateLiveStreamReq) auction.StepResult
	CreateAuctionRule(ctx context.Context, actor auction.Actor, productID int64, req auction.CreateAuctionRuleReq) auction.StepResult
	CreateAuctionAs(ctx context.Context, actor auction.Actor, req auction.CreateAuctionReq) auction.StepResult
	WaitAuctionStarted(ctx context.Context, auctionID int64, interval, timeout time.Duration) auction.StepResult
	WaitAuctionEnded(ctx context.Context, auctionID int64, interval, timeout time.Duration) auction.StepResult
	GetAuctionResult(ctx context.Context, auctionID int64) (auction.AuctionResult, auction.StepResult)
	CreateFixedPriceItem(ctx context.Context, actor auction.Actor, req auction.CreateFixedPriceItemReq) auction.StepResult
	StartLive(ctx context.Context, actor auction.Actor, liveStreamID int64) auction.StepResult
	GetLiveStream(ctx context.Context, actor auction.Actor, liveStreamID int64) (auction.LiveStream, auction.StepResult)
	ListFixedPriceItemsByLiveStream(ctx context.Context, actor auction.Actor, liveStreamID int64) ([]auction.FixedPriceItem, auction.StepResult)
	FollowLiveStream(ctx context.Context, actor auction.Actor, liveStreamID int64) auction.StepResult
	GetFollowStatus(ctx context.Context, actor auction.Actor, liveStreamID int64) (bool, auction.StepResult)
	PlaceBid(ctx context.Context, userID, auctionID int64, amount float64) auction.StepResult
	SubscribeSkyLamp(ctx context.Context, userID, auctionID int64) auction.StepResult
	PurchaseFixedPriceItem(ctx context.Context, actor auction.Actor, itemID int64, idemKey string) (int64, auction.StepResult)
	GetMyFixedPricePurchase(ctx context.Context, actor auction.Actor, itemID int64) (auction.FixedPricePurchase, auction.StepResult)
	FindOrdersByAuction(ctx context.Context, winnerID, auctionID int64) ([]auction.Order, auction.StepResult)
	GetUserBalance(ctx context.Context, actor auction.Actor) (string, auction.StepResult)
}

type InternalClient interface {
	EnsureUsers(ctx context.Context, actors []auction.Actor) auction.StepResult
	TopUpUserBalance(ctx context.Context, userID int64, amount string) (string, auction.StepResult)
}

type SeedRecorder interface {
	Add(ctx context.Context, testID, kind string, refID int64) error
	DeleteByTestID(ctx context.Context, testID string) error
}

type Config struct {
	TestID              string `json:"test_id,omitempty"`
	IncludeReminder     *bool  `json:"include_reminder,omitempty"`
	IncludeSkyLamp      *bool  `json:"include_sky_lamp,omitempty"`
	IncludeFixedPrice   *bool  `json:"include_fixed_price,omitempty"`
	AuctionDurationSec  int    `json:"auction_duration_sec,omitempty"`
	BuyerCount          int    `json:"buyer_count,omitempty"`
	KeepEvidence        *bool  `json:"keep_evidence,omitempty"`
	BuyerID             int64  `json:"buyer_id,omitempty"`
	MerchantID          int64  `json:"merchant_id,omitempty"`
	BalanceTopUpAmount  string `json:"balance_top_up_amount,omitempty"`
	FixedPriceItemPrice string `json:"fixed_price_item_price,omitempty"`
}

type Report struct {
	TestRunID        string               `json:"test_run_id"`
	BuyerID          int64                `json:"buyer_id"`
	MerchantID       int64                `json:"merchant_id"`
	ProductID        int64                `json:"product_id"`
	LiveStreamID     int64                `json:"live_stream_id"`
	AuctionID        int64                `json:"auction_id"`
	FixedPriceItemID int64                `json:"fixed_price_item_id"`
	OrderID          int64                `json:"order_id"`
	BalanceBefore    string               `json:"balance_before"`
	BalanceAfter     string               `json:"balance_after"`
	StockBefore      int64                `json:"stock_before"`
	StockAfter       int64                `json:"stock_after"`
	Steps            []auction.StepResult `json:"steps"`
	AllOK            bool                 `json:"all_ok"`
	Warnings         []string             `json:"warnings,omitempty"`
	Error            string               `json:"error,omitempty"`
}

type Orchestrator struct {
	biz      BusinessClient
	internal InternalClient
	rec      SeedRecorder
	cfg      Config
}

func New(biz BusinessClient, internal InternalClient, rec SeedRecorder, cfg Config) *Orchestrator {
	applyDefaults(&cfg)
	return &Orchestrator{biz: biz, internal: internal, rec: rec, cfg: cfg}
}

func (o *Orchestrator) Run(ctx context.Context, p runner.ProgressEmitter) (report *Report, err error) {
	rep := &Report{
		TestRunID:   o.cfg.TestID,
		BuyerID:     o.cfg.BuyerID,
		MerchantID:  o.cfg.MerchantID,
		Steps:       make([]auction.StepResult, 0, 8),
		Warnings:    make([]string, 0),
		StockBefore: 1,
	}
	defer func() {
		if o.cfg.keepEvidence() || o.rec == nil {
			return
		}
		if cleanupErr := o.rec.DeleteByTestID(ctx, o.cfg.TestID); cleanupErr != nil {
			rep.Warnings = append(rep.Warnings, "cleanup failed: "+cleanupErr.Error())
		}
	}()
	buyer := auction.Actor{UserID: o.cfg.BuyerID, Username: fmt.Sprintf("buyer_%d", o.cfg.BuyerID), Role: auction.RoleUser}
	merchant := auction.Actor{UserID: o.cfg.MerchantID, Username: fmt.Sprintf("merchant_%d", o.cfg.MerchantID), Role: auction.RoleMerchant}

	if err := o.prepare(ctx, rep, p, buyer, merchant); err != nil {
		rep.AllOK = false
		rep.Error = err.Error()
		return rep, err
	}
	if err := o.enterLive(ctx, rep, p, buyer); err != nil {
		return o.fail(rep, err), err
	}
	if o.cfg.includeReminder() {
		if err := o.reminder(ctx, rep, p, buyer); err != nil {
			return o.fail(rep, err), err
		}
	}
	if err := o.auctionBid(ctx, rep, p, buyer); err != nil {
		return o.fail(rep, err), err
	}
	if o.cfg.includeSkyLamp() {
		if err := o.skyLamp(ctx, rep, p, buyer); err != nil {
			return o.fail(rep, err), err
		}
	}
	if o.cfg.includeFixedPrice() {
		if err := o.fixedPricePurchase(ctx, rep, p, buyer); err != nil {
			return o.fail(rep, err), err
		}
	}
	if err := o.verify(ctx, rep, p, buyer); err != nil {
		return o.fail(rep, err), err
	}
	rep.AllOK = computeAllOK(rep.Steps)
	return rep, nil
}

func (o *Orchestrator) prepare(ctx context.Context, rep *Report, p runner.ProgressEmitter, buyer, merchant auction.Actor) error {
	ensureUsers := o.internal.EnsureUsers(ctx, []auction.Actor{buyer, merchant})
	if !ensureUsers.OK {
		return o.recordAndError(rep, p, 5, "prepare", ensureUsers, "prepare ensure_users failed")
	}

	balanceBefore, balanceStep := o.biz.GetUserBalance(ctx, buyer)
	if !balanceStep.OK {
		return fmt.Errorf("prepare get_balance failed: %s", balanceStep.Message)
	}
	rep.BalanceBefore = balanceBefore

	product := o.biz.CreateProductAs(ctx, merchant, auction.CreateProductReq{
		Name:        fmt.Sprintf("TEST_USER_JOURNEY_%s 商品", o.cfg.TestID),
		Description: "TEST_USER_JOURNEY_" + o.cfg.TestID,
		Status:      1,
	})
	if !product.OK {
		return o.recordAndError(rep, p, 10, "prepare", product, "prepare create_product failed")
	}
	rep.ProductID = product.RefID
	o.addSeed(ctx, "product", rep.ProductID)

	live := o.biz.CreateLiveStream(ctx, merchant, auction.CreateLiveStreamReq{
		Name:        fmt.Sprintf("TEST_USER_JOURNEY_%s 直播间", o.cfg.TestID),
		Description: "TEST_USER_JOURNEY_" + o.cfg.TestID,
		ProductID:   rep.ProductID,
	})
	if !live.OK {
		return o.recordAndError(rep, p, 10, "prepare", live, "prepare create_live_stream failed")
	}
	rep.LiveStreamID = live.RefID
	o.addSeed(ctx, "live_stream", rep.LiveStreamID)

	rule := o.biz.CreateAuctionRule(ctx, merchant, rep.ProductID, auction.CreateAuctionRuleReq{
		StartPrice:         100,
		Increment:          10,
		Duration:           o.cfg.AuctionDurationSec,
		DelayDuration:      2,
		MaxDelayTime:       2,
		TriggerDelayBefore: 1,
	})
	if !rule.OK {
		return o.recordAndError(rep, p, 10, "prepare", rule, "prepare create_auction_rule failed")
	}

	auctionStep := o.biz.CreateAuctionAs(ctx, merchant, auction.CreateAuctionReq{
		ProductID:  rep.ProductID,
		StartPrice: 100,
		Increment:  10,
		Duration:   o.cfg.AuctionDurationSec,
	})
	if !auctionStep.OK {
		return o.recordAndError(rep, p, 10, "prepare", auctionStep, "prepare create_auction failed")
	}
	rep.AuctionID = auctionStep.RefID
	o.addSeed(ctx, "auction", rep.AuctionID)

	fixed := o.biz.CreateFixedPriceItem(ctx, merchant, auction.CreateFixedPriceItemReq{
		LiveStreamID: rep.LiveStreamID,
		ProductID:    rep.ProductID,
		Price:        o.cfg.FixedPriceItemPrice,
		Stock:        rep.StockBefore,
	})
	if !fixed.OK {
		return o.recordAndError(rep, p, 10, "prepare", fixed, "prepare create_fixed_price_item failed")
	}
	rep.FixedPriceItemID = fixed.RefID
	o.addSeed(ctx, "fixed_price_item", rep.FixedPriceItemID)

	_, topUp := o.internal.TopUpUserBalance(ctx, buyer.UserID, o.cfg.BalanceTopUpAmount)
	if !topUp.OK {
		return o.recordAndError(rep, p, 10, "prepare", topUp, "prepare top_up_balance failed")
	}

	start := o.biz.StartLive(ctx, merchant, rep.LiveStreamID)
	if !start.OK {
		return o.recordAndError(rep, p, 10, "prepare", start, "prepare start_live failed")
	}
	o.record(rep, p, 15, auction.StepResult{Step: "prepare", OK: true, RefID: rep.LiveStreamID})
	return nil
}

func (o *Orchestrator) enterLive(ctx context.Context, rep *Report, p runner.ProgressEmitter, buyer auction.Actor) error {
	live, liveStep := o.biz.GetLiveStream(ctx, buyer, rep.LiveStreamID)
	if !liveStep.OK || live.ID == 0 {
		return o.recordAndError(rep, p, 25, "enter_live", liveStep, "enter_live get_live_stream failed")
	}
	items, itemStep := o.biz.ListFixedPriceItemsByLiveStream(ctx, buyer, rep.LiveStreamID)
	if !itemStep.OK || len(items) == 0 {
		return o.recordAndError(rep, p, 25, "enter_live", itemStep, "enter_live list_fixed_price_items failed")
	}
	o.record(rep, p, 25, auction.StepResult{Step: "enter_live", OK: true, RefID: rep.LiveStreamID})
	return nil
}

func (o *Orchestrator) reminder(ctx context.Context, rep *Report, p runner.ProgressEmitter, buyer auction.Actor) error {
	if step := o.biz.FollowLiveStream(ctx, buyer, rep.LiveStreamID); !step.OK {
		return o.recordAndError(rep, p, 35, "reminder", step, "reminder follow failed")
	}
	ok, step := o.biz.GetFollowStatus(ctx, buyer, rep.LiveStreamID)
	if !step.OK || !ok {
		return o.recordAndError(rep, p, 35, "reminder", step, "reminder follow-status failed")
	}
	o.record(rep, p, 35, auction.StepResult{Step: "reminder", OK: true, RefID: rep.LiveStreamID})
	return nil
}

func (o *Orchestrator) auctionBid(ctx context.Context, rep *Report, p runner.ProgressEmitter, buyer auction.Actor) error {
	wait := o.biz.WaitAuctionStarted(ctx, rep.AuctionID, 200*time.Millisecond, time.Duration(o.cfg.AuctionDurationSec+5)*time.Second)
	if !wait.OK {
		return o.recordAndError(rep, p, 45, "auction_bid", wait, "auction_bid wait_started failed")
	}

	step := o.biz.PlaceBid(ctx, buyer.UserID, rep.AuctionID, 110)
	if !step.OK {
		return o.recordAndError(rep, p, 50, "auction_bid", step, "auction_bid failed")
	}
	o.record(rep, p, 50, auction.StepResult{Step: "auction_bid", OK: true, RefID: rep.AuctionID})
	return nil
}

func (o *Orchestrator) skyLamp(ctx context.Context, rep *Report, p runner.ProgressEmitter, buyer auction.Actor) error {
	step := o.biz.SubscribeSkyLamp(ctx, buyer.UserID, rep.AuctionID)
	if !step.OK {
		return o.recordAndError(rep, p, 62, "sky_lamp", step, "sky_lamp failed")
	}
	o.record(rep, p, 62, auction.StepResult{Step: "sky_lamp", OK: true, RefID: step.RefID})
	return nil
}

func (o *Orchestrator) fixedPricePurchase(ctx context.Context, rep *Report, p runner.ProgressEmitter, buyer auction.Actor) error {
	orderID, step := o.biz.PurchaseFixedPriceItem(ctx, buyer, rep.FixedPriceItemID, "user_journey_"+o.cfg.TestID)
	if !step.OK {
		return o.recordAndError(rep, p, 75, "fixed_price_purchase", step, "fixed_price_purchase failed")
	}
	purchase, purchaseStep := o.biz.GetMyFixedPricePurchase(ctx, buyer, rep.FixedPriceItemID)
	if !purchaseStep.OK || purchase.OrderID == 0 {
		return o.recordAndError(rep, p, 75, "fixed_price_purchase", purchaseStep, "fixed_price_purchase my-purchase failed")
	}
	rep.OrderID = orderID
	if rep.OrderID == 0 {
		rep.OrderID = purchase.OrderID
	}
	rep.StockAfter = 0
	o.addSeed(ctx, "order", rep.OrderID)
	o.record(rep, p, 75, auction.StepResult{Step: "fixed_price_purchase", OK: true, RefID: rep.OrderID})
	return nil
}

func (o *Orchestrator) verify(ctx context.Context, rep *Report, p runner.ProgressEmitter, buyer auction.Actor) error {
	wait := o.biz.WaitAuctionEnded(ctx, rep.AuctionID, 200*time.Millisecond, time.Duration(o.cfg.AuctionDurationSec+5)*time.Second)
	if !wait.OK {
		return o.recordAndError(rep, p, 95, "verify", wait, "verify wait_ended failed")
	}

	result, resultStep := o.biz.GetAuctionResult(ctx, rep.AuctionID)
	if !resultStep.OK || result.Status < 3 || result.WinnerID != buyer.UserID || result.FinalPrice <= 0 {
		return o.recordAndError(rep, p, 100, "verify", resultStep, "verify auction_result failed")
	}
	balanceAfter, balanceStep := o.biz.GetUserBalance(ctx, buyer)
	if !balanceStep.OK {
		return o.recordAndError(rep, p, 100, "verify", balanceStep, "verify balance failed")
	}
	rep.BalanceAfter = balanceAfter
	o.record(rep, p, 100, auction.StepResult{Step: "verify", OK: true, RefID: rep.OrderID})
	return nil
}

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

func (o *Orchestrator) recordAndError(rep *Report, p runner.ProgressEmitter, progress int, stepName string, step auction.StepResult, prefix string) error {
	step.Step = stepName
	if step.Message == "" {
		step.Message = prefix
	}
	o.record(rep, p, progress, step)
	return fmt.Errorf("%s: %s", prefix, step.Message)
}

func (o *Orchestrator) addSeed(ctx context.Context, kind string, refID int64) {
	if o.rec != nil && refID > 0 {
		_ = o.rec.Add(ctx, o.cfg.TestID, kind, refID)
	}
}

func (o *Orchestrator) fail(rep *Report, err error) *Report {
	rep.AllOK = false
	rep.Error = err.Error()
	return rep
}

func applyDefaults(c *Config) {
	if c.TestID == "" {
		c.TestID = "user_journey"
	}
	if c.BuyerID == 0 {
		c.BuyerID = 2001
	}
	if c.MerchantID == 0 {
		c.MerchantID = 9001
	}
	if c.AuctionDurationSec == 0 {
		c.AuctionDurationSec = 30
	}
	if c.BuyerCount == 0 {
		c.BuyerCount = 1
	}
	if c.BalanceTopUpAmount == "" {
		c.BalanceTopUpAmount = "1000.00"
	}
	if c.FixedPriceItemPrice == "" {
		c.FixedPriceItemPrice = "100.00"
	}
}

func (c Config) includeReminder() bool {
	return c.IncludeReminder == nil || *c.IncludeReminder
}

func (c Config) includeSkyLamp() bool {
	return c.IncludeSkyLamp == nil || *c.IncludeSkyLamp
}

func (c Config) includeFixedPrice() bool {
	return c.IncludeFixedPrice == nil || *c.IncludeFixedPrice
}

func (c Config) keepEvidence() bool {
	return c.KeepEvidence == nil || *c.KeepEvidence
}

func computeAllOK(steps []auction.StepResult) bool {
	if len(steps) == 0 {
		return false
	}
	for _, step := range steps {
		if !step.OK {
			return false
		}
	}
	return true
}

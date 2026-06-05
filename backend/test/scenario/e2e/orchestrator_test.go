package e2e

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"test-service/client/auction"
)

// ---------- 桩 ----------

type fakeClient struct {
	mu sync.Mutex

	// 计数 / 记录
	createProductCalls int
	createRuleCalls    int
	createAuctionCalls int
	bidCalls           int
	subCalls           int
	getCalls           int
	findOrdersCalls    int

	// 可注入的失败开关
	failCreateProduct bool
	failCreateRule    bool
	failCreateAuction bool
	failBidFor        map[int64]bool // userID -> 是否失败
	subscribeFail     bool
	getStatusSeq      []int // 模拟 GetAuction 状态变化序列
	winnerID          int64
	currentPrice      float64
	delayUsed         int
	lastProductActor  auction.Actor
	lastRuleActor     auction.Actor
	lastAuctionActor  auction.Actor

	orders []auction.Order
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		failBidFor:   map[int64]bool{},
		getStatusSeq: []int{2}, // 默认直接 ended
		winnerID:     1001,
		currentPrice: 199.0,
	}
}

func (f *fakeClient) CreateProduct(ctx context.Context, userID int64, req auction.CreateProductReq) auction.StepResult {
	return f.CreateProductAs(ctx, auction.Actor{UserID: userID, Role: auction.RoleUser}, req)
}

func (f *fakeClient) CreateProductAs(ctx context.Context, actor auction.Actor, req auction.CreateProductReq) auction.StepResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createProductCalls++
	f.lastProductActor = actor
	if f.failCreateProduct {
		return auction.StepResult{Step: "create_product", OK: false, Message: "stub fail"}
	}
	return auction.StepResult{Step: "create_product", OK: true, RefID: 9001, StatusCode: 200, DurationMs: 1}
}

func (f *fakeClient) CreateAuctionRule(ctx context.Context, actor auction.Actor, productID int64, req auction.CreateAuctionRuleReq) auction.StepResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createRuleCalls++
	f.lastRuleActor = actor
	if f.failCreateRule {
		return auction.StepResult{Step: "create_auction_rule", OK: false, Message: "stub fail"}
	}
	return auction.StepResult{Step: "create_auction_rule", OK: true, RefID: productID, StatusCode: 200, DurationMs: 1}
}

func (f *fakeClient) CreateAuction(ctx context.Context, userID int64, req auction.CreateAuctionReq) auction.StepResult {
	return f.CreateAuctionAs(ctx, auction.Actor{UserID: userID, Role: auction.RoleUser}, req)
}

func (f *fakeClient) CreateAuctionAs(ctx context.Context, actor auction.Actor, req auction.CreateAuctionReq) auction.StepResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createAuctionCalls++
	f.lastAuctionActor = actor
	if f.failCreateAuction {
		return auction.StepResult{Step: "create_auction", OK: false, Message: "stub fail"}
	}
	return auction.StepResult{Step: "create_auction", OK: true, RefID: 8001, StatusCode: 200, DurationMs: 1}
}

func (f *fakeClient) PlaceBid(ctx context.Context, userID, auctionID int64, amount float64) auction.StepResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bidCalls++
	if f.failBidFor[userID] {
		return auction.StepResult{Step: "bid", OK: false, RefID: userID, Message: "bid fail"}
	}
	return auction.StepResult{Step: "bid", OK: true, RefID: userID, StatusCode: 200, DurationMs: 1}
}

func (f *fakeClient) GetAuction(ctx context.Context, auctionID int64) (auction.Auction, auction.StepResult) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls++
	idx := f.getCalls - 1
	if idx >= len(f.getStatusSeq) {
		idx = len(f.getStatusSeq) - 1
	}
	a := auction.Auction{
		ID:           auctionID,
		Status:       f.getStatusSeq[idx],
		WinnerID:     f.winnerID,
		CurrentPrice: f.currentPrice,
		DelayUsed:    f.delayUsed,
	}
	return a, auction.StepResult{Step: "get_auction", OK: true, StatusCode: 200, DurationMs: 1}
}

func (f *fakeClient) WaitAuctionStarted(ctx context.Context, auctionID int64, interval, timeout time.Duration) auction.StepResult {
	return auction.StepResult{Step: "wait_started", OK: true, StatusCode: 200, DurationMs: 1}
}

func (f *fakeClient) WaitAuctionEnded(ctx context.Context, auctionID int64, interval, timeout time.Duration) auction.StepResult {
	return auction.StepResult{Step: "wait_ended", OK: true, StatusCode: 200, DurationMs: 1}
}

func (f *fakeClient) SubscribeSkyLamp(ctx context.Context, userID, auctionID int64) auction.StepResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.subCalls++
	if f.subscribeFail {
		return auction.StepResult{Step: "skylamp_subscribe", OK: false, Message: "sub fail"}
	}
	return auction.StepResult{Step: "skylamp_subscribe", OK: true, RefID: int64(7000 + f.subCalls), StatusCode: 200, DurationMs: 1}
}

func (f *fakeClient) FindOrdersByAuction(ctx context.Context, winnerID, auctionID int64) ([]auction.Order, auction.StepResult) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.findOrdersCalls++
	return f.orders, auction.StepResult{Step: "find_orders", OK: true, StatusCode: 200, DurationMs: 1}
}

// fakeRecorder 内存版 SeedRecorder
type fakeRecorder struct {
	mu      sync.Mutex
	added   []seedEntry // 顺序记录
	deleted bool
}

type seedEntry struct {
	kind  string
	refID int64
}

func (r *fakeRecorder) Add(ctx context.Context, testID, kind string, refID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.added = append(r.added, seedEntry{kind: kind, refID: refID})
	return nil
}

func (r *fakeRecorder) DeleteByTestID(ctx context.Context, testID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deleted = true
	return nil
}

// fakeEmitter 收集 emit 事件
type fakeEmitter struct {
	mu     sync.Mutex
	events []string // 步骤名拼接
}

func (e *fakeEmitter) Emit(progress int, step string, metrics map[string]any) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, step)
}

// ---------- 测试用例 ----------

// TestOrchestrator_HappyPath: 正常流程跑通，setup→run→verify→cleanup 各阶段步骤均上报，cleanup 被调用
func TestOrchestrator_HappyPath(t *testing.T) {
	fc := newFakeClient()
	fc.orders = []auction.Order{{ID: 6001, AuctionID: 8001, WinnerID: 1001, FinalPrice: 199.0, Status: 1}}
	rec := &fakeRecorder{}
	em := &fakeEmitter{}

	o := New(fc, rec, Config{
		TestID:       "test-happy",
		SellerID:     2001,
		BidderIDs:    []int64{1001, 1002, 1003},
		SubscriberID: 3001,
		StartPrice:   100,
		Increment:    10,
		Duration:     30,
		PollInterval: 10 * time.Millisecond,
		PollTimeout:  500 * time.Millisecond,
	})

	report, err := o.Run(context.Background(), em)
	if err != nil {
		t.Fatalf("Run unexpected err: %v", err)
	}
	if !report.AllOK {
		t.Fatalf("expected AllOK=true, got steps=%+v", report.Steps)
	}
	if fc.createProductCalls != 1 || fc.createRuleCalls != 1 || fc.createAuctionCalls != 1 {
		t.Fatalf("setup not called once: product=%d rule=%d auction=%d", fc.createProductCalls, fc.createRuleCalls, fc.createAuctionCalls)
	}
	if fc.lastProductActor.UserID != 2001 || fc.lastProductActor.Role != auction.RoleMerchant {
		t.Fatalf("product must be created as merchant seller, got %+v", fc.lastProductActor)
	}
	if fc.lastRuleActor.UserID != 2001 || fc.lastRuleActor.Role != auction.RoleMerchant {
		t.Fatalf("auction rule must be created as merchant seller, got %+v", fc.lastRuleActor)
	}
	if fc.lastAuctionActor.UserID != 2001 || fc.lastAuctionActor.Role != auction.RoleMerchant {
		t.Fatalf("auction must be created as merchant seller, got %+v", fc.lastAuctionActor)
	}
	if fc.bidCalls != len(o.cfg.BidderIDs) {
		t.Fatalf("expected %d bids, got %d", len(o.cfg.BidderIDs), fc.bidCalls)
	}
	if fc.subCalls < 1 {
		t.Fatalf("expected at least 1 subscribe call, got %d", fc.subCalls)
	}
	if !rec.deleted {
		t.Fatalf("cleanup recorder.DeleteByTestID was not called")
	}
	if len(rec.added) < 2 { // 至少 product + auction
		t.Fatalf("seed records too few: %+v", rec.added)
	}
	// 必须包含某些关键步骤
	wantSteps := []string{"create_product", "create_auction_rule", "create_auction", "wait_started", "bid", "wait_ended", "find_orders"}
	for _, w := range wantSteps {
		if !contains(em.events, w) {
			t.Fatalf("emitter missing step %q in %v", w, em.events)
		}
	}
}

// TestOrchestrator_SetupFailure: setup 阶段失败 → 后续阶段跳过，但 cleanup 仍执行
func TestOrchestrator_SetupFailure(t *testing.T) {
	fc := newFakeClient()
	fc.failCreateAuction = true
	rec := &fakeRecorder{}
	em := &fakeEmitter{}

	o := New(fc, rec, Config{
		TestID:       "test-setup-fail",
		SellerID:     2001,
		BidderIDs:    []int64{1001},
		SubscriberID: 3001,
		StartPrice:   100,
		Increment:    10,
		Duration:     30,
		PollInterval: 10 * time.Millisecond,
		PollTimeout:  100 * time.Millisecond,
	})

	report, err := o.Run(context.Background(), em)
	if err == nil {
		t.Fatalf("expected setup error, got nil; report=%+v", report)
	}
	if report.AllOK {
		t.Fatalf("expected AllOK=false")
	}
	if fc.bidCalls != 0 {
		t.Fatalf("bids must be skipped when setup fails, got %d", fc.bidCalls)
	}
	if !rec.deleted {
		t.Fatalf("cleanup must run even on setup failure")
	}
}

// TestOrchestrator_VerifyFailure: 验证阶段订单数 > 1 → AllOK=false 但 cleanup 仍执行
func TestOrchestrator_VerifyFailure(t *testing.T) {
	fc := newFakeClient()
	fc.orders = []auction.Order{
		{ID: 6001, AuctionID: 8001, WinnerID: 1001, FinalPrice: 199.0, Status: 1},
		{ID: 6002, AuctionID: 8001, WinnerID: 1001, FinalPrice: 199.0, Status: 1},
	}
	rec := &fakeRecorder{}
	em := &fakeEmitter{}

	o := New(fc, rec, Config{
		TestID:       "test-verify-fail",
		SellerID:     2001,
		BidderIDs:    []int64{1001},
		SubscriberID: 3001,
		StartPrice:   100,
		Increment:    10,
		Duration:     30,
		PollInterval: 10 * time.Millisecond,
		PollTimeout:  100 * time.Millisecond,
	})

	report, err := o.Run(context.Background(), em)
	if err != nil {
		// verify 失败不应作为 Run 的 error 抛出（区别于 setup 错误）
		t.Fatalf("verify failure should not bubble error, got %v", err)
	}
	if report.AllOK {
		t.Fatalf("AllOK should be false when verify fails")
	}
	if !rec.deleted {
		t.Fatalf("cleanup must run on verify failure")
	}
	// 至少应有一个 step 失败
	hasFail := false
	for _, s := range report.Steps {
		if !s.OK {
			hasFail = true
			break
		}
	}
	if !hasFail {
		t.Fatalf("expected at least one failed step in report")
	}
}

// TestOrchestrator_ContextCancelled: 跑到一半 ctx 取消 → 仍执行 cleanup
func TestOrchestrator_ContextCancelled(t *testing.T) {
	fc := newFakeClient()
	fc.orders = []auction.Order{{ID: 6001, AuctionID: 8001, WinnerID: 1001, FinalPrice: 199.0, Status: 1}}
	rec := &fakeRecorder{}
	em := &fakeEmitter{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	o := New(fc, rec, Config{
		TestID:       "test-ctx",
		SellerID:     2001,
		BidderIDs:    []int64{1001, 1002},
		SubscriberID: 3001,
		StartPrice:   100,
		Increment:    10,
		Duration:     30,
		PollInterval: 10 * time.Millisecond,
		PollTimeout:  100 * time.Millisecond,
	})

	_, err := o.Run(ctx, em)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if !rec.deleted {
		t.Fatalf("cleanup must run on ctx cancel")
	}
}

func contains(s []string, want string) bool {
	for _, v := range s {
		if v == want {
			return true
		}
	}
	return false
}

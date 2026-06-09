# H5 Demo Concurrent Bids Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 H5 `DemoConsole` 的「并发压测」按钮接入真实后端 demo 链路，用串行快速递增出价稳定抬高竞拍价格，让当前用户继续使用旧价出价时稳定失败。

**Architecture:** 前端只调用 `test-service` 的 `/api/test/demo/concurrent-bids`，不直连业务子服务。`test-service` 复用 demo JWT 鉴权，通过现有 auction SDK 经 `gateway-service` `/api/v1` 发起真实 `PlaceBid`，金额计算保持 `decimal.Decimal`，SDK 边界才转 `float64`。出价采用串行快速递增而非真并发，避免 `AuctionBidLock` 造成不稳定失败，并在触及 `cap_price` 前提前停止，避免直接成交。

**Tech Stack:** Go 1.24+, Hertz, `shopspring/decimal`, React 18, TypeScript, Jest, React Testing Library.

---

## File Structure

- Modify: `backend/test/client/auction/client.go`
  - 为 `AuctionRules` 增加 `CapPrice decimal.Decimal`，让 test-service 可读取业务规则封顶价。
- Modify: `backend/test/client/auction/client_test.go`
  - 增加 SDK 解析 `rules.cap_price` 的单测。
- Modify: `backend/test/handler/demo.go`
  - 增加 `concurrentBidsRequest`、校验/金额计算逻辑、`PostConcurrentBids` handler。
- Modify: `backend/test/handler/demo_test.go`
  - 增强 fake auction client，覆盖请求校验、串行递增、cap_price 提前停止、部分失败、全部失败。
- Modify: `backend/test/main.go`
  - 注册 `POST /api/test/demo/concurrent-bids` 路由。
- Modify: `frontend/h5/src/services/demoApi.ts`
  - 增加 `triggerConcurrentBids` API 方法与响应类型。
- Modify: `frontend/h5/src/services/__tests__/demoApi.test.ts`
  - 覆盖 endpoint 与 snake_case body。
- Modify: `frontend/h5/src/components/DemoConsole/index.tsx`
  - 将 placeholder 按钮替换为真实 handler，接入 pending、toast、auth retry。
- Modify: `frontend/h5/src/components/DemoConsole/__tests__/DemoConsole.test.tsx`
  - 覆盖点击触发、无当前竞拍、成功 toast、失败 toast。

---

### Task 1: Auction SDK CapPrice Contract

**Files:**
- Modify: `backend/test/client/auction/client.go`
- Test: `backend/test/client/auction/client_test.go`

- [ ] **Step 1: Write the failing SDK parsing test**

Append this test after `TestSDK_GetAuctionParsesRuleIncrement` in `backend/test/client/auction/client_test.go`:

```go
func TestSDK_GetAuctionParsesRuleCapPrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":7,"status":1,"current_price":"150.50","rules":{"increment":"20.00","cap_price":"300.00"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3*time.Second)
	a, step := c.GetAuction(context.Background(), 7)
	if !step.OK {
		t.Fatalf("GetAuction failed: %s err=%v", step.Message, step.Err)
	}
	if a.Rules == nil {
		t.Fatalf("rules must be parsed")
	}
	if got := a.Rules.CapPrice.StringFixed(2); got != "300.00" {
		t.Fatalf("cap_price: want 300.00, got %s", got)
	}
}
```

- [ ] **Step 2: Run the SDK test and verify it fails**

Run:

```bash
cd backend/test
go test ./client/auction -run TestSDK_GetAuctionParsesRuleCapPrice -count=1
```

Expected: FAIL with compile error similar to `a.Rules.CapPrice undefined`.

- [ ] **Step 3: Add `CapPrice` to SDK rules**

Modify `AuctionRules` in `backend/test/client/auction/client.go`:

```go
type AuctionRules struct {
	StartPrice decimal.Decimal `json:"start_price"`
	Increment  decimal.Decimal `json:"increment"`
	CapPrice   decimal.Decimal `json:"cap_price"`
}
```

- [ ] **Step 4: Run the SDK test and verify it passes**

Run:

```bash
cd backend/test
go test ./client/auction -run 'TestSDK_GetAuctionParsesRule(Increment|CapPrice)|TestSDK_GetAuctionParsesStringCurrentPrice' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/test/client/auction/client.go backend/test/client/auction/client_test.go
git commit -m "test: parse demo auction cap price"
```

---

### Task 2: Demo Handler Failing Tests

**Files:**
- Modify: `backend/test/handler/demo_test.go`

- [ ] **Step 1: Extend `fakeDemoAuctionClient` to observe bids**

Modify the fake struct and methods in `backend/test/handler/demo_test.go`:

```go
type fakeDemoAuctionClient struct {
	nextProductID        int64
	nextLiveStreamID     int64
	nextAuctionID        int64
	nextFixedID          int64
	waitStartedCalls     int
	startedLiveStreamIDs []int64
	productReqs          []auctioncli.CreateProductReq
	publishedProductIDs  []int64
	ruleReqs             []auctioncli.CreateAuctionRuleReq
	liveStreamReqs       []auctioncli.CreateLiveStreamReq
	auctionReqs          []auctioncli.CreateAuctionReq
	fixedReqs            []auctioncli.CreateFixedPriceItemReq
	skyLampUserID        int64
	skyLampAuctionID     int64
	followCalls          []struct {
		userID       int64
		liveStreamID int64
	}
	createAuctionConflict bool

	auction    auctioncli.Auction
	getStep    auctioncli.StepResult
	bidResults []auctioncli.StepResult
	bidCalls   []struct {
		userID    int64
		auctionID int64
		amount    float64
	}
}

func (f *fakeDemoAuctionClient) GetAuction(_ context.Context, auctionID int64) (auctioncli.Auction, auctioncli.StepResult) {
	if f.getStep.Step != "" {
		return f.auction, f.getStep
	}
	if f.auction.ID != 0 {
		return f.auction, f.ok("get_auction", f.auction.ID)
	}
	return auctioncli.Auction{ID: auctionID, CurrentPrice: 100, Rules: &auctioncli.AuctionRules{Increment: decimal.NewFromInt(10)}}, f.ok("get_auction", auctionID)
}

func (f *fakeDemoAuctionClient) PlaceBid(_ context.Context, userID int64, auctionID int64, amount float64) auctioncli.StepResult {
	f.bidCalls = append(f.bidCalls, struct {
		userID    int64
		auctionID int64
		amount    float64
	}{userID: userID, auctionID: auctionID, amount: amount})
	if len(f.bidResults) > 0 {
		result := f.bidResults[0]
		f.bidResults = f.bidResults[1:]
		return result
	}
	return f.ok("place_bid", auctionID)
}
```

- [ ] **Step 2: Add handler tests**

Append these tests near other `DemoHandler` tests in `backend/test/handler/demo_test.go`:

```go
func TestPostConcurrentBidsRejectsMissingAuctionID(t *testing.T) {
	const secret = "demo-secret"
	h := NewDemoHandler(&fakeDemoAuctionClient{}, nil, secret)
	c := newDemoRequestContext(t, secret, `{"bid_count":3}`)

	h.PostConcurrentBids(context.Background(), c)

	if c.Response.StatusCode() != 400 {
		t.Fatalf("status=%d want 400 body=%s", c.Response.StatusCode(), c.Response.Body())
	}
}

func TestPostConcurrentBidsPlacesSerialIncrementalBids(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{
		auction: auctioncli.Auction{
			ID:           77,
			CurrentPrice: 100,
			Rules: &auctioncli.AuctionRules{
				StartPrice: decimal.NewFromInt(80),
				Increment:  decimal.NewFromInt(10),
			},
		},
	}
	h := NewDemoHandler(fake, nil, secret)
	c := newDemoRequestContext(t, secret, `{"auction_id":77,"bid_count":3,"interval_ms":0}`)

	h.PostConcurrentBids(context.Background(), c)

	if c.Response.StatusCode() != 200 {
		t.Fatalf("status=%d want 200 body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	wantAmounts := []float64{110, 120, 130}
	if len(fake.bidCalls) != len(wantAmounts) {
		t.Fatalf("bid calls=%d want %d", len(fake.bidCalls), len(wantAmounts))
	}
	for i, want := range wantAmounts {
		if fake.bidCalls[i].userID != buyerBUserID {
			t.Fatalf("call %d user_id=%d want %d", i, fake.bidCalls[i].userID, buyerBUserID)
		}
		if fake.bidCalls[i].auctionID != 77 {
			t.Fatalf("call %d auction_id=%d want 77", i, fake.bidCalls[i].auctionID)
		}
		if fake.bidCalls[i].amount != want {
			t.Fatalf("call %d amount=%f want %f", i, fake.bidCalls[i].amount, want)
		}
	}
}

func TestPostConcurrentBidsStopsBeforeCapPrice(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{
		auction: auctioncli.Auction{
			ID:           77,
			CurrentPrice: 100,
			Rules: &auctioncli.AuctionRules{
				StartPrice: decimal.NewFromInt(80),
				Increment:  decimal.NewFromInt(10),
				CapPrice:   decimal.NewFromInt(125),
			},
		},
	}
	h := NewDemoHandler(fake, nil, secret)
	c := newDemoRequestContext(t, secret, `{"auction_id":77,"bid_count":5,"interval_ms":0}`)

	h.PostConcurrentBids(context.Background(), c)

	if c.Response.StatusCode() != 200 {
		t.Fatalf("status=%d want 200 body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	wantAmounts := []float64{110, 120}
	if len(fake.bidCalls) != len(wantAmounts) {
		t.Fatalf("bid calls=%d want %d", len(fake.bidCalls), len(wantAmounts))
	}
	for i, want := range wantAmounts {
		if fake.bidCalls[i].amount != want {
			t.Fatalf("call %d amount=%f want %f", i, fake.bidCalls[i].amount, want)
		}
	}
}

func TestPostConcurrentBidsReturnsOKWhenPartiallyFailed(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{
		auction: auctioncli.Auction{
			ID:           77,
			CurrentPrice: 100,
			Rules:        &auctioncli.AuctionRules{Increment: decimal.NewFromInt(10)},
		},
		bidResults: []auctioncli.StepResult{
			{Step: "bid", OK: false, StatusCode: 400, Message: "出价过于频繁，请稍后再试"},
			{Step: "bid", OK: true, StatusCode: 200},
		},
	}
	h := NewDemoHandler(fake, nil, secret)
	c := newDemoRequestContext(t, secret, `{"auction_id":77,"bid_count":2,"interval_ms":0}`)

	h.PostConcurrentBids(context.Background(), c)

	if c.Response.StatusCode() != 200 {
		t.Fatalf("status=%d want 200 body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	var body map[string]any
	if err := json.Unmarshal(c.Response.Body(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["ok"] != true || int(body["success_count"].(float64)) != 1 || int(body["failure_count"].(float64)) != 1 {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestPostConcurrentBidsReturnsBadRequestWhenAllFailed(t *testing.T) {
	const secret = "demo-secret"
	fake := &fakeDemoAuctionClient{
		auction: auctioncli.Auction{
			ID:           77,
			CurrentPrice: 100,
			Rules:        &auctioncli.AuctionRules{Increment: decimal.NewFromInt(10)},
		},
		bidResults: []auctioncli.StepResult{
			{Step: "bid", OK: false, StatusCode: 400, Message: "竞拍已结束，无法出价"},
			{Step: "bid", OK: false, StatusCode: 400, Message: "竞拍已结束，无法出价"},
		},
	}
	h := NewDemoHandler(fake, nil, secret)
	c := newDemoRequestContext(t, secret, `{"auction_id":77,"bid_count":2,"interval_ms":0}`)

	h.PostConcurrentBids(context.Background(), c)

	if c.Response.StatusCode() != 400 {
		t.Fatalf("status=%d want 400 body=%s", c.Response.StatusCode(), c.Response.Body())
	}
	var body map[string]any
	if err := json.Unmarshal(c.Response.Body(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["ok"] != false || int(body["success_count"].(float64)) != 0 || body["last_error"] != "竞拍已结束，无法出价" {
		t.Fatalf("unexpected response: %+v", body)
	}
}
```

- [ ] **Step 3: Run handler tests and verify they fail**

Run:

```bash
cd backend/test
go test ./handler -run 'TestPostConcurrentBids|TestPostFollowBid' -count=1
```

Expected: FAIL with compile error `h.PostConcurrentBids undefined`.

- [ ] **Step 4: Commit failing tests**

```bash
git add backend/test/handler/demo_test.go
git commit -m "test: cover demo concurrent bid handler"
```

---

### Task 3: Demo Handler Implementation

**Files:**
- Modify: `backend/test/handler/demo.go`
- Test: `backend/test/handler/demo_test.go`

- [ ] **Step 1: Add request type and constants**

Add below `followBidRequest` in `backend/test/handler/demo.go`:

```go
type concurrentBidsRequest struct {
	AuctionID  int64           `json:"auction_id"`
	BidCount   int             `json:"bid_count,omitempty"`
	IntervalMS int             `json:"interval_ms,omitempty"`
	Increment  json.RawMessage `json:"increment,omitempty"`
}
```

Add near the demo constants:

```go
const (
	defaultConcurrentBidCount      = 6
	maxConcurrentBidCount          = 20
	defaultConcurrentBidIntervalMS = 80
	maxConcurrentBidIntervalMS     = 1000
)
```

- [ ] **Step 2: Add validation and amount helpers**

Add these helpers near `computeFollowBidAmount`:

```go
func normalizeConcurrentBidsRequest(req *concurrentBidsRequest) error {
	if req.AuctionID <= 0 {
		return fmt.Errorf("auction_id is required")
	}
	if req.BidCount == 0 {
		req.BidCount = defaultConcurrentBidCount
	}
	if req.BidCount < 1 || req.BidCount > maxConcurrentBidCount {
		return fmt.Errorf("bid_count must be between 1 and %d", maxConcurrentBidCount)
	}
	if req.IntervalMS == 0 {
		req.IntervalMS = defaultConcurrentBidIntervalMS
	}
	if req.IntervalMS < 0 || req.IntervalMS > maxConcurrentBidIntervalMS {
		return fmt.Errorf("interval_ms must be between 0 and %d", maxConcurrentBidIntervalMS)
	}
	return nil
}

func effectiveConcurrentIncrement(requested *decimal.Decimal, ruleIncrement decimal.Decimal) decimal.Decimal {
	increment := ruleIncrement
	if requested != nil && requested.IsPositive() {
		increment = *requested
	}
	if ruleIncrement.IsPositive() && increment.LessThan(ruleIncrement) {
		increment = ruleIncrement
	}
	if !increment.IsPositive() {
		increment = decimal.NewFromInt(1)
	}
	return increment
}

func concurrentBidAmount(baseline, increment decimal.Decimal, index int) decimal.Decimal {
	return baseline.Add(increment.Mul(decimal.NewFromInt(int64(index + 1))))
}
```

- [ ] **Step 3: Implement `PostConcurrentBids`**

Add after `PostFollowBid` in `backend/test/handler/demo.go`:

```go
// PostConcurrentBids 以统一 seed 的买家B身份串行快速递增出价，稳定抬高当前拍卖价格。
func (h *DemoHandler) PostConcurrentBids(ctx context.Context, c *app.RequestContext) {
	if !h.authorizeDemoRequest(c) {
		return
	}
	if h.bizCli == nil {
		c.JSON(500, map[string]any{"error": "demo auction client is not configured"})
		return
	}

	var req concurrentBidsRequest
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		c.JSON(400, map[string]any{"error": "invalid concurrent bids request"})
		return
	}
	if err := normalizeConcurrentBidsRequest(&req); err != nil {
		c.JSON(400, map[string]any{"error": err.Error()})
		return
	}

	requestedIncrement, err := parseOptionalFollowBidAmount(req.Increment)
	if err != nil {
		c.JSON(400, map[string]any{"error": "invalid increment"})
		return
	}

	auction, step := h.bizCli.GetAuction(ctx, req.AuctionID)
	if !step.OK {
		c.JSON(400, map[string]any{"error": step.Message, "status": step.StatusCode})
		return
	}

	current := decimal.NewFromFloat(auction.CurrentPrice)
	start := decimal.Zero
	ruleIncrement := decimal.Zero
	capPrice := decimal.Zero
	if auction.Rules != nil {
		start = auction.Rules.StartPrice
		ruleIncrement = auction.Rules.Increment
		capPrice = auction.Rules.CapPrice
	}
	baseline := current
	if start.GreaterThan(baseline) {
		baseline = start
	}
	increment := effectiveConcurrentIncrement(requestedIncrement, ruleIncrement)

	successCount := 0
	failureCount := 0
	highestAmount := decimal.Zero
	lastError := ""

	for i := 0; i < req.BidCount; i++ {
		if i > 0 && req.IntervalMS > 0 {
			select {
			case <-ctx.Done():
				lastError = ctx.Err().Error()
				failureCount++
				i = req.BidCount
				continue
			case <-time.After(time.Duration(req.IntervalMS) * time.Millisecond):
			}
		}

		amount := concurrentBidAmount(baseline, increment, i)
		if capPrice.IsPositive() && amount.GreaterThanOrEqual(capPrice) {
			break
		}

		bidAmount, err := decimalToBidAmount(amount)
		if err != nil {
			lastError = err.Error()
			failureCount++
			continue
		}

		hlog.CtxInfof(ctx, "[demo] concurrent-bids auction=%d amount=%s as buyerB=%d", req.AuctionID, amount, buyerBUserID)
		step := h.bizCli.PlaceBid(ctx, buyerBUserID, req.AuctionID, bidAmount)
		if !step.OK {
			lastError = step.Message
			failureCount++
			continue
		}
		successCount++
		highestAmount = amount
	}

	status := 200
	ok := true
	if successCount == 0 {
		status = 400
		ok = false
		if lastError == "" {
			lastError = "no bid was placed"
		}
	}

	c.JSON(status, map[string]any{
		"ok":             ok,
		"auction_id":     req.AuctionID,
		"buyer_user_id":  buyerBUserID,
		"success_count":  successCount,
		"failure_count":  failureCount,
		"highest_amount": highestAmount.String(),
		"last_error":     lastError,
	})
}
```

- [ ] **Step 4: Run handler tests**

Run:

```bash
cd backend/test
go test ./handler -run 'TestPostConcurrentBids|TestPostFollowBid|TestComputeFollowBidAmount' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit implementation**

```bash
git add backend/test/handler/demo.go backend/test/handler/demo_test.go
git commit -m "feat: add demo concurrent bids handler"
```

---

### Task 4: Test-Service Route Registration

**Files:**
- Modify: `backend/test/main.go`

- [ ] **Step 1: Register the route**

Modify the demo route group in `backend/test/main.go`:

```go
demo := api.Group("/demo")
demo.POST("/follow-bid", demoHandler.PostFollowBid)
demo.POST("/concurrent-bids", demoHandler.PostConcurrentBids)
demo.POST("/sky-lamp", demoHandler.PostSkyLamp)
demo.POST("/recharge", demoHandler.PostRecharge)
demo.POST("/auctions/shorten", demoHandler.PostShortenAuction)
```

- [ ] **Step 2: Compile test-service**

Run:

```bash
cd backend/test
go test ./... -run 'TestPostConcurrentBids|TestSDK_GetAuctionParsesRuleCapPrice' -count=1
```

Expected: PASS.

- [ ] **Step 3: Commit route registration**

```bash
git add backend/test/main.go
git commit -m "feat: register demo concurrent bids endpoint"
```

---

### Task 5: H5 Demo API Client

**Files:**
- Modify: `frontend/h5/src/services/demoApi.ts`
- Test: `frontend/h5/src/services/__tests__/demoApi.test.ts`

- [ ] **Step 1: Write the failing frontend API test**

Modify imports in `frontend/h5/src/services/__tests__/demoApi.test.ts`:

```ts
import {
  createDemoFixedPriceItem,
  createDemoMerchantAuction,
  rechargeDemoUser,
  shortenDemoAuction,
  triggerConcurrentBids,
  triggerOtherSkyLamp,
  triggerFollowBid,
} from '../demoApi';
```

Add this test before the final `});`:

```ts
  it('posts concurrent bids request to the demo endpoint with snake_case fields', async () => {
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ ok: true, highest_amount: '160' }),
    } as Response);
    global.fetch = fetchMock;

    await triggerConcurrentBids({ auctionId: 456, bidCount: 6, intervalMs: 80, increment: '10' });

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/test/demo/concurrent-bids');
    expect(JSON.parse(init.body as string)).toEqual({
      auction_id: 456,
      bid_count: 6,
      interval_ms: 80,
      increment: '10',
    });
  });
```

- [ ] **Step 2: Run the frontend API test and verify it fails**

Run:

```bash
cd frontend/h5
npm test -- --runTestsByPath src/services/__tests__/demoApi.test.ts --runInBand
```

Expected: FAIL with `triggerConcurrentBids is not a function` or export error.

- [ ] **Step 3: Implement `triggerConcurrentBids`**

Add types near existing demo input types in `frontend/h5/src/services/demoApi.ts`:

```ts
export type TriggerConcurrentBidsInput = {
  auctionId: number;
  bidCount?: number;
  intervalMs?: number;
  increment?: MoneyInput;
};

export type TriggerConcurrentBidsResponse = {
  ok: boolean;
  auction_id: number;
  buyer_user_id?: number;
  success_count: number;
  failure_count: number;
  highest_amount: string;
  last_error?: string;
};
```

Add function after `triggerFollowBid`:

```ts
export function triggerConcurrentBids(input: TriggerConcurrentBidsInput) {
  const body: Record<string, unknown> = {
    auction_id: input.auctionId,
  };

  if (input.bidCount !== undefined) {
    body.bid_count = input.bidCount;
  }
  if (input.intervalMs !== undefined) {
    body.interval_ms = input.intervalMs;
  }
  if (input.increment !== undefined) {
    body.increment = toMoneyString(input.increment);
  }

  return postDemo<TriggerConcurrentBidsResponse>('/concurrent-bids', body);
}
```

- [ ] **Step 4: Run frontend API tests**

Run:

```bash
cd frontend/h5
npm test -- --runTestsByPath src/services/__tests__/demoApi.test.ts --runInBand
```

Expected: PASS.

- [ ] **Step 5: Commit frontend API client**

```bash
git add frontend/h5/src/services/demoApi.ts frontend/h5/src/services/__tests__/demoApi.test.ts
git commit -m "feat: add h5 demo concurrent bids api"
```

---

### Task 6: DemoConsole Integration

**Files:**
- Modify: `frontend/h5/src/components/DemoConsole/index.tsx`
- Test: `frontend/h5/src/components/DemoConsole/__tests__/DemoConsole.test.tsx`

- [ ] **Step 1: Write failing DemoConsole tests**

Modify imports in `frontend/h5/src/components/DemoConsole/__tests__/DemoConsole.test.tsx`:

```ts
import {
  createDemoFixedPriceItem,
  createDemoMerchantAuction,
  rechargeDemoUser,
  shortenDemoAuction,
  triggerConcurrentBids,
  triggerOtherSkyLamp,
  triggerFollowBid,
} from '../../../services/demoApi';
```

Modify the `jest.mock('../../../services/demoApi', ...)` block:

```ts
jest.mock('../../../services/demoApi', () => ({
  createDemoFixedPriceItem: jest.fn(),
  createDemoMerchantAuction: jest.fn(),
  shortenDemoAuction: jest.fn(),
  triggerConcurrentBids: jest.fn(),
  triggerOtherSkyLamp: jest.fn(),
  triggerFollowBid: jest.fn(),
  rechargeDemoUser: jest.fn(),
}));
```

Add mocked function near the other mocks:

```ts
const mockedTriggerConcurrentBids = triggerConcurrentBids as jest.MockedFunction<typeof triggerConcurrentBids>;
```

Add default mock in `beforeEach`:

```ts
mockedTriggerConcurrentBids.mockResolvedValue({
  ok: true,
  auction_id: 777,
  success_count: 6,
  failure_count: 0,
  highest_amount: '160',
  last_error: '',
});
```

Add these tests near the existing demo menu tests:

```ts
  it('triggers concurrent bids for the current auction and reports the raised price', async () => {
    const user = userEvent.setup();
    renderConsole(777);

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '演示' }));
    await user.click(screen.getByRole('button', { name: '并发压测' }));

    expect(mockedTriggerConcurrentBids).toHaveBeenCalledWith({ auctionId: 777 });
    expect(mockShowToast).toHaveBeenCalledWith('并发出价已抬到 ¥160，请尝试用旧价出价', 'success', 2500);
  });

  it('warns and skips concurrent bids when there is no current auction', async () => {
    const user = userEvent.setup();
    renderConsole(null);

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '演示' }));
    await user.click(screen.getByRole('button', { name: '并发压测' }));

    expect(mockShowToast).toHaveBeenCalledWith('请先进入直播间', 'warning', 2500);
    expect(mockedTriggerConcurrentBids).not.toHaveBeenCalled();
  });

  it('shows a short error toast when concurrent bids fail', async () => {
    const user = userEvent.setup();
    mockedTriggerConcurrentBids.mockRejectedValueOnce(new Error('竞拍已结束，无法出价'));
    renderConsole(777);

    await user.click(screen.getByTestId('demo-console-fab'));
    await user.click(screen.getByRole('button', { name: '演示' }));
    await user.click(screen.getByRole('button', { name: '并发压测' }));

    expect(mockShowToast).toHaveBeenCalledWith('并发压测失败：竞拍已结束，无法出价', 'error', 2500);
  });
```

- [ ] **Step 2: Run DemoConsole tests and verify they fail**

Run:

```bash
cd frontend/h5
npm test -- --runTestsByPath src/components/DemoConsole/__tests__/DemoConsole.test.tsx --runInBand
```

Expected: FAIL because the button still calls placeholder toast and never calls `triggerConcurrentBids`.

- [ ] **Step 3: Implement DemoConsole handler**

Modify import in `frontend/h5/src/components/DemoConsole/index.tsx`:

```ts
import {
  createDemoFixedPriceItem,
  createDemoMerchantAuction,
  rechargeDemoUser,
  shortenDemoAuction,
  triggerConcurrentBids,
  triggerOtherSkyLamp,
  triggerFollowBid,
} from '../../services/demoApi';
```

Add handler after `handleFollowBid`:

```tsx
  const handleConcurrentBids = async () => {
    if (!currentAuctionId) {
      showToast('请先进入直播间', 'warning', TOAST_DURATION_MS);
      return;
    }

    setRunningAction('concurrent-bids');
    try {
      const result = await runWithDemoAuthRetry(() => triggerConcurrentBids({ auctionId: currentAuctionId }));
      if (result.highest_amount) {
        showToast(`并发出价已抬到 ¥${result.highest_amount}，请尝试用旧价出价`, 'success', TOAST_DURATION_MS);
      } else {
        showToast('已触发并发出价', 'success', TOAST_DURATION_MS);
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : '请稍后重试';
      showToast(`并发压测失败：${message}`, 'error', TOAST_DURATION_MS);
    } finally {
      setRunningAction(null);
    }
  };
```

Replace the placeholder button:

```tsx
              <button
                type="button"
                className="demo-console__item"
                onClick={handleConcurrentBids}
                disabled={runningAction === 'concurrent-bids'}
              >
                并发压测
              </button>
```

Remove `showPromptOnlyAction` if it has no remaining callers.

- [ ] **Step 4: Run DemoConsole tests**

Run:

```bash
cd frontend/h5
npm test -- --runTestsByPath src/components/DemoConsole/__tests__/DemoConsole.test.tsx --runInBand
```

Expected: PASS.

- [ ] **Step 5: Commit DemoConsole integration**

```bash
git add frontend/h5/src/components/DemoConsole/index.tsx frontend/h5/src/components/DemoConsole/__tests__/DemoConsole.test.tsx
git commit -m "feat: wire demo console concurrent bids"
```

---

### Task 7: Cross-Layer Verification

**Files:**
- Verify only, no planned source edits.

- [ ] **Step 1: Run backend focused tests**

Run:

```bash
cd backend/test
go test ./client/auction ./handler -run 'TestSDK_GetAuctionParsesRuleCapPrice|TestPostConcurrentBids|TestPostFollowBid|TestComputeFollowBidAmount' -count=1
```

Expected: PASS.

- [ ] **Step 2: Run frontend focused tests**

Run:

```bash
cd frontend/h5
npm test -- --runTestsByPath src/services/__tests__/demoApi.test.ts src/components/DemoConsole/__tests__/DemoConsole.test.tsx --runInBand
```

Expected: PASS.

- [ ] **Step 3: Run compile/build checks**

Run:

```bash
cd backend/test
go test ./... -count=1
```

Expected: PASS.

Run:

```bash
cd frontend/h5
npm run build
```

Expected: PASS.

- [ ] **Step 4: Manual verification in local demo**

After local services are running, verify:

```text
1. 使用 H5 进入正在竞拍直播间。
2. 打开 DemoConsole -> 演示 -> 并发压测。
3. 页面出现真实 bid_placed 驱动的飘屏/排行/热度变化。
4. toast 显示「并发出价已抬到 ¥<amount>，请尝试用旧价出价」。
5. 当前用户继续用旧价点击「立即出价」。
6. 页面显示现有出价失败提示，原因是价格已被抬高。
7. 当前竞拍未因 cap_price 被直接成交结束。
```

- [ ] **Step 5: Final commit if verification changed files**

If no source files changed during verification:

```bash
git status --short
```

Expected: only intended committed changes, or clean working tree.

If verification fixes were required:

```bash
git add <changed-files>
git commit -m "fix: stabilize demo concurrent bids"
```

---

## Self-Review

- Spec coverage:
  - Backend API `/api/test/demo/concurrent-bids`: Task 3 + Task 4.
  - Serial fast incremental bidding: Task 2 + Task 3.
  - Demo JWT auth and fixed demo user: Task 3 reuses `authorizeDemoRequest` and `buyerBUserID`.
  - Gateway-only real bid path: Task 3 uses existing `demoAuctionClient.PlaceBid`.
  - `decimal.Decimal` money handling: Task 3 keeps decimal until `decimalToBidAmount`.
  - `cap_price` boundary: Task 1 + Task 2 + Task 3.
  - Frontend API and DemoConsole: Task 5 + Task 6.
  - Tests and verification: Task 1, Task 2, Task 5, Task 6, Task 7.
- Placeholder scan:
  - No deferred work markers, no vague edge handling, no unspecified test steps.
- Type consistency:
  - `triggerConcurrentBids({ auctionId })` matches `TriggerConcurrentBidsInput`.
  - `highest_amount` response field matches backend JSON and frontend toast logic.
  - `AuctionRules.CapPrice` is defined before handler code reads it.

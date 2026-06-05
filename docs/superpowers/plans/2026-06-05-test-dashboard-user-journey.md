# Test Dashboard User Journey Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `frontend/test-dashboard` 和 `backend/test` 落地买家视角的 `user_journey` 验收场景，复用现有测试平台任务、WS 进度和报告能力，覆盖准备数据、进直播间、关注、出价、点天灯、一口价购买与最终校验。

**Architecture:** 后端沿用 `handler -> runner -> scenario` 模式，在 `backend/test/scenario/user_journey` 实现 orchestration，并扩展 `backend/test/client/auction` 支持多角色、余额充值和买家链路 API；前端沿用单场景页面模式，在 `frontend/test-dashboard` 增加 `UserJourney` 页面、路由和 API DTO，复用 `usePollReport`、`wsStore` 和 `StepTimeline` 展示实时进度与最终报告。T0（商家开播鉴权修复）视为已完成前置，不在本计划内重复实现。

**Tech Stack:** Go 1.24+, Hertz, GORM, shopspring/decimal, React, TypeScript, Axios, React Router.

---

## File Scope

- Modify: `backend/test/client/auction/client.go`
- Modify: `backend/test/client/auction/client_test.go`
- Modify: `backend/test/model/test.go`
- Modify: `backend/test/handler/test.go`
- Modify: `backend/test/main.go`
- Modify: `backend/test/scenario/e2e/orchestrator.go`（仅在复用 helper 需要时）
- Create: `backend/test/scenario/user_journey/orchestrator.go`
- Create: `backend/test/scenario/user_journey/orchestrator_test.go`
- Create: `backend/test/scenario/user_journey/scenario.go`
- Modify: `backend/auction/main.go`
- Modify: `backend/auction/handler/user_balance_http.go`
- Modify: `backend/auction/handler/user_balance_http_test.go`
- Modify: `frontend/test-dashboard/src/api/test.ts`
- Modify: `frontend/test-dashboard/src/App.tsx`
- Modify: `frontend/test-dashboard/src/components/Layout.tsx`
- Modify: `frontend/test-dashboard/src/components/StepTimeline.tsx`
- Create: `frontend/test-dashboard/src/pages/UserJourney.tsx`
- Modify: `frontend/test-dashboard/src/pages/Report.tsx`

## Preconditions

- T0 已完成且已合入 `main`：`c542b4fc fix: enforce merchant-owned live start auth`
- user journey 继续基于 Gateway `/api/v1` 入口，不允许前端或 `backend/test` 直接访问后端子服务业务接口
- 金额字段统一使用 decimal string，不得在报告结构中暴露 float 业务金额

## Task T1: Extend test-service business client and result contract

**Files:**
- Modify: `backend/test/client/auction/client.go`
- Modify: `backend/test/client/auction/client_test.go`
- Modify: `backend/test/model/test.go`

- [ ] **Step 1: Write failing client tests for multi-role headers and new user journey endpoints**

Add tests in `backend/test/client/auction/client_test.go` for:

```go
func TestDoSetsMerchantIdentityHeaders(t *testing.T) {}
func TestTopUpUserBalanceCallsInternalEndpoint(t *testing.T) {}
func TestPurchaseFixedPriceIncludesIdempotencyKey(t *testing.T) {}
func TestFollowAndFollowStatusUseBuyerIdentity(t *testing.T) {}
```

Each test should assert:
- role `merchant` maps to `X-User-Role: merchant`
- role `user` maps to `X-User-Role: user`
- internal top-up hits `/internal/test/user-balance`
- fixed-price purchase sets `X-Idempotency-Key`

- [ ] **Step 2: Run the focused client tests to verify they fail**

Run:

```bash
cd backend/test
go test ./client/auction -run 'Test(DoSetsMerchantIdentityHeaders|TopUpUserBalanceCallsInternalEndpoint|PurchaseFixedPriceIncludesIdempotencyKey|FollowAndFollowStatusUseBuyerIdentity)' -count=1
```

Expected: FAIL because the current client hardcodes `X-User-Role: user` and does not expose top-up / follow / fixed-price purchase methods.

- [ ] **Step 3: Implement the minimal client extensions**

In `backend/test/client/auction/client.go`:

- introduce an explicit actor model:

```go
type Actor struct {
	UserID   int64
	Username string
	Role     string // "user" | "merchant"
}
```

- replace the raw `userID int64` request identity path with actor-aware helpers
- add methods:
  - `CreateLiveStream`
  - `StartLive`
  - `ListFixedPriceItemsByLiveStream`
  - `FollowLiveStream`
  - `GetFollowStatus`
  - `GetUserBalance`
  - `TopUpUserBalance`
  - `CreateFixedPriceItem`
  - `PurchaseFixedPriceItem`
  - `GetMyFixedPricePurchase`
  - `ListOrders`
- keep existing `e2e` methods working

- [ ] **Step 4: Add test type constant for user journey**

Update `backend/test/model/test.go`:

```go
const (
    ...
    TypeUserJourney = "user_journey"
)
```

- [ ] **Step 5: Run client package tests**

Run:

```bash
cd backend/test
go test ./client/auction -count=1
```

Expected: PASS.

## Task T2: Add internal buyer balance top-up endpoint

**Files:**
- Modify: `backend/auction/handler/user_balance_http.go`
- Modify: `backend/auction/handler/user_balance_http_test.go`
- Modify: `backend/auction/main.go`

- [ ] **Step 1: Write failing handler tests for internal top-up**

Add tests in `backend/auction/handler/user_balance_http_test.go` for:

```go
func TestTopUpUserBalanceInternalAddsAmount(t *testing.T) {}
func TestTopUpUserBalanceInternalRejectsInvalidDecimal(t *testing.T) {}
func TestTopUpUserBalanceInternalRejectsNonPositiveAmount(t *testing.T) {}
```

Expected assertions:
- request body `{"user_id":1001,"amount":"500.00"}` succeeds
- response returns updated balance as decimal string
- invalid decimal returns 400
- zero / negative amount returns 400

- [ ] **Step 2: Run focused handler tests to verify they fail**

Run:

```bash
cd backend/auction
go test ./handler -run 'TestTopUpUserBalanceInternal' -count=1
```

Expected: FAIL because the internal top-up handler/route does not exist yet.

- [ ] **Step 3: Implement the internal top-up handler**

In `backend/auction/handler/user_balance_http.go`:
- add a request DTO with `user_id` and decimal `amount`
- reuse existing balance DAO / service path if possible
- return fail-closed JSON on invalid input

Example response shape:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": 1001,
    "balance": "500.00"
  }
}
```

- [ ] **Step 4: Register the internal route**

In `backend/auction/main.go`, add under `/internal`:

```go
internal.POST("/test/user-balance", userBalanceHandler.TopUpInternal)
```

This route must remain protected by `InternalAuthMiddleware`.

- [ ] **Step 5: Run affected auction handler tests**

Run:

```bash
cd backend/auction
go test ./handler -run 'Test(TopUpUserBalanceInternal|GetUserBalance)' -count=1
```

Expected: PASS.

## Task T3: Implement backend test-service user_journey scenario

**Files:**
- Create: `backend/test/scenario/user_journey/orchestrator.go`
- Create: `backend/test/scenario/user_journey/orchestrator_test.go`
- Create: `backend/test/scenario/user_journey/scenario.go`
- Modify: `backend/test/handler/test.go`
- Modify: `backend/test/main.go`

- [ ] **Step 1: Write failing orchestrator tests**

Create `backend/test/scenario/user_journey/orchestrator_test.go` with tests for:

```go
func TestRunHappyPathProducesEvidenceReport(t *testing.T) {}
func TestPrepareFailsClosedWhenTopUpFails(t *testing.T) {}
func TestPrepareSkipsCleanupAndStillRecordsSeedRefs(t *testing.T) {}
func TestReminderStepUsesFollowAndFollowStatusOnly(t *testing.T) {}
```

The fake business client should assert the following sequence:
- create product
- create live stream
- create auction
- create fixed-price item
- top up buyer balance
- start live
- enter live / list fixed-price items
- follow / follow-status
- bid
- sky-lamp
- purchase fixed price
- verify orders / balance

- [ ] **Step 2: Run focused orchestrator tests to verify they fail**

Run:

```bash
cd backend/test
go test ./scenario/user_journey -run 'Test(RunHappyPathProducesEvidenceReport|PrepareFailsClosedWhenTopUpFails|PrepareSkipsCleanupAndStillRecordsSeedRefs|ReminderStepUsesFollowAndFollowStatusOnly)' -count=1
```

Expected: FAIL because the package does not exist yet.

- [ ] **Step 3: Implement scenario config/report types and orchestration**

In `backend/test/scenario/user_journey/orchestrator.go`:
- define config matching the spec
- define report fields:
  - `test_run_id`
  - buyer / merchant ids
  - `product_id`, `live_stream_id`, `auction_id`, `fixed_price_item_id`, `order_id`
  - `balance_before`, `balance_after`
  - `stock_before`, `stock_after`
  - `steps`
  - `all_ok`
  - `warnings`
- emit progress via `runner.ProgressEmitter`
- record seed refs through `SeedRecorder.Add`
- do **not** call cleanup by default

- [ ] **Step 4: Register scenario type and HTTP endpoint**

Update:
- `backend/test/main.go` to register `user_journey.NewScenario(...)`
- `backend/test/handler/test.go` to add:

```go
func (h *TestHandler) PostUserJourney(ctx context.Context, c *app.RequestContext)
```

and route:

```go
api.POST("/user-journey", th.PostUserJourney)
```

- [ ] **Step 5: Run backend test-service focused regression**

Run:

```bash
cd backend/test
go test ./scenario/user_journey ./handler -run 'Test(UserJourney|PostUserJourney)' -count=1
```

Expected: PASS.

## Task T4: Add frontend test-dashboard user journey page

**Files:**
- Modify: `frontend/test-dashboard/src/api/test.ts`
- Modify: `frontend/test-dashboard/src/App.tsx`
- Modify: `frontend/test-dashboard/src/components/Layout.tsx`
- Modify: `frontend/test-dashboard/src/components/StepTimeline.tsx`
- Create: `frontend/test-dashboard/src/pages/UserJourney.tsx`
- Modify: `frontend/test-dashboard/src/pages/Report.tsx`

- [ ] **Step 1: Write failing frontend tests or type-level checks**

If the project has no existing frontend test harness for this area, use build/type verification as the minimum gate and record that in state before implementation.

Add the following type shapes in `src/api/test.ts` first:

```ts
export interface UserJourneyConfig { ... }
export interface UserJourneyReport { ... }
export async function startUserJourney(config: UserJourneyConfig): Promise<string> {}
```

- [ ] **Step 2: Run frontend build to verify missing symbols**

Run:

```bash
cd frontend/test-dashboard
npm run build
```

Expected: FAIL once `UserJourney.tsx` / route references are added before implementation is complete.

- [ ] **Step 3: Implement the page and route**

In `frontend/test-dashboard/src/pages/UserJourney.tsx`:
- follow the `E2E.tsx` interaction model
- expose toggles:
  - `include_reminder`
  - `include_sky_lamp`
  - `include_fixed_price`
  - `auction_duration_sec`
  - `buyer_count`
  - `keep_evidence`
- show:
  - runtime `test_id`
  - WS progress
  - timeline
  - evidence cards
  - link to `/test/report/:id`

Update navigation and routes:
- `App.tsx` add `/test/user-journey`
- `Layout.tsx` add sidebar nav entry

Update `StepTimeline.tsx` labels for new user-journey steps:
- `prepare`
- `enter_live`
- `reminder`
- `auction_bid`
- `sky_lamp`
- `fixed_price_purchase`
- `verify`

- [ ] **Step 4: Improve report readability for user_journey**

In `frontend/test-dashboard/src/pages/Report.tsx`, keep generic JSON rendering, but if `TestType === "user_journey"` add a compact summary block above raw JSON for:
- ids
- balances
- stock
- all_ok / warnings

- [ ] **Step 5: Run frontend build**

Run:

```bash
cd frontend/test-dashboard
npm run build
```

Expected: PASS.

## Task T5: Cross-module verification and documentation sync

**Files:**
- Modify only if required by implementation fallout.

- [ ] **Step 1: Run backend affected package tests**

Run:

```bash
cd backend/test
go test ./client/auction ./scenario/user_journey ./handler -count=1
cd ../auction
go test ./handler -run 'Test(TopUpUserBalanceInternal|GetUserBalance|.*StartLive)' -count=1
```

- [ ] **Step 2: Run frontend build and smoke checks**

Run:

```bash
cd frontend/test-dashboard
npm run build
```

- [ ] **Step 3: Run git diff contract scan**

Run:

```bash
rg -n "user-journey|user_journey|/test/user-journey|/api/test/user-journey|/internal/test/user-balance" backend frontend docs
```

Expected: all introduced routes and test types are consistently wired.

- [ ] **Step 4: Update SDD state before reporting**

Record:
- modified files
- red/green test evidence
- skipped validations, if any
- residual risks

## Spec Coverage Check

- `5.1.1 多角色造数` → T1, T3
- `5.1.2 买家余额准备` → T2, T3
- `5.2 前端页面` → T4
- `6 P0 用户验收剧本` → T3, T4
- `6.2 证据保留` → T3, T4
- `7 P1 稳定性扩展` → 本计划不实现，仅保留配置与后续扩展位

## Placeholder Scan

- No `TODO` / `TBD`
- each task has explicit files and commands
- T0 marked as prerequisite, not duplicated

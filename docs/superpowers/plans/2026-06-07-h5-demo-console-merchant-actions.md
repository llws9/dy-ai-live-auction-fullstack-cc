# H5 Demo Console 商家动作实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 H5 Demo Console 增加一级菜单 `商家`，二级菜单包含 `即将开播`、`正在竞拍`、`一口价`。所有动作都走真实后端链路，并且每次点击创建新的 demo 商品，天然规避“同一商品同一时间只能有一个活跃竞拍”的约束。

**Architecture:** H5 只负责触发 `/api/test/demo/merchant/*` 与展示 toast；test-service 作为 demo 编排层，使用统一 seed 的商家账号 `9103` 通过 gateway 调用 auction-service/product-service 真实 API；auction-service 需要补齐受控 `start_time` 与 `live_stream_id` 创建字段，保证 `即将开播` 不是前端假状态，而是真实 Pending 竞拍。

**Tech Stack:** Go (Hertz, GORM, shopspring/decimal)、React 18、Jest、go test。

**重要决策:**
- `一口价` 在非直播间页面**不禁用**，点击后 toast：`请先进入直播间`；原因是 Demo Console 是全局浮层，禁用按钮不能解释失败原因。
- `即将开播` / `正在竞拍` 拆成两个二级按钮，不用一个“一键开拍”混合创建两类竞拍；原因是演示时需要明确、可重复、可解释。
- 每次点击都创建新的 demo 商品和必要关联数据，不复用固定商品，不取消旧场次，不修改历史事实。
- test-service 不直接查库/写库；跨服务访问只走 gateway/API。

**前置状态:**
- 当前分支已有未提交的 Demo Console bugfix：充值二级菜单、登录页切账号跳转首页。执行本计划前不要丢弃这些改动。
- Demo 接口当前已有 `/api/test/demo/follow-bid`、`/api/test/demo/recharge`，并已做 JWT + demo 用户白名单。

---

## 新增交互

Demo Console 一级菜单：

```text
账号
演示
充值
商家
关闭
```

`商家` 二级菜单：

```text
即将开播
正在竞拍
一口价
返回
```

成功 toast：
- `即将开播`：`已创建1分钟后开播的竞拍`
- `正在竞拍`：`已创建正在竞拍场次`
- `一口价`：`已为当前直播间创建一口价商品`

失败/前置条件 toast：
- 非直播间点击 `一口价`：`请先进入直播间`
- API 失败：`商家动作失败：<error>`

---

## API Contract

### `POST /api/test/demo/merchant/auctions`

Request:

```json
{
  "mode": "upcoming"
}
```

`mode` 可取：
- `upcoming`: 创建 1 分钟后开始的 Pending 竞拍
- `ongoing`: 创建可立即进入/出价的竞拍

Response:

```json
{
  "ok": true,
  "mode": "upcoming",
  "product_id": 123,
  "live_stream_id": 456,
  "auction_id": 789,
  "start_time": "2026-06-07T05:00:00+08:00",
  "end_time": "2026-06-07T05:03:00+08:00"
}
```

### `POST /api/test/demo/merchant/fixed-price-items`

Request:

```json
{
  "live_stream_id": 456
}
```

Response:

```json
{
  "ok": true,
  "product_id": 124,
  "live_stream_id": 456,
  "item_id": 888,
  "price": "99.00",
  "stock": 10
}
```

---

## Task 1: auction-service 支持受控 `start_time` 与 `live_stream_id`

**目标:** `POST /api/v1/auctions` 保持兼容现有调用，同时允许受控传入 `start_time`、`live_stream_id`，供 demo 编排创建真实“即将开播”和直播间归属竞拍。

**Files:**
- `backend/auction/handler/auction.go`
- `backend/auction/service/auction.go`
- `backend/auction/handler/auction_test.go` 或新增聚焦测试
- `backend/test/client/auction/client.go`
- `backend/test/client/auction/client_test.go`

**实现要点:**
- `handler.CreateAuctionRequest` 新增可选字段：
  - `StartTime *time.Time json:"start_time,omitempty"`
  - `LiveStreamID *int64 json:"live_stream_id,omitempty"`
- 未传 `start_time` 时维持现状：`time.Now()`。
- 传入 `start_time` 时，`end_time = start_time + duration`。
- `service.CreateAuctionRequest` 新增 `LiveStreamID *int64` 并写入 `model.Auction.LiveStreamID`。
- SDK `CreateAuctionReq` 同步新增 `StartTime string/json` 或 `*time.Time`、`LiveStreamID int64`。
- 不引入 float 金额业务逻辑；本任务只补时间/直播间归属。

**TDD:**
- [ ] 写 handler/service 单测：传入未来 `start_time` 后创建出的 auction `Status=Pending`、`StartTime` 接近请求值、`LiveStreamID` 被保存。
- [ ] 写 SDK 单测：`CreateAuctionAs` 请求 body 包含 `start_time` 与 `live_stream_id`。
- [ ] 跑失败测试：

```bash
cd backend/auction && go test ./handler ./service -run 'Test.*CreateAuction.*StartTime|Test.*CreateAuction.*LiveStream' -count=1 -v
cd backend/test && go test ./client/auction/ -run TestSDK_CreateAuction -count=1 -v
```

- [ ] 最小实现。
- [ ] 验证：

```bash
cd backend/auction && go test ./handler ./service -run 'Test.*CreateAuction.*' -count=1 -v
cd backend/test && go test ./client/auction/ -run TestSDK_CreateAuction -count=1 -v
```

**风险:** 如果已有 admin 创建竞拍链路假设 `start_time=now`，必须确保新字段 optional 且默认行为不变。

---

## Task 2: test-service 增加商家 demo 编排

**目标:** 新增 `/api/test/demo/merchant/auctions` 与 `/api/test/demo/merchant/fixed-price-items`，所有 demo 数据每次创建新商品。

**Files:**
- `backend/test/handler/demo.go`
- `backend/test/handler/demo_test.go`
- `backend/test/main.go`

**实现要点:**
- 扩展 `demoAuctionClient` interface：
  - `CreateProductAs`
  - `CreateAuctionRule`
  - `CreateLiveStream`
  - `CreateAuctionAs`
  - `CreateFixedPriceItem`
  - `WaitAuctionStarted`（仅 `ongoing`）
- 固定商家 actor：`auctioncli.Actor{UserID: 9103, Role: RoleMerchant}`。
- `createDemoProductName(kind string, now time.Time)` 生成唯一商品名，例如 `DEMO_商家动作_ongoing_1700000000000`。
- `upcoming` 编排：
  1. 创建新商品
  2. 创建直播间
  3. 创建规则
  4. 创建竞拍，`start_time = now + 1 minute`，`live_stream_id = live.ID`
- `ongoing` 编排：
  1. 创建新商品
  2. 创建直播间
  3. 创建规则
  4. 创建竞拍，`start_time = now`
  5. 等待 scheduler 将其置为 Ongoing；超时返回清晰错误
- `fixed-price` 编排：
  1. 校验 `live_stream_id > 0`
  2. 创建新商品
  3. 创建一口价 item，`price="99.00"`，`stock=10`
- 所有接口继续复用 `authorizeDemoRequest`，只允许合法 demo JWT 调用。

**TDD:**
- [ ] 写纯函数测试：
  - `validateMerchantAuctionMode("upcoming"/"ongoing")` 通过，其他拒绝
  - `validateDemoLiveStreamID(0)` 拒绝
  - demo 商品名连续两次不重复
- [ ] 写 handler 编排 fake client 测试：
  - `upcoming` 创建两个商品/竞拍时 product id 不相同
  - `fixed-price` 会把请求里的 `live_stream_id` 传给 `CreateFixedPriceItem`
  - 非法 mode 返回 400
- [ ] 跑失败测试：

```bash
cd backend/test && go test ./handler/ -run 'TestMerchantDemo' -count=1 -v
```

- [ ] 最小实现并注册路由：

```go
merchant := demo.Group("/merchant")
merchant.POST("/auctions", demoHandler.PostMerchantAuction)
merchant.POST("/fixed-price-items", demoHandler.PostMerchantFixedPriceItem)
```

- [ ] 验证：

```bash
cd backend/test && go test ./handler/ -run 'TestMerchantDemo|TestDemoUserIDFromAuthorization' -count=1 -v
cd backend/test && go build ./...
```

**风险:** `ongoing` 依赖 scheduler 周期。若本地 scheduler 不稳定，执行阶段可以将等待超时设置为 5s 并返回“创建成功但尚未开始”的明确错误；不要直接写库改状态。

---

## Task 3: 前端 demoApi 增加商家动作请求

**目标:** 在 H5 封装 `/api/test/demo/merchant/*`，不走 `services/api.ts` 的 `/api/v1` baseURL。

**Files:**
- `frontend/h5/src/services/demoApi.ts`
- `frontend/h5/src/services/__tests__/demoApi.test.ts`

**实现要点:**
- 新增：

```ts
export type DemoMerchantAuctionMode = 'upcoming' | 'ongoing';
export function createDemoMerchantAuction(mode: DemoMerchantAuctionMode) { ... }
export function createDemoFixedPriceItem(liveStreamId: number) { ... }
```

- 请求继续带本地 token：`Authorization: Bearer <auth_token>`。

**TDD:**
- [ ] 测试 `createDemoMerchantAuction('upcoming')` 请求路径和 body。
- [ ] 测试 `createDemoFixedPriceItem(456)` 请求 body 为 `{ live_stream_id: 456 }`。
- [ ] 验证：

```bash
cd frontend/h5 && npx jest src/services/__tests__/demoApi.test.ts --runInBand
```

---

## Task 4: DemoContext 增加当前直播间上下文

**目标:** 让全局 Demo Console 能知道是否处于直播间，以及当前 `liveStreamId`。

**Files:**
- `frontend/h5/src/store/demoContext.tsx`
- `frontend/h5/src/store/__tests__/demoContext.test.tsx`
- `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`
- `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx`

**实现要点:**
- `DemoContextType` 新增：

```ts
currentLiveStreamId: number | null;
setCurrentLiveStreamId: (id: number | null) => void;
```

- `LiveRoomSlide` 在 `active` 时写入 `liveStreamId`，卸载或 inactive 时清空。
- 保留现有 `currentAuctionId` 行为。

**TDD:**
- [ ] `demoContext.test.tsx` 覆盖 liveStreamId setter。
- [ ] `LiveRoomSlide.test.tsx` 覆盖 active 房间写入当前直播间。
- [ ] 验证：

```bash
cd frontend/h5 && npx jest src/store/__tests__/demoContext.test.tsx src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand
```

---

## Task 5: DemoConsole 增加 `商家` 菜单与三项动作

**目标:** UI 层接入新接口，保持全局浮层可解释、可重复点击。

**Files:**
- `frontend/h5/src/components/DemoConsole/index.tsx`
- `frontend/h5/src/components/DemoConsole/__tests__/DemoConsole.test.tsx`

**实现要点:**
- `MenuView` 新增 `merchant`。
- root 菜单新增 `商家`。
- `merchant` 二级菜单新增：
  - `即将开播` -> `createDemoMerchantAuction('upcoming')`
  - `正在竞拍` -> `createDemoMerchantAuction('ongoing')`
  - `一口价` -> 若 `currentLiveStreamId == null`，只 toast `请先进入直播间`；否则 `createDemoFixedPriceItem(currentLiveStreamId)`
  - `返回`
- 每个动作独立 `runningAction` key，避免重复点击并发触发同一个按钮。
- 成功后不强制跳转，先只 toast；如果后续要“创建后直接进入直播间”，另开小任务。

**TDD:**
- [ ] 测试 root 菜单有 `商家`。
- [ ] 测试点击 `商家` 展示三项二级按钮。
- [ ] 测试 `即将开播` 调 `createDemoMerchantAuction('upcoming')`。
- [ ] 测试 `正在竞拍` 调 `createDemoMerchantAuction('ongoing')`。
- [ ] 测试非直播间点击 `一口价`：toast `请先进入直播间`，不调用 API。
- [ ] 测试直播间点击 `一口价`：调 `createDemoFixedPriceItem(currentLiveStreamId)`。
- [ ] 验证：

```bash
cd frontend/h5 && npx jest src/components/DemoConsole/__tests__/DemoConsole.test.tsx --runInBand
```

---

## Task 6: 本地联调与回归

**目标:** 在当前本地服务栈验证真实链路。

**Commands:**

```bash
cd backend/auction && go test ./handler ./service -run 'Test.*CreateAuction.*' -count=1 -v
cd backend/test && go test ./handler/ -run 'TestMerchantDemo|TestValidateRechargeRequest|TestDemoUserIDFromAuthorization' -count=1 -v
cd backend/test && go test ./client/auction/ -run 'TestSDK_CreateAuction|TestSDK_CreateFixedPriceItem' -count=1 -v
cd backend/test && go build ./...
cd frontend/h5 && npx jest src/components/DemoConsole/__tests__/DemoConsole.test.tsx src/services/__tests__/demoApi.test.ts src/store/__tests__/demoContext.test.tsx src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand
cd frontend/h5 && npm run build
git diff --check
```

**Manual smoke:**
- 登录买家A。
- `商家 -> 即将开播`：返回成功 toast；后台确认 auction `status=Pending` 且 `start_time≈now+1m`。
- `商家 -> 正在竞拍`：返回成功 toast；H5 可进入对应直播间/竞拍，或 API 查询为 Ongoing。
- 非直播间 `商家 -> 一口价`：只 toast `请先进入直播间`。
- 直播间 `商家 -> 一口价`：返回成功 toast，当前直播间一口价列表出现新商品或收到 WS 列表更新。
- 连续点击 `即将开播` / `正在竞拍`：每次创建不同 `product_id`，不触发活跃竞拍唯一性冲突。

---

## Out Of Scope

- 不做 R4 防作弊演示。
- 不直接写数据库清理旧竞拍。
- 不做创建后自动跳转直播间。
- 不做商家动作的环境隔离开关；安全边界沿用当前 demo endpoint 的 JWT + demo 用户白名单。

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-07-h5-demo-console-merchant-actions.md`.

Two execution options:

1. **Subagent-Driven（推荐）**：按 Task 1-6 拆分执行；Task 1 后端契约改动是后续任务依赖，必须先完成。
2. **Inline Execution**：当前会话直接逐 Task 实现。


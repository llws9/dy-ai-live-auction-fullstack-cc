# H5 Demo Console「并发压测出价」设计

- 日期: 2026-06-10
- 范围: `frontend/h5` + `backend/test`
- 驱动: 在 H5 直播间一键制造真实竞价竞争，让当前用户随后用旧价格出价时稳定失败

## 1. 目标

将 H5 浮动演示菜单中的「并发压测」从 placeholder 接入后端。点击后，由 `test-service` 代表演示买家发起一组真实出价请求，快速抬高当前竞拍价格。H5 端通过现有 WebSocket 出价事件看到真实出价动画、排行刷新和战况热度变化；当前用户再点击「立即出价」时，因为金额已经低于最新价格而失败。

## 2. 非目标

- 不做泛化长跑压测，不复用测试大屏 `pressure` 场景的 WS 进度流。
- 不在 H5 本地伪造出价动画，所有可见出价都必须来自真实后端出价链路。
- 不侵入 `auction-service` 业务主路径，不新增 if-demo/if-chaos 逻辑。
- 不绕过 `gateway-service`，前端仍只调用 `/api/test/demo/*`，真实出价由 `test-service` 经 gateway 的 `/api/v1` 入口完成。

## 3. 用户体验

入口保持在 `DemoConsole` 的「演示」二级菜单。

流程:

1. 用户进入正在竞拍的直播间。
2. 点击「演示」->「并发压测」。
3. 按钮进入 pending 状态，防止重复触发。
4. 后端在短时间内提交多笔递增出价。
5. 页面通过现有 `bid_placed` 事件展示真实出价飘屏、排行变化和热度变化。
6. toast 提示最高价已被抬高，例如「并发出价已抬到 ¥160，请尝试用旧价出价」。
7. 用户点击「立即出价」，若输入仍是旧价格，后端返回业务失败，前端显示现有失败提示。

## 4. 后端 API

新增 endpoint:

```text
POST /api/test/demo/concurrent-bids
```

请求体:

```json
{
  "auction_id": 123,
  "bid_count": 6,
  "interval_ms": 80,
  "increment": "10"
}
```

字段规则:

- `auction_id`: 必填，必须大于 0。
- `bid_count`: 可选，默认 6，范围 1-20。
- `interval_ms`: 可选，默认 80，范围 0-1000。
- `increment`: 可选，默认使用竞拍规则 `increment`；若规则缺失则使用 `1`。

响应体:

```json
{
  "ok": true,
  "auction_id": 123,
  "success_count": 5,
  "failure_count": 1,
  "highest_amount": "160",
  "last_error": "amount must be greater than current price"
}
```

成功判定:

- 至少 1 笔真实出价成功即返回 HTTP 200。
- 所有出价失败时返回 HTTP 400，并携带最后一个失败原因。

## 5. 并发策略

后端逻辑放在 `backend/test/handler/demo.go` 的 `DemoHandler`，复用现有 demo 鉴权和 `demoAuctionClient.PlaceBid`。

执行步骤:

1. 校验 demo 用户 JWT，仅允许固定 demo 用户触发。
2. 读取当前拍卖，得到 `current_price`、`start_price`、`rules.increment`。
3. 计算基准价 `baseline = max(current_price, start_price)`。
4. 启动最多 `bid_count` 个 goroutine，金额为 `baseline + increment * (i + 1)`。
5. 默认用 `buyerBUserID=9102` 发起出价；金额递增保证同一用户连续出价也能形成价格抬升。
6. 若配置了 `interval_ms > 0`，每个 goroutine 按 index 错开启动，避免完全同一时刻导致本地资源尖刺。
7. 汇总成功数、失败数、最高成功金额和最后错误。

选择递增金额而不是同金额盲压的原因:

- 目标是让当前用户稳定因「价格被超越」失败，而不是随机制造锁冲突。
- 递增金额会触发真实 `bid_placed` 广播，H5 可见效果更稳定。
- 仍保留短间隔并发形态，足以展示竞价竞争，而不把 demo 入口变成不可控压测器。

## 6. 前端集成

新增 `frontend/h5/src/services/demoApi.ts` 方法:

```ts
triggerConcurrentBids({ auctionId, bidCount, intervalMs, increment })
```

`DemoConsole` 改动:

- 将「并发压测」按钮从 `showPromptOnlyAction('并发压测暂未接入后端链路')` 替换为真实 handler。
- 无 `currentAuctionId` 时提示「请先进入直播间」。
- pending key 使用 `concurrent-bids`，按钮 disabled 防重复。
- 调用失败时显示「并发压测失败：{message}」。
- 成功时显示最高价提示，若响应没有 `highest_amount`，退化为「已触发并发出价」。

不需要改 `LiveRoomSlide`:

- 真实出价已走 `/api/v1/auctions/:id/bids`。
- 现有 `ws.on('bid_placed', ...)` 已负责飘屏、排行和热度。
- 当前用户出价失败继续复用 `handleBid` 的现有错误处理。

## 7. 数据流

```text
DemoConsole button
  -> POST /api/test/demo/concurrent-bids
  -> test-service DemoHandler
  -> gateway-service /api/v1/auctions/:id/bids
  -> auction-service PlaceBid
  -> WebSocket bid_placed
  -> H5 LiveRoomSlide 飘屏 / 排行 / 热度
  -> 用户点击立即出价
  -> auction-service 因金额低于最新价拒绝
```

## 8. 安全与边界

- demo endpoint 继续要求 Bearer JWT，且只允许固定 demo 用户或 `DEMO_ALLOWED_USER_IDS`。
- `bid_count` 上限为 20，防止 H5 按钮变成高风险压测入口。
- 金额计算在 `test-service` 内保持 `decimal.Decimal`，只在调用现有 SDK 边界时转换为 `float64`。
- 不直接查业务库，不跨服务查库；竞拍状态只通过现有 API/RPC 客户端读取。
- 不保证每笔并发都成功；失败会被计入响应，但只要有成功出价即可达成演示目标。

## 9. 测试计划

- `backend/test/handler/demo_test.go`
  - 校验无 `auction_id` 返回 400。
  - 校验 `bid_count` 超限返回 400。
  - 校验读取当前价格后发起多笔递增 `PlaceBid`。
  - 校验部分失败时仍返回 200 且包含 `failure_count`。
  - 校验全部失败时返回 400。
- `frontend/h5/src/services/__tests__/demoApi.test.ts`
  - 校验请求路径为 `/api/test/demo/concurrent-bids`。
  - 校验 body 使用 snake_case 字段。
- `frontend/h5/src/components/DemoConsole/__tests__/DemoConsole.test.tsx`
  - 校验点击按钮调用 `triggerConcurrentBids({ auctionId })`。
  - 校验无当前竞拍时不调用 API。
  - 校验成功和失败 toast。

## 10. 验收标准

- 点击「并发压测」不再提示未接入后端。
- 在直播间点击后，H5 能看到真实出价动画或排行/热度变化。
- 再使用旧金额点击「立即出价」会因价格已被抬高而失败。
- 该能力仅存在于 `test-service` demo 控制面，不污染业务服务主路径。

# Test Dashboard 用户验收剧本设计

> 日期：2026-06-05
> 范围：`frontend/test-dashboard`、`backend/test`
> 状态：待用户评审

## 1. 背景

`frontend/test-dashboard` 最初围绕竞拍展示、压力测试、防狙击、回调投递和故障注入建设。经过后续迭代，主业务新增了提醒、点天灯、一口价、订单详情、余额、直播间买家入口等用户链路能力。现有测试平台仍偏单点场景，不能直观证明“一个真实买家从进入直播间到完成交易结果验证”的闭环是否成立。

本设计聚焦补齐测试系统的验收演示能力，并为后续压测稳定性扩展留下接口。

## 2. 目标

* 新增一个面向买家视角的用户验收剧本，作为 `test-dashboard` 的第一入口之一。

* 场景启动时自动造数，避免依赖固定 seed 或人工准备数据。

* 通过 Gateway `/api/v1` 调用业务接口，不直连后端子服务。

* 以步骤时间线和证据报告展示每个关键链路是否成立。

* 保留运行数据和资源 ID，便于从 Admin、H5、DB 或日志复查。

* 在 P1 支持基于同一剧本的循环和并发稳定性验证。

## 3. 非目标

* P0 不覆盖管理员/商家权限矩阵。

* P0 不覆盖弹幕、飘屏、直播互动消息。

* P0 不重构现有 `pressure`、`e2e`、`antisnipe`、`callback`、`chaos` 页面。

* P0 不做通用可视化编排引擎。

* P0 不追求极限 QPS 压测；极限压测继续由现有 `Pressure` 场景承担。

## 4. 方案选择

采用方案 A：新增 `user_journey` 场景作为主线验收入口，复用现有测试平台的任务、WebSocket、历史记录和报告能力。

未选择的方案：

* 快速补洞：只在现有页面增加按钮。实现快，但会继续形成碎片化测试入口，无法证明业务闭环。

* 全量重构：先做通用场景编排器。长期更完整，但当前目标是补齐验收演示，重构会拖慢交付。

## 5. 架构设计

### 5.1 后端

在 `backend/test` 新增场景：

```text
backend/test/scenario/user_journey
```

职责：

* 生成 `test_run_id`。

* 自动准备测试买家、商家、商品、直播间、竞拍、一口价商品和余额。

* 通过 Gateway `/api/v1` 调用业务接口。

* 按步骤发出进度事件。

* 收集每一步输入、HTTP 状态、关键响应字段和断言结果。

* 将最终报告写入现有 `test_results.ResultJSON`。

#### 5.1.1 多角色造数（关键约束）

纯买家视角剧本在造数阶段无法只用买家身份完成，必须显式分配商家与买家两类角色，全部走 Gateway `/api/v1`。平台管理员不参与开播；管理员只负责平台治理动作，例如中断直播间、封禁直播间或审计。

角色模型（来自 `gateway/middleware/rbac.go`）：`0=用户`、`1=商家/主播`、`2=管理员`；`RequireMerchantOnly` 是 ExactRole(1)，会主动拒绝管理员。

| 造数动作            | 接口                                   | 业务期望角色 | 当前网关约束（代码现状）  |
| --------------- | ------------------------------------ | ------ | ----------------- |
| 创建商品            | `POST /api/v1/products`              | merchant | 无角色约束（公开 `v1` 路由，仅需登录身份） |
| 创建直播间           | `POST /api/v1/admin/live-streams`    | merchant | `RequireMerchantOnly` |
| 上架一口价商品         | `POST /api/v1/fixed-price/items`     | merchant | `RequireMerchantOnly` |
| 创建竞拍            | `POST /api/v1/auctions`              | merchant | `RequireMerchantOnly` |
| 开播              | `POST /api/v1/live-streams/:id/start`| merchant | **`RequireAdmin`（与业务设定冲突，见下）** |
| 买家全部交易动作        | 竞拍/一口价/点天灯/订单                         | user     | 仅需登录身份 |

因此现有 `backend/test/client/auction` 写死的 `X-User-Role: "user"` 必须扩展为按调用传入角色（`user` / `merchant`），并允许传入对应的测试身份 ID。该多角色能力仅用于 `prepare` 阶段，主链路（`enter_live` 之后）仍只用买家身份，因此 P0 不验证权限矩阵的结论依然成立。

#### 5.1.1.1 开播鉴权偏差（P0 阻塞前置项）

代码现状确认：`POST /api/v1/live-streams/:id/start` 当前为 `RequireAdmin()`（`role>=2`），并经 gateway 的 `LiveStartHandler` 转发到 `/internal/live-streams/:id/start`。这与业务设定（开播是商家权利，平台管理员只能中断/治理）直接冲突。

影响：在不修正鉴权的前提下，测试剧本要让"商家开播"会被 gateway 以 403 拒绝；若改用管理员身份开播，又会固化"管理员代运营"的错误语义。因此这是 P0 的阻塞前置项，必须先决策：

* 推荐：先修正开播鉴权为商家可开播（建议 `RequireMerchantOnly` 或主播级 `RequireStreamer`），并在 auction-service 内补 owner 归属校验（商家只能开自己拥有的直播间）；管理员仅保留 `end` / `ban` 等治理动作。

* 该修正属于业务后端改动，不应由测试剧本绕过；实施计划须将其列为独立前置任务并补单测。

#### 5.1.2 买家余额准备

一口价抢购与点天灯依赖买家余额，但 Gateway 仅暴露 `GET /api/v1/user/balance`，无写余额接口。决策：在 `auction-service` 现有 `/internal/*` 路由组（已由 `InternalAuthMiddleware` / `X-Internal-Token` 保护）下新增受控测试充值接口，例如：

```text
POST /internal/test/user-balance   （body: user_id, amount）
```

约束：

* 仅注册在 `/internal` 组，必须校验 `X-Internal-Token`，不经 Gateway 暴露给前端。

* `backend/test` 通过内部 token 调用，仅用于测试环境准备余额。

* 金额使用 `shopspring/decimal`，2 位精度。

* 若该接口不可用，`prepare` 必须 fail-closed 直接失败，不得假设环境已有余额。

新增启动接口建议复用现有测试接口风格：

```text
POST /test/user-journey
```

请求配置：

```ts
interface UserJourneyConfig {
  include_reminder?: boolean
  include_sky_lamp?: boolean
  include_fixed_price?: boolean
  auction_duration_sec?: number
  buyer_count?: number
  keep_evidence?: boolean
}
```

默认值：

* `include_reminder=true`

* `include_sky_lamp=true`

* `include_fixed_price=true`

* `auction_duration_sec=30`

* `buyer_count=1`

* `keep_evidence=true`

### 5.2 前端

在 `frontend/test-dashboard` 新增页面：

```text
frontend/test-dashboard/src/pages/UserJourney.tsx
```

新增路由：

```text
/test/user-journey
```

页面结构：

* 启动区：一键启动、取消、`test_id`、Gateway 地址、运行状态。

* 配置区：展示必要开关，自动造数细节默认隐藏。

* 步骤时间线：展示每一步状态、耗时、失败原因。

* 证据卡片：展示资源 ID、订单 ID、余额变化、库存变化、关键响应字段。

* 稳定性入口：P1 使用，支持循环和并发运行。

* 报告入口：跳转现有 `/test/report/:id`。

## 6. P0 用户验收剧本

P0 剧本只覆盖买家交易主链路。

| 步骤                     | 目的       | 关键接口 | 核心断言                    |
| ---------------------- | -------- | ---- | ----------------------- |
| `prepare`              | 自动准备测试数据 | 多角色造数（见 5.1.1）+ 内部充值（见 5.1.2）+ 商家开播 | 商品、直播间(ongoing)、竞拍、一口价商品、买家余额均可用 |
| `enter_live`           | 买家进入直播间  | `GET /api/v1/live-streams/:id`、`GET /api/v1/live-streams/:id/fixed-price/items` | 直播详情可读且状态为开播，竞拍与一口价列表非空 |
| `reminder`             | 验证提醒链路   | `POST /api/v1/live-streams/:id/follow` + `GET /api/v1/live-streams/:id/follow-status` | 关注成功且 follow-status 反映已关注 |
| `auction_bid`          | 验证普通竞拍   | `POST /api/v1/auctions/:id/bids` + `GET /api/v1/auctions/:id` | 出价成功，当前价/领先者更新正确 |
| `sky_lamp`             | 验证点天灯链路  | `POST /api/v1/sky-lamp/subscriptions` + `GET /api/v1/sky-lamp/subscriptions/:id` | 订阅成功，余额与状态变化可验证 |
| `fixed_price_purchase` | 验证一口价抢购  | `POST /api/v1/fixed-price/items/:id/purchase`（带 `X-Idempotency-Key`）+ `GET .../my-purchase` | 幂等键生效，库存扣减，订单生成，重复购买被拦截 |
| `verify`               | 汇总验收结果   | `GET /api/v1/orders`、`GET /api/v1/user/balance` | 订单、余额、库存、竞拍状态、关键响应字段一致 |

> 备注：`reminder` 步骤 P0 只验证"关注/取消关注 + follow-status"这一条链路；`PUT /notification` 与 `/live/pending-reminder` 的提醒推送验证放入 P1，避免 P0 步骤语义发散。

### 6.1 失败策略

* 关键断言失败即标记场景失败。

* 非关键证据采集失败可标记为 `warning`，但不得掩盖核心断言。

* 不允许静默跳过已启用的步骤。

* 如果自动造数失败，场景直接失败，不回退到固定 seed。

### 6.2 证据保留

每次运行保留业务数据，追踪策略分两层：

* **可读标记**：仅对支持自定义文本的字段写入 `TEST_USER_JOURNEY_<test_run_id>`，明确落点为 product 的 `name`/`description` 与直播间 `title`。订单、余额、竞拍等无自定义备注字段的实体不强写标记。

* **关联追踪**：所有创建的实体（含订单、竞拍、一口价、直播间）通过现有 `test_seed_data` 表按 `test_run_id` 关联记录 `kind` + `ref_id`，作为复查与回收的权威依据。

与现有 E2E 编排器不同，`user_journey` 复用 `SeedRecorder.Add` 记录引用，但**默认跳过 cleanup**（不调用 `DeleteByTestID`，也不删业务表），以保留验收证据。清理由 7.x 之外的独立回收任务按 `test_run_id` 或创建时间执行。

报告必须记录（金额字段一律用 decimal 字符串，禁止 float）：

* `test_run_id`

* 创建的资源 ID（product / live_stream / auction / fixed_price_item / order）

* 买家/商家测试身份 ID

* 订单 ID

* 竞拍 ID

* 一口价商品 ID

* 运行前后余额（decimal 字符串）

* 运行前后库存

* 每一步 HTTP 状态和关键响应字段

* 失败原因或 warning

## 7. P1 稳定性扩展

P1 不改变 P0 剧本语义，而是在同一造数和断言能力上扩展运行模式。

建议新增配置：

```ts
interface UserJourneyLoadConfig extends UserJourneyConfig {
  mode: 'single' | 'load'
  loops?: number
  concurrent_buyers?: number
  duration_sec?: number
}
```

P1 指标：

* 成功率

* P95/P99 延迟

* 订单一致性

* 库存不超卖

* 重复购买拦截率

* WebSocket 进度丢失率

* 错误码分布

P1 优先覆盖：

* 一口价抢购并发正确性。

* 普通竞拍出价并发正确性。

点天灯压测暂不作为 P1 首批目标。

## 8. 与现有系统关系

* `Dashboard`：保留 dummy 联调入口。

* `E2E`：保留原竞拍全链路专项页。

* `Pressure`：保留极限压力测试专项页。

* `AntiSnipe`：保留防狙击专项页。

* `History` / `Report`：复用并增强，针对 `user_journey` 做结构化渲染。

* `Screen`：后续可展示 `user_journey` 的运行状态，但不纳入 P0。

## 9. 验收标准

P0 完成标准：

* `frontend/test-dashboard` 能启动 `user_journey` 场景。

* 页面能实时展示步骤进度、状态和失败原因。

* 后端能自动造数并通过 Gateway 完成买家链路。

* 报告能展示资源 ID、订单、余额、库存和关键响应字段。

* 单次默认运行不依赖固定 seed。

* 运行失败时能定位到具体步骤和断言。

* 不影响现有 `dummy`、`pressure`、`e2e`、`antisnipe`、`callback`、`chaos` 场景。

P1 完成标准：

* 支持基于用户验收剧本的循环或并发运行。

* 输出成功率、延迟、库存一致性、订单一致性和错误码分布。

* 复用 P0 证据模型，不另起一套报告格式。

## 10. 风险与约束

* **造数依赖多角色**：纯买家剧本的 `prepare` 阶段必须使用 merchant 创建商品、直播间、一口价、竞拍并开启自己的直播间，使用 user 执行买家动作（见 5.1.1）。test client 需支持多 `X-User-Role`，否则造数无法完成。这是 P0 的首要实现前提。

* **余额准备需新增内部接口**：Gateway 无写余额接口，须在 auction-service `/internal` 组新增受 `X-Internal-Token` 保护的测试充值接口（见 5.1.2）。该接口属新增后端改动，应在实施计划中独立成任务并补单测。

* 场景主链路必须走 Gateway `/api/v1`，不能为了方便直连子服务；仅余额准备这一受控例外走 `/internal`。

* 金额字段校验必须按业务精度（decimal，2 位）处理，报告中金额一律用字符串，避免 float 误差影响验收结论。

* 保留证据会产生测试数据积累，需要后续通过独立回收任务按 `test_run_id` 或创建时间回收；`user_journey` 默认不 cleanup。

* **开播鉴权与业务设定冲突（P0 阻塞）**：代码现状 `POST /api/v1/live-streams/:id/start` 为 `RequireAdmin`，与"开播是商家权利"冲突（见 5.1.1.1）。必须先修正鉴权为商家可开播 + owner 归属校验，否则商家开播会被 403 拦截，测试剧本无法成立。

* `enter_live` 依赖直播间为开播状态，因此 `prepare` 必须串入商家开播步骤；该步骤的可行性取决于上面的开播鉴权修正完成。

* P0 不包含权限矩阵，管理员/商家权限应另起独立验收剧本。

## 11. 后续拆分建议

实施计划建议拆成以下任务：

* T0（前置）：修正开播鉴权为商家可开播 + owner 归属校验，管理员仅保留 end/ban 治理动作 + 单测。

* T1：后端 `user_journey` 数据模型、报告结构和单元测试。

* T2：test client 多角色（user/merchant）支持 + Gateway 造数能力。

* T3：auction-service `/internal/test/user-balance` 受控充值接口 + 单测。

* T4：买家主链路步骤实现（含商家开播串接）。

* T5：前端 API、路由、导航和页面骨架。

* T6：步骤时间线与证据卡片。

* T7：报告结构化渲染。

* T8：回归验证现有测试场景不受影响。

* T9：P1 稳定性扩展。

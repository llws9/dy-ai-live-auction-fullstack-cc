# Test Dashboard 用户验收剧本设计

> 日期：2026-06-05
> 范围：`frontend/test-dashboard`、`backend/test`
> 状态：待用户评审

## 1. 背景

`frontend/test-dashboard` 最初围绕竞拍展示、压力测试、防狙击、回调投递和故障注入建设。经过后续迭代，主业务新增了提醒、点天灯、一口价、订单详情、余额、直播间买家入口等用户链路能力。现有测试平台仍偏单点场景，不能直观证明“一个真实买家从进入直播间到完成交易结果验证”的闭环是否成立。

本设计聚焦补齐测试系统的验收演示能力，并为后续压测稳定性扩展留下接口。

## 2. 目标

- 新增一个面向买家视角的用户验收剧本，作为 `test-dashboard` 的第一入口之一。
- 场景启动时自动造数，避免依赖固定 seed 或人工准备数据。
- 通过 Gateway `/api/v1` 调用业务接口，不直连后端子服务。
- 以步骤时间线和证据报告展示每个关键链路是否成立。
- 保留运行数据和资源 ID，便于从 Admin、H5、DB 或日志复查。
- 在 P1 支持基于同一剧本的循环和并发稳定性验证。

## 3. 非目标

- P0 不覆盖管理员/商家权限矩阵。
- P0 不覆盖弹幕、飘屏、直播互动消息。
- P0 不重构现有 `pressure`、`e2e`、`antisnipe`、`callback`、`chaos` 页面。
- P0 不做通用可视化编排引擎。
- P0 不追求极限 QPS 压测；极限压测继续由现有 `Pressure` 场景承担。

## 4. 方案选择

采用方案 A：新增 `user_journey` 场景作为主线验收入口，复用现有测试平台的任务、WebSocket、历史记录和报告能力。

未选择的方案：

- 快速补洞：只在现有页面增加按钮。实现快，但会继续形成碎片化测试入口，无法证明业务闭环。
- 全量重构：先做通用场景编排器。长期更完整，但当前目标是补齐验收演示，重构会拖慢交付。

## 5. 架构设计

### 5.1 后端

在 `backend/test` 新增场景：

```text
backend/test/scenario/user_journey
```

职责：

- 生成 `test_run_id`。
- 自动准备测试买家、商家、商品、直播间、竞拍、一口价商品和余额。
- 通过 Gateway `/api/v1` 调用业务接口。
- 按步骤发出进度事件。
- 收集每一步输入、HTTP 状态、关键响应字段和断言结果。
- 将最终报告写入现有 `test_results.ResultJSON`。

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

- `include_reminder=true`
- `include_sky_lamp=true`
- `include_fixed_price=true`
- `auction_duration_sec=30`
- `buyer_count=1`
- `keep_evidence=true`

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

- 启动区：一键启动、取消、`test_id`、Gateway 地址、运行状态。
- 配置区：展示必要开关，自动造数细节默认隐藏。
- 步骤时间线：展示每一步状态、耗时、失败原因。
- 证据卡片：展示资源 ID、订单 ID、余额变化、库存变化、关键响应字段。
- 稳定性入口：P1 使用，支持循环和并发运行。
- 报告入口：跳转现有 `/test/report/:id`。

## 6. P0 用户验收剧本

P0 剧本只覆盖买家交易主链路。

| 步骤 | 目的 | 核心断言 |
|---|---|---|
| `prepare` | 自动准备测试数据 | 商品、直播间、竞拍、一口价商品、买家余额均可用 |
| `enter_live` | 买家进入直播间 | 可读取直播详情、竞拍信息和一口价列表 |
| `reminder` | 验证提醒链路 | 关注/预约或待提醒状态读取成功 |
| `auction_bid` | 验证普通竞拍 | 出价成功，领先状态和竞拍响应字段正确 |
| `sky_lamp` | 验证点天灯链路 | 订阅或启动成功，余额与状态变化可验证 |
| `fixed_price_purchase` | 验证一口价抢购 | 幂等键生效，库存扣减，订单生成，重复购买被拦截 |
| `verify` | 汇总验收结果 | 订单、余额、库存、竞拍状态、关键响应字段一致 |

### 6.1 失败策略

- 关键断言失败即标记场景失败。
- 非关键证据采集失败可标记为 `warning`，但不得掩盖核心断言。
- 不允许静默跳过已启用的步骤。
- 如果自动造数失败，场景直接失败，不回退到固定 seed。

### 6.2 证据保留

每次运行保留业务数据，并尽量在可控字段中写入 `test_run_id` 标记，例如：

```text
TEST_USER_JOURNEY_<test_run_id>
```

报告必须记录：

- `test_run_id`
- 创建的资源 ID
- 买家/商家用户 ID
- 订单 ID
- 竞拍 ID
- 一口价商品 ID
- 运行前后余额
- 运行前后库存
- 每一步 HTTP 状态和关键响应字段
- 失败原因或 warning

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

- 成功率
- P95/P99 延迟
- 订单一致性
- 库存不超卖
- 重复购买拦截率
- WebSocket 进度丢失率
- 错误码分布

P1 优先覆盖：

- 一口价抢购并发正确性。
- 普通竞拍出价并发正确性。

点天灯压测暂不作为 P1 首批目标。

## 8. 与现有系统关系

- `Dashboard`：保留 dummy 联调入口。
- `E2E`：保留原竞拍全链路专项页。
- `Pressure`：保留极限压力测试专项页。
- `AntiSnipe`：保留防狙击专项页。
- `History` / `Report`：复用并增强，针对 `user_journey` 做结构化渲染。
- `Screen`：后续可展示 `user_journey` 的运行状态，但不纳入 P0。

## 9. 验收标准

P0 完成标准：

- `frontend/test-dashboard` 能启动 `user_journey` 场景。
- 页面能实时展示步骤进度、状态和失败原因。
- 后端能自动造数并通过 Gateway 完成买家链路。
- 报告能展示资源 ID、订单、余额、库存和关键响应字段。
- 单次默认运行不依赖固定 seed。
- 运行失败时能定位到具体步骤和断言。
- 不影响现有 `dummy`、`pressure`、`e2e`、`antisnipe`、`callback`、`chaos` 场景。

P1 完成标准：

- 支持基于用户验收剧本的循环或并发运行。
- 输出成功率、延迟、库存一致性、订单一致性和错误码分布。
- 复用 P0 证据模型，不另起一套报告格式。

## 10. 风险与约束

- 自动造数需要稳定的测试身份和余额准备能力；如果现有业务接口不支持完整造数，需在 `backend/test` 内通过受控测试辅助能力补齐。
- 场景必须走 Gateway `/api/v1`，不能为了方便直连子服务。
- 金额字段校验必须按业务精度处理，避免 float 误差影响验收结论。
- 保留证据会产生测试数据积累，需要后续通过现有清理任务按 `test_run_id` 或创建时间回收。
- P0 不包含权限矩阵，管理员/商家权限应另起独立验收剧本。

## 11. 后续拆分建议

实施计划建议拆成以下任务：

- T1：后端 `user_journey` 数据模型、报告结构和单元测试。
- T2：自动造数与 Gateway client。
- T3：买家主链路步骤实现。
- T4：前端 API、路由、导航和页面骨架。
- T5：步骤时间线与证据卡片。
- T6：报告结构化渲染。
- T7：回归验证现有测试场景不受影响。
- T8：P1 稳定性扩展。

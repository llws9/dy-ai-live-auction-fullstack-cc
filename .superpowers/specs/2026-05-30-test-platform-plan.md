# 竞拍系统展示测试平台 - 实施计划 (Plan)

> 配套规格：[2026-05-30-test-platform-spec.md](./2026-05-30-test-platform-spec.md)
> 决策记录：MVP 锁定 P0 五场景（A/E/F/H/G）；混沌支持 Redis 闪断 + MQ 暂停 + 网络丢包；Mock 外部平台采用内置形态
> 生成时间：2026-05-30

***

## 0. 第一性原理与边界

**本平台不是什么（不要做的）**：

* 不是替代正式 CI/CD 测试体系

* 不追求覆盖所有边界场景，只追求**演示说服力**

* 不在生产环境运行，只在测试/演示环境

**本平台是什么（必须做的）**：

* 一键演示"平台核心差异化能力"的工具

* 故事线驱动：性能 → 业务 → 公平 → 严谨 → 抗故障

* 可视化指标 + 实时进度 + A/B 对照

***

## 1. 总体架构

### 1.1 服务边界

```
┌─────────────────────── frontend/test-dashboard (Vite + React) ───────────────────────┐
│  /test            场景选择 + 实时指标                                                  │
│  /test/screen     大屏模式（演示主屏）                                                 │
│  /test/history    历史记录                                                            │
│  /test/report/:id 测试报告                                                            │
└──────────────────────────────────┬───────────────────────────────────────────────────┘
                                   │ HTTP / WS（必经 gateway-service:18080）
┌──────────────────────────────────▼───────────────────────────────────────────────────┐
│                         backend/test  (test-service:18090)                           │
│  ┌─────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │handler  │  │ runner   │  │ scenario │  │ mock     │  │ chaos    │  │ progress │  │
│  │(API/WS) │  │(执行引擎)│  │(A/E/F/H/G│  │(Partner) │  │(injector)│  │ broker   │  │
│  └─────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │
│                            │              │              │                           │
│                            │              │              │                           │
└────────────────────────────┼──────────────┼──────────────┼───────────────────────────┘
                             │              │              │
              ┌──────────────▼──┐   ┌───────▼────┐  ┌──────▼─────────┐
              │被测：auction-svc │   │ Mock 接收  │  │ toxiproxy /    │
              │     product-svc  │   │ /redis ctl │  │ MQ admin       │
              └──────────────────┘   └────────────┘  └────────────────┘
                             │
                       ┌─────▼─────┐
                       │  MySQL    │  test_results 表
                       └───────────┘
```

### 1.2 核心约束（来自项目记忆）

* 前端流量统一走 `gateway-service:18080`（HTTP/WS）

* test-service 作为新服务接入，路径前缀 `/api/test/*` 与 `/ws/test/*`

* CSS Modules + 项目级 CSS 变量

* 后端遵循 DDD 分层：handler → service → dao

* 用 Outbox 模式发送测试事件（与回调测试天然契合）

***

## 2. 模块拆分

### 2.1 后端模块（backend/test）

| 模块                 | 路径                                | 职责                           |
| ------------------ | --------------------------------- | ---------------------------- |
| 入口                 | `main.go`                         | 启动 Hertz、装配依赖、注册路由           |
| 配置                 | `config/`                         | Nacos + 环境变量 fallback        |
| handler            | `handler/test.go`、`handler/ws.go` | API 路由 + 进度 WS               |
| runner             | `runner/runner.go`                | 任务调度、生命周期管理、replay\_token 生成 |
| scenario/pressure  | `scenario/pressure/`              | 场景 A：goroutine pool 压测出价     |
| scenario/e2e       | `scenario/e2e/`                   | 场景 E：编排完整业务链路                |
| scenario/antisnipe | `scenario/antisnipe/`             | 场景 F：末刻出价 + delay 检查         |
| scenario/callback  | `scenario/callback/`              | 场景 H：触发回调，记录状态机轨迹            |
| scenario/chaos     | `scenario/chaos/`                 | 场景 G：调用 chaos 注入器 + 观测       |
| chaos/redis        | `chaos/redis/`                    | Redis 闪断（toxiproxy）          |
| chaos/mq           | `chaos/mq/`                       | MQ 消费者暂停/恢复                  |
| chaos/network      | `chaos/network/`                  | 网络丢包/延迟（toxiproxy）           |
| mock/partner       | `mock/partner/`                   | 内置外部平台 Mock（含可配置故障）          |
| dao                | `dao/result.go`                   | test\_results 表读写            |
| model              | `model/test.go`                   | 配置/结果数据结构                    |
| ws                 | `ws/progress.go`                  | 进度广播                         |
| metrics            | `pkg/metrics/`                    | 复用 backend/pkg/metrics       |

### 2.2 前端模块（frontend/test-dashboard）

| 模块     | 路径                                  | 职责                |
| ------ | ----------------------------------- | ----------------- |
| 入口     | `src/main.tsx`、`App.tsx`            | 路由 + 全局 Provider  |
| 页面     | `src/pages/Dashboard/`              | 主页：场景选择 + 实时指标    |
| <br /> | `src/pages/Screen/`                 | 大屏模式              |
| <br /> | `src/pages/History/`                | 历史列表              |
| <br /> | `src/pages/Report/`                 | 报告详情              |
| 组件     | `src/components/ScenarioCard/`      | 场景卡片              |
| <br /> | `src/components/MetricsPanel/`      | 实时指标面板            |
| <br /> | `src/components/StateMachineTrace/` | 状态机轨迹流（场景 H）      |
| <br /> | `src/components/AntiSnipeTimeline/` | 防狙击时间轴（场景 F）      |
| <br /> | `src/components/ChaosControl/`      | 故障注入控制            |
| <br /> | `src/components/ABCompare/`         | A/B 同屏对比          |
| 状态     | `src/store/testStore.ts`            | Zustand：测试任务      |
| <br /> | `src/store/wsStore.ts`              | Zustand：WebSocket |
| API    | `src/api/test.ts`                   | Axios 封装          |

***

## 3. 关键技术方案

### 3.1 任务调度模型 (runner)

* 每次"启动测试"由 `runner.Submit(scenario, config)` 接收

* 同步动作：写入 `test_results` 一条 `running` 记录，返回 `test_id`

* 异步动作：goroutine 执行 `scenario.Run(ctx, cfg, progressCh)`

* progress 通道由 `progress.Broker` 广播给订阅了 `test_id` 的 WS 客户端

* 完成后更新状态为 `completed`/`failed` 并写入 `result_json`

```go
type Scenario interface {
    Type() string
    Run(ctx context.Context, cfg json.RawMessage, p ProgressEmitter) (any, error)
}

type ProgressEmitter interface {
    Emit(progress int, step string, metrics map[string]any)
}
```

**取消语义**：`runner` 持有每个 task 的 `context.CancelFunc`，`/api/test/cancel/:id` 调用即可。

### 3.2 场景 A 压力测试

* 基于 goroutine pool（自实现，不引第三方库）

* 入口请求：调用 `gateway-service` 的 `POST /api/auctions/{id}/bid`

* 采样：原子计数器 + 滑动窗口（每秒一次 emit）

* 指标：QPS / avg latency / P99 / 成功率 / 错误码分布

* P99 用 [`hdrhistogram-go`](https://github.com/HdrHistogram/hdrhistogram-go) 或简化的桶式直方图（先用桶式，单依赖）

### 3.3 场景 E 全链路 E2E

时序：

```
[setup]  POST /api/products       (创建拍品)
         POST /api/live-streams   (创建直播间)
         POST /api/auctions       (创建拍卖)
[run]    POST /api/auctions/{id}/start
         (并发) POST /api/auctions/{id}/bid × N
         (订阅) POST /api/sky-lamp/subscribe × M
[verify] GET /api/auctions/{id}        (状态 = ENDED)
         GET /api/auctions/{id}/winner (中标用户唯一)
         GET /api/orders?auction_id=   (恰好 1 单)
         (轮询) Mock Partner 收到 1 次回调
[clean]  可选：清理测试数据
```

**StepResult 统一结构**：每步记录耗时、成功否、错误信息，前端按步骤可视化。

### 3.4 场景 F 防狙击延时

* 设置一个截拍倒计时短的拍卖（如 30s）

* 末段窗口（最后 5s）每 200ms 出价一次

* 观测 `auction.delay_used` 与 `auction.end_time` 的变化

* 同时校验 5 类用例（参见 spec §F）

* 前端时间轴组件：把"出价点 / 延时累计 / 截拍点"画在同一时间线上

### 3.5 场景 H 回调可靠投递

**Mock Partner 形态**：

* 在 test-service 进程内启动一个独立 HTTP server（端口 18091）

* 路由：

  * `POST /partner/orders` 接收回调

  * `GET /partner/orders/by-idempotency-key/:key` 用于 Probe

  * `POST /partner/_admin/config` 配置故障：`delay_ms` / `fail_rate` / `tamper_signature`

  * `GET /partner/_admin/inbox` 查看收到的回调（用于断言）

* 落盘到内存 map（演示场景够用）

**测试用例触发方式**：

| 用例             | 注入手段                                          |
| -------------- | --------------------------------------------- |
| normal         | 默认配置                                          |
| timeout        | `delay_ms = 65000` 触发回调超时 → Outbox 进入 Unknown |
| duplicate      | runner 主动重发 5 次                               |
| tampered       | `tamper_signature = true`                     |
| dlq            | `fail_rate = 1.0` 持续 N 次                      |
| out\_of\_order | 先发新事件再发旧事件                                    |

**状态机轨迹**：从 `outbox_events` 表读取 `state` 历史并回传前端可视化。

### 3.6 场景 G 混沌测试

| 故障       | 工具        | 实现细节                                                                                                    |
| -------- | --------- | ------------------------------------------------------------------------------------------------------- |
| Redis 闪断 | toxiproxy | 在 docker-compose 增加 toxiproxy；test-service 调 `POST localhost:8474/proxies/redis/toxics` 增加 `down` toxic |
| MQ 消费者暂停 | 应用内特性开关   | auction-service 增 `/internal/mq/pause` `/internal/mq/resume`（仅监听 127.0.0.1）                             |
| 网络丢包/延迟  | toxiproxy | latency / bandwidth toxic                                                                               |

**docker-compose.yml 改动**：把 redis、rabbitmq 的暴露端口改为通过 toxiproxy 透传：

* redis-real: 6379（仅内网）

* toxiproxy: 26379 → 6379

* 应用配置改为连 toxiproxy

**观测项**：

* 用户可见错误数（HTTP 5xx + WS 断连数）

* 系统恢复时间（从注入到指标恢复正常的 ms 数）

* 数据丢失检测（出价数 = 入库数 = WS 推送数）

### 3.7 进度 WebSocket

* 路径：`/ws/test/progress`

* 鉴权：复用 gateway 的 JWT

* 消息：spec §2.2 已定义

* 扩展：

  * `type: state_machine_step` 用于场景 H

  * `type: timeline_event` 用于场景 F

  * `type: chaos_event` 用于场景 G

### 3.8 replay\_token

* 任何场景启动时生成 `uuid` 作为 replay\_token

* 同时把 `config_json` + 关键随机种子（用户 ID 列表、出价时序）落库

* `POST /api/test/replay/:token` 读出原始输入再次 `runner.Submit`

***

## 4. 数据模型与存储

```sql
-- 已在 spec 中定义；这里追加：
CREATE TABLE test_seed_data (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    test_id VARCHAR(36) NOT NULL,
    kind VARCHAR(20),           -- product/auction/user
    ref_id BIGINT,              -- 业务表主键
    INDEX idx_test_id (test_id)
);
-- 用于 E2E 测试结束后清理
```

**清理策略**：

* 每个 E2E 测试在 `defer` 中调清理（按 `test_seed_data` 反向删）

* 每日 cron 清理 7 天前的 `test_results`

***

## 5. 部署与配置

### 5.1 服务端口

| 服务              | 端口    | 备注          |
| --------------- | ----- | ----------- |
| test-service    | 18090 | 主 HTTP      |
| Mock Partner    | 18091 | 进程内子 server |
| toxiproxy admin | 8474  | 故障控制        |
| toxiproxy redis | 26379 | Redis 透传    |

### 5.2 网关路由新增

`backend/gateway/router/router.go` 增加：

* `/api/test/*` → test-service:18090

* `/ws/test/*`  → test-service:18090

### 5.3 docker-compose 改动

* 增加 `toxiproxy` 服务

* `auction-service` / `product-service` 的 `REDIS_HOST` 指向 toxiproxy

* 增加 `test-service` 服务

### 5.4 Nacos 配置

新增 `test-config.yaml`：

* `target.gateway_url`

* `target.auction_url`（直连用于自检）

* `mock.partner_port`

* `chaos.toxiproxy_url`

* `cleanup.retention_days`

***

## 6. 里程碑

> 不给绝对时间估算，按交付物里程碑切分。

### M1 - 骨架可跑通 (P0)

* test-service 工程骨架（main + handler + runner + dao）

* `test_results` 表迁移

* 网关路由透传

* 前端工程骨架（Vite + 路由 + Layout + Zustand）

* 一个 dummy 场景 + 进度 WS 跑通

* **验收**：在 dashboard 点击按钮 → 看到进度条 → 看到完成

### M2 - 场景 A 压力测试 (P0)

* runner 完善 + goroutine pool

* 指标采集（QPS/Latency/P99）

* 前端实时图表（Recharts 或 Chart.js）

* **验收**：1000 并发出价跑通，图表实时刷新

### M3 - 场景 E E2E 全链路 (P0)

* 业务编排器（依次调 product/auction/bid API）

* StepResult 可视化组件

* test\_seed\_data 清理逻辑

* **验收**：一键播放→完整状态机轨迹→中标判定正确

### M4 - 场景 F 防狙击 (P0)

* 末刻出价模拟器

* AntiSnipeTimeline 组件

* **验收**：5 个测试用例全部 PASS，时间轴可视化清晰

### M5 - 场景 H 回调可靠投递 (P0)

* Mock Partner（内置 server + 故障开关 + Probe API）

* 6 个用例触发逻辑

* StateMachineTrace 组件

* **验收**：超时 → Probe → 幂等拒绝路径完整呈现

### M6 - 场景 G 混沌注入 (P0)

* toxiproxy 接入 + docker-compose 改造

* MQ 暂停/恢复内部接口（auction-service 改动）

* ChaosControl 组件

* **验收**：3 种故障可注入，恢复时间和数据无丢失指标可见

### M7 - 演示增强 (P0+)

* 5 个剧本配置化

* A/B 对比模式

* 大屏模式 `/test/screen`

* replay\_token 复现

* **验收**：现场盲演可走完 5 个剧本不出错

### M8 - 加固与文档

* 历史/报告页面完善

* Grafana 大盘 link

* 操作手册 + 演示话术（可选）

***

## 7. 风险与缓解

| 风险                                   | 影响 | 缓解                                             |
| ------------------------------------ | -- | ---------------------------------------------- |
| toxiproxy 引入改 docker-compose 影响其他开发者 | 中  | 用环境变量开关；默认不启用，演示前一键打开                          |
| Mock Partner 与真实 SDK 行为偏差            | 中  | 严格按 SDK 设计文档实现签名/幂等                            |
| 测试数据污染主库                             | 高  | 使用独立 schema 或 `test_` 前缀；test\_seed\_data 兜底清理 |
| 1000 并发把测试机自身打挂                      | 中  | 限制单机最大并发；超过则提示扩容                               |
| WS 进度消息洪水                            | 低  | 服务端节流（每 200ms 最多一帧）                            |
| auction-service 暴露 chaos 内部接口被误调     | 高  | 仅监听 127.0.0.1 + 内网鉴权                           |

***

## 8. 决策记录 (ADR)

| 决策                           | 选择                      | 理由                     |
| ---------------------------- | ----------------------- | ---------------------- |
| 测试服务独立 vs 嵌入 auction-service | 独立                      | 隔离故障域、独立部署、独立伸缩        |
| Mock Partner 形态              | 内置进程内                   | 部署简单、零额外服务             |
| 混沌注入工具                       | toxiproxy               | 业界主流、社区成熟、CLI/HTTP 双形态 |
| P99 计算                       | 桶式直方图                   | 零依赖、精度足够演示             |
| 状态管理                         | Zustand                 | 与既有 admin 项目一致         |
| 清理策略                         | test\_seed\_data + cron | 既保护现网数据、又允许复盘          |

***

## 9. 与既有项目的接触面

| 修改点                   | 文件                                                                                                                  | 改动类型                              |
| --------------------- | ------------------------------------------------------------------------------------------------------------------- | --------------------------------- |
| 网关路由                  | [router.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go) | 新增 `/api/test/*`、`/ws/test/*`     |
| auction-service MQ 控制 | `backend/auction/mq/consumer.go`                                                                                    | 新增 `Pause/Resume`，仅 127.0.0.1 暴露  |
| docker-compose        | [docker-compose.yml](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docker-compose.yml)      | 新增 toxiproxy、test-service         |
| Nacos 配置              | [configs/nacos](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/configs/nacos)                | 新增 test-config.yaml               |
| 数据库                   | 迁移脚本                                                                                                                | 新增 test\_results、test\_seed\_data |

**承诺零侵入**：不修改 auction-service、product-service 的业务逻辑；MQ 控制接口为可选编译标签 `chaos_enabled`。

***

## 10. 下一步

1. 用户确认本 plan
2. 进入 `tasks.md` 阶段，把 M1-M8 拆为具体可执行任务清单
3. 按 M1 开工


# 竞拍系统展示测试平台 - 任务清单 (Tasks)

> 配套：[spec](./2026-05-30-test-platform-spec.md) · [plan](./2026-05-30-test-platform-plan.md)
> **状态**: 当前任务状态文档；M1 已按 2026-05-30 代码现状同步，M2+ 仍为目标态任务
> **Gateway 本地端口 SSOT**: `8080`
> 任务编号规则：`<里程碑>.<序号>`，例如 `M1.3`
> 状态约定：☐ 未开始 / ◐ 进行中 / ✅ 已完成
> 每个任务含：**输入** / **产物** / **验收点** / **依赖**

---

## 全局规约（任务执行前必读）

1. **路径前缀**：后端新代码统一在 `backend/test/`；前端在 `frontend/test-dashboard/`
2. **包名规范**：Go 包名 `test-service` 的 module 路径 `test-service`，与 auction/product 同级
3. **代码风格**：沿用 `backend/auction` 既有风格（DDD 分层、Hertz handler 签名）
4. **配置加载**：复用 `backend/pkg/nacos`，环境变量 fallback
5. **日志**：复用 `backend/pkg/logger`
6. **指标**：复用 `backend/pkg/metrics`
7. **DB 迁移**：SQL 文件放 `backend/test/migrations/`，命名 `001_create_test_results.sql`
8. **每完成一个任务**：本地 `go build` / `npm run build` 通过即可视为完成；不要求写单测除非显式标注

---

## M1 - 骨架可跑通

> 目标：从前端点按钮 → 后端起 dummy 任务 → WS 推送进度 → 前端看到完成

### M1.1 ✅ 创建后端工程骨架
**输入**：plan §2.1 模块拆分
**产物**：
- `backend/test/go.mod`（module 名 `test-service`）
- `backend/test/main.go`（启动 Hertz on :18090，挂一个 `/health` 路由）
- `backend/test/config/config.go`（参照 `backend/auction/config`）
- `backend/test/handler/health.go`
**验收**：`go build ./...` 通过；`./test-service` 启动后 `curl :18090/health` 返回 200
**依赖**：无

### M1.2 ✅ 创建数据库迁移脚本
**输入**：spec §2.3 + plan §4
**产物**：
- `backend/test/migrations/001_create_test_results.sql`（含 spec 中字段及 `replay_token`/`script_name`）
- `backend/test/migrations/002_create_test_seed_data.sql`
**验收**：在本地 MySQL 手动执行通过；表结构与 spec/plan 完全一致
**依赖**：M1.1

### M1.3 ✅ 实现 DAO 层
**输入**：M1.2 表结构
**产物**：
- `backend/test/model/test.go`（`TestResult`、`TestSeedData` 结构体；`Status` 常量）
- `backend/test/dao/db.go`（GORM 初始化）
- `backend/test/dao/result.go`（`Save`、`UpdateStatus`、`GetByID`、`GetHistory`）
- `backend/test/dao/seed.go`（`AddSeed`、`ListByTestID`、`DeleteByTestID`）
**验收**：单元自检：起服务后插入一条 → 查询能读出
**依赖**：M1.2

### M1.4 ✅ 实现 runner 与 Scenario 接口
**输入**：plan §3.1
**产物**：
- `backend/test/runner/scenario.go`（`Scenario` 与 `ProgressEmitter` 接口）
- `backend/test/runner/runner.go`（`Submit`、`Cancel`、`Get`，内部用 `sync.Map` 维护活跃任务）
- `backend/test/runner/dummy.go`（dummy 场景，10 步×500ms，按比例 emit progress）
**验收**：`runner.Submit("dummy", nil)` 后能从 `Get(id)` 读到 progress 单调递增到 100
**依赖**：M1.3

### M1.5 ✅ 实现进度 Broker 与 WS Handler
**输入**：plan §3.7
**产物**：
- `backend/test/ws/broker.go`（订阅/发布；按 test_id 分发；服务端节流 200ms）
- `backend/test/handler/ws.go`（升级 WS、订阅 broker、按需关闭）
- 在 `main.go` 注册 `/ws/test/progress`
**验收**：`wscat` 连接 → 跑 dummy 场景 → 能持续收到 JSON 进度消息
**依赖**：M1.4

### M1.6 ✅ 实现 HTTP API 入口（最小集）
**输入**：spec §2.2 API 表
**产物**：
- `backend/test/handler/test.go`：
  - `POST /api/test/dummy`（仅 M1 用）
  - `GET /api/test/status/:id`
  - `GET /api/test/history`
  - `GET /api/test/report/:id`
  - `POST /api/test/cancel/:id`
- 在 `main.go` 注册路由组
**验收**：用 curl 走通"启动 → 查询 → 完成"全流程
**依赖**：M1.4

### M1.7 ✅ 网关路由透传
**输入**：plan §5.2
**产物**：[router.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go) 新增 `/api/test/*` 透传到 test-service:18090，并提供 `/ws/test/progress` endpoint discovery
**验收**：通过 gateway:8080 能访问到 test-service 的接口
**依赖**：M1.6

### M1.8 ✅ 创建前端工程骨架
**输入**：plan §2.2
**产物**：
- `frontend/test-dashboard/`（Vite + React 18 + TS + Zustand + Axios + React Router）
- `src/main.tsx`、`App.tsx`、基础 Layout
- 路由：`/test`、`/test/screen`、`/test/history`、`/test/report/:id`
- `src/api/test.ts`（含 dummy/status/history/cancel）
- `src/store/testStore.ts`、`src/store/wsStore.ts`
- 环境变量：`VITE_API_BASE = /api`、`VITE_WS_BASE = /ws`
**验收**：`npm run dev` 启动后 4 个路由均可访问，无报错
**依赖**：无

### M1.9 ✅ 前端骨架联调（Dummy 场景跑通）
**输入**：M1.6 + M1.8
**产物**：
- `src/pages/Dashboard.tsx`：放一个"启动 Dummy 测试"按钮
- 进度条组件 `src/components/ProgressBar/`
- WS 自动连接 + 接收消息逻辑
**验收**：点击按钮 → 进度条从 0% → 100%，每步状态文案能看到
**依赖**：M1.7、M1.8

---

## M2 - 场景 A 压力测试

### M2.1 ✅ 实现 goroutine pool
**产物**：`backend/test/runner/pool.go`（固定大小 worker pool，支持优雅关闭）
**验收**：单测：1000 并发任务执行无 goroutine 泄漏
**依赖**：M1.4

### M2.2 ☐ 实现指标采集器（HDR/桶式直方图）
**产物**：
- `backend/test/scenario/pressure/metrics.go`：原子计数 + 桶式直方图（`[1ms, 5ms, 10ms, 50ms, 100ms, 500ms, 1s, 5s, +∞]`）
- 提供 `Snapshot()` 返回 QPS / Avg / P50 / P95 / P99 / 错误码分布
**验收**：单测覆盖：注入 10000 个延迟样本，校验 P99 误差 < 5%
**依赖**：无

### M2.3 ☐ 实现压测客户端
**产物**：
- `backend/test/scenario/pressure/client.go`：HTTP client，调 `gateway:8080/api/auctions/{id}/bid`
- 支持自定义 header（鉴权 Token 注入）
**验收**：手动调一次返回 2xx
**依赖**：M2.1

### M2.4 ☐ 实现 PressureScenario
**输入**：spec §A
**产物**：`backend/test/scenario/pressure/pressure.go` 实现 `Scenario` 接口
- 配置：`ConcurrentUsers`、`Duration`、`TargetAuctionID`
- 每 1s emit 一次实时指标
**验收**：跑 100 并发 × 10s，指标符合预期
**依赖**：M2.1、M2.2、M2.3

### M2.5 ☐ 注册 API：/api/test/pressure
**产物**：`backend/test/handler/test.go` 增加 handler，反序列化配置 → 调 runner
**验收**：curl 启动压测，能从 status 接口读到完整结果
**依赖**：M2.4

### M2.6 ☐ 前端压测页面
**产物**：
- `src/pages/Dashboard/Pressure.tsx`：参数表单 + 启动按钮 + 实时指标面板
- `src/components/MetricsPanel/`：QPS/延迟图表（用 Recharts）
- WS 消息接入 `metrics` 字段实时刷新
**验收**：点击启动 → 看到曲线实时更新；结束后看到完整报告
**依赖**：M2.5、M1.9

### M2.7 ✅ 1000 并发压测验证
**产物**：实测报告（截图 + 数值，可放在 `docs/test-platform-screenshots/m2.md`）
**验收**：1000 并发 60s 跑通，前端图表流畅；test-service 自身无 OOM
**依赖**：M2.6

---

## M3 - 场景 E E2E 全链路

### M3.1 ✅ 实现业务客户端 SDK（内部用）
**产物**：`backend/test/client/auction/`（封装 product/auction/bid/skylamp/orders 的 HTTP 调用，统一返回 `StepResult{step,ok,status_code,ref_id,message,duration_ms}`）
**验收**：8 项单测 GREEN（含 happy path + 轮询 + 错误码）
**依赖**：M1.1

### M3.2 ✅ 实现 E2E 编排器
**输入**：plan §3.3 时序
**产物**：`backend/test/scenario/e2e/orchestrator.go`：
- setup → run → verify → cleanup 四阶段
- 每步产出 `StepResult` 并通过 `ProgressEmitter` 实时上报
- 通过 `SeedRecorder` 接口记录到 `test_seed_data`，cleanup 阶段反向删
- 抽象 `BizClient` 接口便于桩注入
**验收**：4 项编排器单测 GREEN（HappyPath / SetupFailure / VerifyFailure / ContextCancelled）；任何阶段失败 cleanup 必跑
**依赖**：M3.1、M1.3

### M3.3 ✅ 中标判定与订单核验
**产物**：合并到 orchestrator verify 阶段，包含 `verify_winner`、`verify_order_unique` 两步
**验收**：`TestOrchestrator_VerifyFailure` 覆盖订单数 = 0 场景；happy path 覆盖订单数 = 1
**依赖**：M3.2

### M3.4 ✅ 注册 API：/api/test/e2e
**产物**：`handler.PostE2E` + `runner.Register(e2e.NewScenario(...))`，gateway 自动透传
**验收**：`curl POST /api/test/e2e` 返回 test_id
**依赖**：M3.2

### M3.5 ✅ 前端 StepResult 可视化
**产物**：`src/components/StepTimeline.tsx` + `src/pages/E2E.tsx`，纵向时间轴含状态色（绿/红/灰）+ 耗时 + 中文步骤名 + 多次同名步骤自动编号
**验收**：vite build 通过；导航增加 "E2E 全链路" 入口
**依赖**：M3.4、M1.9

### M3.6 ✅ 清理任务定时器
**产物**：`backend/test/service/cron/cleanup.go`（time.Ticker 实现，每日 1 次清理 7 天前 test_results；启动时立即跑一次回收启动前积压）
**验收**：4 项 cron 单测 GREEN（RunOnce / Periodic / StopHaltsTicker / ErrorDoesNotPanic）
**依赖**：M1.3

---

## M4 - 场景 F 防狙击延时

### M4.1 ✅ 实现末刻出价模拟器
**输入**：spec §F
**产物**：`backend/test/scenario/antisnipe/simulator.go`：
- 创建一个 30s 倒计时拍卖
- 末段 5s 内每 200ms 一次出价
- 记录每次出价后的 `delay_used`、`end_time`
**验收**：跑一次能拿到完整时间轴
**依赖**：M3.1
**完成证据**：simulator_test.go 4 单测 GREEN（fakeClock + fakeAuctionAPI 注入）

### M4.2 ✅ 实现 5 个用例的断言
**产物**：`backend/test/scenario/antisnipe/cases.go`：
- 末刻触发延时
- 延时累计上限
- 多用户连环触发
- 安全期不触发
- 已封顶不再延时
**验收**：5 个用例 PASS/FAIL 都正确报告
**依赖**：M4.1
**完成证据**：cases_test.go 3 单测 GREEN，含 Scenario.Run 全用例集成

### M4.3 ✅ 注册 API：/api/test/antisnipe
**依赖**：M4.2
**完成证据**：handler/test.go::PostAntiSnipe + main.go 注册 antisnipe.NewScenario

### M4.4 ✅ 前端 AntiSnipeTimeline 组件
**产物**：`src/components/AntiSnipeTimeline.tsx`（横向时间轴：原计划截拍点、出价点、延时累计条、实际截拍点）+ `src/pages/AntiSnipe.tsx`
**验收**：肉眼能一秒看出"是否触发延时"
**依赖**：M4.3、M1.9
**完成证据**：npm run build PASS，路由 /test/antisnipe 已挂载

---

## M5 - 场景 H 回调可靠投递

### M5.1 ✅ 实现 Mock Partner Server
**输入**：plan §3.5
**产物**：
- `backend/test/mock/partner/server.go`（独立 :18091）
- 路由：
  - `POST /partner/orders`：接收回调，校验 HMAC，落 inbox
  - `GET /partner/orders/by-idempotency-key/:key`
  - `POST /partner/_admin/config`
  - `GET /partner/_admin/inbox`
  - `POST /partner/_admin/reset`
- 内存存储 + 互斥锁
**验收**：单测覆盖 6 种用例的 server 行为
**依赖**：M1.1
**完成证据**：server_test.go 6 单测 GREEN

### M5.2 ✅ 实现回调测试场景
**产物**：`backend/test/scenario/callback/callback.go`：
- 通过 reset → config → 触发 → 等待 → 断言 顺序跑 6 用例
- 状态机轨迹从 outbox 表读取（如果项目还没有 outbox 表，先用 mock 数据演示——加 TODO 标注）
**验收**：6 用例输出明确 PASS/FAIL + 状态轨迹
**依赖**：M5.1
**完成证据**：callback_test.go 6 单测 GREEN；现走 Mock 路径（详见 M5.5）

### M5.3 ✅ 注册 API：/api/test/callback
**依赖**：M5.2
**完成证据**：handler/test.go::PostCallback + main.go 内嵌启动 Mock Partner Server :18091 + 注册 callback.NewScenario

### M5.4 ✅ 前端 StateMachineTrace 组件
**产物**：`src/components/StateMachineTrace.tsx`（节点流图：Pending → Sending → Unknown → Probing → Confirmed/DLQ）+ `src/pages/Callback.tsx`
**验收**：6 用例的轨迹路径直观可辨
**依赖**：M5.3、M1.9
**完成证据**：npm run build PASS，路由 /test/callback 已挂载

### M5.5 ✅ outbox 表与现状对齐（调研型任务）
**产物**：`docs/test-platform-outbox-status.md`：记录当前 auction-service 是否已有 outbox 实现；若没有，给出最小实现路径或 mock 方案
**验收**：明确告诉 M5.2 数据来源是 真实表 还是 mock
**依赖**：无（可前置并行）
**完成证据**：项目无 outbox/HMAC/partner_callback 任何实现，本里程碑全部走 Mock 路径

---

## M6 - 场景 G 混沌注入

> **MVP 实现说明（2026-05-30）**：
> 已采用进程内故障注入 MVP（不依赖 toxiproxy / docker-compose 改造）：在 test-service 内部
> 实现 `ChaosBroker` + `ChaosTransport`（http.RoundTripper 装饰器），通过 HTTP 客户端注入
> 延迟 / 错误率 / 强制断连。M6.1/M6.2/M6.3/M6.5/M6.6 在 MVP 中跳过；M6.4 用进程内 latency
> 注入替代网络层注入。

### M6.1 N/A 引入 toxiproxy（MVP 已用进程内注入替代）
**产物**：
- 修改 [docker-compose.yml](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docker-compose.yml)：增加 `toxiproxy` 服务（暴露 8474、26379、25672）
- 默认 disabled，通过 `COMPOSE_PROFILES=chaos` 启用
- 改 auction/product 的 `REDIS_HOST`/`MQ_HOST` 配置（环境变量切换）
**验收**：`docker compose --profile chaos up` 启动正常；不带 profile 时与现状一致
**依赖**：无

### M6.2 ✅ 实现 chaos broker（替代 toxiproxy 客户端）
**产物**：[broker.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/chaos/broker.go) + [broker_test.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/chaos/broker_test.go)
（FaultType 枚举 + Profile + Broker 单例 + ChaosTransport http.RoundTripper 装饰器；5 单测全绿）
**依赖**：无

### M6.3 N/A 实现 Redis 闪断 chaos（MVP 跳过）
**产物**：`backend/test/chaos/redis/redis.go`（Inject/Recover）
**依赖**：M6.2

### M6.4 ✅ 实现进程内延迟/错误率注入（替代网络层）
**产物**：通过 Broker 的 `FaultLatency` / `FaultErrorRate` / `FaultDisconnect`，在
[ChaosTransport.RoundTrip](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/chaos/broker.go) 内 sleep + deny。
**依赖**：M6.2

### M6.5 N/A auction-service 增加 MQ 暂停接口（MVP 跳过）
**产物**：
- `backend/auction/handler/internal_chaos.go`（编译标签 `//go:build chaos_enabled`）
- 路由 `/internal/mq/pause`、`/internal/mq/resume`，仅 127.0.0.1 监听
- 在 `mq/consumer.go` 增 `Pause()`、`Resume()` 方法
**验收**：默认编译不含此 handler；`go build -tags chaos_enabled` 启用
**依赖**：无

### M6.6 N/A 实现 MQ 暂停 chaos 客户端（MVP 跳过）
**产物**：`backend/test/chaos/mq/mq.go`（调 auction-service 内部接口）
**依赖**：M6.5

### M6.7 ✅ 实现 ChaosScenario
**输入**：spec §G
**产物**：[chaos.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/scenario/chaos/chaos.go) + [chaos_test.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/scenario/chaos/chaos_test.go)
（三阶段 baseline → inject → recover；每秒 bucket；输出 detection_latency_ms / recovery_latency_ms / 错误率统计）
**依赖**：M6.2 + M6.4

### M6.8 ✅ 注册 API：/api/test/chaos
**产物**：[handler/test.go PostChaos](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/handler/test.go) + [main.go 注册](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/main.go)
**依赖**：M6.7

### M6.9 ✅ 前端 ChaosControl + Chaos 页面
**产物**：[Chaos.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/test-dashboard/src/pages/Chaos.tsx)
（故障类型选择 + 强度 + 持续时间；实时进度 + 阶段错误率指标 + 每秒错误曲线 SVG）
**依赖**：M6.8、M1.9

---

## M7 - 演示增强

### M7.1 ✅ 剧本配置化
**产物**：
- [scenario/script/script.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/scenario/script/script.go)：内置 5 个剧本（quickstart/antisnipe/reliability/chaos/fullshow），按顺序串行执行 sub-scenario，子进度映射到统一进度条
- [scenario/script/script_test.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/scenario/script/script_test.go)：5 单测全绿（Library + 子进度映射 + 失败传播）
- API：`POST /api/test/script/:name`（[handler/test.go PostScript](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/handler/test.go)）
- runner 增加 `Get(type)` 方法供 ScenarioGetter 接口使用
**说明**：使用 Go 内置 `Library` map 替代外部 yaml 文件（避免引入 yaml 依赖）
**依赖**：M2-M6 完成

### M7.2 ☐ replay_token 复现（未做）
**产物**：
- `backend/test/runner/runner.go` 在 Submit 时生成 token + 落 `seed_data` 中关键随机种子
- API：`POST /api/test/replay/:token`
**验收**：跑一次 → 用 token 复现 → 输入序列一致
**依赖**：M2.4

### M7.3 ✅ A/B 对比模式
**产物**：
- 后端：`POST /api/test/compare`（[handler/test.go PostCompare](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/handler/test.go)）
- 前端：[Compare.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/test-dashboard/src/pages/Compare.tsx)（同屏左右两块指标面板 + 3 个预设：pressure/chaos/antisnipe）
**依赖**：M4 + M7.1

### M7.4 ✅ 大屏模式
**产物**：[Screen.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/test-dashboard/src/pages/Screen.tsx)（暗色主题 + 大字号统计 + 4 卡片自动轮播 5s + 时钟 + 隐藏侧栏，路由独立于 Layout）
**依赖**：M2-M6

### M7.5 ☐ 5 个剧本端到端联调（未做）
**产物**：5 段录屏或截图清单
**验收**：现场盲演可不 NG 走完
**依赖**：M7.1-M7.4

---

## M8 - 加固与文档

### M8.1 ☐ 历史/报告页面完善
**产物**：
- `src/pages/History/`：列表 + 筛选 + 分页
- `src/pages/Report/`：详情页 + 重跑按钮 + replay 按钮
**依赖**：M1.6

### M8.2 ☐ Grafana 大盘 link
**产物**：导航上常驻按钮，跳转到 Grafana 既有大盘
**依赖**：无

### M8.3 ☐ 操作手册
**产物**：`docs/test-platform-operation.md`
- 启动顺序
- 5 个剧本演示流程 + 推荐话术
- 故障排查 FAQ
**依赖**：M1-M7

### M8.4 ☐ 性能与稳定性回归
**产物**：跑全部场景一轮，记录指标基线到 `docs/test-platform-baseline.md`
**依赖**：全部前置

---

## 任务统计

| 里程碑 | 任务数 | 关键风险点 |
|------|------|----------|
| M1 | 9 | gateway 路由调整冲突 |
| M2 | 7 | 1000 并发本机资源 |
| M3 | 6 | 测试数据清理 |
| M4 | 4 | 30s 倒计时不稳定 |
| M5 | 5 | outbox 表是否就绪 |
| M6 | 9 | docker-compose 改动影响他人 |
| M7 | 5 | 剧本时序耦合 |
| M8 | 4 | — |
| **合计** | **49** | |

---

## 任务依赖图（关键路径）

```
M1.1 → M1.2 → M1.3 → M1.4 → M1.5 → M1.6 → M1.7
                              ↓
                            M1.8 → M1.9
                                    ↓
        ┌─────────────┬────────────┴────────────┬──────────────┐
        ↓             ↓                         ↓              ↓
       M2.x         M3.x                       M4.x          M5.x
        └─────────────┴────────────┬────────────┴──────────────┘
                                   ↓
                                  M6.x
                                   ↓
                                  M7.x
                                   ↓
                                  M8.x
```

**关键路径**：M1.1 → M1.7 → M2.1-M2.6 → M3.1-M3.5 → M4 → M5 → M6 → M7 → M8

---

## 下一步

- 当前状态：M1 骨架和 Dummy 联调已按代码现状同步为完成
- 建议起手：进入 **M2 压力测试场景**，先实现 `M2.1 → M2.2 → M2.3`
- 一句话指令开工：「开始 M2.1」/「按顺序推进 M2 压测场景」

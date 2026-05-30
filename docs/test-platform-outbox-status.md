# Test Platform - Outbox 现状对齐报告

> 配套：[场景 H 回调可靠投递](../.superpowers/specs/2026-05-30-test-platform-spec.md)
> 任务来源：tasks.md M5.5

## TL;DR

**当前 auction-service 不存在 outbox 表，也无 partner / external callback / HMAC 任何实现。**
M5（场景 H 回调测试）将完全基于 **Mock 实现**：Mock Partner Server + 模拟状态机轨迹（in-memory）。

## 调研结论

### 关键词命中（在 backend/ 全目录搜索，不区分大小写）

| 关键词 | 命中数 | 说明 |
|---|---|---|
| outbox | 0 | 无 |
| event_log | 0 | 无 |
| partner_callback / PartnerCallback | 0 | 无 |
| HMAC / Hmac | 0 | 无 |
| webhook | 0 | 无 |
| callback | 27 | 全部为无关用途（gorm.Callback / growthbook.trackingCallback / mock 通知 / TypeCallback 字符串常量等） |

### Migration 文件（全 3 个 SQL 文件均与 outbox 无关）

- [001_create_notifications.sql](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/migration/001_create_notifications.sql)（站内通知表）
- [001_create_test_results.sql](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/migrations/001_create_test_results.sql)
- [002_create_test_seed_data.sql](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/migrations/002_create_test_seed_data.sql)

### 通知机制现状

项目目前所有"通知"均为站内通知（`model.Notification` + WebSocket 推送），无对外回调链路。
OpenAPI/SDK 设计虽已在 [docs/superpowers/specs/2026-05-30-live-auction-openapi-sdk-design.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/superpowers/specs/2026-05-30-live-auction-openapi-sdk-design.md) 中规划，但代码层面尚未落地。

## M5 数据来源决策

| 数据项 | 来源 | 备注 |
|---|---|---|
| 状态机轨迹 (Pending → Sending → Unknown → Probing → Confirmed/DLQ) | **Mock**：在 callback 场景内实现一个轻量内存状态机 | 不依赖真实 outbox 表 |
| 回调 HTTP 请求与响应 | Mock Partner Server (:18091) | M5.1 |
| 幂等键 (idempotency_key) | 由 callback scenario 内部生成 + 写入 mock 状态机 | UUID v4 |
| HMAC 密钥 | 写死在 callback scenario 与 mock partner 间共享（默认 `test-secret-key`） | 后续可配置 |
| 死信队列 (DLQ) | Mock Partner Server 配置：连续 N 次失败后回调将停止重试，进入 DLQ 状态 | 在内存维护 |

## M5 实现路径

1. **M5.1 Mock Partner Server**（独立 :18091 HTTP server，内存 inbox + admin 配置故障）
2. **M5.2 Callback Scenario**：内置一个轻量"投递器 + 状态机"，按 6 用例配置 Mock Partner 行为，触发投递、观察轨迹、断言结果
3. **M5.4 前端 StateMachineTrace**：直接消费 scenario 输出的轨迹数组

## 后续真实化路径（非本期范围）

若未来要把 callback 接入真实 outbox：
1. auction-service 增加 `callback_outbox` 表（`id`/`event_type`/`payload`/`idempotency_key`/`status`/`retry_count`/`last_error`/`created_at`/`next_retry_at`）
2. 出价中标后在事务内 Insert outbox 记录 → 异步 worker 拉取投递
3. 投递失败按 Probe-before-Retry 走探测流程（探测命中即 Confirmed，否则继续重试到 DLQ 阈值）
4. callback scenario 改为读真实 outbox 表，**Mock Partner Server 仍可保留** 作为可控故障注入工具

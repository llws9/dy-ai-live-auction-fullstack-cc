# SDD Run State - 2026-06-01-fixed-price-m1-backend

> SSOT for the M1 (后端核心抢购链路) execution. Plan checkbox 直接作为 tasks。

## Run Metadata

| Key | Value |
| --- | --- |
| Run ID | `2026-06-02-fixed-price-m1-backend` |
| Topic | `fixed-price-m1-backend` |
| Goal | 实现后端一口价上下架 + 抢购（幂等/Saga补偿/零超卖），单元+集成测试可演示闭环 |
| Mode | `subagent-driven` |
| Branch | `feat/fixed-price-m1` |
| Worktree | `/Users/bytedance/.config/superpowers/worktrees/dy-ai-live-auction-fullstack-cc/feat-fixed-price-m1` |
| Base Branch | `main` |
| Started At | `2026-06-02 23:20` |
| Owner | `main-agent` |
| Status | `active` |

## Input Documents

| Type | Path | Required | Loaded |
| --- | --- | --- | --- |
| Agent Rules | `AGENTS.md` | yes | yes |
| SDD Runbook | `docs/superpowers/sdd/RUNBOOK.md` | yes | yes |
| Spec | `docs/superpowers/specs/2026-06-01-fixed-price-sale-design.md` | yes | yes |
| Plan / Tasks | `docs/superpowers/plans/2026-06-01-fixed-price-m1-backend.md` | yes | yes |

## Execution Summary

| Metric | Value |
| --- | --- |
| Total Tasks | `13` |
| Done | `5` |
| Blocked | `0` |
| In Progress | `0` |
| Pending | `8` |
| Last Updated | `2026-06-03` |

> **W4 架构决策（RESOLVED）：** 经调查（`orders` 表无任何真实建单链路、CreateOrder 仅单测调用、拍卖成交不落单、全仓无 Outbox 设施、`user_balance` 与 `fixed_price_*` 同在 auction 库），用户拍板采用**方案③ purchase 自成闭环**：`fixed_price_purchases` 即购买凭证，auction 单库单事务完成 `扣余额+扣库存+写purchase+幂等`，**不写 product 的 orders 表**。已撤销 T1 的 `orders.source` 列 DDL 及 `FixedPricePurchase.OrderID`/`order_id` 列（commit f6855288）。T7 阻塞解除，不再依赖 product service。

## Status Legend

`pending` → `assigned` → `in_progress` → `verifying` → `review` → `done`；旁路 `blocked` / `changes_requested`。

## Task Matrix

| Task ID | Title | Status | Owner | Wave | Depends On | Allowed Files |
| --- | --- | --- | --- | --- | --- | --- |
| `T1` | DDL 迁移 + model 定义 | `done` | `subagent` | W1 | - | `backend/migrations/2026060101_*.sql`, `backend/auction/model/fixed_price.go` |
| `T2` | dao FixedPriceItem CRUD | `done` | `subagent` | W2 | T1 | `backend/auction/dao/fixed_price_item.go(+_test)` |
| `T3` | dao FixedPricePurchase 唯一键 | `done` | `subagent` | W2 | T1 | `backend/auction/dao/fixed_price_purchase.go(+_test)` |
| `T4` | Lua 原子库存抢占 | `done` | `subagent` | W3 | T1 | `backend/auction/service/fixed_price_lua.go(+_test)` |
| `T5` | 幂等存储 | `done` | `subagent` | W3 | T1 | `backend/auction/service/fixed_price_idem.go(+_test)` |
| `T6` | service 上架接口 | `pending` | - | W4 | T2,T4 | `backend/auction/service/fixed_price.go(+_test)` |
| `T7` | service 抢购（Lua+Tx+Saga+幂等） | `pending` | - | W4 | T2,T3,T4,T5 | `backend/auction/service/fixed_price.go(+_test)` |
| `T8` | service 下架（软标记+5s清Redis） | `pending` | - | W4 | T2,T4 | `backend/auction/service/fixed_price.go(+_test)` |
| `T9` | handler 抢购 + 错误码映射 | `pending` | - | W5 | T7 | `backend/auction/handler/fixed_price.go(+_test)` |
| `T10` | handler 上架/下架/详情/my-purchase + 路由 | `pending` | - | W5 | T6,T8 | `backend/auction/handler/fixed_price.go`, `fixed_price_http.go`, `router.go` |
| `T11` | gateway 转发路由 | `pending` | - | W6 | T9,T10 | `backend/gateway/router.go` |
| `T12` | Toxiproxy 集成测试（网络异常补偿） | `pending` | - | W7 | T7 | `backend/auction/service/fixed_price_toxi_test.go` |
| `T13` | E2E 冒烟 + 拍卖回归 | `pending` | - | W7 | T11 | `backend/auction/...`（只读回归 + 冒烟脚本） |

## Wave Plan

| Wave | Goal | Tasks | 并行性 | Start Condition | Completion Condition |
| --- | --- | --- | --- | --- | --- |
| W1 | 数据契约就绪 | T1 | 单 | state 初始化 | DDL+model 编译通过 |
| W2 | dao 层 | T2,T3 | 并行（不同文件） | T1 done | dao 测试全绿 |
| W3 | Redis 原语 | T4,T5 | 并行（不同文件） | T1 done | lua/idem 测试全绿 |
| W4 | service 编排 | T6→T7→T8 | 串行（同 fixed_price.go） | W2,W3 done | service 测试全绿 |
| W5 | handler 层 | T9→T10 | 串行（同 handler 文件） | W4 done | handler 测试全绿 |
| W6 | gateway 转发 | T11 | 单 | W5 done | 转发路由测试通过 |
| W7 | 集成 + 回归 | T12,T13 | 并行 | T7(T12)/T11(T13) done | Toxiproxy+冒烟+回归全绿 |

## Task Records

（每个 task 派发后由 subagent 回填：Status / Modified Files / 测试命令+结果 / Commit / Risks。）

### T1 - DDL 迁移 + model 定义
| Status | `done` |
| --- | --- |

**Scope**：一口价数据契约定义（DDL 迁移 + GORM model）。

**Modified / Created Files**
- Create: `backend/migrations/2026060101_create_fixed_price_tables.up.sql`（建 `fixed_price_items`、`fixed_price_purchases` 两表）
- Create: `backend/migrations/2026060101_create_fixed_price_tables.down.sql`（反向 drop 两表）
- Create: `backend/auction/model/fixed_price.go`（`FixedPriceStatus` 枚举、`FixedPriceItem`、`FixedPricePurchase`，金额字段用 `shopspring/decimal`，gorm/json tag 风格对齐 `auction.go`）

**约定对齐**
- `decimal.Decimal` + `gorm:"type:decimal(10,2);not null"` 与现有 `Order.FinalPrice`/`Auction.CurrentPrice` 一致。
- tag 顺序统一为 `json` 在前、`gorm` 在后，与本仓库现有 model 风格一致。

**验证（无 TDD 红灯的替代理由）**
本任务为纯数据契约定义，无独立单测；按 plan Step 1.5 以编译 + vet 为验收：
- `cd backend/auction && go build ./...` → 退出码 0，输出 `BUILD_OK`
- `cd backend/auction && go vet ./model/` → 退出码 0，输出 `VET_OK`

**Risks / Blockers**
- ✅ **订单模式已定（方案③ purchase 自成闭环）**：经调查现有 orders 表无真实建单链路（详见 Execution Summary）。已撤销 `orders.source` 列 DDL 与 `FixedPricePurchase.OrderID`/`order_id` 列（commit f6855288）。一口价不写 product orders 表，purchase 即凭证。T7 无跨服务依赖。
- 迁移如何被应用：本仓库主要依赖各 service `main.go` 的 GORM `AutoMigrate`；新表的真实建表挂载需在后续 task 将 `FixedPriceItem`/`FixedPricePurchase` 加入 `auction/main.go` 的 AutoMigrate 列表（不属 T1 范围）。

**Commit**
- `68e26ae4` — `feat(fixed-price): add DDL and models for fixed price sale (M1.T1)`（初版，含已撤销字段）
- `f6855288` — `refactor(fixed-price): drop orders.source and purchase.order_id for self-contained M1`（方案③修订）

### 测试基建（W2/W3 公共前置，main-agent 直接落地）
| Status | `done` |
| --- | --- |

auction-service 原本无 dao/redis 单测 fixture（现有 redis 测试均 `t.Skip("需要实际Redis连接")`）。引入：
- `github.com/glebarez/sqlite`（纯 Go GORM driver）→ `dao/testutil_test.go` 的 `setupTestDB`（隔离命名内存库 + AutoMigrate）。
- `github.com/alicebob/miniredis/v2`（支持 Lua EVAL）→ `service/testutil_test.go` 的 `setupTestRedis`。
- **Commit** `81874f3b` — `test(fixed-price): add in-memory sqlite + miniredis test helpers`

### T2 - dao FixedPriceItem CRUD
| Status | `done` |
| --- | --- |
- TDD：3 用例（CreateAndGet / UpdateStatus 合法转换 / ListByLiveStreamID）Red→Green。
- 状态流转表：OnSale→{SoldOut,Offline}、SoldOut→{Offline}、Offline→{}。
- 验证：`go test ./dao/ -run TestFixedPriceItemDAO -v` PASS(3)、`go vet ./dao/` clean。
- **Commit** `59988280` — `feat(fixed-price): add FixedPriceItemDAO with status transition guard (M1.T2)`

### T3 - dao FixedPricePurchase 唯一键
| Status | `done` |
| --- | --- |
- TDD：2 用例（Insert 唯一键冲突→ErrAlreadyBought / GetByItemAndUser）Red→Green。无 OrderID（方案③）。
- 关键：`isDuplicateKey` 同时匹配 MySQL `Duplicate entry` 与 sqlite `UNIQUE constraint failed`，保证测试环境可捕获冲突。提供 `InsertWithTx` 供 T7 Saga 外部事务复用。
- 验证：`go test ./dao/ -run TestFixedPricePurchaseDAO -v` PASS(2)、`go vet` clean。
- **Commit** `b4efcd54` — `feat(fixed-price): add FixedPricePurchaseDAO with unique key guard (M1.T3)`

### T4 - Lua 原子库存抢占 StockGuard
| Status | `done` |
| --- | --- |
- TDD：5 用例（Success/AlreadyBought/SoldOut/Uninitialized/Compensate）Red→Green。
- Lua：EXISTS→-3、SISMEMBER→-2、DECR<0 回滚 INCR→-1、SADD→1。提供 Init/TryAcquire/Compensate/Cleanup/Remaining。脚本仅用基础命令，miniredis 与真实 Redis 行为一致。
- 验证：`go test ./service/ -run TestStockGuard -v` PASS(5)、`go vet` clean。
- **Commit** `36bbe387` — `feat(fixed-price): add Lua-based atomic stock guard with compensation (M1.T4)`

### T5 - 幂等存储 IdemStore
| Status | `done` |
| --- | --- |
- TDD：4 用例（首次 miss / 二次命中 / UUID 校验 / TTL≈10min）Red→Green。
- key `fp:idem:%d:%d:%s`，TTL 10min；存的整数为 purchase ID 语义（方案③，无 order 概念）。
- 验证：`go test ./service/ -run TestIdemStore -v` PASS(4)、`go vet` clean。
- **Commit** `d3f35562` — `feat(fixed-price): add idempotency store with UUID validation (M1.T5)`

## Test Commands

| Area | Command | Last Result |
| --- | --- | --- |
| Backend Auction (dao+service) | `cd backend/auction && go test ./dao/ ./service/` | `pass (ok dao / ok service)` |
| Backend Auction (T1) | `cd backend/auction && go build ./... && go vet ./model/` | `pass (BUILD_OK / VET_OK)` |
| Backend Gateway | `cd backend/gateway && go test ./...` | `not_run` |

## Final Review Checklist

- [ ] 13 个 task 状态全部终态（done/blocked）。
- [ ] 每个实现型 task 有 TDD Red→Green→Verify 证据。
- [ ] 每个 subagent 回答第一句为 `当前分支/worktree：...`。
- [ ] DDL/契约变更与 spec 一致。
- [ ] 用户已获下一步选项。

## Final Handoff

当前分支/worktree：feat/fixed-price-m1 @ /Users/bytedance/.config/superpowers/worktrees/dy-ai-live-auction-fullstack-cc/feat-fixed-price-m1

**状态**
- `initialized` - W1 待派发

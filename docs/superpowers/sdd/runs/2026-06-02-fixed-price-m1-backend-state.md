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
| Done | `1` |
| Blocked | `0` |
| In Progress | `0` |
| Pending | `12` |
| Last Updated | `2026-06-03` |

## Status Legend

`pending` → `assigned` → `in_progress` → `verifying` → `review` → `done`；旁路 `blocked` / `changes_requested`。

## Task Matrix

| Task ID | Title | Status | Owner | Wave | Depends On | Allowed Files |
| --- | --- | --- | --- | --- | --- | --- |
| `T1` | DDL 迁移 + model 定义 | `done` | `subagent` | W1 | - | `backend/migrations/2026060101_*.sql`, `backend/auction/model/fixed_price.go`, `backend/auction/model/order.go` |
| `T2` | dao FixedPriceItem CRUD | `pending` | - | W2 | T1 | `backend/auction/dao/fixed_price_item.go(+_test)` |
| `T3` | dao FixedPricePurchase 唯一键 | `pending` | - | W2 | T1 | `backend/auction/dao/fixed_price_purchase.go(+_test)` |
| `T4` | Lua 原子库存抢占 | `pending` | - | W3 | T1 | `backend/auction/service/fixed_price_lua.go(+_test)` |
| `T5` | 幂等存储 | `pending` | - | W3 | T1 | `backend/auction/service/fixed_price_idem.go(+_test)` |
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
- Create: `backend/migrations/2026060101_create_fixed_price_tables.up.sql`（建 `fixed_price_items`、`fixed_price_purchases` 两表 + `ALTER TABLE orders ADD COLUMN source`）
- Create: `backend/migrations/2026060101_create_fixed_price_tables.down.sql`（反向：drop source 列 + drop 两表）
- Create: `backend/auction/model/fixed_price.go`（`FixedPriceStatus` 枚举、`FixedPriceItem`、`FixedPricePurchase`，金额字段用 `shopspring/decimal`，gorm/json tag 风格对齐 `auction.go`/`order.go`）

**约定对齐**
- `decimal.Decimal` + `gorm:"type:decimal(10,2);not null"` 与现有 `Order.FinalPrice`/`Auction.CurrentPrice` 一致。
- tag 顺序统一为 `json` 在前、`gorm` 在后，与本仓库现有 model 风格一致。

**验证（无 TDD 红灯的替代理由）**
本任务为纯数据契约定义，无独立单测；按 plan Step 1.5 以编译 + vet 为验收：
- `cd backend/auction && go build ./...` → 退出码 0，输出 `BUILD_OK`
- `cd backend/auction && go vet ./model/` → 退出码 0，输出 `VET_OK`

**Risks / Blockers**
- ⚠️ **Order model 的 Source 字段改动未落地（blocked，等待确认）**：plan 假设 `backend/auction/model/order.go` 存在，但实际 `Order` struct 定义在 **product service** 的 `backend/product/model/order.go`，且该文件不在本任务允许修改清单内。为避免越界改他人服务的 model，未追加 Go 字段。
  - DDL 层的 `ALTER TABLE orders ADD COLUMN source`（up）与 `DROP COLUMN source`（down）已按 plan 保留，数据契约完整。
  - 待确认事项：`Source` 字段的 Go model 改动应归属哪个 task / 是否扩大 T1 允许文件范围到 `backend/product/model/order.go`。后续 T7（抢购写订单）若需读写 `source`，依赖此项落地。
- 迁移如何被应用：本仓库主要依赖各 service `main.go` 的 GORM `AutoMigrate`；新表的真实建表挂载需在后续 task 将 `FixedPriceItem`/`FixedPricePurchase` 加入 `auction/main.go` 的 AutoMigrate 列表（不属 T1 范围）。

**Commit**：`68e26ae4a07bbebe90dd0d372551eaa3e83a572b` — `feat(fixed-price): add DDL and models for fixed price sale (M1.T1)`（未含 order.go，见 blocker）。

## Test Commands

| Area | Command | Last Result |
| --- | --- | --- |
| Backend Auction | `cd backend/auction && go test ./...` | `not_run` |
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

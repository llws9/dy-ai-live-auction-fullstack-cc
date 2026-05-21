---
description: "Task list for live auction system implementation"
---

# Tasks: 直播竞拍全栈系统

**Input**: Design documents from `/specs/20260521-live-auction-system/`
**Prerequisites**: plan.md, spec.md, data-model.md, contracts/

**Tests**: 测试任务可选，本项目中标注为建议项。

**Organization**: 任务按用户故事分组，支持独立实现和测试。

## Format: `[ID] [P?] [Story] Description`
- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 所属用户故事（US1-US8）
- 包含精确文件路径

## Path Conventions
- **Backend**: `backend/gateway/`, `backend/product/`, `backend/auction/`
- **Frontend**: `frontend/h5/`, `frontend/admin/`
- **Root**: `docker-compose.yml`, `scripts/`

---

## Phase 1: Setup (项目初始化)

**Purpose**: 创建项目结构和基础配置

- [x] T001 创建后端目录结构 `backend/{gateway,product,auction}/`
- [x] T002 创建前端目录结构 `frontend/{h5,admin}/`
- [x] T003 [P] 初始化 Go 模块 `backend/gateway/go.mod`
- [x] T004 [P] 初始化 Go 模块 `backend/product/go.mod`
- [x] T005 [P] 初始化 Go 模块 `backend/auction/go.mod`
- [x] T006 [P] 初始化前端 H5 项目 `frontend/h5/package.json`
- [x] T007 [P] 初始化前端 Admin 项目 `frontend/admin/package.json`
- [x] T008 创建 Docker Compose 配置 `docker-compose.yml`

---

## Phase 2: Foundational (基础设施)

**Purpose**: 所有用户故事依赖的核心基础设施

**⚠️ CRITICAL**: 此阶段必须完成后才能开始任何用户故事

- [x] T009 创建数据库初始化脚本 `scripts/init.sql`
- [x] T010 [P] 创建 User 模型 `backend/product/model/user.go`
- [x] T011 [P] 创建 Product 模型 `backend/product/model/product.go`
- [x] T012 [P] 创建 AuctionRule 模型 `backend/product/model/auction_rule.go`
- [x] T013 [P] 创建 Auction 模型 `backend/auction/model/auction.go`
- [x] T014 [P] 创建 Bid 模型 `backend/auction/model/bid.go`
- [x] T015 [P] 创建 Order 模型 `backend/product/model/order.go`
- [x] T016 配置数据库连接 `backend/product/dao/db.go`
- [x] T017 [P] 配置 Redis 连接 `backend/auction/dao/redis.go`
- [x] T018 创建 Gateway 路由框架 `backend/gateway/router/router.go`
- [x] T019 [P] 创建限流中间件 `backend/gateway/middleware/ratelimit.go`

**Checkpoint**: 基础设施就绪 - 用户故事实现可以开始

---

## Phase 3: User Story 1 - 竞拍商品发布与管理 (Priority: P1) 🎯 MVP

**Goal**: 主播通过 PC 管理后台发布商品、配置竞拍规则

**Independent Test**: 通过 API 创建商品、配置规则、查询列表来独立测试

### Implementation for User Story 1

- [x] T020 [US1] 创建商品 DAO 层 `backend/product/dao/product.go`
- [x] T021 [US1] 创建竞拍规则 DAO 层 `backend/product/dao/auction_rule.go`
- [x] T022 [US1] 创建商品 Service `backend/product/service/product.go`
- [x] T023 [US1] 创建商品 Handler `backend/product/handler/product.go`
- [x] T024 [US1] 创建规则配置 Handler `backend/product/handler/rule.go`
- [x] T025 [US1] Product 服务主入口 `backend/product/main.go`
- [x] T026 [US1] Gateway 路由配置 - 商品服务 `backend/gateway/router/product.go`
- [x] T027 [P] [US1] Admin 商品列表页 `frontend/admin/src/pages/Product/List.tsx`
- [x] T028 [P] [US1] Admin 商品创建页 `frontend/admin/src/pages/Product/Create.tsx`
- [x] T029 [P] [US1] Admin 规则配置页 `frontend/admin/src/pages/Product/RuleConfig.tsx`
- [x] T030 [US1] Admin API 服务封装 `frontend/admin/src/services/api.ts`

**Checkpoint**: US1 完成 - 商品发布与规则配置可独立测试

---

## Phase 4: User Story 4 - 竞拍状态机管理 (Priority: P1)

**Goal**: 系统管理竞拍全生命周期状态流转

**Independent Test**: 通过模拟不同时间点和操作来测试状态转换

**Note**: 此故事为 US2/US3 的前置依赖

### Implementation for User Story 4

- [x] T031 [US4] 创建状态机定义 `backend/auction/service/state_machine.go`
- [x] T032 [US4] 创建竞拍 DAO 层 `backend/auction/dao/auction.go`
- [x] T033 [US4] 创建竞拍 Service `backend/auction/service/auction.go`
- [x] T034 [US4] 创建竞拍 Handler `backend/auction/handler/auction.go`
- [x] T035 [US4] 状态转换定时任务 `backend/auction/service/scheduler.go`

**Checkpoint**: US4 完成 - 竞拍状态机可独立测试

---

## Phase 5: User Story 2 - 实时出价 (Priority: P1)

**Goal**: 用户参与竞拍，系统校验规则、更新价格、广播通知

**Independent Test**: 通过模拟多用户并发出价来测试分布式锁和幂等性

### Implementation for User Story 2

- [x] T036 [US2] 创建 Redis 分布式锁 `backend/auction/lock/redis_lock.go`
- [x] T037 [US2] 创建出价 DAO 层 `backend/auction/dao/bid.go`
- [x] T038 [US2] 创建出价 Service `backend/auction/service/bid.go`
- [x] T039 [US2] 创建出价 Handler `backend/auction/handler/bid.go`
- [x] T040 [US2] Auction 服务主入口 `backend/auction/main.go`
- [x] T041 [US2] Gateway 路由配置 - 竞拍服务 `backend/gateway/router/auction.go`
- [x] T042 [P] [US2] H5 竞拍详情页 `frontend/h5/src/pages/Auction/index.tsx`
- [x] T043 [P] [US2] H5 出价面板组件 `frontend/h5/src/components/BidButton/index.tsx`
- [x] T044 [US2] H5 价格展示组件 `frontend/h5/src/components/PriceDisplay/index.tsx`

**Checkpoint**: US2 完成 - 实时出价可独立测试

---

## Phase 6: User Story 3 - 自动延时机制 (Priority: P1)

**Goal**: 竞拍结束前出价自动延时，但有最大上限

**Independent Test**: 通过模拟倒计时结束前出价来测试延时触发和上限控制

### Implementation for User Story 3

- [x] T045 [US3] 创建延时检查逻辑 `backend/auction/service/delay.go`
- [x] T046 [US3] 集成延时到出价流程 `backend/auction/service/bid.go` (修改 T038)
- [x] T047 [US3] 延时通知消息类型 `backend/auction/websocket/message.go`

**Checkpoint**: US3 完成 - 自动延时可独立测试

---

## Phase 7: User Story 5 - WebSocket 实时通信 (Priority: P1)

**Goal**: 建立 WebSocket 长连接，实现房间隔离、消息推送、断线重连

**Independent Test**: 通过建立多个 WebSocket 连接测试房间隔离和消息广播

### Implementation for User Story 5

- [x] T048 [US5] 创建 WebSocket Hub `backend/auction/websocket/hub.go`
- [x] T049 [US5] 创建房间管理 `backend/auction/websocket/room.go`
- [x] T050 [US5] 创建客户端连接管理 `backend/auction/websocket/client.go`
- [x] T051 [US5] 创建消息类型定义 `backend/auction/websocket/message.go`
- [x] T052 [US5] WebSocket Handler `backend/auction/handler/ws.go`
- [x] T053 [P] [US5] H5 WebSocket 服务封装 `frontend/h5/src/services/websocket.ts`
- [x] T054 [P] [US5] H5 WebSocket Hook `frontend/h5/src/hooks/useWebSocket.ts`
- [x] T055 [US5] H5 竞拍状态管理 `frontend/h5/src/store/auctionContext.tsx`

**Checkpoint**: US5 完成 - WebSocket 实时通信可独立测试

---

## Phase 8: User Story 6 - 倒计时毫秒级精度 (Priority: P2)

**Goal**: 确保所有用户看到的倒计时精确到毫秒，误差 < 100ms

**Independent Test**: 通过多个客户端对比倒计时来测试同步精度

### Implementation for User Story 6

- [x] T056 [US6] 创建时间同步机制 `backend/auction/websocket/time_sync.go`
- [x] T057 [US6] H5 倒计时 Hook `frontend/h5/src/hooks/useCountdown.ts`

**Checkpoint**: US6 完成 - 倒计时精度可独立测试

---

## Phase 9: User Story 7 - 防抖节流 (Priority: P2)

**Goal**: 实现出价按钮防抖和 WebSocket 消息节流

**Independent Test**: 通过快速点击出价按钮来测试防抖效果

### Implementation for User Story 7

- [x] T058 [US7] 创建消息节流控制 `backend/auction/service/throttle.go`
- [x] T059 [US7] H5 防抖 Hook `frontend/h5/src/hooks/useDebounce.ts`
- [x] T060 [US7] 修改出价按钮集成防抖 `frontend/h5/src/components/BidButton/index.tsx` (修改 T043)

**Checkpoint**: US7 完成 - 防抖节流可独立测试

---

## Phase 10: User Story 8 - 用户查看竞拍结果与历史 (Priority: P3)

**Goal**: 用户查看竞拍结果、模拟支付、历史记录

**Independent Test**: 通过 API 调用来独立测试

### Implementation for User Story 8

- [x] T061 [US8] 创建订单 DAO 层 `backend/product/dao/order.go`
- [x] T062 [US8] 创建订单 Service `backend/product/service/order.go`
- [x] T063 [US8] 创建订单 Handler `backend/product/handler/order.go`
- [x] T064 [P] [US8] H5 竞拍结果页 `frontend/h5/src/pages/Result/index.tsx`
- [x] T065 [P] [US8] H5 历史记录页 `frontend/h5/src/pages/History/index.tsx`

**Checkpoint**: US8 完成 - 竞拍结果与历史可独立测试

---

## Phase 11: Polish & Cross-Cutting Concerns

**Purpose**: 跨故事的改进和优化

- [x] T066 [P] Gateway 服务主入口 `backend/gateway/main.go`
- [x] T067 [P] H5 首页商品列表 `frontend/h5/src/pages/Home/index.tsx`
- [x] T068 [P] H5 直播画面组件 `frontend/h5/src/pages/Auction/LiveVideo.tsx`
- [x] T069 [P] H5 实时排名组件 `frontend/h5/src/pages/Auction/Ranking.tsx`
- [x] T070 [P] H5 倒计时组件 `frontend/h5/src/pages/Auction/Countdown.tsx`
- [x] T071 Admin 竞拍列表页 `frontend/admin/src/pages/Auction/List.tsx`
- [x] T072 Admin 竞拍详情页 `frontend/admin/src/pages/Auction/Detail.tsx`
- [x] T073 Admin 订单列表页 `frontend/admin/src/pages/Order/List.tsx`
- [x] T074 快速启动验证 - 按 quickstart.md 验证

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖 - 可立即开始
- **Foundational (Phase 2)**: 依赖 Setup 完成 - **阻塞所有用户故事**
- **US4 (Phase 4)**: 依赖 Foundational - 为 US2/US3 提供状态机
- **US2 (Phase 5)**: 依赖 US4（状态机）
- **US3 (Phase 6)**: 依赖 US2（出价流程）
- **US5 (Phase 7)**: 可与 US2/US3 并行
- **US6/US7 (Phase 8-9)**: 依赖 US5（WebSocket）
- **US8 (Phase 10)**: 可独立开始
- **Polish (Phase 11)**: 依赖所有用户故事完成

### User Story Dependencies

```
Phase 2 (Foundational)
    │
    ├──▶ US1 (商品发布) ────────────────────────────────────── 独立
    │
    ├──▶ US4 (状态机) ──▶ US2 (出价) ──▶ US3 (延时)
    │                           │
    │                           └──▶ US5 (WebSocket) ──▶ US6 (倒计时)
    │                                    │
    │                                    └──▶ US7 (防抖节流)
    │
    └──▶ US8 (结果历史) ───────────────────────────────────── 独立
```

### Parallel Opportunities

**Setup 阶段可并行**:
- T003, T004, T005 (Go 模块初始化)
- T006, T007 (前端项目初始化)

**Foundational 阶段可并行**:
- T010-T015 (所有模型创建)
- T017 (Redis 配置)

**用户故事可并行**:
- US1 与 US4 可同时开始
- US5 可与 US2 并行开始
- US8 可在任何时间开始

---

## Implementation Strategy

### MVP First (仅 User Story 1 + 4 + 2)

1. 完成 Phase 1: Setup
2. 完成 Phase 2: Foundational
3. 完成 Phase 3: US1 商品发布
4. 完成 Phase 4: US4 状态机
5. 完成 Phase 5: US2 出价
6. **停止并验证**: 独立测试核心流程
7. 可演示/部署

### Incremental Delivery

1. Setup + Foundational → 基础就绪
2. US1 商品发布 → 测试 → 部署
3. US4 状态机 + US2 出价 → 测试 → 部署（MVP！）
4. US3 延时 + US5 WebSocket → 测试 → 部署
5. US6-US8 体验优化 → 测试 → 部署

---

## Summary

| Phase | 任务数 | 描述 |
|-------|--------|------|
| Phase 1: Setup | 8 | 项目初始化 |
| Phase 2: Foundational | 11 | 基础设施 |
| Phase 3: US1 | 11 | 商品发布管理 |
| Phase 4: US4 | 5 | 状态机 |
| Phase 5: US2 | 9 | 实时出价 |
| Phase 6: US3 | 3 | 自动延时 |
| Phase 7: US5 | 8 | WebSocket |
| Phase 8: US6 | 2 | 倒计时精度 |
| Phase 9: US7 | 3 | 防抖节流 |
| Phase 10: US8 | 5 | 结果历史 |
| Phase 11: Polish | 9 | 收尾优化 |
| **Total** | **74** | |

---

## Notes

- [P] 任务 = 不同文件，无依赖
- [Story] 标签将任务映射到用户故事
- 每个用户故事应独立可完成和测试
- 在每个检查点独立验证故事
- 提交时机：每个任务或逻辑组完成后

# Tasks: 直播竞拍系统核心功能完善

**Input**: 设计文档来自 `/specs/20260522-live-auction-core/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: 本项目不包含测试任务（规格说明中未明确要求TDD）

**Organization**: 任务按用户故事分组，支持独立实施和测试

## 格式说明: `[ID] [P?] [Story] 描述`
- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 任务所属用户故事（US1, US2, US3, US4）
- 描述中包含精确的文件路径

---

## Phase 1: Setup（共享基础设施）

**目的**: 项目初始化和基础结构准备

- [ ] T001 创建数据库迁移文件，新增 orders 表和索引
- [ ] T002 [P] 扩展 Product 模型，添加 Status 字段
- [ ] T003 [P] 扩展 User 模型，添加 Role 字段
- [ ] T004 [P] 创建前端类型定义文件

**Checkpoint**: 数据库结构就绪，可开始用户故事实施

---

## Phase 2: Foundational（阻塞性前置条件）

**目的**: 所有用户故事依赖的核心基础设施

**⚠️ 关键**: 此阶段必须完成，否则无法开始任何用户故事

- [ ] T005 实现 Order 模型和 DAO 层
- [ ] T006 [P] 创建 ConnectionState 和 SyncState 的 Redis 存储管理器
- [ ] T007 [P] 扩展 WebSocket 消息类型定义（rank_update, sync_response, time_sync）
- [ ] T008 [P] 实现消息节流器 Throttler

**Checkpoint**: 基础设施就绪 - 用户故事实施可并行开始

---

## Phase 3: User Story 1 - 实时排名同步 (Priority: P1) 🎯 MVP核心

**目标**: 出价成功后实时广播排名更新到所有参与用户

**独立测试标准**: 模拟多用户同时出价，验证所有客户端在200ms内收到排名更新

### Implementation for User Story 1

- [ ] T009 [US1] 修改 backend/auction/service/bid.go，添加 broadcastRanking 方法
- [ ] T010 [US1] 修改 backend/auction/websocket/message.go，新增 rank_update 消息类型
- [ ] T011 [US1] 修改 backend/auction/service/bid.go#PlaceBid，出价成功后调用 broadcastRanking
- [ ] T012 [P] [US1] 修改 frontend/h5/src/services/websocket.ts，处理 rank_update 消息
- [ ] T013 [P] [US1] 修改 frontend/h5/src/pages/Auction/Ranking.tsx，实现实时排名展示
- [ ] T014 [US1] 修改 frontend/h5/src/pages/Auction/index.tsx，集成排名更新逻辑

**Checkpoint**: 用户故事1完成，实时排名同步功能可独立测试

---

## Phase 4: User Story 2 - 断线重连机制 (Priority: P1)

**目标**: WebSocket自动重连与状态同步

**独立测试标准**: 手动关闭WebSocket连接，验证系统自动重连并恢复状态

### Implementation for User Story 2

- [ ] T015 [US2] 新增 backend/auction/websocket/state_sync.go，实现状态同步管理器
- [ ] T016 [US2] 修改 backend/auction/websocket/client.go#ReadPump，完善心跳检测
- [ ] T017 [US2] 修改 backend/auction/websocket/client.go，添加 handleReconnect 方法
- [ ] T018 [P] [US2] 新增 frontend/h5/src/hooks/useReconnect.ts，实现指数退避重连逻辑
- [ ] T019 [P] [US2] 修改 frontend/h5/src/services/websocket.ts，集成自动重连逻辑
- [ ] T020 [US2] 修改 frontend/h5/src/pages/Auction/index.tsx，重连后同步最新状态

**Checkpoint**: 用户故事2完成，断线重连功能可独立测试

---

## Phase 5: User Story 3 - PC管理后台 (Priority: P1)

**目标**: 商品、竞拍、订单管理功能

**独立测试标准**: 登录管理后台，执行商品创建、查看竞拍列表、管理订单状态

### Implementation for User Story 3

#### 后端 API 实现

- [ ] T021 [P] [US3] 新增 backend/product/service/product.go，实现商品服务层
- [ ] T022 [P] [US3] 新增 backend/product/handler/product.go，实现商品管理API
- [ ] T023 [P] [US3] 新增 backend/product/service/order.go，实现订单服务层
- [ ] T024 [P] [US3] 新增 backend/product/handler/order.go，实现订单管理API
- [ ] T025 [US3] 修改 backend/auction/handler/auction.go，新增竞拍管理API
- [ ] T026 [US3] 配置 JWT + RBAC 权限中间件

#### 前端页面实现

- [ ] T027 [P] [US3] 新增 frontend/admin/src/pages/Product/List.tsx，商品列表页
- [ ] T028 [P] [US3] 新增 frontend/admin/src/pages/Product/Create.tsx，商品创建页
- [ ] T029 [P] [US3] 新增 frontend/admin/src/pages/Product/Edit.tsx，商品编辑页
- [ ] T030 [P] [US3] 新增 frontend/admin/src/pages/Auction/List.tsx，竞拍列表页
- [ ] T031 [P] [US3] 新增 frontend/admin/src/pages/Auction/Detail.tsx，竞拍详情页
- [ ] T032 [P] [US3] 新增 frontend/admin/src/pages/Order/List.tsx，订单列表页
- [ ] T033 [P] [US3] 新增 frontend/admin/src/pages/Order/Detail.tsx，订单详情页
- [ ] T034 [US3] 新增 frontend/admin/src/services/api.ts，管理后台API封装

**Checkpoint**: 用户故事3完成，PC管理后台功能可独立测试

---

## Phase 6: User Story 4 - 体验优化功能 (Priority: P2)

**目标**: 动画效果、倒计时精度、历史记录查询

**独立测试标准**: 在竞拍页面出价观察动画效果、检查倒计时精度、查看历史记录页面

### Implementation for User Story 4

#### 动画效果优化

- [ ] T035 [P] [US4] 新增 frontend/h5/src/utils/animations.ts，统一动画效果管理
- [ ] T036 [US4] 修改 frontend/h5/src/components/BidButton/index.tsx，出价成功动画反馈
- [ ] T037 [US4] 修改 frontend/h5/src/components/PriceDisplay/index.tsx，价格变化动画效果

#### 倒计时精度优化

- [ ] T038 [US4] 新增 backend/auction/websocket/time_sync.go，服务端时间同步机制
- [ ] T039 [US4] 修改 backend/auction/websocket/message.go，新增 time_sync 消息类型
- [ ] T040 [P] [US4] 新增 frontend/h5/src/hooks/useServerTime.ts，服务端时间校准Hook
- [ ] T041 [US4] 修改 frontend/h5/src/pages/Auction/Countdown.tsx，毫秒级倒计时显示

#### 历史记录功能

- [ ] T042 [US4] 新增 backend/product/handler/order.go#GetUserHistory，获取用户历史记录API
- [ ] T043 [US4] 新增 backend/product/service/order.go#GetUserHistory，查询用户参与的竞拍历史
- [ ] T044 [P] [US4] 新增 frontend/h5/src/pages/History/index.tsx，用户竞拍历史列表
- [ ] T045 [US4] 修改 frontend/h5/src/services/api.ts，历史记录查询API

**Checkpoint**: 用户故事4完成，体验优化功能可独立测试

---

## Phase 7: Polish & Cross-Cutting Concerns（收尾与跨切面关注点）

**目的**: 影响多个用户故事的改进

- [ ] T046 [P] 更新 API 文档
- [ ] T047 [P] 代码清理和重构
- [ ] T048 性能优化和压力测试
- [ ] T049 [P] 安全加固（输入验证、权限检查）
- [ ] T050 执行 quickstart.md 验证流程

---

## Dependencies & Execution Order（依赖与执行顺序）

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖，立即开始
- **Foundational (Phase 2)**: 依赖 Setup 完成 - **阻塞所有用户故事**
- **User Stories (Phase 3-6)**: 都依赖 Foundational 阶段完成
  - 用户故事可并行执行（如有足够人力）
  - 或按优先级顺序执行（P1 → P1 → P1 → P2）
- **Polish (Phase 7)**: 依赖所有期望的用户故事完成

### User Story Dependencies

- **User Story 1 (P1 - 实时排名同步)**: Foundational 完成后可开始 - 无其他故事依赖
- **User Story 2 (P1 - 断线重连)**: Foundational 完成后可开始 - 无其他故事依赖
- **User Story 3 (P1 - PC管理后台)**: Foundational 完成后可开始 - 无其他故事依赖
- **User Story 4 (P2 - 体验优化)**: Foundational 完成后可开始 - 无其他故事依赖

**所有用户故事均可独立实施和测试**

### Within Each User Story

- 后端任务先于前端任务
- 服务层先于Handler层
- 核心实现先于集成
- 故事完成后才能进入下一个优先级

### Parallel Opportunities

- 所有 Setup 阶段标记 [P] 的任务可并行执行
- 所有 Foundational 阶段标记 [P] 的任务可并行执行
- Foundational 完成后，所有用户故事可并行开始（如有团队容量）
- 每个用户故事内标记 [P] 的任务可并行执行
- 不同用户故事可由不同团队成员并行工作

---

## Parallel Example: User Story 1（并行示例）

```bash
# 同时启动 US1 的所有前端任务:
Task: "修改 frontend/h5/src/services/websocket.ts，处理 rank_update 消息"
Task: "修改 frontend/h5/src/pages/Auction/Ranking.tsx，实现实时排名展示"
```

---

## Implementation Strategy（实施策略）

### MVP First（仅 User Story 1）

1. 完成 Phase 1: Setup
2. 完成 Phase 2: Foundational（**关键 - 阻塞所有故事**）
3. 完成 Phase 3: User Story 1
4. **停止并验证**: 独立测试 User Story 1
5. 如就绪可部署/演示

### Incremental Delivery（增量交付）

1. 完成 Setup + Foundational → 基础就绪
2. 添加 User Story 1 → 独立测试 → 部署/演示（MVP!）
3. 添加 User Story 2 → 独立测试 → 部署/演示
4. 添加 User Story 3 → 独立测试 → 部署/演示
5. 添加 User Story 4 → 独立测试 → 部署/演示
6. 每个故事增加价值而不破坏之前的故事

### Parallel Team Strategy（并行团队策略）

多人团队协作：

1. 团队共同完成 Setup + Foundational
2. Foundational 完成后：
   - 开发者 A: User Story 1（实时排名同步）
   - 开发者 B: User Story 2（断线重连）
   - 开发者 C: User Story 3（PC管理后台）
3. 故事独立完成并集成

---

## Task Summary（任务摘要）

**总任务数**: 50个

**按用户故事统计**:
- Setup: 4个任务
- Foundational: 4个任务
- User Story 1 (实时排名同步): 6个任务
- User Story 2 (断线重连): 6个任务
- User Story 3 (PC管理后台): 14个任务
- User Story 4 (体验优化): 11个任务
- Polish: 5个任务

**并行机会**:
- Setup 阶段: 3个任务可并行
- Foundational 阶段: 3个任务可并行
- User Story 1: 2个任务可并行
- User Story 2: 2个任务可并行
- User Story 3: 10个任务可并行
- User Story 4: 4个任务可并行

**建议 MVP 范围**: Setup + Foundational + User Story 1（实时排名同步）

---

## Notes

- [P] 任务 = 不同文件，无依赖
- [Story] 标签映射任务到具体用户故事，便于追溯
- 每个用户故事应独立完成和测试
- 每个任务或逻辑组完成后提交
- 可在任何 checkpoint 停止验证故事独立性
- 避免：模糊任务、同文件冲突、破坏独立性的跨故事依赖

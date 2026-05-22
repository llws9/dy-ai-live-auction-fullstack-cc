---
description: "Task list for MVP notification and reporting features"
---

# Tasks: MVP阶段功能完善

**Input**: Design documents from `/specs/20260522-mvp-notification-report/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: 包含单元测试和E2E测试任务（FR-011至FR-013要求）

**Organization**: 任务按用户故事分组，支持独立实现和测试

## Format: `[ID] [P?] [Story] Description`
- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 所属用户故事（US1, US2, US3, US4）
- 包含精确文件路径

## Path Conventions
- **Backend**: `backend/auction/`, `backend/product/`, `backend/gateway/`
- **Frontend H5**: `frontend/h5/src/`
- **Frontend Admin**: `frontend/admin/src/`

---

## Phase 1: Setup (共享基础设施)

**Purpose**: 项目初始化和Swagger工具配置

- [x] T001 [P] [Setup] 安装Swagger CLI: `go install github.com/swaggo/swag/cmd/swag@latest`
- [x] T002 [P] [Setup] 安装前端依赖: `cd frontend/admin && npm install recharts`
- [x] T003 [Setup] 创建notifications表迁移文件 `backend/auction/migration/001_create_notifications.sql`

---

## Phase 2: Foundational (阻塞前置条件)

**Purpose**: 核心基础设施，必须在任何用户故事之前完成

**⚠️ CRITICAL**: 此阶段完成前，不能开始任何用户故事

- [x] T004 [Foundational] 创建Notification模型 `backend/auction/model/notification.go`
- [x] T005 [Foundational] 创建NotificationDAO `backend/auction/dao/notification.go`
- [x] T006 [Foundational] 集成Swagger中间件到Gateway `backend/gateway/main.go`
- [x] T007 [Foundational] 添加Swagger路由 `backend/gateway/router/router.go`

**Checkpoint**: 基础就绪 - 用户故事实现可以开始

---

## Phase 3: User Story 4 - API文档生成 (Priority: P1) 🎯 MVP

**Goal**: 集成Swagger/OpenAPI，自动生成API文档，提升前后端协作效率

**Independent Test**: 访问 `/swagger/index.html` 验证文档渲染

### Implementation for User Story 4

- [x] T008 [P] [US4] 添加Auction Handler Swagger注解 `backend/auction/handler/auction.go`
- [x] T009 [P] [US4] 添加Bid Handler Swagger注解 `backend/auction/handler/bid.go`
- [x] T010 [P] [US4] 添加Order Handler Swagger注解 `backend/product/handler/order.go`
- [x] T011 [P] [US4] 添加Product Handler Swagger注解 `backend/product/handler/product.go`
- [x] T012 [US4] 生成Swagger文档 `swag init -g gateway/main.go -o ./docs`
- [x] T013 [US4] 验证Swagger UI可访问 `/swagger/index.html`

**Checkpoint**: API文档生成完成，Swagger UI可访问

---

## Phase 4: User Story 1 - 消息通知系统 (Priority: P1) 🎯 MVP

**Goal**: 实现用户实时通知功能，包括出价提醒、中标通知、订单状态变更等消息推送

**Independent Test**: 通过出价测试验证通知发送，通过WebSocket连接测试验证实时推送

### Tests for User Story 1

- [x] T014 [P] [US1] 创建Notification Service单元测试 `backend/auction/service/notification_test.go`

### Implementation for User Story 1

- [x] T015 [US1] 创建NotificationService（含接口定义） `backend/auction/service/notification.go`
- [x] T016 [US1] 创建Notification Handler/API `backend/auction/handler/notification.go`
- [x] T017 [US1] 修改WebSocket message.go添加通知消息类型 `backend/auction/websocket/message.go`
- [x] T018 [US1] 修改BidService.PlaceBid触发出价超越通知 `backend/auction/service/bid.go`
- [x] T019 [US1] 修改AuctionService.EndAuction触发中标/未中标通知 `backend/auction/service/auction.go`
- [x] T020 [US1] 修改OrderService Mock触发订单通知 `backend/product/service/order.go`
- [x] T021 [P] [US1] 创建前端通知组件 `frontend/h5/src/components/Notification/index.tsx`
- [x] T022 [P] [US1] 创建通知Hook `frontend/h5/src/hooks/useNotification.ts`
- [x] T023 [US1] 修改WebSocket服务处理通知消息 `frontend/h5/src/services/websocket.ts`
- [x] T024 [US1] 添加Notification Handler Swagger注解 `backend/auction/handler/notification.go`

**Checkpoint**: 消息通知系统功能完整，可独立测试

---

## Phase 5: User Story 3 - 测试覆盖增强 (Priority: P1)

**Goal**: 增强测试覆盖率，核心业务逻辑单元测试覆盖率>80%

**Independent Test**: 运行 `go test ./...` 验证单元测试，运行 `npx playwright test` 验证E2E测试

### Tests for User Story 3

- [x] T025 [P] [US3] 创建Auction Service单元测试 `backend/auction/service/auction_test.go`
- [x] T026 [P] [US3] 创建Bid Service单元测试 `backend/auction/service/bid_test.go`
- [x] T027 [P] [US3] 增强E2E测试场景 `frontend/h5/e2e/auction.spec.ts`

**Checkpoint**: 测试覆盖率达标，所有测试通过

---

## Phase 6: User Story 2 - 数据分析报表 (Priority: P2)

**Goal**: 为管理后台提供数据统计和可视化报表，包括竞拍统计、收入分析、用户活跃度等

**Independent Test**: 通过API调用验证统计数据正确性，通过管理后台验证图表渲染

### Implementation for User Story 2

- [x] T028 [P] [US2] 创建Statistics Service `backend/product/service/statistics.go`
- [x] T029 [P] [US2] 创建Statistics DAO `backend/product/dao/statistics.go`
- [x] T030 [US2] 创建Statistics Handler/API `backend/product/handler/statistics.go`
- [x] T031 [P] [US2] 创建Admin数据大屏页面 `frontend/admin/src/pages/Dashboard/index.tsx`
- [x] T032 [P] [US2] 创建统计报表页面 `frontend/admin/src/pages/Statistics/index.tsx`
- [x] T033 [P] [US2] 创建图表组件 `frontend/admin/src/components/Charts/index.tsx`
- [x] T034 [US2] 添加Statistics Handler Swagger注解 `backend/product/handler/statistics.go`

**Checkpoint**: 数据分析报表功能完整，可独立测试

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: 跨切面改进和最终验证

- [x] T035 [Polish] 更新API文档（重新生成Swagger） `swag init`
- [x] T036 [Polish] 验证所有测试通过 `go test ./... && npx playwright test`
- [x] T037 [Polish] 运行quickstart.md验证场景
- [x] T038 [Polish] 代码清理和注释完善

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖 - 可立即开始
- **Foundational (Phase 2)**: 依赖Setup完成 - 阻塞所有用户故事
- **User Story 4 (Phase 3)**: 依赖Foundational完成 - API文档优先
- **User Story 1 (Phase 4)**: 依赖Foundational完成 - 可与US4并行
- **User Story 3 (Phase 5)**: 依赖US1完成 - 测试依赖通知系统
- **User Story 2 (Phase 6)**: 依赖Foundational完成 - 可与其他US并行
- **Polish (Phase 7)**: 依赖所有用户故事完成

### User Story Dependencies

- **User Story 4 (P1)**: 可在Foundational后开始 - 无其他依赖
- **User Story 1 (P1)**: 可在Foundational后开始 - 无其他依赖
- **User Story 3 (P1)**: 依赖US1完成（测试通知功能）
- **User Story 2 (P2)**: 可在Foundational后开始 - 无其他依赖

### Within Each User Story

- Tests before implementation (TDD)
- Models before services
- Services before handlers
- Core before integration

### Parallel Opportunities

- T001, T002 可并行执行
- T008, T009, T010, T011 可并行执行（不同Handler文件）
- T021, T022 可并行执行（不同前端文件）
- T025, T026, T027 可并行执行（不同测试文件）
- T028, T029 可并行执行（不同后端文件）
- T031, T032, T033 可并行执行（不同前端文件）

---

## Parallel Example: User Story 1

```bash
# 并行启动前端通知组件和Hook:
Task: "创建前端通知组件 frontend/h5/src/components/Notification/index.tsx"
Task: "创建通知Hook frontend/h5/src/hooks/useNotification.ts"
```

---

## Implementation Strategy

### MVP First (User Story 4 + 1 + 3)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL)
3. Complete Phase 3: User Story 4 (API文档)
4. Complete Phase 4: User Story 1 (消息通知)
5. Complete Phase 5: User Story 3 (测试增强)
6. **STOP and VALIDATE**: 测试所有功能独立运行
7. Deploy MVP

### Incremental Delivery

1. Setup + Foundational → 基础就绪
2. Add US4 → API文档可访问 → Demo
3. Add US1 → 通知功能完整 → Demo
4. Add US3 → 测试覆盖达标 → Deploy
5. Add US2 → 数据报表完整 → Deploy

---

## Notes

- [P] 任务 = 不同文件，无依赖，可并行
- [Story] 标签映射任务到用户故事
- 每个用户故事应独立可测试
- 每个任务或逻辑组完成后提交
- 在任何checkpoint可停止验证
- **强制要求**: 每次新增/修改业务功能同步更新API文档

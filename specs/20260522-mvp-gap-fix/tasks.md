---
description: "Task list for MVP gap fixes"
---

# Tasks: MVP版本不符合预期项修复

**Input**: Design documents from `/specs/20260522-mvp-gap-fix/`
**Prerequisites**: spec.md

**Organization**: 任务按用户故事分组,支持独立实现和测试

## Format: `[ID] [P?] [Story] Description`
- **[P]**: 可并行执行(不同文件,无依赖)
- **[Story]**: 所属用户故事(US1, US2, US3, US4, US5)
- 包含精确文件路径

---

## Phase 1: Setup

**Purpose**: 安装依赖和准备开发环境

- [x] T001 [P] [Setup] 安装前端依赖: `cd frontend/admin && npm install recharts`
- [x] T002 [P] [Setup] 验证后端服务可编译: `cd backend && go build ./...`

---

## Phase 2: User Story 1 - 管理后台核心功能补齐 (Priority: P0)

**Goal**: 补齐商品编辑、竞拍详情、竞拍取消功能

**Independent Test**: 可通过管理后台UI独立测试每个功能

### Backend Changes

- [x] T003 [P] [US1] 添加出价记录查询API `backend/auction/handler/auction.go#GetAuctionBids`
- [x] T004 [P] [US1] 创建出价记录Service方法 `backend/auction/service/auction.go#GetAuctionBids`

### Frontend Changes

- [x] T005 [P] [US1] 创建商品编辑页面 `frontend/admin/src/pages/Product/Edit.tsx`
- [x] T006 [US1] 修改商品列表页添加编辑按钮 `frontend/admin/src/pages/Product/List.tsx`
- [x] T007 [P] [US1] 实现竞拍详情页面 `frontend/admin/src/pages/Auction/Detail.tsx`
- [x] T008 [P] [US1] 创建竞拍信息组件 `frontend/admin/src/components/AuctionInfo/index.tsx`
- [x] T009 [P] [US1] 创建出价记录组件 `frontend/admin/src/components/BidHistory/index.tsx`
- [x] T010 [US1] 修改竞拍列表页添加取消功能 `frontend/admin/src/pages/Auction/List.tsx`
- [x] T011 [US1] 添加API服务方法 `frontend/admin/src/services/api.ts`

**Checkpoint**: 管理后台核心功能完整,可独立测试

---

## Phase 3: User Story 2 - 数据统计报表系统 (Priority: P0)

**Goal**: 实现数据大屏、统计页面和图表组件

**Independent Test**: 可通过管理后台访问统计页面验证

### Frontend Changes

- [x] T012 [P] [US2] 创建数据大屏页面 `frontend/admin/src/pages/Dashboard/index.tsx`
- [x] T013 [P] [US2] 创建统计报表主页 `frontend/admin/src/pages/Statistics/Index.tsx`
- [x] T014 [P] [US2] 创建竞拍统计页 `frontend/admin/src/pages/Statistics/Auction.tsx`
- [x] T015 [P] [US2] 创建收入统计页 `frontend/admin/src/pages/Statistics/Revenue.tsx`
- [x] T016 [P] [US2] 创建用户统计页 `frontend/admin/src/pages/Statistics/User.tsx`
- [x] T017 [P] [US2] 创建折线图组件 `frontend/admin/src/components/Charts/LineChart.tsx`
- [x] T018 [P] [US2] 创建柱状图组件 `frontend/admin/src/components/Charts/BarChart.tsx`
- [x] T019 [P] [US2] 创建饼图组件 `frontend/admin/src/components/Charts/PieChart.tsx`
- [x] T020 [P] [US2] 创建统计卡片组件 `frontend/admin/src/components/Charts/StatCard.tsx`
- [x] T021 [US2] 更新路由配置 `frontend/admin/src/App.tsx`

**Checkpoint**: 数据统计报表系统完整,图表正常渲染

---

## Phase 4: User Story 3 - 后端测试覆盖提升 (Priority: P1)

**Goal**: Gateway 0%→60%, Product-service 20%→80%

**Independent Test**: 运行 `go test ./...` 验证覆盖率

### Gateway Tests

- [x] T022 [P] [US3] 创建认证Handler测试 `backend/gateway/handler/auth_test.go`
- [x] T023 [P] [US3] 创建JWT中间件测试 `backend/gateway/middleware/jwt_test.go`
- [x] T024 [P] [US3] 创建限流中间件测试 `backend/gateway/middleware/ratelimit_test.go`
- [x] T025 [P] [US3] 创建RBAC中间件测试 `backend/gateway/middleware/rbac_test.go`

### Product-service Tests

- [x] T026 [P] [US3] 创建商品服务测试 `backend/product/service/product_test.go`
- [x] T027 [P] [US3] 创建统计服务测试 `backend/product/service/statistics_test.go`
- [x] T028 [P] [US3] 创建商品Handler测试 `backend/product/handler/product_test.go`

**Checkpoint**: 测试覆盖率达标,所有测试通过

---

## Phase 5: User Story 4 - WebSocket消息节流 (Priority: P1)

**Goal**: 实现前端消息节流机制

**Independent Test**: 高频消息场景下前端不卡顿

### Frontend Changes

- [x] T029 [US4] 创建节流工具函数 `frontend/h5/src/utils/throttle.ts`
- [x] T030 [US4] 修改WebSocket服务添加节流 `frontend/h5/src/services/websocket.ts`

**Checkpoint**: WebSocket消息处理流畅,无卡顿

---

## Phase 6: User Story 5 - 用户中心页面 (Priority: P2)

**Goal**: 实现用户中心页面

**Independent Test**: 用户可访问个人中心查看信息

### Frontend Changes

- [x] T031 [P] [US5] 创建用户中心页面 `frontend/h5/src/pages/User/Index.tsx`
- [x] T032 [P] [US5] 创建用户信息组件 `frontend/h5/src/components/UserInfo/index.tsx`
- [x] T033 [P] [US5] 创建用户统计组件 `frontend/h5/src/components/UserStats/index.tsx`

**Checkpoint**: 用户中心页面完成,可独立访问

---

## Phase 7: Polish & Validation

**Purpose**: 最终验证和清理

- [x] T034 [Polish] 运行所有测试验证 `go test ./... && npm test`
- [x] T035 [Polish] 验证前端构建 `cd frontend/admin && npm run build`
- [x] T036 [Polish] 代码清理和注释完善

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖 - 可立即开始
- **User Story 1 (Phase 2)**: 依赖Setup完成
- **User Story 2 (Phase 3)**: 依赖Setup完成 - 可与US1并行
- **User Story 3 (Phase 4)**: 可与其他US并行
- **User Story 4 (Phase 5)**: 可与其他US并行
- **User Story 5 (Phase 6)**: 可与其他US并行
- **Polish (Phase 7)**: 依赖所有US完成

### Parallel Opportunities

- T001, T002 可并行
- T003, T004, T005, T007, T008, T009 可并行(不同文件)
- T012-T020 可并行(不同文件)
- T022-T028 可并行(不同测试文件)
- T031-T033 可并行

---

## Implementation Strategy

### 优先级执行顺序

1. **P0 核心功能 (第一周)**:
   - Phase 1: Setup
   - Phase 2: US1 管理后台核心功能
   - Phase 3: US2 数据统计报表

2. **P1 质量保障 (第二周)**:
   - Phase 4: US3 测试覆盖
   - Phase 5: US4 消息节流

3. **P2 体验优化 (第三周)**:
   - Phase 6: US5 用户中心
   - Phase 7: Polish

---

## Notes

- [P] 任务 = 不同文件,无依赖,可并行
- 每个用户故事应独立可测试
- 测试任务优先于实现任务(TDD)
- 每个Phase完成后验证

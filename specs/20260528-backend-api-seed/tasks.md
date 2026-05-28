---
description: "Task list for backend API supplement and test data generation"
---

# Tasks: 后端API补充与测试数据生成

**Input**: Design documents from `/specs/20260528-backend-api-seed/`
**Prerequisites**: plan.md, spec.md, data-model.md, contracts/api-contracts.md

**Tests**: 本任务列表不包含测试任务（spec未明确要求TDD）

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)

## Path Conventions
- **Web app**: `backend/gateway/`, `backend/product/`, `backend/seed/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 确认现有服务状态和依赖

- [x] T001 确认Gateway服务运行正常 (curl localhost:8080/health)
- [x] T002 确认Product Service运行正常
- [x] T003 确认数据库连接配置正确 (检查backend/.env)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 核心基础设施 - MUST complete before ANY user story

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 [P] 创建Category模型 - backend/product/model/category.go
- [x] T005 [P] 修改Product模型添加CategoryID字段 - backend/product/model/product.go
- [x] T006 Product Service AutoMigrate添加Category表 - backend/product/main.go

**Checkpoint**: Foundation ready - Category表已创建，Product表已添加category_id字段

---

## Phase 3: User Story 1 - Gateway路由补充 (Priority: P1) 🎯 MVP

**Goal**: 补充Gateway缺失路由，使前端能正常调用订单发货、历史、直播间接口

**Independent Test**: Swagger测试新增接口，验证路由注册和权限控制

### Implementation for User Story 1

- [x] T007 [P] [US1] Gateway注册订单发货路由 - backend/gateway/router/router.go
- [x] T008 [P] [US1] Gateway注册订单历史路由 - backend/gateway/router/router.go
- [x] T009 [P] [US1] Gateway注册管理端直播间列表路由 - backend/gateway/router/router.go
- [x] T010 [P] [US1] Gateway注册直播间详情路由 - backend/gateway/router/router.go
- [x] T011 [US1] Product Service实现ListAdminLiveStreams - backend/product/handler/live_stream.go
- [x] T012 [US1] Product Service实现GetLiveStreamDetail - backend/product/handler/live_stream.go
- [x] T013 [US1] Product Service补充LiveStream DAO方法 - backend/product/dao/live_stream.go

**Checkpoint**: User Story 1 complete - 4个新路由已注册，Handler已实现

---

## Phase 4: User Story 2 - 动态商品类别系统 (Priority: P2)

**Goal**: 实现Category CRUD API，支持动态类别管理

**Independent Test**: API创建、查询、更新、删除类别，验证删除保护逻辑

### Implementation for User Story 2

- [x] T014 [US2] 创建CategoryDAO - backend/product/dao/category.go
- [x] T015 [US2] 创建CategoryService - backend/product/service/category.go
- [x] T016 [US2] 创建CategoryHandler (List/Create/Update/Delete) - backend/product/handler/category.go
- [x] T017 [US2] Product Service注册Category路由 - backend/product/main.go
- [x] T018 [P] [US2] Gateway注册类别列表路由 - backend/gateway/router/router.go
- [x] T019 [P] [US2] Gateway注册类别创建路由(需管理员权限) - backend/gateway/router/router.go
- [x] T020 [P] [US2] Gateway注册类别更新路由(需管理员权限) - backend/gateway/router/router.go
- [x] T021 [P] [US2] Gateway注册类别删除路由(需管理员权限) - backend/gateway/router/router.go
- [x] T022 [US2] CategoryService实现删除保护逻辑(检查商品关联) - backend/product/service/category.go

**Checkpoint**: User Story 2 complete - 类别管理功能完整可用

---

## Phase 5: User Story 3 - 测试数据生成脚本 (Priority: P3)

**Goal**: 编写Go Seed脚本生成多样性测试数据

**Independent Test**: 执行Seed脚本后检查数据库，验证数据数量和关联关系

### Implementation for User Story 3

- [x] T023 [US3] 创建Seed脚本目录结构 - backend/seed/
- [x] T024 [US3] 创建Seed配置文件(数据数量和分布) - backend/seed/config.go
- [x] T025 [US3] 实现GenerateCategories函数 - backend/seed/generators.go
- [x] T026 [US3] 实现GenerateUsers函数 - backend/seed/generators.go
- [x] T027 [US3] 实现GenerateProducts函数 - backend/seed/generators.go
- [x] T028 [US3] 实现GenerateLiveStreams函数 - backend/seed/generators.go
- [x] T029 [US3] 实现GenerateAuctionRules函数 - backend/seed/generators.go
- [x] T030 [US3] 实现GenerateAuctions函数 - backend/seed/generators.go
- [x] T031 [US3] 实现GenerateBids函数 - backend/seed/generators.go
- [x] T032 [US3] 实现GenerateOrders函数 - backend/seed/generators.go
- [x] T033 [US3] 实现GenerateNotifications函数 - backend/seed/generators.go
- [x] T034 [US3] 创建Seed主程序入口(协调生成顺序) - backend/seed/main.go

**Checkpoint**: User Story 3 complete - Seed脚本可生成约300条多样性数据

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 改进和验证

- [x] T035 重新生成Swagger文档 - backend/*/docs/swagger.json
- [x] T036 运行quickstart.md验证流程
- [x] T037 前端Admin验证商品列表显示类别 (Note: 前端表格有"类别"列，后端API返回category_id，但响应格式不匹配 - pre-existing issue)
- [ ] T038 前端Admin验证订单发货功能
- [ ] T039 前端H5验证首页竞拍列表加载真实数据
- [ ] T038 前端Admin验证订单发货功能
- [ ] T039 前端H5验证首页竞拍列表加载真实数据

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖 - 可立即开始
- **Foundational (Phase 2)**: 依赖Setup完成 - BLOCKS所有User Stories
- **User Stories (Phase 3-5)**: 都依赖Foundational完成
- **Polish (Phase 6)**: 依赖所有User Stories完成

### User Story Dependencies

- **US1 (P1)**: 可在Foundational完成后开始 - 无其他Story依赖
- **US2 (P2)**: 可在Foundational完成后开始 - 无其他Story依赖(Category模型已在Foundational创建)
- **US3 (P3)**: 可在Foundational完成后开始 - 需要Category数据，但可使用空表生成

### Parallel Opportunities

- T004, T005 可并行（不同文件）
- T007-T010 可并行（同文件但不同路由行）
- T018-T021 可并行
- Seed生成函数T025-T033可并行开发（不同函数）

---

## Parallel Example: User Story 1

```bash
# Launch Gateway路由注册任务并行:
Task: "Gateway注册订单发货路由"
Task: "Gateway注册订单历史路由"
Task: "Gateway注册管理端直播间列表路由"
Task: "Gateway注册直播间详情路由"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1: Setup (确认服务状态)
2. Phase 2: Foundational (Category模型)
3. Phase 3: User Story 1 (Gateway路由)
4. **STOP and VALIDATE**: Swagger测试新接口
5. 部署/演示

### Incremental Delivery

1. Setup + Foundational → 基础就绪
2. User Story 1 → 测试 → 部署 (MVP!)
3. User Story 2 → 测试 → 部署
4. User Story 3 → 测试 → 部署
5. Polish → 最终验证

---

## Notes

- [P] tasks = 不同文件，无依赖
- [Story] label = 任务所属User Story
- 每个User Story独立可完成和测试
- 完成每个Checkpoint后验证
- 无测试任务（spec未要求TDD）
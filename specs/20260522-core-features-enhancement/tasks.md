# Tasks: 直播竞拍系统核心功能完善

**Input**: Design documents from `/specs/20260522-core-features-enhancement/`
**Prerequisites**: plan.md, spec.md, data-model.md, quickstart.md

**Tests**: 本次变更需要测试验证，包含测试任务。

**Organization**: 任务按用户故事分组，支持独立实现和测试。

## Format: `[ID] [P?] [Story] Description`
- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 所属用户故事（US1-US4）

## Path Conventions
- **Web app**: `backend/auction/`, `backend/product/`, `backend/gateway/`

---

## Phase 1: Setup (Redis环境准备)

**Purpose**: 启动Redis服务，验证连接

- [x] T001 启动Redis容器: `docker-compose up -d redis`
- [x] T002 验证Redis连接: `docker-compose exec redis redis-cli ping`
- [x] T003 [P] 检查auction服务Redis配置: `backend/auction/dao/redis.go`

**Checkpoint**: Redis环境就绪

---

## Phase 2: Foundational (数据库迁移)

**Purpose**: 添加必要数据库字段，所有用户故事的前置条件

**⚠️ CRITICAL**: 必须在任何用户故事开始前完成

- [x] T004 添加users.role字段: `ALTER TABLE users ADD COLUMN role INT DEFAULT 0`
- [x] T005 [P] 添加auctions.creator_id字段: `ALTER TABLE auctions ADD COLUMN creator_id BIGINT DEFAULT NULL`
- [x] T006 [P] 创建creator_id索引: `CREATE INDEX idx_auctions_creator_id ON auctions(creator_id)`
- [x] T007 运行现有测试验证无回归: `cd backend/auction && go test ./...`

**Checkpoint**: 数据库迁移完成，可以开始用户故事实现

---

## Phase 3: User Story 1 - Redis状态同步与分布式锁 (Priority: P1) 🎯 MVP

**Goal**: 启用Redis状态同步，实现分布式锁防止并发冲突

**Independent Test**: 并发出价测试通过，WebSocket重连状态恢复

### Tests for User Story 1

- [x] T008 [P] [US1] 创建分布式锁测试: `backend/auction/service/lock_test.go`
- [x] T009 [P] [US1] 创建WebSocket管理器测试: `backend/auction/websocket/manager_test.go`

### Implementation for User Story 1

- [x] T010 [P] [US1] 创建DistributedLockService: `backend/auction/service/lock.go`
  ```go
  type DistributedLockService struct {
      redis      *redis.Client
      localLocks sync.Map
      defaultTTL time.Duration
  }
  ```
- [x] T011 [P] [US1] 创建WebSocketManager: `backend/auction/websocket/manager.go`
  ```go
  type WebSocketManager struct {
      hub          *Hub
      stateManager *StateManager
      redis        *redis.Client
  }
  ```
- [x] T012 [US1] 修改Hub集成StateManager: `backend/auction/websocket/hub.go`
  - 添加 stateManager 字段
  - 添加 SetStateManager 方法
- [x] T013 [US1] 修改Client连接时保存状态: `backend/auction/websocket/client.go`
  - 在连接时调用 stateManager.SaveConnectionState
  - 在断开时调用 stateManager.DeleteConnectionState
- [x] T014 [US1] 修改BidService.PlaceBid使用分布式锁: `backend/auction/service/bid.go`
  - 在出价前获取锁: `lock:auction:{auction_id}:bid`
  - 在出价后释放锁
- [x] T015 [US1] 修改main.go初始化新服务: `backend/auction/main.go`
  - 创建 DistributedLockService
  - 创建 WebSocketManager
  - 注入到 Hub
- [x] T016 [US1] 运行User Story 1测试验证

**Checkpoint**: US1完成 - 分布式锁工作正常，WebSocket状态可持久化

---

## Phase 4: User Story 2 - 用户历史记录真实查询 (Priority: P2)

**Goal**: 替换模拟数据为真实数据库查询

**Independent Test**: API返回真实历史数据，包含商品名、出价次数

### Implementation for User Story 2

- [x] T017 [P] [US2] 创建HistoryDAO: `backend/product/dao/history.go`
  ```go
  type HistoryDAO struct {
      db *gorm.DB
  }
  
  func (d *HistoryDAO) QueryUserHistory(ctx context.Context, userID int64, page, pageSize int) ([]UserHistoryItem, int64, error)
  ```
- [x] T018 [P] [US2] 创建HistoryService: `backend/product/service/history.go`
  ```go
  type HistoryService struct {
      historyDAO *dao.HistoryDAO
  }
  ```
- [x] T019 [US2] 修改OrderService.GetUserHistory调用HistoryService: `backend/product/service/order.go`
  - 替换硬编码数据为真实查询
- [x] T020 [US2] 运行User Story 2测试验证: `curl http://localhost:8081/api/v1/orders/history`

**Checkpoint**: US2完成 - 用户历史记录返回真实数据

---

## Phase 5: User Story 3 - 时间同步周期性推送 (Priority: P2)

**Goal**: 每5秒向进行中的竞拍推送服务器时间

**Independent Test**: WebSocket客户端每5秒收到时间同步消息

### Implementation for User Story 3

- [x] T021 [US3] 修改Scheduler添加时间同步任务: `backend/auction/service/scheduler.go`
  - 添加 startTimeSyncTask 方法
  - 添加 broadcastTimeSync 方法
  - 每5秒查询进行中的竞拍并推送
- [x] T022 [US3] 修改TimeSyncService添加广播方法: `backend/auction/websocket/time_sync.go`
  - 添加 BroadcastTimeSync 方法
- [x] T023 [US3] 运行User Story 3测试验证
  - 检查WebSocket日志确认每5秒收到消息

**Checkpoint**: US3完成 - 时间同步推送工作正常

---

## Phase 6: User Story 4 - RBAC权限验证 (Priority: P3)

**Goal**: 实现三角色权限控制

**Independent Test**: 不同角色用户权限正确控制

### Implementation for User Story 4

- [x] T024 [P] [US4] 创建RBAC中间件(gateway): `backend/gateway/middleware/rbac.go`
  ```go
  func RBACMiddleware(requiredRole int) app.HandlerFunc
  ```
- [x] T025 [P] [US4] 创建RBAC中间件(auction): `backend/auction/middleware/rbac.go`
- [x] T026 [P] [US4] 修改User模型添加Role常量: `backend/auction/model/user.go`
  ```go
  type Role int
  const (
      RoleUser     Role = 0
      RoleStreamer Role = 1
      RoleAdmin    Role = 2
  )
  ```
- [x] T027 [US4] 修改路由添加RBAC中间件: `backend/gateway/router/router.go`
  - 创建竞拍: 需要 RoleStreamer 或 RoleAdmin
  - 取消竞拍: 需要 RoleStreamer 或 RoleAdmin
- [x] T028 [US4] 修改JWT中间件解析role字段: `backend/gateway/middleware/jwt.go`
- [x] T029 [US4] 运行User Story 4测试验证
  - 普通用户尝试创建竞拍应返回403
  - 主播创建竞拍应成功

**Checkpoint**: US4完成 - 权限验证工作正常

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: 整体测试和文档更新

- [x] T030 运行所有后端测试: `cd backend && go test ./...`
- [x] T031 [P] 更新spec.md补充实施记录
- [x] T032 [P] 运行quickstart.md验证清单
- [x] T033 提交代码变更

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖，可立即开始
- **Foundational (Phase 2)**: 依赖Setup完成，**阻塞所有用户故事**
- **User Stories (Phase 3-6)**: 全部依赖Foundational完成
  - US1 (P1) → US2 (P2) → US3 (P2) → US4 (P3) 按优先级顺序
  - 或并行执行（如有足够人力）
- **Polish (Phase 7)**: 依赖所有期望的用户故事完成

### User Story Dependencies

- **US1 (P1)**: 可在Foundational后立即开始 - 无其他故事依赖
- **US2 (P2)**: 可在Foundational后立即开始 - 独立可测试
- **US3 (P2)**: 可在Foundational后立即开始 - 独立可测试
- **US4 (P3)**: 可在Foundational后立即开始 - 独立可测试

### Parallel Opportunities

- T004, T005, T006 可并行（不同数据库操作）
- T008, T009 可并行（不同测试文件）
- T010, T011 可并行（不同新文件）
- T017, T018 可并行（不同新文件）
- T024, T025, T026 可并行（不同文件）

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. 完成 Phase 1: Setup
2. 完成 Phase 2: Foundational
3. 完成 Phase 3: User Story 1
4. **停止并验证**: 独立测试US1
5. 如需可部署

### Incremental Delivery

1. Setup + Foundational → 基础就绪
2. 添加 US1 → 独立测试 → 部署/演示 (MVP!)
3. 添加 US2 → 独立测试 → 部署/演示
4. 添加 US3 → 独立测试 → 部署/演示
5. 添加 US4 → 独立测试 → 部署/演示

---

## Summary

| Phase | Tasks | Parallelizable |
|-------|-------|----------------|
| Setup | 3 | 1 |
| Foundational | 4 | 2 |
| US1 (P1) | 9 | 4 |
| US2 (P2) | 4 | 2 |
| US3 (P2) | 3 | 0 |
| US4 (P3) | 6 | 3 |
| Polish | 4 | 2 |
| **Total** | **33** | **14** |

**MVP Scope**: Phase 1 + Phase 2 + Phase 3 (US1) = 16 tasks

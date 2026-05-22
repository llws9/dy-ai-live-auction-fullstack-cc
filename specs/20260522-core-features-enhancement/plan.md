# Implementation Plan: 直播竞拍系统核心功能完善

**Feature**: `20260522-core-features-enhancement` | **Date**: 2026-05-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/20260522-core-features-enhancement/spec.md`

## Summary

本功能旨在完善直播竞拍系统的四个核心领域：
1. **Redis状态同步**：启用WebSocket连接状态持久化，支持重连恢复
2. **分布式锁**：防止竞拍出价并发冲突，确保数据一致性
3. **用户历史记录**：替换模拟数据为真实数据库查询
4. **RBAC权限验证**：实现三角色权限控制（普通用户/主播/平台管理员）

采用方案B（完整重构方案），在现有架构基础上引入新的服务层组件，提升系统的可扩展性和可靠性。

---

## Technical Context

**Language/Version**: Go 1.21+ (后端), TypeScript/React 18 (前端)
**Primary Dependencies**: 
- 后端: Hertz (HTTP), gorilla/websocket, go-redis/v9, GORM
- 前端: React, React Router, Vite
**Storage**: MySQL 8.0 (关系型), Redis 7 (缓存/分布式锁)
**Testing**: Go testing + testify, Jest (前端)
**Target Platform**: Linux server (Docker)
**Project Type**: Web应用 (前后端分离)
**Performance Goals**: 
- 竞拍出价延迟 < 100ms
- WebSocket消息推送延迟 < 50ms
- 并发用户支持 10000+
**Constraints**: 
- 分布式锁超时 < 5s
- 时间同步误差 < 500ms
- Redis降级不影响主流程
**Scale/Scope**: 
- 3个微服务 (auction, product, gateway)
- 约15个代码文件变更
- 预计开发时间 1-2工作日

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Check Result |
|-----------|--------|--------------|
| **I. 全栈一体化** | ✅ PASS | 前后端代码统一管理，API变更同步更新 |
| **II. 实时性优先** | ✅ PASS | 分布式锁确保竞拍操作原子性，时间同步保证公平性 |
| **III. 质量保障** | ✅ PASS | 保持现有测试通过，增量添加测试覆盖 |
| **IV. 可扩展性** | ✅ PASS | 新服务层组件遵循现有架构规范，配置化设计 |

**Fixed Rules Check**:
- ✅ Commit Workflow: 变更完成后执行提交
- ✅ Code Consistency: 复用现有代码模式（StateManager, Scheduler等）
- ✅ Real-Time Changes: 评估分布式锁和WebSocket变更的延迟影响
- ✅ API First: 前后端接口定义一致

**Gate Status**: ✅ ALL PASSED

---

## Project Structure

### Documentation (this feature)

```
specs/20260522-core-features-enhancement/
├── spec.md              # 功能规格说明
├── plan.md              # 本文件 (实施计划)
├── research.md          # Phase 0 研究文档 (如需)
├── data-model.md        # 数据模型设计
├── quickstart.md        # 快速启动指南
├── contracts/           # API契约定义
└── tasks.md             # 任务分解 (待执行 /adk:sdd:tasks)
```

### Source Code (repository root)

```
backend/
├── auction/                 # 竞拍服务 (端口 8082/8083)
│   ├── service/
│   │   ├── lock.go          # [NEW] DistributedLockService
│   │   ├── bid.go           # [MOD] 使用分布式锁
│   │   └── scheduler.go     # [MOD] 时间同步推送
│   ├── websocket/
│   │   ├── manager.go       # [NEW] WebSocketManager
│   │   ├── hub.go           # [MOD] 集成StateManager
│   │   ├── client.go        # [MOD] 状态同步
│   │   └── state_sync.go    # [EXISTING] StateManager
│   ├── middleware/
│   │   └── rbac.go          # [NEW] RBAC中间件
│   ├── model/
│   │   └── user.go          # [MOD] 添加Role常量
│   └── main.go              # [MOD] 服务初始化
│
├── product/                 # 商品/订单服务 (端口 8081)
│   ├── dao/
│   │   └── history.go       # [NEW] HistoryDAO
│   └── service/
│       ├── order.go         # [MOD] 调用HistoryService
│       └── history.go       # [NEW] HistoryService
│
└── gateway/                 # API网关 (端口 8080)
    ├── middleware/
    │   ├── jwt.go           # [EXISTING] JWT中间件
    │   └── rbac.go          # [NEW] RBAC中间件
    └── router/
        └── router.go        # [MOD] 添加RBAC中间件

frontend/
└── h5/                      # H5用户端 (端口 3000)
    └── src/
        └── (无变更)

docker-compose.yml           # [EXISTING] Redis已配置
```

**Structure Decision**: 采用现有Web应用结构，前后端分离。后端为微服务架构，本次变更涉及auction、product、gateway三个服务。

---

## Implementation Phases

### Phase 1: Redis环境准备与分布式锁 (P1)

**目标**: 启用Redis并实现分布式锁服务

**变更文件**:
| File | Change Type | Description |
|------|-------------|-------------|
| `backend/auction/service/lock.go` | NEW | DistributedLockService |
| `backend/auction/service/bid.go` | MOD | PlaceBid使用分布式锁 |
| `backend/auction/main.go` | MOD | 创建DistributedLockService实例 |

**验收标准**:
- Redis服务启动并连接成功
- 并发出价测试通过，无数据竞争
- Redis不可用时降级为本地锁

---

### Phase 2: WebSocket状态同步集成 (P1)

**目标**: 在WebSocket连接中启用StateManager

**变更文件**:
| File | Change Type | Description |
|------|-------------|-------------|
| `backend/auction/websocket/manager.go` | NEW | WebSocketManager |
| `backend/auction/websocket/hub.go` | MOD | 添加stateManager字段 |
| `backend/auction/websocket/client.go` | MOD | 连接时保存状态 |
| `backend/auction/main.go` | MOD | 创建WebSocketManager |

**验收标准**:
- WebSocket连接时状态保存到Redis
- 重连后状态恢复成功
- 现有WebSocket测试保持通过

---

### Phase 3: 用户历史记录真实查询 (P2)

**目标**: 实现真实的用户竞拍历史查询

**变更文件**:
| File | Change Type | Description |
|------|-------------|-------------|
| `backend/product/dao/history.go` | NEW | HistoryDAO |
| `backend/product/service/history.go` | NEW | HistoryService |
| `backend/product/service/order.go` | MOD | 调用HistoryService |

**验收标准**:
- GetUserHistory返回真实数据
- 包含商品名、出价次数、是否中标
- 查询响应时间<500ms

---

### Phase 4: 时间同步周期性推送 (P2)

**目标**: 每5秒向进行中的竞拍推送服务器时间

**变更文件**:
| File | Change Type | Description |
|------|-------------|-------------|
| `backend/auction/service/scheduler.go` | MOD | 添加时间同步任务 |
| `backend/auction/websocket/time_sync.go` | MOD | BroadcastTimeSync方法 |

**验收标准**:
- 每5秒推送时间同步消息
- 推送间隔误差<100ms
- 仅向进行中的竞拍推送

---

### Phase 5: RBAC权限验证 (P3)

**目标**: 实现三角色权限控制

**变更文件**:
| File | Change Type | Description |
|------|-------------|-------------|
| `backend/gateway/middleware/rbac.go` | NEW | RBAC中间件 |
| `backend/auction/middleware/rbac.go` | NEW | RBAC中间件 |
| `backend/gateway/router/router.go` | MOD | 添加RBAC中间件 |
| `backend/auction/model/user.go` | MOD | 添加Role常量 |

**DB变更**:
| Table | Field | Type | Default | Description |
|-------|-------|------|---------|-------------|
| `users` | `role` | INT | 0 | 用户角色 |
| `auctions` | `creator_id` | BIGINT | NULL | 竞拍创建者ID |

**验收标准**:
- 普通用户无法创建/取消竞拍
- 主播只能操作自己的竞拍
- 平台管理员可操作所有资源

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Redis不可用 | 降级为本地内存锁，记录错误日志 |
| 破坏现有功能 | 保持所有现有测试通过，增量添加测试 |
| 权限配置错误 | 默认为普通用户，手动提权 |
| 并发竞拍冲突 | 分布式锁使用PEXPIRE自动续期 |

---

## Dependencies

### Infrastructure
- ✅ Redis 7 - 已在docker-compose.yml配置
- ✅ MySQL 8 - 现有数据库
- ✅ Docker - 容器化部署

### Libraries
- ✅ go-redis/v9 - 现有依赖
- ✅ gorilla/websocket - 现有依赖
- ✅ Hertz - 现有HTTP框架

---

## Next Steps

执行 `/adk:sdd:tasks` 生成详细任务分解，开始实施。

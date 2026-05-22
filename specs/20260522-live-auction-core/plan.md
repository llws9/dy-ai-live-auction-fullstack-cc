# Implementation Plan: 直播竞拍系统核心功能完善

**Feature**: `20260522-live-auction-core` | **Date**: 2026-05-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/20260522-live-auction-core/spec.md`

## Summary

本功能模块完善直播竞拍系统的核心体验，包括：
1. **实时排名同步**：出价成功后实时广播排名更新到所有参与用户
2. **断线重连机制**：WebSocket自动重连与状态同步
3. **PC管理后台**：商品、竞拍、订单管理功能
4. **体验优化**：动画效果、倒计时精度、历史记录查询

技术方案采用渐进式实施，分三周完成，优先保证实时性和稳定性。

## Technical Context

**Language/Version**: Go 1.21+ (backend), TypeScript 5.0+ (frontend)
**Primary Dependencies**:
- Backend: Hertz, gorilla/websocket, GORM, go-redis
- Frontend: React 18, TypeScript, Context API, CSS Transitions

**Storage**: MySQL 8.0 (persistent), Redis 7 (cache + distributed lock)
**Testing**: Go testing package, Jest
**Target Platform**: Linux server (backend), Browser - Mobile H5 & PC Admin (frontend)
**Project Type**: Full-stack web application
**Performance Goals**:
- 出价响应时间 < 200ms (P99)
- WebSocket 消息推送延迟 < 100ms
- 支持 1000 个并发 WebSocket 连接
- 倒计时显示误差 < 100ms

**Constraints**:
- 实时性优先：实时通道不得引入不必要的中间层
- 最终一致性：状态同步必须保证最终一致性
- 断线恢复：网络波动后必须自动重连

**Scale/Scope**:
- 100+ 并发出价
- 1000 并发 WebSocket 连接
- 4 个核心功能模块
- 预估变更文件：25-30个

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### ✅ I. 全栈一体化 (Full-Stack Integration)

**要求**：API 变更必须同步更新前后端代码，共享类型定义优先使用共享模块

**符合性检查**：
- ✅ 排名同步：后端 `broadcastRanking` + 前端 WebSocket 处理同步设计
- ✅ 断线重连：后端 `state_sync.go` + 前端 `useReconnect.ts` 协同实现
- ✅ 管理后台：API 接口定义优先，前后端对齐
- ✅ 历史记录：共享 API 类型定义

### ✅ II. 实时性优先 (Real-Time Priority)

**要求**：实时通道路径不得引入不必要的中间层，关键实时操作必须有超时和重试机制

**符合性检查**：
- ✅ 排名广播：直接通过 WebSocket Hub 广播，无中间层
- ✅ 断线重连：指数退避重试机制（1s→2s→4s→8s→max 30s）
- ✅ 心跳保活：30s ping 间隔，60s 超时
- ✅ 消息节流：防止消息洪泛，每 200ms 最多推送一次排名更新

### ✅ III. 质量保障 (Quality Assurance)

**要求**：所有代码变更必须通过 CI 检查，关键业务逻辑必须有单元测试覆盖

**符合性检查**：
- ✅ 单元测试：每个模块必须包含核心逻辑测试
- ✅ 集成测试：WebSocket 连接、断线重联场景测试
- ✅ 性能测试：并发出价、排名同步延迟测试
- ✅ Code Review：所有变更必须通过审查

### ✅ IV. 可扩展性 (Scalability)

**要求**：新模块必须遵循现有架构规范，配置化优于硬编码

**符合性检查**：
- ✅ 遵循现有目录结构：`handler/service/dao/model` 分层
- ✅ 配置化：延时策略、重连参数均可配置
- ✅ 复用现有基础设施：WebSocket Hub、Redis 分布式锁
- ✅ 统一错误处理和日志规范

**Gate Status**: ✅ PASS - 所有核心原则符合，无需例外说明

## Project Structure

### Documentation (this feature)

```
specs/20260522-live-auction-core/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── ranking-api.yaml
│   ├── reconnection-api.yaml
│   └── admin-api.yaml
└── tasks.md             # Phase 2 output (/adk:sdd:tasks)
```

### Source Code (repository root)

```
backend/
├── auction/
│   ├── service/
│   │   ├── bid.go           # 修改：添加 broadcastRanking
│   │   └── state_sync.go    # 新增：重连状态同步
│   ├── websocket/
│   │   ├── hub.go           # 已存在：房间管理
│   │   ├── client.go        # 修改：心跳检测优化
│   │   ├── message.go       # 修改：新增消息类型
│   │   ├── state_sync.go    # 新增：状态同步逻辑
│   │   └── time_sync.go     # 新增：时间同步机制
│   └── handler/
│       └── bid.go           # 已存在：出价处理
├── product/
│   ├── handler/
│   │   ├── product.go       # 新增：商品管理API
│   │   └── order.go         # 新增：订单管理API
│   └── service/
│       ├── product.go       # 新增：商品服务层
│       └── order.go         # 新增：订单服务层
└── gateway/
    └── middleware/          # 已存在：权限验证

frontend/
├── h5/
│   ├── src/
│   │   ├── services/
│   │   │   ├── websocket.ts # 修改：重连逻辑
│   │   │   └── api.ts       # 修改：历史记录API
│   │   ├── hooks/
│   │   │   ├── useReconnect.ts    # 新增：重连Hook
│   │   │   └── useServerTime.ts   # 新增：时间同步Hook
│   │   ├── pages/
│   │   │   ├── Auction/
│   │   │   │   ├── index.tsx      # 修改：排名更新
│   │   │   │   ├── Ranking.tsx    # 修改：实时排名
│   │   │   │   └── Countdown.tsx  # 修改：毫秒级倒计时
│   │   │   └── History/
│   │   │       └── index.tsx      # 新增：历史记录页
│   │   └── utils/
│   │       └── animations.ts      # 新增：动画工具
│   └── tests/
└── admin/
    └── src/
        ├── pages/
        │   ├── Product/
        │   │   ├── List.tsx       # 新增：商品列表
        │   │   ├── Create.tsx     # 新增：商品创建
        │   │   └── Edit.tsx       # 新增：商品编辑
        │   ├── Auction/
        │   │   ├── List.tsx       # 新增：竞拍列表
        │   │   └── Detail.tsx     # 新增：竞拍详情
        │   └── Order/
        │       ├── List.tsx       # 新增：订单列表
        │       └── Detail.tsx     # 新增：订单详情
        └── services/
            └── api.ts             # 新增：管理后台API
```

**Structure Decision**: 采用 Option 2 (Web application) 结构，前后端分离但统一管理。遵循现有目录结构，新增文件按功能模块组织。

## Complexity Tracking

*No violations - Constitution Check passed without exceptions*

---

## Phase 0: Research

**Status**: ✅ Complete

**Output**: [research.md](./research.md)

**Key Findings**:
1. **WebSocket 重连策略**：指数退避 + 心跳保活，最大重试10次
2. **排名广播优化**：消息节流 + 批量广播，每200ms推送一次
3. **时间同步机制**：服务端时间下发 + 客户端定期校准（每10秒）
4. **动画性能优化**：CSS 硬件加速 + 自动降级（< 30fps 禁用动画）
5. **权限控制**：JWT + RBAC，支持 admin/operator/viewer 角色

**No NEEDS CLARIFICATION markers** - 所有技术决策点已明确。

---

## Phase 1: Design & Contracts

**Status**: ✅ Complete

### Data Model

**Output**: [data-model.md](./data-model.md)

**New Entities**:
- `Order` - 订单表（状态流转：pending → paid → shipped → completed）
- `ConnectionState` - 连接状态（Redis）
- `SyncState` - 同步状态（Redis）

**Extended Entities**:
- `Product.Status` - 商品状态（草稿、上架、下架）
- `User.Role` - 用户角色（user, admin, operator）

**WebSocket Message Types**:
- Server→Client: `rank_update`, `sync_response`, `time_sync`, `delay_triggered`, `auction_ended`
- Client→Server: `ping`, `sync_request`

### API Contracts

**Output**: [contracts/](./contracts/)

1. **[ranking-api.yaml](./contracts/ranking-api.yaml)** - 排名查询 API
   - `GET /auctions/{auctionId}/ranking` - 获取排名列表

2. **[reconnection-api.yaml](./contracts/reconnection-api.yaml)** - WebSocket 连接 API
   - `GET /ws?auction_id={id}&user_id={id}|token={jwt}` - WebSocket 连接端点
   - 消息类型定义（ServerMessage, ClientMessage）

3. **[admin-api.yaml](./contracts/admin-api.yaml)** - 管理后台 API
   - 商品管理：`GET/POST/PUT/DELETE /products`
   - 竞拍管理：`GET /auctions`, `PUT /auctions/{id}/cancel`
   - 订单管理：`GET/PUT /orders`, `POST /orders/{id}/pay`
   - 用户历史：`GET /users/me/history`

### Quickstart Guide

**Output**: [quickstart.md](./quickstart.md)

**实施顺序**:
- **第一周**：实时排名同步（Day 1-2）+ 断线重连（Day 3-4）
- **第二周**：PC管理后台（商品管理 Day 1-2，竞拍/订单管理 Day 3-4）
- **第三周**：体验优化（动画 Day 1-2，倒计时 Day 3，历史记录 Day 4）

**关键代码示例**:
- `broadcastRanking` 方法实现
- `useReconnect` Hook 实现
- `useServerTime` Hook 实现
- 动画配置

---

## Next Steps

实施计划已完成，下一步：

1. **验证计划**：确认技术方案和实施顺序是否符合预期
2. **生成任务**：执行 `/adk:sdd:tasks` 生成详细的任务分解
3. **开始实施**：按照 quickstart.md 指南开始编码

**注意**：所有生成的文档均已保存在 `specs/20260522-live-auction-core/` 目录下。

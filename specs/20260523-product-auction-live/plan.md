---
description: "Implementation plan for user-facing features in live auction system"
---

# Implementation Plan: 用户功能开发

**Feature**: `20260523-product-auction-live` | **Date**: 2026-05-23 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification focusing on User Story 2.5 (关注直播间) and 2.6 (用户出价)

**Note**: 本计划专注于用户端功能的开发，基于已完成的商家和管理员功能。

## Summary

本计划专注于用户端核心功能的开发，包括：
1. **用户关注直播间功能**（User Story 2.5）：用户可以关注/取消关注直播间，接收通知推送
2. **用户出价竞拍功能**（User Story 2.6）：用户可以在竞拍中出价，系统验证用户登录状态

这两个功能是竞拍系统的核心闭环，直接影响用户参与度和业务收入。后端基础已实现，前端用户端（H5）需要完整开发。

## Technical Context

**Language/Version**:
- Backend: Go 1.21+ (已完成)
- Frontend H5: TypeScript + React 18+

**Primary Dependencies**:
- Backend: Hertz (HTTP), GORM (ORM), JWT认证
- Frontend H5: React, React Router, Axios, WebSocket (原生)

**Storage**: MySQL 8.0 (已有schema)
**Testing**: Jest (前端), Go testing (后端)
**Target Platform**:
- Backend: Linux服务器
- Frontend H5: 移动端浏览器 (iOS Safari, Android Chrome)

**Project Type**: Full-stack web application (前端H5 + 后端API)
**Performance Goals**:
- 出价延迟 < 500ms (端到端)
- WebSocket推送延迟 < 100ms
- 支持1000并发用户同时出价

**Constraints**:
- JWT token有效期：7天
- 出价必须实时同步，不允许延迟
- 关注列表支持分页，每页20条

**Scale/Scope**:
- 预计用户数：10万+
- 并发出价：1000/秒
- 关注关系：平均每用户关注5个直播间

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. 全栈一体化 ✅ PASS
- 前端H5和后端API在同一仓库
- API变更已同步更新（spec.md中的api-documentation.md已更新）
- 共享数据模型定义（Bid, UserLiveStreamFollow）

### II. 实时性优先 ✅ PASS
- 出价功能使用WebSocket实时推送
- 状态同步保证最终一致性
- 关键实时操作有超时和重试机制

### III. 质量保障 ✅ PASS
- 后端已有单元测试（follow_test.go）
- 前端需要添加组件测试
- CI检查已配置

### IV. 可扩展性 ✅ PASS
- 遵循现有前后端架构
- 配置化通知设置
- 复用现有JWT认证和通知系统

**Gate Result**: ✅ 所有原则通过，可以继续Phase 0研究

## Project Structure

### Documentation (this feature)

```
specs/20260523-product-auction-live/
├── plan.md              # 本文件
├── spec.md              # 功能规范（已更新，包含User Story 2.5和2.6）
├── api-documentation.md # API文档（已更新，包含出价API）
├── data-model.md        # 数据模型（已存在）
├── research.md          # Phase 0研究输出（待生成）
├── quickstart.md        # 快速开始指南（已存在）
├── contracts/           # API契约（已存在）
└── tasks.md             # 任务分解（待生成）
```

### Source Code (repository root)

```
backend/
├── auction/
│   ├── handler/
│   │   ├── bid.go              ✅ 已实现（出价Handler）
│   │   └── follow.go           ✅ 已实现（关注Handler）
│   ├── service/
│   │   ├── bid.go              ✅ 已实现（出价Service）
│   │   ├── follow.go           ✅ 已实现（关注Service）
│   │   └── follow_test.go      ✅ 已实现（单元测试）
│   ├── dao/
│   │   ├── bid.go              ✅ 已实现（出价DAO）
│   │   └── follow.go           ✅ 已实现（关注DAO）
│   └── model/
│       ├── bid.go              ✅ 已实现（出价Model）
│       └── user_live_stream_follow.go ✅ 已实现（关注Model）
│
├── gateway/
│   └── middleware/
│       └── auth.go             ✅ 已实现（JWT认证中间件）

frontend/
├── h5/                         🚧 用户端（本次开发重点）
│   ├── src/
│   │   ├── pages/
│   │   │   ├── Live/
│   │   │   │   └── index.tsx   ✅ 已存在（直播间页面）
│   │   │   ├── Follow/
│   │   │   │   └── index.tsx   ❌ 未实现（我的关注页面）
│   │   │   └── Login/
│   │   │       └── index.tsx   ❌ 未实现（登录页面）
│   │   ├── services/
│   │   │   ├── api.ts          ✅ 已存在（API服务）
│   │   │   ├── auth.ts         ❌ 未实现（认证服务）
│   │   │   └── websocket.ts    ❌ 未实现（WebSocket服务）
│   │   ├── components/
│   │   │   ├── BidInput.tsx    ❌ 未实现（出价输入组件）
│   │   │   └── FollowButton.tsx ❌ 未实现（关注按钮组件）
│   │   └── store/
│   │       └── authContext.tsx ❌ 未实现（认证上下文）
│   └── tests/
│       └── *.test.tsx          ❌ 未实现（前端测试）
│
└── admin/                      ✅ 管理端（已完成）
    └── src/pages/
        ├── Product/
        ├── Auction/
        └── LiveStream/
```

**Structure Decision**: 采用Web Application结构，后端已完成，前端H5需要开发用户功能模块。重点开发：1) 认证系统（登录/登出）2) 出价UI组件 3) 关注功能 4) WebSocket实时推送

## Phase 0: Research Output ✅

**Status**: 已完成
**Output**: [research.md](./research.md)

**关键决策**:
1. ✅ JWT认证方案：token存储在localStorage，Axios拦截器自动添加认证头
2. ✅ WebSocket实时推送：使用原生WebSocket API + 自动重连
3. ✅ 出价验证：前端实时验证 + 后端二次验证
4. ✅ 关注UI交互：乐观更新 + 异步同步
5. ✅ 移动端适配：响应式设计 + 触摸优化
6. ✅ 状态管理：React Context + useReducer

**所有NEEDS CLARIFICATION已解决，无技术债务**。

## Phase 1: Design & Contracts ✅

**Status**: 部分完成（数据模型已存在，需要补充用户端API契约）

### 数据模型
**Status**: ✅ 已完成
**Output**: [data-model.md](./data-model.md)

关键实体已定义：
- ✅ Bid（出价记录）- 已实现
- ✅ UserLiveStreamFollow（关注关系）- 已实现
- ✅ User（用户角色：0=普通用户, 1=商家, 2=管理员）

### API契约
**Status**: ✅ 已完成
**Output**: [contracts/](./contracts/) 和 [api-documentation.md](./api-documentation.md)

已定义API：
- ✅ POST /auctions/:id/bids - 用户出价（需JWT认证）
- ✅ GET /auctions/:id/ranking - 获取排名
- ✅ POST /live-streams/:id/follow - 关注直播间（需JWT认证）
- ✅ DELETE /live-streams/:id/follow - 取消关注
- ✅ GET /user/followed-live-streams - 获取关注列表（需JWT认证）

### 快速开始指南
**Status**: ✅ 已完成
**Output**: [quickstart.md](./quickstart.md)

## Complexity Tracking

**无违规项** - 所有设计决策符合Constitution原则，无需复杂度跟踪。

## Next Steps

**Phase 0 和 Phase 1 已完成**，可以执行：

### `/adk:sdd:tasks` - 生成任务分解

根据本计划和research.md的技术决策，生成详细的开发任务列表。

**预期任务范围**:
1. 前端H5 - 认证系统（登录页、AuthContext、API拦截器）
2. 前端H5 - 出价功能（出价输入组件、验证逻辑、WebSocket集成）
3. 前端H5 - 关注功能（关注按钮、我的关注页面）
4. 前端H5 - WebSocket服务（连接管理、消息处理、重连机制）
5. 测试 - 前端组件测试、E2E测试

---

**Plan Status**: ✅ 完成
**Generated**: 2026-05-23
**Last Updated**: 2026-05-23

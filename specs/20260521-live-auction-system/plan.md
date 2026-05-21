---
description: "Implementation plan for live auction system"
---

# Implementation Plan: 直播竞拍全栈系统

**Feature**: `20260521-live-auction-system` | **Date**: 2026-05-21 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/20260521-live-auction-system/spec.md`

## Summary

构建抖音电商直播竞拍全栈系统，实现商品发布、实时出价、自动延时、WebSocket 实时通信等核心功能。采用 Go + Hertz 后端、React + TypeScript 前端、MySQL + Redis 存储的微服务架构。

## Technical Context

**Language/Version**: Go 1.21+, Node.js 18+, TypeScript 5.0+
**Primary Dependencies**: 
- Backend: Hertz, gorilla/websocket, GORM, go-redis
- Frontend: React 18, TypeScript, Context API
**Storage**: MySQL 8.0 (持久化), Redis 7.0 (缓存/分布式锁)
**Testing**: Go testing, Jest, Cypress
**Target Platform**: Linux server (Docker), Mobile H5
**Project Type**: 微服务全栈项目
**Performance Goals**: 出价响应 < 200ms, WebSocket 推送 < 100ms
**Constraints**: 1000 并发连接, 100+ 人同时出价
**Scale/Scope**: 单直播间 1000 用户, 支持 10+ 并发竞拍

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| 原则 | 检查项 | 状态 |
|------|--------|------|
| **全栈一体化** | 前后端 API 契约是否同步定义 | ✅ API 接口已定义 |
| **实时性优先** | WebSocket 路径是否避免中间层 | ✅ 直连 auction-service |
| **质量保障** | 是否有测试策略 | ✅ 单元测试 + 集成测试 |
| **可扩展性** | 模块化设计 | ✅ 3 个独立微服务 |

**Fixed Rules 检查**：
- ✅ API First: 接口定义先于实现
- ✅ Real-Time Changes: 已评估延迟影响和回滚策略

## Project Structure

### Documentation (this feature)

```
specs/20260521-live-auction-system/
├── spec.md              # 功能规格
├── plan.md              # 本文件
├── data-model.md        # 数据模型
├── quickstart.md        # 快速启动指南
├── contracts/           # API 合约
│   └── openapi.yaml
└── checklists/          # 检查清单
```

### Source Code (repository root)

```
backend/
├── gateway/             # API 网关
│   ├── main.go
│   ├── middleware/
│   │   └── ratelimit.go
│   └── router/
│       └── router.go
├── product/             # 商品服务
│   ├── main.go
│   ├── handler/
│   ├── service/
│   ├── dao/
│   └── model/
└── auction/             # 竞拍服务
    ├── main.go
    ├── handler/
    ├── service/
    ├── websocket/
    ├── lock/
    └── model/

frontend/
├── h5/                  # 用户端 H5
│   ├── src/
│   │   ├── pages/
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── store/
│   │   └── services/
│   └── package.json
└── admin/               # 管理后台
    ├── src/
    └── package.json

docker-compose.yml
```

**Structure Decision**: 采用微服务 + Monorepo 结构，前后端代码在同一仓库，服务独立部署。

## Complexity Tracking

无宪法违规项。所有设计符合项目原则。

## Risk Points

| 风险类型 | 描述 | 缓解措施 |
|---------|------|---------|
| **高并发** | 100+人同时出价 | Redis 分布式锁 + 网关限流 |
| **WebSocket 稳定性** | 网络波动断连 | 心跳保活 + 指数退避重连 |
| **数据一致性** | 竞拍状态同步 | Redis 缓存 + MySQL 事务 |
| **延时精度** | 倒计时毫秒级 | requestAnimationFrame + 时间校准 |
| **分布式锁** | 锁竞争性能 | 锁粒度优化 + TTL 防死锁 |

## Development Phases

### Phase 1 - MVP 核心（第一周）

1. 项目初始化：目录结构、Docker Compose
2. 数据库设计：MySQL 表结构
3. 商品服务：CRUD API
4. 竞拍服务：出价核心逻辑、分布式锁
5. WebSocket 房间管理
6. 竞拍状态机

### Phase 2 - 完善功能（第二周）

1. 自动延时机制
2. 实时排名同步
3. 断线重连
4. PC 管理后台
5. H5 用户端核心页面

### Phase 3 - 体验优化（第三周）

1. 动画效果
2. 倒计时精度优化
3. 历史记录
4. 模拟支付
5. 性能测试与优化

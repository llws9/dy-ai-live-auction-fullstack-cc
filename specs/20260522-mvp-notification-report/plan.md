---
description: "Implementation plan for MVP notification and reporting features"
---

# Implementation Plan: MVP阶段功能完善

**Feature**: `20260522-mvp-notification-report` | **Date**: 2026-05-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/20260522-mvp-notification-report/spec.md`

## Summary

完善直播竞拍系统MVP阶段功能，包括消息通知系统（P1）、数据分析报表（P2）、测试覆盖增强（P1）、API文档生成（P1）。采用并行开发策略，API文档和测试优先完成，通知和报表并行开发。关键设计：预留订单事件接口，支持二期无缝接入真实订单系统。

## Technical Context

**Language/Version**: Go 1.21+ (backend), TypeScript 5.x / React 18 (frontend)  
**Primary Dependencies**: Hertz (HTTP), gorilla/websocket, go-redis v9, GORM, swaggo/swag, Valtio (frontend state)  
**Storage**: MySQL 8.0 (notifications table), Redis 7 (WebSocket state, caching)  
**Testing**: go test (unit), playwright (E2E)  
**Target Platform**: Linux server (Docker containers)
**Project Type**: web (frontend + backend microservices)  
**Performance Goals**: Notification delivery <1s, Statistics API <3s, WebSocket connection stable  
**Constraints**: Order notification chain uses Mock triggers (phase 2 integration via reserved interfaces)  
**Scale/Scope**: 5k concurrent users, 4 backend services, 2 frontend apps

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

✅ **Single Project Structure**: Web application with frontend + backend microservices
✅ **No Duplicate Abstractions**: Using existing WebSocketManager, no new messaging layer
✅ **Standard Patterns**: REST API for notifications/statistics, WebSocket for real-time push
✅ **No Premature Optimization**: Statistics caching only if query time >1s
✅ **Test Coverage Target**: Core services >80%, E2E 5 core scenarios

**All gates passed.**

## Project Structure

### Documentation (this feature)

```
specs/20260522-mvp-notification-report/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (OpenAPI specs)
└── tasks.md             # Phase 2 output (from /adk:sdd:tasks)
```

### Source Code (repository root)

```
backend/
├── auction/
│   ├── service/
│   │   ├── notification.go    # [NEW] Notification service
│   │   ├── bid.go             # [MOD] Add notification trigger
│   │   └── auction.go         # [MOD] Add notification trigger
│   ├── dao/
│   │   └── notification.go    # [NEW] Notification DAO
│   ├── handler/
│   │   └── notification.go    # [NEW] Notification API
│   └── websocket/
│       └── message.go         # [MOD] Add notification message type
├── product/
│   ├── service/
│   │   ├── statistics.go      # [NEW] Statistics service
│   │   └── order.go           # [MOD] Mock notification trigger
│   ├── dao/
│   │   └── statistics.go      # [NEW] Statistics DAO
│   └── handler/
│       └── statistics.go      # [NEW] Statistics API
└── gateway/
    ├── main.go                # [MOD] Swagger integration
    └── router/
        └── router.go          # [MOD] Swagger routes

frontend/
├── h5/
│   └── src/
│       ├── components/
│       │   └── Notification/  # [NEW] Notification UI
│       └── hooks/
│           └── useNotification.ts  # [NEW] Notification hook
└── admin/
    └── src/
        ├── pages/
        │   ├── Dashboard/     # [NEW] Data dashboard
        │   └── Statistics/    # [NEW] Statistics reports
        └── components/
            └── Charts/        # [NEW] Chart components

docs/
├── swagger.json               # [NEW] Generated Swagger doc
└── swagger.yaml               # [NEW] Generated Swagger doc
```

**Structure Decision**: Web application with microservices backend (auction, product, gateway) and two frontend apps (H5 user端, Admin管理后台). Following existing project structure.

## Complexity Tracking

*No Constitution Check violations - all design decisions within standard patterns.*

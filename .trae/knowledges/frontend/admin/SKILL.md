---
name: knowledge-frontend-admin
description: >
  Covers Admin 管理后台的权限、API 封装、页面组织、实验配置、编码修复和测试约束。
  Navigate when: modifying frontend/admin pages, adding merchant/admin features, changing admin API calls, debugging role access, GrowthBook, orders, auctions, statistics, or live stream management.
  Excludes: H5 用户端和 Test Dashboard；Test Dashboard context is in ../test-dashboard/SKILL.md.
  Keywords: frontend/admin, Admin, RequireRole, RoleRoute, isAllowedRole, request.ts, /admin/orders, GrowthBook, decodePossibleMojibake, normalizeAuctionText
---

## Module Structure

Admin 是管理后台前端，面向商家和管理员角色；核心风险集中在角色权限、管理端 API 路径、响应归一化、实验配置隔离和生产构建路径。

### Directory Layout
- `frontend/admin/src/App.tsx` — Admin 路由和角色路由入口。
- `frontend/admin/src/components/Layout.tsx` — 后台布局与动态菜单。
- `frontend/admin/src/shared/auth/` — 登录态、角色判断和鉴权上下文。
- `frontend/admin/src/shared/api/` — 管理后台 API 封装、请求基础设施、类型定义和编码归一化。
- `frontend/admin/src/pages-new/` — 主要新版页面目录。
- `frontend/admin/src/pages/` — 存量页面目录，部分页面仍在这里维护。
- `frontend/admin/e2e/` — Playwright 管理后台 E2E 测试。
- `frontend/admin/nginx/`、`frontend/admin/Dockerfile` — 管理后台容器和静态服务配置。

### Key Entry Points
- `frontend/admin/src/shared/auth/roles.ts` — 角色权限判定工具。
- `frontend/admin/src/shared/api/request.ts` — Admin 统一请求封装。
- `frontend/admin/src/shared/api/index.ts` — 业务 API 聚合入口和响应归一化。
- `frontend/admin/src/shared/growthbook/GrowthBookContextProvider.tsx` — GrowthBook Provider。
- `frontend/admin/src/pages/LiveStreamFixedPrice/index.tsx` — 直播间一口价上下架和冲突处理。

## Gotchas
- 商家和管理员菜单与页面权限由 `RequireRole`、`RoleRoute`、`isAllowedRole` 多处共同约束，新增页面时必须同步路由守卫和菜单可见性，否则会出现可见但不可访问或可访问但不可见的错位（`frontend/admin/src/App.tsx`, `frontend/admin/src/components/Layout.tsx`, `frontend/admin/src/shared/auth/roles.ts`）
- Admin 端订单列表必须使用 `/admin/orders`，不能复用用户端 `/orders`，否则会被 `X-User-ID` 语义过滤成当前用户订单而不是管理视角订单（`frontend/admin/src/shared/api/index.ts`）
- `request.ts` 会在 401 时清理 token 并跳转登录页，新增静默探测类接口时需要显式控制错误展示策略，避免后台页面被非关键请求打断（`frontend/admin/src/shared/api/request.ts`）
- GrowthBook 必须在组件级用 `useMemo` 创建实例，模块级单例会导致属性在用户/环境之间泄漏，影响实验判断（`frontend/admin/src/shared/growthbook/GrowthBookContextProvider.tsx`）
- 后端返回的用户名、竞拍和出价文本可能存在 UTF-8 被误解析为 Windows-1252 的乱码，渲染前需走既有修复函数而不是在页面局部手写替换（`frontend/admin/src/shared/auth/AuthContext.tsx`, `frontend/admin/src/shared/api/auctionEncoding.ts`, `frontend/admin/src/shared/api/bidEncoding.ts`）
- 一口价商品若已被其他竞拍绑定，后端返回 409；前端需捕获后刷新可售商品列表，不能只弹错误后保留旧选项（`frontend/admin/src/pages/LiveStreamFixedPrice/index.tsx`）
- 封禁状态直播间必须在 UI 层禁用开启直播动作，不能只依赖后端拒绝，否则演示时会暴露可点击但失败的操作路径（`frontend/admin/src/pages-new/LiveDetail.tsx`）

## Architecture
- Admin API 层集中在 `shared/api/`，页面应通过封装函数访问后端，避免散落 axios/fetch 导致鉴权、错误提示和编码归一化不一致（`frontend/admin/src/shared/api/request.ts`, `frontend/admin/src/shared/api/index.ts`）
- 页面迁移处于 `pages-new/` 与 `pages/` 并存状态，修改导航或路由时必须确认目标页面实际位于哪个目录，不能假设所有页面都已迁移到新版目录（`frontend/admin/src/App.tsx`）
- 角色模型至少区分商家 `role=1` 与管理员 `role=2`，商家页面和管理员页面是同一 Admin 应用内的分支，而不是两个独立应用（`frontend/admin/src/App.tsx`, `frontend/admin/src/shared/auth/roles.ts`）

## Patterns
- 列表查询参数通过 `buildQuery` 过滤空值和 `undefined`，新增筛选器时应复用它，避免把空字符串或未定义参数发给后端造成过滤语义漂移（`frontend/admin/src/shared/api/request.ts`）
- 收入统计响应格式不稳定时通过 `normalizeRevenueStatsResponse` 兜底，页面不应直接假设单一响应结构（`frontend/admin/src/shared/api/index.ts`）
- 竞拍和出价列表进入 UI 前先归一化编码，后续组件应消费归一化后的对象而不是重复处理原始响应（`frontend/admin/src/shared/api/auctionEncoding.ts`, `frontend/admin/src/shared/api/bidEncoding.ts`）

## Conventions
- Admin 登录页 `/admin-login` 是独立入口，不走后台 Layout；新增登录相关跳转时不要把它放入普通菜单体系（`frontend/admin/src/App.tsx`）
- 共享基础组件放在 `components/shared/`，Radix 封装 UI 组件放在 `components/ui/`；新增组件时按复用范围放置，避免页面私有组件污染共享目录（`frontend/admin/src/components/shared/`, `frontend/admin/src/components/ui/`）
- Admin 生产构建和部署路径独立于 H5，demo 发布时继续使用 `/admin/` 子路径构建（`deploy/demo/MAIN_DEPLOY_QUICKSTART.md`, `frontend/admin/package.json`）

## Testing Strategy
- API 模块测试使用 MSW 模拟服务端响应，新增 API 归一化逻辑时优先补充 `shared/api` 附近的单元测试而不是只测页面渲染（`frontend/admin/src/shared/api/__tests__/`, `frontend/admin/src/mocks/handlers.ts`）
- E2E 覆盖商品管理、竞拍管理、统计报表等核心后台流程，改动这些路径时需要评估是否更新 Playwright 用例（`frontend/admin/e2e/`）

## Feature Knowledge

### 商品 AI 文案生成 (Product AI Copywriting)

**功能概述**：Admin 端提供一键 AI 生成商品文案功能，商家输入商品基础信息后，后端调用 Doubao/Ark 大模型生成营销文案。

**技术架构**：
- **入口**：Admin 商品编辑页面 (`frontend/admin/src/pages-new/goodsEditAi.ts`)
- **API**：`POST /api/v1/products/ai/copywriting`（经 Gateway 转发至 product-service）
- **后端实现**：`backend/product/handler/copywriting.go` + `backend/product/service/copywriting.go`
- **LLM 供应商**：`backend/shared/llm/` 抽象层，当前实现为 Doubao Provider

**关键约束**：
- **API 密钥管理**：生产环境 `ARK_API_KEY` 通过服务器环境文件（如 `/srv/auction/env/.env.demo`）配置，由 `product-service` 容器读取；Gateway 仅负责请求转发和鉴权，不直接调用 AI 接口
- **安全规范**：严禁将 API Key 提交至 Git 仓库或在对话中明文传输；更新密钥需修改服务器 `.env` 文件并重启对应服务
- **前端字段映射**：后端返回字段名为 `available_amount`（注意不是 `available` 或 `balance`），前端需正确读取避免余额显示为 0

**测试要点**：
- Admin 端单元测试覆盖 API 调用和错误处理 (`frontend/admin/src/pages-new/__tests__/goodsEditAi.test.ts`)
- 后端测试覆盖文案生成逻辑和 LLM 供应商封装 (`backend/product/handler/copywriting_test.go`, `backend/shared/llm/*_test.go`)

**来源**：session:6a2879d10bfcee1b04fc3745

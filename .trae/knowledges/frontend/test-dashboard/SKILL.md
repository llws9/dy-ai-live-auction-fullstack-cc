---
name: knowledge-frontend-test-dashboard
description: >
  Covers Test Dashboard 的测试任务 API、WebSocket 进度流、Zustand 状态、A/B 对比、演示大屏和 Docker/Nginx 运行约束。
  Navigate when: modifying frontend/test-dashboard, debugging progress WebSocket, adding test scenarios, changing report polling, demo theater, Grafana-facing demos, or test-dashboard deployment.
  Excludes: Admin 管理后台; Admin context is in ../admin/SKILL.md.
  Keywords: frontend/test-dashboard, Dashboard, Screen, wsStore, testStore, discoverWS, VITE_WS_BASE, StepTimeline, AntiSnipeTimeline, demoTheater, Zustand, WebSocket
---

## Module Structure

Test Dashboard 是测试与演示控制台，负责任务启动、实时进度、报告查询、A/B 对比和演示大屏；它依赖 Gateway 暴露的 HTTP API 与 WebSocket discovery。

### Directory Layout
- `frontend/test-dashboard/src/api/test.ts` — 测试任务 API、报告查询、取消任务、WebSocket discovery。
- `frontend/test-dashboard/src/store/wsStore.ts` — WebSocket 连接状态、消息历史和清理逻辑。
- `frontend/test-dashboard/src/store/testStore.ts` — 当前测试任务状态。
- `frontend/test-dashboard/src/pages/Dashboard.tsx` — 主控制台页面。
- `frontend/test-dashboard/src/pages/Compare.tsx` — A/B 对比页面和轮询逻辑。
- `frontend/test-dashboard/src/pages/Screen.tsx` — 1920×1080 演示大屏模式。
- `frontend/test-dashboard/src/pages/demoTheater.ts` — 用户旅程事件到演示状态的映射模型。
- `frontend/test-dashboard/src/components/` — 进度、时间轴、状态机和演示组件。

### Key Entry Points
- `frontend/test-dashboard/src/App.tsx` — `/test` 与 `/test/screen` 路由入口。
- `frontend/test-dashboard/src/api/test.ts` — 所有测试 API 与 `discoverWS` 入口。
- `frontend/test-dashboard/src/store/wsStore.ts` — WebSocket 生命周期控制。
- `frontend/test-dashboard/vite.config.ts` — React 去重和开发代理配置。

## Gotchas
- 建立新 WebSocket 前必须先关闭旧连接，`connect()` 内部先调用 `disconnect()` 是防止连接泄漏和跨任务串消息的关键约束（`frontend/test-dashboard/src/store/wsStore.ts`）
- Dashboard 页面卸载时必须清理 WS 与全局 store，否则切换页面后会保留幻影进度和旧任务状态（`frontend/test-dashboard/src/pages/Dashboard.tsx`）
- `discoverWS` 使用独立 axios 实例，不走通用 `API_BASE`；修改 WebSocket discovery 时要同时考虑 `VITE_WS_BASE` 和 Nginx `/ws/` 反代（`frontend/test-dashboard/src/api/test.ts`, `deploy/demo/nginx-ip.conf`）
- `recharts` 等依赖可能引入第二份 React 导致 `Invalid hook call`，`vite.config.ts` 的 `resolve.dedupe: ['react', 'react-dom']` 不能随意删除（`frontend/test-dashboard/vite.config.ts`）
- WebSocket 消息历史有最大 200 条限制，新增高频消息类型时不能绕过 `wsStore` 直接无限追加到组件状态（`frontend/test-dashboard/src/store/wsStore.ts`）
- A/B 对比轮询用 ref 持有最新结果，不能把完整响应对象放入 effect 依赖导致 interval 反复重启（`frontend/test-dashboard/src/pages/Compare.tsx`）

## Architecture
- Test Dashboard 的数据流是启动测试拿 `test_id`、通过 `discoverWS(test_id)` 获取 WS URL、WebSocket 收 progress/step/metrics、最终轮询 `getReport(test_id)` 获取报告（`frontend/test-dashboard/src/api/test.ts`, `frontend/test-dashboard/src/store/wsStore.ts`）
- `/test` 是带侧栏的控制台路由，`/test/screen` 是无侧栏大屏模式；演示投屏相关修改应优先检查 Screen 路由而不是 Dashboard 主页面（`frontend/test-dashboard/src/App.tsx`, `frontend/test-dashboard/src/pages/Screen.tsx`）
- 测试类型覆盖压测、E2E、用户旅程、防狙击、回调投递、故障注入和 A/B 对比；新增测试场景应先落在 `src/api/test.ts` 的 API 层，再接入页面状态和可视化组件（`frontend/test-dashboard/src/api/test.ts`）

## Patterns
- `wsStore` 管连接和消息历史，`testStore` 管当前运行任务；跨组件状态不要在页面局部重复实现，否则清理和重连语义会分叉（`frontend/test-dashboard/src/store/wsStore.ts`, `frontend/test-dashboard/src/store/testStore.ts`）
- `StepTimeline` 对同名步骤做 `#N` 编号，后端新增重复 step 时前端不需要强制改名，应该保留编号展示语义（`frontend/test-dashboard/src/components/StepTimeline.tsx`）
- `demoTheater` 将 UserJourney 事件映射为演示状态，新增演示事件应在模型层映射，避免 Screen 组件直接理解后端原始事件细节（`frontend/test-dashboard/src/pages/demoTheater.ts`, `frontend/test-dashboard/src/pages/Screen.tsx`）

## Conventions
- Test Dashboard 开发端口为 5174，用于避开 Admin/H5 常用端口；本地排障端口冲突时不要随意修改主干配置（`frontend/test-dashboard/SKILL.md`, `AGENTS.md`）
- `/api` 和 `/ws` 都应通过 Gateway/Nginx 入口代理，前端不应直连测试服务容器内部地址（`frontend/test-dashboard/vite.config.ts`, `deploy/demo/nginx-ip.conf`）
- 页面级组件放在 `src/pages/`，可复用可视化组件放在 `src/components/`，类型定义和 API 函数同置在 `src/api/test.ts`（`frontend/test-dashboard/src/pages/`, `frontend/test-dashboard/src/components/`, `frontend/test-dashboard/src/api/test.ts`）

## Testing Strategy
- 测试运行态是 HTTP 启动 + WebSocket 进度 + 报告轮询的组合，验证时不能只看启动接口 200，还要确认 WS discovery 返回 JSON 且包含 `ws_url`（`scripts/test-deploy-prod-scripts.sh`, `deploy/demo/MAIN_DEPLOY_QUICKSTART.md`）
- Demo Theater 依赖 UserJourney 的 prepare/enter_live/reminder/auction_bid/sky_lamp/fixed_price_purchase/verify/cleanup 等事件名，后端事件改名会直接影响大屏展示（`frontend/test-dashboard/src/pages/demoTheater.ts`）

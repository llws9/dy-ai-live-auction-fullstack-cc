---
name: knowledge-project-overview
description: >
  Covers project-level navigation for the dy-ai-live-auction-fullstack-cc knowledge tree.
  Navigate when: deciding where to load focused context for H5, Admin, Test Dashboard, backend services, deployment, or project-wide constraints.
  Excludes: detailed module-specific frontend constraints, which live under child nodes.
  Keywords: project overview, knowledge tree, frontend admin, test dashboard, H5, live auction, deployment
---

## Module Structure

本节点是项目知识库入口，用于路由到更具体的知识节点；项目是直播竞拍全栈系统，前端包含 H5 用户端、Admin 管理后台和 Test Dashboard，后端包含 gateway/product/auction/test 等服务。

### Directory Layout
- `frontend/admin/` — 管理后台，覆盖商家/管理员页面、权限、API 封装、统计与管理端测试。
- `frontend/test-dashboard/` — 测试与演示控制台，覆盖测试任务编排、WebSocket 进度流、报告轮询和演示大屏。
- `frontend/h5/` — H5 用户端，覆盖首页、直播间、竞拍、提醒、订单等移动端体验。
- `backend/` — Go 后端服务与测试支撑模块。
- `deploy/`、`scripts/` — 本地和 demo 生产部署入口。

### Key Entry Points
- `AGENTS.md` — 仓库级工程约束、部署与 SDD 执行规则。
- `deploy/demo/MAIN_DEPLOY_QUICKSTART.md` — demo 生产部署目录、入口和验证清单。
- `scripts/deploy-prod.sh` — demo 生产部署脚本入口。
- `scripts/deploy-dev.sh` — 本地开发环境部署脚本入口。

## Gotchas
- 前端流量必须经 `gateway-service` 的 `/api/v1` 入口，新增前端接口时不要直连后端子服务或绕过统一网关（`AGENTS.md`）
- demo 生产环境不能把 Nacos、Prometheus、Loki、GrowthBook 等控制面端口直接公网裸开，观测入口应走受保护的 Nginx 反代（`deploy/demo/MAIN_DEPLOY_QUICKSTART.md`）
- 线上 H5 图片默认值不能依赖 `copilot-cn.bytedance.net` 等内网/IDE 域名，普通浏览器可能无法访问；兜底图应使用同源静态资源（`frontend/h5/src/pages/Home/index.tsx`, `frontend/h5/src/utils/imageFallback.ts`）

## Architecture
- 前端按 H5、Admin、Test Dashboard 拆分，各自独立构建，但 API/WS 公共入口统一由 Gateway 与 Nginx 代理承载（`deploy/demo/MAIN_DEPLOY_QUICKSTART.md`）
- demo 生产发布以远端 `.deploy-ref` 和部署脚本验证为准，不能只用首页 HTTP 200 判断全链路健康（`deploy/demo/MAIN_DEPLOY_QUICKSTART.md`, `scripts/deploy-prod.sh`）

## Conventions
- 技术方案、执行计划和项目知识优先使用中文，代码标识、API/RPC 名称、文件路径和配置键保留 canonical 写法（`AGENTS.md`）
- 知识库节点只沉淀稳定约束、踩坑、架构决策和模块边界，避免把临时执行日志或工具输出写入长期知识（`frontend/admin/SKILL.md`, `frontend/test-dashboard/SKILL.md`）

## Child Knowledge Nodes
- `./frontend/admin/SKILL.md` — Admin 管理后台：角色权限、管理端 API、GrowthBook、编码修复、统计与测试约束。
- `./frontend/test-dashboard/SKILL.md` — Test Dashboard：测试任务、WebSocket 连接、演示大屏、状态管理和报告轮询。

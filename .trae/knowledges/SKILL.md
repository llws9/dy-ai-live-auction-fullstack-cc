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
- `frontend/h5/` — H5 用户端，覆盖首页、直播间、竞拍、提醒、订单等移动端体验。
- `frontend/admin/` — 管理后台，覆盖商家/管理员页面、权限、API 封装、统计与管理端测试。
- `frontend/test-dashboard/` — 测试与演示控制台，覆盖测试任务编排、WebSocket 进度流、报告轮询和演示大屏。
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
- **公网 H5 图片兜底资源必须使用同源静态资源，禁止使用内网域名**。`copilot-cn.bytedance.net` 等 ByteDance 内网域名在公网环境会解析失败（ERR_NAME_NOT_RESOLVED），导致图片无法显示。兜底图应放在 `frontend/h5/public/assets/` 并使用相对路径引用（`frontend/h5/src/pages/Home/index.tsx`, `frontend/h5/src/utils/imageFallback.ts`, `test-service/client/client.go`）
- 后端测试 SDK 造数时使用的图片 URL 也必须使用公网可达资源，不能依赖内网图片生成服务，否则公网环境数据会显示破图（`test-service/client/client.go`）
- **Promtail 镜像版本必须与 Docker API 兼容**。线上 Docker API 1.44+ 不兼容 Promtail 2.9.x 版本，会导致日志采集失败、Loki `service_name` 标签为空。应使用兼容的 Promtail 版本（如 2.8.x 或 3.x）（`deploy/demo/docker-compose.yml`, `scripts/test-deploy-prod-scripts.sh`）
- **Grafana 日志大盘变量查询应使用真实的 Docker 采集流**。Loki 日志流查询应使用 `{job="docker"}` 而非假设的 `service_name` 标签，以确保能正确拉取服务列表（`deploy/demo/grafana/dashboards/microservices-logs.json`）
- **看直播领宝箱系统必须实现后端可信计时**。前端不可信，所有计时逻辑必须在服务端完成，防止用户通过修改本地时间刷币
- **金币资产必须与现金余额隔离**。领宝箱获得的金币是独立资产类型，不能与用户的现金余额混用，避免资产混淆和审计困难
- **宝箱领取必须基于唯一键实现幂等**。同一用户同一宝箱只能领取一次，使用数据库唯一键或分布式锁防止并发重复领取
- **宝箱系统需实现每日分桶重置**。每日观看进度和领取状态按自然日分桶，跨日自动重置，避免状态累积
- **演示模式（剧场模式）需增加状态锁**。防止在实验进行中重复触发，避免并发执行导致状态混乱和结果不可信
- **部署验证必须校验响应体关键字段**。不能仅用 HTTP 200 判断成功，需检查响应体中是否包含关键字段（如 `ws_url`），防止 Nginx 错误返回 HTML 导致静默失败
- **`/dp-prod` 部署脚本的阻断条件**：要求工作区干净且 `HEAD == origin/main`，任一条件不满足都会阻断部署计划生成。本地领先远端的提交不会自动部署，需先 `git push origin main` 同步
- **远端 Compose Project 名称不一致**：当前 demo 生产环境实际运行的容器 project 名为 `app`（如 `app-gateway-1`、`app-auction-1`），但部署脚本默认期望 `auction-demo`。直接使用默认 project name 会导致端口冲突，需显式设置 `COMPOSE_PROJECT_NAME=app` 或执行 project 迁移
- **Compose Project 迁移风险**：切换 project name 会影响命名卷（如 `app_mysql-demo-data` → `auction-demo_mysql-demo-data`），可能导致新容器使用空卷。迁移前必须备份数据卷，或显式复用旧卷

## Architecture
- 前端按 H5、Admin、Test Dashboard 拆分，各自独立构建，但 API/WS 公共入口统一由 Gateway 与 Nginx 代理承载（`deploy/demo/MAIN_DEPLOY_QUICKSTART.md`）
- demo 生产发布以远端 `.deploy-ref` 和部署脚本验证为准，不能只用首页 HTTP 200 判断全链路健康（`deploy/demo/MAIN_DEPLOY_QUICKSTART.md`, `scripts/deploy-prod.sh`）

## Conventions
- 技术方案、执行计划和项目知识优先使用中文，代码标识、API/RPC 名称、文件路径和配置键保留 canonical 写法（`AGENTS.md`）
- 知识库节点只沉淀稳定约束、踩坑、架构决策和模块边界，避免把临时执行日志或工具输出写入长期知识（`frontend/admin/SKILL.md`, `frontend/test-dashboard/SKILL.md`）

## Patterns

### UX 增强开发流程
对于复杂的 UX 增强任务，项目采用以下标准化流程：
1. **brainstorming** — 明确动机、边界和取舍，输出候选方案清单
2. **ui-design-trio** — 对视觉方案进行三版推演（如极简/赛博/仿生），浏览器预览后选定
3. **writing-plans** — 生成详细实施计划（含任务拆分、写集、测试哨兵）
4. **sdd-run** — 按 SDD 协议执行，使用独立 worktree 隔离开发
5. **verification-before-completion** — 本地验证后合并

该流程已在直播间战况热度条 (A1)、演示台剧场模式 (C1) 和 H5 个人中心重构中验证有效。

### H5 个人中心重构决策流程
本次重构展示了从问题定义到落地的完整 UX 决策链：
1. **问题定位** — 通过 brainstorming 明确核心痛点是信息层级混乱（B）和空间利用率低（C）
2. **核心动作识别** — 确定用户最高频动作是「交易闭环」（看竞拍/中标 → 去付款）
3. **足迹功能边界** — 明确纯前端实现（localStorage），避免把 UI 重构撑成大 feature
4. **布局方案对比** — 在「优先级瀑布流」和「交易聚合+服务网格」间选择前者，更贴合现有结构
5. **细节迭代** — 服务区横向图标+文字、角标外移到右上角、数字颜色统一等微观调优
6. **导航链路优化** — 中标待支付 CTA 跳订单页、钱包入口跳独立钱包页、消息通知移到核心统计区
7. **CSS 优先级陷阱** — 数字角标颜色需用精确选择器 `.metricCard .metricBadge` 覆盖，避免被 `.metricCard span` 等更高优先级规则覆盖
8. **悬浮状态陷阱** — 日间模式下主 CTA 悬浮时需显式锁定内部 `strong` 颜色，防止全局 anchor hover 规则导致文字不可见

### 项目亮点文档化流程
当需要将项目成果沉淀为可展示的亮点材料时，采用以下流程：
1. **brainstorming** — 明确亮点叙事主线，区分「业务功能亮点」与「工程技术难点」
2. **载体选择** — 飞书文档适合协作评审和字节内网背书，HTML 展示站适合高表现力可视化和交互演示；两者可组合使用（HTML 嵌入飞书）
3. **内容结构化** — 采用「六大难点」框架组织技术深度：并发控制、状态机、Presence、宝箱系统、故障注入、可观测性
4. **Skill 资产沉淀** — 将开发流程抽象为可复用 Skill（/dp-dev, /dp-prod, /sdd-run, /ui-design-trio），作为工程方法论亮点
5. **部署与回贴** — HTML 站托管至 GitHub Pages，通过链接/截图回贴到交付文档

### HTML 展示站构建要点
- **设计系统**：暗色优先的工程美学，配色使用近黑墨色底 `#0E0F13` / 暖白 `#F7F5F1`，强调色用珊瑚红 `#FF3B5C`（克制使用），次级用青蓝 `#3DD4D0`
- **字体选择**：展示体 `Space Grotesk`，代码 `JetBrains Mono`，中文走 `PingFang SC`（macOS 原生）
- **结构布局**：左侧锚点导航 + 右侧内容流（scrollytelling），架构/并发/链路用 CSS+SVG 手绘图与轻动画
- **交互组件**：在 `#skills` 章节嵌入可交互的「三版直播间战况热度条评审器」，支持点击切换不同版本样式和热度档位
- **主题切换**：HTML 站本身作为项目「日/夜双主题」系统的活体演示，提供主题切换按钮
- **公网部署**：使用独立 GitHub Pages 仓库托管，通过 `.deploy-ref` 版本校验确保部署一致性

来源：session:6a25c5830bfcee1b04fb1c9e, session:6a25c5830bfcee1b04fb1c9e

## Child Knowledge Nodes
- `./frontend/h5/SKILL.md` — H5 用户端：首页、直播间、个人中心、图片兜底策略、足迹功能、移动端布局约束，以及直播间战况热度条 (BidHeatBar) 的 UX 增强决策。
- `./frontend/admin/SKILL.md` — Admin 管理后台：角色权限、管理端 API、GrowthBook、编码修复、统计与测试约束。
- `./frontend/test-dashboard/SKILL.md` — Test Dashboard：测试任务、WebSocket 进度流、演示大屏、状态管理、报告轮询，以及剧场模式 (Chaos Theater) 的 UX 增强决策。

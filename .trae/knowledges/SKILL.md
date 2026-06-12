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
- **Skill vs MCP 决策准则**：优先通过 Skill/Script 编排现有工具，仅在满足「高频复用、需结构化返回、需屏蔽跨平台差异、或 LLM 解析文本易错」时才考虑 MCP 化。环境事实查询是唯一经核实值得做的 P0 级 MCP 候选，其余（运行时诊断、外部系统桥接）已被现有 Skill/MCP 覆盖（`trace-query`、`argos-log`、`mcp_GitHub`、`feishu-cli`）（session:6a2990a00bfcee1b04fc825e）
- **运行事实查询需建立完整关联链路**：验证结论一致性时必须确认「端口/URL -> 进程 -> 工作区 -> Git Commit -> 目标分支」的完整链路，防止在错误分支或陈旧运行环境下进行验证（session:6a2990a00bfcee1b04fc825e）
- 前端流量必须经 `gateway-service` 的 `/api/v1` 入口，新增前端接口时不要直连后端子服务或绕过统一网关（`AGENTS.md`）
- demo 生产环境不能把 Nacos、Prometheus、Loki、GrowthBook 等控制面端口直接公网裸开，观测入口应走受保护的 Nginx 反代（`deploy/demo/MAIN_DEPLOY_QUICKSTART.md`）
- **公网 H5 图片兜底资源必须使用同源静态资源，禁止使用内网域名**。`copilot-cn.bytedance.net` 等 ByteDance 内网域名在公网环境会解析失败（ERR_NAME_NOT_RESOLVED），导致图片无法显示。兜底图应放在 `frontend/h5/public/assets/` 并使用相对路径引用（`frontend/h5/src/pages/Home/index.tsx`, `frontend/h5/src/utils/imageFallback.ts`, `test-service/client/client.go`）
- 后端测试 SDK 造数时使用的图片 URL 也必须使用公网可达资源，不能依赖内网图片生成服务，否则公网环境数据会显示破图（`test-service/client/client.go`）
- **Promtail 镜像版本必须与 Docker API 兼容**。线上 Docker API 1.44+ 不兼容 Promtail 2.9.x 版本，会导致日志采集失败、Loki `service_name` 标签为空。应使用兼容的 Promtail 版本（如 2.8.x 或 3.x）（`deploy/demo/docker-compose.yml`, `scripts/test-deploy-prod-scripts.sh`）
- **API 密钥安全管理规范**：生产环境 AI 服务（如 Ark/Doubao）的 API Key 通过服务器环境文件（如 `/srv/auction/env/.env.demo`）配置，由 `product-service` 容器读取；Gateway 仅负责请求转发和鉴权，不直接调用 AI 接口。严禁将 API Key 提交至 Git 仓库或在对话中明文传输；更新密钥需修改服务器 `.env` 文件并重启对应服务
- **Grafana 日志大盘变量查询应使用真实的 Docker 采集流**。Loki 日志流查询应使用 `{job="docker"}` 而非假设的 `service_name` 标签，以确保能正确拉取服务列表（`deploy/demo/grafana/dashboards/microservices-logs.json`）
- **直播间状态 SSOT 收敛设计**：直播间生命周期状态必须以 product-service `live_streams.status` 为唯一 SSOT，auction-service 只维护提醒投影。商家 start/end 直播都先改直播间状态，Gateway 负责同步调用 auction-service 更新提醒态（session:6a25c4110bfcee1b04fb1b82）
- **直播间空态布局陷阱**：避免使用 `align-items: center` 导致上半屏留白，推荐内容被压到下半屏。应使用 `flex-start` 贴近顶部并保留安全区边距（`frontend/h5/src/pages/Live/`）
- **看直播领宝箱系统必须实现后端可信计时**。前端不可信，所有计时逻辑必须在服务端完成，防止用户通过修改本地时间刷币。前端每 30s 上报心跳，后端按真实间隔累加，单帧封顶 30s
- **金币资产必须与现金余额隔离**。领宝箱获得的金币是独立资产类型，不能与用户的现金余额混用，避免资产混淆和审计困难
- **宝箱领取必须基于唯一键实现幂等**。同一用户同一宝箱只能领取一次，使用数据库唯一键 `(user_id, stat_date, tier)` 防止并发重复领取
- **宝箱系统需实现每日分桶重置**。每日观看进度和领取状态按自然日分桶，跨日自动重置，避免状态累积
- **宝箱系统接口契约**：`POST /watch/heartbeat`（上报时长）、`GET /treasure/status`（查询状态）、`POST /treasure/claim`（领取发币），全部走 Gateway `/api/v1` 入口
- **演示模式（剧场模式）需增加状态锁**。防止在实验进行中重复触发，避免并发执行导致状态混乱和结果不可信
- **部署验证必须校验响应体关键字段**。不能仅用 HTTP 200 判断成功，需检查响应体中是否包含关键字段（如 `ws_url`），防止 Nginx 错误返回 HTML 导致静默失败
- **测试平台部署需按依赖顺序启动**：基础设施 → 业务后端 → test-service → test-dashboard。test-service 未启动或 Gateway 未配置 `/api/test` 路由会导致 404；下游服务（product/auction）异常会导致 `user_journey` 准备阶段 500
- **`/dp-prod` 部署脚本的阻断条件**：要求工作区干净且 `HEAD == origin/main`，任一条件不满足都会阻断部署计划生成。本地领先远端的提交不会自动部署，需先 `git push origin main` 同步
- **部署脚本 `.gitignore` 文件放行规则**：工作区干净检查应忽略命中 `.gitignore` 的文件改动（包括已跟踪但路径命中 `.gitignore` 的构建产物）。使用 `git status --porcelain` 时，需过滤掉 `!!` 标记的被忽略文件，避免构建产物（如 `frontend/admin/dist/index.html`）阻断部署
- **远端 Compose Project 名称不一致**：当前 demo 生产环境实际运行的容器 project 名为 `app`（如 `app-gateway-1`、`app-auction-1`），但部署脚本默认期望 `auction-demo`。直接使用默认 project name 会导致端口冲突，需显式设置 `COMPOSE_PROJECT_NAME=app` 或执行 project 迁移
- **本地部署唯一性约束**：`/dp-dev` 必须保证本机只运行唯一一套开发环境。实现方式：启动前清理其他 worktree 的 Docker 残留容器（按服务名匹配），按固定端口清理后端/前端进程，后端 `screen` session 使用全局固定命名（如 `dy-auction-local-<service>`）而非按 worktree 哈希命名
- **本地后端进程托管方案**：macOS 环境下，`screen` 和 `nohup` 在 Trae sandbox 中无法保持 detached session（进程会被回收）。应使用 `launchctl` 作为持久化托管方案：每个服务生成固定 label 的 LaunchAgent plist，通过 `launchctl bootstrap gui/$UID` 启动，`launchctl bootout` 停止
- **本地账号初始化幂等脚本**：README 中的测试账号（H5 用户、商家、Admin）需通过幂等脚本写入数据库。脚本需处理：已存在账号则更新（upsert）、缺失的认证字段（email/phone/password）自动补齐、密码使用 bcrypt 哈希。脚本应在 `deploy-dev restart` 的基础设施就绪后、后端启动前执行
- **本地数据源一致性验证**：验证 seed 脚本时必须直接链接 Docker 容器内的数据库（`docker exec` 或 `COMPOSE_PROJECT_NAME=dy-ai-live-auction-fullstack-cc`），严禁使用 host fallback（如本地 MySQL），否则会将"当前 worktree / 当前 Docker 数据源"混为一谈，导致验证结果不可信
- **脚本级契约测试**：对部署脚本（`/dp-dev`、`/dp-prod`）的修复必须通过脚本级回归测试验证（如 `scripts/test-deploy-prod-scripts.sh`），确保修复不是靠手工绕过。测试应覆盖：后台进程保活、基础设施 ready 等待、干净 worktree 首次构建等真实故障场景
- **临时表 Collation 兼容性**：seed 脚本创建临时表时需显式指定 `utf8mb4_unicode_ci` collation，避免跨 MySQL 版本或 Docker/MySQL 与 host MySQL 之间的默认 collation 不一致导致脚本失败
- **Compose Project 迁移风险**：切换 project name 会影响命名卷（如 `app_mysql-demo-data` → `auction-demo_mysql-demo-data`），可能导致新容器使用空卷。迁移前必须备份数据卷，或显式复用旧卷
- **Project 迁移执行方案**：统一成 `auction-demo` 需执行维护窗口迁移，步骤为：备份数据和静态资源 → 停 `app` 容器（不删卷）→ 复制 `app_*` 卷到 `auction-demo_*` → 启动 `auction-demo` → seed demo 用户 → 验证全链路。严禁执行 `down -v`，必须保留 `app_*` 卷作为回滚点
- **迁移执行经验**：卷复制时使用国内镜像源（如 `docker.m.daocloud.io/library/alpine:3.19`）避免拉取失败；迁移后必须重新初始化演示账号（`scripts/init-demo-users.sh`）
- **迁移前置检查清单**：
  - [ ] 本地 `HEAD == origin/main` 且工作区干净（无未提交的非 ignored 改动）
  - [ ] 远端备份已完成（数据卷、静态资源、MySQL dump、`.deploy-ref`）
  - [ ] 确认目标 project 名称（`auction-demo`）与脚本期望一致
  - [ ] 确认命名卷映射关系（`app_*` → `auction-demo_*`），避免数据丢失
- **迁移后验证要点**：`scripts/deploy-prod.sh verify` 会检查远端 `.deploy-ref` 与本地 HEAD 是否一致；若迁移过程中该文件未更新或版本不一致，verify 会报错阻断
- **部署脚本回归测试原则**：对 `/dp-dev` 和 `/dp-prod` 的修复必须通过脚本级回归测试验证，确保修复不是靠手工绕过。测试应覆盖：后台进程不被 shell 退出回收、基础设施 ready 等待、干净 worktree 首次构建成功等真实故障场景（`scripts/test-deploy-prod-scripts.sh`）
- **部署脚本干净 worktree 首次构建模式**：`/dp-prod` 部署脚本在干净 worktree（无 `node_modules`）中首次构建前端时会失败，因为 `build_frontend` 步骤缺少 `npm ci` 依赖安装。修复方案：在 `build_frontend` 函数开头显式执行 `npm ci`，确保依赖存在后再构建（`scripts/deploy-prod.sh`）
- **独立测试平台的 WebSocket 访问必须通过 Nginx 反向代理**。公网环境下严禁浏览器直连微服务私有端口（如 `ws://ip:18092`），应统一通过 Nginx 路径（如 `/test-ws`）转发，避免 WS 连接挂起（`deploy/demo/nginx-ip.conf`, `scripts/deploy-prod.sh`）
- **Docker 端口清理策略陷阱**：`deploy-dev.sh` 在清理端口时若把监听 18090/3000 的 `com.docker` 进程当成本项目服务进程 kill 掉，会导致 Docker engine 被杀死（macOS 上 Docker 发布端口由 Docker Desktop backend 进程持有）。修复时应排除 `com.docker` 进程或改用更精确的进程匹配规则（`scripts/deploy-dev.sh`）
- **Skill 与 Command 机制错配陷阱**：`/xxx` 斜杠调用的是 **command**（必须放在 `.claude/commands/`，文件名即命令名），而 skill 不通过 `/` 触发，它靠描述自动召回或经 Skill 工具调用。将 skill 放在 `.trae/skills/` 等非标准目录会导致无法通过 `/` 调用。正确做法：在 `.claude/commands/` 下新建 slash command 作为 `/` 入口，skill 放在 `.claude/skills/` 供自动召回（`session:6a2713650bfcee1b04fb7f9e`）
- **GrowthBook 镜像拉取国内镜像源限制**：国内常用 Docker 镜像代理（如 `docker.m.daocloud.io`）对 `growthbook/growthbook:latest` 返回 403，无法直接拉取。解决方案：在能访问 Docker Hub 的机器上拉取镜像，通过 `docker save | scp | docker load` 方式导入远端服务器，绕过镜像源限制
- **HTTP Client 响应体必须 drain 以保证连接复用**。使用 `http.Client` 进行内部服务调用时，即使不读取响应体也必须执行 `defer io.Copy(io.Discard, resp.Body)`，否则连接无法回收到连接池，高 QPS 下会导致端口耗尽（`backend/product/client/auction_client.go`, `backend/auction/client/live_stream_client.go`）
- **`.trae/skill-config.json` 文件需纳入版本控制**。该文件若作为未跟踪文件存在，会触发 `/dp-prod` 部署脚本的"Git 工作区保护"机制而阻断部署。解决方案：检查文件内容无敏感信息后，提交并推送至 `origin/main`，然后重新生成部署计划（`scripts/deploy-prod.sh`）
- **部署目标提交变更需重新确认**。当部署过程中目标提交发生变化（如先提交了未跟踪文件），必须重新执行 `plan` 并显式确认后才能 `apply`，不能沿用之前的确认（`scripts/deploy-prod.sh`）
- **部署文件提交范围控制**。提交部署相关改动时，应只提交与本次部署直接相关的文件（如 `docker-compose.demo.yml`、环境变量模板、健康检查配置等），避免将无关的 H5 页面改动、`.gitignore` 变更、临时文件等混入部署提交。提交前使用 `git diff --name-only` 和 `git status` 审查变更范围，确保提交内容精准

### Admin 竞拍规则模板功能实现 (Admin Rule Template Integration)

**功能概述**：完成了 Admin 管理后台竞拍规则模板功能的端到端实现，包括模板 CRUD 和「选择模板创建竞拍」的完整链路。

**实现范围**：
1. **后端** (`product-service`)
   - 规则模板 CRUD 接口：`/api/v1/admin/auction-rule-templates`
   - 模板应用接口：`POST /api/v1/admin/products/:id/apply-rule-template`
   - DAO 层 `Upsert` 语义修复：显式字段覆盖，确保 nil/zero 值能正确清空旧字段

2. **前端** (`frontend/admin`)
   - 规则模板管理页面：列表、创建、编辑、删除
   - 创建竞拍表单：选择商品 + 选择模板 → 应用规则 → 创建竞拍
   - API 响应码修复：兼容后端返回的 `code: 201` 成功状态

**关键修复**：
- **Upsert 语义**：旧实现使用 GORM `Assign().FirstOrCreate()`，无法将已有字段清零（如取消封顶价）
- **响应码处理**：前端 `request.ts` 原本只认 `code: 0/200`，后端创建成功返回 `code: 201` 被误判为失败

**与测试平台的兼容性**：
- 独立测试平台不依赖 Admin 模板功能，直接调用 `POST /api/v1/products/:id/rules` 创建规则
- 模板功能仅服务于 Admin 商家后台的「配置复用」场景
- 两者底层共享 `auction_rules` 表，但入口独立，互不干扰

**来源**：session:6a241e023eefb8c530aa78a6

- **Gateway 启动期间的 502 是正常现象**。部署后 `gateway` 容器处于 `health: starting` 状态时，API 接口可能短暂返回 `502`，需等待健康检查通过后再验证
- **线上环境登录账号密码不匹配排查路径**：当用户反馈 README 中的账号密码登录失败时，排查顺序为：(1) 确认 README 账号区分（线上 `13800138001/02/03 + Demo@123456` vs 本地 `18600000001 + 123456`）→ (2) 直接调用线上登录 API 验证账号是否存在 → (3) 检查 bcrypt 哈希是否匹配 → (4) 确认 demo 用户初始化脚本是否在生产环境正确执行
- **H5 开发端口冲突排查**：当 Vite dev server 显示启动成功但浏览器看到旧代码时，可能是端口被多个 Node 进程占用。macOS 上 `localhost` 可能解析到 IPv6 `[::1]`，而旧进程监听 IPv4 `127.0.0.1`，导致同一端口实际有两个服务。排查命令：`lsof -i :<port> | grep LISTEN`，清理旧进程后重试（`frontend/h5/`）
- **INTERNAL_API_TOKEN 运行时必填**：部署/本地启动前需要注入同一个 token 给 gateway 和 auction-service，用于服务间内部鉴权。该 token 通过环境变量配置，禁止硬编码或提交到 Git
- **`net::ERR_ABORTED` 错误诊断**：浏览器控制台出现 `net::ERR_ABORTED` 不代表服务端返回错误，而是请求被浏览器取消。常见场景：页面跳转/刷新/HMR 导致未完成请求被取消、未登录触发 401 后跳转登录页。判断标准：偶发且伴随刷新/切页可忽略；持续每次出现需查登录态和 Gateway。GrowthBook 等特性开关服务的请求失败会降级继续渲染，不影响主业务流程
- **Gateway 角色透传修复**：Gateway 的 `RequireMerchant()` 仅负责入口鉴权，通过后生成 `X-User-ID` 和 `X-User-Role`，但转发到下游时需显式将 `X-User-Role` 加入请求头，否则下游服务无法识别调用者角色（`backend/gateway/handler/proxy.go`）
- **缺失用户 Token 鉴权修复**：当 JWT token 解析出的 `user_id` 在数据库中不存在时，应返回 401 Unauthorized 而非 500 Internal Server Error，避免未登录/过期 token 被误判为服务端故障（`backend/gateway/handler/auth.go`）
- **API 响应 Content-Type 解析陷阱**：前端 API 层若只按 `application/json` 解析响应，但后端返回 `text/plain; charset=utf-8` 包裹的 JSON，会导致解析结果为空/undefined。修复需在 `api.ts` 层增加对 `text/plain` 类型响应的 JSON 解析支持（`frontend/h5/src/services/api.ts`）
- **前端并发请求竞态**：React StrictMode 或路由重渲染可能触发同一接口的重复请求，若后端接口具有「claim 后失效」语义（如开播提醒查询），会导致第一次请求 claim 了资源，第二次请求拿到空结果。需实现模块级请求去重（pending promise 复用）和 token 级 stale 校验（`frontend/h5/src/components/MobileShell/MobileContainer.tsx`）
- **Presence 实名数据隐私边界**：直播间 WebSocket 的 `live_presence_update` 广播包含用户实名信息（`user_id`, `name`, `avatar_url`），必须只向 `Authenticated=true` 的 JWT 鉴权客户端发送；未鉴权连接（历史兼容的 `user_id` query 参数建连）不得接收实名列表，防止用户隐私泄露（`backend/auction/websocket/livestream_room.go`）
- **动画定位必须使用容器相对定位**：直播间动画组件（如成交动画「一锤定音」）若使用 `position: fixed` 会取浏览器视口中心，而非手机预览容器中心，导致偏移。必须使用 `position: absolute` 并确保父级建立定位上下文（`position: relative`）（`frontend/h5/src/pages/Live/Live.module.css`）

### WebSocket 同步状态 fallback 修复 (WebSocket Sync State Recovery)

**问题背景**：线上环境 `sync_request` 返回 `Failed to sync state`，用户进入直播间后状态不同步。

**根因分析**：
- 当前竞拍启动后，`StateManager` 仅在第一次出价时才写入 Redis 同步状态
- 用户进入直播间时，如果 Redis 中没有缓存状态，直接返回错误而不是从 DB 回读
- 新启动的竞拍还没有人出价，所以永远读不到状态

**解决方案**：
1. `sync_request` 收到请求后，如果 Redis miss，从 DB 读取最新竞拍快照
2. 解析快照生成同步状态后，热写入 Redis 供后续请求复用
3. DB 读取失败才返回错误，Redis miss 不直接报错

**关键决策**：
- 所有状态变化（启动竞拍、出价、结束竞拍、取消竞拍）都需要热写 Redis
- 确保最新状态总是可获取，不依赖第一次出价触发写入
- 错误语义修正：Redis miss 不是错误，DB 读失败才是错误

**实现要点**：
```go
// 同步状态获取流程
state := sm.GetSyncState(auctionID)
if state == nil {
    // Redis miss → fallback to DB
    auction, err := a.auctionRepo.GetByID(ctx, auctionID)
    if err != nil {
        return nil, err  // 只有 DB 失败才返回错误
    }
    state = syncStateFromAuction(auction)
    sm.SetSyncState(auctionID, state)  // 热写入 Redis
}
return state, nil
```

**来源**：session:6a1c56f7959156a8dfc84fae

### 商品发布状态过滤原则 (Product Publication Status Filtering)

**问题背景**：管理端标记为"未发布"的商品在 H5 用户端仍可见，导致用户能看到商家尚未准备好的商品。

**根因分析**：
- 后端商品列表接口未过滤 `published` 或 `status` 字段
- 管理端保存商品时状态字段语义与 H5 列表查询条件不一致
- 列表接口返回了全量商品，未区分已发布/未发布状态

**修复方案**：
1. **后端列表查询**：在 `GET /api/v1/products` 等用户端列表接口增加 `status=1`（已发布）过滤条件
2. **契约对齐**：确保管理端保存商品时状态字段与 H5 查询条件使用同一套枚举值
3. **测试覆盖**：增加单元测试验证未发布商品不出现在列表响应中

**关键代码模式**：
```go
// 用户端列表查询：只返回已发布商品
WHERE status = 1 AND deleted_at IS NULL

// 管理端可查看全部状态商品（含草稿/未发布）
// 通过角色权限区分查询范围，而非简单过滤 status
```

**来源**：session:6a23e7b52ec60aa1a73a602c

### 列表页字段缺失导致前端显示异常 (List Page Field Missing Display Issue)

**问题背景**：列表页（如首页收藏直播间卡片）显示的关注人数为 "1" 或归零，而非真实的 "有人关注" 状态。

**根因分析**：
- 直播间详情页会额外查询关注统计（`is_following` / `followers_count`）
- 但首页"收藏"卡片直接渲染列表接口返回的 `followers_count`，该字段缺失时被前端 `toNumber` 归零
- 列表接口未返回该字段，导致前端无法正确显示

**修复方案**：
1. **后端列表接口补全字段**：在 `GET /api/v1/products` 等列表接口响应中补充 `followers_count` 字段
2. **避免前端猜测**：权威计数应由后端提供，避免前端基于不完整数据做推断

**关键代码模式**：
```go
// 列表查询时 JOIN 关注统计表或从 Redis 获取
SELECT ls.*, COALESCE(fs.count, 0) as followers_count
FROM live_streams ls
LEFT JOIN follower_stats fs ON fs.live_stream_id = ls.id
WHERE ...
```

**来源**：session:6a25aaac0bfcee1b04fb1002

---

### 列表接口非核心元数据软依赖原则 (List Interface Non-Core Metadata Soft Dependency)

**问题背景**：列表接口（如首页竞拍列表）常需聚合非核心元数据（如直播间观看人数、主播头像等），这些数据的查询失败不应导致整个列表接口 5xx。

**核心原则**：
1. **软依赖定义**：非核心元数据（如 `viewer_count`、`host_avatar`）是「锦上添花」而非「必不可少」，查询失败时应降级处理而非报错
2. **降级策略**：依赖服务不可用时，字段降级为默认值（如 `viewer_count=0`、头像为空字符串），并记录 Warn 级日志（每请求最多 1 条）
3. **接口稳定性**：列表接口的核心职责是返回主数据（如竞拍列表），非核心元数据查询失败不应破坏主流程

**实现模式**（以观看人数批量回填为例）：
```go
// 批量获取直播间摘要（Redis 优先/DB 兜底）
summaryMap, err := liveStreamClient.BatchGetSummary(ctx, streamIDs)
if err != nil {
    // 降级：记录 WARN 日志，返回空 map，让 viewer_count 默认为 0
    log.Printf("[WARN] batch get live stream summary failed: %v", err)
    summaryMap = make(map[int64]*LiveStreamSummary)
}
// 组装响应时，summaryMap 中不存在的 ID 自动使用零值
```

**前端配合**：
- 展示逻辑应处理零值/空值情况（如 `viewer_count > 0` 才显示角标）
- 不过度依赖非核心字段的存在性

**来源**：session:6a28a0a30bfcee1b04fc5ce6

### 接口契约变更的跨层同步原则 (API Contract Change Synchronization)

**问题背景**：后端修改了 `won_bid` 字段类型（从 number 改为 object），但测试 SDK 仍按 float 解析，导致契约不匹配。

**根因分析**：
- 后端 handler 返回的 `won_bid` 类型从数字改为对象（含 `id`, `amount`, `user_id` 等字段）
- 前端 H5 已更新类型定义，但 `backend/test` SDK 仍使用旧类型
- 契约变更未做到「后端-前端-测试 SDK」三层同步

**修复方案**：
1. **类型定义同步**：修改 `backend/test/client/auction/types.go` 中的 `AuctionResult` 结构体
2. **解析逻辑同步**：更新 SDK 中 `won_bid` 的 JSON 解析逻辑
3. **测试验证**：确保用户旅程测试能正确解析新的 result 结构

**关键教训**：
- 接口契约变更必须同步更新：后端 → 前端 → 测试 SDK
- 测试 SDK 作为调用方也需要防御性编程，不能假设服务端总是返回符合旧契约的数据
- 代码审查时应检查：handler 返回类型、前端消费类型、测试 SDK 类型三者是否一致

**代码审查清单**：
- [ ] Handler 层响应结构变更
- [ ] 前端 API 类型定义更新
- [ ] 测试 SDK 类型定义更新
- [ ] 集成测试验证通过

**来源**：session:6a24716000057ea64ca294db

### 竞拍结束自动创建待支付订单模式 (Auction End Auto-Create Order Pattern)

**问题背景**：用户反馈"我的竞拍"页面显示竞拍成功数量为0，但消息通知中能看到中标记录。根因是竞拍结束产生中标事实后，没有自动创建待支付订单，导致两个数据源不一致。

**解决方案（最小正确方案）**：
在 `auction-service` 结束竞拍时，通过内部 API 调用 `product-service` 创建订单，补全"竞拍中标事实 -> 待支付订单"的领域动作。

**核心改动**：
1. **product-service 新增内部接口**：`POST /internal/orders/from-auction-result`
   - 入参：`auction_id`, `product_id`, `winner_id`, `final_price`
   - 幂等保护：`orders.auction_id` 唯一索引，重复创建直接返回已有订单
2. **auction-service 调用时机**：`EndAuction` 确定 `winner_id/final_price` 后，先创建订单，成功后再发"竞拍中标"通知
3. **顺序保证**：订单创建成功后才发通知，避免"有通知没订单"的状态不一致

**关键代码模式**：
```go
// auction-service EndAuction 中
order, err := productClient.CreateOrderFromAuction(ctx, CreateOrderReq{
    AuctionID: auction.ID,
    ProductID: auction.ProductID,
    WinnerID:  winnerID,
    FinalPrice: finalPrice,
})
if err != nil {
    return err // 订单创建失败，不发送中标通知
}
// 订单创建成功后再发送通知
notificationService.SendAuctionWonNotification(ctx, winnerID, auction)
```

**测试覆盖要点**：
- 正常创建订单路径
- 重复事件幂等（同一 auction_id 多次调用只创建一个订单）
- 订单创建失败时不误报成功（通知不应发出）

**与 Outbox 模式的权衡**：
- 本方案采用同步内部 API 调用，实现简单，适合 MVP/简单支付链路场景
- 如需更高可靠性（订单创建失败可重试），可演进为 Outbox + 异步消费模式

**来源**：session:6a23e4a22ec60aa1a73a5f31

### PC 管理端直播开播定位 (Admin Live Stream Start Positioning)

**设计背景**：PC 管理端需要控制直播间业务状态以跑通 H5 观看、竞拍、一口价交易链路，但真实推流能力计划二期在移动端实现。

**核心决策**：
1. **PC 定位**：PC 管理端是「直播经营控制台」而非「直播设备」，负责配置直播间、商品、竞拍、一口价、开播预告
2. **按钮文案**：保留 `开始直播`，但配合弱提示说明当前为演示开播状态
3. **入口位置**：从 Dashboard `window.prompt` 手输 ID 迁移到直播间详情页，商家在自己直播间详情页执行开播

**角色权限边界**：
| 动作 | 商家 | 管理员 | 说明 |
|------|:---:|:---:|:---|
| 开始直播 | ✅ | ❌ | 商家只能开播自己拥有的直播间，后端 `RequireMerchantOnly` 鉴权 |
| 结束直播 | ❌ | ✅ | 一期仅管理员可关闭，接口为 `RequireAdmin()` |
| 封禁直播间 | ❌ | ✅ | 管理员治理动作，封禁后直播间不可开播 |

**前端实现要点**：
1. **详情页 Scoped 接口**：使用 `/admin/live-streams/:id` 带 owner scope 校验，非 owner 在读取阶段 403
2. **封禁状态保护**：`status === 3` 时禁用开播按钮，不能只依赖后端拒绝
3. **状态枚举补全**：前端 `types.ts` 需补全 `3=已封禁` 注释，与后端 `live_stream.go` 保持一致

**二期预留**：
- 移动端主播页推流凭证生成
- 推流成功回调触发开播状态
- 商家自关闭直播接口（如需）

**来源**：session:6a22c1da2ec60aa1a73a1cdd

## Architecture
- 前端按 H5、Admin、Test Dashboard 拆分，各自独立构建，但 API/WS 公共入口统一由 Gateway 与 Nginx 代理承载（`deploy/demo/MAIN_DEPLOY_QUICKSTART.md`）
- demo 生产发布以远端 `.deploy-ref` 和部署脚本验证为准，不能只用首页 HTTP 200 判断全链路健康（`deploy/demo/MAIN_DEPLOY_QUICKSTART.md`, `scripts/deploy-prod.sh`）

## Conventions
- 技术方案、执行计划和项目知识优先使用中文，代码标识、API/RPC 名称、文件路径和配置键保留 canonical 写法（`AGENTS.md`）
- 知识库节点只沉淀稳定约束、踩坑、架构决策和模块边界，避免把临时执行日志或工具输出写入长期知识（`frontend/admin/SKILL.md`, `frontend/test-dashboard/SKILL.md`）

## Patterns

### 环境事实查询模式 (Runtime Environment Fact Query)

**问题背景**：开发者和 agent 经常在错误的分支或陈旧的运行环境下进行验证，导致验证结论失真。需要建立「端口/URL -> 进程 -> 工作区 -> Git Commit -> 目标分支」的完整关联链路。

**核心对象模型**：
- `Workspace` — 代码工作区绝对路径
- `Branch` — 当前 Git 分支
- `Commit` — HEAD commit hash
- `DirtyStatus` — 是否有未提交改动
- `Process` — 监听端口的进程信息
- `PortBinding` — 端口与进程的映射
- `Container` — Docker 容器状态
- `ServiceEndpoint` — 服务端点 URL
- `RuntimeSource` — 运行事实的统一快照

**最小能力包**：
1. 查询某端口/URL/进程对应的运行主体
2. 查询某服务当前代码来源（worktree + commit）
3. 输出统一的运行事实快照
4. 比较「目标代码」与「实际运行代码」是否一致

**Skill vs MCP 边界判据**：

| 维度 | 做 Skill | 做 MCP |
|------|----------|--------|
| 核心价值 | 理解、编排、解释、归因、决定下一步 | 稳定取事实、结构化返回、封装跨平台差异、被多个 Skill 复用 |
| 适用场景 | 流程脑力工作（方案规划、代码审查、知识提炼） | 高频、易错、需确定性解析的环境查询 |
| 实现成本 | 低（prompt + 脚本编排） | 高（独立 server、协议维护） |

**推荐路径（YAGNI 原则）**：
1. **先做成 Skill/Script**：把运行事实快照用一个 skill 固化，跑几次验证高频性和实用性
2. **当出现这两个信号之一再 MCP 化**：① 多个项目复用；② LLM 解析 shell 文本出过错导致误判
3. **避免重复建设**：`trace-query`、`argos-log`、`mcp_GitHub`、`feishu-cli` 已覆盖运行时诊断和外部系统桥接，无需新建 MCP

**来源**：session:6a2990a00bfcee1b04fc825e

---

### 历史会话审计方法论 (Historical Session Audit Methodology)

**问题背景**：项目积累大量历史会话后，需要从中提炼可复用的 workflow 或 skill，但缺乏系统性的审计方法。

**审计流程**：
1. **范围划定**：确定审计的时间窗口（如最近 7-14 天）和会话来源（memory/history 目录）
2. **主题聚类**：按 session 主题进行初步分组，识别重复出现的模块或问题域
3. **频次统计**：统计每个主题的出现次数，按「复现频率 × 单次耗时」排序
4. **痛点分级**：
   - **P0**：绝对高频，几乎每天出现，阻塞开发
   - **P1**：高频但非每日，有明显 workaround
   - **P2**：偶发，但单次排查耗时较长
   - **P3**：一次性问题，复用价值低

**输出物**：
- 痛点频次报告（按 P0/P1/P2/P3 分级）
- 每个痛点的具体形态、根因分析、现有覆盖情况
- 推荐沉淀形态（Skill/Script/Knowledge/无需新建）

**关键判断原则**：
- 区分「事实查询」与「诊断决策」：前者只给事实，后者需要推理和决策树
- 区分「通用流程」与「项目特有」：通用流程可能已被 RUNBOOK 覆盖
- 避免重复造轮子：先核对现有 skills、MCPs、docs 是否已覆盖

**来源**：session:6a29adb40bfcee1b04fc8fa9

---

### 环境事实查询增强设计 (Runtime Facts Staleness Diagnosis)

**问题背景**：`runtime-facts` 仅提供「代码版本对不对」的事实查询，但开发者最常遇到的「改了不生效」问题需要判断「进程是否吃进最新改动」。

**增强目标**：将 runtime-facts 从「代码版本验证」升级为「验证可信度诊断」，建立「改了不生效」的诊断决策树。

**Script 层新增 Finding**：

| Finding | 判定条件 | 说明 |
|---------|----------|------|
| `NO_LISTENER` | 目标端口无进程监听 | 服务未启动 |
| `MULTIPLE_LISTISTENERS` | 同一端口多个进程 | 端口冲突，需清理旧进程 |
| `PROCESS_CWD_OUTSIDE_REPO` | 进程工作区不在当前仓库 | 可能跑了其他项目/旧 worktree |
| `DIRTY_TREE_NOT_DEPLOYED` | 工作区有未提交改动且进程 commit 落后 | 改动未部署 |
| `STALE_PROCESS_BEFORE_CHANGE` | 进程启动时间早于文件修改时间 | 进程未重启，未吃进改动 |

**Skill 层诊断决策树**：
```
验证失败/改了不生效
├── 端口无监听 → 服务未启动，启动服务
├── 多进程冲突 → 清理旧进程，重启服务
├── 进程工作区异常 → 检查是否跑了其他 worktree
├── 有未提交改动 → 提交后重新部署
└── 进程启动时间早于改动时间 → 重启进程
    └── 重启后仍无效 → 检查 Docker 缓存/Vite 缓存/浏览器缓存
```

**Stale 判定严谨性设计**（4点补充）：

1. **时间比较公式**：统一转为 timezone-aware `epoch seconds` 后比较
   ```text
   stale_threshold = code_changed_at_epoch - stale_tolerance_seconds
   if process_started_at_epoch < stale_threshold:
       emit STALE_PROCESS_BEFORE_CHANGE
   ```

2. **`code_changed_at` 计算**：取以下两者最大值
   - HEAD commit time
   - dirty runtime-input 文件的最新 mtime

3. **`runtime-input files` 白名单**（避免文档改动误报）：
   `backend/**`, `frontend/**/src/**`, `frontend/**/public/**`, `frontend/**/package*.json`, `frontend/**/vite.config.*`, `docker-compose*.yml`, `deploy/**`, `scripts/deploy*.sh`, `scripts/start*.sh`, `nginx*.conf`, `*.env.example`
   - 纯 `docs/**`、测试报告、构建缓存变更不触发 stale

4. **容差参数可配置**：
   - 默认 `STALE_TOLERANCE_SECONDS = 5`
   - CLI 暴露 `--stale-tolerance-seconds` 参数
   - 测试显式传参，避免硬编码

**误报控制**：
- `STALE_PROCESS` 判定需有明确的进程启动时间戳，缺证据时不报
- 时间比较使用 5 秒容差，避免边界抖动
- 跨平台解析失败时降级为提示而非错误

**与现有资产的关系**：
- 复用 `runtime-facts` 已采集的事实
- 仅对 stale 判定补充 `ps -o lstart` 等最小增量采集
- 诊断逻辑作为 Skill 层编排，不重复建设底层能力

**来源**：session:6a29adb40bfcee1b04fc8fa9

---

### 本地开发部署唯一性约束模式 (Local Dev Deployment Uniqueness)

**问题背景**：多 worktree 开发时，每个 worktree 可能启动独立的 Docker 容器和后台进程，导致端口冲突、数据库连接混乱、资源浪费。

**核心约束**：
1. **服务唯一性**：同一时刻本机只能运行一套 MySQL/Redis/RabbitMQ 基础设施
2. **进程唯一性**：后端服务（gateway/product/auction）只能有一组进程在监听固定端口
3. **命名唯一性**：Docker 容器名、screen session 名、launchctl label 使用全局固定标识，不按 worktree 隔离

**实现策略**：
```bash
# 1. 清理其他 worktree 的 Docker 残留（按服务名匹配）
docker ps --filter "name=mysql|redis|rabbitmq" --format "{{.Names}}"
# 删除非当前 project 的同名服务容器

# 2. 端口占用清理
lsof -i :8080 | grep LISTEN | awk '{print $2}' | xargs kill -9
lsof -i :5173 | grep LISTEN | awk '{print $2}' | xargs kill -9

# 3. 进程托管统一使用 launchctl（macOS）
# plist label 格式：dy.auction.local.<service>
# 示例：dy.auction.local.gateway, dy.auction.local.product
```

**与远端部署的区别**：
| 维度 | 本地 `/dp-dev` | 远端 `/dp-prod` |
|------|---------------|----------------|
| 进程托管 | launchctl / screen | Docker 容器 |
| 唯一性保证 | 清理同名服务容器 + 固定端口 | 固定 COMPOSE_PROJECT_NAME |
| 状态检查 | PID 文件 + 端口监听 | 容器状态 + 健康检查 |
| 日志查看 | `tail -f .tmp/local-backend/*.log` | `docker logs <container>` |

**launchctl 使用要点**：
```bash
# 生成 plist 并启动
cat > ~/Library/LaunchAgents/dy.auction.local.gateway.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" ...>
<plist version="1.0">
<dict>
    <key>Label</key><string>dy.auction.local.gateway</string>
    <key>ProgramArguments</key><array><string>/path/to/gateway</string></array>
    <key>RunAtLoad</key><true/>
    <key>KeepAlive</key><true/>
</dict>
</plist>
EOF
launchctl bootstrap gui/$UID ~/Library/LaunchAgents/dy.auction.local.gateway.plist

# 停止
launchctl bootout gui/$UID ~/Library/LaunchAgents/dy.auction.local.gateway.plist
```

**来源**：session:6a22abbf2ec60aa1a73a0cac

### 远端 Compose Project 唯一性检查模式 (Remote Compose Project Uniqueness)

**问题背景**：远端服务器可能因手动操作或历史遗留存在多个 Docker Compose project，导致同一套服务（如 mysql/gateway）被多次部署，产生端口冲突或数据不一致。

**核心约束**：
1. **固定 Project Name**：所有远端 compose 命令必须显式指定 `-p auction-demo` 或设置 `COMPOSE_PROJECT_NAME=auction-demo`
2. **预检扫描**：`plan` 阶段扫描远端是否存在非目标 project 的同名服务容器
3. **阻断机制**：发现冲突时直接失败并提示清理，不静默继续

**检查逻辑**：
```bash
# 扫描是否存在其他 project 的同名服务
ssh $REMOTE "docker ps --format '{{.Names}}' | grep -E '^(mysql|gateway|auction|product)-' | grep -v '^auction-demo-'"
# 若有输出，说明存在非 auction-demo project 的同名服务，需阻断部署
```

**修复路径**：
1. 备份数据卷（如有需要）
2. 停止并删除旧 project 容器：`docker compose -p <old-project> down`
3. 确认无残留后重新部署

**与本地部署的区别**：
- 本地靠「清理同名容器 + 固定端口」保证唯一性
- 远端靠「固定 project name + 预检扫描」保证唯一性
- 远端不依赖目录路径推导 project name，显式指定避免歧义

**来源**：session:6a22abbf2ec60aa1a73a0cac

### 部署脚本就绪等待门模式 (Deployment Readiness Gate Pattern)

**问题背景**：全量服务部署时，容器启动顺序和数据库迁移时序可能导致竞态问题（如 `init_demo_users` 的 `ALTER TABLE` 在 `users` 表创建前执行）。

**核心设计**：
1. **`wait_for_users_table`**（数据库表就绪）：轮询 `information_schema.TABLES` 直到 product 容器完成 `users` 表迁移，避免 seed 脚本因 "Table doesn't exist" 中断
2. **`wait_for_http_ready`**（本地 HTTP 就绪）：在 `init_demo_users` 之后、`verify_local` 之前轮询慢启动服务（gateway 8080 / product 8081 / nacos 8848 / growthbook 3200）
3. **`wait_for_remote_http_ready`**（远端 HTTP 就绪）：在 `restart_remote` 之后、`verify_prod` 之前软等待三个对外 URL

**软等待 vs 硬校验**：
- **软等待**：超时仅告警不阻断，把硬失败判定留给后续的 `verify_local`/`verify_prod`，不掩盖真实故障
- **硬校验**：`verify_local`/`verify_prod` 和 `verify_remote_containers` 承担最终失败判定

**实现要点**：
```bash
# 本地 HTTP 就绪等待示例
wait_for_http_ready() {
  local services=("http://localhost:8080/" "http://localhost:8081/" "http://localhost:8848/nacos")
  for url in "${services[@]}"; do
    for i in {1..30}; do
      if curl -s "$url" >/dev/null; then break; fi
      sleep 2
    done
  done
}
```

**TDD 验证**：
- 先写失败测试（红灯）验证等待行为
- 再实现等待逻辑（绿灯）
- 测试脚本 `test-deploy-dev-scripts.sh` / `test-deploy-prod-scripts.sh` 覆盖就绪门断言

**来源**：session:6a2713650bfcee1b04fb7f9e

### 项目部署 Skill 设计规范 (Project Deploy Skill Design)

**Skill 定位**：将项目部署流程产品化为可调用命令 `/dp-dev`（本地部署）和 `/dp-prod`（线上部署）。

**核心设计决策**：
1. **命令式触发**：通过 `/dp-dev` 和 `/dp-prod` 文本触发，而非多轮对话
2. **安全门禁**：
   - 强制检查 `HEAD == origin/main`，本地领先远端的提交不会自动部署
   - 工作区干净检查（可配置忽略 `.gitignore` 文件）
   - 部署前生成 plan，用户确认后才执行 apply
3. **验证闭环**：部署后自动执行 verify，检查关键接口和 WebSocket 可用性

**Skill 与 Command 机制区分**：
- **Skill**：通过描述自动召回或经 Skill 工具调用，不通过 `/` 触发
- **Command**：通过 `/xxx` 斜杠语法触发，必须放在 `.claude/commands/` 目录，文件名即命令名
- **正确做法**：
  - Skill 定义放在 `.claude/skills/project-deploy/SKILL.md`，供自动召回
  - Command 入口放在 `.claude/commands/dp-dev.md` 和 `.claude/commands/dp-prod.md`，内容为调用 Skill 的委托

**常见陷阱**：
- 将 skill 放在 `.trae/skills/`（错误目录）会导致无法识别
- 将 command 内容直接写在 skill 文件中，期望用 `/` 触发，会失败

**Skill 存放位置规范**：
- **项目内**：`.claude/skills/project-deploy/SKILL.md` —— 随仓库同步，团队成员共享
- **全局**：`~/.trae-cn/skills/project-deploy/SKILL.md` —— 个人本地缓存，可能落后于项目版本
- **推荐**：优先使用项目内 skill，确保团队一致性

**触发文本与 skill 名称关系**：
- `/dp-dev` 和 `/dp-prod` 不是独立的 skill 文件名
- 它们是 `project-deploy` 这个 skill 的触发文本（triggers）
- 在 `SKILL.md` 的 frontmatter 中声明：
```yaml
---
name: project-deploy
triggers: ['/dp-dev', '/dp-prod']
---
```

**脚本结构**：
- `scripts/deploy-dev.sh` —— 本地部署，支持 `status|restart|verify` 子命令
- `scripts/deploy-prod.sh` —— 线上部署，支持 `plan|apply|verify` 子命令

**来源**：session:6a2138fc2ec60aa1a7394fdf, session:6a2713650bfcee1b04fb7f9e

### 全栈功能开发 Workflow (Fullstack Feature Development Workflow)

项目采用标准化的 5 阶段全栈开发流程（开发域，不含部署），仅适用于需要落库、多步骤实现、跨前后端协作或需要 SDD 的开发任务；一次性问答、纯分析、简单文案和无需落库的小任务走轻量通道。

```
[0] 需求澄清     brainstorming        → spec.md
[1] UI 设计      ui-design-trio (等)   → 选定 UI 稿
[2] 契约先行     brainstorming(轻)     → 契约 SSOT（见必填项）
[3] 计划拆分     writing-plans        → plan.md + tasks.md (+checklist)
[4] 实现波次     sdd-run              → 按交付域实现 + TDD 证据（按契约）
[5] 知识沉淀评估  knowledges-update    → 更新知识树或记录 no-op
```

**核心原则**：
- **契约先行**：接口契约优先于实现，作为前后端单一事实源（SSOT），避免 mock 与真实后端语义漂移
- **writing-plans 是开发型 brainstorming 进入 SDD 前的前置条件**：不允许从 spec 直接跳到 sdd-run；纯讨论/纯文档类 brainstorming 不受此约束
- **实现波次按交付域拆分**：阶段 4 不强制区分"前端波次/后端波次"，而是按实际交付域（前端/后端/跨服务）拆分；串并行由 state 的 `Wave Plan` 与 `Parallel Group` 根据契约冻结状态、依赖、write set 和本地服务占用决定
- **适用范围内不可跳**：需求澄清、计划拆分和知识沉淀评估；轻量通道任务除外
- **最小路径内嵌 task 例外**：单交付域小改（如纯前端排序下拉）可内嵌 task 定义，免写独立 `plan.md`/`tasks.md`，但仍须声明 scope/write set/read set/regression sentinel/verification command

**契约 SSOT 最小合格定义**（阶段2输出必须包含）：
- 接口路径与方法必须走 `gateway-service` 的 `/api/v1` 入口（如 `GET /api/v1/products/:id`），禁止前端直连子服务
- 请求/响应字段名、类型、是否必填
- 鉴权字段：是否需要 JWT，下游身份统一使用网关派生的 `X-User-ID`，禁止硬编码用户身份或内部 Token
- 错误码与 HTTP 状态码映射
- 金额字段必须使用 `shopspring/decimal`（AGENTS.md 硬约束），禁止 float
- 分页参数规范（如 `page`, `page_size`, `total`）
- 跨服务依赖需声明调用方、被调用方、RPC/API 路径与降级语义；禁止跨服务直接查库

**契约一致性原则**：
- `grep -R "<api-path-or-field>" -n frontend backend docs` 只是 regression sentinel 的最低档，只能证明字符串存在
- 核心/跨服务接口必须额外配至少一种：OpenAPI 校验、type check、API 测试或前后端 mock parity
- 前后端并行前必须冻结契约；开发中若改契约，先更新契约 SSOT，并在 state 的 `API Contract Changes`、`Cross-Task Decisions` 和 `Wave Plan` 中登记影响与重新对齐条件

**规模驱动的拆分形态**：
- **默认形态（小/中功能）**：一份 plan 拆前后端两组 tasks，共享上下文
- **大需求形态**：命中以下任一判据时，按受影响交付域/服务各自独立走一轮 writing-plans + sdd-run
  - 涉及跨服务调用或数据契约
  - 涉及数据模型或 DB migration
  - 需要并行多 subagent，或存在共享 write set 需隔离
  - 接口/页面数量多、状态机复杂、兼容性风险高，单份 plan 难以承载
  - **注意**："按交付域拆分"意味着有前端才拆前端，纯后端跨服务需求不应创建空的前端计划

**知识沉淀评估原则**：
- 适用范围内的功能闭环后必须评估是否更新知识库
- 仅针对新增约束、非直观决策或可复用经验进行持久化
- 无 durable knowledge 时记录 no-op，避免将临时执行日志、工具输出、协调信息写入长期知识

**最小闭环路径表**（根据任务类型选择必经阶段）：

| 任务类型 | 必经阶段 | 跳过阶段 | 说明 |
|---------|---------|---------|------|
| Fullstack（前后端+契约） | `[0][1][2][3][4][5]` | 无 | 完整流程 |
| Frontend-only，无契约变更 | `[0][1][3][4][5]` | `[2]` 契约 | 纯前端改动，接口不变 |
| Backend-only，无契约变更 | `[0][3][4][5]` | `[1][2]` | 纯后端改动，无 UI |
| 轻量通道（简单文案/配置/一次性问答） | 走轻量通道 | `[0-5]` | 见第 0 节适用范围 |

**计划形态选择**：
- **标准形态**：独立 `plan.md` + `tasks.md` + 可选 `checklist.md`
- **最小形态**：task 定义内嵌于 state 文件或 sdd-run 输入，适用于单交付域小改（如纯前端排序下拉），但仍须声明 scope/write set/read set/regression sentinel/verification command

**判定顺序**：
1. 先判是否命中轻量通道（一次性问答、纯分析、简单文案、无需落库）
2. 再判 contract impact（是否改接口/改契约）
3. 最后判交付域（前端/后端/全栈）

**来源**：session:6a29bfdc0bfcee1b04fc9f45, `docs/superpowers/specs/2026-06-11-fullstack-feature-workflow.md`

**验证记录**：
- 已完成 Skill 产品化（`.agents/skills/fullstack-feature-workflow/SKILL.md`）
- 已通过 3 个 eval prompts 轻量评测（主路径/边界条件/跳过规则）
- 已完成真实场景试运行：Admin 商品列表纯前端排序下拉（`feat/admin-goods-sort` → `main`）
- 试运行发现流程缺口并修补：Stage 3 增加「最小路径内嵌 task 例外」档位

**阶段编号语义统一原则**：
- 原设计使用 `[4] Frontend wave / [5] Backend wave`，但在纯后端/多服务场景产生歧义
- 修正为 `[4] Implementation wave`（实现波次），按交付域而非端绑定
- 最小闭环表同步调整：`Frontend-only` → `[0][1][3][4][5]`，`Backend-only` → `[0][3][4][5]`

**Workflow 试运行方法论**：
1. **选择最小有效验证任务**：选一个小但完整的功能（如纯前端排序下拉），能验证轻量通道、跳过规则、writing-plans 路由
2. **隔离执行**：使用独立 worktree（`git worktree add`）执行，避免污染 main
3. **验证要点**：
   - Skill 是否正确分流（选最小路径、跳过契约、UI 降级单方案）
   - 是否强制经 writing-plans 再 sdd-run，而非直接改代码
   - TDD red-green 证据是否完整
4. **缺口修补**：试运行中发现的流程缺口应同步回 Skill 文档
5. **闭环决策**：试运行代码选择提交保留或丢弃，不强制合入

**来源**：session:6a29bfdc0bfcee1b04fc9f45, `docs/superpowers/specs/2026-06-11-fullstack-feature-workflow.md`

---

### 文档断链预防 (Documentation Link Integrity)

**问题背景**：workflow 文档引用了 `layered-dev-methodology.md`，但该文件未入库（untracked），导致提交后存在断链风险。

**根因分析**：
- 文档中使用了相对路径引用其他文档（如 `./2026-06-11-layered-dev-methodology.md`）
- 被引用文件存在于工作区但未被 Git 跟踪
- 其他开发者拉取代码后将看到断链

**预防措施**：
1. **提交前核实引用**：使用 `git ls-files` 确认所有仓库内引用目标已入库
2. **断链修复策略**：
   - 优先将被引用文档入库（最干净方案）
   - 或修改为不依赖未入库文档的描述
3. **审查清单**：多角度审核时必须包含「文档引用完整性」检查项

**验证命令**：
```bash
# 检查 workflow 文档引用的文件是否已入库
git ls-files | grep "layered-dev-methodology"
# 输出为空表示未入库，存在断链风险
```

**来源**：session:6a29bfdc0bfcee1b04fc9f45

---

### SDD 启动命令完整性 (SDD Launch Command Completeness)

**问题背景**：文档中写的 `sdd_run.py` 路径不完整，agent 照抄可能找不到或执行失败。

**根因分析**：
- 简写路径：`python3 docs/superpowers/sdd/scripts/sdd_run.py`
- 真实入口需要额外参数：`--repo-root . --input "<用户的 /sdd-run 输入>"`
- 不完整命令会导致执行失败或行为不符合预期

**正确做法**：
1. **优先使用 `/sdd-run` 指令**：这是推荐入口，Skill 自动处理参数
2. **脚本等价入口**（需完整参数）：
   ```bash
   python3 docs/superpowers/sdd/scripts/sdd_run.py --repo-root . --input "<用户输入>"
   ```
3. **文档中注明**：若写脚本路径，必须带完整参数示例

**关联约束**：
- 启动命令完整性检查应纳入 `verification-before-completion` 阶段
- 命令变更需同步更新文档和 Skill 定义

**来源**：session:6a29bfdc0bfcee1b04fc9f45

---

### 契约冻结状态落点 (Contract Freeze State Tracking)

**问题背景**：workflow 要求"契约确认后标记为 frozen"，但未明确 frozen 记录在哪里，导致后续难以追踪契约版本和变更影响。

**解决方案**：
在契约 SSOT 与 state 文件的多个表中记录冻结状态：

1. **Cross-Task Decisions 表**：
   - `contract_version`: 契约版本号
   - `frozen_at`: 冻结时间戳
   - `owner`: 契约负责人

2. **API Contract Changes 表**：
   - 记录每次契约变更的影响范围
   - 标记受影响波次和重新对齐条件

3. **Wave Plan 表**：
   - 记录各波次对契约版本的依赖
   - Start/Completion Condition 中声明契约版本要求

**冻结闸门机制**：
- 契约确认后显式标记 frozen
- 开发中若需改契约，先更新 SSOT
- 在 state 表中登记影响，阻塞受影响波次直到重新对齐

**来源**：session:6a29bfdc0bfcee1b04fc9f45

---

### Skill vs Command 机制区分 (Skill vs Command Mechanism)

**关键区分**：
- **Skill**：通过描述自动召回或经 Skill 工具调用，**不通过 `/` 触发**
- **Command**：通过 `/xxx` 斜杠语法触发，必须放在 `.claude/commands/` 目录，文件名即命令名

**常见陷阱**：
- 将 skill 放在 `.trae/skills/`（错误目录）会导致无法识别
- 将 command 内容直接写在 skill 文件中，期望用 `/` 触发，会失败
- 混淆 Skill 的 `triggers` frontmatter 与 Command 的斜杠调用机制

**正确做法**：
- Skill 定义放在 `.claude/skills/<skill-name>/SKILL.md`，供自动召回
- Command 入口放在 `.claude/commands/<command-name>.md`，内容为调用 Skill 的委托
- Skill 的 `triggers` 仅用于描述匹配，不是斜杠命令

**项目内 Skill 存放位置**：
- `.agents/skills/<skill-name>/SKILL.md` —— 随仓库同步，团队成员共享
- 包含 `evals/evals.json` 用于轻量评测
- 包含 `README.md` 说明触发条件和边界

**来源**：session:6a29bfdc0bfcee1b04fc9f45

---

### Skill 评估与验证方法论 (Skill Evaluation Methodology)

**问题背景**：新创建的 Skill 需要验证其触发边界、避免与现有 Skill 重叠、确保可执行性。

**评估流程（轻量 dry-run 法）**：
1. **设计 3 个 eval prompts**：覆盖主路径、边界条件、跳过规则
2. **手工 dry-run**：不真正改业务代码，只验证 Skill 的阶段判断和路由决策
3. **审查输出**：确认阶段判断、跳过规则、下一 skill 路由符合预期
4. **边界不重叠验证**：确保新 Skill 不会抢 `brainstorming` / `writing-plans` / `sdd-run` 的职责
5. **微调并提交**：根据 dry-run 结果调整 `description` 和正文，提交 refinement

**evals.json 断言设计**：
- 为每个 eval 增加 `expectations` 字段，覆盖触发、阶段路由、跳过规则、边界不重叠
- 断言应客观可检查，不依赖主观判断
- 示例断言：
  - `triggered_skill`: "fullstack-feature-workflow"
  - `current_stage`: 0
  - `next_action`: "invoke brainstorming"
  - `skip_stages`: [2]
  - `boundary_check`: "does not implement sdd-run details"

**编排型 Skill 的特殊考量**：
- 编排型 Skill（如 `fullstack-feature-workflow`）只决定阶段、输入输出、跳过规则和何时交给子 Skill，不重复实现子 Skill 的细节
- 第一性原理：先验证"触发和边界"是否正确，再继续扩展功能
- 避免与 `brainstorming`、`writing-plans`、`sdd-run` 等现有 Skill 重叠
- 阶段编号语义应与交付域解耦（如用"实现波次"而非"前端波次/后端波次"）

**试运行（Real-World Validation）阶段**：
1. **选择最小有效验证任务**：选一个小但完整的功能（如纯前端排序下拉），能验证轻量通道、跳过规则、writing-plans 路由
2. **隔离执行**：使用独立 worktree（`git worktree add`）执行，避免污染 main
3. **验证要点**：
   - Skill 是否正确分流（选最小路径、跳过契约、UI 降级单方案）
   - 是否强制经 writing-plans 再 sdd-run，而非直接改代码
   - TDD red-green 证据是否完整
4. **缺口修补**：试运行中发现的流程缺口（如最小路径内嵌 task 例外）应同步回 Skill 文档
5. **闭环决策**：试运行代码选择提交保留或丢弃，不强制合入

**来源**：session:6a29bfdc0bfcee1b04fc9f45

---

### UX 增强开发流程 (UX Enhancement Development Process)
对于复杂的 UX 增强任务，项目采用以下标准化流程：
1. **brainstorming** — 明确动机、边界和取舍，输出候选方案清单
2. **ui-design-trio** — 对视觉方案进行三版推演（如极简/赛博/仿生），浏览器预览后选定
3. **writing-plans** — 生成详细实施计划（含任务拆分、写集、测试哨兵）
4. **sdd-run** — 按 SDD 协议执行，使用独立 worktree 隔离开发
5. **verification-before-completion** — 本地验证后合并

该流程已在直播间战况热度条 (A1)、演示台剧场模式 (C1)、H5 个人中心重构、出价排行视觉优化、直播间倒计时与流拍动画、直播间互动 UI 升级中验证有效。

**来源**：session:6a27ede70bfcee1b04fbc3b6

### 直播间互动 UI 升级设计决策 (Live Room Interactive UI Upgrade)

**决策背景**：通过 `brainstorming` Skill 探索直播间互动增强机会，最终选定 9 项 UI 升级点进行系统化设计。

**探索流程**：
1. **功能地图绘制**：标注现有互动模块，识别 6 个优化机会区（出价氛围感、社交互动、紧迫感&转化、信息层级/UI、引导&留存、视觉打磨）
2. **候选清单收敛**：将机会区拆分为 15 个具体点子，标注价值/成本（快赢/中等/较重）
3. **用户多选确认**：用户在浏览器预览界面勾选本次要做的功能点

**最终选定的 9 项设计点**：

| 类别 | 编号 | 功能点 | 设计要点 |
|------|------|--------|----------|
| 氛围感 | a1 | 出价飘屏升级 | 他人出价时飘入头像+金额+连击数，自己出价用高亮色，错位堆叠 |
| 氛围感 | a2 | 领先/被超状态条 | 出价坞顶部窄条：领先=金色脉冲，被超=红色抖动+快捷反超按钮 |
| 氛围感 | a3 | 加价震动+音效 | `navigator.vibrate` + 轻量音效，<10s 进入心跳节奏，预留静音开关 |
| 社交 | s1 | 点赞红心 | 右下角红心按钮，双击/点按触发飘心动画（随机轨迹/大小/向上飘散） |
| 社交 | s2 | 快捷弹幕气泡 | 发言入口旁预设短语（+1/冲/捡漏了/还能加吗），点按即发 |
| 紧迫感 | c1 | 倒计时强化 | <30s 显示进度环+颜色渐变（橙→红），<10s 数字放大+全屏边缘红光呼吸 |
| 紧迫感 | c2 | 实时热度文案 | 价格区下方显示「🔥 已有 N 人出价 · M 人围观」，数字变化时滚动/跳动 |
| 视觉 | p1 | 主播区/状态胶囊精致化 | 统一毛玻璃、圆角、阴影；直播中红点呼吸动效；日/夜对比度对齐 |
| 视觉 | p2 | 渐变遮罩可读性 | 优化 `.videoGradient`，顶部/底部按内容区动态加深，保证文字可读 |

**设计交付物**：
- 整体视觉方向说明（配色取向、动效基调、层级原则）
- 逐点 HTML 结构（JSX 片段）+ CSS Module 样式（含 dark/light 两套）
- 明确标注每个新元素需要的数据字段（用 `// TODO: 需要 props.xxx` 注释）
- 改动清单：文件、类名、测试更新需求

**技术约束**：
- 框架：React + TypeScript + CSS Modules，禁止引入新 UI 库
- 主题：`<html data-theme="dark"|"light">` 控制，使用项目设计 token
- 动画：优先 `transform`/`opacity`，支持 `prefers-reduced-motion`
- 边界：只改视觉/交互层，不改动数据来源、WebSocket、API、状态管理

**设计文档**：`docs/superpowers/specs/2026-06-09-live-room-ui-design.md`

**来源**：session:6a26eb200bfcee1b04fb47ca

### 直播间动画设计决策模式 (Live Room Animation Design Pattern)

**问题背景**：竞拍关键节点（倒计时、成交、流拍）需要强烈的视觉反馈，但直接在业务代码中试错成本高。

**设计流程**（以倒计时/流拍动画为例）：

**1. 需求澄清**
- 明确动画目标：紧迫感营造（倒计时）/ 结果确认（成交/流拍）
- 确定情感基调：赛博紧张感 / 高奢仪式感 / 遗憾消散感

**2. 多方案推演**
- 使用 `ui-design-trio` Skill 生成三版差异化方案
- 每版方案包含：视觉隐喻、适用场景、技术实现要点

**3. 独立原型验证**
- 创建独立 HTML 文件（如 `countdown_animations.html`）
- 纯 CSS 动画实现（`@keyframes` + `cubic-bezier`），零外部依赖
- 内置控制面板，支持：版本切换、主题切换（日/夜）、重新播放

**4. 用户确认与细化**
- 浏览器预览三版方案在日/夜主题下的表现
- 用户选定方向后，可在原型基础上继续细化（如增加抽屉状态联动）

**5. 生产合入**
- 将原型中的核心动画代码抽离到 React 组件
- 保持业务逻辑不变，仅替换视觉层
- TDD 验证：先写测试锁定动画触发时机，再合入代码

**技术要点**：
- 使用 CSS Variables 支持双主题，避免 JavaScript 介入
- 动画性能：优先使用 `transform` 和 `opacity`，确保 60fps
- 可访问性：支持 `prefers-reduced-motion` 媒体查询降级
- 定位约束：动画组件必须使用 `position: absolute` 基于容器定位，禁止使用 `vw/vh`

**来源**：session:6a26f0ae0bfcee1b04fb4e40

### 出价排行视觉设计决策模式 (Bid Ranking Design Pattern)

**问题背景**：直播间出价排行视觉冲击不足，需要增强紧迫感与社会认同感。

**设计探索方法**：
- 使用独立 HTML 原型页面进行三版方案推演（琉璃微光/电竞强对比/优雅极简）
- 在浏览器中直接预览日/夜双主题效果
- 用户确认后再合入真实业务代码

**核心设计决策**：
1. **视觉风格**：采用「琉璃微光」Glassmorphism 风格，毛玻璃效果保证文字对比度同时不遮挡直播画面
2. **荣誉体系**：前三名使用金银铜徽章 + 文字渐变，第一名增加「呼吸闪烁」动画（2.5s周期，scale+shadow+opacity组合）
3. **固定席位**：始终展示前三名，空缺时显示「虚位以待」占位态（视觉变灰、透明度降低）
4. **自我关联**：底部「我的出价」采用悬浮轻量卡片，当前用户上榜时名称显示为「我自己 (当前领先)」

**关键实现细节**：
- 呼吸动画使用自定义 `@keyframes breathe`，比 Tailwind `animate-pulse` 更丰富
- 上榜用户行添加微弱高亮边框，便于自我定位
- 占位态价格显示 `-`，名称显示「虚位以待」

**来源**：session:6a24637400057ea64ca28666

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

### UX 原型验证模式 (UX Prototype Validation)

**问题背景**：在合入真实业务代码前，需要快速验证动画效果、交互节奏和视觉方案，避免在复杂业务逻辑中反复调试 UI 细节。

**核心流程**：
1. **独立原型**：在项目根目录创建独立 HTML 文件（如 `bid-flair-prototype.html`），脱离复杂业务逻辑
2. **视觉验证**：使用 `web-design-engineer` Skill 快速构建可交互原型，在浏览器中验证动画曲线、时机和视觉效果
3. **确认后合入**：用户确认效果后，再将核心动画代码抽离到真实 React 组件和 CSS Module 中

**关键原则**：
- 原型文件仅用于预览，不修改真实业务代码（`LiveRoomSlide.tsx`、`BidDock.tsx` 等）
- 真实合入阶段只做最小改动：抽离动画组件、在成功回调中触发、保持原有抽屉逻辑不变
- 原型与生产代码分离，避免污染主干

**来源**：session:6a228ebf2ec60aa1a739fdcb

---

### 出价动画现状与实现路径 (Bid Animation Status)

**现状核实结论**：
- **被超价通知**：已实现，运行态触发位置在 `LiveRoomSlide` 的 WebSocket 通知回调里
  - WebSocket 收到 `notification` 类型消息且 `type === 'bid_outbid'` 时触发全局 toast
  - 链路：`WebSocketService` → `LiveRoomSlide.onNotification` → `showGlobalToast`
- **出价成功动画**：原仅有价格重渲染 + 成功 toast，无专门视觉反馈
  - 已实现「出价成功飘窗动画」作为补充，详见 `frontend/h5/SKILL.md` 的「H5 直播间出价成功飘窗动画」

**实现路径（以出价飘窗为例）**：
1. **原型验证**：先用独立 HTML 原型验证动画效果（`web-design-engineer` Skill）
2. **代码审查**：使用 `TRAE-code-review` Skill 审查实现质量
3. **最小合入**：确认效果后，将核心动画代码抽离到 React 组件，保持原有业务逻辑不变
4. **测试覆盖**：新增测试锁定动画触发时机，确保点天灯链路不触发普通出价动画

**来源**：session:6a228ebf2ec60aa1a739fdcb

---

### 服务版本同步诊断模式 (Service Version Sync Diagnostics)

**问题背景**：前端功能已更新并部署，但点击按钮后报 404 错误，提示接口不存在。

**根因分析**：
1. **代码版本不一致**：主仓库代码已更新（如新增 `/api/test/demo/sky-lamp` 接口），但实际运行的服务 worktree 停留在旧提交
2. **worktree 隔离陷阱**：本地开发使用 `deploy-local-main` 等独立 worktree 运行服务，与主仓库代码分离
3. **路由未注册**：新接口在 Gateway/test-service 中未注册，前端请求到达 Gateway 后无法匹配路由

**诊断方法**：
```bash
# 1. 检查主仓库当前提交
cd /path/to/main-repo && git rev-parse --short HEAD
# 输出：be1e2b13

# 2. 检查运行服务的 worktree 提交
cd /path/to/deploy-local-main && git rev-parse --short HEAD
# 输出：ee61090f（落后主仓库）

# 3. 验证接口是否存在（应返回 401/400，不应返回 404）
curl -s http://localhost:8080/api/test/demo/sky-lamp
# 404 表示路由未注册，服务版本过旧
```

**修复方案**：
1. 停止旧 worktree 运行的服务
2. 在最新代码 worktree 中重新构建并启动服务
3. 验证接口路由已注册（返回非 404 状态码）

**预防机制**：
- `/dp-dev` 部署脚本自动清理旧进程并使用当前 worktree 代码
- 部署前检查 `HEAD == origin/main` 确保版本对齐

**来源**：session:6a25604000057ea64ca2d08d

---

### 服务版本同步诊断模式 (Service Version Sync Diagnostics)

**问题背景**：前端功能已更新并部署，但点击按钮后报 404 错误，提示接口不存在。

**根因分析**：
1. **代码版本不一致**：主仓库代码已更新（如新增接口），但实际运行的服务 worktree 停留在旧提交
2. **worktree 隔离陷阱**：本地开发使用独立 worktree 运行服务，与主仓库代码分离
3. **路由未注册**：新接口在 Gateway/test-service 中未注册，前端请求到达 Gateway 后无法匹配路由

**诊断方法**：
```bash
# 1. 检查主仓库当前提交
cd /path/to/main-repo && git rev-parse --short HEAD

# 2. 检查运行服务的 worktree 提交
cd /path/to/deploy-local-main && git rev-parse --short HEAD

# 3. 验证接口是否存在（应返回 401/400，不应返回 404）
curl -s http://localhost:8080/api/test/demo/sky-lamp
# 404 表示路由未注册，服务版本过旧
```

**修复方案**：
1. 停止旧 worktree 运行的服务
2. 在最新代码 worktree 中重新构建并启动服务
3. 验证接口路由已注册（返回非 404 状态码）

**预防机制**：
- `/dp-dev` 部署脚本自动清理旧进程并使用当前 worktree 代码
- 部署前检查 `HEAD == origin/main` 确保版本对齐

**来源**：session:6a26ce7a0bfcee1b04fb3ddc

---

### 复杂动画设计评审模式 (Complex Animation Design Review)

**问题背景**：竞拍成功、成交确认等关键节点需要强烈的视觉反馈，但直接在业务代码中试错成本高。

**设计流程**（以竞拍成功动画为例）：

**1. 需求澄清**
- 明确动画目标：正向反馈（情绪价值）+ 结果确认（信息传达清晰度）
- 确定情感基调：庆祝/高奢/动感等方向

**2. 多方案推演**
- 使用 `ui-design-trio` Skill 生成三版差异化方案
- 每版方案包含：视觉隐喻、适用场景、技术实现要点

**示例方案对比**：
| 方案 | 风格 | 视觉隐喻 | 适用场景 |
|------|------|----------|----------|
| V1 经典欢庆 | 大众电商 | 重力印章 + 彩屑喷射 | 标准竞拍，直白成就感 |
| V2 尊享高奢 | 高客单价 | 长缓动曲线 + 金属流光 | 艺术品/名表/珠宝 |
| V3 游戏动感 | 潮玩直播 | 3D 翻转 + 非对称布局 | 盲盒/抢购，竞技快感 |

**3. 独立原型验证**
- 创建独立 HTML 文件（如 `bid_success_animations.html`）
- 纯 CSS 动画实现（`@keyframes` + `cubic-bezier`），零外部依赖
- 内置控制面板，支持：版本切换、主题切换（日/夜）、重新播放

**4. 用户确认与细化**
- 浏览器预览三版方案在日/夜主题下的表现
- 用户选定方向后，可在原型基础上继续细化（如增加「一锤定音」前置动画）

**5. 生产合入**
- 将原型中的核心动画代码抽离到 React 组件
- 保持业务逻辑不变，仅替换视觉层
- TDD 验证：先写测试锁定动画触发时机，再合入代码

**技术要点**：
- 使用 CSS Variables 支持双主题，避免 JavaScript 介入
- 动画性能：优先使用 `transform` 和 `opacity`，确保 60fps
- 可访问性：支持 `prefers-reduced-motion` 媒体查询降级

**来源**：session:6a2464ce00057ea64ca286e5

### 演示剧本设计流程
为竞赛演示录制视频时，采用以下流程设计口播剧本：
1. **叙事策略**：制造「基线 vs 生产级」的认知落差，每个模块先说"别人怎么做"，再说"我为什么这么做"
2. **模块取舍**：完整版约 5-6 分钟覆盖全部亮点，精华版 3 分钟只保留核心模块（完整闭环、并发控制、混沌工程、收尾升华）
3. **台词结构**：统一为「钩子句 → 操作旁白/画面指引 → 考量维度金句」
4. **口吻风格**：采用亲近、自然的聊天式叙述，避免演讲腔和刻意对比句式
   - 去掉"这里不是…而是""我最想展示"等演讲腔，改用"…后，会有…""大家可以注意一下"等聊天式叙述
   - 开场/收尾加"大家好""谢谢大家"，更像当面讲解
   - 技术内容、参数和考量金句保留，不影响拿分点
5. **交付物**：分镜脚本表（画面+口播+操作提示+时间戳）+ 纯口播稿两份合一

**来源**：session:6a2875380bfcee1b04fc33e8

### 竞赛提交文档生成流程
生成比赛交卷文档时采用以下流程：
1. **交付方式**：先在仓库 `docs/` 下生成完整 markdown，review 后再导入飞书
2. **内容分工**：AI 填写技术细节（核心功能、架构、AI能力、工程难点等），用户补全个人/项目基础信息
3. **表现力增强**：对于架构图、压测报告等表现力要求高的部分，制作独立 HTML 专题页部署至公网，在文档中深链引用
   - 压测/混沌报告页：`perf.html`，复用主站设计 token 和双主题系统
   - 架构图专题页：独立页面展示五层架构与调用关系
4. **架构图生成**：使用 `mindai-chart-generation` Skill 生成专业架构图，通过飞书画板插入文档
   - 使用 `lark-cli` + `whiteboard-cli` 将 MindAI 生成的图插入飞书文档
   - 需确保 lark-cli 已登录且有 docs/board 权限
5. **飞书权限处理**：向外部模板文档直接写入可能因权限不足失败，应新建个人文档后导入内容
6. **最后提交字段维护**：竞赛提交文档中通常包含"最后提交"字段，指向最新代码提交的 hash。每次实质性代码变更后需同步更新该字段
   - 定位文档中的 commit hash（如 `40af592a`）
   - 更新为当前 `origin/main` 最新提交（如 `96bc99ba`）
   - 单独提交该文档更新，commit message 如 `docs: update last commit ref`

**来源**：session:6a2875380bfcee1b04fc33e8, session:6a2879d10bfcee1b04fc3745

### lark-cli 独立授权机制
使用 `lark-cli` + `whiteboard-cli` 向飞书文档插入画板/图表时，需要**独立的 user 授权**，与 `feishu-cli` 的授权互不相通：

- **授权方式**：Device Flow，需在浏览器打开授权链接完成
- **权限域**：需确保有 `docs`（文档写入）和 `board`（画板）权限
- **授权命令**：`lark-cli login --domain docs --recommend`
- **验证状态**：`lark-cli whoami` 检查 `user` token 是否 `valid`
- **常见错误**：`no_token` 表示需要重新授权；`token_expired` 表示 token 过期需刷新

**与 feishu-cli 的区别**：
- `feishu-cli` 用于导入 markdown 内容到飞书文档
- `lark-cli` 用于操作画板、图表等高级功能
- 两者使用不同的 token 体系，不能互相替代

**来源**：session:6a2875380bfcee1b04fc33e8

### 可打印版本交付物格式

当需要生成可打印的剧本/文档时，采用**打印友好的单文件 HTML** 格式：

**技术要点**：
- **A4 分页**：使用 `@page A4` 和 `@media print` 控制打印尺寸
- **防跨页断裂**：表格行和段落使用 `break-inside: avoid` 防止被分页截断
- **黑白友好**：避免依赖彩色背景传达信息，使用边框/图标辅助
- **打印按钮隐藏**：屏幕显示的工具栏（如「打印/导出 PDF」按钮）使用 `@media print { display: none }` 在打印时自动隐藏
- **分页控制**：大章节使用 `page-break-before: always` 确保新章节起新页

**内容结构**：
- 分镜脚本表：表格形式，含段号/时间轴/画面/操作/口播
- 纯口播稿：连续文本，去掉画面操作提示，适合直接朗读配音
- 精华版指引：标注哪些模块进精简版

**使用方式**：浏览器打开 HTML → `Cmd/Ctrl+P` → 目标选「存储为 PDF」

**来源**：session:6a2875380bfcee1b04fc33e8

### Grafana Dashboard 划分与配置原则

**问题背景**：单一大盘图表过多导致信息杂乱，需要按决策场景重新编排。

**划分原则**：
1. **按决策场景分组**（推荐）：每个看板回答一个核心运营问题
   - `直播竞拍业务总览看板` — 经营总览、核心 KPI
   - `竞拍链路实时监控看板` — 实时出价、延迟、WS 链路健康
   - `交易收入与履约看板` — GMV、订单、支付、履约闭环

2. **避免按角色或图表类型划分**：会导致同一决策需要跨多个看板查看

**配置文件位置**：
- 实际加载目录：`observability/grafana/provisioning/dashboards/`
- 新增 JSON 文件后 Grafana 会自动扫描加载
- 注意：不要放到 `monitoring/grafana/dashboards/`，该目录不会被当前运行的 Grafana 容器加载

**监控指标数据源问题诊断**：
- **`0` 值**：指标存在但当前值为 0（如 `watch_user_count = 0` 表示无观众在线）
- **`No data`**：指标序列不存在，Prometheus 中无该 metric 的时间序列
  - 根因：对应业务埋点未调用，或服务未暴露该指标
  - 解决：补齐埋点或执行真实业务链路产生事件
- **`0%`**：rate 计算窗口内无有效事件，分母/分子接近 0

**来源**：session:6a24696600057ea64ca28a2a

### 批量操作事务边界决策模式 (Batch Operation Transaction Boundary)

**问题背景**：批量操作（如热拉生成多条通知）中，每个子操作独立事务 vs 全局事务的取舍。

**决策矩阵**：

| 策略 | 适用场景 | 优点 | 缺点 |
|------|----------|------|------|
| 全成功/全失败 | 强一致性要求 | 数据一致 | 单点失败阻断全部 |
| 部分成功+继续 | 通知/日志类场景 | 最大化成功数 | 需幂等保证可重试 |
| 部分成功+中断 | 列表处理 | 明确错误边界 | 调用方需处理部分结果 |

**推荐模式（通知类场景）**：
```go
var created []*Notification
for _, item := range items {
    result, err := process(item)
    if err != nil {
        log.Printf("[WARN] process failed for item=%d: %v", item.ID, err)
        continue // 记录日志并继续，不中断
    }
    created = append(created, result)
}
// 返回成功创建的部分，不报错
return created, nil
```

**关键原则**：
- 通知类写入优先选择"部分成功+继续"策略
- 必须配合幂等机制，保证重试安全
- 日志需记录失败详情，便于排查

**来源**：session:6a21af602ec60aa1a739c0d9

### 热拉入库通知设计模式 (Hot-Pull Notification Design)

**问题背景**：需要在用户登录/回前台时，将订阅的即将开始事件转为未读通知驱动红点，但不想引入弹窗复杂度。

**设计要点**：
1. **只做热拉，不做冷推**：MVP 阶段成本最低，覆盖核心场景
2. **明确阈值**：如"30分钟内开始"才算即将开始
3. **幂等回执表**：防止重复生成通知
4. **不抢弹窗通道**：仅生成未读通知，不触发弹窗

**数据模型**：
```
user_product_reminders (订阅源)
    ↓
product_reminder_receipts (幂等回执: user_id + auction_id)
    ↓
notifications (未读通知: type=auction_starting)
    ↓
铃铛红点 (/notifications/unread-count)
```

**接口契约**：
- `POST /notifications/hot-pull` — 热拉入口
- 响应包含新生成的通知列表
- 前端调用后需刷新未读数

**来源**：session:6a21af602ec60aa1a739c0d9

### 视觉辅助服务环境适配 (Visual Aid Service Environment Adaptation)

**问题背景**：在特定环境中使用 `nohup` 启动后台视觉预览服务时，进程会被 shell 退出时回收，导致服务实际上未在监听端口。

**根因**：某些环境会把脚本里 `nohup` 拉起的后台服务回收掉，"启动成功信息"可能是假的成功，进程马上消失。

**解决方案**：
- 使用 `--foreground` 参数启动服务
- 配合异步终端长驻进程模式保活
- 通过 `curl` 或 `lsof` 验证端口真实监听状态

**验证命令**：
```bash
# 验证服务是否真实监听
curl -s http://localhost:<port>/health || echo "服务未启动"
lsof -i :<port> | grep LISTEN
```

**来源**：session:6a1fd603867f95f321bde97f

### 网关限流 TTL 修复 (Gateway Rate Limit TTL Repair)

**问题背景**：本地开发时前端登录失败，提示"请求过于频繁"，但 Redis 中的限流 key 没有设置过期时间，导致限流状态永久存在。

**根因分析**：
- 网关限流中间件使用 `redis-cli INCR` 增加计数，但未给新创建的 key 设置 TTL
- 首次请求创建 key 后，没有 `EXPIRE` 操作，key 永久存在
- 即使窗口期已过，旧 key 仍然存在，导致后续请求被错误限流

**修复方案**：
- 在 `INCR` 后检查 key 是否刚创建（值为 1），如果是则立即设置 TTL
- 使用 Redis 事务或 Lua 脚本确保原子性
- 默认窗口期：60 秒，最大请求数：100

**关键代码模式**：
```go
// 限流 key 格式
ratelimit:{client_ip}

// 修复逻辑
count, err := redis.Incr(key)
if err != nil {
    return false, err
}
if count == 1 {
    // 新创建的 key，设置过期时间
    redis.Expire(key, windowSeconds)
}
```

**测试验证**：
- 使用 `miniredis` 进行单元测试，验证 TTL 是否正确设置
- 模拟窗口期过期后，确认 key 被自动清理

**来源**：session:6a1c56f7959156a8dfc84fae

### HTTP Client 响应体 Drain 模式 (HTTP Response Body Drain Pattern)

**问题背景**：内部服务调用时，如果 HTTP 响应体未被完全读取（或 drain），`http.Client` 的连接无法回收到连接池，高 QPS 下会导致端口耗尽。

**根因分析**：
- Go 的 `http.Transport` 依赖响应体被完全读取后才能复用连接
- 当只需要检查状态码而不读取 Body 时，连接会被标记为未关闭
- 大量未关闭连接累积后，系统会耗尽可用端口或文件描述符

**解决方案**：
```go
// 即使不读取内容，也必须 drain 响应体
resp, err := httpClient.Do(req)
if err != nil {
    return err
}
defer func() {
    // 必须 drain body 才能复用连接
    io.Copy(io.Discard, resp.Body)
    resp.Body.Close()
}()

// 或者使用工具函数
func drainAndClose(resp *http.Response) {
    if resp != nil && resp.Body != nil {
        io.Copy(io.Discard, resp.Body)
        resp.Body.Close()
    }
}
```

**代码审查要点**：
- 所有 `httpClient.Do()` 调用后必须检查是否有 `defer resp.Body.Close()`
- 如果响应体未被显式读取，必须补充 `io.Copy(io.Discard, resp.Body)`
- 错误路径中也要确保 Body 被关闭（使用 `defer` 或提前关闭）

**测试验证**：
- 高并发压测下观察 `ESTABLISHED` 连接数是否稳定
- 检查是否有大量 `TIME_WAIT` 状态的连接堆积

**来源**：session:6a2419153eefb8c530aa7658

### 测试 SDK 计时粒度问题 (Test SDK Timing Granularity Issue)

**问题背景**：`TestSDK_CreateProduct` 测试在本地运行失败，因为 `DurationMs` 断言要求 `> 0`，但本地 mock HTTP 调用太快时 `time.Since(start).Milliseconds()` 返回 0。

**根因分析**：
- `time.Since(start).Milliseconds()` 是向下取整的整数毫秒
- `httptest` 本地调用经常小于 1ms，导致成功请求记录成 `0`
- 这不是业务失败，是 SDK 计时粒度和测试断言不一致

**修复方案**：
在 SDK 层统一保护：真实发生过耗时但不足 1ms 时记录为至少 1ms，避免每个测试去睡眠或放宽断言。

```go
elapsedMs := time.Since(start).Milliseconds()
if elapsedMs == 0 && time.Since(start) > 0 {
    elapsedMs = 1 // 保护：真实发生过耗时但不足 1ms
}
```

**设计原则**：
- SDK 层统一处理计时粒度问题，不分散到各个测试
- 只有时钟异常或未发生耗时时才可能为 0
- 避免测试中使用 `time.Sleep` 来「凑」耗时

**来源**：session:6a24716000057ea64ca294db, session:6a25bce00bfcee1b04fb15bd

### Git 工作流：干净 worktree + cherry-pick 合并 (Git Clean Worktree Merge)

**问题背景**：本地分支领先远端且工作区存在无关脏改动时，直接 push 会被拒绝，直接 rebase 会混入无关改动。

**解决方案**：
1. 在干净目录新建临时 worktree 基于最新 `origin/main`
2. 将本次提交 cherry-pick 到该 worktree
3. 在合并后的结果上验证（测试、构建）
4. 验证通过后推送到远程 `main`

**关键步骤**：
```bash
# 1. 创建干净 worktree
git worktree add /tmp/clean-worktree origin/main

# 2. cherry-pick 本次提交
cd /tmp/clean-worktree
git cherry-pick <commit-hash>

# 3. 验证
npm test && npm run build

# 4. 推送
git push origin HEAD:main
```

**适用场景**：
- 本地领先远端多个提交
- 工作区存在未提交的无关改动
- 需要确保只推送特定提交到远程

**变体：本地 main 分叉合并**
当本地 `main` 和远端 `main` 各有一个独立提交（分叉状态）时：
1. 临时 stash 本地冲突文件（如 `frontend/h5/e2e/home.spec.ts`）
2. `git merge origin/main` 合并远端变更
3. 恢复 stash 并解决冲突
4. 重新跑定向测试验证

**后端修复 TDD 模式（在隔离 worktree 中执行）**：
1. **检查工作区状态**：确认 `main` 存在未跟踪调试文件和测试产物
2. **创建隔离 worktree**：`git worktree add ../fix/<branch-name> -b fix/<branch-name>`
3. **RED 阶段**：新增失败测试，证明 DAO 查询必须用应用传入的 `now` 而非数据库 `NOW()`
4. **GREEN 阶段**：修改 DAO 层实现，将 `NOW()` 改为参数化查询
5. **验证**：后端全量测试通过后再合并到 `main`
6. **合并推送**：在干净 worktree 中 cherry-pick 提交，验证后推送

**来源**：session:6a1fd603867f95f321bde97f, session:6a1fffc7867f95f321be0ce6, session:6a203cf7867f95f321be373d, session:6a2138fc2ec60aa1a7394fdf, session:6a21d2c92ec60aa1a739da7c, session:6a2139882ec60aa1a7395093, session:6a274d560bfcee1b04fba6a8

---

### Git 工作流：业务改动与知识库改动分离提交 (Business vs Knowledge Changes Separation)

**问题背景**：工作区同时存在业务代码改动、知识库文档改动和构建缓存删除，需要按类别分离提交，避免把非业务改动混入业务提交。

**提交边界划分**：
1. **业务改动**：`backend/*`、`frontend/*` 的代码变更，以及业务相关的 spec/plan 文档
2. **知识库改动**：`.trae/knowledges/*` 的 SKILL.md 等知识沉淀文件
3. **缓存/生成物**：`node_modules/.vite/*`、`frontend/*/dist/*` 等构建产物

**推荐提交顺序**：
```bash
# 1. 先提交业务改动（不包含知识库和缓存）
git add backend/ frontend/ docs/superpowers/specs/2026-06-10-xxx-design.md
git commit -m "feat: xxx business feature"

# 2. 再提交知识库改动
git add .trae/knowledges/
git commit -m "docs: update knowledges"

# 3. 恢复/清理缓存删除（不提交）
git checkout -- frontend/*/node_modules/.vite/deps/*
# 或添加到 .gitignore 后清理
```

**关键原则**：
- `node_modules/.vite/deps/*` 缓存删除不应提交，建议恢复或添加到 `.gitignore`
- 知识库改动与业务改动分离提交，便于回滚和审查
- 提交前使用 `git diff --staged` 确认暂存区内容符合预期

**来源**：session:6a2879d10bfcee1b04fc3745

---

### Git 工作流：部分暂存提交（Partial Staging）(Git Partial Staging Workflow)

**问题背景**：工作区存在多个不相关的改动（如不同功能的修改、调试文件、文档草稿），但需要只提交其中一部分到远程，避免混入无关改动。

**核心流程**：
1. **审查改动范围**：`git status` 和 `git diff` 确认所有待提交内容
2. **选择性暂存**：
   - 按文件暂存：`git add <file>`
   - 按 hunk 暂存：`git add -p <file>` 交互式选择代码块
3. **验证待提交内容**：`git diff --staged` 确认暂存区只包含目标改动
4. **临时收起未暂存改动**（可选）：`git stash push -m "pre-commit" --include-untracked`
5. **测试验证**：运行相关测试确保干净提交通过
6. **提交推送**：`git commit` 和 `git push`
7. **恢复未暂存改动**：`git stash pop`

**关键命令示例**：
```bash
# 查看所有改动
git status
git diff

# 只暂存特定文件的特定 hunk
git add -p frontend/h5/src/pages/Live/LiveRoomSlide.tsx

# 确认暂存区内容
git diff --staged

# 临时收起未暂存改动，验证干净提交
git stash push -m "pre-commit" --include-untracked
npm test -- --runTestsByPath src/pages/Live/__tests__/LiveRoomSlide.test.ts

# 提交并推送
git commit -m "fix: unsold animation triggers on local countdown zero"
git push origin main

# 恢复未暂存改动
git stash pop
```

**适用场景**：
- 工作区同时存在多个功能改动，需要分批提交
- 调试代码、临时文件与正式改动混在一起
- 需要精确控制提交范围，避免无关改动进入版本历史

**来源**：session:6a274a280bfcee1b04fba36e

---

### Git 工作流：功能分支合入与脏工作区处理 (Git Feature Branch Merge with Dirty Worktree)

**问题背景**：功能分支开发完成后，本地 `main` 存在未提交的脏改动（如 spec 文档、调试文件），直接合入会冲突或覆盖。

**解决方案**：
1. **功能分支提交**：在 worktree 内完成功能开发并提交为独立 commit
2. **主 worktree 保护**：不清理用户脏文件，使用 stash 保护未提交改动
3. **合并执行**：`git merge` 功能分支，产生 merge commit
4. **选择性恢复**：从 stash 恢复非冲突的原有改动，同名旧文件不恢复

**关键步骤**：
```bash
# 1. 功能分支提交
git add <业务文件>
git commit -m "feat: xxx"

# 2. 回到主 worktree，stash 保护脏状态
git stash push -m "pre-merge-state" --include-untracked

# 3. 合并功能分支
git merge feat/xxx --no-ff -m "Merge feat/xxx"

# 4. 选择性恢复 stash（排除冲突文件）
git stash show -p | grep -v "冲突文件" | git apply
```

**注意事项**：
- 同名文件冲突时优先保留功能分支版本（新功能）
- 旧 spec/plan 文档留在 stash 备份，不覆盖新合入版本
- 合并后验证 `main` 已有 sentinel 和新增 sentinel 都不红

**来源**：session:6a25c4110bfcee1b04fb1b82

---

### 多任务代码工作交叉控制 (Multi-Task Code Interleaving Control)

**问题背景**：多个任务同时修改同一批文件时，容易出现代码回退问题——修复只存在于某个 worktree 或 dev server 里，没有形成 `main` commit；或旧分支从旧 `main` 派生，后合入时覆盖新修复。

**核心原则**：
1. **`main` 永远是唯一事实源**：所有演示环境、dev server、后端服务尽量从同一个 `main` worktree 启动。不要让前端跑 `deploy-local-main`，后端跑主仓库，另一个任务又在第三个 worktree 改代码。

2. **每个任务必须短分支、短生命周期**：任务分支只做一件事，例如 `fix/h5-ended-summary`、`fix/demo-upcoming-auction`、`fix/live-drawer-layout`。不要在一个历史分支上连续叠加多个不相关修复。超过半天没合入，就先 rebase/merge 最新 `main`。

3. **开始任务前强制同步**：
```bash
git checkout main
git pull --ff-only origin main
git checkout -b fix/xxx
```

4. **所有修复必须有「防回退测试」**：尤其是 UI 样式修复。只改 CSS 没测试，后续很容易被覆盖且无冲突。例如：
   - 抽屉比例 `50dvh / 50%`
   - 商品名 `text-overflow: ellipsis`
   - 结束卡片 `SOLD` 水印、扫光、成交价字号

5. **合入前只允许 clean worktree**：提交前检查 `git status --short` 和 `git diff --name-only`，只提交本任务文件。

6. **合入动作要机械化**：
```bash
git fetch origin main
git rebase origin/main
npm test -- LiveLayoutCss.test.ts --runInBand
git status --short
git push origin HEAD
```

7. **并行任务按「文件所有权」拆**：如果两个任务都要改 `Live.module.css`、`LiveRoomSlide.tsx`，它们不是独立任务，不能并行。

**来源**：session:6a25619300057ea64ca2d2d8

---

### Git 冲突处理：语义合并原则 (Git Conflict Semantic Merge)

**问题背景**：当 A 分支合入 main 产生冲突时，如果机械地「以 main 为准」，可能导致 A 分支的真实优化被吞掉。

**核心原则**：**`main` 是事实源，不是永远胜者。**

合冲突时应该回答三个问题：
1. main 这边的改动解决了什么问题？
2. A 分支的改动解决了什么问题？
3. 两边是否可以同时保留？如果不能，哪个需求现在仍然有效？

**冲突处理规则**：
- `main` 代表当前已验证的基线，不能被旧代码无意识覆盖
- A 分支的优化如果仍然有效，必须被重新应用到最新 main 上
- 处理冲突时**禁止整文件选 `main` 或整文件选 `A`**，除非明确记录理由
- 冲突解决后必须跑 A 分支优化对应的 regression sentinel，证明优化没有丢
- 如果 A 没有 sentinel，那合入前先补一个，否则无法判断「优化是否还在」

**示例**：
```text
main:
- 修了抽屉高度 50dvh
- 修了 h1 溢出

A:
- 优化了结束卡片黑金扫光

冲突文件都是 Live.module.css
```

**错误做法**：`git checkout --ours Live.module.css`（会吞掉 A 的结束卡片优化）

**正确做法**：
1. 保留 main 的抽屉和 h1 修复
2. 再把 A 的结束卡片优化迁移到当前 CSS 结构
3. 然后跑：抽屉 sentinel、h1 overflow sentinel、ended summary sentinel

**一句话总结**：**以 main 为基线，不等于以 main 覆盖一切；它的意思是把 A 的有效优化重新移植到最新 main 上。**

**来源**：session:6a25619300057ea64ca2d2d8

### 商品分类契约对齐修复案例 (Product Category Contract Alignment)

**问题背景**：管理端存在大量未分类商品（`category_id IS NULL`），但创建商品时类别是必选项；H5首页分类tab只显示"全部"和"收藏"，没有展示其他类别。

**根因分析**：
- 后端商品接口使用 `category_id` 字段，但早期部分数据未正确写入
- Admin 前端使用旧的 `category?: string` 字段，与后端契约不一致
- H5 首页分类tab依赖 `/categories` 接口，但商品列表未正确关联分类

**修复方案（最短改动路径）**：

**1. 后端契约扩展** (`backend/product/`)
- `POST /api/v1/products`：请求体新增 `category_id?: number | null`，非法分类返回 `400`
- `PUT /api/v1/products/:id`：支持编辑商品分类
- `GET /api/v1/products/:id` 和 `GET /api/v1/products`：响应补 `category_id`、`category_name`

**2. Admin TS 类型对齐** (`frontend/admin/src/shared/api/`)
```ts
interface Product {
  id: number;
  name: string;
  category_id?: number | null;  // 从旧的 category?: string 迁移
  category_name?: string;       // 新增展示字段
  // ...
}

interface Category {
  id: number;
  name: string;
  code: string;
}
```

**3. Admin 页面适配**
- `GoodsEdit.tsx`：表单字段从 `category: string` 改为 `category_id?: number | null`，下拉选项从硬编码改为接口数据
- `GoodsList.tsx`：类别列改为 `item.category_name || '未分类'`

**4. 数据治理**
- 编写 `scripts/backfill_product_category.sql` 回填历史数据
- 对无法自动映射的商品保留 `NULL`，但标记为"未分类"
- 删除测试残留数据（无业务语义的E2E生成数据）

**关键决策**：
- H5 首页只显示"全部/收藏"不是因为商品里有"未分类"，而是因为 `/categories` 主数据未正确关联；修复重点是把 Admin 和商品接口接回同一套 `category_id`
- 数据回填时区分"可推断业务分类"和"测试残留"，后者直接删除而非硬猜分类

**来源**：session:6a21350f2ec60aa1a7394bad

### 直播间状态判定修复 (Live Auction Time-State Fix)

**问题背景**：直播间显示"直播中"，但进入后倒计时已结束（显示 `00:00`），且"立即出价"按钮仍可点击并能成功出价。

**根因分析**：
- 后端判定"是否可出价/是否当前竞拍"主要看 `status` 字段，没有同时校验 `end_time`
- 数据库里存在 `status=1`（进行中）但 `end_time` 已过的竞拍记录
- 首页列表按 `status` 筛选显示"直播中"，直播间倒计时按 `end_time` 计算显示"已结束"
- 后端出价接口未校验 `end_time`，导致过期竞拍仍可出价

**修复方案**：
1. **列表查询**：首页和直播间 feed 的"进行中"判定从单纯 `status=1` 升级为 `status=1 AND end_time > NOW()`
2. **出价权限**：`Bid` 接口增加 `end_time` 校验，已结束竞拍返回 `AUCTION_ENDED` 错误码
3. **状态同步**：直播间详情接口返回的 `status` 字段应基于真实时间计算，而非直接返回数据库值

**关键代码模式**：
```go
// 列表查询：进行中竞拍
WHERE status = 1 AND end_time > NOW()

// 出价校验
if auction.Status != 1 || auction.EndTime.Before(time.Now()) {
    return ErrAuctionEnded
}
```

**时间源统一原则（第一性原理修复）**：
- 状态机的时间来源应由应用层统一，而不是依赖数据库所在机器时区
- 将 DAO 层查询从 `NOW()` 改为接收 `now time.Time` 参数，SQL 使用 `start_time <= ?` / `end_time <= ?`
- 同一轮状态推进共享同一个 `now`，避免 start/end 判断跨秒漂移
- 测试需覆盖"DB `NOW()` 时区不同也不影响，因为比较值来自应用层"的行为

**教训**：
- 涉及"时间状态"的业务逻辑不能只依赖数据库 `status` 字段，必须结合 `end_time` 实时计算
- 列表筛选条件和详情页展示逻辑必须保持一致，否则会出现"首页显示直播中但详情已结束"的认知断裂
- 出价接口必须有兜底校验，防止前端状态滞后或绕过校验导致错误出价
- 改 MySQL 时区只能修"这个环境的 `NOW()`"，不能保证代码在 CI、Docker、生产、不同 DB session 下都正确

**来源**：session:6a21350f2ec60aa1a7394bad, session:6a21d2c92ec60aa1a739da7c

### Demo Console 开播状态一致性 (Demo Console Live Stream State Consistency)

**问题背景**：通过 Demo Console「正在竞拍」按钮创建的竞拍，用户关注后重新登录看不到开播提醒弹窗，且进入直播间提示"资源不存在"。

**根因分析**：
- Demo Console 的 `PostMerchantAuction`（`ongoing` 模式）仅创建商品、直播间、规则和竞拍记录
- **缺失关键环节**：未调用 `StartLive` 启动直播间，导致 `live_stream:{id}:stats.started_at` 未写入
- 开播提醒依赖 Redis 中的 `started_at` 字段判断是否有待提醒的直播 session
- 直播间状态未正确设置为「直播中」，导致进入时的资源校验失败

**修复方案**：
1. **后端修复**：`PostMerchantAuction` 在创建 ongoing 竞拍后，显式调用 `StartLive` 启动直播间
2. **测试加固**：增加断言确保"ongoing demo 必须启动直播间"
3. **数据回填**：对历史未启动的直播间补全启动状态

**关键洞察**：
- Demo Console 的「正在竞拍」模式必须完整模拟真实业务链路，不能只创建数据记录而不触发状态变更
- 竞拍状态与直播间状态是两个独立但关联的领域，必须保持一致
- 开播提醒、直播间进入等下游功能依赖直播间状态的完整性

**相关文档**：详见 `frontend/test-dashboard/SKILL.md` 中的 "Demo Console '正在竞拍' 模式开播状态问题"

**来源**：session:6a257c8d0bfcee1b04fafe04

### 数据回填与测试残留清理策略 (Data Backfill & Cleanup)

**问题背景**：历史数据中存在 `category_id IS NULL` 的商品，需要治理但不宜盲目回填。

**回填策略**：
1. **自动映射**：基于商品名称关键词（如包含"玉"、"镯"映射到珠宝首饰）自动回填
2. **人工确认**：对关键词不明确的商品，先列出名称和描述供人工确认后再回填
3. **保留 NULL**：对无法推断业务分类且不影响主流程的数据，保留 `NULL` 并显示"未分类"

**测试残留识别标准**：
- 名称包含 "E2E 测试拍品"、"Fixed Price Demo" 等测试标记
- 描述包含 "orchestrator auto-generated"、随机 UUID
- 无任何业务语义，无法映射到真实分类

**清理执行原则**：
- 先检查外键依赖（`auction_rules`、`auctions`、`orders` 等表），确认无引用后再删除
- 使用 `SELECT` 先确认影响范围，再执行 `DELETE`
- 删除后复核 `NULL` 计数，确保符合预期

**来源**：session:6a21350f2ec60aa1a7394bad

### 部署版本验证与缓存问题 (Deployment Version Verification & Caching)

**问题背景**：部署后用户仍看到旧版本内容，或无法确定线上运行的是否为最新 main 分支产物。

**版本验证方法**：
1. **Git 提交验证**：使用 `git merge-base --is-ancestor <fix-commit> HEAD` 确认修复提交是否包含在当前部署版本中
2. **静态资源特征扫描**：检查线上 JS/CSS 文件中是否包含特定修复代码特征（如 `textEncoding`、`repairUtf8Mojibake` 等函数名）
3. **HTTP 响应头验证**：检查静态资源 `Last-Modified` 时间（注意：仅说明"曾部署过"，不能证明是最新版本）

**浏览器缓存陷阱**：
- H5 JS 资源可能配置长期缓存（如 `Cache-Control: public, immutable`）
- 即使文件名带 hash，若浏览器持有旧 HTML/入口文件，可能继续加载旧资源
- 微信/内嵌浏览器缓存尤其顽固

**强制刷新策略**：
- 无痕窗口测试
- URL 后加时间戳参数（如 `?t=now`）
- 清理站点缓存后重开
- 换用不同浏览器（如从微信切换到 Safari/Chrome）

**来源**：session:6a203cf7867f95f321be373d

### 部署文档精简策略 (Deployment Documentation Strategy)

**问题背景**：原部署 README 过长，日常复用时需要快速找到"当前 main 分支的最短部署路径"。

**解决方案**：
- 保留原 `deploy/demo/README.md` 作为完整参考
- 新增 `deploy/demo/MAIN_DEPLOY_QUICKSTART.md` 精简版，只包含：
  - 首次部署步骤
  - 日常增量发布（如仅 H5 改动时的最短路径）
  - 验证清单
  - 回滚方法
- 在根目录 `README.md` 开头添加跳转链接，指向精简版

**关键信息收敛**：
- 当前 demo 形态：`ECS + Nginx + docker-compose.demo.yml`
- 入口约定：H5 `/`、Admin `/admin/`、API `/api/v1`、WS `/api/v1/ws`
- 最短发布路径：仅 H5 改动时，本地 `npm run build` 后 `rsync` 到 `/var/www/auction-h5`
- 探针经验：不把 `/api/v1/health` 当唯一验证，优先验证真实业务接口 `/api/v1/products`

**来源**：session:6a1fffdd867f95f321be0cfd

### SDD 执行模式：直播间弹幕 MVP (SDD Execution: Live Chat MVP)

**任务规模**：13 个 TDD 任务（T1-T13），涵盖后端双 Room 抽象、弹幕 Filter/Throttle、ChatHandler、前端 ChatPanel、飘屏组件等全链路。

**执行模式**：Subagent-Driven，主 Agent 负责任务派发、状态同步和关键决策，Subagent 负责具体实现。

**关键决策点**：
1. **T7 范围调整**：原计划修改现有 WS Hub 内嵌 read 循环，调整为替换为 Client.ReadPump/WritePump 模式，需确保不影响现有竞拍消息路径
2. **T13 冒烟测试分段**：分为阶段 A（后端 WS 直连冒烟）和阶段 B（前端 UI 冒烟），阶段 B 需先修复 Vite WS 代理配置

**代码审查发现**：
- `ChatMessage` 类型约束：`user_id` 应为 `number` 而非 `string`，避免类型漂移
- Redis TTL 行为：`ChatThrottle` 的频控 key 需显式设置过期时间，避免 key 永久堆积
- 错误码一致性：`chat_error` 的 code 定义需与 `ErrorCode` 枚举保持一致
- `StateManager` 注入：`Hub` 启动时需正确注入 `StateManager`，否则 `sync_request` 会失败
- 空房间回收：`LiveStreamRoom` 需定期清理无客户端的空房间，防止内存泄漏

**修复验证**：
- 后端 Go 全量测试：`auction`、`gateway`、`product`、`test`、`seed` 模块全部通过
- 前端 Jest 测试：WebSocket 相关测试 10/10 通过
- 类型检查：`frontend/h5` 和 `frontend/admin` TypeScript 编译通过

**来源**：session:6a1c56f7959156a8dfc84fae

### SDD 执行模式：直播间 Feed 改造 (SDD Execution: Live Room Feed)

**任务规模**：10 个 TDD 任务（T1-T10），涵盖后端契约、前端 Feed 骨架、手势交互、抽屉重构、返回键策略、WS 归属校验等全链路。

**执行模式**：Subagent-Driven，主 Agent 负责任务派发、状态同步和关键决策，Subagent 负责具体实现。

**关键决策点**：
1. **T5 范围调整**：原计划修改 `Live/index.tsx` 作为入口，但发现会破坏现有 `LiveRoom.test.tsx`，调整为仅扩展 api + 新建 `LiveFeedPage`，入口切换推迟到 T6
2. **T8 范围调整**：出价表单移入 sheet 会导致现有测试失败，授权 subagent 同步修改测试（开 sheet → 点出价，断言等价）
3. **T9 返回键策略**：从「手动 popstate 监听」改为「URL searchParams 单源驱动」，打开抽屉时 `push` sheet 参数，返回键自然消费

**代码审查发现**：
- `loadingMoreRef` 竞态：effect cleanup 未重置 ref，导致快速切房后分页加载卡死
- 修复：cleanup 中同步重置 `loadingMoreRef.current = false`，不依赖异步 `.finally()`

**经验沉淀**：
- 重构前务必检查现有测试依赖，避免「改入口破坏测试」的返工
- ref 清理逻辑必须在 effect cleanup 中同步完成，不能依赖异步回调
- Subagent-Driven 模式下，主 Agent 需明确定义任务边界和 allowed files，避免 subagent 越界修改

**来源**：session:6a1ea710959156a8dfc8a36e

### sdd-run Skill 状态选择安全契约 (sdd-run State Selection Safety Contract)

**问题背景**：`sdd-run` Skill 的 `infer_context()` 逻辑曾将「唯一 active 且 Pending 非 0 的 state」当作最高优先级，导致用户仅输入 `/sdd-run`（无其他限定）时自动恢复旧任务上下文，造成误执行。

**核心设计原则（失败关闭 Fail-Closed）**：
- `active state` 是恢复上下文，不是新任务上下文
- 任何可能复用旧上下文的行为，必须由用户显式确认或返回 `needs_selection`
- 空 `/sdd-run` 不得自动续跑 active state

**状态选择优先级（从高到低）**：
1. **显式 `state:` 路径** — 直接使用指定 state 文件，不参与任何推断
2. **显式 `continue/resume/继续` + 唯一 active state** — 允许恢复
3. **显式 `plan/tasks` 路径** — 用于新任务，不受 active state 影响
4. **空输入** — 若存在 active state，返回 `needs_selection`；若不存在且只有唯一 plan/tasks 对，创建新 state

**关键安全机制**：

| 机制 | 实现 | 目的 |
|------|------|------|
| resume 意图识别 | `has_resume_intent(input)` 检测 `继续/resume/continue` 等关键词，但增加否定词保护（`不要恢复`、`不续跑`、`do not resume`） | 防止误触发恢复 |
| active 判定严格化 | `state_is_active()` 必须同时满足 `Status=active` 且 `Pending` 明确大于 0；缺失 `Pending` 不再当作 active | 避免旧格式 state 被误判 |
| 路径碰撞检测 | 新建 state 时若默认路径 `docs/superpowers/sdd/runs/YYYY-MM-DD-<topic>-state.md` 已存在，且用户未显式指定 `state:`、`--force` 或 `continue/resume`，返回 `needs_selection` | 防止静默复用旧 state |
| 具体化 reason | `needs_selection.reason` 区分 `active_state_requires_resume_intent`、`existing_state_path_collision`、`ambiguous_plan_tasks_candidates` | 便于用户/agent 决策 |

**边界用例清单**：
- 空 `/sdd-run` + active state → `needs_selection: true`
- `/sdd-run 继续` + 唯一 active state → `inference_source: active_state`, `resume: true`
- `/sdd-run 不要恢复旧 state` + active state → 不得恢复，返回 `needs_selection`
- 显式 `plan/tasks` + 默认 state 文件已存在 → 路径冲突，不静默复用
- 缺少 `Pending` 的旧 active state → 不作为 active candidate
- `Pending=0` 的 active state → 不作为 active candidate
- 多个 active state + `继续` → `needs_selection`，列出多个 `state_candidates`

**相关文件**：
- `.agents/skills/sdd-run/SKILL.md` — Skill 定义与触发规则
- `.agents/skills/sdd-run/evals/evals.json` — 评估用例
- `docs/superpowers/sdd/scripts/sdd_run.py` — 核心实现
- `docs/superpowers/sdd/scripts/test_sdd_run.py` — 单元测试（21 个用例覆盖安全契约）

**来源**：session:6a217eda2ec60aa1a739926f

---

### sdd-run Skill 增强：任务隔离与防回退机制 (sdd-run Enhancement: Task Isolation & Regression Prevention)

**问题背景**：SDD 执行过程中存在多任务并行时文件冲突、修复回退、运行环境不一致等问题。

**核心增强点**：

**1. Write Set / Read Set 声明**
- 每个任务必须声明可修改文件（write set）和只读文件（read set）
- 重叠 write set 默认串行，避免多个任务同时改同一文件
- 示例：
```yaml
write_set:
  - frontend/h5/src/pages/Live/Live.module.css
  - frontend/h5/src/pages/Live/__tests__/LiveLayoutCss.test.ts
read_set:
  - frontend/h5/src/pages/Live/BidDock.tsx
```

**2. Regression Sentinel（防回退测试）**
- Bugfix、UI、接口契约、演示链路修复必须补「防回退测试」
- 这类测试不是为了验证业务正确性，而是为了防止别人把样式/契约改回去
- 示例 sentinel：
```typescript
// 锁定抽屉比例为 50dvh/50%
expect(sheetOpen.height).toBe('50dvh');
expect(videoAreaCompact.height).toBe('20%');

// 锁定长标题溢出处理
expect(productCardH1.textOverflow).toBe('ellipsis');
```

**3. 运行环境来源记录**
- 本地服务/前端 dev server 验证时必须记录：
  - `branch`：当前 Git 分支
  - `worktree`：绝对路径
  - `commit`：HEAD commit hash
  - `dirty status`：是否有未提交改动
  - `command`：启动命令
- 确保浏览器看到的页面确实来自刚提交的代码

**4. 旧分支安全规则**
- 旧分支不能整分支合入 main
- 优先 rebase 后 cherry-pick 必要 commit
- 冲突解决时禁止整文件选 `main` 或整文件选 `branch`

**5. 冲突处理语义合并原则**
- 以 `main` 为基线做语义合并，但不得直接丢弃任务分支的有效行为
- 任务分支的有效优化必须通过 sentinel 测试或明确决策记录证明被保留、替代或废弃
- 冲突解决后的验收标准：`main` 已有 sentinel 不能红，任务分支新增 sentinel 也不能红

**来源**：session:6a25619300057ea64ca2d2d8

---

### SDD 执行中的 Scope Contamination 风险 (SDD Scope Contamination Risk)

**问题背景**：SDD 执行任务时，某任务的代码修复被混入其他任务的提交中，导致：
- 功能分支提交不是原子性的
- 回滚该提交会影响任务外改动

**处理原则**：
1. 不回滚已混入的提交（避免影响任务外改动）
2. 在状态文件明确记录 scope contamination 风险
3. 用复核和 sentinel 测试保证功能不回退

**预防措施**：
- 严格保持任务边界，避免不同任务的改动混入同一提交
- 每个任务完成后及时合并到 main，减少并行任务间的交叉
- 使用独立的 worktree 隔离不同任务的开发

**来源**：session:6a26ce7a0bfcee1b04fb3ddc

### Gateway 角色透传修复 (Gateway Role Forwarding Fix)

**问题背景**：商家账号调用 AI 文案生成接口时被拒绝，提示无权限，但 Gateway 的鉴权中间件已正确通过。

**根因分析**：
- Gateway 的 `RequireMerchant()` 仅负责入口鉴权，通过后会剥离 JWT，生成 `X-User-ID` 和 `X-User-Role`
- 但在转发到下游 `product-service` 时，`proxy.go` 未将 `X-User-Role` 加入转发请求头
- `product-service` 的 `CopywritingHandler` 无法识别调用者角色，默认只允许管理员访问

**修复方案**：
1. **透传 Header**：在 `backend/gateway/handler/proxy.go` 的 `ProxyWithTarget` 中，将 `X-User-Role` 加入转发请求头
2. **下游识别**：`product-service` 从 `X-User-Role` 读取角色，支持 `merchant`、`admin`、`streamer` 访问 AI 文案接口
3. **测试验证**：新增 `copywriting_route_test.go` 验证角色透传和权限判断逻辑

**关键代码模式**：
```go
// proxy.go 转发时透传角色
req.Header.Set("X-User-ID", userID)
req.Header.Set("X-User-Role", userRole)  // 新增
```

**关键提交**：
- `f0d2b9af fix(gateway): forward merchant role to downstream`

**来源**：session:6a2134612ec60aa1a7394b27

### 飞书文档权限处理
- **权限错误**：向比赛方提供的模板文档直接写入会因权限不足（code=1770032 forbidden）失败
- **解决方案**：通过「创建副本」或新建个人文档方式获取写入权限，再将内容导入
- **工具链**：使用 `feishu-cli` 导入 markdown 内容，使用 `lark-cli` + `whiteboard-cli` 插入画板/图表

来源：session:6a25c5830bfcee1b04fb1c9e, session:6a25c5830bfcee1b04fb1c9e, session:6a2875380bfcee1b04fc33e8

### LLM 供应商抽象层 (Shared LLM Provider)

**设计目标**：为后端服务提供统一的 LLM 调用抽象，支持不同供应商（Doubao/Ark、OpenAI 等）的切换和扩展。

**目录结构**：
- `backend/shared/llm/provider.go` — Provider 接口定义
- `backend/shared/llm/doubao_provider.go` — Doubao/Ark 实现
- `backend/shared/llm/factory.go` — Provider 工厂

**关键约束**：
- **环境变量注入**：API Key 通过环境变量注入，禁止硬编码或从配置文件读取后提交到 Git
- **超时控制**：LLM 调用需设置合理超时（如 30s），避免阻塞业务请求
- **错误处理**：区分网络错误、配额超限、内容审核等不同错误类型，返回有意义的错误信息

**使用模式**：
```go
provider := llm.NewProvider(cfg)
response, err := provider.Generate(ctx, prompt)
```

**来源**：session:6a2879d10bfcee1b04fc3745

### 一口价秒杀功能（Fixed Price Sale）后端架构

**功能概述**：直播间内的一口价秒杀功能，用户点击立即购买后扣减余额直接下单，支持库存预扣、幂等重试、实时库存广播。

**服务归属**：
- `auction-service` — 核心购买链路、库存管理、WebSocket 广播
- `gateway-service` — 路由转发、鉴权、商品信息聚合
- `product-service` — 商品基础信息提供

**核心 API**：
- `POST /api/v1/fixed-price/items` — 主播上架
- `POST /api/v1/fixed-price/items/{id}/offline` — 主播下架
- `GET /api/v1/live-streams/{id}/fixed-price/items` — 直播间商品列表
- `POST /api/v1/fixed-price/items/{id}/purchase` — 抢购（核心）
- `GET /api/v1/fixed-price/items/{id}/my-purchase` — 查询是否已购买

**数据模型**：
- `fixed_price_items` — 商品表（id, live_stream_id, product_id, price, total_stock, remaining_stock, max_per_user, status）
- `fixed_price_purchases` — 购买记录表（id, item_id, user_id, order_id, price, created_at），含 UNIQUE(item_id, user_id) 约束

**错误码矩阵**：
| code | HTTP | 触发条件 | 前端交互 |
|------|------|----------|----------|
| `FP_NOT_ON_SALE` | 409 | 商品已下架或未初始化 | Toast + 刷新 |
| `FP_SOLD_OUT` | 409 | 库存为 0 | Toast + 按钮置灰 |
| `FP_ALREADY_BOUGHT` | 409 | 用户已购买（限购） | Toast + 跳转订单 |
| `FP_INSUFFICIENT_BALANCE` | 402 | 余额不足 | 独立弹窗 + 跳转充值 |
| `FP_RATE_LIMITED` | 429 | 网关限流 | 静默重试 1 次 |
| `FP_IDEMPOTENT_REPLAY` | 200 | 幂等命中 | 直接跳成功页 |

**Saga 风格两阶段补偿（库存与扣款一致性）**：
```
Step 1: Redis Lua 原子预扣 → 失败直接返回
Step 2: DB 事务
    BEGIN
      INSERT fixed_price_purchases (唯一键兜底)
      UPDATE user_balances SET available = available - price WHERE user_id=? AND available >= price
      INSERT orders (source='fixed_price')
      INSERT outbox (event=fp_sold)
    COMMIT
Step 3: DB 失败 → Lua 反向补偿: INCR 库存 + SREM 用户集合
Step 4: DB 成功 → 异步广播 LiveStreamRoom + remaining_stock 兜底刷写
```

**幂等设计（网络抖动重试）**：
- 强制请求头：`X-Idempotency-Key: <uuid v4>`
- Redis Key: `fp:idem:{user_id}:{item_id}:{key}` → order_id，TTL 10min
- DB 兜底：`fixed_price_purchases` 上 `UNIQUE(item_id, user_id)`，冲突返回 `FP_IDEMPOTENT_REPLAY`

**Lua 脚本原子操作**：
```lua
-- 检查是否已购买 + 扣减库存 + 记录购买者
local bought = redis.call('SISMEMBER', KEYS[2], ARGV[1])
if bought == 1 then return -2 end  -- 已购买
local stock = redis.call('DECR', KEYS[1])
if stock < 0 then
  redis.call('INCR', KEYS[1])  -- 回滚
  return -1  -- 售罄
end
redis.call('SADD', KEYS[2], ARGV[1])
return stock  -- 返回剩余库存
```

**WebSocket 消息类型**：
- `fixed_price_listed` — 主播上架
- `fixed_price_stock` — 库存变化（节流 1 条/秒/item）
- `fixed_price_sold_out` — 售罄
- `fixed_price_offline` — 下架
- `fixed_price_flair` — 购买成功飘屏

**商品状态机**：
```
draft → on_sale → sold_out → offline (终态)
   ↓       ↓         ↓
  上架   自动售罄   主动下架
```
- 关键规则：`on_sale → offline` 是软标记，不改 Redis；5s 后异步清 Redis
- 不存在 `offline → on_sale` 复活路径（MVP 简化）

**对现有系统影响**：
- `auctions` 表 / 出价引擎 / 状态机：**完全不动**
- `LiveStreamRoom`：仅新增 5 个 message type，不改协议
- `orders` 表：新增 `source` 列（0=auction, 1=fixed_price），DDL 兼容
- `user_balances`：复用已有扣款方法，零改动
- 测试平台：`/test/*` 脚本针对英式拍卖，一口价完全独立，**零影响**

**里程碑拆分**：
- **M1**：后端核心抢购链路（DDL + Lua + 购买接口 + 单元测试）
- **M2**：实时同步层（WS 消息 + Outbox 路由 + 节流策略）
- **M3**：前端 H5 + 监控接入（Prometheus/Grafana）

**来源**：session:6a1c5b0b959156a8dfc850b7

### 本地商家账号创建约束 (Local Merchant Account Creation)

**问题背景**：本地开发时需要商家/主播账号进行测试，但注册接口默认只能创建普通用户(role=0)。

**根因分析**：
- 注册接口 `/api/v1/auth/register` 只能创建 `role=0` 的普通用户
- 商家角色 `role=1` 无法通过正常注册流程获得
- 需要"注册后提权"或"直接写库"创建商家账号

**解决方案**：
1. **方案 A（推荐）**：直接写库创建商家账号
   - 在本地 MySQL 中直接插入 `role=1` 的用户记录
   - 同时配置 `email` 和 `phone`，确保 Admin 和 H5 两边都能登录
   - 密码使用 bcrypt 哈希后的值

2. **方案 B**：注册后提权
   - 先通过注册接口创建普通用户
   - 再执行 SQL 更新 `role=0` → `role=1`

**账号约定**：
- 本地测试商家账号建议统一使用特定号段（如 `18600000001` 起）
- 同时配置 `email` 和 `phone` 字段，兼容不同登录方式
- 商家账号可用于 H5 直播间开播、Admin 后台商品管理等场景

**来源**：session:6a2145b92ec60aa1a7396725

### 统一演示账号 Seed 设计模式 (Unified Demo Account Seed Pattern)

**问题背景**：本地开发和 demo 生产环境使用不同的演示账号体系（本地用 186 号段，线上用 138 号段），导致环境切换时登录凭据混乱，且旧演示数据可能占用关键登录标识（如 `merchant@example.com`）。

**统一 Seed 设计决策**：
1. **账号体系统一**：所有环境使用同一套演示账号（Buyer A/B、Merchant、Admin），固定 ID（9101-9104）和统一密码
2. **号段区分**：本地用 `186` 号段，线上用 `138` 号段，但账号角色和 ID 保持一致
3. **幂等写入**：脚本需支持重复执行，已存在账号则更新（upsert），缺失字段自动补齐
4. **冲突处理**：当旧演示数据占用关键登录标识时，优先释放标识（归档旧账号的 phone/email）而非静默覆盖

**脚本契约**：
```bash
# 脚本必须验证的契约
- 四个账号落库（Buyer A/B、Merchant、Admin）
- 计数为 4（幂等验证）
- 密码 hash 唯一（bcrypt 正确生成）
- 能通过真实登录链路（非仅写库）
```

**关键实现细节**：
- 临时表使用 `utf8mb4_unicode_ci` collation 保证跨环境兼容
- 密码使用 bcrypt 哈希，cost 值与后端登录一致
- 同时配置 `email` 和 `phone`，兼容不同登录方式
- 固定 ID（9101-9104）确保前端逻辑（如天灯归属判断）可稳定匹配

**验证策略**：
- 直接对 Docker MySQL 执行验证（`docker exec`），不使用 host fallback
- 登录 smoke 测试：验证账号能通过真实登录链路而非仅写库
- 脚本级契约测试：作为 `scripts/test-deploy-prod-scripts.sh` 的一部分

**来源**：session:6a2445bb00057ea64ca2701b

---

### 测试平台 Fixture 默认图片策略 (Test Platform Fixture Default Image Strategy)

**问题背景**：独立测试平台自动创建的竞拍品在 H5 移动端显示"暂无图片"，因为测试 fixture 创建商品时未传入 `Images` 字段。

**三种方案对比**：

| 方案 | 实现位置 | 优点 | 缺点 | 推荐度 |
|------|----------|------|------|--------|
| 1. 前端兜底 | `frontend/h5/src/utils/imageFallback.ts` | 成本最低 | 只是展示补丁，各端重复兜底 | ⭐ |
| 2. **测试平台 Fixture 层补图** | `backend/test/client/auction.go` | 数据层根治，所有展示端受益 | 仅影响测试平台创建的数据 | ⭐⭐⭐ |
| 3. product-service 自动写默认图 | `backend/product/service/product.go` | 全局生效 | 掩盖真实商家商品缺图问题 | ⭐ |

**推荐方案（方案2）实现要点**：
1. **公共入口**：在 `backend/test/client/auction.Client.CreateProductAs` 中统一处理
2. **默认图定义**：在 `backend/test/client/auction` 或测试 fixture helper 中定义 `DefaultProductImages`
3. **填充逻辑**：当 `CreateProductReq.Images` 为空时填入默认图片；场景自己传了图片则保留场景图片
4. **受益范围**：E2E、UserJourney、AntiSnipe、Pressure 等所有测试平台创建商品的场景自动受益

**关键代码模式**：
```go
// backend/test/client/auction/client.go
var DefaultProductImages = []string{
    "https://example.com/default-auction-cover.svg",
}

func (c *Client) CreateProductAs(ctx context.Context, merchantID int64, req CreateProductReq) (*Product, error) {
    // 自动填充默认图片
    if len(req.Images) == 0 {
        req.Images = DefaultProductImages
    }
    // ... 继续创建逻辑
}
```

**设计原则**：
- 数据源治理优于前端兜底：让测试平台生成的竞拍品天然带图，而非依赖各端重复兜底
- 默认图使用公网可达 URL（避免 `copilot-cn.bytedance.net` 等内网域名），确保本地和演示环境都可用
- 仅对测试平台自动创建商品生效，不影响真实商家商品数据

**来源**：session:6a242a7d00057ea64ca26118

### 设计文档审查清单 (Design Document Review Checklist)

**问题背景**：复杂功能的设计文档需要多轮审查才能定稿，常见问题包括业务边界不清晰、跨服务边界表述错误、旧引用残留等。

**审查维度**：

| 维度 | 检查项 | 常见陷阱 |
|------|--------|----------|
| 业务边界 | 状态机定义是否完整 | "待开始"是瞬时态还是支持预约 |
| 跨服务边界 | 事务边界是否清晰 | 声称"同一事务"但涉及跨服务写操作 |
| 数据一致性 | 唯一性约束如何实现 | 只说"校验唯一"未说明并发安全方案 |
| 旧引用 | 删除的字段是否还有残留 | `result_status` 改为 `winner_id` 后文档中仍有旧引用 |
| 编号/格式 | 章节编号是否重复 | 两个 `6.3` 章节 |
| 测试链路 | test-service 是否被影响 | 只覆盖 Admin 前端，未说明 fixture 链路适配 |

**关键决策点**：
1. **直播间归属**：明确由哪个服务创建/管理直播间，跨服务时如何通过 API 协作
2. **活跃唯一性**：同一商品多个活跃竞拍的并发控制方案（生成列+唯一索引 vs 锁表 vs Redis 锁）
3. **规则模板边界**：前端先 apply 还是后端统一编排，接口契约需一致
4. **状态派生 vs 存储**：列表筛选状态是实时计算还是冗余存储

**来源**：session:6a24571300057ea64ca27a83

## Child Knowledge Nodes
- `./frontend/h5/SKILL.md` — H5 用户端：首页、直播间、个人中心、图片兜底策略、足迹功能、移动端布局约束，以及直播间战况热度条 (BidHeatBar) 的 UX 增强决策。
- `./frontend/admin/SKILL.md` — Admin 管理后台：角色权限、管理端 API、GrowthBook、编码修复、统计与测试约束。
- `./frontend/test-dashboard/SKILL.md` — Test Dashboard：测试任务、WebSocket 进度流、演示大屏、报告轮询，以及剧场模式 (Chaos Theater) 的 UX 增强决策。
- `./backend/product/SKILL.md` — product-service：商品管理、AI 文案生成、LLM 供应商集成、Nacos 配置管理。
- `./backend/auction/SKILL.md` — auction-service：竞拍核心、点天灯、出价引擎、通知系统、WebSocket 实时同步、商品提醒热拉。

## Feature Knowledge

### 用户触达系统 (User Touchpoints)

**功能概述**：H5 端的用户触达能力，包括红点提醒、Toast 通知、开播提醒弹窗，形成移动端一期触达能力的最小闭环。

**核心组件**：
- **BadgeDot**：红点组件，支持纯红点、数字、`99+`、不展示四种状态
- **Toast**：通知提示组件，支持 success/warning/danger/error/info/loading 样式，最多 3 条堆叠
- **LiveReminderModal**：开播提醒弹窗，登录后一次性展示

**技术架构**：
```
H5 前端 → Gateway BFF → auction-service (通知汇总)
                ↓
         product-service (直播间元数据)
```

**关键接口**：
- `GET /api/v1/notifications/unread-count` — 未读消息数
- `GET /api/v1/notifications/summary` — 红点汇总（含待支付订单数）
- `GET /api/v1/notifications/hot-pull` — 热拉实时通知
- `POST /api/v1/live-streams/{id}/remind` — 开播提醒设置

**实现要点**：
- BadgeDot 支持 `count`、`max`、`dot`、`ariaLabel`、`className` 属性
- Toast 支持对象签名 `showToast({ type, title, message, duration, actionText, onAction })`
- 开播提醒弹窗在 `MobileContainer` 首次挂载时检查 `pending_live_reminder` 标记
- 所有 UI 组件使用 theme-ready CSS 变量，支持未来日间/夜间一键切换

**埋点追踪**：
- 前端统一 `trackEvent()` 封装，上报到 `POST /api/v1/track`
- Gateway 记录 `touchpoint_event_total` Prometheus 指标
- 核心事件：红点曝光、入口点击、通知列表曝光、通知点击、标记已读、热拉触发、开播提醒曝光/点击/关闭

**来源**：session:6a1a57f7959156a8dfc8139e

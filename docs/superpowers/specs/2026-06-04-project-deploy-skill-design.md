# Project Deploy Skill Design

## 目标

把本项目的本地部署和线上 demo 部署沉淀为可复用 skill，让用户通过 `/dp-dev` 和 `/dp-prod` 触发标准部署流程。

核心目标不是创建两个命令别名，而是把部署前检查、风险确认、执行步骤、验证和回滚策略固化为统一协议，减少重复排障和误操作。

## 命令语义

### `/dp-prod`

用于部署线上 demo 环境。

- 代码来源：固定使用 `origin/main` 最新提交。
- 执行模式：确认后执行。
- 目标环境：火山引擎 ECS `14.103.53.55`。
- 登录方式：`root` + SSH 私钥 `/Users/bytedance/Downloads/dy-auction.pem`。
- 部署入口：以 `deploy/demo/MAIN_DEPLOY_QUICKSTART.md` 为 SSOT。
- 默认不部署：`test-service`、`grafana`、`prometheus`、`growthbook`。

`/dp-prod` 在执行任何线上变更前，必须先输出部署计划并等待用户确认。

### `/dp-dev`

用于重建本地开发环境。

- 代码来源：固定使用 `origin/main` 最新提交。
- 执行模式：强制重启。
- 目标环境：本机 macOS。
- 本地基础设施：MySQL `3306`、Redis `6379`、RabbitMQ `5672`。
- 后端端口：Gateway `8080`、Product `8081`、Auction HTTP `8082`、Auction WS `8083`。
- 前端端口：H5 `5173`、Admin `5175`。

`/dp-dev` 可以停止本地旧进程和冲突容器，但必须只针对本项目已知端口与服务，不得扩大清理范围。

## 总体架构

采用 `Skill + 仓库脚本` 的混合方案。

### Skill 负责

- 识别 `/dp-prod` 与 `/dp-dev` 意图。
- 读取项目部署规则和上下文。
- 检查 Git 状态、端口、依赖、服务器连通性。
- 判断应该全量部署还是增量部署。
- 对 `/dp-prod` 输出部署计划、风险、回滚点，并等待用户确认。
- 调用仓库脚本执行部署。
- 汇总验证结果和后续排障建议。

### 仓库脚本负责

- 执行可复用、可测试、可版本化的部署动作。
- 承载容易变化的 shell 命令。
- 通过参数区分 `plan`、`apply`、`verify`、`rollback`。
- 输出结构化日志，方便 skill 解析。

建议新增或标准化以下脚本：

- `scripts/deploy-prod.sh`
- `scripts/deploy-dev.sh`

## `/dp-prod` 流程

### 1. 预检查

必须检查：

- 当前仓库是否能访问 `origin/main`。
- `origin/main` 最新提交 ID。
- 当前工作区是否存在未提交改动。
- 本地是否已同步远程 main。
- SSH 是否可连接 `root@14.103.53.55`。
- 远端部署目录是否存在：
  - `/srv/auction/app`
  - `/srv/auction/env/.env.demo`
  - `/var/www/auction-h5`
  - `/var/www/auction-admin`
- 远端是否存在 Nginx 配置：
  - `/etc/nginx/sites-available/auction-demo.conf`
- 远端 Docker Compose 是否可用。

如果任一关键检查失败，停止并给出根因，不进入部署。

### 2. 生成部署计划

部署计划必须包含：

- 待部署提交：`origin/main@<sha>`。
- 远端当前提交或部署版本。
- 变更类型判断：
  - 仅前端变更。
  - 仅后端变更。
  - 配置/Nginx 变更。
  - 全量变更。
- 预计动作：
  - 构建 H5。
  - 构建 Admin。
  - 同步静态资源。
  - 同步后端源码。
  - 重建后端容器。
  - 重载 Nginx。
- 验证命令。
- 回滚点和回滚方式。

计划输出后，必须等待用户明确确认后才执行线上变更。

### 3. 执行部署

确认后执行：

- 本地 checkout 或校验 `origin/main`。
- H5 构建：`frontend/h5 npm run build`。
- Admin 构建：使用 `npx vite build --base=/admin/`，不要直接依赖默认 `npm run build`。
- 使用 `rsync` 同步静态产物到：
  - `/var/www/auction-h5`
  - `/var/www/auction-admin`
- 同步后端源码到 `/srv/auction/app`。
- 在服务器执行：
  - `docker compose --env-file /srv/auction/env/.env.demo -f docker-compose.demo.yml up -d --build`
  - `nginx -t`
  - `systemctl reload nginx`

### 4. 验证

必须验证：

- `curl -I http://14.103.53.55/`
- `curl -I http://14.103.53.55/admin/`
- `curl http://14.103.53.55/api/v1/products`
- 远端容器状态：
  - `gateway`
  - `product`
  - `auction`
- 必要时检查远端日志：
  - `docker compose --env-file /srv/auction/env/.env.demo -f docker-compose.demo.yml logs --tail=100 gateway product auction`

不使用 `/api/v1/health` 作为唯一验证依据，因为当前项目未统一提供该探针。

### 5. 回滚

`/dp-prod` 必须在计划阶段记录回滚点。

推荐回滚策略：

- 前端回滚：保留上一版静态资源目录备份，失败时恢复。
- 后端回滚：远端源码回到上一提交，重新执行 Docker Compose。
- Nginx 回滚：保留上一版配置文件备份，执行 `nginx -t` 后 reload。

## `/dp-dev` 流程

### 1. 预检查

必须检查：

- 当前仓库可访问 `origin/main`。
- 当前本地工作区状态。
- 本地端口占用：
  - `3306`
  - `6379`
  - `5672`
  - `8080`
  - `8081`
  - `8082`
  - `8083`
  - `5173`
  - `5175`
- Docker 是否可用。
- Go、Node、npm 是否可用。

### 2. 代码同步

`/dp-dev` 默认部署 `origin/main`。

如果当前工作区存在未提交改动，不应直接覆盖。推荐策略：

- 若工作区干净：执行 `git pull --ff-only` 或等价同步。
- 若工作区不干净：提示用户当前有本地改动，并建议使用隔离 worktree 从 `origin/main` 启动本地服务。

### 3. 强制重启本地服务

允许清理：

- `5173`、`5175` 上的 Vite 进程。
- `8080`、`8081`、`8082`、`8083` 上的本项目 Go 后端进程。
- 与本项目冲突的 `gateway`、`product`、`auction` Docker 容器。

不得清理：

- 未确认属于本项目的随机系统进程。
- 用户未授权的其他服务。
- 非本项目 Docker 容器。

### 4. 启动顺序

推荐执行：

```bash
INTERNAL_API_TOKEN=dev docker compose up -d mysql redis rabbitmq
./scripts/start-local-backend.sh restart
./scripts/start-frontend.sh
```

如果 `scripts/start-frontend.sh` 的 kill 行为过强，后续实现可拆分为更安全的 `scripts/deploy-dev.sh`。

### 5. 验证

必须验证：

- `http://localhost:5173`
- `http://localhost:5175`
- `http://localhost:8080`
- `http://localhost:8081`
- `http://localhost:8082`
- `ws://localhost:8083/ws`

后端根路径返回 `404` 可以视为服务已监听，但不能证明业务接口可用。应至少补充一个业务接口验证，例如：

- `curl http://localhost:8080/api/v1/products`

## Skill 触发设计

建议创建一个部署 skill，例如 `project-deploy`。

描述中显式包含触发词：

- `/dp-prod`
- `/dp-dev`
- 部署线上
- 部署本地
- 重启本地环境
- 发布 demo 环境

如果当前 Agent 平台支持自定义 slash command，则将：

- `/dp-prod` 映射到 `project-deploy` 的 prod 模式。
- `/dp-dev` 映射到 `project-deploy` 的 dev 模式。

如果平台不支持真正的 slash command，则通过 skill description 触发，用户输入 `/dp-prod` 或 `/dp-dev` 时由 skill 解析命令文本。

## 脚本接口设计

### `scripts/deploy-prod.sh`

建议支持：

```bash
scripts/deploy-prod.sh plan
scripts/deploy-prod.sh apply
scripts/deploy-prod.sh verify
scripts/deploy-prod.sh rollback
```

环境变量：

- `DEPLOY_HOST=14.103.53.55`
- `DEPLOY_USER=root`
- `DEPLOY_KEY=/Users/bytedance/Downloads/dy-auction.pem`
- `DEPLOY_REF=origin/main`

### `scripts/deploy-dev.sh`

建议支持：

```bash
scripts/deploy-dev.sh restart
scripts/deploy-dev.sh status
scripts/deploy-dev.sh verify
scripts/deploy-dev.sh stop
```

默认行为：

- `restart` 执行强制本地重启。
- `status` 只读检查。
- `verify` 只验证。
- `stop` 停止本项目本地服务。

## 安全约束

- `/dp-prod` 不允许无确认修改线上。
- `/dp-prod` 不提交、覆盖或泄露 `.env.demo`。
- `/dp-prod` 不把 `ARK_API_KEY`、`JWT_SECRET`、`INTERNAL_API_TOKEN` 打印到日志。
- `/dp-dev` 不通过修改主干配置来规避 `localhost`、IPv6 或端口问题。
- 前端流量必须经 Gateway `/api/v1`。
- 不允许前端直连后端子服务。
- 部署前不得自动丢弃用户本地未提交改动。
- 所有验证必须基于新鲜命令输出，不能只凭之前的运行结果。

## 错误处理

### Git 不同步

- `/dp-prod`：停止，提示先同步或确认部署指定远程提交。
- `/dp-dev`：如果工作区不干净，建议使用隔离 worktree 或请求用户确认处理方式。

### SSH 不可达

- 停止线上部署。
- 输出 SSH 命令和可能原因：
  - 私钥路径不存在。
  - 私钥权限错误。
  - 服务器不可达。
  - root 登录被禁用。

### 构建失败

- 停止部署。
- 输出失败项目、命令、关键日志。
- 不同步部分产物到线上。

### 容器启动失败

- 保留远端日志。
- 输出 `docker compose ps` 和对应服务日志。
- 给出回滚命令。

### 验证失败

- 标记部署未完成。
- 输出失败验证项。
- 提供回滚建议，不声称部署成功。

## 测试与验收

### `/dp-prod` 验收

- 输入 `/dp-prod` 后先输出部署计划，不直接 SSH 修改线上。
- 用户确认后才执行部署。
- 能部署 `origin/main` 到 `14.103.53.55`。
- H5、Admin、API 三类入口验证通过。
- 部署失败时能停止并给出根因和回滚方案。

### `/dp-dev` 验收

- 输入 `/dp-dev` 后执行本地强制重启。
- 能处理旧进程和端口冲突。
- 能启动基础设施、后端、前端。
- 能验证 `5173`、`5175`、`8080`、`8081`、`8082`、`8083`。
- 不修改主干配置来绕过本机环境问题。

## 非目标

- 不把生产密钥提交到仓库。
- 不把 demo 部署扩展成完整多环境发布平台。
- 不引入 Kubernetes、CI/CD 平台或蓝绿发布。
- 不处理监控组件部署。
- 不改变现有业务服务架构。

## 后续实施顺序

1. 创建 `scripts/deploy-dev.sh`，先收敛本地强制重启和验证。
2. 创建 `scripts/deploy-prod.sh plan/verify`，先实现只读计划和验证。
3. 补齐 `scripts/deploy-prod.sh apply`，接入用户确认门禁。
4. 创建 `project-deploy` skill，解析 `/dp-dev` 与 `/dp-prod`。
5. 用 `/dp-dev` 做本地闭环验证。
6. 用 `/dp-prod` 先跑 plan，再执行一次线上 demo 部署验证。

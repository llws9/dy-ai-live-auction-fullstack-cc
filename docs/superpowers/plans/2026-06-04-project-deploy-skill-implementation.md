# Project Deploy Skill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a project deployment skill that lets the user trigger local deployment with `/dp-dev` and production demo deployment with `/dp-prod`.

**Architecture:** Keep deployment mechanics in versioned repository scripts and keep the skill focused on intent parsing, safety gates, confirmations, and result interpretation. `/dp-dev` performs a strong local restart from `origin/main`; `/dp-prod` first produces a plan and only applies changes after explicit user confirmation.

**Tech Stack:** Bash, Git, Docker Compose, npm/Vite, Go services, rsync, SSH, Trae skill markdown.

---

## File Structure

- Create: `scripts/deploy-dev.sh`
  - Owns local status, stop, restart, and verify flows for H5/Admin/Gateway/Product/Auction.
  - Reuses `scripts/start-local-backend.sh` and `scripts/start-frontend.sh` rather than duplicating service startup logic.
- Create: `scripts/deploy-prod.sh`
  - Owns prod plan, apply, verify, and rollback commands for the demo ECS environment.
  - Uses `deploy/demo/MAIN_DEPLOY_QUICKSTART.md` as the deployment SSOT.
- Create: `/Users/bytedance/.trae-cn/skills/project-deploy/SKILL.md`
  - Defines skill trigger behavior for `/dp-dev` and `/dp-prod`.
  - Must be created only after invoking the `skill-creator` skill in the implementation session.
- Modify: `README.md`
  - Adds a short note that `/dp-dev` and `/dp-prod` exist as agent skill entry points.
- Test/verify: shell syntax checks, local read-only status checks, prod plan checks, and skill trigger review.

---

### Task 1: Add Local Deployment Script

**Files:**
- Create: `scripts/deploy-dev.sh`

- [ ] **Step 1: Write the script file**

Create `scripts/deploy-dev.sh` with this exact content:

```bash
#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

usage() {
  cat <<'EOF'
用法: scripts/deploy-dev.sh <status|verify|restart|stop>

说明:
  status   只读检查本地端口、Git 和 Docker 基础设施状态
  verify   验证本地前后端端口和核心 API 是否可访问
  restart  强制重启本地基础设施、后端和前端
  stop     停止本项目本地前后端服务
EOF
}

require_cmd() {
  local cmd=$1
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo -e "${RED}错误: 缺少命令 $cmd${NC}" >&2
    exit 1
  fi
}

port_owner() {
  local port=$1
  lsof -nP -iTCP:"$port" -sTCP:LISTEN 2>/dev/null || true
}

print_port_status() {
  for port in 3306 6379 5672 8080 8081 8082 8083 5173 5175; do
    echo "=== port $port ==="
    port_owner "$port" || true
  done
}

assert_origin_main() {
  cd "$PROJECT_ROOT"
  git fetch origin main >/dev/null
  local head
  local remote
  head="$(git rev-parse HEAD)"
  remote="$(git rev-parse origin/main)"
  if [[ "$head" != "$remote" ]]; then
    echo -e "${RED}错误: 当前 HEAD 不等于 origin/main${NC}" >&2
    echo "HEAD:        $head"
    echo "origin/main: $remote"
    echo "请先同步到 origin/main，或在隔离 worktree 中执行本地部署。"
    exit 1
  fi
}

assert_clean_tree() {
  cd "$PROJECT_ROOT"
  if [[ -n "$(git status --porcelain)" ]]; then
    echo -e "${RED}错误: 当前工作区存在未提交改动，拒绝从本目录强制部署${NC}" >&2
    git status --short
    echo "建议先提交/暂存改动，或创建干净 worktree 后执行 /dp-dev。"
    exit 1
  fi
}

stop_frontend_ports() {
  for port in 5173 5175; do
    local pids
    pids="$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
    if [[ -n "$pids" ]]; then
      echo -e "${YELLOW}停止端口 $port 上的前端进程: $pids${NC}"
      kill $pids 2>/dev/null || true
      sleep 1
      pids="$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
      if [[ -n "$pids" ]]; then
        kill -9 $pids 2>/dev/null || true
      fi
    fi
  done
}

stop_backend() {
  cd "$PROJECT_ROOT"
  ./scripts/start-local-backend.sh stop || true
  for port in 8080 8081 8082 8083; do
    local pids
    pids="$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
    if [[ -n "$pids" ]]; then
      echo -e "${YELLOW}停止端口 $port 上的后端进程: $pids${NC}"
      kill $pids 2>/dev/null || true
      sleep 1
      pids="$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
      if [[ -n "$pids" ]]; then
        kill -9 $pids 2>/dev/null || true
      fi
    fi
  done
}

stop_conflicting_containers() {
  cd "$PROJECT_ROOT"
  INTERNAL_API_TOKEN=dev docker compose stop gateway product auction >/dev/null 2>&1 || true
}

start_infra() {
  cd "$PROJECT_ROOT"
  INTERNAL_API_TOKEN=dev docker compose up -d mysql redis rabbitmq
}

start_backend() {
  cd "$PROJECT_ROOT"
  ./scripts/start-local-backend.sh start
}

start_frontend() {
  cd "$PROJECT_ROOT"
  ./scripts/start-frontend.sh
}

curl_code() {
  local url=$1
  curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$url" || true
}

verify_local() {
  local failed=0
  local code

  for url in \
    "http://localhost:5173" \
    "http://localhost:5175" \
    "http://localhost:8080" \
    "http://localhost:8081" \
    "http://localhost:8082"; do
    code="$(curl_code "$url")"
    echo "$url -> $code"
    if [[ "$code" == "000" ]]; then
      failed=1
    fi
  done

  code="$(curl_code "http://localhost:8080/api/v1/products")"
  echo "http://localhost:8080/api/v1/products -> $code"
  if [[ "$code" == "000" ]]; then
    failed=1
  fi

  if [[ -z "$(port_owner 8083)" ]]; then
    echo -e "${RED}ws://localhost:8083/ws -> 端口未监听${NC}"
    failed=1
  else
    echo "ws://localhost:8083/ws -> 端口已监听"
  fi

  if [[ "$failed" -ne 0 ]]; then
    echo -e "${RED}本地验证失败${NC}" >&2
    exit 1
  fi
  echo -e "${GREEN}本地验证通过${NC}"
}

show_status() {
  cd "$PROJECT_ROOT"
  echo "=== Git ==="
  git status --short --branch
  echo "HEAD:        $(git rev-parse HEAD)"
  git fetch origin main >/dev/null
  echo "origin/main: $(git rev-parse origin/main)"
  echo ""
  echo "=== Docker infra ==="
  INTERNAL_API_TOKEN=dev docker compose ps mysql redis rabbitmq || true
  echo ""
  echo "=== Ports ==="
  print_port_status
}

restart_all() {
  require_cmd git
  require_cmd docker
  require_cmd lsof
  require_cmd curl
  require_cmd go
  require_cmd npm
  assert_origin_main
  assert_clean_tree
  stop_frontend_ports
  stop_backend
  stop_conflicting_containers
  start_infra
  start_backend
  start_frontend
  verify_local
}

stop_all() {
  stop_frontend_ports
  stop_backend
}

case "${1:-}" in
  status)
    require_cmd git
    require_cmd docker
    require_cmd lsof
    show_status
    ;;
  verify)
    require_cmd curl
    require_cmd lsof
    verify_local
    ;;
  restart)
    restart_all
    ;;
  stop)
    stop_all
    ;;
  help|--help|-h|"")
    usage
    ;;
  *)
    usage
    exit 1
    ;;
esac
```

- [ ] **Step 2: Make the script executable**

Run:

```bash
chmod +x scripts/deploy-dev.sh
```

Expected: command exits with code `0`.

- [ ] **Step 3: Run shell syntax validation**

Run:

```bash
bash -n scripts/deploy-dev.sh
```

Expected: command exits with code `0` and prints no output.

- [ ] **Step 4: Run read-only status check**

Run:

```bash
scripts/deploy-dev.sh status
```

Expected:

```text
=== Git ===
```

The command may show port owners, Docker containers, or a dirty tree. That is acceptable because `status` is read-only.

- [ ] **Step 5: Commit local deploy script**

Run:

```bash
git add scripts/deploy-dev.sh
git commit -m "feat: add local deploy script"
```

Expected: one commit containing only `scripts/deploy-dev.sh`.

---

### Task 2: Add Production Deployment Script

**Files:**
- Create: `scripts/deploy-prod.sh`

- [ ] **Step 1: Write the script file**

Create `scripts/deploy-prod.sh` with this exact content:

```bash
#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

DEPLOY_HOST="${DEPLOY_HOST:-14.103.53.55}"
DEPLOY_USER="${DEPLOY_USER:-root}"
DEPLOY_KEY="${DEPLOY_KEY:-/Users/bytedance/Downloads/dy-auction.pem}"
DEPLOY_REF="${DEPLOY_REF:-origin/main}"
REMOTE_APP_DIR="${REMOTE_APP_DIR:-/srv/auction/app}"
REMOTE_ENV_FILE="${REMOTE_ENV_FILE:-/srv/auction/env/.env.demo}"
REMOTE_H5_DIR="${REMOTE_H5_DIR:-/var/www/auction-h5}"
REMOTE_ADMIN_DIR="${REMOTE_ADMIN_DIR:-/var/www/auction-admin}"
REMOTE_NGINX_CONF="${REMOTE_NGINX_CONF:-/etc/nginx/sites-available/auction-demo.conf}"
REMOTE_BACKUP_DIR="${REMOTE_BACKUP_DIR:-/srv/auction/backups}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

usage() {
  cat <<'EOF'
用法: scripts/deploy-prod.sh <plan|apply|verify|rollback>

说明:
  plan      只读生成线上部署计划，不修改服务器
  apply     执行线上部署；调用前必须由 skill 获取用户确认
  verify    验证线上 H5、Admin、API 和容器状态
  rollback  使用最近一次静态资源备份做前端回滚，并提示后端回滚命令
EOF
}

require_cmd() {
  local cmd=$1
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo -e "${RED}错误: 缺少命令 $cmd${NC}" >&2
    exit 1
  fi
}

ssh_base() {
  ssh -i "$DEPLOY_KEY" -o BatchMode=yes -o StrictHostKeyChecking=accept-new "$DEPLOY_USER@$DEPLOY_HOST" "$@"
}

scp_base() {
  scp -i "$DEPLOY_KEY" -o BatchMode=yes -o StrictHostKeyChecking=accept-new "$@"
}

rsync_base() {
  rsync -az --delete -e "ssh -i $DEPLOY_KEY -o BatchMode=yes -o StrictHostKeyChecking=accept-new" "$@"
}

assert_key() {
  if [[ ! -f "$DEPLOY_KEY" ]]; then
    echo -e "${RED}错误: SSH 私钥不存在: $DEPLOY_KEY${NC}" >&2
    exit 1
  fi
}

local_sha() {
  cd "$PROJECT_ROOT"
  git fetch origin main >/dev/null
  git rev-parse "$DEPLOY_REF"
}

remote_sha() {
  ssh_base "cd '$REMOTE_APP_DIR' 2>/dev/null && git rev-parse HEAD 2>/dev/null || true"
}

assert_clean_for_ref() {
  cd "$PROJECT_ROOT"
  git fetch origin main >/dev/null
  local head
  local target
  head="$(git rev-parse HEAD)"
  target="$(git rev-parse "$DEPLOY_REF")"
  if [[ "$head" != "$target" ]]; then
    echo -e "${RED}错误: 当前 HEAD 不等于 $DEPLOY_REF${NC}" >&2
    echo "HEAD:       $head"
    echo "$DEPLOY_REF: $target"
    echo "请在干净且同步的 main 上执行，或在隔离 worktree 中执行。"
    exit 1
  fi
  if [[ -n "$(git status --porcelain)" ]]; then
    echo -e "${RED}错误: 当前工作区存在未提交改动，拒绝线上部署${NC}" >&2
    git status --short
    exit 1
  fi
}

remote_precheck() {
  assert_key
  ssh_base "test -d '$REMOTE_APP_DIR' && test -f '$REMOTE_ENV_FILE' && test -d '$REMOTE_H5_DIR' && test -d '$REMOTE_ADMIN_DIR' && test -f '$REMOTE_NGINX_CONF' && docker compose version >/dev/null && nginx -v >/dev/null"
}

changed_files() {
  cd "$PROJECT_ROOT"
  local remote
  remote="$(remote_sha)"
  if [[ -z "$remote" ]]; then
    git diff --name-only "$(git rev-list --max-parents=0 HEAD)" "$DEPLOY_REF"
  else
    git diff --name-only "$remote" "$DEPLOY_REF"
  fi
}

classify_changes() {
  local files
  files="$(changed_files)"
  local frontend=0
  local backend=0
  local config=0

  if echo "$files" | grep -E '^(frontend/h5|frontend/admin)/' >/dev/null; then
    frontend=1
  fi
  if echo "$files" | grep -E '^(backend/|docker-compose.demo.yml|go.work|go.mod|go.sum)' >/dev/null; then
    backend=1
  fi
  if echo "$files" | grep -E '^(\.env\.demo\.example|deploy/demo/|docker-compose.demo.yml)' >/dev/null; then
    config=1
  fi

  echo "frontend=$frontend backend=$backend config=$config"
}

plan() {
  require_cmd git
  require_cmd ssh
  require_cmd rsync
  require_cmd npm
  require_cmd scp
  assert_key
  local target
  local remote
  target="$(local_sha)"
  remote="$(remote_sha)"

  echo "=== /dp-prod 部署计划 ==="
  echo "目标服务器: $DEPLOY_USER@$DEPLOY_HOST"
  echo "部署引用:   $DEPLOY_REF"
  echo "待部署提交: $target"
  echo "远端提交:   ${remote:-unknown}"
  echo "变更分类:   $(classify_changes)"
  echo ""
  echo "预计动作:"
  echo "- 构建 frontend/h5"
  echo "- 以 --base=/admin/ 构建 frontend/admin"
  echo "- 备份远端静态资源到 $REMOTE_BACKUP_DIR"
  echo "- 同步 H5 到 $REMOTE_H5_DIR"
  echo "- 同步 Admin 到 $REMOTE_ADMIN_DIR"
  echo "- 同步仓库源码到 $REMOTE_APP_DIR"
  echo "- 执行 docker compose --env-file $REMOTE_ENV_FILE -f docker-compose.demo.yml up -d --build"
  echo "- 执行 nginx -t && systemctl reload nginx"
  echo ""
  echo "验证:"
  echo "- curl -I http://$DEPLOY_HOST/"
  echo "- curl -I http://$DEPLOY_HOST/admin/"
  echo "- curl http://$DEPLOY_HOST/api/v1/products"
  echo "- docker compose ps gateway product auction"
  echo ""
  echo "回滚:"
  echo "- 前端静态资源使用 $REMOTE_BACKUP_DIR 最近备份恢复"
  echo "- 后端进入 $REMOTE_APP_DIR checkout 上一提交后重建容器"
}

build_frontend() {
  cd "$PROJECT_ROOT/frontend/h5"
  npm run build
  cd "$PROJECT_ROOT/frontend/admin"
  npx vite build --base=/admin/
}

backup_remote() {
  local stamp
  stamp="$(date +%Y%m%d%H%M%S)"
  ssh_base "mkdir -p '$REMOTE_BACKUP_DIR/$stamp' && cp -a '$REMOTE_H5_DIR' '$REMOTE_BACKUP_DIR/$stamp/auction-h5' && cp -a '$REMOTE_ADMIN_DIR' '$REMOTE_BACKUP_DIR/$stamp/auction-admin' && echo '$stamp' > '$REMOTE_BACKUP_DIR/latest'"
}

sync_frontend() {
  rsync_base "$PROJECT_ROOT/frontend/h5/dist/" "$DEPLOY_USER@$DEPLOY_HOST:$REMOTE_H5_DIR/"
  rsync_base "$PROJECT_ROOT/frontend/admin/dist/" "$DEPLOY_USER@$DEPLOY_HOST:$REMOTE_ADMIN_DIR/"
}

sync_backend() {
  rsync_base \
    --exclude '.git/' \
    --exclude 'node_modules/' \
    --exclude 'frontend/h5/dist/' \
    --exclude 'frontend/admin/dist/' \
    --exclude '.tmp/' \
    "$PROJECT_ROOT/" "$DEPLOY_USER@$DEPLOY_HOST:$REMOTE_APP_DIR/"
}

restart_remote() {
  ssh_base "cd '$REMOTE_APP_DIR' && docker compose --env-file '$REMOTE_ENV_FILE' -f docker-compose.demo.yml up -d --build && nginx -t && systemctl reload nginx"
}

verify_prod() {
  local failed=0
  for url in "http://$DEPLOY_HOST/" "http://$DEPLOY_HOST/admin/" "http://$DEPLOY_HOST/api/v1/products"; do
    local code
    code="$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "$url" || true)"
    echo "$url -> $code"
    if [[ "$code" == "000" ]]; then
      failed=1
    fi
  done
  ssh_base "cd '$REMOTE_APP_DIR' && docker compose --env-file '$REMOTE_ENV_FILE' -f docker-compose.demo.yml ps gateway product auction"
  if [[ "$failed" -ne 0 ]]; then
    echo -e "${RED}线上验证失败${NC}" >&2
    exit 1
  fi
  echo -e "${GREEN}线上验证通过${NC}"
}

apply() {
  require_cmd git
  require_cmd ssh
  require_cmd rsync
  require_cmd npm
  require_cmd curl
  assert_clean_for_ref
  remote_precheck
  build_frontend
  backup_remote
  sync_frontend
  sync_backend
  restart_remote
  verify_prod
}

rollback() {
  assert_key
  ssh_base "test -f '$REMOTE_BACKUP_DIR/latest'"
  local latest
  latest="$(ssh_base "cat '$REMOTE_BACKUP_DIR/latest'")"
  ssh_base "rsync -a --delete '$REMOTE_BACKUP_DIR/$latest/auction-h5/' '$REMOTE_H5_DIR/' && rsync -a --delete '$REMOTE_BACKUP_DIR/$latest/auction-admin/' '$REMOTE_ADMIN_DIR/' && nginx -t && systemctl reload nginx"
  echo "已恢复前端静态资源备份: $latest"
  echo "如需回滚后端，请在服务器 $REMOTE_APP_DIR 中 checkout 上一提交并重新执行 docker compose up -d --build。"
}

case "${1:-}" in
  plan)
    plan
    ;;
  apply)
    apply
    ;;
  verify)
    require_cmd curl
    verify_prod
    ;;
  rollback)
    rollback
    ;;
  help|--help|-h|"")
    usage
    ;;
  *)
    usage
    exit 1
    ;;
esac
```

- [ ] **Step 2: Make the script executable**

Run:

```bash
chmod +x scripts/deploy-prod.sh
```

Expected: command exits with code `0`.

- [ ] **Step 3: Run shell syntax validation**

Run:

```bash
bash -n scripts/deploy-prod.sh
```

Expected: command exits with code `0` and prints no output.

- [ ] **Step 4: Run prod plan**

Run:

```bash
scripts/deploy-prod.sh plan
```

Expected output includes:

```text
=== /dp-prod 部署计划 ===
```

If SSH is unavailable or the key is missing, record the error and stop. Do not move to `apply`.

- [ ] **Step 5: Commit prod deploy script**

Run:

```bash
git add scripts/deploy-prod.sh
git commit -m "feat: add prod deploy script"
```

Expected: one commit containing only `scripts/deploy-prod.sh`.

---

### Task 3: Create Project Deploy Skill

**Files:**
- Create: `/Users/bytedance/.trae-cn/skills/project-deploy/SKILL.md`
- Modify: `README.md`

- [ ] **Step 1: Invoke skill creation guidance**

Before creating the skill file, invoke:

```text
Skill tool: skill-creator
```

Expected: the implementation session loads the skill creation instructions before writing `/Users/bytedance/.trae-cn/skills/project-deploy/SKILL.md`.

- [ ] **Step 2: Create skill directory**

Run:

```bash
mkdir -p /Users/bytedance/.trae-cn/skills/project-deploy
```

Expected: command exits with code `0`.

- [ ] **Step 3: Write the skill file**

Create `/Users/bytedance/.trae-cn/skills/project-deploy/SKILL.md` with this exact content:

```markdown
---
name: project-deploy
description: Use when the user types /dp-dev or /dp-prod, or asks to deploy this dy-ai-live-auction-fullstack-cc project locally or to the demo production server. /dp-dev restarts the local macOS development stack from origin/main. /dp-prod deploys origin/main to the demo ECS server, but must first produce a deployment plan and wait for explicit user confirmation before making online changes.
---

# Project Deploy Skill

## Scope

This skill is only for `/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc`.

It supports two commands:

- `/dp-dev`: strong local restart from `origin/main`.
- `/dp-prod`: demo production deployment to `14.103.53.55`, plan first, confirm before apply.

## Mandatory First Step

Always read:

- `AGENTS.md`
- `docs/superpowers/specs/2026-06-04-project-deploy-skill-design.md`
- `deploy/demo/MAIN_DEPLOY_QUICKSTART.md`

Then run:

```bash
git fetch origin main
git status --short --branch
git rev-parse HEAD
git rev-parse origin/main
```

Do not claim anything is synced until the command output confirms it.

## `/dp-dev` Workflow

Use this when the user enters `/dp-dev`.

1. Explain that `/dp-dev` will force restart local project services.
2. Run read-only status:

```bash
scripts/deploy-dev.sh status
```

3. If status shows the working tree is not clean or HEAD is not `origin/main`, stop and ask whether to create an isolated worktree or sync the current tree.
4. If safe, run:

```bash
scripts/deploy-dev.sh restart
```

5. Verify with:

```bash
scripts/deploy-dev.sh verify
```

6. Report exact URLs:

- H5: `http://localhost:5173`
- Admin: `http://localhost:5175`
- Gateway API: `http://localhost:8080/api/v1`
- Auction WS: `ws://localhost:8083/ws`

Never stop a dev server after giving the user a preview URL unless the user explicitly asks to stop it.

## `/dp-prod` Workflow

Use this when the user enters `/dp-prod`.

1. Run:

```bash
scripts/deploy-prod.sh plan
```

2. Summarize:

- target commit
- remote current commit
- changed areas
- expected actions
- verification commands
- rollback point

3. Ask the user for explicit confirmation before any online mutation.

Use this exact confirmation prompt:

```text
确认执行线上部署吗？回复“确认部署”后我才会执行 apply。
```

4. Only if the user replies exactly `确认部署`, run:

```bash
scripts/deploy-prod.sh apply
```

5. Run fresh verification:

```bash
scripts/deploy-prod.sh verify
```

6. Report success only if the apply and verify commands both exit with code `0`.

## Safety Rules

- `/dp-prod` must never run `apply` before explicit confirmation.
- `/dp-prod` must never print `.env.demo`, `ARK_API_KEY`, `JWT_SECRET`, or `INTERNAL_API_TOKEN`.
- `/dp-prod` must not use `/api/v1/health` as the only health check.
- `/dp-dev` must not change source config to work around localhost, IPv6, or port conflicts.
- Do not use `git reset --hard`, `git checkout --`, or destructive cleanup unless the user explicitly approves.
- Do not silently discard local changes.

## Failure Handling

If a command fails:

1. Read the error output.
2. Identify the failing layer: Git, local ports, SSH, build, rsync, Docker, Nginx, or HTTP verification.
3. Report the exact failing command and root cause.
4. Do not continue to the next phase after a failed phase.

## Completion Report

Final response must include:

- current branch and worktree first line, matching `AGENTS.md`
- command invoked: `/dp-dev` or `/dp-prod`
- target commit
- commands executed
- verification evidence
- remaining risks or follow-up actions
```

- [ ] **Step 4: Add README note**

Append this section near the local deployment section in `README.md`:

```markdown
### Agent 部署命令

项目部署流程已沉淀为 agent skill：

- `/dp-dev`：从 `origin/main` 强制重启本地开发环境，覆盖 H5、Admin、Gateway、Product、Auction。
- `/dp-prod`：从 `origin/main` 部署线上 demo 环境；执行线上变更前必须先输出部署计划并等待确认。

详细设计见 `docs/superpowers/specs/2026-06-04-project-deploy-skill-design.md`。
```

- [ ] **Step 5: Verify skill file structure**

Run:

```bash
head -n 8 /Users/bytedance/.trae-cn/skills/project-deploy/SKILL.md
grep -n "/dp-dev\\|/dp-prod\\|确认部署" /Users/bytedance/.trae-cn/skills/project-deploy/SKILL.md
```

Expected output includes:

```text
name: project-deploy
/dp-dev
/dp-prod
确认部署
```

- [ ] **Step 6: Commit README note**

Run:

```bash
git add README.md
git commit -m "docs: document deploy skill commands"
```

Expected: commit includes only `README.md`.

Note: `/Users/bytedance/.trae-cn/skills/project-deploy/SKILL.md` lives outside the repository and is not committed to this repo.

---

### Task 4: Verify End-to-End Behavior

**Files:**
- Verify: `scripts/deploy-dev.sh`
- Verify: `scripts/deploy-prod.sh`
- Verify: `/Users/bytedance/.trae-cn/skills/project-deploy/SKILL.md`

- [ ] **Step 1: Verify repository script syntax**

Run:

```bash
bash -n scripts/deploy-dev.sh
bash -n scripts/deploy-prod.sh
```

Expected: both commands exit with code `0`.

- [ ] **Step 2: Verify `/dp-dev` read-only path**

Run:

```bash
scripts/deploy-dev.sh status
```

Expected output includes:

```text
=== Git ===
=== Docker infra ===
=== Ports ===
```

- [ ] **Step 3: Verify `/dp-dev` active path only when safe**

Run:

```bash
git status --short --branch
```

If output is exactly clean and `HEAD` equals `origin/main`, run:

```bash
scripts/deploy-dev.sh restart
scripts/deploy-dev.sh verify
```

Expected output includes:

```text
本地验证通过
```

If the working tree is not clean or `HEAD` differs from `origin/main`, do not run `restart`. Report that `/dp-dev` correctly refuses unsafe deployment.

- [ ] **Step 4: Verify `/dp-prod` plan path**

Run:

```bash
scripts/deploy-prod.sh plan
```

Expected output includes:

```text
=== /dp-prod 部署计划 ===
```

This step must not modify the remote server.

- [ ] **Step 5: Verify `/dp-prod` apply gate manually**

Read `/Users/bytedance/.trae-cn/skills/project-deploy/SKILL.md` and confirm it contains this exact line:

```text
Only if the user replies exactly `确认部署`, run:
```

Run:

```bash
grep -n "Only if the user replies exactly.*确认部署" /Users/bytedance/.trae-cn/skills/project-deploy/SKILL.md
```

Expected: one matching line.

- [ ] **Step 6: Optional prod apply verification**

Only run this step if the user explicitly replies `确认部署` in the execution session.

Run:

```bash
scripts/deploy-prod.sh apply
scripts/deploy-prod.sh verify
```

Expected:

```text
线上验证通过
```

- [ ] **Step 7: Final repository state check**

Run:

```bash
git status --short --branch
git log --oneline -5 --decorate
```

Expected:

- Repository changes are committed.
- Any unpushed commits are reported clearly.
- No generated `dist/`, `.env.demo`, secrets, or temporary deployment backups are staged.

- [ ] **Step 8: Completion handoff**

Report:

```text
当前分支/worktree：<branch> @ <absolute-worktree-path>
```

Then include:

- implemented scripts
- created skill path
- commands verified
- whether `/dp-dev restart` was actually run
- whether `/dp-prod apply` was skipped or executed
- current Git ahead/behind state

---

## Self-Review Checklist

- Spec coverage: `/dp-dev`, `/dp-prod`, confirmation gate, origin/main source, script boundary, and verification requirements are covered.
- Safety coverage: no prod apply without exact confirmation; no secrets printed; no destructive git command; no silent local change discard.
- Testing coverage: script syntax, dev status, prod plan, skill confirmation gate, optional active deployment, final Git state.
- Scope check: this plan implements one bounded deployment skill and two repository scripts; it does not introduce CI/CD, Kubernetes, monitoring, or multi-environment deployment.

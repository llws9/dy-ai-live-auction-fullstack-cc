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
  cat <<'USAGE'
用法: scripts/deploy-prod.sh <plan|apply|verify|rollback>

说明:
  plan      只读生成线上部署计划，不修改服务器
  apply     执行线上部署；调用前必须由 skill 获取用户确认
  verify    验证线上 H5、Admin、API 和容器状态
  rollback  使用最近一次静态资源备份做前端回滚，并提示后端回滚命令
USAGE
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
  ssh_base "cd '$REMOTE_APP_DIR' 2>/dev/null && { cat .deploy-ref 2>/dev/null || git rev-parse HEAD 2>/dev/null || true; }"
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
    echo "__ALL__"
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

  if echo "$files" | grep -E '^__ALL__$|^(frontend/h5|frontend/admin)/' >/dev/null; then
    frontend=1
  fi
  if echo "$files" | grep -E '^__ALL__$|^(backend/|docker-compose.demo.yml|go.work|go.mod|go.sum)' >/dev/null; then
    backend=1
  fi
  if echo "$files" | grep -E '^__ALL__$|^(\.env\.demo\.example|deploy/demo/|docker-compose.demo.yml)' >/dev/null; then
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
  ssh_base "cd '$REMOTE_APP_DIR' && echo '$(local_sha)' > .deploy-ref"
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

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
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-auction-demo}"
DEMO_BASIC_AUTH_USER="${DEMO_BASIC_AUTH_USER:-ByteDance}"
DEMO_BASIC_AUTH_PASSWORD="${DEMO_BASIC_AUTH_PASSWORD:-ByteDance}"
REMOTE_BASIC_AUTH_FILE="${REMOTE_BASIC_AUTH_FILE:-/etc/nginx/.auction-demo.htpasswd}"

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

shell_quote() {
  printf '%q' "$1"
}

ssh_dest() {
  echo "$DEPLOY_USER@$DEPLOY_HOST"
}

ssh_opts() {
  echo "-i $(shell_quote "$DEPLOY_KEY") -o BatchMode=yes -o StrictHostKeyChecking=accept-new"
}

redact_sensitive() {
  sed -E \
    -e 's/(ARK_API_KEY=)[^[:space:]]+/\1[REDACTED]/g' \
    -e 's/(JWT_SECRET=)[^[:space:]]+/\1[REDACTED]/g' \
    -e 's/(INTERNAL_API_TOKEN=)[^[:space:]]+/\1[REDACTED]/g'
}

compose_cmd() {
  local env_file=$1
  echo "docker compose --project-name $(shell_quote "$COMPOSE_PROJECT_NAME") --env-file $env_file -f docker-compose.demo.yml"
}

demo_full_stack_services() {
  echo "gateway product auction mysql redis rabbitmq nacos nacos-mysql test-service test-dashboard loki promtail prometheus grafana growthbook growthbook-db"
}

ssh_base() {
  local ssh_opts=(-i "$DEPLOY_KEY" -o BatchMode=yes -o StrictHostKeyChecking=accept-new)
  ssh "${ssh_opts[@]}" "$(ssh_dest)" "$@"
}

scp_base() {
  local ssh_opts=(-i "$DEPLOY_KEY" -o BatchMode=yes -o StrictHostKeyChecking=accept-new)
  scp "${ssh_opts[@]}" "$@"
}

rsync_base() {
  rsync -az --delete -e "ssh $(ssh_opts)" "$@"
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
  local app_dir
  app_dir="$(shell_quote "$REMOTE_APP_DIR")"
  ssh_base "cd $app_dir 2>/dev/null && { cat .deploy-ref 2>/dev/null || git rev-parse HEAD 2>/dev/null || true; }" | redact_sensitive
}

paths_ignored() {
  local path
  for path in "$@"; do
    [[ -z "$path" ]] && continue
    if ! git check-ignore --no-index -q -- "$path"; then
      return 1
    fi
  done
}

classify_worktree_changes() {
  local ignored=()
  local blocking=()
  local entry
  local status
  local path
  local old_path
  local display_path

  while IFS= read -r -d '' entry; do
    [[ -z "$entry" ]] && continue
    status="${entry:0:2}"
    path="${entry:3}"

    if [[ "$status" == R* || "$status" == C* ]]; then
      old_path=""
      IFS= read -r -d '' old_path || true
      display_path="$old_path -> $path"
      if [[ -n "$old_path" ]] && paths_ignored "$path" "$old_path"; then
        ignored+=("$display_path")
      else
        blocking+=("$display_path")
      fi
      continue
    fi

    if paths_ignored "$path"; then
      ignored+=("$path")
    else
      blocking+=("$path")
    fi
  done < <(git status --porcelain=v1 -z)

  if [[ "${#blocking[@]}" -gt 0 ]]; then
    echo -e "${RED}错误: 当前工作区存在未提交且未被 .gitignore 覆盖的改动，拒绝部署${NC}" >&2
    printf -- '- %s\n' "${blocking[@]}" >&2
    exit 1
  fi

  if [[ "${#ignored[@]}" -gt 0 ]]; then
    echo -e "${YELLOW}检测到仅含 .gitignore 覆盖的本地改动，部署将忽略这些文件：${NC}"
    printf -- '- %s\n' "${ignored[@]}"
  fi
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
  classify_worktree_changes
}

remote_precheck() {
  assert_key
  local app_dir env_file h5_dir admin_dir nginx_conf
  app_dir="$(shell_quote "$REMOTE_APP_DIR")"
  env_file="$(shell_quote "$REMOTE_ENV_FILE")"
  h5_dir="$(shell_quote "$REMOTE_H5_DIR")"
  admin_dir="$(shell_quote "$REMOTE_ADMIN_DIR")"
  nginx_conf="$(shell_quote "$REMOTE_NGINX_CONF")"
  ssh_base "test -d $app_dir && test -f $env_file && test -d $h5_dir && test -d $admin_dir && test -f $nginx_conf && docker compose version >/dev/null && nginx -v >/dev/null && openssl version >/dev/null"
  verify_remote_compose_uniqueness
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
  assert_clean_for_ref
  remote_precheck
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
  echo "预检状态:   Git/SSH/远端目录/Docker/Nginx 已通过"
  echo ""
  echo "预计动作:"
  echo "- 在 frontend/h5 执行 npm ci && npm run build"
  echo "- 在 frontend/admin 执行 npm ci && npx vite build --base=/admin/"
  echo "- 备份远端静态资源到 $REMOTE_BACKUP_DIR"
  echo "- 同步 H5 到 $REMOTE_H5_DIR"
  echo "- 同步 Admin 到 $REMOTE_ADMIN_DIR"
  echo "- 同步仓库源码到 $REMOTE_APP_DIR"
  echo "- 同步 Nginx 配置到 $REMOTE_NGINX_CONF"
  echo "- 创建或更新演示入口 Basic Auth 文件 $REMOTE_BASIC_AUTH_FILE"
  echo "- 执行 docker compose --project-name $COMPOSE_PROJECT_NAME --env-file $REMOTE_ENV_FILE -f docker-compose.demo.yml up -d --build --remove-orphans，覆盖全量服务:"
  echo "  $(demo_full_stack_services)"
  echo "- 执行 nginx -t && systemctl reload nginx"
  echo ""
  echo "验证:"
  echo "- curl -I http://$DEPLOY_HOST/"
  echo "- curl -I http://$DEPLOY_HOST/admin/"
  echo "- curl http://$DEPLOY_HOST/api/v1/products"
  echo "- docker compose --project-name $COMPOSE_PROJECT_NAME ps $(demo_full_stack_services)"
  echo ""
  echo "回滚:"
  echo "- 前端静态资源使用 $REMOTE_BACKUP_DIR 最近备份恢复"
  echo "- 后端进入 $REMOTE_APP_DIR checkout 上一提交后重建容器"
}

build_frontend() {
  cd "$PROJECT_ROOT/frontend/h5"
  npm ci
  npm run build
  cd "$PROJECT_ROOT/frontend/admin"
  npm ci
  npx vite build --base=/admin/
}

backup_remote() {
  local stamp
  local backup_dir h5_dir admin_dir
  stamp="$(date +%Y%m%d%H%M%S)"
  backup_dir="$(shell_quote "$REMOTE_BACKUP_DIR/$stamp")"
  h5_dir="$(shell_quote "$REMOTE_H5_DIR")"
  admin_dir="$(shell_quote "$REMOTE_ADMIN_DIR")"
  ssh_base "mkdir -p $backup_dir && cp -a $h5_dir $backup_dir/auction-h5 && cp -a $admin_dir $backup_dir/auction-admin && echo $(shell_quote "$stamp") > $(shell_quote "$REMOTE_BACKUP_DIR/latest")"
}

sync_frontend() {
  rsync_base "$PROJECT_ROOT/frontend/h5/dist/" "$DEPLOY_USER@$DEPLOY_HOST:$REMOTE_H5_DIR/"
  rsync_base "$PROJECT_ROOT/frontend/admin/dist/" "$DEPLOY_USER@$DEPLOY_HOST:$REMOTE_ADMIN_DIR/"
}

sync_backend() {
  rsync_base \
    --filter=':- .gitignore' \
    --exclude '.git/' \
    --exclude 'node_modules/' \
    --exclude 'frontend/h5/dist/' \
    --exclude 'frontend/admin/dist/' \
    --exclude '.tmp/' \
    "$PROJECT_ROOT/" "$DEPLOY_USER@$DEPLOY_HOST:$REMOTE_APP_DIR/"
  local app_dir target
  app_dir="$(shell_quote "$REMOTE_APP_DIR")"
  target="$(local_sha)"
  ssh_base "cd $app_dir && echo $(shell_quote "$target") > .deploy-ref"
}

sync_nginx_config() {
  scp_base "$PROJECT_ROOT/deploy/demo/nginx-ip.conf" "$(ssh_dest):$REMOTE_NGINX_CONF"
}

ensure_basic_auth_file() {
  local user password auth_file
  user="$(shell_quote "$DEMO_BASIC_AUTH_USER")"
  password="$(shell_quote "$DEMO_BASIC_AUTH_PASSWORD")"
  auth_file="$(shell_quote "$REMOTE_BASIC_AUTH_FILE")"
  ssh_base "hash=\$(openssl passwd -apr1 $password) && printf '%s:%s\n' $user \"\$hash\" > $auth_file && chown root:www-data $auth_file && chmod 640 $auth_file"
}

restart_remote() {
  local app_dir env_file compose
  app_dir="$(shell_quote "$REMOTE_APP_DIR")"
  env_file="$(shell_quote "$REMOTE_ENV_FILE")"
  compose="$(compose_cmd "$env_file")"
  ssh_base "cd $app_dir && $compose up -d --build --remove-orphans && nginx -t && systemctl reload nginx"
}

http_expect() {
  local url=$1
  local expected_regex=$2
  local code
  code="$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "$url" || true)"
  echo "$url -> $code"
  [[ "$code" =~ $expected_regex ]]
}

http_basic_expect() {
  local url=$1
  local expected_regex=$2
  local code
  code="$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 -u "$DEMO_BASIC_AUTH_USER:$DEMO_BASIC_AUTH_PASSWORD" "$url" || true)"
  echo "$url with Basic Auth -> $code"
  [[ "$code" =~ $expected_regex ]]
}

http_body_contains() {
  local url=$1
  local expected=$2
  local body
  body="$(curl -s --max-time 10 "$url" || true)"
  if [[ "$body" == *"$expected"* ]]; then
    echo "$url contains $expected"
    return 0
  fi
  echo "$url missing $expected"
  return 1
}

wait_for_remote_http_ready() {
  echo -e "${BLUE}等待线上服务 HTTP 就绪...${NC}"

  local url
  local code
  for url in \
    "http://$DEPLOY_HOST/" \
    "http://$DEPLOY_HOST/admin/" \
    "http://$DEPLOY_HOST/api/v1/products"; do
    local ready=0
    for _ in $(seq 1 60); do
      code="$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "$url" || true)"
      if [[ "$code" != "000" ]]; then
        ready=1
        break
      fi
      sleep 2
    done
    if [[ "$ready" -eq 1 ]]; then
      echo -e "${GREEN}✓ $url 已就绪 ($code)${NC}"
    else
      echo -e "${YELLOW}警告: $url 在预期时间内未就绪，继续验证阶段${NC}"
    fi
  done
}

print_remote_logs_hint() {
  local app_dir env_file compose
  app_dir="$(shell_quote "$REMOTE_APP_DIR")"
  env_file="$(shell_quote "$REMOTE_ENV_FILE")"
  compose="$(compose_cmd "$env_file")"
  echo "远端日志排查命令:"
  echo "ssh -i $(shell_quote "$DEPLOY_KEY") $(ssh_dest) 'cd $app_dir && $compose logs --tail=100 $(demo_full_stack_services)'"
}

verify_remote_compose_uniqueness() {
  local project
  project="$(shell_quote "$COMPOSE_PROJECT_NAME")"
  ssh_base "target_project=$project; failed=0; for service in $(demo_full_stack_services); do docker ps -a --filter \"label=com.docker.compose.service=\$service\" --format '{{.ID}}|{{.Names}}|{{.Label \"com.docker.compose.project\"}}|{{.Label \"com.docker.compose.service\"}}' | awk -F'|' -v target=\"\$target_project\" 'length(\$1) > 0 && \$3 != target { printf \"duplicate compose service: service=%s container=%s project=%s expected=%s\\n\", \$4, \$2, \$3, target > \"/dev/stderr\"; bad=1 } END { exit bad }' || failed=1; done; exit \$failed" | redact_sensitive
}

verify_remote_containers() {
  local app_dir env_file compose output failed=0
  app_dir="$(shell_quote "$REMOTE_APP_DIR")"
  env_file="$(shell_quote "$REMOTE_ENV_FILE")"
  compose="$(compose_cmd "$env_file")"
  verify_remote_compose_uniqueness
  output="$(ssh_base "cd $app_dir && $compose ps --format json $(demo_full_stack_services)" | redact_sensitive)"
  echo "$output"

  for service in $(demo_full_stack_services); do
    if ! echo "$output" | grep -E "\"Service\":\"$service\"|\"Name\":\"[^\"]*$service[^\"]*\"" >/dev/null; then
      echo -e "${RED}错误: 远端容器缺失: $service${NC}" >&2
      failed=1
      continue
    fi
    if ! echo "$output" | grep -E "\"Service\":\"$service\".*\"State\":\"running\"|\"Name\":\"[^\"]*$service[^\"]*\".*\"State\":\"running\"" >/dev/null; then
      echo -e "${RED}错误: 远端容器未 running: $service${NC}" >&2
      failed=1
    fi
  done

  if [[ "$failed" -ne 0 ]]; then
    print_remote_logs_hint >&2
    exit 1
  fi
}

verify_prod() {
  local failed=0

  http_expect "http://$DEPLOY_HOST/" '^(2|3)[0-9][0-9]$' || failed=1
  http_expect "http://$DEPLOY_HOST/admin/" '^(2|3)[0-9][0-9]$' || failed=1
  http_expect "http://$DEPLOY_HOST/test-dashboard/" '^401$' || failed=1
  http_expect "http://$DEPLOY_HOST/grafana/" '^401$' || failed=1
  http_basic_expect "http://$DEPLOY_HOST/test-dashboard/" '^(2|3)[0-9][0-9]$' || failed=1
  http_basic_expect "http://$DEPLOY_HOST/grafana/" '^(2|3)[0-9][0-9]$' || failed=1
  http_expect "http://$DEPLOY_HOST/api/v1/products" '^200$' || failed=1
  http_body_contains "http://$DEPLOY_HOST/ws/test/progress?test_id=verify" 'ws_url' || failed=1

  if [[ "$failed" -ne 0 ]]; then
    print_remote_logs_hint >&2
    echo -e "${RED}线上验证失败${NC}" >&2
    exit 1
  fi

  verify_remote_containers
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
  sync_nginx_config
  ensure_basic_auth_file
  restart_remote
  wait_for_remote_http_ready
  verify_prod
}

rollback() {
  assert_key
  local backup_latest backup_dir h5_dir admin_dir nginx_conf
  backup_latest="$(shell_quote "$REMOTE_BACKUP_DIR/latest")"
  ssh_base "test -f $backup_latest"
  local latest
  latest="$(ssh_base "cat $backup_latest")"
  backup_dir="$(shell_quote "$REMOTE_BACKUP_DIR/$latest")"
  h5_dir="$(shell_quote "$REMOTE_H5_DIR")"
  admin_dir="$(shell_quote "$REMOTE_ADMIN_DIR")"
  ssh_base "rsync -a --delete $backup_dir/auction-h5/ $h5_dir/ && rsync -a --delete $backup_dir/auction-admin/ $admin_dir/ && nginx -t && systemctl reload nginx"
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
    require_cmd ssh
    assert_key
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

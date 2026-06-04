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
  cat <<'USAGE'
用法: scripts/deploy-dev.sh <status|verify|restart|stop>

说明:
  status   只读检查本地端口、Git 和 Docker 基础设施状态
  verify   验证本地前后端端口和核心 API 是否可访问
  restart  强制重启本地基础设施、后端和前端
  stop     停止本项目本地前后端服务
USAGE
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

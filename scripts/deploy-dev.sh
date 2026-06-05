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

assert_clean_tree() {
  cd "$PROJECT_ROOT"
  classify_worktree_changes
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

compose_project_name() {
  basename "$PROJECT_ROOT"
}

local_compose_services() {
  cd "$PROJECT_ROOT"
  INTERNAL_API_TOKEN=dev docker compose config --services 2>/dev/null
}

remove_foreign_project_containers() {
  local current_project
  local service
  local line
  local id
  local name
  local project
  local workdir

  current_project="$(compose_project_name)"
  while IFS= read -r service; do
    [[ -z "$service" ]] && continue
    while IFS='|' read -r id name project workdir; do
      [[ -z "$id" ]] && continue
      if [[ "$project" != "$current_project" && -n "$workdir" && "$workdir" != "$PROJECT_ROOT"* ]]; then
        echo -e "${YELLOW}删除其他 worktree 残留的 $service 容器: $name (compose project: ${project:-unknown}, working_dir: $workdir)${NC}"
        docker rm -f "$id" >/dev/null
      fi
    done < <(docker ps -a \
      --filter "label=com.docker.compose.service=$service" \
      --format '{{.ID}}|{{.Names}}|{{.Label "com.docker.compose.project"}}|{{.Label "com.docker.compose.project.working_dir"}}')
  done < <(local_compose_services)
}

port_has_non_docker_listener() {
  local port=$1
  local pid
  local comm

  while IFS= read -r pid; do
    [[ -z "$pid" ]] && continue
    comm="$(ps -p "$pid" -o comm= 2>/dev/null || true)"
    if [[ "$comm" != *com.docker* && "$comm" != *Docker* ]]; then
      return 0
    fi
  done < <(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)

  return 1
}

stop_non_docker_infra_ports() {
  local port
  local pid
  local comm
  local formula

  for port in 3306 6379 5672; do
    formula=""
    case "$port" in
      3306) formula="mysql" ;;
      6379) formula="redis" ;;
      5672) formula="rabbitmq" ;;
    esac

    if [[ -n "$formula" ]] && command -v brew >/dev/null 2>&1 && port_has_non_docker_listener "$port"; then
      brew services stop "$formula" >/dev/null 2>&1 || true
    fi

    while IFS= read -r pid; do
      [[ -z "$pid" ]] && continue
      comm="$(ps -p "$pid" -o comm= 2>/dev/null || true)"
      if [[ "$comm" == *com.docker* || "$comm" == *Docker* ]]; then
        continue
      fi

      echo -e "${YELLOW}停止端口 $port 上的非 Docker 基础设施进程: $pid ($comm)${NC}"
      kill "$pid" 2>/dev/null || true
      sleep 1
      if kill -0 "$pid" >/dev/null 2>&1; then
        kill -9 "$pid" 2>/dev/null || true
      fi
    done < <(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)
  done
}

wait_for_infra_ready() {
  cd "$PROJECT_ROOT"
  echo -e "${BLUE}等待本地基础设施就绪...${NC}"

  for _ in $(seq 1 60); do
    if INTERNAL_API_TOKEN=dev docker compose exec -T mysql mysqladmin ping -h 127.0.0.1 -uroot -proot --silent >/dev/null 2>&1; then
      echo -e "${GREEN}✓ MySQL 已就绪${NC}"
      break
    fi
    sleep 1
  done
  if ! INTERNAL_API_TOKEN=dev docker compose exec -T mysql mysqladmin ping -h 127.0.0.1 -uroot -proot --silent >/dev/null 2>&1; then
    echo -e "${RED}错误: MySQL 未能在预期时间内就绪${NC}" >&2
    INTERNAL_API_TOKEN=dev docker compose logs --tail=80 mysql >&2 || true
    exit 1
  fi

  for _ in $(seq 1 60); do
    if INTERNAL_API_TOKEN=dev docker compose exec -T rabbitmq rabbitmq-diagnostics -q ping >/dev/null 2>&1; then
      echo -e "${GREEN}✓ RabbitMQ 已就绪${NC}"
      break
    fi
    sleep 1
  done
  if ! INTERNAL_API_TOKEN=dev docker compose exec -T rabbitmq rabbitmq-diagnostics -q ping >/dev/null 2>&1; then
    echo -e "${RED}错误: RabbitMQ 未能在预期时间内就绪${NC}" >&2
    INTERNAL_API_TOKEN=dev docker compose logs --tail=80 rabbitmq >&2 || true
    exit 1
  fi
}

start_infra() {
  cd "$PROJECT_ROOT"
  remove_foreign_project_containers
  stop_non_docker_infra_ports
  INTERNAL_API_TOKEN=dev docker compose up -d mysql redis rabbitmq
  wait_for_infra_ready
}

init_local_auth_users() {
  cd "$PROJECT_ROOT"
  ./scripts/init-local-auth-users.sh
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
  init_local_auth_users
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

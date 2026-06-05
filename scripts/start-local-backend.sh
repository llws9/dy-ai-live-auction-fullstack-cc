#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
RUN_DIR="$PROJECT_ROOT/.tmp/local-backend"

mkdir -p "$RUN_DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

require_cmd() {
  local cmd=$1
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo -e "${RED}错误: 缺少命令 $cmd${NC}"
    exit 1
  fi
}

port_owner() {
  local port=$1
  lsof -nP -iTCP:"$port" -sTCP:LISTEN 2>/dev/null || true
}

ensure_port_free() {
  local port=$1
  local service=$2
  local owner
  owner="$(port_owner "$port")"
  if [[ -n "$owner" ]]; then
    echo -e "${RED}错误: 端口 $port 已被占用，拒绝启动 $service${NC}"
    echo "$owner"
    exit 1
  fi
}

ensure_dependency_port() {
  local port=$1
  local dependency=$2
  if [[ -z "$(port_owner "$port")" ]]; then
    echo -e "${RED}错误: 依赖端口 $port 未就绪 ($dependency)${NC}"
    exit 1
  fi
}

pid_file() {
  echo "$RUN_DIR/$1.pid"
}

log_file() {
  echo "$RUN_DIR/$1.log"
}

launchctl_label() {
  local service=$1
  echo "com.dyauction.local.$service"
}

plist_file() {
  echo "$HOME/Library/LaunchAgents/$(launchctl_label "$1").plist"
}

xml_escape() {
  sed \
    -e 's/&/\&amp;/g' \
    -e 's/</\&lt;/g' \
    -e 's/>/\&gt;/g' \
    -e 's/"/\&quot;/g' \
    -e "s/'/\&apos;/g"
}

launchctl_pid() {
  local label=$1
  launchctl print "gui/$UID/$label" 2>/dev/null | awk -F'= ' '/pid =/ {print $2; exit}'
}

launchctl_running() {
  local label=$1
  [[ -n "$(launchctl_pid "$label")" ]]
}

handle_running() {
  local handle=$1
  if [[ "$handle" == launchctl:* ]]; then
    launchctl_running "${handle#launchctl:}"
    return
  fi
  kill -0 "$handle" >/dev/null 2>&1
}

write_launchctl_plist() {
  local service=$1
  local cwd=$2
  local command=$3
  local log=$4
  local plist=$5
  local label

  label="$(launchctl_label "$service")"
  cat >"$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>$(printf '%s' "$label" | xml_escape)</string>
  <key>WorkingDirectory</key>
  <string>$(printf '%s' "$cwd" | xml_escape)</string>
  <key>ProgramArguments</key>
  <array>
    <string>/bin/bash</string>
    <string>-lc</string>
    <string>$(printf '%s' "$command" | xml_escape)</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>StandardOutPath</key>
  <string>$(printf '%s' "$log" | xml_escape)</string>
  <key>StandardErrorPath</key>
  <string>$(printf '%s' "$log" | xml_escape)</string>
</dict>
</plist>
EOF
}

launchctl_bootout_service() {
  local service=$1
  local label
  local plist

  label="$(launchctl_label "$service")"
  plist="$(plist_file "$service")"
  launchctl unload "$plist" >/dev/null 2>&1 || \
    launchctl bootout "gui/$UID/$label" >/dev/null 2>&1 || true
}

cleanup_stale_pid() {
  local service=$1
  local file
  file="$(pid_file "$service")"
  if [[ -f "$file" ]]; then
    local handle
    handle="$(cat "$file")"
    if ! handle_running "$handle"; then
      rm -f "$file"
    fi
  fi
}

ensure_service_not_running() {
  local service=$1
  cleanup_stale_pid "$service"
  local file
  file="$(pid_file "$service")"
  if [[ -f "$file" ]]; then
    local handle
    handle="$(cat "$file")"
    if handle_running "$handle"; then
      echo -e "${RED}错误: $service 已在运行 ($handle)${NC}"
      exit 1
    fi
  fi
}

wait_for_port() {
  local port=$1
  local service=$2
  local handle=$3
  local log=$4

  for _ in $(seq 1 30); do
    if [[ -n "$(port_owner "$port")" ]]; then
      return 0
    fi
    if ! handle_running "$handle"; then
      echo -e "${RED}错误: $service 启动失败${NC}"
      tail -n 40 "$log" 2>/dev/null || true
      exit 1
    fi
    sleep 1
  done

  echo -e "${RED}错误: $service 未能在预期时间内监听端口 $port${NC}"
  tail -n 40 "$log" 2>/dev/null || true
  exit 1
}

start_service() {
  local service=$1
  local port=$2
  local cwd=$3
  local command=$4
  local log
  local pidf
  local plist
  local label
  local handle

  ensure_service_not_running "$service"
  log="$(log_file "$service")"
  pidf="$(pid_file "$service")"
  plist="$(plist_file "$service")"
  label="$(launchctl_label "$service")"

  echo -e "${BLUE}启动 $service ...${NC}"
  : >"$log"
  mkdir -p "$HOME/Library/LaunchAgents"
  launchctl_bootout_service "$service"
  write_launchctl_plist "$service" "$cwd" "$command" "$log" "$plist"
  launchctl load "$plist"
  handle="launchctl:$label"
  echo "$handle" >"$pidf"

  wait_for_port "$port" "$service" "$handle" "$log"
  local listener_pid
  listener_pid="$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null | head -n 1)"
  if [[ -n "$listener_pid" ]]; then
    echo "$listener_pid" >"$pidf"
  fi
  echo -e "${GREEN}✓ $service 已启动 (PID ${listener_pid:-$handle}, 端口 $port)${NC}"
}

stop_service() {
  local service=$1
  local file
  file="$(pid_file "$service")"
  cleanup_stale_pid "$service"
  if [[ ! -f "$file" ]]; then
    launchctl_bootout_service "$service"
    return
  fi

  local handle
  handle="$(cat "$file")"
  if [[ "$handle" == launchctl:* ]]; then
    local label
    label="${handle#launchctl:}"
    launchctl bootout "gui/$UID/$label" >/dev/null 2>&1 || true
    for _ in $(seq 1 10); do
      if ! launchctl_running "$label"; then
        break
      fi
      sleep 1
    done
    echo -e "${GREEN}✓ 已停止 $service ($handle)${NC}"
  elif kill -0 "$handle" >/dev/null 2>&1; then
    kill "$handle" >/dev/null 2>&1 || true
    for _ in $(seq 1 10); do
      if ! kill -0 "$handle" >/dev/null 2>&1; then
        break
      fi
      sleep 1
    done
    if kill -0 "$handle" >/dev/null 2>&1; then
      kill -9 "$handle" >/dev/null 2>&1 || true
    fi
    echo -e "${GREEN}✓ 已停止 $service (PID $handle)${NC}"
  fi
  launchctl_bootout_service "$service"
  rm -f "$file"
}

stop_backend_ports() {
  for port in 8080 8081 8082 8083; do
    local pids
    pids="$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
    if [[ -n "$pids" ]]; then
      echo -e "${YELLOW}停止端口 $port 上的残留后端进程: $pids${NC}"
      kill $pids 2>/dev/null || true
      sleep 1
      pids="$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
      if [[ -n "$pids" ]]; then
        kill -9 $pids 2>/dev/null || true
      fi
    fi
  done
}

show_status() {
  for service in auction product gateway; do
    cleanup_stale_pid "$service"
    local file
    file="$(pid_file "$service")"
    if [[ -f "$file" ]]; then
      local handle
      handle="$(cat "$file")"
      echo "$service: running ($handle), log=$(log_file "$service")"
    else
      echo "$service: stopped"
    fi
  done
}

start_all() {
  require_cmd go
  require_cmd lsof
  require_cmd bash

  ensure_dependency_port 3306 "MySQL"
  ensure_dependency_port 6379 "Redis"
  ensure_dependency_port 5672 "RabbitMQ"

  ensure_port_free 8080 "gateway"
  ensure_port_free 8081 "product-service"
  ensure_port_free 8082 "auction-service HTTP"
  ensure_port_free 8083 "auction-service WS"

  local auction_cmd="JWT_SECRET=dev-secret INTERNAL_API_TOKEN=dev REDIS_ADDR=127.0.0.1:6379 DB_HOST=127.0.0.1 DB_PORT=3306 DB_USER=root DB_PASSWORD=root DB_NAME=auction HTTP_PORT=:8082 WS_PORT=:8083 RABBITMQ_HOST=127.0.0.1 RABBITMQ_PORT=5672 RABBITMQ_USER=guest RABBITMQ_PASSWORD=guest RABBITMQ_VHOST=/ go run main.go"
  local product_cmd="NACOS_SERVER_ADDR=127.0.0.1:1 INTERNAL_API_TOKEN=dev DB_HOST=127.0.0.1 DB_PORT=3306 DB_USER=root DB_PASSWORD=root DB_NAME=auction REDIS_ADDR=127.0.0.1:6379 AUCTION_SERVICE_URL=http://127.0.0.1:8082 PRODUCT_SERVICE_PORT=:8081 go run main.go"
  local gateway_cmd="JWT_SECRET=dev-secret INTERNAL_API_TOKEN=dev NACOS_SERVER_ADDR=127.0.0.1:1 GATEWAY_PORT=:8080 PRODUCT_SERVICE_URL=http://127.0.0.1:8081 AUCTION_SERVICE_URL=http://127.0.0.1:8082 TEST_SERVICE_URL=http://127.0.0.1:18090 TEST_SERVICE_WS_URL=ws://127.0.0.1:18092 REDIS_ADDR=127.0.0.1:6379 GROWTHBOOK_ENABLED=false go run main.go"

  start_service "auction" 8082 "$PROJECT_ROOT/backend/auction" "$auction_cmd"
  wait_for_port 8083 "auction-service WS" "$(cat "$(pid_file "auction")")" "$(log_file "auction")"
  start_service "product" 8081 "$PROJECT_ROOT/backend/product" "$product_cmd"
  start_service "gateway" 8080 "$PROJECT_ROOT/backend/gateway" "$gateway_cmd"

  echo ""
  echo -e "${GREEN}本地后端启动完成${NC}"
  echo "Gateway: http://localhost:8080"
  echo "Product: http://localhost:8081"
  echo "Auction HTTP: http://localhost:8082"
  echo "Auction WS: ws://localhost:8083"
  echo "日志目录: $RUN_DIR"
}

stop_all() {
  stop_service "gateway"
  stop_service "product"
  stop_service "auction"
  stop_backend_ports
}

show_help() {
  cat <<EOF
用法: $0 <start|stop|restart|status>

说明:
  - 启动前会严格检查 8080/8081/8082/8083 是否已被占用
  - 任一端口冲突都会直接报错退出，不会自动杀进程
  - 本地启动强制使用环境变量，不依赖 Nacos 在线配置
EOF
}

case "${1:-start}" in
  start)
    start_all
    ;;
  stop)
    stop_all
    ;;
  restart)
    stop_all
    start_all
    ;;
  status)
    show_status
    ;;
  help|--help|-h)
    show_help
    ;;
  *)
    show_help
    exit 1
    ;;
esac

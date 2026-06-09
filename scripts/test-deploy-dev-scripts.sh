#!/bin/bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

assert_contains() {
  local file=$1
  local pattern=$2
  local message=$3

  if ! rg -q "$pattern" "$file"; then
    fail "$message"
  fi
}

assert_not_contains() {
  local file=$1
  local pattern=$2
  local message=$3

  if rg -q "$pattern" "$file"; then
    fail "$message"
  fi
}

assert_compose_dependency_condition() {
  local service=$1
  local dependency=$2
  local condition=$3
  local message=$4

  python3 - "$ROOT/docker-compose.yml" "$service" "$dependency" "$condition" "$message" <<'PY'
import re
import sys
from pathlib import Path

path, service, dependency, condition, message = sys.argv[1:]
text = Path(path).read_text()
service_match = re.search(rf"^  {re.escape(service)}:\n(?P<body>.*?)(?=^  [A-Za-z0-9_-]+:|\Z)", text, re.M | re.S)
if not service_match:
    print(f"FAIL: service {service} not found", file=sys.stderr)
    sys.exit(1)
body = service_match.group("body")
dep_match = re.search(rf"^      {re.escape(dependency)}:\n(?P<body>.*?)(?=^      [A-Za-z0-9_-]+:|^    [A-Za-z0-9_-]+:|\Z)", body, re.M | re.S)
if not dep_match or f"condition: {condition}" not in dep_match.group("body"):
    print(f"FAIL: {message}", file=sys.stderr)
    sys.exit(1)
PY
}

assert_contains \
  "$ROOT/scripts/start-local-backend.sh" \
  'launchctl load "\$plist"' \
  "start-local-backend.sh must use launchctl to keep backend processes alive after the agent command exits"

assert_contains \
  "$ROOT/scripts/start-local-backend.sh" \
  'launchctl unload "\$plist"' \
  "start-local-backend.sh must unload launchctl jobs when stopping backend services"

assert_contains \
  "$ROOT/scripts/start-local-backend.sh" \
  'Library/LaunchAgents' \
  "start-local-backend.sh must place launchctl plists in the macOS user LaunchAgents directory"

assert_contains \
  "$ROOT/scripts/start-local-backend.sh" \
  'com.dyauction.local.\$service' \
  "start-local-backend.sh launchctl labels must be globally unique across worktrees"

assert_not_contains \
  "$ROOT/scripts/start-local-backend.sh" \
  'cksum' \
  "start-local-backend.sh must not include worktree-specific hashes in service supervisor labels"

assert_not_contains \
  "$ROOT/scripts/start-local-backend.sh" \
  'screen -dmS' \
  "start-local-backend.sh must not rely on screen for persistent backend supervision"

assert_contains \
  "$ROOT/scripts/start-local-backend.sh" \
  'stop_backend_ports' \
  "start-local-backend.sh stop must clean orphan backend listener processes by port"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  '8080 8081 8082 8083 18090 18091 18092' \
  "deploy-dev.sh must clean backend and test-service host ports before Docker full-stack restart"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  '3000 3001 5173 5175' \
  "deploy-dev.sh must clean both Docker frontend ports and Vite frontend ports before restart"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'listener_pids_excluding_docker' \
  "deploy-dev.sh port cleanup must exclude Docker Desktop port proxy PIDs instead of killing the Docker engine"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'Skipping Docker-owned listener' \
  "deploy-dev.sh must explicitly skip Docker-owned port listeners when clearing published compose ports"

assert_contains \
  "$ROOT/scripts/start-local-backend.sh" \
  'listener_pids_excluding_docker' \
  "start-local-backend.sh stop must exclude Docker Desktop port proxy PIDs instead of killing the Docker engine"

assert_contains \
  "$ROOT/scripts/start-local-backend.sh" \
  'Skipping Docker-owned backend listener' \
  "start-local-backend.sh must explicitly skip Docker-owned backend port listeners"

assert_not_contains \
  "$ROOT/scripts/start-local-backend.sh" \
  'nohup bash -lc' \
  "start-local-backend.sh must not rely on nohup because Trae sandbox reaps child processes"

assert_contains \
  "$ROOT/scripts/start-local-backend.sh" \
  'StandardOutPath' \
  "start-local-backend.sh launchctl plist must redirect stdout to service logs"

assert_contains \
  "$ROOT/scripts/start-local-backend.sh" \
  'listener_pid="\$\(lsof -tiTCP:"\$port" -sTCP:LISTEN' \
  "start-local-backend.sh must persist the actual listener PID after startup"

assert_contains \
  "$ROOT/scripts/start-local-backend.sh" \
  'StandardErrorPath' \
  "start-local-backend.sh launchctl plist must redirect stderr to service logs"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'wait_for_infra_ready' \
  "deploy-dev.sh must wait for infrastructure readiness after docker compose up"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'mysqladmin ping' \
  "deploy-dev.sh must check MySQL readiness, not only port listening"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'rabbitmq-diagnostics -q ping' \
  "deploy-dev.sh must check RabbitMQ readiness, not only container running"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'remove_foreign_project_containers' \
  "deploy-dev.sh must clean same-project containers left by another worktree before restart"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'docker compose config --services' \
  "deploy-dev.sh must derive the local service set from docker compose instead of hardcoding only infra"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'local_full_stack_services' \
  "deploy-dev.sh must define the full local docker compose service set for deployment and verification"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'docker compose up -d --build --remove-orphans' \
  "deploy-dev.sh restart must deploy all local docker compose services, not only core backend services"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'verify_local_containers' \
  "deploy-dev.sh verify must check every local docker compose service container"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'gateway product auction mysql redis rabbitmq nacos nacos-mysql frontend-h5 frontend-admin test-service test-dashboard loki promtail prometheus grafana growthbook growthbook-db' \
  "deploy-dev.sh must include all mature-project local services in the full deployment set"

assert_contains \
  "$ROOT/docker-compose.yml" \
  'context: \./backend' \
  "docker-compose.yml product build must use backend as context so shared/llm is available"

assert_contains \
  "$ROOT/docker-compose.yml" \
  'dockerfile: product/Dockerfile' \
  "docker-compose.yml product build must point to product/Dockerfile from backend context"

assert_contains \
  "$ROOT/docker-compose.yml" \
  'PRODUCT_SERVICE_URL=http://product:8081' \
  "docker-compose.yml gateway and auction must use the Docker network product hostname, not localhost"

assert_contains \
  "$ROOT/docker-compose.yml" \
  'AUCTION_SERVICE_URL=http://auction:8082' \
  "docker-compose.yml gateway and product must use the Docker network auction hostname, not localhost"

assert_contains \
  "$ROOT/docker-compose.yml" \
  'DB_HOST=mysql' \
  "docker-compose.yml application services must use the Docker network mysql hostname, not localhost"

assert_contains \
  "$ROOT/docker-compose.yml" \
  'REDIS_ADDR=redis:6379' \
  "docker-compose.yml application services must use the Docker network redis hostname, not localhost"

assert_contains \
  "$ROOT/docker-compose.yml" \
  'RABBITMQ_HOST=rabbitmq' \
  "docker-compose.yml auction service must use the Docker network rabbitmq hostname, not localhost"

assert_contains \
  "$ROOT/docker-compose.yml" \
  'JWT_SECRET=dev-secret' \
  "docker-compose.yml gateway, auction, and test-service must share the same local JWT secret"

assert_contains \
  "$ROOT/docker-compose.yml" \
  'INTERNAL_API_TOKEN=\$\{INTERNAL_API_TOKEN:\?set INTERNAL_API_TOKEN\}' \
  "docker-compose.yml services that perform internal calls must receive INTERNAL_API_TOKEN at runtime"

assert_contains \
  "$ROOT/docker-compose.yml" \
  'MYSQL_DATABASE_NUM=1' \
  "docker-compose.yml Nacos service must declare the MySQL database count for external datasource initialization"

assert_contains \
  "$ROOT/docker-compose.yml" \
  'MYSQL_SERVICE_DB_PARAM=.*allowPublicKeyRetrieval=true' \
  "docker-compose.yml Nacos service must pass MySQL connection parameters compatible with MySQL 8"

assert_compose_dependency_condition \
  product \
  mysql \
  service_healthy \
  "docker-compose.yml product must wait for MySQL health before starting"

assert_compose_dependency_condition \
  auction \
  mysql \
  service_healthy \
  "docker-compose.yml auction must wait for MySQL health before starting"

assert_compose_dependency_condition \
  auction \
  rabbitmq \
  service_healthy \
  "docker-compose.yml auction must wait for RabbitMQ health before starting"

assert_compose_dependency_condition \
  test-service \
  mysql \
  service_healthy \
  "docker-compose.yml test-service must wait for MySQL health before enabling /api/test routes"

test -f "$ROOT/frontend/h5/Dockerfile" || fail "frontend-h5 compose service must have a Dockerfile"
test -f "$ROOT/frontend/admin/Dockerfile" || fail "frontend-admin compose service must have a Dockerfile"
test -f "$ROOT/frontend/h5/.dockerignore" || fail "frontend-h5 Docker build must ignore local dependency artifacts"
test -f "$ROOT/frontend/admin/.dockerignore" || fail "frontend-admin Docker build must ignore local dependency artifacts"

assert_contains \
  "$ROOT/frontend/h5/Dockerfile" \
  'nginx:1\.27-alpine' \
  "frontend-h5 Dockerfile must serve built assets through nginx"

assert_contains \
  "$ROOT/frontend/admin/Dockerfile" \
  'nginx:1\.27-alpine' \
  "frontend-admin Dockerfile must serve built assets through nginx"

assert_contains \
  "$ROOT/frontend/h5/nginx/default.conf" \
  'resolver 127\.0\.0\.11' \
  "frontend-h5 nginx must use Docker DNS so gateway container recreation does not leave a stale upstream IP"

assert_contains \
  "$ROOT/frontend/h5/nginx/default.conf" \
  'location = /api/v1/ws' \
  "frontend-h5 nginx must proxy auction WebSocket before the generic /api location"

assert_contains \
  "$ROOT/frontend/h5/nginx/default.conf" \
  'proxy_pass http://\$auction_ws_upstream/ws\$is_args\$args' \
  "frontend-h5 nginx must rewrite /api/v1/ws to auction /ws and preserve query params"

assert_contains \
  "$ROOT/frontend/h5/nginx/default.conf" \
  'location \^~ /@vite/' \
  "frontend-h5 nginx must not rewrite stale Vite dev-client module requests to index.html"

assert_contains \
  "$ROOT/frontend/h5/nginx/default.conf" \
  'Cache-Control "no-cache, no-store, must-revalidate"' \
  "frontend-h5 nginx must prevent stale index.html after local rebuilds"

assert_contains \
  "$ROOT/frontend/admin/nginx/default.conf" \
  'resolver 127\.0\.0\.11' \
  "frontend-admin nginx must use Docker DNS so gateway container recreation does not leave a stale upstream IP"

assert_contains \
  "$ROOT/frontend/admin/nginx/default.conf" \
  'location = /api/v1/ws' \
  "frontend-admin nginx must proxy auction WebSocket before the generic /api location"

assert_contains \
  "$ROOT/frontend/h5/.dockerignore" \
  '^node_modules$' \
  "frontend-h5 Docker build context must exclude node_modules"

assert_contains \
  "$ROOT/frontend/admin/.dockerignore" \
  '^node_modules$' \
  "frontend-admin Docker build context must exclude node_modules"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'stop_non_docker_infra_ports' \
  "deploy-dev.sh must stop host-level infra listeners before starting Docker infra"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  '3306 6379 5672' \
  "deploy-dev.sh must enforce uniqueness for MySQL, Redis, and RabbitMQ ports"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'brew services stop' \
  "deploy-dev.sh must stop Homebrew-managed infra services that would otherwise respawn on macOS"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'com.docker.compose.service' \
  "deploy-dev.sh must identify local service containers by Docker compose service labels"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'com.docker.compose.project.working_dir' \
  "deploy-dev.sh must use compose working_dir labels to distinguish other worktrees from current repo sub-compose projects"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  '"\$workdir" != "\$PROJECT_ROOT"\*' \
  "deploy-dev.sh must not delete compose projects launched from the current repository tree"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'wait_for_users_table' \
  "deploy-dev.sh must wait for product service to migrate the users table before seeding demo users"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  "information_schema.TABLES WHERE TABLE_SCHEMA='auction' AND TABLE_NAME='users'" \
  "deploy-dev.sh must detect users table readiness via information_schema instead of guessing"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'wait_for_http_ready' \
  "deploy-dev.sh must wait for slow-start application HTTP services before verification to avoid false 000 failures"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'init_demo_users' \
  "deploy-dev.sh restart must initialize unified demo users after MySQL is ready"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  './scripts/init-demo-users.sh' \
  "deploy-dev.sh must delegate demo user seeding to scripts/init-demo-users.sh"

test -x "$ROOT/scripts/init-demo-users.sh" || fail "init-demo-users.sh must exist and be executable"

legacy_seed_script="$ROOT/scripts/init-local-auth"'-users.sh'
if [[ -e "$legacy_seed_script" ]]; then
  fail "legacy local auth seed script must be renamed to init-demo-users.sh"
fi

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  'ensure_column email' \
  "init-demo-users.sh must repair old users tables that lack email"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  'ensure_column phone' \
  "init-demo-users.sh must repair old users tables that lack phone"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  'ensure_column password' \
  "init-demo-users.sh must repair old users tables that lack password"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  'ON DUPLICATE KEY UPDATE' \
  "init-demo-users.sh must seed demo users idempotently"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  'validate_no_conflicts' \
  "init-demo-users.sh must fail closed before seeding when target ids, phones, or emails conflict"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  'SIGNAL SQLSTATE' \
  "init-demo-users.sh must abort MySQL execution on demo user seed conflicts"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  'assert_seeded_users' \
  "init-demo-users.sh must verify final seeded user bindings after upsert"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  'rebind_legacy_demo_user_ids' \
  "init-demo-users.sh must migrate legacy demo users from old ids to fixed ids"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  'SET FOREIGN_KEY_CHECKS = 0' \
  "init-demo-users.sh legacy id migration must handle immediate foreign-key checks while rebinding parent ids"

assert_not_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  'SET phone = NULL' \
  "init-demo-users.sh must not silently clear phones from non-demo users"

assert_not_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  'legacy\+' \
  "init-demo-users.sh must not silently rewrite emails on non-demo users"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  'mysql -h127\.0\.0\.1 -P3306' \
  "init-demo-users.sh must fall back to the host MySQL used by local backend processes"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  '13800138001' \
  "init-demo-users.sh must seed buyer A phone"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  '13800138004' \
  "init-demo-users.sh must seed buyer B phone"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  '13800138002' \
  "init-demo-users.sh must seed merchant phone"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  '13800138003' \
  "init-demo-users.sh must seed admin phone"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  'Demo@123456' \
  "init-demo-users.sh must document the unified demo password"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  '9101' \
  "init-demo-users.sh must seed buyer A with fixed id 9101"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  '9102' \
  "init-demo-users.sh must seed buyer B with fixed id 9102"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  '9103' \
  "init-demo-users.sh must seed merchant with fixed id 9103"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  '9104' \
  "init-demo-users.sh must seed admin with fixed id 9104"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  '\$2a\$10\$qLMubs2jJ79\.H6tSKQRkruqVRbEH2Af91ljpMEAhSsLf642SC6wki' \
  "init-demo-users.sh must use the fixed bcrypt hash for Demo@123456"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  "\(9101, '演示买家A', '', NULL, '13800138001', '\\\$\{DEMO_PASSWORD_HASH\}', 0, 1, NOW\(\)\)" \
  "init-demo-users.sh must keep buyer A full seed row binding"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  "\(9102, '演示买家B', '', NULL, '13800138004', '\\\$\{DEMO_PASSWORD_HASH\}', 0, 1, NOW\(\)\)" \
  "init-demo-users.sh must keep buyer B full seed row binding"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  "\(9103, '演示商家', '', 'merchant@example.com', '13800138002', '\\\$\{DEMO_PASSWORD_HASH\}', 1, 1, NOW\(\)\)" \
  "init-demo-users.sh must keep merchant full seed row binding"

assert_contains \
  "$ROOT/scripts/init-demo-users.sh" \
  "\(9104, '系统管理员', '', 'admin@example.com', '13800138003', '\\\$\{DEMO_PASSWORD_HASH\}', 2, 1, NOW\(\)\)" \
  "init-demo-users.sh must keep admin full seed row binding"

if grep -q '18600000001\|18600000002\|admin123\|本地测试用户' "$ROOT/scripts/init-demo-users.sh"; then
  fail "init-demo-users.sh must not keep legacy local 186/admin123 account seeds"
fi

echo "deploy dev script checks passed"

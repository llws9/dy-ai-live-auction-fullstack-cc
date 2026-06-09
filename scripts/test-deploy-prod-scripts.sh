#!/bin/bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="$ROOT/scripts/deploy-prod.sh"
DEMO_NGINX="$ROOT/deploy/demo/nginx-ip.conf"
DEPLOY_QUICKSTART="$ROOT/deploy/demo/MAIN_DEPLOY_QUICKSTART.md"
DEMO_COMPOSE="$ROOT/docker-compose.demo.yml"
MICROSERVICES_LOGS_DASHBOARD="$ROOT/observability/grafana/provisioning/dashboards/microservices-logs.json"
PROMTAIL_CONFIG="$ROOT/observability/promtail/promtail-config.yaml"
PRODUCT_MAIN="$ROOT/backend/product/main.go"
AUCTION_MAIN="$ROOT/backend/auction/main.go"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

assert_contains() {
  local pattern=$1
  local message=$2

  if ! rg -q -- "$pattern" "$SCRIPT"; then
    fail "$message"
  fi
}

assert_not_contains() {
  local pattern=$1
  local message=$2

  if rg -q -- "$pattern" "$SCRIPT"; then
    fail "$message"
  fi
}

assert_file_contains() {
  local file=$1
  local pattern=$2
  local message=$3

  if ! rg -q -- "$pattern" "$file"; then
    fail "$message"
  fi
}

assert_file_not_contains() {
  local file=$1
  local pattern=$2
  local message=$3

  if rg -q -- "$pattern" "$file"; then
    fail "$message"
  fi
}

assert_contains 'shell_quote\(\)' "deploy-prod.sh must shell-quote dynamic SSH/rsync paths and arguments"
assert_contains 'ssh_dest\(\)' "deploy-prod.sh must build SSH destination through a helper"
assert_contains 'ssh_opts=\(' "deploy-prod.sh must keep SSH options in an argv array"
assert_contains 'ConnectTimeout=' "deploy-prod.sh must bound SSH connection hangs"
assert_contains 'ssh_with_retry\(\)' "deploy-prod.sh must retry transient SSH failures"
assert_contains 'remote_precheck' "deploy-prod.sh must keep remote precheck as a first-class function"
assert_contains 'verify_remote_deploy_ref' "deploy-prod.sh verify must check the remote .deploy-ref marker"
assert_contains 'assert_clean_for_ref' "deploy-prod.sh plan/apply must check local ref and blocking changes"
assert_contains 'redact_sensitive' "deploy-prod.sh must redact sensitive values from displayed output"
assert_contains 'http_expect\(\)' "deploy-prod.sh verify must check expected HTTP status classes"
assert_contains 'wait_for_remote_http_ready' "deploy-prod.sh must soft-wait for slow-start remote HTTP services before verifying to avoid false failures right after restart"
assert_contains 'verify_remote_containers' "deploy-prod.sh verify must check remote gateway/product/auction containers"
assert_contains 'COMPOSE_PROJECT_NAME="\$\{COMPOSE_PROJECT_NAME:-auction-demo\}"' "deploy-prod.sh must pin the remote compose project name"
assert_contains 'compose_cmd\(\)' "deploy-prod.sh must centralize docker compose invocation with the pinned project name"
assert_contains 'demo_full_stack_services\(\)' "deploy-prod.sh must define the full demo docker compose service set"
assert_contains 'gateway product auction mysql redis rabbitmq nacos nacos-mysql test-service test-dashboard loki promtail prometheus grafana growthbook growthbook-db' "deploy-prod.sh must verify all mature-project demo services, not only core services"
assert_contains 'verify_remote_compose_uniqueness' "deploy-prod.sh must fail when duplicate auction service containers exist in another compose project"
assert_contains 'com.docker.compose.project' "deploy-prod.sh must inspect compose project labels for uniqueness checks"
assert_contains 'com.docker.compose.service' "deploy-prod.sh must inspect compose service labels for uniqueness checks"
assert_contains 'compose ps --format json' "deploy-prod.sh must inspect remote compose services with machine-readable ps output"
assert_contains '--remove-orphans' "deploy-prod.sh restart must remove orphan containers inside the pinned compose project"
assert_contains 'logs --tail=100 \$\(demo_full_stack_services\)' "deploy-prod.sh must provide bounded remote log evidence for the full service set on verification failure"
assert_contains 'backup_remote' "deploy-prod.sh apply must keep rollback backups before mutating static resources"
assert_contains 'sync_nginx_config' "deploy-prod.sh apply must sync the repo Nginx config so public demo routes persist across deployments"
assert_contains 'deploy/demo/nginx-ip.conf' "deploy-prod.sh must use the versioned demo Nginx config as the source of truth"
assert_contains 'ensure_basic_auth_file' "deploy-prod.sh apply must create the Basic Auth password file required by protected demo routes"
assert_contains 'openssl passwd -apr1' "deploy-prod.sh must generate an htpasswd-compatible hash without storing hashes in git"
assert_contains 'init_demo_users' "deploy-prod.sh apply must initialize unified demo users on the remote stack"
assert_contains 'scripts/init-demo-users.sh' "deploy-prod.sh must delegate demo user seeding to the shared seed script"
assert_contains 'http_basic_expect\(\)' "deploy-prod.sh verify must prove protected routes are reachable with Basic Auth"
assert_contains 'http_body_contains\(\)' "deploy-prod.sh verify must validate endpoint bodies when a 200 fallback could mask routing bugs"
assert_contains 'ws_url' "deploy-prod.sh verify must prove test-dashboard WS discovery returns JSON, not an HTML fallback"
assert_contains '/test-ws/ws/test/progress' "deploy-prod.sh verify must prove test-dashboard WS discovery returns an Nginx-routed websocket URL"
assert_contains 'test-dashboard.*\^401\$' "deploy-prod.sh verify must check the protected Test Dashboard public route"
assert_contains 'grafana.*\^401\$' "deploy-prod.sh verify must check the protected Grafana public route"
assert_contains 'cd "\$PROJECT_ROOT/frontend/h5"' "deploy-prod.sh must build H5 from the frontend/h5 workspace"
assert_contains 'cd "\$PROJECT_ROOT/frontend/admin"' "deploy-prod.sh must build Admin from the frontend/admin workspace"
assert_contains 'npm ci' "deploy-prod.sh must install frontend dependencies in a clean worktree before building"
assert_file_contains "$DEMO_NGINX" 'location \^~ /ws/' "demo Nginx must expose test-dashboard WS discovery instead of falling back to H5 index"
assert_file_contains "$DEMO_NGINX" 'proxy_pass http://127\.0\.0\.1:8080/ws/' "demo Nginx /ws/ route must proxy to gateway /ws/"
assert_file_contains "$DEMO_NGINX" 'location \^~ /test-ws/' "demo Nginx must expose an Nginx-routed test-service websocket endpoint"
assert_file_contains "$DEMO_NGINX" 'proxy_pass http://127\.0\.0\.1:18092/' "demo Nginx /test-ws/ route must proxy websocket traffic to test-service WS"
assert_file_contains "$DEMO_NGINX" 'location \^~ /@vite/' "demo Nginx must not rewrite stale Vite dev-client module requests to index.html"
assert_file_contains "$DEMO_NGINX" 'Cache-Control "no-cache, no-store, must-revalidate"' "demo Nginx must prevent stale index.html after prod deployments"
assert_file_contains "$DEPLOY_QUICKSTART" 'test-dashboard' "deploy quickstart must document the protected Test Dashboard public entry"
assert_file_contains "$DEPLOY_QUICKSTART" 'grafana' "deploy quickstart must document the protected Grafana public entry"
assert_file_contains "$DEMO_COMPOSE" 'TEST_SERVICE_WS_URL=ws://\$\{APP_PUBLIC_HOST:-127\.0\.0\.1\}/test-ws' "demo gateway must return the public Nginx-routed test-service websocket URL, not an unreachable direct port"
assert_file_not_contains "$DEMO_COMPOSE" 'TEST_SERVICE_WS_URL=ws://\$\{APP_PUBLIC_HOST:-127\.0\.0\.1\}:18092' "demo gateway must not return the direct 18092 websocket port to browsers"
assert_file_contains "$DEMO_COMPOSE" '"127\.0\.0\.1:18092:18092"' "demo test-service websocket port must be host-local and only exposed publicly through Nginx /test-ws/"
assert_file_contains "$DEMO_COMPOSE" 'grafana/promtail:3\.' "demo Promtail image must use a 3.x release compatible with Docker API 1.44+"
assert_file_contains "$MICROSERVICES_LOGS_DASHBOARD" '"stream": "\{service_name=~' "microservices logs dashboard service variable must query streams that actually expose service_name"
assert_file_contains "$PROMTAIL_CONFIG" '__meta_docker_container_label_service_name' "Promtail must promote Docker service_name labels so non-JSON product/auction logs appear in Loki"
assert_file_contains "$PROMTAIL_CONFIG" 'target_label: service_name' "Promtail Docker discovery must write the service_name Loki label"
assert_file_contains "$PRODUCT_MAIN" 'RequestLogger\(middleware\.LoggerConfig\{' "product service must mount the structured access logger"
assert_file_contains "$PRODUCT_MAIN" 'ServiceName: "product"' "product service must emit structured access logs with service_name=product"
assert_file_contains "$AUCTION_MAIN" 'RequestLogger\(middleware\.LoggerConfig\{' "auction service must mount the structured access logger"
assert_file_contains "$AUCTION_MAIN" 'ServiceName: "auction"' "auction service must emit structured access logs with service_name=auction"

assert_not_contains 'cat .*\$REMOTE_ENV_FILE' "deploy-prod.sh must not print remote .env.demo"
assert_not_contains 'git rev-parse HEAD 2>/dev/null' "deploy-prod.sh must not depend on remote .git metadata after rsync deploy"
assert_not_contains 'grep .*ARK_API_KEY' "deploy-prod.sh must not grep or print ARK_API_KEY"
assert_not_contains 'grep .*JWT_SECRET' "deploy-prod.sh must not grep or print JWT_SECRET"
assert_not_contains 'grep .*INTERNAL_API_TOKEN' "deploy-prod.sh must not grep or print INTERNAL_API_TOKEN"
assert_file_not_contains "$DEPLOY_QUICKSTART" '暂不部署.*test-service.*grafana.*prometheus.*growthbook' "deploy quickstart must not claim mature demo services are not deployed"
assert_file_not_contains "$DEMO_COMPOSE" 'grafana/promtail:2\.9\.' "demo Promtail must not use the Docker API 1.42-limited 2.9.x image"

echo "deploy prod script checks passed"

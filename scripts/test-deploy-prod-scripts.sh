#!/bin/bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="$ROOT/scripts/deploy-prod.sh"
DEMO_NGINX="$ROOT/deploy/demo/nginx-ip.conf"

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

assert_contains 'shell_quote\(\)' "deploy-prod.sh must shell-quote dynamic SSH/rsync paths and arguments"
assert_contains 'ssh_dest\(\)' "deploy-prod.sh must build SSH destination through a helper"
assert_contains 'ssh_opts=\(' "deploy-prod.sh must keep SSH options in an argv array"
assert_contains 'remote_precheck' "deploy-prod.sh must keep remote precheck as a first-class function"
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
assert_contains 'http_basic_expect\(\)' "deploy-prod.sh verify must prove protected routes are reachable with Basic Auth"
assert_contains 'http_body_contains\(\)' "deploy-prod.sh verify must validate endpoint bodies when a 200 fallback could mask routing bugs"
assert_contains 'ws_url' "deploy-prod.sh verify must prove test-dashboard WS discovery returns JSON, not an HTML fallback"
assert_contains 'test-dashboard.*\^401\$' "deploy-prod.sh verify must check the protected Test Dashboard public route"
assert_contains 'grafana.*\^401\$' "deploy-prod.sh verify must check the protected Grafana public route"
assert_contains 'cd "\$PROJECT_ROOT/frontend/h5"' "deploy-prod.sh must build H5 from the frontend/h5 workspace"
assert_contains 'cd "\$PROJECT_ROOT/frontend/admin"' "deploy-prod.sh must build Admin from the frontend/admin workspace"
assert_contains 'npm ci' "deploy-prod.sh must install frontend dependencies in a clean worktree before building"
assert_file_contains "$DEMO_NGINX" 'location \^~ /ws/' "demo Nginx must expose test-dashboard WS discovery instead of falling back to H5 index"
assert_file_contains "$DEMO_NGINX" 'proxy_pass http://127\.0\.0\.1:8080/ws/' "demo Nginx /ws/ route must proxy to gateway /ws/"

assert_not_contains 'cat .*\$REMOTE_ENV_FILE' "deploy-prod.sh must not print remote .env.demo"
assert_not_contains 'grep .*ARK_API_KEY' "deploy-prod.sh must not grep or print ARK_API_KEY"
assert_not_contains 'grep .*JWT_SECRET' "deploy-prod.sh must not grep or print JWT_SECRET"
assert_not_contains 'grep .*INTERNAL_API_TOKEN' "deploy-prod.sh must not grep or print INTERNAL_API_TOKEN"

echo "deploy prod script checks passed"

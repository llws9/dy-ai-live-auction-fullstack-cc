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
  'init_local_auth_users' \
  "deploy-dev.sh restart must initialize README local auth users after MySQL is ready"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  './scripts/init-local-auth-users.sh' \
  "deploy-dev.sh must delegate local auth user seeding to scripts/init-local-auth-users.sh"

test -x "$ROOT/scripts/init-local-auth-users.sh" || fail "init-local-auth-users.sh must exist and be executable"

assert_contains \
  "$ROOT/scripts/init-local-auth-users.sh" \
  'ensure_column email' \
  "init-local-auth-users.sh must repair old users tables that lack email"

assert_contains \
  "$ROOT/scripts/init-local-auth-users.sh" \
  'ensure_column phone' \
  "init-local-auth-users.sh must repair old users tables that lack phone"

assert_contains \
  "$ROOT/scripts/init-local-auth-users.sh" \
  'ON DUPLICATE KEY UPDATE' \
  "init-local-auth-users.sh must be idempotent"

assert_contains \
  "$ROOT/scripts/init-local-auth-users.sh" \
  'mysql -h127\.0\.0\.1 -P3306' \
  "init-local-auth-users.sh must fall back to the host MySQL used by local backend processes"

assert_contains \
  "$ROOT/scripts/init-local-auth-users.sh" \
  '18600000001' \
  "init-local-auth-users.sh must seed README H5 user phone"

assert_contains \
  "$ROOT/scripts/init-local-auth-users.sh" \
  'merchant@example.com' \
  "init-local-auth-users.sh must seed README merchant email"

assert_contains \
  "$ROOT/scripts/init-local-auth-users.sh" \
  'admin@example.com' \
  "init-local-auth-users.sh must seed README admin email"

echo "deploy dev script checks passed"

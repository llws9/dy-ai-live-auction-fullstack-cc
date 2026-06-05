#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

mysql_exec() {
  cd "$PROJECT_ROOT"
  local mysql_container
  mysql_container="$(INTERNAL_API_TOKEN=dev docker compose ps -q mysql 2>/dev/null || true)"
  if [[ -n "$mysql_container" ]] && [[ "$(docker inspect -f '{{.State.Running}}' "$mysql_container" 2>/dev/null || true)" == "true" ]]; then
    INTERNAL_API_TOKEN=dev docker compose exec -T mysql mysql -uroot -proot auction "$@"
    return
  fi

  mysql -h127.0.0.1 -P3306 -uroot -proot auction "$@"
}

mysql_scalar() {
  mysql_exec -N -B -e "$1" | tr -d '[:space:]'
}

ensure_column() {
  local column=$1
  local definition=$2
  local exists

  exists="$(mysql_scalar "SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = '$column';")"
  if [[ "$exists" == "0" ]]; then
    mysql_exec -e "ALTER TABLE users ADD COLUMN $column $definition;"
  fi
}

ensure_index() {
  local index=$1
  local ddl=$2
  local exists

  exists="$(mysql_scalar "SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND INDEX_NAME = '$index';")"
  if [[ "$exists" == "0" ]]; then
    mysql_exec -e "$ddl"
  fi
}

echo -e "${BLUE}初始化本地 README 登录账号...${NC}"

ensure_column email "VARCHAR(128) NULL"
ensure_column phone "VARCHAR(20) NULL"
ensure_column password "VARCHAR(256) NOT NULL DEFAULT ''"
ensure_column role "TINYINT DEFAULT 0"
ensure_column status "TINYINT DEFAULT 1"
ensure_column last_login_at "TIMESTAMP NULL"

ensure_index idx_users_email "CREATE UNIQUE INDEX idx_users_email ON users (email);"
ensure_index idx_users_phone "CREATE UNIQUE INDEX idx_users_phone ON users (phone);"

mysql_exec <<'SQL'
UPDATE users
SET
  name = '本地测试用户',
  avatar = '',
  email = NULL,
  password = '$2a$10$BNzNS6qrCs4z0zPrTB01m.OlGPNBYq5o3d.8JlTrz2O5laOi6gxWy',
  role = 0,
  status = 1
WHERE phone = '18600000001';

INSERT INTO users (id, name, avatar, email, phone, password, role, status, created_at)
SELECT 9001, '本地测试用户', '', NULL, '18600000001', '$2a$10$BNzNS6qrCs4z0zPrTB01m.OlGPNBYq5o3d.8JlTrz2O5laOi6gxWy', 0, 1, NOW()
WHERE NOT EXISTS (SELECT 1 FROM users WHERE phone = '18600000001')
  AND NOT EXISTS (SELECT 1 FROM users WHERE id = 9001);

INSERT INTO users (name, avatar, email, phone, password, role, status, created_at)
SELECT '本地测试用户', '', NULL, '18600000001', '$2a$10$BNzNS6qrCs4z0zPrTB01m.OlGPNBYq5o3d.8JlTrz2O5laOi6gxWy', 0, 1, NOW()
WHERE NOT EXISTS (SELECT 1 FROM users WHERE phone = '18600000001');

UPDATE users
SET
  name = '本地商家账号',
  avatar = '',
  phone = '18600000002',
  password = '$2a$10$BNzNS6qrCs4z0zPrTB01m.OlGPNBYq5o3d.8JlTrz2O5laOi6gxWy',
  role = 1,
  status = 1
WHERE email = 'merchant@example.com';

INSERT INTO users (id, name, avatar, email, phone, password, role, status, created_at)
SELECT 9002, '本地商家账号', '', 'merchant@example.com', '18600000002', '$2a$10$BNzNS6qrCs4z0zPrTB01m.OlGPNBYq5o3d.8JlTrz2O5laOi6gxWy', 1, 1, NOW()
WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'merchant@example.com' OR phone = '18600000002')
  AND NOT EXISTS (SELECT 1 FROM users WHERE id = 9002);

INSERT INTO users (name, avatar, email, phone, password, role, status, created_at)
SELECT '本地商家账号', '', 'merchant@example.com', '18600000002', '$2a$10$BNzNS6qrCs4z0zPrTB01m.OlGPNBYq5o3d.8JlTrz2O5laOi6gxWy', 1, 1, NOW()
WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'merchant@example.com' OR phone = '18600000002');

UPDATE users
SET
  name = '系统管理员',
  avatar = '',
  email = 'admin@example.com',
  phone = NULL,
  password = '$2a$10$dAlzKYPTCJMOrgoGXm/FFubDiWeI7.JS4hNYhXp7gZRwBwV6Vwu0e',
  role = 2,
  status = 1
WHERE id = 999 OR email = 'admin@example.com';

INSERT INTO users (id, name, avatar, email, phone, password, role, status, created_at)
SELECT 999, '系统管理员', '', 'admin@example.com', NULL, '$2a$10$dAlzKYPTCJMOrgoGXm/FFubDiWeI7.JS4hNYhXp7gZRwBwV6Vwu0e', 2, 1, NOW()
WHERE NOT EXISTS (SELECT 1 FROM users WHERE id = 999 OR email = 'admin@example.com');
SQL

echo -e "${GREEN}✓ 本地 README 登录账号已就绪${NC}"

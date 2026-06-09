#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
export INTERNAL_API_TOKEN="${INTERNAL_API_TOKEN:-dev}"

GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

mysql_exec() {
  cd "$PROJECT_ROOT"
  local mysql_container
  local compose_cmd=(docker compose)
  if [[ -n "${COMPOSE_PROJECT_NAME:-}" ]]; then
    compose_cmd+=(--project-name "$COMPOSE_PROJECT_NAME")
  fi
  if [[ -n "${COMPOSE_ENV_FILE:-}" ]]; then
    compose_cmd+=(--env-file "$COMPOSE_ENV_FILE")
  fi
  mysql_container="$("${compose_cmd[@]}" ps -q mysql 2>/dev/null || true)"
  if [[ -n "$mysql_container" ]] && [[ "$(docker inspect -f '{{.State.Running}}' "$mysql_container" 2>/dev/null || true)" == "true" ]]; then
    local mysql_password
    mysql_password="$(docker exec "$mysql_container" sh -lc 'printf "%s" "${MYSQL_ROOT_PASSWORD:-root}"')"
    docker exec -i "$mysql_container" mysql --default-character-set=utf8mb4 -uroot -p"$mysql_password" auction "$@"
    return
  fi

  mysql --default-character-set=utf8mb4 -h127.0.0.1 -P3306 -uroot -p"${MYSQL_ROOT_PASSWORD:-root}" auction "$@"
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

validate_no_conflicts() {
  mysql_exec <<'SQL'
DROP TEMPORARY TABLE IF EXISTS expected_demo_users;
CREATE TEMPORARY TABLE expected_demo_users (
  id BIGINT PRIMARY KEY,
  phone VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  email VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  role TINYINT NOT NULL
);

INSERT INTO expected_demo_users VALUES
  (9101, '13800138001', NULL, 0),
  (9102, '13800138004', NULL, 0),
  (9103, '13800138002', 'merchant@example.com', 1),
  (9104, '13800138003', 'admin@example.com', 2);

DROP TEMPORARY TABLE IF EXISTS demo_user_seed_conflicts;
CREATE TEMPORARY TABLE demo_user_seed_conflicts (
  conflict_type VARCHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  existing_id BIGINT NOT NULL,
  expected_id BIGINT NOT NULL,
  existing_phone VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  expected_phone VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  existing_email VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  expected_email VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  existing_role TINYINT NULL,
  expected_role TINYINT NULL
);

INSERT INTO demo_user_seed_conflicts
SELECT
  'target_id_login_conflict',
  u.id,
  e.id,
  u.phone,
  e.phone,
  u.email,
  e.email,
  u.role,
  e.role
FROM users u
JOIN expected_demo_users e ON e.id = u.id
WHERE (u.phone IS NOT NULL AND u.phone <> e.phone)
   OR (u.email IS NOT NULL AND NOT (u.email <=> e.email));

INSERT INTO demo_user_seed_conflicts
SELECT
  'phone_conflict',
  u.id,
  e.id,
  u.phone,
  e.phone,
  u.email,
  e.email,
  u.role,
  e.role
FROM users u
JOIN expected_demo_users e ON e.phone = u.phone
WHERE u.id <> e.id;

INSERT INTO demo_user_seed_conflicts
SELECT
  'email_conflict',
  u.id,
  e.id,
  u.phone,
  e.phone,
  u.email,
  e.email,
  u.role,
  e.role
FROM users u
JOIN expected_demo_users e ON e.email = u.email
WHERE e.email IS NOT NULL
  AND u.id <> e.id;

SELECT COUNT(*) INTO @demo_user_seed_conflict_count
FROM demo_user_seed_conflicts;

SELECT *
FROM demo_user_seed_conflicts
WHERE @demo_user_seed_conflict_count > 0
ORDER BY conflict_type, expected_id, existing_id;

DROP PROCEDURE IF EXISTS fail_demo_user_seed_on_conflict;
DELIMITER //
CREATE PROCEDURE fail_demo_user_seed_on_conflict()
BEGIN
  IF @demo_user_seed_conflict_count > 0 THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'demo user seed conflict';
  END IF;
END//
DELIMITER ;
CALL fail_demo_user_seed_on_conflict();
DROP PROCEDURE IF EXISTS fail_demo_user_seed_on_conflict;
SQL
}

rebind_legacy_demo_user_ids() {
  mysql_exec <<'SQL'
DROP TEMPORARY TABLE IF EXISTS expected_demo_users;
CREATE TEMPORARY TABLE expected_demo_users (
  id BIGINT PRIMARY KEY,
  phone VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  email VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL
);

INSERT INTO expected_demo_users VALUES
  (9101, '13800138001', NULL),
  (9102, '13800138004', NULL),
  (9103, '13800138002', 'merchant@example.com'),
  (9104, '13800138003', 'admin@example.com');

START TRANSACTION;
SET FOREIGN_KEY_CHECKS = 0;

UPDATE auctions a
JOIN users u ON u.id = a.winner_id
JOIN expected_demo_users e ON e.phone = u.phone
LEFT JOIN users target ON target.id = e.id
SET a.winner_id = e.id
WHERE u.id <> e.id
  AND target.id IS NULL;

UPDATE bids b
JOIN users u ON u.id = b.user_id
JOIN expected_demo_users e ON e.phone = u.phone
LEFT JOIN users target ON target.id = e.id
SET b.user_id = e.id
WHERE u.id <> e.id
  AND target.id IS NULL;

UPDATE orders o
JOIN users u ON u.id = o.winner_id
JOIN expected_demo_users e ON e.phone = u.phone
LEFT JOIN users target ON target.id = e.id
SET o.winner_id = e.id
WHERE u.id <> e.id
  AND target.id IS NULL;

UPDATE user_balances b
JOIN users u ON u.id = b.user_id
JOIN expected_demo_users e ON e.phone = u.phone
LEFT JOIN users target ON target.id = e.id
LEFT JOIN user_balances target_balance ON target_balance.user_id = e.id
SET b.user_id = e.id
WHERE u.id <> e.id
  AND target.id IS NULL
  AND target_balance.user_id IS NULL;

UPDATE users u
JOIN expected_demo_users e ON e.phone = u.phone
LEFT JOIN users target ON target.id = e.id
SET u.id = e.id
WHERE u.id <> e.id
  AND target.id IS NULL;

SET FOREIGN_KEY_CHECKS = 1;
COMMIT;
SQL
}

assert_seeded_users() {
  mysql_exec <<SQL
DROP TEMPORARY TABLE IF EXISTS expected_demo_users;
CREATE TEMPORARY TABLE expected_demo_users (
  id BIGINT PRIMARY KEY,
  name VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  avatar VARCHAR(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  email VARCHAR(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL,
  phone VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  password VARCHAR(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  role TINYINT NOT NULL,
  status TINYINT NOT NULL
);

INSERT INTO expected_demo_users VALUES
  (9101, '演示买家A', '', NULL, '13800138001', '${DEMO_PASSWORD_HASH}', 0, 1),
  (9102, '演示买家B', '', NULL, '13800138004', '${DEMO_PASSWORD_HASH}', 0, 1),
  (9103, '演示商家', '', 'merchant@example.com', '13800138002', '${DEMO_PASSWORD_HASH}', 1, 1),
  (9104, '系统管理员', '', 'admin@example.com', '13800138003', '${DEMO_PASSWORD_HASH}', 2, 1);

SELECT COUNT(*) INTO @demo_user_seed_mismatch_count
FROM expected_demo_users e
LEFT JOIN users u ON u.id = e.id
WHERE u.id IS NULL
   OR NOT (u.name <=> e.name)
   OR NOT (u.avatar <=> e.avatar)
   OR NOT (u.email <=> e.email)
   OR NOT (u.phone <=> e.phone)
   OR NOT (u.password <=> e.password)
   OR NOT (u.role <=> e.role)
   OR NOT (u.status <=> e.status);

SELECT
  e.id AS expected_id,
  u.id AS existing_id,
  u.name AS existing_name,
  e.name AS expected_name,
  u.avatar AS existing_avatar,
  e.avatar AS expected_avatar,
  u.email AS existing_email,
  e.email AS expected_email,
  u.phone AS existing_phone,
  e.phone AS expected_phone,
  u.role AS existing_role,
  e.role AS expected_role,
  u.status AS existing_status,
  e.status AS expected_status
FROM expected_demo_users e
LEFT JOIN users u ON u.id = e.id
WHERE @demo_user_seed_mismatch_count > 0
  AND (
    u.id IS NULL
    OR NOT (u.name <=> e.name)
    OR NOT (u.avatar <=> e.avatar)
    OR NOT (u.email <=> e.email)
    OR NOT (u.phone <=> e.phone)
    OR NOT (u.password <=> e.password)
    OR NOT (u.role <=> e.role)
    OR NOT (u.status <=> e.status)
  )
ORDER BY e.id;

DROP PROCEDURE IF EXISTS fail_demo_user_seed_verification;
DELIMITER //
CREATE PROCEDURE fail_demo_user_seed_verification()
BEGIN
  IF @demo_user_seed_mismatch_count > 0 THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'demo user seed verification failed';
  END IF;
END//
DELIMITER ;
CALL fail_demo_user_seed_verification();
DROP PROCEDURE IF EXISTS fail_demo_user_seed_verification;
SQL
}

DEMO_PASSWORD='Demo@123456'
DEMO_PASSWORD_HASH='$2a$10$qLMubs2jJ79.H6tSKQRkruqVRbEH2Af91ljpMEAhSsLf642SC6wki'

echo -e "${BLUE}初始化统一演示账号...${NC}"

ensure_column email "VARCHAR(128) NULL"
ensure_column phone "VARCHAR(20) NULL"
ensure_column password "VARCHAR(256) NOT NULL DEFAULT ''"
ensure_column role "TINYINT DEFAULT 0"
ensure_column status "TINYINT DEFAULT 1"
ensure_column last_login_at "TIMESTAMP NULL"

rebind_legacy_demo_user_ids
validate_no_conflicts
ensure_index idx_users_email "CREATE UNIQUE INDEX idx_users_email ON users (email);"
ensure_index idx_users_phone "CREATE UNIQUE INDEX idx_users_phone ON users (phone);"

# Hash generated by:
# htpasswd -bnBC 10 "" 'Demo@123456' | tr -d ':\n' | sed 's/$2y/$2a/'
mysql_exec <<SQL
INSERT INTO users (id, name, avatar, email, phone, password, role, status, last_login_at)
VALUES
  (9101, '演示买家A', '', NULL, '13800138001', '${DEMO_PASSWORD_HASH}', 0, 1, NOW()),
  (9102, '演示买家B', '', NULL, '13800138004', '${DEMO_PASSWORD_HASH}', 0, 1, NOW()),
  (9103, '演示商家', '', 'merchant@example.com', '13800138002', '${DEMO_PASSWORD_HASH}', 1, 1, NOW()),
  (9104, '系统管理员', '', 'admin@example.com', '13800138003', '${DEMO_PASSWORD_HASH}', 2, 1, NOW())
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  avatar = VALUES(avatar),
  email = VALUES(email),
  phone = VALUES(phone),
  password = VALUES(password),
  role = VALUES(role),
  status = VALUES(status),
  last_login_at = VALUES(last_login_at);
SQL

assert_seeded_users

echo -e "${GREEN}✓ 统一演示账号已就绪${NC}"
echo "买家A: 13800138001 / ${DEMO_PASSWORD}"
echo "买家B: 13800138004 / ${DEMO_PASSWORD}"
echo "商家: merchant@example.com / 13800138002 / ${DEMO_PASSWORD}"
echo "管理员: admin@example.com / 13800138003 / ${DEMO_PASSWORD}"
echo "提示: 后端服务与前端登录需使用一致的 JWT_SECRET，否则已签发 token 无法跨服务校验。"

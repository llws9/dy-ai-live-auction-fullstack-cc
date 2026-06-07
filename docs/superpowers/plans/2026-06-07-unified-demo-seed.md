# 统一演示账号 Seed Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将演示账号 seed 收敛为唯一、幂等、线上线下同源的 `scripts/init-demo-users.sh`，固定生成 `13800138001/002/003/004 + Demo@123456` 四个可登录账号。

**Architecture:** 复用现有 shell seed 的 MySQL 连接、补列、补索引能力，先用脚本契约测试锁定新文件名和 138 账号口径，再 `git mv` 改造脚本并同步 `deploy-dev.sh` 与 README。账号 seed 只负责认证账号 SSOT，不初始化余额；余额仍由既有内部充值接口负责，避免扩大独立任务边界。

**Tech Stack:** Bash, MySQL 8, Docker Compose, Go/Hertz auth login API, bcrypt.

---

## Scope Boundary

本计划只实现 `docs/superpowers/specs/2026-06-06-unified-demo-seed-design.md` 的“任务2：统一 seed（独立）”。

包含：
- `scripts/init-local-auth-users.sh` 重命名为 `scripts/init-demo-users.sh`。
- 四个演示账号固定为 `13800138001/002/003/004`，密码统一 `Demo@123456`。
- `scripts/deploy-dev.sh` 本地部署自动调用新脚本。
- `scripts/test-deploy-dev-scripts.sh` 作为脚本契约测试同步更新。
- `README.md` 账号口径同步，删除本地 186/旧密码依赖。
- 本地幂等验证和登录验证。

不包含：
- 不改 `backend/seed/generators.go` 的批量造数逻辑；它不是可登录演示账号 SSOT。
- 不改压测 fixture 造用户逻辑；它依赖 JWT 直签和压测准备链路。
- 不 seed `user_balances`；H5 演示充值仍走已有 `/internal/test/user-balance` 或后续 Demo Console 接口。
- 不修改 JWT secret 配置；只在脚本输出中提示 gateway 与 auction 必须同值。

## File Structure

- Modify: `scripts/test-deploy-dev-scripts.sh`
  - 作为 TDD 的先行失败测试，更新旧脚本名、旧 186 账号、旧函数描述为新契约。
- Rename: `scripts/init-local-auth-users.sh` -> `scripts/init-demo-users.sh`
  - 唯一演示账号 seed 脚本；保留 `mysql_exec`、`ensure_column`、`ensure_index`，替换账号 SQL。
- Modify: `scripts/deploy-dev.sh`
  - 将 `init_local_auth_users` 语义更新为 `init_demo_users`，调用 `./scripts/init-demo-users.sh`。
- Modify: `README.md`
  - 线上账号增加买家B；本地测试账号改为同一套 138 账号，并声明 seed 来源。

---

### Task 1: Update Script Contract Tests First

**Files:**
- Modify: `scripts/test-deploy-dev-scripts.sh:142-192`

- [ ] **Step 1: Write the failing contract test**

Replace the existing `init-local-auth-users.sh` assertions in `scripts/test-deploy-dev-scripts.sh` with this exact block:

```bash
assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  'init_demo_users' \
  "deploy-dev.sh restart must initialize unified demo users after MySQL is ready"

assert_contains \
  "$ROOT/scripts/deploy-dev.sh" \
  './scripts/init-demo-users.sh' \
  "deploy-dev.sh must delegate demo user seeding to scripts/init-demo-users.sh"

test -x "$ROOT/scripts/init-demo-users.sh" || fail "init-demo-users.sh must exist and be executable"

if [[ -e "$ROOT/scripts/init-local-auth-users.sh" ]]; then
  fail "legacy init-local-auth-users.sh must be renamed to init-demo-users.sh"
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

if grep -q '18600000001\|18600000002\|admin123\|本地测试用户' "$ROOT/scripts/init-demo-users.sh"; then
  fail "init-demo-users.sh must not keep legacy local 186/admin123 account seeds"
fi
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
bash scripts/test-deploy-dev-scripts.sh
```

Expected: FAIL, because `scripts/init-demo-users.sh` does not exist yet and `deploy-dev.sh` still calls `init-local-auth-users.sh`.

- [ ] **Step 3: Commit only if this task is executed in its own commit**

Do not commit yet if using small local iterations; this test is intentionally red until Task 2.

---

### Task 2: Rename and Implement Unified Seed Script

**Files:**
- Rename: `scripts/init-local-auth-users.sh` -> `scripts/init-demo-users.sh`
- Modify: `scripts/init-demo-users.sh`
- Modify: `scripts/deploy-dev.sh:284-287`
- Test: `scripts/test-deploy-dev-scripts.sh`

- [ ] **Step 1: Rename the script with Git**

Run:

```bash
git mv scripts/init-local-auth-users.sh scripts/init-demo-users.sh
chmod +x scripts/init-demo-users.sh
```

Expected: `git status --short` shows a rename from `scripts/init-local-auth-users.sh` to `scripts/init-demo-users.sh`.

- [ ] **Step 2: Replace the seed script content**

Replace the full content of `scripts/init-demo-users.sh` with:

```bash
#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# bcrypt cost=10 for plaintext Demo@123456.
# Regenerate on macOS with:
#   htpasswd -bnBC 10 "" 'Demo@123456' | tr -d ':\n' | sed 's/$2y/$2a/'
DEMO_PASSWORD='Demo@123456'
DEMO_PASSWORD_HASH='$2a$10$qLMubs2jJ79.H6tSKQRkruqVRbEH2Af91ljpMEAhSsLf642SC6wki'

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

echo -e "${BLUE}初始化统一演示账号...${NC}"

ensure_column email "VARCHAR(128) NULL"
ensure_column phone "VARCHAR(20) NULL"
ensure_column password "VARCHAR(256) NOT NULL DEFAULT ''"
ensure_column role "TINYINT DEFAULT 0"
ensure_column status "TINYINT DEFAULT 1"
ensure_column last_login_at "TIMESTAMP NULL"

ensure_index idx_users_email "CREATE UNIQUE INDEX idx_users_email ON users (email);"
ensure_index idx_users_phone "CREATE UNIQUE INDEX idx_users_phone ON users (phone);"

mysql_exec <<SQL
UPDATE users
SET phone = NULL
WHERE phone IN ('13800138001', '13800138002', '13800138003', '13800138004')
  AND id NOT IN (9101, 9102, 9103, 9104);

UPDATE users
SET email = CONCAT('legacy+', id, '+', REPLACE(email, '@', '_at_'))
WHERE email IN ('merchant@example.com', 'admin@example.com')
  AND id NOT IN (9103, 9104);

INSERT INTO users (id, name, avatar, email, phone, password, role, status, created_at)
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
  status = VALUES(status);
SQL

echo -e "${GREEN}✓ 统一演示账号已就绪${NC}"
echo
echo "买家A: 13800138001 / ${DEMO_PASSWORD}"
echo "买家B: 13800138004 / ${DEMO_PASSWORD}"
echo "商家:  13800138002 / ${DEMO_PASSWORD}"
echo "管理员: 13800138003 / ${DEMO_PASSWORD}"
echo
echo -e "${YELLOW}提示: 登录后访问鉴权接口要求 gateway 与 auction 使用同一个 JWT_SECRET。本地默认应为 dev-secret。${NC}"
```

- [ ] **Step 3: Update deploy-dev hook**

In `scripts/deploy-dev.sh`, replace:

```bash
init_local_auth_users() {
  cd "$PROJECT_ROOT"
  ./scripts/init-local-auth-users.sh
}
```

with:

```bash
init_demo_users() {
  cd "$PROJECT_ROOT"
  ./scripts/init-demo-users.sh
}
```

Then replace the call site:

```bash
init_local_auth_users
```

with:

```bash
init_demo_users
```

- [ ] **Step 4: Run contract test to verify it passes**

Run:

```bash
bash scripts/test-deploy-dev-scripts.sh
```

Expected: PASS with final line:

```text
deploy dev script checks passed
```

- [ ] **Step 5: Check legacy script name is gone**

Run:

```bash
rg -n "init-local-auth-users|init_local_auth_users" scripts README.md docs
```

Expected: only old references inside historical spec docs are acceptable if they describe the pre-change problem. There must be no reference in executable scripts or README.

- [ ] **Step 6: Commit**

Run:

```bash
git add scripts/test-deploy-dev-scripts.sh scripts/deploy-dev.sh scripts/init-demo-users.sh
git add -u scripts/init-local-auth-users.sh
git commit -m "feat(seed): unify demo account seed script"
```

---

### Task 3: Verify Database Idempotency

**Files:**
- Test runtime: local MySQL via Docker Compose or host MySQL
- No source file changes

- [ ] **Step 1: Start MySQL**

Run:

```bash
INTERNAL_API_TOKEN=dev docker compose up -d mysql
```

Expected: MySQL container starts or is already running.

- [ ] **Step 2: Run seed twice**

Run:

```bash
./scripts/init-demo-users.sh
./scripts/init-demo-users.sh
```

Expected: both runs exit 0 and print the four demo accounts.

- [ ] **Step 3: Verify account rows**

Run:

```bash
INTERNAL_API_TOKEN=dev docker compose exec -T mysql mysql -uroot -proot auction -e "
SELECT id, name, email, phone, role, status
FROM users
WHERE phone IN ('13800138001','13800138002','13800138003','13800138004')
ORDER BY id;
"
```

Expected output must contain exactly:

```text
id	name	email	phone	role	status
9101	演示买家A	NULL	13800138001	0	1
9102	演示买家B	NULL	13800138004	0	1
9103	演示商家	merchant@example.com	13800138002	1	1
9104	系统管理员	admin@example.com	13800138003	2	1
```

- [ ] **Step 4: Verify idempotent count**

Run:

```bash
INTERNAL_API_TOKEN=dev docker compose exec -T mysql mysql -N -B -uroot -proot auction -e "
SELECT COUNT(*)
FROM users
WHERE phone IN ('13800138001','13800138002','13800138003','13800138004');
"
```

Expected:

```text
4
```

- [ ] **Step 5: Verify password hash is stable**

Run:

```bash
INTERNAL_API_TOKEN=dev docker compose exec -T mysql mysql -N -B -uroot -proot auction -e "
SELECT COUNT(DISTINCT password)
FROM users
WHERE phone IN ('13800138001','13800138002','13800138003','13800138004');
"
```

Expected:

```text
1
```

---

### Task 4: Verify Login Connectivity

**Files:**
- Test runtime: gateway + auction local services
- No source file changes

- [ ] **Step 1: Start backend services**

Run:

```bash
./scripts/start-local-backend.sh start
```

Expected: gateway listens on `8080`, auction listens on `8082`, websocket listens on `8083`.

- [ ] **Step 2: Login as buyer A**

Run:

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"phone":"13800138001","password":"Demo@123456"}'
```

Expected: JSON response has `code: 200` or success-equivalent response body with a non-empty token. If the local handler returns envelope code `0`, accept that only when token is present.

- [ ] **Step 3: Login as buyer B, merchant, admin**

Run:

```bash
for phone in 13800138004 13800138002 13800138003; do
  echo "login $phone"
  curl -sS -X POST http://localhost:8080/api/v1/auth/login \
    -H 'Content-Type: application/json' \
    -d "{\"phone\":\"${phone}\",\"password\":\"Demo@123456\"}"
  echo
done
```

Expected: all three responses contain a non-empty token.

- [ ] **Step 4: Verify old password is rejected**

Run:

```bash
curl -sS -i -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"phone":"13800138001","password":"123456"}'
```

Expected: HTTP `401` and response message contains `密码错误`.

- [ ] **Step 5: Commit verification note if using commits per task**

No file commit is required for runtime-only verification. Record command outputs in the SDD state file if executing through SDD.

---

### Task 5: Update README Account SSOT

**Files:**
- Modify: `README.md:16-21`
- Modify: `README.md:149-164`

- [ ] **Step 1: Update online account section**

Replace:

```markdown
## 线上账号信息

- 普通用户：手机号 `13800138001`，密码 `Demo@123456`
- 商家账号：手机号 `13800138002`，密码 `Demo@123456`
- 管理员账号：手机号 `13800138003`，密码 `Demo@123456`
```

with:

```markdown
## 演示账号信息

账号由 `scripts/init-demo-users.sh` 统一 seed，本地与 demo 服务器同源。

- 买家A：手机号 `13800138001`，密码 `Demo@123456`
- 买家B：手机号 `13800138004`，密码 `Demo@123456`
- 商家账号：手机号 `13800138002`，密码 `Demo@123456`
- 管理员账号：手机号 `13800138003`，密码 `Demo@123456`
```

- [ ] **Step 2: Replace local account section**

Replace the full `#### 4. 本地测试账号` block with:

````markdown
#### 4. 本地测试账号

本地账号与 demo 服务器账号完全一致，由 `scripts/init-demo-users.sh` 在本地部署时自动初始化。

```text
H5 买家A：
手机号：13800138001
密码：Demo@123456

H5 买家B（后台跟价/演示用）：
手机号：13800138004
密码：Demo@123456

商家账号：
手机号：13800138002
邮箱：merchant@example.com
密码：Demo@123456

Admin 管理端：
手机号：13800138003
邮箱：admin@example.com
密码：Demo@123456
```
````

- [ ] **Step 3: Verify no executable/doc user-facing old account dependency remains**

Run:

```bash
rg -n "18600000001|18600000002|admin123|init-local-auth-users" README.md scripts
```

Expected: no output.

- [ ] **Step 4: Run script contract test again**

Run:

```bash
bash scripts/test-deploy-dev-scripts.sh
```

Expected: PASS with final line:

```text
deploy dev script checks passed
```

- [ ] **Step 5: Commit**

Run:

```bash
git add README.md scripts/test-deploy-dev-scripts.sh
git commit -m "docs(readme): document unified demo accounts"
```

---

### Task 6: Final Verification and Handoff

**Files:**
- No source file changes

- [ ] **Step 1: Run final grep checks**

Run:

```bash
rg -n "init-local-auth-users|init_local_auth_users|18600000001|18600000002|admin123" README.md scripts
```

Expected: no output.

- [ ] **Step 2: Run script tests**

Run:

```bash
bash scripts/test-deploy-dev-scripts.sh
```

Expected:

```text
deploy dev script checks passed
```

- [ ] **Step 3: Run idempotency verification**

Run:

```bash
./scripts/init-demo-users.sh
./scripts/init-demo-users.sh
```

Expected: both runs exit 0.

- [ ] **Step 4: Run login smoke test**

Run:

```bash
for phone in 13800138001 13800138004 13800138002 13800138003; do
  curl -sS -X POST http://localhost:8080/api/v1/auth/login \
    -H 'Content-Type: application/json' \
    -d "{\"phone\":\"${phone}\",\"password\":\"Demo@123456\"}" | rg -q 'token'
done
```

Expected: command exits 0.

- [ ] **Step 5: Report verification evidence**

Final execution report must include:
- `bash scripts/test-deploy-dev-scripts.sh`
- `./scripts/init-demo-users.sh` twice
- SQL count result for four phones
- login smoke result for all four phones
- `rg` result proving old script name and old local accounts are gone from `README.md` and `scripts`

---

## Self-Review

Spec coverage:
- SSOT script rename to `scripts/init-demo-users.sh`: Task 2.
- 138001 account mouthpiece and buyer B: Tasks 2 and 5.
- `Demo@123456` bcrypt hash: Task 2, fixed hash included.
- `deploy-dev.sh` integration: Task 2.
- Idempotency and login verification: Tasks 3 and 4.
- README sync: Task 5.
- Old file name and 186 dependency removal: Tasks 2, 5, 6.

Intentional scope decision:
- `user_balances` is not seeded in this independent plan. The root problem is login account inconsistency, and existing recharge/top-up APIs already cover demo balance preparation.

Placeholder scan:
- No placeholder markers, no unspecified paths, no omitted test commands.

Type/name consistency:
- Script function name: `init_demo_users`.
- Script file name: `scripts/init-demo-users.sh`.
- Account IDs: `9101/9102/9103/9104`.
- Phone order: buyer A `001`, merchant `002`, admin `003`, buyer B `004`.

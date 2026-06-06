# SDD Run State - Unified Demo Seed

## Run Metadata

| Key | Value |
| --- | --- |
| Run ID | `2026-06-07-unified-demo-seed` |
| Topic | `unified-demo-seed` |
| Goal | 统一演示账号 seed 为 `scripts/init-demo-users.sh`，生成线上线下同源的 138 演示账号 |
| Mode | `subagent-driven` |
| Branch | `feat/unified-demo-seed` |
| Worktree | `/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/.worktrees/feat-unified-demo-seed` |
| Base Branch | `main` |
| Started At | `2026-06-07 00:13` |
| Owner | `main-agent` |
| Status | `active` |

## Input Documents

| Type | Path | Required | Loaded |
| --- | --- | --- | --- |
| Agent Rules | `AGENTS.md` | yes | yes |
| Constitution | `docs/CONSTITUTION.md` | yes | yes |
| Coding Standards | `docs/CODING.md` | yes | yes |
| Spec | `docs/superpowers/specs/2026-06-06-unified-demo-seed-design.md` | yes | loaded from parent workspace |
| Plan | `docs/superpowers/plans/2026-06-07-unified-demo-seed.md` | yes | loaded from parent workspace |
| Checklist | this state file | yes | yes |

## Execution Summary

| Metric | Value |
| --- | --- |
| Total Tasks | `4` |
| Done | `4` |
| Blocked | `0` |
| In Progress | `0` |
| Pending | `0` |
| Last Updated | `2026-06-07 01:58` |

## Task Matrix

| Task ID | Title | Status | Owner | Parallel Group | Depends On | Scope | Allowed Files |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `T001` | 更新脚本契约测试 | `done` | `subagent` | `P1` | `-` | 先写失败测试，锁定新脚本名与 138 账号契约 | `scripts/test-deploy-dev-scripts.sh` |
| `T002` | 重命名并实现统一 seed 脚本 | `done` | `subagent` | `P2` | `T001` | `git mv` 脚本，改造 SQL，更新 deploy hook | `scripts/init-local-auth-users.sh`, `scripts/init-demo-users.sh`, `scripts/deploy-dev.sh`, `scripts/test-deploy-dev-scripts.sh` |
| `T003` | 同步 README 账号 SSOT | `done` | `subagent` | `P3` | `T002` | 文档改为统一 138 演示账号，清理 186/旧密码依赖 | `README.md`, `docs/superpowers/sdd/runs/2026-06-07-unified-demo-seed-state.md` |
| `T004` | 最终验证与证据汇总 | `done` | `main-agent` | `P4` | `T003` | 运行契约测试、幂等验证、登录 smoke 或记录阻塞 | `docs/superpowers/sdd/runs/2026-06-07-unified-demo-seed-state.md` |

## Wave Plan

| Wave | Goal | Tasks | Start Condition | Completion Condition |
| --- | --- | --- | --- | --- |
| `W1` | 契约先行 | `T001` | baseline test passed | 测试红灯符合预期 |
| `W2` | 脚本实现 | `T002` | `T001 done` | 契约测试通过，旧脚本名不再被执行脚本引用 |
| `W3` | 文档同步 | `T003` | `T002 done` | README 与脚本账号一致 |
| `W4` | 验证收尾 | `T004` | `T003 done` | 证据完整，状态文件更新 |

## Task Records

### T001 - 更新脚本契约测试

| Key | Value |
| --- | --- |
| Status | `done` |
| Owner | `subagent` |
| Started At | `2026-06-07 00:14` |
| Completed At | `2026-06-07 00:14` |
| Branch | `feat/unified-demo-seed` |
| Worktree | `/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/.worktrees/feat-unified-demo-seed` |
| Depends On | `-` |
| Parallel Group | `P1` |

**Scope**

- 更新 `scripts/test-deploy-dev-scripts.sh` 中关于本地 auth seed 的断言。
- 期望新测试在实现前失败，失败点为 `scripts/init-demo-users.sh` 不存在或 `deploy-dev.sh` 未引用。

**Review Notes**

- Code quality finding fixed: 原断言只检查手机号和 `Demo@123456`，现在补充固定账号 ID、固定 bcrypt hash，以及四个账号的完整 seed row 绑定关系断言，要求同一行绑定 `id/name/avatar/email/phone/${DEMO_PASSWORD_HASH}/role/status/NOW()`，避免 ID/name 与 phone/hash/role 错误映射仍通过。
- Review row binding finding fixed: 已将分散的 ID/name 与 phone/hash/role 片段检查替换为四条完整组合断言，且 `${DEMO_PASSWORD_HASH}` 在测试脚本中按字面量检查，不展开当前 shell 变量。

**Allowed Files**

- `scripts/test-deploy-dev-scripts.sh`
- this state file

**Verification Evidence**

| Command | Expected | Actual | Result |
| --- | --- | --- | --- |
| `bash scripts/test-deploy-dev-scripts.sh` before change | pass baseline | `deploy dev script checks passed` | `pass` |
| `bash scripts/test-deploy-dev-scripts.sh` after test change | fail for missing new script | `exit 1; FAIL: deploy-dev.sh restart must initialize unified demo users after MySQL is ready` | `expected_fail` |
| `bash scripts/test-deploy-dev-scripts.sh` after review assertion hardening | fail for deploy hook or missing script, not shell syntax | `exit 1; FAIL: deploy-dev.sh restart must initialize unified demo users after MySQL is ready` | `expected_fail` |
| `bash -n scripts/test-deploy-dev-scripts.sh` after review assertion hardening | syntax ok | `exit 0` | `pass` |
| `bash -n scripts/test-deploy-dev-scripts.sh` after full row binding hardening | syntax ok | `exit 0; stdout: bash: no job control in this shell` | `pass` |
| `bash scripts/test-deploy-dev-scripts.sh` after full row binding hardening | fail for deploy hook or missing script, not shell syntax | `exit 1; FAIL: deploy-dev.sh restart must initialize unified demo users after MySQL is ready` | `expected_fail` |
| `git diff --check` after review assertion hardening | no whitespace errors | `exit 0` | `pass` |
| `git diff --check` after full row binding hardening | no whitespace errors | `exit 0` | `pass` |

**Modified Files**

- `scripts/test-deploy-dev-scripts.sh`
- `docs/superpowers/sdd/runs/2026-06-07-unified-demo-seed-state.md`

**Commits**

- `not_committed`

**Risks / Blockers**

- T001 只写契约测试，后续 T002 需要实现 `scripts/init-demo-users.sh` 并更新 `scripts/deploy-dev.sh` 才能转绿。

### T002 - 重命名并实现统一 seed 脚本

| Key | Value |
| --- | --- |
| Status | `done` |
| Owner | `subagent` |
| Started At | `2026-06-07 01:18` |
| Completed At | `2026-06-07 01:22` |
| Branch | `feat/unified-demo-seed` |
| Worktree | `/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/.worktrees/feat-unified-demo-seed` |
| Depends On | `T001` |
| Parallel Group | `P2` |

**Scope**

- `git mv scripts/init-local-auth-users.sh scripts/init-demo-users.sh`。
- 保留 MySQL 连接、补列、补索引逻辑。
- 写入四个账号：买家A `13800138001`/9101，买家B `13800138004`/9102，商家 `13800138002`/9103，管理员 `13800138003`/9104。
- 密码统一 `Demo@123456`，bcrypt hash 固定为 `$2a$10$qLMubs2jJ79.H6tSKQRkruqVRbEH2Af91ljpMEAhSsLf642SC6wki`。
- `deploy-dev.sh` 调用 `init_demo_users` 和 `./scripts/init-demo-users.sh`。

**Allowed Files**

- `scripts/init-local-auth-users.sh`
- `scripts/init-demo-users.sh`
- `scripts/deploy-dev.sh`
- `scripts/test-deploy-dev-scripts.sh`
- this state file

**Review Notes**

- 已用 `git mv` 将旧脚本重命名为 `scripts/init-demo-users.sh`，可执行权限保留为 `100755`。
- seed 脚本保留 Docker Compose MySQL 优先、host `mysql -h127.0.0.1 -P3306 -uroot -proot auction` fallback，保留 `ensure_column` / `ensure_index`。
- 已补齐 `email`、`phone`、`password`、`role`、`status`、`last_login_at` 列保障与 `idx_users_email`、`idx_users_phone` 唯一索引保障。
- Code quality P1 fixed: 已删除非目标账号 `SET phone = NULL` 与 `legacy+...` email 静默改写逻辑，改为 seed 前 `validate_no_conflicts` fail-closed 预检；任何目标 id/phone/email 被不一致账号占用时，先输出冲突详情，再由 MySQL `SIGNAL SQLSTATE '45000'` 终止。
- 已保留单个 `INSERT INTO users (...) VALUES ... ON DUPLICATE KEY UPDATE ...` 幂等写入 9101/9102/9103/9104，且仅在冲突预检通过后执行，避免不一致唯一键冲突更新错误行。
- 已增加 `assert_seeded_users`，upsert 后断言四个演示账号最终 `name/avatar/email/phone/password/role/status` 与期望完全匹配；不匹配时输出差异并 `SIGNAL SQLSTATE '45000'`。
- 自查未修改 README；README 同步留给 T003。

**Verification Evidence**

| Command | Expected | Actual | Result |
| --- | --- | --- | --- |
| `bash scripts/test-deploy-dev-scripts.sh` before implementation | expected fail from T001 contract | `exit 1; FAIL: deploy-dev.sh restart must initialize unified demo users after MySQL is ready` | `expected_fail` |
| `bash -n scripts/init-demo-users.sh` | syntax ok | `exit 0; stdout: bash: no job control in this shell` | `pass` |
| `bash -n scripts/deploy-dev.sh` | syntax ok | `exit 0; stdout: bash: no job control in this shell` | `pass` |
| `bash scripts/test-deploy-dev-scripts.sh` | pass | `exit 0; stdout: bash: no job control in this shell; deploy dev script checks passed` | `pass` |
| `rg -n "init-local-auth-users\|init_local_auth_users" scripts README.md` | no output in scripts/README.md | `exit 1; no output` | `pass` |
| `git diff --check` | no whitespace errors | `exit 0; no output` | `pass` |
| `bash scripts/test-deploy-dev-scripts.sh` after fail-closed contract additions, before script fix | expected fail for missing fail-closed precheck | `exit 1; FAIL: init-demo-users.sh must fail closed before seeding when target ids, phones, or emails conflict` | `expected_fail` |
| `bash -n scripts/init-demo-users.sh` after fail-closed fix | syntax ok | `exit 0; stdout: bash: no job control in this shell` | `pass` |
| `bash -n scripts/test-deploy-dev-scripts.sh` after fail-closed contract additions | syntax ok | `exit 0; stdout: bash: no job control in this shell` | `pass` |
| `bash scripts/test-deploy-dev-scripts.sh` after fail-closed fix | pass | `exit 0; stdout: bash: no job control in this shell; deploy dev script checks passed` | `pass` |
| `rg -n "SET phone = NULL\|legacy\\+" scripts/init-demo-users.sh` | no output | `exit 1; no output` | `pass` |
| `git diff --check` after fail-closed fix | no whitespace errors | `exit 0; no output` | `pass` |

**Modified Files**

- `scripts/init-local-auth-users.sh` -> `scripts/init-demo-users.sh`
- `scripts/deploy-dev.sh`
- `scripts/test-deploy-dev-scripts.sh`
- `docs/superpowers/sdd/runs/2026-06-07-unified-demo-seed-state.md`

**Commits**

- `not_committed`

**Risks / Blockers**

- 本轮修复未执行真实 MySQL seed 幂等运行，避免在 review fix 中改写本地数据；`T004` 仍需覆盖 `./scripts/init-demo-users.sh && ./scripts/init-demo-users.sh` 与登录 smoke。

### T003 - 同步 README 账号 SSOT

| Key | Value |
| --- | --- |
| Status | `done` |
| Owner | `subagent` |
| Started At | `2026-06-07 01:36` |
| Completed At | `2026-06-07 01:36` |
| Branch | `feat/unified-demo-seed` |
| Worktree | `/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/.worktrees/feat-unified-demo-seed` |
| Depends On | `T002` |
| Parallel Group | `P3` |

**Scope**

- README 线上账号段改为“演示账号信息”，增加买家B。
- README 本地测试账号段改为同一套 138 账号。
- 明确账号由 `scripts/init-demo-users.sh` 统一 seed。

**Allowed Files**

- `README.md`
- this state file

**Verification Evidence**

| Command | Expected | Actual | Result |
| --- | --- | --- | --- |
| `rg -n "18600000001|18600000002|admin123|123456" README.md` before README change | find old README account wording | `exit 0; README.md:153 18600000001; README.md:154 123456; README.md:157 18600000002; README.md:159 123456; README.md:163 admin123` | `expected_fail` |
| `rg -n "18600000001|18600000002|admin123|init-local-auth-users" README.md scripts` | no output | `exit 0; matched unrelated out-of-scope script references: scripts/migrations/002_add_auth_fields.sql admin123 comment; scripts/test-deploy-dev-scripts.sh negative assertion; scripts/performance docker/guide admin passwords` | `known_risk` |
| `rg -n "18600000001|18600000002|admin123|init-local-auth-users" README.md scripts/init-demo-users.sh scripts/deploy-dev.sh` | no output in README and relevant seed/deploy scripts | `exit 1; no output` | `pass` |
| `rg -n "13800138001|13800138004|13800138002|13800138003|scripts/init-demo-users.sh" README.md` | show unified demo accounts and seed script wording | `exit 0; README.md lines 18-22 and 153,157-158,162,167 contain the new seed script and 138 accounts` | `pass` |
| `bash scripts/test-deploy-dev-scripts.sh` | pass | `exit 0; stdout: bash: no job control in this shell; deploy dev script checks passed` | `pass` |
| `git diff --check` | no whitespace errors | `exit 0; no output` | `pass` |

**Modified Files**

- `README.md`
- `docs/superpowers/sdd/runs/2026-06-07-unified-demo-seed-state.md`

**Commits**

- `not_committed`

**Risks / Blockers**

- The prescribed broad command `rg -n "18600000001|18600000002|admin123|init-local-auth-users" README.md scripts` still matches out-of-scope `admin123` references under `scripts/migrations`, `scripts/performance`, and the test script's own negative assertion. T003 did not modify those files because this task is scoped to README and state only.

### T004 - 最终验证与证据汇总

| Key | Value |
| --- | --- |
| Status | `done` |
| Owner | `main-agent` |
| Started At | `2026-06-07 01:39` |
| Completed At | `2026-06-07 01:44` |
| Branch | `feat/unified-demo-seed` |
| Worktree | `/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/.worktrees/feat-unified-demo-seed` |
| Depends On | `T003` |
| Parallel Group | `P4` |

**Scope**

- 运行脚本契约测试。
- 尝试运行 seed 幂等验证。
- 尝试登录 smoke；若本地服务未启动，记录阻塞和可复现命令。

**Allowed Files**

- this state file

**Verification Evidence**

| Command | Expected | Actual | Result |
| --- | --- | --- | --- |
| `bash scripts/test-deploy-dev-scripts.sh && bash -n scripts/init-demo-users.sh && bash -n scripts/deploy-dev.sh && git diff --check` | all pass | `exit 0; deploy dev script checks passed; no syntax or whitespace errors` | `pass` |
| `rg -n "init-local-auth-users|init_local_auth_users|18600000001|18600000002|admin123" README.md scripts/init-demo-users.sh scripts/deploy-dev.sh` | no output | `exit 1; no output` | `pass` |
| `COMPOSE_PROJECT_NAME=dy-ai-live-auction-fullstack-cc INTERNAL_API_TOKEN=dev docker compose up -d mysql` | use Docker MySQL | `exit 1; worktree compose project attempted to bind 3306 and failed because main Docker MySQL already owns the port` | `environment_note` |
| direct Docker MySQL cleanup for old local demo login identifiers | release legacy demo-only email/phone conflicts | `updated old local demo rows only: id 999 email NULL; id 1001 phone NULL; id 880102 email/phone NULL` | `pass` |
| `COMPOSE_PROJECT_NAME=dy-ai-live-auction-fullstack-cc INTERNAL_API_TOKEN=dev ./scripts/init-demo-users.sh` twice | pass twice against Docker MySQL | `exit 0 twice; printed four demo accounts` | `pass` |
| Docker SQL query for four phones | four rows with ids 9101/9102/9103/9104 and roles 0/0/1/2 | `9101 演示买家A NULL 13800138001 0 1; 9102 演示买家B NULL 13800138004 0 1; 9103 演示商家 merchant@example.com 13800138002 1 1; 9104 系统管理员 admin@example.com 13800138003 2 1` | `pass` |
| Docker SQL count for four phones | `4` | `4` | `pass` |
| Docker SQL distinct password count | `1` | `1` | `pass` |
| Docker fail-closed conflict scenario | target phone occupied by non-target id must fail and not overwrite | `temporary row 991001 occupied 13800138001; script exited 1 with phone_conflict and SIGNAL SQLSTATE 45000; after deleting 991001, rerun restored 9101 phone` | `pass` |
| authoritative old business-account grep | no old app account source except negative test assertions; performance admin passwords are unrelated infra credentials | `only hits: test negative assertion, scripts/performance Grafana/InfluxDB admin123` | `pass` |
| migration SSOT review fix | `002_add_auth_fields.sql` must not create admin user directly | `admin insert removed; migration now states demo accounts are created only by scripts/init-demo-users.sh` | `pass` |
| login smoke for `13800138001/004/002/003` via `POST http://localhost:8080/api/v1/auth/login` | token present for all four | `token_present for all four phones` | `pass` |
| login old password `13800138001 / 123456` | 401 | `HTTP/1.1 401 Unauthorized; message 密码错误` | `pass` |
| login old phone `18600000001 / 123456` | 401 | `HTTP/1.1 401 Unauthorized; message 用户不存在` | `pass` |

**Modified Files**

 - `docs/superpowers/sdd/runs/2026-06-07-unified-demo-seed-state.md`
 - `scripts/migrations/002_add_auth_fields.sql`
 - `scripts/migrations/fix_admin_role.sql`
 - `docs/ADMIN_CREDENTIALS.md`
 - `docs/TROUBLESHOOTING.md`
 - `docs/LOGIN_ROLE_CHECK_FIX.md`
 - `docs/FRONTEND_API_PROXY_FIX.md`
 - `docs/ROLE_FIX_DOCUMENTATION.md`

**Commits**

- `not_committed`

**Risks / Blockers**

- Worktree-local compose project cannot start a second MySQL because the main Docker MySQL already binds port 3306. Verification therefore explicitly used `COMPOSE_PROJECT_NAME=dy-ai-live-auction-fullstack-cc` to operate against the existing Docker MySQL data source.
- The first Docker seed run failed closed on old local demo login identifiers (`merchant@example.com`, `admin@example.com`, `18600000001`, `18600000002`). Per user instruction to operate directly on the Docker database for local data source consistency, those legacy demo login identifiers were released in Docker MySQL before rerunning the new seed.

## Cross-Task Decisions

| Time | Decision | Reason | Impact | Owner |
| --- | --- | --- | --- | --- |
| `2026-06-07 00:13` | 不在 seed 脚本初始化 `user_balances` | 统一 seed 独立任务的本质是认证账号 SSOT；余额已有内部充值接口 | 降低任务边界和回归风险 | `main-agent` |
| `2026-06-07 01:48` | 目标 demo ID 自身字段缺失时允许自愈，目标 phone/email 被其他 ID 占用时 fail-closed | Docker 冲突验证发现过严的 id mismatch 会阻止脚本修复保留 demo ID；真正风险是非演示账号占用登录标识 | 保留 fail-closed，同时支持幂等修复部分写入状态 | `main-agent` |
| `2026-06-07 01:56` | 认证字段迁移只保留 schema，不再写入演示管理员账号 | 迁移内裸 upsert 会绕过 `init-demo-users.sh` 的 fail-closed 保护，破坏演示账号 SSOT | 账号创建唯一入口收敛到 `scripts/init-demo-users.sh` | `main-agent` |

## Test Commands

| Area | Command | Required | Last Result | Notes |
| --- | --- | --- | --- | --- |
| Script Contract | `bash scripts/test-deploy-dev-scripts.sh` | yes | `pass` | output: `deploy dev script checks passed` |
| Grep Cleanup | `rg -n "admin123|18600000001|18600000002|init-local-auth-users|id = 999|WHERE id = 999|\\(999," scripts docs README.md -g '!docs/superpowers/sdd/runs/2026-06-07-unified-demo-seed-state.md'` | yes | `pass` | only remaining hits are test negative assertion and unrelated performance infra passwords |
| Seed Idempotency | `COMPOSE_PROJECT_NAME=dy-ai-live-auction-fullstack-cc INTERNAL_API_TOKEN=dev ./scripts/init-demo-users.sh && ...` | yes | `pass` | ran twice against Docker MySQL |
| Fail-Closed Integration | temporary Docker MySQL conflict row `991001 / 13800138001` then run seed | yes | `pass` | script exited 1 with `phone_conflict`; cleanup + rerun restored 9101 |
| Login Smoke | `curl POST /api/v1/auth/login` for four phones | yes | `pass` | all four returned token |

## Final Review Checklist

- [x] 所有任务状态已更新。
- [x] 没有未解释的 `blocked` 任务。
- [x] 每个 `done` 任务都有测试或替代验证证据。
- [x] 每个实现型任务都遵循 TDD 或写明无法 TDD 的原因。
- [x] 文档与脚本账号口径一致。
- [x] 最终回答第一句展示当前分支/worktree。

## Final Handoff

当前分支/worktree：`feat/unified-demo-seed @ /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/.worktrees/feat-unified-demo-seed`

**完成项**

- T001 契约测试已切换到 `init-demo-users.sh` 新口径，并锁定完整账号 row 绑定。
- T002 已重命名并实现统一 seed 脚本，包含 fail-closed 冲突预检和最终字段断言。
- T003 README 已统一演示账号口径。
- T004 已完成 Docker MySQL 幂等验证、fail-closed 冲突验证和四账号登录 smoke。
- Final review 反馈已处理：可执行认证字段迁移不再写管理员账号，权威账号/排障文档已从 `id=999/admin123` 收敛到统一演示管理员 `id=9104/13800138003/Demo@123456`。

**未完成项**

- none

**验证结果**

- `bash scripts/test-deploy-dev-scripts.sh` passed.
- `COMPOSE_PROJECT_NAME=dy-ai-live-auction-fullstack-cc INTERNAL_API_TOKEN=dev ./scripts/init-demo-users.sh` ran twice against Docker MySQL and passed.
- Docker fail-closed conflict scenario returned `phone_conflict` and did not overwrite non-target row; cleanup then restored seed.
- Docker SQL confirmed four rows and one shared password hash.
- Four login smoke requests returned token; old password and old 186 phone returned 401.

**建议下一步**

- 发起最终 code review 或按 finishing branch 流程合入。

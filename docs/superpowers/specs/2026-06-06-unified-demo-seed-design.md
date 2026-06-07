# 统一演示账号 Seed 脚本设计 Spec

- **创建日期**：2026-06-06
- **作者**：Brainstorming session（用户 + Assistant）
- **状态**：待执行（建议在独立子会话用 TDD/幂等脚本执行）
- **关联**：H5 Demo Console 设计（`2026-06-06-h5-demo-console-design.md`）的前置依赖；与防狙击改造 spec 并列
- **执行分支建议**：`feat/unified-demo-seed`

---

## 1. 背景与问题

线上 H5 能用 README 里的 `13800138001 / Demo@123456` 登录，本地却登不上——根因是**这套 138 账号在整个代码仓库没有任何 seed 来源，是线上手动 `INSERT` 的「黑户」**。仓库里现存 4 处造 users 的来源互不统一、各有缺陷：

| 来源 | 文件 | 手机号前缀 | 密码 | 能否手机号登录 |
|---|---|---|---|---|
| A `init.sql`（compose 首次初始化） | [scripts/init.sql:86-89](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/scripts/init.sql#L86-L89) | 无 phone | 无 password 列 | 否（纯外键占位） |
| B 本地登录脚本 | [scripts/init-local-auth-users.sh:62-114](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/scripts/init-local-auth-users.sh#L62-L114) | 186 | bcrypt(`123456`/`admin123`) | 是（但前缀 186 ≠ 线上 138） |
| C product seed | [backend/seed/generators.go:104-149](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/seed/generators.go#L104-L149) | 138/158/186 随机 | 明文 `password123_hash` | 否（非 bcrypt） |
| D 压测 fixture | [backend/test/main.go:293-326](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/test/main.go#L293-L326) | nil | bcrypt(`123456`) | 否（无 phone，仅 JWT 直签） |

**没有任何一处生成「138 前缀 + `Demo@123456` + 可手机号登录」的账号。** 这就是「线上能登、本地不能登」的全部原因。

### 1.1 目标

建立**唯一、幂等、线上线下同源**的演示账号 seed：跑一次脚本，无论本地还是 demo 服务器，都得到一套完全一致的、可用手机号登录的 138 演示账号，密码统一 `Demo@123456`。彻底消灭手动 INSERT 黑户。

### 1.2 非目标

- 不重构来源 C（product seed 批量造数）/ D（压测 fixture）的造号逻辑——它们服务于各自场景（批量商品外键、压测 JWT 直签），不需要手机号登录，保持现状。
- 不引入独立 auth 服务（登录逻辑现物理上在 auction-service）。
- 不改 users 表的业务语义，只补认证所需列与演示账号数据。

---

## 2. 已确认的现状事实（执行者无需重新调查）

### 2.1 登录链路

- 入口：`POST /api/v1/auth/login`（gateway 公开透传 [router.go:51](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go#L51) → auction [main.go:410](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/main.go#L410)）。
- 匹配字段：请求体 `email` 或 `phone` 二选一 + `password`（[auth.go:180-190](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auth.go#L180-L190)）。
- 密码校验：**bcrypt**，`bcrypt.CompareHashAndPassword`（[auth.go:217](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auth.go#L217)），失败 401。
- 状态校验：`status != 1` 返回 403（[auth.go:208-214](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auth.go#L208-L214)）。

**登录可用三要素**：`phone` 唯一 + 正确 bcrypt `password` + `status=1`。

### 2.2 users 表结构（权威定义 [model/user.go:29-40](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/model/user.go#L29-L40)）

`id` / `name` / `avatar` / `email`(uniq,可空) / `phone`(uniq,可空) / `password`(bcrypt) / `role` / `status` / `last_login_at` / `created_at`。

- **role**（[model/user.go:8-12](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/model/user.go#L8-L12)）：`0=买家`、`1=商家/主播`、`2=管理员`。
- **建表 SQL 过时**：[init.sql:5-10](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/scripts/init.sql#L5-L10) 只建了 `id/name/avatar/created_at`，缺 email/phone/password/role/status。这正是 [init-local-auth-users.sh](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/scripts/init-local-auth-users.sh#L52-L57) 要 `ensure_column` 动态补列的原因——**统一脚本必须沿用这个补列兜底，不能假设列已存在**。

### 2.3 余额（独立表，非 users 列）

`user_balances` 表（[model/user_balance.go:17-23](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/model/user_balance.go#L17-L23)）：主键 `user_id`，`available_amount`/`frozen_amount` 为 `decimal(10,2)`，`currency` 默认 `CNY`。演示账号若需初始余额，要单独 UPSERT 此表。

### 2.4 bcrypt hash 现状

- 现有可登录 hash：`$2a$10$BNzNS6qrCs4z0zPrTB01m...` = 明文 `123456`；`$2a$10$dAlzKYPTCJMOrgoGXm/FF...` = 明文 `admin123`。
- **仓库没有 `Demo@123456` 的 bcrypt hash，也没有现成的 hash 生成 CLI**（仅 auth.go 注册接口内部会 `GenerateFromPassword`）。统一脚本必须自带 hash 生成步骤。

### 2.5 JWT secret 不一致（隐患，需在脚本/文档点明）

- 代码默认：`your-secret-key-change-in-production`（auction [config.go:113](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/config/config.go#L113)、gateway 同）。
- 本地启动脚本：`JWT_SECRET=dev-secret`（[start-local-backend.sh:313,315](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/scripts/start-local-backend.sh#L313-L315)）。
- `backend/.env:22`：`your_jwt_secret_key_change_in_production`（又一第三值）。
- demo：`docker-compose.demo.yml` 强制外部注入 `${JWT_SECRET:?}`。

> 与本 spec 直接相关性：seed 只写库不签发 token，JWT secret 不影响密码登录。但「gateway 与 auction 的 secret 必须同值」是登录后访问鉴权接口的前提，执行者落地时须确认本地两服务都为 `dev-secret`，并在 seed 脚本输出末尾给出提示。**本 spec 不修改 secret 配置，仅记录该隐患。**

---

## 3. 统一账号口径（SSOT）

以线上 README（[README.md:18-20](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/README.md#L18-L20)）为基准，对齐 `13800138001` 系并补一个买家B（供 Demo Console 后台跟价）：

| 角色 | 手机号 | 固定 id | role | 用途 |
|---|---|---|---|---|
| 买家A（主视角） | `13800138001` | 9101 | 0 | H5 主演示身份 |
| 买家B（影子跟价） | `13800138004` | 9102 | 0 | Demo Console 后台代为出价，制造「被超价」 |
| 商家 | `13800138002` | 9103 | 1 | 创建商品/竞拍规则 |
| 管理员 | `13800138003` | 9104 | 2 | 管理端登录 |

- **统一密码**：`Demo@123456`（全角色、全环境）。
- **固定 id**：用 9101-9104 高位固定值，避免与来源 B（999/9001/9002）、C/D（10万段）冲突，且让脚本可幂等按 id UPSERT。
- **email**：管理员可保留 `admin@example.com` 以兼容既有管理端登录路径；其余仅用 phone。
- **buyer B 编号说明**：README 线上只有 001/002/003 三个账号，无买家B。004 为本 spec 新增、线上重跑 seed 时一并补齐，不破坏既有 001-003。

> 若后续决定改用 `13800000001` 系（草案旧编号），仅需改本节表格与脚本常量，链路设计不变。

---

## 4. 方案：单一幂等 seed 脚本

### 4.1 定位

**在现有 [scripts/init-local-auth-users.sh](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/scripts/init-local-auth-users.sh) 基础上改造，使其成为唯一的演示账号 SSOT**，而非新建第二套——它已被 [deploy-dev.sh:286](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/scripts/deploy-dev.sh#L286) 接入本地部署流程，已有 `ensure_column`/`ensure_index`/幂等 UPSERT 范式，复用成本最低。

**命名（已定）**：将脚本 `git mv` 重命名为 `scripts/init-demo-users.sh`（语义从「本地 README 账号」升级为「全环境统一演示账号」），并同步更新 [deploy-dev.sh:284-286](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/scripts/deploy-dev.sh#L284-L286) 的 `init_local_auth_users()` 调用引用（函数体内 `./scripts/init-local-auth-users.sh` → `./scripts/init-demo-users.sh`；函数名可一并改为 `init_demo_users` 或保留，二者不影响调用）。改名须用 `git mv` 保留历史。改名后全仓库 grep `init-local-auth-users` 确认无其它残留引用。

### 4.2 脚本职责（在现有结构上增改）

1. **保留** `ensure_column`/`ensure_index` 补列补索引逻辑（认证字段兜底，§2.2）。
2. **替换** 现有 186 段账号 SQL，改为写入 §3 的四个 138 演示账号（幂等 UPSERT：按固定 id `ON DUPLICATE KEY UPDATE` 或 `UPDATE ... WHERE phone=`+`INSERT ... WHERE NOT EXISTS` 双保险，沿用现有脚本风格）。
3. **统一 hash**：用 `Demo@123456` 的 bcrypt hash（生成方式见 §4.3），所有四个账号同一明文。
4. **可选** UPSERT `user_balances` 给买家A/B 初始 `available_amount`（如 50000），供充值/出价演示有底；若不做则依赖 `/test/demo/recharge` 现充。执行者按 Demo Console 是否需要决定。
5. **输出提示**：脚本末尾打印四个账号 + 密码 + JWT secret 一致性提醒（§2.5）。

### 4.3 bcrypt hash 生成（解决「仓库无 Demo@123456 hash」）

bcrypt 同明文每次 hash 不同但都可校验通过，因此**预生成一个固定 hash 硬编码进脚本**即可（与现有脚本硬编码 hash 一致风格）。生成方式三选一，执行者择一并把产出的 hash 写进脚本常量：

- 方式1（推荐，零新增依赖）：`htpasswd -bnBC 10 "" 'Demo@123456' | tr -d ':\n' | sed 's/$2y/$2a/'`（macOS 自带 httpd 工具；注意 `$2y`→`$2a` 前缀归一，Go bcrypt 两者兼容但统一为 `$2a`）。
- 方式2：临时 Go 片段 `bcrypt.GenerateFromPassword([]byte("Demo@123456"), 10)` 打印结果。
- 方式3：调用本地已起的注册接口 `POST /api/v1/auth/register` 造一个账号，再从库里 `SELECT password` 取 hash。

> 执行者须在脚本注释中写明「此 hash 对应明文 `Demo@123456`」，并保留生成命令供后人复算。

### 4.4 线上同源

- demo 服务器执行同一脚本（通过 `mysql_exec` 已兼容 docker compose 与直连两种 MySQL，[init-local-auth-users.sh:12-22](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/scripts/init-local-auth-users.sh#L12-L22)）。
- 线上重跑：因全程幂等，对既有 001/002/003 仅 `UPDATE`（统一密码/状态），新增 004，不丢数据。
- **同步更新 README**：[README.md:18-20](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/README.md#L18-L20) 增补买家B `13800138004`，并注明「账号由 `scripts/init-demo-users.sh` 统一 seed，本地线上同源」，消除「文档声明 vs 无 seed 来源」的割裂。

---

## 5. TDD / 验证大纲

脚本类改造以**幂等性 + 登录连通性**为验证核心，不强求 Go 单测：

| # | 验证项 | 方法 | 期望 |
|---|---|---|---|
| V1 | 列兜底 | 全新空库跑脚本 | users 表含 phone/password/role/status 列，脚本零报错 |
| V2 | 四账号落库 | `SELECT phone,role,status FROM users WHERE phone IN(...)` | 四行，role 分别 0/0/1/2，status 均 1 |
| V3 | 幂等 | 连跑脚本两次 | 第二次零报错、账号数不增、密码 hash 不变 |
| V4 | 登录连通（核心） | 起后端后 `curl POST /api/v1/auth/login {"phone":"13800138001","password":"Demo@123456"}` | 200 且返回 JWT；商家、管理员同样可登 |
| V5 | 旧密码失效 | 用 `123456` 登 `13800138001` | 401（确认密码已切到 Demo@123456） |
| V6 | 余额（若做 §4.2.4） | `SELECT available_amount FROM user_balances WHERE user_id=9101` | 等于设定初值 |

> V4 是 Definition of Done 的硬门槛——直接复现并解决「本地登不上」的原始痛点。

---

## 6. 执行顺序与提交粒度

1. **Task 1**：生成 `Demo@123456` 的 bcrypt hash（§4.3），记录命令与产物。
2. **Task 2**：`git mv scripts/init-local-auth-users.sh scripts/init-demo-users.sh`，同步改 [deploy-dev.sh](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/scripts/deploy-dev.sh#L284-L286) 引用，并改写脚本内容（§4.2）——替换账号 SQL、统一 hash、保留补列逻辑、加末尾提示。grep `init-local-auth-users` 确认无残留。
3. **Task 3**：跑 V1-V3 幂等/落库验证 → commit。
4. **Task 4**：起后端跑 V4/V5 登录连通验证 → commit。
5. **Task 5**：同步 README（§4.4）+（可选）user_balances 初值 → commit。
6. **Task 6（可选）**：在 demo 服务器执行同一脚本，验证线上同源。

提交信息示例：
- `refactor(seed): init-local-auth-users.sh 重命名为 init-demo-users.sh`
- `feat(seed): 统一演示账号为 138 系 + Demo@123456，根除线上黑户`
- `docs(readme): 同步统一演示账号口径`

---

## 7. 风险与权衡

| 风险 | 应对 |
|---|---|
| 线上重跑覆盖既有 001/002/003 密码 | 本就是统一目标；幂等 UPDATE 仅改密码/状态，不删数据；执行前线上库备份 |
| 固定 id 9101-9104 与未来数据冲突 | 选用高位空段，且 `ON DUPLICATE KEY` 兜底；如冲突改用纯 `phone` 唯一键 UPSERT |
| `$2y` vs `$2a` 前缀（htpasswd 产物） | 统一改写为 `$2a`，Go bcrypt 兼容，但脚本内保持一致便于排查 |
| JWT secret 三处不一致（§2.5）导致登录后接口 401 | 本 spec 不改 secret；执行者确认本地 gateway/auction 同为 `dev-secret`，列入 V4 验证环境前提 |
| init.sql 仍过时，新库走 GORM AutoMigrate 不确定 | 脚本 `ensure_column` 兜底；可选附带修 init.sql 补认证列（建议另起小改动，不阻塞本 spec） |

---

## 8. 验收标准（Definition of Done）

- [ ] 唯一 seed 脚本（已重命名为 `scripts/init-demo-users.sh`，deploy-dev.sh 引用同步）可幂等执行（V1-V3 通过）。
- [ ] 旧文件名 `init-local-auth-users.sh` 全仓库无残留引用。
- [ ] 四个 138 演示账号（A/B/商家/管理员）均可用 `Demo@123456` 手机号登录（V4 通过）。
- [ ] 本地与 demo 服务器执行同一脚本得到一致账号（线上同源）。
- [ ] README 账号口径与脚本一致，且注明 seed 来源。
- [ ] 旧 186 段/旧密码不再是登录依赖（V5 通过）。

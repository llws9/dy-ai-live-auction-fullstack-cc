# H5 演示控制面板 (Demo Console) 设计 Spec

- **创建日期**：2026-06-06
- **作者**：Brainstorming session（用户 + Assistant）
- **状态**：待执行
- **前置依赖**：
  - 统一演示账号 Seed（`2026-06-06-unified-demo-seed-design.md`）——一键切换/跟价依赖可登录的 138 账号
  - 防狙击延时可见链路改造（`2026-06-06-antisnipe-delay-visibility-design.md`）——「竞拍延时」演示依赖延时实时可见
- **执行分支建议**：`feat/h5-demo-console`

---

## 1. 目标与定位

向评委做技术演示时，需要「自动播放」般的业务闭环视觉反馈。在 H5 页面常驻一个**演示控制面板（Demo Console）**：AssistiveTouch 风格悬浮球，点击扇形展开多级菜单，让演示者**不切页面、不刷新状态**地一键触发「换身份 / 他人跟价 / 并发压测 / 竞拍延时 / 充值」。

**核心约束**：
1. **纯演示定位**：不做任何环境暗门/隔离，直接写进 H5 主代码——本项目部署到线上也只是演示沙盒，不会真正上线（用户已明确）。
2. **账号口径统一**：抛弃本地/线上差异，统一用 138 体系（见 §2.3），由 seed spec 保证可登录。
3. **视觉不遮挡**：圆盘悬浮于右下角，展开不挡价格、点天灯、出价核心区。

---

## 2. 架构与边界

### 2.1 前端：H5 Demo Console

- **形态**：毛玻璃质感常驻悬浮球（右下角），点击展开扇形多级菜单（SVG 线框图标，已通过浏览器原型确认 AssistiveTouch 圆盘方案）。
- **挂载点**：[frontend/h5/src/App.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/App.tsx) 全局挂载，覆盖路由出口之上，高 `z-index`，避免路由切换重置状态。
- **菜单结构**：

  | Level 1 | Level 2 |
  |---|---|
  | **账号**（Login） | 买家A / 商家 / 管理员 / 返回 |
  | **演示**（Action） | 他人跟价(B账户) / 并发压测 / 竞拍延时 / 返回 |
  | **充值**（Recharge） | （直接执行，给当前用户加余额） |
  | **关闭** | 收起悬浮球 |

- **状态依赖**：读取当前 `auctionId`（用于对特定场次跟价/延时/压测）与 `authContext`（用于切换账号）。

> **菜单语义关键澄清（用户纠正）**：
> - 「账号」下的项是**纯切换当前 H5 登录身份**（用 138 账号 + `Demo@123456` 调 login API 免密一键登录）。
> - 「演示 → 他人跟价(B账户)」**不是切换到 B 账户**，而是通过后台接口让 B 账户（`13800138004`）代为出价一次，使**当前用户（仍是买家A）在自己屏幕上看到「被超价」动画**，并触发点天灯自动反击。绝不切走当前身份。

### 2.2 后端：Test Service 扩展

所有「非当前账号发起」的业务操作（如 B 账户跟价）都经 `test-service` 代理，绝不污染真实业务的权限校验链路。统一挂在 `/api/v1/test/demo/*`（与既有演示触发接口同前缀，符合 project_memory 约定）：

- `POST /test/demo/follow-bid`：查目标 `auction_id` 当前价与步长，以买家B（`13800138004`）身份（HTTP + `X-User-ID` 头，无需签 JWT）发起一次合法加价 → H5 经 WS 广播看到「别人超价」+ 点天灯自动反击。
- `POST /test/demo/pressure`：轻量封装现有压测逻辑，接受 `auction_id`，启动约 10s 高频短压测。
- `POST /test/demo/recharge`：给当前登录 H5 用户加余额（UPSERT `user_balances.available_amount`，**非 users 表**，见 §2.4）。
- 「竞拍延时」**不新增 demo 接口**：见 §3.2，复用防狙击真实链路。

### 2.3 数据基线：账号口径（与 seed spec 一致）

| 角色 | 手机号 | 用途 |
|---|---|---|
| 买家A（主视角） | `13800138001` | H5 主演示身份 |
| 买家B（影子跟价） | `13800138004` | follow-bid 后台代出价 |
| 商家 | `13800138002` | 创建商品/竞拍 |
| 管理员 | `13800138003` | 管理端登录 |

- 统一密码 `Demo@123456`。账号的落库与可登录性由 **seed spec** 负责，本 spec 仅消费。
- > 注：对齐线上 README 现有 001/002/003，买家B `004` 为新增。若 seed spec 最终改用 `13800000001` 系，本表同步即可。

### 2.4 余额（充值演示的正确落点）

充值演示写 `user_balances` 表（主键 `user_id`，`available_amount` 为 `decimal(10,2)`，[model/user_balance.go:17-23](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/model/user_balance.go#L17-L23)），**不是 users 表的字段**。recharge 接口须 UPSERT 该表并用 `shopspring/decimal` 处理金额（项目硬约束：金额禁用 float）。

---

## 3. 各演示动作的实现路径

### 3.1 他人跟价（B账户）— 后台代出价

- B 账户出价**无需签 JWT**：走 HTTP + `X-User-ID: 13800138004` 头即可（出价 handler 认证靠该头）。
- 天然触发点天灯自动跟价：复用现有 `PlaceBid` → 触发 `rank_update`/`bid_placed` 广播 → 当前买家A 屏幕看到被超价 → 点天灯自动反击。
- 失败容忍：高并发下乐观锁/Redis 锁冲突可能返回 400，前端弹 toast「跟价冲突，请重试」，不阻断演示。

### 3.2 竞拍延时（防狙击）— 复用真实链路，不造假

> **关键事实**：原设想「点一下把 end_time 改成剩 10s」对已打开页面无效——H5 倒计时是前端基于初始 end_time 本地自减，唯一更新途径是 WS 消息。且防狙击延时本身存在真实 bug（落库后不广播 / 前端不监听 / 调度器漏 Delayed 状态），已拆为独立 **防狙击改造 spec**。

本 spec 的「竞拍延时」演示**依赖防狙击 spec 先落地**，落地后演示动作为：

1. 演示者进入一个临近结束的竞拍直播间（或用 follow-bid/手动把场次选到临近结束）。
2. 在「即将结束」窗口内发起一次合法出价（可手动，或借「他人跟价」触发）。
3. **预期**：后端 `tryExtendAuction` 广播 `delay_triggered` → H5 倒计时立即回弹 + 弹「触发防狙击」提示。

> 即「竞拍延时」按用户拍板的「修真bug + 成就演示」方案——演示的是被修复后的真实链路，而非假动作。Demo Console 侧无需新增延时接口，至多提供一个「跳到临近结束场次」的便捷入口（可选）。

### 3.3 并发压测

轻量封装现有压测（project_memory：吞吐压测走 auction sharding、压测 fixture 自愈、客户端 drain & close）。Demo Console 仅作触发入口 + 结果 toast，压测主体复用 `backend/test` 既有能力。

### 3.4 充值

调 `/test/demo/recharge`，UPSERT 当前用户 `user_balances`（§2.4），前端 toast 提示新余额。

---

## 4. 不在本轮范围（明确剔除）

- **AI 防作弊 R4 演示**：经调查，antifraud 功能（规则引擎 R1/R4/R5 + bid hook + handler 错误码映射 + 前端二次确认框）在代码中**完全未实现**，仅有设计文档 [2026-06-01-antifraud-mvp-design.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/superpowers/specs/2026-06-01-antifraud-mvp-design.md)（状态=待实施），且隔离 worktree `feat-antifraud-mvp-sdd` 下也无实现。**用户已拍板本轮不做 R4**——故 Demo Console 菜单**不含「触发防作弊」项**。待 antifraud MVP 落地后再单独追加。

---

## 5. 实现步骤

### Phase 0：前置（独立 spec / 子会话）
- 执行 **seed spec**：四个 138 账号可用 `Demo@123456` 登录。
- 执行 **防狙击改造 spec**：延时实时可见（「竞拍延时」演示的前提）。

### Phase 1：后端 Demo API（test-service）
1. 新增 `backend/test/handler/demo.go`，注册 `/api/v1/test/demo/*`。
2. `follow-bid`：查当前价+步长，以买家B `X-User-ID` 头代出价。
3. `recharge`：UPSERT `user_balances.available_amount`（decimal）。
4. `pressure`：封装现有压测，接受 `auction_id`，约 10s 短压测。

### Phase 2：前端交互层
1. 新增 `frontend/h5/src/components/DemoConsole`（SVG + CSS/JS 扇形菜单，参考已确认原型）。
2. 集成 `authContext`：免密一键切换买家A/商家/管理员（138 账号 + `Demo@123456` 调 login）。
3. 集成各 demo API + 全局 Toast。
4. 「竞拍延时」入口对接防狙击已打通的真实链路（必要时加「选临近结束场次」便捷跳转）。
5. App.tsx 全局挂载。

---

## 6. 风险与权衡

| 风险 | 应对 |
|---|---|
| follow-bid 高并发下冲突失败 | 允许失败返回 400，前端 toast「跟价冲突，请重试」 |
| 悬浮球随路由切换状态重置 | 挂 App.tsx 顶层，独立于路由出口 |
| 「竞拍延时」演示依赖未落地的防狙击 spec | Phase 0 强约束先行；防狙击未完成前该菜单项置灰或隐藏 |
| 充值误写 users 表 | 明确写 `user_balances`，用 decimal，遵守金额硬约束 |
| 纯演示无隔离被误用到真实环境 | 用户已确认本项目仅演示沙盒，接受该取舍 |

---

## 7. 验收标准（Definition of Done）

- [ ] 悬浮球常驻 H5，扇形菜单展开/收起流畅、不遮挡核心区。
- [ ] 账号菜单可免密一键切换买家A/商家/管理员（依赖 seed）。
- [ ] 「他人跟价」让当前买家A 看到被超价动画 + 点天灯反击，身份不切走。
- [ ] 「充值」正确增加当前用户 `user_balances` 余额并前端可见。
- [ ] 「并发压测」可触发短压测并回显结果。
- [ ] 「竞拍延时」在防狙击 spec 落地后，能让 H5 倒计时实时回弹。
- [ ] 菜单不含防作弊项（本轮不做 R4）。
- [ ] 全程无环境暗门，代码直接集成。

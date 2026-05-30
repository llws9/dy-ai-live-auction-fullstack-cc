# H5 接口缺口闭环 · 总览 Tasks

**日期**：2026-05-30

**总览 Spec**：[2026-05-30-h5-missing-interfaces-closure.md](./2026-05-30-h5-missing-interfaces-closure.md)

**子 Spec**：
- [A · 用户中心](./2026-05-30-h5-missing-a-user-center.md)
- [B · 直播间收藏](./2026-05-30-h5-missing-b-livestream.md)
- [C · 商品/竞拍/分类](./2026-05-30-h5-missing-c-product-auction.md)
- [D · OrderDetail + Home 未读数](./2026-05-30-h5-missing-d-order-detail.md)

---

## 1. 执行原则

1. **按里程碑顺序推进**：M1（P0 安全修复）→ M2（P1 核心闭环）→ M3（P2 体验完善）。每个里程碑结束运行单测+验证后，再开下一个里程碑。
2. **每个 task 必须有验证步骤**。先写测试再写实现（TDD）。
3. **跨服务变更分两步合**：先合下游（被调方），再合上游（调用方），避免空指针/404。
4. **Gateway 路由变更必须重启**。每次改 [router.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go) 后 `kill && restart gateway-service` 并 `curl /health` 验证。
5. **所有提交保留旧测试**直到对应路由/组件确认下线，避免静默回归。

---

## 2. M1：P0 安全修复（最先做）

> 目标：消除 `orders/history` 越权风险。变更必须可独立部署。

### T1.1 后端：`orders/history` JWT 化

**子 Spec**：C / F-C3
**影响文件**：

- [backend/product/handler/order.go#L219-L249](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/order.go#L219-L249)
- [backend/gateway/router/router.go#L100](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go#L100)
- [backend/gateway/handler/proxy.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/handler/proxy.go)（如需 X-User-ID 注入）

**步骤**：

1. Gateway：将 `/orders/history` 从 `v1` 移到 `authGroup`（需 JWT）。
2. Gateway proxy：JWT 中间件解析后将 `user_id` 注入 `X-User-ID` header 透传给下游。
3. Product `OrderHandler.GetUserHistory`：从 `c.GetHeader("X-User-ID")` 读取（fallback 到 `c.GetInt64("user_id")`），**移除** query `user_id` 解析。
4. 单测：覆盖「无 token / token 有效 / X-User-ID 与 query 都无 → 401」三态。
5. 前端 [api.ts orderApi.history](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts#L340) 已不传 user_id，无需改动。

**验证**：

- `cd backend/product && go test ./handler/...`
- `cd backend/gateway && go test ./...`
- 启动全栈，登录后访问 H5 `/history` 页确认列表非空。
- 用 admin token 访问 `/api/v1/orders/history` 不应返回他人记录。

---

## 3. M2：P1 核心数据闭环

> 目标：补齐首页分类、直播间真实信息、收藏交互、竞拍结果、用户统计五条核心闭环。

### T2.1 product-service 内部接口（被其他服务调用）

**子 Spec**：B / C
**新增**：

- `GET /internal/products?category_id=&fields=id` 仅返回 product_id 列表，供 auction-service 做分类过滤
- `POST /internal/products/batch` Body `{ids: []}` 返回 product 摘要列表（id/name/images[0]/category_id）
- `GET /internal/users/:id` 由 auction-service 暴露（B 用），返回 name/avatar；接口前缀 `/internal/`

**约束**：

- 仅监听内部端口或仅在内网可达，**禁止注册到 Gateway**。
- 5 分钟 Redis 缓存（key: `product:summary:{id}`、`user:profile:{id}`）。
- 单测覆盖空 ids、无效 id、缓存命中。

### T2.2 auction-service：`/auctions` 分类过滤 + product 摘要内嵌

**子 Spec**：C / F-C1
**影响文件**：

- [backend/auction/handler/auction.go#L201-L281](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction.go#L201-L281)
- [backend/auction/dao/auction.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/dao/auction.go)

**步骤**：

1. handler 接收 `category_id` query。
2. 若有 category_id：先调 product-service `/internal/products?category_id=` 取 ids，再 `WHERE product_id IN (...)` 查 auctions。
3. list 返回前批量调 `/internal/products/batch` 取 product 摘要附加到每个 item。
4. product-service 不可达时降级：返回 auctions 但 `product` 字段为 null，前端容错。
5. 单测覆盖：无 category_id / 有 category_id / product-service 失败降级。

### T2.3 auction-service：`/auctions/:id/result` 字段扩展

**子 Spec**：C / F-C2
**影响文件**：[backend/auction/handler/auction.go#L122-L155](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction.go#L122-L155)

**步骤**：

1. response 增加 `won_bid`（= final_price 别名，便于前端语义清晰）、`product` 摘要（调 `/internal/products/batch`）。
2. 保持向前兼容：原字段 auction_id/product_id/status/final_price/winner_id/started_at/ended_at/delay_used 不变。
3. 单测覆盖：未结束 / 已结束有中标人 / 已结束无中标人。

### T2.4 product-service：live-stream 详情字段扩展

**子 Spec**：B / F-B1
**影响文件**：

- [backend/product/model/live_stream.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/model/live_stream.go)（新增 `video_url varchar(512) nullable`）
- [backend/product/handler/live_stream.go#L54-L88](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/live_stream.go#L54-L88)

**步骤**：

1. live_streams 表 AutoMigrate 添加 `video_url`。
2. GetDetail 通过 `creator_id` 调 auction-service `/internal/users/:id` 取 host_name/host_avatar。
3. `viewer_count` 本期降级返回 `null`（前端显示 `-`）；实现上预留接口位，下期接入。
4. `is_following`：当请求带 token 时调 auction-service follow service 查询，否则不返回该字段。
5. 单测：未登录 / 已登录已收藏 / 已登录未收藏 / 跨服务失败降级。

### T2.5 Gateway：JWTAuthOptional 中间件

**子 Spec**：B / F-B1
**影响文件**：

- [backend/gateway/middleware/jwt.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/middleware/jwt.go)
- [backend/gateway/router/router.go#L104](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go#L104)

**步骤**：

1. 新增 `JWTAuthOptional`：有 token 则解析并注入 user_id；无 token 不阻断。
2. `/live-streams/:id` 详情路由从 `v1` 改为应用 `JWTAuthOptional`，保持公开访问，但带 token 时透传 X-User-ID。
3. 单测覆盖：无 token / 有效 token / 失效 token（应 401 还是放行——本中间件**放行**，让下游决定）。

### T2.6 auction-service：`/live-streams/:id/follow-status`

**子 Spec**：B / F-B2
**影响文件**：[backend/auction/handler/follow.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/follow.go)

**步骤**：

1. 新增 handler `GetFollowStatus`，返回 `{is_following: bool}`。
2. Gateway 在 authGroup 注册 `GET /live-streams/:id/follow-status`。
3. 单测：已收藏 / 未收藏 / 无 token → 401。

### T2.7 auction-service：`/users/me/stats`

**子 Spec**：A / F-A1
**步骤**：

1. 选定方案：**Gateway BFF 聚合**（避免 auction ↔ product 双向依赖）。Gateway 新增 `/users/me/stats` 路由，handler 并行调 auction-service follow count、product-service orders/history count。
2. Gateway 在 [handler/](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/handler) 新增 `stats.go`。
3. 失败降级：单一下游失败时该字段返回 `null`，前端显示 `-`。
4. 单测：双下游成功 / 一方超时 / 两方失败。

### T2.8 前端：UI 重命名「关注」→「收藏」

**子 Spec**：B / F-B4
**影响文件**：

- [FollowButton.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/FollowButton.tsx)：文案 `关注/已关注/处理中` → `收藏/已收藏/处理中`；图标改为心形 ❤
- [Follow/index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Follow/index.tsx)：标题「我的收藏」
- [BottomNav.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/MobileShell/BottomNav.tsx)：tab 文案改为「收藏」
- [Live/index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/index.tsx)：直播间右上角按钮文案

**步骤**：

1. 仅改文案与图标，不重命名文件/组件名/Props/服务函数。
2. 同步更新断言：[FollowButton.test.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/__tests__/FollowButton.test.tsx)、[Following.test.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Follow/__tests__/Following.test.tsx)、[LiveRoom.test.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/__tests__/LiveRoom.test.tsx) 中所有 `关注`/`已关注` → `收藏`/`已收藏`。
3. `npm test` 全绿。

### T2.9 前端：is_following 由后端权威接管

**子 Spec**：B / F-B1, F-B2
**影响文件**：

- [api.ts followApi](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts#L375-L395)：新增 `getFollowStatus(liveStreamId)`
- [Live/index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/index.tsx)：进入页面读 `live_stream.is_following`，用作 FollowButton 的 `initialFollowed`
- [FollowButton.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/FollowButton.tsx)：`initialFollowed` 缺失时不再默认 false，改为 loading 占位

**步骤**：

1. 直播间详情若返回 is_following，直接用；否则调 `getFollowStatus`。
2. 失败 → 按钮灰态 + 重试。
3. 单测覆盖三态。

### T2.10 前端：auctions list 接收 product 摘要 + 分类参数

**子 Spec**：C / F-C1
**影响文件**：

- [api.ts auctionApi.list](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts#L312)：新增 `category_id` 参数
- [Home/index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Home/index.tsx)：分类 Tab 切换时透传 category_id；卡片优先读 `auction.product`，缺失时按 `product_id` fallback

**步骤**：

1. 新增 adapter 函数 `getAuctionProductSummary(auction)`：依次读 `auction.product` → `auction.product_summary` → null。
2. 卡片图片缺失时显示本地空态，**不**用占位图。
3. 单测覆盖 adapter。

---

## 4. M3：P2 体验完善

### T3.1 后端：`/user/balance` 只读余额

**子 Spec**：A / F-A2
**步骤**：

1. auction-service 新建 `user_balances` 表（user_id PK、available_amount、frozen_amount、currency 默认 CNY、updated_at）。AutoMigrate。
2. 新增 handler `GET /api/v1/user/balance`，无记录返回 `{available_amount: 0, frozen_amount: 0, currency: "CNY"}`。
3. Gateway authGroup 注册路由。
4. 单测覆盖：无记录 / 有记录。

### T3.2 后端：`/users/me/addresses` CRUD

**子 Spec**：A / F-A3
**步骤**：

1. 新建 `user_addresses` 表（id, user_id, recipient_name, phone, province, city, district, detail, is_default, created_at, updated_at）。
2. 实现 GET 列表 / POST 创建 / PUT 更新 / DELETE 删除（硬删除）/ POST `/:id/default` 设默认。
3. 业务规则：每用户最多 20 条；设默认时其他记录的 is_default 清零（同事务）。
4. Gateway authGroup 注册 5 个路由。
5. 单测覆盖：上限 / 默认切换 / 越权（A 用户改 B 地址 → 404）。

### T3.3 后端：`followed-live-streams` 列表项扩展

**子 Spec**：B / F-B3
**步骤**：

1. follow handler [GetUserFollowsHandler](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/follow.go#L91) 返回前批量调 product-service 取 host_avatar、auction_count（直播间当前进行中竞拍数）。
2. viewer_count 同 T2.4 降级返回 null。
3. 单测覆盖。

### T3.4 后端：商品规则语义修正

**子 Spec**：C / F-C4
**影响文件**：[backend/product/handler/rule.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/rule.go)

**步骤**：

1. 删除 service 层 product_id ↔ auction_id 兼容映射代码。
2. 接口契约文档更新：path id 一律为 product_id。
3. **数据校验**：执行一次性脚本检查 `auction_rules.product_id` 字段是否存在以 auction_id 写入的脏数据；如有则修正。
4. 单测保持只接 product_id。

### T3.5 前端：OrderDetail 页面

**子 Spec**：D / F-D1
**新增文件**：

- `frontend/h5/src/pages/Order/Detail.tsx`
- `frontend/h5/src/pages/Order/Detail.module.css`
- `frontend/h5/src/pages/Order/__tests__/Detail.test.tsx`

**步骤**：

1. 新增路由 `/order/:id` 注册到 [App.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/App.tsx)。
2. UI 区块：状态徽章 / 商品摘要 / 金额 / 时间线（创建/支付/发货）/ 操作区（仅返回 + 联系客服占位，**无支付按钮**）。
3. 数据：`orderApi.get(id)`；404 → 空态。
4. 主题适配：使用 `--touchpoint-*` CSS 变量；按钮 ≥44px。
5. 入口：[Notifications/index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Notifications/index.tsx) 通知 `data.order_id` 跳转；[Auction/Result.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Auction/Result.tsx) 中标按钮在有 order_id 时启用。
6. 单测：渲染 / loading / 404 / 时间线展开。

### T3.6 前端：Home 接入真实未读数

**子 Spec**：D / F-D2
**步骤**：

1. [api.ts](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts) 新增 `notificationApi` 命名空间，含 `getUnreadCount()`。
2. [Home/index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Home/index.tsx) mount + visibilitychange 时调用，count > 0 显示 BadgeDot。
3. 失败降级：不显示红点（不阻塞页面）。
4. 单测覆盖 / 集成测试 [Home.integration.test.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/__tests__/integration/Home.integration.test.tsx) 同步。

### T3.7 前端：Profile 接入 stats / balance / addresses

**子 Spec**：A
**步骤**：

1. [api.ts userApi](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts#L282) 增加 `getStats / getBalance / listAddresses / createAddress / updateAddress / deleteAddress / setDefaultAddress`。
2. [User/Index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/User/Index.tsx) 接入 stats 显示真实数；缺字段显示 `-`。
3. 余额区改为只读展示（不含充值入口）。
4. 地址列表：新增 `/addresses` 子页面或弹层（spec D 范围之外，本期可仅做列表+默认切换，不做新增/编辑表单）。
5. 单测同步。

---

## 5. 提交与 Review 节奏

| 阶段 | 提交边界 |
|---|---|
| M1 完成后 | 提交一个 PR 仅含 T1.1，独立合入 |
| M2 中 | T2.1（内部接口）单独 PR；T2.2-T2.7（后端）一个 PR；T2.8-T2.10（前端）一个 PR |
| M3 中 | A 域（T3.1, T3.2, T3.7）一个 PR；B 域（T3.3）一个 PR；D 域（T3.5, T3.6）一个 PR；C 域 T3.4 单独 PR |

每个 PR 必须包含：

- 单测全绿（`go test`、`npm test`）
- Gateway/前端构建通过
- 影响路径的 E2E 简单回归（[e2e/auction.spec.ts](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/e2e/auction.spec.ts)）

---

## 6. 风险登记

| 风险 | 缓解 |
|---|---|
| 跨服务调用 N+1、超时 | 5min Redis 缓存 + 并发限制；单一下游失败降级 null |
| Gateway 重启导致 in-flight 请求丢失 | 部署时滚动更新（本地开发可忽略） |
| follow 表数据语义被 UI 重命名误导 | 在 [README.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/README.md) 或开发者文档中明确标注「后端 follow = UI 收藏」 |
| F-C3 破坏性变更 | 已确认 admin 后台不依赖 `orders/history`；前端已不传 user_id |
| F-C4 脏数据 | T3.4 包含数据校验脚本步骤 |

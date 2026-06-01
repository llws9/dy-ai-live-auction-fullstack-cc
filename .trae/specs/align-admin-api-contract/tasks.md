# Tasks - Admin 端接口对齐

> change-id: `align-admin-api-contract`
> 全部任务遵循 TDD：每个后端任务先写 handler/dao 单元测试再实现；每个 VO/字段变更需有断言。

## 任务总览

按"修一处收益最大 + 阻塞优先"排序：

- T1（P0）分页响应统一 `list`
- T2（P0）订单管理端语义拆分
- T3（P1）拍卖详情/列表聚合 VO + 过滤参数
- T4（P1）直播间字段对齐 + 控制接口
- T5（P1）Statistics 数组化 + Dashboard 聚合
- T6（P1）拍卖规则模板模块
- T7（P1）权限与管理员账户模块
- T8（P2）个人资料编辑
- T9（P2）媒体上传
- T10（P2）`/orders/:id/pay` 方法一致性复核
- T11 前端联调与回归
- T12 OpenAPI 文档与回归测试

---

## T1 分页响应统一 `list`（P0）

- [ ] T1.1 后端：[backend/product/handler/product.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/product.go) `List` 响应字段 `items` → `list`
- [ ] T1.2 后端：[backend/product/handler/order.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/order.go) `List` 响应字段 `items` → `list`（C 端 `/orders` 同时改）
- [ ] T1.3 后端：[backend/auction/handler/auction_list.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction_list.go) `List` 响应字段 `items` → `list`
- [ ] T1.4 测试：handler 层断言 JSON 结构含 `list/total/page/page_size`
- [ ] T1.5 H5 兼容核查：grep 全仓库 `data.items / .items` 出现位置（以及 H5 与 admin 相同 endpoint 的消费）

## T2 订单管理端语义（P0）

- [ ] T2.1 设计：在 [backend/gateway/router/router.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go) 增加 `GET /admin/orders`、`GET /admin/orders/:id`，使用 `RequireAdmin` 中间件并透传到 product
- [ ] T2.2 product 服务：新增 `handler.AdminOrderList / AdminOrderGet`，**不**读 `X-User-ID`，支持 `status / user_id / page / page_size`
- [ ] T2.3 联表 VO：在 DAO 增加 `ListAdminOrders` 联表 `products / users`，返回 `OrderAdminVO{ id, product_id, product_name, product_image, user_id, user_name, auction_id, final_price, status, created_at, paid_at, shipped_at }`
- [ ] T2.4 前端：[frontend/admin/src/shared/api/index.ts](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/shared/api/index.ts) `orderApi.list` 切换到 `/admin/orders`；`orderApi.get` 切换到 `/admin/orders/:id`
- [ ] T2.5 测试：admin token 能取全量；非 admin 返回 403；C 端 `/orders` 行为不变

## T3 拍卖列表/详情聚合 VO（P1）

> 已存在：`AuctionListItem` 已嵌入 `product` 摘要（[auction_list.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction_list.go)），`AuctionFilters` 已识别 `Search/LiveStreamName/LiveStreamID/Status`。

- [ ] T3.1 List：补 `live_stream_name`（沿用 `BuildFollowedLiveStreams` 思路联表 / 内部批量接口）和 `bid_count`（按 `auction_id` 聚合 bids 表）
- [ ] T3.2 Detail：增加 `winner_name / bid_count`，并嵌套 `product` 与 `rules`（rules 由 product 服务内部接口聚合或 auction 直接读 `auction_rules` 表）
- [ ] T3.3 Bids：联表 `users` 返回 `bid.user_name`
- [ ] T3.4 列表过滤 DAO 实现复核：确认 [backend/auction/dao/auction.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/dao/auction.go) 中 `Search`（拍卖名 LIKE）、`LiveStreamName`（直播间名 LIKE）的 WHERE 子句已生效；缺失则补
- [ ] T3.5 测试：handler 单测覆盖以上字段与过滤；含旧字段兼容断言

## T4 直播间字段对齐与控制（P1）

> 已存在：`POST /api/v1/live-streams/:id/start`（admin）已实现 [router.go#L96](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go#L96)。本次只需"对接"而非"新建"。

- [ ] T4.1 数据库：迁移 `live_streams`：`host_name → streamer_name`、新增 `streamer_avatar VARCHAR(255)`、`viewer_count INT DEFAULT 0`、`ban_reason VARCHAR(255) NULL`
- [ ] T4.2 后端列表：`viewer_count` 取自 Redis `live:viewer:{id}`（缺省 0）；`auction_count` 由 `auctions WHERE live_stream_id=?` 聚合
- [ ] T4.3 后端：`status` 入参过滤
- [ ] T4.4 ~~创建直播间~~ → **接入已有 `POST /live-streams/:id/start`**：前端 Dashboard"开启直播"按钮解除 disabled，点击后弹直播间选择器后调用此接口
- [ ] T4.5 强制结束：`PUT /admin/live-streams/:id/end`，状态置 `ended` + WS 广播 `live_stream_ended`
- [ ] T4.6 封禁：`PUT /admin/live-streams/:id/ban { reason }`
- [ ] T4.7 前端：`pages-new/LiveDetail.tsx` 三个 disabled 按钮接入 `liveStreamApi.end / ban`，`api/index.ts` 增加方法（`start` 接口已存在则直接封装）
- [ ] T4.8 测试：DAO + handler 单测；rename 字段的兼容性 grep

## T5 Statistics 整改（P1）

- [ ] T5.1 Overview：[backend/product/handler/statistics.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/statistics.go) `GetOverview` 补 `ongoing_auctions, today_revenue, total_orders`
- [ ] T5.2 新增 `GET /statistics/dashboard` 一次返回 6 项 KPI；前端 Dashboard 改用此接口
- [ ] T5.3 `GetAuction` 改为 `[]AuctionStatPoint{ date, auction_count, success_rate, bid_count }`，按 `group_by=day|month` 聚合
- [ ] T5.4 `GetRevenue` 改为：`group_by=day|month` → `[]{ date, revenue, order_count }`；`group_by=category` → `[]{ category, revenue }`
- [ ] T5.5 `GetUser` 改为 `[]{ date, new_users, active_users }`
- [ ] T5.6 测试：每个 group_by 分支独立断言；空区间返回 `[]`
- [ ] T5.7 前端：`pages-new/Stats.tsx` `.slice() / .map()` 适配新结构（理论上无需改）

## T6 拍卖规则模板（P1）

- [ ] T6.1 数据库迁移：`auction_rule_templates`（id, name, description, start_price decimal, increment decimal, cap_price decimal, duration int, delay_duration int, max_delay_time int, trigger_delay_before int, created_at, updated_at, deleted_at）
- [ ] T6.2 product 服务：handler `auction_rule_template.go` 实现 5 个 endpoint
- [ ] T6.3 Apply：`POST /products/:id/apply-rule-template { template_id }` → upsert `auction_rules`
- [ ] T6.4 Gateway：透传 `/auction-rule-templates*` 与 `/products/:id/apply-rule-template`
- [ ] T6.5 前端：[frontend/admin/src/pages-new/AuctionRules.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/AuctionRules.tsx) 移除 mock，接入新 API；`api/index.ts` 增加 `auctionRuleTemplateApi`
- [ ] T6.6 测试：CRUD + Apply 流程

## T7 权限与管理员账户（P1）

- [ ] T7.1 数据库迁移：
  - `roles(id, name UNIQUE, description, is_system BOOL, created_at)`
  - `role_permissions(role_id, permission_key, PRIMARY KEY(role_id, permission_key))`
  - `admin_users` 复用现有 `users.role IN ('admin','merchant','streamer')`，**不新增表**
- [ ] T7.2 auction 服务：`role.go`、`admin_user.go` handler
- [ ] T7.3 路由：
  - `GET/POST/PUT/DELETE /admin/roles[/:id]`
  - `PUT /admin/roles/:id/permissions`
  - `GET/POST/PUT/DELETE /admin/users[/:id]`（POST 用于创建 admin/merchant/streamer 账户）
- [ ] T7.4 中间件：所有 `/admin/*` 路由必须 `RequireAdmin`
- [ ] T7.5 前端：[Permissions.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/Permissions.tsx) 移除 mock，接入 `roleApi / adminUserApi`
- [ ] T7.6 测试：CRUD + permission 全量替换 + 非 admin 403

## T8 个人资料编辑（P2）

- [ ] T8.1 数据库：`users` 增加 `preferences JSON NULL`、确保有 `avatar`
- [ ] T8.2 auction 服务：`PUT /users/me` / `PUT /users/me/password` / `PUT /users/me/preferences`
- [ ] T8.3 修改密码：bcrypt 校验 + 重写
- [ ] T8.4 Gateway 透传
- [ ] T8.5 前端：[Profile.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/Profile.tsx) 启用按钮，调用对应 API
- [ ] T8.6 测试：旧密码错误 400；密码强度（≥6 位）

## T9 媒体上传（P2）

- [ ] T9.1 product 服务：`POST /uploads`（multipart, field=`file`），写入 `static/uploads/yyyymm/<uuid>.<ext>`
- [ ] T9.2 校验：mime ∈ {jpeg,png,webp}，size ≤ 5MB
- [ ] T9.3 静态资源服务：gateway 暴露 `/static/*` 反代到 product
- [ ] T9.4 前端：`GoodsEdit.tsx` 图片上传接 `uploadApi.upload`
- [ ] T9.5 测试：超大文件 / 错误 mime 拒绝

## T10 `/orders/:id/pay` 方法收敛（P2）

> 已发现：[product/main.go#L132-L133](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/main.go#L132-L133) 同时注册了 `POST` 和 `PUT`，gateway 仅透传 `POST`，与前端 `orderApi.pay` 一致。

- [ ] T10.1 删除 [product/main.go#L133](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/main.go#L133) 的 `v1.PUT("/orders/:id/pay", orderHandler.Pay)` 冗余注册
- [ ] T10.2 端到端冒烟：admin/user 走 `POST /api/v1/orders/:id/pay` 仍能成功

## T11 前端联调与回归

- [ ] T11.1 [api/index.ts](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/shared/api/index.ts)：补 `liveStreamApi.{start, end, ban}`、`auctionRuleTemplateApi`、`roleApi`、`adminUserApi`、`uploadApi`、`profileApi`
- [ ] T11.2 各页面跑通：Dashboard / OrderList / OrderDetail / Stats / AuctionRules / Permissions / Profile
- [ ] T11.3 Jest：核心 API 调用快照 + 字段断言

## T12 文档与质量门

- [ ] T12.1 更新 [docs/api_documentation.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/api_documentation.md) 与 [docs/api-interface-list.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/api-interface-list.md)
- [ ] T12.2 [docs/DATABASE_SCHEMA.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/DATABASE_SCHEMA.md) 同步新表/新列
- [ ] T12.3 全量 `go test ./...` + `pnpm test` 通过

---

## Task Dependencies

```
T1 ──┐
T2 ──┼─→ T11 ─→ T12
T3 ──┤
T4 ──┤
T5 ──┤
T6 ──┤
T7 ──┤
T8 ──┤
T9 ──┘
T10 独立可与任何任务并行
```

并行建议：
- 第一波（独立、可并行）：**T1 / T3 / T5 / T6 / T7 / T9 / T10**
- 第二波（依赖 T1 完成）：**T2**
- 第三波（依赖结构变更）：**T4 / T8**
- 收尾：**T11 → T12**

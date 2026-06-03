# Admin 端接口对齐 Spec

> change-id: `align-admin-api-contract`
> 日期：2026-05-31
> 来源：[admin-api-audit.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/admin-api-audit.md)

## Why

管理端 `frontend/admin` 13 个页面与后端 `gateway-service / auction-service / product-service` 的接口存在系统性错位：分页响应字段不统一（`items` vs `list`）、订单语义错位（管理端只能看到自己的订单）、详情接口缺联表聚合、统计接口返回单对象但前端按数组消费（运行时抛错）、规则模板与权限管理两个模块完全是 mock。这些问题导致管理端核心页面（订单列表、统计、规则、权限）不可用。

## What Changes

围绕"最小成本贴合现有前端调用"原则，按领域拆分整改：

* **L1 统一分页响应**：`GET /products`、`GET /orders`、`GET /auctions` 后端响应统一为 `{ list, total, page, page_size }`。

* **L2 订单管理端语义**：新增 `GET /admin/orders`、`GET /admin/orders/:id`，仅 admin 可访问，返回联表后的扁平 VO；保留 C 端 `/orders` 的本人语义。

* **L3 拍卖详情聚合 VO**：`GET /auctions` **list 已嵌入** **`product`** **摘要**（见 [auction\_list.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction_list.go) `AuctionListItem`），本次只需在列表补 `live_stream_name / bid_count`；`GET /auctions/:id` 补 `product / rules / winner_name / bid_count`；`GET /auctions/:id/bids` 联表补 `bid.user_name`；列表已支持 `search / live_stream_name`（`AuctionFilters` 已识别），重点是 DAO 层 `WHERE` 实现是否覆盖。

* **L4 直播间字段对齐与控制接口**：列表/详情字段重命名为 `streamer_name / streamer_avatar`、补 `viewer_count / auction_count`，支持 `status` 过滤；**Dashboard "开启直播" 按钮直接对接已有** **`POST /api/v1/live-streams/:id/start`**（admin only，已存在 [liveStartHandler.StartLive](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/handler/live_start.go)），不再新增创建接口；新增 `PUT /admin/live-streams/:id/end`、`PUT /admin/live-streams/:id/ban`。

* **L5 统计接口结构整改**：

  * `GET /statistics/overview` 补 `ongoing_auctions / today_revenue / total_orders`；

  * `GET /statistics/auctions|revenue|users` 改为按 `start_date/end_date`+`group_by` 返回**数组**；

  * 新增 `GET /statistics/dashboard` 一次性供 Dashboard 6 卡片消费。

* **L6 拍卖规则模板**：新增 `auction_rule_templates` 表与 `GET/POST/PUT/DELETE /auction-rule-templates[/:id]`、`POST /products/:id/apply-rule-template`。

* **L7 权限管理**：新增 `roles`、`role_permissions`、`admin_users` 三类资源的 CRUD 接口（仅 admin）。

* **L8 个人资料编辑**：新增 `PUT /users/me`、`PUT /users/me/password`、`PUT /users/me/preferences`。

* **L9 媒体上传**：新增 `POST /uploads`（本地存储或 OSS 预签名，v1 走本地存储）。

* **L10 方法一致性**：[product/main.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/main.go#L132-L133) 已同时注册 `POST` + `PUT /orders/:id/pay`，与 gateway `POST` 已兼容。任务降级为：删除冗余 `PUT` 注册，保留 `POST` 单一方法。

* **L11 Dashboard 待办静态化**（用户明确指定）：v1 阶段 Dashboard 待办事项卡片保留**纯静态 UI 占位**，**不**进入本次接口对齐范围。

**BREAKING**：

* `GET /products`、`GET /orders`、`GET /auctions` 响应字段 `items` → `list`。所有调用方需同步切换。

* `GET /statistics/auctions|revenue|users` 由单对象改为数组，原结构废弃。

## Impact

* **Affected specs**：以本 spec 为唯一 SSOT；`docs/admin-api-audit.md` 为输入证据。

* **现有可复用资产**（避免重复造轮子）：

  * `POST /api/v1/live-streams/:id/start`：admin 开播 [router.go#L96](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go#L96)

  * `GET /api/v1/users/me/stats`：BFF 用户统计聚合 [router.go#L61](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go#L61)

  * `GET /api/v1/orders/summary`：订单触点摘要 [main.go#L129](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/main.go#L129)

  * `AuctionListItem` 已嵌入 `product` 摘要 [auction\_list.go#L36](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction_list.go#L36)

  * `AuctionFilters{Search, LiveStreamName, LiveStreamID, Status, ProductIDs}`：列表过滤入参对象已存在，DAO 实现需复核

* **Affected code**（关键文件）：

  * 后端

    * `backend/gateway/router/router.go`（新增 admin 路由 + 透传）

    * `backend/product/main.go`、`backend/product/handler/{order,product,statistics,live_stream}.go`、`backend/product/dao/*`

    * `backend/auction/main.go`、`backend/auction/handler/{auction,auction_list,auth,user_profile}.go`、`backend/auction/dao/*`

    * `backend/auction/handler/`：新增 `role.go`、`admin_user.go`、`auction_rule_template.go`

    * `backend/product/handler/`：新增 `upload.go`

    * `backend/*/model/*`：新增表 `roles / role_permissions / admin_users / auction_rule_templates`

  * 前端（仅必要修补，不主动重构）

    * `frontend/admin/src/shared/api/index.ts`（订单切到 `/admin/orders`、统计入参补 `group_by`）

    * 各 `pages-new/*.tsx`：仅在字段名变更时同步（如 streamer\_name 已使用前端定义）

* **数据库**：新增 4 张表（templates/roles/role\_permissions/admin\_users），新增列若干（`live_streams.viewer_count`、`live_streams.streamer_avatar`、`users.preferences` JSON 等）。

* **不影响**：H5 端 `/orders/history`、`/auctions` C 端读取行为保持现状。

***

## ADDED Requirements

### Requirement: 管理端订单列表与详情

The system SHALL provide a dedicated admin endpoint to list and inspect all orders without `X-User-ID` scoping.

#### Scenario: Admin 查看全量订单

* **WHEN** admin 携带 admin JWT 调用 `GET /admin/orders?status=&page=&page_size=`

* **THEN** 返回 `{ code, message, data: { list, total, page, page_size } }`，`list[]` 每项包含 `id, product_id, product_name, product_image, user_id, user_name, auction_id, final_price, status, created_at, paid_at, shipped_at`

* **AND** 非 admin 调用返回 403

#### Scenario: Admin 查看订单详情

* **WHEN** admin 调用 `GET /admin/orders/:id`

* **THEN** 返回上述同字段的扁平 VO

### Requirement: 拍卖规则模板管理

The system SHALL allow admins to manage reusable auction rule templates.

#### Scenario: 模板 CRUD

* **WHEN** admin 调用 `GET/POST/PUT/DELETE /auction-rule-templates[/:id]`

* **THEN** 完成 CRUD；模板字段包含 `id, name, description, start_price, increment, cap_price, duration, delay_duration, max_delay_time, trigger_delay_before, created_at, updated_at`

#### Scenario: 应用模板到商品

* **WHEN** admin 调用 `POST /products/:id/apply-rule-template { template_id }`

* **THEN** 系统将模板字段写入或覆盖该商品的 `auction_rules` 记录，并返回最新规则

### Requirement: 权限与管理员账户管理

The system SHALL provide admin-scoped CRUD for roles, role permissions, and admin users.

#### Scenario: 角色 CRUD

* **WHEN** admin 调用 `GET/POST/PUT/DELETE /admin/roles[/:id]`

* **THEN** 完成 CRUD；角色字段：`id, name, description, is_system, created_at`

#### Scenario: 角色权限分配

* **WHEN** admin 调用 `PUT /admin/roles/:id/permissions { permission_keys: [] }`

* **THEN** 全量替换该角色的权限键集合

#### Scenario: 管理员账户 CRUD

* **WHEN** admin 调用 `GET/POST/PUT/DELETE /admin/users[/:id]`

* **THEN** 完成对 `users.role IN ('admin','merchant','streamer')` 子集的查询与维护；不暴露密码哈希

### Requirement: Dashboard 聚合接口

The system SHALL expose a single endpoint that fulfills all 6 KPI cards on the admin dashboard.

#### Scenario: 一次取齐 KPI

* **WHEN** admin 调用 `GET /statistics/dashboard`

* **THEN** 返回 `{ total_auctions, ongoing_auctions, total_revenue, today_revenue, total_users, total_orders }`

### Requirement: 统计接口数组化

The system SHALL return time-series statistics as arrays grouped by `group_by`.

#### Scenario: 收入按天分组

* **WHEN** admin 调用 `GET /statistics/revenue?group_by=day&start_date=&end_date=`

* **THEN** 返回 `Array<{ date, revenue, order_count }>`

#### Scenario: 收入按品类分组

* **WHEN** admin 调用 `GET /statistics/revenue?group_by=category`

* **THEN** 返回 `Array<{ category, revenue }>`

#### Scenario: 拍卖按月统计

* **WHEN** admin 调用 `GET /statistics/auctions?group_by=month`

* **THEN** 返回 `Array<{ date, auction_count, success_rate, bid_count }>`

#### Scenario: 用户按天统计

* **WHEN** admin 调用 `GET /statistics/users?group_by=day`

* **THEN** 返回 `Array<{ date, new_users, active_users }>`

### Requirement: 直播间管理与控制

The system SHALL allow admins to start, force-end, and ban live streams.

#### Scenario: 开启直播（已存在）

* **WHEN** admin 调用 `POST /live-streams/:id/start`

* **THEN** 复用已实现的 [liveStartHandler.StartLive](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/handler/live_start.go)；前端 Dashboard 解除 disabled 直接调用，本 spec 不重复建接口

#### Scenario: 强制结束

* **WHEN** admin 调用 `PUT /admin/live-streams/:id/end`

* **THEN** 状态置为 `ended`，并广播 WS 通知

#### Scenario: 直播间封禁

* **WHEN** admin 调用 `PUT /admin/live-streams/:id/ban { reason }`

* **THEN** 状态置为 `banned`，写入 `ban_reason`

#### Scenario: 状态过滤

* **WHEN** admin 调用 `GET /admin/live-streams?status=live`

* **THEN** 仅返回符合状态的直播间

### Requirement: 个人资料编辑

The system SHALL allow current user to update profile, password, and preferences.

#### Scenario: 修改基本信息

* **WHEN** 用户调用 `PUT /users/me { name?, email?, phone?, avatar? }`

* **THEN** 更新成功并返回最新 user 对象

#### Scenario: 修改密码

* **WHEN** 用户调用 `PUT /users/me/password { old_password, new_password }`

* **THEN** 校验旧密码，bcrypt 重写新密码；旧密码错误返回 400

#### Scenario: 修改偏好

* **WHEN** 用户调用 `PUT /users/me/preferences { notification_enabled?, two_factor_enabled? }`

* **THEN** 写入 `users.preferences` JSON 列

### Requirement: 媒体上传

The system SHALL accept image uploads from admin clients.

#### Scenario: 上传图片

* **WHEN** admin 通过 `POST /uploads` (multipart/form-data, field=`file`)

* **THEN** 返回 `{ url, size, mime }`，文件存储在 `static/uploads/yyyymm/`，仅允许 `image/jpeg|png|webp`，大小 ≤ 5MB

***

## MODIFIED Requirements

### Requirement: 分页响应统一为 `list`

The system SHALL return paginated responses with the field name `list` (not `items`) for all admin-relevant list endpoints.

#### Scenario: 商品/订单/拍卖列表

* **WHEN** 调用 `GET /products | /orders | /admin/orders | /auctions`

* **THEN** `data` 字段结构为 `{ list: [], total, page, page_size }`

### Requirement: 拍卖列表/详情返回聚合 VO

The system SHALL embed product / live stream / winner info in auction list and detail responses.

#### Scenario: 列表项含商品与直播间

* **WHEN** 调用 `GET /auctions`

* **THEN** 每项含 `product:{id,name,image,category_id}`（已实现，见 `AuctionListItem`）、`live_stream_name`（**新增**）、`bid_count`（**新增**）、`current_price`、`status`、`start_time`、`end_time`

#### Scenario: 详情含赢家与规则

* **WHEN** 调用 `GET /auctions/:id`

* **THEN** 含 `product`、`rules`（直接嵌套）、`winner_name`、`bid_count`

#### Scenario: 出价记录含用户名

* **WHEN** 调用 `GET /auctions/:id/bids`

* **THEN** 每项含 `id, user_id, user_name, price, created_at`

#### Scenario: 列表过滤参数

* **WHEN** 调用 `GET /auctions?search=&live_stream_name=&live_stream_id=&status=`

* **THEN** 后端按 LIKE / 等值过滤生效

### Requirement: 直播间字段命名与聚合

The system SHALL rename live stream fields to match the admin frontend.

#### Scenario: 列表/详情字段

* **WHEN** 调用 `GET /admin/live-streams | GET /live-streams/:id`

* **THEN** 字段为 `streamer_id, streamer_name, streamer_avatar, viewer_count, auction_count, status, name, start_time, end_time`

* **AND** `viewer_count` 来自 Redis 计数器（无数据则为 0），`auction_count` 来自 `auctions` 聚合

### Requirement: Overview 字段补齐

The system SHALL include `ongoing_auctions, today_revenue, total_orders` in `/statistics/overview`.

#### Scenario: 6 项 KPI

* **WHEN** 调用 `GET /statistics/overview`

* **THEN** 返回 `{ total_auctions, ongoing_auctions, total_revenue, today_revenue, total_users, total_orders }`

### Requirement: `/orders/:id/pay` 方法收敛

The system SHALL register `/orders/:id/pay` with a single HTTP method `POST` in product service.

#### Scenario: gateway 与 service 方法一致

* **WHEN** 复核 [router.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go) 与 [product/main.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/main.go)

* **THEN** product 服务删除冗余 `PUT /orders/:id/pay` 注册（[main.go#L133](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/main.go#L133)），仅保留 `POST`，与 gateway/前端 `orderApi.pay` 一致

***

## REMOVED Requirements

### Requirement: 旧统计接口"单对象"返回结构

**Reason**：与前端数组消费方式不兼容，会运行时抛错；且无法表达 group\_by 时间序列。
**Migration**：所有调用方迁移到新数组结构；保留 `/statistics/overview` 用于聚合 KPI（其本身仍是单对象）。

### Requirement: `GET /orders` 在管理端的全量语义（如有过期理解）

**Reason**：当前实现按 `X-User-ID` 过滤，本就不是"全量"。
**Migration**：管理端切到新增的 `/admin/orders`；`/orders` 保持 C 端"我的订单"语义不变。

***

## 范围裁剪与不做项

* **Dashboard 待办事项**：v1 不接入接口，保留前端静态占位卡片。后续若有需求另立 spec。

* **OSS 预签名上传**：v1 走本地静态目录，URL 由后端拼接；v2 再考虑 OSS。

* **导出 CSV（O03）**：本次不实现，沉淀到 backlog。

* **催付/催发货（O01/O02）**：本次不实现，沉淀到 backlog。

* **两步验证真实下发**：v1 仅提供 `preferences` 开关字段持久化，TOTP 等实际机制不做。

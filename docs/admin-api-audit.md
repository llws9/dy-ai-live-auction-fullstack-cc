# Admin 前端接口核查报告

> 范围：`frontend/admin`（管理端 PC 后台，13 个页面）
> 后端：`gateway-service` (8080) → `auction-service` / `product-service`
> 编制日期：2026-05-31
> 编制人：Trae Agent
> 数据源：
> - 前端：`frontend/admin/src/App.tsx`（路由）、`frontend/admin/src/shared/api/index.ts`（接口封装）、`frontend/admin/src/pages-new/*.tsx`（页面）
> - 后端：`backend/gateway/router/router.go`、`backend/{auction,product}/main.go`、`backend/{auction,product}/handler/*.go`

---

## 一、文档使用说明

本报告分三章：
- 第二章：**按界面梳理功能点 → 接口需求 → 后端命中情况**（One‑Sheet）
- 第三章：**缺失接口清单**（前端期望但后端无）
- 第四章：**字段/结构不匹配清单**（路由存在但请求参数或响应字段对不齐）

每条记录均使用统一的"严重度"标签：
- **P0**：阻塞核心流程（用户登录后无法使用核心功能）
- **P1**：影响某页面正常展示或操作（数据缺失/字段错位）
- **P2**：体验或扩展能力缺失（按钮当前已 disabled）

---

## 二、按界面 × 功能点 × 接口对照

### 1. 登录页 [Login.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/Login.tsx)

| 功能点 | 前端调用 | 后端实现 | 命中 | 备注 |
|---|---|---|---|---|
| 邮箱/手机+密码登录 | `POST /api/v1/auth/login` | [auth.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auth.go) `Login` | ✅ | 返回 `{token, user}`，已对齐 |
| 登录态本地化 | `localStorage admin_auth_token / admin_auth_user` | — | — | 纯前端 |
| 角色校验（仅 admin/streamer/merchant） | 前端校验 `user.role` | JWT Claims `role` | ✅ | 已实现 |

---

### 2. 工作台首页 [Dashboard.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/Dashboard.tsx)

| 功能点 | 前端调用 | 后端实现 | 命中 | 备注 |
|---|---|---|---|---|
| 6 项 KPI（总场次/进行中/总收入/用户数/今日收入/订单数） | `GET /statistics/overview` | [statistics.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/statistics.go) `GetOverview` | ⚠️ | **字段缺失**：`ongoing_auctions / today_revenue / total_orders` 后端未返回 |
| 近 7 日趋势（按天） | `GET /statistics/revenue?group_by=day` | `GetRevenue` | ⚠️ | **结构不匹配**：后端返回单对象 `{daily_revenue, category_distribution, ...}`，前端按数组消费 |
| 收入构成（按品类） | `GET /statistics/revenue?group_by=category` | `GetRevenue` | ⚠️ | 同上，且 `group_by` 入参后端未识别（仅按 start/end_date 分流） |
| 待办事项卡片 | 静态/无接口 | — | — | UI 占位，后续可对接 |
| 快捷入口"开启直播" | 按钮已 `disabled` | — | — | 后端无创建直播间接口（见缺失‑M01） |

---

### 3. 商品列表 [GoodsList.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/GoodsList.tsx)

| 功能点 | 前端调用 | 后端实现 | 命中 | 备注 |
|---|---|---|---|---|
| 列表 + 状态筛选 + 分页 | `GET /products?status&page&page_size` | [product.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/product.go) `List` | ⚠️ | **字段不匹配**：后端 `{items, total, page, page_size}`，前端解 `list` |
| 删除 | `DELETE /products/:id` | `Delete` | ✅ | — |
| 上架 | `POST /products/:id/publish` | `Publish` | ✅ | — |
| 下架 + 原因 | `POST /products/:id/unpublish` | `Unpublish` | ✅ | 注意 body `{reason}` 后端是否落库未确认 |

---

### 4. 商品创建/编辑 [GoodsEdit.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/GoodsEdit.tsx)

| 功能点 | 前端调用 | 后端实现 | 命中 | 备注 |
|---|---|---|---|---|
| 详情回填 | `GET /products/:id` | `Get` | ✅ | — |
| 新建 | `POST /products` | `Create` | ✅ | 字段：`name/category/brand/description/images` |
| 编辑 | `PUT /products/:id` | `Update` | ✅ | — |
| 提交后立即上架 | `POST /products/:id/publish` | `Publish` | ✅ | — |
| 图片上传 | 前端直传/无接口 | — | ⚠️ | 后端无统一上传接口（见缺失‑U01） |

---

### 5. 拍卖列表 [AuctionList.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/AuctionList.tsx)

| 功能点 | 前端调用 | 后端实现 | 命中 | 备注 |
|---|---|---|---|---|
| 列表 + status/search/page | `GET /auctions` | [auction_list.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction_list.go) `List` | ⚠️ | **字段不匹配**：`{items, total}` ↔ `{list, total}`；`search/live_stream_name` 入参后端未支持 |
| 列表项展示 product/live_stream/bid_count/current_price | 期望 `auction.product.name`、`live_stream_name`、`bid_count` | `model.Auction` | ⚠️ | **字段缺失**：后端 `Auction` 不嵌套 `product`、无 `live_stream_name`、`bid_count` 需聚合 |
| 取消 | `PUT /auctions/:id/cancel` | `Cancel` | ✅ | — |

---

### 6. 拍卖详情 [AuctionDetail.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/AuctionDetail.tsx)

| 功能点 | 前端调用 | 后端实现 | 命中 | 备注 |
|---|---|---|---|---|
| 拍卖详情 | `GET /auctions/:id` | [auction.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction.go) `Get` | ⚠️ | 期望 `winner_name / bid_count / product / rules` 嵌套，后端仅返回扁平 `model.Auction` |
| 出价记录 | `GET /auctions/:id/bids` | `GetBids` | ⚠️ | 期望 `bid.user_name`，后端 `model.Bid` 仅有 `user_id`（需加联表） |
| 商品信息 | `GET /products/:id` | product `Get` | ✅ | — |
| 规则信息 | `GET /products/:productId/rules` | [product.go]() `GetRules` | ✅ | 字段：`start_price/increment/cap_price/duration/trigger_delay_before/delay_duration` |
| 强制结束/取消 | `PUT /auctions/:id/cancel` | `Cancel` | ✅ | — |

---

### 7. 拍卖规则 [AuctionRules.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/AuctionRules.tsx)

| 功能点 | 前端调用 | 后端实现 | 命中 | 备注 |
|---|---|---|---|---|
| **规则模板**列表/创建/编辑/删除 | 完全 mock 数据 | — | ❌ | **整模块缺失**（见缺失‑R01） |
| 应用模板到商品 | mock | — | ❌ | 见缺失‑R01 |
| 商品规则 CRUD（已存在） | `GET/POST /products/:productId/rules` | [product.go]() `GetRules / CreateRules` | ✅ | 仅"商品级规则"，与"模板"是不同维度 |

---

### 8. 直播列表 [LiveList.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/LiveList.tsx)

| 功能点 | 前端调用 | 后端实现 | 命中 | 备注 |
|---|---|---|---|---|
| 管理端列表 | `GET /admin/live-streams?page&page_size` | [live_stream.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/live_stream.go) `ListAdmin` | ⚠️ | **字段不匹配**：前端期望 `streamer_name / streamer_avatar / viewer_count / auction_count`，后端返回 `host_name=""` 占位、缺 `auction_count` |
| 状态筛选（直播中/已结束/未开始） | `status` 入参 | `ListAdmin` | ❌ | 后端未实现 status 过滤 |

---

### 9. 直播详情 [LiveDetail.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/LiveDetail.tsx)

| 功能点 | 前端调用 | 后端实现 | 命中 | 备注 |
|---|---|---|---|---|
| 详情 | `GET /live-streams/:id` | `GetDetail` | ⚠️ | 同上字段命名差异 |
| 直播间下的拍卖列表 | `GET /auctions?live_stream_id=` | `List` | ⚠️ | 入参 `live_stream_id` 后端是否过滤需确认；返回结构同列表 P1 |
| **强制结束/封禁/警告** | 三个按钮均 `disabled` | — | ❌ | 见缺失‑L01/L02 |

---

### 10. 订单列表 [OrderList.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/OrderList.tsx)

| 功能点 | 前端调用 | 后端实现 | 命中 | 备注 |
|---|---|---|---|---|
| 列表 + 状态/分页 | `GET /orders?status&page&page_size` | [order.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/order.go) `List` | ❌ | **关键不匹配**：① 返回 `items` ↔ `list`；② **强依赖 `X-User-ID` 仅查本人订单**，管理端需要全量 |
| 列表字段 `product_name / user_name / final_price / status` | 期望嵌套 | `model.Order` | ⚠️ | 后端 `Order` 仅含外键 `user_id / product_id / auction_id`，需联表后再返回 |

---

### 11. 订单详情 [OrderDetail.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/OrderDetail.tsx)

| 功能点 | 前端调用 | 后端实现 | 命中 | 备注 |
|---|---|---|---|---|
| 详情 | `GET /orders/:id` | `Get` | ⚠️ | 期望 `product_name / product_image / paid_at / shipped_at / user_name`；后端 `model.Order` 缺这些聚合字段 |
| 发货 | `PUT /orders/:id/ship` | `Ship` | ✅ | — |
| 催付 / 催发货 / 联系用户 | 无接口 | — | ❌ | 见缺失‑O01/O02 |

---

### 12. 数据统计 [Stats.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/Stats.tsx)

| 功能点 | 前端调用 | 后端实现 | 命中 | 备注 |
|---|---|---|---|---|
| 拍卖统计（按时间数组） | `GET /statistics/auctions` | [statistics.go]() `GetAuction` | ⚠️ | **类型不匹配**：后端返回 `AuctionStatistics` 单对象，前端 `as any[]` 后 `.map`，运行时会异常 |
| 收入统计（按月） | `GET /statistics/revenue?group_by=month` | `GetRevenue` | ⚠️ | 同上类型不匹配；`group_by` 后端未识别 |
| 用户统计 | `GET /statistics/users` | `GetUser` | ⚠️ | 同上类型不匹配 |
| 期望字段 `auction_count / success_rate / bid_count / new_users / active_users / revenue` | — | — | ❌ | 字段命名/聚合粒度需重新设计 |

---

### 13. 个人资料 [Profile.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/Profile.tsx)

| 功能点 | 前端调用 | 后端实现 | 命中 | 备注 |
|---|---|---|---|---|
| 当前用户信息 | `GET /users/me` | [auth.go]() `GetCurrentUser` | ✅ | — |
| 修改基本信息 | 按钮 `disabled` | — | ❌ | 见缺失‑U02 |
| 修改密码 | 按钮 `disabled` | — | ❌ | 见缺失‑U03 |
| 两步验证 / 通知偏好 | 按钮 `disabled` | — | ❌ | 见缺失‑U04 |

---

### 14. 权限管理 [Permissions.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/admin/src/pages-new/Permissions.tsx)

| 功能点 | 前端调用 | 后端实现 | 命中 | 备注 |
|---|---|---|---|---|
| 角色列表 / CRUD | mock | — | ❌ | 见缺失‑P01 |
| 管理员账户列表 / CRUD | mock | — | ❌ | 见缺失‑P02 |
| 角色 ↔ 权限矩阵分配 | mock | — | ❌ | 见缺失‑P03 |

---

## 三、缺失接口清单

> 含义：前端 UI 已存在或按钮已占位，但后端**完全没有对应接口**。建议补齐前先跟产品确认 v1 范围。

| 编号 | 模块 | 接口 | 用途 | 严重度 | 来源页面 |
|---|---|---|---|---|---|
| U01 | 资源 | `POST /uploads`（图片/媒体直传或预签名） | 商品图、头像、Banner 上传 | P1 | GoodsEdit |
| U02 | 用户 | `PUT /users/me` | 修改本人资料（昵称/邮箱/手机/头像） | P2 | Profile |
| U03 | 用户 | `PUT /users/me/password` | 修改密码 | P2 | Profile |
| U04 | 用户 | `PUT /users/me/preferences` | 通知/两步验证偏好 | P2 | Profile |
| M01 | 直播 | `POST /live-streams` | 创建直播间（Dashboard 已占位按钮） | P1 | Dashboard |
| L01 | 直播 | `PUT /admin/live-streams/:id/end` | 强制结束直播 | P2 | LiveDetail |
| L02 | 直播 | `PUT /admin/live-streams/:id/ban` | 直播间封禁/警告 | P2 | LiveDetail |
| L03 | 直播 | `GET /admin/live-streams` 增加 `status` 过滤 | 列表筛选 | P1 | LiveList |
| R01 | 拍卖规则 | `GET/POST/PUT/DELETE /auction-rule-templates[/:id]`<br>`POST /products/:id/apply-rule-template` | **规则模板**整套 CRUD + 应用到商品 | P1 | AuctionRules |
| O01 | 订单 | `POST /admin/orders/:id/remind-payment` | 催付（站内信/通知） | P2 | OrderDetail |
| O02 | 订单 | `POST /admin/orders/:id/remind-shipment` | 催发货（针对商家） | P2 | OrderDetail |
| O03 | 订单 | `GET /admin/orders/export` | 导出 CSV | P2 | OrderList |
| P01 | 权限 | `GET/POST/PUT/DELETE /admin/roles[/:id]` | 角色 CRUD | P1 | Permissions |
| P02 | 权限 | `GET/POST/PUT/DELETE /admin/users[/:id]` | 管理员账户 CRUD | P1 | Permissions |
| P03 | 权限 | `PUT /admin/roles/:id/permissions` | 角色↔权限点分配 | P1 | Permissions |
| S01 | 统计 | `GET /statistics/dashboard` | Dashboard 6 卡片专用聚合 | P1 | Dashboard |

---

## 四、字段/结构不匹配清单

> 含义：路由/接口已存在，但请求参数、响应结构或字段命名与前端不一致，**必须改一边或两边**才能正确工作。

### 4.1 列表响应包装：`items` vs `list`（统一性问题）

前端在 `frontend/admin/src/shared/api/index.ts` 全部解构 `{list, total, page, page_size}`，后端则返回 `{items, total, page, page_size}`。

| 编号 | 接口 | 后端实际返回 | 前端期望 | 处理建议 |
|---|---|---|---|---|
| F01 | `GET /products` | `{items, total, page, page_size}` | `{list, total, page, page_size}` | 后端改字段名为 `list`（成本最低） |
| F02 | `GET /orders` | `{items, total}` | `{list, total}` | 同上 |
| F03 | `GET /auctions` | `{items, total, page, page_size}` | `{list, total}` | 同上 |
| F04 | `GET /admin/live-streams` | `{list, total}` | `{list, total}` | ✅ 一致 |

**统一约定建议**：所有分页接口统一为 `{ list, total, page, page_size }`。

---

### 4.2 订单接口语义错位（P0 阻塞）

| 编号 | 接口 | 现状 | 问题 | 建议 |
|---|---|---|---|---|
| F05 | `GET /orders` | 强依赖 `X-User-ID`，仅返回**当前登录用户**的订单 | 管理端需要**全量订单**，进入页面会只看到 admin 自己的订单（多半为空） | 拆分：① C 端用 `GET /orders/my`；② 管理端 `GET /admin/orders`，可按 `user_id` 过滤；或保留 `/orders` 但根据 `RequireAdmin` 中间件返回全量 |

---

### 4.3 订单/拍卖详情字段缺失（P1）

| 编号 | 接口 | 后端缺字段 | 前端依赖 |
|---|---|---|---|
| F06 | `GET /orders/:id` | `product_name / product_image / user_name / paid_at / shipped_at / auction_id` | OrderDetail 直接使用，缺失会显示 "—" |
| F07 | `GET /orders` | 同上聚合字段 | OrderList 列表展示 |
| F08 | `GET /auctions/:id` | `product`（嵌套对象） / `winner_name` / `bid_count` | AuctionDetail / AuctionList |
| F09 | `GET /auctions` | `live_stream_name / bid_count` | AuctionList |
| F10 | `GET /auctions/:id/bids` | `bid.user_name` | AuctionDetail 出价记录列 |

**通用建议**：在 `Get/List` handler 内做联表（products / users / live_streams）后返回扁平视图对象 `OrderDetailVO / AuctionListItemVO`，避免前端二次拉取。

---

### 4.4 直播间字段命名（P1）

| 编号 | 接口 | 后端字段 | 前端字段 | 建议 |
|---|---|---|---|---|
| F11 | `GET /admin/live-streams` / `GET /live-streams/:id` | `host_name`（且为空字符串占位） | `streamer_name` | 后端联表 `users` 取主播名，并改字段名 `streamer_name / streamer_avatar` |
| F12 | 同上 | 缺 `viewer_count`（恒 0） | `viewer_count` | 接入运行时计数（Redis 计数器或 WS 在线统计） |
| F13 | 同上 | 缺 `auction_count` | `auction_count` | 联表 `auctions` 聚合 count |

---

### 4.5 统计接口结构错位（P1，运行时会抛错）

前端 `statisticsApi.getAuctionStats / getRevenueStats / getUserStats` 类型为 `any[]`，且页面使用 `.slice() / .map()` 数组操作；后端则返回**单对象**。

| 编号 | 接口 | 后端返回 | 前端期望 | 建议 |
|---|---|---|---|---|
| F14 | `GET /statistics/auctions` | `AuctionStatistics{ ... }` 单对象 | `Array<{date, auction_count, success_rate, bid_count}>` | 改为按 `start_date/end_date` + `group_by` 返回数组 |
| F15 | `GET /statistics/revenue` | `RevenueStatistics{ daily_revenue, category_distribution }` | `Array<{date, revenue, category, order_count}>` | 改为：`group_by=day/month` 返回 `[{date, revenue, order_count}]`；`group_by=category` 返回 `[{category, revenue}]` |
| F16 | `GET /statistics/users` | `UserStatistics{ ... }` 单对象 | `Array<{date, new_users, active_users}>` | 同 F14 |
| F17 | `GET /statistics/overview` | 缺 `ongoing_auctions / today_revenue / total_orders` | 6 项 KPI | 后端补字段，或新增 `GET /statistics/dashboard` 专用接口（见 S01） |

---

### 4.6 入参未识别（P2）

| 编号 | 接口 | 前端会传 | 后端处理 | 建议 |
|---|---|---|---|---|
| F18 | `GET /auctions` | `search / live_stream_name` | 未识别，被忽略 | 后端补 `WHERE`（按拍卖名/直播间名 LIKE） |
| F19 | `GET /admin/live-streams` | `status` | 未识别 | 后端补 `WHERE status=?` |
| F20 | `GET /statistics/revenue` | `group_by=day/month/category` | 未识别 | 与 F15 一并设计 |

---

### 4.7 方法不一致（P2，需复核）

| 编号 | 接口 | gateway 路由方法 | service 实际方法 | 风险 |
|---|---|---|---|---|
| F21 | `/orders/:id/pay` | `POST` | product `main.go` 中疑似 `PUT` | 复核 [product/main.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/main.go) 注册行；以 gateway 透传方法为准 |

---

## 五、整改优先级建议

按"修一处收益最大"排序：

1. **P0 - 一次解决全列表**：统一所有分页响应为 `{list, total, page, page_size}`（F01/F02/F03）。
2. **P0 - 订单语义**：拆分管理端 `GET /admin/orders`（F05）。
3. **P1 - 详情聚合 VO**：订单/拍卖/直播详情统一在后端联表，避免前端 N+1（F06–F13）。
4. **P1 - Dashboard 统计**：新增 `GET /statistics/dashboard` 专门服务首页 6 卡片（S01 + F17），同时把 `/statistics/{auctions,revenue,users}` 改为数组返回（F14/F15/F16）。
5. **P1 - 规则模板**：`auction-rule-templates` 模块从 0 → 1（R01）。
6. **P1 - 权限管理**：roles / admin-users / role-permissions（P01–P03）。
7. **P2 - 体验类**：直播控制（L01/L02）、订单催付催发（O01/O02/O03）、Profile 编辑（U02–U04）、媒体上传（U01）。

---

## 六、附录：后端接口现状速查（Gateway 暴露）

数据来源：[router.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go)

| 路由 | 方法 | 透传服务 | 当前 admin 端可用度 |
|---|---|---|---|
| `/auth/login`、`/auth/register` | POST | auction | ✅ |
| `/users/me`、`/users/me/stats`、`/users/me/addresses` | GET/...| auction | ✅（Profile 仅查询） |
| `/products`、`/products/:id`、`/products/:id/{publish,unpublish,rules}` | CRUD | product | ⚠️ list 字段需改 |
| `/auctions`、`/auctions/:id`、`/auctions/:id/{bids,ranking,result,cancel}` | CRUD | auction | ⚠️ 字段聚合不足 |
| `/orders`、`/orders/:id`、`/orders/:id/{pay,ship}`、`/orders/history` | CRUD | product | ❌ 管理端语义不正确 |
| `/live-streams`、`/live-streams/:id`、`/admin/live-streams`、`/live-streams/:id/{follow,notification}` | CRUD | product | ⚠️ 字段命名/控制接口缺失 |
| `/notifications`、`/notifications/{unread-count,read-all,:id/read}` | GET/PUT | auction | ✅ |
| `/statistics/{overview,auctions,revenue,users}` | GET | product | ⚠️ 单对象/字段缺失 |
| `/categories` | GET | product | ✅ |

— 报告完 —

# H5 新移动端迁移后端接口缺口核对

## 判定原则

新移动端 UI 需要展示或触发、但旧 H5 service wrapper、现有页面调用或后端契约无法明确支撑的能力，记录为缺口。缺口不等同于“完全没有后端代码”，需要按真实状态拆分：

| 分类 | 判定标准 | 处理原则 |
| --- | --- | --- |
| 真缺后端能力 | 未发现数据库表、model、DAO、handler、路由或 WS 协议 | 后端补接口/模型；前端只展示空态或禁用按钮 |
| 已有但契约不足 | 后端有接口或数据结构，但字段、过滤参数、鉴权语义、Gateway 暴露不满足 UI | 明确扩展契约；前端 adapter 只兼容真实字段 |
| 前端未接入/页面缺失 | 后端能力已存在，但 H5 页面或 service 未接入 | 前端补接入或新增页面，不把它记录为后端缺失 |

前端迁移阶段只能使用明确安全的降级，不用 mock 数据掩盖领域数据缺失。

## 核对结论总览

| 状态 | 能力 | 结论 |
| --- | --- | --- |
| 真缺后端能力 | 收藏列表、收藏筛选 | 未发现通用收藏表、model、DAO、handler 或路由；直播间关注不能等价为商品/竞拍收藏。 |
| 真缺后端能力 | 直播聊天消息列表/发送 | 未发现 `chat_message`、`send_chat` WS 协议，也未发现 `GET/POST /live-streams/:id/messages`。 |
| 真缺后端能力 | 用户统计 | 未发现 `GET /api/v1/users/me/stats`；现有统计多为直播间/管理统计，不是 Profile 所需字段。 |
| 真缺后端能力 | 钱包余额、充值、保证金 | 未发现用户余额、钱包、保证金表或后端路由。 |
| 真缺后端能力 | 收货地址、用户设置 | 未发现地址、收货人或用户设置相关 model、表或路由。 |
| 已有但契约不足 | 分类 Tab | 后端已有 `categories` 表、`products.category_id` 和 Product Service `/categories`；但 Gateway 未暴露 `/categories`，`GET /auctions` 不支持 `category/category_id` 过滤。 |
| 已有但契约不足 | 直播间详情 | `GET /api/v1/live-streams/:id` 已存在，但字段缺 `host_name/host_avatar/viewer_count/video_url/is_following`。 |
| 已有但契约不足 | 关注状态 | 已有 follow/unfollow/followed-list，但无单直播间 `follow-status`，详情也不返回 `is_following`。 |
| 已有但契约不足 | 商品规则 | 已有 `GET /api/v1/products/:id/rules`，但 `GET /products/:id` 不内嵌 `rules`；规则 handler 仍有 product_id/auction_id 临时映射风险。 |
| 已有但契约不足 | 竞拍结果 | `GET /api/v1/auctions/:id/result` 已存在，但缺 `order_id/product/won_bid`。 |
| 已有但契约不足 | 竞拍历史 | `GET /api/v1/orders/history` 已存在，但仍用 `user_id` query，且缺商品图片、`my_highest_bid`、`ended_at`、`product`。 |
| 已有但契约不足 | 订单支付 | Gateway 注册 `POST /orders/:id/pay`，Product Service 实际注册 `PUT /orders/:id/pay`，方法不一致。 |
| 前端未接入/页面缺失 | Home 通知未读数 | 后端 `GET /api/v1/notifications/unread-count` 已存在；Home 未接入真实未读数。 |
| 前端未接入/页面缺失 | 订单详情页 | 后端 `GET /api/v1/orders/:id` 已存在；H5 缺 `/order` 页面或订单通知承接页。 |
| 已确认可用 | Following 列表/取消关注 | 后端已有 `GET /api/v1/user/followed-live-streams` 和 `DELETE /api/v1/live-streams/:id/follow`。 |
| 已确认可用 | Notifications 列表/已读 | 后端已有通知列表、未读数、单条已读、全部已读；跳转 ID 属于 `data` 字段生产契约。 |
| 已确认可用 | Login 手机号登录 | 后端 `POST /api/v1/auth/login` 支持 `phone/password`，返回 `data.token` 和 `data.user`。 |

## 真缺后端能力

| 页面 | 新 UI 需求 | 当前核对结果 | 缺口 | 建议接口/契约 | 前端临时策略 |
| --- | --- | --- | --- | --- | --- |
| `Home` | 收藏 Tab 展示用户收藏竞拍 | 未发现通用收藏表、model、DAO、handler 或路由 | 缺收藏业务能力；直播间关注不是商品/竞拍收藏 | `GET /api/v1/users/me/favorites?page=&page_size=` 或 `GET /api/v1/auctions?favorite_only=true`；需定义收藏对象是 `product`、`auction` 还是 `live_stream` | 收藏 Tab 显示空态和“待开放”，不能用全量列表冒充收藏。 |
| `LiveRoom` | 聊天消息列表和发送聊天 | 当前 WS 消息类型只有 `ping/sync/bid/rank/notification/sky_lamp` 等，未发现聊天协议 | 缺聊天消息协议、持久化和发送接口 | WS 增加 `chat_message`、`send_chat`，或 HTTP 增加 `GET/POST /api/v1/live-streams/:id/messages` | 聊天输入禁用或提示“聊天协议尚未开放”，不展示静态假聊天。 |
| `Profile` | 关注数、粉丝数、竞拍记录数 | 未发现用户维度 stats 接口；现有 stats 主要是直播间热度或管理统计 | 缺统一用户统计接口 | `GET /api/v1/users/me/stats` 返回 `following_count`、`followers_count`、`auction_history_count` | 显示 `---` 或仅展示真实可得订单条数，不拼凑统计。 |
| `Profile` | 钱包余额、可用余额、充值、保证金 | 未发现钱包、余额、保证金表或路由 | 缺资产账户能力和充值/保证金接口 | `GET /api/v1/user/balance`；充值、保证金需单独定义资金流和支付契约 | 显示“暂不可用”，充值/保证金入口禁用。 |
| `Profile` | 收货地址、设置 | 未发现地址、收货人、用户设置相关模型或路由 | 缺用户地址和设置能力 | 地址：`/api/v1/users/me/addresses`；设置接口按产品项定义 | 入口隐藏或标记待开放，不写假地址。 |

## 已有但契约不足

| 页面 | 新 UI 需求 | 当前核对结果 | 契约缺口 | 建议接口/契约 | 前端临时策略 |
| --- | --- | --- | --- | --- | --- |
| `Home` | 分类 Tab：`珠宝腕表/艺术品/奢侈品/收藏品` | 后端已有 `categories` 表、`products.category_id` 和 Product Service `/api/v1/categories`；Gateway 未暴露 `/categories`；`GET /api/v1/auctions` 只支持 `status/live_stream_id/live_stream_name/search/page/page_size` | 不是缺分类表，而是缺 Gateway 分类路由和竞拍列表按商品分类过滤 | Gateway 增加 `GET /api/v1/categories`；`GET /api/v1/auctions?category_id=` 通过 `auctions.product_id -> products.category_id` 过滤，或返回内嵌 `product.category_id/category` 供前端安全筛选 | 非收藏 Tab 可展示全部或仅按已补齐的商品分类字段前端筛选；不能声称后端已支持分类筛选。 |
| `Home` | 首页卡片直接展示商品名称和首图 | `auctionApi.list` 返回竞拍字段；`Product` 有 `images/category_id`；页面可按 `product_id` 补 `productApi.get` | 列表接口未稳定内嵌 `product`，导致 N+1 请求或卡片字段不完整 | `GET /api/v1/auctions` 返回必要商品摘要：`product: { id, name, images, category_id }` | 页面优先读 `auction.product`，缺失时按 `product_id` 补详情；失败显示竞拍空态，不写 mock 图。 |
| `LiveRoom` | 直播间详情展示主播、头像、在线人数、封面/视频源、关注状态 | `GET /api/v1/live-streams/:id` 已存在，但当前只返回 `id/name/description/cover_image/status/creator_id/created_at` | 缺 `host_name`、`host_avatar`、`viewer_count`、`video_url`、`is_following` | 详情返回 `host_name`、`host_avatar`、`viewer_count`、`cover_image`、`video_url`、`status`、`is_following` | adapter 兼容已有字段；缺视频和封面时显示“暂无直播画面”，不使用外部占位图。 |
| `LiveRoom` | 关注按钮状态和切换 | 已有 `POST/DELETE /api/v1/live-streams/:id/follow` 和关注列表；没有单直播间状态查询 | 缺 `is_following` 查询契约 | `GET /api/v1/live-streams/:id/follow-status`，或直播详情在认证态下返回 `is_following` | 没有状态前不默认点亮关注；切换失败回滚 UI。 |
| `LiveRoom` | 出价排行实时同步 | `bidApi.getRanking`、`auctionApi.getBids`、WS 出价/排行消息存在 | WS `rank_update`、`bid_placed`、`sync_response` 字段结构需要固定前端类型 | WS 固定推送 `{ ranking, current_price, status, end_time }`；HTTP 排行固定 `items/list` 之一 | 页面 adapter 兼容 `ranking/bids/list/items`，不制造排行。 |
| `LiveRoom` | 单直播间多商品竞拍浮层 | 首页入口是单个 `auction_id`；竞拍列表支持 `live_stream_id` 过滤 | 不算完全缺失，但缺明确直播间聚合接口和 UI 契约 | 明确使用 `GET /api/v1/auctions?live_stream_id=`，或新增 `GET /api/v1/live-streams/:id/auctions` 返回直播间竞拍商品摘要 | 当前页面按单 `auction_id` 展示；多商品浮层等契约明确后再恢复。 |
| `ProductDetail` | 商品规则 | 已有 `GET /api/v1/products/:id/rules`；`GET /products/:id` 不返回 `rules`；规则 handler 注释显示临时使用 `product_id` 作为 `auction_id` | 不是缺接口，而是规则归属和详情内嵌契约不清 | 明确规则属于 `product` 还是 `auction`；如果 UI 需要商品详情一次返回，则 `GET /products/:id` 内嵌 `rules` | 优先调用 `productApi.getRules(productId)`；缺规则时只使用安全默认加价幅度，不伪造业务规则。 |
| `ProductDetail` | 分享按钮 | 无分享目标、埋点、口令或后端分享记录契约；浏览器 Web Share API 可选 | 分享业务动作未定义 | 如需后端记录，定义 `POST /api/v1/share-events`；否则明确纯前端 Web Share | 展示“分享暂未开放”或只启用浏览器原生分享，不假装已分享成功。 |
| `AuctionResult` | 权威展示中标者、成交价、商品和订单 | `GET /api/v1/auctions/:id/result` 已存在，返回 `auction_id/product_id/status/final_price/winner_id/started_at/ended_at/delay_used` | 缺 `order_id`、`product`、`won_bid`；页面仍需补商品详情，无法直接支付订单 | 结果接口返回 `winner_id`、`final_price`、`order_id`、`product`、`won_bid`；或提供按 `auction_id` 查订单能力 | 页面优先使用 result 权威字段；只用 `productApi.get` 补展示，不长期从最高出价推断中标。 |
| `AuctionResult` | 中标后支付订单 | 后端有 `orders` 表、`GET /orders/:id`、支付 service；但 Gateway 是 `POST /orders/:id/pay`，Product Service 是 `PUT /orders/:id/pay` | 支付方法不一致；`POST /orders` 创建路由未注册；result 不返回 `order_id` | 统一支付方法，建议 Gateway 和 Product 都支持同一方法；竞拍结束后订单创建链路需明确，result 返回 `order_id` | 无 `order_id` 时按钮显示“订单待生成”并禁用，不伪造订单。 |
| `AuctionHistory` | 用户竞拍历史 | `GET /api/v1/orders/history` 已存在，返回 `auction_id/product_name/final_price/is_winner/bid_count/created_at` | 仍要求 `user_id` query，不符合 JWT 用户语义；缺商品图片、`my_highest_bid`、`ended_at`、`product` | 从 JWT/Gateway 身份获取用户；扩展返回 `product`、`my_highest_bid`、`ended_at` | 页面只展示真实可得字段；缺图片用本地空态，不随机金额。 |
| `Following` | 关注直播间列表展示主播、观看人数、竞拍数 | `GET /api/v1/user/followed-live-streams` 已存在，返回 `data.items` | 基础接口存在，但直播字段如 `host_avatar/viewer_count/title/auction_count` 可能不完整 | 关注列表项统一返回直播间卡片摘要字段 | adapter 兼容多响应形态；缺封面/头像时显示本地空态。 |
| `Notifications` | 通知跳转到直播、竞拍结果、详情、订单 | 通知模型有 `data JSON`；通知列表/已读接口存在 | 直接字段和 `data` 字段未统一；订单详情页缺失时不能跳死链 | 统一 `data.live_stream_id`、`data.auction_id`、`data.order_id`；或拉平成直接字段 | 有 ID 才跳转；缺 ID 只展示内容；订单通知提示订单详情页待开放。 |

## 前端未接入或页面缺失

| 页面 | 新 UI 需求 | 后端核对结果 | 前端缺口 | 建议处理 | 临时策略 |
| --- | --- | --- | --- | --- | --- |
| `Home` | 通知入口展示真实未读红点 | 后端已有 `GET /api/v1/notifications/unread-count`，Gateway 已暴露 | Home 未接入真实未读数 | 在 Home 接入 `notificationApi.getUnreadCount()`，仅 `count > 0` 显示红点 | 不写死静态红点。 |
| `AuctionResult` / `Notifications` / `Profile` | 查看订单详情 | 后端已有 `GET /api/v1/orders/:id`；H5 service 有 `orderApi.get` | 缺用户端 `/order` 页面或通知订单承接页 | 新增 `OrderDetail` 页面，或结果页直接承接支付/订单状态 | 没有页面前按钮禁用或跳 `/history`，不生成死链。 |
| `Following` | 取消关注按钮 | 后端已有 `DELETE /api/v1/live-streams/:id/follow` | 新 UI 原始设计没有取消关注按钮，属于 UI 行为缺口 | 在卡片操作区保留明确“取消关注”按钮，成功后从列表移除 | 失败保留原列表并提示错误。 |
| `Login` | 手机号/密码登录和 redirect 回跳 | 后端 `POST /api/v1/auth/login` 支持 email 或 phone，并返回 `{ code, message, data: { token, user } }` | 后端不缺；前端需保持 token key 和 401 redirect 一致 | 页面发送归一化手机号和密码；401 使用 `redirect` 回跳 | 登录失败展示服务端 `message` 或安全错误文案。 |

## 已确认后端可用能力

| 能力 | 后端现状 | 注意事项 |
| --- | --- | --- |
| 通知列表/未读/单条已读/全部已读 | Gateway 已暴露 `GET /notifications`、`GET /notifications/unread-count`、`PUT /notifications/:id/read`、`PUT /notifications/read-all` | 需要统一分页响应和通知 `data` 字段。 |
| 关注/取消关注/关注列表 | Gateway 已暴露 `POST /live-streams/:id/follow`、`DELETE /live-streams/:id/follow`、`GET /user/followed-live-streams` | 缺单个直播间关注状态查询。 |
| 商品详情 | Product Service 和 Gateway 已暴露 `GET /products/:id` | 详情不内嵌规则；规则需单独查。 |
| 商品规则 | Product Service 和 Gateway 已暴露 `GET /products/:id/rules` | product_id/auction_id 语义需修正。 |
| 竞拍详情/出价/排行/结果 | Auction Service 和 Gateway 已暴露相关接口 | result 字段不满足支付和商品展示闭环。 |
| 订单列表/详情 | Product Service 和 Gateway 已暴露 `GET /orders`、`GET /orders/:id` | H5 订单详情页缺失。 |
| 手机号登录 | Auth handler 支持 `phone/password` | 认证 token key 需统一为 `auth_token/auth_user`。 |

## 响应契约待统一

| 能力 | 现状 | 风险 | 建议 |
| --- | --- | --- | --- |
| 分页字段 | 页面和接口混用 `list/items/data.list/data.items` | 页面迁移时容易空列表 | 建立 H5 adapter，统一读 `list ?? items ?? data.list ?? data.items`。 |
| Gateway 暴露 | Product Service 有 `/categories`，Gateway 未注册；所有 H5 HTTP 应走 Gateway | 前端不能安全直连下游服务 | Gateway 补齐 H5 需要的只读路由，避免绕过统一鉴权、实验上下文和错误处理。 |
| 订单支付方法 | Gateway 使用 `POST /orders/:id/pay`，Product Service 使用 `PUT /orders/:id/pay` | 支付请求经 Gateway 可能 404/405 | 统一为一个方法，并同步前端 wrapper、Swagger 和测试。 |
| 用户身份语义 | `orders/history` 仍读取 `user_id` query | 用户可越权查询其他用户历史 | 改为从 JWT/Gateway 注入身份读取用户 ID。 |
| 出价接口命名 | 新源使用 `auctionApi.placeBid`，旧 H5 service 同时有 `auctionApi.bid` 和 `bidApi.placeBid` | 重复 wrapper 容易接错路径 | 统一保留一个页面级调用入口。 |
| 登录 token key | 旧 `api.ts` 使用 `token/userInfo`，旧 `auth.ts` 使用 `auth_token/auth_user` | 认证态不一致导致 401 或页面误判未登录 | 只保留 `store/authContext` 约定，并让 request 读取同一 token key。 |
| 直播间列表参数 | 新源 `liveStreamApi.list({ page, page_size })`，旧 H5 `liveStreamApi.list(page, pageSize)` | 直接迁移会类型/运行时不兼容 | 统一 service 签名或做兼容重载。 |

## 安全降级规则

| 场景 | 允许做法 | 禁止做法 |
| --- | --- | --- |
| 接口缺失 | 展示空态、禁用按钮、写入本文档 | 写硬编码 mock 数据冒充真实业务数据。 |
| 字段缺失 | adapter 映射已有字段，并记录契约差异 | 用随机数或固定文案制造用户资产/竞拍结果。 |
| 页面缺失 | 新增可达路由并保留旧源码 | 生成死链或直接删除仍被引用的旧页面。 |
| Gateway 未暴露 | 记录为契约缺口并等待 Gateway 补路由 | 前端绕过 Gateway 直连下游服务。 |

## 后续优先级

| 优先级 | 事项 | 原因 |
| --- | --- | --- |
| P0 | 统一 `orders/:id/pay` 的 Gateway/Product 方法 | 当前支付链路可能不可用。 |
| P0 | `orders/history` 改为 JWT 用户语义 | 防止越权查询历史记录。 |
| P1 | Gateway 暴露 `/api/v1/categories`，并为 `GET /auctions` 增加分类过滤或商品摘要 | 首页分类 Tab 的真实数据来源。 |
| P1 | 扩展 `GET /live-streams/:id` 和 `GET /auctions/:id/result` 字段 | LiveRoom 和 Result 页面核心展示闭环。 |
| P1 | 补用户 stats、钱包/保证金、收藏、地址接口 | Profile 资产和用户中心能力。 |
| P2 | 定义直播聊天协议和分享契约 | 互动能力，不应使用假消息或无动作按钮。 |
| P2 | 新增 H5 `OrderDetail` 页面 | 后端订单详情已具备，缺用户端承接页。 |

# H5 新移动端迁移缺失接口盘点

## 判定原则

新移动端 UI 需要展示或触发、但旧 H5 service wrapper、现有页面调用或后端契约无法明确支撑的能力，记录为缺失接口或契约缺口。前端迁移阶段只能使用明确安全的降级，不用 mock 数据掩盖领域数据缺失。

## 初始缺口

| 页面 | 新 UI 需求 | 当前可用能力 | 缺口 | 建议接口/契约 | 前端临时策略 |
| --- | --- | --- | --- | --- | --- |
| `Home` | 分类 Tab：`全部`、`收藏`、`珠宝腕表`、`艺术品`、`奢侈品`、`收藏品` | `auctionApi.list` 支持 `status/page/page_size`，新源代码注释说明后端暂不支持分类筛选 | 缺分类筛选与收藏列表 | `GET /api/v1/auctions?category=&favorite_only=` 或独立收藏列表接口 | 非收藏 Tab 暂显示全部；收藏 Tab 显示空态并标注待开放。 |
| `Home` | 通知红点/未读状态 | `notificationApi.getUnreadCount` 在旧通知 service 中存在 | H5 页面未接入通知未读数 | 复用 `GET /api/v1/notifications/unread-count`，统一响应 `{ count }` | 首轮可静态红点或接入 hook，不能写死有未读。 |
| `LiveRoom` | 直播间详情包含主播、头像、在线人数、封面/视频源 | `liveStreamApi.get` 可获取直播间详情，但字段是否包含 `host_avatar/host_name/viewer_count/video_url` 未确认 | 直播详情字段契约不明确 | `GET /api/v1/live-streams/:id` 返回 `host_name`、`host_avatar`、`viewer_count`、`cover_image`、`video_url`、`status` | 缺字段时显示安全空态或商品图，不使用外部占位图。 |
| `LiveRoom` | 聊天消息列表和发送聊天 | 旧 `WebSocketService` 支持通用消息、通知、排行、出价同步 | 没有明确聊天消息协议和发送接口 | WebSocket 增加 `chat_message`、`send_chat` 协议，或 `GET/POST /live-streams/:id/messages` | 聊天输入先禁用或仅保留 UI，不发送假消息。 |
| `LiveRoom` | 分享、更多菜单、关注按钮 | 旧 `followApi` 支持关注直播间，但新 `LiveRoom` 当前心形按钮无逻辑 | 分享/更多无接口；关注状态和切换契约需确认 | `GET /live-streams/:id/follow-status` 或详情返回 `is_following`；分享使用 Web Share API 可无后端 | 分享/更多先无操作或隐藏；关注需对接后再启用。 |
| `ProductDetail` | 商品详情中展示 `product.rules` | 旧 `productApi.get` 与新源 `productApi.getRules` 能力不一致 | `productApi.get` 是否内嵌 `rules` 不明确 | `GET /api/v1/products/:id` 返回 `rules`，或迁移时补 `productApi.getRules(productId)` | 优先补前端 adapter；缺规则时以服务端兜底规则展示并记录。 |
| `AuctionResult` | 判断当前用户是否中标 | 新源代码从 `getBids` 推断最高价；旧 `getResult` 返回 `winner_id` | 结果页权威中标判定接口未统一 | 优先 `GET /api/v1/auctions/:id/result` 返回 `winner_id`、`final_price`、`order_id`、`product`、`won_bid` | 不使用 `wonBid.user_id === user.id` 作为长期方案；首轮可前端临时适配。 |
| `AuctionResult` | 查看订单入口 `/order?id=` | 旧 H5 无订单详情页路由，service 有 `orderApi.get/pay` | 用户端订单详情页缺失 | 新增 `OrderDetail` 页面或调整结果页直接承接支付/订单状态 | 若无订单页，按钮禁用或跳回历史页并记录。 |
| `Profile` | 关注数、粉丝数、竞拍记录数 | 旧 `User/Index` 有 `/api/v1/users/me/stats` | 新 UI 所需统计字段与旧 stats 字段不一致 | `GET /api/v1/users/me/stats` 返回 `following_count`、`followers_count`、`auction_history_count` | 先显示 `---`，不伪造统计。 |
| `Profile` | 钱包余额、充值 | 旧 service 有 `userApi.getBalance`，但后端可用性未确认 | 余额和充值能力未确认 | `GET /api/v1/user/balance`；充值需支付/钱包接口 | 显示“暂不可用”，充值按钮禁用或提示待开放。 |
| `Profile` | 我的收藏、保证金、收货地址、设置 | 现有 H5 未发现对应用户端 service | 缺完整用户资产/地址/设置接口 | 收藏：`GET /api/v1/users/me/favorites`；保证金：`GET /api/v1/users/me/deposits`；地址：`/addresses`；设置：待定义 | 入口标记暂未开放，不接入假数据。 |
| `AuctionHistory` | 用户参与的竞拍历史，区分成功/失败和我的出价 | 历史接口文档已提供 `GET /api/v1/orders/history?page=&page_size=`，后端返回 `auction_id`、`product_name`、`final_price`、`is_winner`、`bid_count`、`created_at` | 缺商品图片、`my_highest_bid`、`ended_at` 等新 UI 完整字段；后端实现的 `user_id` 查询参数与文档 JWT 语义需统一 | 建议扩展 `/api/v1/orders/history` 返回 `product`、`my_highest_bid`、`ended_at`，并以 JWT 用户身份为准 | 迁移时已移除 `Math.random()` 类展示，页面只展示真实可得字段。 |
| `Following` | 用户关注直播间列表 | 旧 `followApi.getFollowedLiveStreams` 存在；新源临时用 `liveStreamApi.list` | 新旧 wrapper 命名与响应字段不统一 | 统一到 `GET /api/v1/user/followed-live-streams?page=&page_size=` 返回 `list/items` 之一并建 adapter | 使用旧 `followApi`，不要用全量直播列表冒充关注。 |
| `Following` | 取消关注 | 旧 `followApi.unfollowLiveStream` 存在 | 新 UI 当前没有取消关注按钮 | 非接口缺失，是 UI 行为缺口 | 迁移时按新 UI 增加明确按钮，或记录产品确认。 |
| `Notifications` | 通知列表跳转需要 `live_stream_id`、`auction_id`、`order_id` | 旧 `notificationApi.list` 存在 | 通知 data 字段与新页面直接字段不统一 | 统一通知响应字段，或 adapter 从 `notification.data` 映射实体 ID | 无 ID 时通知不可跳转，只展示内容。 |
| `Login` | 手机号/密码登录和 redirect 回跳 | 旧 `authService` 偏 email；旧页面表单同时传 email/phone | 手机号登录是否为主契约需确认 | `POST /api/v1/auth/login` 支持 `{ phone, password }` 并返回 `{ token, user }` | 若后端只支持 email，需要产品确认是否改 UI 或增加账号字段。 |

## 响应契约待统一

| 能力 | 现状 | 风险 | 建议 |
| --- | --- | --- | --- |
| 分页字段 | 新源代码多用 `response.list`，旧 `notification.ts` 用 `items`，旧 `Follow` 用 `response.data.items` | 页面迁移时容易空列表 | 建立 H5 adapter，统一读 `list ?? items ?? data.items`。 |
| 出价接口命名 | 新源使用 `auctionApi.placeBid`，旧 H5 service 同时有 `auctionApi.bid` 和 `bidApi.placeBid` | 重复 wrapper 容易接错路径 | 统一保留一个页面级调用入口。 |
| 登录 token key | 旧 `api.ts` 使用 `token/userInfo`，旧 `auth.ts` 使用 `auth_token/auth_user` | 认证态不一致导致 401 或页面误判未登录 | 迁移时只保留 `store/authContext` 的约定，并让 request 读取同一 token key。 |
| 直播间列表参数 | 新源 `liveStreamApi.list({ page, page_size })`，旧 H5 `liveStreamApi.list(page, pageSize)` | 直接迁移会类型/运行时不兼容 | 迁移前统一 service 签名或做兼容重载。 |

## 安全降级规则

| 场景 | 允许做法 | 禁止做法 |
| --- | --- | --- |
| 接口缺失 | 展示空态、禁用按钮、写入本文档 | 写硬编码 mock 数据冒充真实业务数据。 |
| 字段缺失 | adapter 映射已有字段，并记录契约差异 | 用随机数或固定文案制造用户资产/竞拍结果。 |
| 页面缺失 | 新增可达路由并保留旧源码 | 直接删除旧页面或后端接口。 |


## Task 3 Home 迁移确认

| 页面 | 新 UI 需求 | 当前可用能力 | 缺口 | 前端临时策略 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `Home` | 首页卡片直接展示商品名称和首图 | `auctionApi.list` 返回竞拍字段；`productApi.get(product_id)` 可补商品详情 | 列表接口未稳定内嵌 `product`，需要额外并发请求商品详情 | 页面级 adapter 优先使用 `auction.product`，缺失时按 `product_id` 调 `productApi.get`；商品获取失败只展示竞拍场次空态，不写 mock 图片 | `frontend/h5/src/pages/Home/index.tsx` |
| `Home` | 分类 Tab 按 `珠宝腕表/艺术品/奢侈品/收藏品` 筛选 | 当前后端竞拍列表未确认支持 `category` 参数 | 分类筛选契约缺失 | 若商品返回 `category/category_name` 则前端可筛选；缺分类字段时非收藏 Tab 暂显示全部，保留既有缺口记录 | `frontend/h5/src/pages/Home/index.tsx` |
| `Home` | 收藏 Tab 展示用户收藏竞拍 | 当前 H5 无收藏竞拍列表接口 | 收藏列表接口缺失 | 收藏 Tab 显示空态和“待开放”说明，不用全量列表冒充收藏 | `frontend/h5/src/pages/Home/index.tsx` |
| `Home` | 通知入口展示真实未读红点 | 后续 `Notifications` 任务会统一通知接口 | Home 阶段未接入未读数 | 移除新源静态红点，只保留 `/notifications` 入口，避免伪造未读状态 | `frontend/h5/src/pages/Home/index.tsx` |

## Task 4 LiveRoom 迁移确认

| 页面 | 新 UI 需求 | 当前可用能力 | 缺口 | 前端临时策略 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `LiveRoom` | 直播间详情展示主播、头像、在线人数、封面/视频源、关注状态 | `liveStreamApi.get` 可获取直播间详情，`followApi.getFollowersStats` 可获取关注统计 | `GET /api/v1/live-streams/:id` 字段是否稳定包含 `host_name`、`host_avatar`、`viewer_count`、`cover_image`、`video_url`、`is_following` 未确认 | 通过 adapter 兼容 `host_name/creator_name`、`host_avatar/avatar`；缺视频和封面时显示“暂无直播画面”，不使用外部占位图 | `frontend/h5/src/pages/Live/index.tsx` |
| `LiveRoom` | 出价排行实时同步 | `bidApi.getRanking`、`auctionApi.getBids`、`WebSocketService` 可用 | WebSocket `rank_update`、`bid_placed`、`sync_response` 的字段结构未形成统一前端类型 | 页面使用 `extractList` 兼容 `ranking/bids/list/items`；长期建议后端固定推送 `{ ranking, current_price, status, end_time }` | `frontend/h5/src/pages/Live/index.tsx` |
| `LiveRoom` | 直播聊天消息列表和发送聊天 | 当前 `WebSocketService` 仅明确支持通用消息、通知、排行、出价同步 | 缺少 `chat_message`、`send_chat` 协议或 HTTP 聊天接口 | 保留“直播互动”入口并提示聊天协议尚未开放，不展示静态假聊天消息，不发送聊天 | `frontend/h5/src/pages/Live/index.tsx` |
| `LiveRoom` | 分享和更多菜单 | 当前无明确后端接口；浏览器可选 Web Share API 未做产品确认 | 分享目标、埋点、更多菜单动作缺失 | Task 4 不保留无动作按钮，避免用户可点但无结果 | `frontend/h5/src/pages/Live/index.tsx` |
| `LiveRoom` | 单直播间多商品竞拍浮层 | 当前首页入口传单个 `auction_id`；可用接口是单竞拍详情 | 缺直播间内竞拍商品聚合接口 | 当前页面按新源和入口语义迁移单 `auction_id`；如要恢复多商品浮层，建议新增 `GET /api/v1/live-streams/:id/auctions` | `frontend/h5/src/pages/Live/index.tsx` |

## Task 5 ProductDetail 迁移确认

| 页面 | 新 UI 需求 | 当前可用能力 | 缺口 | 前端临时策略 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `ProductDetail` | 商品详情页展示商品图文、竞拍状态、当前价、起拍价、出价记录和竞拍规则 | `auctionApi.get`、`auctionApi.getBids`、`productApi.get`、`bidApi.placeBid` 可支撑核心链路 | `product.rules` 是否稳定由 `GET /api/v1/products/:id` 返回仍未确认 | 页面 adapter 从 `product.rules`、`product.start_price/increment/cap_price`、`auction.start_price/increment/cap_price` 依次读取；缺失时只使用明确安全默认加价幅度 `100`，不伪造业务结果 | `frontend/h5/src/pages/ProductDetail/index.tsx` |
| `ProductDetail` | 分享按钮 | 当前无分享目标、埋点和后端接口契约 | 分享动作缺失 | 仅展示“分享暂未开放”占位，不触发假分享 | `frontend/h5/src/pages/ProductDetail/index.tsx` |

## Task 6 AuctionResult 迁移确认

| 页面 | 新 UI 需求 | 当前可用能力 | 缺口 | 前端临时策略 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `AuctionResult` | 权威展示竞拍结果、中标者、成交价和商品信息 | `auctionApi.getResult` 可读取 `/api/v1/auctions/:id/result`，`productApi.get` 可按 `product_id` 补商品详情 | `getResult` 响应字段需稳定包含 `winner_id`、`final_price`、`order_id`、`won_bid`；若缺少 `product` 仍需额外请求商品 | 页面优先使用 `getResult`，仅用 `productApi.get` 补商品展示；不再从前端最高出价长期推断中标结果 | `frontend/h5/src/pages/Result/index.tsx` |
| `AuctionResult` | 中标后支付订单 | `orderApi.pay(order_id)` 可触发模拟支付，前端已在有 `order_id` 时接入 | 当前没有用户端订单详情页；`orderApi.create` wrapper 存在但 Gateway/Product Service 未注册 `POST /orders` 创建路由 | 不伪造订单；无 `order_id` 时按钮显示“订单待生成”并禁用，订单详情页留给后续 `OrderDetail` 或历史页任务确认 | `frontend/h5/src/pages/Result/index.tsx` |

## Task 7 Profile 迁移确认

| 页面 | 新 UI 需求 | 当前可用能力 | 缺口 | 前端临时策略 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `Profile` | 展示用户资料、角色和 ID | `userApi.getProfile` 可读取 `/api/v1/user/profile`；`AuthContext` 可提供登录态用户兜底 | 用户头像、角色字段是否稳定返回仍需后端契约确认 | 优先展示接口用户资料；缺头像时用姓名首字兜底，不使用外部占位图 | `frontend/h5/src/pages/User/Index.tsx` |
| `Profile` | 钱包余额、可用余额、充值入口 | `userApi.getBalance` 可读取 `/api/v1/user/balance` | 充值/钱包流水接口缺失；余额字段 `balance/available_balance/frozen_amount` 需稳定 | 有余额则展示真实余额；充值按钮禁用并标注待开放，不伪造资产数据 | `frontend/h5/src/pages/User/Index.tsx` |
| `Profile` | 最近订单和“我的竞拍/订单”入口 | `orderApi.list` 可读取 `/api/v1/orders`；历史页 `/history` 已存在 | 用户端订单详情页仍缺失，订单列表响应字段可能是 `list/items/orders/data.items` 多形态 | 页面 adapter 兼容常见列表字段，最近订单点击跳 `/history`，不新增未确认订单详情路由 | `frontend/h5/src/pages/User/Index.tsx` |
| `Profile` | 关注数、粉丝数、竞拍记录数 | 当前仅有 `followApi.getFollowedLiveStreams` 和 `orderApi.list` | 缺统一用户统计接口：`following_count`、`followers_count`、`auction_history_count` | 统计数字继续显示 `---` 或订单条数，不用旧 `/users/me/stats` 字段冒充新 UI 统计 | `frontend/h5/src/pages/User/Index.tsx` |
| `Profile` | 收藏、保证金、地址、设置 | H5 现有 service 未提供完整用户资产/地址/设置能力 | 后端接口和产品交互未定义 | 仅保留设置待开放入口；收藏、保证金、地址不作为 Task 7 可用入口暴露 | `frontend/h5/src/pages/User/Index.tsx` |

## Task 8 AuctionHistory 迁移确认

| 页面 | 新 UI 需求 | 当前可用能力 | 缺口 | 前端临时策略 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `AuctionHistory` | 展示用户参与竞拍记录、成功/失败状态、最终成交价和参与信息 | 历史接口文档提供 `GET /api/v1/orders/history?page=&page_size=`；后端当前返回 `auction_id`、`product_name`、`final_price`、`is_winner`、`bid_count`、`created_at` 等字段 | 仍缺商品图片、`my_highest_bid`、`ended_at`、`product` 嵌套信息；后端实现当前还存在 `user_id` 查询参数与接口文档 JWT 语义不一致的风险 | 页面只展示真实可得字段：商品名、成功/未中标、出价次数、最终成交价；缺图片时显示本地空态，不使用随机金额或外部占位图 | `frontend/h5/src/pages/History/index.tsx`、`frontend/h5/src/services/api.ts` |
| `AuctionHistory` | 结果/详情入口 | 现有 `/result?id=` 和 `/detail?id=` 已可达 | 缺用户端订单详情页，不适合在历史页承接支付 | 成功记录跳结果页，未中标记录跳详情页；不新增订单详情路由，不恢复历史页支付弹窗 | `frontend/h5/src/pages/History/index.tsx` |

## Task 9 Following 迁移确认

| 页面 | 新 UI 需求 | 当前可用能力 | 缺口 | 前端临时策略 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `Following` | 展示当前用户关注的直播间列表、直播状态、主播、观看人数和竞拍数 | `followApi.getFollowedLiveStreams(page, pageSize)` 可读取 `GET /api/v1/user/followed-live-streams?page=&page_size=` | 后端分页响应字段仍可能返回 `list/items/data.list/data.items` 多形态；部分直播字段如 `host_avatar/viewer_count/title` 可能缺失 | 页面 adapter 兼容多响应形态；缺封面/头像时显示本地空态，不使用全量直播列表或 mock 数据冒充关注 | `frontend/h5/src/pages/Follow/index.tsx` |
| `Following` | 取消关注后列表原地更新 | `followApi.unfollowLiveStream(liveStreamId)` 可调用 `DELETE /api/v1/live-streams/:id/follow` | 新源 UI 没有取消关注按钮，属于行为缺口而非后端接口缺口 | 迁移时在卡片操作区增加明确“取消关注”按钮，成功后从列表移除；失败保留原列表并提示错误 | `frontend/h5/src/pages/Follow/index.tsx` |
| `Following` | 进入直播间 | 迁移后的 `LiveRoom` 目标路由是 `/live?id=` | 旧 Follow 页面跳 `/live/:id`，与新移动端路由不一致 | 进入直播间统一跳 `/live?id=${liveStreamId}`，保留 App 中旧 `/follow` 到 `/following` 兼容跳转 | `frontend/h5/src/pages/Follow/index.tsx`、`frontend/h5/src/App.tsx` |

## Task 10 Notifications 迁移确认

| 页面 | 新 UI 需求 | 当前可用能力 | 缺口 | 前端临时策略 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `Notifications` | 通知中心列表、未读数、单条已读、全部已读 | `notificationApi.list`、`getUnreadCount`、`markAsRead`、`markAllAsRead` 已存在 | 后端分页响应字段可能是 `items/list/data.items/data.list` 多形态 | 页面 adapter 兼容多响应形态，失败展示错误态，不写 mock 通知 | `frontend/h5/src/pages/Notifications/index.tsx` |
| `Notifications` | 开播、竞拍结果、竞拍提醒跳转到业务页面 | 通知 `data` 可携带 `live_stream_id`、`auction_id`；新 H5 已有 `/live?id=`、`/result?id=`、`/detail?id=` | 直接字段和 `data` 字段未统一，且缺字段时无法安全跳转 | 优先读取直接字段，其次读取 `notification.data`；缺 ID 时只展示内容，不生成空 ID 路由 | `frontend/h5/src/pages/Notifications/index.tsx` |
| `Notifications` | 订单通知跳转订单详情 | `notificationApi.list` 可返回订单类通知；`orderApi.get/pay` wrapper 存在 | 用户端没有 `/order` 或订单详情页路由，不能从通知中心跳死链 | 订单通知只展示内容并提示“订单详情页尚未开放”，长期建议新增 `OrderDetail` 页面或定义订单通知承接页 | `frontend/h5/src/pages/Notifications/index.tsx` |

## Task 11 Login 迁移确认

| 页面 | 新 UI 需求 | 当前可用能力 | 缺口 | 前端临时策略 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `Login` | 手机号/密码登录 | 旧页面已直接请求 `/api/v1/auth/login` 并可同时发送 `email/phone/password`；`AuthContext.setAuth` 负责写入 `auth_token/auth_user` | 后端需稳定支持 `{ phone, password }` 登录契约，并返回 `{ data: { token, user } }` | 页面只发送归一化手机号和密码；失败展示服务端 `message` 或安全错误文案，不降级到邮箱 UI | `frontend/h5/src/pages/Login/index.tsx` |
| `Login` | 登录后回到触发登录前页面 | 登录页可读取 `redirect` 参数；`api.ts` 可统一处理 401 | 401 跳转如不带当前路径会丢失用户上下文 | 新增 `buildLoginRedirectPath()` 构造 `/login?redirect=` 当前路径；登录成功后 `navigate(redirectUrl)` 回跳 | `frontend/h5/src/pages/Login/index.tsx`、`frontend/h5/src/services/api.ts` |

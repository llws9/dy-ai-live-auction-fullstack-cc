# H5 新移动端迁移冗余接口盘点

## 判定原则

新移动端 UI 是保留功能的准绳。旧 H5 页面中存在、但新移动端页面不再展示或不再触发的接口，先记录为冗余接口，不在页面迁移阶段直接删除 service、后端接口或旧源码。

## 初始盘点

| 页面 | 旧页面/来源 | 旧接口或能力 | 新 UI 是否需要 | 初始结论 | 后续动作 |
| --- | --- | --- | --- | --- | --- |
| `Home` | `frontend/h5/src/pages/Home/index.tsx` | `/api/v1/auctions` 的 `all/ongoing/ended` Tab 筛选 | 部分需要 | 新 `Home` 使用商品分类 Tab，旧 `ongoing/ended` Tab 可移除 | 迁移时仅保留 `auctionApi.list`，状态筛选若后续产品确认再恢复。 |
| `Home` | `frontend/h5/src/pages/Home/index.tsx` | 头部关注入口 `/follow`、历史入口 `/history` | 部分需要 | 入口位置冗余，不代表页面能力删除 | 按新 `MobileContainer` 底部导航和新页面入口重排。 |
| `LiveRoom` | `frontend/h5/src/pages/Live/index.tsx` | 旧 mock 直播间、mock 商品列表、mock 出价记录 | 不需要 | 冗余 | 不迁移 mock 数据；真实接口以 `Auction` 页面和 service 为准。 |
| `LiveRoom` | `frontend/h5/src/pages/Live/index.tsx` | 多商品同屏竞拍 mock 交互 | 不需要 | 待确认冗余 | 新 `LiveRoom` 当前围绕单个 `auction_id`；若产品仍要求单直播多商品，需要补后端聚合接口而不是复用 mock。 |
| `LiveRoom` | `frontend/h5/src/pages/Auction/index.tsx` | `/api/v1/auctions/:id` 直接 `fetch` | 需要能力，不需要调用方式 | 调用方式冗余 | 迁移为 `auctionApi.get`，不要继续页面内裸 `fetch`。 |
| `LiveRoom` | `frontend/h5/src/pages/Auction/index.tsx` | `/api/v1/auctions/:id/bids` 直接 `fetch` | 需要能力，不需要调用方式 | 调用方式冗余 | 迁移为 `auctionApi.getBids` 或统一 `bidApi`。 |
| `AuctionResult` | `frontend/h5/src/pages/Result/index.tsx` | `/api/v1/auctions/:id/result` 的简化结果页 | 部分需要 | 旧 UI 和简化中标判断冗余 | 新结果页以竞拍、商品、出价记录组装；是否改用 `auctionApi.getResult` 后续逐页确认。 |
| `AuctionResult` | `frontend/h5/src/pages/Result/index.tsx` | `alert('支付功能开发中...')` | 不需要 | 冗余 | 新 UI 通过订单入口跳转，不保留 alert 式支付占位。 |
| `Profile` | `frontend/h5/src/pages/User/Index.tsx` | `/api/v1/users/me/stats`、最近竞拍统计卡片 | 部分不需要 | 待确认冗余 | 新 `Profile` 只显示关注/粉丝/竞拍记录入口，统计数字当前为占位；若保留统计需补接口。 |
| `Profile` | `frontend/h5/src/pages/User/Index.tsx` | 功能菜单“我的消息”指向 `/notifications` | 需要能力，不需要旧菜单 UI | 旧菜单 UI 冗余 | 新页面已有功能列表和底部导航入口。 |
| `AuctionHistory` | `frontend/h5/src/pages/History/index.tsx` | `/api/v1/orders` 订单列表 | 可能不需要 | 待确认冗余 | 新 `AuctionHistory` 语义是用户参与竞拍记录，不等同订单列表。 |
| `AuctionHistory` | `frontend/h5/src/pages/History/index.tsx` | `/api/v1/orders/:id/pay` 页面内支付弹窗 | 新页面不需要 | 冗余 | 支付能力应由订单/结果链路承接，历史页不保留支付弹窗。 |
| `Following` | `frontend/h5/src/pages/Follow/index.tsx` | 搜索关注直播间名称 | 新页面当前不需要 | 待确认冗余 | 若产品要求搜索，再按新 UI 加搜索入口。 |
| `Following` | `frontend/h5/src/pages/Follow/index.tsx` | 加载更多分页按钮 | 新页面当前不需要 | 待确认冗余 | 新页面初版可先保留列表；分页能力可在接口对接时按数据规模决定。 |
| `Login` | `frontend/h5/src/pages/Login/index.tsx` | 注册切换 `/api/v1/auth/register` | 新 `Login` 不需要 | 待确认冗余 | 用户端登录页先按新 UI 只保留登录；注册入口是否舍弃需产品确认。 |
| `Login` | `frontend/h5/src/pages/Login/index.tsx` | 邮箱登录字段 | 新 `Login` 不需要 | 待确认冗余 | 新 UI 以手机号/密码为主；若后端必须邮箱登录，需要改新 UI 或做兼容。 |
| 全局 | `frontend/h5/src/App.tsx` | 启动时 mock 开播提醒弹窗 | 新 UI 不需要 | 冗余 | Task 2 应移除硬编码演示数据和自动弹窗。 |
| 全局 | `frontend/h5/src/services/skyLamp.ts` | 天灯订阅相关接口 | 新移动端目标页未出现 | 待确认冗余 | 暂不删除，等待完整页面迁移后确认是否还有入口。 |

## 逐页迁移时追加字段

| 字段 | 说明 |
| --- | --- |
| `确认结果` | `保留`、`冗余但保留 service`、`待产品确认`、`后端可删除候选`。 |
| `证据` | 新 UI 中缺少对应入口/按钮/状态，或被其他页面能力替代。 |
| `处理 PR/提交` | 记录实际移除页面调用的位置，不记录后端删除。 |

## 当前不删除的内容

| 内容 | 原因 |
| --- | --- |
| `frontend/h5/src/services/api.ts` 中未被新页面首轮使用的 wrapper | Task 1 只盘点；删除会影响后续页面确认和测试。 |
| 旧页面源码文件 | Spec 明确要求最终迁移完成后再询问是否删除旧移动端代码。 |
| 后端接口 | 冗余接口需要最终确认，不在前端页面迁移中直接删除。 |


## Task 3 Home 迁移确认

| 页面 | 旧页面/来源 | 旧接口或能力 | 确认结果 | 证据 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `Home` | `frontend/h5/src/pages/Home/index.tsx` | 页面内裸 `fetch('/api/v1/auctions')` 和 `data.auctions` 历史响应读取 | 冗余但保留后端接口 | 新首页改用 `auctionApi.list({ page, page_size })`，并通过 adapter 兼容 `list/items/auctions` 响应字段 | `frontend/h5/src/pages/Home/index.tsx` |
| `Home` | `frontend/h5/src/pages/Home/index.tsx` | 旧 `all/ongoing/ended` 状态 Tab UI | 冗余 | 新移动端首页使用 `全部/收藏/珠宝腕表/艺术品/奢侈品/收藏品` 分类 Tab，状态筛选不再作为首页主交互 | `frontend/h5/src/pages/Home/index.tsx` |
| `Home` | `frontend/h5/src/pages/Home/index.tsx` | 接口失败后的硬编码演示竞拍数据 | 冗余 | 新首页失败时展示空态，不用 mock 数据冒充真实业务数据 | `frontend/h5/src/pages/Home/index.tsx` |
| `Home` | `frontend/h5/src/pages/Home/index.tsx` | 旧首页整卡跳转 `/auction/:id` | 调用路径冗余 | 新 UI 拆分为 `详情`、`进入直播`、`查看结果` 三个显式按钮，目标路由分别为 `/detail?id=`、`/live?id=&auction_id=`、`/result?id=` | `frontend/h5/src/pages/Home/index.tsx` |

## Task 4 LiveRoom 迁移确认

| 页面 | 旧页面/来源 | 旧接口或能力 | 确认结果 | 证据 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `LiveRoom` | `frontend/h5/src/pages/Live/index.tsx` | 页面内硬编码 `mockLiveRoom.products` 和 `mock` 出价记录 | 冗余 | 新直播间以 URL `auction_id` 为入口，通过真实 `auctionApi`、`productApi`、`bidApi` 拉取竞拍、商品和排行；接口失败显示空态或降级，不写假数据 | `frontend/h5/src/pages/Live/index.tsx` |
| `LiveRoom` | `frontend/h5/src/pages/Live/index.tsx` | 外部演示视频和外部图片 URL 作为直播背景 | 冗余 | 新页面只使用接口返回的 `video_url`、商品图片或直播间封面；缺失时显示“暂无直播画面”安全空态 | `frontend/h5/src/pages/Live/index.tsx` |
| `LiveRoom` | `frontend/h5/src/pages/Live/index.tsx` | 旧多商品同屏竞拍 mock 交互 | 待产品确认 | 新源 `LiveRoom` 和当前首页入口均围绕单个 `auction_id`；如需“单直播间多商品同时竞拍”，应补后端聚合契约，而不是保留旧 mock | `frontend/h5/src/pages/Live/index.tsx` |
| `LiveRoom` | `frontend/h5/src/pages/Auction/index.tsx` | 页面内裸 `fetch('/api/v1/auctions/:id')` 和 `fetch('/api/v1/auctions/:id/bids')` 调用方式 | 调用方式冗余 | 新 `LiveRoom` 使用统一 `services/api.ts` wrapper：`auctionApi.get`、`bidApi.getRanking`，并仅在排名缺失时 fallback 到 `auctionApi.getBids` | `frontend/h5/src/pages/Live/index.tsx` |
| `LiveRoom` | 新源 `src/mobile/pages/LiveRoom.tsx` | 静态聊天消息示例 | 冗余 | 当前 H5 迁移保留“直播互动”入口，但不显示假聊天消息；聊天协议缺失已记录到缺失接口文档 | `frontend/h5/src/pages/Live/index.tsx` |

## Task 5 ProductDetail 迁移确认

| 页面 | 旧页面/来源 | 旧接口或能力 | 确认结果 | 证据 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `ProductDetail` | `frontend/h5/src/pages/Auction/index.tsx` | 页面内裸 `fetch('/api/v1/auctions/:id')` 和 `fetch('/api/v1/auctions/:id/bids')` 调用方式 | 调用方式冗余 | 新商品详情页使用 `auctionApi.get`、`auctionApi.getBids`、`productApi.get`、`bidApi.placeBid`，不再在新页面内直接裸 `fetch` | `frontend/h5/src/pages/ProductDetail/index.tsx` |
| `ProductDetail` | `frontend/h5/src/pages/Auction/index.tsx` | 旧视频背景、外部演示视频和直播化竞拍布局 | 冗余 | 新 `ProductDetail` 是商品图文详情和快捷出价页，不承载直播视频背景；直播能力已由 `LiveRoom` 承接 | `frontend/h5/src/pages/ProductDetail/index.tsx` |
| `ProductDetail` | `frontend/h5/src/pages/Auction/index.tsx` | 出价记录接口失败后的硬编码用户 A/B/C 假数据 | 冗余 | 新商品详情页接口失败时展示空出价记录，不用 mock 数据冒充真实业务数据 | `frontend/h5/src/pages/ProductDetail/index.tsx` |

## Task 6 AuctionResult 迁移确认

| 页面 | 旧页面/来源 | 旧接口或能力 | 确认结果 | 证据 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `AuctionResult` | `frontend/h5/src/pages/Result/index.tsx` | 页面内裸 `fetch('/api/v1/auctions/:id/result')` 调用方式 | 调用方式冗余 | 新结果页使用统一 `auctionApi.getResult` wrapper，保留 `/result?id=` 和 `/result/:id` 兼容入口 | `frontend/h5/src/pages/Result/index.tsx` |
| `AuctionResult` | `frontend/h5/src/pages/Result/index.tsx` | `winner_id === 1` 的简化中标判断 | 冗余 | 新结果页使用登录用户 `user.id` 与权威结果 `winner_id` 比对，不再硬编码用户 ID | `frontend/h5/src/pages/Result/index.tsx` |
| `AuctionResult` | `frontend/h5/src/pages/Result/index.tsx` | `alert('支付功能开发中...')` 占位支付 | 冗余 | 中标且存在 `order_id` 时直接调用 `orderApi.pay(order_id)`；无订单时禁用按钮并记录缺口 | `frontend/h5/src/pages/Result/index.tsx` |

## Task 7 Profile 迁移确认

| 页面 | 旧页面/来源 | 旧接口或能力 | 确认结果 | 证据 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `Profile` | `frontend/h5/src/pages/User/Index.tsx` | 页面内裸 `fetch('/api/v1/users/me')` 调用方式 | 调用方式冗余 | 新个人中心统一使用 `userApi.getProfile`，并通过 `api.ts` 读取 `auth_token` 作为认证来源 | `frontend/h5/src/pages/User/Index.tsx`、`frontend/h5/src/services/api.ts` |
| `Profile` | `frontend/h5/src/pages/User/Index.tsx` | `/api/v1/users/me/stats`、旧 `UserStats` 统计卡片和最近竞拍统计结构 | 冗余但保留后端接口待确认 | 新 `Profile` 展示入口型统计和最近订单，核心数据来自 `orderApi.list`；关注/粉丝/竞拍记录数缺统一契约，继续按缺失接口记录 | `frontend/h5/src/pages/User/Index.tsx` |
| `Profile` | `frontend/h5/src/pages/User/Index.tsx` | 旧功能菜单“我的订单/我的消息/退出登录”的内联样式 UI | UI 冗余 | 新页面改为移动端目标功能列表，入口对齐 `/history`、`/following`、`/notifications`，退出登录保留 `AuthContext.logout` | `frontend/h5/src/pages/User/Index.tsx` |

## Task 8 AuctionHistory 迁移确认

| 页面 | 旧页面/来源 | 旧接口或能力 | 确认结果 | 证据 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `AuctionHistory` | `frontend/h5/src/pages/History/index.tsx` | 页面内裸 `fetch('/api/v1/orders')` 订单列表 | 调用方式和语义冗余 | 新竞拍历史页改用 `orderApi.history` 请求接口文档中的 `/orders/history`；历史页展示用户参与竞拍记录，不再把订单列表当竞拍记录 | `frontend/h5/src/pages/History/index.tsx`、`frontend/h5/src/services/api.ts` |
| `AuctionHistory` | `frontend/h5/src/pages/History/index.tsx` | 接口失败后的硬编码订单 mock 数据和外部商品图片 | 冗余 | 新页面接口失败展示错误态或空态，不写 mock 数据、不引入外部占位图 | `frontend/h5/src/pages/History/index.tsx` |
| `AuctionHistory` | `frontend/h5/src/pages/History/index.tsx` | `/api/v1/orders/:id/pay` 页面内支付弹窗和模拟支付成功 | 冗余 | 新移动端竞拍记录只承接结果/详情入口；支付由结果页和订单链路处理，历史页不触发模拟支付 | `frontend/h5/src/pages/History/index.tsx` |

## Task 9 Following 迁移确认

| 页面 | 旧页面/来源 | 旧接口或能力 | 确认结果 | 证据 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `Following` | `frontend/h5/src/pages/Follow/index.tsx` | 搜索关注直播间名称 | 冗余但待产品确认 | 新移动端 `Following` 目标 UI 不包含搜索入口；首轮迁移聚焦关注列表、直播状态、进入直播间和取消关注 | `frontend/h5/src/pages/Follow/index.tsx` |
| `Following` | `frontend/h5/src/pages/Follow/index.tsx` | 加载更多分页按钮 | 冗余但保留接口分页能力 | 新移动端目标 UI 不展示加载更多按钮；页面仍按 `page=1&page_size=20` 读取关注列表，后续若产品要求无限滚动再恢复分页交互 | `frontend/h5/src/pages/Follow/index.tsx` |
| `Following` | 新源 `src/mobile/pages/Following.tsx` | 用 `liveStreamApi.list` 临时替代关注列表 | 冗余且禁止迁移 | H5 已存在真实 `followApi.getFollowedLiveStreams`；不能用全量直播列表冒充关注列表 | `frontend/h5/src/pages/Follow/index.tsx` |
| `Following` | `frontend/h5/src/pages/Follow/index.tsx` | 旧 `/live/:id` 跳转 | 调用路径冗余 | 新迁移链路的直播间入口使用 `/live?id=`，与 `LiveRoom` 目标路由对齐 | `frontend/h5/src/pages/Follow/index.tsx` |

## Task 10 Notifications 迁移确认

| 页面 | 旧页面/来源 | 旧接口或能力 | 确认结果 | 证据 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `Notifications` | `frontend/h5/src/App.tsx` | `/notifications` 内联占位页 | 冗余 | 新增真实通知中心页面，并通过 lazy route 渲染；占位文案不再用户可见 | `frontend/h5/src/App.tsx`、`frontend/h5/src/pages/Notifications/index.tsx` |
| `Notifications` | `frontend/h5/src/components/Notification/index.tsx` | 全局铃铛下拉通知 UI | UI 冗余但保留组件源码 | 新移动端目标是独立通知中心页；旧组件未作为 `/notifications` 页面复用，避免弹层和页面状态重复 | `frontend/h5/src/pages/Notifications/index.tsx` |
| `Notifications` | 新源 `src/mobile/pages/Notifications.tsx` | 订单通知跳 `/order?id=` | 路径冗余/待补页面 | 当前 H5 App 没有 `/order` 路由，迁移时不生成死链；订单承接页需后续产品确认 | `frontend/h5/src/pages/Notifications/index.tsx` |

## Task 11 Login 迁移确认

| 页面 | 旧页面/来源 | 旧接口或能力 | 确认结果 | 证据 | 处理位置 |
| --- | --- | --- | --- | --- | --- |
| `Login` | `frontend/h5/src/pages/Login/index.tsx` | 注册切换和 `/api/v1/auth/register` 页面调用 | 冗余但保留后端接口待确认 | 新移动端 `Login` 目标页只提供手机号/密码登录；Task 11 不暴露注册入口，也不删除注册接口或旧能力记录 | `frontend/h5/src/pages/Login/index.tsx` |
| `Login` | `frontend/h5/src/pages/Login/index.tsx` | 邮箱登录输入字段 | 冗余但待产品确认 | 新移动端目标 UI 以手机号作为主账号字段；页面请求体只发送 `{ phone, password }`，避免继续展示邮箱/手机号双入口造成契约歧义 | `frontend/h5/src/pages/Login/index.tsx` |
| `Login` | `frontend/h5/src/pages/Login/index.tsx` | 内联浅色卡片样式 | UI 冗余 | 登录页已迁移为移动端深色奢华风格，样式进入 CSS Module，沿用全局 mobile token | `frontend/h5/src/pages/Login/index.tsx`、`frontend/h5/src/pages/Login/Login.module.css` |

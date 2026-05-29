# H5 新移动端迁移页面盘点

## 目标

以 `/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-ui/src/mobile` 为新移动端 UI 的 Single Source of Truth，逐页替换 `frontend/h5` 的用户端页面。本文档只做 Task 1 的迁移盘点，不执行 UI 替换。

## 路由对比

| 新移动端页面 | 新移动端路由 | 旧 H5 对应页面 | 旧 H5 当前路由 | 目标 H5 路由 | 状态 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `Home.tsx` | `/` | `frontend/h5/src/pages/Home/index.tsx` | `/` | `/` | 已迁移 | 已替换为新移动端奢华竞拍首页；通过 `auctionApi.list` + `productApi.get` adapter 对接竞拍和商品字段，保留详情、直播、结果、关注、通知入口。 |
| `LiveRoom.tsx` | `/live` | `frontend/h5/src/pages/Live/index.tsx`、`frontend/h5/src/pages/Auction/index.tsx` | `/live`、`/auction/:id` | `/live?id=:liveStreamId&auction_id=:auctionId` | 已迁移 | 新直播间已接入 `/live`，旧 `/auction/:id` 不再渲染旧竞拍页，统一兼容跳转到 `/detail?id=`。 |
| `ProductDetail.tsx` | `/detail` | `frontend/h5/src/pages/Auction/index.tsx` 的部分能力 | 无独立路由 | `/detail?id=:auctionId` | 已迁移 | 已新增 H5 `ProductDetail` 页面，拆分商品、规则、出价记录和快捷出价能力；旧 `/auction/:id` 暂保留兼容。 |
| `AuctionResult.tsx` | `/result` | `frontend/h5/src/pages/Result/index.tsx` | `/result/:id` | `/result?id=:auctionId` | 已迁移 | 已替换为新移动端结果页结构，优先使用 `auctionApi.getResult` 权威结果，接入商品展示和中标订单支付。 |
| `Profile.tsx` | `/profile` | `frontend/h5/src/pages/User/Index.tsx` | 当前 `App.tsx` 未接入 | `/profile` | 新增保留 | 旧页面存在源码但不可达；迁移时需要接入路由与登录保护。 |
| `AuctionHistory.tsx` | `/history` | `frontend/h5/src/pages/History/index.tsx` | `/history` | `/history` | 已迁移 | 已替换为新移动端竞拍记录页，接入 `GET /api/v1/orders/history`，移除旧订单支付历史行为。 |
| `Notifications.tsx` | `/notifications` | `frontend/h5/src/services/notification.ts`、`frontend/h5/src/hooks/useNotification.ts` | 无页面路由 | `/notifications` | 新增保留 | 旧 H5 有通知 service/hook 和全局通知逻辑，但没有独立通知页。 |
| `Following.tsx` | `/following` | `frontend/h5/src/pages/Follow/index.tsx` | `/follow` | `/following` | 已迁移 | 已替换为新移动端关注直播间列表，`/follow` 保留兼容跳转到 `/following`。 |
| `Login.tsx` | `/login` | `frontend/h5/src/pages/Login/index.tsx` | `/login` | `/login` | 已迁移 | 新页面只保留手机号/密码登录；旧注册切换和邮箱字段不再作为用户可达 UI 暴露。 |

## 旧 H5 页面保留与舍弃

| 旧 H5 页面 | 当前路由 | 处理结论 | 原因 |
| --- | --- | --- | --- |
| `frontend/h5/src/pages/Home/index.tsx` | `/` | 替换为新 `Home` | 新移动端已有等价首页。 |
| `frontend/h5/src/pages/Live/index.tsx` | `/live` | 从用户可达 UI 中舍弃，能力并入新 `LiveRoom` | 旧页面为 mock 风格直播间；新 `LiveRoom` 是目标实现。 |
| `frontend/h5/src/pages/Auction/index.tsx` | `/auction/:id` | 从用户可达 UI 中舍弃，源码保留 | 新移动端不保留旧 `/auction/:id` 页面语义；Task 12 后该路径只兼容跳转 `/detail?id=:id`，不再渲染旧页面。 |
| `frontend/h5/src/pages/Result/index.tsx` | `/result/:id` | 替换为新 `AuctionResult` | 新移动端已有等价结果页。 |
| `frontend/h5/src/pages/History/index.tsx` | `/history` | 替换为新 `AuctionHistory` | 新移动端关注用户竞拍记录，不直接复用旧订单历史 UI。 |
| `frontend/h5/src/pages/Follow/index.tsx` | `/follow` | 替换为新 `Following`，路由改为 `/following` | 新移动端路由命名不同。 |
| `frontend/h5/src/pages/Login/index.tsx` | `/login` | 替换为新 `Login` | 新移动端已有等价登录页。 |
| `frontend/h5/src/pages/User/Index.tsx` | 未接入 | 作为 `Profile` 旧能力参考，不保留旧 UI | 旧源码存在但当前不可达；新移动端有 `Profile`。 |

## 页面功能边界

| 页面 | 新 UI 功能边界 | 旧 H5 相关能力 | 初始接口依赖 |
| --- | --- | --- | --- |
| `Home` | 分类 Tab、竞拍列表、详情入口、直播入口、结果入口、通知入口 | 首页竞拍列表、状态筛选、直播/关注/历史入口 | `auctionApi.list` 或旧 `/api/v1/auctions` |
| `LiveRoom` | 直播信息、当前竞拍、出价、出价记录、倒计时、聊天输入、结果跳转 | 旧 `Live` mock 直播商品列表；旧 `Auction` 真实竞拍、出价、WebSocket 排行 | `auctionApi.get`、`auctionApi.getBids`、`auctionApi.placeBid`/`bidApi.placeBid`、`productApi.get`、`liveStreamApi.get`、`WebSocketService` |
| `ProductDetail` | 商品图文、竞拍规则、出价记录、快捷出价、结果入口 | 旧 `Auction` 竞拍详情与出价能力 | `auctionApi.get`、`auctionApi.getBids`、`auctionApi.placeBid`/`bidApi.placeBid`、`productApi.get` |
| `AuctionResult` | 最终成交价、中标者、商品信息、返回首页、订单入口 | 旧 `Result` 只展示结果和模拟支付按钮 | `auctionApi.get`、`auctionApi.getBids`、`productApi.get`、后续可能需要 `auctionApi.getResult`、`orderApi.get` |
| `Profile` | 用户资料、角色、关注/粉丝/竞拍记录入口、钱包占位、退出登录 | 旧 `User/Index` 用户信息、用户统计、最近竞拍、功能菜单 | `authContext`、`userApi.getProfile`、旧 `/api/v1/users/me`、旧 `/api/v1/users/me/stats` |
| `AuctionHistory` | 用户参与竞拍记录、成功/失败状态、结果/详情入口 | 旧 `History` 订单列表、支付、订单状态筛选 | 当前新 UI 临时用 `auctionApi.list(status=3)`；旧页用 `/api/v1/orders`、`/api/v1/orders/:id/pay` |
| `Notifications` | 通知列表、未登录保护、按通知类型跳转 | 旧通知 service/hook、全局通知弹窗；无独立页面 | `notificationApi.list`、`notificationApi.getUnreadCount`、`notificationApi.markAsRead` |
| `Following` | 关注直播间列表、直播状态、进入直播间、取消关注 | 旧 `Follow` 关注列表、搜索、加载更多、取消关注、进入直播间 | `followApi.getFollowedLiveStreams`、`followApi.unfollowLiveStream` |
| `Login` | 手机号/密码登录、redirect 参数、错误提示 | 旧登录/注册、邮箱/手机号、登录成功事件 | `authApi.login`、`AuthProvider`/`authContext` |

## 建议迁移顺序

1. 全局框架与 `MobileContainer`、底部导航、样式基线。
2. `Home`，因为它决定主要入口和后续页面可达性。
3. `LiveRoom`，因为它承接竞拍核心链路。
4. `ProductDetail`，从竞拍页拆分详情能力。
5. `AuctionResult`，闭合竞拍结束链路。
6. `Profile`，接入个人中心和认证保护。
7. `AuctionHistory`，接入用户竞拍记录。
8. `Following`，接入关注列表。
9. `Notifications`，接入通知中心。
10. `Login`，最后统一认证交互和 401 回跳。

## 待确认

| 问题 | 影响 | 默认处理 |
| --- | --- | --- |
| 新移动端使用 query 参数，旧 H5 部分路由使用 path 参数 | 路由兼容和历史链接跳转 | 迁移目标以新移动端路由为准，必要时保留 redirect/兼容跳转。 |
| 旧 H5 当前无 `/profile`、`/notifications`、`/detail` | 页面可达性缺口 | 新增保留页面并接入 H5 路由。 |
| 旧 H5 `Live` 大量 mock 数据 | 可能误导接口对接 | 不复用旧 UI，只把真实接口能力从 `Auction`、service、WebSocket 中迁出。 |
| 旧 H5 `User/Index.tsx` 当前未接入 | 旧能力可见性不一致 | 作为能力参考，不恢复旧 UI。 |

## Task 12 路由收口确认

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 新移动端 retained pages 可达 | 通过 | H5 `App.tsx` 保留 `/`、`/live`、`/detail`、`/result`、`/profile`、`/history`、`/notifications`、`/following`、`/login`。 |
| 旧 H5-only 页面下线 | 通过 | `/auction/:id` 不再 lazy import 或渲染 `Auction` 页面，只重定向到 `/detail?id=`；旧源码文件保留。 |
| 旧 path 参数兼容 | 通过 | `/result/:id` 重定向到 `/result?id=`；`/follow` 继续重定向到 `/following`。 |
| 底部导航范围 | 通过 | `BottomNav` 仅暴露 `首页`、`直播间`、`我的` 三个主入口，其他 retained pages 由页面按钮、个人中心或通知入口进入。 |

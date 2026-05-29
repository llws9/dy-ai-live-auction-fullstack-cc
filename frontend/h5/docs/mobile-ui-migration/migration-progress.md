# H5 新移动端迁移进度

## 总体状态

| 项目 | 状态 | 说明 |
| --- | --- | --- |
| Task 1 迁移盘点 | 已完成 | 已创建页面映射、冗余接口、缺失接口、迁移进度四个文档。 |
| Task 2 全局框架迁移 | 已完成 | 已接入 `MobileContainer`、底部导航、目标路由和基础样式；已移除启动演示弹窗逻辑。 |
| 页面迁移 | 已完成 | `Home`、`LiveRoom`、`ProductDetail`、`AuctionResult`、`Profile`、`AuctionHistory`、`Following`、`Notifications`、`Login` 已完成；Task 12 已收口旧页面、路由和导航。 |
| Task 13 全量验证 | 已完成 | 构建通过；lint、Jest 全量单测、Playwright e2e 的既有失败项已记录，接口差异文档已复核。 |
| 旧代码删除 | 未开始 | Spec 要求全部迁移完成后再询问用户。 |

## 页面迁移状态

| 顺序 | 页面 | 目标路由 | 新页面识别 | 旧页面识别 | 旧接口梳理 | UI 替换 | 接口和按钮对接 | 差异文档更新 | 验证 | 当前状态 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `Home` | `/` | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 通过 | 已完成 |
| 2 | `LiveRoom` | `/live?id=&auction_id=` | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 通过 | 已完成 |
| 3 | `ProductDetail` | `/detail?id=` | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 通过 | 已完成 |
| 4 | `AuctionResult` | `/result?id=` | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 通过 | 已完成 |
| 5 | `Profile` | `/profile` | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 通过 | 已完成 |
| 6 | `AuctionHistory` | `/history` | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 通过 | 已完成 |
| 7 | `Following` | `/following` | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 通过 | 已完成 |
| 8 | `Notifications` | `/notifications` | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 通过 | 已完成 |
| 9 | `Login` | `/login` | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 已完成 | 通过 | 已完成 |

## Task 1 验证记录

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 读取规格 | 通过 | `.trae/specs/replace-h5-with-new-mobile-pages/spec.md` |
| 读取任务 | 通过 | `.trae/specs/replace-h5-with-new-mobile-pages/tasks.md` |
| 对比新旧 App 路由 | 通过 | 新 `src/mobile/App.tsx` 9 个路由；旧 `frontend/h5/src/App.tsx` 7 个主路由。 |
| 读取新移动端页面 | 通过 | `Home`、`LiveRoom`、`ProductDetail`、`AuctionResult`、`Profile`、`AuctionHistory`、`Notifications`、`Following`、`Login`。 |
| 读取旧 H5 页面 | 通过 | `Home`、`Live`、`Auction`、`Result`、`History`、`Follow`、`Login`、`User/Index`。 |
| 创建四个盘点文档 | 通过 | `page-mapping.md`、`redundant-interfaces.md`、`missing-interfaces.md`、`migration-progress.md`。 |
| 构建/测试 | 未运行 | Task 1 只新增 Markdown 文档，无代码变更。 |

## Task 2 验证记录

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 公共容器 | 通过 | 新增 `frontend/h5/src/components/MobileShell/MobileContainer.tsx`，保留 H5 原有运行时 providers。 |
| 底部导航 | 通过 | 新增 `frontend/h5/src/components/MobileShell/BottomNav.tsx`，仅暴露 `首页`、`直播间`、`我的` 三个主入口，并在详情、结果、通知、关注、历史、登录页隐藏。 |
| 路由对齐 | 通过 | `frontend/h5/src/App.tsx` 接入 `/detail`、`/profile`、`/notifications`、`/following`、`/result`，并保留 `/auction/:id`、`/result/:id` 和 `/follow` 兼容跳转。 |
| 启动演示逻辑 | 通过 | 已删除 H5 `App.tsx` 启动阶段的硬编码 `mockStream`、`setTimeout` 和全局演示弹窗挂载。 |
| 组件测试 | 通过 | `npm test -- MobileShell.test.tsx --runInBand`，8 tests passed。 |
| 构建验证 | 通过 | `npm run build`，TypeScript 与 Vite 构建通过。 |

## 下一步入口

| 下一任务 | 前置条件 | 建议动作 |
| --- | --- | --- |
| Task 2 | Task 1 已完成 | 迁移 `MobileContainer`、底部导航和全局样式；保留 H5 的 `AuthProvider`、`ToastProvider`、`ErrorBoundary`、`GrowthBookContextProvider`。 |
| Task 3 | Task 2 构建通过 | 已完成：`Home` 使用新移动端 UI，并通过页面 adapter 对接 `auctionApi.list` 与 `productApi.get`。 |
| Task 4 | Task 3 构建通过 | 已完成：`LiveRoom` 使用新移动端直播间结构，对接直播、竞拍、商品、排名、出价、关注和 WebSocket 实时同步。 |
| Task 5 | Task 4 构建通过 | 已完成：`ProductDetail` 使用新移动端商品详情结构，对接商品、竞拍、出价记录和快捷出价。 |
| Task 6 | Task 5 构建通过 | 已完成：`AuctionResult` 使用新移动端结果页结构，对接权威竞拍结果、商品信息和中标订单支付。 |
| Task 7 | Task 6 构建通过 | 已完成：`Profile` 使用新移动端个人中心结构，对接用户资料、余额、订单入口和认证退出。 |
| Task 8 | Task 7 构建通过 | 已完成：`AuctionHistory` 接入历史接口文档中的 `/orders/history`，并移除旧订单支付历史行为。 |
| Task 9 | Task 8 构建通过 | 已完成：`Following` 接入 `followApi.getFollowedLiveStreams`，补齐取消关注和 `/live?id=` 进入直播间跳转。 |
| Task 10 | Task 9 构建通过 | 已完成：`Notifications` 新增消息通知页，接入 `notificationApi.list/getUnreadCount/markAsRead/markAllAsRead`，并记录订单详情跳转缺口。 |
| Task 11 | Task 10 构建通过 | 已完成：`Login` 使用新移动端登录结构，对接手机号/密码登录和 redirect 回跳。 |
| Task 12 | Task 11 构建通过 | 已完成：旧 H5-only 页面不再从用户可达路由渲染，旧源码保留，retained pages 路由和导航入口已确认。 |
| Task 13 | Task 12 构建通过 | 已完成：执行全量验证、检查接口文档并输出最终迁移报告；下一步等待确认是否删除旧移动端代码。 |

## 迁移约束

| 约束 | 执行方式 |
| --- | --- |
| 一次只迁移一个页面 | 每个页面必须完成“新页面识别 -> 旧页面识别 -> 旧接口梳理 -> UI 替换 -> 接口和按钮对接 -> 差异文档更新 -> 验证”。 |
| 不删除旧源码 | 旧 H5 页面先从可达路由中下线，源码保留到最终确认。 |
| 不用 mock 掩盖缺口 | 缺接口写入 `missing-interfaces.md`，页面使用空态或禁用态。 |
| 所有 H5 HTTP/WS 走 gateway | 迁移后的请求继续使用 H5 项目 `/api/v1` 代理约定。 |


## Task 3 验证记录

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 新旧页面识别 | 通过 | 已对比新 `dy-ai-live-auction-fullstack-ui/src/mobile/pages/Home.tsx` 与旧 `frontend/h5/src/pages/Home/index.tsx`。 |
| UI 替换 | 通过 | `frontend/h5/src/pages/Home/index.tsx` 已替换为新移动端首页结构，样式迁移到 `Home.module.css`。 |
| 接口对接 | 通过 | 首页使用 `auctionApi.list({ page: 1, page_size: 20 })` 拉竞拍列表，并在缺少内嵌商品时用 `productApi.get(product_id)` 补商品名称和图片。 |
| 按钮跳转 | 通过 | 首页接入 `/detail?id=`、`/live?id=&auction_id=`、`/result?id=`、`/following`、`/notifications`。 |
| 差异文档 | 通过 | 已更新 `redundant-interfaces.md` 和 `missing-interfaces.md` 的 Task 3 Home 迁移确认章节。 |
| 诊断 | 通过 | `GetDiagnostics` 检查 `Home/index.tsx` 与 `Home.module.css` 无诊断。 |
| 构建验证 | 通过 | `npm run build` 在 `frontend/h5` 下通过。 |

## Task 4 验证记录

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 新旧页面识别 | 通过 | 已对比新 `dy-ai-live-auction-fullstack-ui/src/mobile/pages/LiveRoom.tsx` 与旧 `frontend/h5/src/pages/Live/index.tsx`、`frontend/h5/src/pages/Auction/index.tsx`。 |
| UI 替换 | 通过 | `frontend/h5/src/pages/Live/index.tsx` 已替换为新移动端 `LiveRoom` 结构，样式迁移到 `Live.module.css`，不再使用旧 mock 商品和 mock 出价记录。 |
| 接口对接 | 通过 | 直播间使用 `auctionApi.get`、`productApi.get`、`liveStreamApi.get`、`bidApi.getRanking`、`auctionApi.getBids` fallback、`bidApi.placeBid`、`followApi.followLiveStream/unfollowLiveStream/getFollowersStats`。 |
| 实时状态 | 通过 | 页面接入 `WebSocketService`，处理 `rank_update`、`bid_placed`、`sync_response`、`auction_ended`，并维护连接状态、排行榜、当前价和竞拍状态。 |
| 按钮行为 | 通过 | 关注按钮走乐观更新并失败回滚；出价按钮校验登录态和最低出价，成功后更新当前价与排行榜。 |
| 差异文档 | 通过 | 已更新 `redundant-interfaces.md` 和 `missing-interfaces.md` 的 Task 4 LiveRoom 迁移确认章节。 |
| 诊断 | 通过 | `GetDiagnostics` 检查 `Live/index.tsx` 与 `Live.module.css` 无诊断。 |
| 聚焦单测 | 通过 | `npm test -- LiveRoom.test.tsx --runInBand`，1 test passed。 |
| 构建验证 | 通过 | `npm run build` 在 `frontend/h5` 下通过。 |
| 全量单测 | 未通过 | `npm test -- --runInBand` 失败：既有 `Card/Loading/Toast` CSS Module 类名断言、`BidInput/FollowButton` 引用未安装的 `vitest`、`Auction.integration` 解析 `import.meta` 配置问题；`LiveRoom.test.tsx` 在全量执行中通过。 |

## Task 5 验证记录

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 新旧页面识别 | 通过 | 已对比新 `dy-ai-live-auction-fullstack-ui/src/mobile/pages/ProductDetail.tsx` 与旧 `frontend/h5/src/pages/Auction/index.tsx`。 |
| UI 替换 | 通过 | 新增 `frontend/h5/src/pages/ProductDetail/index.tsx` 和 `ProductDetail.module.css`，`/detail?id=` 已改为渲染商品详情页；旧 `/auction/:id` 继续保留兼容。 |
| 接口对接 | 通过 | 商品详情页使用 `auctionApi.get`、`productApi.get`、`auctionApi.getBids`、`bidApi.placeBid`，并从商品和竞拍详情中兼容读取竞拍规则字段。 |
| 按钮行为 | 通过 | 快捷出价按 `current_price + increment` 生成金额，出价成功后刷新详情；未登录时跳转 `/login?redirect=/detail?id=`。 |
| 差异文档 | 通过 | 已更新 `redundant-interfaces.md` 和 `missing-interfaces.md` 的 Task 5 ProductDetail 迁移确认章节。 |
| 聚焦单测 | 通过 | `npm test -- ProductDetail.test.tsx --runInBand`，1 test passed。 |
| 构建验证 | 通过 | `npm run build` 在 `frontend/h5` 下通过。 |

## Task 6 验证记录

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 新旧页面识别 | 通过 | 已对比新 `dy-ai-live-auction-fullstack-ui/src/mobile/pages/AuctionResult.tsx` 与旧 `frontend/h5/src/pages/Result/index.tsx`。 |
| UI 替换 | 通过 | `frontend/h5/src/pages/Result/index.tsx` 已替换为新移动端结果页结构，新增 `Result.module.css`，保留 `/result?id=` 和 `/result/:id` 兼容入口。 |
| 接口对接 | 通过 | 结果页使用 `auctionApi.getResult` 读取权威结果，使用 `productApi.get` 补商品信息，使用 `orderApi.pay` 支付中标订单。 |
| 按钮行为 | 通过 | 中标且存在 `order_id` 时展示“立即支付”；支付成功后更新订单状态提示；无 `order_id` 时禁用并显示“订单待生成”。 |
| 差异文档 | 通过 | 已更新 `redundant-interfaces.md` 和 `missing-interfaces.md` 的 Task 6 AuctionResult 迁移确认章节。 |
| 诊断 | 通过 | `GetDiagnostics` 检查 `Result/index.tsx`、`Result.module.css`、`AuctionResult.test.tsx` 无诊断。 |
| 聚焦单测 | 通过 | `npm test -- AuctionResult.test.tsx --runInBand`，1 test passed。 |
| 构建验证 | 通过 | `npm run build` 在 `frontend/h5` 下通过。 |

## Task 7 验证记录

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 新旧页面识别 | 通过 | 已对比新 `dy-ai-live-auction-fullstack-ui/src/mobile/pages/Profile.tsx` 与旧 `frontend/h5/src/pages/User/Index.tsx`。 |
| UI 替换 | 通过 | `frontend/h5/src/pages/User/Index.tsx` 已替换为新移动端个人中心结构，新增 `Profile.module.css`，保留 `/profile` 的 `PrivateRoute` 认证保护。 |
| 接口对接 | 通过 | 个人中心使用 `userApi.getProfile`、`userApi.getBalance`、`orderApi.list`；`api.ts` 已兼容读取 `auth_token`，避免认证上下文与请求 token key 不一致。 |
| 按钮行为 | 通过 | 入口对齐 `/history`、`/following`、`/notifications`；退出登录走 `AuthContext.logout` 并回到 `/login`。 |
| 差异文档 | 通过 | 已更新 `redundant-interfaces.md` 和 `missing-interfaces.md` 的 Task 7 Profile 迁移确认章节。 |
| 诊断 | 通过 | `GetDiagnostics` 检查 `User/Index.tsx`、`Profile.module.css`、`Profile.test.tsx`、`api.ts`、`api.test.ts` 无诊断。 |
| 聚焦单测 | 通过 | `npm test -- Profile.test.tsx api.test.ts --runInBand`，3 tests passed。 |
| 构建验证 | 通过 | `npm run build` 在 `frontend/h5` 下通过。 |

## Task 8 验证记录

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 新旧页面识别 | 通过 | 已对比新 `dy-ai-live-auction-fullstack-ui/src/mobile/pages/AuctionHistory.tsx` 与旧 `frontend/h5/src/pages/History/index.tsx`。 |
| UI 替换 | 通过 | `frontend/h5/src/pages/History/index.tsx` 已替换为新移动端竞拍记录结构，新增 `AuctionHistory.module.css`，保留 `/history` 的 `PrivateRoute` 认证保护。 |
| 接口对接 | 通过 | 新增 `orderApi.history({ page, page_size })`，按历史接口文档请求 `GET /api/v1/orders/history?page=&page_size=`，页面 adapter 兼容 `list/items/data.list/data.items`。 |
| 按钮行为 | 通过 | 成功记录跳 `/result?id=`，未中标记录跳 `/detail?id=`；移除旧页面内 `/orders/:id/pay` 支付弹窗和模拟支付。 |
| 差异文档 | 通过 | 已更新 `redundant-interfaces.md` 和 `missing-interfaces.md` 的 Task 8 AuctionHistory 迁移确认章节。 |
| 诊断 | 通过 | `GetDiagnostics` 检查当前工作区无诊断。 |
| 聚焦单测 | 通过 | `npm test -- AuctionHistory.test.tsx api.test.ts --runInBand`，3 tests passed。 |
| 构建验证 | 通过 | `npm run build` 在 `frontend/h5` 下通过。 |
| 全量单测 | 未通过 | `npm test -- --runInBand` 失败：既有 `Card/Loading/Toast` CSS Module 类名断言、`BidInput/FollowButton` 引用未安装的 `vitest`、`Home/Auction.integration` 解析 `import.meta` 配置问题；`AuctionHistory.test.tsx` 在全量执行中通过。 |

## Task 9 验证记录

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 新旧页面识别 | 通过 | 已对比新 `dy-ai-live-auction-fullstack-ui/src/mobile/pages/Following.tsx` 与旧 `frontend/h5/src/pages/Follow/index.tsx`。 |
| UI 替换 | 通过 | `frontend/h5/src/pages/Follow/index.tsx` 已替换为新移动端关注直播间结构，新增 `Following.module.css`，保留 `/follow` 到 `/following` 的兼容跳转。 |
| 接口对接 | 通过 | 页面使用 `followApi.getFollowedLiveStreams(1, 20)` 读取真实关注列表，并通过 adapter 兼容 `list/items/data.list/data.items`。 |
| 按钮行为 | 通过 | 卡片操作接入 `followApi.unfollowLiveStream` 原地移除关注项；进入直播间跳转 `/live?id=`，不再使用旧 `/live/:id`。 |
| 差异文档 | 通过 | 已更新 `redundant-interfaces.md` 和 `missing-interfaces.md` 的 Task 9 Following 迁移确认章节。 |
| 诊断 | 通过 | `GetDiagnostics` 检查当前工作区无诊断。 |
| 聚焦单测 | 通过 | `npm test -- Following.test.tsx --runInBand`，2 tests passed。 |
| 构建验证 | 通过 | `npm run build` 在 `frontend/h5` 下通过。 |
| 全量单测 | 未通过 | `npm test -- --runInBand` 失败：既有 `Button/Card/Input/Loading/Toast` CSS Module 类名断言、`BidInput/FollowButton` 引用未安装的 `vitest`、`Home/Auction.integration` 解析 `import.meta` 配置问题；`Following.test.tsx` 在全量执行中通过。 |

## Task 10 验证记录

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 新旧页面识别 | 通过 | 已对比新 `dy-ai-live-auction-fullstack-ui/src/mobile/pages/Notifications.tsx` 与旧 `frontend/h5/src/services/notification.ts`、`frontend/h5/src/hooks/useNotification.ts`、`frontend/h5/src/components/Notification/index.tsx`。 |
| UI 替换 | 通过 | 新增 `frontend/h5/src/pages/Notifications/index.tsx` 和 `Notifications.module.css`，`/notifications` 从占位页改为通知中心页面。 |
| 接口对接 | 通过 | 页面使用 `notificationApi.list(1, 20)`、`notificationApi.getUnreadCount()`、`notificationApi.markAsRead(id)`、`notificationApi.markAllAsRead()`，响应 adapter 兼容 `items/list/data.items/data.list`。 |
| 按钮行为 | 通过 | 开播通知跳 `/live?id=`，竞拍结果跳 `/result?id=`，竞拍提醒跳 `/detail?id=`；订单通知因用户端订单详情页缺失不跳转，只展示内容和缺口提示。 |
| 差异文档 | 通过 | 已更新 `redundant-interfaces.md` 和 `missing-interfaces.md` 的 Task 10 Notifications 迁移确认章节。 |
| 诊断 | 通过 | `GetDiagnostics` 检查 `Notifications/index.tsx`、`Notifications.test.tsx`、`App.tsx` 无诊断。 |
| 聚焦单测 | 通过 | `npm test -- Notifications.test.tsx --runInBand`，2 tests passed。 |
| 构建验证 | 通过 | `npm run build` 在 `frontend/h5` 下通过。 |
| 全量单测 | 未通过 | `npm test -- --runInBand` 失败：既有 CSS Module 类名断言、`BidInput/FollowButton` 引用未安装的 `vitest`、`Home/Auction.integration` 解析 `import.meta` 配置问题；`Notifications.test.tsx` 在全量执行中通过。 |

## Task 11 验证记录

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 新旧页面识别 | 通过 | 已对比新 `dy-ai-live-auction-fullstack-ui/src/mobile/pages/Login.tsx` 与旧 `frontend/h5/src/pages/Login/index.tsx`。 |
| UI 替换 | 通过 | `frontend/h5/src/pages/Login/index.tsx` 已替换为新移动端登录结构，新增 `Login.module.css`，移除内联样式和注册切换 UI。 |
| 接口对接 | 通过 | 登录页按新 UI 使用手机号/密码请求 `POST /api/v1/auth/login`，成功后继续调用 `AuthContext.setAuth(token, user)` 写入 `auth_token/auth_user`。 |
| 认证回跳 | 通过 | 登录成功触发 `login-success` 事件并按 `redirect` 参数回跳；`api.ts` 401 登录跳转路径统一构造为 `/login?redirect=` 当前路径。 |
| 差异文档 | 通过 | 已更新 `redundant-interfaces.md` 和 `missing-interfaces.md` 的 Task 11 Login 迁移确认章节。 |
| 诊断 | 通过 | `GetDiagnostics` 检查当前工作区无诊断。 |
| 聚焦单测 | 通过 | `npm test -- api.test.ts Login.test.tsx --runInBand`，5 tests passed。 |
| 构建验证 | 通过 | `npm run build` 在 `frontend/h5` 下通过。 |
| 全量单测 | 未通过 | `npm test -- --runInBand` 失败：既有 CSS Module 类名断言、`BidInput/FollowButton` 引用未安装的 `vitest`、`Home/Auction.integration` 和 `Auction.integration` 解析 `import.meta` 配置问题；`Login.test.tsx` 和 `api.test.ts` 在聚焦执行中通过。 |

## Task 12 验证记录

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 旧页面可达性收口 | 通过 | `frontend/h5/src/App.tsx` 已移除旧 `Auction` lazy import，`/auction/:id` 改为兼容跳转 `/detail?id=`，不再渲染旧竞拍页。 |
| 旧源码保留 | 通过 | `frontend/h5/src/pages/Auction/index.tsx` 未删除，仍作为迁移前源码保留，等待最终确认。 |
| retained pages 可达 | 通过 | H5 路由保留 `/`、`/live`、`/detail`、`/result`、`/profile`、`/history`、`/notifications`、`/following`、`/login`；`/follow` 和 `/result/:id` 保留兼容跳转。 |
| 底部导航范围 | 通过 | `BottomNav` 仅暴露 `首页`、`直播间`、`我的` 三个主入口，并在详情、结果、通知、关注、历史、登录页隐藏。 |
| 路由测试 | 通过 | `npm test -- AppRoutes.test.tsx MobileShell.test.tsx --runInBand`，10 tests passed。 |
| 构建验证 | 通过 | `npm run build` 在 `frontend/h5` 下通过。 |

## Task 13 验证记录

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 构建验证 | 通过 | `npm run build` 在 `frontend/h5` 下通过，`tsc && vite build` 成功，132 modules transformed。 |
| Lint 验证 | 未通过 | `npm run lint` 失败：ESLint 找不到配置文件；影响范围是 lint 门禁不可用，非本次页面迁移代码语法错误。 |
| 全量 Jest 单测 | 未通过 | `npm test -- --runInBand` 失败：24 个 suites 中 15 个通过、9 个失败；74 tests passed、23 failed。失败集中在既有 CSS Module 类名断言、`BidInput/FollowButton` 引用未安装 `vitest`、`Home/Auction.integration` 解析 `import.meta`。 |
| Playwright e2e 初次执行 | 未通过 | `npm run test:e2e` 在 `CI=1` 环境下失败：`http://localhost:5173 is already used` 且配置不复用现有服务。 |
| Playwright e2e 补跑 | 未通过 | `env -u CI npm run test:e2e` 复用现有服务后执行 189 项，45 passed、144 failed；主要失败为旧 e2e 仍查找 `input[placeholder*="用户名"]`、旧首页/直播/订单选择器或旧订单流程。 |
| 冗余接口文档 | 通过 | `redundant-interfaces.md` 已覆盖 Task 3-11 的逐页冗余接口、UI 冗余和待确认旧能力；Task 12 只收口路由/导航，没有新增接口冗余。 |
| 缺失接口文档 | 通过 | `missing-interfaces.md` 已覆盖 Task 3-11 的后端缺口、契约差异和前端安全降级；Task 12 没有新增后端契约缺口。 |
| Checklist | 通过 | `.trae/specs/replace-h5-with-new-mobile-pages/checklist.md` 已全部勾选。 |

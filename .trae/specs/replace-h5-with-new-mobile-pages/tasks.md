# Tasks

- [x] Task 1: 建立迁移盘点文档
  - [x] SubTask 1.1: 对比 `dy-ai-live-auction-fullstack-ui/src/mobile/App.tsx` 与 `frontend/h5/src/App.tsx`，整理新旧路由和页面映射。
  - [x] SubTask 1.2: 读取新移动端 `src/mobile/pages/**` 与旧 H5 `frontend/h5/src/pages/**`，确认每个页面的功能边界。
  - [x] SubTask 1.3: 创建 `frontend/h5/docs/mobile-ui-migration/page-mapping.md`，记录页面映射、保留页面、舍弃页面、迁移顺序。
  - [x] SubTask 1.4: 创建 `frontend/h5/docs/mobile-ui-migration/redundant-interfaces.md`，用于记录旧页面有但新页面不需要的接口。
  - [x] SubTask 1.5: 创建 `frontend/h5/docs/mobile-ui-migration/missing-interfaces.md`，用于记录新页面需要但现有后端缺失的接口。
  - [x] SubTask 1.6: 创建 `frontend/h5/docs/mobile-ui-migration/migration-progress.md`，用于逐页记录迁移状态和验证结果。

- [x] Task 2: 迁移全局移动端框架和基础组件
  - [x] SubTask 2.1: 将新移动端公共容器、底部导航和全局样式适配到 `frontend/h5/src`，保留 H5 现有 `AuthProvider`、`ToastProvider`、`ErrorBoundary`、`GrowthBookContextProvider` 等运行时能力。
  - [x] SubTask 2.2: 对齐新移动端路由与旧 H5 路由，确保新页面能在 H5 工程内渲染。
  - [x] SubTask 2.3: 移除 H5 启动阶段的硬编码演示数据和仅用于测试展示的弹窗触发逻辑，避免影响迁移验证。
  - [x] SubTask 2.4: 运行 `npm run build`，确认基础框架迁移后 TypeScript 与 Vite 构建通过。

- [x] Task 3: 迁移 `Home` 主页并完成接口对接
  - [x] SubTask 3.1: 读取新页面 `src/mobile/pages/Home.tsx` 和旧页面 `frontend/h5/src/pages/Home/index.tsx`，记录旧主页接口和按钮行为。
  - [x] SubTask 3.2: 用新主页 UI 替换旧 H5 主页，保留新主页实际需要的直播间、商品、竞拍入口、关注入口等交互。
  - [x] SubTask 3.3: 对接现有 `liveStreamApi`、`productApi`、`auctionApi` 或必要 service adapter；接口响应不匹配时以新 UI 所需字段为目标做前端映射。
  - [x] SubTask 3.4: 把旧主页冗余接口写入 `redundant-interfaces.md`，把主页缺失接口写入 `missing-interfaces.md`。
  - [x] SubTask 3.5: 运行主页相关单测或 `npm run build`，并更新 `migration-progress.md`。

- [x] Task 4: 迁移 `LiveRoom` 直播间并完成接口对接
  - [x] SubTask 4.1: 读取新页面 `src/mobile/pages/LiveRoom.tsx` 和旧页面 `frontend/h5/src/pages/Live/index.tsx`、`frontend/h5/src/pages/Auction/index.tsx`，确认直播、竞拍、出价、关注、WebSocket 的边界。
  - [x] SubTask 4.2: 用新直播间 UI 替换旧 H5 直播间/竞拍入口相关界面，保留用户进入直播间后的核心操作。
  - [x] SubTask 4.3: 对接 `liveStreamApi`、`auctionApi`、`bidApi`、`followApi`、`websocket`，并确保出价、关注、排行榜、倒计时等按钮或状态可用。
  - [x] SubTask 4.4: 将新直播间不需要的旧接口写入 `redundant-interfaces.md`，将缺失的直播/竞拍接口写入 `missing-interfaces.md`。
  - [x] SubTask 4.5: 运行直播间相关单测、WebSocket 测试或 `npm run build`，并更新 `migration-progress.md`。

- [x] Task 5: 迁移 `ProductDetail` 商品详情并完成接口对接
  - [x] SubTask 5.1: 读取新页面 `src/mobile/pages/ProductDetail.tsx`，并在旧 H5 中确认对应能力主要来自 `Auction`、`Live` 或商品接口。
  - [x] SubTask 5.2: 在 H5 路由中保留商品详情页，若旧 H5 没有等价页面，则按新页面作为新增保留页面处理。
  - [x] SubTask 5.3: 对接 `productApi.get`、相关竞拍详情接口和按钮跳转逻辑。
  - [x] SubTask 5.4: 记录商品详情相关冗余接口和缺失接口。
  - [x] SubTask 5.5: 运行构建或相关测试，并更新 `migration-progress.md`。

- [x] Task 6: 迁移 `AuctionResult` 竞拍结果并完成接口对接
  - [x] SubTask 6.1: 读取新页面 `src/mobile/pages/AuctionResult.tsx` 和旧页面 `frontend/h5/src/pages/Result/index.tsx`，确认竞拍成功、失败、订单入口、支付入口的差异。
  - [x] SubTask 6.2: 用新竞拍结果 UI 替换旧结果页；如果新页面提供旧 H5 不存在的结果界面状态，则保留该状态。
  - [x] SubTask 6.3: 对接 `auctionApi.getResult`、`orderApi.create`、`orderApi.pay` 或记录缺失能力。
  - [x] SubTask 6.4: 记录结果页冗余接口和缺失接口。
  - [x] SubTask 6.5: 运行结果页相关测试或 `npm run build`，并更新 `migration-progress.md`。

- [x] Task 7: 迁移 `Profile` 个人中心并完成接口对接
  - [x] SubTask 7.1: 读取新页面 `src/mobile/pages/Profile.tsx` 和旧页面 `frontend/h5/src/pages/User/Index.tsx`，确认用户资料、余额、订单、历史、关注入口。
  - [x] SubTask 7.2: 用新个人中心 UI 替换旧 H5 用户页，保留认证保护。
  - [x] SubTask 7.3: 对接 `userApi.getProfile`、`userApi.getBalance`、`orderApi.list` 以及页面按钮跳转。
  - [x] SubTask 7.4: 记录个人中心冗余接口和缺失接口。
  - [x] SubTask 7.5: 运行认证/用户页相关测试或 `npm run build`，并更新 `migration-progress.md`。

- [x] Task 8: 迁移 `AuctionHistory` 竞拍历史并完成接口对接
  - [x] SubTask 8.1: 读取新页面 `src/mobile/pages/AuctionHistory.tsx` 和旧页面 `frontend/h5/src/pages/History/index.tsx`，确认历史记录数据来源和状态分类。
  - [x] SubTask 8.2: 用新竞拍历史 UI 替换旧历史页，保留登录态访问控制。
  - [x] SubTask 8.3: 对接现有订单、竞拍或出价历史接口；如无明确接口，写入 `missing-interfaces.md`。
  - [x] SubTask 8.4: 记录历史页冗余接口和缺失接口。
  - [x] SubTask 8.5: 运行相关测试或 `npm run build`，并更新 `migration-progress.md`。

- [x] Task 9: 迁移 `Following` 关注列表并完成接口对接
  - [x] SubTask 9.1: 读取新页面 `src/mobile/pages/Following.tsx` 和旧页面 `frontend/h5/src/pages/Follow/index.tsx`，确认关注列表、取消关注、进入直播间逻辑。
  - [x] SubTask 9.2: 用新关注列表 UI 替换旧关注页，保留登录态访问控制。
  - [x] SubTask 9.3: 对接 `followApi.getFollowedLiveStreams`、`followApi.unfollowLiveStream` 和直播间跳转。
  - [x] SubTask 9.4: 记录关注页冗余接口和缺失接口。
  - [x] SubTask 9.5: 运行关注相关测试或 `npm run build`，并更新 `migration-progress.md`。

- [x] Task 10: 迁移 `Notifications` 消息通知并完成接口对接
  - [x] SubTask 10.1: 读取新页面 `src/mobile/pages/Notifications.tsx` 和旧 H5 `notification` service、`Notification` component，确认消息提醒、开播提醒、系统通知边界。
  - [x] SubTask 10.2: 在 H5 中保留消息通知页；如果旧 H5 无等价页面，则按新增保留页面处理。
  - [x] SubTask 10.3: 对接现有通知接口；如果仅有组件或本地通知逻辑而无后端列表接口，写入 `missing-interfaces.md`。
  - [x] SubTask 10.4: 记录通知页冗余接口和缺失接口。
  - [x] SubTask 10.5: 运行通知相关测试或 `npm run build`，并更新 `migration-progress.md`。

- [x] Task 11: 迁移 `Login` 登录页并完成接口对接
  - [x] SubTask 11.1: 读取新页面 `src/mobile/pages/Login.tsx` 和旧页面 `frontend/h5/src/pages/Login/index.tsx`，确认登录方式、token 存储、跳转逻辑。
  - [x] SubTask 11.2: 用新登录 UI 替换旧 H5 登录页，保留 `authContext` 和 `auth` service 约定。
  - [x] SubTask 11.3: 对接现有登录接口、登出逻辑和 401 回跳逻辑。
  - [x] SubTask 11.4: 记录登录页冗余接口和缺失接口。
  - [x] SubTask 11.5: 运行认证相关测试或 `npm run build`，并更新 `migration-progress.md`。

- [x] Task 12: 收口旧页面、路由和导航
  - [x] SubTask 12.1: 从 H5 用户可达路由和导航中移除旧 H5 有但新移动端没有的页面入口。
  - [x] SubTask 12.2: 保留旧页面源码文件，不做物理删除，等待最终用户确认。
  - [x] SubTask 12.3: 确认所有 retained pages 都能从路由或导航进入。
  - [x] SubTask 12.4: 更新 `page-mapping.md` 和 `migration-progress.md`。

- [x] Task 13: 全量验证和最终迁移报告
  - [x] SubTask 13.1: 运行 `npm run build`。
  - [x] SubTask 13.2: 运行可用的单元测试或关键 e2e 测试；如果测试因既有环境问题失败，记录失败命令、错误摘要和影响范围。
  - [x] SubTask 13.3: 检查 `redundant-interfaces.md` 是否列出所有不再使用但未删除的旧接口。
  - [x] SubTask 13.4: 检查 `missing-interfaces.md` 是否列出所有新 UI 需要但后端缺失的接口。
  - [x] SubTask 13.5: 最终响应用户：说明迁移完成，并询问是否删除旧移动端代码。

# Task Dependencies
- Task 2 depends on Task 1.
- Task 3 depends on Task 2.
- Task 4 depends on Task 3.
- Task 5 depends on Task 4.
- Task 6 depends on Task 5.
- Task 7 depends on Task 6.
- Task 8 depends on Task 7.
- Task 9 depends on Task 8.
- Task 10 depends on Task 9.
- Task 11 depends on Task 10.
- Task 12 depends on Task 11.
- Task 13 depends on Task 12.

# Execution Rule
- 每个页面任务必须完成“新页面识别 → 旧页面识别 → 旧接口梳理 → UI 替换 → 接口和按钮对接 → 差异文档更新 → 验证”后，才能进入下一个页面任务。

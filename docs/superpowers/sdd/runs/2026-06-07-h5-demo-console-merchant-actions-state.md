# SDD Run State - H5 Demo Console Merchant Actions

## Run Metadata

| Key | Value |
| --- | --- |
| Run ID | `2026-06-07-h5-demo-console-merchant-actions` |
| Topic | `h5-demo-console-merchant-actions` |
| Goal | 在 H5 Demo Console 增加商家二级菜单，并完善演示动作，支持即将开播、正在竞拍、一口价、竞拍倒计时压缩等真实链路动作。 |
| Mode | `inline-sdd` |
| Branch | `fix/demo-console-recharge-submenu` |
| Worktree | `/Users/bytedance/.config/superpowers/worktrees/dy-ai-live-auction-fullstack-cc/deploy-local-main` |
| Base Branch | `origin/main` |
| Started At | `2026-06-07 04:36` |
| Owner | `main-agent` |
| Status | `completed` |

## Input Documents

| Type | Path | Required | Loaded |
| --- | --- | --- | --- |
| Agent Rules | `AGENTS.md` | yes | yes |
| Plan | `docs/superpowers/plans/2026-06-07-h5-demo-console-merchant-actions.md` | yes | yes |
| Existing Plan | `docs/superpowers/plans/2026-06-06-h5-demo-console.md` | yes | yes |

## Execution Summary

| Metric | Value |
| --- | --- |
| Total Tasks | `14` |
| Done | `14` |
| Blocked | `0` |
| In Progress | `0` |
| Pending | `0` |
| Last Updated | `2026-06-07 18:19` |

## Task Matrix

| Task ID | Title | Status | Owner | Depends On | Scope |
| --- | --- | --- | --- | --- |
| `T000` | 创建状态文件 | `done` | main-agent | - | SDD SSOT |
| `T001` | auction-service start_time/live_stream_id 契约 | `done` | main-agent | T000 | backend/auction + SDK |
| `T002` | test-service 商家 demo 编排 | `done` | main-agent | T001 | backend/test |
| `T003` | H5 API/context/menu 接线 | `done` | main-agent | T002 | frontend/h5 |
| `T004` | 本地重启与验证 | `done` | main-agent | T003 | local services |
| `T005` | 竞拍延时按钮压缩倒计时到 10 秒 | `done` | main-agent | T003 | backend/auction + backend/test + frontend/h5 |
| `T006` | 修复成交动画与 10 秒防狙击反馈 | `done` | main-agent | T005 | backend/auction + backend/test |
| `T007` | 修复排行榜低于起拍价的异常数值 | `done` | main-agent | T006 | backend/auction + backend/test |
| `T008` | 修复点天灯首次出价低于起拍价 | `done` | main-agent | T007 | backend/auction |
| `T009` | 修复成交动画白色卡片残留 | `done` | main-agent | T006 | frontend/h5 |
| `T010` | 降低竞拍结束到成交动画的结算触发延迟 | `done` | main-agent | T006 | backend/auction |
| `T011` | 自动跟价复用点天灯飘窗提醒 | `done` | main-agent | T008 | backend/auction + frontend/h5 |
| `T012` | 自动跟价事件实时刷新最高价和排行榜 | `done` | main-agent | T011 | frontend/h5 |
| `T013` | 首页无出价竞拍价格回退起拍价 | `done` | main-agent | T007 | frontend/h5 |

## Cross-Task Decisions

| Time | Decision | Reason | Impact | Owner |
| --- | --- | --- | --- | --- |
| `2026-06-07 04:36` | 非直播间点击 `一口价` 时 toast 提示，不禁用按钮 | 全局浮层需要可解释反馈，禁用无法说明原因 | 前端测试覆盖不发请求只提示 | main-agent |
| `2026-06-07 04:36` | 每次商家动作创建新 demo 商品 | 从根上避免同商品活跃竞拍唯一性冲突，保留审计事实 | 后端编排不取消/复用旧记录 | main-agent |
| `2026-06-07 05:04` | demo 商品上架一口价前必须先发布商品 | 固定价服务只接受已发布商品，不能绕过领域发布逻辑 | SDK 增加 `PublishProductAs`，后端编排测试覆盖 publish 调用 | main-agent |
| `2026-06-07 05:22` | `PublishProductAs` 调用真实 `POST /api/v1/products/:id/publish` | 已修复 product-service `PublishHandler` 读取 gateway 透传的 `X-User-ID/X-User-Role`，不再需要 admin update 模拟发布 | 控制面板和独立测试都走真实 publish 链路 | main-agent |
| `2026-06-07 05:22` | 创建竞拍传入 `live_stream_id` 时必须校验直播间归属 | 生产 `/api/v1/auctions` 暴露给商家，不能允许把竞拍挂到他人直播间 | auction-service 通过 product-service 内部接口校验 `creator_id == X-User-ID`，缺少校验器时 fail-closed | main-agent |
| `2026-06-07 16:48` | `竞拍延时` 按钮语义改为把当前竞拍剩余时间压缩到 10 秒 | 用户目标是演示倒计时快速结束，不是触发防狙击“延长”语义 | 新增 demo API `/api/test/demo/auctions/shorten`，auction-service 内部接口更新 `end_time` 并广播 `time_sync` | main-agent |
| `2026-06-07 17:17` | demo 商家竞拍的防狙击窗口调整为 10 秒 | 用户实际演示动作是在最后 10 秒出价，原 demo 规则为 5 秒导致 6-10 秒不会触发 | `PostMerchantAuction` 创建规则时写入 `TriggerDelayBefore: 10`，并增加 handler 测试断言 | main-agent |
| `2026-06-07 17:17` | 竞拍结果通知必须实时推送 | H5 成交动画由 WebSocket `auction_won` notification 触发，仅落库不会播放动画 | `auction_won/auction_lost` notification 设置 `Immediately: true`，并增加 service 测试断言 | main-agent |
| `2026-06-07 17:34` | 出价最低价必须以 `max(current_price, start_price) + increment` 计算 | 只用 `current_price + increment` 会让新竞拍从 0 开始跟价，排行榜出现低于起拍价的 10/20/30 | `BidService` 与 Demo follow-bid 使用同一最低价口径，新增服务层和 handler 测试 | main-agent |
| `2026-06-07 17:42` | 点天灯首次出价复用同一最低价口径 | 修复 `BidService` 后，点天灯仍用旧口径计算首次出价会被后端正确拒绝 | `SkyLampService.StartSubscription` 改为 `minimumBidAmount`，新增真实订阅回归测试 | main-agent |
| `2026-06-07 17:47` | 成交动画关闭定时器不依赖父组件回调身份 | 直播间倒计时/状态更新会触发父组件重渲染，内联 `onAnimationEnd` 会重置定时器，导致白色成交卡片残留 | `BidSuccessAnimation` 使用 ref 保存最新回调，挂载后 3 秒固定触发卸载 | main-agent |
| `2026-06-07 17:52` | 保持动画由真实中标通知触发，缩短后端到期扫描间隔 | 提前在前端本地播放可能误判 winner；延时主要来自后端 1 秒定时扫描 | Scheduler 竞拍状态检查间隔从 1 秒降到 200ms，保留结算/通知 SSOT | main-agent |
| `2026-06-07 18:02` | 点天灯自动跟价成功必须广播可见事件 | 只更新 DB/排行会让用户无法理解系统已自动守价 | 自动跟价成功后广播 `sky_lamp_auto_bid`，H5 复用点天灯飘窗展示自动跟价金额 | main-agent |
| `2026-06-07 18:08` | `sky_lamp_auto_bid` 必须驱动价格和排行榜本地更新 | 普通 `rank_update` 有 200ms 节流，用户出价和自动跟价间隔过短时，自动跟价后的排名广播可能被抑制 | H5 收到 `sky_lamp_auto_bid` 后立即更新 `current_price` 和本地 ranking，并重排列表 | main-agent |
| `2026-06-07 18:19` | 首页卡片无出价时展示起拍价而不是 0，后端列表必须提供起拍价 | 新竞拍的 `current_price=0` 是无出价状态，不是业务最低价；前端回退逻辑依赖列表接口提供 `start_price` | `/api/v1/auctions` 列表按本页 `product_id` 批量查询 `auction_rules` 并返回 `start_price`，H5 使用 `current_price > 0 ? current_price : start_price` | main-agent |

## Test Commands

| Area | Command | Required | Last Result | Notes |
| --- | --- | --- | --- | --- |
| Backend Auction | `cd backend/auction && go test ./handler ./service -run 'Test.*CreateAuction.*' -count=1 -v` | yes | `pass` | start_time/live_stream_id |
| Backend Auction Full Target | `cd backend/auction && go test ./handler ./service -count=1` | yes | `pass` | handler/service 回归 |
| Backend Auction Shorten | `cd backend/auction && go test ./handler -run 'TestInternalDemoAuctionShorten' -count=1 -v` | yes | `pass` | 更新 end_time + WebSocket time_sync |
| Backend Auction Handler/DAO | `cd backend/auction && go test ./handler ./dao -count=1 && go build ./...` | yes | `pass` | handler/dao 回归 + 编译 |
| Backend Product Publish | `cd backend/product && go test ./handler -run 'TestProductHandlerPublishUsesForwardedUserHeaders|TestProductHandler_AdminCreate' -count=1 -v` | yes | `pass` | publish header 鉴权 |
| Backend Product Handler | `cd backend/product && go test ./handler -count=1` | yes | `pass` | handler 回归 |
| Backend Test Service | `cd backend/test && go test ./handler/ -run 'TestMerchantDemo|TestValidateRechargeRequest|TestDemoUserIDFromAuthorization' -count=1 -v` | yes | `pass` | demo 编排 |
| Backend SDK | `cd backend/test && go test ./client/auction/ -run 'TestSDK_CreateAuction|TestSDK_CreateFixedPriceItem|TestSDK_PublishProductAs' -count=1 -v` | yes | `pass` | SDK body + 发布商品 |
| Backend Test Target | `cd backend/test && go test ./client/auction ./handler -count=1` | yes | `pass` | SDK + handler 回归 |
| Backend Test Shorten | `cd backend/test && go test ./handler -run 'TestDemoShortenAuction' -count=1 -v && go test ./client/auction -run 'TestSDK_ShortenAuction' -count=1 -v` | yes | `pass` | demo 白名单 + internal SDK |
| Backend Test Handler/SDK | `cd backend/test && go test ./handler ./client/auction -count=1 && go build ./...` | yes | `pass` | handler/SDK 回归 + 编译 |
| Backend Test Build | `cd backend/test && go build ./...` | yes | `pass` | 编译验证 |
| Backend Auction Realtime Notification | `cd backend/auction && go test ./service -run TestEndAuctionCreatesPendingOrderBeforeWinnerNotification -count=1 -v` | yes | `pass` | 中标通知必须 `Immediately=true`，用于 H5 成交动画 |
| Backend Demo Anti-Snipe Window | `cd backend/test && go test ./handler -run TestMerchantDemoAuctionCreatesFreshProductsForRepeatedOngoingClicks -count=1 -v` | yes | `pass` | demo 商家竞拍规则窗口必须为 10 秒 |
| Backend Focus Regression | `cd backend/auction && go test ./service ./handler -run 'TestEndAuction|TestCapPrice|TestDelay|TestStateMachine|TestInternalDemoAuctionShorten' -count=1` | yes | `pass` | 结算、防狙击、压缩倒计时回归 |
| Backend Demo Focus Regression | `cd backend/test && go test ./handler ./client/auction -run 'TestMerchantDemo|TestDemoShorten|TestSDK_|TestValidateRechargeRequest|TestDemoUserIDFromAuthorization' -count=1` | yes | `pass` | demo 编排、白名单、SDK 回归 |
| Backend Bid Start Price Guard | `cd backend/auction && go test ./service -run 'TestPlaceBidRejectsAmountBelowStartPrice|TestPlaceBidAtCapPriceFinalizesAuctionResult|TestMinimumBidAmount' -count=1` | yes | `pass` | 出价下限不能低于起拍价口径 |
| Backend SkyLamp Start Price Guard | `cd backend/auction && go test ./service -run 'TestSkyLampStartSubscriptionUsesStartPriceForInitialBid|TestSkyLamp|TestPlaceBidRejectsAmountBelowStartPrice|TestMinimumBidAmount' -count=1` | yes | `pass` | 点天灯首次出价从起拍价加价开始 |
| Backend Scheduler Responsiveness | `cd backend/auction && go test ./service -run 'TestSchedulerDefaultAuctionCheckIntervalKeepsEndAnimationResponsive|TestEndAuction|TestSkyLamp|TestPlaceBid|TestMinimumBidAmount|TestCapPrice|TestDelay|TestStateMachine' -count=1` | yes | `pass` | 竞拍到期扫描间隔降低到 200ms |
| Backend SkyLamp Auto Bid Broadcast | `cd backend/auction && go test ./service -run 'TestSkyLampBroadcast|TestSkyLampStartSubscriptionUsesStartPriceForInitialBid|TestSkyLamp' -count=1` | yes | `pass` | 自动跟价成功广播 `sky_lamp_auto_bid` |
| Backend Test Follow Bid Guard | `cd backend/test && go test ./handler -run 'TestComputeFollowBidAmount|TestMerchantDemo' -count=1` | yes | `pass` | Demo 跟价从起拍价加价开始 |
| H5 Bid Success Animation | `cd frontend/h5 && npx jest src/components/auction/__tests__/BidSuccessAnimation.test.tsx --runInBand` | yes | `pass` | 父组件重渲染不重置自动关闭定时器 |
| H5 Live Regression | `cd frontend/h5 && npx jest src/pages/Live/__tests__/LiveRoom.test.tsx src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand` | yes | `pass` | 直播间成交动画链路回归 |
| H5 SkyLamp Realtime Ranking | `cd frontend/h5 && npx jest src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand` | yes | `pass` | 自动跟价事件会立刻更新最高价、排行榜和飘窗 |
| H5 Home Start Price Fallback | `cd frontend/h5 && npx jest src/pages/Home/__tests__/Home.test.tsx --runInBand` | yes | `pass` | 首页无出价竞拍显示起拍价，不显示 0 |
| Backend Auction List Start Price | `cd backend/auction && go test ./handler -count=1 && go build ./...` | yes | `pass` | 列表接口返回 `start_price` 供首页回退展示 |
| H5 Jest | `cd frontend/h5 && npx jest src/components/DemoConsole/__tests__/DemoConsole.test.tsx src/services/__tests__/demoApi.test.ts src/store/__tests__/demoContext.test.tsx src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand` | yes | `pass` | UI/context/api |
| H5 Shorten Jest | `cd frontend/h5 && npx jest src/services/__tests__/demoApi.test.ts src/components/DemoConsole/__tests__/DemoConsole.test.tsx --runInBand` | yes | `pass` | 竞拍延时按钮和 API |
| H5 Build | `cd frontend/h5 && npm run build` | yes | `pass` | production build |
| Diff Check | `git diff --check` | yes | `pass` | whitespace |

## Local Smoke Evidence

| Action | Result |
| --- | --- |
| Restart backend | `scripts/start-local-backend.sh restart` 成功；gateway/product/auction 分别监听 `8080/8081/8082`，auction WS 监听 `8083` |
| Restart test-service | `go run main.go` 成功；test-service 监听 `18090/18092` |
| `POST /api/test/demo/merchant/auctions {"mode":"upcoming"}` | `200`，返回 `auction_id/product_id/live_stream_id/start_time/end_time` |
| `POST /api/test/demo/merchant/auctions {"mode":"ongoing"}` | `200`，返回 `auction_id/product_id/live_stream_id/start_time/end_time` |
| `POST /api/test/demo/merchant/fixed-price-items {"live_stream_id":2}` | `200`，返回 `item_id/product_id/live_stream_id/price/stock`；链路为 create product -> publish product -> create fixed-price item |
| 用户反馈：最后 10 秒出价未触发延时 | 根因是 demo 商家规则窗口为 5 秒；修复后新建 demo 竞拍 `auction_id=4/product_id=8` 规则为 `trigger_delay_before=10`，shorten 到 10 秒后 follow-bid 可触发延时，DB 观测 `delay_used=5`、`winner_id=9102` |
| 用户反馈：竞拍成功动画未出现 | 根因是 `auction_won` notification 未实时 WebSocket 推送；修复后通知请求 `Immediately=true`，结算后 notification 落库且可被 H5 `notification` listener 消费 |
| 用户反馈：排行榜数值不对 | 根因是出价下限忽略 `start_price`，导致 demo 新竞拍能从 `0 + increment` 开始出价；修复后新建 `auction_id=6/product_id=10`，`follow-bid` 返回 `amount=110`，排行榜接口返回 `110` |
| 用户反馈：点天灯首次出价失败 | 根因是点天灯首次出价仍用旧的 `current_price + increment` 口径；修复后新建 `auction_id=8/product_id=12`，`POST /api/v1/sky-lamp/subscriptions` 返回 `initial_bid_amount=110`，排行榜接口返回 `110` |
| 用户反馈：成交动画展示后白色卡片应消失 | 根因是父组件重渲染导致 `BidSuccessAnimation` 的 3 秒关闭定时器被反复重置；修复后定时器只在组件挂载时启动，3 秒后调用最新 `onAnimationEnd` 卸载卡片 |
| 用户反馈：竞拍结束到锤子动画出现中间有延时 | 根因是后端每 1 秒扫描一次已到期竞拍，`auction_won` 通知必须等扫描触发结算后才推送；修复后状态扫描间隔为 200ms，并已重启 `auction-service` 生效 |
| 用户反馈：点天灯自动跟价没有提醒和动画 | 根因是 `sky_lamp_auto_bid` 契约已存在但自动跟价成功后未广播，前端也未监听；修复后服务端广播该事件，H5 复用点天灯飘窗显示“自动跟价 ¥金额” |
| 用户反馈：自动跟价后最高价和排行榜没有实时更新 | 根因是自动跟价发生在普通出价后很短时间内，`rank_update` 可能被节流；前端原先只用 `sky_lamp_auto_bid` 展示飘窗，没有同步更新价格/排行 |
| 用户反馈：主页最低价显示 0 | 根因是首页卡片直接展示 `current_price`，且 `/api/v1/auctions` 列表未返回 `start_price`，新竞拍无出价时只能拿到 `current_price=0`；修复后列表返回 `start_price=100`，前端回退展示起拍价，仍保留“暂无出价”文案 |

## T006 Delivery Record

| Field | Value |
| --- | --- |
| Scope | 修复 demo 竞拍 10 秒防狙击窗口与竞拍结果实时通知 |
| Dependencies | T005 已提供 shorten 到 10 秒的 demo API 与 WebSocket `time_sync` 广播 |
| Root Cause 1 | `PostMerchantAuction` 创建 demo rule 时 `TriggerDelayBefore=5`，用户在剩余 6-10 秒出价不会触发延时 |
| Root Cause 2 | `SendAuctionResultNotifications` 未设置 `Immediately=true`，H5 只能收到落库通知，无法通过 WebSocket 触发成交动画 |
| Changed Files | `backend/test/handler/demo.go`, `backend/test/handler/demo_test.go`, `backend/auction/service/auction_settlement.go`, `backend/auction/service/auction_test.go` |
| Validation Commands | `go test` focused regression, `go build ./...`, `git diff --check` |
| Risk | 代码和服务端 smoke 已验证；浏览器端成交动画还需要刷新 H5 后按真实演示路径肉眼确认 |
| Delivery Conclusion | 已完成修复，当前实现满足“最后 10 秒出价触发延时”和“竞拍成功实时推送动画触发事件” |

## T007 Delivery Record

| Field | Value |
| --- | --- |
| Scope | 修复排行榜显示低于起拍价的异常金额 |
| Dependencies | T002/T005 的 Demo 商家竞拍与 follow-bid 编排 |
| Root Cause | `BidService.PlaceBid` 和 Demo follow-bid 都以 `current_price + increment` 为最低出价；新竞拍 `current_price=0` 时会接受 `10/20/30` 等低于 `start_price=100` 的出价 |
| Changed Files | `backend/auction/service/bid.go`, `backend/auction/service/bid_test.go`, `backend/auction/service/bid_cap_price_settlement_test.go`, `backend/test/handler/demo.go`, `backend/test/handler/demo_test.go`, `backend/test/client/auction/client.go` |
| Validation Commands | `go test` focused regression, `go build ./...`, `git diff --check` |
| Runtime Evidence | 重启 `auction-service/test-service` 后，新建 demo auction `6`，`POST /api/test/demo/follow-bid` 返回 `amount=110`，`GET /api/v1/auctions/6/ranking` 返回 `110` |
| Risk | 已有脏数据 auction `5` 仍保留历史出价 `10/20/30`，需要新建竞拍验证修复效果 |
| Delivery Conclusion | 已完成修复，新出价和 Demo 跟价统一遵循起拍价下限 |

## T008 Delivery Record

| Field | Value |
| --- | --- |
| Scope | 修复点天灯开启时首次出价低于起拍价导致失败 |
| Dependencies | T007 的统一最低出价口径 |
| Root Cause | `SkyLampService.StartSubscription` 仍以 `auction.CurrentPrice + rule.Increment` 计算首次出价；新竞拍 `current_price=0` 时会提交 `10`，被 `BidService` 新校验拒绝 |
| Changed Files | `backend/auction/service/sky_lamp.go`, `backend/auction/service/sky_lamp_test.go` |
| Validation Commands | `go test ./service ./handler -run 'TestSkyLamp|TestPlaceBid|TestMinimumBidAmount|TestCapPrice|TestDelay|TestStateMachine|TestInternalDemoAuctionShorten' -count=1`, `go build ./...`, `git diff --check` |
| Runtime Evidence | 重启 `auction-service` 后，新建 demo auction `8`，开启点天灯返回 `initial_bid_amount=110`，排行榜接口返回 `110` |
| Risk | 已有失败的点天灯请求不会自动重放；刷新后对新竞拍重新开启即可 |
| Delivery Conclusion | 已完成修复，点天灯首次出价与普通出价、Demo 跟价使用同一最低价口径 |

## T009 Delivery Record

| Field | Value |
| --- | --- |
| Scope | 修复竞拍成功动画结束后白色成交卡片残留 |
| Dependencies | T006 的实时 `auction_won` notification 触发动画 |
| Root Cause | `BidSuccessAnimation` 的关闭 timer 依赖 `onAnimationEnd`，父组件每次重渲染都会传入新函数并重置 timer |
| Changed Files | `frontend/h5/src/components/auction/BidSuccessAnimation.tsx`, `frontend/h5/src/components/auction/__tests__/BidSuccessAnimation.test.tsx` |
| Validation Commands | `npx jest src/components/auction/__tests__/BidSuccessAnimation.test.tsx --runInBand`, `npx jest src/pages/Live/__tests__/LiveRoom.test.tsx src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand`, `npm run build`, `git diff --check` |
| Runtime Evidence | 当前 H5 dev server 使用 Vite，源码变更会通过 HMR 生效；测试覆盖父组件重渲染后 3 秒仍触发关闭 |
| Risk | 未做浏览器肉眼复测；行为由组件计时器测试和直播间回归覆盖 |
| Delivery Conclusion | 已完成修复，动画展示完成后会自动卸载白色成交卡片 |

## T010 Delivery Record

| Field | Value |
| --- | --- |
| Scope | 降低竞拍结束到锤子成交动画出现之间的等待时间 |
| Dependencies | T006 的实时 `auction_won` notification |
| Root Cause | `Scheduler` 每 1 秒扫描已到 `end_time` 的竞拍，扫描到后才执行 `EndAuction -> FinalizeEndedAuction -> auction_won notification` |
| Changed Files | `backend/auction/service/scheduler.go`, `backend/auction/service/scheduler_test.go` |
| Validation Commands | `go test ./service -run 'TestSchedulerDefaultAuctionCheckIntervalKeepsEndAnimationResponsive|TestEndAuction|TestSkyLamp|TestPlaceBid|TestMinimumBidAmount|TestCapPrice|TestDelay|TestStateMachine' -count=1`, `go test ./handler -run TestInternalDemoAuctionShorten -count=1`, `go build ./...`, `git diff --check` |
| Runtime Evidence | 已重启 `auction-service`；日志显示同一秒内多次执行到期扫描，符合 200ms 检查间隔 |
| Risk | 仍保留订单/通知写入耗时；不会通过前端本地预测提前播放，避免误判 winner |
| Delivery Conclusion | 已完成修复，结算触发抖动从最多约 1 秒降低到最多约 200ms |

## T011 Delivery Record

| Field | Value |
| --- | --- |
| Scope | 自动跟价成功后复用点天灯飘窗提醒 |
| Dependencies | T008 的点天灯出价链路修复 |
| Root Cause | `SkyLampService.processOneSubscription` 自动跟价成功后只更新订阅统计和指标，没有广播 `sky_lamp_auto_bid`；H5 只在开启点天灯时展示飘窗 |
| Changed Files | `backend/auction/service/sky_lamp.go`, `backend/auction/service/sky_lamp_broadcast_test.go`, `backend/auction/main.go`, `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`, `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx` |
| Validation Commands | `go test ./service -run 'TestSkyLampBroadcast|TestSkyLampStartSubscriptionUsesStartPriceForInitialBid|TestSkyLamp' -count=1`, `npx jest src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand`, `go test ./service ./handler -run 'TestSkyLamp|TestPlaceBid|TestMinimumBidAmount|TestCapPrice|TestDelay|TestStateMachine|TestInternalDemoAuctionShorten' -count=1`, `go build ./...`, `npm run build`, `git diff --check` |
| Runtime Evidence | 已重启 `auction-service`，`8082/8083` 正常监听；H5 dev server 通过 Vite HMR 加载前端变更 |
| Risk | 未做浏览器肉眼复测；已由 WS handler 单测覆盖自动跟价飘窗展示和跨房过滤 |
| Delivery Conclusion | 已完成修复，自动跟价成功会广播并触发点天灯飘窗重播 |

## T012 Delivery Record

| Field | Value |
| --- | --- |
| Scope | 修复点天灯自动跟价后当前最高价和排行榜不实时更新 |
| Dependencies | T011 的 `sky_lamp_auto_bid` 广播与前端监听 |
| Root Cause | `BidService.broadcastRanking` 有 200ms 节流；用户B出价和用户A自动跟价太接近时，自动跟价后的 `rank_update` 可能被抑制，H5 又没有从 `sky_lamp_auto_bid` 更新价格/排行 |
| Changed Files | `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`, `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx` |
| Validation Commands | `npx jest src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand`, `npm run build`, `git diff --check` |
| Runtime Evidence | H5 dev server 为 Vite，前端变更通过 HMR 生效；测试覆盖 `130/B=140/A=130 -> sky_lamp_auto_bid A=150` 后最高价和排行榜立刻更新 |
| Risk | 仅本地乐观更新 top ranking；后续完整 `rank_update/sync_response` 仍会校正最终排行 |
| Delivery Conclusion | 已完成修复，自动跟价事件现在会立刻刷新最高价、排行榜和点天灯飘窗 |

## T013 Delivery Record

| Field | Value |
| --- | --- |
| Scope | 修复首页竞拍卡片无出价时价格显示为 0 |
| Dependencies | T007 的起拍价/最低出价统一口径 |
| Root Cause | 首页 `normalizeAuction` 直接使用 `auction.current_price ?? 0`，而新竞拍未出价时 `current_price=0`，实际展示价格应回退到 `start_price` |
| Changed Files | `backend/auction/dao/auction_rule.go`, `backend/auction/handler/auction_list.go`, `backend/auction/handler/auction.go`, `backend/auction/handler/auction_detail.go`, `backend/auction/handler/auction_list_test.go`, `backend/auction/handler/auction_detail_test.go`, `frontend/h5/src/pages/Home/index.tsx`, `frontend/h5/src/pages/Home/__tests__/Home.test.tsx` |
| Validation Commands | `go test ./handler -run 'TestBuildAuctionListResponse|TestAuctionListResponseShape|TestBuildAuctionDetailResponseIncludesAuctionRule' -count=1`, `go test ./handler -count=1`, `go build ./...`, `npx jest src/pages/Home/__tests__/Home.test.tsx --runInBand`, `npm run build`, `git diff --check` |
| Runtime Evidence | 已重启 `auction-service`；`curl /api/v1/auctions?page=1&page_size=1` 返回 `current_price:"0"` 同时包含 `start_price:"100"`；H5 dev server 为主仓库源码且包含回退逻辑 |
| Risk | 仅影响首页列表展示价和列表响应新增字段；竞拍详情和直播间已有独立起拍价回退逻辑 |
| Delivery Conclusion | 已完成修复，首页无出价竞拍显示起拍价而不是 0 |

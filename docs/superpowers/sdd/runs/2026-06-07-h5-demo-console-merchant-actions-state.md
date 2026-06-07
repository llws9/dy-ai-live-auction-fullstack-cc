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
| Total Tasks | `6` |
| Done | `6` |
| Blocked | `0` |
| In Progress | `0` |
| Pending | `0` |
| Last Updated | `2026-06-07 16:48` |

## Task Matrix

| Task ID | Title | Status | Owner | Depends On | Scope |
| --- | --- | --- | --- | --- |
| `T000` | 创建状态文件 | `done` | main-agent | - | SDD SSOT |
| `T001` | auction-service start_time/live_stream_id 契约 | `done` | main-agent | T000 | backend/auction + SDK |
| `T002` | test-service 商家 demo 编排 | `done` | main-agent | T001 | backend/test |
| `T003` | H5 API/context/menu 接线 | `done` | main-agent | T002 | frontend/h5 |
| `T004` | 本地重启与验证 | `done` | main-agent | T003 | local services |
| `T005` | 竞拍延时按钮压缩倒计时到 10 秒 | `done` | main-agent | T003 | backend/auction + backend/test + frontend/h5 |

## Cross-Task Decisions

| Time | Decision | Reason | Impact | Owner |
| --- | --- | --- | --- | --- |
| `2026-06-07 04:36` | 非直播间点击 `一口价` 时 toast 提示，不禁用按钮 | 全局浮层需要可解释反馈，禁用无法说明原因 | 前端测试覆盖不发请求只提示 | main-agent |
| `2026-06-07 04:36` | 每次商家动作创建新 demo 商品 | 从根上避免同商品活跃竞拍唯一性冲突，保留审计事实 | 后端编排不取消/复用旧记录 | main-agent |
| `2026-06-07 05:04` | demo 商品上架一口价前必须先发布商品 | 固定价服务只接受已发布商品，不能绕过领域发布逻辑 | SDK 增加 `PublishProductAs`，后端编排测试覆盖 publish 调用 | main-agent |
| `2026-06-07 05:22` | `PublishProductAs` 调用真实 `POST /api/v1/products/:id/publish` | 已修复 product-service `PublishHandler` 读取 gateway 透传的 `X-User-ID/X-User-Role`，不再需要 admin update 模拟发布 | 控制面板和独立测试都走真实 publish 链路 | main-agent |
| `2026-06-07 05:22` | 创建竞拍传入 `live_stream_id` 时必须校验直播间归属 | 生产 `/api/v1/auctions` 暴露给商家，不能允许把竞拍挂到他人直播间 | auction-service 通过 product-service 内部接口校验 `creator_id == X-User-ID`，缺少校验器时 fail-closed | main-agent |
| `2026-06-07 16:48` | `竞拍延时` 按钮语义改为把当前竞拍剩余时间压缩到 10 秒 | 用户目标是演示倒计时快速结束，不是触发防狙击“延长”语义 | 新增 demo API `/api/test/demo/auctions/shorten`，auction-service 内部接口更新 `end_time` 并广播 `time_sync` | main-agent |

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

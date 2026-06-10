# SDD Run State - 2026-06-10-h5-home-auction-viewer-count

> SSOT for this SDD run. Updated by main agent before/after each task.

## Run Metadata

| Key | Value |
| --- | --- |
| Run ID | `2026-06-10-2026-06-10-h5-home-auction-viewer-count` |
| Topic | `2026-06-10-h5-home-auction-viewer-count` |
| Goal | `H5 首页进行中普通竞拍卡片展示真实快照观看人数` |
| Mode | `subagent-driven` |
| Branch | `feat/h5-home-viewer-count` |
| Worktree | `/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/.worktrees/feat-h5-home-viewer-count` |
| Base Commit | `3f55af4a97147925cdea9159f25f4556dcaa798c` |
| Target Branch | `main` |
| Worktree Dirty | `no (fresh worktree + copied docs)` |
| Started At | `2026-06-10 07:43` |
| Owner | `main-agent` |
| Status | `active` |

## Worktree Selection Note

RUNBOOK 默认 worktree 路径 `/Users/bytedance/.config/superpowers/worktrees/...` 在当前沙盒 allowlist 外不可用，故偏离至仓库内 `.worktrees/feat-h5-home-viewer-count`（已有 `.worktrees/feat-live-reminder-modal-v1` 先例，且 `.worktrees` 已 gitignored）。H5 `node_modules` 已随 worktree 创建可用（jest/tsc/build 可跑）。用户已确认此选址。

## Input Documents

| Type | Path | Required | Loaded |
| --- | --- | --- | --- |
| Agent Rules | `AGENTS.md` | yes | yes |
| SDD Runbook | `docs/superpowers/sdd/RUNBOOK.md` | yes | yes |
| Plan | `docs/superpowers/plans/2026-06-10-h5-home-auction-viewer-count.md` | yes | yes |
| Spec | `docs/superpowers/specs/2026-06-10-h5-home-auction-viewer-count-design.md` | yes | yes |

## Execution Summary

| Metric | Value |
| --- | --- |
| Total Tasks | `5` |
| Done | `4` |
| Blocked | `0` |
| In Progress | `0` |
| Pending | `1 (T005 部署验证待用户决定)` |
| Last Updated | `2026-06-10 08:10` |

## Key Decisions (from定稿 spec)

- 降级语义：viewer_count 批量查询失败仅降级（填 0、整页 200），商品摘要失败仍维持原 5xx。
- 后端 batch 接口不做 status 过滤（纯数据语义），过滤交前端。
- 前端显示双条件：`statusInfo.live && viewerCount > 0`，判定主语是 `auction.status`。
- viewer_count 为直播间维度（同 live_stream_id 多卡片显示相同人数为预期）。

## Task Matrix

| Task ID | Title | Status | Owner | Wave | Depends On | Write Set | Read Set | Regression Sentinels |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `T001` | product 内部批量接口回填 viewer_count | `pending` | unassigned | `W1` | `-` | `backend/product/handler/internal.go`, `backend/product/main.go`, `backend/product/handler/internal_test.go` | `backend/product/service/live_stream.go`, plan/spec | `TestInternalHandler_BatchLiveStreams_ViewerCountRedisFirst/DBFallback` |
| `T002` | auction client 透传 viewer_count | `pending` | unassigned | `W1` | `-` | `backend/auction/client/live_stream_client.go`, `backend/auction/client/live_stream_client_test.go` | plan/spec | `TestHTTPLiveStreamClient_BatchDecodesViewerCount` |
| `T003` | auction 列表编排回填 viewer_count（含降级） | `pending` | unassigned | `W2` | `T002` | `backend/auction/handler/auction_list.go`, `backend/auction/handler/auction.go`, `backend/auction/handler/auction_list_test.go` | `backend/auction/client/live_stream_client.go`, plan/spec | `TestBuildAuctionListResponse_ViewerCount`（含降级不 5xx 子测试） |
| `T004` | H5 首页类型扩展 + 进行中卡片渲染 pill | `pending` | unassigned | `W1` | `-` | `frontend/h5/src/pages/Home/index.tsx`, `frontend/h5/src/pages/Home/Home.module.css`, `frontend/h5/src/pages/Home/__tests__/Home.test.tsx` | plan/spec | `Home.test.tsx` 3 用例（展示/降级 0 不展示/已结束不展示） |
| `T005` | 本地联调与部署验证 | `pending` | unassigned | `W3` | `T001,T003,T004` | `-（验证 only）` | 全链路 | curl `viewer_count` 透出 + 降级 + H5 视觉 |

## Wave Plan

| Wave | Goal | Tasks | Start Condition | Completion Condition |
| --- | --- | --- | --- | --- |
| `W1` | 无写集重叠的契约/前端并行实现 | `T001`, `T002`, `T004` | state 就绪 | 三任务 TDD 通过并评审 |
| `W2` | auction 列表编排回填（依赖 T002 字段） | `T003` | T002 done | TDD 通过并评审 |
| `W3` | 本地部署联调验证 | `T005` | T001/T003/T004 done | 接口透出 + 降级 + 视觉确认 |

> 写集分析：T001(product/) / T002(auction/client/) / T004(h5 Home/) 三者文件无交集且互不依赖 → 可并行。T003 写 auction/handler/ 与 T002 无文件交集，但依赖 T002 新增的 `LiveStreamSummary.ViewerCount` 字段 → 串行排 W2。

## Task Records

### T001 - product 内部批量接口回填 viewer_count
| Key | Value |
| --- | --- |
| Status | `done` |
| Verification | Red: NewInternalHandler 2 参编译失败；Green: TestInternalHandler_BatchLiveStreams_ViewerCountRedisFirst/DBFallback PASS；build ./... OK；go test ./handler/ ./service/ PASS |
| Scope Expansion | 批准：越界改 `internal_live_stream_test.go`(1) + `admin_route_test.go`(4) 补 `nil` 入参（签名变更必要的机械适配，不改测试语义） |
| Modified Files | internal.go, main.go, internal_test.go, internal_live_stream_test.go, admin_route_test.go |
| Commit | `b90b8b91` |

### T002 - auction client 透传 viewer_count
| Key | Value |
| --- | --- |
| Status | `done` |
| Verification | Red: ViewerCount undefined；Green: TestHTTPLiveStreamClient_BatchDecodesViewerCount PASS；build + go test ./client/ PASS |
| Modified Files | live_stream_client.go, live_stream_client_test.go |
| Commit | `e1c2dc73` |

### T003 - auction 列表编排回填（含降级）
| Key | Value |
| --- | --- |
| Status | `done` |
| Verification | Red: 6 参签名/ViewerCount 字段编译失败；Green: TestBuildAuctionListResponse_ViewerCount 4 子测试 PASS（含降级不 5xx）+ 既有 7 用例不回归；build + go test ./... PASS |
| Modified Files | auction_list.go, auction.go, auction_list_test.go |
| Commit | `be210d81` |

### T004 - H5 首页渲染 pill
| Key | Value |
| --- | --- |
| Status | `done` |
| Verification | Red: 128 观看 断言失败；Green: jest 31 passed 全通过；tsc/build 因既有 zustand 缺失报错（write set 外、git stash 复核确认无关） |
| Modified Files | Home/index.tsx, Home/Home.module.css, Home/__tests__/Home.test.tsx |
| Commit | `8634188f` |

### T005 - 本地联调验证
| Key | Value |
| --- | --- |
| Status | `pending（待用户决定是否 deploy-dev 联调）` |
| Verification | `not_run` |

## Runtime Sources

| Service | Command | Branch | Worktree | Commit | Dirty | Ports | Owner |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `-` | `-` | `-` | `-` | `-` | `-` | `-` | `-` |

## Final Handoff

当前分支/worktree：feat/h5-home-viewer-count @ /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/.worktrees/feat-h5-home-viewer-count

**状态**

- `4/5 任务 done（T001-T004 全 TDD 通过并提交）。T005 部署联调 + 合入 main 待用户决定。`
- `diff vs main 仅 13 文件，范围 = 三块 write set + 批准的 5 行调用方适配。backend product/auction 全包测试通过；H5 jest 31 passed。`

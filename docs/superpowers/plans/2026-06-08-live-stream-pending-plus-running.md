# 实施计划：直播间「1 进行中 + 1 即将开始」竞拍约束

- **关联 Spec**：`docs/superpowers/specs/2026-06-08-live-stream-pending-plus-running-design.md`
- **创建日期**：2026-06-08
- **执行分支**：`feat/live-stream-pending-plus-running`
- **目标分支**：`main`
- **执行模式**：SDD + TDD

---

## T1 — 数据库约束拆分

**范围**

- 新增 migration：拆掉旧 `active_live_stream_key` / `uk_active_live_stream`，新增 `pending_live_stream_key` / `running_live_stream_key` 与对应唯一索引。
- 更新运行时 ensure：`backend/auction/dao/auction_schema.go`。
- 更新启动调用点：`backend/auction/main.go`。

**Write Set**

- `backend/migrations/*split_live_stream_active_unique*.sql`
- `backend/auction/dao/auction_schema.go`
- `backend/auction/main.go`

**验证**

- `cd backend/auction && go test ./dao ./...`
- MySQL schema 中存在 `uk_pending_live_stream`、`uk_running_live_stream`，不存在旧 `uk_active_live_stream`。

## T2 — DAO 查询细分

**范围**

- 新增 `GetPendingByLiveStreamID(ctx, liveStreamID)`。
- 新增 `GetRunningByLiveStreamID(ctx, liveStreamID)`。
- 保留或移除旧 `GetActiveByLiveStreamID`，以最终调用关系为准。

**Write Set**

- `backend/auction/dao/auction.go`
- `backend/auction/dao/*test.go`

**验证**

- DAO 单测覆盖 pending/running 命中与未命中。

## T3 — CreateAuction 语义改造

**范围**

- 创建竞拍时仅拒绝同直播间已有 `Pending`。
- 已有 `Ongoing/Delayed` 时允许创建新的 `Pending`。
- 新增 `ErrPendingLiveStreamAuctionExists`，handler 映射为 409。
- 唯一约束冲突映射识别 `uk_pending_live_stream`。

**Write Set**

- `backend/auction/service/auction.go`
- `backend/auction/handler/auction.go`
- `backend/auction/service/auction_test.go`

**Regression Sentinels**

- 已有 `Ongoing` 时创建另一商品 Pending 成功。
- 已有 `Pending` 时创建第二个 Pending 失败。
- 同商品 active 仍失败。

## T4 — StartAuction 兜底与时间重算

**范围**

- `StartAuction` 在 `Pending -> Ongoing` 前检查同直播间 running 占用。
- running 忙时保持 `Pending` 并让 scheduler 静默跳过。
- 开始时按原始 duration 重算 `StartTime/EndTime`。

**Write Set**

- `backend/auction/service/auction.go`
- `backend/auction/service/auction_test.go`

**Regression Sentinels**

- 同直播间已有 `Ongoing` 时，Pending 到点不会被启动。
- running 结束后，Pending 下一次启动且 `EndTime = now + 原始时长`。
- 延迟启动不会秒结束。

## T5 — 验证与交付

**范围**

- 更新 SDD 状态文件。
- 执行后端 auction 测试。
- 复核 diff，确认没有引入 H5 首页优化或其他无关改动。

**验证命令**

```bash
cd backend/auction && go test ./...
git diff main...HEAD --name-only
```

**交付标准**

- T1-T4 全部完成并有测试证据。
- 状态文件记录任务结果、验证命令、风险与结论。
- 最终报告包含 branch/worktree。

# Debug: auction-status-stale

Status: [OPEN]

## Symptom

H5 首页直播间/竞拍卡片显示 `开拍 2026/6/5 01:40:00`，当前时间已晚于开拍时间，但状态 Badge 仍为 `即将开始`。

## Hypotheses

1. 前端状态计算只依赖后端 `status`，未在 `status=0` 且 `start_time <= now` 时派生为进行中或已结束。
2. 后端列表接口未在读取前执行状态推进，导致数据库中 `status=0` 的过期开拍记录继续原样返回。
3. 定时任务或状态机存在未启动/未被调用的问题，`UpdateExpiredAuctions` / `UpdateStartedAuctions` 没有持续推进。
4. 前后端时间格式或时区解析不一致，导致 `start_time` 在浏览器显示为 01:40，但程序判断仍未到时间。
5. 首页卡片使用的是拍品竞拍状态，不是直播间状态，用户视觉上把直播间状态和竞拍状态混在一起。

## Evidence Log

- Static frontend evidence: `frontend/h5/src/pages/Home/index.tsx` maps `status=0` directly to `即将开始`; only `status=1/2` with expired `end_time` is downgraded to `已结束`.
- Static backend evidence: `backend/auction/service/scheduler.go` starts a 1-second scheduler; `backend/auction/dao/auction.go` uses MySQL `NOW()` in `GetPendingAuctionsToStart` and `GetExpiredAuctions`.
- Runtime API evidence: `GET http://localhost:8082/api/v1/auctions?page=1&page_size=10` returned `id=993304 status=0 start_time=2026-06-05T01:40:00+08:00 end_time=2026-06-05T03:00:00+08:00` while macOS local time was `2026-06-05 03:35:38 +0800`.
- Runtime scheduler evidence: auction-service log shows scheduler started, but `SELECT * FROM auctions WHERE status = 0 AND start_time <= NOW()` and `SELECT * FROM auctions WHERE status IN (1,2) AND end_time <= NOW()` repeatedly returned `rows:0`.
- Runtime DB evidence: MySQL `NOW()` was `2026-06-04 19:35:38`, `@@system_time_zone=UTC`, while macOS was `2026-06-05 03:35:38 +0800`; for `id=993304`, `start_time <= NOW()` and `end_time <= NOW()` both evaluated to `0`.

## Current Conclusion

Confirmed root cause: database session time is UTC while auction timestamps are stored/displayed as local `+08:00` wall-clock values. Scheduler compares local wall-clock timestamps against MySQL UTC `NOW()`, so records appear "not started/not ended" for roughly 8 hours in scheduler SQL. Frontend then trusts stale `status=0` and renders `即将开始`.

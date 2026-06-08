# 直播间「1 进行中 + 1 即将开始」竞拍约束 Spec

- **创建日期**：2026-06-08
- **作者**：Brainstorming session（用户 + Assistant）
- **状态**：待评审
- **执行分支建议**：`feat/live-stream-pending-plus-running`
- **核心方案**：最小语义改造：把「每直播间最多 1 个 active 竞拍」拆为「最多 1 个 Ongoing/Delayed + 最多 1 个 Pending」，不引入排队状态机。

---

## 1. 背景与问题

`auction-service` 当前对直播间维度的约束是：每个直播间同一时间最多只能有 1 个 active 竞拍，其中 active = `status IN (Pending, Ongoing, Delayed)`。

数据库通过生成列与唯一索引强制：

```sql
active_live_stream_key = CASE WHEN status IN (0,1,2) THEN live_stream_id ELSE NULL END
UNIQUE KEY uk_active_live_stream (active_live_stream_key)
```

这导致商家在一个直播间内，当前竞拍未结束时无法预排下一场 `Pending`。

## 2. 目标语义

允许同一直播间同时存在：

- 最多 1 个 `Ongoing/Delayed` 竞拍。
- 最多 1 个 `Pending` 竞拍。
- 同一商品仍最多 1 个 active 竞拍，`uk_active_product` 不变。

非目标：

- 不引入排队状态机。
- 不新增「排队中」状态。
- 不新增商家手动开始竞拍入口。
- 不允许前端绕过后端/数据库约束。

## 3. 核心设计

### 3.1 拆分 live stream 唯一约束

用两个生成列替代旧的 `active_live_stream_key`：

```sql
pending_live_stream_key = CASE WHEN status = 0 THEN live_stream_id ELSE NULL END
running_live_stream_key = CASE WHEN status IN (1,2) THEN live_stream_id ELSE NULL END
```

分别创建唯一索引：

```sql
UNIQUE KEY uk_pending_live_stream (pending_live_stream_key)
UNIQUE KEY uk_running_live_stream (running_live_stream_key)
```

### 3.2 创建竞拍

`CreateAuction` 只拒绝同直播间已有 `Pending` 的情况：

- 已有 `Pending`：返回 `ErrPendingLiveStreamAuctionExists`。
- 已有 `Ongoing/Delayed`：允许创建新的 `Pending`。
- 同商品已有 active：仍返回 `ErrActiveAuctionExists`。

### 3.3 开始竞拍

`StartAuction` 在 `Pending -> Ongoing` 前检查同直播间是否已有 `Ongoing/Delayed`：

- 有：跳过，保持 `Pending`。
- 无：开始竞拍。

### 3.4 延迟开始时重算时间

当 `Pending` 被前一场竞拍压住时，不能沿用创建时的绝对 `EndTime`，否则开始后可能立即过期。

开始时按原始时长重算：

```text
duration = EndTime - StartTime
StartTime = now
EndTime = now + duration
```

## 4. 边界问题

| 编号 | 问题 | 处理 |
| --- | --- | --- |
| B1 | Pending 延迟开始后旧 `EndTime` 已过期，可能秒结束 | `StartAuction` 重算 `StartTime/EndTime` |
| B2 | scheduler 到点强行开始 Pending，可能撞 running 唯一索引 | `StartAuction` 增加 live stream running 兜底并静默 skip |
| B3 | 现有测试断言「已有 Ongoing 时创建新竞拍失败」 | 改为断言允许创建 Pending，并新增双 Pending 拒绝测试 |

## 5. 验收标准

- 同直播间已有 `Ongoing` 时，可以创建另一商品的 `Pending`。
- 同直播间已有 `Pending` 时，创建第二个 `Pending` 失败。
- `Pending` 到点但同直播间有 `Ongoing/Delayed` 时保持 `Pending`。
- 前一场结束后，下一次调度能启动 `Pending`。
- 延迟启动后的 `EndTime` 为 `now + 原始时长`，不会秒结束。
- 同商品 active 唯一约束不回退。

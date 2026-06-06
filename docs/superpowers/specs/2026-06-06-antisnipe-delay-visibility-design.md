# 防狙击延时「实时可见」链路改造 Spec

- **创建日期**：2026-06-06
- **作者**：Brainstorming session（用户 + Assistant）
- **状态**：待执行（建议在独立子会话用 TDD 执行）
- **关联**：H5 Demo Console 设计（`2026-06-06-h5-demo-console-design.md`）的前置依赖之一
- **执行分支建议**：`feat/antisnipe-delay-visibility`

---

## 1. 背景与问题

防狙击（antisnipe）规则本身是工作的：出价命中延时窗口后，后端 `tryExtendAuction` 会把 `auctions.end_time` 延长并把状态切到 `Delayed`。但**这个延时在 H5 直播间画面上完全不可见**，是一个真实存在的线上 bug，而非仅演示问题。

### 1.1 根因（三处缺陷，已核查源码确认）

| # | 缺陷 | 证据 | 后果 |
|---|---|---|---|
| C1 | 延时落库后**不广播**任何 WS 消息 | [bid.go:385-405](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/service/bid.go#L385-L405) 事务后只有 `fmt.Printf` + metrics | 前端无从得知 end_time 变了 |
| C2 | 前端**不监听** `delay_triggered`（也不监听 `time_sync`） | [LiveRoomSlide.tsx:491-523](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L491-L523) 只注册了 `chat_message/rank_update/bid_placed/sync_response/auction_ended` | 即使后端广播也没人处理 |
| C3 | 调度器周期性 `time_sync` **只查 `status=1`(Ongoing)** | [scheduler.go:100](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/service/scheduler.go#L100) | 延时后状态变 `status=2`(Delayed)，该场次反而被踢出周期校时 |

> H5 倒计时是**前端基于初始 `end_time` 本地每秒自减**（[LiveRoomSlide.tsx:213-217](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L213-L217)、[:468-471](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L468-L471)），唯一更新 `end_time` 的途径是 WS 消息。因此「直接改库」对已打开页面无效，必须走 WS 广播。

### 1.2 目标

出价触发防狙击延时后，H5 直播间倒计时**立即回弹/延长**，让演示与真实用户都能直观看到「防狙击生效」。

### 1.3 非目标

- 不改前端本地倒计时自减机制。
- 不引入 Demo Console（本 Spec 是其前置依赖，独立交付）。
- 不改 `tryExtendAuction` 的延时判定/状态机逻辑，只在其成功后补广播。

---

## 2. 已确认的现状事实（执行者无需重新调查）

- **end_time / current_price 的 SoT 是 MySQL `auctions` 表**；Redis 仅用于出价分布式锁，`sync:state` 缓存生产从未写入。
- **延时落库路径**：[bid.go:354-405](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/service/bid.go#L354-L405) `tryExtendAuction`，事务内 `ExtendEndTime` + `UpdateStatus(Delayed)`。
- **`BidService` 已持有 `hub *websocket.Hub`**（[bid.go:30](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/service/bid.go#L30)），通过 `SetHub` 注入；`broadcastRanking` 已在用 `s.hub.BroadcastToRoom`（[bid.go:496-538](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/service/bid.go#L496-L538)）。
- **WS 消息已就绪**：`MessageTypeDelayTriggered`、`DelayTriggeredData{AuctionID, DelayDuration, NewEndTime(ms), RemainingDelay, MaxDelay}`、`NewDelayTriggeredMessage` 均已定义于 [message.go:94-101,238-241](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/websocket/message.go#L94-L101)，**当前无任何调用方**。
- **Hub 是具体类型** `*websocket.Hub`，`websocket.NewHub()` + `go hub.Run()` + 注册 `*websocket.Client{Send: chan}` 可在单测中断言广播，范式见 [fixed_price_broadcaster_test.go:16-52](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/service/fixed_price_broadcaster_test.go#L16-L52)。
- **前端归属校验**：`belongsToThisRoom(data)` 已存在（[LiveRoomSlide.tsx:484-489](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L484-L489)），新监听必须复用它防跨房污染。
- **前端 `auction.end_time`** 用 `new Date(auction.end_time)` 解析（对 string 与 number 均兼容）；`sync_response` 分支已直接写入 `end_time`（[:506-515](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L506-L515)）。

---

## 3. 改造方案

### 3.1 后端改造 A：延时成功后广播 `delay_triggered`（修 C1）

在 `BidService` 上新增一个**只依赖 hub 的可测小函数** `broadcastDelayTriggered`，并在 `tryExtendAuction` 事务成功后调用。

**新增函数**（`backend/auction/service/bid.go`，置于 `broadcastRanking` 附近）：

```go
// broadcastDelayTriggered 广播防狙击延时消息，使前端实时更新倒计时。
// 仅依赖 hub，便于单测；hub 为 nil 时安全跳过。
func (s *BidService) broadcastDelayTriggered(auctionID int64, delayDuration int, newEndTime time.Time, remainingDelay, maxDelay int) {
	if s.hub == nil {
		return
	}
	msg := websocket.NewDelayTriggeredMessage(&websocket.DelayTriggeredData{
		AuctionID:      auctionID,
		DelayDuration:  delayDuration,
		NewEndTime:     newEndTime.UnixMilli(),
		RemainingDelay: remainingDelay,
		MaxDelay:       maxDelay,
	})
	s.hub.BroadcastToRoom(auctionID, msg)
}
```

**在 `tryExtendAuction` 事务成功后调用**（[bid.go:397-405](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/service/bid.go#L397-L405) 区域，替换原 `fmt.Printf` 之后的尾部）：

```go
	if txErr != nil {
		return
	}

	// 重新读取最新 end_time / delay_used，广播 delay_triggered 让前端实时回弹倒计时
	updated, err := s.auctionDAO.GetByID(ctx, auctionID)
	if err == nil {
		remainingDelay := maxDelayTime - updated.DelayUsed
		if remainingDelay < 0 {
			remainingDelay = 0
		}
		s.broadcastDelayTriggered(auctionID, actualDelay, updated.EndTime, remainingDelay, maxDelayTime)
	}

	if s.metrics != nil {
		s.metrics.RecordDelayTriggered(auctionID)
	}
```

> 说明：重新 `GetByID` 而非内存计算 `EndTime`，确保拿到 DB 真值（`ExtendEndTime` 用的是 `DATE_ADD`，且 `delay_used` 已累加）。`err != nil` 时静默跳过广播（fail-soft），不影响已落库的延时。

### 3.2 后端改造 B：周期性 `time_sync` 覆盖 Delayed 状态（修 C3）

`broadcastTimeSync` 当前只取 `status=1`。延时后状态变 `status=2`，导致中途进房用户的周期校时丢失。改为同时广播 Ongoing 与 Delayed。

**修改** [scheduler.go:92-109](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/service/scheduler.go#L92-L109)：

```go
// broadcastTimeSync 广播时间同步消息（覆盖进行中 + 延时中的竞拍）
func (s *Scheduler) broadcastTimeSync() {
	if s.hub == nil {
		return
	}

	ctx := context.Background()

	// status=1 进行中 + status=2 延时中，二者倒计时都在跑，都需周期校时
	statuses := []int{1, 2}
	for _, st := range statuses {
		auctions, err := s.auctionService.GetAuctionsByStatus(ctx, st)
		if err != nil {
			log.Printf("Error getting auctions(status=%d) for time sync: %v", st, err)
			continue
		}
		for _, auction := range auctions {
			s.timeSyncService.BroadcastTimeSync(auction.ID, auction.EndTime.UnixMilli())
		}
	}
}
```

> `GetAuctionsByStatus` 签名不变（[auction.go:229-233](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/service/auction.go#L229-L233)），调两次合并，零 DAO 改动。

### 3.3 前端改造：监听 `delay_triggered` 更新倒计时（修 C2）

在 [LiveRoomSlide.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx) 的 WS 监听注册区（`auction_ended` 监听之后、`:523` 附近）新增：

```tsx
    ws.on('delay_triggered', (data) => {
      if (!belongsToThisRoom(data)) return;
      const newEnd = data?.new_end_time;
      if (newEnd) {
        // new_end_time 为后端 UnixMilli(ms)，统一转 ISO 字符串写回，保持 end_time 字段类型一致
        const iso = new Date(Number(newEnd)).toISOString();
        setAuction((previous) => previous ? { ...previous, end_time: iso, status: 2 } : previous);
      }
      // 轻量提示：让评委直观感知"防狙击生效"
      showGlobalToast({ type: 'info', title: '触发防狙击', message: '已有新出价，竞拍时间自动延长' });
    });
```

> - `belongsToThisRoom` 复用现有跨房校验。
> - `status: 2` 与后端 `Delayed` 对齐；前端 `auction_ended` 已用裸数字状态（`status: 3`），此处一致。
> - `showGlobalToast` 已在该组件作用域内可用（[:538](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L538)）。若其 `type` 枚举不含 `info`，执行时改用现有合法值（如 `success`）。

---

## 4. TDD 测试大纲

### 4.1 后端单测：`broadcastDelayTriggered`（新增）

**文件**：`backend/auction/service/delay_broadcast_test.go`

参考 [fixed_price_broadcaster_test.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/service/fixed_price_broadcaster_test.go) 的 hub+client 范式：

| # | 用例 | 期望 |
|---|---|---|
| U1 | hub 注册 client 后调用 `broadcastDelayTriggered(auctionID, 30, endTime, 60, 90)` | client.Send 收到 1 条 `type=delay_triggered` 消息，`data.new_end_time == endTime.UnixMilli()`、`delay_duration==30`、`remaining_delay==60`、`max_delay==90` |
| U2 | `s.hub == nil` 时调用 | 不 panic，无副作用 |
| U3 | 跨房：client 在 auction 1001，向 1002 广播 | client 在 50ms 内**收不到**消息 |

测试构造 `BidService` 仅需设置 `hub` 字段：`svc := &BidService{}; svc.SetHub(hub)`。

**运行**：`cd backend/auction && go test ./service/ -run TestDelayBroadcast -v` → 全 PASS。

### 4.2 后端：`time_sync` 覆盖 Delayed（按现有测试条件）

若 `scheduler` 已有测试则补一条「status=2 的竞拍也被 BroadcastTimeSync」；若无 scheduler 测试且依赖真实 DAO 难以隔离，则在本 Spec 执行中**以编译 + 现有回归为准**，不强制新增 scheduler 单测（避免为此引入 DAO mock 重构）。执行者须在 PR 描述中说明该取舍。

### 4.3 前端单测：`delay_triggered` 更新 end_time

**文件**：`frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.delay.test.tsx`（或并入既有 LiveRoomSlide 测试）

| # | 用例 | 期望 |
|---|---|---|
| F1 | mock WS 推送 `{type:'delay_triggered', data:{auction_id: <当前>, new_end_time: <未来ms>}}` | `auction.end_time` 被更新为对应 ISO，倒计时显示随之变长 |
| F2 | 推送 `auction_id` 为**其他房间**的 `delay_triggered` | `end_time` 不变（被 `belongsToThisRoom` 拦截） |

> 若现有前端测试对 `LiveRoomSlide` 的 WS mock 基建不足，至少覆盖 F1；F2 可用 `belongsToThisRoom` 的纯函数级断言替代。

### 4.4 端到端手动验证（演示场景）

1. 本地起全栈，进入某进行中竞拍的 H5 直播间。
2. 用后台接口/脚本把该竞拍 `end_time` 调到剩约 10s（注意：此步仅为制造"临近结束"条件，前端倒计时不会因改库变化，需刷新页面或重连使其读到新初始值；演示时可直接选一个本就临近结束的场次）。
3. 在倒计时 < `trigger_delay_before` 窗口内发起一次合法出价。
4. **预期**：H5 倒计时**立即回弹**（如从 8s 跳回 38s），并弹出「触发防狙击」提示。

> E2E 不纳入自动化门槛，作为执行者交付前的人工冒烟。

---

## 5. 执行顺序与提交粒度

1. **Task 1（后端 C1）**：写 `broadcastDelayTriggered` 失败单测 → 实现函数 → `tryExtendAuction` 接入 → 单测 PASS → commit。
2. **Task 2（后端 C3）**：改 `broadcastTimeSync` 覆盖 status∈{1,2} → `go build` + 现有回归 PASS → commit。
3. **Task 3（前端 C2）**：写 `delay_triggered` 监听失败单测 → 实现监听 → 前端测试 PASS + `npm run build` → commit。
4. **Task 4**：人工 E2E 冒烟（§4.4），更新 PR 描述。

每个 Task 独立提交，提交信息示例：
- `fix(auction): 防狙击延时后广播 delay_triggered 修复 H5 倒计时不回弹`
- `fix(auction): time_sync 覆盖 Delayed 状态竞拍`
- `fix(h5): 直播间监听 delay_triggered 实时延长倒计时`

---

## 6. 风险与权衡

| 风险 | 应对 |
|---|---|
| `broadcastDelayTriggered` 内 `GetByID` 失败导致不广播 | fail-soft：延时已落库，仅丢一次实时提示；下个 `time_sync`（改造 B 后覆盖 Delayed）兜底校时 |
| 节流：延时广播未走 `rankThrottle` | 延时是低频事件（每场次受 `MaxDelayTime` 限制），无需节流；不复用 `broadcastRanking` 的 200ms 节流 |
| 前端 `showGlobalToast` 的 `type` 枚举不含 `info` | 执行时改用现有合法枚举值，不新增 toast 类型 |
| `end_time` 字段前后端类型混用（string vs number ms） | 前端统一 `new Date(Number(ms)).toISOString()` 写回，与 HTTP 初始值（ISO string）保持一致 |

---

## 7. 验收标准（Definition of Done）

- [ ] `broadcastDelayTriggered` 单测 U1/U2/U3 通过。
- [ ] `tryExtendAuction` 延时成功后广播一条 `delay_triggered`（含正确 `new_end_time`）。
- [ ] `broadcastTimeSync` 覆盖 Ongoing + Delayed。
- [ ] 前端监听 `delay_triggered` 并更新 `end_time`，F1 通过。
- [ ] `cd backend/auction && go test ./...` 与 `cd frontend/h5 && npm run build` 均通过。
- [ ] 人工 E2E：H5 倒计时在出价触发延时后实时回弹。

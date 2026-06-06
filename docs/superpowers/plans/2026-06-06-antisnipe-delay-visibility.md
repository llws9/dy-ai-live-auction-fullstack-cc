# Antisnipe Delay Visibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 出价触发防狙击延时后，H5 直播间倒计时立即回弹，并通过 `time_sync` 对 Delayed 状态竞拍提供周期校时兜底。

**Architecture:** 后端在 `tryExtendAuction` 延时落库成功后广播 `delay_triggered`，并让调度器同时为 Ongoing 与 Delayed 竞拍广播 `time_sync`。H5 在直播间 WebSocket 链路中监听 `delay_triggered` 和 `time_sync`，统一把后端毫秒时间戳归一化为 ISO `end_time`，驱动现有倒计时本地自减逻辑。

**Tech Stack:** Go 1.24+、Hertz/auction-service、auction WebSocket Hub、React 18、TypeScript、Jest、Testing Library。

---

## Source Spec

- `docs/superpowers/specs/2026-06-06-antisnipe-delay-visibility-design.md`

## File Structure

- Modify: `backend/auction/service/bid.go`
  - 新增 `broadcastDelayTriggered`，只依赖 `*websocket.Hub`。
  - 在 `tryExtendAuction` 事务成功后重新读取 DB 真值并广播 `delay_triggered`。
- Create: `backend/auction/service/delay_broadcast_test.go`
  - 覆盖正常广播、`hub == nil` 不 panic、跨房不污染。
- Modify: `backend/auction/service/scheduler.go`
  - `broadcastTimeSync` 同时查询 `model.AuctionStatusOngoing` 与 `model.AuctionStatusDelayed`。
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`
  - 新增 `toEndTimeIso` 归一化函数。
  - `sync_response`、`delay_triggered`、`time_sync` 统一写回 ISO `end_time`。
  - `delay_triggered` 触发全局 Toast，`time_sync` 仅做静默校时。
- Modify: `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx`
  - 复用现有 mock WebSocket 基建，验证 `delay_triggered` 更新时间并拦截跨房消息，验证 `time_sync` 兜底更新时间。

## Execution Notes

- 本计划修复真实链路问题，不实现 H5 Demo Console。
- `delay_triggered` 是即时用户可见事件；`time_sync` 是兜底校时事件，不弹 Toast。
- 计划中新增 `time_sync` 前端监听是对 spec 的闭环修正：只改后端 C3 无法让当前 H5 页面消费周期校时。
- 金额逻辑不变，本任务不引入任何 float 金额计算。

---

### Task 1: 后端广播 `delay_triggered`

**Files:**
- Create: `backend/auction/service/delay_broadcast_test.go`
- Modify: `backend/auction/service/bid.go`

- [ ] **Step 1: Write the failing broadcast tests**

Create `backend/auction/service/delay_broadcast_test.go`:

```go
package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"auction-service/websocket"
)

func newDelayBroadcastTestClient(t *testing.T, auctionID int64) (*websocket.Hub, *websocket.Client) {
	t.Helper()

	hub := websocket.NewHub()
	go hub.Run()

	client := &websocket.Client{
		ID:        "delay-broadcast-test-client",
		AuctionID: auctionID,
		UserID:    42,
		Send:      make(chan *websocket.Message, 16),
	}
	hub.Register <- client
	time.Sleep(20 * time.Millisecond)

	return hub, client
}

func recvDelayBroadcastMsg(t *testing.T, ch <-chan *websocket.Message) *websocket.Message {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected websocket message")
		return nil
	}
}

func assertNoDelayBroadcastMsg(t *testing.T, ch <-chan *websocket.Message) {
	t.Helper()
	select {
	case msg := <-ch:
		t.Fatalf("expected no message, got %s", msg.Type)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestDelayBroadcast_BroadcastsDelayTriggeredToAuctionRoom(t *testing.T) {
	hub, client := newDelayBroadcastTestClient(t, 1001)

	svc := &BidService{}
	svc.SetHub(hub)
	endTime := time.UnixMilli(1780761600000)

	svc.broadcastDelayTriggered(1001, 30, endTime, 60, 90)

	msg := recvDelayBroadcastMsg(t, client.Send)
	require.Equal(t, websocket.MessageTypeDelayTriggered, msg.Type)
	data, ok := msg.Data.(*websocket.DelayTriggeredData)
	require.True(t, ok)
	assert.Equal(t, int64(1001), data.AuctionID)
	assert.Equal(t, 30, data.DelayDuration)
	assert.Equal(t, endTime.UnixMilli(), data.NewEndTime)
	assert.Equal(t, 60, data.RemainingDelay)
	assert.Equal(t, 90, data.MaxDelay)
}

func TestDelayBroadcast_NoHubDoesNotPanic(t *testing.T) {
	svc := &BidService{}

	require.NotPanics(t, func() {
		svc.broadcastDelayTriggered(1001, 30, time.UnixMilli(1780761600000), 60, 90)
	})
}

func TestDelayBroadcast_DoesNotLeakAcrossAuctionRooms(t *testing.T) {
	hub, client := newDelayBroadcastTestClient(t, 1001)

	svc := &BidService{}
	svc.SetHub(hub)

	svc.broadcastDelayTriggered(1002, 30, time.UnixMilli(1780761600000), 60, 90)

	assertNoDelayBroadcastMsg(t, client.Send)
}
```

- [ ] **Step 2: Run the failing test**

Run:

```bash
cd backend/auction && go test ./service/ -run TestDelayBroadcast -v
```

Expected: FAIL because `svc.broadcastDelayTriggered` is not defined.

- [ ] **Step 3: Add `broadcastDelayTriggered` to `BidService`**

Modify `backend/auction/service/bid.go` near `broadcastRanking`:

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

- [ ] **Step 4: Run the broadcast tests**

Run:

```bash
cd backend/auction && go test ./service/ -run TestDelayBroadcast -v
```

Expected: PASS.

- [ ] **Step 5: Wire the broadcast into `tryExtendAuction`**

In `backend/auction/service/bid.go`, replace the tail of `tryExtendAuction` after `txErr` handling:

```go
	if txErr != nil {
		return
	}

	fmt.Printf("Auction %d delayed by %d seconds\n", auctionID, actualDelay)

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

- [ ] **Step 6: Run backend service tests**

Run:

```bash
cd backend/auction && go test ./service/ -run 'TestDelayBroadcast|TestFixedPriceWSBroadcaster' -v
```

Expected: PASS.

- [ ] **Step 7: Commit Task 1**

```bash
git add backend/auction/service/delay_broadcast_test.go backend/auction/service/bid.go
git commit -m "fix(auction): broadcast antisnipe delay updates"
```

---

### Task 2: 让 `time_sync` 覆盖 Delayed 状态

**Files:**
- Modify: `backend/auction/service/scheduler.go`

- [ ] **Step 1: Update scheduler imports**

Modify `backend/auction/service/scheduler.go` imports:

```go
import (
	"context"
	"log"
	"time"

	"auction-service/model"
	"auction-service/websocket"
)
```

- [ ] **Step 2: Update `broadcastTimeSync`**

Replace `broadcastTimeSync` in `backend/auction/service/scheduler.go`:

```go
// broadcastTimeSync 广播时间同步消息（覆盖进行中 + 延时中的竞拍）
func (s *Scheduler) broadcastTimeSync() {
	if s.hub == nil {
		return
	}

	ctx := context.Background()

	statuses := []model.AuctionStatus{
		model.AuctionStatusOngoing,
		model.AuctionStatusDelayed,
	}
	for _, status := range statuses {
		auctions, err := s.auctionService.GetAuctionsByStatus(ctx, int(status))
		if err != nil {
			log.Printf("Error getting auctions(status=%d) for time sync: %v", status, err)
			continue
		}
		for _, auction := range auctions {
			s.timeSyncService.BroadcastTimeSync(auction.ID, auction.EndTime.UnixMilli())
		}
	}
}
```

- [ ] **Step 3: Run backend compile-level verification**

Run:

```bash
cd backend/auction && go test ./service/ -run TestDelayBroadcast -v
```

Expected: PASS and `scheduler.go` compiles with the new import.

- [ ] **Step 4: Run full auction backend tests**

Run:

```bash
cd backend/auction && go test ./...
```

Expected: PASS.

- [ ] **Step 5: Commit Task 2**

```bash
git add backend/auction/service/scheduler.go
git commit -m "fix(auction): include delayed auctions in time sync"
```

---

### Task 3: H5 监听延时和校时消息

**Files:**
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`
- Modify: `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx`

- [ ] **Step 1: Add failing WebSocket tests**

In `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx`, add this helper after `renderSlide`:

```tsx
const getWebSocketHandler = (type: string) => mockWebSocketInstance.on.mock.calls.find((call) => call[0] === type)?.[1] as
  | ((data: any) => void)
  | undefined;
```

Then add these tests inside `describe('LiveRoomSlide', () => { ... })`, after the existing cross-room WebSocket test:

```tsx
  it('updates end_time and shows toast when delay_triggered belongs to this room', async () => {
    const baseNow = new Date('2026-06-06T00:00:00.000Z').getTime();
    const dateNowSpy = jest.spyOn(Date, 'now').mockReturnValue(baseNow);
    mockedAuctionApi.get.mockResolvedValue({
      id: 5,
      product_id: 7,
      live_stream_id: 3,
      status: 1,
      current_price: 1200,
      end_time: new Date(baseNow + 30_000).toISOString(),
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    expect(await screen.findByText('00:30')).toBeInTheDocument();

    const delayHandler = getWebSocketHandler('delay_triggered');
    expect(delayHandler).toBeDefined();

    act(() => {
      delayHandler!({
        auction_id: 5,
        delay_duration: 30,
        new_end_time: baseNow + 180_000,
        remaining_delay: 60,
        max_delay: 90,
      });
    });

    expect(await screen.findByText('03:00')).toBeInTheDocument();
    expect(mockShowGlobalToast).toHaveBeenCalledWith(expect.objectContaining({
      type: 'info',
      title: '触发防狙击',
      message: '已有新出价，竞拍时间自动延长',
    }));

    dateNowSpy.mockRestore();
  });

  it('ignores delay_triggered messages from other auction rooms', async () => {
    const baseNow = new Date('2026-06-06T00:00:00.000Z').getTime();
    const dateNowSpy = jest.spyOn(Date, 'now').mockReturnValue(baseNow);
    mockedAuctionApi.get.mockResolvedValue({
      id: 5,
      product_id: 7,
      live_stream_id: 3,
      status: 1,
      current_price: 1200,
      end_time: new Date(baseNow + 30_000).toISOString(),
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    expect(await screen.findByText('00:30')).toBeInTheDocument();

    const delayHandler = getWebSocketHandler('delay_triggered');
    expect(delayHandler).toBeDefined();

    act(() => {
      delayHandler!({
        auction_id: 999,
        delay_duration: 30,
        new_end_time: baseNow + 180_000,
        remaining_delay: 60,
        max_delay: 90,
      });
    });

    expect(screen.getByText('00:30')).toBeInTheDocument();
    expect(screen.queryByText('03:00')).not.toBeInTheDocument();
    expect(mockShowGlobalToast).not.toHaveBeenCalledWith(expect.objectContaining({
      title: '触发防狙击',
    }));

    dateNowSpy.mockRestore();
  });

  it('updates end_time from time_sync without showing antisnipe toast', async () => {
    const baseNow = new Date('2026-06-06T00:00:00.000Z').getTime();
    const dateNowSpy = jest.spyOn(Date, 'now').mockReturnValue(baseNow);
    mockedAuctionApi.get.mockResolvedValue({
      id: 5,
      product_id: 7,
      live_stream_id: 3,
      status: 2,
      current_price: 1200,
      end_time: new Date(baseNow + 30_000).toISOString(),
    });

    renderSlide({ liveStreamId: 3, currentAuctionId: 5 });

    expect((await screen.findAllByText('明代紫砂壶')).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole('button', { name: '出价' }));
    expect(await screen.findByText('00:30')).toBeInTheDocument();

    const timeSyncHandler = getWebSocketHandler('time_sync');
    expect(timeSyncHandler).toBeDefined();

    act(() => {
      timeSyncHandler!({
        server_time: baseNow,
        end_time: baseNow + 120_000,
      });
    });

    expect(await screen.findByText('02:00')).toBeInTheDocument();
    expect(mockShowGlobalToast).not.toHaveBeenCalledWith(expect.objectContaining({
      title: '触发防狙击',
    }));

    dateNowSpy.mockRestore();
  });
```

- [ ] **Step 2: Run the failing frontend test**

Run:

```bash
cd frontend/h5 && npm test -- LiveRoomSlide.test.tsx --runInBand
```

Expected: FAIL because `delay_triggered` and `time_sync` handlers are not registered.

- [ ] **Step 3: Add end-time normalization helper**

In `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`, add this helper near `toAmount`:

```tsx
const toEndTimeIso = (value: unknown): string | undefined => {
  if (value == null) return undefined;

  const parsed = typeof value === 'number'
    ? value
    : /^\d+$/.test(String(value).trim())
      ? Number(value)
      : new Date(String(value)).getTime();

  if (!Number.isFinite(parsed)) return undefined;
  return new Date(parsed).toISOString();
};
```

- [ ] **Step 4: Normalize `sync_response` end_time**

Replace the current `sync_response` handler block in `LiveRoomSlide.tsx`:

```tsx
    ws.on('sync_response', (data) => {
      if (!belongsToThisRoom(data)) return;
      if (data?.current_price !== undefined || data?.status !== undefined || data?.end_time !== undefined) {
        const nextEndTime = toEndTimeIso(data?.end_time);
        setAuction((previous) => previous ? {
          ...previous,
          current_price: data.current_price !== undefined ? toAmount(data.current_price, toAmount(previous.current_price)) : previous.current_price,
          status: data.status ?? previous.status,
          end_time: nextEndTime ?? previous.end_time,
        } : previous);
      }
      if (data?.ranking) {
        setRanking(normalizeRanking(extractList(data)));
      }
    });
```

- [ ] **Step 5: Register `delay_triggered` and `time_sync` handlers**

In the WebSocket `useEffect` in `LiveRoomSlide.tsx`, add these named handlers before `ws.on('chat_message', onChatMessage);`:

```tsx
    const onDelayTriggered = (data: any) => {
      if (!belongsToThisRoom(data)) return;
      const nextEndTime = toEndTimeIso(data?.new_end_time);
      if (!nextEndTime) return;

      setAuction((previous) => previous ? { ...previous, end_time: nextEndTime, status: 2 } : previous);
      showGlobalToast({
        type: 'info',
        title: '触发防狙击',
        message: '已有新出价，竞拍时间自动延长',
      });
    };

    const onTimeSync = (data: any) => {
      if (!belongsToThisRoom(data)) return;
      const nextEndTime = toEndTimeIso(data?.end_time);
      if (!nextEndTime) return;

      setAuction((previous) => previous ? { ...previous, end_time: nextEndTime } : previous);
    };
```

Then register them:

```tsx
    ws.on('chat_message', onChatMessage);
    ws.on('delay_triggered', onDelayTriggered);
    ws.on('time_sync', onTimeSync);
```

Update cleanup in the same `useEffect`:

```tsx
      ws.off('chat_message', onChatMessage);
      ws.off('delay_triggered', onDelayTriggered);
      ws.off('time_sync', onTimeSync);
```

- [ ] **Step 6: Run the frontend test**

Run:

```bash
cd frontend/h5 && npm test -- LiveRoomSlide.test.tsx --runInBand
```

Expected: PASS.

- [ ] **Step 7: Build H5**

Run:

```bash
cd frontend/h5 && npm run build
```

Expected: PASS.

- [ ] **Step 8: Commit Task 3**

```bash
git add frontend/h5/src/pages/Live/LiveRoomSlide.tsx frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx
git commit -m "fix(h5): show antisnipe delay updates in live room"
```

---

### Task 4: 全链路验证与交付记录

**Files:**
- No code changes required.

- [ ] **Step 1: Run backend full tests**

Run:

```bash
cd backend/auction && go test ./...
```

Expected: PASS.

- [ ] **Step 2: Run frontend focused test and build**

Run:

```bash
cd frontend/h5 && npm test -- LiveRoomSlide.test.tsx --runInBand
cd frontend/h5 && npm run build
```

Expected: both PASS.

- [ ] **Step 3: Manual E2E smoke**

Run local services with the project deploy flow, then verify:

```bash
# In the repo root, use the existing local deploy command or /dp-dev flow.
# Open an active H5 live room whose auction can trigger antisnipe delay.
```

Manual expected result:

1. H5 live room is connected and displays `实时同步中`.
2. Auction is inside `trigger_delay_before` seconds.
3. A legal bid triggers backend antisnipe delay.
4. H5 countdown jumps upward immediately, for example `00:08` to `00:38`.
5. Toast appears with title `触发防狙击`.

- [ ] **Step 4: Final status check**

Run:

```bash
git status --short
```

Expected: only intentional committed changes remain clean, or only unrelated pre-existing untracked spec files remain.

---

## Self-Review

- Spec C1 coverage: Task 1 adds `broadcastDelayTriggered` and calls it after successful `tryExtendAuction` transaction.
- Spec C2 coverage: Task 3 registers `delay_triggered` and updates H5 `end_time` with cross-room protection.
- Spec C3 coverage: Task 2 broadcasts `time_sync` for Ongoing and Delayed; Task 3 consumes `time_sync` as silent fallback.
- Cross-room safety: Task 3 reuses `belongsToThisRoom` for both event handlers.
- Type consistency: backend sends UnixMilli `new_end_time`; frontend converts numeric and string timestamps to ISO `end_time`.
- Verification coverage: backend broadcast unit tests, frontend WS handler tests, backend full tests, H5 build.
- Manual smoke: not executed; residual browser E2E risk is tracked in Task 4 state.

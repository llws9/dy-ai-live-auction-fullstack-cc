# Touchpoint Metrics Tracking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add user-touchpoint exposure and interaction tracking through the existing H5 `/api/v1/track` -> gateway Prometheus -> Grafana metrics pipeline.

**Architecture:** The frontend adds a single `trackEvent()` utility that posts low-cardinality touchpoint events to the existing gateway `POST /api/v1/track` endpoint. Gateway extends its Prometheus metrics collector with `touchpoint_event_total{event,source,entry,type,result}` and records only normalized, low-cardinality labels. Touchpoint UI code calls the utility at exposure and interaction boundaries without blocking user flows.

**Tech Stack:** React 18, TypeScript, Jest, CloudWeGo Hertz, Go, Prometheus client_golang, Grafana/PromQL.

---

## File Structure

- Create `frontend/h5/src/utils/trackEvent.ts`: Frontend tracking utility with `sendBeacon` first, `fetch` fallback, event typing, count bucketing, and dev-only logging.
- Create `frontend/h5/src/utils/__tests__/trackEvent.test.ts`: Unit tests for payload shape, beacon priority, fetch fallback, and non-throwing failures.
- Modify `frontend/h5/src/hooks/useTouchpointNotifications.ts`: Emit `summary_exposed` after a current-identity summary response is applied.
- Modify `frontend/h5/src/components/MobileShell/BottomNav.tsx`: Emit `entry_clicked` when the profile tab touchpoint entry is clicked.
- Modify `frontend/h5/src/pages/User/Index.tsx`: Emit `entry_clicked` for “我的竞拍” and “消息通知” touchpoint entries.
- Modify `frontend/h5/src/pages/Home/index.tsx`: Emit `entry_clicked` for the notification bell.
- Modify `frontend/h5/src/pages/Notifications/index.tsx`: Emit notification list exposure, item click, and mark-all read events.
- Modify `frontend/h5/src/hooks/useNotification.ts`: Emit hot-pull trigger events for success, failure, and debounce skip.
- Modify `frontend/h5/src/components/MobileShell/MobileContainer.tsx`: Emit `live_reminder_exposed` when the modal is opened from backend data.
- Modify `frontend/h5/src/components/LiveReminderModal/index.tsx`: Emit `live_reminder_clicked` and `live_reminder_dismissed`.
- Modify `frontend/h5/src/__tests__/components/MobileShell.test.tsx`: Mock and assert summary/reminder tracking calls.
- Modify `frontend/h5/src/pages/Notifications/__tests__/Notifications.test.tsx`: Mock and assert notification center tracking calls.
- Modify `frontend/h5/src/pages/User/__tests__/Profile.test.tsx`: Mock and assert profile entry click tracking.
- Modify `frontend/h5/src/pages/Home/__tests__/Home.test.tsx`: Mock and assert notification bell click tracking.
- Create `frontend/h5/src/hooks/__tests__/useNotification.test.ts`: Assert hot-pull success, failure, and debounce tracking.
- Modify `backend/gateway/pkg/metrics/metrics.go`: Add `TouchpointEvent` CounterVec, register it, and add `RecordTouchpointEvent`.
- Modify `backend/gateway/pkg/metrics/handler.go`: Route `event_type=touchpoint_event` into `RecordTouchpointEvent` with label normalization.
- Create `backend/gateway/pkg/metrics/handler_test.go`: Verify touchpoint metrics recording, unknown fallback, and no high-cardinality labels.

---

## Task 1: Gateway Touchpoint Metric

**Files:**
- Modify: `backend/gateway/pkg/metrics/metrics.go`
- Modify: `backend/gateway/pkg/metrics/handler.go`
- Create: `backend/gateway/pkg/metrics/handler_test.go`

- [ ] **Step 1: Write failing gateway metrics tests**

Create `backend/gateway/pkg/metrics/handler_test.go` with focused tests using a fresh Prometheus registry to avoid global registry pollution:

```go
package metrics

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMetrics(t *testing.T) *Metrics {
	t.Helper()
	reg := prometheus.NewRegistry()
	m := NewMetrics("gateway", reg)
	require.NotNil(t, m)
	return m
}

func TestTrackEventRecordsTouchpointMetric(t *testing.T) {
	m := newTestMetrics(t)
	c := app.NewContext(0)
	c.Request.Header.SetMethod("POST")
	c.Request.SetRequestURI("/api/v1/track")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody([]byte(`{
		"event_type":"touchpoint_event",
		"event_name":"summary_exposed",
		"user_id":"999",
		"params":{
			"source":"bottom_nav",
			"entry":"profile_tab",
			"type":"all",
			"result":"success",
			"notification_id":"123456"
		},
		"timestamp":1780300800000
	}`))

	TrackEvent(m)(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	assert.Equal(t, 1.0, testutil.ToFloat64(m.TouchpointEvent.WithLabelValues(
		"summary_exposed",
		"bottom_nav",
		"profile_tab",
		"all",
		"success",
	)))
}

func TestTrackEventNormalizesUnknownTouchpointLabels(t *testing.T) {
	m := newTestMetrics(t)
	c := app.NewContext(0)
	c.Request.Header.SetMethod("POST")
	c.Request.SetRequestURI("/api/v1/track")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody([]byte(`{
		"event_type":"touchpoint_event",
		"event_name":"not-in-allowlist",
		"params":{
			"source":"user-123456789",
			"entry":"dynamic-entry-123456789",
			"type":"unknown-dynamic-type",
			"result":"unexpected-result"
		}
	}`))

	TrackEvent(m)(context.Background(), c)

	assert.Equal(t, 200, c.Response.StatusCode())
	assert.Equal(t, 1.0, testutil.ToFloat64(m.TouchpointEvent.WithLabelValues(
		"unknown",
		"unknown",
		"unknown",
		"unknown",
		"unknown",
	)))
}

func TestTrackEventRejectsInvalidJSON(t *testing.T) {
	m := newTestMetrics(t)
	c := app.NewContext(0)
	c.Request.Header.SetMethod("POST")
	c.Request.SetRequestURI("/api/v1/track")
	c.Request.Header.SetContentTypeBytes([]byte("application/json"))
	c.Request.SetBody([]byte(`{"event_type":`))

	TrackEvent(m)(context.Background(), c)

	assert.Equal(t, 400, c.Response.StatusCode())
}

func TestTouchpointMetricDoesNotExposeHighCardinalityLabels(t *testing.T) {
	m := newTestMetrics(t)
	m.RecordTouchpointEvent("summary_exposed", "bottom_nav", "profile_tab", "all", "success")

	output, err := testutil.CollectAndLint(m.TouchpointEvent)
	require.NoError(t, err)
	assert.Empty(t, output)

	require.NoError(t, testutil.CollectAndCompare(m.TouchpointEvent, strings.NewReader(`
# HELP touchpoint_event_total 用户触达曝光和交互事件总数
# TYPE touchpoint_event_total counter
touchpoint_event_total{entry="profile_tab",event="summary_exposed",result="success",source="bottom_nav",type="all"} 1
`)))
}
```

- [ ] **Step 2: Run gateway test and verify failure**

Run:

```bash
cd backend/gateway
go test ./pkg/metrics -run 'TestTrackEvent|TestTouchpointMetric' -count=1
```

Expected: FAIL because `NewMetrics`, `TouchpointEvent`, and `RecordTouchpointEvent` are not defined yet.

- [ ] **Step 3: Add metrics constructor and touchpoint counter**

Refactor `backend/gateway/pkg/metrics/metrics.go` so `Init` delegates to a registry-aware constructor and registers `TouchpointEvent`. Apply these exact structural edits:

- Add `TouchpointEvent *prometheus.CounterVec` to the `Metrics` struct immediately after `ExperimentCompleted`.
- Add the `registerer` interface below `var defaultMetrics *Metrics`.
- Rename the current `func Init(serviceName string) *Metrics` body to `func NewMetrics(serviceName string, reg registerer) *Metrics`.
- Inside `NewMetrics`, leave every current metric initializer exactly as it is and add the `TouchpointEvent` initializer shown here to the `m := &Metrics{...}` literal.
- Replace the existing `prometheus.MustRegister(...)` call with `reg.MustRegister(...)`.
- Add `m.TouchpointEvent` to the registration list immediately after `m.ExperimentCompleted`.
- Add the new `Init` wrapper and `RecordTouchpointEvent` method shown here.

```go
TouchpointEvent *prometheus.CounterVec

type registerer interface {
	MustRegister(...prometheus.Collector)
}

TouchpointEvent: prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "touchpoint_event_total",
		Help: "用户触达曝光和交互事件总数",
	},
	[]string{"event", "source", "entry", "type", "result"},
),

func Init(serviceName string) *Metrics {
	m := NewMetrics(serviceName, prometheus.DefaultRegisterer)
	defaultMetrics = m
	return m
}

func (m *Metrics) RecordTouchpointEvent(event, source, entry, touchpointType, result string) {
	m.TouchpointEvent.WithLabelValues(event, source, entry, touchpointType, result).Inc()
}
```

- [ ] **Step 4: Add touchpoint event normalization**

Update `backend/gateway/pkg/metrics/handler.go` with allowlists and a new switch case:

```go
var allowedTouchpointEvents = map[string]struct{}{
	"summary_exposed":            {},
	"entry_clicked":              {},
	"notification_list_exposed":  {},
	"notification_item_clicked":  {},
	"mark_read":                  {},
	"hot_pull_triggered":         {},
	"live_reminder_exposed":      {},
	"live_reminder_clicked":      {},
	"live_reminder_dismissed":    {},
}

var allowedTouchpointSources = map[string]struct{}{
	"home":                {},
	"bottom_nav":          {},
	"profile":             {},
	"notification_center": {},
	"mobile_shell":        {},
	"notification_hook":   {},
}

var allowedTouchpointEntries = map[string]struct{}{
	"notification_bell":    {},
	"profile_tab":          {},
	"auction_history":      {},
	"notification_center":  {},
	"notification_item":    {},
	"mark_all_read":        {},
	"hot_pull":             {},
	"live_reminder_modal":  {},
}

var allowedTouchpointTypes = map[string]struct{}{
	"all":             {},
	"pending_payment": {},
	"outbid":          {},
	"ending_soon":     {},
	"live_start":      {},
	"notification":    {},
}

var allowedTouchpointResults = map[string]struct{}{
	"success":   {},
	"failed":    {},
	"clicked":   {},
	"dismissed": {},
	"debounced": {},
}

func normalizeLabel(value string, allowed map[string]struct{}) string {
	if _, ok := allowed[value]; ok {
		return value
	}
	return "unknown"
}

func recordTouchpointEvent(m *Metrics, req TrackEventRequest) {
	event := normalizeLabel(req.EventName, allowedTouchpointEvents)
	source := normalizeLabel(getStringParam(req.Params, "source", "unknown"), allowedTouchpointSources)
	entry := normalizeLabel(getStringParam(req.Params, "entry", "unknown"), allowedTouchpointEntries)
	touchpointType := normalizeLabel(getStringParam(req.Params, "type", "unknown"), allowedTouchpointTypes)
	result := normalizeLabel(getStringParam(req.Params, "result", "unknown"), allowedTouchpointResults)
	m.RecordTouchpointEvent(event, source, entry, touchpointType, result)
}
```

Add the switch case:

```go
case "touchpoint_event":
	recordTouchpointEvent(m, req)
```

- [ ] **Step 5: Run gateway tests and format**

Run:

```bash
cd backend/gateway
gofmt -w pkg/metrics/metrics.go pkg/metrics/handler.go pkg/metrics/handler_test.go
go test ./pkg/metrics -run 'TestTrackEvent|TestTouchpointMetric' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit gateway metrics**

```bash
git add backend/gateway/pkg/metrics/metrics.go backend/gateway/pkg/metrics/handler.go backend/gateway/pkg/metrics/handler_test.go
git commit -m "feat(gateway): record touchpoint metrics"
```

---

## Task 2: Frontend Tracking Utility

**Files:**
- Create: `frontend/h5/src/utils/trackEvent.ts`
- Create: `frontend/h5/src/utils/__tests__/trackEvent.test.ts`

- [ ] **Step 1: Write failing utility tests**

Create `frontend/h5/src/utils/__tests__/trackEvent.test.ts`:

```ts
import { getCountBucket, trackEvent } from '../trackEvent';

const originalFetch = global.fetch;

describe('trackEvent', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(Date, 'now').mockReturnValue(1780300800000);
    global.fetch = jest.fn().mockResolvedValue({ ok: true }) as jest.Mock;
    Object.defineProperty(navigator, 'sendBeacon', {
      value: jest.fn(() => true),
      configurable: true,
    });
  });

  afterEach(() => {
    jest.restoreAllMocks();
    global.fetch = originalFetch;
  });

  it('sends touchpoint payload through sendBeacon first', () => {
    trackEvent('summary_exposed', {
      source: 'bottom_nav',
      entry: 'profile_tab',
      type: 'all',
      result: 'success',
      countBucket: '2_5',
    });

    expect(navigator.sendBeacon).toHaveBeenCalledTimes(1);
    const [url, body] = (navigator.sendBeacon as jest.Mock).mock.calls[0];
    expect(url).toBe('/api/v1/track');
    expect(JSON.parse(String(body))).toEqual({
      event_type: 'touchpoint_event',
      event_name: 'summary_exposed',
      params: {
        source: 'bottom_nav',
        entry: 'profile_tab',
        type: 'all',
        result: 'success',
        count_bucket: '2_5',
      },
      timestamp: 1780300800000,
    });
    expect(global.fetch).not.toHaveBeenCalled();
  });

  it('falls back to fetch keepalive when sendBeacon returns false', () => {
    (navigator.sendBeacon as jest.Mock).mockReturnValue(false);

    trackEvent('entry_clicked', {
      source: 'profile',
      entry: 'auction_history',
      type: 'pending_payment',
      result: 'clicked',
    });

    expect(global.fetch).toHaveBeenCalledWith('/api/v1/track', expect.objectContaining({
      method: 'POST',
      keepalive: true,
      headers: { 'Content-Type': 'application/json' },
    }));
  });

  it('does not throw when reporting fails', () => {
    (navigator.sendBeacon as jest.Mock).mockReturnValue(false);
    (global.fetch as jest.Mock).mockRejectedValue(new Error('network'));

    expect(() =>
      trackEvent('hot_pull_triggered', {
        source: 'notification_hook',
        entry: 'hot_pull',
        type: 'live_start',
        result: 'failed',
      }),
    ).not.toThrow();
  });

  it.each([
    [0, '0'],
    [1, '1'],
    [2, '2_5'],
    [5, '2_5'],
    [6, '6_10'],
    [10, '6_10'],
    [11, '10_plus'],
  ])('maps count %s to bucket %s', (count, expected) => {
    expect(getCountBucket(count)).toBe(expected);
  });
});
```

- [ ] **Step 2: Run utility test and verify failure**

Run:

```bash
cd frontend/h5
npm test -- src/utils/__tests__/trackEvent.test.ts --runInBand
```

Expected: FAIL because `../trackEvent` does not exist.

- [ ] **Step 3: Implement tracking utility**

Create `frontend/h5/src/utils/trackEvent.ts`:

```ts
import { IS_DEV } from './env';

export type TouchpointEventName =
  | 'summary_exposed'
  | 'entry_clicked'
  | 'notification_list_exposed'
  | 'notification_item_clicked'
  | 'mark_read'
  | 'hot_pull_triggered'
  | 'live_reminder_exposed'
  | 'live_reminder_clicked'
  | 'live_reminder_dismissed';

export type TouchpointSource =
  | 'home'
  | 'bottom_nav'
  | 'profile'
  | 'notification_center'
  | 'mobile_shell'
  | 'notification_hook';

export type TouchpointEntry =
  | 'notification_bell'
  | 'profile_tab'
  | 'auction_history'
  | 'notification_center'
  | 'notification_item'
  | 'mark_all_read'
  | 'hot_pull'
  | 'live_reminder_modal';

export type TouchpointType =
  | 'all'
  | 'pending_payment'
  | 'outbid'
  | 'ending_soon'
  | 'live_start'
  | 'notification';

export type TouchpointResult = 'success' | 'failed' | 'clicked' | 'dismissed' | 'debounced';
export type CountBucket = '0' | '1' | '2_5' | '6_10' | '10_plus';

export interface TouchpointEventParams {
  source: TouchpointSource;
  entry: TouchpointEntry;
  type: TouchpointType;
  result: TouchpointResult;
  countBucket?: CountBucket;
}

interface TrackEventPayload {
  event_type: 'touchpoint_event';
  event_name: TouchpointEventName;
  params: {
    source: TouchpointSource;
    entry: TouchpointEntry;
    type: TouchpointType;
    result: TouchpointResult;
    count_bucket?: CountBucket;
  };
  timestamp: number;
}

const TRACK_ENDPOINT = '/api/v1/track';

export function getCountBucket(count: number): CountBucket {
  if (count <= 0) return '0';
  if (count === 1) return '1';
  if (count <= 5) return '2_5';
  if (count <= 10) return '6_10';
  return '10_plus';
}

function buildPayload(eventName: TouchpointEventName, params: TouchpointEventParams): TrackEventPayload {
  return {
    event_type: 'touchpoint_event',
    event_name: eventName,
    params: {
      source: params.source,
      entry: params.entry,
      type: params.type,
      result: params.result,
      ...(params.countBucket ? { count_bucket: params.countBucket } : {}),
    },
    timestamp: Date.now(),
  };
}

function reportFailure(error: unknown) {
  if (IS_DEV) {
    console.warn('[trackEvent] failed to report touchpoint event', error);
  }
}

export function trackEvent(eventName: TouchpointEventName, params: TouchpointEventParams): void {
  const body = JSON.stringify(buildPayload(eventName, params));

  try {
    if (typeof navigator !== 'undefined' && typeof navigator.sendBeacon === 'function') {
      const sent = navigator.sendBeacon(TRACK_ENDPOINT, body);
      if (sent) return;
    }

    if (typeof fetch === 'function') {
      void fetch(TRACK_ENDPOINT, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body,
        keepalive: true,
      }).catch(reportFailure);
    }
  } catch (error) {
    reportFailure(error);
  }
}
```

- [ ] **Step 4: Run utility test**

Run:

```bash
cd frontend/h5
npm test -- src/utils/__tests__/trackEvent.test.ts --runInBand
```

Expected: PASS.

- [ ] **Step 5: Commit tracking utility**

```bash
git add frontend/h5/src/utils/trackEvent.ts frontend/h5/src/utils/__tests__/trackEvent.test.ts
git commit -m "feat(h5): add touchpoint track event utility"
```

---

## Task 3: Summary Exposure and Entry Click Tracking

**Files:**
- Modify: `frontend/h5/src/hooks/useTouchpointNotifications.ts`
- Modify: `frontend/h5/src/components/MobileShell/BottomNav.tsx`
- Modify: `frontend/h5/src/pages/User/Index.tsx`
- Modify: `frontend/h5/src/pages/Home/index.tsx`
- Modify: `frontend/h5/src/__tests__/components/MobileShell.test.tsx`
- Modify or create: `frontend/h5/src/pages/User/__tests__/Profile.test.tsx`
- Modify or create: `frontend/h5/src/pages/Home/__tests__/Home.test.tsx`

- [ ] **Step 1: Write failing summary exposure tests**

In `frontend/h5/src/__tests__/components/MobileShell.test.tsx`, mock `trackEvent`:

```ts
import { trackEvent } from '../../utils/trackEvent';

jest.mock('../../utils/trackEvent', () => ({
  trackEvent: jest.fn(),
  getCountBucket: (count: number) => (count <= 0 ? '0' : count === 1 ? '1' : count <= 5 ? '2_5' : count <= 10 ? '6_10' : '10_plus'),
}));

const mockTrackEvent = trackEvent as jest.MockedFunction<typeof trackEvent>;
```

Add assertion to the existing “shows unread total badge” test:

```ts
await waitFor(() =>
  expect(mockTrackEvent).toHaveBeenCalledWith('summary_exposed', {
    source: 'bottom_nav',
    entry: 'profile_tab',
    type: 'all',
    result: 'success',
    countBucket: '6_10',
  }),
);
```

Add click assertion:

```ts
it('tracks profile tab entry clicks from bottom navigation', async () => {
  render(
    <MemoryRouter initialEntries={['/']} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <BottomNav />
    </MemoryRouter>,
  );

  const profileLink = await screen.findByRole('link', { name: /我的/ });
  fireEvent.click(profileLink);

  expect(mockTrackEvent).toHaveBeenCalledWith('entry_clicked', {
    source: 'bottom_nav',
    entry: 'profile_tab',
    type: 'all',
    result: 'clicked',
  });
});
```

- [ ] **Step 2: Run focused shell tests and verify failure**

Run:

```bash
cd frontend/h5
npm test -- src/__tests__/components/MobileShell.test.tsx --runInBand
```

Expected: FAIL because no code calls `trackEvent` yet.

- [ ] **Step 3: Implement summary exposure tracking**

Update `frontend/h5/src/hooks/useTouchpointNotifications.ts`:

```ts
import { getCountBucket, trackEvent } from '../utils/trackEvent';
```

Inside the `getTouchpointSummary().then(...)` branch, immediately after `setSummary(next)`:

```ts
trackEvent('summary_exposed', {
  source: 'bottom_nav',
  entry: 'profile_tab',
  type: 'all',
  result: 'success',
  countBucket: getCountBucket(next.unreadTotal ?? 0),
});
```

- [ ] **Step 4: Implement bottom nav click tracking**

Update `frontend/h5/src/components/MobileShell/BottomNav.tsx`:

```ts
import { trackEvent } from '../../utils/trackEvent';
```

Add helper:

```ts
function trackNavClick(path: string) {
  if (path === '/profile') {
    trackEvent('entry_clicked', {
      source: 'bottom_nav',
      entry: 'profile_tab',
      type: 'all',
      result: 'clicked',
    });
  }
}
```

Add to the `Link`:

```tsx
onClick={() => trackNavClick(item.path)}
```

- [ ] **Step 5: Add profile and home entry tracking**

Update `frontend/h5/src/pages/User/Index.tsx`:

```ts
import { trackEvent } from '../../utils/trackEvent';
```

Add handlers:

```ts
const trackAuctionHistoryClick = () => {
  trackEvent('entry_clicked', {
    source: 'profile',
    entry: 'auction_history',
    type: 'pending_payment',
    result: 'clicked',
  });
};

const trackNotificationCenterClick = () => {
  trackEvent('entry_clicked', {
    source: 'profile',
    entry: 'notification_center',
    type: 'notification',
    result: 'clicked',
  });
};
```

Attach them:

```tsx
<Link to="/history" className={styles.menuItem} onClick={trackAuctionHistoryClick}>
```

```tsx
<Link to="/notifications" className={styles.menuItem} onClick={trackNotificationCenterClick}>
```

Update `frontend/h5/src/pages/Home/index.tsx`:

```ts
import { trackEvent } from '@/utils/trackEvent';
```

Attach to notification bell link:

```tsx
<Link
  className={styles.iconButton}
  to="/notifications"
  aria-label="消息通知"
  onClick={() =>
    trackEvent('entry_clicked', {
      source: 'home',
      entry: 'notification_bell',
      type: 'notification',
      result: 'clicked',
    })
  }
>
```

- [ ] **Step 6: Run focused frontend tests**

Run:

```bash
cd frontend/h5
npm test -- src/__tests__/components/MobileShell.test.tsx --runInBand
```

Expected: PASS.

If existing Home/Profile tests are updated, run:

```bash
cd frontend/h5
npm test -- src/pages/User/__tests__/Profile.test.tsx src/pages/Home/__tests__/Home.test.tsx --runInBand
```

Expected: PASS.

- [ ] **Step 7: Commit summary and entry tracking**

```bash
git add frontend/h5/src/hooks/useTouchpointNotifications.ts frontend/h5/src/components/MobileShell/BottomNav.tsx frontend/h5/src/pages/User/Index.tsx frontend/h5/src/pages/Home/index.tsx frontend/h5/src/__tests__/components/MobileShell.test.tsx frontend/h5/src/pages/User/__tests__/Profile.test.tsx frontend/h5/src/pages/Home/__tests__/Home.test.tsx
git commit -m "feat(h5): track touchpoint summary and entry clicks"
```

---

## Task 4: Notification Center and Hot Pull Tracking

**Files:**
- Modify: `frontend/h5/src/pages/Notifications/index.tsx`
- Modify: `frontend/h5/src/pages/Notifications/__tests__/Notifications.test.tsx`
- Modify: `frontend/h5/src/hooks/useNotification.ts`
- Create or modify: `frontend/h5/src/hooks/__tests__/useNotification.test.ts`

- [ ] **Step 1: Write failing notification center tests**

Update `frontend/h5/src/pages/Notifications/__tests__/Notifications.test.tsx`:

```ts
import { trackEvent } from '../../../utils/trackEvent';

jest.mock('../../../utils/trackEvent', () => ({
  trackEvent: jest.fn(),
  getCountBucket: (count: number) => (count <= 0 ? '0' : count === 1 ? '1' : count <= 5 ? '2_5' : count <= 10 ? '6_10' : '10_plus'),
}));

const mockTrackEvent = trackEvent as jest.MockedFunction<typeof trackEvent>;
```

Add assertions:

```ts
await waitFor(() =>
  expect(mockTrackEvent).toHaveBeenCalledWith('notification_list_exposed', {
    source: 'notification_center',
    entry: 'notification_center',
    type: 'notification',
    result: 'success',
    countBucket: '2_5',
  }),
);
```

In click test after click:

```ts
expect(mockTrackEvent).toHaveBeenCalledWith('notification_item_clicked', {
  source: 'notification_center',
  entry: 'notification_item',
  type: 'live_start',
  result: 'clicked',
});
```

Add mark-all test:

```ts
it('tracks mark all read success', async () => {
  render(
    <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <NotificationsPage />
    </MemoryRouter>,
  );

  const button = await screen.findByRole('button', { name: '全部已读' });
  fireEvent.click(button);

  await waitFor(() => expect(mockedNotificationApi.markAllAsRead).toHaveBeenCalled());
  expect(mockTrackEvent).toHaveBeenCalledWith('mark_read', {
    source: 'notification_center',
    entry: 'mark_all_read',
    type: 'all',
    result: 'success',
  });
});
```

- [ ] **Step 2: Run notification center tests and verify failure**

Run:

```bash
cd frontend/h5
npm test -- src/pages/Notifications/__tests__/Notifications.test.tsx --runInBand
```

Expected: FAIL because tracking calls are not implemented.

- [ ] **Step 3: Implement notification center tracking**

Update `frontend/h5/src/pages/Notifications/index.tsx`:

```ts
import { getCountBucket, trackEvent } from '../../utils/trackEvent';
```

After successful list load:

```ts
const items = extractList(listResponse);
setNotifications(items);
setUnreadCount(unreadResponse.count || 0);
trackEvent('notification_list_exposed', {
  source: 'notification_center',
  entry: 'notification_center',
  type: 'notification',
  result: 'success',
  countBucket: getCountBucket(items.length),
});
```

In `handleOpenNotification`, after `await markOneAsRead(notification)` and before navigation:

```ts
trackEvent('notification_item_clicked', {
  source: 'notification_center',
  entry: 'notification_item',
  type: notification.type === 'live_stream_now_live' || notification.type === 'live_stream_starting_soon' ? 'live_start' : 'notification',
  result: 'clicked',
});
```

In `handleMarkAllAsRead`, after local success update:

```ts
trackEvent('mark_read', {
  source: 'notification_center',
  entry: 'mark_all_read',
  type: 'all',
  result: 'success',
});
```

In `catch` of `handleMarkAllAsRead`:

```ts
trackEvent('mark_read', {
  source: 'notification_center',
  entry: 'mark_all_read',
  type: 'all',
  result: 'failed',
});
```

- [ ] **Step 4: Write failing hot-pull hook tests**

Create `frontend/h5/src/hooks/__tests__/useNotification.test.ts` with `renderHook` from `@testing-library/react`:

```ts
import { act, renderHook, waitFor } from '@testing-library/react';
import { useNotification } from '../useNotification';
import { notificationApi } from '../../services/notification';
import { trackEvent } from '../../utils/trackEvent';

jest.mock('../../services/notification', () => ({
  notificationApi: {
    list: jest.fn(),
    getUnreadCount: jest.fn(),
    markAsRead: jest.fn(),
    markAllAsRead: jest.fn(),
    hotPull: jest.fn(),
  },
}));

jest.mock('../../utils/trackEvent', () => ({
  trackEvent: jest.fn(),
  getCountBucket: (count: number) => (count <= 0 ? '0' : count === 1 ? '1' : count <= 5 ? '2_5' : count <= 10 ? '6_10' : '10_plus'),
}));

const mockedNotificationApi = notificationApi as jest.Mocked<typeof notificationApi>;
const mockTrackEvent = trackEvent as jest.MockedFunction<typeof trackEvent>;

describe('useNotification touchpoint tracking', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedNotificationApi.list.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20 });
    mockedNotificationApi.getUnreadCount.mockResolvedValue({ count: 0 });
    mockedNotificationApi.hotPull.mockResolvedValue({
      notifications: [{ id: 1, type: 'live_stream_now_live', title: '开播', content: '已开播', created_at: '2026-06-02T00:00:00Z' }],
      has_more: false,
    });
  });

  it('tracks hot pull success with returned notification bucket', async () => {
    const { result } = renderHook(() => useNotification());

    await act(async () => {
      await result.current.hotPullNotifications();
    });

    expect(mockTrackEvent).toHaveBeenCalledWith('hot_pull_triggered', {
      source: 'notification_hook',
      entry: 'hot_pull',
      type: 'live_start',
      result: 'success',
      countBucket: '1',
    });
  });

  it('tracks debounce skip when hot pull is called twice quickly', async () => {
    const { result } = renderHook(() => useNotification());

    await act(async () => {
      await result.current.hotPullNotifications();
      await result.current.hotPullNotifications();
    });

    await waitFor(() =>
      expect(mockTrackEvent).toHaveBeenCalledWith('hot_pull_triggered', {
        source: 'notification_hook',
        entry: 'hot_pull',
        type: 'live_start',
        result: 'debounced',
        countBucket: '0',
      }),
    );
  });
});
```

- [ ] **Step 5: Implement hot-pull tracking**

Update `frontend/h5/src/hooks/useNotification.ts`:

```ts
import { getCountBucket, trackEvent } from '../utils/trackEvent';
```

In debounce branch:

```ts
trackEvent('hot_pull_triggered', {
  source: 'notification_hook',
  entry: 'hot_pull',
  type: 'live_start',
  result: 'debounced',
  countBucket: '0',
});
```

After `const data = await notificationApi.hotPull();`:

```ts
trackEvent('hot_pull_triggered', {
  source: 'notification_hook',
  entry: 'hot_pull',
  type: 'live_start',
  result: 'success',
  countBucket: getCountBucket(data.notifications?.length ?? 0),
});
```

In `catch`:

```ts
trackEvent('hot_pull_triggered', {
  source: 'notification_hook',
  entry: 'hot_pull',
  type: 'live_start',
  result: 'failed',
  countBucket: '0',
});
```

- [ ] **Step 6: Run notification and hook tests**

Run:

```bash
cd frontend/h5
npm test -- src/pages/Notifications/__tests__/Notifications.test.tsx --runInBand
```

```bash
cd frontend/h5
npm test -- src/hooks/__tests__/useNotification.test.ts --runInBand
```

Expected: PASS.

- [ ] **Step 7: Commit notification tracking**

```bash
git add frontend/h5/src/pages/Notifications/index.tsx frontend/h5/src/pages/Notifications/__tests__/Notifications.test.tsx frontend/h5/src/hooks/useNotification.ts frontend/h5/src/hooks/__tests__/useNotification.test.ts
git commit -m "feat(h5): track notification touchpoints"
```

---

## Task 5: Live Reminder Modal Tracking and Final Verification

**Files:**
- Modify: `frontend/h5/src/components/MobileShell/MobileContainer.tsx`
- Modify: `frontend/h5/src/components/LiveReminderModal/index.tsx`
- Modify: `frontend/h5/src/__tests__/components/MobileShell.test.tsx`

- [ ] **Step 1: Write failing live reminder tests**

In `frontend/h5/src/__tests__/components/MobileShell.test.tsx`, add assertions to the existing “opens live reminder once” test:

```ts
await waitFor(() =>
  expect(mockTrackEvent).toHaveBeenCalledWith('live_reminder_exposed', {
    source: 'mobile_shell',
    entry: 'live_reminder_modal',
    type: 'live_start',
    result: 'success',
  }),
);
```

Add click and dismiss test:

```ts
it('tracks live reminder click and dismiss actions', async () => {
  mockGetPendingLiveReminder.mockResolvedValue({
    hasReminder: true,
    stream: {
      id: 1,
      name: '云端珍藏直播间',
      avatarUrl: '',
      statusText: '正在直播',
      liveRoomId: 1,
      startedAt: 1717000000000,
    },
  });

  render(
    <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <ThemeProvider>
        <MobileContainer>
          <main>页面内容</main>
        </MobileContainer>
      </ThemeProvider>
    </MemoryRouter>,
  );

  fireEvent.click(await screen.findByRole('button', { name: '立即前往' }));
  expect(mockTrackEvent).toHaveBeenCalledWith('live_reminder_clicked', {
    source: 'mobile_shell',
    entry: 'live_reminder_modal',
    type: 'live_start',
    result: 'clicked',
  });
});
```

- [ ] **Step 2: Run shell tests and verify failure**

Run:

```bash
cd frontend/h5
npm test -- src/__tests__/components/MobileShell.test.tsx --runInBand
```

Expected: FAIL because live reminder tracking is not implemented.

- [ ] **Step 3: Implement live reminder exposure tracking**

Update `frontend/h5/src/components/MobileShell/MobileContainer.tsx`:

```ts
import { trackEvent } from '../../utils/trackEvent';
```

After `setIsReminderOpen(true)`:

```ts
trackEvent('live_reminder_exposed', {
  source: 'mobile_shell',
  entry: 'live_reminder_modal',
  type: 'live_start',
  result: 'success',
});
```

- [ ] **Step 4: Implement modal click and dismiss tracking**

Update `frontend/h5/src/components/LiveReminderModal/index.tsx`:

```ts
import { trackEvent } from '../../utils/trackEvent';
```

Add helper functions:

```ts
const trackDismiss = () => {
  trackEvent('live_reminder_dismissed', {
    source: 'mobile_shell',
    entry: 'live_reminder_modal',
    type: 'live_start',
    result: 'dismissed',
  });
  onClose();
};

const handleJump = () => {
  trackEvent('live_reminder_clicked', {
    source: 'mobile_shell',
    entry: 'live_reminder_modal',
    type: 'live_start',
    result: 'clicked',
  });
  onClose();
  navigate(`/live`);
};
```

Replace overlay and cancel button close handlers:

```tsx
onClick={trackDismiss}
```

```tsx
onClick={trackDismiss}
```

- [ ] **Step 5: Run shell tests**

Run:

```bash
cd frontend/h5
npm test -- src/__tests__/components/MobileShell.test.tsx --runInBand
```

Expected: PASS.

- [ ] **Step 6: Run full focused verification**

Run:

```bash
cd backend/gateway
go test ./pkg/metrics ./handler ./router
```

Expected: PASS.

Run:

```bash
cd frontend/h5
npm test -- src/utils/__tests__/trackEvent.test.ts src/__tests__/components/MobileShell.test.tsx src/pages/Notifications/__tests__/Notifications.test.tsx --runInBand
```

Expected: PASS.

Run diagnostics for edited TS/TSX files with the editor diagnostics tool and fix introduced issues.

- [ ] **Step 7: Commit live reminder tracking and verification docs**

```bash
git add frontend/h5/src/components/MobileShell/MobileContainer.tsx frontend/h5/src/components/LiveReminderModal/index.tsx frontend/h5/src/__tests__/components/MobileShell.test.tsx
git commit -m "feat(h5): track live reminder touchpoints"
```

---

## Task 6: Final Review and Delivery

**Files:**
- Modify: no code changes expected unless verification reveals a defect.

- [ ] **Step 1: Inspect commit series and worktree**

Run:

```bash
git log --oneline -n 8
git status --short
```

Expected: implementation commits are present and worktree is clean.

- [ ] **Step 2: Run final backend and frontend checks**

Run:

```bash
cd backend/gateway
go test ./...
```

Expected: PASS.

Run:

```bash
cd frontend/h5
npm test -- --runInBand
```

Expected: PASS or document existing unrelated failures with exact failing suite names and error messages.

- [ ] **Step 3: Manual metric smoke check**

If local services are running, POST one synthetic event:

```bash
curl -X POST http://localhost:8080/api/v1/track \
  -H 'Content-Type: application/json' \
  -d '{"event_type":"touchpoint_event","event_name":"summary_exposed","params":{"source":"bottom_nav","entry":"profile_tab","type":"all","result":"success"},"timestamp":1780300800000}'
```

Expected response:

```json
{"status":"ok"}
```

Then check:

```bash
curl http://localhost:9090/metrics | grep touchpoint_event_total
```

Expected: output contains `touchpoint_event_total{entry="profile_tab",event="summary_exposed",result="success",source="bottom_nav",type="all"}`.

- [ ] **Step 4: Prepare delivery summary**

Summarize:
- Commits created.
- Events instrumented.
- Metrics added.
- Tests run and results.
- Any skipped manual smoke checks and why.

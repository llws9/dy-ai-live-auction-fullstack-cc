# Test Dashboard Demo Theater Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `frontend/test-dashboard` 的 `/test/screen` 从历史统计大屏升级为面向评委的一键 `UserJourney` 直播竞拍演示剧场。

**Architecture:** 后端 `user_journey` 报告增加向后兼容的 `demo_snapshot`，为演示主屏提供当前价、领先者、事件数、订单数和库存变化等可视化事实。前端新增纯映射层 `demoTheater.ts`，将 WebSocket 进度和最终报告转换成 UI view model；`Screen.tsx` 只负责一键启动、连接进度流、轮询报告和渲染剧场。技术控制台 `E2E`、`UserJourney`、`History`、`Report` 不改变职责。

**Tech Stack:** Go `backend/test`、React 18、TypeScript、Vite、Zustand、Axios、Vitest、React Testing Library。

---

## Scope And Safety

当前主工作区已有与本计划无关的未提交改动。执行本计划前必须使用隔离 worktree，避免污染或覆盖用户改动。

执行约束：

- 前端测试平台仍通过 gateway `/api` 与 `/ws` 入口访问测试服务。
- `/test/screen` 不暴露 `seller_id`、`bidder_ids`、`duration` 等技术参数。
- 主屏允许展示演示快照，但最终可信结论必须来自 `UserJourneyReport.all_ok` 和步骤断言。
- 不删除 `/test/history` 的历史记录能力；历史记录继续由 `History.tsx` 承载。
- 每个任务完成后单独提交。

## File Structure

Backend:

- Modify: `backend/test/scenario/user_journey/orchestrator.go`
  - 新增 `DemoSnapshot`。
  - `Report` 增加可选 `demo_snapshot`。
  - 在竞拍、点天灯、购买、校验阶段更新快照。
  - WebSocket progress metrics 携带当前快照。
- Modify: `backend/test/scenario/user_journey/orchestrator_test.go`
  - 验证最终报告包含快照。
  - 验证 progress metrics 包含快照。

Frontend:

- Modify: `frontend/test-dashboard/package.json`
- Modify: `frontend/test-dashboard/package-lock.json`
- Modify: `frontend/test-dashboard/vite.config.ts`
- Create: `frontend/test-dashboard/src/test/setup.ts`
- Modify: `frontend/test-dashboard/src/api/test.ts`
- Create: `frontend/test-dashboard/src/pages/demoTheater.ts`
- Create: `frontend/test-dashboard/src/pages/demoTheater.test.ts`
- Modify: `frontend/test-dashboard/src/pages/Screen.tsx`
- Create: `frontend/test-dashboard/src/pages/Screen.test.tsx`

## Task 0: Isolate Worktree

**Files:**
- No repository file changes.

- [ ] **Step 1: Create isolated worktree**

Run from the main workspace:

```bash
git fetch origin
git worktree add ../dy-ai-live-auction-fullstack-cc-demo-theater -b feat/test-dashboard-demo-theater main
cd ../dy-ai-live-auction-fullstack-cc-demo-theater
```

Expected: new worktree on `feat/test-dashboard-demo-theater`.

- [ ] **Step 2: Verify baseline**

Run:

```bash
git branch --show-current
pwd
git status --short
```

Expected:

```text
feat/test-dashboard-demo-theater
<absolute path ending with dy-ai-live-auction-fullstack-cc-demo-theater>
```

`git status --short` must be empty.

## Task 1: Add Demo Snapshot To UserJourney Report

**Files:**
- Modify: `backend/test/scenario/user_journey/orchestrator.go`
- Modify: `backend/test/scenario/user_journey/orchestrator_test.go`

- [ ] **Step 1: Write failing backend tests**

In `backend/test/scenario/user_journey/orchestrator_test.go`, append these assertions to `TestRunHappyPathProducesEvidenceReport` after `assert.Equal(t, int64(0), report.StockAfter)`:

```go
require.NotNil(t, report.DemoSnapshot)
assert.Equal(t, "110.00", report.DemoSnapshot.CurrentPrice)
assert.Equal(t, "买家 2001", report.DemoSnapshot.LeaderLabel)
assert.Equal(t, int64(1), report.DemoSnapshot.BidCount)
assert.Equal(t, int64(1), report.DemoSnapshot.OrderCount)
assert.Equal(t, int64(1), report.DemoSnapshot.StockBefore)
assert.Equal(t, int64(0), report.DemoSnapshot.StockAfter)
assert.Equal(t, "verify", report.DemoSnapshot.HighlightedEvent)
```

Add this test near the other `user_journey` report tests:

```go
func TestRunEmitsDemoSnapshotInProgressMetrics(t *testing.T) {
	ctx := context.Background()
	emitter := &fakeEmitter{}

	report, err := New(newFakeBiz(), &fakeInternalClient{}, &fakeSeedRecorder{}, Config{TestID: "tj_demo"}).Run(ctx, emitter)
	require.NoError(t, err)
	require.NotNil(t, report.DemoSnapshot)

	metrics := emitter.metricsForStep("sky_lamp")
	require.NotNil(t, metrics)
	snapshot, ok := metrics["demo_snapshot"].(DemoSnapshot)
	require.True(t, ok)
	assert.Equal(t, "110.00", snapshot.CurrentPrice)
	assert.Equal(t, "买家 2001", snapshot.LeaderLabel)
	assert.Equal(t, "sky_lamp", snapshot.HighlightedEvent)
}
```

Replace the existing `fakeEmitter` helper with this exact version:

```go
type fakeEmitter struct {
	steps   []string
	metrics map[string]map[string]any
}

func (f *fakeEmitter) Emit(_ int, step string, metrics map[string]any) {
	f.steps = append(f.steps, step)
	if f.metrics == nil {
		f.metrics = make(map[string]map[string]any)
	}
	f.metrics[step] = metrics
}

func (f *fakeEmitter) lastStep() string {
	if len(f.steps) == 0 {
		return ""
	}
	return f.steps[len(f.steps)-1]
}

func (f *fakeEmitter) metricsForStep(step string) map[string]any {
	if f.metrics == nil {
		return nil
	}
	return f.metrics[step]
}
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd backend/test
go test ./scenario/user_journey -run 'TestRun(HappyPathProducesEvidenceReport|EmitsDemoSnapshotInProgressMetrics)' -count=1
```

Expected: FAIL because `Report.DemoSnapshot` and `DemoSnapshot` are undefined.

- [ ] **Step 3: Implement backend snapshot**

In `backend/test/scenario/user_journey/orchestrator.go`, add above `Report`:

```go
type DemoSnapshot struct {
	CurrentPrice     string `json:"current_price,omitempty"`
	LeaderLabel      string `json:"leader_label,omitempty"`
	BidCount         int64  `json:"bid_count,omitempty"`
	OrderCount       int64  `json:"order_count,omitempty"`
	StockBefore      int64  `json:"stock_before,omitempty"`
	StockAfter       int64  `json:"stock_after,omitempty"`
	HighlightedEvent string `json:"highlighted_event,omitempty"`
}
```

Add this field to `Report`:

```go
DemoSnapshot *DemoSnapshot `json:"demo_snapshot,omitempty"`
```

Initialize it in the `rep := &Report{}` literal in `Run`:

```go
DemoSnapshot: &DemoSnapshot{
	CurrentPrice: "100.00",
	StockBefore:  1,
	StockAfter:   1,
},
```

Add this helper below `recordAndError`:

```go
func (o *Orchestrator) updateDemoSnapshot(rep *Report, mutate func(*DemoSnapshot)) {
	if rep.DemoSnapshot == nil {
		rep.DemoSnapshot = &DemoSnapshot{}
	}
	mutate(rep.DemoSnapshot)
}
```

Update scenario phases:

```go
// In auctionBid, after successful PlaceBid and before o.record(...)
o.updateDemoSnapshot(rep, func(s *DemoSnapshot) {
	s.CurrentPrice = "110.00"
	s.LeaderLabel = fmt.Sprintf("买家 %d", buyer.UserID)
	s.BidCount = 1
	s.StockBefore = rep.StockBefore
	s.StockAfter = rep.StockBefore
	s.HighlightedEvent = "bid"
})

// In skyLamp, after successful SubscribeSkyLamp and before o.record(...)
o.updateDemoSnapshot(rep, func(s *DemoSnapshot) {
	s.CurrentPrice = "110.00"
	s.LeaderLabel = fmt.Sprintf("买家 %d", buyer.UserID)
	s.BidCount = 1
	s.HighlightedEvent = "sky_lamp"
})

// In fixedPricePurchase, immediately after rep.StockAfter = 0
o.updateDemoSnapshot(rep, func(s *DemoSnapshot) {
	s.OrderCount = 1
	s.StockBefore = rep.StockBefore
	s.StockAfter = rep.StockAfter
	s.HighlightedEvent = "order"
})

// In verify, before o.record(...)
o.updateDemoSnapshot(rep, func(s *DemoSnapshot) {
	s.StockBefore = rep.StockBefore
	s.StockAfter = rep.StockAfter
	s.HighlightedEvent = "verify"
})
```

Replace the `p.Emit(...)` block inside `record` with:

```go
metrics := map[string]any{
	"ok":          step.OK,
	"duration_ms": step.DurationMs,
	"ref_id":      step.RefID,
	"message":     step.Message,
	"status_code": step.StatusCode,
}
if rep.DemoSnapshot != nil {
	metrics["demo_snapshot"] = *rep.DemoSnapshot
}
p.Emit(progress, step.Step, metrics)
```

- [ ] **Step 4: Run backend tests**

Run:

```bash
cd backend/test
go test ./scenario/user_journey -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit backend snapshot**

Run:

```bash
git add backend/test/scenario/user_journey/orchestrator.go backend/test/scenario/user_journey/orchestrator_test.go
git commit -m "feat(test): expose user journey demo snapshot"
```

Expected: commit succeeds with only the two listed files.

## Task 2: Add Test Infrastructure For Test Dashboard

**Files:**
- Modify: `frontend/test-dashboard/package.json`
- Modify: `frontend/test-dashboard/package-lock.json`
- Modify: `frontend/test-dashboard/vite.config.ts`
- Create: `frontend/test-dashboard/src/test/setup.ts`

- [ ] **Step 1: Install frontend test dependencies**

Run:

```bash
cd frontend/test-dashboard
npm install -D vitest jsdom @testing-library/react @testing-library/jest-dom @testing-library/user-event
```

Expected: `package.json` and `package-lock.json` update.

- [ ] **Step 2: Add package scripts**

Modify `frontend/test-dashboard/package.json` scripts to:

```json
{
  "dev": "vite",
  "build": "tsc && vite build",
  "preview": "vite preview",
  "test": "vitest",
  "test:run": "vitest run"
}
```

- [ ] **Step 3: Configure Vitest**

Replace `frontend/test-dashboard/vite.config.ts` with:

```ts
/// <reference types="vitest" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': path.resolve(__dirname, 'src') },
    dedupe: ['react', 'react-dom'],
  },
  optimizeDeps: {
    include: ['react', 'react-dom', 'recharts'],
  },
  server: {
    port: 5174,
    host: '0.0.0.0',
    proxy: {
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
      '/ws': { target: 'http://localhost:8080', changeOrigin: true, ws: true },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    globals: true,
  },
});
```

Create `frontend/test-dashboard/src/test/setup.ts`:

```ts
import '@testing-library/jest-dom/vitest';
```

- [ ] **Step 4: Verify test runner and build**

Run:

```bash
cd frontend/test-dashboard
npm run test:run -- --passWithNoTests
npm run build
```

Expected: both commands PASS.

- [ ] **Step 5: Commit test infrastructure**

Run:

```bash
git add frontend/test-dashboard/package.json frontend/test-dashboard/package-lock.json frontend/test-dashboard/vite.config.ts frontend/test-dashboard/src/test/setup.ts
git commit -m "test(test-dashboard): add frontend test runner"
```

Expected: commit succeeds with only the listed files.

## Task 3: Add Demo Theater Mapping Layer

**Files:**
- Modify: `frontend/test-dashboard/src/api/test.ts`
- Create: `frontend/test-dashboard/src/pages/demoTheater.ts`
- Create: `frontend/test-dashboard/src/pages/demoTheater.test.ts`

- [ ] **Step 1: Extend frontend API types**

In `frontend/test-dashboard/src/api/test.ts`, add above `UserJourneyReport`:

```ts
export interface DemoSnapshot {
  current_price?: string;
  leader_label?: string;
  bid_count?: number;
  order_count?: number;
  stock_before?: number;
  stock_after?: number;
  highlighted_event?: 'bid' | 'sky_lamp' | 'fixed_price' | 'order' | 'verify';
}
```

Add to `UserJourneyReport`:

```ts
demo_snapshot?: DemoSnapshot;
```

- [ ] **Step 2: Write failing mapping tests**

Create `frontend/test-dashboard/src/pages/demoTheater.test.ts`:

```ts
import { describe, expect, it } from 'vitest';
import type { UserJourneyReport } from '@/api/test';
import type { ProgressMsg } from '@/store/wsStore';
import { buildDemoTheaterModel, DEMO_USER_JOURNEY_CONFIG } from './demoTheater';

describe('demoTheater', () => {
  it('uses standard user journey config', () => {
    expect(DEMO_USER_JOURNEY_CONFIG).toEqual({
      include_reminder: true,
      include_sky_lamp: true,
      include_fixed_price: true,
      auction_duration_sec: 30,
      buyer_count: 1,
      keep_evidence: true,
    });
  });

  it('builds idle model', () => {
    const model = buildDemoTheaterModel(baseInput());
    expect(model.stage).toBe('idle');
    expect(model.currentPrice).toBe('待启动');
    expect(model.technicalLine).toBe('等待一键启动 UserJourney 标准剧本');
  });

  it('maps sky lamp progress into live story', () => {
    const model = buildDemoTheaterModel({
      ...baseInput(),
      connected: true,
      testID: 'tj_live',
      progress: 62,
      step: 'sky_lamp',
      history: [
        progress('tj_live', 50, 'auction_bid', {
          demo_snapshot: {
            current_price: '110.00',
            leader_label: '买家 2001',
            bid_count: 1,
            stock_before: 1,
            stock_after: 1,
            highlighted_event: 'bid',
          },
        }),
        progress('tj_live', 62, 'sky_lamp', {
          demo_snapshot: {
            current_price: '110.00',
            leader_label: '买家 2001',
            bid_count: 1,
            highlighted_event: 'sky_lamp',
          },
        }),
      ],
    });
    expect(model.stage).toBe('running');
    expect(model.liveBadge).toBe('LIVE');
    expect(model.currentPrice).toBe('¥110.00');
    expect(model.leaderLabel).toBe('买家 2001');
    expect(model.events.at(-1)?.title).toBe('点天灯触发');
  });

  it('shows success conclusions from report', () => {
    const report: UserJourneyReport = {
      test_run_id: 'tj_done',
      all_ok: true,
      order_id: 501,
      stock_before: 1,
      stock_after: 0,
      demo_snapshot: {
        current_price: '110.00',
        leader_label: '买家 2001',
        bid_count: 1,
        order_count: 1,
        stock_before: 1,
        stock_after: 0,
        highlighted_event: 'verify',
      },
    };
    const model = buildDemoTheaterModel({ ...baseInput(), testID: 'tj_done', progress: 100, step: 'verify', report });
    expect(model.stage).toBe('success');
    expect(model.conclusions.every((item) => item.status === 'passed')).toBe(true);
    expect(model.reportPath).toBe('/test/report/tj_done');
    expect(model.stockLabel).toBe('1 → 0');
  });

  it('shows business-stage failure', () => {
    const model = buildDemoTheaterModel({
      ...baseInput(),
      testID: 'tj_fail',
      progress: 62,
      step: 'sky_lamp',
      report: { test_run_id: 'tj_fail', all_ok: false, error: 'sky_lamp failed: upstream timeout' },
    });
    expect(model.stage).toBe('failed');
    expect(model.failureTitle).toBe('点天灯阶段失败');
    expect(model.failureMessage).toContain('upstream timeout');
    expect(model.reportPath).toBe('/test/report/tj_fail');
  });
});

function baseInput() {
  return {
    connected: false,
    testID: null,
    progress: 0,
    step: '',
    history: [],
    report: null,
    error: null,
    starting: false,
  };
}

function progress(testID: string, value: number, step: string, metrics: Record<string, unknown>): ProgressMsg {
  return { test_id: testID, progress: value, step, metrics, ts: Date.now() };
}
```

- [ ] **Step 3: Run mapping tests to verify failure**

Run:

```bash
cd frontend/test-dashboard
npm run test:run -- src/pages/demoTheater.test.ts
```

Expected: FAIL because `./demoTheater` does not exist.

- [ ] **Step 4: Implement mapping module**

Create `frontend/test-dashboard/src/pages/demoTheater.ts` with:

```ts
import type { DemoSnapshot, UserJourneyConfig, UserJourneyReport } from '@/api/test';
import type { ProgressMsg } from '@/store/wsStore';

export const DEMO_USER_JOURNEY_CONFIG: Required<UserJourneyConfig> = {
  include_reminder: true,
  include_sky_lamp: true,
  include_fixed_price: true,
  auction_duration_sec: 30,
  buyer_count: 1,
  keep_evidence: true,
};

export type DemoStage = 'idle' | 'starting' | 'running' | 'success' | 'failed';
export type ConclusionStatus = 'pending' | 'passed' | 'failed';

export interface DemoEvent {
  step: string;
  title: string;
  description: string;
  tone: 'neutral' | 'blue' | 'orange' | 'green' | 'red';
}

export interface DemoConclusion {
  title: string;
  description: string;
  status: ConclusionStatus;
}

export interface DemoTheaterModel {
  stage: DemoStage;
  heroTitle: string;
  primaryActionLabel: string;
  liveBadge: 'READY' | 'STARTING' | 'LIVE' | 'DONE' | 'FAILED';
  currentPrice: string;
  leaderLabel: string;
  bidCount: number;
  orderCount: number;
  stockLabel: string;
  highlightedEvent: DemoSnapshot['highlighted_event'] | 'idle';
  events: DemoEvent[];
  conclusions: DemoConclusion[];
  progressLabel: string;
  technicalLine: string;
  reportPath: string | null;
  failureTitle: string | null;
  failureMessage: string | null;
}

export interface BuildDemoTheaterModelInput {
  connected: boolean;
  testID: string | null;
  progress: number;
  step: string;
  history: ProgressMsg[];
  report: UserJourneyReport | null;
  error: string | null;
  starting: boolean;
}

const STEP_EVENTS: Record<string, DemoEvent> = {
  prepare: { step: 'prepare', title: '演示资产已创建', description: '商家、商品、直播间、竞拍规则和买家资金已准备完成', tone: 'blue' },
  enter_live: { step: 'enter_live', title: '买家进入直播间', description: '直播间切换为可交互状态', tone: 'blue' },
  reminder: { step: 'reminder', title: '关注提醒已验证', description: '买家关注直播间，提醒状态完成回读', tone: 'neutral' },
  auction_bid: { step: 'auction_bid', title: '实时出价发生', description: '当前价刷新，领先者进入竞拍态', tone: 'blue' },
  sky_lamp: { step: 'sky_lamp', title: '点天灯触发', description: '高权重竞价反馈出现，领先状态被锁定展示', tone: 'orange' },
  fixed_price_purchase: { step: 'fixed_price_purchase', title: '一口价成交', description: '订单生成，库存开始扣减', tone: 'green' },
  verify: { step: 'verify', title: '闭环校验通过', description: '订单、库存、余额和竞拍结果完成一致性校验', tone: 'green' },
  cleanup: { step: 'cleanup', title: '证据已保留', description: '演示报告可用于技术下钻', tone: 'neutral' },
};

const CONCLUSIONS: DemoConclusion[] = [
  { title: '业务闭环成立', description: '进房、竞拍、成交、订单链路通过', status: 'pending' },
  { title: '并发结果唯一', description: '赢家唯一，订单唯一，无重复成交', status: 'pending' },
  { title: '资产状态一致', description: '库存、余额、订单状态对齐', status: 'pending' },
];

export function buildDemoTheaterModel(input: BuildDemoTheaterModelInput): DemoTheaterModel {
  const snapshot = latestSnapshot(input);
  const reportID = input.report?.test_run_id || input.testID;
  const stage = resolveStage(input);
  const events = input.history.map((m) => STEP_EVENTS[m.step]).filter((e): e is DemoEvent => Boolean(e));
  return {
    stage,
    heroTitle: 'AI 直播竞拍全链路验收',
    primaryActionLabel: stage === 'failed' || stage === 'success' ? '重新演示' : '开始演示',
    liveBadge: resolveLiveBadge(stage),
    currentPrice: snapshot?.current_price ? `¥${snapshot.current_price}` : '待启动',
    leaderLabel: snapshot?.leader_label || '等待领先者',
    bidCount: snapshot?.bid_count ?? 0,
    orderCount: snapshot?.order_count ?? 0,
    stockLabel: formatStock(snapshot, input.report),
    highlightedEvent: snapshot?.highlighted_event || 'idle',
    events,
    conclusions: CONCLUSIONS.map((item) => ({ ...item, status: stage === 'success' ? 'passed' : stage === 'failed' ? 'failed' : 'pending' })),
    progressLabel: `${Math.max(0, Math.min(100, input.progress))}%`,
    technicalLine: technicalLine(input),
    reportPath: reportID ? `/test/report/${reportID}` : null,
    failureTitle: stage === 'failed' ? `${stepBusinessName(input.step)}阶段失败` : null,
    failureMessage: input.error || input.report?.error || null,
  };
}

function latestSnapshot(input: BuildDemoTheaterModelInput): DemoSnapshot | undefined {
  const fromHistory = [...input.history].reverse().map((m) => m.metrics?.demo_snapshot).find(Boolean) as DemoSnapshot | undefined;
  return input.report?.demo_snapshot || fromHistory;
}

function resolveStage(input: BuildDemoTheaterModelInput): DemoStage {
  if (input.error || input.report?.all_ok === false) return 'failed';
  if (input.report?.all_ok === true) return 'success';
  if (input.starting) return 'starting';
  if (input.testID || input.connected || input.progress > 0) return 'running';
  return 'idle';
}

function resolveLiveBadge(stage: DemoStage): DemoTheaterModel['liveBadge'] {
  if (stage === 'idle') return 'READY';
  if (stage === 'starting') return 'STARTING';
  if (stage === 'success') return 'DONE';
  if (stage === 'failed') return 'FAILED';
  return 'LIVE';
}

function formatStock(snapshot: DemoSnapshot | undefined, report: UserJourneyReport | null): string {
  const before = snapshot?.stock_before ?? report?.stock_before;
  const after = snapshot?.stock_after ?? report?.stock_after;
  if (before == null && after == null) return '待验证';
  return `${before ?? '-'} → ${after ?? '-'}`;
}

function stepBusinessName(step: string): string {
  const names: Record<string, string> = {
    prepare: '演示准备',
    enter_live: '进直播间',
    reminder: '关注提醒',
    auction_bid: '出价',
    sky_lamp: '点天灯',
    fixed_price_purchase: '一口价购买',
    verify: '汇总校验',
    cleanup: '证据清理',
  };
  return names[step] || '演示';
}

function technicalLine(input: BuildDemoTheaterModelInput): string {
  if (!input.testID && !input.report?.test_run_id) return '等待一键启动 UserJourney 标准剧本';
  const ws = input.connected ? 'WS 已连接' : 'WS 未连接';
  return `test_id=${input.testID || input.report?.test_run_id || '-'} · ${ws} · step=${input.step || '-'}`;
}
```

- [ ] **Step 5: Run mapping tests**

Run:

```bash
cd frontend/test-dashboard
npm run test:run -- src/pages/demoTheater.test.ts
```

Expected: PASS.

- [ ] **Step 6: Commit mapping layer**

Run:

```bash
git add frontend/test-dashboard/src/api/test.ts frontend/test-dashboard/src/pages/demoTheater.ts frontend/test-dashboard/src/pages/demoTheater.test.ts
git commit -m "feat(test-dashboard): map user journey to demo theater model"
```

Expected: commit succeeds with only the listed files.

## Task 4: Replace Screen With One-Click Demo Theater

**Files:**
- Modify: `frontend/test-dashboard/src/pages/Screen.tsx`
- Create: `frontend/test-dashboard/src/pages/Screen.test.tsx`

- [ ] **Step 1: Write failing Screen tests**

Create `frontend/test-dashboard/src/pages/Screen.test.tsx`:

```tsx
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Screen from './Screen';
import { useWSStore } from '@/store/wsStore';
import { discoverWS, startUserJourney } from '@/api/test';

vi.mock('@/api/test', () => ({
  startUserJourney: vi.fn(async () => 'tj_demo'),
  discoverWS: vi.fn(async () => 'ws://localhost:18092/ws/test/progress?test_id=tj_demo'),
  cancelTest: vi.fn(async () => undefined),
}));

vi.mock('@/hooks/usePollReport', () => ({
  usePollReport: () => ({ start: vi.fn(), cancel: vi.fn() }),
}));

describe('Screen demo theater', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useWSStore.setState({ connected: false, testID: null, progress: 0, step: '', metrics: {}, history: [], socket: null });
  });

  it('renders judge-facing theater instead of history dashboard', () => {
    renderScreen();
    expect(screen.getByText('AI 直播竞拍全链路验收')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '开始演示' })).toBeInTheDocument();
    expect(screen.getByText('业务闭环成立')).toBeInTheDocument();
    expect(screen.queryByText('总任务')).not.toBeInTheDocument();
  });

  it('starts standard user journey demo from the big screen', async () => {
    const user = userEvent.setup();
    renderScreen();
    await user.click(screen.getByRole('button', { name: '开始演示' }));
    await waitFor(() => {
      expect(startUserJourney).toHaveBeenCalledWith({
        include_reminder: true,
        include_sky_lamp: true,
        include_fixed_price: true,
        auction_duration_sec: 30,
        buyer_count: 1,
        keep_evidence: true,
      });
      expect(discoverWS).toHaveBeenCalledWith('tj_demo');
    });
  });

  it('shows live bid and sky lamp story from ws history', () => {
    useWSStore.setState({
      connected: true,
      testID: 'tj_live',
      progress: 62,
      step: 'sky_lamp',
      metrics: {},
      history: [
        { test_id: 'tj_live', progress: 50, step: 'auction_bid', metrics: { demo_snapshot: { current_price: '110.00', leader_label: '买家 2001', bid_count: 1, stock_before: 1, stock_after: 1, highlighted_event: 'bid' } }, ts: Date.now() },
        { test_id: 'tj_live', progress: 62, step: 'sky_lamp', metrics: { demo_snapshot: { current_price: '110.00', leader_label: '买家 2001', bid_count: 1, highlighted_event: 'sky_lamp' } }, ts: Date.now() },
      ],
      socket: null,
    });
    renderScreen();
    expect(screen.getByText('¥110.00')).toBeInTheDocument();
    expect(screen.getByText('买家 2001')).toBeInTheDocument();
    expect(screen.getByText('点天灯触发')).toBeInTheDocument();
  });
});

function renderScreen() {
  render(
    <MemoryRouter>
      <Screen />
    </MemoryRouter>,
  );
}
```

- [ ] **Step 2: Run Screen tests to verify failure**

Run:

```bash
cd frontend/test-dashboard
npm run test:run -- src/pages/Screen.test.tsx
```

Expected: FAIL because current `Screen.tsx` renders the historical dashboard and does not call `startUserJourney`.

- [ ] **Step 3: Implement Screen theater**

Replace `frontend/test-dashboard/src/pages/Screen.tsx` using these implementation rules:

- Import `startUserJourney`, `discoverWS`, `cancelTest`, `UserJourneyReport`.
- Import `usePollReport`, `useWSStore`, `buildDemoTheaterModel`, `DEMO_USER_JOURNEY_CONFIG`.
- On `开始演示`, call `startUserJourney(DEMO_USER_JOURNEY_CONFIG)`, then `discoverWS(id)`, then `connect(wsURL, id)`, then `pollReport.start(id, setReport, setError)`.
- On unmount, call `disconnect()`.
- Render these exact user-visible texts:
  - `AI 直播竞拍全链路验收`
  - `开始演示`
  - `参演角色`
  - `商家开播`
  - `买家出价`
  - `点天灯`
  - `本场挑战`
  - `当前最高价`
  - `验收结论`
  - `业务闭环成立`
  - `并发结果唯一`
  - `资产状态一致`
  - `查看技术报告` when `model.reportPath` exists

Use CSS-in-TS constants inside `Screen.tsx`, following the existing project style. Keep the file self-contained; do not introduce a component library.

- [ ] **Step 4: Run Screen tests**

Run:

```bash
cd frontend/test-dashboard
npm run test:run -- src/pages/Screen.test.tsx
```

Expected: PASS.

- [ ] **Step 5: Run frontend verification**

Run:

```bash
cd frontend/test-dashboard
npm run test:run
npm run build
```

Expected: both commands PASS.

- [ ] **Step 6: Commit Screen theater**

Run:

```bash
git add frontend/test-dashboard/src/pages/Screen.tsx frontend/test-dashboard/src/pages/Screen.test.tsx
git commit -m "feat(test-dashboard): turn screen into demo theater"
```

Expected: commit succeeds with only the listed files.

## Task 5: Integration Verification And SDD Evidence

**Files:**
- Modify only if executing under SDD: `docs/superpowers/sdd/<state-file>.md`

- [ ] **Step 1: Run backend verification**

Run:

```bash
cd backend/test
go test ./scenario/user_journey ./handler -run 'Test(UserJourney|RunHappyPathProducesEvidenceReport|RunEmitsDemoSnapshotInProgressMetrics)' -count=1
go test ./client/auction ./scenario/user_journey ./handler -count=1
```

Expected: both commands PASS.

- [ ] **Step 2: Run frontend verification**

Run:

```bash
cd frontend/test-dashboard
npm run test:run
npm run build
```

Expected: both commands PASS.

- [ ] **Step 3: Manual smoke test**

Run local services if needed:

```bash
./scripts/deploy-dev.sh
```

Open:

```text
http://localhost:5174/test/screen
```

Expected:

- Page title is `AI 直播竞拍全链路验收`.
- Clicking `开始演示` starts a `user_journey` task.
- Big screen shows `LIVE` during progress.
- `auction_bid` changes current price to `¥110.00`.
- `sky_lamp` shows `点天灯触发`.
- Completion shows three passed conclusions.
- `查看技术报告` opens `/test/report/<test_id>`.

- [ ] **Step 4: Record SDD evidence when applicable**

If executing under SDD, add this section to the state file:

```markdown
### Test Dashboard Demo Theater

- Scope: `/test/screen` 一键 UserJourney 演示剧场
- Backend verification:
  - `cd backend/test && go test ./scenario/user_journey ./handler -run 'Test(UserJourney|RunHappyPathProducesEvidenceReport|RunEmitsDemoSnapshotInProgressMetrics)' -count=1`
  - `cd backend/test && go test ./client/auction ./scenario/user_journey ./handler -count=1`
- Frontend verification:
  - `cd frontend/test-dashboard && npm run test:run`
  - `cd frontend/test-dashboard && npm run build`
- Manual smoke:
  - `/test/screen` 一键启动 UserJourney
  - 直播间主视觉、点天灯、订单、库存、报告下钻已验证
- Risks:
  - `demo_snapshot` 是演示快照，不作为业务事实 SSOT
  - 真实可信结论仍来自 `UserJourneyReport.all_ok` 和步骤断言
- Delivery conclusion: PASS
```

- [ ] **Step 5: Commit SDD state when changed**

Run only when a state file changed:

```bash
git add docs/superpowers/sdd/<state-file>.md
git commit -m "docs: record demo theater verification"
```

Expected: commit succeeds with the state file only.

## Self-Review

Spec coverage:

- `/test/screen` 复用并升级：Task 4。
- 一键 `UserJourney`：Task 4。
- 标准配置不暴露测试参数：Task 3、Task 4。
- 直播间主视觉、当前价、领先者、点天灯、订单、库存：Task 1、Task 3、Task 4。
- 技术证据下钻：Task 3、Task 4。
- 失败态显示业务阶段和报告入口：Task 3、Task 4。
- 不改变 `E2E`、`UserJourney` 控制台职责：Task 4 只修改 `Screen.tsx`。
- 测试策略：Task 1、Task 2、Task 3、Task 4、Task 5。

Placeholder scan:

- Plan contains no unresolved placeholder steps.
- New types and exported functions are defined before use.

Type consistency:

- Backend `DemoSnapshot` JSON fields match frontend `DemoSnapshot`.
- `demo_snapshot` is optional in both Go JSON and TypeScript report type.
- `buildDemoTheaterModel` consumes existing `ProgressMsg` and `UserJourneyReport`.
- `Screen.tsx` uses `DEMO_USER_JOURNEY_CONFIG` from `demoTheater.ts`, preventing duplicated standard config.

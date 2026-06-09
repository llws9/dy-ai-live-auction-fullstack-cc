# [C1] Chaos Theater Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a judge-facing Chaos Theater Mode to `frontend/test-dashboard` with one-click preset execution, terminal-style narration, Recharts anchors, and derived demo metrics.

**Architecture:** Keep backend/API contracts unchanged. Add pure exported helpers in `Chaos.tsx` (or a colocated utility if needed) for narration, start-disabled logic, curve anchors, and demo metrics. Reuse existing `startChaos`/`discoverWS`/`poll` flow; `start()` accepts an optional config so theater preset does not depend on async `setForm` timing. Enhance `ResilienceCurve` with optional `anchors`, `demoMetrics`, and `narration` props.

**Tech Stack:** React, TypeScript, Recharts, Vitest, test-dashboard existing CSS variables/inline style system.

---

### Task 1: Pure Theater Helpers

**Files:**
- Modify: `frontend/test-dashboard/src/pages/Chaos.tsx`
- Modify: `frontend/test-dashboard/src/pages/Chaos.test.ts`

- [x] Write failing tests for `buildNarration`, `buildCurveAnchors`, `buildDemoMetrics`, `isChaosStartDisabled`, and `describeChaosStartButton`.
- [x] Run `cd frontend/test-dashboard && npm test -- Chaos.test.ts` and confirm RED.
- [x] Implement exported pure helpers with no backend/API changes.
- [x] Run `cd frontend/test-dashboard && npm test -- Chaos.test.ts` and confirm GREEN.

### Task 2: One-Click Theater Execution

**Files:**
- Modify: `frontend/test-dashboard/src/pages/Chaos.tsx`
- Modify: `frontend/test-dashboard/src/pages/Chaos.test.ts`

- [x] Refactor `start` to accept optional `ChaosConfig` override.
- [x] Add `startTheaterMode()` using preset `{ fault_type: 'error_rate', probe_qps: 20, baseline_sec: 3, inject_sec: 8, recover_sec: 5, error_rate: 0.5, latency_ms: 0, jitter_ms: 0 }`.
- [x] Add `./start_theater.sh` button beside manual Start/Cancel, using lifecycle-aware disabled logic (`running || testID exists and not terminal`).
- [x] Add/adjust tests that prove a running test disables restart beyond the initial `running` flag.

### Task 3: Narration, Anchors, Metrics UI

**Files:**
- Modify: `frontend/test-dashboard/src/pages/Chaos.tsx`
- Modify: `frontend/test-dashboard/src/pages/Chaos.test.ts`

- [x] Import `ReferenceLine` from `recharts`.
- [x] Compute `narration`, `anchors`, and `demoMetrics` from `step/progress/form/displayedReport/liveBuckets`.
- [x] Render terminal-style narration (`> ...`) in/above `ResilienceCurve`.
- [x] Render `ReferenceLine` anchors for inject, SLA breach, and recover.
- [x] Add Metric cards for peak error rate, lost QPS, and recovery duration.
- [x] Run `cd frontend/test-dashboard && npm test -- Chaos.test.ts`.
- [x] Run `cd frontend/test-dashboard && npm run build`.

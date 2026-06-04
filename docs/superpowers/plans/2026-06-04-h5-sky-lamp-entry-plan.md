# H5 Sky Lamp Entry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 H5 直播间底部出价抽屉中接入点天灯入口，采用 A+C 方案：左侧点天灯按钮占 30%，右侧立即出价按钮占 70%，点天灯点击后需要二次确认再调用后端订阅接口。

**Architecture:** 前端继续通过 `frontend/h5/src/services/api.ts` 的 `/api/v1` 网关入口访问后端，不直连 auction-service。`LiveRoomSlide` 负责点天灯确认层、pending 状态、成功/失败提示；`Live.module.css` 负责按钮比例、天灯 icon 与确认层视觉。测试先覆盖 API 契约和直播间交互，再实现最小代码。

**Tech Stack:** React 18, TypeScript, CSS Modules, Jest, Testing Library, Vite.

---

## File Map

- Modify: `frontend/h5/src/services/api.ts`，新增 `skyLampApi.startSubscription(auctionId)`，请求 `POST /api/v1/sky-lamp/subscriptions`，body 为 `{ auction_id }`。
- Modify: `frontend/h5/src/services/__tests__/api.test.ts`，新增 API 契约测试。
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`，新增点天灯按钮、确认层、pending 状态、成功/失败处理。
- Modify: `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx`，新增 A+C 按钮布局与确认后调用接口的行为测试。
- Modify: `frontend/h5/src/pages/Live/Live.module.css`，新增 3:7 action bar、点天灯按钮、天灯 icon、确认层样式。

## Tasks

### Task 1: Sky Lamp API Contract

**Files:**
- Modify: `frontend/h5/src/services/api.ts`
- Modify: `frontend/h5/src/services/__tests__/api.test.ts`

- [ ] Step 1: Add failing test asserting `skyLampApi.startSubscription(5)` calls `POST /api/v1/sky-lamp/subscriptions` with body `{"auction_id":5}`.
- [ ] Step 2: Run `npm test -- --runTestsByPath src/services/__tests__/api.test.ts --runInBand` from `frontend/h5`; expected failure because `skyLampApi` is not exported.
- [ ] Step 3: Export `skyLampApi` with `startSubscription` using existing `post` helper.
- [ ] Step 4: Run the same test; expected pass.

### Task 2: Live Room A+C Interaction

**Files:**
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`
- Modify: `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx`
- Modify: `frontend/h5/src/pages/Live/Live.module.css`

- [ ] Step 1: Add failing test that opens bid sheet, sees `点天灯` and `立即出价`, clicks `点天灯`, sees `确认开启点天灯？`, clicks `确认开启`, and asserts `skyLampApi.startSubscription(5)` was called.
- [ ] Step 2: Run `npm test -- --runTestsByPath src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand`; expected failure because UI/API wiring is absent.
- [ ] Step 3: Add `skyLampApi` import, `skyLampConfirmOpen` and `skyLampPending` state, `handleStartSkyLamp` callback, A+C action bar markup, and confirmation panel.
- [ ] Step 4: Add CSS for 3:7 layout, lamp button, CSS-drawn sky lantern icon at `left:-10px; top:-8px`, and confirmation layer.
- [ ] Step 5: Run the same test; expected pass.

### Task 3: Verification

**Files:**
- Update: `docs/superpowers/sdd/runs/2026-06-04-h5-sky-lamp-entry-state.md`

- [ ] Step 1: Run `npm test -- --runTestsByPath src/services/__tests__/api.test.ts src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand` from `frontend/h5`; expected pass.
- [ ] Step 2: Run `npm run build` from `frontend/h5`; expected pass.
- [ ] Step 3: Update SDD state with modified files, test commands, results, and residual risks.

### Task 4: Sky Lamp Success State

**Files:**
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`
- Modify: `frontend/h5/src/pages/Live/BidDock.tsx`
- Modify: `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx`
- Modify: `frontend/h5/src/pages/Live/Live.module.css`

- [x] Step 1: Add failing test for success state after sky lamp confirmation.

Run: `npm test -- --runTestsByPath src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand`
Expected: FAIL because confirmed sky lamp does not close drawer or mark dock/image/button as active.
Actual RED: FAIL with location still `?sheet=bid` after confirmation.

- [x] Step 2: Implement success state.

Implementation:
- Add `skyLampActive` and `skyLampNoticeVisible` state in `LiveRoomSlide`.
- On successful `skyLampApi.startSubscription`, mark sky lamp active, show notice, close drawer, and keep ordinary bid flow otherwise unchanged.
- Pass `skyLampActive` to `BidDock`.
- Add dock glow and product image sky-lamp badge in `BidDock`.
- Add CSS animations for floating lamp, dock aura, and livestream notice.

- [x] Step 3: Verify success state.

Run: `npm test -- --runTestsByPath src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand`
Expected: PASS.
Actual: PASS, 10 tests passed.

- [x] Step 4: Final verification.

Run: `npm test -- --runTestsByPath src/services/__tests__/api.test.ts src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand`
Actual: PASS, 17 tests passed.

Run: `npm run build`
Actual: PASS, `tsc && vite build` completed.

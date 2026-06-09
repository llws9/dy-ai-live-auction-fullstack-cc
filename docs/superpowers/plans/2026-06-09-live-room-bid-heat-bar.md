# [A1] Live Room Bid Heat Bar Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the static live-room heat marquee with a cyber-glow `BidHeatBar` driven by recent bid frequency.

**Architecture:** `useBidHeat` owns the 10s sliding window and exposes `level`, `markBid`, and `reset`; `BidHeatBar` is a pure visual component using existing theme tokens; `LiveRoomSlide` feeds the hook from `bid_placed`, `sky_lamp_auto_bid`, and successful local `handleBid`, then replaces the old heat marquee.

**Tech Stack:** React, TypeScript, CSS Modules, Jest/Vitest-style frontend tests.

---

### Task 1: Bid Heat Hook

**Files:**
- Create: `frontend/h5/src/hooks/useBidHeat.ts`
- Create: `frontend/h5/src/hooks/__tests__/useBidHeat.test.ts`

- [ ] Write failing tests for initial calm state, warming at 2 bids, blazing at 5 bids, decay after 10s, and reset.
- [ ] Run `cd frontend/h5 && npm test -- useBidHeat.test.ts` and confirm failure.
- [ ] Implement `useBidHeat` with timestamp array, 10s pruning, 1s interval, and cleanup.
- [ ] Run `cd frontend/h5 && npm test -- useBidHeat.test.ts` and confirm pass.

### Task 2: Bid Heat Bar Component

**Files:**
- Create: `frontend/h5/src/components/LiveRoom/BidHeatBar.tsx`
- Create: `frontend/h5/src/components/LiveRoom/BidHeatBar.module.css`
- Create: `frontend/h5/src/components/LiveRoom/__tests__/BidHeatBar.test.tsx`

- [ ] Write failing render tests for `calm`, `warming`, and `blazing` labels and stats.
- [ ] Run `cd frontend/h5 && npm test -- BidHeatBar.test.tsx` and confirm failure.
- [ ] Implement component and cyber-glow CSS using existing tokens and `prefers-reduced-motion`.
- [ ] Run `cd frontend/h5 && npm test -- BidHeatBar.test.tsx` and confirm pass.

### Task 3: LiveRoomSlide Integration

**Files:**
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`
- Read: `frontend/h5/src/pages/Live/Live.module.css`

- [ ] Import `useBidHeat` and `BidHeatBar`.
- [ ] Call `markBid()` in `bid_placed`, `sky_lamp_auto_bid`, and successful local `handleBid`.
- [ ] Reset heat on `auctionId` change.
- [ ] Replace old static `heatMarqueeText` with `<BidHeatBar level={heatLevel} bidderCount={ranking.length} viewerCount={viewerCount} />`.
- [ ] Run `cd frontend/h5 && npm test -- useBidHeat.test.ts BidHeatBar.test.tsx`.
- [ ] Run `cd frontend/h5 && npm run build`.


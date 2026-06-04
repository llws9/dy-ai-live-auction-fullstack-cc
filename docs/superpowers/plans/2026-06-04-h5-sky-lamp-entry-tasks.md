# H5 Sky Lamp Entry Tasks

## T001 - API Contract
- Status: pending
- Scope: `frontend/h5/src/services/api.ts`, `frontend/h5/src/services/__tests__/api.test.ts`
- Depends On: none
- Expected Tests: `npm test -- --runTestsByPath src/services/__tests__/api.test.ts --runInBand`
- Acceptance: `skyLampApi.startSubscription(auctionId)` posts to `/api/v1/sky-lamp/subscriptions` with `{ auction_id: auctionId }`.

## T002 - Live Room UI and Interaction
- Status: pending
- Scope: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`, `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx`, `frontend/h5/src/pages/Live/Live.module.css`
- Depends On: T001
- Expected Tests: `npm test -- --runTestsByPath src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand`
- Acceptance: bid drawer shows 3:7 `点天灯` and `立即出价` action bar; `点天灯` has a sky lantern icon at the selected left-down position; clicking it opens confirmation; confirming calls `skyLampApi.startSubscription` and shows success/failure feedback.

## T003 - Final Verification
- Status: pending
- Scope: state file and verification only
- Depends On: T001,T002
- Expected Tests: `npm test -- --runTestsByPath src/services/__tests__/api.test.ts src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand`, `npm run build`
- Acceptance: targeted tests and H5 build pass; state file records evidence.

## T004 - Sky Lamp Success State
- Status: done
- Scope: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`, `frontend/h5/src/pages/Live/BidDock.tsx`, `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx`, `frontend/h5/src/pages/Live/Live.module.css`
- Depends On: T002
- Expected Tests: `npm test -- --runTestsByPath src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand`, `npm test -- --runTestsByPath src/services/__tests__/api.test.ts src/pages/Live/__tests__/LiveRoomSlide.test.tsx --runInBand`, `npm run build`
- Acceptance: after successful confirmation, the sheet closes, the sky lamp button is locked as `守护中`, a floating lamp icon is shown, the dock gets a glow, the product image gets a sky-lamp badge, and the livestream notice appears.

# H5 Profile Center Redesign

## Context

Current H5 profile center uses a flat stack of user header, statistics, wallet, orders, menu entries, and logout. The user identified two primary problems:

- Information hierarchy is weak: important auction/payment actions compete with low-frequency account entries.
- Space utilization is low: standalone statistic cards and inactive/low-value entries occupy prominent space.

The selected direction is **方案 A：交易指挥台** from the UI design trio iteration.

## Theme Detection

- UI suites: `dark` and `light`.
- Switch mechanism: `html[data-theme="dark|light"]`.
- Persistence key: `localStorage['h5.theme']`.
- Default theme: `dark`.
- Token source: `frontend/h5/src/styles/tokens/colors.css`.
- Runtime theme source: `frontend/h5/src/store/themeContext.tsx`.

The redesign must keep using existing semantic tokens such as:

- `--page-gradient-profile`
- `--bg-surface`
- `--surface-glass`
- `--item-subtle-bg`
- `--text-primary`
- `--text-secondary`
- `--text-brand`
- `--card-border-accent`
- `--icon-tile-bg`
- `--danger-*`

## Selected Design

### 1. Bare User Header

The user identity area remains visually lightweight and must not be wrapped in a card.

Structure:

- Avatar on the left.
- `My Account` eyebrow, display name, role and ID chips on the right.
- Header sits directly on the profile page gradient.

Reason:

- The user header is identity context, not a primary action.
- Removing the card lowers visual noise and keeps the top area breathable.

### 2. Auction Command Card

The main card becomes the primary task module.

Content:

- Section title: `我的竞拍`.
- Helper text: `记录含中标`.
- Primary CTA: `2 件中标待支付`.
- CTA subtitle: `从竞拍记录查看全部中标与出价`.
- CTA target: `/history`.
- Metrics row:
  - `竞拍记录`
  - `中标`
  - `收藏`

Important interaction change:

- Remove the standalone `中标` entry/button that previously linked to `/history?filter=won`.
- `竞拍记录` becomes the single entry point for auction history, including won records.
- The pending-payment count badge should be placed at the top-right of the `竞拍记录` metric card, not inline with text.

Reason:

- The core user action is "check auction/won status and pay".
- A single auction-history entry avoids duplicate navigation and keeps text centered.

### 3. Footprints Module

Add a recent live-room browsing module below the auction command card.

Data contract:

- Storage: `localStorage`.
- Key: implementation may define a namespaced H5 key, e.g. `h5.liveRoomFootprints`.
- Write timing: record when entering a live room.
- Fields:
  - `live_stream_id`
  - `name`
  - `cover`
  - `enteredAt`
- Deduplication: same live room updates timestamp and moves to the top.
- Capacity: keep the latest 10 records.

Display:

- Horizontal list of recent live rooms.
- Each item shows cover, room name, and relative time.
- Empty state should be compact and must not dominate the page.

Reason:

- It supports the second priority user action: return to recently browsed live rooms.
- Local-only storage matches the current scope and avoids backend expansion.

### 4. Account And Service Grid

Replace the old menu-like area with a compact 2x2 service grid.

Entries:

- Wallet: `钱包` / `可用 ¥0`
- Address: `收货地址` / `管理配送`
- Personal seller application: `个人卖家申请` / `暂未开放`
- Enterprise merchant onboarding: `企业商家入驻` / `暂未开放`

Layout:

- Each service card uses horizontal icon + text.
- Icon on the left, title and description on the right.
- `个人卖家申请` and `企业商家入驻` may show a small `新` badge.
- The `新` badge must sit outside the text flow at the top-right of the service card, slightly offset outward, so it does not cover the title.

Reason:

- Horizontal icon + text makes these entries read like compact actionable functions.
- It reduces vertical height compared with vertical icon stacks.

### 5. Recent Orders

Recent orders are no longer the main visual center.

Recommended handling:

- Keep access to all orders via the auction/order area where needed.
- If recent order summaries remain, they should be visually secondary to the auction command card.

Reason:

- The user's primary task is auction/won/payment, not generic order browsing.

### 6. Logout

Logout remains a full-width danger-styled button near the bottom.

Reason:

- It is important but low-frequency and destructive, so it should not compete with primary actions.

## Visual Rules

- Use existing H5 semantic tokens; do not introduce hard-coded theme colors in production CSS.
- Support both `dark` and `light` themes.
- Prefer `transform` and `opacity` for any micro-interactions.
- Respect `prefers-reduced-motion`.
- Avoid adding card chrome around the top identity header.
- Keep numeric badges outside text layout to avoid text misalignment.

## Scope

In scope:

- Refactor `frontend/h5/src/pages/User/Index.tsx` structure.
- Refactor `frontend/h5/src/pages/User/Profile.module.css`.
- Add frontend-only live-room footprint storage utilities.
- Hook footprint recording into live-room entry.
- Add or update focused tests for localStorage footprint behavior and profile render structure.

Out of scope:

- Backend footprint persistence.
- Actual seller application backend integration.
- Actual enterprise onboarding backend integration.
- New payment or order APIs.

## Acceptance Criteria

- Profile header is rendered without a surrounding card.
- Standalone `中标` stat/button is removed from the top statistics grid.
- `竞拍记录` remains the single auction-history entry and can represent won/pending-payment information.
- Pending-payment badge is positioned at the top-right of the metric card and does not shift text alignment.
- Four service entries use horizontal icon + text layout.
- Seller/onboarding `新` badges do not overlap titles.
- Footprints keep the latest 10 live-room records in localStorage.
- Re-entering the same live room updates its timestamp and moves it to the top.
- Page remains visually correct in both `dark` and `light` themes.

## Risks

- LocalStorage footprints are device-local and do not sync across browsers or devices.
- If live-room data lacks a stable cover field, the footprint item needs a graceful visual fallback.
- Too many red badges can dilute the importance of the pending-payment badge; only core reminders should use numeric red badges.

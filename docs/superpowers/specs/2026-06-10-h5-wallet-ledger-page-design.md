# H5 Wallet Ledger Page Design

## Decision

Selected design: **B · 流水账本**.

## Context

The profile center now has a dedicated wallet service entry. The next step is to create a standalone wallet page instead of keeping wallet information embedded in the profile page.

The selected wallet page should emphasize explainability and traceability:

- What is the available balance?
- Why is money frozen?
- Which order or auction caused each balance change?
- Which records are income, payment, freeze, or release?

## Theme Detection

- UI suites: `dark` and `light`.
- Switch mechanism: `html[data-theme="dark|light"]`.
- Persistence key: `localStorage['h5.theme']`.
- Default theme: `dark`.
- Token source: `frontend/h5/src/styles/tokens/colors.css`.
- Theme context: `frontend/h5/src/store/themeContext.tsx`.

The implementation must use existing semantic tokens:

- `--page-gradient-profile`
- `--bg-surface`
- `--surface-glass`
- `--item-subtle-bg`
- `--text-primary`
- `--text-secondary`
- `--text-brand`
- `--card-border-accent`
- `--icon-tile-bg`
- `--danger-text`
- `--color-success-500`

## Selected UI

### Page Structure

Top navigation:

- Back button.
- Title: `钱包`.
- Right action: `筛选`.

Balance card:

- Label: `Available`.
- Main amount: available balance, for example `¥12,288`.
- Helper text: `钱包余额 · 记录所有竞拍资金流`.

Ledger section:

- Title: `最近流水`.
- Filter pill: `全部`.
- Timeline-style list.
- Each ledger row includes:
  - Transaction title.
  - Related auction/order context.
  - Time.
  - Signed amount.

Funds status section:

- Available balance.
- Frozen amount.

### Transaction Types

Initial UI should support at least these display types:

- `订单支付`: negative amount.
- `竞拍冻结`: negative or frozen amount.
- `冻结释放`: positive amount.
- `充值入账`: positive amount, if future recharge records exist.

Positive records should use success color. Negative/frozen records should use danger emphasis.

## Navigation Decision

Profile wallet entry should navigate to the wallet page once implemented.

Current temporary behavior may keep pointing to `/orders` only until the wallet route exists. After wallet page implementation:

- Profile service entry `钱包` target: `/wallet`.
- The top pending-payment CTA in profile remains `/orders`, because it represents order payment, not generic wallet browsing.

## Scope

In scope for implementation:

- Add H5 wallet page route, e.g. `/wallet`.
- Render balance overview from existing `userApi.getBalance()`.
- Render a compact ledger list using available data or a frontend-derived demo model if backend ledger API does not exist yet.
- Update profile wallet service entry to `/wallet`.
- Preserve dark/light theme support.
- Add focused render tests.

Out of scope:

- Real recharge.
- Real withdrawal.
- Backend wallet ledger API design, unless separately requested.
- Payment flow changes.

## Acceptance Criteria

- Wallet page exists and is reachable from profile wallet entry.
- Page uses the selected B ledger-first layout.
- Available balance and frozen amount are displayed clearly.
- Ledger rows visually distinguish income/release from payment/freeze.
- Empty ledger state remains compact and clearly says no transaction records are available.
- Page supports both `dark` and `light` themes through existing tokens.
- Profile top pending-payment CTA continues to navigate to `/orders`.

## Risks

- If no backend ledger API exists, ledger rows are demo/derived UI only and must not imply audited financial records.
- Too much payment action on the wallet page would duplicate the order page; keep wallet focused on balance explanation and transaction traceability.

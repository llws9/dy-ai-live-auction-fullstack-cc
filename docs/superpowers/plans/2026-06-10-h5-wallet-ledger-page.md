# H5 Wallet Ledger Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `/wallet` H5 page using the selected B · 流水账本 design and connect the profile wallet entry to it.

**Architecture:** Implement the wallet page as a focused route component under `frontend/h5/src/pages/Wallet/`. The page reads existing `userApi.getBalance()` for available/frozen funds, then renders a compact frontend-derived ledger model until a backend ledger API exists. Profile remains the entry surface; pending-payment CTA stays `/orders`, while wallet service entry becomes `/wallet`.

**Tech Stack:** React 18, TypeScript, React Router, CSS Modules, Jest, Testing Library.

---

## File Map

- Create: `frontend/h5/src/pages/Wallet/Index.tsx`
- Create: `frontend/h5/src/pages/Wallet/Wallet.module.css`
- Create: `frontend/h5/src/pages/Wallet/__tests__/Wallet.test.tsx`
- Modify: `frontend/h5/src/App.tsx`
- Modify: `frontend/h5/src/pages/User/Index.tsx`
- Modify: `frontend/h5/src/pages/User/__tests__/Profile.test.tsx`

---

## Task 1: Wallet Page Component

**Files:**
- Create: `frontend/h5/src/pages/Wallet/Index.tsx`
- Create: `frontend/h5/src/pages/Wallet/Wallet.module.css`
- Create: `frontend/h5/src/pages/Wallet/__tests__/Wallet.test.tsx`

- [ ] **Step 1: Write failing render tests**

Create `frontend/h5/src/pages/Wallet/__tests__/Wallet.test.tsx` with tests that mock `userApi.getBalance()`, render the page in `MemoryRouter`, and assert:

- page title `钱包`
- available amount `¥12,288`
- frozen amount `¥600`
- ledger section `最近流水`
- ledger rows `订单支付`, `竞拍冻结`, `冻结释放`
- compact error/retry behavior when balance loading fails

- [ ] **Step 2: Run test and confirm failure**

Run:

```bash
cd frontend/h5 && npm test -- src/pages/Wallet/__tests__/Wallet.test.tsx --runInBand
```

Expected: FAIL because `../Index` does not exist.

- [ ] **Step 3: Implement Wallet page**

Create `Index.tsx` with:

- `userApi.getBalance()` data load.
- local `formatCurrency()`.
- derived ledger rows from available and frozen balance.
- back button using `useNavigate()`.
- selected B ledger-first layout.

- [ ] **Step 4: Implement CSS module**

Create `Wallet.module.css` using existing semantic tokens only:

- `--page-gradient-profile`
- `--bg-surface`
- `--item-subtle-bg`
- `--text-primary`
- `--text-secondary`
- `--text-brand`
- `--card-border-accent`
- `--icon-tile-bg`
- `--danger-text`
- `--color-success-500`

- [ ] **Step 5: Verify wallet test passes**

Run:

```bash
cd frontend/h5 && npm test -- src/pages/Wallet/__tests__/Wallet.test.tsx --runInBand
```

Expected: PASS.

---

## Task 2: Route And Profile Entry

**Files:**
- Modify: `frontend/h5/src/App.tsx`
- Modify: `frontend/h5/src/pages/User/Index.tsx`
- Modify: `frontend/h5/src/pages/User/__tests__/Profile.test.tsx`

- [ ] **Step 1: Update profile test**

Change wallet entry assertion from `/orders` to `/wallet`.

- [ ] **Step 2: Run profile test and confirm failure**

Run:

```bash
cd frontend/h5 && npm test -- src/pages/User/__tests__/Profile.test.tsx --runInBand
```

Expected: FAIL because profile still links wallet to `/orders`.

- [ ] **Step 3: Wire route and profile link**

Modify `App.tsx`:

- add `const Wallet = lazy(() => import('./pages/Wallet'))`
- add protected route `/wallet`

Modify `User/Index.tsx`:

- wallet service entry link becomes `/wallet`
- top pending-payment CTA remains `/orders`

- [ ] **Step 4: Verify profile test passes**

Run:

```bash
cd frontend/h5 && npm test -- src/pages/User/__tests__/Profile.test.tsx --runInBand
```

Expected: PASS.

---

## Task 3: Final Verification

**Files:**
- No additional files.

- [ ] **Step 1: Focused tests**

Run:

```bash
cd frontend/h5 && npm test -- src/pages/Wallet/__tests__/Wallet.test.tsx src/pages/User/__tests__/Profile.test.tsx --runInBand
```

Expected: PASS.

- [ ] **Step 2: Build**

Run:

```bash
cd frontend/h5 && npm run build
```

Expected: PASS.

- [ ] **Step 3: Diagnostics**

Check diagnostics for the modified TSX/CSS/test files.

Expected: no newly introduced diagnostics.

- [ ] **Step 4: Local H5 deploy**

Run:

```bash
INTERNAL_API_TOKEN=dev docker compose up -d --build frontend-h5
curl -s -o /dev/null -w 'h5 / %{http_code}\n' http://localhost:3000/
curl -s -o /dev/null -w 'h5 /@vite/client %{http_code}\n' http://localhost:3000/@vite/client
```

Expected: H5 returns `200`, stale Vite client returns `404`.

---

## Self-Review

- Spec coverage: `/wallet` route, ledger-first layout, available/frozen balance, derived ledger, profile wallet entry, and pending-payment CTA are covered.
- Placeholder scan: no `TBD`, `TODO`, or unspecified future steps.
- Type consistency: route path `/wallet`, balance fields, and profile link target are consistent.

# H5 Theme Phase 2 Visual Polish Design

- **Author:** brainstorming session
- **Date:** 2026-06-03
- **Scope:** `frontend/h5`
- **Status:** Draft - pending user review
- **Related Phase 1 spec:** `docs/superpowers/specs/2026-05-30-h5-theme-toggle-design.md`
- **Visual preview:** `.superpowers/brainstorm/8307-1780419209/content/profile-theme-preview.html`

---

## 1. Background

Phase 1 delivered the theme switching MVP:

- `<html data-theme="dark" | "light">` controls theme state.
- `ThemeProvider` persists the selected theme in `localStorage['h5.theme']`.
- `ThemeToggle` is available in standard page headers.
- Core page/surface/text tokens exist for dark and light modes.
- Several pages and shared cards/toasts read from semantic tokens.

The MVP works functionally, but the light theme is not visually complete. The most visible issue is the `/profile` page: cards turn light while the page hero background, avatar frame, chips, order item surfaces, shadows, and decorative layers still use dark-mode values. This creates a mixed theme where the upper screen remains night-styled and the lower screen becomes white-card styled.

Phase 2 upgrades the feature from "theme can switch" to "core H5 pages render as coherent dark/light experiences."

---

## 2. Goals

1. Make the H5 light theme visually complete on core user-facing pages.
2. Preserve the current luxury dark style as the default night mode.
3. Extend semantic tokens for decorative but theme-sensitive UI layers.
4. Fix `/profile` as the visual benchmark for theme quality.
5. Cover newly added H5 pages that were not part of the Phase 1 page list.
6. Keep the implementation incremental and avoid a full cross-product design-system rewrite.

---

## 3. Non-Goals

- Do not modify `frontend/admin` or `frontend/test-dashboard`.
- Do not redesign page structure or navigation.
- Do not rebuild the whole CSS token system from scratch.
- Do not convert every hardcoded brand/status color if it is intentionally decorative or functional.
- Do not add a full settings page in this phase unless explicitly approved later.
- Do not block delivery on full e2e visual snapshot infrastructure.

---

## 4. Phase 1 Follow-Ups Now In Scope

The following items were explicitly left out of Phase 1 and are now part of Phase 2:

| Follow-up | Phase 2 Decision |
| --- | --- |
| Incrementally tokenize remaining hardcoded colors | In scope for core H5 pages and shared components only |
| Decorative gradients, shadows, and single-point brand surfaces | In scope when they affect light/dark visual coherence |
| `/profile` settings entry as a future theme anchor | Keep as future work; Phase 2 may prepare components/tokens but does not build settings |
| e2e/visual snapshot repair | Keep out of scope; add lightweight manual visual checklist |
| `prefers-color-scheme` third mode | Keep out of scope; revisit with settings page |
| Admin/test-dashboard theme support | Keep out of scope |

---

## 5. Current Problem Analysis

### 5.1 Profile Page

`frontend/h5/src/pages/User/Profile.module.css` still contains theme-sensitive dark-only values:

- Page background uses fixed dark radial/linear gradients.
- Avatar frame uses dark translucent background and dark shadow.
- User badges use a dark translucent chip background.
- Card border colors use fixed gold alpha values.
- Order item background uses dark translucent black.
- Danger/error colors are hardcoded.

This is why the light theme looks incomplete: only the surface cards read from `--bg-surface`, while the page atmosphere remains dark.

### 5.2 Pages Added Or Missed After Phase 1

The original Phase 1 plan covered 9 page CSS files. The current H5 codebase also includes pages that should join the Phase 2 theme pass:

- `frontend/h5/src/pages/Addresses/Addresses.module.css`
- `frontend/h5/src/pages/Order/Detail.module.css`

These pages must follow the same theme quality bar as Profile, History, Notifications, and Follow.

### 5.3 Shared Components

Several shared or reusable components still contain hardcoded colors or old token usage. Phase 2 should prioritize components that visibly appear in core H5 flows:

- `Button`
- `Input`
- `Loading`
- `Skeleton`
- `BadgeDot`
- `LiveReminderModal`
- notification-related components

---

## 6. Proposed Architecture

Phase 2 should extend the existing architecture rather than replacing it.

```text
ThemeProvider
  -> writes <html data-theme>
  -> CSS resolves global semantic tokens
  -> page-specific theme tokens derive page atmosphere
  -> shared components consume stable semantic tokens
```

The key design rule is:

> Components should not decide whether the app is dark or light. Components should consume semantic tokens that already encode the selected theme.

---

## 7. Token Model

### 7.1 Keep Existing Core Tokens

The existing tokens remain the foundation:

```css
--bg-page
--bg-surface
--bg-elevated
--text-primary
--text-secondary
--text-brand
--border-subtle
--shadow-key
```

### 7.2 Add Atmosphere Tokens

These tokens cover visual layers that are not plain surfaces:

```css
--page-gradient-profile
--surface-glass
--surface-muted
--chip-bg
--chip-border
--avatar-bg
--avatar-border
--avatar-shadow
--icon-tile-bg
--card-border-accent
--item-subtle-bg
--danger-bg
--danger-border
--danger-text
```

### 7.3 Token Intent

| Token | Purpose |
| --- | --- |
| `--page-gradient-profile` | Profile hero/page atmosphere, different in dark and light |
| `--surface-glass` | translucent elevated surfaces such as bottom nav and buttons |
| `--surface-muted` | secondary backgrounds inside cards/lists |
| `--chip-bg` / `--chip-border` | role/id badges and compact metadata pills |
| `--avatar-*` | avatar frame styling independent from generic cards |
| `--icon-tile-bg` | menu icons such as A/F/N/D/S tiles |
| `--card-border-accent` | low-emphasis luxury gold card border |
| `--item-subtle-bg` | nested order/list item background |
| `--danger-*` | logout/error states with theme-aware contrast |

These names are intentionally semantic. They describe UI role, not color values.

---

## 8. Visual Direction

### 8.1 Light Theme

The light theme should be warm, luxury, and low-glare rather than pure white:

- Page background: ivory/warm champagne gradient.
- Cards: soft white or translucent warm surface.
- Gold accents: darker brown-gold for contrast.
- Shadows: soft warm shadows, not black-heavy shadows.
- Chips and icon tiles: subtle warm fills instead of dark translucent blocks.
- Bottom nav: translucent white with warm border.

### 8.2 Dark Theme

The dark theme should preserve the existing luxury feel:

- Page background: deep charcoal with subtle gold radial light.
- Cards: dark elevated translucent surfaces.
- Gold accents: current warm gold.
- Shadows: stronger depth is acceptable.
- Chips and icon tiles: dark translucent fills remain valid.

### 8.3 Profile As Benchmark

The `/profile` page defines the Phase 2 quality bar because it combines:

- hero identity area
- stats cards
- wallet card
- order summary
- menu list
- bottom navigation
- destructive action

If Profile looks coherent in both themes, the token set is likely sufficient for other standard H5 pages.

---

## 9. Page Scope

### 9.1 Must Polish

These pages are in scope for Phase 2:

- `/profile` via `frontend/h5/src/pages/User/Profile.module.css`
- `/addresses` via `frontend/h5/src/pages/Addresses/Addresses.module.css`
- `/order/detail` via `frontend/h5/src/pages/Order/Detail.module.css`
- `/notifications` via `frontend/h5/src/pages/Notifications/Notifications.module.css`
- `/following` via `frontend/h5/src/pages/Follow/Following.module.css`
- `/history` via `frontend/h5/src/pages/History/AuctionHistory.module.css`

### 9.2 Smoke Check Only

These pages already participated in Phase 1 and should be smoke checked for regressions:

- `/`
- `/detail`
- `/result`
- `/live`

### 9.3 Explicitly Excluded

- `/login` remains visually independent and does not expose theme toggle.
- `frontend/admin`
- `frontend/test-dashboard`

---

## 10. Component Scope

### 10.1 Must Polish

Shared components in scope:

- `frontend/h5/src/components/shared/Button.module.css`
- `frontend/h5/src/components/shared/Input.module.css`
- `frontend/h5/src/components/shared/Loading.module.css`
- `frontend/h5/src/components/shared/Skeleton.module.css`
- `frontend/h5/src/components/BadgeDot/BadgeDot.module.css`
- `frontend/h5/src/components/LiveReminderModal/LiveReminderModal.module.css`

### 10.2 Optional If Touched By Core Flows

- `frontend/h5/src/components/Notification/Notification.css`
- `frontend/h5/src/components/Toast/Toast.module.css`
- inline-styled legacy components such as `UserInfo`, `UserStats`, `RankingList`, `BidInput`

Optional items should only be touched if they are visible in the Phase 2 page checklist or are trivial to tokenize safely.

---

## 11. Migration Rules

1. Replace theme-sensitive hardcoded colors with semantic tokens.
2. Keep functional colors when they represent status semantics, unless contrast fails.
3. Keep brand gradients only if they look correct in both themes.
4. Do not introduce per-component theme branching in React.
5. Prefer CSS variables over duplicated `[data-theme]` selectors inside component CSS.
6. Use page-specific tokens only for page atmosphere, not generic text/surface needs.
7. Avoid weakening dark mode while improving light mode.

---

## 12. UX And Accessibility

- Preserve the current top/header `ThemeToggle` entry.
- Do not add another floating theme button.
- Maintain touch targets of at least 44px for primary interactive controls.
- Ensure key text/background pairs meet WCAG AA contrast where practical.
- Verify destructive actions such as logout remain visually clear in both themes.
- Avoid high-glare pure white full-screen surfaces in light mode.

---

## 13. Testing Strategy

### 13.1 Automated Checks

- Run `npx tsc --noEmit`.
- Run theme-relevant unit tests:
  - `themeContext`
  - `ThemeToggle`
  - `PageHeader`
  - touched page/component tests

### 13.2 Manual Visual Checklist

Verify both dark and light modes on:

| Route | Checks |
| --- | --- |
| `/profile` | hero, avatar, badges, stats, wallet, orders, menu, bottom nav |
| `/addresses` | page background, form/list surfaces, text, empty states |
| `/order/detail` | detail cards, status labels, price text, action areas |
| `/notifications` | list surfaces, empty state, page header |
| `/following` | follow cards/list rows |
| `/history` | order/auction history rows |
| `/` | no regression from Phase 1 |
| `/live` | overlay remains readable and does not inherit unsuitable light surfaces |

### 13.3 Visual Preview

Use the local preview file as the reference for Profile's intended direction:

```text
.superpowers/brainstorm/8307-1780419209/content/profile-theme-preview.html
```

The implementation does not need pixel-perfect parity with the preview, but should preserve the same design intent.

---

## 14. Acceptance Criteria

1. `/profile` no longer mixes dark-only page background with light cards.
2. `/profile` light mode looks warm, coherent, and comparable to the approved preview direction.
3. Dark mode remains close to the current luxury night style.
4. Addresses and order detail pages are included in the H5 theme pass.
5. Shared components visible in scoped pages consume semantic tokens.
6. Theme switching still persists across refreshes.
7. `/login` remains excluded from theme toggle exposure.
8. Type checking passes.
9. Theme-relevant tests pass, with any unrelated pre-existing test failures documented separately.

---

## 15. Risks

| Risk | Mitigation |
| --- | --- |
| Token names become too page-specific | Keep page-specific tokens only for page atmosphere; generic components use generic semantic tokens |
| Light mode becomes too plain | Use warm gradients, translucent surfaces, and subtle gold borders |
| Dark mode regresses | Compare dark mode before/after on Profile and core pages |
| Scope expands into full design-system rewrite | Limit required work to listed H5 pages and visible shared components |
| Tests fail for unrelated legacy reasons | Run scoped tests plus typecheck; document unrelated failures instead of hiding them |

---

## 16. Future Work

These items remain outside Phase 2:

- Full settings page from `/profile` with theme radio cards.
- Third mode: "follow system" using `prefers-color-scheme`.
- Visual regression automation with screenshot diff.
- Cross-app theme support for `frontend/admin` and `frontend/test-dashboard`.
- Full hardcoded color elimination across all legacy components.

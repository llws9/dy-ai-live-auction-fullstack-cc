# H5 Theme Phase 2 Visual Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade the H5 theme feature from functional dark/light switching to a coherent visual experience across core user-facing pages.

**Architecture:** Keep the existing `ThemeProvider` and `<html data-theme>` architecture. Extend semantic CSS tokens in `colors.css`, then migrate Profile, newly missed pages, scoped standard pages, and visible shared components to consume those tokens. Add lightweight CSS contract tests that read source CSS files and guard against dark-only values returning to theme-sensitive surfaces.

**Tech Stack:** React 18 + TypeScript + Vite + CSS Modules + Jest + ts-jest + Node `fs` contract tests.

---

## File Structure

| File | Type | Responsibility |
| --- | --- | --- |
| `frontend/h5/src/styles/tokens/colors.css` | Modify | Add Phase 2 atmosphere tokens for dark/light themes |
| `frontend/h5/src/styles/tokens/__tests__/themeTokens.test.ts` | Create | Assert new theme tokens exist in both dark/default and light blocks |
| `frontend/h5/src/pages/User/Profile.module.css` | Modify | Use Phase 2 tokens for Profile page atmosphere and benchmark polish |
| `frontend/h5/src/pages/User/__tests__/ProfileThemeTokens.test.ts` | Create | Assert Profile no longer uses dark-only values for theme-sensitive blocks |
| `frontend/h5/src/pages/Addresses/Addresses.module.css` | Modify | Tokenize missed Addresses page surfaces and actions |
| `frontend/h5/src/pages/Order/Detail.module.css` | Modify | Tokenize missed Order detail surfaces, buttons, badges, toast |
| `frontend/h5/src/pages/__tests__/Phase2PageThemeTokens.test.ts` | Create | Assert scoped page CSS files use Phase 2 semantic tokens and avoid old surface tokens |
| `frontend/h5/src/pages/Notifications/Notifications.module.css` | Modify | Polish scoped standard page with Phase 2 tokens where needed |
| `frontend/h5/src/pages/Follow/Following.module.css` | Modify | Polish scoped standard page with Phase 2 tokens where needed |
| `frontend/h5/src/pages/History/AuctionHistory.module.css` | Modify | Polish scoped standard page with Phase 2 tokens where needed |
| `frontend/h5/src/components/shared/Button.module.css` | Modify | Make outline/ghost/default hover states theme-aware |
| `frontend/h5/src/components/shared/Input.module.css` | Modify | Replace old light-only tokens with semantic theme tokens |
| `frontend/h5/src/components/shared/Loading.module.css` | Modify | Make fullscreen overlay and spinner track theme |
| `frontend/h5/src/components/shared/Skeleton.module.css` | Modify | Make skeleton/wave theme-aware |
| `frontend/h5/src/components/BadgeDot/BadgeDot.module.css` | Modify | Replace local fallback variables with app semantic danger tokens |
| `frontend/h5/src/components/LiveReminderModal/LiveReminderModal.module.css` | Modify | Replace old `--bg-primary/secondary` and `--border-light` usage |
| `frontend/h5/src/components/__tests__/Phase2SharedThemeTokens.test.ts` | Create | Assert scoped shared CSS files do not depend on old light-only tokens |
| `docs/superpowers/specs/2026-06-03-h5-theme-phase2-polish-design.md` | Reference | Source of truth for scope and acceptance |

---

## Task 1: Add Phase 2 Theme Tokens

**Files:**
- Modify: `frontend/h5/src/styles/tokens/colors.css`
- Create: `frontend/h5/src/styles/tokens/__tests__/themeTokens.test.ts`

- [ ] **Step 1: Create token contract test**

Create `frontend/h5/src/styles/tokens/__tests__/themeTokens.test.ts`:

```ts
import { readFileSync } from 'fs';
import { join } from 'path';

const colorsCss = readFileSync(
  join(__dirname, '..', 'colors.css'),
  'utf8',
);

const phase2Tokens = [
  '--page-gradient-profile',
  '--surface-glass',
  '--surface-muted',
  '--chip-bg',
  '--chip-border',
  '--avatar-bg',
  '--avatar-border',
  '--avatar-shadow',
  '--icon-tile-bg',
  '--card-border-accent',
  '--item-subtle-bg',
  '--danger-bg',
  '--danger-border',
  '--danger-text',
  '--skeleton-bg',
  '--skeleton-wave',
  '--focus-ring',
];

describe('phase 2 theme tokens', () => {
  it('defines every phase 2 token in the dark/default theme block', () => {
    const darkBlock = colorsCss.match(/:root\[data-theme="dark"\],[\s\S]*?\n\}/)?.[0] ?? '';

    for (const token of phase2Tokens) {
      expect(darkBlock).toContain(token);
    }
  });

  it('defines every phase 2 token in the light theme block', () => {
    const lightBlock = colorsCss.match(/:root\[data-theme="light"\] \{[\s\S]*?\n\}/)?.[0] ?? '';

    for (const token of phase2Tokens) {
      expect(lightBlock).toContain(token);
    }
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd frontend/h5 && npx jest src/styles/tokens/__tests__/themeTokens.test.ts
```

Expected: FAIL because `themeTokens.test.ts` exists but Phase 2 tokens are not present in `colors.css`.

- [ ] **Step 3: Add Phase 2 tokens to dark/default block**

In `frontend/h5/src/styles/tokens/colors.css`, inside the existing dark/default block after `--shadow-key`, add:

```css
  --page-gradient-profile:
    radial-gradient(circle at 16% 0, rgba(201, 169, 110, 0.22), transparent 30%),
    linear-gradient(180deg, #24211b 0%, #171717 44%, #101010 100%);
  --surface-glass: rgba(44, 44, 44, 0.78);
  --surface-muted: rgba(26, 26, 26, 0.64);
  --chip-bg: rgba(44, 44, 44, 0.82);
  --chip-border: rgba(255, 255, 255, 0.08);
  --avatar-bg: rgba(58, 58, 58, 0.82);
  --avatar-border: rgba(201, 169, 110, 0.75);
  --avatar-shadow: 0 18px 48px rgba(0, 0, 0, 0.34), 0 0 24px rgba(201, 169, 110, 0.08);
  --icon-tile-bg: rgba(201, 169, 110, 0.10);
  --card-border-accent: rgba(201, 169, 110, 0.12);
  --item-subtle-bg: rgba(26, 26, 26, 0.64);
  --danger-bg: rgba(239, 68, 68, 0.08);
  --danger-border: rgba(239, 68, 68, 0.22);
  --danger-text: #f87171;
  --skeleton-bg: rgba(255, 255, 255, 0.08);
  --skeleton-wave: rgba(255, 255, 255, 0.14);
  --focus-ring: rgba(201, 169, 110, 0.22);
```

- [ ] **Step 4: Add Phase 2 tokens to light block**

In `frontend/h5/src/styles/tokens/colors.css`, inside the existing light block after `--shadow-key`, add:

```css
  --page-gradient-profile:
    radial-gradient(circle at 18% 0, rgba(201, 169, 110, 0.22), transparent 32%),
    linear-gradient(180deg, #fff8ea 0%, #faf7f2 38%, #f2eadf 100%);
  --surface-glass: rgba(255, 255, 255, 0.78);
  --surface-muted: rgba(138, 106, 42, 0.06);
  --chip-bg: rgba(255, 255, 255, 0.58);
  --chip-border: rgba(89, 67, 32, 0.10);
  --avatar-bg: linear-gradient(145deg, #fffaf0, #eadcc4);
  --avatar-border: rgba(138, 106, 42, 0.55);
  --avatar-shadow: 0 16px 36px rgba(138, 106, 42, 0.16);
  --icon-tile-bg: rgba(138, 106, 42, 0.08);
  --card-border-accent: rgba(138, 106, 42, 0.16);
  --item-subtle-bg: rgba(138, 106, 42, 0.06);
  --danger-bg: rgba(220, 38, 38, 0.08);
  --danger-border: rgba(220, 38, 38, 0.20);
  --danger-text: #b91c1c;
  --skeleton-bg: rgba(138, 106, 42, 0.10);
  --skeleton-wave: rgba(255, 255, 255, 0.56);
  --focus-ring: rgba(138, 106, 42, 0.22);
```

- [ ] **Step 5: Run token contract test**

Run:

```bash
cd frontend/h5 && npx jest src/styles/tokens/__tests__/themeTokens.test.ts
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```bash
git add frontend/h5/src/styles/tokens/colors.css frontend/h5/src/styles/tokens/__tests__/themeTokens.test.ts
git commit -m "feat(h5/theme): add phase 2 atmosphere tokens"
```

---

## Task 2: Polish Profile Theme Benchmark

**Files:**
- Modify: `frontend/h5/src/pages/User/Profile.module.css`
- Create: `frontend/h5/src/pages/User/__tests__/ProfileThemeTokens.test.ts`
- Test existing: `frontend/h5/src/pages/User/__tests__/Profile.test.tsx`

- [ ] **Step 1: Create Profile CSS contract test**

Create `frontend/h5/src/pages/User/__tests__/ProfileThemeTokens.test.ts`:

```ts
import { readFileSync } from 'fs';
import { join } from 'path';

const css = readFileSync(join(__dirname, '..', 'Profile.module.css'), 'utf8');

describe('Profile phase 2 theme tokens', () => {
  it('uses profile atmosphere and semantic surface tokens', () => {
    expect(css).toContain('background: var(--page-gradient-profile);');
    expect(css).toContain('border: 2px solid var(--avatar-border);');
    expect(css).toContain('background: var(--avatar-bg);');
    expect(css).toContain('box-shadow: var(--avatar-shadow);');
    expect(css).toContain('background: var(--chip-bg);');
    expect(css).toContain('border: 1px solid var(--card-border-accent);');
    expect(css).toContain('background: var(--item-subtle-bg);');
    expect(css).toContain('background: var(--icon-tile-bg);');
    expect(css).toContain('color: var(--danger-text);');
  });

  it('does not keep dark-only values in theme-sensitive Profile blocks', () => {
    expect(css).not.toContain('linear-gradient(180deg, #242424 0%, #171717 42%, #101010 100%)');
    expect(css).not.toContain('background: rgba(44, 44, 44, 0.82);');
    expect(css).not.toContain('background: rgba(26, 26, 26, 0.64);');
    expect(css).not.toContain('color: #f87171;');
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd frontend/h5 && npx jest src/pages/User/__tests__/ProfileThemeTokens.test.ts
```

Expected: FAIL because `Profile.module.css` still contains dark-only backgrounds and hardcoded danger text.

- [ ] **Step 3: Replace Profile page background**

In `frontend/h5/src/pages/User/Profile.module.css`, replace the `.page` block with:

```css
.page {
  min-height: 100%;
  padding: calc(var(--spacing-6) + env(safe-area-inset-top, 0px)) var(--spacing-5)
    calc(104px + env(safe-area-inset-bottom, 0px));
  background: var(--page-gradient-profile);
  color: var(--text-primary);
}
```

- [ ] **Step 4: Replace avatar and badge surfaces**

In `Profile.module.css`, update `.avatarFrame` and `.badges span`:

```css
.avatarFrame {
  display: flex;
  width: 82px;
  height: 82px;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 2px solid var(--avatar-border);
  border-radius: var(--radius-full);
  background: var(--avatar-bg);
  box-shadow: var(--avatar-shadow);
}

.badges span {
  padding: 4px var(--spacing-2);
  border: 1px solid var(--chip-border);
  border-radius: var(--radius-full);
  background: var(--chip-bg);
  color: var(--text-secondary);
  font-size: 10px;
}
```

- [ ] **Step 5: Replace Profile card and action surfaces**

In `Profile.module.css`, update shared surfaces and actions:

```css
.statCard,
.walletCard,
.orderCard,
.menu {
  border: 1px solid var(--card-border-accent);
  background: var(--bg-surface);
  box-shadow: var(--shadow-key);
}

.disabledAction,
.retryButton {
  border: 1px solid var(--card-border-accent);
  border-radius: var(--radius-lg);
  background: var(--surface-glass);
  color: var(--text-brand);
  font-weight: var(--font-weight-bold);
}

.orderItem {
  gap: var(--spacing-3);
  padding: var(--spacing-3);
  border-radius: 18px;
  background: var(--item-subtle-bg);
  color: var(--text-primary);
}

.menuIcon {
  display: inline-flex;
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border-radius: 14px;
  background: var(--icon-tile-bg);
  color: var(--text-brand);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-bold);
}
```

- [ ] **Step 6: Replace Profile danger/loading colors**

In `Profile.module.css`, update danger and spinner blocks:

```css
.logoutButton {
  width: 100%;
  margin-top: var(--spacing-6);
  padding: var(--spacing-4);
  border: 1px solid var(--danger-border);
  border-radius: 22px;
  background: var(--danger-bg);
  color: var(--danger-text);
  font-weight: var(--font-weight-bold);
}

.spinner {
  width: 32px;
  height: 32px;
  border: 2px solid var(--card-border-accent);
  border-top-color: var(--text-brand);
  border-radius: var(--radius-full);
  animation: spin 0.8s linear infinite;
}

.errorText {
  color: var(--danger-text);
}
```

- [ ] **Step 7: Run Profile tests**

Run:

```bash
cd frontend/h5 && npx jest src/pages/User/__tests__/ProfileThemeTokens.test.ts src/pages/User/__tests__/Profile.test.tsx
```

Expected: PASS.

- [ ] **Step 8: Commit**

Run:

```bash
git add frontend/h5/src/pages/User/Profile.module.css frontend/h5/src/pages/User/__tests__/ProfileThemeTokens.test.ts
git commit -m "feat(h5/theme): polish profile light and dark surfaces"
```

---

## Task 3: Polish Missed Addresses And Order Detail Pages

**Files:**
- Modify: `frontend/h5/src/pages/Addresses/Addresses.module.css`
- Modify: `frontend/h5/src/pages/Order/Detail.module.css`
- Create: `frontend/h5/src/pages/__tests__/Phase2PageThemeTokens.test.ts`
- Test existing: `frontend/h5/src/pages/Addresses/__tests__/Addresses.test.tsx`
- Test existing: `frontend/h5/src/pages/Order/__tests__/Detail.test.tsx`

- [ ] **Step 1: Create scoped page contract test**

Create `frontend/h5/src/pages/__tests__/Phase2PageThemeTokens.test.ts`:

```ts
import { readFileSync } from 'fs';
import { join } from 'path';

function readPageCss(relativePath: string) {
  return readFileSync(join(__dirname, '..', relativePath), 'utf8');
}

describe('phase 2 scoped page theme tokens', () => {
  it('tokenizes Addresses page surfaces and actions', () => {
    const css = readPageCss('Addresses/Addresses.module.css');

    expect(css).toContain('background: var(--page-gradient-profile);');
    expect(css).toContain('border: 1px solid var(--card-border-accent);');
    expect(css).toContain('background: var(--surface-glass);');
    expect(css).toContain('color: var(--danger-text);');
    expect(css).not.toContain('var(--bg-page-start, #1a1a1a)');
    expect(css).not.toContain('background: rgba(255, 255, 255, 0.06);');
    expect(css).not.toContain('color: #ffb3b3;');
  });

  it('tokenizes Order detail page surfaces and inline feedback', () => {
    const css = readPageCss('Order/Detail.module.css');

    expect(css).toContain('border: 1px solid var(--card-border-accent);');
    expect(css).toContain('background: var(--bg-surface);');
    expect(css).toContain('background: var(--surface-glass);');
    expect(css).toContain('background: var(--item-subtle-bg);');
    expect(css).not.toContain('background: rgba(0, 0, 0, 0.78);');
    expect(css).not.toContain('color: #fff;');
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd frontend/h5 && npx jest src/pages/__tests__/Phase2PageThemeTokens.test.ts
```

Expected: FAIL because `Addresses` and `Order/Detail` still include legacy/fallback surfaces.

- [ ] **Step 3: Update Addresses theme surfaces**

In `frontend/h5/src/pages/Addresses/Addresses.module.css`, apply these replacements:

```css
.page {
  min-height: 100vh;
  padding-bottom: calc(96px + env(safe-area-inset-bottom, 0px));
  background: var(--page-gradient-profile);
  color: var(--text-primary);
}

.backButton {
  display: grid;
  width: 40px;
  height: 40px;
  place-items: center;
  border: 1px solid var(--card-border-accent);
  border-radius: var(--radius-full);
  background: var(--surface-glass);
  color: var(--text-primary);
  font-size: 30px;
  line-height: 1;
}

.errorBanner,
.loading,
.emptyState {
  margin: var(--spacing-4) 0;
  padding: var(--spacing-4);
  border: 1px solid var(--card-border-accent);
  border-radius: 22px;
  background: var(--bg-surface);
  text-align: center;
}

.errorBanner {
  color: var(--danger-text);
  font-size: var(--font-size-sm);
}

.card {
  padding: var(--spacing-4);
  border: 1px solid var(--card-border-accent);
  border-radius: 22px;
  background: var(--bg-surface);
  box-shadow: var(--shadow-key);
}

.setDefaultButton {
  min-height: 36px;
  padding: 0 var(--spacing-3);
  border: 1px solid var(--card-border-accent);
  border-radius: 12px;
  background: var(--surface-glass);
  color: var(--text-brand);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-bold);
}
```

- [ ] **Step 4: Update Order detail theme surfaces**

In `frontend/h5/src/pages/Order/Detail.module.css`, apply these replacements:

```css
.backButton {
  display: inline-flex;
  width: 44px;
  height: 44px;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--card-border-accent);
  border-radius: var(--radius-full);
  background: var(--surface-glass);
  color: var(--text-brand);
  font-size: 28px;
  line-height: 1;
  cursor: pointer;
}

.statusBadge {
  display: inline-flex;
  align-self: flex-start;
  padding: 6px var(--spacing-4);
  border-radius: var(--radius-full);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-bold);
  background: var(--icon-tile-bg);
  color: var(--text-brand);
}

.card {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-3);
  padding: var(--spacing-5);
  border: 1px solid var(--card-border-accent);
  border-radius: var(--radius-lg);
  background: var(--bg-surface);
}

.timeline li {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-left: var(--spacing-4);
  border-left: 2px solid var(--card-border-accent);
}

.secondaryButton {
  border: 1px solid var(--border-subtle);
  background: var(--surface-glass);
  color: var(--text-primary);
}

.primaryButton {
  border: none;
  background: var(--text-brand);
  color: var(--bg-surface);
}

.toast {
  position: fixed;
  left: 50%;
  bottom: calc(var(--spacing-8) + env(safe-area-inset-bottom, 0px));
  transform: translateX(-50%);
  padding: var(--spacing-3) var(--spacing-5);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  background: var(--surface-glass);
  color: var(--text-primary);
  font-size: var(--font-size-sm);
  z-index: 50;
}
```

- [ ] **Step 5: Run scoped page tests**

Run:

```bash
cd frontend/h5 && npx jest src/pages/__tests__/Phase2PageThemeTokens.test.ts src/pages/Addresses/__tests__/Addresses.test.tsx src/pages/Order/__tests__/Detail.test.tsx
```

Expected: PASS.

- [ ] **Step 6: Commit**

Run:

```bash
git add frontend/h5/src/pages/Addresses/Addresses.module.css frontend/h5/src/pages/Order/Detail.module.css frontend/h5/src/pages/__tests__/Phase2PageThemeTokens.test.ts
git commit -m "feat(h5/theme): polish addresses and order detail themes"
```

---

## Task 4: Polish Scoped Standard Pages

**Files:**
- Modify: `frontend/h5/src/pages/Notifications/Notifications.module.css`
- Modify: `frontend/h5/src/pages/Follow/Following.module.css`
- Modify: `frontend/h5/src/pages/History/AuctionHistory.module.css`
- Modify: `frontend/h5/src/pages/__tests__/Phase2PageThemeTokens.test.ts`
- Test existing: `frontend/h5/src/pages/Notifications/__tests__/Notifications.test.tsx`
- Test existing: `frontend/h5/src/pages/Follow/__tests__/Following.test.tsx`
- Test existing: `frontend/h5/src/pages/History/__tests__/AuctionHistory.test.tsx`

- [ ] **Step 1: Extend scoped page contract test**

Append this test to `frontend/h5/src/pages/__tests__/Phase2PageThemeTokens.test.ts`:

```ts
  it.each([
    ['Notifications/Notifications.module.css'],
    ['Follow/Following.module.css'],
    ['History/AuctionHistory.module.css'],
  ])('keeps %s on semantic theme surfaces', (relativePath) => {
    const css = readPageCss(relativePath);

    expect(css).toContain('var(--bg-page)');
    expect(css).toContain('var(--bg-surface)');
    expect(css).toContain('var(--border-subtle)');
    expect(css).not.toContain('var(--bg-primary)');
    expect(css).not.toContain('var(--bg-secondary)');
    expect(css).not.toContain('var(--border-light)');
  });
```

- [ ] **Step 2: Run test and note current result**

Run:

```bash
cd frontend/h5 && npx jest src/pages/__tests__/Phase2PageThemeTokens.test.ts
```

Expected: PASS if Phase 1 already covered these files, or FAIL only for old tokens that this task must remove.

- [ ] **Step 3: Replace remaining old tokens in scoped standard pages**

For each of these files:

```text
frontend/h5/src/pages/Notifications/Notifications.module.css
frontend/h5/src/pages/Follow/Following.module.css
frontend/h5/src/pages/History/AuctionHistory.module.css
```

Apply the following mapping wherever it appears:

```text
var(--bg-primary)    -> var(--bg-surface)
var(--bg-secondary)  -> var(--surface-muted)
var(--bg-tertiary)   -> var(--item-subtle-bg)
var(--border-light)  -> var(--border-subtle)
var(--border-default)-> var(--border-subtle)
```

Do not change intentionally functional colors such as success/warning/error badges unless contrast is visibly wrong.

- [ ] **Step 4: Run scoped standard page tests**

Run:

```bash
cd frontend/h5 && npx jest src/pages/__tests__/Phase2PageThemeTokens.test.ts src/pages/Notifications/__tests__/Notifications.test.tsx src/pages/Follow/__tests__/Following.test.tsx src/pages/History/__tests__/AuctionHistory.test.tsx
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add frontend/h5/src/pages/Notifications/Notifications.module.css frontend/h5/src/pages/Follow/Following.module.css frontend/h5/src/pages/History/AuctionHistory.module.css frontend/h5/src/pages/__tests__/Phase2PageThemeTokens.test.ts
git commit -m "feat(h5/theme): align standard pages with phase 2 tokens"
```

---

## Task 5: Polish Shared Components

**Files:**
- Modify: `frontend/h5/src/components/shared/Button.module.css`
- Modify: `frontend/h5/src/components/shared/Input.module.css`
- Modify: `frontend/h5/src/components/shared/Loading.module.css`
- Modify: `frontend/h5/src/components/shared/Skeleton.module.css`
- Modify: `frontend/h5/src/components/BadgeDot/BadgeDot.module.css`
- Modify: `frontend/h5/src/components/LiveReminderModal/LiveReminderModal.module.css`
- Create: `frontend/h5/src/components/__tests__/Phase2SharedThemeTokens.test.ts`
- Test existing: `frontend/h5/src/components/BadgeDot/__tests__/BadgeDot.test.tsx`

- [ ] **Step 1: Create shared component contract test**

Create `frontend/h5/src/components/__tests__/Phase2SharedThemeTokens.test.ts`:

```ts
import { readFileSync } from 'fs';
import { join } from 'path';

const componentRoot = join(__dirname, '..');

function readComponentCss(relativePath: string) {
  return readFileSync(join(componentRoot, relativePath), 'utf8');
}

describe('phase 2 shared component theme tokens', () => {
  it.each([
    ['shared/Input.module.css'],
    ['shared/Loading.module.css'],
    ['shared/Skeleton.module.css'],
    ['LiveReminderModal/LiveReminderModal.module.css'],
  ])('removes old light-only tokens from %s', (relativePath) => {
    const css = readComponentCss(relativePath);

    expect(css).not.toContain('var(--bg-primary)');
    expect(css).not.toContain('var(--bg-secondary)');
    expect(css).not.toContain('var(--bg-tertiary)');
    expect(css).not.toContain('var(--border-light)');
    expect(css).not.toContain('var(--border-default)');
  });

  it('uses phase 2 danger tokens in BadgeDot fallback styling', () => {
    const css = readComponentCss('BadgeDot/BadgeDot.module.css');

    expect(css).toContain('var(--danger-text)');
    expect(css).toContain('var(--bg-surface)');
  });

  it('uses theme-aware hover states in shared Button', () => {
    const css = readComponentCss('shared/Button.module.css');

    expect(css).toContain('background: var(--surface-muted);');
    expect(css).toContain('outline: 2px solid var(--focus-ring);');
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd frontend/h5 && npx jest src/components/__tests__/Phase2SharedThemeTokens.test.ts
```

Expected: FAIL because shared components still use old light-only tokens.

- [ ] **Step 3: Update shared Button**

In `frontend/h5/src/components/shared/Button.module.css`, apply these replacements:

```css
.button:focus-visible {
  outline: 2px solid var(--focus-ring);
  outline-offset: 2px;
}

.outline {
  background: transparent;
  border: 2px solid var(--text-brand);
  color: var(--text-brand);
}

.outline:hover:not(:disabled) {
  background: var(--surface-muted);
}

.ghost {
  background: transparent;
  color: var(--text-brand);
}

.ghost:hover:not(:disabled) {
  background: var(--surface-muted);
}
```

Keep `.primary` and `.secondary` brand variants unchanged unless visual validation shows a contrast failure.

- [ ] **Step 4: Update shared Input**

In `frontend/h5/src/components/shared/Input.module.css`, apply these replacements:

```css
.inputWrapper {
  display: flex;
  align-items: center;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-input);
  transition: all var(--transition-fast);
}

.inputWrapper:hover:not(.disabled) {
  border-color: var(--card-border-accent);
}

.inputWrapper.focused {
  border-color: var(--text-brand);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.inputWrapper.disabled {
  background: var(--surface-muted);
  cursor: not-allowed;
}

.input::placeholder {
  color: var(--text-secondary);
}

.input:disabled {
  cursor: not-allowed;
  color: var(--text-secondary);
}

.clearButton {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  margin-right: var(--spacing-2);
  background: var(--surface-muted);
  border: none;
  border-radius: var(--radius-full);
  color: var(--text-secondary);
  font-size: 10px;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.clearButton:hover {
  background: var(--icon-tile-bg);
  color: var(--text-brand);
}
```

- [ ] **Step 5: Update Loading and Skeleton**

In `frontend/h5/src/components/shared/Loading.module.css`, replace:

```css
.fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--surface-glass);
  z-index: 9999;
}

.spinner {
  border: 3px solid var(--skeleton-bg);
  border-top-color: var(--text-brand);
  border-radius: var(--radius-full);
  animation: spin 0.8s linear infinite;
}
```

In `frontend/h5/src/components/shared/Skeleton.module.css`, replace:

```css
.skeleton {
  display: block;
  background: var(--skeleton-bg);
}

.wave::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(
    90deg,
    transparent,
    var(--skeleton-wave),
    transparent
  );
  animation: wave 2s linear infinite;
}
```

- [ ] **Step 6: Update BadgeDot**

In `frontend/h5/src/components/BadgeDot/BadgeDot.module.css`, replace the local fallback variables at the top of `.badge` with:

```css
.badge {
  --touchpoint-danger: var(--danger-text);
  --touchpoint-badge-text: var(--bg-surface);
  --touchpoint-border: var(--bg-surface);

  position: absolute;
```

Keep the remaining `.badge`, `.count`, and `.dot` layout unchanged.

- [ ] **Step 7: Update LiveReminderModal**

In `frontend/h5/src/components/LiveReminderModal/LiveReminderModal.module.css`, apply this mapping:

```text
var(--bg-primary, #ffffff)       -> var(--bg-surface)
var(--bg-secondary, #f5f5f5)     -> var(--surface-muted)
var(--bg-tertiary, #e5e5e5)      -> var(--item-subtle-bg)
var(--border-light, #e5e5e5)     -> var(--border-subtle)
var(--color-primary-50, #fff7ed) -> var(--icon-tile-bg)
var(--color-primary-500, #f97316)-> var(--text-brand)
```

Also replace modal shadow with:

```css
box-shadow: var(--shadow-key);
```

- [ ] **Step 8: Run shared component tests**

Run:

```bash
cd frontend/h5 && npx jest src/components/__tests__/Phase2SharedThemeTokens.test.ts src/components/BadgeDot/__tests__/BadgeDot.test.tsx
```

Expected: PASS.

- [ ] **Step 9: Commit**

Run:

```bash
git add frontend/h5/src/components/shared/Button.module.css frontend/h5/src/components/shared/Input.module.css frontend/h5/src/components/shared/Loading.module.css frontend/h5/src/components/shared/Skeleton.module.css frontend/h5/src/components/BadgeDot/BadgeDot.module.css frontend/h5/src/components/LiveReminderModal/LiveReminderModal.module.css frontend/h5/src/components/__tests__/Phase2SharedThemeTokens.test.ts
git commit -m "feat(h5/theme): polish shared component theme surfaces"
```

---

## Task 6: Theme Regression Verification

**Files:**
- No production file changes expected
- Optional update only if previous tasks reveal missing contract coverage: `frontend/h5/src/pages/__tests__/Phase2PageThemeTokens.test.ts`

- [ ] **Step 1: Run all Phase 2 contract tests**

Run:

```bash
cd frontend/h5 && npx jest \
  src/styles/tokens/__tests__/themeTokens.test.ts \
  src/pages/User/__tests__/ProfileThemeTokens.test.ts \
  src/pages/__tests__/Phase2PageThemeTokens.test.ts \
  src/components/__tests__/Phase2SharedThemeTokens.test.ts
```

Expected: PASS.

- [ ] **Step 2: Run theme-related behavior tests**

Run:

```bash
cd frontend/h5 && npx jest \
  src/store/__tests__/themeContext.test.tsx \
  src/components/ThemeToggle/__tests__/ThemeToggle.test.tsx \
  src/components/shared/__tests__/PageHeader.test.tsx \
  src/pages/User/__tests__/Profile.test.tsx \
  src/pages/Addresses/__tests__/Addresses.test.tsx \
  src/pages/Order/__tests__/Detail.test.tsx \
  src/pages/Notifications/__tests__/Notifications.test.tsx \
  src/pages/Follow/__tests__/Following.test.tsx \
  src/pages/History/__tests__/AuctionHistory.test.tsx
```

Expected: PASS.

- [ ] **Step 3: Run type checking**

Run:

```bash
cd frontend/h5 && npx tsc --noEmit
```

Expected: no TypeScript errors.

- [ ] **Step 4: Run production build**

Run:

```bash
cd frontend/h5 && npm run build
```

Expected: build completes successfully.

- [ ] **Step 5: Start local dev server for visual validation**

Run:

```bash
cd frontend/h5 && npm run dev
```

Expected: Vite prints a local URL such as `http://localhost:5173/`.

- [ ] **Step 6: Manual dark/light visual checklist**

Use the dev server and manually verify both `data-theme="dark"` and `data-theme="light"` on these routes:

```text
/profile
/addresses
/order/56
/notifications
/following
/history
/
/live
```

For each route, verify:

```text
- Page background matches the selected theme.
- Card/list surfaces do not mix dark-only backgrounds into light mode.
- Primary and secondary text remain readable.
- Gold brand accents remain visible without being too bright.
- Bottom navigation remains readable.
- Destructive/logout/error states remain clear.
```

- [ ] **Step 7: Document unrelated failures only if they appear**

If a targeted Jest test fails for a reason unrelated to Phase 2 styling, add a short note to the final handoff message. Do not change unrelated production logic in this task.

- [ ] **Step 8: Final verification commit if needed**

If Task 6 required no file changes, do not create a commit. If it required a small contract-test fix, run:

```bash
git add frontend/h5/src/pages/__tests__/Phase2PageThemeTokens.test.ts frontend/h5/src/components/__tests__/Phase2SharedThemeTokens.test.ts
git commit -m "test(h5/theme): tighten phase 2 theme contracts"
```

---

## Self-Review

- Spec §2 goals are covered by Tasks 1-6.
- Spec §4 follow-ups are covered by Tasks 1-5; settings page, system theme, visual snapshots, and admin/test-dashboard remain out of scope.
- Spec §7 token model is covered by Task 1.
- Spec §8 Profile benchmark is covered by Task 2.
- Spec §9 page scope is covered by Tasks 2-4.
- Spec §10 component scope is covered by Task 5.
- Spec §13 testing strategy is covered by Task 6.
- No task touches `/login`, `frontend/admin`, or `frontend/test-dashboard`.
- Contract tests intentionally inspect CSS source because CSS Modules do not expose computed runtime values in Jest.
- Each task ends with a focused commit and avoids unrelated files.


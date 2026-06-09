# Live Reminder Modal V1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 H5 开播提醒弹窗落地为已选定的方案 A：现代浮动、悬浮 SVG 图标、内嵌直播间卡片、渐变主按钮与双主题 token 适配。

**Architecture:** 复用现有 `LiveReminderModal` 组件与 `MobileContainer` 挂载逻辑，不改接口、不改业务流、不新增依赖。只调整组件 DOM 的图标结构、CSS Module 视觉样式，并补充组件级测试锁定关键语义与可访问性。

**Tech Stack:** React 18、TypeScript、CSS Modules、Jest、Testing Library、Vite H5。

---

## File Structure

- Modify: `frontend/h5/src/components/LiveReminderModal/index.tsx`
  - 责任：保留弹窗行为、埋点、导航逻辑；将 emoji 替换为内联 SVG `aria-hidden` 摄像机图标。
- Modify: `frontend/h5/src/components/LiveReminderModal/LiveReminderModal.module.css`
  - 责任：实现方案 A 视觉，包括顶部悬浮图标、24px 圆角、内嵌信息卡、渐变主按钮、token 适配与 reduced-motion。
- Modify: `frontend/h5/src/components/LiveReminderModal/__tests__/LiveReminderModal.test.tsx`
  - 责任：补充测试，确保弹窗标题/按钮/直播状态仍可见，emoji 不再作为文本渲染，SVG 图标为装饰性元素。

---

### Task 1: 组件语义与图标替换

**Files:**
- Modify: `frontend/h5/src/components/LiveReminderModal/__tests__/LiveReminderModal.test.tsx`
- Modify: `frontend/h5/src/components/LiveReminderModal/index.tsx`

- [ ] **Step 1: Write the failing test**

在 `frontend/h5/src/components/LiveReminderModal/__tests__/LiveReminderModal.test.tsx` 的 `describe` 内追加：

```tsx
it('renders the v1 decorative svg camera icon without emoji text', () => {
  render(
    <MemoryRouter>
      <LiveReminderModal
        isOpen
        onClose={() => {}}
        stream={{ id: 6, name: 'Demo 商家直播间', avatarUrl: '', statusText: '正在直播' }}
      />
    </MemoryRouter>,
  );

  const dialog = screen.getByRole('dialog', { name: '直播开播提醒' });
  expect(dialog).toBeInTheDocument();
  expect(screen.queryByText('🎥')).not.toBeInTheDocument();
  expect(screen.getByTestId('live-reminder-camera-icon')).toHaveAttribute('aria-hidden', 'true');
  expect(screen.getByText('正在直播')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: '立即前往' })).toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd frontend/h5 && npm test -- LiveReminderModal.test.tsx --runInBand
```

Expected: FAIL，因为当前组件仍渲染 emoji `🎥`，且没有 `data-testid="live-reminder-camera-icon"`。

- [ ] **Step 3: Write minimal component implementation**

将 `frontend/h5/src/components/LiveReminderModal/index.tsx` 中 `styles.iconWrapper` 内的 emoji 替换为：

```tsx
<svg
  className={styles.cameraIcon}
  data-testid="live-reminder-camera-icon"
  aria-hidden="true"
  xmlns="http://www.w3.org/2000/svg"
  width="32"
  height="32"
  viewBox="0 0 24 24"
  fill="none"
>
  <defs>
    <linearGradient id="live-reminder-camera-gradient" x1="0" y1="0" x2="24" y2="24" gradientUnits="userSpaceOnUse">
      <stop offset="0" stopColor="var(--color-primary-500)" />
      <stop offset="1" stopColor="var(--color-primary-600)" />
    </linearGradient>
  </defs>
  <path
    d="m22 8-6 4 6 4V8Z"
    stroke="url(#live-reminder-camera-gradient)"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  />
  <rect
    width="14"
    height="12"
    x="2"
    y="6"
    rx="2"
    ry="2"
    stroke="url(#live-reminder-camera-gradient)"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  />
</svg>
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd frontend/h5 && npm test -- LiveReminderModal.test.tsx --runInBand
```

Expected: PASS。

---

### Task 2: 方案 A CSS 视觉落地

**Files:**
- Modify: `frontend/h5/src/components/LiveReminderModal/LiveReminderModal.module.css`
- Modify: `frontend/h5/src/components/LiveReminderModal/__tests__/LiveReminderModal.test.tsx`

- [ ] **Step 1: Write the CSS contract test**

在 `LiveReminderModal.test.tsx` 顶部加入：

```tsx
import fs from 'fs';
import path from 'path';
```

在 `describe` 内追加：

```tsx
it('keeps v1 visual styles bound to semantic theme tokens', () => {
  const css = fs.readFileSync(
    path.join(__dirname, '../LiveReminderModal.module.css'),
    'utf8',
  );

  expect(css).toContain('border-radius: 24px');
  expect(css).toContain('background: var(--bg-surface)');
  expect(css).toContain('background: var(--item-subtle-bg)');
  expect(css).toContain('background: var(--gradient-primary)');
  expect(css).toContain('color: var(--text-secondary');
  expect(css).toContain('@media (prefers-reduced-motion: reduce)');
});
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd frontend/h5 && npm test -- LiveReminderModal.test.tsx --runInBand
```

Expected: FAIL，因为当前 CSS 使用 `var(--radius-card, 16px)`、底部平铺按钮且没有 `prefers-reduced-motion`。

- [ ] **Step 3: Implement CSS**

将 `LiveReminderModal.module.css` 调整为方案 A：

```css
.overlay {
  position: fixed;
  inset: 0;
  background-color: var(--bg-overlay, rgba(0, 0, 0, 0.6));
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 0 32px;
  animation: fadeIn var(--transition-fast, 0.2s) ease-out forwards;
}

.modal {
  position: relative;
  width: 100%;
  max-width: 320px;
  padding: 32px 24px 24px;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: 24px;
  box-shadow: var(--shadow-key);
  text-align: center;
  animation: slideUp var(--transition-fast, 0.2s) ease-out forwards;
}

.header {
  padding: 0;
  text-align: center;
}

.iconWrapper {
  position: absolute;
  top: -32px;
  left: 50%;
  width: 64px;
  height: 64px;
  transform: translateX(-50%);
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full, 9999px);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.10);
}

.cameraIcon {
  width: 32px;
  height: 32px;
  display: block;
}

.title {
  font-size: 20px;
  font-weight: var(--font-weight-semibold, 600);
  color: var(--text-primary, #171717);
  margin: 16px 0 8px;
}

.content {
  padding: 0;
  text-align: center;
}

.message {
  font-size: var(--font-size-sm, 14px);
  color: var(--text-secondary, #525252);
  margin: 0 0 24px;
  line-height: 1.5;
}

.streamInfo {
  display: flex;
  align-items: center;
  gap: 12px;
  background: var(--item-subtle-bg);
  padding: 12px;
  border-radius: 16px;
  text-align: left;
  margin-bottom: 24px;
}

.footer {
  display: flex;
  gap: 12px;
  border-top: 0;
}

.button {
  flex: 1;
  padding: 12px 0;
  border: none;
  border-radius: 12px;
  font-size: 15px;
  font-weight: var(--font-weight-medium, 500);
  cursor: pointer;
  transition: transform var(--transition-fast, 0.2s), background var(--transition-fast, 0.2s), box-shadow var(--transition-fast, 0.2s);
}

.buttonCancel {
  color: var(--text-secondary, #525252);
  background: var(--item-subtle-bg);
}

.buttonConfirm {
  color: var(--text-inverse, #fff);
  background: var(--gradient-primary);
  box-shadow: 0 4px 12px rgba(249, 115, 22, 0.30);
}

.button:active {
  transform: scale(0.98);
}

@media (prefers-reduced-motion: reduce) {
  .overlay,
  .modal,
  .liveDot {
    animation: none;
  }

  .button {
    transition: none;
  }
}
```

保留并兼容原文件中 `.streamAvatar`、`.streamAvatarFallback`、`.streamDetails`、`.streamName`、`.streamStatus`、`.liveDot`、`fadeIn`、`fadeOut`、`slideUp`、`slideDown`、`pulseDot` 等现有选择器与关键帧。

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd frontend/h5 && npm test -- LiveReminderModal.test.tsx --runInBand
```

Expected: PASS。

---

### Task 3: 回归验证

**Files:**
- Verify: `frontend/h5/src/components/LiveReminderModal/index.tsx`
- Verify: `frontend/h5/src/components/LiveReminderModal/LiveReminderModal.module.css`
- Verify: `frontend/h5/src/components/LiveReminderModal/__tests__/LiveReminderModal.test.tsx`

- [ ] **Step 1: Run focused tests**

Run:

```bash
cd frontend/h5 && npm test -- LiveReminderModal.test.tsx --runInBand
```

Expected: PASS，包含新增两条测试和原头像兜底测试。

- [ ] **Step 2: Run integration guard tests**

Run:

```bash
cd frontend/h5 && npm test -- MobileShell.test.tsx --runInBand
```

Expected: PASS，确保 `MobileContainer` 的弹窗拉取、关闭、跳转、埋点行为未破坏。

- [ ] **Step 3: Run diagnostics/lint**

Run:

```bash
cd frontend/h5 && npm run lint
```

Expected: PASS 或仅出现与本次改动无关的既有问题；本次改动文件不得新增 lint error。

- [ ] **Step 4: Commit implementation**

```bash
git add frontend/h5/src/components/LiveReminderModal/index.tsx \
  frontend/h5/src/components/LiveReminderModal/LiveReminderModal.module.css \
  frontend/h5/src/components/LiveReminderModal/__tests__/LiveReminderModal.test.tsx \
  docs/superpowers/plans/2026-06-10-live-reminder-modal-v1.md \
  docs/superpowers/specs/ui/live-popup-v1.md
git commit -m "feat(h5): refresh live reminder modal"
```

---

## Self-Review

- Spec coverage: 方案 A 的悬浮 SVG、渐变、内嵌卡片、24px 圆角、直播状态呼吸灯、双主题 token 均有对应任务。
- Placeholder scan: 无 `TBD`、`TODO`、`implement later` 等占位内容。
- Type consistency: 未引入新 props；`LiveReminderModalProps`、`StreamInfo`、埋点和导航逻辑保持不变。

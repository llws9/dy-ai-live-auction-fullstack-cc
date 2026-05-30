# 用户触达体系（一期） Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `frontend/h5` 落地一期用户触达闭环：红点徽标、顶部 Toast、重新登录后一次性开播弹窗。

**Architecture:** 采用“先 UI 设计、再业务接入”的两段式方案：先完成红点与顶部 Toast 的 theme-ready 静态 UI，再接入 Mock 数据、全局 Toast 行为和登录后弹窗触发。触达数据、Toast 展示入口、登录弹窗标记各自保持单一事实源，避免新增并行 Toast 体系；日间/夜间一键切换本期不实现开关，但所有新增触达 UI 必须通过 CSS 变量具备主题适配能力。

**Tech Stack:** React 18、TypeScript、CSS Modules、React Router、Jest、Testing Library、Vite。

---

## File Structure

- Create/Modify: `frontend/h5/src/components/BadgeDot/index.tsx`
- Create/Modify: `frontend/h5/src/components/BadgeDot/BadgeDot.module.css`
- Create: `frontend/h5/src/components/BadgeDot/__tests__/BadgeDot.test.tsx`
- Create: `frontend/h5/src/hooks/useTouchpointNotifications.ts`
- Create/Modify: `frontend/h5/src/components/Toast/Toast.module.css`
- Create: `frontend/h5/src/components/Toast/__tests__/ToastProvider.test.tsx`
- Modify: `frontend/h5/src/components/Toast/index.tsx`
- Modify: `frontend/h5/src/components/MobileShell/BottomNav.tsx`
- Modify: `frontend/h5/src/components/MobileShell/MobileContainer.tsx`
- Modify: `frontend/h5/src/components/MobileShell/MobileShell.module.css`
- Modify: `frontend/h5/src/__tests__/components/MobileShell.test.tsx`
- Modify: `frontend/h5/src/pages/User/Index.tsx`
- Modify: `frontend/h5/src/pages/User/Profile.module.css`
- Modify: `frontend/h5/src/pages/User/__tests__/Profile.test.tsx`
- Modify: `frontend/h5/src/store/authContext.tsx`
- Modify: `frontend/h5/src/pages/Live/index.tsx`
- Modify: `frontend/h5/src/pages/Live/Live.module.css`
- Test commands:
  - `npm test -- --runInBand src/components/BadgeDot/__tests__/BadgeDot.test.tsx`
  - `npm test -- --runInBand src/__tests__/components/MobileShell.test.tsx`
  - `npm test -- --runInBand src/pages/User/__tests__/Profile.test.tsx`
  - `npm test -- --runInBand src/components/Toast/__tests__/ToastProvider.test.tsx`
  - `npm run build`

---

## Theme-Ready UI Contract

新增红点和 Toast UI 必须满足：
- 不实现主题切换按钮，不修改全局主题系统；只预留日间/夜间主题变量。
- CSS 不硬编码核心语义颜色，必须优先使用局部 CSS variables，并提供 fallback。
- 组件根节点或容器要能跟随未来 `[data-theme='light']` / `[data-theme='dark']` 或上层全局 CSS variables 自动切换。
- 触控目标不低于 44px；Toast action/close 按钮需要可点击、可聚焦。
- 暗色默认值适配当前 H5；浅色变量必须保证文字对比度和红点/Toast 类型色可辨识。

Recommended local variables:

```css
--touchpoint-surface;
--touchpoint-surface-strong;
--touchpoint-text;
--touchpoint-text-muted;
--touchpoint-accent;
--touchpoint-warning;
--touchpoint-danger;
--touchpoint-success;
--touchpoint-info;
--touchpoint-shadow;
--touchpoint-border;
```

---

### Task 0: 红点与 Toast 静态 UI 设计

**Files:**
- Create/Modify: `frontend/h5/src/components/BadgeDot/index.tsx`
- Create/Modify: `frontend/h5/src/components/BadgeDot/BadgeDot.module.css`
- Modify: `frontend/h5/src/components/Toast/index.tsx`
- Create/Modify: `frontend/h5/src/components/Toast/Toast.module.css`

**Boundary:**
- 本任务只做 UI 与组件接口，不接 Mock 数据、不改 `BottomNav`、不改 `Profile`、不改 `Live`、不改登录逻辑。
- `ToastProvider` 可以保留现有行为，也可以先用静态演示结构，但必须保持后续 Task 3 能接入 `showToast` 行为。
- 必须支持未来日间/夜间一键切换：用 CSS variables 做 theme-ready，不在本期实现切换按钮。

- [ ] **Step 1: 交付 UI 设计提示词给外部模型**

Use this exact prompt:

```text
请先只做移动端 H5 用户触达 UI 设计与静态组件实现，不接业务逻辑。

目标文件：
1. frontend/h5/src/components/BadgeDot/index.tsx
2. frontend/h5/src/components/BadgeDot/BadgeDot.module.css
3. frontend/h5/src/components/Toast/index.tsx
4. frontend/h5/src/components/Toast/Toast.module.css

要求：
1. 不修改登录、路由、MobileContainer、Profile、BottomNav、Live 页面业务逻辑。
2. 不引入新依赖。
3. 使用 React 18 + TypeScript + CSS Modules。
4. 保持移动端 H5/iOS-like 触控体验，按钮高度至少 44px。
5. 视觉风格贴合直播竞拍产品：高级、低打扰、清晰可读。
6. 必须支持未来日间/夜间一键切换：不要实现切换按钮，但 CSS 必须使用 theme-ready variables，并为暗色/浅色提供可覆盖变量。
7. BadgeDot 支持 count、max、dot、ariaLabel、className。
8. BadgeDot 支持 4 种状态：纯红点、数字、99+、0 不展示。
9. Toast 支持 success/warning/danger/error/info/loading 样式。
10. Toast UI 支持 title、message、actionText、关闭按钮、最多 3 条堆叠。
11. 保留旧 showToast(message, type, duration) 调用兼容性；对象签名 showToast({ type, title, message, duration, actionText, onAction }) 后续会接入。
12. 输出完整代码，不要改其他文件。
```

- [ ] **Step 2: 核对 UI 文件边界**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc
git diff --name-only
```

Expected: 只出现以下 4 个 UI 文件，或其中一部分：

```text
frontend/h5/src/components/BadgeDot/index.tsx
frontend/h5/src/components/BadgeDot/BadgeDot.module.css
frontend/h5/src/components/Toast/index.tsx
frontend/h5/src/components/Toast/Toast.module.css
```

- [ ] **Step 3: 核对 theme-ready 变量**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc
python3 - <<'PY'
from pathlib import Path
for root in [Path("frontend/h5/src/components/BadgeDot"), Path("frontend/h5/src/components/Toast")]:
    for path in root.rglob("*"):
        if path.is_file() and path.suffix in {".css", ".tsx"}:
            for line in path.read_text().splitlines():
                if "touchpoint-" in line:
                    print(f"{path}: {line.strip()}")
PY
```

Expected: 输出包含 `--touchpoint-surface`、`--touchpoint-text`、`--touchpoint-danger`、`--touchpoint-success`、`--touchpoint-warning`。

- [ ] **Step 4: 运行 TypeScript 构建**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm run build
```

Expected: PASS。若失败，只修复 UI 组件类型错误，不接业务逻辑。

- [ ] **Step 5: 提交**

```bash
git add frontend/h5/src/components/BadgeDot frontend/h5/src/components/Toast
git commit -m "feat(h5): design touchpoint UI components"
```

---

### Task 1: BadgeDot UI 验证与 Mock 数据源

**Files:**
- Create/Modify: `frontend/h5/src/components/BadgeDot/index.tsx`
- Create/Modify: `frontend/h5/src/components/BadgeDot/BadgeDot.module.css`
- Create: `frontend/h5/src/components/BadgeDot/__tests__/BadgeDot.test.tsx`
- Create: `frontend/h5/src/hooks/useTouchpointNotifications.ts`

- [ ] **Step 1: 写 BadgeDot 失败用例**

Create `frontend/h5/src/components/BadgeDot/__tests__/BadgeDot.test.tsx`:

```tsx
import { render, screen } from '@testing-library/react';
import BadgeDot from '../index';

describe('BadgeDot', () => {
  it('does not render for empty count', () => {
    const { container } = render(<BadgeDot count={0} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders numeric count', () => {
    render(<BadgeDot count={3} />);
    expect(screen.getByText('3')).toBeInTheDocument();
  });

  it('caps count with max suffix', () => {
    render(<BadgeDot count={120} max={99} />);
    expect(screen.getByText('99+')).toBeInTheDocument();
  });

  it('renders dot mode without number text', () => {
    render(<BadgeDot dot ariaLabel="有新提醒" />);
    expect(screen.getByLabelText('有新提醒')).toBeInTheDocument();
    expect(screen.queryByText('1')).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 2: 运行失败测试**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm test -- --runInBand src/components/BadgeDot/__tests__/BadgeDot.test.tsx
```

Expected: FAIL，报错包含 `Cannot find module '../index'`。

- [ ] **Step 3: 核对或补齐 BadgeDot 接口与 Mock hook**

If Task 0 already delivered `BadgeDot`, keep its visual DOM and CSS class names where possible, but ensure this public interface and behavior exist:

`frontend/h5/src/components/BadgeDot/index.tsx`:

```tsx
import styles from './BadgeDot.module.css';

interface BadgeDotProps {
  count?: number;
  max?: number;
  dot?: boolean;
  ariaLabel?: string;
  className?: string;
}

function BadgeDot({
  count = 0,
  max = 99,
  dot = false,
  ariaLabel,
  className = '',
}: BadgeDotProps) {
  if (!dot && count <= 0) {
    return null;
  }

  const displayText = count > max ? `${max}+` : String(count);
  const classes = [styles.badge, dot ? styles.dot : styles.count, className].filter(Boolean).join(' ');

  return (
    <span className={classes} aria-label={ariaLabel || (dot ? '有新提醒' : `${displayText} 条待处理提醒`)}>
      {!dot && displayText}
    </span>
  );
}

export default BadgeDot;
```

If Task 0 did not deliver CSS, use this theme-ready fallback. If Task 0 already delivered a stronger visual design, only verify it keeps the same class names or compatible exported behavior.

`frontend/h5/src/components/BadgeDot/BadgeDot.module.css`:

```css
.badge {
  --touchpoint-danger: var(--color-danger, #ff3b30);
  --touchpoint-border: var(--color-surface, #1a1a1a);
  --touchpoint-badge-text: var(--color-on-danger, #fff);

  position: absolute;
  top: -4px;
  right: -8px;
  min-width: 8px;
  height: 8px;
  border: 1px solid var(--touchpoint-border);
  border-radius: 999px;
  background: var(--touchpoint-danger);
  box-shadow: 0 2px 8px rgba(255, 59, 48, 0.35);
}

.count {
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  color: var(--touchpoint-badge-text);
  font-size: 10px;
  font-weight: 700;
  line-height: 16px;
  text-align: center;
}

.dot {
  width: 8px;
}
```

Create `frontend/h5/src/hooks/useTouchpointNotifications.ts`:

```ts
export interface TouchpointNotifications {
  pendingPayment: number;
  unreadTotal: number;
}

export function useTouchpointNotifications(): TouchpointNotifications {
  return {
    pendingPayment: 1,
    unreadTotal: 3,
  };
}
```

- [ ] **Step 4: 运行 BadgeDot 测试**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm test -- --runInBand src/components/BadgeDot/__tests__/BadgeDot.test.tsx
```

Expected: PASS，4 个用例通过。

- [ ] **Step 5: 提交**

```bash
git add frontend/h5/src/components/BadgeDot frontend/h5/src/hooks/useTouchpointNotifications.ts
git commit -m "feat(h5): add touchpoint badge dot"
```

---

### Task 2: 底部导航与个人中心挂载红点

**Files:**
- Modify: `frontend/h5/src/components/MobileShell/BottomNav.tsx`
- Modify: `frontend/h5/src/components/MobileShell/MobileShell.module.css`
- Modify: `frontend/h5/src/__tests__/components/MobileShell.test.tsx`
- Modify: `frontend/h5/src/pages/User/Index.tsx`
- Modify: `frontend/h5/src/pages/User/Profile.module.css`
- Modify: `frontend/h5/src/pages/User/__tests__/Profile.test.tsx`

- [ ] **Step 1: 写 BottomNav 红点失败用例**

Modify `frontend/h5/src/__tests__/components/MobileShell.test.tsx`，在 `shows retained bottom navigation entries and active route state` 后追加：

```tsx
  it('shows unread total badge on profile nav item', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <BottomNav />
      </MemoryRouter>,
    );

    expect(screen.getByLabelText('3 条待处理提醒')).toHaveTextContent('3');
  });
```

- [ ] **Step 2: 写 Profile 红点失败用例**

Modify `frontend/h5/src/pages/User/__tests__/Profile.test.tsx`，在 `wires retained profile entry buttons and logout` 中 `expect(screen.getByRole('link', { name: /我的竞拍/ }))` 前追加：

```tsx
    expect(screen.getByLabelText('1 条待处理提醒')).toHaveTextContent('1');
```

- [ ] **Step 3: 运行失败测试**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm test -- --runInBand src/__tests__/components/MobileShell.test.tsx src/pages/User/__tests__/Profile.test.tsx
```

Expected: FAIL，缺少 `3 条待处理提醒` 和 `1 条待处理提醒`。

- [ ] **Step 4: BottomNav 挂载徽标**

Modify `frontend/h5/src/components/MobileShell/BottomNav.tsx`:

```tsx
import { Link, useLocation } from 'react-router-dom';
import BadgeDot from '../BadgeDot';
import { useTouchpointNotifications } from '../../hooks/useTouchpointNotifications';
import styles from './MobileShell.module.css';

const hiddenNavPaths = new Set([
  '/detail',
  '/result',
  '/notifications',
  '/following',
  '/history',
  '/login',
]);

const navItems = [
  { path: '/', label: '首页', icon: '⌂' },
  { path: '/live', label: '直播间', icon: '▶' },
  { path: '/profile', label: '我的', icon: '○', badge: true },
];

function isHiddenPath(pathname: string) {
  return hiddenNavPaths.has(pathname);
}

function isActivePath(pathname: string, itemPath: string) {
  return itemPath === '/' ? pathname === '/' : pathname.startsWith(itemPath);
}

function BottomNav() {
  const { pathname } = useLocation();
  const { unreadTotal } = useTouchpointNotifications();

  if (isHiddenPath(pathname)) {
    return null;
  }

  return (
    <nav className={styles.bottomNav} aria-label="底部导航">
      {navItems.map((item) => {
        const isActive = isActivePath(pathname, item.path);

        return (
          <Link
            key={item.path}
            to={item.path}
            className={`${styles.navItem} ${isActive ? styles.navItemActive : ''}`}
            aria-current={isActive ? 'page' : undefined}
          >
            <span className={styles.navIconWrap}>
              <span className={styles.navIcon} aria-hidden="true">
                {item.icon}
              </span>
              {item.badge && <BadgeDot count={unreadTotal} />}
            </span>
            <span>{item.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}

export default BottomNav;
```

Modify `frontend/h5/src/components/MobileShell/MobileShell.module.css`，在 `.navIcon` 前追加：

```css
.navIconWrap {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
```

- [ ] **Step 5: Profile 挂载徽标**

Modify `frontend/h5/src/pages/User/Index.tsx`，增加 imports：

```tsx
import BadgeDot from '../../components/BadgeDot';
import { useTouchpointNotifications } from '../../hooks/useTouchpointNotifications';
```

在组件内部读取数据：

```tsx
  const { pendingPayment } = useTouchpointNotifications();
```

将「我的竞拍」菜单文本替换为：

```tsx
          <span className={styles.menuLabel}>
            我的竞拍
            <BadgeDot count={pendingPayment} className={styles.menuBadge} />
          </span>
```

Modify `frontend/h5/src/pages/User/Profile.module.css`，追加：

```css
.menuLabel {
  position: relative;
  display: inline-flex;
  align-items: center;
}

.menuBadge {
  top: -10px;
  right: -22px;
}
```

- [ ] **Step 6: 运行红点相关测试**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm test -- --runInBand src/components/BadgeDot/__tests__/BadgeDot.test.tsx src/__tests__/components/MobileShell.test.tsx src/pages/User/__tests__/Profile.test.tsx
```

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add frontend/h5/src/components/MobileShell frontend/h5/src/__tests__/components/MobileShell.test.tsx frontend/h5/src/pages/User
git commit -m "feat(h5): show touchpoint badges"
```

---

### Task 3: 接入全局 Toast Provider 行为

**Files:**
- Modify: `frontend/h5/src/components/Toast/index.tsx`
- Create/Modify: `frontend/h5/src/components/Toast/Toast.module.css`
- Create: `frontend/h5/src/components/Toast/__tests__/ToastProvider.test.tsx`

- [ ] **Step 1: 写 ToastProvider 失败用例**

Create `frontend/h5/src/components/Toast/__tests__/ToastProvider.test.tsx`:

```tsx
import { fireEvent, render, screen } from '@testing-library/react';
import { ToastProvider, useToast } from '../index';

function LegacyTrigger() {
  const { showToast } = useToast();
  return <button onClick={() => showToast('旧提示', 'success', 3000)}>legacy</button>;
}

function RichTrigger({ onAction }: { onAction: () => void }) {
  const { showToast } = useToast();
  return (
    <button
      onClick={() =>
        showToast({
          type: 'danger',
          title: '您已被超价',
          message: '当前最高价已更新',
          actionText: '重新出价',
          onAction,
          duration: 3000,
        })
      }
    >
      rich
    </button>
  );
}

function QueueTrigger() {
  const { showToast } = useToast();
  return (
    <button
      onClick={() => {
        showToast({ type: 'info', message: '一', duration: 3000 });
        showToast({ type: 'info', message: '二', duration: 3000 });
        showToast({ type: 'info', message: '三', duration: 3000 });
        showToast({ type: 'info', message: '四', duration: 3000 });
      }}
    >
      queue
    </button>
  );
}

describe('ToastProvider', () => {
  it('keeps legacy showToast signature', () => {
    render(
      <ToastProvider>
        <LegacyTrigger />
      </ToastProvider>,
    );

    fireEvent.click(screen.getByText('legacy'));
    expect(screen.getByRole('status')).toHaveTextContent('旧提示');
  });

  it('renders rich toast and runs action before closing', () => {
    const onAction = jest.fn();

    render(
      <ToastProvider>
        <RichTrigger onAction={onAction} />
      </ToastProvider>,
    );

    fireEvent.click(screen.getByText('rich'));
    expect(screen.getByText('您已被超价')).toBeInTheDocument();
    expect(screen.getByText('当前最高价已更新')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '重新出价' }));
    expect(onAction).toHaveBeenCalledTimes(1);
    expect(screen.queryByText('您已被超价')).not.toBeInTheDocument();
  });

  it('shows at most three toast items at once', () => {
    render(
      <ToastProvider>
        <QueueTrigger />
      </ToastProvider>,
    );

    fireEvent.click(screen.getByText('queue'));
    expect(screen.getAllByRole('status')).toHaveLength(3);
    expect(screen.getByText('一')).toBeInTheDocument();
    expect(screen.getByText('三')).toBeInTheDocument();
    expect(screen.queryByText('四')).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 2: 运行失败测试**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm test -- --runInBand src/components/Toast/__tests__/ToastProvider.test.tsx
```

Expected: FAIL，现有 `showToast` 不支持对象签名与 action。

- [ ] **Step 3: 接入 ToastProvider 行为并保留 UI 设计**

Keep Task 0 visual decisions and theme-ready variables. Replace or merge only the React behavior needed for legacy signature, rich config signature, action close, and max 3 visible items.

Replace `frontend/h5/src/components/Toast/index.tsx` with:

```tsx
import { createContext, useCallback, useContext, useMemo, useState, ReactNode } from 'react';
import styles from './Toast.module.css';

type ToastType = 'success' | 'error' | 'danger' | 'warning' | 'info' | 'loading';

interface ToastConfig {
  type?: ToastType;
  title?: string;
  message: string;
  duration?: number;
  actionText?: string;
  onAction?: () => void;
}

interface ToastMessage extends ToastConfig {
  id: number;
  type: ToastType;
  duration: number;
}

interface ToastContextType {
  showToast: {
    (message: string, type?: ToastType, duration?: number): number;
    (config: ToastConfig): number;
  };
  showLoading: (message: string) => () => void;
}

const ToastContext = createContext<ToastContextType | null>(null);
let toastId = 0;
const MAX_VISIBLE_TOASTS = 3;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  const removeToast = useCallback((id: number) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
  }, []);

  const showToast = useCallback((input: string | ToastConfig, type: ToastType = 'info', duration = 3000) => {
    const config: ToastConfig = typeof input === 'string' ? { message: input, type, duration } : input;
    const id = ++toastId;
    const toast: ToastMessage = {
      id,
      type: config.type || 'info',
      title: config.title,
      message: config.message,
      duration: config.duration ?? duration,
      actionText: config.actionText,
      onAction: config.onAction,
    };

    setToasts((prev) => [...prev, toast]);

    if (toast.duration > 0 && toast.type !== 'loading') {
      window.setTimeout(() => removeToast(id), toast.duration);
    }

    return id;
  }, [removeToast]);

  const showLoading = useCallback((message: string) => {
    const id = showToast({ message, type: 'loading', duration: 0 });
    return () => removeToast(id);
  }, [removeToast, showToast]);

  const visibleToasts = useMemo(() => toasts.slice(0, MAX_VISIBLE_TOASTS), [toasts]);

  return (
    <ToastContext.Provider value={{ showToast, showLoading }}>
      {children}
      <div className={styles.container} aria-live="polite">
        {visibleToasts.map((toast) => (
          <div key={toast.id} className={`${styles.toast} ${styles[toast.type]}`} role="status">
            <span className={styles.icon} aria-hidden="true">{getIcon(toast.type)}</span>
            <span className={styles.body}>
              {toast.title && <strong className={styles.title}>{toast.title}</strong>}
              <span className={styles.message}>{toast.message}</span>
            </span>
            {toast.actionText && (
              <button
                type="button"
                className={styles.action}
                onClick={() => {
                  toast.onAction?.();
                  removeToast(toast.id);
                }}
              >
                {toast.actionText}
              </button>
            )}
            <button
              type="button"
              className={styles.close}
              aria-label="关闭提示"
              onClick={() => removeToast(toast.id)}
            >
              ×
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return context;
}

function getIcon(type: ToastType): string {
  switch (type) {
    case 'success':
      return '✓';
    case 'error':
    case 'danger':
      return '!';
    case 'warning':
      return '⏱';
    case 'loading':
      return '';
    case 'info':
      return 'i';
  }
}

export default ToastProvider;
```

If Task 0 did not deliver CSS, use this theme-ready fallback. If Task 0 delivered CSS, keep its visual design and only ensure these class names exist: `container`、`toast`、`success`、`warning`、`danger`、`error`、`info`、`loading`、`icon`、`body`、`title`、`message`、`action`、`close`.

`frontend/h5/src/components/Toast/Toast.module.css`:

```css
.container {
  --touchpoint-surface: var(--color-surface-elevated, rgba(28, 28, 30, 0.96));
  --touchpoint-text: var(--color-text-primary, #fff);
  --touchpoint-text-muted: var(--color-text-secondary, rgba(255, 255, 255, 0.78));
  --touchpoint-accent: var(--color-accent, #d4af37);
  --touchpoint-warning: var(--color-warning, #f5c542);
  --touchpoint-danger: var(--color-danger, #ff3b30);
  --touchpoint-success: var(--color-success, #d4af37);
  --touchpoint-info: var(--color-info, #64d2ff);
  --touchpoint-shadow: var(--shadow-floating, 0 16px 40px rgba(0, 0, 0, 0.32));

  position: fixed;
  top: calc(16px + env(safe-area-inset-top, 0px));
  left: 50%;
  z-index: 1000;
  width: min(90vw, 388px);
  display: flex;
  flex-direction: column;
  gap: 10px;
  pointer-events: none;
  transform: translateX(-50%);
}

.toast {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 12px 12px 14px;
  border-left: 4px solid rgba(255, 255, 255, 0.2);
  border-radius: 12px;
  background: var(--touchpoint-surface);
  color: var(--touchpoint-text);
  box-shadow: var(--touchpoint-shadow);
  animation: slideDown 180ms ease-out;
  pointer-events: auto;
}

.success {
  border-left-color: var(--touchpoint-success);
}

.warning {
  border-left-color: var(--touchpoint-warning);
}

.danger,
.error {
  border-left-color: var(--touchpoint-danger);
}

.info,
.loading {
  border-left-color: var(--touchpoint-info);
}

.icon {
  width: 20px;
  min-width: 20px;
  height: 20px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.12);
  font-size: 12px;
  font-weight: 700;
  line-height: 20px;
  text-align: center;
}

.body {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.title {
  font-size: 13px;
  line-height: 1.3;
}

.message {
  color: var(--touchpoint-text-muted);
  font-size: 12px;
  line-height: 1.4;
}

.action,
.close {
  border: 0;
  color: var(--touchpoint-accent);
  background: transparent;
  font: inherit;
  cursor: pointer;
}

.action {
  min-height: 32px;
  padding: 0 4px;
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.close {
  width: 28px;
  height: 28px;
  color: rgba(255, 255, 255, 0.6);
  font-size: 18px;
  line-height: 28px;
}

@keyframes slideDown {
  from {
    opacity: 0;
    transform: translateY(-12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
```

- [ ] **Step 4: 运行 ToastProvider 测试**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm test -- --runInBand src/components/Toast/__tests__/ToastProvider.test.tsx
```

Expected: PASS。

- [ ] **Step 5: 运行 API 错误提示兼容构建**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm run build
```

Expected: PASS，确认 `services/api.ts` 旧签名注入仍可通过 TypeScript。

- [ ] **Step 6: 提交**

```bash
git add frontend/h5/src/components/Toast
git commit -m "feat(h5): support rich top toast"
```

---

### Task 4: 复用现有开播弹窗并实现重新登录后一次性触发

**Boundary:**
- 复用现有 `frontend/h5/src/components/LiveReminderModal/` 前端界面，不重做弹窗 DOM 结构和视觉样式。
- 本任务只负责三件事：登录成功写入触发标记、`MobileContainer` 读取标记并挂载现有弹窗、给现有弹窗补轻量无障碍属性和移除外链兜底图。
- 不新增新的弹窗组件，不迁移样式，不改「稍后再看 / 立即前往」交互语义。

**Files:**
- Modify: `frontend/h5/src/store/authContext.tsx`
- Modify: `frontend/h5/src/components/MobileShell/MobileContainer.tsx`
- Modify: `frontend/h5/src/__tests__/components/MobileShell.test.tsx`
- Modify: `frontend/h5/src/components/LiveReminderModal/index.tsx`

- [ ] **Step 1: 写 MobileContainer 弹窗失败用例**

Modify `frontend/h5/src/__tests__/components/MobileShell.test.tsx`，追加：

```tsx
  it('opens live reminder once when pending login marker exists', () => {
    localStorage.setItem('pending_live_reminder', '1');

    render(
      <MemoryRouter>
        <MobileContainer>
          <main>页面内容</main>
        </MobileContainer>
      </MemoryRouter>,
    );

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('直播开播提醒')).toBeInTheDocument();
    expect(localStorage.getItem('pending_live_reminder')).toBeNull();
  });

  it('does not open live reminder without pending login marker', () => {
    localStorage.removeItem('pending_live_reminder');

    render(
      <MemoryRouter>
        <MobileContainer>
          <main>页面内容</main>
        </MobileContainer>
      </MemoryRouter>,
    );

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
```

Add cleanup to `afterEach`:

```tsx
    localStorage.clear();
```

- [ ] **Step 2: 运行失败测试**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm test -- --runInBand src/__tests__/components/MobileShell.test.tsx
```

Expected: FAIL，找不到 `role="dialog"`。

- [ ] **Step 3: 登录成功写入标记**

Modify `frontend/h5/src/store/authContext.tsx`，在 `login` 成功分支中 `setToken(result.token);` 前追加：

```tsx
      localStorage.setItem('pending_live_reminder', '1');
```

完整 login 函数保持为：

```tsx
  const login = async (req: LoginRequest) => {
    try {
      const result = await authService.login(req);
      localStorage.setItem('pending_live_reminder', '1');
      setToken(result.token);
      setUser(result.user);
      setIsAuthenticated(true);
    } catch (error) {
      setIsAuthenticated(false);
      setUser(null);
      setToken(null);
      throw error;
    }
  };
```

- [ ] **Step 4: 在现有弹窗上做轻量增强**

保留 `frontend/h5/src/components/LiveReminderModal/index.tsx` 的现有 UI、按钮和动画，只做以下最小修改：

Modify `frontend/h5/src/components/LiveReminderModal/index.tsx`：

```tsx
      <div
        className={`${styles.modal} ${!isOpen ? styles.slideDown : ''}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby="live-reminder-title"
        onClick={e => e.stopPropagation()}
      >
```

Modify title:

```tsx
          <h3 id="live-reminder-title" className={styles.title}>直播开播提醒</h3>
```

Modify image src:

```tsx
              src={stream.avatarUrl}
```

Do not modify `frontend/h5/src/components/LiveReminderModal/LiveReminderModal.module.css` in this task.

- [ ] **Step 5: MobileContainer 挂载现有弹窗**

Replace `frontend/h5/src/components/MobileShell/MobileContainer.tsx`:

```tsx
import { ReactNode, useEffect, useState } from 'react';
import LiveReminderModal, { StreamInfo } from '../LiveReminderModal';
import BottomNav from './BottomNav';
import styles from './MobileShell.module.css';

interface MobileContainerProps {
  children: ReactNode;
}

const mockLiveReminderStream: StreamInfo = {
  id: 'mock-live-reminder',
  name: '云端珍藏直播间',
  avatarUrl: 'data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 width=%22120%22 height=%22120%22 viewBox=%220 0 120 120%22%3E%3Crect width=%22120%22 height=%22120%22 rx=%2232%22 fill=%22%2327272a%22/%3E%3Ccircle cx=%2260%22 cy=%2252%22 r=%2222%22 fill=%22%23d4af37%22/%3E%3Cpath d=%22M28 104c5-20 18-30 32-30s27 10 32 30%22 fill=%22%23f5f0e8%22/%3E%3C/svg%3E',
  statusText: '正在直播',
};

function MobileContainer({ children }: MobileContainerProps) {
  const [isReminderOpen, setIsReminderOpen] = useState(false);

  useEffect(() => {
    if (localStorage.getItem('pending_live_reminder') !== '1') {
      return;
    }

    localStorage.removeItem('pending_live_reminder');
    setIsReminderOpen(true);
  }, []);

  return (
    <div className={styles.shell} data-testid="mobile-shell">
      <div className={styles.viewport}>
        <div className={styles.content}>{children}</div>
        <BottomNav />
        <LiveReminderModal
          isOpen={isReminderOpen}
          onClose={() => setIsReminderOpen(false)}
          stream={mockLiveReminderStream}
        />
      </div>
    </div>
  );
}

export default MobileContainer;
```

- [ ] **Step 6: 运行弹窗测试**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm test -- --runInBand src/__tests__/components/MobileShell.test.tsx
```

Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add frontend/h5/src/store/authContext.tsx frontend/h5/src/components/MobileShell/MobileContainer.tsx frontend/h5/src/components/LiveReminderModal/index.tsx frontend/h5/src/__tests__/components/MobileShell.test.tsx
git commit -m "feat(h5): show live reminder after login"
```

---

### Task 5: 开发环境 Toast Demo 触发器与最终验证

**Files:**
- Modify: `frontend/h5/src/pages/Live/index.tsx`
- Modify: `frontend/h5/src/pages/Live/Live.module.css`

- [ ] **Step 1: 在 Live 页面引入全局 Toast**

Modify `frontend/h5/src/pages/Live/index.tsx`，增加 import：

```tsx
import { useToast } from '../../components/Toast';
```

在组件内部加入：

```tsx
  const { showToast: showGlobalToast } = useToast();
```

- [ ] **Step 2: 添加开发环境触发器渲染**

在 Live 页面 JSX 中靠近页面主体顶部或底部安全位置加入：

```tsx
      {import.meta.env.DEV && (
        <div className={styles.toastDemoPanel} aria-label="触达 Toast 测试">
          <button
            type="button"
            onClick={() =>
              showGlobalToast({
                type: 'warning',
                title: '截拍预警',
                message: '当前拍品即将截拍，请及时确认出价',
                duration: 3000,
              })
            }
          >
            截拍预警
          </button>
          <button
            type="button"
            onClick={() =>
              showGlobalToast({
                type: 'danger',
                title: '您已被超价',
                message: '当前最高价已更新，可立即重新出价',
                actionText: '重新出价',
                onAction: () => window.scrollTo({ top: document.body.scrollHeight, behavior: 'smooth' }),
                duration: 3000,
              })
            }
          >
            被超价
          </button>
          <button
            type="button"
            onClick={() =>
              showGlobalToast({
                type: 'success',
                title: '恭喜中标',
                message: '您已成功拍下当前拍品，请尽快支付',
                actionText: '去支付',
                onAction: () => window.location.assign('/result'),
                duration: 3000,
              })
            }
          >
            中标结果
          </button>
        </div>
      )}
```

Modify `frontend/h5/src/pages/Live/Live.module.css`，追加：

```css
.toastDemoPanel {
  position: fixed;
  right: 16px;
  bottom: calc(88px + var(--safe-bottom));
  z-index: 30;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.toastDemoPanel button {
  min-height: 36px;
  padding: 0 12px;
  border: 1px solid rgba(212, 175, 55, 0.45);
  border-radius: 999px;
  color: #f5f0e8;
  background: rgba(26, 26, 26, 0.88);
  font-size: 12px;
}
```

- [ ] **Step 3: 运行目标单测**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm test -- --runInBand src/components/BadgeDot/__tests__/BadgeDot.test.tsx src/__tests__/components/MobileShell.test.tsx src/pages/User/__tests__/Profile.test.tsx src/components/Toast/__tests__/ToastProvider.test.tsx
```

Expected: PASS。

- [ ] **Step 4: 运行完整构建**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm run build
```

Expected: PASS。

- [ ] **Step 5: 人工验收**

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm run dev
```

Expected:
- 首页底部「我的」显示数字 `3`。
- 进入 `/profile` 后「我的竞拍」显示数字 `1`。
- 登录后出现「直播开播提醒」弹窗；关闭后刷新不再出现。
- 进入 `/live` 后开发环境测试按钮能触发截拍预警、被超价、中标结果三类顶部 Toast。

- [ ] **Step 6: 提交**

```bash
git add frontend/h5/src/pages/Live
git commit -m "feat(h5): add touchpoint toast demo triggers"
```

---

## Final Verification

Run:

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm test -- --runInBand src/components/BadgeDot/__tests__/BadgeDot.test.tsx src/__tests__/components/MobileShell.test.tsx src/pages/User/__tests__/Profile.test.tsx src/components/Toast/__tests__/ToastProvider.test.tsx
npm run build
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc
python3 - <<'PY'
from pathlib import Path
for root in [Path("frontend/h5/src/components/BadgeDot"), Path("frontend/h5/src/components/Toast")]:
    for path in root.rglob("*"):
        if path.is_file() and path.suffix in {".css", ".tsx"}:
            for line in path.read_text().splitlines():
                if "touchpoint-" in line:
                    print(f"{path}: {line.strip()}")
PY
```

Expected:
- Jest: all listed suites PASS。
- Build: `tsc && vite build` exits with code 0。
- Theme readiness: grep output contains local `touchpoint-` CSS variables for BadgeDot and Toast。

---

## Self-Review

- Spec coverage: 先 UI 设计、日间/夜间 theme-ready、红点、Mock 数据、顶部 Toast、旧签名兼容、新对象签名、重新登录弹窗、开发环境 Demo、自动化测试与人工验收均有对应任务。
- Placeholder scan: 本计划不包含未决占位、未定义接口或延后实现项。
- Type consistency: `showToast` 旧签名和对象签名在 Task 3 定义，Task 5 只使用对象签名；`BadgeDot` 的 `count/max/dot/className` 在 Task 0/1 定义，Task 2 复用同一接口。

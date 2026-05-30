# 用户触达体系（一期） Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `frontend/h5` 落地一期用户触达闭环：红点徽标、顶部 Toast、重新登录后一次性开播弹窗。

**Architecture:** 采用适配当前仓库的最小改动方案：新增纯展示 `BadgeDot` 和 Mock hook，复用并升级现有 `components/Toast`，在 `MobileContainer` 挂载 `LiveReminderModal`。触达数据、Toast 展示入口、登录弹窗标记各自保持单一事实源，避免新增并行 Toast 体系。

**Tech Stack:** React 18、TypeScript、CSS Modules、React Router、Jest、Testing Library、Vite。

---

## File Structure

- Create: `frontend/h5/src/components/BadgeDot/index.tsx`
- Create: `frontend/h5/src/components/BadgeDot/BadgeDot.module.css`
- Create: `frontend/h5/src/components/BadgeDot/__tests__/BadgeDot.test.tsx`
- Create: `frontend/h5/src/hooks/useTouchpointNotifications.ts`
- Create: `frontend/h5/src/components/Toast/Toast.module.css`
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

### Task 1: BadgeDot 与 Mock 数据源

**Files:**
- Create: `frontend/h5/src/components/BadgeDot/index.tsx`
- Create: `frontend/h5/src/components/BadgeDot/BadgeDot.module.css`
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

- [ ] **Step 3: 实现 BadgeDot**

Create `frontend/h5/src/components/BadgeDot/index.tsx`:

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

Create `frontend/h5/src/components/BadgeDot/BadgeDot.module.css`:

```css
.badge {
  position: absolute;
  top: -4px;
  right: -8px;
  min-width: 8px;
  height: 8px;
  border: 1px solid #1a1a1a;
  border-radius: 999px;
  background: #ff3b30;
  box-shadow: 0 2px 8px rgba(255, 59, 48, 0.35);
}

.count {
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  color: #fff;
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

### Task 3: 升级全局 Toast Provider

**Files:**
- Modify: `frontend/h5/src/components/Toast/index.tsx`
- Create: `frontend/h5/src/components/Toast/Toast.module.css`
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

- [ ] **Step 3: 改造 ToastProvider**

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

Create `frontend/h5/src/components/Toast/Toast.module.css`:

```css
.container {
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
  background: rgba(28, 28, 30, 0.96);
  color: #fff;
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.32);
  animation: slideDown 180ms ease-out;
  pointer-events: auto;
}

.success {
  border-left-color: #d4af37;
}

.warning {
  border-left-color: #f5c542;
}

.danger,
.error {
  border-left-color: #ff3b30;
}

.info,
.loading {
  border-left-color: #64d2ff;
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
  color: rgba(255, 255, 255, 0.78);
  font-size: 12px;
  line-height: 1.4;
}

.action,
.close {
  border: 0;
  color: #d4af37;
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

### Task 4: 重新登录后一次性弹窗

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

- [ ] **Step 4: 给弹窗补 role 与去外链占位图**

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

- [ ] **Step 5: MobileContainer 挂载弹窗**

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
```

Expected:
- Jest: all listed suites PASS。
- Build: `tsc && vite build` exits with code 0。

---

## Self-Review

- Spec coverage: 红点、Mock 数据、顶部 Toast、旧签名兼容、新对象签名、重新登录弹窗、开发环境 Demo、自动化测试与人工验收均有对应任务。
- Placeholder scan: 本计划不包含未决占位、未定义接口或延后实现项。
- Type consistency: `showToast` 旧签名和对象签名在 Task 3 定义，Task 5 只使用对象签名；`BadgeDot` 的 `count/max/dot/className` 在 Task 1 定义，Task 2 复用同一接口。

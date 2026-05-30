# H5 日/夜主题一键切换 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `frontend/h5` 上线由 `<html data-theme>` 驱动的日/夜主题一键切换，默认夜间，FOUC-free，并把核心语义颜色 token 化以让切换真正生效。

**Architecture:** 新增 `ThemeContext` 维护 `'dark' | 'light'` 状态并写入 `<html data-theme>` 与 `localStorage`；在 `index.html <head>` 内嵌 inline script 预设主题避免首屏闪白；新增 `ThemeToggle` 浮层按钮由 `MobileShell` 统一渲染；重写 `tokens/colors.css` 提供 dark/light 双套语义变量并 token 化 `MobileShell` 与各 `page.module.css` 最外层容器。

**Tech Stack:** React 18 + TypeScript + Vite + CSS Modules + Jest + Testing Library。设计文档参见 [`docs/superpowers/specs/2026-05-30-h5-theme-toggle-design.md`](../specs/2026-05-30-h5-theme-toggle-design.md)。

---

## 文件结构

| 文件 | 类型 | 职责 |
| --- | --- | --- |
| `frontend/h5/src/styles/tokens/colors.css` | 修改 | 在文末追加 `[data-theme="dark"]` / `[data-theme="light"]` 双套语义变量 |
| `frontend/h5/src/styles/base/globals.css` | 修改 | `color-scheme` 改为 `light dark`，`body` 颜色改读 token |
| `frontend/h5/index.html` | 修改 | `<head>` 内嵌 inline script 预设 `data-theme` |
| `frontend/h5/src/store/themeContext.tsx` | 新建 | `ThemeProvider` + `useTheme` hook |
| `frontend/h5/src/store/__tests__/themeContext.test.tsx` | 新建 | `ThemeContext` 单元测试 |
| `frontend/h5/src/components/ThemeToggle/index.tsx` | 新建 | 浮层切换按钮组件 |
| `frontend/h5/src/components/ThemeToggle/ThemeToggle.module.css` | 新建 | 按钮样式 |
| `frontend/h5/src/__tests__/components/ThemeToggle.test.tsx` | 新建 | 组件单元测试 |
| `frontend/h5/src/components/MobileShell/MobileContainer.tsx` | 修改 | 渲染 `<ThemeToggle />` |
| `frontend/h5/src/components/MobileShell/MobileShell.module.css` | 修改 | 颜色 token 化 |
| `frontend/h5/src/App.tsx` | 修改 | 在 `AuthProvider` 同层包裹 `ThemeProvider` |
| 9 个 `pages/*/*.module.css` | 修改 | 最外层 `.page`/`.header` 颜色 token 化 |
| `frontend/h5/src/components/shared/Card.module.css` | 修改 | 默认背景与边框 token 化 |
| `frontend/h5/src/components/shared/Toast.module.css` | 修改 | 默认背景与文字 token 化 |

---

## Task 1：建立双套语义 tokens 与 globals 配色基线

**Files:**
- Modify: `frontend/h5/src/styles/tokens/colors.css`（在文件末尾追加）
- Modify: `frontend/h5/src/styles/base/globals.css:11-23`

> 本任务是纯样式基础设施搭建，无 React 测试可写；通过下一任务的端到端集成测试间接覆盖。

- [ ] **Step 1：在 `colors.css` 末尾追加双套语义变量**

打开 [colors.css](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/styles/tokens/colors.css)，在文件末尾（`}` 之后）新增以下内容：

```css

/* ===== 主题语义 tokens（日/夜） ===== */
/* 切换由 <html data-theme="dark" | "light"> 控制 */

:root[data-theme="dark"] {
  --bg-page: #1a1a1a;
  --bg-surface: #262626;
  --bg-elevated: rgba(44, 44, 44, 0.78);
  --text-primary: #f5f0e8;
  --text-secondary: #a09888;
  --text-brand: #c9a96e;
  --border-subtle: rgba(255, 255, 255, 0.08);
  --shadow-key: 0 8px 24px rgba(0, 0, 0, 0.40);
}

:root[data-theme="light"] {
  --bg-page: #faf7f2;
  --bg-surface: #ffffff;
  --bg-elevated: rgba(255, 255, 255, 0.92);
  --text-primary: #2a2520;
  --text-secondary: #6b6358;
  --text-brand: #8a6a2a;
  --border-subtle: rgba(0, 0, 0, 0.08);
  --shadow-key: 0 8px 24px rgba(0, 0, 0, 0.08);
}

/* 兜底：未设置 data-theme 时按 dark 渲染 */
:root:not([data-theme]) {
  --bg-page: #1a1a1a;
  --bg-surface: #262626;
  --bg-elevated: rgba(44, 44, 44, 0.78);
  --text-primary: #f5f0e8;
  --text-secondary: #a09888;
  --text-brand: #c9a96e;
  --border-subtle: rgba(255, 255, 255, 0.08);
  --shadow-key: 0 8px 24px rgba(0, 0, 0, 0.40);
}
```

- [ ] **Step 2：调整 `globals.css` 让 body 走主题 token 且不再硬锁 light**

打开 [globals.css](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/styles/base/globals.css)，把第 11-13 行的 `:root` 块替换为：

```css
/* ===== 根元素 ===== */
:root {
  color-scheme: light dark;
}
```

并将第 16-24 行的 `body` 块的 `background-color` 与 `color` 行确认为读 token（已是 `var(--text-primary)` / `var(--bg-primary)`，需把 `var(--bg-primary)` 改为 `var(--bg-page)`）：

```css
body {
  font-family: var(--font-family-base);
  font-size: var(--font-size-base);
  line-height: var(--line-height-normal);
  color: var(--text-primary);
  background-color: var(--bg-page);
  min-height: 100vh;
  min-height: 100dvh;
}
```

- [ ] **Step 3：本地手工验证**

无需运行命令；下一任务通过 `index.html` inline script 注入 `data-theme="dark"` 后，将以下 dev 命令验证：

```bash
cd frontend/h5 && pnpm dev
```

预期：页面加载后 `<html data-theme="dark">`，背景仍为 `#1a1a1a`。本步骤先跳过实际验证，等 Task 3 完成后回头观察。

- [ ] **Step 4：提交**

```bash
git add frontend/h5/src/styles/tokens/colors.css frontend/h5/src/styles/base/globals.css
git commit -m "feat(h5/theme): add dark/light semantic tokens and unlock color-scheme"
```

---

## Task 2：实现 `themeContext` 并做 TDD 单元测试

**Files:**
- Create: `frontend/h5/src/store/themeContext.tsx`
- Test: `frontend/h5/src/store/__tests__/themeContext.test.tsx`

- [ ] **Step 1：写失败测试**

新建 [themeContext.test.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/store/__tests__/themeContext.test.tsx)：

```tsx
import { renderHook, act } from '@testing-library/react';
import { ThemeProvider, useTheme } from '../themeContext';
import { ReactNode } from 'react';

const wrapper = ({ children }: { children: ReactNode }) => (
  <ThemeProvider>{children}</ThemeProvider>
);

describe('themeContext', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  it('默认初始化为 dark 并写入 DOM', () => {
    const { result } = renderHook(() => useTheme(), { wrapper });
    expect(result.current.theme).toBe('dark');
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
  });

  it('localStorage 优先级高于默认值', () => {
    localStorage.setItem('h5.theme', 'light');
    const { result } = renderHook(() => useTheme(), { wrapper });
    expect(result.current.theme).toBe('light');
    expect(document.documentElement.getAttribute('data-theme')).toBe('light');
  });

  it('非法持久化值回落 dark', () => {
    localStorage.setItem('h5.theme', 'neon');
    const { result } = renderHook(() => useTheme(), { wrapper });
    expect(result.current.theme).toBe('dark');
  });

  it('toggle 在 dark/light 间切换并同步 DOM 与 storage', () => {
    const { result } = renderHook(() => useTheme(), { wrapper });
    act(() => result.current.toggle());
    expect(result.current.theme).toBe('light');
    expect(document.documentElement.getAttribute('data-theme')).toBe('light');
    expect(localStorage.getItem('h5.theme')).toBe('light');

    act(() => result.current.toggle());
    expect(result.current.theme).toBe('dark');
    expect(localStorage.getItem('h5.theme')).toBe('dark');
  });

  it('setTheme 直接覆盖', () => {
    const { result } = renderHook(() => useTheme(), { wrapper });
    act(() => result.current.setTheme('light'));
    expect(result.current.theme).toBe('light');
  });

  it('useTheme 在 Provider 之外应抛错', () => {
    expect(() => renderHook(() => useTheme())).toThrow(
      /useTheme must be used within a ThemeProvider/,
    );
  });
});
```

- [ ] **Step 2：运行测试确认失败**

```bash
cd frontend/h5 && npx jest src/store/__tests__/themeContext.test.tsx
```

Expected: FAIL，错误为 `Cannot find module '../themeContext'`。

- [ ] **Step 3：实现最小代码让测试通过**

新建 [themeContext.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/store/themeContext.tsx)：

```tsx
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  ReactNode,
} from 'react';

export type Theme = 'dark' | 'light';

interface ThemeContextValue {
  theme: Theme;
  toggle: () => void;
  setTheme: (t: Theme) => void;
}

const STORAGE_KEY = 'h5.theme';
const DEFAULT_THEME: Theme = 'dark';

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

function readInitialTheme(): Theme {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === 'dark' || stored === 'light') return stored;
  } catch {
    /* 隐私模式或 SSR：忽略 */
  }
  return DEFAULT_THEME;
}

function applyDom(theme: Theme) {
  try {
    document.documentElement.setAttribute('data-theme', theme);
  } catch (err) {
    // eslint-disable-next-line no-console
    console.warn('[theme] failed to write data-theme', err);
  }
}

function persist(theme: Theme) {
  try {
    localStorage.setItem(STORAGE_KEY, theme);
  } catch {
    /* 隐私模式：忽略 */
  }
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(() => readInitialTheme());

  // 首次挂载将状态同步到 DOM（兼容 inline script 缺失或被禁用的情况）
  useEffect(() => {
    applyDom(theme);
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const setTheme = useCallback((next: Theme) => {
    setThemeState(next);
    applyDom(next);
    persist(next);
  }, []);

  const toggle = useCallback(() => {
    setThemeState((prev) => {
      const next: Theme = prev === 'dark' ? 'light' : 'dark';
      applyDom(next);
      persist(next);
      return next;
    });
  }, []);

  const value = useMemo<ThemeContextValue>(
    () => ({ theme, toggle, setTheme }),
    [theme, toggle, setTheme],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return ctx;
}
```

- [ ] **Step 4：运行测试确认通过**

```bash
cd frontend/h5 && npx jest src/store/__tests__/themeContext.test.tsx
```

Expected: PASS（7 个用例）。

- [ ] **Step 5：提交**

```bash
git add frontend/h5/src/store/themeContext.tsx frontend/h5/src/store/__tests__/themeContext.test.tsx
git commit -m "feat(h5/theme): add ThemeProvider with localStorage persistence"
```

---

## Task 3：在 `index.html` 注入 FOUC-free 引导脚本，并在 `App.tsx` 包裹 Provider

**Files:**
- Modify: `frontend/h5/index.html:3-8`
- Modify: `frontend/h5/src/App.tsx:64-112`

- [ ] **Step 1：在 `index.html <head>` 内嵌 inline script**

打开 [index.html](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/index.html)，在 `<head>` 末尾、`</head>` 之前插入：

```html
    <script>
      (function () {
        try {
          var stored = localStorage.getItem('h5.theme');
          var theme = stored === 'light' || stored === 'dark' ? stored : 'dark';
          document.documentElement.setAttribute('data-theme', theme);
        } catch (e) {
          document.documentElement.setAttribute('data-theme', 'dark');
        }
      })();
    </script>
```

完整 `<head>` 应为：

```html
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>直播竞拍</title>
    <script>
      (function () {
        try {
          var stored = localStorage.getItem('h5.theme');
          var theme = stored === 'light' || stored === 'dark' ? stored : 'dark';
          document.documentElement.setAttribute('data-theme', theme);
        } catch (e) {
          document.documentElement.setAttribute('data-theme', 'dark');
        }
      })();
    </script>
  </head>
```

- [ ] **Step 2：在 `App.tsx` 顶部 import 并包裹 `ThemeProvider`**

打开 [App.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/App.tsx)，在第 5 行下方加入：

```tsx
import { ThemeProvider } from './store/themeContext'
```

把 `App` 函数中 `<ErrorBoundary>` 紧内层从 `<ToastProvider>` 改为 `<ThemeProvider><ToastProvider>...</ToastProvider></ThemeProvider>`。完整 return 应为：

```tsx
function App() {
  return (
    <ErrorBoundary>
      <ThemeProvider>
        <ToastProvider>
          <ToastInitializer />
          <AuthProvider>
            <ErrorMonitorInitializer />
            <GrowthBookContextProvider>
              <AuctionProvider>
                <MobileContainer>
                  <Suspense fallback={<LoadingSpinner />}>
                    <Routes>
                      <Route path="/login" element={<Login />} />
                      <Route path="/" element={<Home />} />
                      <Route path="/live" element={<Live />} />
                      <Route path="/detail" element={<ProductDetail />} />
                      <Route path="/auction/:id" element={<LegacyAuctionRedirect />} />
                      <Route path="/result" element={<Result />} />
                      <Route path="/result/:id" element={<LegacyResultRedirect />} />
                      <Route path="/profile" element={
                        <PrivateRoute>
                          <Profile />
                        </PrivateRoute>
                      } />
                      <Route path="/notifications" element={
                        <PrivateRoute>
                          <Notifications />
                        </PrivateRoute>
                      } />
                      <Route path="/following" element={
                        <PrivateRoute>
                          <Follow />
                        </PrivateRoute>
                      } />
                      <Route path="/follow" element={<Navigate to="/following" replace />} />
                      <Route path="/history" element={
                        <PrivateRoute>
                          <History />
                        </PrivateRoute>
                      } />
                    </Routes>
                  </Suspense>
                </MobileContainer>
              </AuctionProvider>
            </GrowthBookContextProvider>
          </AuthProvider>
        </ToastProvider>
      </ThemeProvider>
    </ErrorBoundary>
  )
}
```

- [ ] **Step 3：本地启动验证（可选）**

```bash
cd frontend/h5 && pnpm dev
```

Expected：
- 首屏背景为 `#1a1a1a`，无白闪。
- DevTools Console 输入 `document.documentElement.dataset.theme` 返回 `'dark'`。
- 输入 `localStorage.setItem('h5.theme','light'); location.reload()` 后 `<html data-theme="light">`，但页面颜色暂时不变（待 Task 5/6 改 CSS 后才生效）。

- [ ] **Step 4：提交**

```bash
git add frontend/h5/index.html frontend/h5/src/App.tsx
git commit -m "feat(h5/theme): bootstrap data-theme before React hydration to avoid FOUC"
```

---

## Task 4：实现 `ThemeToggle` 组件（TDD）

**Files:**
- Create: `frontend/h5/src/components/ThemeToggle/index.tsx`
- Create: `frontend/h5/src/components/ThemeToggle/ThemeToggle.module.css`
- Test: `frontend/h5/src/__tests__/components/ThemeToggle.test.tsx`

- [ ] **Step 1：写失败测试**

新建 [ThemeToggle.test.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/__tests__/components/ThemeToggle.test.tsx)：

```tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { ThemeProvider } from '../../store/themeContext';
import ThemeToggle from '../../components/ThemeToggle';

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>
    </MemoryRouter>,
  );
}

describe('ThemeToggle', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  it('默认 dark 时按钮 aria-label 提示切换为亮色', () => {
    renderAt('/');
    expect(screen.getByRole('button', { name: /切换为亮色模式/ })).toBeInTheDocument();
  });

  it('点击后 data-theme 翻转且 aria-label 更新', () => {
    renderAt('/');
    fireEvent.click(screen.getByRole('button', { name: /切换为亮色模式/ }));
    expect(document.documentElement.getAttribute('data-theme')).toBe('light');
    expect(screen.getByRole('button', { name: /切换为夜间模式/ })).toBeInTheDocument();
  });

  it('在 /login 路径下不渲染', () => {
    renderAt('/login');
    expect(screen.queryByRole('button')).toBeNull();
  });
});
```

- [ ] **Step 2：运行测试确认失败**

```bash
cd frontend/h5 && npx jest src/__tests__/components/ThemeToggle.test.tsx
```

Expected: FAIL，错误为 `Cannot find module '../../components/ThemeToggle'`。

- [ ] **Step 3：创建 CSS Module**

新建 [ThemeToggle.module.css](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/ThemeToggle/ThemeToggle.module.css)：

```css
.toggle {
  position: fixed;
  top: calc(env(safe-area-inset-top, 0px) + 12px);
  right: 12px;
  z-index: 100;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  background: var(--bg-elevated);
  color: var(--text-brand);
  font-size: 20px;
  line-height: 1;
  cursor: pointer;
  backdrop-filter: blur(12px);
  box-shadow: var(--shadow-key);
  transition: transform 250ms ease, color 200ms ease, background 200ms ease;
}

.toggle:active {
  transform: scale(0.92);
}

.icon {
  display: inline-block;
  transition: transform 250ms ease;
}

.iconRotated {
  transform: rotate(180deg);
}

@media (min-width: 431px) {
  .toggle {
    /* 在 PC 居中预览模式下，跟随 viewport 边缘 */
    right: max(12px, calc((100vw - 430px) / 2 + 12px));
  }
}
```

- [ ] **Step 4：实现组件**

新建 [ThemeToggle/index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/ThemeToggle/index.tsx)：

```tsx
import { useLocation } from 'react-router-dom';
import { useTheme } from '../../store/themeContext';
import styles from './ThemeToggle.module.css';

const HIDDEN_PATHS = new Set(['/login']);

function ThemeToggle() {
  const { pathname } = useLocation();
  const { theme, toggle } = useTheme();

  if (HIDDEN_PATHS.has(pathname)) {
    return null;
  }

  const isDark = theme === 'dark';
  const ariaLabel = isDark ? '切换为亮色模式' : '切换为夜间模式';
  const icon = isDark ? '☾' : '☀';

  return (
    <button
      type="button"
      className={styles.toggle}
      aria-label={ariaLabel}
      onClick={toggle}
    >
      <span
        className={`${styles.icon} ${isDark ? '' : styles.iconRotated}`}
        aria-hidden="true"
      >
        {icon}
      </span>
    </button>
  );
}

export default ThemeToggle;
```

- [ ] **Step 5：运行测试确认通过**

```bash
cd frontend/h5 && npx jest src/__tests__/components/ThemeToggle.test.tsx
```

Expected: PASS（3 个用例）。

- [ ] **Step 6：提交**

```bash
git add frontend/h5/src/components/ThemeToggle/ frontend/h5/src/__tests__/components/ThemeToggle.test.tsx
git commit -m "feat(h5/theme): add floating ThemeToggle button hidden on /login"
```

---

## Task 5：把 `ThemeToggle` 接入 `MobileShell` 并 token 化其 CSS

**Files:**
- Modify: `frontend/h5/src/components/MobileShell/MobileContainer.tsx`
- Modify: `frontend/h5/src/components/MobileShell/MobileShell.module.css:1-90`

- [ ] **Step 1：在 `MobileContainer` 中渲染 `ThemeToggle`**

打开 [MobileContainer.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/MobileShell/MobileContainer.tsx)，替换为：

```tsx
import { ReactNode } from 'react';
import BottomNav from './BottomNav';
import ThemeToggle from '../ThemeToggle';
import styles from './MobileShell.module.css';

interface MobileContainerProps {
  children: ReactNode;
}

function MobileContainer({ children }: MobileContainerProps) {
  return (
    <div className={styles.shell} data-testid="mobile-shell">
      <div className={styles.viewport}>
        <ThemeToggle />
        <div className={styles.content}>{children}</div>
        <BottomNav />
      </div>
    </div>
  );
}

export default MobileContainer;
```

- [ ] **Step 2：把 `MobileShell.module.css` 颜色 token 化**

打开 [MobileShell.module.css](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/components/MobileShell/MobileShell.module.css)，逐处替换：

```css
.shell {
  min-height: 100vh;
  min-height: 100dvh;
  background: var(--bg-page);
  color: var(--text-primary);
  display: flex;
  justify-content: center;
  font-family: var(--font-body);
}

.viewport {
  position: relative;
  width: 100%;
  min-height: 100vh;
  min-height: 100dvh;
  background: var(--bg-page);
  overflow-x: hidden;
}

.content {
  min-height: 100vh;
  min-height: 100dvh;
  padding-bottom: calc(80px + var(--safe-bottom));
}

.bottomNav {
  position: fixed;
  left: 50%;
  bottom: 0;
  z-index: 20;
  width: 100%;
  max-width: 430px;
  height: calc(64px + var(--safe-bottom));
  padding: 8px 24px calc(8px + var(--safe-bottom));
  display: flex;
  align-items: center;
  justify-content: space-around;
  background: var(--bg-elevated);
  border-top: 1px solid var(--border-subtle);
  backdrop-filter: blur(16px);
  transform: translateX(-50%);
}

.navItem {
  min-width: 64px;
  color: var(--text-secondary);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  line-height: 1;
  transition: color var(--transition-fast);
}

.navItem:hover,
.navItemActive {
  color: var(--text-brand);
}

.navIcon {
  font-size: 22px;
  line-height: 1;
}

@media (min-width: 431px) {
  .shell {
    align-items: center;
    padding: 24px 0;
  }

  .viewport {
    max-width: 430px;
    min-height: min(100vh, 812px);
    border: 1px solid var(--border-subtle);
    border-radius: 32px;
    box-shadow: var(--shadow-key);
  }

  .content {
    min-height: min(100vh, 812px);
  }

  .bottomNav {
    position: absolute;
    left: 0;
    max-width: none;
    transform: none;
  }
}
```

- [ ] **Step 3：本地启动验证**

```bash
cd frontend/h5 && pnpm dev
```

Expected：
- 浮层按钮出现在右上角，点击后 `<html data-theme>` 切换为 `light`，且 `MobileShell` 背景立刻变白、底部导航变浅。
- 各 page 内部仍未切换（待 Task 6）。

- [ ] **Step 4：运行已有的 MobileShell 测试确保没破坏**

```bash
cd frontend/h5 && npx jest src/__tests__/components/MobileShell.test.tsx
```

Expected: PASS。

- [ ] **Step 5：提交**

```bash
git add frontend/h5/src/components/MobileShell/
git commit -m "feat(h5/theme): mount ThemeToggle in MobileShell and tokenize shell colors"
```

---

## Task 6：批量 token 化 9 个 page 最外层容器与 header

**Files:**
- Modify: `frontend/h5/src/pages/Home/Home.module.css:1-17`
- Modify: `frontend/h5/src/pages/Live/Live.module.css`（前 ~60 行 `.page` 与 `.header`）
- Modify: `frontend/h5/src/pages/Auction/Auction.module.css`（前 ~30 行 `.page`）
- Modify: `frontend/h5/src/pages/ProductDetail/ProductDetail.module.css`（前 ~40 行 `.page` 与 `.header`）
- Modify: `frontend/h5/src/pages/Result/Result.module.css`（前 ~40 行 `.page`）
- Modify: `frontend/h5/src/pages/User/Profile.module.css`（前 ~30 行 `.page`/`.hero`）
- Modify: `frontend/h5/src/pages/History/AuctionHistory.module.css`（前 ~40 行 `.page` 与 `.header`）
- Modify: `frontend/h5/src/pages/Follow/Following.module.css`（前 ~40 行 `.page` 与 `.header`）
- Modify: `frontend/h5/src/pages/Notifications/Notifications.module.css`（前 ~30 行 `.page` 与 `.header`）

> 不动 `Login.module.css`（设计已确定登录页保留独立配色）。

判定规则统一为：
- `background: #1a1a1a` / `#121212` / `#262626` 类容器底色 → `var(--bg-page)`
- `background: #2c2c2c` / `#3a3a3a` 类卡片底 → `var(--bg-surface)`
- `background: rgba(26,26,26,0.96)` 类 sticky header → `var(--bg-elevated)`
- `color: #f5f0e8` → `var(--text-primary)`
- `color: #a09888` → `var(--text-secondary)`
- `color: #c9a96e` / `#d4af37` → `var(--text-brand)`
- 仅替换上述清单文件中**最外层 `.page`** 与紧邻的 `.header`/`.hero`/`.topBar` 块；其他装饰色暂不动（设计已澄清）。

- [ ] **Step 1：替换 `Home.module.css` 的 `.page` 与 `.header`（前 17 行）**

打开 [Home.module.css](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Home/Home.module.css)，把第 1-17 行替换为：

```css
.page {
  min-height: 100%;
  background: var(--bg-page);
  color: var(--text-primary);
}

.header {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: calc(var(--spacing-5) + env(safe-area-inset-top, 0px)) var(--spacing-6) var(--spacing-4);
  background: var(--bg-elevated);
  backdrop-filter: blur(16px);
}
```

- [ ] **Step 2：依次处理其余 8 个 `.module.css`**

对每个文件先 `Read` 出最外层 `.page`/`.header` 块，按上一步的"判定规则"用 Edit 工具替换硬编码颜色为对应 token；保持其它行（padding/positioning/transition/shadow 等）不变。

完成后跑下面的简单 grep 验证（应仅剩品牌装饰处的硬编码，不含 `.page`/`.header` 顶层规则）：

```bash
cd frontend/h5 && rg -n 'background: #|color: #' src/pages/*/[A-Z]*.module.css | grep -E '\.page \{|\.header \{' || echo 'OK: no top-level hardcoded colors'
```

Expected: 输出 `OK: no top-level hardcoded colors`（或仅剩 `Login.module.css` 中的硬编码——预期保留）。

- [ ] **Step 3：本地启动验证 4 个核心页**

```bash
cd frontend/h5 && pnpm dev
```

依次访问 `/`、`/live`、`/detail`、`/profile`，分别在 dark/light 下检查：
- 最外层背景与文字颜色随主题切换。
- 卡片/详情区装饰色保持原样（已声明非目标）。
- 主文字对背景对比度大致达标（凭目测）。

- [ ] **Step 4：跑 page 相关单元测试确保没破坏**

```bash
cd frontend/h5 && npx jest src/pages
```

Expected: 已有用例继续 PASS（CSS Module 通过 `identity-obj-proxy` 不会因颜色变化失败）。

- [ ] **Step 5：提交**

```bash
git add frontend/h5/src/pages
git commit -m "feat(h5/theme): tokenize page and header colors across 9 pages"
```

---

## Task 7：token 化共享组件 `Card` 与 `Toast` 默认配色

**Files:**
- Modify: `frontend/h5/src/components/shared/Card.module.css`
- Modify: `frontend/h5/src/components/shared/Toast.module.css`

- [ ] **Step 1：阅读两文件并按规则替换**

```bash
# 仅参考文件位置；实际通过 Read + Edit 工具操作
```

把以下硬编码处替换为 token：
- `Card.module.css` 的默认 `.card { background; color; border; }` → `var(--bg-surface)` / `var(--text-primary)` / `1px solid var(--border-subtle)`
- `Toast.module.css` 的默认 `.toast { background; color; }` → `var(--bg-surface)` / `var(--text-primary)`；保留 success/error/info 变体的语义色（来自功能色 token）

- [ ] **Step 2：跑共享组件单测**

```bash
cd frontend/h5 && npx jest src/__tests__/components/Card.test.tsx src/__tests__/components/Toast.test.tsx
```

Expected: PASS。

- [ ] **Step 3：提交**

```bash
git add frontend/h5/src/components/shared/Card.module.css frontend/h5/src/components/shared/Toast.module.css
git commit -m "feat(h5/theme): tokenize shared Card and Toast surface colors"
```

---

## Task 8：全量回归测试与最终视觉验收

**Files:** 无新增

- [ ] **Step 1：跑全量单测**

```bash
cd frontend/h5 && npx jest
```

Expected: 全部 PASS；如某个 visual / e2e 失败，按设计 §8 风险 B，记录到 follow-up，不在本期修复。

- [ ] **Step 2：跑类型检查**

```bash
cd frontend/h5 && npx tsc --noEmit
```

Expected: 无错误。

- [ ] **Step 3：本地手工验收清单**

启动 `pnpm dev`，按下表逐项核对：

| 验收项 | 预期 |
| --- | --- |
| 首屏 `/` 加载 | 不闪白；`<html data-theme="dark">` |
| 点击右上角 ☾ | 切换至亮色，按钮变 ☀；`localStorage['h5.theme'] === 'light'` |
| 刷新页面 | 仍为亮色 |
| 进入 `/login` | 不显示 ThemeToggle |
| 在 `/`、`/live`、`/detail`、`/profile` 之间切换 | 主题保持，最外层背景/文字一致 |
| 切回 dark | 视觉与重构前几乎一致 |

- [ ] **Step 4：构建产物验证**

```bash
cd frontend/h5 && pnpm build
```

Expected: 构建成功，无报错。

- [ ] **Step 5：合并提交（可选 squash）**

整体功能已分散在前述 7 个 commit 中，无需额外提交。如需汇总为 PR，可在 PR 描述中链接 spec 与 plan：

```
spec: docs/superpowers/specs/2026-05-30-h5-theme-toggle-design.md
plan: docs/superpowers/plans/2026-05-30-h5-theme-toggle.md
```

---

## Self-Review

- ✅ 覆盖 spec §3.1 双套 tokens → Task 1
- ✅ 覆盖 spec §3.2 ThemeContext + 持久化 + 错误兜底 → Task 2
- ✅ 覆盖 spec §3.3 ThemeToggle + `/login` 隐藏 → Task 4
- ✅ 覆盖 spec §3.4 关键 CSS 重构（MobileShell + 9 page + Card/Toast）→ Task 5/6/7
- ✅ 覆盖 spec §4 FOUC-free 引导 → Task 3
- ✅ 覆盖 spec §6 测试 → Task 2/4 单元；Task 8 回归
- ✅ 覆盖 spec §7 验收 → Task 8 手工清单
- ✅ 无 TBD/TODO；类型 `Theme = 'dark' | 'light'`、API `theme/toggle/setTheme` 在 Task 2/4 一致
- ✅ 命名一致：`STORAGE_KEY = 'h5.theme'`、`data-theme` 属性、`useTheme` 钩子
- ✅ 跳过项与设计 §1.4 非目标对齐：不改 admin/test-dashboard、不动 Login 配色、不修复 e2e snapshot

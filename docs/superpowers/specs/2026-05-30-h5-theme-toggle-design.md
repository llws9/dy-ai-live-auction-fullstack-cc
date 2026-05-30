# H5 日/夜主题一键切换 设计文档

- **Author:** brainstorming session
- **Date:** 2026-05-30
- **Scope:** `frontend/h5`（仅用户端 H5，不含 `admin` / `test-dashboard`）
- **Status:** Draft – pending implementation

---

## 1. 背景与第一性目标

### 1.1 现状

- `frontend/h5` 已存在 design tokens 体系（`src/styles/tokens/colors.css`），但定义为亮色，且 `globals.css` 硬性 `color-scheme: light`。
- 实际页面**全部写死为深色**（如 `MobileShell.module.css` 用 `#121212/#f5f0e8`，`Home.module.css` 用 `#1a1a1a/#c9a96e`）。
- 全 `frontend/h5/src` 内硬编码颜色 **504 处，分布在 27 个文件**——design tokens 几乎无人消费。
- 个人中心已有「设置（暂未开放）」入口，可作为视觉锚点。

### 1.2 第一性根因

要做"一键日夜切换"，**真正阻碍不是切换机制本身，而是颜色没有被 token 化**——只要颜色是硬编码的，就没有"开关可拨"。所以本次工作的最大成本会落在**核心语义色 token 化重构**，而不是开关 UI。

### 1.3 目标

让"日/夜"成为**一个 CSS 变量值**，由 React 状态驱动；用户开关时仅切 `<html data-theme>`，不重排不重渲染组件树，并通过 `localStorage` 持久化。

### 1.4 非目标

- 不改造 `frontend/admin` 与 `frontend/test-dashboard`（后续阶段处理）。
- 不做"跟随系统 prefers-color-scheme"模式（默认夜间已能满足主流诉求）。
- 不一次性消化全部 504 处硬编码颜色，仅 token 化核心语义色。
- 不修复因配色变更导致的 e2e snapshot 失效（放入 follow-up）。
- 登录页 `/login` 不暴露主题切换入口，亦不强制兼容亮色。

---

## 2. 关键决策

| 决策点 | 选项 | 决议 |
| --- | --- | --- |
| 覆盖范围 | h5 / h5+admin / 三端 | **仅 h5** |
| 默认取向 | 保留深色为夜间，新增亮色 / 重构为亮色默认 / 跟随系统 | **保留现有深色为夜间，新增亮色** |
| 首次访问默认 | 默认夜间 / 跟随系统 / 默认亮色 | **默认夜间** |
| 入口与 UI | 新建 `/settings` 页 / 个人中心 inline / 顶部快捷图标 | **顶部快捷图标切换** |
| 颜色改造范围 | 全量 token 化 / 只 token 化核心语义色 / override layer | **只 token 化核心语义色** |

---

## 3. 架构

切分为四个独立单元，每个单元有清晰边界与可独立测试的接口。

```
┌──────────────────────────────┐    DOM <html data-theme>    ┌──────────────────┐
│ ThemeContext (Provider/Hook)│ ─────────────────────────▶  │ Theme tokens CSS │
│  - state: 'dark' | 'light'   │                             │  双套语义变量    │
│  - localStorage 持久化       │                             └──────────────────┘
│  - 暴露 useTheme()           │
└──────────────────────────────┘
              ▲                                                       ▲
              │ context                                               │ var(--*)
              │                                                       │
┌──────────────────────────────┐                            ┌──────────────────────────┐
│ ThemeToggle 组件             │                            │ 关键 CSS 重构 (40~50 声明)│
│  - 浮层按钮 ☾ ↔ ☀             │                            │  - MobileShell           │
│  - 250ms 旋转过渡            │                            │  - 9 page 最外层容器     │
│  - 隐藏路径: /login          │                            │  - 公共组件 Card/Button… │
└──────────────────────────────┘                            └──────────────────────────┘
```

### 3.1 单元 1：Theme tokens 双套（CSS）

**职责**：用 `[data-theme="dark"]` / `[data-theme="light"]` 作用域，提供两套语义变量；保留品牌 hue 不随主题切换。

**文件**：重写 `frontend/h5/src/styles/tokens/colors.css` 中的语义层，新增以下变量：

```css
:root[data-theme="dark"] {
  --bg-page: #1a1a1a;
  --bg-surface: #262626;            /* 卡片/导航等浮层 */
  --bg-elevated: rgba(44,44,44,.78);
  --text-primary: #f5f0e8;
  --text-secondary: #a09888;
  --text-brand: #c9a96e;            /* 暗金 */
  --border-subtle: rgba(255,255,255,.08);
  --shadow-key: 0 8px 24px rgba(0,0,0,.40);
}

:root[data-theme="light"] {
  --bg-page: #faf7f2;               /* 象牙白 */
  --bg-surface: #ffffff;
  --bg-elevated: rgba(255,255,255,.92);
  --text-primary: #2a2520;
  --text-secondary: #6b6358;
  --text-brand: #8a6a2a;            /* 深棕金 */
  --border-subtle: rgba(0,0,0,.08);
  --shadow-key: 0 8px 24px rgba(0,0,0,.08);
}
```

**保留不变**：

- `--color-primary-*`（橙色品牌色阶）
- 功能色 `--color-success-* / --color-warning-* / --color-error-* / --color-info-*`
- 排版/间距/圆角/动效 tokens

**移除**：`globals.css` 中 `color-scheme: light;` 改为 `color-scheme: light dark;`，由 `data-theme` 决定实际 scheme。

### 3.2 单元 2：ThemeContext

**职责**：维护当前主题状态、订阅切换、写 DOM、写 localStorage。

**文件**：新建 `frontend/h5/src/store/themeContext.tsx`

**接口**：

```ts
type Theme = 'dark' | 'light';
interface ThemeContextValue {
  theme: Theme;
  toggle: () => void;
  setTheme: (t: Theme) => void;
}
const useTheme: () => ThemeContextValue;
```

**初始化优先级**：`localStorage.getItem('h5.theme')` → 默认 `'dark'`。

**切换实现**：写 `document.documentElement.dataset.theme` + 写 `localStorage`；视觉切换由 CSS 变量重新解析驱动，不需要 React 重渲染整个组件树（仅 `ThemeToggle` 等少量消费者会因 context 更新而重渲染）。

### 3.3 单元 3：ThemeToggle 组件

**职责**：呈现切换按钮、调用 `useTheme().toggle()`。

**文件**：新建 `frontend/h5/src/components/ThemeToggle/index.tsx` + `ThemeToggle.module.css`

**渲染策略**：

- 由 `MobileShell` 渲染（不在每个 page 里手动放）。
- `position: fixed; top: calc(env(safe-area-inset-top, 0px) + 12px); right: 12px; z-index: 100`
- 通过 `useLocation()` 判断是否处于 `/login`，是则不渲染（与 `BottomNav` 隐藏策略对齐，但隐藏集合可独立维护）。
- 图标：`☾`（dark 状态）↔ `☀`（light 状态），切换时旋转 180°，`transition: transform 250ms`。
- 触控目标 ≥ 44×44 px（符合 iOS HIG）。
- `aria-label` 动态："切换为亮色模式" / "切换为夜间模式"。

### 3.4 单元 4：关键 CSS 重构

**职责**：把核心语义颜色由硬编码改为 `var(--*)`，让主题切换真正生效。

**精确改造清单（约 40~50 个声明）**：

- `frontend/h5/src/components/MobileShell/MobileShell.module.css`
  - `.shell` background/color
  - `.viewport` background
  - `.bottomNav` background/border
  - `.navItem` color
  - `.navItemActive` color → `var(--text-brand)`
- 9 个页面 `.module.css` 中**最外层 `.page`** 与 **`.header`** 的 `background/color`：
  - `pages/Home/Home.module.css`
  - `pages/Live/Live.module.css`
  - `pages/Auction/Auction.module.css`
  - `pages/ProductDetail/ProductDetail.module.css`
  - `pages/Result/Result.module.css`
  - `pages/User/Profile.module.css`
  - `pages/History/AuctionHistory.module.css`
  - `pages/Follow/Following.module.css`
  - `pages/Notifications/Notifications.module.css`
  - `pages/Login/Login.module.css`（保留独立配色，不强制使用 token）
- 公共组件：
  - `components/shared/Card.module.css`：背景与边框
  - `components/shared/Button.module.css`：默认态背景与文字（强品牌按钮保留硬编码）
  - `components/shared/Toast.module.css`：背景与文字
- 跳过：`BidButton`、`PriceDisplay`、`Countdown` 等强品牌色组件——保留品牌 hue 硬编码，但文字色尽量改用 `var(--text-primary)`。

**判定规则**：

- 容器/卡片背景 → `var(--bg-page)` 或 `var(--bg-surface)`
- 主文字 → `var(--text-primary)`
- 次要文字 → `var(--text-secondary)`
- 品牌强调（金色字、品牌按钮）→ `var(--text-brand)` 或保留硬编码
- 边框 → `var(--border-subtle)`

**未在本期处理**：装饰渐变、特定阴影、单点强品牌色——保持原状，后续按视觉验收增量替换。

---

## 4. 数据流

```
用户点击 ThemeToggle
        │
        ▼
useTheme().toggle()
        │
        ├─▶ 写 localStorage['h5.theme']
        │
        └─▶ document.documentElement.dataset.theme = next
                    │
                    ▼
            CSS 变量按 [data-theme] 重新解析
                    │
                    ▼
            视觉立即切换（仅 Paint，无 Layout）
```

页面首屏初始化路径：

```
index.html <head> inline script
        │
        ├─▶ 读 localStorage['h5.theme']
        │
        └─▶ document.documentElement.dataset.theme = stored ?? 'dark'
                    │
                    ▼
        浏览器解析 CSS 时已确定主题（不闪白 / FOUC-free）
                    │
                    ▼
        React 挂载 ThemeProvider，state 与 DOM 一致
```

---

## 5. 错误处理

| 失败场景 | 处理 |
| --- | --- |
| `localStorage` 不可用（隐私模式 / iOS Safari quirks） | 兜底使用内存态 + 默认 `'dark'`；不抛错 |
| `data-theme` 写入异常 | `try/catch` 包裹；失败时记录 `console.warn`，不中断 UI |
| 旧用户存在非法持久化值 | 只接受 `'dark'` 或 `'light'`，否则回落默认 |

---

## 6. 测试

| 类型 | 用例 |
| --- | --- |
| 单元测试 | `themeContext` 初始化优先级（localStorage > 默认 dark）、`toggle` 后 DOM 与 storage 同步、非法值回落 |
| 集成测试 | 渲染 `MobileShell` + `ThemeToggle`，点击后 `<html>` 上 `data-theme` 翻转 |
| 视觉验收 | 主页 / 直播 / 详情 / 我的 4 页两套主题截图对比 |
| 可访问性 | 4 页主文字对背景对比度 ≥ WCAG AA（4.5:1） |

**不做**：e2e snapshot 修复（follow-up）。

---

## 7. 验收标准

1. 首屏不闪白（FOUC-free）。
2. 用户切换后刷新页面，主题保留。
3. 主页 / 直播 / 详情 / 我的 4 个核心页两套主题下视觉与文字可读性均达标。
4. `/login` 不暴露切换入口。
5. `themeContext` 单测通过；初始化优先级符合规范。

---

## 8. 风险与决策

- **风险 A**：未 token 化的硬编码颜色（如装饰金 `#c9a96e`、特定阴影）在亮色模式下可能"格格不入"。
  **决策**：核心容器走 token；装饰色暂保留硬编码。验收时若发现单点过丑，再增量替换。
- **风险 B**：现有测试快照失效。
  **决策**：本次不修复 e2e snapshot，作为 follow-up。
- **风险 C**：登录页配色独立。
  **决策**：登录页与本 feature 解耦，不暴露 toggle，不强制兼容亮色。
- **风险 D**：`color-scheme` 改动影响表单原生控件外观。
  **决策**：改为 `color-scheme: light dark`，由 `data-theme` 控制；如个别表单控件不协调，逐个用 token 覆盖。

---

## 9. 实施清单（实现阶段交付物）

1. `frontend/h5/src/styles/tokens/colors.css` —— 新增双套语义变量。
2. `frontend/h5/src/styles/base/globals.css` —— 调整 `color-scheme`、`body` 颜色读 token。
3. `frontend/h5/index.html` —— `<head>` 内嵌 inline script，预设 `data-theme`。
4. `frontend/h5/src/store/themeContext.tsx` —— Provider + `useTheme` hook。
5. `frontend/h5/src/App.tsx` —— 包裹 `ThemeProvider`（位置：`AuthProvider` 同层即可）。
6. `frontend/h5/src/components/ThemeToggle/` —— 组件 + CSS Module。
7. `frontend/h5/src/components/MobileShell/MobileContainer.tsx` —— 渲染 `ThemeToggle`。
8. 关键 CSS 重构（清单见 §3.4）。
9. 单元测试：`store/__tests__/themeContext.test.tsx`。

---

## 10. Follow-ups（不在本期范围）

- 增量 token 化剩余硬编码颜色（剩余 ~450+ 处）。
- 修复因配色变更导致的 e2e / visual snapshot。
- 扩展到 `frontend/admin` 与 `frontend/test-dashboard`。
- 引入"跟随系统"模式（`prefers-color-scheme`）作为第三档。
- 在个人中心 `/profile` 的「设置」入口（当前禁用）落地一个完整的设置页，把主题选择以单选卡片形式列入。

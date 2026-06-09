# H5 Bottom Navigation Shared Indicator Design

## 背景

H5 底部导航当前使用三个入口：首页、直播间、我的。原实现中每个 `navItem` 自身通过 `::before` 生成选中胶囊，选中态只在当前 Tab 内淡入，缺少“从一个入口切换到另一个入口”的空间连续性。

本次 UI 优化选择 `A · 一体滑行` 方案：将选中态的胶囊和顶部金线从单个 Tab 内抽离，改为 `nav` 内唯一的共享选中指示器。切换底部导航时，指示器沿导航栏横向滑到目标入口。

## 目标

- 图标从字符 `⌂/▶/○` 升级为线性 SVG，保持高端拍卖风格。
- 选中态由共享胶囊承载，不再由每个 Tab 独立生成。
- 顶部金线与胶囊同步移动，形成稳定、克制的“一体滑行”反馈。
- 支持 H5 现有 `dark` / `light` 两套 UI。
- 动效仅使用 `transform` 与 `opacity`，并支持 `prefers-reduced-motion` 降级。

## UI 主题识别

H5 已有两套 UI：

- `dark`
- `light`

切换机制：

- `ThemeProvider` 读取 `localStorage` key `h5.theme`。
- 通过 `document.documentElement.setAttribute('data-theme', theme)` 写入 `html[data-theme="dark|light"]`。
- 默认主题为 `dark`。
- 未设置 `data-theme` 时按 `dark` 兜底。

Token 来源：

- `frontend/h5/src/styles/tokens/colors.css`
- `frontend/h5/src/store/themeContext.tsx`
- `frontend/h5/src/components/MobileShell/MobileShell.module.css`

## 选定方案

### A · 一体滑行

胶囊和顶部金线作为同一个共享选中态同步移动。

核心视觉：

- `nav` 保持现有底部玻璃底座。
- `nav` 内新增 `.navIndicator`，用于渲染选中胶囊。
- `nav` 内新增 `.navIndicatorLine`，用于渲染顶部短金线。
- 当前 active Tab 只负责文字和图标颜色、轻微上浮、语义状态，不再渲染自己的背景胶囊。

选择理由：

- 与“高端拍卖”调性一致，克制但有质感。
- 动效稳定，不抢直播间内容焦点。
- 可直接复用现有 `--nav-active-bg`、`--nav-active-ring`、`--nav-active-shadow`。
- 比“金线先导”和“压感回弹”实现风险更低。

## 交互规则

### 状态

- `active`：当前路由匹配的 Tab。
- `inactive`：非当前路由 Tab。
- `pressed`：用户点击瞬间，Tab 自身允许轻微 `scale(0.98)`。

### 位置计算（固定宽度胶囊）

P1 决策：胶囊采用**固定视觉宽度** `--nav-indicator-width`（建议 `72px`），切换时只做 `translateX`，**宽度恒定不形变**。这是“一体滑行”稳定感的核心。

当路由或 active Tab 变化时：

1. 测量 active Tab 与 `nav` 的 DOM 矩形：`tabRect`、`navRect`。
2. 固定胶囊宽度记为 `W`（= `--nav-indicator-width`）。
3. 计算胶囊左边界（Tab 内水平居中）：

   ```
   x = (tabRect.left - navRect.left) + (tabRect.width - W) / 2
   ```

4. 写入 CSS 变量：
   - `--nav-indicator-x`：胶囊左边界 `x`。
   - `--nav-indicator-width`：固定值 `W`，**不随 Tab 宽度变化**。
5. `.navIndicator` 使用 `transform: translate3d(var(--nav-indicator-x), 0, 0)`，`width` 恒为 `W`。
6. `.navIndicatorLine` 金线中心 = `x + W / 2`，即金线用 `transform: translate3d(calc(var(--nav-indicator-x) + var(--nav-indicator-width) / 2 - 金线半宽), 0, 0)` 与胶囊同步，且金线居中恒定依赖固定 `W`，不会因 Tab 宽度变化而偏移。

固定宽度的好处：

- 胶囊永不一边平移一边变宽/变窄，符合“稳定克制”的目标。
- 金线居中公式只依赖固定 `W`，恒定可靠。
- `(tabRect.width - W) / 2` 居中项保证：即使某个 Tab 因文案或 `43` 角标被撑宽，胶囊仍视觉居中于该 Tab。

不采用纯 CSS 乘法计算位置，原因：

- `calc(var(--index) * var(--step))` 可读性和兼容性不如 DOM 测量稳定。
- 实际 Tab 宽度、字体渲染、角标尺寸和响应式布局会影响 Tab 左边界与宽度。
- DOM 测量 + Tab 内居中能保证胶囊在每个 Tab 下视觉居中，且与 active Tab 严格对应。

### React 测量实现要点

- 为每个 Tab 绑定 `ref`，维护 `tabRefs` 数组；`nav` 自身也需 `ref`。
- 使用 `useLayoutEffect`（**非** `useEffect`）在 DOM 布局后、绘制前完成首次测量，避免首帧胶囊位置闪烁。
- 依赖项必须包含**当前路由 / active index**，路由变化时重新测量并写入 CSS 变量。
- `resize` 监听中对 active Tab 重新测量（无动画），保证响应式下重新对齐。

### 初始测量时机

- 首次测量除 `useLayoutEffect` 外，还需在 `document.fonts.ready` resolve 后**重测一次**：web font 加载完成会改变 Tab 文案宽度，导致 `tabRect.width` 变化，若不重测会出现初始错位。
- 首次定位（含字体重测、resize 重测）必须**禁用过渡**（临时 `transition: none` + `requestAnimationFrame` 恢复），避免页面加载或缩放时胶囊从原点滑入的突兀动画。

### 定位基准约束

- 公式 `tabRect.left - navRect.left` 成立的前提是 `nav` **左右无 border**（当前仅有 `border-top`），此时 border box 左边与 padding box 左边一致。
- 约束：后续不得给 `bottomNav` 添加左/右 border，否则需改用 padding-box 基准重新推导，否则胶囊会整体横向偏移。

## 结构设计

建议结构：

```tsx
<nav className={styles.bottomNav} aria-label="底部导航">
  <span className={styles.navIndicator} aria-hidden="true" />
  <span className={styles.navIndicatorLine} aria-hidden="true" />
  {navItems.map((item) => (
    <Link ref={...} className={...}>
      <span className={styles.navIconWrap}>...</span>
      <span className={styles.navLabel}>{item.label}</span>
    </Link>
  ))}
</nav>
```

层级规则：

- `bottomNav` 使用 `isolation: isolate`。
- `navIndicator` 和 `navIndicatorLine` 使用 `z-index: 0`。
- `navItem` 使用 `z-index: 1`。
- 禁止使用 `z-index: -1`，否则会被 `nav` 背景遮挡，导致胶囊不可见或动画异常。

## 图标设计

替换字符图标：

- 首页：线性 Home 图标。
- 直播间：线性 Play / Live 图标。
- 我的：线性 User 图标。

约束：

- SVG 使用 `currentColor`，随 `active/inactive` 状态继承颜色。
- SVG 尺寸保持约 `22px`，不改变现有点击热区。
- 不引入外部图标资源，避免额外依赖。

## 动效参数

推荐参数：

- 指示器位移：`260ms cubic-bezier(.22, .9, .25, 1)`
- 文字与图标颜色：`160ms ease`
- active 图标：`translateY(-1px) scale(1.06)`
- active label：`translateY(-1px)`
- 固定胶囊只过渡 `transform`，**不过渡 `width`**（宽度恒定，无需过渡）。
- 以指示器位移 `260ms` 为唯一权威时长；预览中的 `switching` 辅助态计时需与之对齐，落地不保留 `460ms` 旧值。

降级：

```css
@media (prefers-reduced-motion: reduce) {
  .navIndicator,
  .navIndicatorLine,
  .navItem,
  .navIcon,
  .navLabel {
    transition: none;
  }
}
```

## 双主题适配

暗色主题：

- 使用 `--bg-elevated` 作为导航底座。
- 使用 `--text-brand` 作为 active 文案、图标与金线色。
- 使用现有 `--nav-active-bg`、`--nav-active-ring`、`--nav-active-shadow` 表达胶囊质感。

浅色主题：

- 复用 `:root[data-theme='light'] .bottomNav` 中现有的 `--nav-active-*` 覆盖。
- 金线使用 `--text-brand`，保证与浅色页面的暖金调一致。
- 角标仍使用现有 `BadgeDot` 体系，不改变通知语义和颜色策略。

## 可访问性

- `nav` 保持 `aria-label="底部导航"`。
- 当前入口保持 `aria-current="page"`。
- 共享指示器使用 `aria-hidden="true"`。
- 点击热区不得小于当前 `min-height: 48px`，满足移动端触控需求。

## 验证要求

视觉验证：

- 在 `dark` 和 `light` 两套 UI 下切换：首页、直播间、我的。
- 胶囊左边界经 Tab 内居中后，应使胶囊视觉居中于 active Tab。
- 胶囊宽度在三个入口间切换时**保持恒定**，不得出现变宽/变窄。
- 顶部金线应始终居中于胶囊。
- 角标不得被胶囊或金线遮挡。

技术验证：

- 路由直达 `/`、`/live`、`/profile` 时，初始指示器位置正确。
- web font 加载完成（`document.fonts.ready`）后指示器重新对齐，无初始错位。
- 浏览器 resize 或移动端横竖尺寸变化后，指示器重新对齐 active Tab。
- `prefers-reduced-motion` 下无滑行动画但状态正确。
- 现有底部导航隐藏路径仍不渲染导航。

## 非目标

- 不调整底部导航入口数量。
- 不改变提醒角标数据来源。
- 不改变 H5 主题切换机制。
- 不引入新的图标库或动画库。

## 已选决策

- 视觉方向：高端拍卖。
- 静态方案：`A · 典藏印章`。
- 动效方案：`A · 一体滑行`。
- 实现策略：共享指示器 + DOM 测量定位。
- 胶囊宽度（P1）：固定视觉宽度 `--nav-indicator-width`（建议 `72px`），切换只 `translateX` 不变宽。

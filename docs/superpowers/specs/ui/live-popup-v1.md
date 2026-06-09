# 直播开播弹窗设计决策 (Live Popup UI Design)

## 选定方案：方案 A（现代浮动 Playful）

### 核心视觉特征
1. **打破边界**：采用顶部悬浮 SVG 矢量图标（摄像机），打破常规方框的视觉束缚。
2. **渐变质感**：图标描边与主按钮（立即前往）采用品牌主色（Primary）的线性渐变，提升视觉焦点和转化率。
3. **卡片内嵌**：直播间信息（Logo、名称、状态）包裹在浅色内嵌卡片（`item-subtle-bg`）中，层次分明。
4. **大圆角**：弹窗容器使用 `24px` 圆角，更加现代、年轻化。
5. **动态呼吸灯**：“正在直播”状态配有脉冲动画（pulse）的绿点。

### Theme Token 适配（支持 Dark / Light 双主题）

设计方案已全面适配项目中现有的 `colors.css` 语义 token。

| UI 元素 | Token (Light/Dark 自动适配) |
| --- | --- |
| 弹窗背景 | `var(--bg-surface)` |
| 悬浮图标背景 | `var(--bg-surface)` |
| 悬浮图标外框 | `1px solid var(--border-subtle)` |
| 标题文字 | `var(--text-primary)` |
| 副标题文字 | `var(--text-secondary)` |
| 内部卡片背景 | `var(--item-subtle-bg)` |
| 直播间名称 | `var(--text-primary)` |
| 状态文字/绿点 | `var(--color-success-500)` |
| 渐变主按钮 | `var(--gradient-primary)` |
| 次要按钮背景 | `var(--item-subtle-bg)` |
| 次要按钮文字 | `var(--text-secondary)` |

### 兜底与预览
- HTML 原型产物路径：`/tmp/ui-design-trio-live-popup/index.html`

### 后续开发实施建议
1. 提取 SVG 矢量图标封装为独立组件（如 `LiveCameraIcon`）。
2. 在 H5 端应用时，复用项目中已有的 `Modal` 或 `Overlay` 基础组件（如果存在），否则按该方案的 CSS 实现。
3. 确保外层包裹 `<ThemeProvider>`，以保证 `data-theme` 正确挂载和 token 的生效。

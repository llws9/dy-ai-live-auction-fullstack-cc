# Live Room Top-Right Layout Redesign

## 背景与目标
当前直播间右上角的组角（音频开关、在线人数、商品详情、直播状态）采用纯纵向 Flex 堆叠，占据过多纵向空间。
本次优化目标：移除不再需要的“直播状态”展示，新增“点赞数”展示，并将整体布局重构为“数据岛与主操作分离”的悬浮布局。

## 选定方案
**方案 3: 数据岛与主操作分离 (Floating Island)**
- **数据岛 (Data Island)**：将“在线人数”和“点赞数”作为纯数据展示，聚合在一个底色较弱的胶囊容器中。
- **主操作 (Main Actions)**：将“声音开关”和“商品详情”独立放置，保持原本的操作按钮视觉，强调可交互属性。
- 空间排布上分为两组，减少单列过长带来的视线遮挡。

## 视觉与交互规范
1. **数据岛 (Data Island)**
   - 包含内容：在线人数（带头像组）、点赞数（心形图标 + 数量）。
   - 样式：背景采用半透明弱色 `rgba(0, 0, 0, 0.2)`，边框 `1px solid rgba(255, 255, 255, 0.1)`，带高斯模糊 `backdrop-filter: blur(12px)`。
   - 内部元素：保留现有的 `viewerAvatar` 头像层叠，右侧紧跟点赞数（点赞数使用品牌强调色或红色 `--color-accent-500` / `#ef4444` 作为图标颜色）。

2. **主操作区**
   - 包含内容：声音开关 (`🎵 ON / OFF`)、商品详情 (`商品详情 ›`)。
   - 样式：使用系统原有的 `--live-pill-bg` 和 `--live-pill-border`，维持与现有操作一致的视觉权重。
   - 布局：采用 `flex` 排布，可以根据空间横向并排或紧凑排列。

## UI 主题适配
项目支持 `dark` 和 `light` 双主题，通过 `<html data-theme="dark|light">` 切换。
- 深色主题 (`data-theme="dark"`)：使用 `rgba(8, 12, 20, 0.62)` 胶囊背景，`rgba(232, 200, 115, 0.28)` 边框。
- 浅色主题 (`data-theme="light"`)：使用 `rgba(0, 0, 0, 0.42)` 胶囊背景，`rgba(255, 255, 255, 0.28)` 边框。
- 文字颜色统一使用 `--live-pill-text`，点赞图标颜色使用 CSS 变量 `--like-color`（如红色 `#ef4444`）。

## 变更范围
- `frontend/h5/src/pages/Live/Live.module.css`：重构 `.rightActions` 的样式，新增 `.dataIsland`、`.likesPill` 等。
- `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`：移除 `statusPill` 渲染逻辑，新增点赞数读取与展示，调整 DOM 结构。

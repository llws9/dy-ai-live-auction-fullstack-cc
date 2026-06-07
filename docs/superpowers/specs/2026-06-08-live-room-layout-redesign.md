# Live Room Layout Redesign - Design Spec

## 1. 概览 (Overview)

为了提供更加沉浸和符合用户心智的直播竞拍体验，我们计划对 H5 用户端直播间 (`frontend/h5/src/pages/Live/LiveRoomSlide.tsx`) 的 UI 布局进行重构。主要目标是释放屏幕中下部的视觉空间，优化辅助信息的展示位置，并确保与抖音等主流直播平台的心智模型保持一致。

## 2. 核心调整项 (Key Changes)

### 2.1 移除商品大卡片，提升出价排行
- **当前状态**：屏幕中下部有一个 `<article className={styles.productCard}>` 占据了较大面积。
- **调整方案**：彻底移除该 `<article>` 标签及其包含的商品大卡片。
- **关联影响**：商品卡片移除后，下方的“出价排行” (`<section className={styles.rankingBlock}>`) 将自然上移，获得更多的展示空间。

### 2.2 主播信息与收藏按钮融合 (左上角)
- **当前状态**：收藏按钮原本附着在商品卡片上。
- **调整方案**：
  - 增强 `styles.topBar` 的左侧部分 (`hostPill`)。
  - 将原本位于商品卡片内的“收藏直播间”按钮迁移至左上角，直接跟在主播信息（头像、名称、点赞/在线说明）的右侧。
  - 采用高亮色（如抖音红/主题粉）设计，提升辨识度。

### 2.3 独立的在线人数与商品详情组件 (右上角)
- **当前状态**：在线人数附属于主播名称下方，商品详情无明确轻量化入口。
- **调整方案**：
  - **右上角第一行**：将“在线人数”提取为独立组件，置于右上角。左侧展示最多 3 个用户的重叠头像，右侧为半透明的数字胶囊，最右侧保留退出 (`X`) 按钮。
  - **右上角第二行 (方案 C 融合)**：在在线人数下方，新增一个白底微胶囊样式的“商品详情 >”按钮。
  - **交互**：点击“商品详情”按钮，从底部弹出半屏面板 (Bottom Sheet) 展示详细的商品和拍卖规则信息。

## 3. 组件结构更新示例 (Component Structure)

重构后的 `<header>` 区域大致结构如下：

```tsx
<header className={styles.topBar}>
  {/* 左侧：主播信息 + 收藏 */}
  <div className={styles.hostPill}>
    {/* 头像、名称、副标题 */}
    <div className={styles.hostInfo}>...</div>
    {/* 收藏按钮 */}
    <button className={styles.followBtn}>
      {following ? '已收藏' : '收藏'}
    </button>
  </div>

  {/* 右侧：观众区 + 详情入口 */}
  <div className={styles.rightActions}>
    {/* 观众区 */}
    <div className={styles.viewersRow}>
      <div className={styles.avatarsGroup}>...</div>
      <div className={styles.viewerCount}>{(liveStream?.viewer_count ?? 0).toLocaleString()}</div>
      <Link className={styles.closeBtn} to="/">X</Link>
    </div>
    
    {/* 商品详情入口 (原方案C) */}
    <button className={styles.productDetailBtn} onClick={() => setDetailSheetVisible(true)}>
      商品详情 &gt;
    </button>
  </div>
</header>
```

## 4. 视觉与样式指南 (Visual & Styling Guidelines)

- **背景与层级**：顶部栏背景使用半透明渐变或毛玻璃效果 (`backdrop-filter: blur(8px)`)，以保证在视频或封面图背景上文字的可读性。
- **色彩规范**：
  - 收藏按钮（未收藏状态）：推荐使用 `var(--color-dy-pink)` 或类似的高亮警示色。
  - 商品详情胶囊：推荐使用 `rgba(255, 255, 255, 0.95)` 搭配深色文字，保持轻量感。
- **头像层叠**：右侧观众头像使用负 `margin-left` 和递减的 `z-index` 实现层叠排列。

## 5. 后续步骤 (Next Steps)
1. **获取确认**：等待用户确认此 Design Spec。
2. **实施计划**：调用 `writing-plans` 技能，基于此 Spec 生成详细的代码修改任务列表。
3. **代码实施**：按计划修改 `LiveRoomSlide.tsx` 及其对应的 CSS Module。

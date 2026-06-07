# Live Room Layout Redesign - Design Spec

## 1. 概览 (Overview)

为了提供更加沉浸和符合用户心智的直播竞拍体验，我们计划对 H5 用户端直播间 (`frontend/h5/src/pages/Live/LiveRoomSlide.tsx`) 的 UI 布局进行重构。主要目标是释放屏幕中下部的视觉空间，优化辅助信息的展示位置，并确保与抖音等主流直播平台的心智模型保持一致。

## 2. 核心调整项 (Key Changes)

### 2.1 移除商品大卡片，提升出价排行
- **当前状态**：底部抽屉（`BidDock` 的 `.sheet`，仅 `sheet !== null` 时展开）内有一个 `<article className={styles.productCard}>` 占据了较大面积。注意它并非屏幕常驻元素，而是抽屉展开后才出现。
- **调整方案**：彻底移除该 `<article>` 标签及其包含的商品大卡片（商品图、名称、简介、抽屉内收藏行）。
- **关联影响**：在展开的抽屉内，商品卡片移除后下方的“出价排行” (`<section className={styles.rankingBlock}>`) 将自然上移，获得更多的展示空间，提升抽屉内布局适配性。

### 2.2 主播信息与收藏按钮融合 (左上角)
- **当前状态**：收藏按钮原本附着在商品卡片上。
- **调整方案**：
  - 增强 `styles.topBar` 的左侧部分 (`hostPill`)。
  - 将原本位于商品卡片内的“收藏直播间”按钮迁移至左上角，直接跟在主播信息（头像、名称、点赞/在线说明）的右侧。
  - 采用当前主题体系内的高亮色设计，提升辨识度；不得直接依赖未定义的 `var(--color-dy-pink)`。
  - 推荐新增局部 CSS 变量 `--live-follow-accent` / `--live-follow-accent-muted`，并分别在日/夜主题下给出可读、可点击的颜色值。

### 2.3 独立的在线人数与商品详情组件 (右上角)
- **当前状态**：在线人数附属于主播名称下方，商品详情无明确轻量化入口。
- **调整方案**：
  - **右上角第一行**：将“在线人数”提取为独立组件，置于右上角。左侧展示最多 3 个用户的重叠头像，右侧为半透明的数字胶囊，最右侧保留退出 (`X`) 按钮。
  - **右上角第二行 (方案 C 融合)**：在在线人数下方，新增一个主题适配的微胶囊样式“商品详情 >”按钮。
  - **头像数据 MVP 规则**：本次不新增后端接口契约。若现有直播间详情没有在线观众头像列表，则头像区只使用已有可用头像（如主播头像、当前用户头像）和本地占位头像；不足 3 个不强行补真实用户数据，不伪造在线用户身份。
  - **交互（已确认：小工作量方案）**：点击“商品详情”按钮直接跳转至现有商品详情页 `/detail?id=<auctionId>`（对应 `App.tsx` 中 `path="/detail"` 的 `ProductDetail` 页面，通过 `useSearchParams().get('id')` 接收 auctionId）。**不开抽屉、不新增侧边卡片、不新增 `sheet=info` 内容**。实现使用 react-router 的 `<Link to={`/detail?id=${auctionId}`}>`，离开直播间后用户可通过浏览器返回键回到直播间。

### 2.4 系统提示保留与强化
- **当前状态**：直播互动层已有 `ChatPanel`，视觉方案中出现“系统提示：欢迎来到直播间！”样式。
- **调整方案**：
  - 保留系统提示作为直播间氛围与引导信息的一部分。
  - 顶部栏、右上角观众组件、商品详情按钮不得遮挡系统提示区域。
  - 若实现中需要调整 `ChatPanel` 的提示样式，应保持轻量、半透明、可读，不引入新的业务接口。

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

    {/* 商品详情入口 (方案C)，直接跳转现有详情页 */}
    <Link className={styles.productDetailBtn} to={`/detail?id=${auctionId}`}>
      商品详情 &gt;
    </Link>
  </div>
</header>
```

点击“商品详情”跳转至 `/detail?id=<auctionId>`，由现有 `ProductDetail` 页面渲染完整商品信息（商品图片、名称、简介、价格、加价信息等），无需在直播间内新增详情面板。底部抽屉（`BidDock` / `sheet`）经本次改造后仅承载出价相关内容（价格、倒计时、出价排行、出价框），不再包含商品大卡片。

## 4. 视觉与样式指南 (Visual & Styling Guidelines)

- **背景与层级**：顶部栏背景使用半透明底色或毛玻璃效果 (`backdrop-filter: blur(8px)`)，以保证在视频或封面图背景上文字的可读性。避免依赖复杂渐变作为唯一可读性来源。
- **色彩规范**：
  - 收藏按钮（未收藏状态）：推荐使用局部变量 `--live-follow-accent`，日间/夜间分别映射到当前主题中对比度足够的高亮色。
  - 已收藏/处理中状态：使用 `--live-follow-accent-muted` 或现有 disabled 语义样式，避免与未收藏主操作抢视觉焦点。
  - 商品详情胶囊：不得硬编码只适合单一主题的白底黑字。日间可使用浅色半透明底，夜间应切换为深色半透明底或主题 surface 色，并保证文字对比度。
- **头像层叠**：右侧观众头像使用负 `margin-left` 和递减的 `z-index` 实现层叠排列。
- **日/夜主题适配**：
  - 所有新增样式优先使用现有 CSS 变量（如 `--bg-*`、`--text-*`、`--border-*`、`--radius-*`、`--spacing-*`）。
  - 必须同时覆盖 `:global(:root[data-theme='dark'])` 和默认/日间主题下的显示效果。
  - 半透明底、主题适配胶囊、头像边框、关闭按钮、详情按钮均需在直播画面深色/浅色背景上保持可读。
  - 禁止为绕过主题问题直接写死只在夜间有效的颜色组合。

## 5. 后续步骤 (Next Steps)
1. **获取确认**：等待用户确认此 Design Spec。
2. **实施计划**：调用 `writing-plans` 技能，基于此 Spec 生成详细的代码修改任务列表。
3. **代码实施**：按计划修改 `LiveRoomSlide.tsx` 及其对应的 CSS Module。

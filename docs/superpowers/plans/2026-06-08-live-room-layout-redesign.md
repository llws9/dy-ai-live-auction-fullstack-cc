# 直播间布局重构 实施计划 (Live Room Layout Redesign Implementation Plan)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 重构 H5 直播间 UI：将收藏按钮迁移至左上角主播信息区、移除底部抽屉内的商品大卡片让出价排行上移、在右上角新增独立的在线人数组件与跳转到商品详情页的入口，并保证日/夜双主题适配。

**Architecture:** 仅改动前端 H5 两个文件——`LiveRoomSlide.tsx`（JSX 结构与跳转）与 `Live.module.css`（样式与双主题）。商品详情走现有 `/detail?id=<auctionId>` 路由，不新增组件、不新增后端契约、不改抽屉 URL 机制。收藏逻辑（`handleFollow` / `following` / `followersCount` / `followingPending`）保持不变，仅迁移其渲染位置。

**Tech Stack:** React 18 + TypeScript + Vite，CSS Modules（`Live.module.css`，CSS 变量 + `:global(:root[data-theme='dark'])` 双主题），react-router-dom（`Link`），Jest + React Testing Library（`MemoryRouter`）。

**关键事实（实施前必读）:**
- 商品大卡片 `<article className={styles.productCard}>` 位于 [LiveRoomSlide.tsx#L1029-L1041](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L1029-L1041)，它是 `<BidDock>` 的 children，**仅在抽屉展开时渲染**，非屏幕常驻。
- 顶部栏 `<header className={styles.topBar}>` 在 [LiveRoomSlide.tsx#L933-L952](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L933-L952)，当前左侧 `hostPill`（含 `viewerCount` "X 在线"），右侧 `statusPill`。
- `auctionId` 在组件内已有同名变量可用（见 [LiveRoomSlide.tsx#L408](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L408) 上下文，组件早已使用 `auctionId`）。实施时确认其在 render 作用域内可见。
- `hostAvatar`（[L265](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L265)）可能为空字符串，头像区需做占位兜底。
- 已确认的 CSS 变量：`--text-primary`、`--text-secondary`、`--text-brand`、`--bg-surface`、`--bg-page`、`--border-subtle`、`--radius-full`、`--radius-md`、`--spacing-2`、`--spacing-3`、`--spacing-4`、`--font-size-xs`、`--font-size-sm`、`--font-weight-bold`。**不存在 `--color-dy-pink`**。
- 测试命令工作目录：`frontend/h5`。运行单测：`npm test -- <pattern>`。

---

## 文件结构 (File Structure)

| 文件 | 责任 | 操作 |
|---|---|---|
| `frontend/h5/src/pages/Live/LiveRoomSlide.tsx` | 直播间页面 JSX 结构：顶部栏重构、收藏按钮迁移、在线人数组件、详情入口、移除商品卡片 | Modify |
| `frontend/h5/src/pages/Live/Live.module.css` | 新增 `.followBtn`/`.rightActions`/`.viewersRow`/`.avatarsGroup`/`.viewerCountPill`/`.closeBtn`/`.productDetailBtn` 等样式及日/夜双主题规则；移除/清理废弃的 `.productCard` 相关样式 | Modify |
| `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx` | 新增断言：收藏按钮在顶部、详情链接指向 `/detail?id=`、商品卡片已移除 | Modify |

---

## Task 1: 顶部栏新增收藏按钮与右上角操作区（含商品详情入口）

**Files:**
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx` (header 区 L933-L952)
- Modify: `frontend/h5/src/pages/Live/Live.module.css`
- Test: `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx`

- [ ] **Step 1: Write the failing test**

在 `LiveRoomSlide.test.tsx` 中新增（放在已有 describe 块内，沿用文件顶部已 mock 的 api / router / auth；参考现有用例的 render 与 `auctionApi.get` mock 数据准备方式）：

```tsx
it('在顶部栏渲染收藏按钮，并提供跳转商品详情页的入口', async () => {
  // 复用本文件已有的 renderComponent / mock 数据准备方式（auctionApi.get、bidApi.getRanking 等已在 beforeEach 设置）
  renderComponent();

  // 收藏按钮迁移到顶部（不再依赖打开抽屉）
  const followBtn = await screen.findByRole('button', { name: /收藏/ });
  expect(followBtn).toBeInTheDocument();

  // 商品详情入口为链接，指向 /detail?id=<auctionId>
  const detailLink = screen.getByRole('link', { name: /商品详情/ });
  expect(detailLink).toHaveAttribute('href', expect.stringContaining('/detail?id='));
});
```

> 注意：若本文件无 `renderComponent` 帮助函数，按文件现有写法用 `render(<MemoryRouter>...<LiveRoomSlide /></MemoryRouter>)` 并在 `beforeEach` 中设置 `mockedAuctionApi.get.mockResolvedValue(...)` 等（参考文件 L118+ 的 mock 变量与现有 it 用例）。`auctionId` 取自 mock 的 auction 数据 `id`。

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend/h5 && npm test -- LiveRoomSlide`
Expected: FAIL —— 找不到顶部的收藏按钮 / 找不到 `商品详情` 链接（当前收藏按钮在抽屉里，详情入口不存在）。

- [ ] **Step 3: 重构 header JSX**

将 [LiveRoomSlide.tsx#L933-L952](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L933-L952) 的 `<header>` 替换为下面结构（保留原 `hostPill` 内已有的 `backLink`/`avatar`/`hostName`/`viewerCount`，在其右侧加收藏按钮；右侧用 `rightActions` 容器承载在线人数行与详情入口；原 `statusPill` 移入右侧操作区内或保留——此处保留 `statusPill` 于在线人数行之后，避免丢失开播状态展示）：

```tsx
<header className={styles.topBar}>
  <div className={styles.hostPill}>
    <Link className={styles.backLink} to="/">‹</Link>
    <div className={styles.avatar}>
      {hostAvatar ? (
        <img src={hostAvatar} alt={hostName} />
      ) : (
        <span>{hostName.slice(0, 1)}</span>
      )}
    </div>
    <div>
      <p className={styles.hostName}>{hostName}</p>
      <p className={styles.viewerCount}>{(liveStream?.viewer_count ?? 0).toLocaleString()} 在线</p>
    </div>
    <button
      className={styles.followBtn}
      disabled={followingPending}
      onClick={handleFollow}
      type="button"
    >
      {followingPending ? '处理中...' : following ? '已收藏' : '收藏'}
    </button>
  </div>

  <div className={styles.rightActions}>
    <div className={styles.viewersRow}>
      <div className={styles.avatarsGroup}>
        {hostAvatar ? (
          <span className={styles.viewerAvatar}><img src={hostAvatar} alt="" /></span>
        ) : (
          <span className={styles.viewerAvatar}>{hostName.slice(0, 1)}</span>
        )}
      </div>
      <span className={styles.viewerCountPill}>{(liveStream?.viewer_count ?? 0).toLocaleString()}</span>
      <Link className={styles.closeBtn} to="/" aria-label="退出直播间">✕</Link>
    </div>
    <div className={styles.statusPill}>
      <span className={isActive ? styles.liveDot : styles.statusDot} />
      {getEffectiveStatusText(auction.status, hasReachedEndTime)}
    </div>
    <Link className={styles.productDetailBtn} to={`/detail?id=${auctionId}`}>
      商品详情 ›
    </Link>
  </div>
</header>
```

> 说明：头像区 MVP 仅展示主播/占位头像（不伪造在线用户身份，符合 Spec §2.3）。`auctionId` 已是组件内现有变量；如作用域内不可见，使用 `auction?.id`。确认 `Link` 已从 `react-router-dom` 导入（文件顶部应已有 `import { Link } from 'react-router-dom'`，若无则补充）。

- [ ] **Step 4: 新增样式（含日/夜双主题）**

在 `Live.module.css` 顶部栏样式区（`.statusPill` 之后，约 L328 附近）追加：

```css
.followBtn {
  margin-left: var(--spacing-2);
  padding: 5px 14px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--live-follow-accent);
  color: #fff;
  cursor: pointer;
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-bold);
  white-space: nowrap;
}

.followBtn:disabled {
  cursor: not-allowed;
  background: var(--live-follow-accent-muted);
  opacity: 0.8;
}

.rightActions {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
}

.viewersRow {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 6px 3px 3px;
  border-radius: var(--radius-full);
  background: rgba(0, 0, 0, 0.48);
  backdrop-filter: blur(16px);
}

.avatarsGroup {
  display: flex;
  align-items: center;
}

.viewerAvatar {
  display: inline-flex;
  width: 22px;
  height: 22px;
  align-items: center;
  justify-content: center;
  margin-left: -8px;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.7);
  border-radius: var(--radius-full);
  background: var(--bg-surface);
  color: var(--text-brand);
  font-size: 10px;
  font-weight: var(--font-weight-bold);
}

.avatarsGroup .viewerAvatar:first-child {
  margin-left: 0;
}

.viewerAvatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.viewerCountPill {
  color: rgba(255, 255, 255, 0.9);
  font-size: 11px;
  font-weight: var(--font-weight-bold);
}

.closeBtn {
  display: inline-flex;
  width: 22px;
  height: 22px;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.9);
  font-size: 13px;
  line-height: 1;
  text-decoration: none;
}

.productDetailBtn {
  padding: 5px 12px;
  border: 1px solid rgba(255, 255, 255, 0.28);
  border-radius: var(--radius-full);
  background: rgba(0, 0, 0, 0.42);
  backdrop-filter: blur(12px);
  color: #fff;
  font-size: 11px;
  font-weight: var(--font-weight-bold);
  text-decoration: none;
  white-space: nowrap;
}
```

在 `:root`（文件顶部 `.page` 或全局变量定义处；若该 module 内无 `:root` 块，则在默认主题下用 `.page` 选择器或直接在以上规则里给默认值，并补 dark 覆盖）定义局部变量。推荐在文件已有的双主题块旁追加：

```css
:global(:root:not([data-theme])) .topBar,
:global(:root[data-theme='light']) .topBar {
  --live-follow-accent: linear-gradient(135deg, #c9a96e, #d4af37);
  --live-follow-accent-muted: rgba(201, 169, 110, 0.55);
}

:global(:root[data-theme='dark']) .topBar {
  --live-follow-accent: linear-gradient(135deg, #d4af37, #e8c873);
  --live-follow-accent-muted: rgba(212, 175, 55, 0.45);
}
```

> 顶部栏叠在视频/封面图上，文字与半透明底在日/夜下均可读；收藏按钮用项目既有金色语义，避免引入未定义变量。如项目实际有更合适的全局高亮变量，可替换 `--live-follow-accent` 的取值，但需同时覆盖两套主题。

- [ ] **Step 5: Run test to verify it passes**

Run: `cd frontend/h5 && npm test -- LiveRoomSlide`
Expected: 新增用例 PASS（收藏按钮可见、详情链接 href 含 `/detail?id=`）。其余既有用例保持 PASS。

- [ ] **Step 6: Commit**

```bash
git add frontend/h5/src/pages/Live/LiveRoomSlide.tsx frontend/h5/src/pages/Live/Live.module.css frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx
git commit -m "feat(live): 顶部栏迁移收藏按钮并新增右上角在线人数与商品详情入口"
```

---

## Task 2: 移除抽屉内商品大卡片，让出价排行上移

**Files:**
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx` (L1029-L1041)
- Modify: `frontend/h5/src/pages/Live/Live.module.css` (清理 `.productCard` 相关)
- Test: `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx`

- [ ] **Step 1: Write the failing test**

在 `LiveRoomSlide.test.tsx` 新增用例，断言打开抽屉后不再出现商品卡片内的收藏按钮/简介（抽屉内收藏行已删除，收藏统一在顶部）：

```tsx
it('打开出价抽屉后，抽屉内不再渲染商品大卡片的收藏行', async () => {
  renderComponent();
  // 打开抽屉（点击底部 dock，触发 openSheet('info') 或点出价）
  const dock = await screen.findByTestId('bid-dock');
  fireEvent.click(dock);

  // 抽屉内的“X 人收藏”文案应不存在（该行随 productCard 一并移除）
  await waitFor(() => {
    expect(screen.queryByText(/人收藏/)).not.toBeInTheDocument();
  });
  // 出价排行仍在
  expect(screen.getByText('出价排行')).toBeInTheDocument();
});
```

> `bid-dock` testid 来自 [BidDock.tsx#L97](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/BidDock.tsx#L97)。出价排行标题来自 [LiveRoomSlide.tsx#L1045-L1047](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L1045-L1047)。

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend/h5 && npm test -- LiveRoomSlide`
Expected: FAIL —— 当前抽屉内仍有 `{followersCount} 人收藏`（[L1038](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L1038)），`queryByText(/人收藏/)` 命中导致断言失败。

- [ ] **Step 3: 删除商品大卡片 JSX**

删除 [LiveRoomSlide.tsx#L1029-L1041](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L1029-L1041) 整段：

```tsx
<article className={styles.productCard}>
  {productImage ? <img src={productImage} alt={productName} /> : <div className={styles.productFallback}>暂无图片</div>}
  <div>
    <h1>{productName}</h1>
    <p>{productIntro}</p>
    <div className={styles.followRow}>
      <button className={styles.followButton} disabled={followingPending} onClick={handleFollow} type="button">
        {followingPending ? '处理中...' : following ? '已收藏' : '收藏'}
      </button>
      <span>{followersCount.toLocaleString()} 人收藏</span>
    </div>
  </div>
</article>
```

删除后，抽屉 children 顺序变为：`priceBlock` → `countdown` → `rankingBlock` → `bidBox`，排行自然上移。

- [ ] **Step 4: 清理废弃样式**

从 `Live.module.css` 删除仅服务于该卡片、且全局无其它引用的规则：`.productCard`、`.productCard img`、`.productFallback`、`.followRow`、`.followRow span`、`.followButton`、`.followButton:disabled`（[L742-L768](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/Live.module.css#L742-L768) 及其 dark 主题覆盖）。

> 删除前用 Grep 确认 `styles.followButton` / `styles.followRow` / `styles.productCard` / `styles.productFallback` 在 `.tsx` 中已无其它引用（Task 1 已改用 `styles.followBtn`，与此不同名）。若有引用残留则保留对应类。

- [ ] **Step 5: Run test to verify it passes**

Run: `cd frontend/h5 && npm test -- LiveRoomSlide`
Expected: 本任务用例 PASS；Task 1 用例继续 PASS。

- [ ] **Step 6: Commit**

```bash
git add frontend/h5/src/pages/Live/LiveRoomSlide.tsx frontend/h5/src/pages/Live/Live.module.css frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx
git commit -m "refactor(live): 移除抽屉内商品大卡片，出价排行上移"
```

---

## Task 3: 全量验证与双主题人工核对

**Files:**
- 无新增；运行验证。

- [ ] **Step 1: 运行 Live 目录全部单测**

Run: `cd frontend/h5 && npm test -- src/pages/Live`
Expected: 全部 PASS（含 `LiveRoomSlide.test.tsx`、`BidDock.test.tsx`、`LiveRoom.test.tsx`、`LiveLayoutCss.test.ts` 等）。

- [ ] **Step 2: TypeScript 类型检查**

Run: `cd frontend/h5 && npx tsc --noEmit`
Expected: 无类型错误（重点：`auctionId` 在 render 作用域可用、`Link` 已导入、删除卡片后无遗留引用 `productIntro` 等未使用变量告警——如出现未使用变量，按需删除其声明）。

- [ ] **Step 3: 启动 dev server 人工核对日/夜双主题**

Run: `cd frontend/h5 && npm run dev`（前台长进程，端口 5173/5174）
人工核对项：
1. 默认（日间）主题：顶部左侧主播信息 + 收藏按钮可读；右上角在线人数行、状态、商品详情按钮在浅色视频背景上可读。
2. 切换 `data-theme='dark'`：以上元素在深色背景上同样可读，收藏按钮金色语义正确。
3. 点击「收藏」→ 文案在 收藏/已收藏/处理中 间切换，禁用态样式正确。
4. 点击「商品详情 ›」→ 跳转 `/detail?id=<auctionId>`，浏览器返回键可回到直播间。
5. 打开出价抽屉 → 无商品大卡片，出价排行位置上移、布局正常。
6. 系统提示「欢迎来到直播间！」未被顶部栏/右上角组件遮挡。

- [ ] **Step 4: Commit（如人工核对触发微调）**

```bash
git add -A
git commit -m "fix(live): 双主题与交互核对后的样式微调"
```

> 若 Step 3 无需修改，则跳过本 Step。

---

## Self-Review 记录

- **Spec 覆盖**：§2.1（Task 2 移除卡片+排行上移）、§2.2（Task 1 收藏按钮迁移+`--live-follow-accent`）、§2.3（Task 1 在线人数组件+头像兜底+详情跳转 `/detail`）、§2.4（Task 3 Step3-6 人工核对系统提示不被遮挡）、§4（Task 1 Step4 双主题样式 + Task 3 双主题核对）。全部有对应任务。
- **占位符扫描**：无 TBD/TODO；测试与实现均给出完整代码。
- **类型一致性**：收藏按钮统一用新类名 `styles.followBtn`（与被删除的旧 `styles.followButton` 不同名，避免冲突）；`auctionId`/`auction?.id`、`Link`、`hostAvatar`、`following`/`followingPending`/`handleFollow` 均为现有标识符。

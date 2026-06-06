# Auction History Unread Won Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 H5「我的竞拍记录」页面中，让“竞拍成功但待处理/待支付”的记录能和已处理成功记录、未中标记录明确区分。

**Architecture:** 以 `/orders/history` 返回的单条记录为 SSOT：`is_winner=true` 表示竞拍成功，`status=0` 或 `status='pending'/'pending_payment'/'unpaid'` 表示待处理。前端只在 `History` 页面增加状态派生函数、可访问标签、按钮文案和 CSS 视觉层级，不引入本地已读状态，也不联动通知 `read_at`。

**Tech Stack:** React 18, TypeScript, CSS Modules, Jest, React Testing Library

---

## File Structure

- Modify: `frontend/h5/src/pages/History/__tests__/AuctionHistory.test.tsx`
  - 负责锁定业务语义：待支付中标记录显示「待处理」，已支付中标记录显示「竞拍成功」，未中标记录仍显示「未中标」。
- Modify: `frontend/h5/src/pages/History/index.tsx`
  - 负责从 `HistoryRecord` 派生 `pendingWon`，并把结果接入 `article` 可访问名称、状态 badge、CTA 文案与样式 class。
- Modify: `frontend/h5/src/pages/History/AuctionHistory.module.css`
  - 负责给待处理中标卡片增加更强视觉层级，避免和普通「竞拍成功」卡片混淆。

---

### Task 1: 写失败测试，锁定待处理中标记录语义

**Files:**
- Modify: `frontend/h5/src/pages/History/__tests__/AuctionHistory.test.tsx`

- [ ] **Step 1: 扩充测试数据，覆盖待处理中标、已处理中标、未中标三类记录**

Replace the existing `mockedOrderApi.history.mockResolvedValue` block in `beforeEach` with:

```tsx
    mockedOrderApi.history.mockResolvedValue({
      list: [
        {
          auction_id: 12,
          product_name: '鎏金香炉',
          final_price: 6800,
          is_winner: true,
          status: 0,
          bid_count: 5,
          created_at: '2026-05-29T12:00:00Z',
        },
        {
          auction_id: 14,
          product_name: '青花瓷茶具',
          final_price: 570,
          is_winner: true,
          status: 1,
          bid_count: 1,
          created_at: '2026-05-30T12:00:00Z',
        },
        {
          auction_id: 13,
          product_name: '宋瓷盏',
          final_price: 4200,
          is_winner: false,
          status: 0,
          bid_count: 2,
          created_at: '2026-05-28T12:00:00Z',
        },
      ],
      total: 3,
    });
```

- [ ] **Step 2: 更新既有列表测试的断言**

In `loads documented history records without order payment behavior`, replace the assertions after `expect(await screen.findByText('鎏金香炉')).toBeInTheDocument();` with:

```tsx
    expect(screen.getByText('青花瓷茶具')).toBeInTheDocument();
    expect(screen.getByText('宋瓷盏')).toBeInTheDocument();
    expect(screen.getByText('待处理')).toBeInTheDocument();
    expect(screen.getAllByText('竞拍成功').length).toBeGreaterThan(0);
    expect(screen.getAllByText('未中标').length).toBeGreaterThan(0);
    expect(screen.getByText('出价 5 次')).toBeInTheDocument();
    expect(screen.getAllByText('¥6,800').length).toBeGreaterThan(0);
```

- [ ] **Step 3: 新增待处理中标卡片区分测试**

Add this test after `loads documented history records without order payment behavior`:

```tsx
  it('distinguishes pending won records from processed won records', async () => {
    render(
      <ThemeProvider>
        <MemoryRouter future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
          <HistoryPage />
        </MemoryRouter>
      </ThemeProvider>
    );

    expect(await screen.findByRole('article', { name: /LOT 12 待处理/ })).toBeInTheDocument();
    expect(screen.getByRole('article', { name: /LOT 14 已处理/ })).toBeInTheDocument();
    expect(screen.getByRole('article', { name: /LOT 13 未中标/ })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '查看并处理' })).toHaveAttribute('href', '/result?id=12');
    expect(screen.getByRole('link', { name: '查看结果' })).toHaveAttribute('href', '/result?id=14');
    expect(screen.getByRole('link', { name: '查看详情' })).toHaveAttribute('href', '/detail?id=13');
  });
```

- [ ] **Step 4: 更新 won filter 测试，确认中标筛选仍包含待处理和已处理成功记录**

Replace the body assertions in `opens won filter from profile deep link` with:

```tsx
    expect(await screen.findByText('鎏金香炉')).toBeInTheDocument();
    expect(screen.getByText('青花瓷茶具')).toBeInTheDocument();
    expect(screen.queryByText('宋瓷盏')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '竞拍成功' })).toHaveClass('filterActive');
```

- [ ] **Step 5: 运行测试，确认失败来自未实现状态区分**

Run:

```bash
npm --prefix frontend/h5 test -- --runTestsByPath src/pages/History/__tests__/AuctionHistory.test.tsx
```

Expected:

```text
FAIL src/pages/History/__tests__/AuctionHistory.test.tsx
Unable to find role="article" and name /LOT 12 待处理/
```

---

### Task 2: 实现待处理中标状态派生与渲染

**Files:**
- Modify: `frontend/h5/src/pages/History/index.tsx`

- [ ] **Step 1: 增加待处理订单状态判定函数**

Add this function after `isWon(record: HistoryRecord)`:

```tsx
function isPendingOrderStatus(status: HistoryRecord['status']) {
  const normalized = String(status ?? '').toLowerCase();
  return ['0', 'pending', 'pending_payment', 'unpaid'].includes(normalized);
}

function isPendingWonRecord(record: HistoryRecord) {
  return isWon(record) && isPendingOrderStatus(record.status);
}
```

- [ ] **Step 2: 增加展示文案派生函数**

Add these functions after `bidSummary(record: HistoryRecord)`:

```tsx
function getRecordStateLabel(record: HistoryRecord) {
  if (isPendingWonRecord(record)) return '待处理';
  if (isWon(record)) return '已处理';
  return '未中标';
}

function getBadgeText(record: HistoryRecord) {
  if (isPendingWonRecord(record)) return '待处理';
  if (isWon(record)) return '竞拍成功';
  return '未中标';
}

function getActionText(record: HistoryRecord) {
  if (isPendingWonRecord(record)) return '查看并处理';
  if (isWon(record)) return '查看结果';
  return '查看详情';
}
```

- [ ] **Step 3: 在列表渲染中派生 `pendingWon` 和对应 class**

Inside `filteredRecords.map((record) => { ... })`, replace:

```tsx
              const won = isWon(record);
              const image = getProductImage(record);
```

with:

```tsx
              const won = isWon(record);
              const pendingWon = isPendingWonRecord(record);
              const image = getProductImage(record);
              const recordCardClassName = pendingWon
                ? `${styles.recordCard} ${styles.unreadRecordCard}`
                : styles.recordCard;
              const badgeClassName = pendingWon
                ? styles.pendingWonBadge
                : won
                  ? styles.wonBadge
                  : styles.lostBadge;
```

- [ ] **Step 4: 更新 `article` 的 class 和可访问名称**

Replace:

```tsx
                <article className={styles.recordCard} key={String(recordId)}>
```

with:

```tsx
                <article
                  className={recordCardClassName}
                  key={String(recordId)}
                  aria-label={`LOT ${recordId} ${getRecordStateLabel(record)}`}
                >
```

- [ ] **Step 5: 更新状态 badge 文案和 class**

Replace:

```tsx
                      <strong className={won ? styles.wonBadge : styles.lostBadge}>{won ? '竞拍成功' : '未中标'}</strong>
```

with:

```tsx
                      <strong className={badgeClassName}>{getBadgeText(record)}</strong>
```

- [ ] **Step 6: 更新 CTA 文案**

Replace:

```tsx
                    <Link className={won ? styles.primaryAction : styles.secondaryAction} to={won ? `/result?id=${recordId}` : `/detail?id=${recordId}`}>
                      {won ? '查看结果' : '查看详情'}
                    </Link>
```

with:

```tsx
                    <Link className={won ? styles.primaryAction : styles.secondaryAction} to={won ? `/result?id=${recordId}` : `/detail?id=${recordId}`}>
                      {getActionText(record)}
                    </Link>
```

- [ ] **Step 7: 运行测试，确认仍因缺 CSS module export 失败或通过**

Run:

```bash
npm --prefix frontend/h5 test -- --runTestsByPath src/pages/History/__tests__/AuctionHistory.test.tsx
```

Expected:

```text
PASS src/pages/History/__tests__/AuctionHistory.test.tsx
```

If the test environment checks CSS module keys strictly and fails because `unreadRecordCard` or `pendingWonBadge` is missing, continue Task 3 before re-running.

---

### Task 3: 增加待处理中标视觉样式

**Files:**
- Modify: `frontend/h5/src/pages/History/AuctionHistory.module.css`

- [ ] **Step 1: 增加待处理卡片视觉层级**

Add this block after `.recordCard { ... }`:

```css
.unreadRecordCard {
  border-color: rgba(212, 175, 55, 0.72);
  background:
    linear-gradient(135deg, rgba(212, 175, 55, 0.16), rgba(212, 175, 55, 0.04)),
    var(--bg-surface);
  box-shadow: 0 18px 36px rgba(212, 175, 55, 0.18);
}
```

- [ ] **Step 2: 将 `pendingWonBadge` 纳入 badge 基础样式**

Replace:

```css
.wonBadge,
.lostBadge {
```

with:

```css
.pendingWonBadge,
.wonBadge,
.lostBadge {
```

- [ ] **Step 3: 增加待处理 badge 样式**

Add this block before `.wonBadge { ... }`:

```css
.pendingWonBadge {
  background: linear-gradient(135deg, #ff8a00 0%, #d4af37 100%);
  color: #17130b;
  font-weight: 900;
}
```

- [ ] **Step 4: 运行聚焦测试**

Run:

```bash
npm --prefix frontend/h5 test -- --runTestsByPath src/pages/History/__tests__/AuctionHistory.test.tsx
```

Expected:

```text
PASS src/pages/History/__tests__/AuctionHistory.test.tsx
```

- [ ] **Step 5: 运行 H5 lint**

Run:

```bash
npm --prefix frontend/h5 run lint
```

Expected:

```text
No ESLint errors
```

- [ ] **Step 6: Commit**

```bash
git add frontend/h5/src/pages/History/index.tsx frontend/h5/src/pages/History/AuctionHistory.module.css frontend/h5/src/pages/History/__tests__/AuctionHistory.test.tsx
git commit -m "feat: distinguish pending won auction history records"
```

---

## Self-Review

- Spec coverage: 覆盖个人中心红点语义（`wonNotPaid` = 待支付中标）在历史卡片中的可见表达；覆盖待处理、已处理成功、未中标三类记录；覆盖中标筛选不回归。
- Placeholder scan: 本计划没有占位步骤、延后实现语句或未定义函数引用。
- Type consistency: `HistoryRecord['status']` 继续兼容当前 `string | number | undefined`；`isWon` 仍是中标判断 SSOT，`isPendingWonRecord` 只在中标前提下读取订单状态。
- Scope check: 不改通知系统、不改后端 API、不引入本地已读状态；实现范围集中在 H5 历史页。

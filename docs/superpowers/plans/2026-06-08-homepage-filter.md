# 首页筛选器 (Homepage Filter) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 H5 首页分类 Tab 下方新增筛选胶囊，支持「最热」（出价数聚合排序）与「价格区间」（current_price 过滤），实现用户分流。

**Architecture:** 纯 auction-service 改动 + H5 前端改动。后端在 `ListWithFilters` 内用 `LEFT JOIN bids ... GROUP BY ... ORDER BY COUNT(bids.id)` 实现热度排序，用 `WHERE current_price BETWEEN` 实现价格过滤，并通过只读字段 `BidCount` 回传出价数。前端新增筛选状态、胶囊 UI、价格底部抽屉，并在 hot 态跳过客户端 feed 重排。

**Tech Stack:** Go (Hertz + GORM + shopspring/decimal + glebarez/sqlite 测试), React + TypeScript + CSS Modules + Jest/RTL。

设计依据：[2026-06-08-homepage-filter-design.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/superpowers/specs/2026-06-08-homepage-filter-design.md)

---

## File Structure

**后端 (auction-service):**
- Modify: `backend/auction/model/auction.go` — `Auction` 增加只读字段 `BidCount`
- Modify: `backend/auction/dao/auction.go` — `AuctionFilters` 增加 `SortByHot/PriceMin/PriceMax`；`ListWithFilters` 实现过滤+排序+计数
- Test: `backend/auction/dao/auction_filter_test.go` — 新建，覆盖价格过滤、hot 排序、bid_count
- Modify: `backend/auction/handler/auction.go` — `List` 解析 `sort/price_min/price_max`
- Modify: `backend/auction/handler/auction_list.go` — `ListParams` 增加字段，编排时透传进 `AuctionFilters`

**前端 (h5):**
- Modify: `frontend/h5/src/services/api.ts` — `auctionApi.list` 参数扩展
- Modify: `frontend/h5/src/pages/Home/index.tsx` — 筛选状态、胶囊 UI、参数组装、hot 态跳过重排
- Create: `frontend/h5/src/pages/Home/PriceFilterSheet.tsx` — 价格底部抽屉组件
- Modify: `frontend/h5/src/pages/Home/Home.module.css` — 胶囊与抽屉样式
- Test: `frontend/h5/src/pages/Home/__tests__/Home.test.tsx` — 追加筛选交互用例

---

## Task 1: 后端 — Auction 模型增加只读 BidCount 字段

**Files:**
- Modify: `backend/auction/model/auction.go:20-34`

- [ ] **Step 1: 增加只读字段**

在 `Auction` 结构体 `CreatedAt` 字段后增加（`gorm:"->"` 表示只读，不参与建表/写入，仅在 Select 出 `bid_count` 列时回填）：

```go
	CreatedAt    time.Time       `json:"created_at" gorm:"autoCreateTime"`
	// BidCount 出价数聚合，仅热度排序查询时由 SELECT COUNT(bids.id) 回填；
	// 默认/其他查询不 SELECT 该列，值为 0。gorm:"->" 标记为只读虚拟列，不参与建表与写入。
	BidCount     int             `json:"bid_count" gorm:"->;-:migration"`
```

- [ ] **Step 2: 编译验证**

Run: `cd backend/auction && go build ./...`
Expected: 编译通过，无报错。

- [ ] **Step 3: Commit**

```bash
git add backend/auction/model/auction.go
git commit -m "feat(auction): add read-only BidCount field to Auction model"
```

---

## Task 2: 后端 — AuctionFilters 与 ListWithFilters 实现价格过滤与热度排序

**Files:**
- Modify: `backend/auction/dao/auction.go:433-443` (AuctionFilters)
- Modify: `backend/auction/dao/auction.go:309-371` (ListWithFilters)
- Test: `backend/auction/dao/auction_filter_test.go` (新建)

- [ ] **Step 1: 写失败测试**

新建 `backend/auction/dao/auction_filter_test.go`：

```go
package dao

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"auction-service/model"
)

func newFilterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Auction{}, &model.Bid{}))
	return db
}

func TestListWithFiltersPriceRange(t *testing.T) {
	db := newFilterTestDB(t)
	dao := NewAuctionDAO(db)
	ctx := context.Background()

	now := time.Now()
	rows := []model.Auction{
		{ID: 1, ProductID: 1, Status: model.AuctionStatusOngoing, CurrentPrice: decimal.NewFromInt(500), StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)},
		{ID: 2, ProductID: 2, Status: model.AuctionStatusOngoing, CurrentPrice: decimal.NewFromInt(2000), StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)},
		{ID: 3, ProductID: 3, Status: model.AuctionStatusOngoing, CurrentPrice: decimal.NewFromInt(8000), StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)},
	}
	for i := range rows {
		require.NoError(t, db.Create(&rows[i]).Error)
	}

	min := decimal.NewFromInt(1000)
	max := decimal.NewFromInt(5000)
	got, total, err := dao.ListWithFilters(ctx, &AuctionFilters{PriceMin: &min, PriceMax: &max}, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, got, 1)
	require.Equal(t, int64(2), got[0].ID)
}

func TestListWithFiltersSortByHot(t *testing.T) {
	db := newFilterTestDB(t)
	dao := NewAuctionDAO(db)
	ctx := context.Background()

	now := time.Now()
	auctions := []model.Auction{
		{ID: 1, ProductID: 1, Status: model.AuctionStatusOngoing, CurrentPrice: decimal.NewFromInt(100), StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)},
		{ID: 2, ProductID: 2, Status: model.AuctionStatusOngoing, CurrentPrice: decimal.NewFromInt(100), StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour)},
	}
	for i := range auctions {
		require.NoError(t, db.Create(&auctions[i]).Error)
	}
	// auction 2 有 3 次出价，auction 1 有 1 次出价
	bids := []model.Bid{
		{ID: 1, AuctionID: 1, UserID: 10, Amount: decimal.NewFromInt(110), CreatedAt: now},
		{ID: 2, AuctionID: 2, UserID: 11, Amount: decimal.NewFromInt(110), CreatedAt: now},
		{ID: 3, AuctionID: 2, UserID: 12, Amount: decimal.NewFromInt(120), CreatedAt: now},
		{ID: 4, AuctionID: 2, UserID: 13, Amount: decimal.NewFromInt(130), CreatedAt: now},
	}
	for i := range bids {
		require.NoError(t, db.Create(&bids[i]).Error)
	}

	got, total, err := dao.ListWithFilters(ctx, &AuctionFilters{SortByHot: true}, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, got, 2)
	require.Equal(t, int64(2), got[0].ID, "出价多的应排最前")
	require.Equal(t, 3, got[0].BidCount, "应回填 bid_count")
	require.Equal(t, int64(1), got[1].ID)
	require.Equal(t, 1, got[1].BidCount)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd backend/auction && go test ./dao/ -run 'TestListWithFiltersPriceRange|TestListWithFiltersSortByHot' -v`
Expected: 编译失败 / FAIL —— `AuctionFilters` 无 `PriceMin/PriceMax/SortByHot` 字段。

- [ ] **Step 3: 给 AuctionFilters 增加字段**

修改 `backend/auction/dao/auction.go` 的 `AuctionFilters`（约 433-443 行），在 `ProductIDs []int64` 后增加：

```go
	ProductIDs []int64
	// SortByHot 为 true 时按出价数聚合降序排序（替代默认 feed 排序）。
	SortByHot bool
	// PriceMin/PriceMax 按 auctions.current_price 区间过滤；nil 表示该侧不限。
	PriceMin *decimal.Decimal
	PriceMax *decimal.Decimal
```

- [ ] **Step 4: 在 ListWithFilters 实现过滤与排序**

修改 `backend/auction/dao/auction.go` 的 `ListWithFilters`。在「关键词搜索」块（约 350 行）之后、「获取总数」之前，插入价格过滤：

```go
	// 价格区间过滤（按 current_price，spec §2 决策）
	if filters.PriceMin != nil {
		query = query.Where("auctions.current_price >= ?", *filters.PriceMin)
	}
	if filters.PriceMax != nil {
		query = query.Where("auctions.current_price <= ?", *filters.PriceMax)
	}
```

然后替换「分页查询」块（约 357-368 行）为：

```go
	// 分页查询
	offset := (page - 1) * pageSize
	listQuery := query
	if filters.SortByHot {
		// 热度排序：LEFT JOIN bids 聚合出价数，回填 bid_count 只读列。
		// JOIN/GROUP BY 仅作用于 listQuery，不影响上面已算出的 total。
		listQuery = listQuery.
			Select("auctions.*, COUNT(bids.id) AS bid_count").
			Joins("LEFT JOIN bids ON bids.auction_id = auctions.id").
			Group("auctions.id").
			Order("bid_count DESC, auctions.id DESC")
	} else if filters.Upcoming {
		listQuery = listQuery.Order("start_time ASC, id ASC")
	} else {
		listQuery = orderByAuctionFeedPriority(listQuery)
	}
	err := listQuery.
		Offset(offset).
		Limit(pageSize).
		Find(&auctions).Error
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd backend/auction && go test ./dao/ -run 'TestListWithFiltersPriceRange|TestListWithFiltersSortByHot' -v`
Expected: PASS。

- [ ] **Step 6: 跑全量 dao 测试防回归**

Run: `cd backend/auction && go test ./dao/ -v`
Expected: 全部 PASS（含原 `TestListWithFiltersUpcoming...` 等）。

- [ ] **Step 7: Commit**

```bash
git add backend/auction/dao/auction.go backend/auction/dao/auction_filter_test.go
git commit -m "feat(auction): add price-range filter and hot-sort to ListWithFilters"
```

---

## Task 3: 后端 — handler 解析筛选参数并透传

**Files:**
- Modify: `backend/auction/handler/auction_list.go:22-32` (ListParams)
- Modify: `backend/auction/handler/auction_list.go:64-70` (BuildAuctionListResponse filters 装填)
- Modify: `backend/auction/handler/auction.go:386-428` (List 解析)

- [ ] **Step 1: ListParams 增加字段**

修改 `backend/auction/handler/auction_list.go` 的 `ListParams`（22-32 行），在 `PageSize int` 后增加：

```go
	Page           int
	PageSize       int
	SortByHot      bool
	PriceMin       *decimal.Decimal
	PriceMax       *decimal.Decimal
```

（`decimal` 包已在该文件 import，无需新增。）

- [ ] **Step 2: BuildAuctionListResponse 透传到 filters**

修改 `backend/auction/handler/auction_list.go` 的 `filters := &dao.AuctionFilters{...}`（64-70 行），增加三个字段：

```go
	filters := &dao.AuctionFilters{
		Status:         p.Status,
		LiveStreamID:   p.LiveStreamID,
		LiveStreamName: p.LiveStreamName,
		Search:         p.Search,
		Upcoming:       p.Upcoming,
		SortByHot:      p.SortByHot,
		PriceMin:       p.PriceMin,
		PriceMax:       p.PriceMax,
	}
```

- [ ] **Step 3: List handler 解析 query 参数**

修改 `backend/auction/handler/auction.go` 的 `List`。在 `pageSize, _ := strconv.Atoi(...)`（396 行）之后、`params := ListParams{...}` 之前，增加解析逻辑：

```go
	sortStr := c.Query("sort")
	sortByHot := sortStr == "hot"

	var priceMin, priceMax *decimal.Decimal
	if v := c.Query("price_min"); v != "" {
		if d, err := decimal.NewFromString(v); err == nil {
			priceMin = &d
		}
	}
	if v := c.Query("price_max"); v != "" {
		if d, err := decimal.NewFromString(v); err == nil {
			priceMax = &d
		}
	}
```

并在 `params := ListParams{...}` 字面量中追加：

```go
	params := ListParams{
		LiveStreamName: liveStreamName,
		Search:         search,
		Upcoming:       upcoming,
		Page:           page,
		PageSize:       pageSize,
		SortByHot:      sortByHot,
		PriceMin:       priceMin,
		PriceMax:       priceMax,
	}
```

确认文件已 import `github.com/shopspring/decimal`；若未 import 则加入 import 块。

- [ ] **Step 4: 编译并跑 handler 测试**

Run: `cd backend/auction && go build ./... && go test ./handler/ -v`
Expected: 编译通过，handler 测试 PASS。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/handler/auction.go backend/auction/handler/auction_list.go
git commit -m "feat(auction): parse sort/price_min/price_max in GET /auctions"
```

---

## Task 4: 前端 — auctionApi.list 扩展查询参数

**Files:**
- Modify: `frontend/h5/src/services/api.ts:350-360`

- [ ] **Step 1: 扩展 list 参数与拼接**

修改 `frontend/h5/src/services/api.ts` 的 `auctionApi.list`，参数类型与 query 拼接：

```typescript
  list: (params?: { status?: string; upcoming?: boolean; page?: number; page_size?: number; category_id?: number; sort?: string; price_min?: number; price_max?: number }) => {
    const query = new URLSearchParams();
    if (params?.status) query.set('status', params.status);
    if (params?.upcoming !== undefined) query.set('upcoming', String(params.upcoming));
    if (params?.page) query.set('page', String(params.page));
    if (params?.page_size) query.set('page_size', String(params.page_size));
    if (params?.category_id) query.set('category_id', String(params.category_id));
    if (params?.sort) query.set('sort', params.sort);
    if (params?.price_min !== undefined) query.set('price_min', String(params.price_min));
    if (params?.price_max !== undefined) query.set('price_max', String(params.price_max));

    return get<any>(`/auctions?${query.toString()}`);
  },
```

- [ ] **Step 2: 类型检查**

Run: `cd frontend/h5 && npx tsc --noEmit`
Expected: 无类型错误。

- [ ] **Step 3: Commit**

```bash
git add frontend/h5/src/services/api.ts
git commit -m "feat(h5): extend auctionApi.list with sort and price params"
```

---

## Task 5: 前端 — 价格底部抽屉组件 PriceFilterSheet

**Files:**
- Create: `frontend/h5/src/pages/Home/PriceFilterSheet.tsx`
- Modify: `frontend/h5/src/pages/Home/Home.module.css` (追加样式)

- [ ] **Step 1: 创建组件**

新建 `frontend/h5/src/pages/Home/PriceFilterSheet.tsx`：

```tsx
import React, { useState } from 'react';
import styles from './Home.module.css';

export interface PriceRange {
  min?: number;
  max?: number;
}

interface PriceFilterSheetProps {
  open: boolean;
  value: PriceRange;
  onClose: () => void;
  onConfirm: (range: PriceRange) => void;
}

const PRESETS: { label: string; range: PriceRange }[] = [
  { label: '不限', range: {} },
  { label: '0 - 1000', range: { min: 0, max: 1000 } },
  { label: '1000 - 5000', range: { min: 1000, max: 5000 } },
  { label: '5000 以上', range: { min: 5000 } },
];

const PriceFilterSheet: React.FC<PriceFilterSheetProps> = ({ open, value, onClose, onConfirm }) => {
  const [minStr, setMinStr] = useState(value.min !== undefined ? String(value.min) : '');
  const [maxStr, setMaxStr] = useState(value.max !== undefined ? String(value.max) : '');

  if (!open) return null;

  const parse = (s: string): number | undefined => {
    if (s.trim() === '') return undefined;
    const n = Number(s);
    return Number.isFinite(n) && n >= 0 ? n : undefined;
  };

  const min = parse(minStr);
  const max = parse(maxStr);
  const invalid =
    (minStr.trim() !== '' && min === undefined) ||
    (maxStr.trim() !== '' && max === undefined) ||
    (min !== undefined && max !== undefined && min > max);

  const applyPreset = (range: PriceRange) => {
    onConfirm(range);
    onClose();
  };

  const applyCustom = () => {
    if (invalid) return;
    onConfirm({ min, max });
    onClose();
  };

  return (
    <div className={styles.sheetOverlay} role="dialog" aria-label="价格区间筛选" onClick={onClose}>
      <div className={styles.sheet} onClick={(e) => e.stopPropagation()}>
        <div className={styles.sheetTitle}>价格区间</div>
        <div className={styles.sheetPresets}>
          {PRESETS.map((p) => (
            <button
              key={p.label}
              type="button"
              className={styles.sheetPreset}
              onClick={() => applyPreset(p.range)}
            >
              {p.label}
            </button>
          ))}
        </div>
        <div className={styles.sheetCustom}>
          <input
            className={styles.sheetInput}
            inputMode="numeric"
            placeholder="最低价"
            value={minStr}
            onChange={(e) => setMinStr(e.target.value)}
            aria-label="最低价"
          />
          <span className={styles.sheetDash}>-</span>
          <input
            className={styles.sheetInput}
            inputMode="numeric"
            placeholder="最高价"
            value={maxStr}
            onChange={(e) => setMaxStr(e.target.value)}
            aria-label="最高价"
          />
        </div>
        {invalid && <p className={styles.sheetError}>请输入有效价格，且最低价不大于最高价</p>}
        <button type="button" className={styles.sheetConfirm} disabled={invalid} onClick={applyCustom}>
          确定
        </button>
      </div>
    </div>
  );
};

export default PriceFilterSheet;
```

- [ ] **Step 2: 追加样式**

在 `frontend/h5/src/pages/Home/Home.module.css` 末尾追加：

```css
.filters {
  display: flex;
  gap: var(--spacing-2);
  padding: var(--spacing-3) var(--spacing-6);
  overflow-x: auto;
  scrollbar-width: none;
}
.filters::-webkit-scrollbar { display: none; }

.filterPill {
  flex: 0 0 auto;
  padding: 6px 14px;
  border-radius: var(--radius-full);
  border: 1px solid rgba(201, 169, 110, 0.28);
  background: var(--bg-elevated);
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  white-space: nowrap;
}
.filterPillActive {
  background: var(--text-brand);
  color: var(--bg-page);
  border-color: var(--text-brand);
}

.sheetOverlay {
  position: fixed;
  inset: 0;
  z-index: 50;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: flex-end;
}
.sheet {
  width: 100%;
  background: var(--bg-elevated);
  border-radius: var(--radius-lg) var(--radius-lg) 0 0;
  padding: var(--spacing-6);
  padding-bottom: calc(var(--spacing-6) + env(safe-area-inset-bottom, 0px));
}
.sheetTitle {
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-bold);
  color: var(--text-primary);
  margin-bottom: var(--spacing-4);
}
.sheetPresets {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--spacing-3);
  margin-bottom: var(--spacing-4);
}
.sheetPreset {
  padding: 10px;
  border-radius: var(--radius-md);
  border: 1px solid rgba(201, 169, 110, 0.28);
  background: var(--bg-page);
  color: var(--text-primary);
  font-size: var(--font-size-sm);
}
.sheetCustom {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  margin-bottom: var(--spacing-3);
}
.sheetInput {
  flex: 1;
  padding: 10px;
  border-radius: var(--radius-md);
  border: 1px solid rgba(201, 169, 110, 0.28);
  background: var(--bg-page);
  color: var(--text-primary);
  font-size: var(--font-size-sm);
}
.sheetDash { color: var(--text-secondary); }
.sheetError {
  margin: 0 0 var(--spacing-3);
  color: #e0533d;
  font-size: var(--font-size-xs);
}
.sheetConfirm {
  width: 100%;
  padding: 12px;
  border-radius: var(--radius-md);
  border: none;
  background: var(--text-brand);
  color: var(--bg-page);
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
}
.sheetConfirm:disabled { opacity: 0.5; }
```

> 注：若 `--font-size-xs` / `--radius-md` 等变量在主题中不存在，复用 `Home.module.css` 已用到的同类变量（参照文件已有 `.tab`、`.iconButton` 的变量名）。

- [ ] **Step 3: 类型检查**

Run: `cd frontend/h5 && npx tsc --noEmit`
Expected: 无类型错误。

- [ ] **Step 4: Commit**

```bash
git add frontend/h5/src/pages/Home/PriceFilterSheet.tsx frontend/h5/src/pages/Home/Home.module.css
git commit -m "feat(h5): add PriceFilterSheet bottom-sheet component"
```

---

## Task 6: 前端 — Home 集成筛选胶囊、状态与参数组装

**Files:**
- Modify: `frontend/h5/src/pages/Home/index.tsx`
- Test: `frontend/h5/src/pages/Home/__tests__/Home.test.tsx` (追加用例)

- [ ] **Step 1: 写失败测试**

在 `frontend/h5/src/pages/Home/__tests__/Home.test.tsx` 末尾（最后一个 `});` 之前的合适位置）追加：

```tsx
  it('点击最热胶囊后以 sort=hot 调用列表接口', async () => {
    mockedUseAuth.mockReturnValue({ isAuthenticated: false } as any);
    mockedProductApi.listCategories.mockResolvedValue({ list: [] });
    mockedAuctionApi.list.mockResolvedValue({ list: [] });

    await act(async () => {
      renderHome();
    });

    await waitFor(() => expect(mockedAuctionApi.list).toHaveBeenCalled());

    const hotPill = screen.getByRole('button', { name: '最热' });
    await act(async () => {
      fireEvent.click(hotPill);
    });

    await waitFor(() =>
      expect(mockedAuctionApi.list).toHaveBeenLastCalledWith(
        expect.objectContaining({ sort: 'hot' })
      )
    );
  });
```

> 若现有测试已统一在文件顶部为 `listCategories`/`list` 设置默认 mock，删除本用例中重复的 mock 行，仅保留断言。

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend/h5 && npx jest src/pages/Home/__tests__/Home.test.tsx -t '点击最热胶囊'`
Expected: FAIL —— 找不到「最热」按钮。

- [ ] **Step 3: 增加 import 与状态**

在 `frontend/h5/src/pages/Home/index.tsx` 顶部 import 区追加：

```tsx
import PriceFilterSheet, { PriceRange } from './PriceFilterSheet';
```

在 `HomePage` 组件内 `const [activeTab, setActiveTab] = useState<string>('全部');` 附近，新增状态：

```tsx
  const [filterSort, setFilterSort] = useState<'default' | 'hot'>('default');
  const [filterPrice, setFilterPrice] = useState<PriceRange>({});
  const [priceSheetOpen, setPriceSheetOpen] = useState(false);
```

- [ ] **Step 4: fetchAuctions 组装参数并按 hot 态跳过重排**

在 `fetchAuctions` 内，`const params: {...}` 处把类型放宽并追加参数。找到：

```tsx
      const params: { page: number; page_size: number; category_id?: number } = {
        page: 1,
        page_size: 20,
      };
      if (activeTab !== '全部') {
        const matched = categories.find((c) => c.name === activeTab);
        if (matched) {
          params.category_id = matched.id;
        }
      }

      const response = await auctionApi.list(params);
      const rawAuctions = extractList<RawAuction>(response);

      setFavoriteLiveStreams([]);
      setAuctions(sortAuctionsForHome(rawAuctions.map((auction) => normalizeAuction(auction))));
```

替换为：

```tsx
      const params: {
        page: number;
        page_size: number;
        category_id?: number;
        sort?: string;
        price_min?: number;
        price_max?: number;
      } = {
        page: 1,
        page_size: 20,
      };
      if (activeTab !== '全部') {
        const matched = categories.find((c) => c.name === activeTab);
        if (matched) {
          params.category_id = matched.id;
        }
      }
      if (filterSort === 'hot') params.sort = 'hot';
      if (filterPrice.min !== undefined) params.price_min = filterPrice.min;
      if (filterPrice.max !== undefined) params.price_max = filterPrice.max;

      const response = await auctionApi.list(params);
      const rawAuctions = extractList<RawAuction>(response);
      const normalized = rawAuctions.map((auction) => normalizeAuction(auction));

      setFavoriteLiveStreams([]);
      // hot 态保留后端排序，跳过 feed 优先级重排，避免覆盖「最热」结果。
      setAuctions(filterSort === 'hot' ? normalized : sortAuctionsForHome(normalized));
```

- [ ] **Step 5: 把 filterSort/filterPrice 加入 useCallback 依赖**

找到 `fetchAuctions` 的 `useCallback(... , [activeTab, categories]);`，改为：

```tsx
  }, [activeTab, categories, filterSort, filterPrice]);
```

- [ ] **Step 6: 渲染筛选胶囊与抽屉**

在 `<nav className={styles.tabs} ...>...</nav>` 之后、`<main className={styles.content} ...>` 之前，插入（收藏态隐藏筛选）：

```tsx
      {activeTab !== '收藏' && (
        <div className={styles.filters} aria-label="排序与价格筛选">
          <button
            type="button"
            className={`${styles.filterPill} ${filterSort === 'default' ? styles.filterPillActive : ''}`}
            onClick={() => setFilterSort('default')}
          >
            综合
          </button>
          <button
            type="button"
            className={`${styles.filterPill} ${filterSort === 'hot' ? styles.filterPillActive : ''}`}
            onClick={() => setFilterSort('hot')}
          >
            最热
          </button>
          <button
            type="button"
            className={`${styles.filterPill} ${
              filterPrice.min !== undefined || filterPrice.max !== undefined ? styles.filterPillActive : ''
            }`}
            onClick={() => setPriceSheetOpen(true)}
          >
            {filterPrice.min !== undefined || filterPrice.max !== undefined
              ? `¥${filterPrice.min ?? 0}${filterPrice.max !== undefined ? `-${filterPrice.max}` : '+'}`
              : '价格区间'}
          </button>
        </div>
      )}

      <PriceFilterSheet
        open={priceSheetOpen}
        value={filterPrice}
        onClose={() => setPriceSheetOpen(false)}
        onConfirm={(range) => setFilterPrice(range)}
      />
```

- [ ] **Step 7: 运行新增测试确认通过**

Run: `cd frontend/h5 && npx jest src/pages/Home/__tests__/Home.test.tsx -t '点击最热胶囊'`
Expected: PASS。

- [ ] **Step 8: 跑全量 Home 测试防回归**

Run: `cd frontend/h5 && npx jest src/pages/Home/__tests__/Home.test.tsx`
Expected: 全部 PASS。

- [ ] **Step 9: Commit**

```bash
git add frontend/h5/src/pages/Home/index.tsx frontend/h5/src/pages/Home/__tests__/Home.test.tsx
git commit -m "feat(h5): integrate filter pills and price sheet into Home"
```

---

## Task 7: 端到端验收（手动）

**Files:** 无（验证）

- [ ] **Step 1: 启动后端与前端**

按 [STARTUP_GUIDE.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/STARTUP_GUIDE.md) 启动 gateway + auction-service + h5。

- [ ] **Step 2: 浏览器验收**

在首页验证：
1. 分类 Tab 下出现「综合 / 最热 / 价格区间」胶囊。
2. 点「最热」→ 列表按出价数降序刷新（网络请求 query 带 `sort=hot`）。
3. 点「价格区间」→ 底部抽屉弹出；选预设或自定义后胶囊高亮显示区间，列表刷新（query 带 `price_min`/`price_max`）。
4. 自定义输入 min>max → 确定按钮禁用并提示。
5. 切到「收藏」Tab → 筛选胶囊隐藏。
6. 切换日/夜间主题 → 胶囊与抽屉样式正常。

- [ ] **Step 3: 全量回归**

Run: `cd backend/auction && go test ./... && cd ../../frontend/h5 && npx jest`
Expected: 全部 PASS。

---

## 风险与备注

- **GROUP BY 与 total 解耦**：`total` 在加 JOIN/GROUP BY 之前由 `query.Count(&total)` 算出，故热度排序不影响总数正确性。价格过滤的 `WHERE` 加在 `query` 上，会同时作用于 count 与 list（正确）。
- **BidCount 只读列**：仅 hot 排序 SELECT `COUNT(bids.id) AS bid_count`，其他查询该字段为 0；`gorm:"->;-:migration"` 确保不参与建表/写入。
- **current_price=0 的预告场次**：选择 `price_min>0` 区间会过滤掉预告场次，属设计已接受的简化口径。
- **CSS 变量名**：Task 5 样式引用的变量须与 `Home.module.css` 现有变量对齐，缺失时就近复用同类变量。

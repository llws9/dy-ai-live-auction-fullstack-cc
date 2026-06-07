# 一口价商品选择：手填 ID 改为下拉选商品 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 admin 一口价上架页的"商品 ID"手填输入框改为下拉选商品（自有+已发布+未在拍），并在 auction service 增加"商品正在 active 竞拍则拒绝上架"的失败关闭校验。

**Architecture:** 前端 `LiveStreamFixedPrice` 在直播间确定后调 `productApi.list({ display_status: 'schedulable' })` 拉可售商品填充下拉，提交时取选中项 `product_id`；后端 `FixedPriceService.ListItem` 在商品存在校验后，复用 `AuctionDAO.GetActiveByProductID` 判断是否有 active 竞拍占用，有则返回新错误 `ErrProductInAuction`，handler 映射 409 `FP_PRODUCT_IN_AUCTION`。

**Tech Stack:** Go (Hertz, GORM) auction service；React + TypeScript + Jest（admin 前端）；Playwright（E2E）。

---

## File Structure

后端（auction service）：
- `backend/auction/service/fixed_price.go` — 新增 `ErrProductInAuction`、扩展 `AuctionChecker` 接口、`ListItem` 增加 active 竞拍校验。
- `backend/auction/service/fixed_price_testutil_test.go` — `fakeAuctionChecker` 实现新接口方法。
- `backend/auction/service/fixed_price_test.go` — 新增 service RED 测试。
- `backend/auction/handler/fixed_price.go` — 映射 `ErrProductInAuction` → 409 `FP_PRODUCT_IN_AUCTION`。
- `backend/auction/handler/fixed_price_test.go` — 新增 handler 映射测试（若不存在则在 service 测试覆盖即可，见 Task 4）。

前端（admin）：
- `frontend/admin/src/pages/LiveStreamFixedPrice/index.tsx` — 删除商品 ID input，替换为商品下拉，拉取可售商品列表。
- `frontend/admin/src/pages/LiveStreamFixedPrice/__tests__/LiveStreamFixedPrice.test.tsx` — 更新 mock 与断言。

---

## Task 1: 后端 — 扩展 AuctionChecker 接口与 fake

**Files:**
- Modify: `backend/auction/service/fixed_price.go:48-51`
- Modify: `backend/auction/service/fixed_price_testutil_test.go:49-69`

- [ ] **Step 1: 扩展 `AuctionChecker` 接口，新增 active 查询方法**

修改 `backend/auction/service/fixed_price.go` 的接口定义：

```go
// AuctionChecker 校验一口价绑定的竞拍场次是否属于当前直播间与商家，
// 并支持查询某商品是否正处于 active 竞拍（失败关闭）。
type AuctionChecker interface {
	GetByID(ctx context.Context, id int64) (*model.Auction, error)
	GetActiveByProductID(ctx context.Context, productID int64) (*model.Auction, error)
}
```

> 说明：生产实现 `*dao.AuctionDAO` 已有 `GetActiveByProductID`（见 `backend/auction/dao/auction.go:88`），无需改 DAO；`main.go:238` 注入的 `auctionDAO` 自动满足新接口。

- [ ] **Step 2: 给 `fakeAuctionChecker` 增加 `activeByProduct` 字段与方法**

修改 `backend/auction/service/fixed_price_testutil_test.go`，把 struct 与方法改为：

在现有 `fakeAuctionChecker` struct（第 49-52 行）增加一个字段，并新增方法。改为：

```go
type fakeAuctionChecker struct {
	auctions        map[int64]*model.Auction
	missing         map[int64]bool
	activeByProduct map[int64]*model.Auction
}
```

`GetByID` 方法保持原样不动。在其后新增：

```go
func (f *fakeAuctionChecker) GetActiveByProductID(_ context.Context, productID int64) (*model.Auction, error) {
	if f == nil || f.activeByProduct == nil {
		return nil, nil
	}
	return f.activeByProduct[productID], nil
}
```

> 注意：`GetActiveByProductID` 约定"无 active 竞拍"返回 `(nil, nil)`，与 DAO 行为一致（DAO 在无记录时返回 nil, nil，见 `auction.go`）。

- [ ] **Step 3: 编译验证（此时尚无业务逻辑变化，仅接口扩展）**

Run: `cd backend/auction && go build ./... && go vet ./service/`
Expected: 编译通过（service 包还未调用新方法，但接口已扩展，fake 已实现）。

- [ ] **Step 4: Commit**

```bash
git add backend/auction/service/fixed_price.go backend/auction/service/fixed_price_testutil_test.go
git commit -m "refactor(auction): extend AuctionChecker with GetActiveByProductID"
```

---

## Task 2: 后端 — service 失败关闭校验（TDD RED）

**Files:**
- Modify: `backend/auction/service/fixed_price.go:18-26`（新增错误）, `:136-183`（ListItem 加校验）
- Test: `backend/auction/service/fixed_price_test.go`

- [ ] **Step 1: 写失败测试**

在 `backend/auction/service/fixed_price_test.go` 末尾新增。文件顶部已 import `context`/`testing`/`decimal`/`model`/`assert`/`require`。镜像现有 `TestFixedPriceService_List_RejectsAuctionFromOtherLiveStream` 的构造方式（真实 db/redis helper + 11 个位置参数）：

```go
func TestFixedPriceService_List_RejectsProductInActiveAuction(t *testing.T) {
	liveStreamID := int64(1001)
	creatorID := int64(100)
	db := setupServiceDB(t)
	rdb := setupTestRedis(t)
	otherLiveStream := int64(2002)
	svc := NewFixedPriceService(
		db,
		newItemDAO(db), newPurchaseDAO(db), newBalanceDAO(db),
		NewStockGuard(rdb), NewIdemStore(rdb),
		&fakeStreamOwner{owners: map[int64]int64{liveStreamID: creatorID}},
		&fakeProductChecker{},
		&fakeAuctionChecker{
			auctions: map[int64]*model.Auction{
				7001: {ID: 7001, LiveStreamID: &liveStreamID, CreatorID: &creatorID, Status: model.AuctionStatusOngoing},
			},
			activeByProduct: map[int64]*model.Auction{
				5002: {ID: 9999, LiveStreamID: &otherLiveStream, CreatorID: &creatorID, Status: model.AuctionStatusOngoing},
			},
		},
		nil,
		nil,
	)

	_, err := svc.ListItem(context.Background(), ListItemReq{
		AuctionID: 7001, LiveStreamID: liveStreamID, ProductID: 5002, CreatorID: creatorID,
		Price: decimal.NewFromInt(99), TotalStock: 10,
	})

	assert.ErrorIs(t, err, ErrProductInAuction)
}
```

> 该测试绑定的 auction(7001) 归属正确（通过 `validateAuctionBinding`），但搭售商品 5002 自身正处于另一场 active 竞拍(9999)，故应被 `ErrProductInAuction` 拒绝。构造辅助 `setupServiceDB`/`setupTestRedis`/`newItemDAO`/`newPurchaseDAO`/`newBalanceDAO`/`fakeStreamOwner`/`fakeProductChecker` 均为现有测试已用名称（见 `fixed_price_test.go` 第 95-118 行同款写法）。

- [ ] **Step 2: 运行测试，确认失败**

Run: `cd backend/auction && go test ./service/ -run TestFixedPriceService_List_RejectsProductInActiveAuction -v`
Expected: 编译失败或断言失败 —— `ErrProductInAuction` 未定义 / `ListItem` 未返回该错误。

- [ ] **Step 3: 新增错误并实现校验**

在 `backend/auction/service/fixed_price.go` 错误块（`var (...)`，第 17-26 行）新增一行：

```go
	ErrProductInAuction    = errors.New("product is in active auction")
```

在 `ListItem` 中，于 `products.Exists` 校验通过之后、`if r.MaxPerUser <= 0` 之前插入：

```go
	if s.auctions != nil {
		active, err := s.auctions.GetActiveByProductID(ctx, r.ProductID)
		if err != nil {
			return nil, err
		}
		if active != nil {
			return nil, ErrProductInAuction
		}
	}
```

> 位置参考：当前 `fixed_price.go` 第 150-159 行：
> ```go
> exists, err := s.products.Exists(ctx, r.ProductID)
> if err != nil { return nil, err }
> if !exists { return nil, ErrProductNotFound }
> // <-- 在此插入上面的 active 竞拍校验
> if r.MaxPerUser <= 0 { r.MaxPerUser = 1 }
> ```

- [ ] **Step 4: 运行测试，确认通过**

Run: `cd backend/auction && go test ./service/ -run TestFixedPriceService_List -v`
Expected: PASS（新测试通过，且既有 `ListItem` 相关测试不回归）。

- [ ] **Step 5: 跑 service 全量测试防回归**

Run: `cd backend/auction && go test ./service/`
Expected: ok（全部通过）。

- [ ] **Step 6: Commit**

```bash
git add backend/auction/service/fixed_price.go backend/auction/service/fixed_price_test.go
git commit -m "feat(auction): reject fixed-price listing when product is in active auction"
```

---

## Task 3: 后端 — handler 错误码映射

**Files:**
- Modify: `backend/auction/handler/fixed_price.go:231-241`

- [ ] **Step 1: 在 `List` 的 error switch 中映射新错误**

在 `backend/auction/handler/fixed_price.go` 的 `List` 方法 switch（第 231-241 行），在 `ErrAuctionNotAvailable` case 之后、`default` 之前插入：

```go
	case errors.Is(err, service.ErrProductInAuction):
		writeFPErr(c, 409, "FP_PRODUCT_IN_AUCTION", "该商品正在参与竞拍，无法用于一口价", nil)
```

- [ ] **Step 2: 编译验证**

Run: `cd backend/auction && go build ./...`
Expected: 编译通过。

- [ ] **Step 3: 运行 handler 包测试防回归**

Run: `cd backend/auction && go test ./handler/`
Expected: ok。

- [ ] **Step 4: Commit**

```bash
git add backend/auction/handler/fixed_price.go
git commit -m "feat(auction): map ErrProductInAuction to 409 FP_PRODUCT_IN_AUCTION"
```

---

## Task 4: 前端 — 商品下拉（TDD RED）

**Files:**
- Modify: `frontend/admin/src/pages/LiveStreamFixedPrice/index.tsx`
- Test: `frontend/admin/src/pages/LiveStreamFixedPrice/__tests__/LiveStreamFixedPrice.test.tsx`

- [ ] **Step 1: 更新测试 mock，加入 productApi.list**

修改测试文件顶部 `jest.mock('@/shared/api', ...)`，新增 `productApi`：

```ts
jest.mock('@/shared/api', () => ({
  auctionApi: {
    list: jest.fn(),
  },
  liveStreamApi: {
    adminList: jest.fn(),
  },
  productApi: {
    list: jest.fn(),
  },
  fixedPriceAdminApi: {
    list: jest.fn(),
    listItem: jest.fn(),
    offline: jest.fn(),
  },
}))
```

在 mock 句柄区新增：

```ts
import { auctionApi, fixedPriceAdminApi, liveStreamApi, productApi } from '@/shared/api'
// ...
const productListMock = productApi.list as jest.Mock
```

在 `beforeEach` 中新增默认返回（两件可售商品）：

```ts
    productListMock.mockResolvedValue({
      list: [
        { id: 5002, name: '搭售周边A', status: 1, display_status: 'schedulable' },
        { id: 5003, name: '搭售周边B', status: 1, display_status: 'schedulable' },
      ],
      total: 2,
      page: 1,
      page_size: 100,
    })
```

- [ ] **Step 2: 改写"adds a listed row"测试为下拉选择**

把现有 `it('adds a listed row after listing succeeds', ...)` 中对"商品 ID" input 的操作改为下拉选择：

```ts
  it('adds a listed row after listing succeeds', async () => {
    listItemMock.mockResolvedValue({
      id: 7002,
      auction_id: 8001,
      remaining_stock: 5,
      status: 'on_sale',
    })

    renderPage()

    await waitFor(() => expect(screen.getByLabelText('竞拍场次')).toHaveValue('8001'))
    await waitFor(() => expect(productListMock).toHaveBeenCalledWith({ display_status: 'schedulable', page: 1, page_size: 100 }))
    fireEvent.change(screen.getByLabelText('搭售商品'), { target: { value: '5002' } })
    fireEvent.change(screen.getByLabelText('一口价'), { target: { value: '199.00' } })
    fireEvent.change(screen.getByLabelText('库存'), { target: { value: '5' } })
    fireEvent.click(screen.getByRole('button', { name: '新增上架' }))

    await waitFor(() => {
      expect(listItemMock).toHaveBeenCalledWith(1001, {
        auction_id: 8001,
        product_id: 5002,
        price: '199.00',
        stock: 5,
      })
    })
    expect(await screen.findByText('搭售周边A')).toBeInTheDocument()
    expect(screen.getByText('¥199.00')).toBeInTheDocument()
    expect(screen.getByText('5 / 5')).toBeInTheDocument()
  })
```

> 说明：上架成功后行内展示走 `getProductTitle`，回显 `商品 #5002`（因 `listItem` 返回不含 title）。这里断言改为更稳妥的 `¥199.00` 与 `5 / 5`；`搭售周边A` 的断言仅在前端把下拉所选 name 注入新行时成立——若实现不注入 name，则删除该行断言，保留价格/库存断言。实现时见 Step 4 决定是否注入。

- [ ] **Step 3: 新增"无可售商品时下拉禁用"测试**

```ts
  it('disables product select when no schedulable product available', async () => {
    productListMock.mockResolvedValueOnce({ list: [], total: 0, page: 1, page_size: 100 })

    renderPage()

    const select = await screen.findByLabelText('搭售商品')
    await waitFor(() => expect(select).toBeDisabled())
    expect(screen.getByText('暂无可搭售商品，请先创建并发布商品')).toBeInTheDocument()
  })
```

- [ ] **Step 4: 运行测试，确认失败**

Run: `cd frontend/admin && npx jest src/pages/LiveStreamFixedPrice -t "LiveStreamFixedPrice"`
Expected: FAIL —— 找不到 `搭售商品` label / `productApi.list` 未被调用。

- [ ] **Step 5: 实现前端商品下拉**

修改 `frontend/admin/src/pages/LiveStreamFixedPrice/index.tsx`：

(a) import 增加 `productApi` 与 `Product` 类型：

```tsx
import {
  auctionApi,
  fixedPriceAdminApi,
  liveStreamApi,
  productApi,
  type FixedPriceAdminItem,
  type FixedPriceAdminStatus,
  type Product,
} from "@/shared/api"
```

> 若 `@/shared/api` 未 re-export `Product`/`productApi`，改为 `import { productApi } from "@/shared/api/product"` 和 `import type { Product } from "@/shared/api/types"`。先 Grep 确认 `frontend/admin/src/shared/api/index.ts` 是否导出 `productApi`、`Product`。

(b) 新增状态：

```tsx
  const [productOptions, setProductOptions] = React.useState<Product[]>([])
```

(c) 在 `fetchItems` 的 `Promise.all` 中加入商品拉取，并填充下拉默认值。把现有 `Promise.all([...])` 改为三项：

```tsx
      const [response, auctionResponse, productResponse] = await Promise.all([
        fixedPriceAdminApi.list(liveStreamId, { page: 1, page_size: pageSize }),
        auctionApi.list({ live_stream_id: liveStreamId, page: 1, page_size: 100 }),
        productApi.list({ display_status: 'schedulable', page: 1, page_size: 100 }),
      ])
      const nextItems = (response.items || []).map(normalizeItem)
      const nextAuctions = (auctionResponse.list || []).filter((auction: any) => [0, 1, 2].includes(Number(auction.status)))
      const nextProducts = productResponse.list || []
      setItems(nextItems)
      setTotal(response.total ?? nextItems.length)
      setAuctionOptions(nextAuctions)
      setProductOptions(nextProducts)
      setAuctionId((current) => current || (nextAuctions[0]?.id ? String(nextAuctions[0].id) : ""))
      setProductId((current) => current || (nextProducts[0]?.id ? String(nextProducts[0].id) : ""))
```

(d) 把"商品 ID"label 块（第 232-242 行）整体替换为下拉：

```tsx
            <label className="space-y-2 text-sm font-medium text-slate-700">
              搭售商品
              <select
                aria-label="搭售商品"
                className="h-10 w-full rounded-md border border-slate-200 bg-white px-3 text-sm"
                value={productId}
                onChange={(event) => setProductId(event.target.value)}
                required
                disabled={productOptions.length === 0}
              >
                {productOptions.length === 0 ? (
                  <option value="">暂无可搭售商品，请先创建并发布商品</option>
                ) : (
                  productOptions.map((product) => (
                    <option key={product.id} value={product.id}>
                      {product.name}（#{product.id}）
                    </option>
                  ))
                )}
              </select>
            </label>
```

(e) 提交按钮 `disabled` 增加无商品禁用：

```tsx
            <Button type="submit" disabled={submitting || !auctionId || productOptions.length === 0} className="self-end bg-amber-500 text-[#0f172a] hover:bg-amber-600">
```

(f) `Input` import 若仅此处使用且其他字段（一口价/库存）仍在用，则保留 import；不要误删。

> 关于 Step 2 中 `搭售周边A` 行回显：`handleSubmit` 成功后构造 `completedItem`，可选地用所选商品 name 注入 `product_title`：在 `handleSubmit` 内 `Number(productId)` 处取 `productOptions.find(p => String(p.id) === productId)?.name`，赋给 `completedItem.product_title`。若实现注入，则 Step 2 的 `搭售周边A` 断言成立；若不注入，删除该断言。建议注入以提升体验。

- [ ] **Step 6: 运行测试，确认通过**

Run: `cd frontend/admin && npx jest src/pages/LiveStreamFixedPrice`
Expected: PASS（全部用例通过）。

- [ ] **Step 7: 跑 admin 聚焦测试 + 构建防回归**

Run: `cd frontend/admin && npx jest src/pages/LiveStreamFixedPrice src/shared/api && npm run build`
Expected: 测试 PASS，build 成功。

- [ ] **Step 8: Commit**

```bash
git add frontend/admin/src/pages/LiveStreamFixedPrice/index.tsx frontend/admin/src/pages/LiveStreamFixedPrice/__tests__/LiveStreamFixedPrice.test.tsx
git commit -m "feat(admin): replace fixed-price product id input with product picker"
```

---

## Task 5: E2E 验证（Playwright 真实浏览器）

**Files:**
- 沿用现有演示闭环 Playwright 脚本（上一轮 SDD 已建），仅把"手填 product_id"步骤改为"下拉选商品"。

- [ ] **Step 1: 确认本地三服务在跑**

Run: `curl -s http://localhost:8080/health || curl -s http://localhost:8080/api/v1/products`
Expected: 返回 200/JSON（gateway 在跑）。若未跑，按 `AGENTS.md` 本地启动方式用环境变量 `go run main.go` 拉起 gateway(8080)/product(8081)/auction(8082/8083)。

- [ ] **Step 2: 跑链路：商家登录 → 直播间控制台 → 创建竞拍 → 一口价下拉选搭售商品 → 开播 → H5 直播间可见**

把一口价步骤中 `page.fill('[placeholder="例如 5001"]', ...)` 改为：

```ts
await page.selectOption('select[aria-label="搭售商品"]', { index: 0 })
```

Run: 现有 E2E 脚本命令（沿用上轮，例如 `node scripts/e2e/merchant-demo-loop.mjs` 或对应 Playwright runner）。
Expected: 输出 JSON 含 `"h5Verified": true`，且一口价 item 成功上架（无 409）。

- [ ] **Step 3: 反向验证失败关闭（可选但推荐）**

手工或脚本：尝试对一件 `display_status=auctioning` 的商品调 `POST /api/v1/fixed-price/items`（带其 product_id）。
Expected: 返回 409，body `code=FP_PRODUCT_IN_AUCTION`。

- [ ] **Step 4: 清理本地临时产物**

Run: `git status --short && git diff --check`
Expected: 无意外改动（如 `node_modules/.vite/deps/_metadata.json` 被改则 `git restore` 还原），无空白错误。

---

## 自审记录

- Spec §3（前端下拉 + schedulable 过滤 + 空态禁用）→ Task 4 覆盖。
- Spec §4（service 失败关闭 + handler 409 映射）→ Task 1/2/3 覆盖。
- Spec §6（TDD：service RED、handler 映射、前端下拉、E2E）→ Task 2/3/4/5 覆盖。
- Spec §7（不新增 category、不改 migration）→ 计划未触碰 category 与 migration 文件。
- 类型一致性：`GetActiveByProductID(ctx, productID) (*model.Auction, error)` 在接口、fake、service 调用、DAO 现有实现四处一致；前端 `aria-label="搭售商品"`、`product_id` 字段在测试与实现一致。

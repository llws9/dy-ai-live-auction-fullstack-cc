# 一口价商品选择：手填 ID 改为下拉选商品

- 日期：2026-06-08
- 范围：admin 一口价上下架页 + auction service 一口价上架校验
- 目标：把一口价上架时的"商品 ID"手填输入框，改为从商家货品库下拉选商品，并补一条后端"失败关闭"校验。

## 1. 背景与问题

一口价上下架页 [LiveStreamFixedPrice/index.tsx](../../../frontend/admin/src/pages/LiveStreamFixedPrice/index.tsx) 当前用 `<input type="number" placeholder="例如 5001">` 让商家手填 `product_id`。手填一个可能对不上的 ID 是 bug 温床，且无引导。

后端一口价 item 在 [fixed_price.go](../../../backend/auction/service/fixed_price.go) 的 `validateAuctionBinding` 已强制绑定有效 `auction_id`（校验 auction 归属当前直播间 + 当前商家，状态须为 Pending/Ongoing/Delayed）。但对"该 product 是否正被别的竞拍占用"不设防。

## 2. 核心决策（第一性原理）

- **一口价卖的是和竞拍不同的另一件商品（搭售）**，因此保留"选商品"，不删该字段，只改交互。
- **复用同一货品库，不新增分类/商品类型。** `category` 是商品品类（服饰/数码…），与"销售方式"（竞拍/一口价）正交；用 category 区分销售方式会污染品类体系。商品列表 = 货品库，竞拍与一口价都是从同一库取货的两种卖法。
- **下拉只显示自有 + 已发布 + 未在竞拍的商品**，即 admin product list 的 `display_status=schedulable`。
- **后端失败关闭兜底**：上架商品若正处于 active 竞拍（status∈{Pending,Ongoing,Delayed}），拒绝上架。前端过滤只是引导，后端不可绕过。

## 3. 前端改动（admin）

文件：[LiveStreamFixedPrice/index.tsx](../../../frontend/admin/src/pages/LiveStreamFixedPrice/index.tsx)

- 删除"商品 ID"`<input type="number">`，替换为商品下拉 `<select aria-label="搭售商品">`。
- 直播间确定后，调 `productApi.list({ display_status: 'schedulable', page_size: ... })` 拉取本商家可售商品填充下拉。
- 选项文案 `商品名（#id）`，提交时取选中项 `product_id`。
- 空态：无可售商品时下拉禁用，提示"暂无可搭售商品，请先创建并发布商品"。
- 竞拍场次下拉、`auction_id` 上架逻辑保持不变。
- API 层 [shared/api/index.ts](../../../frontend/admin/src/shared/api/index.ts) `fixedPriceAdminApi.listItem` 契约不变（仍带 `auction_id`/`product_id`/`price`/`total_stock`/`max_per_user`）。

## 4. 后端改动（auction service）— 失败关闭

文件：[fixed_price.go](../../../backend/auction/service/fixed_price.go)

- 新增业务错误 `ErrProductInAuction = errors.New("product is in active auction")`。
- `ListItem` 在 `products.Exists` 通过之后，新增校验：若 `product_id` 当前存在 active 竞拍，返回 `ErrProductInAuction`。
- 复用 DAO 已有的 `AuctionDAO.GetActiveByProductID`（见 [auction.go](../../../backend/auction/dao/auction.go)）。在 `AuctionChecker` 接口新增 `GetActiveByProductID(ctx, productID) (*model.Auction, error)`，由现有 auction DAO 适配实现。

文件：[handler/fixed_price.go](../../../backend/auction/handler/fixed_price.go)

- 映射 `ErrProductInAuction` → 错误码 `FP_PRODUCT_IN_AUCTION`，HTTP 409。

## 5. 数据流

商家进页 → 拉竞拍场次 + 拉可售商品（schedulable）→ 选场次 + 选搭售商品 + 填价/库存 → 提交带 `auction_id`+`product_id` → 后端校验（归属 + 商品存在 + 非在拍）→ 落库上架 → H5 一口价区可见。

## 6. 测试（TDD）

- 后端 service：RED 测试——上架一件正处于 active 竞拍的商品应返回 `ErrProductInAuction`；正常 schedulable 商品上架成功。
- 后端 handler：映射 409 / `FP_PRODUCT_IN_AUCTION`。
- 前端：[LiveStreamFixedPrice.test.tsx](../../../frontend/admin/src/pages/LiveStreamFixedPrice/__tests__/LiveStreamFixedPrice.test.tsx) 更新——渲染商品下拉、选择后提交带正确 `product_id`；无商品时下拉禁用。
- E2E：沿用现有 Playwright 链路，把"手填 product_id"换成"下拉选商品"。

## 7. 不做（YAGNI）

- 不新增 category / 商品类型。
- 不引入"一口价专用商品"概念。
- 不改竞拍唯一约束 migration（`2026060702_add_active_live_stream_unique_to_auctions`）。

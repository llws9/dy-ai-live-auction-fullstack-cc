# Homepage Filter Design

## 1. Overview
在 H5 首页分类 Tab 下方新增「筛选胶囊 (Pills)」，支持按「综合」「最热」排序，以及按「价格区间」过滤竞拍商品，实现用户分流，提升查找效率。

UI 方案：胶囊横滑 + 价格区间底部抽屉 (Bottom Sheet)，复用现有 CSS Variables 适配日夜间模式。

## 2. 关键决策与数据口径（已拍板）

| 维度 | 口径 | 理由 |
|---|---|---|
| 热度 | `bids` 表出价次数聚合，DESC 排序 | `bids` 与 `auctions` 同库，auction-service 单服务内 JOIN 即可；预告/直播/结束三态都有意义 |
| 价格过滤 | `auctions.current_price` 区间过滤 | 同库、单服务，与热度排序在同一查询实现，最省路径 |

**已知边界（接受）**：`current_price` 对「即将开始」的场次为 0。因此价格过滤会把预告场次按 0 计入；当选择带 `price_min > 0` 的区间时，预告场次会被过滤掉。此为可接受的简化口径。

## 3. UI / UX Design
- **位置**: [index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Home/index.tsx) 中，位于分类 Tab (`nav.tabs`) 下方、商品列表 (`main.content`) 上方，新增一行横滑筛选区。
- **样式**:
  - 横向滑动容器，隐藏滚动条。
  - 胶囊 (Pill)：未选中灰底，选中时品牌主题色高亮，复用现有 CSS Variables 自动适配日夜间。
- **筛选项与交互**:
  - **综合**: 默认选中，不传排序参数，沿用后端默认的 feed 优先级排序。
  - **最热**: 点击高亮，列表按出价次数降序。
  - **价格区间**: 点击呼出底部抽屉，提供预设区间（0-1000 / 1000-5000 / 5000以上）及自定义输入；选中后抽屉收起，胶囊高亮并显示当前区间文案。
- **收藏 Tab 互斥**: `activeTab === '收藏'` 走 `followApi` 而非 auction list，筛选胶囊在收藏态隐藏（不渲染）。
- **自定义价格校验**: min/max 须为非负数；若同时填写则需 `min <= max`，否则禁用确认并提示；非数字输入忽略。

## 4. State Management
在 `HomePage` 组件新增状态：
- `filterSort`: `'default' | 'hot'`，默认 `'default'`
- `filterPrice`: `{ min?: number; max?: number }`，默认空对象

切换分类 Tab 时是否保留筛选条件：**保留**（仅切到「收藏」时筛选 UI 隐藏，状态不重置）。

## 5. Data Flow & API（后端为必做改动）

### 5.1 现状（确定缺失）
- [ListParams](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction_list.go#L22-L31) 无排序与价格参数。
- [orderByAuctionFeedPriority](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/dao/auction.go#L28-L35) 排序写死。
- list 响应 `AuctionListItem` 不含 `bid_count`。

### 5.2 API 契约变更（GET /api/v1/auctions）
新增 query 参数：

| 参数 | 类型 | 说明 |
|---|---|---|
| `sort` | string | 可选，枚举 `hot`；缺省或其他值走默认 feed 排序 |
| `price_min` | number | 可选，按 `current_price >=` 过滤 |
| `price_max` | number | 可选，按 `current_price <=` 过滤 |

响应新增字段：`AuctionListItem` 增加 `bid_count`（int），供前端展示与排序验证。

### 5.3 后端改动点
1. [handler/auction.go List](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction.go#L385) 解析 `sort` / `price_min` / `price_max`，归一化进 `ListParams`。
2. `ListParams` 与 `dao.AuctionFilters` 新增 `SortByHot bool`、`PriceMin *decimal.Decimal`、`PriceMax *decimal.Decimal`。
3. [dao ListWithFilters](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/dao/auction.go) 中：
   - 价格过滤：`WHERE current_price >= ? / <= ?`。
   - 热度排序：`LEFT JOIN bids b ON b.auction_id = auctions.id GROUP BY auctions.id ORDER BY COUNT(b.id) DESC, auctions.id DESC`，替代 `orderByAuctionFeedPriority`。
   - 同时回传每条的 `COUNT(b.id) AS bid_count`。
   - 金额比较统一使用 `shopspring/decimal`，不得用 float。
4. 列表响应回填 `bid_count`。

### 5.4 前端改动点
1. [api.ts auctionApi.list](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts#L350) 参数扩展 `sort` / `price_min` / `price_max`。
2. `fetchAuctions` 组装参数：
   ```typescript
   if (filterSort === 'hot') params.sort = 'hot';
   if (filterPrice.min !== undefined) params.price_min = filterPrice.min;
   if (filterPrice.max !== undefined) params.price_max = filterPrice.max;
   ```
3. **排序冲突处理（关键）**: 现有 `sortAuctionsForHome` 会按 feed 优先级强制客户端重排，会覆盖后端「最热」结果。当 `filterSort === 'hot'` 时**跳过 `sortAuctionsForHome`**，直接使用后端返回顺序；`default` 时保持现有行为。
4. **空状态**: 筛选后无数据复用现有 `empty` UI，文案改为「暂无符合条件的竞拍」。

## 6. Testing Strategy
- 后端：`ListWithFilters` 在 hot 排序、价格区间、两者组合下的 SQL 行为（含 `bid_count` 计数正确）。遵循 TDD，先写失败测试。
- 前端：胶囊点击更新状态并触发请求；hot 态跳过客户端重排；底部抽屉自定义价格校验（min>max、负数、非数字）；清除筛选恢复默认；收藏态隐藏筛选 UI。
- UI：日间/夜间模式样式；底部抽屉在不同屏幕尺寸的展示完整性。

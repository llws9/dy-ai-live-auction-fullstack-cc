# Homepage Filter Design

## 1. Overview
在 H5 首页分类 Tab 下方新增「筛选胶囊 (Pills)」，支持按照「综合」、「最热」和「价格区间」对竞拍商品进行过滤与排序，实现用户分流，提升查找效率。

## 2. UI / UX Design
- **位置**: `frontend/h5/src/pages/Home/index.tsx` 中，位于分类 Tab (`nav.tabs`) 下方，商品列表 (`main.content`) 上方。
- **样式**:
  - 横向滑动容器，隐藏滚动条。
  - 胶囊 (Pill) 样式：未选中时为灰色背景，选中时为品牌主题色 (Sky/Gold depending on theme) 高亮。
  - 支持日夜间模式自动适配（复用现有的 CSS Variables）。
- **交互**:
  - **综合**: 默认选中，无特殊参数。
  - **最热**: 点击后高亮，列表按热度（如出价次数 `bidCount`）降序排列。
  - **价格区间**: 点击后呼出底部抽屉 (Bottom Sheet)。抽屉内提供预设价格区间（如 0-1000, 1000-5000, 5000以上）及自定义输入框。选中后，抽屉收起，胶囊高亮并显示选中的具体金额范围。

## 3. State Management
在 `HomePage` 组件中新增状态：
- `filterSort`: 排序维度，枚举值 `'default' | 'hot'`
- `filterPrice`: 价格区间对象，形如 `{ min?: number, max?: number }`

## 4. Data Flow & API
- **API 修改/确认**:
  - 确认后端 `auctionApi.list` (通常对应 `GET /api/v1/auctions`) 是否已支持 `sort` (或 `order_by`) 以及 `price_min`, `price_max` 参数。如果不支持，需同步更新后端。
  - 在前端 `fetchAuctions` 方法中，根据 `filterSort` 和 `filterPrice` 组装参数：
    ```typescript
    const params: any = { page: 1, page_size: 20 };
    if (activeTab !== '全部') { /* ... */ }
    if (filterSort === 'hot') params.sort = 'hot'; // 或者是后端约定的枚举
    if (filterPrice?.min !== undefined) params.price_min = filterPrice.min;
    if (filterPrice?.max !== undefined) params.price_max = filterPrice.max;
    ```
- **空状态处理**: 若选择某价格区间后无数据，复用现有的 `empty` 状态 UI，并提示 "暂无符合条件的竞拍"。

## 5. Testing Strategy
- 验证日间/夜间模式下的 UI 样式是否正常。
- 点击各胶囊是否正确更新状态，并触发接口请求。
- 底部抽屉在不同设备屏幕尺寸上的展示是否完整，输入自定义价格后是否能正确过滤。
- 清除筛选条件后，列表是否能正确恢复到默认状态。

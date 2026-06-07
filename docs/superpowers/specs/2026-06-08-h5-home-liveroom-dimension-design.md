# H5 首页直播间维度重构 - Design Spec

> 方案A（首页直播间维度卡片） + ①职责分层（首页 vs 直播 feed 边界划分）

## 1. 背景与问题本质 (Problem)

### 1.1 当前现状
H5 端首页（`frontend/h5/src/pages/Home/index.tsx`，"推荐"tab）调用 `GET /api/v1/auctions` 拉取列表，是 **竞拍 (auction) 维度**：不传 `status`，拉全部状态的竞拍，由前端 `getStatusInfo` 按 `auction.status` + `end_time` 打标"即将开始 / 直播中 / 已结束"。

### 1.2 维度错配
业务设定（已由 schema 约束证实）：
- `live_streams.CreatorID` 唯一索引 → **一个商家一个直播间**。
- 一个直播间同一时间最多 **一个正在竞拍 + 一个即将开始**，但会累积 **大量已结束** 竞拍。

因此首页按竞拍维度铺卡，会出现：同一直播间的历史"已结束"竞拍一条条铺成独立卡片，越堆越多，把真正活跃的竞拍挤到列表下方。这与"一商家一直播间"的用户心智不一致。

### 1.3 "已结束怎么查"是伪问题
原始纠结点是"已结束竞拍如何分页查询"（限每间 X 条？查最近 Y 分钟？）。其根因是把已结束竞拍当成了 **列表中的一等卡片**。一旦把首页改为 **直播间维度**，并明确已结束竞拍的定位是 **氛围营造（留存信号）**，它就降级为附着在直播间卡片上的信息，复杂分页查询不再需要。

> 决策依据：首页"已结束"的角色 = 营造氛围 / 留存（用户已确认），不是历史成交归档列表。

## 2. 目标 (Goals)

1. 首页主列表从 **竞拍维度** 改为 **直播间维度**：一张卡片 = 一个直播间。
2. 每张直播间卡片聚合三类信息：正在竞拍（核心）、即将开始（钩子）、最近成交（氛围）。
3. 明确首页与现有"直播 feed"（`/live`，`LiveFeedPage`）的职责边界，消除二者定位重叠。

### 非目标 (Non-Goals)
- 不改造"直播 feed"（`LiveFeedPage`）的沉浸式滑动形态。
- 不做"历史成交归档/全量已结束竞拍列表"功能（如有需求，归到直播间详情页另立项）。
- 不改 H5 底部导航结构（仍为 首页 / 直播间 / 我的 三 tab）。

## 3. 整体方案 (Architecture)

### 3.1 职责分层（首页 vs 直播 feed）

| 维度 | 首页 `/`（货架） | 直播 feed `/live`（播放器） |
|------|------------------|----------------------------|
| 心智 | 逛、挑、决策"进哪个" | 看、沉浸、消费"就看这个" |
| 布局 | 直播间卡片 grid，滚动浏览多间 | 全屏单间，竖向滑动切间 |
| 数据范围 | 在播 + 即将开始 + 氛围成交 | 仅正在播（status=1 且有在拍竞拍） |
| 信息密度 | 高（摘要、多间并列对比） | 低（单间全屏，强临场感） |
| 核心动作 | 进入直播间 / 预约开拍 | 出价、互动、切下一间 |

二者构成 **「目录 → 内容流」的标准漏斗**，而非重复：
```
首页 /（货架概览）
  └─ 点卡片「进入直播间」
       └─ /live?id=xxx（沉浸 feed，定位该间，上下滑切其他在播间）
            └─ 出价 / 成交 / 互动
```

边界落地原则：首页与 feed 都会包含"在播"直播间，二者差异 **靠形态拉开（摘要卡 grid vs 全屏流），而非靠藏数据**。复用现有 `/live?id=` 进入逻辑，feed 几乎不动。

### 3.2 首页卡片信息分层（方案A）

每张直播间卡片承载三类数据：

| 信息 | 角色 | 数据来源 |
|------|------|----------|
| ① 正在竞拍 | 主视觉，可直接进入 | 现有 `current_auction`（`GetCurrentByLiveStreamIDs`） |
| ② 即将开始 | 预约钩子；无在播时升为主视觉 | **新增** `next_auction`（每间 `status=0` 取最近一条） |
| ③ 最近成交 | 氛围/留存信号，滚动展示 | **新增** `recent_deals`（每间取最近 2~3 条成交） |

卡片状态优先级：有 ① → "直播中"卡（主操作"进入直播间"）；无 ① 有 ② → "即将开始"卡（主操作"预约开拍提醒"）。

## 4. 后端改动 (Backend)

入口仍为 gateway `GET /api/v1/live-streams` → product-service `ListPublic`（`backend/product/handler/live_stream.go:353`）。

### 4.1 放开 status 硬过滤
当前 `ListPublic` 硬编码 `statusFilter = LiveStreamStatusLive(1)`，只返回直播中的间。改为同时纳入"有即将开始竞拍"的直播间，使首页能展示②类卡片。

> 待实现决策（实施计划阶段细化）：是否仍以 `status` 过滤为主，由"是否存在 current/next 竞拍"决定一个直播间是否出现在首页。原则：首页只展示"现在能进或即将能进"的活跃直播间，避免空壳直播间污染列表。

### 4.2 扩展回填字段
在现有 `current_auction_*` 回填基础上，新增对每个直播间的批量回填：
- `next_auction`：复用/新增 auction-service DAO，按 `live_stream_id IN ? AND status=0` 取每间 `start_time ASC` 最近一条（参照现有 `GetCurrentByLiveStreamIDs` 的批量取 Top1 写法，`backend/auction/dao/auction.go:454`）。
- `recent_deals`：按 `live_stream_id IN ? AND status=3`（已成交）取每间 `end_time DESC` 最近 N（默认 2~3）条，仅返回展示所需字段（商品名 + 成交价）。

> 复杂度控制：因首页只查少量活跃直播间（每商家一个，总量极小），上述回填是"先限定少量直播间、再每间取 Top N"，不存在全表扫描已结束竞拍的问题。

### 4.3 响应契约
`/live-streams` 单条 item 在现有字段基础上扩展（字段名最终以实施计划为准）：
```jsonc
{
  "id": 1,
  "name": "瑾瑜珠宝行",
  "status": 1,
  "host_name": "...", "host_avatar": "...", "viewer_count": 1284,
  // 已有
  "current_auction_id": 0, "current_product_id": 0, "current_price": "0",
  // 新增
  "next_auction": { "auction_id": 0, "product_name": "...", "start_time": "...", "start_price": "0" },
  "recent_deals": [ { "product_name": "...", "final_price": "0" } ]
}
```
金额字段沿用 `shopspring/decimal` 字符串序列化口径。跨服务仍走 RPC（productClient / auctionClient），不跨库 JOIN。

## 5. 前端改动 (Frontend)

### 5.1 首页 `Home/index.tsx`
- "推荐"tab 数据源从 `auctionApi.list`（`/auctions`）切换为 `liveStreamApi.list`（`/live-streams`），改为渲染直播间卡片。
- 新建直播间卡片组件，按 §3.2 三层信息渲染；状态优先级决定主视觉与主操作按钮。
- "最近成交"区做轻量滚动/横向展示，不抢主视觉。
- 进入逻辑复用现有 `/live?id=<liveStreamId>`；②类卡主操作复用现有"预约开拍提醒"链路（`handleSubscribeReminder`）。
- 分类 tab、收藏 tab 的既有行为保持（收藏 tab 已是直播间维度，不变）。

### 5.2 直播 feed `LiveFeedPage.tsx`
- 形态与逻辑 **不变**，继续 `GET /live-streams?status=1` + 前端 `hasCurrentAuction` 过滤。
- 仅需确认从首页 `/live?id=` 进入时能正确定位到指定直播间（现有能力，验证即可）。

### 5.3 UI / 主题
- 直播间卡片必须同时适配日 / 夜双主题，优先复用现有 CSS 变量，覆盖 `:root[data-theme='dark']`。
- 顶部"系统提示"等既有氛围元素不受影响（本次不涉及直播间内部）。

## 6. 测试策略 (Testing)
- 后端：为 `ListPublic` 扩展补单测，覆盖 next_auction / recent_deals 回填、空数据兜底、仅在播 / 仅即将开始 / 两者皆有的直播间分类。
- 前端：首页直播间卡片组件单测（三类信息渲染 + 状态优先级 + 双主题），更新 `Home.test.tsx` / 集成测试 `Home.integration.test.tsx`。
- E2E：Playwright 验证"首页看到直播间卡 → 点进入落到对应 feed 直播间"全链路。

## 7. 风险与权衡 (Risks)
- **首页与 feed 在播间重叠**：靠形态区分而非藏数据；需在视觉上明确"摘要卡 vs 全屏流"。
- **`auction.status` 推进依赖 scheduler，存在 stale**：前端保留 `end_time` 兜底显示逻辑。
- **回填增加跨服务调用**：批量接口控制 N 上限，活跃直播间总量小，开销可控。

## 8. 后续步骤 (Next Steps)
1. 用户确认本 Spec。
2. 调用 `writing-plans` 生成实施计划与任务拆分。
3. 按 Subagent-Driven 模式实施，遵循 TDD。

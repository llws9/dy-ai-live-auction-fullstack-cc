# H5 首页普通竞拍卡片「真实快照观看人数」设计方案

- 日期：2026-06-10
- 范围：H5 首页（`/`）普通竞拍卡片 + auction-service 列表编排 + product-service 内部直播间批量接口
- 方案定性：**真实快照**（随 `GET /api/v1/auctions` 一次性返回，非首页实时 WS 跳动）
- 状态：已定稿（决策已确认：后端 batch 接口不做 status 过滤，纯数据语义；过滤交前端）

---

## 1. 背景与目标

首页「全部」Tab 下的普通竞拍卡片（非「收藏」Tab 的直播间卡片）当前封面图区域只展示状态徽章（直播中/即将开始/已结束），**不展示观看人数**。直播间详情页（`LiveRoomSlide`）已有真实 Presence 观看人数，但首页列表没有透出该信息。

目标：在首页**进行中**的普通竞拍卡片封面图右下角，展示**真实直播间观看人数**（来源同直播间详情页的权威口径：Redis 实时计数优先、DB `viewer_count` 兜底）。

### 非目标
- 不做首页 WebSocket 实时跳动（成本不划算，快照足够）。
- 不改「收藏」Tab 直播间卡片（其已展示 `viewer_count`）。
- 不改直播间详情页内的 Presence 实时逻辑。

---

## 2. 设计决策（已与用户确认）

| 决策点 | 结论 | 理由 |
|---|---|---|
| **降级语义** | 直播间批量查询失败 / 拿不到 `viewer_count` 时，**降级显示，不让整页挂**（`/auctions` 仍返回 200，该字段填 0 或前端隐藏角标） | 观看人数是装饰性信息，不应让整页 5xx。**刻意偏离** `BuildAuctionListResponse` 现有「任意下游失败→整页 5xx」的 SSOT，仅对 viewer_count 这一路引入降级例外 |
| **显示规则** | **仅进行中（`auction.status`=1/2）的卡片**显示真实观看人数；待开始(0)、已结束(3/4) 不显示。判定主语是 **auction 状态**（前端 `statusInfo.live`），**与 `live_stream.status` 无关** | 待开始无人观看、已结束人数无意义；卡片是竞拍维度，故以 auction 状态判定 |
| **视觉落点** | 封面图**右下角半透明 pill**，形如 `◉ 128 观看`，复用 `.statusBadge` 风格（`rgba(0,0,0,0.62)` + `backdrop-filter: blur`） | 左上角已被状态徽章占用，右下角空闲且视觉平衡 |

---

## 3. 现状链路（已核对源码）

普通竞拍卡片数据流：

```
H5 Home  ──GET /api/v1/auctions──▶  gateway  ──forward──▶  auction-service
                                                              │
                                                BuildAuctionListResponse 编排：
                                                  Step2 lister 取 auction 分页
                                                  Step3 收集 product_id → product-service /internal/products/batch 取商品摘要
                                                  Step4 回填 → AuctionListItem{Auction, StartPrice, Product}
```

关键事实：
- `model.Auction` 已有 `LiveStreamID *int64`（[auction.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/model/auction.go) 行 24），但响应 `AuctionListItem`（[auction_list.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction_list.go#L47-L51)）**不含 viewer_count**。
- auction-service 已注入 `liveStreamClient client.LiveStreamClient`（[auction.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction.go#L40-L42)），用于竞拍归属校验，但 `BuildAuctionListResponse` 当前调用未传入它。
- product-service 内部接口 `POST /internal/live-streams/batch`（[internal.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/internal.go#L156-L217)）返回的 `liveStreamSummary` **不含 viewer_count**；且 `InternalHandler` 当前只持有 `liveStreamDAO`，未持有能算实时人数的 `liveStreamService`。
- 真实人数 SSOT：`LiveStreamService.ViewerCountForLiveStream`（[live_stream.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/service/live_stream.go#L146-L158)）= Redis 计数优先，DB `viewer_count` 兜底。
- 前端 `RawAuction` 有 `live_stream_id?` 但无 `viewer_count`；`HomeAuction` 无 `viewerCount`（[index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Home/index.tsx#L36-L71)）。

---

## 4. 改造方案（最省正确路径：后端契约打通）

### 4.1 product-service：内部批量接口回填 viewer_count

文件：[internal.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/internal.go)

1. `liveStreamSummary` 新增字段：
   ```go
   type liveStreamSummary struct {
       ID          int64  `json:"id"`
       Name        string `json:"name"`
       CoverImage  string `json:"cover_image"`
       Status      int    `json:"status"`
       CreatorID   int64  `json:"creator_id"`
       ViewerCount int64  `json:"viewer_count"` // 新增：Redis 优先、DB 兜底
   }
   ```
2. `InternalHandler` 注入 viewer 计数能力。为避免直接耦合大对象，**新增最小接口**：
   ```go
   // liveViewerCounter 抽象 LiveStreamService.ViewerCountForLiveStream，便于单测注入 fake。
   type liveViewerCounter interface {
       ViewerCountForLiveStream(ctx context.Context, ls *model.LiveStream) int64
   }
   ```
   `NewInternalHandler(productService, liveStreamDAO, liveViewerCounter)` 增加第三参；`main.go` 行 162 传入已构造的 `liveStreamService`（它已实现该方法）。
3. `BatchLiveStreams` 组装 summary 时回填：
   ```go
   ViewerCount: h.viewerCounter.ViewerCountForLiveStream(ctx, ls),
   ```
   - 注：仅在进行中口径有意义，但**后端不做 status 过滤**——是否展示交由前端按显示规则决定，后端保持「有就给」的纯数据语义，降低耦合。

> 说明：`GetOrCreateActiveLiveStream` 也返回 `liveStreamSummary`，该路径与本需求无关，`ViewerCount` 留零值即可，不必额外查。

### 4.2 auction-service：client 透传 viewer_count

文件：[live_stream_client.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/client/live_stream_client.go)

`LiveStreamSummary` 新增字段（JSON tag 与上游一致）：
```go
type LiveStreamSummary struct {
    ID          int64  `json:"id"`
    Name        string `json:"name"`
    CoverImage  string `json:"cover_image"`
    Status      int    `json:"status"`
    CreatorID   int64  `json:"creator_id"`
    ViewerCount int64  `json:"viewer_count"` // 新增
}
```
解码逻辑无需改动（结构体自动映射）。

### 4.3 auction-service：列表编排回填（含降级）

文件：[auction_list.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction_list.go) + 调用处 [auction.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction.go#L449)

1. `AuctionListItem` 新增字段：
   ```go
   type AuctionListItem struct {
       model.Auction
       StartPrice  *decimal.Decimal      `json:"start_price,omitempty"`
       Product     AuctionProductSummary `json:"product"`
       ViewerCount int64                 `json:"viewer_count"` // 新增：直播间观看人数快照，降级时为 0
   }
   ```
2. `BuildAuctionListResponse` 新增参数 `lsc client.LiveStreamClient`（紧跟现有 client 依赖），并在 Step3/Step4 之间插入 **Step3.5 批量取直播间观看人数**：
   ```go
   // Step 3.5: 收集本页 live_stream_id 批量取直播间摘要（仅用于 viewer_count）。
   // 降级语义：失败不阻断整页，viewer_count 缺省为 0（用户决策：装饰性信息不让整页挂）。
   viewerByStream := map[int64]int64{}
   if lsc != nil {
       streamIDs := collectLiveStreamIDs(auctions) // 去重、跳过 nil/<=0
       if len(streamIDs) > 0 {
           if streams, err := lsc.BatchGetLiveStreams(ctx, streamIDs); err != nil {
               log.Printf("auction list: batch live streams for viewer_count failed (degraded): %v", err)
           } else {
               for id, s := range streams {
                   viewerByStream[id] = s.ViewerCount
               }
           }
       }
   }
   ```
   Step4 回填时：
   ```go
   if a.LiveStreamID != nil {
       item.ViewerCount = viewerByStream[*a.LiveStreamID] // 命中即填，未命中为 0
   }
   ```
3. 调用处（[auction.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction.go#L449)）传入 `h.liveStreamClient`：
   ```go
   items, total, err := BuildAuctionListResponse(ctx, h.productClient, h.liveStreamClient, h.auctionService.ListAuctionsWithFilters, h.ruleFetcher, params)
   ```
   - `h.liveStreamClient` 为 nil 时（单测旧路径）整段跳过，行为向后兼容。

> **降级边界明确**：product 商品摘要（`BatchGetSummaries`）失败仍维持原 5xx 语义（商品是卡片主体，不可降级）；**仅 viewer_count 这一路失败降级**。两者职责清晰分离。

> **语义边界（实现时务必遵守）**：
> - **viewer_count 是直播间维度，不是竞拍维度**。同一 `live_stream_id` 下挂多个 auction 时，这些卡片会显示**相同**人数——这是预期行为，非 bug。
> - **无需分批**：首页 `page_size=20`，去重后 stream id ≤ 20，远低于 `internalLiveStreamBatchMaxIDs=200`，一次 batch 即可，不实现分批逻辑。
> - **降级日志降噪**：失败日志用 Warn 级即可，不要每条 item 打日志（每次列表请求最多 1 条），避免 product 长时间不可用时刷屏。

### 4.4 H5 前端：类型与渲染

文件：[index.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Home/index.tsx) + [Home.module.css](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Home/Home.module.css)

1. `RawAuction` 新增 `viewer_count?: number | string;`
2. `HomeAuction` 新增 `viewerCount: number;`
3. `normalizeAuction` 回填：`viewerCount: toNumber(auction.viewer_count)`
4. 渲染（普通卡片 `.imageWrapper` 内，状态徽章之后）：
   ```tsx
   {statusInfo.live && auction.viewerCount > 0 && (
     <div className={styles.viewerBadge}>
       <span className={styles.viewerDot} />
       {auction.viewerCount.toLocaleString()} 观看
     </div>
   )}
   ```
   - 显示条件双保险：`statusInfo.live`（进行中）**且** `viewerCount > 0`（降级或无人时不渲染，避免出现「0 观看」）。
5. 新增 CSS（参考 `.statusBadge`，定位改右下角）：
   ```css
   .viewerBadge {
     position: absolute;
     right: var(--spacing-2);
     bottom: var(--spacing-2);
     display: inline-flex;
     align-items: center;
     gap: 5px;
     padding: 3px var(--spacing-2);
     border-radius: var(--radius-full);
     background: rgba(0, 0, 0, 0.62);
     color: #fff;
     font-size: 10px;
     backdrop-filter: blur(12px);
   }
   .viewerDot {
     width: 6px; height: 6px; border-radius: 50%;
     background: #ff4d4f;
   }
   ```

---

## 5. 接口契约变更汇总

`POST /internal/live-streams/batch` 响应 `data.items[]` 新增字段：

```json
{ "id": 101, "name": "...", "cover_image": "...", "status": 1, "creator_id": 9, "viewer_count": 128 }
```

`GET /api/v1/auctions` 响应 `data.list[]` 新增字段：

```json
{ "id": 1, "live_stream_id": 101, "status": 1, "product": {...}, "start_price": "100", "viewer_count": 128 }
```

向后兼容：均为新增字段，老客户端忽略即可。

---

## 6. TDD 测试点

按「先写失败测试 → 最小实现 → 验证」推进。

### 6.1 product-service（[internal_test.go]）
- `BatchLiveStreams` 注入 `StaticLiveViewerCounter{101: 42}` + DB `viewer_count=19`，断言返回 `viewer_count=42`（Redis 优先）。
- Redis 计数为 0 时断言返回 DB 兜底值。
- 命中 + 缺失混合 id，断言缺失 id 跳过、命中 id 带正确 viewer_count。

### 6.2 auction-service（[auction_list_test.go]）
- 新增 `fakeLiveStreamClient` 替身。
- 正常：auction 带 `live_stream_id`，client 返回 `viewer_count`，断言 `AuctionListItem.ViewerCount` 正确回填。
- **降级**：`fakeLiveStreamClient.BatchGetLiveStreams` 返回 error，断言整体仍返回 200（无 error）、各 item `ViewerCount=0`、商品摘要正常。
- `liveStreamClient == nil`（旧路径）：不 panic，`ViewerCount=0`。
- 边界：`live_stream_id == nil` 的 auction `ViewerCount=0`，且不进入 streamIDs。

### 6.3 H5（[Home 渲染单测]）
- 进行中卡片 + `viewer_count>0`：渲染 `viewerBadge`，文案含「观看」。
- 进行中卡片 + `viewer_count=0`（降级）：不渲染 `viewerBadge`。
- 待开始 / 已结束卡片：即便带 viewer_count 也不渲染。

---

## 7. 工作量拆分

| 模块 | 内容 | 预估 |
|---|---|---|
| product-service | `liveStreamSummary` 加字段 + handler 注入 counter + main 接线 | 1–1.5h |
| auction-service | client 加字段 + 编排回填 + 降级 | 1.5–2h |
| H5 前端 | 类型 + normalize + 渲染 + CSS | 1–1.5h |
| 单测 | 三层测试点 | 2–3h |
| 本地联调 + 部署验证 | deploy-dev 验证真实人数透出 | 1h |
| **合计** | | **约 0.5–1 天** |

---

## 8. 风险与回滚

- **风险**：批量直播间查询增加一次跨服务 HTTP 调用，列表 RT 略增。已通过「降级不阻断 + 仅本页 id 批量」控制；超时沿用 client 默认 2s。
- **回滚**：前端隐藏 `viewerBadge` 即可视觉回滚；后端新增字段无破坏性，可保留。

---

## 9. 已排除方案

| 方案 | 排除原因 |
|---|---|
| 前端额外请求 `/live-streams` 再 merge | 只覆盖直播中、与 auction 分页口径不一致，易错配 |
| 每卡片请求 `/live-streams/:id` | N+1，首屏性能差 |
| 首页接 WebSocket 实时跳动 | 复杂度高，快照已满足需求，不划算 |

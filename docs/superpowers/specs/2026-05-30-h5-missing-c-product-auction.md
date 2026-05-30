# H5 缺失接口闭环 · 子 spec C：商品/拍卖域接口补齐

**日期**：2026-05-30

**关联总览 spec**：[2026-05-30-h5-missing-interfaces-closure.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/superpowers/specs/2026-05-30-h5-missing-interfaces-closure.md)

**适用仓库**：`/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc`

**定位**：本文件是子 spec（领域分册），仅描述契约、决策与边界，不写实施代码。

---

## 1. 范围

本期闭环商品/拍卖域以下 4 项能力：

| 编号 | 能力 | 优先级 |
|---|---|---|
| F-C1 | `GET /api/v1/auctions` 增加 `category_id` 过滤；响应 items 内嵌 `product` 摘要 | P1 |
| F-C2 | `GET /api/v1/auctions/:id/result` 字段扩展（`won_bid`、`product` 摘要） | P1 |
| F-C3 | `GET /api/v1/orders/history` JWT 化，移除 query `user_id`；响应字段扩展 | P0 |
| F-C4 | 商品规则 path id 语义修正：明确为 `product_id`，移除 `auction_id` 兼容映射 | P2 |

### 1.1 不在本期范围

- 分页协议大改（仍保持 `{items,total,page,page_size}`）。
- admin 后台 orders 接口（`/orders` list）改造，确认无相互影响后保持原状。
- 规则模型重构（仅修语义和命名，不动表结构）。
- result 接口的鉴权收敛（保留公开访问，安全性下期再做）。

---

## 2. 当前仓库事实

- Auctions list handler：[auction.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction.go)（约 L201-L281），目前支持 `status, live_stream_id, live_stream_name, search, page, page_size`，**未支持 `category_id`**。
- Auction model：[auction.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/model/auction.go) 字段含 `product_id, live_stream_id, current_price, winner_id, start_time, end_time`，**无 category_id**。
- Product model：[product.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/model/product.go) 含 `category_id, images`，是分类的归属点。
- Result handler：[auction.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction.go)（约 L122-L155）`GetResult` 当前仅返回 `auction_id/product_id/status/final_price/winner_id/started_at/ended_at/delay_used`。
- Order history：[order.go#L222](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/order.go) 当前 `GetUserHistory` 直接读 `query("user_id")`，**存在水平越权风险**。
- Gateway 路由：[router.go#L100](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go) `v1.GET("/orders/history", productProxy.Forward)` 配置在 v1 顶层而非 authGroup，**JWT 中间件未生效**。
- Gateway JWT 中间件：[jwt.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/middleware/jwt.go)，仅在挂到 authGroup 时注入 user_id 到上下文/header。
- 商品规则 handler：[rule.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/rule.go) 注释提示存在临时 `product_id`/`auction_id` 互用。
- 前端 service：[api.ts](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts) `auctionApi.list / auctionApi.getResult / orderApi.history`。

---

## 3. 共享约定

- Response 包络：`{"code":200, "message":"success", "data": {...}}`；列表统一 `data: {items, total, page, page_size}`，**禁止 `list` / `data.list`**。
- 错误：`{"code": <非 200>, "message": "..."}`，HTTP 状态码与业务 code 保持一致。
- 鉴权：受保护接口经 Gateway authGroup 校验 JWT，user_id 由 Gateway 注入下游 `X-User-ID` header，下游禁止信任 query/body 内的 user_id。
- 内部接口：路径前缀 `/internal/`，**仅服务间 RPC，绝不在 Gateway 注册**。
- 向前兼容：响应只做字段新增，不改动已有字段名/类型；所有新增字段对老客户端必须可忽略。

---

## 4. 接口契约

### 4.1 F-C1 · `GET /api/v1/auctions`（扩展）

**Method/Path**：`GET /api/v1/auctions`

**鉴权**：公开（与现状一致）。

**Request Query**：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| status | string | 否 | 现有，保持不变 |
| live_stream_id | int64 | 否 | 现有 |
| live_stream_name | string | 否 | 现有 |
| search | string | 否 | 现有 |
| page | int | 否 | 现有，默认 1 |
| page_size | int | 否 | 现有，默认 20 |
| **category_id** | int64 | 否 | **新增**。商品分类 id；未传则不按分类过滤 |

**Response 200**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 1001,
        "product_id": 5001,
        "live_stream_id": 200,
        "status": "ongoing",
        "current_price": 8800,
        "start_time": "2026-05-30T12:00:00Z",
        "end_time": "2026-05-30T12:30:00Z",
        "product": {
          "id": 5001,
          "name": "和田玉手镯",
          "image": "https://cdn.example.com/p/5001/0.jpg",
          "category_id": 12
        }
      }
    ],
    "total": 37,
    "page": 1,
    "page_size": 20
  }
}
```

**字段说明**：

- `product`：摘要对象；当 product-service 不可用或对应 product 已删除时为 `null`。
- `product.image`：取 `images[0]`，若 images 为空数组则为空字符串。

**错误码**：

- `400`：`category_id` 非合法整数。
- `502`：上游 product-service 不可达且降级策略选择强失败时（默认走降级，见 §9.2）。

**向前兼容**：
- 老客户端忽略 `product` 字段无感知。
- 不传 `category_id` 时行为与现状一致。

---

### 4.2 F-C2 · `GET /api/v1/auctions/:id/result`（扩展）

**Method/Path**：`GET /api/v1/auctions/:id/result`

**鉴权**：公开（与现状一致；本期不收敛）。

**Response 200**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "auction_id": 1001,
    "product_id": 5001,
    "status": "ended",
    "final_price": 9200,
    "winner_id": 88,
    "started_at": "2026-05-30T12:00:00Z",
    "ended_at": "2026-05-30T12:30:00Z",
    "delay_used": 30,
    "won_bid": 9200,
    "product": {
      "id": 5001,
      "name": "和田玉手镯",
      "images": [
        "https://cdn.example.com/p/5001/0.jpg",
        "https://cdn.example.com/p/5001/1.jpg"
      ]
    }
  }
}
```

**新增字段**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `won_bid` | number | 中标价。本期取值等同 `final_price`，**作为前端契约友好别名**保留；未来若引入 `winner_bid_amount/won_bid_at` 分离语义，再单独扩展，不破坏 `won_bid` |
| `product` | object\|null | 商品摘要：`id, name, images[]`（注意：result 接口给完整 images 数组，与 list 的 `image` 单图不同） |

**错误码**：

- `404`：拍卖不存在或尚未结束（保持现状）。

**向前兼容**：
- 已有字段全部保留、含义不变。
- `won_bid` 与 `final_price` 必须始终相等，前端仅取 `won_bid`，不会出现两值不一致。

---

### 4.3 F-C3 · `GET /api/v1/orders/history`（破坏性变更）

**Method/Path**：`GET /api/v1/orders/history`

**鉴权**：**必须**经 Gateway authGroup（JWT），user_id 从 `X-User-ID` 透传 header 读取。

**Request Query**：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| ~~user_id~~ | ~~int64~~ | ~~是~~ | **移除**。读取该参数会被忽略并按 JWT user_id 处理 |
| page | int | 否 | 默认 1 |
| page_size | int | 否 | 默认 20 |
| status | string | 否 | 现有过滤项保持 |

**Response 200**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 9001,
        "auction_id": 1001,
        "product_id": 5001,
        "user_id": 88,
        "status": 1,
        "final_price": 9200,
        "created_at": "2026-05-30T12:30:00Z",
        "ended_at": "2026-05-30T12:30:00Z",
        "my_highest_bid": 9200,
        "product": {
          "id": 5001,
          "name": "和田玉手镯",
          "image": "https://cdn.example.com/p/5001/0.jpg"
        }
      }
    ],
    "total": 12,
    "page": 1,
    "page_size": 20
  }
}
```

**新增字段**：

| 字段 | 类型 | 含义 |
|---|---|---|
| `product` | object\|null | 商品摘要：`id, name, image`（取 images[0]） |
| `my_highest_bid` | number\|null | 当前用户在该拍卖里的最高出价；无出价（如直接得标但未参与）为 null |
| `ended_at` | string | 拍卖结束时间，ISO8601 |

**错误码**：

- `401`：未登录或 JWT 无效（由 Gateway authGroup 返回）。
- `200` + `data.items=[]`：用户无历史订单。

**破坏性影响评估**：

- 前端 [api.ts orderApi.history](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts) 当前已不传 `user_id`，可直接切换。
- admin 后台使用 `/orders` list 接口，**不**走 `/orders/history`，无影响。
- 路由调整为 authGroup 后，未登录调用从静默返回他人数据变为 `401`，是预期效果。

---

### 4.4 F-C4 · 商品规则接口语义修正

**变更性质**：契约语义修正，**接口路径与字段不变**。

**当前问题**：[rule.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/handler/rule.go) 中 path id 既被解释为 `product_id` 也被兼容为 `auction_id`，导致：
- 调用方语义混乱。
- 不同来源调用者写入到不同维度的规则数据。
- 后续基于 product 的规则查询易出现脏数据漏读。

**本期决策**：

1. 明确：**规则归属于 product**，rules handler 接收的 path id **就是 `product_id`**。
2. service 层移除 `auction_id` → `product_id` 的兼容映射逻辑。
3. 路径参数命名规范化为 `:product_id`（若现有为 `:id`，注释/文档明确其语义为 product_id；不强制改路径以减少变更面）。
4. 调用方（包含 auction-service、admin、前端）必须传 `product_id`，禁止传 `auction_id`。

**数据校验要求（实施时执行）**：

- 上线前对现有规则表做一次校验脚本：抽样比对规则 owner_id 是否能在 `products` 表中命中。
- 若发现以 `auction_id` 写入的脏数据，需先做一次反向映射修复（`auction.product_id`），并回写规则表，再上线契约修正。
- 校验/修复脚本不在本 spec 范围，由实施 plan 输出。

---

## 5. 跨服务调用方案

### 5.1 product-service 新增内部接口

#### 5.1.1 `POST /internal/products/batch`

**用途**：auction-service 在 list/result handler 中按 product_id 列表批量取摘要。

**Method/Path**：`POST /internal/products/batch`

**鉴权**：内部调用，不经 Gateway；可通过内部网络白名单或服务间 token 控制。

**Request Body**：

```json
{
  "ids": [5001, 5002, 5003],
  "fields": ["id", "name", "images", "category_id"]
}
```

- `ids`：单次最多 200 个；超过返回 400。
- `fields`：字段白名单；未传时返回默认摘要 `id, name, images, category_id`。

**Response 200**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [
      {
        "id": 5001,
        "name": "和田玉手镯",
        "images": ["https://cdn.example.com/p/5001/0.jpg"],
        "category_id": 12
      }
    ]
  }
}
```

- 返回顺序不保证；调用方按 id 自行 map。
- 已删除/不存在的 id **不出现在 items 中**，调用方据此判定 null。

**错误码**：

- `400`：ids 为空、超长或非整数。

#### 5.1.2 `GET /internal/products`（扩展或新增）

**用途**：auction-service 实现 `category_id` 过滤的第一步——按分类拉 product_id 列表。

**Method/Path**：`GET /internal/products`

**Request Query**：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| category_id | int64 | 是（本场景） | 商品分类 id |
| fields | string | 否 | 逗号分隔字段白名单，例如 `id` |
| page | int | 否 | 默认 1 |
| page_size | int | 否 | 默认 500（内部场景允许较大）；上限 1000 |

**Response 200**：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [{"id": 5001}, {"id": 5002}],
    "total": 137,
    "page": 1,
    "page_size": 500
  }
}
```

**说明**：

- 当分类下商品数 > 1000 时（极端场景），auction-service 应改为：让 product-service 直接按 `category_id` 反查关联 auction（或后续单独 spec 处理跨服务连接）。本期假设单分类商品数在 1000 以内。

### 5.2 auction-service 调用编排

**list（F-C1）**：

1. 若 `category_id` 存在：调 `GET /internal/products?category_id=...&fields=id` 取 product_id 列表。
2. 用 `WHERE product_id IN (...)` 在 auction 表查询，再叠加其他过滤条件。
3. 收集结果集 product_id 去重，调 `POST /internal/products/batch` 取摘要。
4. 摘要按 id map 回填到每个 auction item 的 `product` 字段。

**result（F-C2）**：

1. 取到 auction 结果后，用单元素 `ids=[product_id]` 调 batch 接口。
2. 摘要回填到 `data.product`。
3. 仅 result 场景返回完整 `images[]`，list 场景仅返回 `image` 单图。

### 5.3 关键约束

- 内部接口路径前缀 `/internal/`，**Gateway 禁止注册任何 `/internal/*` 路由**。
- 服务间调用使用内部 service discovery 域名，不走公网。
- 失败降级见 §9.2。

---

## 6. Gateway 路由调整

### 6.1 `/orders/history` 移到 authGroup

当前：

```
v1.GET("/orders/history", productProxy.Forward)   // ❌ 在 v1 顶层
```

调整为：

```
authGroup.GET("/orders/history", productProxy.Forward)  // ✅
```

未登录访问从“透明转发”变为 `401`。

### 6.2 JWT user_id 透传机制

- Gateway authGroup 中 [jwt.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/middleware/jwt.go) 解析 JWT 后，必须在 proxy 转发前注入：
  - Header：`X-User-ID: <user_id>`
- 下游服务（product-service / auction-service）**仅信任 `X-User-ID` header**，禁止使用 query/body 中的 user_id。
- 下游若直接收到外部请求（理论上应不可能，因网关是唯一入口）需要拒绝。

### 6.3 内部接口隔离

- Gateway 配置必须 explicit 列出对外暴露的 path；`/internal/*` 在 router 层无对应 handler，404 直接返回。
- 部署上 product-service 暴露 `/api/v1/*` 与 `/internal/*` 两组路由；`/internal/*` 只在内部 VPC 可达。

---

## 7. 数据模型变更

**无新增表，无新增字段**。

- F-C1：`auctions` 表无 `category_id`，category 过滤通过跨服务查询解决。
- F-C2/C3：仅响应字段扩展，不动表。
- F-C4：仅修语义，不动表结构；如有脏数据，由数据校验脚本一次性修复。

---

## 8. 前端集成点

### 8.1 [api.ts](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts)

| 接口 | 改动 |
|---|---|
| `auctionApi.list` | 入参增加可选 `category_id?: number`；透传到 query |
| `auctionApi.getResult` | 无入参变化；adapter 处理新 `won_bid`、`product` 字段 |
| `orderApi.history` | 移除 `user_id` 入参；adapter 处理新 `product`、`my_highest_bid`、`ended_at` 字段 |

### 8.2 Adapter 兼容

- 后端返回 `product=null` 时，UI 占位图 + “商品信息加载失败”文案，不阻塞列表渲染。
- 老 mock 数据无 `won_bid` 字段时回退到 `final_price`，保证开发态可用。
- 列表的 `product.image` 与 result 的 `product.images[]` 在 adapter 中归一化为同一前端模型字段（如 `coverImage` + `gallery[]`）。

### 8.3 Home 分类 Tab 联动

- 分类 Tab 切换时调 `auctionApi.list({ category_id, ... })`。
- “全部”Tab 不传 `category_id`。

---

## 9. 风险与边界

### 9.1 跨服务批量调用的 N+1 与缓存

- list 场景已用 batch 接口避免 N+1。
- product 摘要在 product-service 加

### 9.2 product-service 故障降级（已决策 · 2026-05-30）

两个接口分别采用不同降级策略，原因：失败语义、阻塞代价不同。

| 接口 | 失败时行为 | 决策依据 |
|---|---|---|
| F-C1 `/auctions` (list) | **整个接口 5xx**，无静默降级 | 列表是探索/筛选场景，product 摘要承载首图与名称，半数缺失会让用户体验严重劣化；分页查询整体失败更符合直觉，便于客户端做 skeleton/重试。 |
| F-C2 `/auctions/:id/result` (result) | **`product=null` 软降级**，核心字段照常 200 返回 | result 是用户查看中标结果的核心页；`winner_id`、`final_price`、`won_bid` 不依赖 product-service。即使 product-service 抖动，让用户看到中标价值远高于 5xx 阻塞。 |

**实现位置**：
- list：[handler/auction_list.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction_list.go) `BuildAuctionListResponse` — 任一 client 调用 err 直接 bubble，handler 转 500。
- result：[handler/auction_result.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/auction_result.go) `BuildAuctionResultResponse` — `pc.BatchGetSummaries` 错误时吞掉错误，返回 `product=nil`。

**前端契约**：
- list：客户端可信 `items[].product` 必非空。
- result：客户端必须做 `product == null` 兜底（不展示画廊或显示占位文案）。

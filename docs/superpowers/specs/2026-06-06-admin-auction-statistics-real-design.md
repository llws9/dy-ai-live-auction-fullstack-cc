# 管理端真实竞拍统计设计方案

## 1. 背景

管理端 `数据统计 -> 竞拍统计` 页面当前视觉和交互已经存在，但后端接入并不真实。前端 `frontend/admin/src/pages-new/Stats.tsx` 期望 `statisticsApi.getAuctionStats()` 返回数组，并从每一项读取：

| 字段 | 用途 |
|---|---|
| `date` | 图表 X 轴日期 |
| `auction_count` | 每日竞拍场次 |
| `success_rate` | 每日竞拍成功率 |
| `bid_count` | 每日出价次数，用于计算平均出价次数 |

现有后端 `product-service` 的 `GET /statistics/auctions` 返回单个对象 `AuctionStatistics`，并且基于 `orders` 表做代理统计，`avg_bid_count` 仍是固定值 `3.5`。这与页面期望不匹配，也不符合真实竞拍统计的领域边界。

真实竞拍数据属于 `auction-service`：

- `auctions` 表拥有竞拍场次、状态、创建者、开始时间、成交用户。
- `bids` 表拥有真实出价记录。

因此本设计选择将 `GET /api/v1/statistics/auctions` 的后端实现迁移到 `auction-service`，由 Gateway 统一暴露给前端。

## 2. 目标

- 让管理端竞拍统计页面展示真实后端数据，不再依赖静态兜底数据。
- 保持前端调用入口不变：`GET /api/v1/statistics/auctions`。
- 后端按角色区分数据范围：平台管理员看全平台，商家只看自己创建的竞拍。
- 统计口径与字段结构直接匹配前端 `AuctionStatistics[]` 类型。
- 遵守服务边界：前端只走 Gateway `/api/v1`；跨服务不直接查库；竞拍领域统计由 `auction-service` 拥有。

## 3. 非目标

- 不同时实现收入统计、用户统计或 Dashboard 总览统计。
- 不引入离线数仓、定时汇总表或缓存层。
- 不改造整个统计页面的视觉布局。
- 不新增多维筛选 UI，例如类目、直播间、商家下拉筛选。
- 不用 Prometheus 指标替代业务库聚合；本页面需要业务口径数据，不是运行时监控口径。

## 4. 方案选择

### 4.1 选定方案：Gateway 路由到 auction-service 真实聚合

保留外部接口：

```http
GET /api/v1/statistics/auctions?start_date=2026-06-01&end_date=2026-06-07&group_by=day
Authorization: Bearer <admin-or-merchant-jwt>
```

Gateway 验证 JWT 与角色后，将请求转发到 `auction-service`。`auction-service` 读取 `X-User-ID` 与 `X-User-Role` 进行二次兜底：

- `admin`：统计全平台竞拍。
- `merchant`：统计 `auctions.creator_id = X-User-ID` 的竞拍。
- 其他角色：返回 `403`。

### 4.2 放弃方案：继续由 product-service 返回数组

这个方案实现快，但只能从 `orders` 推导成交结果，无法准确统计 `bid_count`。页面标题是“竞拍热度分析”，如果出价次数不真实，指标会误导使用者。

### 4.3 放弃方案：换成成交统计维度

成交统计可直接基于 product-service 的订单数据实现，但它会改变用户原始目标。当前真实竞拍数据已经具备，切换维度不是最短路径。

## 5. 接口契约

### 5.1 请求

```http
GET /api/v1/statistics/auctions
```

Query 参数：

| 参数 | 类型 | 必填 | 默认值 | 规则 |
|---|---|---:|---|---|
| `start_date` | string | 否 | 今天往前 6 天 | 格式 `YYYY-MM-DD` |
| `end_date` | string | 否 | 今天 | 格式 `YYYY-MM-DD`，包含当天 |
| `group_by` | string | 否 | `day` | 第一版仅支持 `day` |

约束：

- `start_date > end_date` 返回 `400`。
- 日期跨度超过 90 天返回 `400`，避免管理端误触发大范围聚合。
- `group_by` 不是空值或 `day` 返回 `400`。

### 5.2 响应

成功响应直接返回数组，保持现有前端 `get<AuctionStatistics[]>` 能消费：

```json
[
  {
    "date": "2026-06-01",
    "auction_count": 12,
    "bid_count": 146,
    "avg_price": 1288.5,
    "success_rate": 83.3
  }
]
```

字段语义：

| 字段 | 类型 | 口径 |
|---|---|---|
| `date` | string | `auctions.start_time` 所在日期，格式 `YYYY-MM-DD` |
| `auction_count` | number | 当天开始的竞拍场次数 |
| `bid_count` | number | 这些竞拍关联的 `bids` 总数 |
| `avg_price` | number | 成功竞拍的平均成交价；无成功竞拍时为 `0` |
| `success_rate` | number | `成功竞拍数 / auction_count * 100`，保留一位小数 |

成功竞拍定义：

```text
auctions.status = 3 AND auctions.winner_id IS NOT NULL
```

状态值沿用 `auction-service/model.AuctionStatusEnded = 3`。

### 5.3 错误响应

错误响应沿用现有服务的简单 JSON 结构：

```json
{
  "code": 400,
  "message": "start_date must be before or equal to end_date"
}
```

| 场景 | HTTP 状态 |
|---|---:|
| 未认证 | 401 |
| 非管理员/商家角色 | 403 |
| 参数非法 | 400 |
| 数据库查询失败 | 500 |

## 6. 数据流

```text
Admin Stats.tsx
  -> statisticsApi.getAuctionStats()
  -> Gateway /api/v1/statistics/auctions
  -> JWTAuth + RequireMerchantOrAdmin
  -> auction-service /api/v1/statistics/auctions
  -> StatisticsHandler reads X-User-ID/X-User-Role
  -> StatisticsService validates date range and scope
  -> StatisticsDAO aggregates auctions + bids
  -> zero-fill missing dates
  -> returns AuctionDailyStat[]
```

Gateway 仍是唯一前端入口。`auction-service` 不信任“只有 Gateway 会调用”的假设，仍必须校验内部 Token 和角色头，保持 fail-closed。

## 7. 后端设计

### 7.1 Gateway

修改 `backend/gateway/router/router.go`：

- 当前 `/statistics/auctions` 转发到 `adminProductProxy`。
- 改为转发到 `adminAuctionProxy`。
- 仍使用 `middleware.RequireMerchantOrAdmin()`。

`/statistics/overview`、`/statistics/revenue`、`/statistics/users` 暂不变，仍归 product-service。

### 7.2 auction-service

新增文件：

- `backend/auction/handler/statistics.go`
- `backend/auction/service/statistics.go`
- `backend/auction/dao/statistics.go`

修改文件：

- `backend/auction/main.go`：初始化 `StatisticsDAO/Service/Handler` 并注册路由。

核心接口：

```go
type AuctionDailyStat struct {
	Date         string  `json:"date"`
	AuctionCount int64  `json:"auction_count"`
	BidCount     int64  `json:"bid_count"`
	AvgPrice     float64 `json:"avg_price"`
	SuccessRate  float64 `json:"success_rate"`
}
```

服务方法：

```go
func (s *StatisticsService) GetAuctionDailyStats(
	ctx context.Context,
	startDate time.Time,
	endDate time.Time,
	creatorID *int64,
) ([]AuctionDailyStat, error)
```

DAO 聚合以 `auctions.start_time` 分组，并左连接 `bids`：

```sql
SELECT
  DATE(a.start_time) AS date,
  COUNT(DISTINCT a.id) AS auction_count,
  COUNT(b.id) AS bid_count,
  COALESCE(AVG(CASE WHEN a.status = 3 AND a.winner_id IS NOT NULL THEN a.current_price END), 0) AS avg_price,
  SUM(CASE WHEN a.status = 3 AND a.winner_id IS NOT NULL THEN 1 ELSE 0 END) AS success_count
FROM auctions a
LEFT JOIN bids b ON b.auction_id = a.id
WHERE a.start_time >= ? AND a.start_time < ?
GROUP BY DATE(a.start_time)
ORDER BY DATE(a.start_time)
```

当 `creatorID != nil` 时追加：

```sql
AND a.creator_id = ?
```

实现需要避免 `COUNT(a.id)` 被 `LEFT JOIN bids` 放大，必须使用 `COUNT(DISTINCT a.id)`。

### 7.3 product-service

product-service 暂不删除旧 `GetAuctionStatistics` 实现，避免直接服务访问或文档引用在本轮产生额外风险。Gateway 改路由后，前端不会再命中旧实现。

后续如要清理旧接口，需要单独做兼容评估。

## 8. 前端设计

修改 `frontend/admin/src/pages-new/Stats.tsx`：

- 竞拍统计请求加默认参数：最近 7 天、`group_by=day`。
- 删除竞拍统计失败后的静态 mock 兜底。
- 失败时显示空图表和错误提示，避免把假数据误认为真实数据。
- 修正平均成功率计算，避免 reduce 中每步除以数组长度导致结果偏低。
- 当返回空数组时，指标卡展示 `0`，图表展示 7 天零值。

修改 `frontend/admin/src/shared/api/types.ts`：

- 保持 `AuctionStatistics` 字段不变。
- 不新增与当前页面无关的字段。

## 9. 测试策略

### 9.1 auction-service 单元测试

覆盖：

- admin 不带 `creatorID` 时聚合全量竞拍。
- merchant 带 `creatorID` 时只聚合自己创建的竞拍。
- `LEFT JOIN bids` 不放大 `auction_count`。
- 缺失日期补零。
- 成功率按 `status=3 AND winner_id IS NOT NULL` 计算。
- 非法日期范围返回参数错误。

### 9.2 Gateway 路由测试

覆盖：

- merchant/admin 访问 `/api/v1/statistics/auctions` 被转发到 auction-service。
- user 访问返回 `403`。
- 转发时保留查询参数。
- 转发时带上 `X-Internal-Token`、`X-User-ID`、`X-User-Role`。

### 9.3 前端测试

覆盖：

- API 返回真实数组时渲染竞拍指标和图表。
- API 返回空数组时渲染 0 值，不抛异常。
- API 失败时不使用静态 mock 数据。

## 10. 风险与决策

| 风险 | 决策 |
|---|---|
| product-service 旧接口仍存在 | 本轮只改 Gateway 路由，降低破坏面；后续清理另起任务 |
| `success_rate` 口径歧义 | 第一版定义为“已结束且有 winner_id”，不是订单支付成功率 |
| `avg_price` 使用 decimal 金额 | 聚合只做读模型输出，数据库 decimal 扫描后转 float64 给图表；业务写入金额仍保持 `shopspring/decimal` |
| 日期口径使用 `created_at` 还是 `start_time` | 使用 `start_time`，因为页面表达的是每日竞拍场次，不是每日创建记录 |
| 管理端误查大区间 | 限制最大 90 天 |

## 11. 验收标准

- 管理端 `/stats/auction` 不再展示静态兜底数据。
- `GET /api/v1/statistics/auctions` 返回 `AuctionStatistics[]` 数组。
- 管理员看到全平台竞拍统计，商家只看到自己创建的竞拍统计。
- `bid_count` 来自真实 `bids` 表。
- 单测覆盖 Gateway 路由、auction-service 聚合、前端统计渲染。
- `go test` 与前端测试通过。

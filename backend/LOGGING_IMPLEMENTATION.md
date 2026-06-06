# 后端日志与监控系统实现文档

## 概述

已为直播竞拍系统的后端服务完善了日志系统和监控指标系统，支持全链路日志追踪、业务指标统计和前端埋点功能。

## 实现的文件清单

### 1. Auction Service

#### 1.1 日志中间件
**文件**: `backend/auction/middleware/logger.go`
- 创建了统一的请求日志中间件
- 记录请求ID、用户信息、操作类型、响应时间等
- JSON格式输出，包含时间戳和服务名称
- 支持链路追踪（通过X-Request-ID）

#### 1.2 竞拍服务日志增强
**文件**: `backend/auction/service/auction.go`
增强的操作日志：
- **创建竞拍**: 记录商品ID、竞拍ID、开始/结束时间、响应时间
- **取消竞拍**: 记录竞拍ID、当前状态、取消原因
- **结束竞拍**: 记录竞拍ID、最终价格、中标者ID、响应时间

#### 1.3 出价服务日志增强
**文件**: `backend/auction/service/bid.go`
增强的操作日志：
- **出价操作**:
  - 记录用户ID、竞拍ID、出价金额
  - 记录出价前验证结果（用户存在性、竞拍状态、出价金额）
  - 记录分布式锁获取情况
  - 记录排名信息
  - 记录完整响应时间

### 2. Product Service

#### 2.1 日志中间件
**文件**: `backend/product/middleware/logger.go`
- 与auction服务相同的日志中间件
- 支持统一的日志格式和字段

#### 2.2 商品服务日志增强
**文件**: `backend/product/service/product.go`
增强的操作日志：
- **创建商品**: 记录商品ID、名称、状态、响应时间
- **更新商品**: 记录更新的字段、商品ID、响应时间
- **删除商品**: 记录商品ID、删除结果
- **创建竞拍规则**: 记录规则ID、商品ID、竞拍参数

#### 2.3 订单服务日志增强
**文件**: `backend/product/service/order.go`
增强的操作日志：
- **订单支付**:
  - 记录订单ID、竞拍ID、商品ID、中标者ID
  - 记录最终价格、响应时间
  - 记录状态变更
- **订单发货**:
  - 记录订单ID、竞拍ID、商品ID、中标者ID
  - 记录响应时间
- **订单完成**:
  - 记录订单ID、竞拍ID、商品ID、中标者ID
  - 记录最终价格、响应时间

### 3. Gateway Service

#### 3.1 日志中间件增强
**文件**: `backend/gateway/middleware/logger.go`
- 增强了原有的日志中间件
- 添加了请求ID生成和传递
- 添加了用户信息提取
- 支持操作类型判断
- JSON格式输出

### 4. 公共日志工具包

#### 4.1 日志工具类
**文件**: `backend/auction/pkg/logger/logger.go` 和 `backend/product/pkg/logger/logger.go`

功能特性：
- 统一的日志记录接口
- 操作类型定义（create, update, delete, query, login, bid, pay, ship, complete）
- 对象类型定义（product, auction, bid, order, user, auth）
- 支持上下文传递（request_id）
- 敏感信息自动脱敏（password, token, secret, api_key）
- 多种日志记录方法：
  - `LogOperation`: 简化版操作日志
  - `LogOperationWithData`: 带数据的操作日志
  - `LogUserOperation`: 用户操作日志
  - `LogHTTPRequest`: HTTP请求日志

## 日志格式规范

### JSON日志字段

```json
{
  "timestamp": "2026-05-23T02:50:00Z",
  "service_name": "auction-service",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "operation_type": "create",
  "object_type": "auction",
  "object_id": "123",
  "user_id": "456",
  "user_name": "张三",
  "success": true,
  "duration": "150ms",
  "error_msg": "",
  "request_data": {
    "product_id": 789,
    "start_time": "2026-05-23T10:00:00Z",
    "end_time": "2026-05-23T12:00:00Z"
  },
  "response_data": {...},
  "client_ip": "192.168.1.100",
  "extra": {}
}
```

## 记录的关键操作

### 1. 用户相关
- 用户注册
- 用户登录
- 用户信息更新

### 2. 商品相关
- 商品创建
- 商品更新
- 商品删除
- 商品发布

### 3. 竞拍相关
- 竞拍创建
- 竞拍开始
- 竞拍取消
- 竞拍结束

### 4. 出价相关
- 出价操作
- 出价验证
- 出价排名更新

### 5. 订单相关
- 订单创建
- 订单支付
- 订单发货
- 订单完成

## 日志特性

### 1. 链路追踪
- 每个请求生成唯一的Request ID
- Request ID通过HTTP Header传递：`X-Request-ID`
- 支持跨服务追踪

### 2. 敏感信息脱敏
自动脱敏以下字段：
- password
- token
- secret
- api_key

### 3. 性能监控
- 记录每个操作的响应时间（duration_ms）
- 支持性能分析和优化

### 4. 错误记录
- 记录详细的错误信息
- 区分成功和失败状态
- 便于问题排查和监控

### 5. 用户行为追踪
- 记录操作用户ID和用户名
- 通过HTTP Header传递：`X-User-ID`, `X-User-Name`

## 使用示例

### 中间件使用

```go
// 在main.go中注册中间件
h.Use(middleware.RequestLogger(middleware.LoggerConfig{
    ServiceName: "auction-service",
}))
```

### 服务层使用

```go
// 创建日志记录器
logger := logger.NewLogger("auction-service")

// 记录操作日志
logger.LogOperationWithData(ctx, logger.OperationCreate, logger.ObjectAuction,
    fmt.Sprintf("%d", auction.ID), true, nil,
    map[string]interface{}{
        "product_id": req.ProductID,
        "duration_ms": time.Since(start).Milliseconds(),
    }, auction)
```

## 日志输出

所有日志以JSON格式输出到标准输出（stdout），便于：
- 日志收集系统采集
- 日志分析平台处理
- 日志搜索和查询
- 监控和告警

## 后续优化建议

1. **日志聚合**: 使用ELK或类似系统聚合各服务日志
2. **日志分析**: 建立日志分析仪表板
3. **告警规则**: 基于日志建立异常告警
4. **性能分析**: 基于duration字段进行性能分析
5. **链路可视化**: 使用Jaeger或Zipkin进行链路可视化

---

## 日志收集平台

### 概述

项目已集成 **Loki + Grafana** 轻量级日志收集与可视化平台，支持通过 `request_id` 进行微服务全链路日志查询。

### 架构

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Gateway   │     │   Product   │     │   Auction   │
│   Service   │     │   Service   │     │   Service   │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └───────────────────┼───────────────────┘
                           │ JSON日志 (stdout)
                           ▼
                    ┌─────────────┐
                    │  Promtail   │  日志采集代理
                    └──────┬──────┘
                           │
                           ▼
                    ┌─────────────┐
                    │    Loki     │  日志存储 (31天保留)
                    └──────┬──────┘
                           │
                           ▼
                    ┌─────────────┐
                    │   Grafana   │  可视化查询
                    └─────────────┘
```

### 组件说明

| 组件 | 版本 | 端口 | 说明 |
|------|------|------|------|
| Loki | 2.9.4 | 3100 | 日志存储，支持31天数据保留 |
| Promtail | 2.9.4 | - | 日志采集代理，自动采集Docker容器日志 |
| Grafana | 10.4.2 | 3001 | 可视化界面，预置仪表板 |

### 快速启动

```bash
# 进入日志平台目录
cd observability

# 启动所有服务
./start.sh start

# 或使用 docker-compose
docker compose up -d
```

### 访问 Grafana

- **URL**: http://localhost:3001
- **用户名**: `admin`
- **密码**: `admin`

### LogQL 查询示例

```logql
# 1. 按 request_id 查询全链路日志
{service_name=~".+"} |= "your-request-id"

# 2. 查询特定服务的日志
{service_name="gateway"}

# 3. 查询错误日志
{service_name=~".+", success="false"}

# 4. 查询特定操作类型
{service_name=~".+", operation_type="create"}

# 5. 统计请求量
sum by (service_name) (count_over_time({service_name=~".+"}[1h]))

# 6. 错误率统计
sum by (service_name) (count_over_time({service_name=~".+", success="false"}[1h]))
    / sum by (service_name) (count_over_time({service_name=~".+"}[1h]))
```

### 全链路追踪

#### 工作原理

1. **Gateway 生成 request_id**
   - 每个请求进入 Gateway 时生成或传递 `X-Request-ID`
   - 存储在日志的 `request_id` 字段

2. **服务间传递**
   - Gateway 转发请求时自动传递 `X-Request-ID`
   - 各服务从 HTTP Header 获取并记录

3. **查询全链路日志**
   ```logql
   {service_name=~".+"} |= "your-request-id"
   ```

#### 示例：追踪一次竞拍请求

```
请求流程: Client → Gateway → Auction Service

日志链路:
  1. Gateway: {request_id: "abc-123", path: "/api/auctions/1/bids", method: "POST"}
  2. Auction: {request_id: "abc-123", operation_type: "bid", object_id: "1"}
```

在 Grafana 中搜索 `abc-123` 即可看到完整链路。

### 配置文件

```
observability/
├── docker-compose.yaml          # Docker Compose 配置
├── start.sh                     # 启动/停止脚本
├── README.md                    # 详细使用文档
├── loki/
│   └── loki-config.yaml         # Loki 配置（保留策略等）
├── promtail/
│   └── promtail-config.yaml     # 日志采集配置
└── grafana/
    └── provisioning/
        ├── datasources/         # 自动配置 Loki 数据源
        └── dashboards/          # 预置仪表板
```

### 管理命令

```bash
./start.sh start     # 启动服务
./start.sh stop      # 停止服务
./start.sh restart   # 重启服务
./start.sh logs      # 查看日志
./start.sh status    # 查看状态
./start.sh clean     # 清理所有数据
```

### 日志保留策略

默认保留 **31 天**，可在 `observability/loki/loki-config.yaml` 中修改：

```yaml
limits_config:
  retention_period: 744h  # 31 days
```

### 预置仪表板

系统预置了「微服务日志仪表板」，包含：

1. **日志查询面板** - 实时日志流，支持文本搜索
2. **请求统计** - 总请求数、错误数、请求速率
3. **请求趋势图** - 时序图展示请求变化
4. **操作分布** - 饼图展示操作类型分布
5. **对象分布** - 饼图展示对象类型分布
6. **错误日志面板** - 仅显示错误日志

### 故障排查

#### 日志未显示

```bash
# 1. 检查服务是否输出日志
docker logs gateway

# 2. 检查 Promtail 是否正常
docker logs promtail

# 3. 检查 Loki 是否正常
curl http://localhost:3100/ready
```

#### Grafana 无法连接 Loki

1. 确认 Loki 运行正常
2. 在 Grafana 中检查数据源配置
3. 测试数据源连接

### 更多文档

详细的日志平台使用说明请参考：`observability/README.md`

---

## 监控指标系统

### 概述

项目已集成 **Prometheus + Grafana** 监控指标系统，支持业务指标统计和可视化，包括前端埋点和后端指标采集。

### 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        前端埋点                                 │
│  trackLiveRoomEnter / trackBidClick / trackPaymentStart        │
└──────────────────────────┬──────────────────────────────────────┘
                           │ POST /api/v1/track
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Gateway Service                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ 埋点 API    │  │ Metrics API │  │ 业务服务    │             │
│  │ /api/v1/track  │  │ /metrics    │  │             │             │
│  └──────┬──────┘  └──────┬──────┘  └─────────────┘             │
└─────────┼────────────────┼──────────────────────────────────────┘
          │                │
          ▼                ▼
   ┌──────────────┐   ┌──────────────┐
   │    Loki      │   │  Prometheus  │
   │   日志存储   │   │  指标存储    │
   └──────┬───────┘   └──────┬───────┘
          └─────────┬────────┘
                    ▼
            ┌──────────────┐
            │   Grafana    │
            │   可视化     │
            └──────────────┘
```

### 组件说明

| 组件 | 版本 | 端口 | 说明 |
|------|------|------|------|
| Prometheus | 2.52.0 | 9090 | 指标收集与存储，15天保留 |
| Grafana | 10.4.2 | 3002 | 统一可视化界面 |
| Loki | 2.9.4 | 3100 | 日志聚合存储 |

### 高并发压测与连接池资源治理

**问题**：
直播竞拍出价是典型高并发写场景。独立测试平台在 `100` 并发、`10` 个拍卖分片的吞吐压测中，曾出现 `0 / 500 / 502` 三类非业务失败。日志显示根因不是 JWT、fixture 或竞拍规则错误，而是本机短时间内大量新建到 `auction-service` 和 Redis 的 TCP 连接，触发 `can't assign requested address`，即本机临时端口被打满。

**方案**：
系统在两条关键链路补齐有界连接池治理：

- `gateway-service` 的 proxy HTTP client 使用连接复用和上游连接上限，避免每个转发请求都新建到 `auction-service` 的短连接。
- `auction-service` 的 Redis client 显式配置 `PoolSize / MaxActiveConns / MaxIdleConns / MinIdleConns`，把 Redis lock、通知、查询链路从无限扩连接改为可控复用。
- 压测平台将正常结束时的 `context canceled / deadline exceeded` 从失败统计中剔除，但保留真实网络错误和业务错误码，保证报表可解释。

**价值**：
连接池不是吞错或拒绝策略，而是通过连接复用和有界排队把“建连风暴”变成“可控背压”。它保护了本机端口、Redis 和下游服务，使压测结果从系统资源错误回到业务本身：热点分片下剩余 `400` 表示出价金额不足或已被超越，属于业务竞争；`0 / 500 / 502` 消失后，平台可以更准确地区分系统稳定性问题和真实业务冲突。

**验证证据**：

```text
吞吐压测配置：100 并发 / 10 拍卖分片 / 30s

修复前：
error_codes={"0":13023,"400":6650,"500":22,"502":41}

修复后：
error_codes={"400":7095}
```

### 已定义的业务指标

#### 直播间指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `live_room_enter_total` | Counter | room_id, user_type | 直播间进入次数 |
| `live_room_current_viewers` | Gauge | - | 当前观看人数 |
| `live_room_peak_viewers` | Gauge | room_id | 峰值观看人数 |

#### 竞拍指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `auction_created_total` | Counter | product_id, status | 竞拍创建数 |
| `auction_bid_total` | Counter | auction_id, status | 出价次数 |
| `auction_bid_amount` | Histogram | auction_id | 出价金额分布 |
| `auction_completed_total` | Counter | auction_id, has_winner | 竞拍完成数 |

#### 订单/成交指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `order_created_total` | Counter | auction_id, product_id | 订单创建数 |
| `order_completed_total` | Counter | auction_id, product_id | 订单成交数 |
| `order_amount` | Histogram | status | 订单金额分布 |

#### 支付指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `payment_initiated_total` | Counter | method | 发起支付次数 |
| `payment_completed_total` | Counter | method | 支付完成次数 |
| `payment_failed_total` | Counter | method, error_code | 支付失败次数 |
| `payment_amount` | Histogram | method | 支付金额分布 |

#### 用户指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `user_register_total` | Counter | source | 用户注册数 |
| `user_login_total` | Counter | method | 用户登录数 |
| `user_active_count` | Gauge | - | 活跃用户数 |

#### WebSocket 指标

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `websocket_connections` | Gauge | - | 当前连接数 |
| `websocket_messages_total` | Counter | type, direction | 消息总数 |
| `websocket_errors_total` | Counter | type | 错误总数 |

### 后端使用示例

#### 初始化指标

```go
// 在 main.go 中初始化
import "gateway-service/pkg/metrics"

func main() {
    m := metrics.Init("gateway")
    // ...
}
```

#### 记录业务指标

```go
// 记录直播间进入
m.RecordLiveRoomEnter("room-123", "vip")

// 记录出价
m.RecordAuctionBid("auction-456", 999.0, true)

// 记录订单完成（成交）
m.RecordOrderCompleted("auction-456", "product-789", 999.0)

// 记录支付
m.RecordPayment("alipay", 999.0, true, "")

// WebSocket 连接
m.IncWSConnections()
m.DecWSConnections()
```

### PromQL 查询示例

```promql
# 1小时内的直播间进入次数
sum(increase(live_room_enter_total[1h]))

# 当前 WebSocket 连接数
websocket_connections

# 成交次数（1小时）
sum(increase(order_completed_total[1h]))

# 成交金额（1小时）
sum(increase(order_amount_sum[1h]))

# 请求成功率
sum(rate(http_requests_total{status=~"2.."}[5m]))
  / sum(rate(http_requests_total[5m]))

# P95 延迟
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# 出价金额分布
histogram_quantile(0.50, sum(rate(auction_bid_amount_bucket[1h])) by (le))
histogram_quantile(0.90, sum(rate(auction_bid_amount_bucket[1h])) by (le))
```

### 前端埋点

#### 初始化 SDK

```typescript
import { initTracking } from '@/shared/tracking';

initTracking({
  endpoint: '/api/v1/track',
  debug: true,
  batchSize: 10,
  flushInterval: 5000,
});
```

#### 使用示例

```typescript
import { getTracker } from '@/shared/tracking';

// 直播间进入
getTracker().trackLiveRoomEnter('room-123', 'vip');

// 出价点击
getTracker().trackBidClick('auction-456', 999);

// 支付发起
getTracker().trackPaymentStart('order-789', 999, 'alipay');

// 用户注册
getTracker().trackUserRegister('wechat');

// 页面浏览
getTracker().trackPageView('auction_detail');

// 自定义事件
getTracker().trackCustom('button_click', {
  button_name: 'submit_bid',
  page: 'auction_detail',
});
```

### 配置文件

```
backend/pkg/metrics/
├── metrics.go      # 指标定义
└── handler.go      # 埋点 API

backend/gateway/
├── main.go         # 注册 metrics 路由
├── middleware/metrics.go  # Metrics 中间件
└── pkg/metrics/    # 指标包

frontend/shared/tracking/
└── index.ts        # 前端埋点 SDK

observability/prometheus/
└── prometheus.yml  # Prometheus 配置
```

### 访问地址

| 服务 | URL | 账号 |
|------|-----|------|
| Grafana | http://localhost:3002 | admin/admin |
| Prometheus | http://localhost:9090 | - |
| Metrics API | http://localhost:8080/metrics | - |
| 埋点 API | POST http://localhost:8080/api/v1/track | - |

### 预置仪表板

1. **业务监控仪表板** (`business-metrics.json`)
   - 直播间进入次数、当前观看人数
   - 成交次数、成交金额
   - 请求 QPS、延迟 P95
   - 出价趋势、支付统计、用户统计

2. **微服务日志仪表板** (`microservices-logs.json`)
   - 全链路日志查询
   - 错误日志过滤

### 测试埋点

```bash
# 测试直播间进入埋点
curl -X POST http://localhost:8080/api/v1/track \
  -H "Content-Type: application/json" \
  -d '{"event_type":"live_room_enter","params":{"room_id":"123","user_type":"vip"}}'

# 测试出价点击埋点
curl -X POST http://localhost:8080/api/v1/track \
  -H "Content-Type: application/json" \
  -d '{"event_type":"bid_click","params":{"auction_id":"456","current_price":999}}'

# 查看指标
curl http://localhost:8080/metrics | grep live_room_enter
```

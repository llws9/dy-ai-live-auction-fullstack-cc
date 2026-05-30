# 直播竞拍 OpenAPI + 轻量 Client SDK 设计

## 1. 背景与目标

本设计面向“主流直播平台后端”接入直播竞拍能力。接入方通常已经拥有用户体系、直播间体系、商品体系、订单系统、支付系统和履约链路，因此本项目不应要求平台替换这些基础能力。

SDK 的核心目标是把直播竞拍能力沉淀为稳定、可复用、可扩展的服务端接入契约：

- 提供平台后端可调用的 OpenAPI。
- 提供 Go、Java、Node 等轻量 Client SDK 封装 HTTP、签名、重试、类型定义。
- 托管竞拍规则、出价、排名、延时、结果确认和实时事件。
- 通过回调把竞拍结果交付给平台订单系统。
- 支持平台接入后继续做二次开发，不泄漏内部微服务实现。

不做的事情：

- 不把现有后端某个服务直接打包成 SDK。
- 不让平台前端持有平台级密钥。
- 不托管平台订单、支付、履约和售后。
- 不把内部数据库模型、微服务拆分、DAO/Service 结构暴露给接入方。

## 2. 方案结论

采纳方案：OpenAPI + 轻量 Client SDK。

第一接入对象：直播平台后端。

订单边界：我们只输出可信竞拍结果，平台订单系统负责创建订单、收款、履约。

结果闭环：竞拍结果回调默认要求平台同步返回 `external_order_id`。

超时处理：平台侧处理超时不是失败，而是未知态。我们必须通过平台提供的订单查询接口按 `idempotency_key` 探测订单状态。

强制平台契约：

- 平台必须基于 `idempotency_key` 做订单业务幂等。
- 平台必须提供 `GET /orders/by-idempotency-key/{idempotency_key}`。
- 平台回调与查询接口都必须支持签名校验。
- 平台重复消费同一竞拍结果时必须返回已有 `external_order_id`。

## 3. 总体架构

```text
Live Platform Backend
  -> OpenAPI Client SDK
  -> Live Auction OpenAPI Gateway
  -> Auction Domain
  -> Callback Outbox
  -> Platform Order Callback
  -> Platform Order System

Live Platform Frontend
  -> Platform Backend
  -> Realtime Token API
  -> Auction WebSocket
```

关键边界：

- 平台后端是可信接入方。
- 平台前端只能拿短期 realtime token，不能拿 `app_secret`。
- 我们的 OpenAPI Gateway 是 SDK 的唯一稳定入口。
- 当前内部 `gateway`、`auction`、`product` 服务可以继续演进，但不能成为外部契约的一部分。

### 3.1 扩展模式设计

扩展设计的目标不是堆叠设计模式，而是让“稳定竞拍内核”和“平台差异、横切能力、事件扩展”分离。核心竞拍规则必须直接、确定、可测试；平台差异和基础设施能力通过明确扩展点接入。

推荐组合：

```text
OpenAPI Client Proxy
  + Middleware Decorator Chain
  + Platform Adapter Port
  + Outbox-backed Event Mediator
```

#### 3.1.1 Proxy：SDK 统一入口

Client SDK 使用代理模式隐藏远端 OpenAPI 调用细节，让接入方像调用本地方法一样使用直播竞拍能力。

```text
Platform Backend
  -> LiveAuctionClient
       -> AuctionAPI
       -> BidAPI
       -> RealtimeAPI
       -> CallbackEventAPI
  -> OpenAPI Gateway
```

适用场景：

- 封装 HTTP 请求、响应解析、错误码映射。
- 隐藏 OpenAPI path、签名细节、序列化格式。
- 让不同语言 SDK 保持一致的业务方法命名。

不适用场景：

- 不承载竞拍业务规则。
- 不缓存关键竞拍状态作为事实源。
- 不替代服务端的权限校验和幂等校验。

#### 3.1.2 Decorator/Middleware：横切能力扩展

SDK 和服务端调用链都可以使用装饰器或中间件链处理横切能力。

```text
LiveAuctionClient
  -> SigningMiddleware
  -> IdempotencyMiddleware
  -> RetryMiddleware
  -> TimeoutMiddleware
  -> LoggingMiddleware
  -> MetricsMiddleware
  -> HTTP Transport
```

适用场景：

- 请求签名。
- 超时控制。
- 自动重试。
- `X-Request-Id` 注入。
- Trace、日志、指标采集。
- 错误码标准化。

不适用场景：

- 不把“是否允许出价”“是否触发延时”“谁是赢家”等业务规则放入 middleware。
- 不用 middleware 改写竞拍结果事实。
- 不在 middleware 中产生新的业务事件。

##### 3.1.2.1 统一错误码标准化

`TimeoutMiddleware`、`RetryMiddleware` 和订单探测逻辑必须输出统一错误码，避免 `callback_event.last_error_code`、`callback_delivery_attempt.error_type`、SDK 异常类型各自使用不同字符串。

本地错误码分为三类：

```text
Retryable
  可自动重试或继续探测。

Permanent
  不应自动重试，需要修复配置、签名、请求结构或平台权限。

Needs Manual Review
  不能简单判定成功或失败，需要人工确认或平台侧修复后重放。
```

回调投递错误码：

```text
CALLBACK_TIMEOUT
  回调请求超时。事件进入 unknown，随后触发订单探测。

CALLBACK_NETWORK_ERROR
  回调连接中断、DNS/TLS/连接失败。事件进入 unknown 或 retrying。

CALLBACK_SEND_FAILED
  请求未成功发出，例如 DNS 解析失败、TLS 握手失败、连接建立失败。事件可直接进入 retrying。

CALLBACK_RESPONSE_LOST
  请求可能已到达平台，但响应读取超时、连接 reset、半开连接断开。事件必须进入 unknown，再通过订单探测确认结果。

CALLBACK_HTTP_5XX
  平台回调接口返回 5xx。事件进入 retrying。

CALLBACK_HTTP_4XX_RETRYABLE
  平台回调接口返回可重试 4xx，例如 408、409、425、429。

CALLBACK_HTTP_4XX_PERMANENT
  平台回调接口返回不可恢复 4xx，例如 400、401、403。

CALLBACK_ACCEPTED_WITHOUT_ORDER_ID
  平台返回 202 ACCEPTED，但未返回 external_order_id。事件进入 probing。

CALLBACK_DUPLICATE_CONFIRMED
  平台返回 DUPLICATE 且包含 external_order_id。视为 succeeded。

CALLBACK_IDEMPOTENCY_CONFLICT
  平台返回同一 idempotency_key 已存在但 payload_hash 不一致。进入人工确认。

CALLBACK_RESPONSE_INVALID
  平台回调接口返回 2xx，但响应体无法解析、code 未定义，或 OK/DUPLICATE 缺少 external_order_id。
```

订单探测错误码：

```text
PROBE_TIMEOUT
  订单查询接口超时。继续 probing 或进入 retrying。

PROBE_NETWORK_ERROR
  订单查询接口网络错误。继续 probing 或进入 retrying。

PROBE_RATE_LIMITED
  订单查询接口返回 429。按 Retry-After 或平台级策略继续 probing。

PROBE_UNAUTHORIZED
  订单查询接口返回 401/403。进入 failed_permanent，除非平台配置修复后人工重放。

PROBE_IDEMPOTENCY_KEY_INVALID
  订单查询接口返回 400 IDEMPOTENCY_KEY_INVALID。说明本地事件或平台配置存在结构性问题，进入 failed_permanent。

PROBE_ORDER_SERVICE_UNAVAILABLE
  订单查询接口返回 500/502/503/504。继续 probing 或进入 retrying。

PROBE_UNEXPECTED_STATUS
  订单查询接口返回未定义 HTTP 状态码。按可重试错误处理。

PROBE_FOUND_WITHOUT_ORDER_ID
  查询返回 FOUND 但缺少 external_order_id。不能标记 succeeded，需要人工确认。

PROBE_RESPONSE_INVALID
  订单查询接口返回 2xx，但响应体无法解析、code 未定义，或必填字段缺失。

PROBE_CREATING_EXPIRED
  平台长期返回 CREATING，超过最大探测窗口。进入 dead_letter 或人工确认。

UNKNOWN_PROBE_CODE
  平台返回未定义业务 code。按可重试探测错误处理。
```

本地与调度错误码：

```text
MISSING_IDEMPOTENCY_KEY
  本地事件缺少 idempotency_key。进入 failed_permanent。

DISPATCH_LEASE_EXPIRED
  事件长时间停留 delivering，说明 dispatcher 可能崩溃。恢复为 unknown 或 retrying。

DISPATCH_CONCURRENCY_CONFLICT
  多 worker 抢占同一事件失败。不是业务错误，只记录并跳过。
```

错误码处理分类：

```text
Retryable:
  CALLBACK_TIMEOUT
  CALLBACK_NETWORK_ERROR
  CALLBACK_SEND_FAILED
  CALLBACK_RESPONSE_LOST
  CALLBACK_HTTP_5XX
  CALLBACK_HTTP_4XX_RETRYABLE
  CALLBACK_ACCEPTED_WITHOUT_ORDER_ID
  CALLBACK_RESPONSE_INVALID
  PROBE_TIMEOUT
  PROBE_NETWORK_ERROR
  PROBE_RATE_LIMITED
  PROBE_ORDER_SERVICE_UNAVAILABLE
  PROBE_UNEXPECTED_STATUS
  PROBE_RESPONSE_INVALID
  UNKNOWN_PROBE_CODE
  DISPATCH_LEASE_EXPIRED

Permanent:
  CALLBACK_HTTP_4XX_PERMANENT
  PROBE_UNAUTHORIZED
  PROBE_IDEMPOTENCY_KEY_INVALID
  MISSING_IDEMPOTENCY_KEY

Needs Manual Review:
  CALLBACK_IDEMPOTENCY_CONFLICT
  PROBE_FOUND_WITHOUT_ORDER_ID
  PROBE_CREATING_EXPIRED
  FAILED(retryable=false)

Informational:
  CALLBACK_DUPLICATE_CONFIRMED
  DISPATCH_CONCURRENCY_CONFLICT
```

错误码映射规则：

```text
Callback delivery:
  request timeout                         -> CALLBACK_TIMEOUT
  DNS/TLS/connect failed before send      -> CALLBACK_SEND_FAILED
  response read timeout/reset after send  -> CALLBACK_RESPONSE_LOST
  unknown whether request was sent        -> CALLBACK_RESPONSE_LOST
  HTTP 202 without external_order_id      -> CALLBACK_ACCEPTED_WITHOUT_ORDER_ID
  HTTP 2xx DUPLICATE with order id        -> CALLBACK_DUPLICATE_CONFIRMED
  HTTP 2xx OK/DUPLICATE without order id  -> CALLBACK_RESPONSE_INVALID
  HTTP 409 DUPLICATE with order id        -> CALLBACK_DUPLICATE_CONFIRMED
  HTTP 409 idempotency conflict           -> CALLBACK_IDEMPOTENCY_CONFLICT
  HTTP 408/425/429                        -> CALLBACK_HTTP_4XX_RETRYABLE
  HTTP 400/401/403/404                    -> CALLBACK_HTTP_4XX_PERMANENT
  HTTP 500/502/503/504                    -> CALLBACK_HTTP_5XX
  malformed JSON or unknown code          -> CALLBACK_RESPONSE_INVALID

Order probe:
  request timeout                         -> PROBE_TIMEOUT
  DNS/TLS/connect/reset error             -> PROBE_NETWORK_ERROR
  HTTP 429                                -> PROBE_RATE_LIMITED
  HTTP 400 IDEMPOTENCY_KEY_INVALID        -> PROBE_IDEMPOTENCY_KEY_INVALID
  HTTP 401/403                            -> PROBE_UNAUTHORIZED
  HTTP 500/502/503/504                    -> PROBE_ORDER_SERVICE_UNAVAILABLE
  other undefined HTTP status             -> PROBE_UNEXPECTED_STATUS
  HTTP 2xx malformed JSON                 -> PROBE_RESPONSE_INVALID
  HTTP 2xx unknown business code          -> UNKNOWN_PROBE_CODE
  code FOUND without external_order_id    -> PROBE_FOUND_WITHOUT_ORDER_ID
  code CREATING beyond max probe window   -> PROBE_CREATING_EXPIRED

Dispatcher:
  delivering lease timeout                -> DISPATCH_LEASE_EXPIRED
  failed to acquire event lease           -> DISPATCH_CONCURRENCY_CONFLICT
```

降级规则：

- `CALLBACK_TIMEOUT` 不能直接判失败，必须进入 `unknown -> probing`。
- `CALLBACK_SEND_FAILED` 说明请求未发出，可进入 `retrying`，但仍必须复用同一 `event_id` 和 `idempotency_key`。
- `CALLBACK_RESPONSE_LOST` 说明请求可能已到达平台，必须进入 `unknown -> probing`，不能直接重试。
- `CALLBACK_ACCEPTED_WITHOUT_ORDER_ID` 不能标记成功，必须进入 `probing`。
- `CALLBACK_DUPLICATE_CONFIRMED` 是成功态，应保存平台返回的 `external_order_id`。
- `CALLBACK_IDEMPOTENCY_CONFLICT` 说明双方幂等事实不一致，必须进入人工确认，不能自动重试。
- `CALLBACK_RESPONSE_INVALID` 不能标记成功，必须记录响应体并进入重试或人工确认。
- `PROBE_TIMEOUT` 不能生成新事件，只能继续探测或复用同一 `event_id`、`idempotency_key` 重试。
- `PROBE_FOUND_WITHOUT_ORDER_ID` 不能标记成功，也不应盲目重试回调，应进入人工确认或死信处理。
- `PROBE_RESPONSE_INVALID` 不能标记成功，应继续探测；连续出现时进入死信并告警。
- `PROBE_CREATING_EXPIRED` 不能继续无限探测，应进入 dead_letter 并告警。
- `PROBE_UNAUTHORIZED` 默认不可自动重试，避免错误配置持续冲击平台接口。
- `DISPATCH_LEASE_EXPIRED` 应触发恢复扫描，不能生成新事件。
- `DISPATCH_CONCURRENCY_CONFLICT` 只代表当前 worker 未抢到事件，不应记录为业务失败。

#### 3.1.3 Adapter：平台差异隔离

不同直播平台的用户、直播间、商品、订单接口差异很大，应通过 Adapter 隔离，而不是污染竞拍核心。

```text
PlatformIntegrationPort
  -> DouyinAdapter
  -> KuaishouAdapter
  -> TaobaoLiveAdapter
  -> CustomPlatformAdapter
```

建议端口：

```text
UserMappingPort
LiveStreamMappingPort
ProductMappingPort
OrderCallbackPort
OrderProbePort
NotificationPort
```

适用场景：

- 外部用户 ID 与内部 `auction_user_id` 映射。
- 外部商品 ID 与竞拍商品快照映射。
- 平台订单回调地址、签名算法、响应格式差异。
- 平台订单查询接口字段差异。

不适用场景：

- 不让平台 Adapter 直接访问竞拍数据库。
- 不让 Adapter 决定竞拍状态机。
- 不在 Adapter 内实现出价排序、延时竞拍、赢家判定。

#### 3.1.4 Mediator/Event Dispatcher：事件编排

中介模式适合做事件编排，不适合承载核心竞拍规则。竞拍域只产生领域事件，事件分发器负责通知下游扩展能力。

```text
Auction Domain
  -> Domain Event: auction.result_confirmed
  -> Outbox
  -> Event Dispatcher
       -> OrderCallbackHandler
       -> RealtimePushHandler
       -> AuditLogHandler
       -> MetricsHandler
       -> NotificationHandler
```

适用场景：

- `auction.result_confirmed` 触发订单回调。
- `bid_placed` 触发实时推送和审计日志。
- `delay_triggered` 触发实时倒计时刷新。
- `callback.dead_letter` 触发告警。

不适用场景：

- 不用一个巨大的 `PlatformMediator` 处理所有平台逻辑。
- 不用内存事件替代可靠 Outbox。
- 不让事件 handler 反向修改已确认的竞拍结果。

#### 3.1.5 Strategy：平台级策略配置

当不同平台存在策略差异时使用策略模式，而不是在主流程中写大量条件分支。

建议策略：

```text
SignatureStrategy
RetryPolicyStrategy
OrderProbeStrategy
CallbackTimeoutStrategy
RateLimitStrategy
```

适用场景：

- 不同平台签名算法不同。
- 不同平台回调超时时间不同。
- 不同平台重试节奏不同。
- 不同平台订单查询接口稳定性不同。

不适用场景：

- 不用策略模式改变竞拍公平性规则。
- 不允许平台自定义赢家判定逻辑，除非未来明确开放独立规则引擎。

#### 3.1.6 设计底线

必须遵守：

- 核心竞拍规则不用模式堆叠，保持直接、确定、可测试。
- 外部平台差异用 Adapter 隔离。
- 横切能力用 Middleware/Decorator 扩展。
- SDK 入口用 Proxy 封装 OpenAPI。
- 回调和扩展能力用 Outbox-backed Event Dispatcher 编排。
- 可靠事件必须先落库再投递，不能只依赖内存事件。

推荐内部层次：

```text
Core Domain
  AuctionService
  BidService
  RankingService
  ResultService

Extension Ports
  OrderCallbackPort
  OrderProbePort
  RealtimePushPort
  NotificationPort
  AuditPort
  MetricsPort

Application Orchestration
  AuctionEventDispatcher
  CallbackDispatcher
  ReplayScheduler

Infrastructure Adapters
  HTTPCallbackAdapter
  PlatformOrderProbeAdapter
  WebSocketPushAdapter
  MetricsAdapter

Client SDK
  OpenAPIClient Proxy
  Middleware Decorator Chain
  Typed API Modules
```

## 4. OpenAPI 能力分组

### 4.1 Platform API

用于平台配置、权限和回调管理。

```http
GET    /platform/profile
POST   /platform/secrets/rotate
GET    /platform/scopes
PUT    /platform/callbacks
POST   /platform/callbacks/test
```

### 4.2 Identity Mapping API

用于把平台外部实体映射到竞拍域稳定 ID。

```http
POST   /mappings/users:upsert
POST   /mappings/live-streams:upsert
POST   /mappings/products:upsert
GET    /mappings/users/{external_user_id}
GET    /mappings/live-streams/{external_live_stream_id}
GET    /mappings/products/{external_product_id}
```

用户映射请求：

```json
{
  "external_user_id": "platform_user_10001",
  "nickname": "Alice",
  "avatar_url": "https://cdn.example.com/a.png",
  "metadata": {
    "level": "vip"
  }
}
```

商品映射请求：

```json
{
  "external_product_id": "sku_888",
  "title": "限量球鞋",
  "description": "平台商品描述",
  "images": ["https://cdn.example.com/p.png"],
  "category": "fashion",
  "metadata": {}
}
```

直播间映射请求：

```json
{
  "external_live_stream_id": "room_123",
  "title": "潮品专场",
  "anchor_id": "anchor_001",
  "status": "live",
  "started_at": 1780070000000
}
```

### 4.3 Auction Core API

用于创建、启动、取消、结束竞拍，以及查询竞拍状态。

```http
POST   /auctions
GET    /auctions/{auction_id}
PATCH  /auctions/{auction_id}
POST   /auctions/{auction_id}:start
POST   /auctions/{auction_id}:cancel
POST   /auctions/{auction_id}:finish
GET    /auctions/{auction_id}/result
GET    /auctions?external_live_stream_id=room_123&status=running
```

创建竞拍请求：

```json
{
  "external_live_stream_id": "room_123",
  "external_product_id": "sku_888",
  "rule": {
    "start_price": 100,
    "increment": 10,
    "cap_price": 1000,
    "duration_seconds": 300,
    "delay_trigger_before_seconds": 15,
    "delay_duration_seconds": 30,
    "max_delay_seconds": 180
  },
  "start_mode": "manual",
  "metadata": {}
}
```

### 4.4 Bid API

平台后端代用户提交出价。第一版不建议让平台前端直接通过 WebSocket 提交可信出价。

```http
POST   /auctions/{auction_id}/bids
GET    /auctions/{auction_id}/bids
GET    /auctions/{auction_id}/ranking
GET    /auctions/{auction_id}/snapshot
```

出价请求：

```json
{
  "external_user_id": "platform_user_10001",
  "amount": 260,
  "client_bid_id": "platform_bid_abc",
  "source": "platform_server"
}
```

出价响应：

```json
{
  "bid_id": "bid_789",
  "auction_id": "auc_123",
  "accepted": true,
  "current_price": 260,
  "winner_user_id": "auc_user_9f3a",
  "rank": 1,
  "delay_triggered": false,
  "server_time": 1780070000000
}
```

### 4.5 Realtime API

用于给平台前端签发短期 WebSocket 连接令牌。

```http
POST   /realtime/tokens
GET    /auctions/{auction_id}/snapshot
```

请求：

```json
{
  "auction_id": "auc_123",
  "external_user_id": "platform_user_10001",
  "ttl_seconds": 300,
  "permissions": ["read_auction", "receive_events"]
}
```

响应：

```json
{
  "ws_url": "wss://api.example.com/openapi/v1/ws",
  "token": "short_lived_realtime_token",
  "expires_at": 1780070300000
}
```

WebSocket 事件类型：

```text
bid_placed
rank_update
overtaken
delay_triggered
auction_ended
time_sync
notification
```

### 4.6 Callback Event API

用于我们侧管理事件投递、查询和重放。

```http
GET  /callback-events
POST /callback-events/{event_id}:replay
POST /callback-events:replay
```

该组接口面向我们自己的管理端或平台运维控制台，不是竞拍业务调用主路径。

## 5. 平台级鉴权设计

### 5.1 平台凭证

平台注册后获得：

- `app_id`
- `app_secret`
- `scopes`
- `callback_url`
- `callback_policy`

### 5.2 请求头

所有平台后端请求必须携带：

```http
X-App-Id: live_platform_001
X-Timestamp: 1780070000000
X-Nonce: random-uuid
X-Signature: hmac_sha256(method + path + timestamp + nonce + body_hash, app_secret)
X-Request-Id: req_001
```

### 5.3 校验顺序

```text
1. 校验 app_id 是否存在。
2. 校验 timestamp 是否在允许窗口内，例如 5 分钟。
3. 校验 nonce 是否未使用，防重放。
4. 校验 body_hash 与 signature。
5. 校验 endpoint 所需 scope。
6. 校验 X-Request-Id 幂等性。
7. 执行业务逻辑。
```

### 5.4 Scope 建议

```text
auction:read
auction:write
bid:write
mapping:write
realtime:token
callback:manage
```

## 6. 竞拍结果回调设计

### 6.1 事件类型

平台订单系统只应依赖最终成交事实事件：

```text
auction.result_confirmed
```

如果需要扩展状态通知，可增加：

```text
auction.ended
auction.cancelled
auction.delay_triggered
```

但订单创建只能依赖 `auction.result_confirmed`。

### 6.2 回调请求头

```http
POST /platform/callback/live-auction
Content-Type: application/json
X-App-Id: live_platform_001
X-Event-Id: evt_01JZ_RESULT_000001
X-Event-Type: auction.result_confirmed
X-Timestamp: 1780070000000
X-Nonce: random-uuid
X-Signature: hmac_sha256(timestamp + nonce + raw_body, app_secret)
X-Delivery-Attempt: 1
X-Idempotency-Key: auction_result:auc_123:v1
```

### 6.3 回调 Payload

```json
{
  "event_id": "evt_01JZ_RESULT_000001",
  "event_type": "auction.result_confirmed",
  "event_version": "1.0",
  "occurred_at": 1780070000000,
  "app_id": "live_platform_001",
  "trace_id": "trace_abc",
  "idempotency_key": "auction_result:auc_123:v1",
  "auction": {
    "auction_id": "auc_123",
    "external_auction_id": "platform_auc_888",
    "status": "ended",
    "started_at": 1780069700000,
    "ended_at": 1780070000000,
    "result_version": 1
  },
  "live_stream": {
    "external_live_stream_id": "room_123",
    "title": "潮品专场",
    "anchor_id": "anchor_001"
  },
  "product": {
    "external_product_id": "sku_888",
    "title": "限量球鞋",
    "snapshot": {
      "description": "平台商品描述",
      "images": ["https://cdn.example.com/p.png"],
      "category": "fashion"
    }
  },
  "winner": {
    "external_user_id": "platform_user_10001",
    "auction_user_id": "auc_user_9f3a",
    "nickname": "Alice"
  },
  "pricing": {
    "currency": "CNY",
    "start_price": 100,
    "final_price": 260,
    "increment": 10,
    "bid_count": 12
  },
  "order_suggestion": {
    "suggested_order_type": "auction_win",
    "pay_amount": 260,
    "quantity": 1,
    "expire_at": 1780071800000
  },
  "metadata": {
    "source": "live_auction_openapi"
  }
}
```

商品必须携带成交瞬间快照，避免平台商品后续变更影响订单事实。

### 6.4 同步成功响应

```json
{
  "code": "OK",
  "message": "order created",
  "external_order_id": "order_987654",
  "idempotency_key": "auction_result:auc_123:v1",
  "order_status": "pending_payment"
}
```

重复消费响应：

```json
{
  "code": "DUPLICATE",
  "message": "order already created",
  "external_order_id": "order_987654",
  "idempotency_key": "auction_result:auc_123:v1",
  "order_status": "pending_payment"
}
```

我们侧把 `2xx + OK`、`2xx + DUPLICATE` 和可配置的 `409 + DUPLICATE` 都视为投递成功。

### 6.5 平台异步接收响应

如果平台订单创建耗时超过回调 SLA，应快速返回 `202 ACCEPTED`：

```json
{
  "code": "ACCEPTED",
  "message": "order creation accepted",
  "idempotency_key": "auction_result:auc_123:v1",
  "external_order_id": null,
  "order_status": "creating"
}
```

我们侧不把 `202 ACCEPTED` 视为最终成功，而是进入订单探测流程。

## 7. 平台订单查询接口

### 7.1 接口定位

平台必须提供：

```http
GET /orders/by-idempotency-key/{idempotency_key}
```

该接口用于处理回调超时、连接中断、`202 ACCEPTED` 等未知态。

接口规则：

- 只读。
- 不能创建订单。
- 不能改变订单状态。
- 只能返回平台侧基于 `idempotency_key` 记录到的事实。
- 必须验签。

### 7.2 请求

```http
GET /orders/by-idempotency-key/auction_result%3Aauc_123%3Av1?include_detail=false
X-App-Id: live_platform_001
X-Timestamp: 1780070000000
X-Nonce: random-uuid
X-Signature: hmac_sha256(method + path + timestamp + nonce + body_hash, app_secret)
X-Request-Id: req_probe_001
```

Path 参数：

```yaml
idempotency_key:
  type: string
  required: true
  example: auction_result:auc_123:v1
```

Query 参数：

```yaml
include_detail:
  type: boolean
  required: false
  default: false
```

### 7.3 响应：FOUND

```http
200 OK
```

```json
{
  "code": "FOUND",
  "message": "order found",
  "idempotency_key": "auction_result:auc_123:v1",
  "external_order_id": "order_987654",
  "order_status": "pending_payment",
  "created_at": 1780070001200,
  "updated_at": 1780070001200,
  "amount": {
    "currency": "CNY",
    "pay_amount": 260
  }
}
```

处理：我们侧标记事件 `succeeded`，保存 `external_order_id`。

### 7.4 响应：CREATING

```http
200 OK
```

```json
{
  "code": "CREATING",
  "message": "order creation is still in progress",
  "idempotency_key": "auction_result:auc_123:v1",
  "external_order_id": null,
  "order_status": "creating",
  "created_at": null,
  "updated_at": 1780070002000
}
```

处理：我们侧继续探测，不立即重试回调。

### 7.5 响应：NOT_FOUND

```http
404 Not Found
```

```json
{
  "code": "NOT_FOUND",
  "message": "idempotency key not found",
  "idempotency_key": "auction_result:auc_123:v1"
}
```

处理：宽限期内继续探测；超过宽限期转自动重试。

### 7.6 响应：FAILED

```http
200 OK
```

```json
{
  "code": "FAILED",
  "message": "order creation failed permanently",
  "idempotency_key": "auction_result:auc_123:v1",
  "external_order_id": null,
  "order_status": "failed",
  "failure": {
    "reason_code": "PRODUCT_UNAVAILABLE",
    "reason_message": "product is unavailable in platform order system",
    "retryable": false
  }
}
```

处理：

- `retryable=true`：进入自动重试。
- `retryable=false`：进入 `failed_permanent` 或 `dead_letter`，等待人工处理。

### 7.7 错误码

```text
FOUND
CREATING
NOT_FOUND
FAILED
INVALID_SIGNATURE
APP_UNAUTHORIZED
IDEMPOTENCY_KEY_INVALID
RATE_LIMITED
INTERNAL_ERROR
ORDER_SERVICE_UNAVAILABLE
```

HTTP 状态码建议：

```text
200 FOUND / CREATING / FAILED
400 IDEMPOTENCY_KEY_INVALID
401 INVALID_SIGNATURE
403 APP_UNAUTHORIZED
404 NOT_FOUND
429 RATE_LIMITED
500 INTERNAL_ERROR
503 ORDER_SERVICE_UNAVAILABLE
```

## 8. 回调状态机

```text
pending
  -> delivering
  -> succeeded

pending
  -> delivering
  -> unknown
  -> probing
  -> succeeded

unknown
  -> probing
  -> retrying
  -> delivering
  -> succeeded

retrying
  -> dead_letter

delivering
  -> failed_permanent
```

状态含义：

- `pending`：事件已生成，未投递。
- `delivering`：正在投递。
- `succeeded`：平台已返回 `external_order_id` 或明确重复成功。
- `unknown`：请求超时或连接中断，平台处理结果未知。
- `probing`：正在通过幂等键查询平台订单。
- `retrying`：等待下一次自动重试。
- `dead_letter`：自动重试耗尽，等待人工重放。
- `failed_permanent`：平台明确拒绝且不可重试。

## 9. 重试、探测与重放策略

### 9.1 超时降级顺序

```text
timeout
  -> unknown
  -> query by idempotency_key
  -> retry same event_id and idempotency_key
  -> dead_letter
  -> manual/API replay
```

不能在超时后直接生成新事件，也不能更换 `idempotency_key`。

### 9.2 推荐参数

```json
{
  "callback_timeout_ms": 3000,
  "probe_after_timeout_ms": 5000,
  "probe_max_attempts": 3,
  "probe_max_window_seconds": 300,
  "probe_interval_ms": 10000,
  "retry_schedule_seconds": [10, 30, 60, 180, 300, 900, 1800, 3600],
  "retry_jitter_ratio": 0.2,
  "per_platform_max_concurrency": 20,
  "dispatcher_lease_seconds": 60,
  "retry_after_max_seconds": 3600,
  "dead_letter_after_seconds": 86400
}
```

### 9.3 重放接口

单事件重放：

```http
POST /callback-events/{event_id}:replay
```

请求：

```json
{
  "reason": "platform_order_service_recovered",
  "force": false
}
```

批量重放：

```http
POST /callback-events:replay
```

请求：

```json
{
  "app_id": "live_platform_001",
  "event_type": "auction.result_confirmed",
  "status": "dead_letter",
  "created_from": 1780060000000,
  "created_to": 1780070000000,
  "limit": 100,
  "reason": "platform incident recovered"
}
```

重放规则：

- 默认重放不生成新 `event_id`。
- 默认重放不改变 `idempotency_key`。
- 结果纠错必须提升 `result_version`，生成新的 `idempotency_key`。
- 手动重放必须记录 `operator`、`reason`、`time`。
- 批量重放必须限流。

### 9.4 网络中断与崩溃恢复边界场景

网络中断不能只按“失败重试”处理。核心判断是：请求是否可能已经到达平台订单系统。如果可能到达，就必须先探测订单状态，避免重复建单。

#### 9.4.1 回调请求未发出

场景：

```text
DNS 解析失败
TLS 握手失败
TCP connect 失败
连接池获取连接失败
请求体尚未写出前失败
```

处理：

```text
error_type = CALLBACK_SEND_FAILED
callback_event.status = retrying
next_retry_at = nextRetryTime(attempt_count) + jitter
```

约束：

- 不生成新 `event_id`。
- 不更换 `idempotency_key`。
- 必须记录 `callback_delivery_attempt`。

#### 9.4.2 请求可能已到达但响应丢失

场景：

```text
请求体已写出后读取响应超时
平台处理过程中连接 reset
半开连接断开
代理层超时但平台服务可能继续执行
客户端进程收到 EOF，但无法确认平台是否处理
```

处理：

```text
error_type = CALLBACK_RESPONSE_LOST
callback_event.status = unknown
next_probe_at = now + probe_after_timeout_ms
```

恢复顺序：

```text
unknown
  -> probing by idempotency_key
  -> FOUND: succeeded
  -> CREATING: continue probing
  -> NOT_FOUND after grace window: retrying
  -> FAILED(retryable=false): failed_permanent/dead_letter
```

约束：

- 不能直接重试回调。
- 不能把网络异常判定为订单创建失败。
- 探测和后续重试必须复用同一个 `event_id`、`idempotency_key`。

#### 9.4.3 平台返回重复消费

场景：

```text
HTTP 2xx + DUPLICATE + external_order_id
HTTP 409 + DUPLICATE + external_order_id
```

处理：

```text
error_type = CALLBACK_DUPLICATE_CONFIRMED
callback_event.status = succeeded
external_order_id = response.external_order_id
```

说明：

- 该场景不是错误，是可靠重试后的成功闭环。
- `409` 不能被无条件映射为 `CALLBACK_HTTP_4XX_RETRYABLE`，必须先解析业务 code。

#### 9.4.4 幂等冲突

场景：

```text
平台返回同一 idempotency_key 已存在
但 payload_hash、auction_id、winner 或 final_price 不一致
```

处理：

```text
error_type = CALLBACK_IDEMPOTENCY_CONFLICT
callback_event.status = dead_letter
```

约束：

- 不自动重试。
- 不自动生成新结果版本。
- 必须人工确认双方事实源，以决定是否纠错并提升 `result_version`。

#### 9.4.5 平台长期返回 CREATING

场景：

```text
订单探测接口持续返回 CREATING
超过 probe_max_window_seconds 或 probe_max_attempts
```

处理：

```text
error_type = PROBE_CREATING_EXPIRED
callback_event.status = dead_letter
```

约束：

- 不继续无限探测。
- 不直接重新生成事件。
- 需要告警并等待平台修复后重放或人工处理。

#### 9.4.6 Dispatcher 崩溃恢复

场景：

```text
callback_event.status = delivering
但 locked_until 已过期
或 updated_at 超过 dispatcher_lease_seconds 未变化
```

处理：

```text
error_type = DISPATCH_LEASE_EXPIRED
if delivery_attempt 已发出请求:
  callback_event.status = unknown
  next_probe_at = now + probe_after_timeout_ms
else:
  callback_event.status = retrying
```

约束：

- dispatcher 领取事件必须使用 lease 机制。
- 推荐使用 `SELECT ... FOR UPDATE SKIP LOCKED` 或等价的条件更新。
- 恢复扫描不能生成新业务事件。

#### 9.4.7 并发投递抢占

场景：

```text
多个 worker 同时扫描到同一个 pending/retrying 事件
其中一个 worker 成功获得 lease
其他 worker 抢占失败
```

处理：

```text
error_type = DISPATCH_CONCURRENCY_CONFLICT
不改变 callback_event 业务状态
当前 worker 跳过
```

说明：

- 该场景不是业务失败。
- 只记录调度日志或指标，不写入 `last_error_code`。

#### 9.4.8 重试风暴防护

平台不可用时，必须避免所有事件同时重试。

规则：

- 每个平台独立限流，使用 `per_platform_max_concurrency`。
- 重试时间必须增加 jitter，避免同一时间批量打爆平台。
- `Retry-After` 只在合理范围内生效，超过 `retry_after_max_seconds` 时按最大值截断。
- 平台连续 5xx 或超时达到阈值后，开启 platform-level circuit breaker。
- 熔断期间不丢弃事件，只推迟 `next_retry_at`。

## 10. 幂等设计

### 10.1 我们侧幂等

```text
idempotency_key = auction_result:{auction_id}:v{result_version}
payload_hash = sha256(canonical_json(payload))
```

唯一约束：

```sql
UNIQUE(app_id, event_id)
UNIQUE(app_id, idempotency_key)
```

重复生成事件时：

- `idempotency_key` 已存在且 `payload_hash` 相同：返回已有事件。
- `idempotency_key` 已存在但 `payload_hash` 不同：拒绝。
- 结果需要变更：提升 `result_version`。

### 10.2 平台侧幂等

平台订单系统必须用 `idempotency_key` 做业务幂等，不应只依赖 `event_id`。

推荐事务：

```text
BEGIN

SELECT * FROM order_idempotency
WHERE idempotency_key = ?
FOR UPDATE

IF exists:
  return DUPLICATE + external_order_id

validate auction result payload
validate product
validate winner
validate amount
create order
insert order_idempotency(idempotency_key, external_order_id, payload_hash)

COMMIT

return OK + external_order_id
```

## 11. 数据表建议

### 11.1 callback_event

```sql
callback_event
- id
- event_id
- app_id
- event_type
- event_version
- aggregate_type
- aggregate_id
- idempotency_key
- payload_json
- payload_hash
- status
- external_order_id
- external_order_status
- accepted_at
- succeeded_at
- unknown_at
- next_probe_at
- next_retry_at
- attempt_count
- probe_count
- max_attempts
- last_error_code
- last_error_message
- created_at
- updated_at
```

### 11.2 callback_delivery_attempt

```sql
callback_delivery_attempt
- id
- event_id
- app_id
- attempt_no
- delivery_mode       -- callback | probe | replay
- request_url
- request_headers_json
- request_body_hash
- response_status
- response_body
- external_order_id
- error_type
- error_message
- started_at
- finished_at
- duration_ms
```

`error_type` 必须使用统一错误码：

```text
CALLBACK_TIMEOUT
CALLBACK_NETWORK_ERROR
CALLBACK_SEND_FAILED
CALLBACK_RESPONSE_LOST
CALLBACK_HTTP_5XX
CALLBACK_HTTP_4XX_RETRYABLE
CALLBACK_HTTP_4XX_PERMANENT
CALLBACK_ACCEPTED_WITHOUT_ORDER_ID
CALLBACK_DUPLICATE_CONFIRMED
CALLBACK_IDEMPOTENCY_CONFLICT
CALLBACK_RESPONSE_INVALID
PROBE_TIMEOUT
PROBE_NETWORK_ERROR
PROBE_RATE_LIMITED
PROBE_UNAUTHORIZED
PROBE_IDEMPOTENCY_KEY_INVALID
PROBE_ORDER_SERVICE_UNAVAILABLE
PROBE_UNEXPECTED_STATUS
PROBE_FOUND_WITHOUT_ORDER_ID
PROBE_RESPONSE_INVALID
PROBE_CREATING_EXPIRED
UNKNOWN_PROBE_CODE
MISSING_IDEMPOTENCY_KEY
DISPATCH_LEASE_EXPIRED
DISPATCH_CONCURRENCY_CONFLICT
```

## 12. 我们侧订单探测代码逻辑

### 12.1 数据结构

```go
type ProbeOrderResponse struct {
	Code            string       `json:"code"`
	Message         string       `json:"message"`
	IdempotencyKey  string       `json:"idempotency_key"`
	ExternalOrderID *string      `json:"external_order_id"`
	OrderStatus     *string      `json:"order_status"`
	CreatedAt       *int64       `json:"created_at"`
	UpdatedAt       *int64       `json:"updated_at"`
	Amount          *OrderAmount `json:"amount"`
	Failure         *FailureInfo `json:"failure"`
}

type OrderAmount struct {
	Currency  string  `json:"currency"`
	PayAmount float64 `json:"pay_amount"`
}

type FailureInfo struct {
	ReasonCode    string `json:"reason_code"`
	ReasonMessage string `json:"reason_message"`
	Retryable     bool   `json:"retryable"`
}
```

### 12.2 探测主流程

```go
func ProbePlatformOrder(ctx context.Context, event CallbackEvent) error {
	if event.IdempotencyKey == "" {
		return MarkFailedPermanent(event.EventID, "MISSING_IDEMPOTENCY_KEY")
	}

	MarkEventStatus(event.EventID, "probing")

	resp, err := platformClient.GetOrderByIdempotencyKey(ctx, event.AppID, event.IdempotencyKey)
	if err != nil {
		return handleProbeTransportError(event, err)
	}

	switch resp.Code {
	case "FOUND":
		if resp.ExternalOrderID == nil || *resp.ExternalOrderID == "" {
			return MarkNeedsManualReview(event.EventID, "PROBE_FOUND_WITHOUT_ORDER_ID")
		}

		return MarkSucceeded(event.EventID, SucceededPatch{
			ExternalOrderID:     *resp.ExternalOrderID,
			ExternalOrderStatus: valueOrEmpty(resp.OrderStatus),
		})

	case "CREATING":
		return ScheduleNextProbe(event.EventID, ProbeSchedule{
			Reason:      "platform_order_creating",
			NextProbeAt: nextProbeTime(event.ProbeCount),
		})

	case "NOT_FOUND":
		if withinProbeGracePeriod(event) {
			return ScheduleNextProbe(event.EventID, ProbeSchedule{
				Reason:      "not_found_within_grace_period",
				NextProbeAt: shortProbeTime(),
			})
		}

		return ScheduleCallbackRetry(event.EventID, RetrySchedule{
			Reason:      "platform_order_not_found",
			NextRetryAt: nextRetryTime(event.AttemptCount),
		})

	case "FAILED":
		if resp.Failure != nil && resp.Failure.Retryable {
			return ScheduleCallbackRetry(event.EventID, RetrySchedule{
				Reason:      resp.Failure.ReasonCode,
				NextRetryAt: nextRetryTime(event.AttemptCount),
			})
		}

		return MarkFailedPermanent(event.EventID, failureReason(resp))

	default:
		return MarkProbeRetry(event.EventID, "UNKNOWN_PROBE_CODE:"+resp.Code)
	}
}
```

### 12.3 HTTP 调用逻辑

```go
func (c *PlatformClient) GetOrderByIdempotencyKey(
	ctx context.Context,
	appID string,
	idempotencyKey string,
) (*ProbeOrderResponse, error) {
	path := "/orders/by-idempotency-key/" + url.PathEscape(idempotencyKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path+"?include_detail=false", nil)
	if err != nil {
		return nil, err
	}

	timestamp := nowMillis()
	nonce := newNonce()
	bodyHash := sha256Hex([]byte{})
	signature := SignHMAC(c.Secret, http.MethodGet, path, timestamp, nonce, bodyHash)

	req.Header.Set("X-App-Id", appID)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Request-Id", newRequestID())

	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	switch res.StatusCode {
	case http.StatusOK:
		var out ProbeOrderResponse
		if err := json.Unmarshal(body, &out); err != nil {
			return nil, NewRetryableProbeError("PROBE_RESPONSE_INVALID", nil)
		}
		return &out, nil

	case http.StatusNotFound:
		return &ProbeOrderResponse{
			Code:           "NOT_FOUND",
			IdempotencyKey: idempotencyKey,
		}, nil

	case http.StatusBadRequest:
		return nil, NewPermanentProbeError("PROBE_IDEMPOTENCY_KEY_INVALID")

	case http.StatusTooManyRequests:
		return nil, NewRetryableProbeError("PROBE_RATE_LIMITED", retryAfter(res.Header))

	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return nil, NewRetryableProbeError("PROBE_ORDER_SERVICE_UNAVAILABLE", nil)

	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, NewPermanentProbeError("PROBE_UNAUTHORIZED")

	default:
		return nil, NewRetryableProbeError("PROBE_UNEXPECTED_STATUS", nil)
	}
}
```

### 12.4 调度策略

```go
func nextProbeTime(probeCount int) time.Time {
	schedule := []time.Duration{
		5 * time.Second,
		10 * time.Second,
		30 * time.Second,
	}

	if probeCount < len(schedule) {
		return time.Now().Add(schedule[probeCount])
	}

	return time.Now().Add(60 * time.Second)
}

func nextRetryTime(attemptCount int) time.Time {
	schedule := []time.Duration{
		10 * time.Second,
		30 * time.Second,
		60 * time.Second,
		3 * time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
		1 * time.Hour,
	}

	if attemptCount < len(schedule) {
		return time.Now().Add(schedule[attemptCount])
	}

	return time.Now().Add(1 * time.Hour)
}
```

## 13. MVP 范围

必须实现：

- 平台签名鉴权。
- 用户、直播间、商品映射。
- 创建竞拍、启动竞拍、出价、排名、快照、结果。
- 短期 realtime token。
- `auction.result_confirmed` 回调。
- 平台同步返回 `external_order_id`。
- 平台订单查询接口。
- `callback_event` 与 `callback_delivery_attempt`。
- 固定重试策略。
- 单事件重放 API。
- `event_id` 与 `idempotency_key` 双唯一约束。

暂缓实现：

- 平台自助注册。
- 多平台级可视化回调策略配置。
- 批量重放管理页面。
- 统计报表。
- 关注体系。
- 通知中心。
- 点天灯。
- 订单完整履约。

## 14. 验收标准

平台接入验收：

- 平台能通过签名调用 OpenAPI。
- 平台能完成用户、直播间、商品映射。
- 平台能创建竞拍并完成出价。
- 竞拍结束后平台能收到 `auction.result_confirmed`。
- 平台能同步返回 `external_order_id`。
- 回调超时时，我们能通过 `idempotency_key` 查询到订单或进入安全重试。
- 重复投递不会导致平台重复创建订单。
- 死信事件能被单事件重放。

安全验收：

- 请求签名不可绕过。
- `timestamp + nonce` 防重放有效。
- 平台不能访问其他平台的事件和订单探测结果。
- 平台前端不能获得 `app_secret`。

可靠性验收：

- 回调事件先落库再投递。
- 每次投递和探测都有 attempt 记录。
- 超时不产生新事件。
- 重试复用同一个 `event_id` 与 `idempotency_key`。
- `FOUND` 但无 `external_order_id` 不可标记成功。

## 15. 后续实施建议

第一阶段实现服务端 OpenAPI 契约和核心回调链路。

第二阶段基于 OpenAPI 生成或手写 Go Client SDK，验证接入体验。

第三阶段扩展 Java、Node SDK，并提供接入示例。

第四阶段增加管理台：回调事件查询、死信查看、单事件重放、投递 SLA 报表。

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
  "probe_interval_ms": 10000,
  "retry_schedule_seconds": [10, 30, 60, 180, 300, 900, 1800, 3600],
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
		return MarkFailedPermanent(event.EventID, "missing_idempotency_key")
	}

	MarkEventStatus(event.EventID, "probing")

	resp, err := platformClient.GetOrderByIdempotencyKey(ctx, event.AppID, event.IdempotencyKey)
	if err != nil {
		return handleProbeTransportError(event, err)
	}

	switch resp.Code {
	case "FOUND":
		if resp.ExternalOrderID == nil || *resp.ExternalOrderID == "" {
			return MarkProbeRetry(event.EventID, "found_without_external_order_id")
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
		return MarkProbeRetry(event.EventID, "unknown_probe_code:"+resp.Code)
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
			return nil, err
		}
		return &out, nil

	case http.StatusNotFound:
		return &ProbeOrderResponse{
			Code:           "NOT_FOUND",
			IdempotencyKey: idempotencyKey,
		}, nil

	case http.StatusTooManyRequests:
		return nil, NewRetryableProbeError("rate_limited", retryAfter(res.Header))

	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return nil, NewRetryableProbeError("platform_order_unavailable", nil)

	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, NewPermanentProbeError("probe_unauthorized")

	default:
		return nil, NewRetryableProbeError("unexpected_probe_status", nil)
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

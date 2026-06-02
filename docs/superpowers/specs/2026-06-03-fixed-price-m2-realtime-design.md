# A5 一口价秒杀 M2 实时同步层设计

> Scope: 后端实时事件产生与 WebSocket 广播。  
> Base: M1 已完成一口价上架、抢购、下架、gateway 转发与核心一致性测试。  
> Worktree: `feat/fixed-price-m1`。

## 1. 背景

M1 已完成后端核心交易链路：

- 主播上架一口价商品。
- 用户通过 `X-Idempotency-Key` 抢购。
- Redis Lua 原子扣库存。
- auction 单库事务扣 `user_balance` 并写 `fixed_price_purchases`。
- 失败时 Saga 补偿库存。
- gateway 转发 `/api/v1/fixed-price/*`。

M2 的问题不是交易正确性，而是直播间内用户无法实时感知一口价商品变化。根因是 M1 只返回 HTTP 响应，没有把状态变化推给同直播间的在线连接。

已有约束：

- 复用 B1/现有 `LiveStreamRoom`，不新建 Room。
- 现有 WebSocket Hub 是进程内 `websocket.Hub`，通过 `BroadcastToRoom(roomID, msg)` 广播。
- 现有 `BidService` 已使用 `SetHub` + `hub.BroadcastToRoom` 模式。
- 本轮只做后端实时事件，不做 H5 消费、不做列表接口、不做跨实例 Redis Pub/Sub。

## 2. 目标与非目标

### 2.1 目标

- 新增一口价 WebSocket 消息类型。
- 在 fixed-price 成功路径产生实时事件。
- 对库存变更做 `1s/item` 合并节流。
- 用单元测试覆盖消息契约、广播触发、失败不广播、售罄、下架和节流。
- 保持交易链路 fail-safe：实时广播失败不影响购买成功。

### 2.2 非目标

- 不实现 H5 端消费和 UI 展示。
- 不实现 `GET /live-streams/{id}/fixed-price/items` 列表聚合。
- 不引入 Redis Pub/Sub、多实例广播、MQ 或 Outbox。
- 不为 `fixed_price_flair` 跨服务查询昵称或商品标题。
- 不改变 M1 HTTP API 契约。

## 3. 方案选择

### 3.1 选定方案：同步内存广播

`FixedPriceService` 注入窄接口 `FixedPriceBroadcaster`。业务成功后，service 调用 broadcaster；生产实现内部复用现有 `websocket.Hub.BroadcastToRoom(liveStreamID, message)`。

选择原因：

- 与当前 WebSocket 架构一致。
- 不引入新基础设施。
- 交易链路和实时层边界清晰。
- 可用 fake broadcaster 做 TDD。

### 3.2 暂不采用的方案

- Redis Pub/Sub：能支持多实例，但需要订阅 lifecycle、重连、错误处理和测试，超出 M2 后端实时事件 MVP。
- DB Outbox：可追溯但过重，且 M1 已明确不引入 Outbox。

## 4. 消息契约

所有消息复用现有 envelope：

```json
{
  "type": "fixed_price_stock",
  "timestamp": 1710000000000,
  "data": {}
}
```

### 4.1 `fixed_price_listed`

触发：主播上架成功后。

Payload：

```json
{
  "item_id": 7001,
  "live_stream_id": 1001,
  "product_id": 5001,
  "price": "99.00",
  "total_stock": 100,
  "remaining_stock": 100,
  "status": "on_sale"
}
```

说明：

- `price` 使用字符串，沿用 M1 金额契约，避免 float。
- 不包含 `product_title`/`cover_image`，避免 auction-service 跨域聚合商品展示信息。

### 4.2 `fixed_price_stock`

触发：购买成功后。

Payload：

```json
{
  "item_id": 7001,
  "remaining_stock": 87
}
```

规则：

- 按 `item_id` 做 1 秒合并节流。
- 同一秒内多次购买只广播最后一次剩余库存。
- 节流只影响 `fixed_price_stock`，不影响 `fixed_price_flair` 和 `fixed_price_sold_out`。

### 4.3 `fixed_price_sold_out`

触发：购买成功后发现 `remaining_stock == 0`。

Payload：

```json
{
  "item_id": 7001
}
```

规则：

- 不节流。
- 一次售罄状态只广播一次。
- 若同一购买同时产生库存更新和售罄事件，顺序为 `fixed_price_stock(0)` 后 `fixed_price_sold_out`。

### 4.4 `fixed_price_offline`

触发：主播下架成功后。

Payload：

```json
{
  "item_id": 7001
}
```

规则：

- 不节流。
- 广播到该 item 对应的 `live_stream_id` 房间。

### 4.5 `fixed_price_flair`

触发：每次购买成功后。

Payload：

```json
{
  "item_id": 7001,
  "buyer_id": 42,
  "price": "99.00"
}
```

说明：

- 本轮不包含 `buyer_nickname` 和 `product_title`。
- 原因：当前 fixed-price service 无无跨域的昵称/商品标题来源；为了装饰字段跨服务查询会破坏 M2 的边界。
- 未来 H5 需要展示昵称或标题时，可以由前端用已有用户态/商品缓存补齐，或另开 BFF 聚合任务。

## 5. 后端架构

### 5.1 接口边界

在 `service/fixed_price.go` 引入窄接口：

```go
type FixedPriceBroadcaster interface {
    Listed(ctx context.Context, item *model.FixedPriceItem)
    StockChanged(ctx context.Context, liveStreamID, itemID int64, remaining int)
    SoldOut(ctx context.Context, liveStreamID, itemID int64)
    Offline(ctx context.Context, liveStreamID, itemID int64)
    Flair(ctx context.Context, liveStreamID, itemID, buyerID int64, price decimal.Decimal)
}
```

原则：

- service 只依赖业务语义接口，不依赖 `websocket.Hub`。
- 方法不返回错误；广播是 best-effort side effect。
- 测试使用 fake broadcaster 捕获事件。

### 5.2 生产实现

新增 auction 内适配器，例如 `fixedPriceWSBroadcaster`：

- 持有 `*websocket.Hub`。
- 使用 `websocket.NewFixedPrice...Message` 构造消息。
- 调用 `hub.BroadcastToRoom(liveStreamID, msg)`。
- 内部维护 `fixed_price_stock` 的 1 秒合并节流器。

### 5.3 装配

`main.go`：

- 创建现有 `hub := websocket.NewHub()`。
- 创建 `fixedPriceBroadcaster := newFixedPriceWSBroadcaster(hub)`。
- `NewFixedPriceService(...)` 增加 broadcaster 参数。
- 测试和不需要广播的调用传 `nil`，service 内部使用 no-op broadcaster。

### 5.4 成功路径触发点

`ListItem`：

- DB create 成功。
- Redis stock init 成功。
- 调用 `broadcaster.Listed(ctx, item)`。

`Purchase`：

- 幂等重放命中：不广播。原因是没有新的业务状态变化。
- 新购买成功：
  - 持久化 purchase 成功。
  - 读取 remaining。
  - 调用 `StockChanged`。
  - 调用 `Flair`。
  - 若 `remaining == 0` 且 DB 状态更新为 sold_out 后，调用 `SoldOut`。

`Offline`：

- 校验 owner 成功。
- DB 状态更新为 offline 成功。
- 调用 `Offline`。
- Redis 延迟清理仍按 M1 保持 5 秒。

### 5.5 失败路径

以下路径不得广播：

- 上架参数非法。
- 非主播上架/下架。
- 商品不存在。
- 购买幂等 key 非法。
- 未在售、售罄、重复购买。
- 余额不足导致事务失败。
- Redis down fail-fast。

## 6. 节流设计

### 6.1 Stock 合并节流

目标：高并发秒杀时，不让每次成功购买都立即广播 `fixed_price_stock`。

规则：

- key：`item_id`。
- 窗口：1 秒。
- 行为：窗口内只保留最后一次 `remaining_stock`。
- 到点发送到 `live_stream_id` 房间。
- 如果 `remaining_stock == 0`，允许立即 flush `stock(0)`，然后发送 `sold_out`，保证售罄 UI 及时。

### 6.2 Flair 不节流

`fixed_price_flair` 是单次购买的反馈，M2 不节流。若未来出现刷屏问题，应在前端展示层或独立飘屏聚合器处理，不把展示策略写进交易 service。

## 7. 一致性与错误处理

- 交易成功以 DB/Redis 写入为准，WebSocket 广播只是通知。
- broadcaster 不返回错误，避免实时层失败回滚交易。
- 如果 room 不存在或没有客户端，`BroadcastToRoom` 自然 no-op。
- `fixed_price_stock` 可能因节流延迟最多 1 秒；HTTP 响应仍返回即时 `remaining_stock`。
- `fixed_price_sold_out` 必须在状态成功更新后发送，避免客户端提前进入售罄态但服务端仍允许购买。

## 8. 测试计划

### 8.1 WebSocket 消息契约测试

文件：`backend/auction/websocket/fixed_price_message_test.go`

覆盖：

- 5 种 type 常量正确。
- `price` 序列化为字符串，保留两位小数。
- payload 字段名稳定。

### 8.2 Broadcaster 测试

文件：`backend/auction/websocket/fixed_price_broadcaster_test.go` 或适配器同包测试。

覆盖：

- listed/offline/sold_out/flair 调用正确 room。
- stock 1 秒合并节流。
- stock(0) 可立即 flush 并保证 sold_out 顺序。

### 8.3 Service 广播触发测试

文件：`backend/auction/service/fixed_price_realtime_test.go`

覆盖：

- `ListItem` 成功广播 listed。
- `Purchase` 新购买成功广播 stock + flair。
- `Purchase` 幂等 replay 不广播。
- 最后一件购买广播 stock(0) + sold_out。
- `Offline` 成功广播 offline。
- 失败路径不广播：余额不足、重复购买、非 owner、未在售。

### 8.4 回归

- `cd backend/auction && go test ./service/ ./websocket/ -run 'TestFixedPrice' -race`
- `cd backend/auction && go test ./... -race`

## 9. 风险与后续

| 风险 | 处理 |
| --- | --- |
| 多实例下只广播本机连接 | 本轮接受；未来 M3 可引入 Redis Pub/Sub 作为跨实例事件总线 |
| stock 节流导致 UI 最多晚 1 秒 | 接受；HTTP 响应仍即时，售罄可立即 flush |
| flair 缺少昵称/商品标题 | 本轮不跨域聚合；前端或后续 BFF 补齐 |
| 广播失败不可见 | 可在后续加入 metrics/log，不影响交易 |

## 10. 验收标准

- 一口价上架成功后，同直播间收到 `fixed_price_listed`。
- 一口价购买成功后，同直播间收到 `fixed_price_stock` 和 `fixed_price_flair`。
- 库存归零时，同直播间收到 `fixed_price_stock(0)` 和 `fixed_price_sold_out`。
- 主播下架成功后，同直播间收到 `fixed_price_offline`。
- 幂等重放和失败路径不重复广播。
- `fixed_price_stock` 对同 item 1 秒内合并，只广播最新库存。
- 全量 `go test ./... -race` 通过。

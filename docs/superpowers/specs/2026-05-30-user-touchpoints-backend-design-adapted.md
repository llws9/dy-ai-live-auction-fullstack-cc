# 用户触达体系（一期）·后端设计文档（当前仓库适配版）

**日期**：2026-05-30

**来源文档**：`/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-ui/docs/2026-05-30-user-touchpoints-backend-design.md`

**适用仓库**：`/Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc`

**范围**：为 H5 用户触达一期提供真实后端数据源，包括红点汇总、顶部 Toast 事件、登录后开播弹窗。

---

## 1. 结论

原始设计方向正确，但不能直接按原文实施。当前仓库已有通知表、通知 Service、用户级 WebSocket 房间、订单状态和直播关注关系，应该优先复用现有能力。

本适配版做以下关键调整：

1. 所有前端访问路径统一经过 `gateway-service` 的 `/api/v1` 前缀。
2. 红点汇总采用 `gateway-service` BFF 聚合，避免 `auction-service` 跨服务读取 `product-service` 订单数据。
3. Toast 实时事件一期复用现有 `/api/v1/ws?auction_id=...` 和 `notification` 消息，不新增 `/ws/user-events`。
4. 开播提醒需要新增可靠的“已提醒”落点，不能只依赖前端 `localStorage`。
5. 本期不引入新中间件；RabbitMQ 已有代码可作为增强路径，但不作为一期硬依赖。

---

## 2. 当前仓库事实

### 2.1 前端约束

- H5 API Base 已统一为 `/api/v1`。
- 所有前端 HTTP / WS 流量必须经过 `gateway-service`。
- 当前触达 Mock 数据源在 `frontend/h5/src/hooks/useTouchpointNotifications.ts`。
- 当前登录后弹窗仍由 `pending_live_reminder` 本地标记触发，后端接入后应替换为接口查询；接口失败不得回退展示 Mock 弹窗。
- 当前 WebSocket 连接为 `/api/v1/ws?auction_id={id}&token={token}`。

### 2.2 后端已有能力

- `auction-service` 已有 `notifications` 表与 `NotificationService`。
- `NotificationService` 已支持 `bid_outbid`、`auction_won` 等通知类型。
- `auction-service/websocket.Hub` 已有 `UserRooms`，支持按 `userID` 推送。
- `product-service` 已有 `orders` 表，`OrderStatusPending = 0` 表示待支付。
- `auction-service` 已有 `user_live_stream_follows`，可作为开播提醒候选用户来源。
- `gateway-service` 已代理通知、订单、直播间、WebSocket 路由。

### 2.3 不应照搬的原设计点

| 原设计 | 当前仓库问题 | 适配方案 |
|---|---|---|
| `/api/notifications/summary` | 缺少 `/api/v1`，且绕过 gateway 约束 | 改为 `/api/v1/notifications/summary` |
| `/api/notifications/read` | 当前已有 `PUT /notifications/:id/read` 与 `PUT /notifications/read-all` | 本期新增分类已读接口时使用 `/api/v1/notifications/read-category` |
| `/ws/user-events` | 当前 WS 为 auction room 维度，要求 `auction_id` | 一期复用 `/api/v1/ws?auction_id=...` 的 `notification` 消息 |
| `auction.outbid` / `auction.won` | 当前后端类型是 `bid_outbid` / `auction_won` | 前端映射现有通知类型到 Toast 类型 |
| 登录后直接前端本地弹窗 | 无后端已提醒状态，刷新/多端不可控 | 新增 `GET /api/v1/live/pending-reminder` 并后端标记已提醒 |
| `pendingPayment` 从 notification 服务给出 | 数据在 `product-service.orders` | gateway 聚合或 product 暴露 count |

---

## 3. 目标接口总览

| 能力 | 通道 | 对外路径 | 所属实现 | 用途 |
|---|---|---|---|---|
| 红点汇总 | HTTP GET | `/api/v1/notifications/summary` | `gateway-service` 聚合 | 提供 `unreadTotal`、`pendingPayment` |
| 通知分类已读 | HTTP POST | `/api/v1/notifications/read-category` | `auction-service` | 按触达分类清零 |
| 开播提醒查询 | HTTP GET | `/api/v1/live/pending-reminder` | `gateway-service` + `auction-service` | 登录后查询是否展示弹窗 |
| Toast 实时事件 | WebSocket | `/api/v1/ws?auction_id={id}` | 复用 `auction-service` WS | 推送 `notification` 类型消息 |

---

## 4. 红点汇总

### 4.1 对外接口

`GET /api/v1/notifications/summary`

**鉴权**：需要 Bearer Token。

**Response**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "unreadTotal": 3,
    "pendingPayment": 1,
    "wonNotPaid": 1,
    "outbid": 0,
    "endingSoon": 0
  }
}
```

**字段定义**：

| 字段 | 类型 | 来源 | 含义 |
|---|---:|---|---|
| `unreadTotal` | int | `auction-service.notifications` | 全局未读通知数，用于底部「我的」红点 |
| `pendingPayment` | int | `product-service.orders` | 待支付订单数，用于「我的竞拍」红点 |
| `wonNotPaid` | int | `product-service.orders` | 中标未支付数，本期可等同 `pendingPayment` |
| `outbid` | int | `auction-service.notifications` | 未读 `bid_outbid` 通知数 |
| `endingSoon` | int | 固定值 | 一期不接截拍调度器，必须返回 `0`；不得临时映射到 `auction_starting` |

### 4.2 实现边界

推荐实现为 `gateway-service` BFF 聚合：

1. `gateway-service` 从 JWT 解析 `user_id`。
2. 调用 `auction-service` 获取通知侧计数。
3. 调用 `product-service` 获取订单侧计数。
4. 聚合成前端 `TouchpointNotifications` 所需结构。

不推荐在 `auction-service` 直接查询 `product-service` 数据库，因为当前仓库服务边界已经拆分为 `auction` 与 `product`。

### 4.3 内部接口建议

`auction-service` 新增：

`GET /api/v1/notifications/summary`

返回通知侧计数：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "unreadTotal": 2,
    "outbid": 1,
    "endingSoon": 0
  }
}
```

`product-service` 新增：

`GET /api/v1/orders/summary`

返回订单侧计数：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "pendingPayment": 1,
    "wonNotPaid": 1
  }
}
```

### 4.4 gateway 错误策略

`gateway-service` 是前端唯一入口，错误处理必须区分“身份错误”和“业务计数降级”：

- gateway JWT 校验失败：直接返回 `401`，不请求上游服务。
- 上游返回 `401/403`：视为身份链路错误，透传为 `401/403`，不能降级成 0，避免掩盖鉴权配置问题。
- 上游超时、5xx、非 JSON 或 `code != 0 && code != 200`：该上游负责的字段降级为 0，整体仍返回 `200`，保证红点失败不阻塞 H5 页面。
- `auction-service` 失败只影响 `unreadTotal/outbid/endingSoon`；`product-service` 失败只影响 `pendingPayment/wonNotPaid`。
- 一期不新增 `partial` 字段；服务端记录日志即可，避免前端契约扩大。

---

## 5. 通知分类已读

### 5.1 对外接口

`POST /api/v1/notifications/read-category`

**鉴权**：需要 Bearer Token。

**Request**：

```json
{
  "category": "pendingPayment"
}
```

**可选值**：

| category | 后端行为 |
|---|---|
| `outbid` | 标记当前用户未读 `bid_outbid` 通知为已读 |
| `endingSoon` | 一期 no-op 返回成功；等截拍调度器落地后再绑定真实通知类型 |
| `all` | 复用当前 `read-all` 语义 |
| `pendingPayment` | 不建议直接标记订单为已读；本期可 no-op 返回成功 |

**Response**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true
  }
}
```

### 5.2 实现说明

当前 `NotificationDAO` 已有 `MarkAllAsRead` 和 `MarkAsRead`，但没有按 `type` 标记已读。需要新增：

- `NotificationDAO.MarkByTypeAsRead(ctx, userID int64, types []model.NotificationType) error`
- `NotificationService.MarkCategoryAsRead(ctx, userID int64, category string) error`
- `NotificationHandler.MarkCategoryAsRead`
- `gateway-service` 路由代理

---

## 6. Toast 实时事件

### 6.1 一期策略

不新增 `/ws/user-events`。一期复用当前：

`/api/v1/ws?auction_id={auctionId}&token={token}`

原因：

1. 前端 `WebSocketService` 已稳定连接该路径。
2. gateway 已代理 `/api/v1/ws`。
3. `auction-service` 已将通知推送到 `UserRooms`。
4. 新增全局用户事件流会扩大改造面，不符合一期最小化原则。

### 6.2 服务端消息格式

复用现有 `notification` 消息：

```json
{
  "type": "notification",
  "timestamp": 1717000000000,
  "data": {
    "id": 1001,
    "type": "bid_outbid",
    "title": "出价被超越",
    "content": "您在竞拍中的出价 8800.00 元已被超越，当前最高价为 9200.00 元",
    "data": {
      "auction_id": 123,
      "old_bid": 8800,
      "new_bid": 9200
    },
    "created_at": "2026-05-30T12:00:00Z"
  }
}
```

### 6.3 前端映射建议

| 后端 `data.type` | Toast type | title | action |
|---|---|---|---|
| `auction_starting` | `warning` | 截拍预警 | 进入直播间 |
| `bid_outbid` | `danger` | 您已被超价 | 重新出价 |
| `auction_won` | `success` | 恭喜中标 | 去支付 |

### 6.4 幂等与去重

原设计使用 `eventId`。当前仓库可以直接使用通知表主键 `notification.id` 作为去重 key。

前端规则：

- 同一个 `notification.id` 只显示一次 Toast。
- 如果 WS 断开，重连后由 `GET /api/v1/notifications/summary` 补齐红点数字。

---

## 7. 开播提醒

### 7.1 对外接口

`GET /api/v1/live/pending-reminder`

**鉴权**：需要 Bearer Token。

**Response：有提醒**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "hasReminder": true,
    "stream": {
      "id": 789,
      "name": "XX 珠宝",
      "avatarUrl": "https://example.com/avatar.png",
      "statusText": "正在直播",
      "liveRoomId": 789,
      "startedAt": 1716999000000
    }
  }
}
```

**Response：无提醒**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "hasReminder": false,
    "stream": null
  }
}
```

### 7.2 必须新增的状态模型

当前 `product-service.live_streams` 只有 `Disabled/Active`，无法表达“正在直播且尚未提醒”。当前 `auction-service.user_live_stream_follows` 只有关注关系和通知开关，也无法记录“某次开播已提醒”。

本期建议新增轻量表：

```sql
CREATE TABLE IF NOT EXISTS live_stream_reminder_receipts (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  live_stream_id BIGINT NOT NULL,
  live_started_at BIGINT NOT NULL,
  reminded_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_user_stream_started (user_id, live_stream_id, live_started_at),
  KEY idx_user_id (user_id)
);
```

### 7.3 查询语义

1. 查询当前用户已关注的一批直播间候选，只保留 `notification_enabled = true` 的候选。
2. 按确定性顺序检查候选是否“正在直播”，并要求存在真实直播 session `started_at`。
3. 使用真实直播 session 的 `started_at` 作为 `live_started_at`；该值必须来自直播状态模型或开播事件，不能用请求时间、小时取整或登录时间合成。
4. 对每个有效候选原子写入 `live_stream_reminder_receipts`，唯一键为 `user_id + live_stream_id + live_started_at`。
5. 只有本次写入成功时返回提醒；唯一键冲突表示该候选 session 已消费，应继续检查下一个候选，而不是直接返回无提醒。
6. 只有全部候选都未开播、无真实 `started_at`、通知关闭或已被消费时，才返回 `hasReminder=false`。
7. 返回第一个可成功 claim 的候选，排序策略必须稳定，避免同一次请求内随机弹不同直播间。

### 7.4 开播状态来源

当前仓库没有真正的“直播开播状态变更事件”。一期有两种可选实现：

| 方案 | 说明 | 推荐 |
|---|---|---|
| A | 使用 `product-service.live_streams.status = Active` 且已有真实 `started_at` | 可作为 MVP |
| B | 为直播间补 `live_status`、`started_at` 字段 | 推荐，符合真实业务 |

本期若没有真实 `started_at`，不要用 `time.Now()` 或小时桶伪造开播 session；接口应返回 `hasReminder=false`，或先补齐 `started_at` 后再接入弹窗。

---

## 8. 安全与错误处理

### 8.1 HTTP

- 所有新增对外接口必须挂在 `gateway-service` 的 JWT 认证组。
- HTTP 未认证返回标准 `401`。
- 响应结构优先使用当前仓库兼容格式：`{ "code": 0 | 200, "message": "success", "data": ... }`。

### 8.2 WebSocket

当前后端在 upgrade 前鉴权失败会返回 HTTP `401`，前端已有 `4401` close code 处理。两者不完全一致。

一期建议：

1. 保持现状，避免扩大 WS 改造面。
2. 若未来要严格满足前端 `4401`，需要在 upgrade 后校验 token 并主动 close `4401`。
3. 前端仍应兼容 HTTP 401 连接失败和 close code 4401 两种情况。

---

## 9. 数据来源

| 字段/事件 | 当前来源 | 说明 |
|---|---|---|
| `unreadTotal` | `auction.notifications WHERE read_at IS NULL` | 已有 DAO 可扩展 |
| `outbid` | `auction.notifications.type = bid_outbid` | 需要按 type count |
| `endingSoon` | 固定值 | 本期必须返回 0，等截拍调度器落地后再接真实来源 |
| `pendingPayment` | `product.orders.status = 0` | 需要 product count 接口 |
| `wonNotPaid` | `product.orders.status = 0` | 本期可等同 `pendingPayment` |
| Toast outbid | `NotificationTypeBidOutbid` | 已有发送方法 |
| Toast won | `NotificationTypeAuctionWon` | 已有发送方法 |
| 开播提醒候选用户 | `user_live_stream_follows` | 已有模型 |

---

## 10. 实施顺序

### M1：红点真实数据

1. `auction-service` 增加通知 summary service/handler/dao。
2. `product-service` 增加 order summary service/handler/dao。
3. `gateway-service` 增加 `/api/v1/notifications/summary` 聚合接口。
4. 前端 `useTouchpointNotifications` 从 Mock 切换为 API。

验收：

- 底部「我的」显示真实 `unreadTotal`。
- 个人中心「我的竞拍」显示真实 `pendingPayment`。
- API 失败时前端按 0 渲染，不阻塞页面。

### M2：开播提醒真实数据

1. 新增 `live_stream_reminder_receipts`。
2. 新增 `/api/v1/live/pending-reminder`。
3. 前端 `MobileContainer` 从本地 `pending_live_reminder` 切换为接口查询。
4. 登录后只查询一次，刷新不重复弹。

验收：

- 有任一已关注且开启通知的直播间开播时，登录后弹一次；首个候选已消费时继续检查后续候选。
- 接口第二次调用返回 `hasReminder=false`。
- 用户关闭弹窗后不影响后续新一期开播提醒。

### M3：Toast 真实事件

1. 复用现有 `notification` WS 消息。
2. 前端监听 `notification`，按 `data.type` 映射 Toast。
3. 使用 `notification.id` 去重。
4. 移除或隐藏 `/live` 开发环境 Toast Demo 触发器。

验收：

- 被超价触发 `danger` Toast。
- 中标触发 `success` Toast。
- 重复通知 ID 不重复显示。
- WS 断开后红点仍可由 summary 补齐。

---

## 11. 需要调整的原始设计

原始文档中的以下内容应废弃或延后：

1. 废弃 `/api/notifications/summary`，统一为 `/api/v1/notifications/summary`。
2. 废弃 `/api/notifications/read`，改为 `/api/v1/notifications/read-category` 或复用现有 read-all。
3. 延后 `/ws/user-events`，一期复用 `/api/v1/ws?auction_id=...`。
4. 延后 `auction.endingSoon` / `auction.outbid` / `auction.won` 新事件名，先复用 `notification` 包装和现有 `NotificationType`。
5. 不直接在 `auction-service` 查询订单数据，订单计数归 `product-service` 或 `gateway-service` 聚合。

---

## 12. 验收标准

- [ ] `GET /api/v1/notifications/summary` 返回真实红点数据。
- [ ] `summary` 接口同时包含通知侧和订单侧字段。
- [ ] `GET /api/v1/live/pending-reminder` 能返回并消费一次性开播提醒。
- [ ] `notification` WS 消息能触发前端 Toast。
- [ ] 同一 `notification.id` 前端只展示一次 Toast。
- [ ] HTTP 鉴权失败返回 401。
- [ ] WS 鉴权失败不进入正常消息流。
- [ ] 所有新增接口均通过 `gateway-service` 暴露。

---

## 13. 测试建议

后端：

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction
go test ./...
```

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product
go test ./...
```

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway
go test ./...
```

前端联调：

```bash
cd /Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5
npm test -- --runInBand src/components/BadgeDot/__tests__/BadgeDot.test.tsx src/__tests__/components/MobileShell.test.tsx src/pages/User/__tests__/Profile.test.tsx src/components/Toast/__tests__/ToastProvider.test.tsx
npm run build
```

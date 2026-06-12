---
name: knowledge-backend-auction
description: >
  Covers auction-service 的竞拍核心、点天灯、出价引擎、通知系统、WebSocket 实时同步和商品提醒热拉。
  Navigate when: modifying backend/auction handlers, bidding logic, notification system, or hot-pull features.
  Excludes: product-service, gateway-service, test-service.
  Keywords: backend/auction, auction-service, bid, sky-lamp, notification, hot-pull, websocket, reminder
---

## Module Structure

auction-service 是竞拍域核心服务，负责出价引擎、点天灯、通知系统（热拉/冷推）、WebSocket 实时同步和直播间状态管理；核心风险集中在并发控制、幂等性、事务边界和状态一致性。

### Directory Layout
- `backend/auction/handler/` — HTTP/WebSocket 处理器，包括竞拍、出价、通知接口。
- `backend/auction/service/` — 业务逻辑层，包括出价引擎、点天灯、通知热拉。
- `backend/auction/dao/` — 数据访问层，包括竞拍、出价、通知、提醒回执。
- `backend/auction/ws/` — WebSocket Hub 和房间管理。
- `backend/auction/model/` — 数据模型定义。

### Key Entry Points
- `backend/auction/service/bid.go` — 出价引擎核心逻辑。
- `backend/auction/service/sky_lamp.go` — 点天灯业务逻辑。
- `backend/auction/service/notification.go` — 通知热拉与冷推。
- `backend/auction/dao/user_product_reminder.go` — 商品提醒数据访问。

## Gotchas

### 点天灯缺失竞拍规则保护 (Sky Lamp Missing Rule Guard)

**问题背景**：点天灯功能因缺失 `auction_rules` 记录导致 500 错误，根因为脏数据未做判空保护。

**修复方案**：
1. **数据修复**：补全缺失的 `auction_rules` 记录
2. **代码保护**：`SkyLampService` 和 `BidService` 对 `rule == nil` 返回明确业务错误（如"竞拍规则不存在"），避免 panic/500
3. **数据源修复**：修复 seed/demo 生成逻辑，确保新增直播竞拍商品同步生成规则

**关键代码模式**：
```go
// 判空保护示例
if rule == nil {
    return nil, errors.New("竞拍规则不存在")
}
```

**来源**：session:6a21af602ec60aa1a739c0d9

### 批量通知创建的事务边界问题

**问题背景**：`persistProductReminderNotifications` 在 for 循环中对每个 candidate 调用 `ClaimAndCreateAuctionStartNotification`，每个调用开启独立事务，导致部分成功问题。

**风险**：
- 第 N 个 candidate 失败时，前 N-1 个通知已提交
- API 返回错误，但副作用已发生
- 用户重试可能导致重复通知

**修复方案**：
- 单个 candidate 失败只记录日志并继续处理剩余候选
- 不中断整个热拉流程
- 幂等回执表保证重复热拉不会重复生成通知

**关键代码模式**：
```go
for _, candidate := range candidates {
    notification, err := s.claimAndCreate(ctx, candidate)
    if err != nil {
        log.Printf("[WARN] create notification failed for auction=%d: %v", candidate.AuctionID, err)
        continue // 继续处理下一个，不中断
    }
    created = append(created, notification)
}
```

**来源**：session:6a21af602ec60aa1a739c0d9

### 竞拍结束结算链路一致性 (Auction End Settlement Consistency)

**问题背景**：竞拍结束后出现三个关联症状：1) 中标人显示缺失；2) 用户未收到中标通知；3) 竞拍记录中无该中标订单。

**根因分析**：
三个症状可被统一根因解释——竞拍结束时的结算任务状态机执行失败：

| 症状 | 根因解释 |
|------|----------|
| 中标人缺失 | `EndAuction` 未固化 `winner_id` 到 auctions 表 |
| 没收到通知 | 结算卡在订单创建阶段，未走到 `notifying` 状态 |
| 竞拍记录漏单 | 订单创建失败，orders 表无记录 |

**结算链路完整流程**：
```
EndAuction → CreatePendingTaskWithTx → FinalizeEndedAuction
    → 创建订单 (order_done) → 发通知 (notifying) → 完成 (done)
```

**关键修复点**：
1. **winner_id 固化**：`EndAuction` 确定中标者后立即更新 `winner_id` 和 `final_price`
2. **订单创建前置**：通知发送必须在订单创建成功后进行，避免"有通知没订单"
3. **结算服务注入**：确保 `settlementService` 的 `orderCreator` 和 `notificationSender` 在 main.go 中正确注入

**注入顺序陷阱**：
```go
// 正确顺序：先 SetSettlementService 再 SetNotificationSender
// 两者必须操作同一对象实例
settlementService := service.NewSettlementService(...)
auctionService.SetSettlementService(settlementService)
settlementService.SetNotificationSender(notificationSvc) // 设到同一对象
```

**测试覆盖要点**：
- `EndAuction` 成功后 `winner_id` 已固化
- 结算任务完成时订单已创建
- 结算任务完成时中标通知已发送
- 订单创建失败时不应发送通知

**来源**：session:6a24716000057ea64ca294db

### 死代码识别与清理

**问题背景**：代码审查发现 `ClaimAuctionStartReminder` 方法全仓只有定义无调用，属于死代码。

**识别方法**：
```bash
# 搜索方法定义
grep -r "ClaimAuctionStartReminder" backend/auction/

# 确认无调用点（除定义外无其他匹配）
```

**处理原则**：
- 确认无调用后应删除，减少维护负担
- 不删除可能导致运行时问题的代码，只删除确认安全的死代码

**来源**：session:6a21af602ec60aa1a739c0d9

### 测试期望与实现语义不一致的修复 (Test Expectation Drift Fix)

**问题背景**：`HotPull` handler 测试 `TestHotPullProductReminder_ErrorResponseDoesNotLeak` 期望 Redis 不可用时返回 500，但当前实现已改为降级返回 200 空列表，导致测试失败。

**根因分析**：
- 提交 `da13f320` 已将 `HotPull` 语义改为"Redis 不可用时返回商品提醒/空列表"
- 但 handler 层的旧安全测试未同步更新期望
- 这不是实现回归，而是测试期望落后于服务语义

**修复方案**：
更新测试期望为当前语义：Redis/DB live source 都不可用时，接口返回 `200 success`、空通知列表，并验证日志记录了降级路径。

**关键断言保留**：
- 响应不能泄露内部 Redis 初始化细节（错误信息脱敏）
- 降级行为可观测（日志记录）

**修复后验证**：
```bash
go test ./handler -count=1
go test ./service ./client ./handler -count=1
```

**来源**：session:6a23e4a22ec60aa1a73a5f31

### 商品开拍提醒热拉入库设计

**功能概述**：用户订阅的商品竞拍即将开始时，通过热拉机制生成未读通知并驱动铃铛红点，不弹窗。

**设计决策**：
1. **只做热拉，不做冷推**：满足"重新登录/回到前台驱动红点"的需求，成本最低
2. **不新增弹窗**：避免与直播间开播弹窗抢优先级，保持单一弹窗通道
3. **30分钟阈值**：竞拍开始前 30 分钟视为"即将开始"
4. **幂等回执表**：`product_reminder_receipts` 按 `user_id + auction_id` 唯一，保证同一用户同一竞拍只生成一次通知

**数据流**：
```
用户登录/回前台 → POST /notifications/hot-pull
                         ↓
              查询 user_product_reminders
                         ↓
              筛选 30 分钟内开始的订阅
                         ↓
              幂等 claim 回执表
                         ↓
              创建 auction_starting 未读通知
                         ↓
              返回通知列表 + 刷新红点
```

**关键接口**：
- `POST /notifications/hot-pull` — 热拉通知入口
- 扩展点：商品提醒持久化逻辑

**来源**：session:6a21af602ec60aa1a739c0d9

### 内部用户批量查询接口 (Internal User Batch Query API)

**功能概述**：为 `product-service` 提供内部 API，批量查询用户摘要信息（用户名、头像），用于 Admin 订单列表的买家信息回填。

**接口契约**：
- `POST /internal/users/batch`
- 请求：`{ user_ids: number[] }`
- 响应：`map[string]{ username, avatar }` — key 为 user_id 的字符串形式

**实现要点**：
```go
func (h *InternalHandler) BatchGetUsers(c *gin.Context) {
    var req BatchGetUsersReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse{Code: "INVALID_PARAM"})
        return
    }

    // 批量查询用户信息
    users, err := h.userRepo.BatchGetByIDs(ctx, req.UserIDs)
    if err != nil {
        c.JSON(500, ErrorResponse{Code: "INTERNAL_ERROR"})
        return
    }

    // 组装为 map 响应
    result := make(map[string]UserSummary)
    for _, user := range users {
        result[strconv.FormatInt(user.ID, 10)] = UserSummary{
            Username: user.Username,
            Avatar:   user.Avatar,
        }
    }
    c.JSON(200, result)
}
```

**调用方 (`product-service`) 使用模式**：
```go
// 从订单列表提取 winner IDs
winnerIDs := extractWinnerIDs(orders)

// 批量获取用户摘要
summaryMap, err := auctionClient.BatchGetUserSummaries(ctx, winnerIDs)
if err != nil {
    // 降级：记录 WARN，继续返回不含买家信息的订单
    log.Printf("[WARN] batch get user summary failed: %v", err)
    summaryMap = make(map[int64]*UserSummary)
}

// 回填买家信息
for _, order := range orders {
    if summary, ok := summaryMap[order.WinnerID]; ok {
        order.BuyerUsername = summary.Username
        order.BuyerAvatar = summary.Avatar
    }
}
```

**HTTP Client 连接复用**：
```go
// 必须 drain 响应体以保证连接复用
resp, err := httpClient.Do(req)
if err != nil {
    return nil, err
}
defer func() {
    io.Copy(io.Discard, resp.Body)
    resp.Body.Close()
}()
```

**安全约束**：
- 该接口为内部服务间调用，不对外暴露
- Gateway 不应路由 `/internal/*` 路径到外部
- 调用方需进行服务间鉴权（如 `INTERNAL_API_TOKEN`）

**测试覆盖**：
- 正常批量查询路径
- 空 ID 列表处理
- 部分 ID 不存在时的响应
- HTTP Client 错误处理

**来源**：session:6a2419153eefb8c530aa7658

## Architecture

### 通知系统架构
```
H5 前端 → Gateway → auction-service (通知汇总)
              ↓
       user_product_reminders (订阅源)
              ↓
       product_reminder_receipts (幂等回执)
              ↓
       notifications (未读通知)
```

### 热拉 vs 冷推
| 维度 | 热拉 (Hot Pull) | 冷推 (Cold Push) |
|------|-----------------|------------------|
| 触发时机 | 用户登录、回前台、刷新 | 后端定时任务/MQ |
| 实时性 | 依赖用户行为 | 不依赖用户在线 |
| 复杂度 | 低 | 高（需定时器/MQ） |
| 适用场景 | MVP 阶段 | 高实时性要求 |

## Patterns

### 幂等通知创建模式
```go
// 1. 尝试 claim 回执（INSERT IGNORE 语义）
claimed, err := dao.ClaimReminderReceipt(ctx, userID, auctionID)
if err != nil {
    return err
}
if !claimed {
    return nil // 已处理过，幂等跳过
}

// 2. 创建通知
notification, err := dao.CreateNotification(ctx, userID, auction)
if err != nil {
    // 回执已写入但通知失败，需补偿或记录
    return err
}
return notification
```

### 软依赖降级模式（列表接口）
```go
// 非核心元数据查询失败时降级为默认值
summaryMap, err := liveStreamClient.BatchGetSummary(ctx, streamIDs)
if err != nil {
    log.Printf("[WARN] batch get live stream summary failed: %v", err)
    summaryMap = make(map[int64]*LiveStreamSummary) // 降级为空 map
}
// 组装时使用零值
```

### 直播间 Presence 实时在线系统 (Live Presence System)

**功能概述**：直播间 WebSocket 连接建立后，实时维护在线用户列表（头像、用户名），并向房间内所有客户端广播 presence 更新。

**核心组件**：
- `LiveStreamRoom` — 房间管理，维护 `clients` 和 `presenceByUserID`
- `Client` — WebSocket 连接封装，含 `Authenticated` 状态标记
- `PresenceSnapshot` — 在线用户快照（人数 + 用户列表）

**消息契约**：
- `live_presence_update` — 在线状态广播，包含 `viewer_count` 和 `viewers` 列表
```json
{
  "type": "live_presence_update",
  "payload": {
    "viewer_count": 42,
    "viewers": [
      {"user_id": 123, "name": "用户A", "avatar_url": "..."}
    ]
  }
}
```

**鉴权边界与隐私保护**：
1. **未鉴权连接处理**：历史兼容场景允许通过 `user_id` query 参数建立 WS 连接（不验证 JWT），这类连接 `Authenticated=false`，不进入 `presenceByUserID` 统计
2. **实名数据广播限制**：`broadcastPresenceSnapshot` 必须只向 `Authenticated=true` 的客户端发送包含实名信息（`user_id`, `name`, `avatar_url`）的完整 snapshot
3. **未鉴权客户端降级**：未鉴权连接只接收匿名人数（或不接收 presence 消息），防止泄露其他用户实名信息

**关键代码模式**：
```go
// 注册客户端时区分鉴权状态
func (r *LiveStreamRoom) addPresenceClient(c *Client) {
    if !c.Authenticated {
        // 未鉴权连接不参与 presence 统计
        return
    }
    // 按 user_id 去重，同一用户多连接只算一个
    r.presenceByUserID[c.UserID] = c
}

// 广播时过滤未鉴权客户端
func (r *LiveStreamRoom) broadcastPresenceSnapshot() {
    snapshot := r.buildPresenceSnapshot()
    for client := range r.clients {
        if !client.Authenticated {
            // 不向未鉴权连接发送实名 presence
            continue
        }
        client.Send(snapshot)
    }
}
```

**测试覆盖要点**：
- 未鉴权 client 不进入 `presenceByUserID` 统计
- 实名用户上线时，未鉴权 client 不收到 `live_presence_update`
- 同一用户多连接按 user_id 去重
- 最后一个连接断开时清理 presence 记录

**来源**：session:6a26bb690bfcee1b04fb3791

### 防狙击延时链路实时可见性改造 (Anti-Snipe Delay Visibility)

**问题背景**：防狙击延时触发后，后端已落库但 H5 端无法实时感知，导致倒计时显示不正确；同时 `time_sync` 周期校时会覆盖 `Delayed` 状态。

**解决方案**：
1. **延时判定移入行锁内**：将 `ShouldTriggerDelay` / `CanDelay` 判定从事务外移入 `FOR UPDATE` 行锁内，避免并发旧快照重复延时
2. **实时广播机制**：延时成功后通过 `TryBroadcastToRoom` 广播 `delay_triggered` 事件，H5 即时更新倒计时并展示 Toast
3. **周期校时兼容**：`time_sync` 携带 `auction_id`，覆盖 `Ongoing` 和 `Delayed` 两种状态的竞拍
4. **H5 跨房防污染**：复用 `belongsToThisRoom` 校验，确保消息只影响当前直播间

**关键代码模式**：
```go
// 事务内行锁 + 重新判定
func (s *BidService) tryExtendAuction(ctx context.Context, auctionID int64) (*DelayResult, error) {
    return s.txMgr.WithTx(ctx, func(tx *gorm.DB) (*DelayResult, error) {
        // FOR UPDATE 读取最新状态
        auction, err := s.auctionRepo.GetByIDWithLock(ctx, tx, auctionID)
        if err != nil {
            return nil, err
        }
        // 重新判定（基于最新快照）
        if !canDelay(auction) {
            return nil, nil // 已被其他请求延时，跳过
        }
        // 执行延时
        newEndTime := auction.EndTime.Add(delayDuration)
        if err := s.auctionRepo.UpdateEndTimeAndStatus(ctx, tx, auctionID, newEndTime, StatusDelayed); err != nil {
            return nil, err
        }
        return &DelayResult{ActualDelay: delayDuration, NewEndTime: newEndTime}, nil
    })
}

// 非阻塞广播（fail-soft）
func (s *BidService) broadcastDelayTriggered(auctionID int64, newEndTime time.Time) {
    s.hub.TryBroadcastToRoom(auctionID, &WSMessage{
        Type: "delay_triggered",
        Payload: map[string]interface{}{
            "auction_id":   auctionID,
            "new_end_time": newEndTime.UnixMilli(),
        },
    })
}
```

**H5 端处理要点**：
```typescript
// 时间格式兼容处理
toEndTimeIso(input: number | string | undefined): string | undefined {
    if (input === undefined || input === null) return undefined;
    if (typeof input === 'number') return new Date(input).toISOString();
    if (/^\d+$/.test(input)) return new Date(Number(input)).toISOString();
    if (typeof input === 'string' && !isNaN(Date.parse(input))) return input;
    return undefined;
}

// 归属校验防跨房污染
const belongsToThisRoom = (msg: WSMessage, currentAuctionId: number) => {
    return msg.payload?.auction_id === currentAuctionId;
};
```

**边界修复**：
- `sync_response` 条件从 `if (state)` 改为 `if (state !== undefined)`，避免 `0`/`false` 等 falsy 值被误判为缺失

**来源**：session:6a24410400057ea64ca26a40

### 看直播领宝箱系统 (Watch Treasure Chest System)

**功能概述**：用户观看直播累计时长（跨直播间、每日重置）可领取宝箱获得金币，金币作为独立虚拟资产落库。

**核心设计决策**：
1. **后端可信计时**：所有计时逻辑必须在服务端完成，前端不可信。前端每 30s 上报一次活跃心跳，后端按「真实间隔」累加，单帧封顶 30s，丢弃异常大跳变
2. **金币与现金隔离**：金币是独立虚拟资产，与 `user_balances` 现金余额完全隔离，避免资产混淆和审计困难
3. **幂等领取**：同一用户同一宝箱只能领取一次，使用数据库唯一键 `(user_id, stat_date, tier)` 保证幂等
4. **每日分桶重置**：观看进度和领取状态按自然日分桶，跨日自动重置

**数据模型**：
```sql
-- 金币资产（1 用户 1 行，永久累积）
CREATE TABLE user_coins (
  user_id     BIGINT PRIMARY KEY,
  balance     BIGINT NOT NULL DEFAULT 0,   -- 整数，无小数
  updated_at  DATETIME NOT NULL
);

-- 今日观看时长（按天分桶，每日 0 点天然失效）
CREATE TABLE user_watch_duration (
  user_id        BIGINT NOT NULL,
  stat_date      DATE   NOT NULL,           -- YYYY-MM-DD
  total_seconds  INT    NOT NULL DEFAULT 0,
  updated_at     DATETIME NOT NULL,
  PRIMARY KEY (user_id, stat_date)
);

-- 宝箱领取记录（幂等 + 防重复领取的 SSOT）
CREATE TABLE treasure_claims (
  user_id    BIGINT NOT NULL,
  stat_date  DATE   NOT NULL,
  tier       TINYINT NOT NULL,             -- 0=3min,1=10min,2=30min
  coins      BIGINT NOT NULL,              -- 当次发放额，留存审计
  claimed_at DATETIME NOT NULL,
  PRIMARY KEY (user_id, stat_date, tier)   -- 唯一键即幂等保证
);
```

**接口契约**（Gateway `/api/v1` 入口）：
| 接口 | 用途 | 防刷点 |
|---|---|---|
| `POST /watch/heartbeat` | 前端每 30s 上报一次，后端累加今日时长 | 服务端按「真实间隔」累加，单次最多记 30s，丢弃异常大跳变 |
| `GET /treasure/status` | 返回今日时长 + 3 宝箱状态（locked/unlockable/claimed）+ 金币余额 | 状态完全由后端时长算出 |
| `POST /treasure/claim` `{tier}` | 领取某宝箱，发币 | 校验 `total_seconds≥门槛` + 唯一键幂等，失败关闭 |

**金币档位**：3min→100币、10min→300币、30min→800币（后端可配置常量）

**前端组件**：`TreasureProgressBar` 挂载于直播间 `hostPill` 下方，展示进度条和 3 个宝箱节点，支持 unlockable 状态的跳动+微开盖+高光动画吸引点击

**来源**：session:6a26f1640bfcee1b04fb4f52

---

### 直播间维度竞拍活跃唯一性约束演进 (Live Stream Auction Uniqueness Evolution)

**原始设计**：每个直播间只能有一个活跃竞拍（`Pending/Ongoing/Delayed` 合计最多 1 个）

**演进后设计**：允许同一直播间同时存在
- 最多 1 个 `Pending`（即将开始/待开始）
- 最多 1 个 `Ongoing/Delayed`（进行中/延时中）

**数据库约束变更**：
```sql
-- 删除旧的统一活跃约束
DROP INDEX uk_active_live_stream;
DROP COLUMN active_live_stream_key;

-- 新增分离约束：Pending 唯一 + Running 唯一
ALTER TABLE auctions ADD COLUMN pending_live_stream_key VARCHAR(255) AS (
    CASE WHEN status = 0 THEN CAST(live_stream_id AS CHAR) ELSE NULL END
) STORED;
ALTER TABLE auctions ADD COLUMN running_live_stream_key VARCHAR(255) AS (
    CASE WHEN status IN (1, 2) THEN CAST(live_stream_id AS CHAR) ELSE NULL END
) STORED;

CREATE UNIQUE INDEX uk_pending_live_stream ON auctions(pending_live_stream_key);
CREATE UNIQUE INDEX uk_running_live_stream ON auctions(running_live_stream_key);
```

**业务语义变更**：
1. **创建 `Pending` 竞拍**：只拒绝同直播间已有 `Pending`，不拒绝已有 `Ongoing/Delayed`
2. **启动竞拍（`Pending -> Ongoing`）**：需检查同直播间是否已有 `Ongoing/Delayed`，有则拒绝或延后
3. **调度策略**：`Pending` 到点时若当前竞拍未结束，保持 `Pending` 不自动启动（最小语义）

**关键代码模式**：
```go
// CreateAuction 校验逻辑
if existingPending, _ := auctionDAO.GetPendingByLiveStreamID(liveStreamID); existingPending != nil {
    return ErrPendingAuctionExists  // 已有待开始竞拍
}
// 不再检查 Ongoing/Delayed，允许创建下一场预告

// StartAuction 校验逻辑
if existingRunning, _ := auctionDAO.GetRunningByLiveStreamID(liveStreamID); existingRunning != nil {
    return ErrRunningAuctionExists  // 已有进行中竞拍，不能启动
}
```

**DDL 执行顺序安全原则**：
1. 先添加新 generated columns + 两个新 unique indexes
2. 全部成功后再 drop 旧索引/旧列
3. 失败时保留旧约束（更严格），不会裸奔

**Down Migration 风险**：
- 新版本允许 `Pending + Ongoing` 同时存在
- Down migration 重新创建 `uk_active_live_stream` 会因唯一键冲突失败
- 需先终止/取消同直播间多 active 数据，或提供 rollback cleanup 脚本

**来源**：session:6a25c4110bfcee1b04fb1b82, session:6a24571300057ea64ca27a83

### 商品-竞拍生命周期设计模式 (Product-Auction Lifecycle Design)

**核心设计决策**：商品可重复拍卖，但竞拍场次不可复用；系统只保证同一商品最多一个活跃竞拍。

**状态定义**：
- 商品状态（经营维度）：`草稿(Draft) / 可排期(Published) / 下架(Offline)`
- 竞拍活跃状态：`Pending(待开始) / Ongoing(进行中) / Delayed(延时中)`
- 竞拍终态：`EndedSold(已成交) / EndedUnsold(流拍) / Cancelled(已取消)`

**活跃唯一性约束实现**：
```sql
-- 生成列：活跃竞拍记录 product_id，终态为 NULL
ALTER TABLE auctions ADD COLUMN active_product_key VARCHAR(255) AS (
    CASE 
        WHEN status IN (0, 1, 3) THEN CAST(product_id AS CHAR)  -- Pending/Ongoing/Delayed
        ELSE NULL  -- 终态释放唯一槽位
    END
) STORED;

-- 唯一索引保证同一商品只有一个活跃竞拍
CREATE UNIQUE INDEX uk_active_product ON auctions(active_product_key);
```

**关键业务规则**：
1. 创建竞拍时检查：商品状态=`Published` + 归属当前商家 + 无活跃竞拍 + 最近终态非成交
2. 流拍商品可重新上架并创建新竞拍（新 auction 记录，非复用旧记录）
3. 已成交商品默认不可再次创建竞拍（除非业务上允许二次拍卖）

**跨服务边界处理**：
- 直播间归属 `product-service`，`auction-service` 不可直写
- 创建竞拍时通过内部 API 获取/创建活跃直播间，再在本地事务创建 auction

**来源**：session:6a24571300057ea64ca27a83

### 一口价商品选择设计模式 (Fixed Price Product Selection Design)

**问题背景**：管理端一口价上下架原使用手填商品 ID，用户体验差且易出错。

**核心决策**：
1. **一口价卖的是和竞拍不同的另一件商品**（搭售），保留"选商品"下拉
2. **商品来自同一货品库**，不新增分类/商品类型（category 是品类，与销售方式正交）
3. **下拉只显示自有 + 已发布 + 未在竞拍**的商品（`display_status=schedulable`）
4. **后端兜底失败关闭**：上架商品若正处于 active 竞拍，拒绝

**前端改动**：
- 删除"商品 ID"`<input type="number">`，替换为商品下拉
- 调 `productApi.list({ display_status: 'schedulable' })` 拉取可售商品
- 选项展示 `商品名（#id）`，提交时取选中项的 `product_id`

**后端改动**：
- `ListItem` 在 `products.Exists` 之后，新增校验：若该 `product_id` 当前存在 active 竞拍，返回 `ErrProductInAuction`
- 错误码映射：`FP_PRODUCT_IN_AUCTION` → HTTP 409

**关键原则**：
- 前端下拉过滤是 UI 引导，后端兜底校验是安全底线
- 符合"失败关闭"安全偏好

**来源**：session:6a25c4110bfcee1b04fb1b82

## Conventions
- 服务层方法需对 DAO 返回的 nil 做判空保护，避免 panic
- 批量操作的事务边界需明确：全成功/部分成功/全失败的选择需与产品对齐
- 死代码应及时清理，减少维护负担
- 热拉日志需覆盖：入口、候选查询、幂等结果、创建成功/失败、最终数量

## Deployment Notes

### 热拉功能上线检查清单
- [ ] `product_reminder_receipts` 表已创建
- [ ] `user_product_reminders` 索引优化（按用户+时间查询）
- [ ] 热拉接口日志级别调整为 INFO
- [ ] 前端已接入 `hotPull()` 调用

### 诊断日志关键词
排查通知生成问题时关注：
- `HotPullProductReminder: query completed` — 候选查询结果
- `HotPullProductReminder: duplicate skipped` — 幂等生效
- `HotPullProductReminder: create failed` — 写入失败
- `HotPull: product reminder persistence completed` — 最终生成数量

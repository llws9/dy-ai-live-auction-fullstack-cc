# Feature Specification: 直播间通知"冷推热拉"机制

**Feature**: `20260528-livestream-notification-cold-push-hot-pull`
**Created**: 2026-05-28
**Status**: Draft
**Input**: 技术方案讨论：区分冷门/热门直播间，采用不同的通知策略

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 冷门直播间开播通知推送 (Priority: P1)

冷门直播间（关注人数 < 200）开播前10分钟，系统主动推送通知给关注用户。

**目标用户**: 关注冷门直播间的用户

**触发条件**: 
- 直播间计划开播时间前10分钟
- 直播间关注人数 < 200
- 用户已关注该直播间且开启通知

**业务规则**:
- 冷门阈值：关注人数 < 200
- 推送时间：开播前10分钟
- 通知颜色：蓝色（直播即将开始）
- 商品维度的"提醒我"订阅同样适用推送逻辑

**Why this priority**: P1 - 核心冷推机制，直接影响冷门直播间的曝光度

**Technical Implementation**:

#### Redis ZSET 数据结构

| Key | 类型 | 用途 |
|-----|------|------|
| `live_stream:cold:start_time` | ZSET | 冷门直播间开播时间，score为时间戳 |
| `live_stream:{id}:stats` | Hash | 直播间热度状态缓存 |

#### 冷推定时任务逻辑

```
每5分钟执行 → ZRANGEBYSCORE [now, now+10min] → 获取关注用户 → 批量推送通知 → ZREM已处理项
```

**Independent Test**: 创建冷门直播间设置开播时间，验证10分钟前收到推送通知

**Acceptance Scenarios**:

1. **Given** 冷门直播间计划11:00开播, **When** 10:50执行冷推任务, **Then** 关注用户收到"直播即将开始"通知
2. **Given** 直播间已开播, **When** 冷推任务执行, **Then** ZSET中该直播间已移除
3. **Given** 用户关闭通知, **When** 冷推执行, **Then** 该用户不收到推送

---

### User Story 2 - 热门直播间热拉通知 (Priority: P1)

热门直播间（关注人数 ≥ 200），用户登录/切换回前台时主动拉取通知。

**目标用户**: 关注热门直播间的用户

**触发条件**: 
- 用户登录成功
- 用户从后台切换回前台（visibilitychange事件）
- 最小间隔30秒

**业务规则**:
- 热门阈值：关注人数 ≥ 200
- 拉取范围：接下来1小时内开播 OR 正在直播
- 通知颜色：蓝色（即将开播）、红色（正在直播）

**Why this priority**: P1 - 核心热拉机制，避免热门直播间通知轰炸

**Technical Implementation**:

#### Redis ZSET/SET 数据结构

| Key | 类型 | 用途 |
|-----|------|------|
| `live_stream:hot:start_time` | ZSET | 热门直播间开播时间 |
| `live_stream:hot:live_now` | SET | 正在直播的热门直播间 |
| `user:{uid}:followed_live_streams` | SET | 用户关注的直播间列表 |

#### 热拉接口逻辑

```
POST /api/v1/notifications/hot-pull → ZRANGEBYSCORE获取即将开播 → SMEMBERS获取正在直播 → 过滤用户关注 → 返回通知列表
```

#### 前端触发

```typescript
// 登录成功后
hotPullNotifications();

// 页面可见性变化
document.addEventListener('visibilitychange', () => {
  if (document.visibilityState === 'visible') {
    hotPullNotifications(); // 30秒最小间隔
  }
});
```

**Independent Test**: 创建热门直播间，用户登录后验证拉取到通知

**Acceptance Scenarios**:

1. **Given** 热门直播间计划12:00开播, **When** 用户11:30登录, **Then** 拉取到"即将开播"通知
2. **Given** 热门直播间正在直播, **When** 用户切换回前台, **Then** 拉取到"正在直播"通知
3. **Given** 用户30秒内多次切换, **When** 第二次切换, **Then** 跳过热拉（间隔不足）

---

### User Story 3 - 直播间热度状态变更 (Priority: P1)

当直播间关注人数变化时，动态更新冷门/热门状态。

**目标用户**: 系统（内部服务）

**触发条件**: 
- 用户关注直播间
- 用户取消关注直播间

**业务规则**:
- 热度阈值：200人
- 状态切换：冷门→热门（ZSET迁移）、热门→冷门（ZSET迁移）
- 状态一致性：DB和Redis同步更新

**Why this priority**: P1 - 冷推热拉依赖正确的热度状态

**Technical Implementation**:

#### 状态变更逻辑

```
Follow → 更新DB → SADD用户关注SET → 更新热度统计 → 判断是否跨阈值 → ZSET迁移（如需要）
```

#### ZSET迁移

```go
if isHot {
    // 冷门 → 热门
    ZRem("live_stream:cold:start_time", id)
    ZAdd("live_stream:hot:start_time", {score, id})
}
```

**Independent Test**: 关注人数从199→200，验证ZSET迁移

**Acceptance Scenarios**:

1. **Given** 直播间有199关注, **When** +1关注, **Then** 迁移到hot ZSET
2. **Given** 直播间有200关注, **When** -1关注, **Then** 迁移到cold ZSET
3. **Given** 直播间状态缓存, **When** 关注变化, **Then** follower_count同步更新

---

### User Story 4 - 商品"提醒我"订阅 (Priority: P2)

用户点击商品"提醒我"按钮，订阅竞拍即将开始的通知。

**目标用户**: H5用户

**触发条件**: 
- 商品列表页点击"提醒我"按钮
- 竞拍尚未开始

**业务规则**:
- 订阅维度：商品（而非直播间）
- 通知类型：竞拍即将开始
- 通知颜色：红色（同"正在直播"，重要/紧急）
- 通知时机：竞拍开始前10分钟（冷推）或热拉时获取

**Why this priority**: P2 - 商品维度订阅，增强用户体验

**Technical Implementation**:

#### 新增数据模型

| 表名 | 字段 | 说明 |
|------|------|------|
| `user_product_reminders` | user_id, product_id, auction_id, notification_enabled | 商品提醒订阅 |

#### Redis ZSET

| Key | 类型 | 用途 |
|-----|------|------|
| `user:{uid}:product_reminders:start_time` | ZSET | 用户商品提醒，score为竞拍开始时间 |

#### API接口

```
POST /api/v1/products/{id}/remind → 创建订阅 → ZADD
DELETE /api/v1/products/{id}/remind → 取消订阅 → ZREM
```

**Independent Test**: 点击"提醒我"，验证热拉时收到通知

**Acceptance Scenarios**:

1. **Given** 竞拍未开始, **When** 用户点击"提醒我", **Then** 订阅成功，返回reminder_id
2. **Given** 用户已订阅, **When** 热拉执行, **Then** 收到"竞拍即将开始"通知
3. **Given** 用户点击取消, **When** DELETE调用, **Then** ZSET移除订阅

---

### User Story 5 - 通知颜色系统 (Priority: P2)

通知列表中不同类型通知使用不同颜色标识。

**目标用户**: 所有用户

**触发条件**: 
- 用户查看通知列表

**业务规则**:
- 红色：正在直播、竞拍即将开始（重要/紧急）
- 蓝色：直播即将开始（冷门热门统一）
- 绿色：竞拍成功
- 橙色：出价被超越
- 棕色：订单状态变更
- 灰色：竞拍未中标

**Why this priority**: P2 - 通知视觉体验优化

**Technical Implementation**:

#### 通知类型扩展

```go
const (
    NotificationTypeLiveStarting    = "live_starting"    // 蓝色
    NotificationTypeLiveNow         = "live_now"         // 红色
    NotificationTypeAuctionStarting = "auction_starting" // 红色
    NotificationTypeAuctionWon      = "auction_won"      // 绿色
    NotificationTypeBidOutbid       = "bid_outbid"       // 橙色
    NotificationTypeOrderStatus     = "order_status"     // 棕色
    NotificationTypeAuctionLost     = "auction_lost"     // 灰色
)
```

#### 前端颜色映射

```typescript
const colorMap = {
  'live_now': 'red',
  'auction_starting': 'red',
  'live_starting': 'blue',
  'auction_won': 'green',
  'bid_outbid': 'orange',
  'order_status': 'brown',
  'auction_lost': 'gray',
};
```

**Independent Test**: 各类型通知显示正确颜色

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统必须区分冷门直播间（关注<200）和热门直播间（关注≥200）
- **FR-002**: 冷推任务必须使用 ZRANGEBYSCORE 精确获取接下来10分钟内开播的冷门直播间
- **FR-003**: 热拉接口必须返回用户关注的接下来1小时内开播或正在直播的热门直播间
- **FR-004**: 热拉必须触发于用户登录和页面可见性变化（visibilitychange）
- **FR-005**: 热拉最小间隔必须为30秒
- **FR-006**: 直播间热度变更时必须同步更新 ZSET（cold→hot 或 hot→cold）
- **FR-007**: 用户关注/取消关注时必须更新 Redis 用户关注 SET 和热度统计
- **FR-008**: 系统必须支持商品维度的"提醒我"订阅
- **FR-009**: 通知必须按类型显示对应颜色（红/蓝/绿/橙/棕/灰）
- **FR-010**: Badge必须显示未读通知总数

### Key Entities

- **LiveStreamStats**: 直播间热度状态缓存（Redis Hash）
- **UserProductReminder**: 商品提醒订阅（DB表）
- **Notification**: 通知实体，新增 color 字段

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 冷推任务执行时间 < 100ms（O(log N)查询）
- **SC-002**: 热拉接口响应时间 < 200ms
- **SC-003**: 热度变更ZSET迁移时间 < 50ms
- **SC-004**: Redis数据与DB状态100%一致
- **SC-005**: 通知颜色100%正确显示
- **SC-006**: Badge未读数准确无误

## Assumptions

- [INFERRED] Redis客户端已配置并可用
- [INFERRED] 用户认证系统正常工作（JWT）
- [INFERRED] WebSocket推送机制已实现

## Dependencies

- Redis服务正常运行
- 用户关注系统（UserLiveStreamFollow）正常工作
- 通知服务（NotificationService）正常工作

## Risk Mitigation

| 风险 | 处理方案 |
|------|----------|
| Redis数据不一致 | 关键操作同时写DB和Redis，定时校验任务 |
| 开播时间变更 | ZRem旧记录 + ZAdd新score |
| Redis内存占用 | 直播结束后清理Key，设置TTL |
| 热拉接口频繁调用 | 前端30秒间隔 + 后端限流 |

## Observability

### Redis 指标

- ZSET大小监控：`live_stream:cold:start_time`、`live_stream:hot:start_time`
- 热拉接口调用频率
- 冷推任务执行耗时

### Prometheus 指标

```go
// 新增指标
coldPushLatency    *prometheus.HistogramVec  // 冷推任务耗时
hotPullLatency     *prometheus.HistogramVec  // 热拉接口耗时
zsetSize           *prometheus.GaugeVec      // ZSET大小
hotnessTransition  *prometheus.CounterVec    // 热度变更次数
```
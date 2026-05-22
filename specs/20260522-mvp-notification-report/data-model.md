# Data Model: MVP阶段功能完善

**Feature**: `20260522-mvp-notification-report`
**Date**: 2026-05-22

## Entities

### Notification

通知消息实体，用于存储用户通知并支持实时推送。

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | bigint | PK, AUTO_INCREMENT | 主键 |
| user_id | bigint | NOT NULL, INDEX | 接收用户ID |
| type | varchar(32) | NOT NULL, INDEX | 通知类型 |
| title | varchar(128) | NOT NULL | 通知标题 |
| content | text | NOT NULL | 通知内容 |
| data | json | NULL | 扩展数据（auction_id, order_id等） |
| read_at | datetime | NULL | 已读时间，NULL表示未读 |
| created_at | datetime | NOT NULL, DEFAULT NOW() | 创建时间 |

**Indexes**:
- `idx_user_id_created_at` (user_id, created_at DESC) - 用户通知列表查询
- `idx_user_id_read_at` (user_id, read_at) - 未读通知计数查询

**Notification Types**:
```go
const (
    NotificationTypeBidOutbid    = "bid_outbid"    // 出价被超越
    NotificationTypeAuctionWon   = "auction_won"   // 竞拍中标
    NotificationTypeAuctionLost  = "auction_lost"  // 竞拍未中标
    NotificationTypeOrderPaid    = "order_paid"    // 订单已支付 (Mock)
    NotificationTypeOrderShipped = "order_shipped" // 订单已发货 (Mock)
    NotificationTypeOrderCompleted = "order_completed" // 订单已完成 (Mock)
)
```

### NotificationRequest (DTO)

通知发送请求，用于通知服务接口。

```go
type NotificationRequest struct {
    UserID      int64                  // 接收用户ID
    Type        NotificationType       // 通知类型
    Title       string                 // 通知标题
    Content     string                 // 通知内容
    Data        map[string]interface{} // 扩展数据
    Immediately bool                   // 是否立即推送（默认true）
}
```

### OrderEvent (DTO - Phase 2)

订单事件，用于二期订单系统集成。

```go
type OrderEvent struct {
    OrderID    int64
    EventType  OrderEventType // PAID, SHIPPED, COMPLETED, CANCELLED
    OldStatus  int
    NewStatus  int
    UserID     int64
    Timestamp  time.Time
    Extra      map[string]interface{}
}
```

## Statistics DTOs

### StatisticsOverview

统计总览数据，用于管理后台首页大屏。

```go
type StatisticsOverview struct {
    TotalAuctions    int64   // 总竞拍场次
    SuccessRate      float64 // 成功率
    TotalRevenue     float64 // 总成交额
    TotalUsers       int64   // 总用户数
    ActiveUsers      int64   // 活跃用户数（近7天）
}
```

### AuctionStatistics

竞拍统计数据。

```go
type AuctionStatistics struct {
    TotalAuctions    int64   // 总场次
    SuccessRate      float64 // 成功率
    AvgBidCount      float64 // 平均出价次数
    TopAuctions      []AuctionSummary // 热门竞拍
}
```

### RevenueStatistics

收入统计数据。

```go
type RevenueStatistics struct {
    TotalRevenue     float64            // 总成交额
    DailyRevenue     []DailyRevenue     // 日均收入趋势
    CategoryDistribution []CategoryRevenue // 类目分布
}
```

### UserStatistics

用户统计数据。

```go
type UserStatistics struct {
    TotalUsers       int64          // 总用户数
    ActiveUsers      int64          // 活跃用户
    NewUsers         int64          // 新用户（近7天）
    BidDistribution  []BidRange     // 出价分布
}
```

## Database Migrations

### Migration: Create notifications table

```sql
CREATE TABLE `notifications` (
    `id` bigint NOT NULL AUTO_INCREMENT,
    `user_id` bigint NOT NULL,
    `type` varchar(32) NOT NULL,
    `title` varchar(128) NOT NULL,
    `content` text NOT NULL,
    `data` json DEFAULT NULL,
    `read_at` datetime DEFAULT NULL,
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX `idx_user_id_created_at` (`user_id`, `created_at` DESC),
    INDEX `idx_user_id_read_at` (`user_id`, `read_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

## State Transitions

### Notification Lifecycle

```
[Created] ──────> [Read] ──────> [Deleted]
     │                │
     └── WebSocket ───┘
         Push
```

### Order Notification Trigger (Mock)

```
[Mock PayOrder] ───> SendNotification(order_paid)
[Mock ShipOrder] ─> SendNotification(order_shipped)
[Mock CompleteOrder] > SendNotification(order_completed)
```

**Phase 2 Transition**:
```
[Real Order Service] ──> OrderEventPublisher ──> NotificationService
```

## Relationships

```
User (1) ────────< (N) Notification
                        │
                        ├── data.auction_id ──> Auction
                        └── data.order_id ───> Order
```

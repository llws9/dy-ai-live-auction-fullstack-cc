# Data Model: 直播竞拍系统核心功能完善

**Feature**: `20260522-live-auction-core`
**Date**: 2026-05-22

## Overview

本文档定义核心功能完善所需的实体模型、状态流转和关系设计。

---

## 1. 排名相关实体

### 1.1 Bid (出价记录) - 已存在

```go
type Bid struct {
    ID         int64     `gorm:"primaryKey"`
    AuctionID  int64     `gorm:"index;not null"`
    UserID     int64     `gorm:"index;not null"`
    Amount     float64   `gorm:"not null"`
    CreatedAt  time.Time `gorm:"autoCreateTime"`
}
```

**新增方法**：
- `GetRanking(auctionID int64, limit int) []Bid` - 获取排名列表
- `GetUserRank(auctionID, userID int64) int` - 获取用户排名

### 1.2 RankingItem (排名项) - 前端展示模型

```typescript
interface RankingItem {
  rank: number;          // 排名位置
  userId: number;        // 用户ID
  userName: string;      // 用户名称
  userAvatar: string;    // 用户头像
  amount: number;        // 出价金额
  bidTime: string;       // 出价时间 (ISO 8601)
}
```

**用途**：前端实时排名列表展示

---

## 2. 断线重连相关实体

### 2.1 ConnectionState (连接状态) - 内存状态

```go
type ConnectionState struct {
    ClientID      string    `json:"client_id"`
    AuctionID     int64     `json:"auction_id"`
    UserID        int64     `json:"user_id"`
    ConnectedAt   time.Time `json:"connected_at"`
    LastPongAt    time.Time `json:"last_pong_at"`
    ReconnectCount int      `json:"reconnect_count"`
}
```

**存储位置**：Redis (key: `conn:state:{client_id}`)

### 2.2 SyncState (同步状态) - 重连状态缓存

```go
type SyncState struct {
    AuctionID     int64     `json:"auction_id"`
    CurrentPrice  float64   `json:"current_price"`
    WinnerID      int64     `json:"winner_id"`
    EndTime       time.Time `json:"end_time"`
    Status        int       `json:"status"`
    UpdatedAt     time.Time `json:"updated_at"`
}
```

**存储位置**：Redis (key: `sync:state:{auction_id}`)

**状态同步流程**：
```
重连成功 → 客户端发送 sync_request → 服务端查询 SyncState → 返回 sync_response
```

---

## 3. 管理后台相关实体

### 3.1 Product (商品) - 已存在

```go
type Product struct {
    ID          int64     `gorm:"primaryKey"`
    Name        string    `gorm:"size:255;not null"`
    Description string    `gorm:"type:text"`
    Images      string    `gorm:"type:text"` // JSON array
    Status      int       `gorm:"default:0"` // 0=draft, 1=active, 2=archived
    CreatedAt   time.Time `gorm:"autoCreateTime"`
    UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}
```

**新增字段**：
- `Status` - 商品状态（草稿、上架、下架）

### 3.2 AuctionRule (竞拍规则) - 已存在

```go
type AuctionRule struct {
    ID                 int64     `gorm:"primaryKey"`
    ProductID          int64     `gorm:"uniqueIndex;not null"`
    StartPrice         float64   `gorm:"default:0"`
    Increment          float64   `gorm:"not null"`
    CapPrice           *float64  `gorm:"default:null"`
    Duration           int       `gorm:"not null"` // 秒
    DelayDuration      int       `gorm:"default:30"`
    MaxDelayTime       int       `gorm:"default:180"`
    TriggerDelayBefore int       `gorm:"default:30"`
    CreatedAt          time.Time `gorm:"autoCreateTime"`
}
```

**无需变更**，现有模型已满足需求

### 3.3 Order (订单) - 新增

```go
type Order struct {
    ID           int64     `gorm:"primaryKey"`
    AuctionID    int64     `gorm:"uniqueIndex;not null"`
    ProductID    int64     `gorm:"index;not null"`
    WinnerID     int64     `gorm:"index;not null"`
    FinalPrice   float64   `gorm:"not null"`
    Status       int       `gorm:"default:0"` // 0=pending, 1=paid, 2=shipped, 3=completed
    PaidAt       *time.Time
    ShippedAt    *time.Time
    CompletedAt  *time.Time
    CreatedAt    time.Time `gorm:"autoCreateTime"`
    UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}
```

**订单状态流转**：
```
pending → paid → shipped → completed
         ↓
       cancelled
```

### 3.4 User (用户) - 已存在

```go
type User struct {
    ID        int64     `gorm:"primaryKey"`
    Name      string    `gorm:"size:100;not null"`
    Avatar    string    `gorm:"size:255"`
    Role      string    `gorm:"size:20;default:'user'"` // user, admin, operator
    CreatedAt time.Time `gorm:"autoCreateTime"`
}
```

**新增字段**：
- `Role` - 用户角色（普通用户、管理员、运营）

---

## 4. 体验优化相关实体

### 4.1 UserHistory (用户历史) - 查询视图

```go
// 不需要单独的表，通过查询关联数据生成
type UserHistoryItem struct {
    AuctionID   int64     `json:"auction_id"`
    ProductName string    `json:"product_name"`
    FinalPrice  float64   `json:"final_price"`
    IsWinner    bool      `json:"is_winner"`
    BidCount    int       `json:"bid_count"`
    CreatedAt   time.Time `json:"created_at"`
}
```

**查询逻辑**：
```sql
SELECT 
    a.id AS auction_id,
    p.name AS product_name,
    a.current_price AS final_price,
    (a.winner_id = ?) AS is_winner,
    COUNT(b.id) AS bid_count,
    a.created_at
FROM auctions a
JOIN products p ON a.product_id = p.id
LEFT JOIN bids b ON a.id = b.auction_id AND b.user_id = ?
WHERE a.id IN (
    SELECT DISTINCT auction_id FROM bids WHERE user_id = ?
)
ORDER BY a.created_at DESC
```

---

## 5. WebSocket 消息类型

### 5.1 服务端 → 客户端

```go
type Message struct {
    Type string          `json:"type"`
    Data json.RawMessage `json:"data"`
}
```

**消息类型定义**：

| 类型 | 描述 | 数据结构 |
|------|------|----------|
| `bid_placed` | 有人出价 | `{amount, user_id, user_name, timestamp}` |
| `rank_update` | 排名更新 | `{rankings: [{rank, user_id, amount}]}` |
| `overtaken` | 被超越通知 | `{by_user_id, by_user_name, new_rank}` |
| `delay_triggered` | 延时触发 | `{new_end_time, delay_duration}` |
| `auction_ended` | 竞拍结束 | `{winner_id, winner_name, final_price}` |
| `time_sync` | 时间同步 | `{server_time, end_time}` |
| `sync_response` | 状态同步响应 | `{auction_state, current_price, winner_id, end_time}` |

### 5.2 客户端 → 服务端

| 类型 | 描述 | 数据结构 |
|------|------|----------|
| `ping` | 心跳 | `{timestamp}` |
| `sync_request` | 状态同步请求 | `{auction_id}` |

---

## 6. 状态机设计

### 6.1 Auction 状态机 - 已存在

```
pending (0) ──(开始)──▶ ongoing (1)
     │                     │
     │                     ├─(延时触发)──▶ delayed (2)
     │                     │                   │
     │                     └─(结束)──▶ ended (3)
     │                                          ▲
     └─(取消)──▶ cancelled (4)                  │
                                           (延时结束)
```

**新增状态**：
- `delayed (2)` - 延时中

### 6.2 Order 状态机 - 新增

```
pending (0) ──(支付)──▶ paid (1) ──(发货)──▶ shipped (2) ──(确认)──▶ completed (3)
     │
     └─(取消)──▶ cancelled (-1)
```

---

## 7. 索引设计

### 7.1 新增索引

```sql
-- 订单表
CREATE INDEX idx_order_product ON orders(product_id);
CREATE INDEX idx_order_winner ON orders(winner_id);
CREATE UNIQUE INDEX idx_order_auction ON orders(auction_id);

-- 用户表
CREATE INDEX idx_user_role ON users(role);
```

### 7.2 已存在索引优化

```sql
-- 出价表（已存在）
-- CREATE INDEX idx_bid_auction ON bids(auction_id);
-- CREATE INDEX idx_bid_user ON bids(user_id);

-- 优化：添加复合索引用于排名查询
CREATE INDEX idx_bid_auction_amount ON bids(auction_id, amount DESC);
```

---

## 8. 数据一致性保证

### 8.1 分布式锁

**现有实现**：`AuctionBidLock` (Redis 分布式锁)

```go
Key: auction:bid:{auction_id}:lock
Value: {user_id}:{timestamp}
TTL: 5秒
```

**适用场景**：
- 并发出价
- 排名更新
- 状态转换

### 8.2 乐观锁

**适用场景**：
- 订单状态更新
- 商品状态更新

```go
// 更新时带版本检查
UPDATE orders SET status = ?, version = version + 1
WHERE id = ? AND version = ?
```

---

## Summary

数据模型设计完成，新增实体：
- `Order` - 订单表
- `ConnectionState` - 连接状态（Redis）
- `SyncState` - 同步状态（Redis）

扩展现有实体：
- `Product.Status` - 商品状态
- `User.Role` - 用户角色

无需变更：
- `Auction`、`Bid`、`AuctionRule` 现有模型已满足需求

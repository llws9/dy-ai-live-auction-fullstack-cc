# Data Model: 直播竞拍系统核心功能完善

**Feature**: `20260522-core-features-enhancement`
**Date**: 2026-05-22

## 概述

本次功能变更主要涉及服务层逻辑，数据库变更仅为添加字段，无需新建表。

---

## 数据库变更

### 1. users 表

**新增字段**:

| 字段名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `role` | INT | 0 | 用户角色 (0=普通用户, 1=主播, 2=平台管理员) |

**迁移SQL**:
```sql
ALTER TABLE users ADD COLUMN role INT DEFAULT 0 COMMENT '用户角色: 0=普通用户, 1=主播, 2=平台管理员';
```

---

### 2. auctions 表

**新增字段**:

| 字段名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `creator_id` | BIGINT | NULL | 竞拍创建者ID（主播），用于归属检查 |

**迁移SQL**:
```sql
ALTER TABLE auctions ADD COLUMN creator_id BIGINT DEFAULT NULL COMMENT '竞拍创建者ID';
CREATE INDEX idx_auctions_creator_id ON auctions(creator_id);
```

---

## Redis 数据结构

### 1. 分布式锁

**Key Pattern**: `lock:auction:{auction_id}:bid`

**Value**: `"locked"`

**TTL**: 5秒

**用途**: 竞拍出价时的分布式锁

---

### 2. WebSocket连接状态

**Key Pattern**: `conn:state:{client_id}`

**Value**: JSON
```json
{
  "client_id": "1-123-4567890123",
  "auction_id": 1,
  "user_id": 123,
  "connected_at": "2026-05-22T10:00:00Z",
  "last_pong_at": "2026-05-22T10:00:30Z",
  "reconnect_count": 0
}
```

**TTL**: 24小时

**用途**: WebSocket连接状态持久化，支持重连恢复

---

### 3. 竞拍同步状态

**Key Pattern**: `sync:state:{auction_id}`

**Value**: JSON
```json
{
  "auction_id": 1,
  "current_price": 150.00,
  "winner_id": 123,
  "end_time": "2026-05-22T11:00:00Z",
  "status": 1,
  "updated_at": "2026-05-22T10:30:00Z"
}
```

**TTL**: 7天

**用途**: 竞拍状态缓存，快速同步

---

## Go 模型定义

### Role 角色枚举

```go
// backend/auction/model/user.go

type Role int

const (
    RoleUser      Role = 0 // 普通用户
    RoleStreamer  Role = 1 // 主播
    RoleAdmin     Role = 2 // 平台管理员
)

func (r Role) String() string {
    switch r {
    case RoleUser:
        return "普通用户"
    case RoleStreamer:
        return "主播"
    case RoleAdmin:
        return "平台管理员"
    default:
        return "未知"
    }
}
```

### DistributedLock 分布式锁

```go
// backend/auction/service/lock.go

type DistributedLockService struct {
    redis       *redis.Client
    localLocks  sync.Map // 本地锁降级
    defaultTTL  time.Duration
}

type Lock struct {
    Key       string
    Value     string
    TTL       time.Duration
    Acquired  bool
}
```

### ConnectionState 连接状态

```go
// backend/auction/websocket/state_sync.go (已存在)

type ConnectionState struct {
    ClientID       string    `json:"client_id"`
    AuctionID      int64     `json:"auction_id"`
    UserID         int64     `json:"user_id"`
    ConnectedAt    time.Time `json:"connected_at"`
    LastPongAt     time.Time `json:"last_pong_at"`
    ReconnectCount int       `json:"reconnect_count"`
}
```

### UserHistory 用户历史记录

```go
// backend/product/service/history.go

type UserHistoryItem struct {
    AuctionID   int64   `json:"auction_id"`
    ProductName string  `json:"product_name"`
    FinalPrice  float64 `json:"final_price"`
    IsWinner    bool    `json:"is_winner"`
    BidCount    int     `json:"bid_count"`
    CreatedAt   string  `json:"created_at"`
}
```

---

## 实体关系

```
┌─────────────┐     1:N      ┌─────────────┐
│   User      │─────────────▶│   Auction   │
│  (role)     │              │ (creator_id)│
└─────────────┘              └─────────────┘
      │                             │
      │ 1:N                         │ 1:N
      ▼                             ▼
┌─────────────┐              ┌─────────────┐
│    Bid      │              │   Order     │
└─────────────┘              └─────────────┘
```

- User 通过 role 字段区分角色
- Auction 通过 creator_id 关联创建者（主播）
- Bid 关联用户和竞拍
- Order 关联竞拍和中标者

# Data Model: 数据模型设计

**Feature**: `20260523-product-auction-live`
**Date**: 2026-05-23

## 实体关系图

```
User (用户) 1--* UserLiveStreamFollow (关注) *--1 LiveStream (直播间)
LiveStream (直播间) 1--* Auction (竞拍) *--1 Product (商品)
Auction (竞拍) 1--* Bid (出价)
User (用户) 1--* Bid (出价)
LiveStream (直播间) 1--1 User (商家/主播)
```

## 核心实体定义

### 1. LiveStream (直播间) - 新增

**用途**: 直播间实体，与商家一对一关联

**表名**: `live_streams`

**字段定义**:

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| creator_id | BIGINT | NOT NULL, UNIQUE, INDEX | 商家ID（用户ID），一对一关系 |
| name | VARCHAR(128) | NOT NULL | 直播间名称 |
| description | TEXT | NULL | 直播间描述 |
| cover_image | VARCHAR(256) | NULL | 封面图URL |
| status | TINYINT | DEFAULT 1 | 状态：0=禁用，1=正常 |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP ON UPDATE | 更新时间 |

**索引**:
- PRIMARY KEY (id)
- UNIQUE KEY uk_creator_id (creator_id)
- INDEX idx_status (status)

**Go模型**:
```go
type LiveStreamStatus int

const (
    LiveStreamStatusDisabled LiveStreamStatus = 0
    LiveStreamStatusActive   LiveStreamStatus = 1
)

type LiveStream struct {
    ID          int64             `json:"id" gorm:"primaryKey;autoIncrement"`
    CreatorID   int64             `json:"creator_id" gorm:"uniqueIndex;not null"`
    Name        string            `json:"name" gorm:"type:varchar(128);not null"`
    Description string            `json:"description" gorm:"type:text"`
    CoverImage  string            `json:"cover_image" gorm:"type:varchar(256)"`
    Status      LiveStreamStatus  `json:"status" gorm:"type:tinyint;default:1"`
    CreatedAt   time.Time         `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt   time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
}

func (LiveStream) TableName() string {
    return "live_streams"
}
```

**状态转换**:
```
Active (正常) ↔ Disabled (禁用)
```

**业务规则**:
- 商家首次发布商品时自动创建
- 禁用直播间时，取消所有待开始和进行中的竞拍
- 管理员可以禁用直播间

---

### 2. UserLiveStreamFollow (用户关注直播间) - 新增

**用途**: 记录用户关注直播间的关系

**表名**: `user_live_stream_follows`

**字段定义**:

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| user_id | BIGINT | NOT NULL, INDEX | 用户ID |
| live_stream_id | BIGINT | NOT NULL, INDEX | 直播间ID |
| notification_enabled | TINYINT | DEFAULT 1 | 是否接收通知：0=否，1=是 |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | 关注时间 |

**索引**:
- PRIMARY KEY (id)
- UNIQUE KEY uk_user_live_stream (user_id, live_stream_id)
- INDEX idx_user_id (user_id)
- INDEX idx_live_stream_id (live_stream_id)

**Go模型**:
```go
type UserLiveStreamFollow struct {
    ID                  int64     `json:"id" gorm:"primaryKey;autoIncrement"`
    UserID              int64     `json:"user_id" gorm:"uniqueIndex:uk_user_live_stream;not null"`
    LiveStreamID        int64     `json:"live_stream_id" gorm:"uniqueIndex:uk_user_live_stream;not null;index"`
    NotificationEnabled bool      `json:"notification_enabled" gorm:"default:true"`
    CreatedAt           time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (UserLiveStreamFollow) TableName() string {
    return "user_live_stream_follows"
}
```

**业务规则**:
- 用户可以关注多个直播间
- 可以开启/关闭特定直播间的通知
- 关注关系不删除，仅通过notification_enabled控制通知

---

### 3. Product (商品) - 修改

**修改内容**: 新增状态值

**原状态**:
- 0: Draft (草稿)
- 1: Published (已发布)

**新增状态**:
- 2: Unpublished (已下架)

**状态转换**:
```
Draft (草稿) → Published (已发布) → Unpublished (已下架) → Published (已发布)
                                    ↓
                                Draft (草稿) [可选]
```

**Go模型修改**:
```go
const (
    ProductStatusDraft       ProductStatus = 0 // 草稿
    ProductStatusPublished   ProductStatus = 1 // 已发布
    ProductStatusUnpublished ProductStatus = 2 // 已下架
)
```

**业务规则**:
- 下架时取消所有待开始和进行中的竞拍
- 已下架商品可以重新发布
- 重新发布时创建新的竞拍记录

---

### 4. Auction (竞拍) - 修改

**修改内容**: 新增字段关联直播间

**新增字段**:

| 字段名 | 类型 | 约束 | 说明 |
|--------|------|------|------|
| live_stream_id | BIGINT | NULL, INDEX | 直播间ID（可选，兼容旧数据） |

**Go模型修改**:
```go
type Auction struct {
    ID           int64          `json:"id" gorm:"primaryKey;autoIncrement"`
    ProductID    int64          `json:"product_id" gorm:"index;not null"`
    LiveStreamID *int64         `json:"live_stream_id" gorm:"index"` // 新增字段
    CreatorID    *int64         `json:"creator_id" gorm:"index"`
    Status       AuctionStatus  `json:"status" gorm:"type:tinyint;default:0"`
    CurrentPrice float64        `json:"current_price" gorm:"type:decimal(10,2);default:0"`
    WinnerID     *int64         `json:"winner_id"`
    StartTime    time.Time      `json:"start_time" gorm:"index;not null"`
    EndTime      time.Time      `json:"end_time" gorm:"not null"`
    DelayUsed    int            `json:"delay_used" gorm:"default:0"`
    CreatedAt    time.Time      `json:"created_at" gorm:"autoCreateTime"`
}
```

**数据迁移**:
```sql
-- 为现有竞拍记录设置live_stream_id
UPDATE auctions a
JOIN users u ON a.creator_id = u.id
JOIN live_streams ls ON ls.creator_id = u.id
SET a.live_stream_id = ls.id
WHERE a.live_stream_id IS NULL;
```

---

### 5. Notification (通知) - 扩展

**扩展内容**: 新增通知类型

**新增通知类型**:
- `new_product`: 新商品发布
- `auction_starting`: 竞拍即将开始（提前30分钟）
- `auction_ended`: 竞拍结束
- `product_unpublished`: 商品下架

**Go模型**:
```go
type NotificationType string

const (
    NotificationTypeNewProduct        NotificationType = "new_product"
    NotificationTypeAuctionStarting   NotificationType = "auction_starting"
    NotificationTypeAuctionEnded      NotificationType = "auction_ended"
    NotificationTypeProductUnpublished NotificationType = "product_unpublished"
    // ... 原有类型
)

type Notification struct {
    ID          int64            `json:"id" gorm:"primaryKey;autoIncrement"`
    UserID      int64            `json:"user_id" gorm:"index;not null"`
    Type        NotificationType `json:"type" gorm:"type:varchar(50);not null"`
    Title       string           `json:"title" gorm:"type:varchar(256);not null"`
    Content     string           `json:"content" gorm:"type:text"`
    Data        JSON             `json:"data" gorm:"type:json"` // 扩展数据（商品ID、竞拍ID等）
    IsRead      bool             `json:"is_read" gorm:"default:false"`
    CreatedAt   time.Time        `json:"created_at" gorm:"autoCreateTime;index"`
}
```

---

## 数据迁移脚本

### 1. 创建新表

```sql
-- 创建直播间表
CREATE TABLE IF NOT EXISTS live_streams (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    creator_id BIGINT NOT NULL UNIQUE COMMENT '商家ID',
    name VARCHAR(128) NOT NULL COMMENT '直播间名称',
    description TEXT COMMENT '直播间描述',
    cover_image VARCHAR(256) COMMENT '封面图',
    status TINYINT DEFAULT 1 COMMENT '状态：0=禁用，1=正常',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='直播间表';

-- 创建用户关注直播间表
CREATE TABLE IF NOT EXISTS user_live_stream_follows (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    live_stream_id BIGINT NOT NULL COMMENT '直播间ID',
    notification_enabled TINYINT DEFAULT 1 COMMENT '是否接收通知',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '关注时间',
    UNIQUE KEY uk_user_live_stream (user_id, live_stream_id),
    INDEX idx_user_id (user_id),
    INDEX idx_live_stream_id (live_stream_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户关注直播间表';
```

### 2. 修改现有表

```sql
-- 为auctions表新增live_stream_id字段
ALTER TABLE auctions
ADD COLUMN live_stream_id BIGINT NULL COMMENT '直播间ID' AFTER product_id,
ADD INDEX idx_live_stream_id (live_stream_id);

-- 为products表新增状态值（已有status字段，无需修改）
-- 更新注释
ALTER TABLE products
MODIFY COLUMN status TINYINT DEFAULT 0 COMMENT '状态: 0=草稿, 1=已发布, 2=已下架';
```

### 3. 数据初始化

```sql
-- 为现有商家创建直播间
INSERT INTO live_streams (creator_id, name, description, status, created_at)
SELECT
    id as creator_id,
    CONCAT(name, '的直播间') as name,
    CONCAT(name, '的个人直播间') as description,
    1 as status,
    created_at
FROM users
WHERE role = 1 -- 主播/商家
ON DUPLICATE KEY UPDATE
    name = VALUES(name);

-- 为现有竞拍记录设置live_stream_id
UPDATE auctions a
JOIN live_streams ls ON a.creator_id = ls.creator_id
SET a.live_stream_id = ls.id
WHERE a.live_stream_id IS NULL;
```

---

## 统计数据查询

### 1. 直播间关注统计

```sql
-- 关注总数
SELECT COUNT(*) FROM user_live_stream_follows WHERE live_stream_id = ?;

-- 今日新增
SELECT COUNT(*) FROM user_live_stream_follows
WHERE live_stream_id = ? AND DATE(created_at) = CURDATE();

-- 本周新增
SELECT COUNT(*) FROM user_live_stream_follows
WHERE live_stream_id = ? AND YEARWEEK(created_at) = YEARWEEK(NOW());

-- 本月新增
SELECT COUNT(*) FROM user_live_stream_follows
WHERE live_stream_id = ? AND DATE_FORMAT(created_at, '%Y-%m') = DATE_FORMAT(NOW(), '%Y-%m');

-- 活跃用户数（近7天有出价）
SELECT COUNT(DISTINCT ulf.user_id)
FROM user_live_stream_follows ulf
JOIN bids b ON ulf.user_id = b.user_id
WHERE ulf.live_stream_id = ? AND b.created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY);

-- 参与过竞拍的关注用户数
SELECT COUNT(DISTINCT ulf.user_id)
FROM user_live_stream_follows ulf
JOIN bids b ON ulf.user_id = b.user_id
JOIN auctions a ON b.auction_id = a.id
WHERE ulf.live_stream_id = ? AND a.live_stream_id = ulf.live_stream_id;
```

### 2. 直播间竞拍统计

```sql
-- 当前活跃竞拍数
SELECT COUNT(*) FROM auctions
WHERE live_stream_id = ? AND status IN (0, 1, 2); -- 待开始、进行中、延时中

-- 历史成交额
SELECT COALESCE(SUM(current_price), 0) FROM auctions
WHERE live_stream_id = ? AND status = 3; -- 已结束
```

---

## 性能优化建议

### 1. 索引优化

- `user_live_stream_follows(live_stream_id, notification_enabled)`: 用于查询可接收通知的关注用户
- `auctions(live_stream_id, status)`: 用于查询直播间下的竞拍
- `live_streams(creator_id, status)`: 用于查询商家的直播间

### 2. 缓存策略

- 直播间关注数: Redis `live_stream:{id}:followers_count`
- 直播间活跃竞拍数: Redis `live_stream:{id}:active_auctions`
- 直播间总成交额: Redis `live_stream:{id}:total_revenue`

### 3. 分页查询

- 用户关注列表: 使用cursor-based分页，避免深分页
- 直播间列表: 使用offset分页，限制最大页数
- 关注用户列表（批量推送）: 使用offset分页，每批1万

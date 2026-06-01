# A5 一口价秒杀 - M1 后端核心抢购链路 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现后端一口价商品的上下架、抢购（含幂等、Saga 补偿、并发零超卖），完成单元 + 集成测试，可通过 curl 演示完整闭环。

**Architecture:** 在 auction-service 内新增 fixed_price 模块（dao/service/handler 三层）；Redis Lua 原子保证库存权威；DB UNIQUE 兜底；强制 X-Idempotency-Key 防重；DB 事务失败触发 Lua 反向补偿。auctions 表零修改。

**Tech Stack:** Go 1.24 + Hertz + GORM + go-redis + shopspring/decimal + Toxiproxy + testify

**Spec：** [2026-06-01-fixed-price-sale-design.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/superpowers/specs/2026-06-01-fixed-price-sale-design.md)

---

## File Structure

**Create:**
- `backend/auction/model/fixed_price.go` - 实体 + 状态枚举 + 错误定义
- `backend/auction/dao/fixed_price_item.go` + `_test.go` - 商品表 CRUD
- `backend/auction/dao/fixed_price_purchase.go` + `_test.go` - 购买记录 CRUD
- `backend/auction/service/fixed_price.go` + `_test.go` - 业务编排
- `backend/auction/service/fixed_price_lua.go` - 内嵌 Lua 脚本
- `backend/auction/service/fixed_price_idem.go` - 幂等存储
- `backend/auction/handler/fixed_price.go` + `_test.go` - HTTP 入口
- `backend/auction/handler/fixed_price_http.go` - 路由注册
- `backend/migrations/2026060101_create_fixed_price_tables.up.sql` - DDL

**Modify:**
- `backend/auction/handler/router.go` - 挂载新路由
- `backend/auction/model/order.go` - 加 `Source` 列（如已有 order model）
- `backend/gateway/router.go` - 转发 `/api/v1/fixed-price/*` 到 auction-service

---

### Task 1: 数据库迁移与 model 定义

**Files:**
- Create: `backend/migrations/2026060101_create_fixed_price_tables.up.sql`
- Create: `backend/migrations/2026060101_create_fixed_price_tables.down.sql`
- Create: `backend/auction/model/fixed_price.go`
- Modify: `backend/auction/model/order.go`（加 source 字段）

- [ ] **Step 1.1: 写 up DDL**

```sql
-- backend/migrations/2026060101_create_fixed_price_tables.up.sql
CREATE TABLE fixed_price_items (
  id              BIGINT AUTO_INCREMENT PRIMARY KEY,
  live_stream_id  BIGINT NOT NULL,
  product_id      BIGINT NOT NULL,
  creator_id      BIGINT NOT NULL,
  price           DECIMAL(10,2) NOT NULL,
  total_stock     INT NOT NULL,
  remaining_stock INT NOT NULL,
  max_per_user    INT NOT NULL DEFAULT 1,
  status          TINYINT NOT NULL DEFAULT 1,
  version         INT NOT NULL DEFAULT 0,
  created_at      DATETIME NOT NULL,
  updated_at      DATETIME NOT NULL,
  INDEX idx_live_stream (live_stream_id, status),
  INDEX idx_creator (creator_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE fixed_price_purchases (
  id         BIGINT AUTO_INCREMENT PRIMARY KEY,
  item_id    BIGINT NOT NULL,
  user_id    BIGINT NOT NULL,
  order_id   BIGINT NOT NULL,
  price      DECIMAL(10,2) NOT NULL,
  created_at DATETIME NOT NULL,
  UNIQUE KEY uniq_item_user (item_id, user_id),
  INDEX idx_user (user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE orders ADD COLUMN source TINYINT NOT NULL DEFAULT 0;
```

- [ ] **Step 1.2: 写 down DDL**

```sql
-- backend/migrations/2026060101_create_fixed_price_tables.down.sql
ALTER TABLE orders DROP COLUMN source;
DROP TABLE fixed_price_purchases;
DROP TABLE fixed_price_items;
```

- [ ] **Step 1.3: 写 model**

```go
// backend/auction/model/fixed_price.go
package model

import (
    "time"
    "github.com/shopspring/decimal"
)

type FixedPriceStatus int8

const (
    FixedPriceStatusOnSale  FixedPriceStatus = 1
    FixedPriceStatusSoldOut FixedPriceStatus = 2
    FixedPriceStatusOffline FixedPriceStatus = 3
)

type FixedPriceItem struct {
    ID             int64            `gorm:"primaryKey;autoIncrement" json:"id"`
    LiveStreamID   int64            `gorm:"index;not null" json:"live_stream_id"`
    ProductID      int64            `gorm:"not null" json:"product_id"`
    CreatorID      int64            `gorm:"index;not null" json:"creator_id"`
    Price          decimal.Decimal  `gorm:"type:decimal(10,2);not null" json:"price"`
    TotalStock     int              `gorm:"not null" json:"total_stock"`
    RemainingStock int              `gorm:"not null" json:"remaining_stock"`
    MaxPerUser     int              `gorm:"not null;default:1" json:"max_per_user"`
    Status         FixedPriceStatus `gorm:"type:tinyint;not null;default:1" json:"status"`
    Version        int              `gorm:"not null;default:0" json:"version"`
    CreatedAt      time.Time        `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt      time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
}

func (FixedPriceItem) TableName() string { return "fixed_price_items" }

type FixedPricePurchase struct {
    ID        int64           `gorm:"primaryKey;autoIncrement" json:"id"`
    ItemID    int64           `gorm:"uniqueIndex:uniq_item_user;not null" json:"item_id"`
    UserID    int64           `gorm:"uniqueIndex:uniq_item_user;not null" json:"user_id"`
    OrderID   int64           `gorm:"not null" json:"order_id"`
    Price     decimal.Decimal `gorm:"type:decimal(10,2);not null" json:"price"`
    CreatedAt time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

func (FixedPricePurchase) TableName() string { return "fixed_price_purchases" }
```

- [ ] **Step 1.4: 在 Order model 加 Source 列**

```go
// backend/auction/model/order.go (在已有 Order struct 末尾加字段)
Source int8 `gorm:"type:tinyint;not null;default:0" json:"source"` // 0=auction 1=fixed_price
```

- [ ] **Step 1.5: 验证迁移与编译**

Run:
```
cd backend/auction && go build ./...
```
Expected: 编译通过，无错误

- [ ] **Step 1.6: Commit**

```bash
git add backend/migrations/2026060101_*.sql backend/auction/model/fixed_price.go backend/auction/model/order.go
git commit -m "feat(fixed-price): add DDL and models for fixed price sale (M1.T1)"
```

---

### Task 2: dao 层 - FixedPriceItem CRUD（TDD）

**Files:**
- Test: `backend/auction/dao/fixed_price_item_test.go`
- Create: `backend/auction/dao/fixed_price_item.go`

- [ ] **Step 2.1: 写失败测试**

```go
// backend/auction/dao/fixed_price_item_test.go
package dao

import (
    "context"
    "testing"
    "github.com/shopspring/decimal"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "your-module/backend/auction/model"
)

func TestFixedPriceItemDAO_CreateAndGet(t *testing.T) {
    db := setupTestDB(t)
    dao := NewFixedPriceItemDAO(db)

    item := &model.FixedPriceItem{
        LiveStreamID: 1001, ProductID: 5001, CreatorID: 100,
        Price: decimal.NewFromFloat(99.00), TotalStock: 100, RemainingStock: 100,
        MaxPerUser: 1, Status: model.FixedPriceStatusOnSale,
    }
    require.NoError(t, dao.Create(context.Background(), item))
    assert.NotZero(t, item.ID)

    got, err := dao.GetByID(context.Background(), item.ID)
    require.NoError(t, err)
    assert.Equal(t, "99.00", got.Price.StringFixed(2))
    assert.Equal(t, model.FixedPriceStatusOnSale, got.Status)
}

func TestFixedPriceItemDAO_UpdateStatus_LegalTransitions(t *testing.T) {
    db := setupTestDB(t)
    dao := NewFixedPriceItemDAO(db)
    item := &model.FixedPriceItem{
        LiveStreamID: 1001, ProductID: 5001, CreatorID: 100,
        Price: decimal.NewFromInt(99), TotalStock: 10, RemainingStock: 10,
        Status: model.FixedPriceStatusOnSale,
    }
    require.NoError(t, dao.Create(context.Background(), item))

    // on_sale -> sold_out OK
    require.NoError(t, dao.UpdateStatus(context.Background(), item.ID, model.FixedPriceStatusSoldOut))
    // sold_out -> on_sale 非法
    err := dao.UpdateStatus(context.Background(), item.ID, model.FixedPriceStatusOnSale)
    assert.ErrorIs(t, err, ErrIllegalStatusTransition)
}

func TestFixedPriceItemDAO_ListByLiveStreamID(t *testing.T) {
    db := setupTestDB(t)
    dao := NewFixedPriceItemDAO(db)
    for i := 0; i < 3; i++ {
        require.NoError(t, dao.Create(context.Background(), &model.FixedPriceItem{
            LiveStreamID: 2002, ProductID: int64(5000 + i), CreatorID: 100,
            Price: decimal.NewFromInt(10), TotalStock: 5, RemainingStock: 5,
            Status: model.FixedPriceStatusOnSale,
        }))
    }
    items, err := dao.ListByLiveStreamID(context.Background(), 2002, []model.FixedPriceStatus{model.FixedPriceStatusOnSale})
    require.NoError(t, err)
    assert.Len(t, items, 3)
}
```

- [ ] **Step 2.2: 跑测试，确认失败**

Run: `cd backend/auction && go test ./dao/ -run TestFixedPriceItemDAO -v`
Expected: FAIL（NewFixedPriceItemDAO undefined）

- [ ] **Step 2.3: 写最小实现**

```go
// backend/auction/dao/fixed_price_item.go
package dao

import (
    "context"
    "errors"
    "gorm.io/gorm"
    "your-module/backend/auction/model"
)

var ErrIllegalStatusTransition = errors.New("illegal fixed price status transition")

type FixedPriceItemDAO struct{ db *gorm.DB }

func NewFixedPriceItemDAO(db *gorm.DB) *FixedPriceItemDAO { return &FixedPriceItemDAO{db: db} }

func (d *FixedPriceItemDAO) Create(ctx context.Context, item *model.FixedPriceItem) error {
    return d.db.WithContext(ctx).Create(item).Error
}

func (d *FixedPriceItemDAO) GetByID(ctx context.Context, id int64) (*model.FixedPriceItem, error) {
    var it model.FixedPriceItem
    err := d.db.WithContext(ctx).First(&it, id).Error
    if err != nil { return nil, err }
    return &it, nil
}

var legalTransitions = map[model.FixedPriceStatus][]model.FixedPriceStatus{
    model.FixedPriceStatusOnSale:  {model.FixedPriceStatusSoldOut, model.FixedPriceStatusOffline},
    model.FixedPriceStatusSoldOut: {model.FixedPriceStatusOffline},
    model.FixedPriceStatusOffline: {},
}

func (d *FixedPriceItemDAO) UpdateStatus(ctx context.Context, id int64, to model.FixedPriceStatus) error {
    cur, err := d.GetByID(ctx, id)
    if err != nil { return err }
    allowed := legalTransitions[cur.Status]
    ok := false
    for _, s := range allowed { if s == to { ok = true; break } }
    if !ok { return ErrIllegalStatusTransition }
    return d.db.WithContext(ctx).Model(&model.FixedPriceItem{}).Where("id = ?", id).
        Update("status", to).Error
}

func (d *FixedPriceItemDAO) ListByLiveStreamID(ctx context.Context, liveStreamID int64, statuses []model.FixedPriceStatus) ([]*model.FixedPriceItem, error) {
    var out []*model.FixedPriceItem
    q := d.db.WithContext(ctx).Where("live_stream_id = ?", liveStreamID)
    if len(statuses) > 0 { q = q.Where("status IN ?", statuses) }
    err := q.Order("created_at DESC").Find(&out).Error
    return out, err
}

// DecrementRemainingStock 用于异步兜底刷写 Redis 真值到 DB
func (d *FixedPriceItemDAO) DecrementRemainingStock(ctx context.Context, id int64, newRemaining int) error {
    return d.db.WithContext(ctx).Model(&model.FixedPriceItem{}).Where("id = ?", id).
        Update("remaining_stock", newRemaining).Error
}
```

- [ ] **Step 2.4: 跑测试确认通过**

Run: `cd backend/auction && go test ./dao/ -run TestFixedPriceItemDAO -v`
Expected: PASS（3 cases）

- [ ] **Step 2.5: Commit**

```bash
git add backend/auction/dao/fixed_price_item*.go
git commit -m "feat(fixed-price): add FixedPriceItemDAO with status transition guard (M1.T2)"
```

---

### Task 3: dao 层 - FixedPricePurchase 唯一键兜底

**Files:**
- Test: `backend/auction/dao/fixed_price_purchase_test.go`
- Create: `backend/auction/dao/fixed_price_purchase.go`

- [ ] **Step 3.1: 写失败测试**

```go
// backend/auction/dao/fixed_price_purchase_test.go
package dao

import (
    "context"
    "testing"
    "github.com/shopspring/decimal"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "your-module/backend/auction/model"
)

func TestFixedPricePurchaseDAO_Insert_UniqueViolation(t *testing.T) {
    db := setupTestDB(t)
    dao := NewFixedPricePurchaseDAO(db)
    p1 := &model.FixedPricePurchase{ItemID: 7001, UserID: 100, OrderID: 88001, Price: decimal.NewFromInt(99)}
    require.NoError(t, dao.Insert(context.Background(), p1))

    p2 := &model.FixedPricePurchase{ItemID: 7001, UserID: 100, OrderID: 88002, Price: decimal.NewFromInt(99)}
    err := dao.Insert(context.Background(), p2)
    assert.ErrorIs(t, err, ErrAlreadyBought)
}

func TestFixedPricePurchaseDAO_GetByItemAndUser(t *testing.T) {
    db := setupTestDB(t)
    dao := NewFixedPricePurchaseDAO(db)
    require.NoError(t, dao.Insert(context.Background(), &model.FixedPricePurchase{
        ItemID: 7001, UserID: 100, OrderID: 88001, Price: decimal.NewFromInt(99),
    }))
    got, err := dao.GetByItemAndUser(context.Background(), 7001, 100)
    require.NoError(t, err)
    assert.Equal(t, int64(88001), got.OrderID)

    _, err = dao.GetByItemAndUser(context.Background(), 7001, 999)
    assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
```

- [ ] **Step 3.2: 跑测试确认失败**

Run: `cd backend/auction && go test ./dao/ -run TestFixedPricePurchaseDAO -v`
Expected: FAIL

- [ ] **Step 3.3: 写实现**

```go
// backend/auction/dao/fixed_price_purchase.go
package dao

import (
    "context"
    "errors"
    "strings"
    "gorm.io/gorm"
    "your-module/backend/auction/model"
)

var ErrAlreadyBought = errors.New("user already bought this fixed price item")

type FixedPricePurchaseDAO struct{ db *gorm.DB }

func NewFixedPricePurchaseDAO(db *gorm.DB) *FixedPricePurchaseDAO { return &FixedPricePurchaseDAO{db: db} }

// InsertWithTx 在外部事务内插入（service 层 Saga 用）
func (d *FixedPricePurchaseDAO) InsertWithTx(ctx context.Context, tx *gorm.DB, p *model.FixedPricePurchase) error {
    err := tx.WithContext(ctx).Create(p).Error
    if err != nil && isDuplicateKey(err) { return ErrAlreadyBought }
    return err
}

func (d *FixedPricePurchaseDAO) Insert(ctx context.Context, p *model.FixedPricePurchase) error {
    return d.InsertWithTx(ctx, d.db, p)
}

func (d *FixedPricePurchaseDAO) GetByItemAndUser(ctx context.Context, itemID, userID int64) (*model.FixedPricePurchase, error) {
    var p model.FixedPricePurchase
    err := d.db.WithContext(ctx).Where("item_id = ? AND user_id = ?", itemID, userID).First(&p).Error
    if err != nil { return nil, err }
    return &p, nil
}

func isDuplicateKey(err error) bool {
    return err != nil && strings.Contains(err.Error(), "Duplicate entry")
}
```

- [ ] **Step 3.4: 跑测试通过**

Run: `cd backend/auction && go test ./dao/ -run TestFixedPricePurchaseDAO -v`
Expected: PASS（2 cases）

- [ ] **Step 3.5: Commit**

```bash
git add backend/auction/dao/fixed_price_purchase*.go
git commit -m "feat(fixed-price): add FixedPricePurchaseDAO with unique key guard (M1.T3)"
```

---

### Task 4: Lua 脚本封装与原子库存抢占

**Files:**
- Test: `backend/auction/service/fixed_price_lua_test.go`
- Create: `backend/auction/service/fixed_price_lua.go`

- [ ] **Step 4.1: 写失败测试**

```go
// backend/auction/service/fixed_price_lua_test.go
package service

import (
    "context"
    "testing"
    "github.com/redis/go-redis/v9"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestStockGuard_Success(t *testing.T) {
    rdb := setupTestRedis(t)
    g := NewStockGuard(rdb)
    require.NoError(t, g.Init(context.Background(), 7001, 100))

    res, err := g.TryAcquire(context.Background(), 7001, 100)
    require.NoError(t, err)
    assert.Equal(t, StockResultSuccess, res)
}

func TestStockGuard_AlreadyBought(t *testing.T) {
    rdb := setupTestRedis(t)
    g := NewStockGuard(rdb)
    require.NoError(t, g.Init(context.Background(), 7001, 100))
    _, _ = g.TryAcquire(context.Background(), 7001, 100)

    res, err := g.TryAcquire(context.Background(), 7001, 100)
    require.NoError(t, err)
    assert.Equal(t, StockResultAlreadyBought, res)
}

func TestStockGuard_SoldOut(t *testing.T) {
    rdb := setupTestRedis(t)
    g := NewStockGuard(rdb)
    require.NoError(t, g.Init(context.Background(), 7001, 1))
    _, _ = g.TryAcquire(context.Background(), 7001, 100)
    res, err := g.TryAcquire(context.Background(), 7001, 200)
    require.NoError(t, err)
    assert.Equal(t, StockResultSoldOut, res)
}

func TestStockGuard_Uninitialized(t *testing.T) {
    rdb := setupTestRedis(t)
    g := NewStockGuard(rdb)
    res, err := g.TryAcquire(context.Background(), 9999, 100)
    require.NoError(t, err)
    assert.Equal(t, StockResultUninitialized, res)
}

func TestStockGuard_Compensate(t *testing.T) {
    rdb := setupTestRedis(t)
    g := NewStockGuard(rdb)
    require.NoError(t, g.Init(context.Background(), 7001, 5))
    _, _ = g.TryAcquire(context.Background(), 7001, 100)
    require.NoError(t, g.Compensate(context.Background(), 7001, 100))

    n, _ := rdb.Get(context.Background(), "fp:stock:7001").Int()
    assert.Equal(t, 5, n)
    isMember, _ := rdb.SIsMember(context.Background(), "fp:bought:7001", "100").Result()
    assert.False(t, isMember)
}
```

- [ ] **Step 4.2: 跑测试确认失败**

Run: `cd backend/auction && go test ./service/ -run TestStockGuard -v`
Expected: FAIL

- [ ] **Step 4.3: 写实现**

```go
// backend/auction/service/fixed_price_lua.go
package service

import (
    "context"
    _ "embed"
    "fmt"
    "github.com/redis/go-redis/v9"
)

type StockResult int

const (
    StockResultSuccess        StockResult = 1
    StockResultSoldOut        StockResult = -1
    StockResultAlreadyBought  StockResult = -2
    StockResultUninitialized  StockResult = -3
)

const acquireLuaScript = `
if redis.call('EXISTS', KEYS[1]) == 0 then return -3 end
if redis.call('SISMEMBER', KEYS[2], ARGV[1]) == 1 then return -2 end
local left = tonumber(redis.call('DECR', KEYS[1]))
if left < 0 then
  redis.call('INCR', KEYS[1])
  return -1
end
redis.call('SADD', KEYS[2], ARGV[1])
return 1
`

var acquireScript = redis.NewScript(acquireLuaScript)

type StockGuard struct{ rdb *redis.Client }

func NewStockGuard(rdb *redis.Client) *StockGuard { return &StockGuard{rdb: rdb} }

func stockKey(itemID int64) string  { return fmt.Sprintf("fp:stock:%d", itemID) }
func boughtKey(itemID int64) string { return fmt.Sprintf("fp:bought:%d", itemID) }

func (g *StockGuard) Init(ctx context.Context, itemID int64, total int) error {
    return g.rdb.Set(ctx, stockKey(itemID), total, 0).Err()
}

func (g *StockGuard) TryAcquire(ctx context.Context, itemID, userID int64) (StockResult, error) {
    res, err := acquireScript.Run(ctx, g.rdb, []string{stockKey(itemID), boughtKey(itemID)}, userID).Int64()
    if err != nil { return 0, err }
    return StockResult(res), nil
}

func (g *StockGuard) Compensate(ctx context.Context, itemID, userID int64) error {
    pipe := g.rdb.TxPipeline()
    pipe.Incr(ctx, stockKey(itemID))
    pipe.SRem(ctx, boughtKey(itemID), userID)
    _, err := pipe.Exec(ctx)
    return err
}

// Cleanup 下架/售罄 5s 后清理
func (g *StockGuard) Cleanup(ctx context.Context, itemID int64) error {
    return g.rdb.Del(ctx, stockKey(itemID), boughtKey(itemID)).Err()
}

func (g *StockGuard) Remaining(ctx context.Context, itemID int64) (int, error) {
    return g.rdb.Get(ctx, stockKey(itemID)).Int()
}
```

- [ ] **Step 4.4: 跑测试通过**

Run: `cd backend/auction && go test ./service/ -run TestStockGuard -v`
Expected: PASS（5 cases）

- [ ] **Step 4.5: Commit**

```bash
git add backend/auction/service/fixed_price_lua*.go
git commit -m "feat(fixed-price): add Lua-based atomic stock guard with compensation (M1.T4)"
```

---

### Task 5: 幂等存储

**Files:**
- Test: `backend/auction/service/fixed_price_idem_test.go`
- Create: `backend/auction/service/fixed_price_idem.go`

- [ ] **Step 5.1: 写失败测试**

```go
// backend/auction/service/fixed_price_idem_test.go
package service

import (
    "context"
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestIdemStore_FirstInsertReturnsZero(t *testing.T) {
    rdb := setupTestRedis(t)
    s := NewIdemStore(rdb)
    orderID, hit, err := s.GetOrInit(context.Background(), 100, 7001, "key-abc", 0)
    require.NoError(t, err)
    assert.False(t, hit)
    assert.Zero(t, orderID)
}

func TestIdemStore_SecondCallReturnsStoredOrderID(t *testing.T) {
    rdb := setupTestRedis(t)
    s := NewIdemStore(rdb)
    _, _, _ = s.GetOrInit(context.Background(), 100, 7001, "key-abc", 0)
    require.NoError(t, s.Persist(context.Background(), 100, 7001, "key-abc", 88001))

    orderID, hit, err := s.GetOrInit(context.Background(), 100, 7001, "key-abc", 0)
    require.NoError(t, err)
    assert.True(t, hit)
    assert.Equal(t, int64(88001), orderID)
}

func TestIdemStore_ValidateUUID(t *testing.T) {
    s := &IdemStore{}
    assert.False(t, s.IsValidKey("not-a-uuid"))
    assert.True(t, s.IsValidKey("550e8400-e29b-41d4-a716-446655440000"))
}

func TestIdemStore_TTL(t *testing.T) {
    rdb := setupTestRedis(t)
    s := NewIdemStore(rdb)
    _, _, _ = s.GetOrInit(context.Background(), 100, 7001, "key-ttl", 0)
    require.NoError(t, s.Persist(context.Background(), 100, 7001, "key-ttl", 88001))
    ttl, _ := rdb.TTL(context.Background(), "fp:idem:100:7001:key-ttl").Result()
    assert.True(t, ttl > 9*time.Minute && ttl <= 10*time.Minute)
}
```

- [ ] **Step 5.2: 跑测试确认失败**

Run: `cd backend/auction && go test ./service/ -run TestIdemStore -v`
Expected: FAIL

- [ ] **Step 5.3: 写实现**

```go
// backend/auction/service/fixed_price_idem.go
package service

import (
    "context"
    "fmt"
    "regexp"
    "strconv"
    "time"
    "github.com/redis/go-redis/v9"
)

const idemTTL = 10 * time.Minute

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type IdemStore struct{ rdb *redis.Client }

func NewIdemStore(rdb *redis.Client) *IdemStore { return &IdemStore{rdb: rdb} }

func (s *IdemStore) IsValidKey(k string) bool { return uuidRegex.MatchString(k) }

func idemRedisKey(userID, itemID int64, key string) string {
    return fmt.Sprintf("fp:idem:%d:%d:%s", userID, itemID, key)
}

// GetOrInit 命中返回 (orderID, true, nil)；未命中返回 (0, false, nil)
func (s *IdemStore) GetOrInit(ctx context.Context, userID, itemID int64, key string, _ int64) (int64, bool, error) {
    val, err := s.rdb.Get(ctx, idemRedisKey(userID, itemID, key)).Result()
    if err == redis.Nil { return 0, false, nil }
    if err != nil { return 0, false, err }
    n, _ := strconv.ParseInt(val, 10, 64)
    return n, true, nil
}

func (s *IdemStore) Persist(ctx context.Context, userID, itemID int64, key string, orderID int64) error {
    return s.rdb.Set(ctx, idemRedisKey(userID, itemID, key), orderID, idemTTL).Err()
}
```

- [ ] **Step 5.4: 跑测试通过**

Run: `cd backend/auction && go test ./service/ -run TestIdemStore -v`
Expected: PASS（4 cases）

- [ ] **Step 5.5: Commit**

```bash
git add backend/auction/service/fixed_price_idem*.go
git commit -m "feat(fixed-price): add idempotency store with UUID validation (M1.T5)"
```

---

### Task 6: service 层 - 上架接口（含主播校验、Redis 初始化）

**Files:**
- Test: `backend/auction/service/fixed_price_test.go`
- Create: `backend/auction/service/fixed_price.go`

- [ ] **Step 6.1: 写失败测试**

```go
// backend/auction/service/fixed_price_test.go (新建文件)
package service

import (
    "context"
    "testing"
    "github.com/shopspring/decimal"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "your-module/backend/auction/model"
)

func TestFixedPriceService_List_ValidatesAndCreates(t *testing.T) {
    svc := setupFixedPriceService(t)
    ctx := context.Background()
    req := ListItemReq{
        LiveStreamID: 1001, ProductID: 5001, CreatorID: 100,
        Price: decimal.NewFromFloat(99), TotalStock: 50, MaxPerUser: 1,
    }
    item, err := svc.ListItem(ctx, req)
    require.NoError(t, err)
    assert.Equal(t, int64(50), int64(item.RemainingStock))
    remain, _ := svc.stock.Remaining(ctx, item.ID)
    assert.Equal(t, 50, remain)
}

func TestFixedPriceService_List_RejectsInvalidPrice(t *testing.T) {
    svc := setupFixedPriceService(t)
    _, err := svc.ListItem(context.Background(), ListItemReq{
        LiveStreamID: 1, ProductID: 1, CreatorID: 1,
        Price: decimal.Zero, TotalStock: 10,
    })
    assert.ErrorIs(t, err, ErrInvalidParam)
}

func TestFixedPriceService_List_RejectsNonOwner(t *testing.T) {
    svc := setupFixedPriceServiceWithStream(t, 1001, 100) // owner=100
    _, err := svc.ListItem(context.Background(), ListItemReq{
        LiveStreamID: 1001, ProductID: 5001, CreatorID: 999, // not owner
        Price: decimal.NewFromInt(99), TotalStock: 10,
    })
    assert.ErrorIs(t, err, ErrNotStreamOwner)
}
```

- [ ] **Step 6.2: 跑测试确认失败**

Run: `cd backend/auction && go test ./service/ -run TestFixedPriceService_List -v`
Expected: FAIL

- [ ] **Step 6.3: 写实现**

```go
// backend/auction/service/fixed_price.go
package service

import (
    "context"
    "errors"
    "github.com/shopspring/decimal"
    "gorm.io/gorm"
    "your-module/backend/auction/dao"
    "your-module/backend/auction/model"
)

var (
    ErrInvalidParam     = errors.New("invalid param")
    ErrNotStreamOwner   = errors.New("not stream owner")
    ErrProductNotFound  = errors.New("product not found")
    ErrNotOnSale        = errors.New("not on sale")
    ErrSoldOut          = errors.New("sold out")
    ErrAlreadyBoughtSvc = errors.New("already bought")
    ErrInsufficient     = errors.New("insufficient balance")
)

type StreamOwnerChecker interface {
    IsOwner(ctx context.Context, liveStreamID, userID int64) (bool, error)
}

type ProductChecker interface {
    Exists(ctx context.Context, productID int64) (bool, error)
}

type FixedPriceService struct {
    db        *gorm.DB
    items     *dao.FixedPriceItemDAO
    purchases *dao.FixedPricePurchaseDAO
    stock     *StockGuard
    idem      *IdemStore
    streams   StreamOwnerChecker
    products  ProductChecker
    balance   BalanceDeducter      // 已有
    orders    OrderCreator         // 已有
    outbox    OutboxAppender       // 已有
}

type ListItemReq struct {
    LiveStreamID int64
    ProductID    int64
    CreatorID    int64
    Price        decimal.Decimal
    TotalStock   int
    MaxPerUser   int
}

func (s *FixedPriceService) ListItem(ctx context.Context, r ListItemReq) (*model.FixedPriceItem, error) {
    if r.Price.LessThanOrEqual(decimal.Zero) || r.TotalStock <= 0 || r.TotalStock > 10000 {
        return nil, ErrInvalidParam
    }
    isOwner, err := s.streams.IsOwner(ctx, r.LiveStreamID, r.CreatorID)
    if err != nil { return nil, err }
    if !isOwner { return nil, ErrNotStreamOwner }
    exists, err := s.products.Exists(ctx, r.ProductID)
    if err != nil { return nil, err }
    if !exists { return nil, ErrProductNotFound }
    if r.MaxPerUser <= 0 { r.MaxPerUser = 1 }

    item := &model.FixedPriceItem{
        LiveStreamID: r.LiveStreamID, ProductID: r.ProductID, CreatorID: r.CreatorID,
        Price: r.Price, TotalStock: r.TotalStock, RemainingStock: r.TotalStock,
        MaxPerUser: r.MaxPerUser, Status: model.FixedPriceStatusOnSale,
    }
    if err := s.items.Create(ctx, item); err != nil { return nil, err }
    if err := s.stock.Init(ctx, item.ID, r.TotalStock); err != nil { return nil, err }
    return item, nil
}
```

- [ ] **Step 6.4: 跑测试通过**

Run: `cd backend/auction && go test ./service/ -run TestFixedPriceService_List -v`
Expected: PASS（3 cases）

- [ ] **Step 6.5: Commit**

```bash
git add backend/auction/service/fixed_price*.go
git commit -m "feat(fixed-price): add ListItem service with owner+product validation (M1.T6)"
```

---

### Task 7: service 层 - 抢购接口（核心：Lua + DB Tx + Saga 补偿 + 幂等）

**Files:**
- Test: `backend/auction/service/fixed_price_test.go`（追加）
- Modify: `backend/auction/service/fixed_price.go`

- [ ] **Step 7.1: 追加 6 个失败测试**

```go
// 追加到 fixed_price_test.go

func TestPurchase_HappyPath(t *testing.T) {
    svc := setupFixedPriceService(t)
    item := setupItem(t, svc, 5, decimal.NewFromInt(99))
    setBalance(t, svc, 100, decimal.NewFromInt(1000))

    res, err := svc.Purchase(context.Background(), PurchaseReq{
        ItemID: item.ID, UserID: 100, IdemKey: "550e8400-e29b-41d4-a716-446655440000",
    })
    require.NoError(t, err)
    assert.NotZero(t, res.OrderID)
    assert.Equal(t, 4, res.RemainingStock)
}

func TestPurchase_SoldOut(t *testing.T) {
    svc := setupFixedPriceService(t)
    item := setupItem(t, svc, 1, decimal.NewFromInt(10))
    setBalance(t, svc, 100, decimal.NewFromInt(100))
    setBalance(t, svc, 200, decimal.NewFromInt(100))
    _, _ = svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
    _, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 200, IdemKey: newKey()})
    assert.ErrorIs(t, err, ErrSoldOut)
}

func TestPurchase_AlreadyBought(t *testing.T) {
    svc := setupFixedPriceService(t)
    item := setupItem(t, svc, 5, decimal.NewFromInt(10))
    setBalance(t, svc, 100, decimal.NewFromInt(100))
    _, _ = svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
    _, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
    assert.ErrorIs(t, err, ErrAlreadyBoughtSvc)
}

func TestPurchase_InsufficientBalance_TriggersCompensation(t *testing.T) {
    svc := setupFixedPriceService(t)
    item := setupItem(t, svc, 5, decimal.NewFromInt(99))
    setBalance(t, svc, 100, decimal.NewFromInt(50))

    _, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: newKey()})
    assert.ErrorIs(t, err, ErrInsufficient)
    remain, _ := svc.stock.Remaining(ctx, item.ID)
    assert.Equal(t, 5, remain) // 已补偿
    bought, _ := svc.stock.rdb.SIsMember(ctx, boughtKey(item.ID), 100).Result()
    assert.False(t, bought)
}

func TestPurchase_IdempotentReplay(t *testing.T) {
    svc := setupFixedPriceService(t)
    item := setupItem(t, svc, 5, decimal.NewFromInt(99))
    setBalance(t, svc, 100, decimal.NewFromInt(1000))
    key := "550e8400-e29b-41d4-a716-446655440001"

    res1, _ := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: key})
    res2, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: 100, IdemKey: key})
    require.NoError(t, err)
    assert.Equal(t, res1.OrderID, res2.OrderID)
    remain, _ := svc.stock.Remaining(ctx, item.ID)
    assert.Equal(t, 4, remain) // 仅扣一次
}

func TestPurchase_Concurrent_NoOversell(t *testing.T) {
    svc := setupFixedPriceService(t)
    item := setupItem(t, svc, 50, decimal.NewFromInt(1))
    var wg sync.WaitGroup
    var success int64
    for i := 0; i < 100; i++ {
        wg.Add(1)
        userID := int64(1000 + i)
        setBalance(t, svc, userID, decimal.NewFromInt(10))
        go func() {
            defer wg.Done()
            _, err := svc.Purchase(ctx, PurchaseReq{ItemID: item.ID, UserID: userID, IdemKey: newKey()})
            if err == nil { atomic.AddInt64(&success, 1) }
        }()
    }
    wg.Wait()
    assert.Equal(t, int64(50), success)
}
```

- [ ] **Step 7.2: 跑测试确认失败**

Run: `cd backend/auction && go test ./service/ -run TestPurchase -v`
Expected: FAIL（Purchase undefined）

- [ ] **Step 7.3: 写实现**

```go
// 追加到 backend/auction/service/fixed_price.go

type PurchaseReq struct {
    ItemID  int64
    UserID  int64
    IdemKey string
}

type PurchaseResult struct {
    OrderID        int64
    ItemID         int64
    Price          decimal.Decimal
    RemainingStock int
    Replayed       bool
}

func (s *FixedPriceService) Purchase(ctx context.Context, r PurchaseReq) (*PurchaseResult, error) {
    if !s.idem.IsValidKey(r.IdemKey) { return nil, ErrInvalidParam }

    // 1. 幂等命中检查
    if oid, hit, err := s.idem.GetOrInit(ctx, r.UserID, r.ItemID, r.IdemKey, 0); err != nil {
        return nil, err
    } else if hit {
        item, _ := s.items.GetByID(ctx, r.ItemID)
        rem, _ := s.stock.Remaining(ctx, r.ItemID)
        return &PurchaseResult{OrderID: oid, ItemID: r.ItemID, Price: item.Price, RemainingStock: rem, Replayed: true}, nil
    }

    // 2. 状态预检
    item, err := s.items.GetByID(ctx, r.ItemID)
    if err != nil { return nil, err }
    if item.Status != model.FixedPriceStatusOnSale { return nil, ErrNotOnSale }

    // 3. Lua 原子预扣
    res, err := s.stock.TryAcquire(ctx, r.ItemID, r.UserID)
    if err != nil { return nil, err }
    switch res {
    case StockResultUninitialized: return nil, ErrNotOnSale
    case StockResultSoldOut:       return nil, ErrSoldOut
    case StockResultAlreadyBought: return nil, ErrAlreadyBoughtSvc
    }

    // 4. DB 事务
    var orderID int64
    txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // 扣余额
        affected, e := s.balance.DeductWithTx(ctx, tx, r.UserID, item.Price)
        if e != nil { return e }
        if affected == 0 { return ErrInsufficient }
        // 创建订单
        oid, e := s.orders.CreateFixedPriceOrderTx(ctx, tx, r.UserID, item.ProductID, item.Price)
        if e != nil { return e }
        orderID = oid
        // 写 purchase
        if e := s.purchases.InsertWithTx(ctx, tx, &model.FixedPricePurchase{
            ItemID: item.ID, UserID: r.UserID, OrderID: oid, Price: item.Price,
        }); e != nil { return e }
        // outbox
        if e := s.outbox.AppendTx(ctx, tx, "fixed_price_sold", map[string]any{
            "item_id": item.ID, "user_id": r.UserID, "order_id": oid,
        }); e != nil { return e }
        return nil
    })

    if txErr != nil {
        // Saga 补偿
        _ = s.stock.Compensate(ctx, r.ItemID, r.UserID)
        return nil, txErr
    }

    // 持久化幂等
    _ = s.idem.Persist(ctx, r.UserID, r.ItemID, r.IdemKey, orderID)

    rem, _ := s.stock.Remaining(ctx, r.ItemID)
    if rem == 0 { _ = s.items.UpdateStatus(ctx, r.ItemID, model.FixedPriceStatusSoldOut) }

    return &PurchaseResult{OrderID: orderID, ItemID: r.ItemID, Price: item.Price, RemainingStock: rem}, nil
}
```

- [ ] **Step 7.4: 跑测试通过**

Run: `cd backend/auction && go test ./service/ -run TestPurchase -v -race`
Expected: PASS（6 cases，含并发零超卖 50/100）

- [ ] **Step 7.5: Commit**

```bash
git add backend/auction/service/fixed_price*.go
git commit -m "feat(fixed-price): implement Purchase with Lua+Tx+Saga+idempotency (M1.T7)"
```

---

### Task 8: service 层 - 下架（软标记 + 5s 异步清 Redis）

**Files:**
- Test: `backend/auction/service/fixed_price_test.go`（追加）
- Modify: `backend/auction/service/fixed_price.go`

- [ ] **Step 8.1: 写失败测试**

```go
func TestOffline_OwnerMarksOnly(t *testing.T) {
    svc := setupFixedPriceService(t)
    item := setupItem(t, svc, 10, decimal.NewFromInt(99))
    require.NoError(t, svc.Offline(ctx, item.ID, item.CreatorID))

    got, _ := svc.items.GetByID(ctx, item.ID)
    assert.Equal(t, model.FixedPriceStatusOffline, got.Status)
    rem, _ := svc.stock.Remaining(ctx, item.ID) // 立即查仍存在
    assert.Equal(t, 10, rem)
}

func TestOffline_NonOwner(t *testing.T) {
    svc := setupFixedPriceService(t)
    item := setupItem(t, svc, 10, decimal.NewFromInt(99))
    err := svc.Offline(ctx, item.ID, 9999)
    assert.ErrorIs(t, err, ErrNotStreamOwner)
}

func TestOffline_AsyncCleanupAfter5s(t *testing.T) {
    clk := newFakeClock()
    svc := setupFixedPriceServiceWithClock(t, clk)
    item := setupItem(t, svc, 10, decimal.NewFromInt(99))
    require.NoError(t, svc.Offline(ctx, item.ID, item.CreatorID))

    clk.Advance(6 * time.Second)
    eventually(t, func() bool {
        _, err := svc.stock.rdb.Get(ctx, stockKey(item.ID)).Result()
        return err == redis.Nil
    })
}
```

- [ ] **Step 8.2: 跑测试确认失败**

Run: `cd backend/auction && go test ./service/ -run TestOffline -v`
Expected: FAIL

- [ ] **Step 8.3: 写实现**

```go
// 追加到 fixed_price.go
const cleanupDelay = 5 * time.Second

func (s *FixedPriceService) Offline(ctx context.Context, itemID, userID int64) error {
    item, err := s.items.GetByID(ctx, itemID)
    if err != nil { return err }
    if item.CreatorID != userID { return ErrNotStreamOwner }
    if err := s.items.UpdateStatus(ctx, itemID, model.FixedPriceStatusOffline); err != nil {
        return err
    }
    s.scheduleCleanup(itemID)
    return nil
}

func (s *FixedPriceService) scheduleCleanup(itemID int64) {
    s.clk.AfterFunc(cleanupDelay, func() {
        bg := context.Background()
        _ = s.stock.Cleanup(bg, itemID)
    })
}
```

- [ ] **Step 8.4: 跑测试通过**

Run: `cd backend/auction && go test ./service/ -run TestOffline -v`
Expected: PASS（3 cases）

- [ ] **Step 8.5: Commit**

```bash
git add backend/auction/service/fixed_price*.go
git commit -m "feat(fixed-price): add Offline with soft-mark + 5s async cleanup (M1.T8)"
```

---

### Task 9: handler 层 - 抢购接口 + 错误码映射

**Files:**
- Test: `backend/auction/handler/fixed_price_test.go`
- Create: `backend/auction/handler/fixed_price.go`

- [ ] **Step 9.1: 写失败测试**

```go
// backend/auction/handler/fixed_price_test.go
package handler

import (
    "encoding/json"
    "net/http/httptest"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestPurchaseHandler_MissingIdemKey_400(t *testing.T) {
    h := setupHandler(t)
    req := httptest.NewRequest("POST", "/api/v1/fixed-price/items/7001/purchase", nil)
    req.Header.Set("X-User-ID", "100")
    w := httptest.NewRecorder()
    h.ServeHTTP(w, req)
    assert.Equal(t, 400, w.Code)
    var body errResp
    _ = json.Unmarshal(w.Body.Bytes(), &body)
    assert.Equal(t, "FP_INVALID_PARAM", body.Code)
}

func TestPurchaseHandler_InvalidIdemFormat_400(t *testing.T) {
    h := setupHandler(t)
    req := httptest.NewRequest("POST", "/api/v1/fixed-price/items/7001/purchase", nil)
    req.Header.Set("X-User-ID", "100")
    req.Header.Set("X-Idempotency-Key", "not-a-uuid")
    w := httptest.NewRecorder()
    h.ServeHTTP(w, req)
    assert.Equal(t, 400, w.Code)
}

func TestPurchaseHandler_InsufficientBalance_402_WithDetails(t *testing.T) {
    h := setupHandlerWithBalance(t, 100, decimal.NewFromInt(50))
    setupItemForHandler(t, h, 7001, 5, decimal.NewFromInt(99))
    req := httptest.NewRequest("POST", "/api/v1/fixed-price/items/7001/purchase", nil)
    req.Header.Set("X-User-ID", "100")
    req.Header.Set("X-Idempotency-Key", "550e8400-e29b-41d4-a716-446655440000")
    w := httptest.NewRecorder()
    h.ServeHTTP(w, req)
    assert.Equal(t, 402, w.Code)
    var body errResp
    _ = json.Unmarshal(w.Body.Bytes(), &body)
    assert.Equal(t, "FP_INSUFFICIENT_BALANCE", body.Code)
    assert.Equal(t, "99.00", body.Details["required"])
    assert.Equal(t, "50.00", body.Details["available"])
    assert.Equal(t, "49.00", body.Details["shortage"])
}

func TestPurchaseHandler_Success_PriceAsString(t *testing.T) {
    h := setupHandlerWithBalance(t, 100, decimal.NewFromInt(1000))
    setupItemForHandler(t, h, 7001, 5, decimal.NewFromInt(99))
    req := httptest.NewRequest("POST", "/api/v1/fixed-price/items/7001/purchase", nil)
    req.Header.Set("X-User-ID", "100")
    req.Header.Set("X-Idempotency-Key", "550e8400-e29b-41d4-a716-446655440001")
    w := httptest.NewRecorder()
    h.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code)
    var body map[string]any
    _ = json.Unmarshal(w.Body.Bytes(), &body)
    assert.Equal(t, "99.00", body["price"]) // string
    assert.Equal(t, float64(4), body["remaining_stock"])
}
```

- [ ] **Step 9.2: 跑测试确认失败**

Run: `cd backend/auction && go test ./handler/ -run TestPurchaseHandler -v`
Expected: FAIL

- [ ] **Step 9.3: 写实现**

```go
// backend/auction/handler/fixed_price.go
package handler

import (
    "context"
    "errors"
    "net/http"
    "strconv"
    "github.com/cloudwego/hertz/pkg/app"
    "your-module/backend/auction/service"
)

type FixedPriceHandler struct{ svc *service.FixedPriceService; balance BalanceQuerier }

type errResp struct {
    Code    string            `json:"code"`
    Message string            `json:"message"`
    Details map[string]string `json:"details,omitempty"`
}

func writeErr(c *app.RequestContext, status int, code, msg string, details map[string]string) {
    c.JSON(status, errResp{Code: code, Message: msg, Details: details})
}

func (h *FixedPriceHandler) Purchase(c context.Context, ctx *app.RequestContext) {
    itemIDStr := ctx.Param("id")
    itemID, err := strconv.ParseInt(itemIDStr, 10, 64)
    if err != nil { writeErr(ctx, 400, "FP_INVALID_PARAM", "item_id invalid", nil); return }

    userID, err := strconv.ParseInt(string(ctx.GetHeader("X-User-ID")), 10, 64)
    if err != nil { writeErr(ctx, 401, "FP_NOT_AUTHENTICATED", "missing user id", nil); return }

    idemKey := string(ctx.GetHeader("X-Idempotency-Key"))
    if idemKey == "" { writeErr(ctx, 400, "FP_INVALID_PARAM", "missing X-Idempotency-Key", nil); return }

    res, err := h.svc.Purchase(c, service.PurchaseReq{ItemID: itemID, UserID: userID, IdemKey: idemKey})
    switch {
    case err == nil:
        ctx.JSON(http.StatusOK, map[string]any{
            "order_id": res.OrderID, "item_id": res.ItemID,
            "price": res.Price.StringFixed(2), "remaining_stock": res.RemainingStock,
            "status": "success",
        })
    case errors.Is(err, service.ErrInvalidParam):
        writeErr(ctx, 400, "FP_INVALID_PARAM", err.Error(), nil)
    case errors.Is(err, service.ErrNotOnSale):
        writeErr(ctx, 409, "FP_NOT_ON_SALE", "item not on sale", nil)
    case errors.Is(err, service.ErrSoldOut):
        writeErr(ctx, 409, "FP_SOLD_OUT", "sold out", nil)
    case errors.Is(err, service.ErrAlreadyBoughtSvc):
        writeErr(ctx, 409, "FP_ALREADY_BOUGHT", "already bought", nil)
    case errors.Is(err, service.ErrInsufficient):
        avail, _ := h.balance.Get(c, userID)
        item, _ := h.svc.GetItem(c, itemID)
        shortage := item.Price.Sub(avail)
        writeErr(ctx, 402, "FP_INSUFFICIENT_BALANCE", "余额不足", map[string]string{
            "required": item.Price.StringFixed(2),
            "available": avail.StringFixed(2),
            "shortage": shortage.StringFixed(2),
        })
    default:
        writeErr(ctx, 500, "FP_INTERNAL", err.Error(), nil)
    }
}
```

- [ ] **Step 9.4: 跑测试通过**

Run: `cd backend/auction && go test ./handler/ -run TestPurchaseHandler -v`
Expected: PASS（4 cases）

- [ ] **Step 9.5: Commit**

```bash
git add backend/auction/handler/fixed_price*.go
git commit -m "feat(fixed-price): add Purchase HTTP handler with error code mapping (M1.T9)"
```

---

### Task 10: handler 层 - 上架/下架/详情/my-purchase + 路由

**Files:**
- Test: `backend/auction/handler/fixed_price_test.go`（追加）
- Modify: `backend/auction/handler/fixed_price.go`
- Create: `backend/auction/handler/fixed_price_http.go`
- Modify: `backend/auction/handler/router.go`

- [ ] **Step 10.1: 追加测试 - 上架/下架/详情**

```go
func TestListItemHandler_Success(t *testing.T) {
    h := setupHandler(t)
    body := `{"live_stream_id":1001,"product_id":5001,"price":"99.00","total_stock":100,"max_per_user":1}`
    req := httptest.NewRequest("POST", "/api/v1/fixed-price/items", strings.NewReader(body))
    req.Header.Set("X-User-ID", "100")
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    h.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code)
}

func TestOfflineHandler_NonOwner_403(t *testing.T) {
    h := setupHandler(t)
    item := setupItemForHandler(t, h, 7001, 10, decimal.NewFromInt(99))
    req := httptest.NewRequest("POST", "/api/v1/fixed-price/items/"+strconv.FormatInt(item.ID, 10)+"/offline", nil)
    req.Header.Set("X-User-ID", "9999")
    w := httptest.NewRecorder()
    h.ServeHTTP(w, req)
    assert.Equal(t, 403, w.Code)
}

func TestMyPurchaseHandler(t *testing.T) {
    h := setupHandlerWithBalance(t, 100, decimal.NewFromInt(1000))
    item := setupItemForHandler(t, h, 7001, 5, decimal.NewFromInt(99))
    purchaseAs(t, h, 100, item.ID)
    req := httptest.NewRequest("GET", "/api/v1/fixed-price/items/"+strconv.FormatInt(item.ID, 10)+"/my-purchase", nil)
    req.Header.Set("X-User-ID", "100")
    w := httptest.NewRecorder()
    h.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code)
}
```

- [ ] **Step 10.2: 跑测试确认失败**

Run: `cd backend/auction && go test ./handler/ -run "TestListItemHandler|TestOfflineHandler|TestMyPurchaseHandler" -v`
Expected: FAIL

- [ ] **Step 10.3: 实现 handler 与路由**

```go
// backend/auction/handler/fixed_price.go (追加)

type listItemBody struct {
    LiveStreamID int64  `json:"live_stream_id"`
    ProductID    int64  `json:"product_id"`
    Price        string `json:"price"`
    TotalStock   int    `json:"total_stock"`
    MaxPerUser   int    `json:"max_per_user"`
}

func (h *FixedPriceHandler) List(c context.Context, ctx *app.RequestContext) {
    var body listItemBody
    if err := ctx.BindJSON(&body); err != nil { writeErr(ctx, 400, "FP_INVALID_PARAM", err.Error(), nil); return }
    userID, _ := strconv.ParseInt(string(ctx.GetHeader("X-User-ID")), 10, 64)
    price, err := decimal.NewFromString(body.Price)
    if err != nil { writeErr(ctx, 400, "FP_INVALID_PARAM", "price format", nil); return }

    item, err := h.svc.ListItem(c, service.ListItemReq{
        LiveStreamID: body.LiveStreamID, ProductID: body.ProductID, CreatorID: userID,
        Price: price, TotalStock: body.TotalStock, MaxPerUser: body.MaxPerUser,
    })
    switch {
    case err == nil:
        ctx.JSON(200, map[string]any{
            "id": item.ID, "status": "on_sale", "remaining_stock": item.RemainingStock,
            "created_at": item.CreatedAt,
        })
    case errors.Is(err, service.ErrInvalidParam): writeErr(ctx, 400, "FP_INVALID_PARAM", err.Error(), nil)
    case errors.Is(err, service.ErrNotStreamOwner): writeErr(ctx, 403, "FP_NOT_STREAM_OWNER", err.Error(), nil)
    case errors.Is(err, service.ErrProductNotFound): writeErr(ctx, 404, "FP_PRODUCT_NOT_FOUND", err.Error(), nil)
    default: writeErr(ctx, 500, "FP_INTERNAL", err.Error(), nil)
    }
}

func (h *FixedPriceHandler) Offline(c context.Context, ctx *app.RequestContext) {
    itemID, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
    userID, _ := strconv.ParseInt(string(ctx.GetHeader("X-User-ID")), 10, 64)
    err := h.svc.Offline(c, itemID, userID)
    switch {
    case err == nil: ctx.JSON(200, map[string]any{"status": "offline"})
    case errors.Is(err, service.ErrNotStreamOwner): writeErr(ctx, 403, "FP_NOT_STREAM_OWNER", err.Error(), nil)
    default: writeErr(ctx, 500, "FP_INTERNAL", err.Error(), nil)
    }
}

func (h *FixedPriceHandler) Detail(c context.Context, ctx *app.RequestContext) {
    itemID, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
    item, err := h.svc.GetItem(c, itemID)
    if err != nil { writeErr(ctx, 404, "FP_NOT_FOUND", err.Error(), nil); return }
    rem, _ := h.svc.RemainingStock(c, itemID)
    ctx.JSON(200, map[string]any{
        "id": item.ID, "live_stream_id": item.LiveStreamID, "product_id": item.ProductID,
        "price": item.Price.StringFixed(2), "total_stock": item.TotalStock,
        "remaining_stock": rem, "max_per_user": item.MaxPerUser,
        "status": statusString(item.Status),
    })
}

func (h *FixedPriceHandler) MyPurchase(c context.Context, ctx *app.RequestContext) {
    itemID, _ := strconv.ParseInt(ctx.Param("id"), 10, 64)
    userID, _ := strconv.ParseInt(string(ctx.GetHeader("X-User-ID")), 10, 64)
    p, err := h.svc.GetMyPurchase(c, itemID, userID)
    if err != nil { ctx.JSON(200, map[string]any{"i_bought": false}); return }
    ctx.JSON(200, map[string]any{"i_bought": true, "order_id": p.OrderID})
}
```

```go
// backend/auction/handler/fixed_price_http.go
package handler

import "github.com/cloudwego/hertz/pkg/route"

func RegisterFixedPriceRoutes(g *route.RouterGroup, h *FixedPriceHandler) {
    fp := g.Group("/fixed-price")
    fp.POST("/items", h.List)
    fp.POST("/items/:id/offline", h.Offline)
    fp.GET("/items/:id", h.Detail)
    fp.POST("/items/:id/purchase", h.Purchase)
    fp.GET("/items/:id/my-purchase", h.MyPurchase)
}
```

- [ ] **Step 10.4: 在 router.go 挂载**

```go
// backend/auction/handler/router.go (追加调用)
RegisterFixedPriceRoutes(v1, fixedPriceHandler)
```

- [ ] **Step 10.5: 跑测试通过**

Run: `cd backend/auction && go test ./handler/ -run "FixedPrice|ListItemHandler|OfflineHandler|MyPurchaseHandler" -v`
Expected: PASS

- [ ] **Step 10.6: Commit**

```bash
git add backend/auction/handler/fixed_price*.go backend/auction/handler/router.go
git commit -m "feat(fixed-price): wire handlers and routes for fixed price endpoints (M1.T10)"
```

---

### Task 11: gateway-service 转发路由

**Files:**
- Modify: `backend/gateway/router.go`

- [ ] **Step 11.1: 加转发**

```go
// backend/gateway/router.go (在 v1 group 内追加)
v1.Any("/fixed-price/*any", proxyTo(auctionService))
v1.GET("/live-streams/:id/fixed-price/items", aggregateFixedPriceList) // M2 实现
```

- [ ] **Step 11.2: 写转发集成测试**

```go
// backend/gateway/fixed_price_proxy_test.go
func TestGateway_ForwardsPurchaseToAuction(t *testing.T) {
    auctionStub := startAuctionStub(t)
    defer auctionStub.Close()
    gw := setupGatewayWithAuction(t, auctionStub.URL)
    req := httptest.NewRequest("POST", "/api/v1/fixed-price/items/7001/purchase", nil)
    req.Header.Set("X-User-ID", "100")
    req.Header.Set("X-Idempotency-Key", "550e8400-e29b-41d4-a716-446655440000")
    w := httptest.NewRecorder()
    gw.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code)
}
```

- [ ] **Step 11.3: 跑测试通过**

Run: `cd backend/gateway && go test ./... -run TestGateway_Forwards -v`
Expected: PASS

- [ ] **Step 11.4: Commit**

```bash
git add backend/gateway/router.go backend/gateway/fixed_price_proxy_test.go
git commit -m "feat(fixed-price): gateway forwards /api/v1/fixed-price/* to auction-service (M1.T11)"
```

---

### Task 12: Toxiproxy 集成测试 - 网络异常补偿

**Files:**
- Test: `backend/auction/integration/fixed_price_toxic_test.go`

- [ ] **Step 12.1: 写测试**

```go
// backend/auction/integration/fixed_price_toxic_test.go
func TestPurchase_NetworkRetryWithSameIdemKey_DeductsOnce(t *testing.T) {
    setupIntegration(t)
    item := createItem(t, 5, decimal.NewFromInt(99))
    setUserBalance(t, 100, decimal.NewFromInt(1000))
    key := "550e8400-e29b-41d4-a716-446655440002"

    res1, err1 := purchaseHTTP(t, item.ID, 100, key)
    require.NoError(t, err1)
    res2, err2 := purchaseHTTP(t, item.ID, 100, key)
    require.NoError(t, err2)
    assert.Equal(t, res1.OrderID, res2.OrderID)

    rem := getRemaining(t, item.ID)
    assert.Equal(t, 4, rem) // 仅扣一次
}

func TestPurchase_RedisDown_FailFast(t *testing.T) {
    setupIntegration(t)
    item := createItem(t, 5, decimal.NewFromInt(99))
    proxy := blockRedis(t, 1*time.Second)
    defer proxy.Restore()

    _, err := purchaseHTTP(t, item.ID, 100, newKey())
    assert.Error(t, err)
    assertHTTPStatus(t, err, 503)
}
```

- [ ] **Step 12.2: 启动 docker-compose 测试栈并跑**

Run:
```
cd backend && docker-compose -f docker-compose.test.yml up -d
cd auction && go test -tags=integration ./integration/ -run TestPurchase_Network -v
```
Expected: PASS（2 cases）

- [ ] **Step 12.3: Commit**

```bash
git add backend/auction/integration/fixed_price_toxic_test.go
git commit -m "test(fixed-price): toxiproxy integration for retry idempotency and redis fail-fast (M1.T12)"
```

---

### Task 13: 端到端冒烟 + 拍卖回归

**Files:**
- 仅运行测试，无新代码

- [ ] **Step 13.1: 全量单元测试**

Run: `cd backend && go test ./... -race`
Expected: 全部 PASS（含原拍卖测试）

- [ ] **Step 13.2: 现有 e2e 演示回归**

Run: 启动现有 test-dashboard，跑 `/test/e2e` 与 `/test/antisnipe` 各一次。
Expected: 与 baseline 一致，拍卖链路无回归

- [ ] **Step 13.3: 手动 curl 冒烟**

```bash
# 上架
curl -X POST http://localhost:8080/api/v1/fixed-price/items \
  -H "Authorization: Bearer <jwt>" \
  -H "Content-Type: application/json" \
  -d '{"live_stream_id":1001,"product_id":5001,"price":"99.00","total_stock":3,"max_per_user":1}'

# 抢购
curl -X POST http://localhost:8080/api/v1/fixed-price/items/<item_id>/purchase \
  -H "Authorization: Bearer <jwt>" \
  -H "X-Idempotency-Key: $(uuidgen)"

# 重复抢购
curl ... # 返回 FP_ALREADY_BOUGHT
```

- [ ] **Step 13.4: 提交 M1 完成 tag**

```bash
git tag fixed-price-m1-complete
git commit --allow-empty -m "chore(fixed-price): M1 backend acceptance (M1.T13)"
```

---

## M1 验收标准

- ✅ 所有 13 个 task 单元 + 集成测试 PASS
- ✅ 并发 100 抢 50 件零超卖（T2.8）
- ✅ 网络重试同 idem key 仅扣一次（T12.1）
- ✅ 拍卖侧 `/test/e2e`、`/test/antisnipe` 演示无回归
- ✅ Saga 补偿可观测（compensate 调用计数 > 0 在余额不足测试中）

**下一步：** M2（实时同步层）需在 M1 完成且 B1 plan 完成后启动。

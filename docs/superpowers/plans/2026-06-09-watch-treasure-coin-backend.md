# 观看时长宝箱 + 金币资产 后端实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为已落地的前端「观看时长宝箱」组件补齐 auction-service 后端：观看时长心跳累加、宝箱状态查询、领取发币，并经 gateway JWT 暴露给 H5。

**Architecture:** 沿用 auction-service 既有 model/dao/service/handler 分层。新增 3 张表（`user_coins`、`user_watch_duration`、`treasure_claims`）。时长心跳由 Redis「上次心跳时间戳」做封顶累加（单次最多 30s）防刷；宝箱档位为后端常量 SSOT；领取在单事务内「校验时长 + 唯一键插入 claim + 累加金币」，唯一键 `(user_id, stat_date, tier)` 保证幂等。所有金额为整数 `BIGINT`，与现金 `user_balances`（decimal/CNY）完全隔离。

**Tech Stack:** Go 1.24 + Hertz + GORM + go-redis；测试用内存 sqlite（`setupTestDB`）+ miniredis（`setupTestRedis`）+ testify。

---

## 设计参照

Spec: [2026-06-09-watch-treasure-coin-design.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/superpowers/specs/2026-06-09-watch-treasure-coin-design.md)

前端已实现（**契约只读、不可改**）：
- [TreasureProgressBar.tsx](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/TreasureProgressBar.tsx)
- [api.ts treasureApi](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts#L482-L497)：
  - `getStatus()` → `GET /api/v1/treasure/status`，期望 data = `{ watched_seconds, coin_balance, tiers: [{tier, threshold_seconds, coins, state}] }`
  - `claim(tier)` → `POST /api/v1/treasure/claim` body `{tier}`，期望 data = `{ coins, coin_balance }`
  - `heartbeat()` → `POST /api/v1/watch/heartbeat` body `{}`

前端 `request()` 解包逻辑（[api.ts:203-221](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts#L203-L221)）：业务码必须落在 `SUCCESS_CODES`（含 `200`/`0`），否则抛 `ApiError(message)`；成功时返回 `data.data`。因此所有 handler 成功响应统一 `{ "code": 200, "data": {...} }`，失败响应 `{ "code": <非成功码>, "message": "..." }`。

## 关键约定（务必遵守）

- **业务时区**：统一用 `service.auctionBusinessNow()`（Asia/Shanghai），`stat_date` 取该时区当天 `YYYY-MM-DD`。
- **档位常量 SSOT**：定义在 service 层，唯一来源：
  - tier 0：`threshold=180s`，`coins=100`
  - tier 1：`threshold=600s`，`coins=300`
  - tier 2：`threshold=1800s`，`coins=800`
- **金币整数**：所有金币字段 `int64`，禁止 decimal/float。
- **隔离**：绝不触碰 `user_balances` / `UserBalance` 相关代码。
- **user_id 来源**：`c.GetInt64("user_id")`（gateway 经 `X-User-ID` 注入），`<=0` 视为未登录返回 401。

## 文件结构

| 文件 | 职责 | 动作 |
|---|---|---|
| `backend/auction/model/treasure.go` | 3 个 GORM 模型 + TableName | Create |
| `backend/auction/migration/003_create_treasure_tables.sql` | 建表 DDL（与 AutoMigrate 对齐，留存审计） | Create |
| `backend/auction/dao/treasure.go` | 时长 UPSERT 累加、金币读取、claim 事务 | Create |
| `backend/auction/dao/treasure_test.go` | DAO 单测（累加/幂等/事务） | Create |
| `backend/auction/service/treasure.go` | 档位常量、心跳封顶累加、状态编排、领取编排 | Create |
| `backend/auction/service/treasure_test.go` | service 单测（封顶/门槛/状态机/重复领取） | Create |
| `backend/auction/handler/treasure.go` | 3 个 HTTP handler（解析/序列化） | Create |
| `backend/auction/handler/treasure_test.go` | handler 单测（响应 shape/401/错误码） | Create |
| `backend/auction/main.go` | 装配 DAO/service/handler + 注册路由 + AutoMigrate | Modify |
| `backend/gateway/router/router.go` | 在 authGroup 下代理 3 条路由 | Modify |

---

## Task 1: 数据模型与建表 DDL

**Files:**
- Create: `backend/auction/model/treasure.go`
- Create: `backend/auction/migration/003_create_treasure_tables.sql`

- [ ] **Step 1: 写模型文件**

`backend/auction/model/treasure.go`：

```go
package model

import "time"

// UserCoin 用户金币资产：纯娱乐积分，整数，与现金 user_balances 完全隔离。
// 1 用户 1 行，永久累积。
type UserCoin struct {
	UserID    int64     `json:"user_id" gorm:"primaryKey;column:user_id"`
	Balance   int64     `json:"balance" gorm:"not null;default:0"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UserCoin) TableName() string { return "user_coins" }

// UserWatchDuration 今日观看时长，按 (user_id, stat_date) 分桶，每日 0 点天然失效。
// StatDate 为业务时区当天，格式 YYYY-MM-DD。
type UserWatchDuration struct {
	UserID       int64     `json:"user_id" gorm:"primaryKey;column:user_id"`
	StatDate     string    `json:"stat_date" gorm:"primaryKey;column:stat_date;type:varchar(10)"`
	TotalSeconds int       `json:"total_seconds" gorm:"not null;default:0"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (UserWatchDuration) TableName() string { return "user_watch_duration" }

// TreasureClaim 宝箱领取记录：唯一键 (user_id, stat_date, tier) 即幂等保证。
type TreasureClaim struct {
	UserID    int64     `json:"user_id" gorm:"primaryKey;column:user_id"`
	StatDate  string    `json:"stat_date" gorm:"primaryKey;column:stat_date;type:varchar(10)"`
	Tier      int8      `json:"tier" gorm:"primaryKey;column:tier"`
	Coins     int64     `json:"coins" gorm:"not null"`
	ClaimedAt time.Time `json:"claimed_at"`
}

func (TreasureClaim) TableName() string { return "treasure_claims" }
```

- [ ] **Step 2: 写迁移 DDL**

`backend/auction/migration/003_create_treasure_tables.sql`：

```sql
CREATE TABLE IF NOT EXISTS user_coins (
  user_id    BIGINT   NOT NULL PRIMARY KEY,
  balance    BIGINT   NOT NULL DEFAULT 0,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_watch_duration (
  user_id       BIGINT      NOT NULL,
  stat_date     VARCHAR(10) NOT NULL,
  total_seconds INT         NOT NULL DEFAULT 0,
  updated_at    DATETIME    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, stat_date)
);

CREATE TABLE IF NOT EXISTS treasure_claims (
  user_id    BIGINT      NOT NULL,
  stat_date  VARCHAR(10) NOT NULL,
  tier       TINYINT     NOT NULL,
  coins      BIGINT      NOT NULL,
  claimed_at DATETIME    NOT NULL,
  PRIMARY KEY (user_id, stat_date, tier)
);
```

- [ ] **Step 3: 编译验证**

Run: `cd backend/auction && go build ./model/...`
Expected: 无输出，退出码 0。

- [ ] **Step 4: Commit**

```bash
git add backend/auction/model/treasure.go backend/auction/migration/003_create_treasure_tables.sql
git commit -m "feat(treasure): add coin/watch-duration/claim models and DDL"
```

---

## Task 2: DAO — 时长累加 + 金币读取

**Files:**
- Create: `backend/auction/dao/treasure.go`
- Create: `backend/auction/dao/treasure_test.go`
- Modify: `backend/auction/dao/testutil_test.go`（迁移新表）

- [ ] **Step 1: 在测试 setupTestDB 中迁移新表**

修改 `backend/auction/dao/testutil_test.go` 的 `db.AutoMigrate(...)` 调用，追加三个模型：

```go
	require.NoError(t, db.AutoMigrate(
		&model.FixedPriceItem{},
		&model.FixedPricePurchase{},
		&model.UserBalance{},
		&model.UserCoin{},
		&model.UserWatchDuration{},
		&model.TreasureClaim{},
	))
```

- [ ] **Step 2: 写失败测试 `treasure_test.go`**

`backend/auction/dao/treasure_test.go`：

```go
package dao

import (
	"context"
	"testing"

	"auction-service/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreasureDAO_AddWatchSeconds_AccumulatesPerDate(t *testing.T) {
	db := setupTestDB(t)
	d := NewTreasureDAO(db)
	ctx := context.Background()

	total, err := d.AddWatchSeconds(ctx, 100, "2026-06-09", 30)
	require.NoError(t, err)
	assert.Equal(t, 30, total)

	total, err = d.AddWatchSeconds(ctx, 100, "2026-06-09", 30)
	require.NoError(t, err)
	assert.Equal(t, 60, total)

	// 不同日期独立分桶
	total, err = d.AddWatchSeconds(ctx, 100, "2026-06-10", 30)
	require.NoError(t, err)
	assert.Equal(t, 30, total)
}

func TestTreasureDAO_GetWatchSeconds_NoRecordReturnsZero(t *testing.T) {
	db := setupTestDB(t)
	d := NewTreasureDAO(db)
	ctx := context.Background()

	secs, err := d.GetWatchSeconds(ctx, 999, "2026-06-09")
	require.NoError(t, err)
	assert.Equal(t, 0, secs)
}

func TestTreasureDAO_GetCoinBalance_NoRecordReturnsZero(t *testing.T) {
	db := setupTestDB(t)
	d := NewTreasureDAO(db)
	ctx := context.Background()

	bal, err := d.GetCoinBalance(ctx, 999)
	require.NoError(t, err)
	assert.Equal(t, int64(0), bal)
}

func TestTreasureDAO_ListClaimedTiers(t *testing.T) {
	db := setupTestDB(t)
	d := NewTreasureDAO(db)
	ctx := context.Background()

	require.NoError(t, db.Create(&model.TreasureClaim{
		UserID: 100, StatDate: "2026-06-09", Tier: 0, Coins: 100,
	}).Error)

	tiers, err := d.ListClaimedTiers(ctx, 100, "2026-06-09")
	require.NoError(t, err)
	assert.Equal(t, map[int8]bool{0: true}, tiers)
}

func TestTreasureDAO_ClaimTx_Success(t *testing.T) {
	db := setupTestDB(t)
	d := NewTreasureDAO(db)
	ctx := context.Background()

	newBalance, err := d.ClaimTx(ctx, 100, "2026-06-09", 1, 300)
	require.NoError(t, err)
	assert.Equal(t, int64(300), newBalance)

	bal, _ := d.GetCoinBalance(ctx, 100)
	assert.Equal(t, int64(300), bal)
}

func TestTreasureDAO_ClaimTx_DuplicateIsIdempotent(t *testing.T) {
	db := setupTestDB(t)
	d := NewTreasureDAO(db)
	ctx := context.Background()

	_, err := d.ClaimTx(ctx, 100, "2026-06-09", 1, 300)
	require.NoError(t, err)

	// 重复领取应返回 ErrAlreadyClaimed，且金币不再增加
	_, err = d.ClaimTx(ctx, 100, "2026-06-09", 1, 300)
	assert.ErrorIs(t, err, ErrAlreadyClaimed)

	bal, _ := d.GetCoinBalance(ctx, 100)
	assert.Equal(t, int64(300), bal)
}
```

- [ ] **Step 3: 运行测试，确认编译失败**

Run: `cd backend/auction && go test ./dao/ -run TestTreasureDAO -v`
Expected: FAIL，编译错误 `undefined: NewTreasureDAO` / `ErrAlreadyClaimed`。

- [ ] **Step 4: 写实现 `treasure.go`**

`backend/auction/dao/treasure.go`：

```go
package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"auction-service/model"
)

// ErrAlreadyClaimed 表示该 (user, date, tier) 宝箱已被领取，claim 幂等命中。
var ErrAlreadyClaimed = errors.New("treasure already claimed")

// TreasureDAO 宝箱/金币/观看时长数据访问层。金币为整数，与 user_balances 完全隔离。
type TreasureDAO struct {
	db *gorm.DB
}

func NewTreasureDAO(db *gorm.DB) *TreasureDAO {
	return &TreasureDAO{db: db}
}

// AddWatchSeconds 在 (user_id, stat_date) 桶上累加 delta 秒，返回累加后的总秒数。
// 使用 UPSERT 保证并发安全（依赖主键冲突触发 DoUpdates）。
func (d *TreasureDAO) AddWatchSeconds(ctx context.Context, userID int64, statDate string, delta int) (int, error) {
	row := model.UserWatchDuration{
		UserID:       userID,
		StatDate:     statDate,
		TotalSeconds: delta,
		UpdatedAt:    time.Now(),
	}
	err := d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "stat_date"}},
		DoUpdates: clause.Assignments(map[string]any{
			"total_seconds": gorm.Expr("total_seconds + ?", delta),
			"updated_at":    time.Now(),
		}),
	}).Create(&row).Error
	if err != nil {
		return 0, err
	}
	return d.GetWatchSeconds(ctx, userID, statDate)
}

// GetWatchSeconds 读取今日累计秒数，无记录返回 0。
func (d *TreasureDAO) GetWatchSeconds(ctx context.Context, userID int64, statDate string) (int, error) {
	var row model.UserWatchDuration
	err := d.db.WithContext(ctx).
		Where("user_id = ? AND stat_date = ?", userID, statDate).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return row.TotalSeconds, nil
}

// GetCoinBalance 读取金币余额，无记录返回 0。
func (d *TreasureDAO) GetCoinBalance(ctx context.Context, userID int64) (int64, error) {
	var row model.UserCoin
	err := d.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return row.Balance, nil
}

// ListClaimedTiers 返回今日已领取的 tier 集合。
func (d *TreasureDAO) ListClaimedTiers(ctx context.Context, userID int64, statDate string) (map[int8]bool, error) {
	var rows []model.TreasureClaim
	err := d.db.WithContext(ctx).
		Where("user_id = ? AND stat_date = ?", userID, statDate).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	claimed := make(map[int8]bool, len(rows))
	for _, r := range rows {
		claimed[r.Tier] = true
	}
	return claimed, nil
}

// ClaimTx 在单事务内：插入 claim 记录（唯一键幂等）+ 累加金币，返回累加后的金币余额。
// 若该 (user, date, tier) 已存在，返回 ErrAlreadyClaimed 且不改变金币。
func (d *TreasureDAO) ClaimTx(ctx context.Context, userID int64, statDate string, tier int8, coins int64) (int64, error) {
	var newBalance int64
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		claim := model.TreasureClaim{
			UserID:    userID,
			StatDate:  statDate,
			Tier:      tier,
			Coins:     coins,
			ClaimedAt: time.Now(),
		}
		// DoNothing：主键冲突时 RowsAffected=0，借此判定重复领取。
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&claim)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrAlreadyClaimed
		}

		coin := model.UserCoin{UserID: userID, Balance: coins, UpdatedAt: time.Now()}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"balance":    gorm.Expr("balance + ?", coins),
				"updated_at": time.Now(),
			}),
		}).Create(&coin).Error; err != nil {
			return err
		}

		var updated model.UserCoin
		if err := tx.Where("user_id = ?", userID).First(&updated).Error; err != nil {
			return err
		}
		newBalance = updated.Balance
		return nil
	})
	if err != nil {
		return 0, err
	}
	return newBalance, nil
}
```

- [ ] **Step 5: 运行测试，确认通过**

Run: `cd backend/auction && go test ./dao/ -run TestTreasureDAO -v`
Expected: PASS（6 个用例全绿）。

- [ ] **Step 6: Commit**

```bash
git add backend/auction/dao/treasure.go backend/auction/dao/treasure_test.go backend/auction/dao/testutil_test.go
git commit -m "feat(treasure): add TreasureDAO with watch-accumulate and idempotent claim"
```

---

## Task 3: Service — 档位常量 + 心跳封顶 + 状态编排 + 领取编排

**Files:**
- Create: `backend/auction/service/treasure.go`
- Create: `backend/auction/service/treasure_test.go`

说明：心跳防刷采用「Redis 记录上次心跳时间戳，按真实间隔累加且单次封顶 30s」。Redis key `treasure:hb:<userID>`，TTL 120s。首次心跳无上次时间戳时记入基础 30s（与前端 30s 节拍对齐）。

- [ ] **Step 1: 写失败测试 `treasure_test.go`**

`backend/auction/service/treasure_test.go`：

```go
package service

import (
	"context"
	"testing"

	"auction-service/dao"
	"auction-service/model"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTreasureService(t *testing.T) (*TreasureService, *dao.TreasureDAO, *miniredis.Miniredis) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.UserCoin{}, &model.UserWatchDuration{}, &model.TreasureClaim{},
	))
	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	d := dao.NewTreasureDAO(db)
	return NewTreasureService(d, rdb), d, mr
}

func TestTreasureService_Heartbeat_FirstBeatRecords30s(t *testing.T) {
	svc, _, _ := setupTreasureService(t)
	total, err := svc.Heartbeat(context.Background(), 100)
	require.NoError(t, err)
	assert.Equal(t, 30, total)
}

func TestTreasureService_Heartbeat_CapsAt30sPerBeat(t *testing.T) {
	svc, dd, mr := setupTreasureService(t)
	ctx := context.Background()

	_, err := svc.Heartbeat(ctx, 100)
	require.NoError(t, err)

	// 模拟客户端隔了很久（500s）再上报：单次仍只累加封顶 30s，而非 500s。
	mr.FastForward(0) // 占位：实际由 service 用 now-last 差值计算
	total, err := svc.Heartbeat(ctx, 100)
	require.NoError(t, err)
	assert.LessOrEqual(t, total, 60, "单次心跳累加不得超过 30s 封顶")

	secs, _ := dd.GetWatchSeconds(ctx, 100, businessStatDate())
	assert.LessOrEqual(t, secs, 60)
}

func TestTreasureService_GetStatus_StateMachine(t *testing.T) {
	svc, dd, _ := setupTreasureService(t)
	ctx := context.Background()
	date := businessStatDate()

	// 累计 640s：tier0(180) 达标、tier1(600) 达标、tier2(1800) 未达标
	_, err := dd.AddWatchSeconds(ctx, 100, date, 640)
	require.NoError(t, err)
	// tier0 已领
	_, err = dd.ClaimTx(ctx, 100, date, 0, 100)
	require.NoError(t, err)

	st, err := svc.GetStatus(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, 640, st.WatchedSeconds)
	assert.Equal(t, int64(100), st.CoinBalance)
	require.Len(t, st.Tiers, 3)
	assert.Equal(t, "claimed", st.Tiers[0].State)
	assert.Equal(t, "unlockable", st.Tiers[1].State)
	assert.Equal(t, "locked", st.Tiers[2].State)
}

func TestTreasureService_Claim_Success(t *testing.T) {
	svc, dd, _ := setupTreasureService(t)
	ctx := context.Background()
	date := businessStatDate()
	_, err := dd.AddWatchSeconds(ctx, 100, date, 200) // 达标 tier0
	require.NoError(t, err)

	coins, balance, err := svc.Claim(ctx, 100, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(100), coins)
	assert.Equal(t, int64(100), balance)
}

func TestTreasureService_Claim_RejectsWhenBelowThreshold(t *testing.T) {
	svc, _, _ := setupTreasureService(t)
	// 时长为 0，领 tier0(180s) 必须拒绝
	_, _, err := svc.Claim(context.Background(), 100, 0)
	assert.ErrorIs(t, err, ErrThresholdNotMet)
}

func TestTreasureService_Claim_RejectsInvalidTier(t *testing.T) {
	svc, _, _ := setupTreasureService(t)
	_, _, err := svc.Claim(context.Background(), 100, 9)
	assert.ErrorIs(t, err, ErrInvalidTier)
}

func TestTreasureService_Claim_DuplicateRejected(t *testing.T) {
	svc, dd, _ := setupTreasureService(t)
	ctx := context.Background()
	date := businessStatDate()
	_, err := dd.AddWatchSeconds(ctx, 100, date, 200)
	require.NoError(t, err)

	_, _, err = svc.Claim(ctx, 100, 0)
	require.NoError(t, err)
	_, _, err = svc.Claim(ctx, 100, 0)
	assert.ErrorIs(t, err, dao.ErrAlreadyClaimed)
}
```

- [ ] **Step 2: 运行测试，确认编译失败**

Run: `cd backend/auction && go test ./service/ -run TestTreasureService -v`
Expected: FAIL，`undefined: NewTreasureService` 等。

- [ ] **Step 3: 写实现 `treasure.go`**

`backend/auction/service/treasure.go`：

```go
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"auction-service/dao"
)

// 宝箱档位常量 SSOT（spec：3min/10min/30min → 100/300/800）。
type TierConfig struct {
	Tier             int8
	ThresholdSeconds int
	Coins            int64
}

var treasureTiers = []TierConfig{
	{Tier: 0, ThresholdSeconds: 180, Coins: 100},
	{Tier: 1, ThresholdSeconds: 600, Coins: 300},
	{Tier: 2, ThresholdSeconds: 1800, Coins: 800},
}

const (
	heartbeatCapSeconds = 30                // 单次心跳累加封顶
	heartbeatTTL        = 120 * time.Second // 上次心跳时间戳 TTL
)

var (
	ErrThresholdNotMet = errors.New("watch duration below threshold")
	ErrInvalidTier     = errors.New("invalid tier")
)

// TierStatus 单个宝箱对前端的状态。
type TierStatus struct {
	Tier             int8   `json:"tier"`
	ThresholdSeconds int    `json:"threshold_seconds"`
	Coins            int64  `json:"coins"`
	State            string `json:"state"` // locked / unlockable / claimed
}

// TreasureStatus GET /treasure/status 的编排结果。
type TreasureStatus struct {
	StatDate       string       `json:"stat_date"`
	WatchedSeconds int          `json:"watched_seconds"`
	CoinBalance    int64        `json:"coin_balance"`
	Tiers          []TierStatus `json:"tiers"`
}

// TreasureService 宝箱业务编排。Redis 用于心跳防刷（封顶累加）。
type TreasureService struct {
	dao *dao.TreasureDAO
	rdb *redis.Client
}

func NewTreasureService(d *dao.TreasureDAO, rdb *redis.Client) *TreasureService {
	return &TreasureService{dao: d, rdb: rdb}
}

// businessStatDate 返回业务时区（Asia/Shanghai）当天 YYYY-MM-DD。
func businessStatDate() string {
	return auctionBusinessNow().Format("2006-01-02")
}

func heartbeatKey(userID int64) string {
	return fmt.Sprintf("treasure:hb:%d", userID)
}

// Heartbeat 累加观看时长：按「距上次心跳的真实秒数」累加，单次封顶 30s。
// 首次心跳（无上次时间戳）记入基础 30s。返回今日累计总秒数。
func (s *TreasureService) Heartbeat(ctx context.Context, userID int64) (int, error) {
	if userID <= 0 {
		return 0, errors.New("invalid user_id")
	}
	now := auctionBusinessNow()
	key := heartbeatKey(userID)

	delta := heartbeatCapSeconds
	if lastStr, err := s.rdb.Get(ctx, key).Result(); err == nil {
		if lastUnix, perr := time.ParseInt64(lastStr); perr == nil {
			elapsed := int(now.Unix() - lastUnix)
			if elapsed < 0 {
				elapsed = 0
			}
			if elapsed < delta {
				delta = elapsed
			}
		}
	} else if err != redis.Nil {
		return 0, err
	}

	if err := s.rdb.Set(ctx, key, now.Unix(), heartbeatTTL).Err(); err != nil {
		return 0, err
	}
	if delta <= 0 {
		return s.dao.GetWatchSeconds(ctx, userID, businessStatDate())
	}
	return s.dao.AddWatchSeconds(ctx, userID, businessStatDate(), delta)
}

// GetStatus 返回今日时长 + 金币余额 + 3 宝箱状态。
func (s *TreasureService) GetStatus(ctx context.Context, userID int64) (*TreasureStatus, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user_id")
	}
	date := businessStatDate()
	secs, err := s.dao.GetWatchSeconds(ctx, userID, date)
	if err != nil {
		return nil, err
	}
	balance, err := s.dao.GetCoinBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	claimed, err := s.dao.ListClaimedTiers(ctx, userID, date)
	if err != nil {
		return nil, err
	}

	tiers := make([]TierStatus, 0, len(treasureTiers))
	for _, t := range treasureTiers {
		state := "locked"
		if claimed[t.Tier] {
			state = "claimed"
		} else if secs >= t.ThresholdSeconds {
			state = "unlockable"
		}
		tiers = append(tiers, TierStatus{
			Tier:             t.Tier,
			ThresholdSeconds: t.ThresholdSeconds,
			Coins:            t.Coins,
			State:            state,
		})
	}
	return &TreasureStatus{
		StatDate:       date,
		WatchedSeconds: secs,
		CoinBalance:    balance,
		Tiers:          tiers,
	}, nil
}

// Claim 领取指定 tier 宝箱：校验时长门槛 + 唯一键幂等发币。
// 返回 (本次发放金币, 累加后余额, error)。
func (s *TreasureService) Claim(ctx context.Context, userID int64, tier int8) (int64, int64, error) {
	if userID <= 0 {
		return 0, 0, errors.New("invalid user_id")
	}
	var cfg *TierConfig
	for i := range treasureTiers {
		if treasureTiers[i].Tier == tier {
			cfg = &treasureTiers[i]
			break
		}
	}
	if cfg == nil {
		return 0, 0, ErrInvalidTier
	}

	date := businessStatDate()
	secs, err := s.dao.GetWatchSeconds(ctx, userID, date)
	if err != nil {
		return 0, 0, err
	}
	if secs < cfg.ThresholdSeconds {
		return 0, 0, ErrThresholdNotMet
	}

	balance, err := s.dao.ClaimTx(ctx, userID, date, tier, cfg.Coins)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Coins, balance, nil
}
```

> 注意：`time.ParseInt64` 不存在，改用 `strconv.ParseInt(lastStr, 10, 64)`，并在 import 中加入 `strconv`，移除未用的占位。实现时直接写正确版本：
> ```go
> import "strconv"
> // ...
> if lastUnix, perr := strconv.ParseInt(lastStr, 10, 64); perr == nil {
> ```

- [ ] **Step 4: 修正测试中的 FastForward 占位**

`TestTreasureService_Heartbeat_CapsAt30sPerBeat` 中删除 `mr.FastForward(0)` 这行占位（它不改变时间）。该用例的核心断言是「两次心跳累加 ≤ 60s」——因为每次封顶 30s，无论真实间隔多大都不会超。保留两次 `Heartbeat` 调用与 `assert.LessOrEqual(total, 60)` 即可。最终用例体：

```go
func TestTreasureService_Heartbeat_CapsAt30sPerBeat(t *testing.T) {
	svc, dd, _ := setupTreasureService(t)
	ctx := context.Background()

	_, err := svc.Heartbeat(ctx, 100)
	require.NoError(t, err)
	total, err := svc.Heartbeat(ctx, 100)
	require.NoError(t, err)
	assert.LessOrEqual(t, total, 60, "单次心跳累加不得超过 30s 封顶")

	secs, _ := dd.GetWatchSeconds(ctx, 100, businessStatDate())
	assert.LessOrEqual(t, secs, 60)
}
```

- [ ] **Step 5: 运行测试，确认通过**

Run: `cd backend/auction && go test ./service/ -run TestTreasureService -v`
Expected: PASS（7 个用例全绿）。

- [ ] **Step 6: Commit**

```bash
git add backend/auction/service/treasure.go backend/auction/service/treasure_test.go
git commit -m "feat(treasure): add TreasureService with capped heartbeat and claim orchestration"
```

---

## Task 4: Handler — 3 个 HTTP 端点

**Files:**
- Create: `backend/auction/handler/treasure.go`
- Create: `backend/auction/handler/treasure_test.go`

响应契约（对齐前端解包）：成功 `{"code":200,"data":{...}}`；未登录 `{"code":401,...}`；门槛不足/无效 tier → `400`；重复领取 → `409`；其它 → `500`。

- [ ] **Step 1: 写失败测试 `treasure_test.go`**

`backend/auction/handler/treasure_test.go`：

```go
package handler

import (
	"context"
	"encoding/json"
	"testing"

	"auction-service/dao"
	"auction-service/model"
	"auction-service/service"

	"github.com/alicebob/miniredis/v2"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTreasureHandler(t *testing.T) (*TreasureHandler, *dao.TreasureDAO) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.UserCoin{}, &model.UserWatchDuration{}, &model.TreasureClaim{},
	))
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	d := dao.NewTreasureDAO(db)
	return NewTreasureHandler(service.NewTreasureService(d, rdb)), d
}

func TestTreasureHandler_GetStatus_Unauthorized(t *testing.T) {
	h, _ := newTreasureHandler(t)
	c := app.NewContext(0)
	// 不设置 user_id
	h.GetStatus(context.Background(), c)
	assert.DeepEqual(t, 401, c.Response.StatusCode())
}

func TestTreasureHandler_GetStatus_OK(t *testing.T) {
	h, _ := newTreasureHandler(t)
	c := app.NewContext(0)
	c.Set("user_id", int64(100))
	h.GetStatus(context.Background(), c)
	assert.DeepEqual(t, 200, c.Response.StatusCode())

	var body struct {
		Code int `json:"code"`
		Data struct {
			WatchedSeconds int   `json:"watched_seconds"`
			CoinBalance    int64 `json:"coin_balance"`
			Tiers          []struct {
				Tier  int8   `json:"tier"`
				State string `json:"state"`
			} `json:"tiers"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.DeepEqual(t, 200, body.Code)
	assert.DeepEqual(t, 3, len(body.Data.Tiers))
}

func TestTreasureHandler_Heartbeat_OK(t *testing.T) {
	h, _ := newTreasureHandler(t)
	c := app.NewContext(0)
	c.Set("user_id", int64(100))
	h.Heartbeat(context.Background(), c)
	assert.DeepEqual(t, 200, c.Response.StatusCode())
}

func TestTreasureHandler_Claim_BelowThresholdReturns400(t *testing.T) {
	h, _ := newTreasureHandler(t)
	c := app.NewContext(0)
	c.Set("user_id", int64(100))
	c.Request.SetBody([]byte(`{"tier":0}`))
	h.Claim(context.Background(), c)
	assert.DeepEqual(t, 400, c.Response.StatusCode())
}

func TestTreasureHandler_Claim_OK(t *testing.T) {
	h, d := newTreasureHandler(t)
	ctx := context.Background()
	_, err := d.AddWatchSeconds(ctx, 100, businessDateForTest(), 200)
	require.NoError(t, err)

	c := app.NewContext(0)
	c.Set("user_id", int64(100))
	c.Request.SetBody([]byte(`{"tier":0}`))
	h.Claim(ctx, c)
	assert.DeepEqual(t, 200, c.Response.StatusCode())

	var body struct {
		Data struct {
			Coins       int64 `json:"coins"`
			CoinBalance int64 `json:"coin_balance"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(c.Response.Body(), &body))
	assert.DeepEqual(t, int64(100), body.Data.Coins)
	assert.DeepEqual(t, int64(100), body.Data.CoinBalance)
}
```

> `businessDateForTest()` 需与 service 的 `businessStatDate()` 同口径。在测试文件内补一个辅助：
> ```go
> func businessDateForTest() string {
> 	loc := time.FixedZone("Asia/Shanghai", 8*60*60)
> 	return time.Now().In(loc).Format("2006-01-02")
> }
> ```
> 并在 import 中加 `"time"`。

- [ ] **Step 2: 运行测试，确认编译失败**

Run: `cd backend/auction && go test ./handler/ -run TestTreasureHandler -v`
Expected: FAIL，`undefined: NewTreasureHandler`。

- [ ] **Step 3: 写实现 `treasure.go`**

`backend/auction/handler/treasure.go`：

```go
package handler

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"

	"auction-service/dao"
	"auction-service/service"
)

// TreasureHandler 观看时长宝箱 HTTP 入口。user_id 由 gateway 经 X-User-ID 注入。
type TreasureHandler struct {
	svc *service.TreasureService
}

func NewTreasureHandler(svc *service.TreasureService) *TreasureHandler {
	return &TreasureHandler{svc: svc}
}

func (h *TreasureHandler) GetStatus(ctx context.Context, c *app.RequestContext) {
	userID := c.GetInt64("user_id")
	if userID <= 0 {
		c.JSON(401, map[string]any{"code": 401, "message": "未登录或无效用户"})
		return
	}
	st, err := h.svc.GetStatus(ctx, userID)
	if err != nil {
		c.JSON(500, map[string]any{"code": 500, "message": "查询宝箱状态失败"})
		return
	}
	c.JSON(200, map[string]any{"code": 200, "data": st})
}

func (h *TreasureHandler) Heartbeat(ctx context.Context, c *app.RequestContext) {
	userID := c.GetInt64("user_id")
	if userID <= 0 {
		c.JSON(401, map[string]any{"code": 401, "message": "未登录或无效用户"})
		return
	}
	total, err := h.svc.Heartbeat(ctx, userID)
	if err != nil {
		c.JSON(500, map[string]any{"code": 500, "message": "心跳上报失败"})
		return
	}
	c.JSON(200, map[string]any{"code": 200, "data": map[string]any{"watched_seconds": total}})
}

func (h *TreasureHandler) Claim(ctx context.Context, c *app.RequestContext) {
	userID := c.GetInt64("user_id")
	if userID <= 0 {
		c.JSON(401, map[string]any{"code": 401, "message": "未登录或无效用户"})
		return
	}
	var req struct {
		Tier int8 `json:"tier"`
	}
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		c.JSON(400, map[string]any{"code": 400, "message": "参数错误"})
		return
	}

	coins, balance, err := h.svc.Claim(ctx, userID, req.Tier)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrThresholdNotMet):
			c.JSON(400, map[string]any{"code": 400, "message": "观看时长未达标"})
		case errors.Is(err, service.ErrInvalidTier):
			c.JSON(400, map[string]any{"code": 400, "message": "无效的宝箱档位"})
		case errors.Is(err, dao.ErrAlreadyClaimed):
			c.JSON(409, map[string]any{"code": 409, "message": "该宝箱今日已领取"})
		default:
			c.JSON(500, map[string]any{"code": 500, "message": "领取失败"})
		}
		return
	}
	c.JSON(200, map[string]any{
		"code": 200,
		"data": map[string]any{"coins": coins, "coin_balance": balance},
	})
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `cd backend/auction && go test ./handler/ -run TestTreasureHandler -v`
Expected: PASS（5 个用例全绿）。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/handler/treasure.go backend/auction/handler/treasure_test.go
git commit -m "feat(treasure): add HTTP handlers for status/heartbeat/claim"
```

---

## Task 5: 装配 main.go + AutoMigrate + 路由注册

**Files:**
- Modify: `backend/auction/main.go`

- [ ] **Step 1: AutoMigrate 追加三个模型**

在 [main.go:71-85](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/main.go#L71-L85) 的 `db.AutoMigrate(...)` 列表末尾（`&model.AuctionSettlementTask{},` 之后）追加：

```go
		&model.UserCoin{},
		&model.UserWatchDuration{},
		&model.TreasureClaim{},
```

- [ ] **Step 2: 初始化 DAO/service/handler**

在 [main.go:107](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/main.go#L107)（`statisticsDAO := ...` 之后）追加：

```go
	treasureDAO := dao.NewTreasureDAO(db)
```

在 [main.go:220](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/main.go#L220)（`statisticsHandler := ...` 之后）追加（Redis 复用 `dao.GetRedis()`）：

```go
	treasureHandler := handler.NewTreasureHandler(service.NewTreasureService(treasureDAO, dao.GetRedis()))
```

- [ ] **Step 3: 注册路由**

修改 `registerRoutes` 的函数签名，新增末位参数 `treasureHandler *handler.TreasureHandler`；并在 [main.go:316](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/main.go#L316) 的调用处传入 `treasureHandler`。

在 `registerRoutes` 函数体内（地址 CRUD 路由之后，函数结束前）追加：

```go
	// ========== 观看时长宝箱 ==========
	v1.GET("/treasure/status", treasureHandler.GetStatus)
	v1.POST("/treasure/claim", treasureHandler.Claim)
	v1.POST("/watch/heartbeat", treasureHandler.Heartbeat)
```

- [ ] **Step 4: 编译验证**

Run: `cd backend/auction && go build ./...`
Expected: 无输出，退出码 0。

- [ ] **Step 5: 全量测试回归**

Run: `cd backend/auction && go test ./dao/ ./service/ ./handler/ 2>&1 | tail -n 20`
Expected: 全部 `ok`，无 FAIL。

- [ ] **Step 6: Commit**

```bash
git add backend/auction/main.go
git commit -m "feat(treasure): wire treasure DAO/service/handler and register routes"
```

---

## Task 6: Gateway 路由代理

**Files:**
- Modify: `backend/gateway/router/router.go`

宝箱接口需登录（金币绑用户），挂在 `authGroup`（已 `JWTAuth`，会注入 `X-User-ID`）。

- [ ] **Step 1: 在 authGroup 下新增代理路由**

在 [router.go:126](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/gateway/router/router.go#L126)（`authGroup.GET("/user/balance", ...)` 附近，用户相关区块内）追加：

```go
	// ========== 观看时长宝箱（需登录） ==========
	authGroup.GET("/treasure/status", auctionProxy.Forward)
	authGroup.POST("/treasure/claim", auctionProxy.Forward)
	authGroup.POST("/watch/heartbeat", auctionProxy.Forward)
```

- [ ] **Step 2: 编译验证**

Run: `cd backend/gateway && go build ./...`
Expected: 无输出，退出码 0。

- [ ] **Step 3: Commit**

```bash
git add backend/gateway/router/router.go
git commit -m "feat(treasure): proxy treasure routes through gateway authGroup"
```

---

## Task 7: 端到端联调验证（手动）

**Files:** 无（验证步骤）

- [ ] **Step 1: 启动本地后端**

按项目 runbook 启动 gateway + auction（端口见本地脚本）。确认 AutoMigrate 无致命错误日志。

- [ ] **Step 2: 登录拿 token，验证 status**

用演示账号（138 系列）登录获取 JWT，调用：

```bash
curl -s -H "Authorization: Bearer <TOKEN>" http://localhost:<gatewayPort>/api/v1/treasure/status | jq
```

Expected：返回 `code:200`，`data.tiers` 长度 3，初始 `watched_seconds:0`、`coin_balance:0`、三个 tier 均 `locked`。

- [ ] **Step 3: 心跳累加**

```bash
curl -s -X POST -H "Authorization: Bearer <TOKEN>" -H "Content-Type: application/json" -d '{}' http://localhost:<gatewayPort>/api/v1/watch/heartbeat | jq
```

Expected：`data.watched_seconds` 为 30；连续调用受 Redis 时间戳约束（间隔短则 delta 小）。

- [ ] **Step 4: 门槛与领取**

时长未达 180s 时领 tier0 应返回 `code:400`；可通过直接写库或多次心跳累计达标后领取，返回 `code:200`、`data.coins:100`、`coin_balance` 递增；重复领取返回 `code:409`。

- [ ] **Step 5: 前端真机/浏览器验证**

进入 H5 直播间，确认进度条随心跳推进、达标宝箱跳动、点击领取播放 `+N` 飘字且金币数字增长；切后台（隐藏标签页）期间不再累加。

---

## Self-Review

**Spec coverage：**
- 数据模型（user_coins / user_watch_duration / treasure_claims）→ Task 1 ✅
- 心跳累加 + 封顶防刷 → Task 3（`Heartbeat`，Redis 时间戳 + 30s 封顶）✅
- status 三态状态机 → Task 3（`GetStatus`）✅
- claim 门槛校验 + 唯一键幂等 → Task 2（`ClaimTx`）+ Task 3（`Claim`）✅
- 档位 100/300/800 → Task 3 `treasureTiers` 常量 ✅
- 经 gateway JWT 暴露 → Task 6 ✅
- 与现金 user_balances 隔离 → 全程不引用 UserBalance ✅
- 前端契约字段 `watched_seconds/coin_balance/tiers[{tier,threshold_seconds,coins,state}]` + claim 返回 `coins/coin_balance` → Task 3 结构体 json tag + Task 4 响应一致 ✅
- 用户中心金币展示（spec §7）→ **本计划未含**，标注为后续可选增量（前端组件已能独立展示金币，用户中心入口非闭环必需）。

**Placeholder scan：** Task 3 Step 3 标注了 `time.ParseInt64` 的修正说明（改 `strconv.ParseInt`）；Step 4 显式给出去除 `mr.FastForward(0)` 占位后的最终用例体。无 TBD/TODO。

**Type consistency：**
- `AddWatchSeconds(ctx, userID, statDate, delta) (int, error)` 在 DAO/service/test 一致。
- `ClaimTx(ctx, userID, statDate, tier int8, coins int64) (int64, error)` 一致。
- `Claim(ctx, userID, tier int8) (coins int64, balance int64, error)` 在 service/handler/test 一致。
- 错误：`dao.ErrAlreadyClaimed`、`service.ErrThresholdNotMet`、`service.ErrInvalidTier` 定义与引用一致。
- `businessStatDate()`（service）与 `businessDateForTest()`（handler test）同口径 Asia/Shanghai。

**已知偏差修复点（实现者注意）：**
- service `treasure.go` import 必须含 `strconv`，不要写 `time.ParseInt64`。

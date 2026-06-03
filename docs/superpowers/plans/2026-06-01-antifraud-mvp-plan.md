# C2 反作弊 MVP 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `auction-service` 出价主链路前置实时风控引擎，落地 R1（高频）/R4（异常加价）/R5（新账号秒拍）三条规则，预留 LLM 解释器接口；新增 `risk_event` 表持久化事件用于运营审核与离线特征工程。

**Architecture:** 在 `backend/auction/service/antifraud/` 包内实现规则引擎，每条规则一个文件，通过 `Rule` 接口正交装配。引擎 hook 在 `BidService.PlaceBid` 第 0.2 步（用户校验后、状态校验前）；规则引擎采用短路模式，命中第一条非 pass 即返回。`RiskExplainer` 接口在 antifraud 包内定义但 MVP 不注入实现，留待 v1.1 接入 LLM。失败策略 fail-open，不阻断主业务。

**Tech Stack:** Go 1.24 + Hertz + GORM + go-redis/v9 + shopspring/decimal + testify + miniredis（新增测试依赖）+ go-sqlmock（新增测试依赖）

**Spec 参考：** [2026-06-01-antifraud-mvp-design.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/superpowers/specs/2026-06-01-antifraud-mvp-design.md)

---

## 文件结构总览

| 文件 | 用途 | 类型 |
|---|---|---|
| `backend/auction/service/antifraud/types.go` | BidEvent / RiskDecision / RiskExplainer / Rule 接口 | 新建 |
| `backend/auction/service/antifraud/engine.go` | Engine 编排 + 短路执行 + 封禁 fast-path | 新建 |
| `backend/auction/service/antifraud/rule_rapid_fire.go` | R1 高频出价规则（Redis ZSET） | 新建 |
| `backend/auction/service/antifraud/rule_abnormal_jump.go` | R4 异常加价规则（DB 查 auction） | 新建 |
| `backend/auction/service/antifraud/rule_fresh_account.go` | R5 新账号秒拍规则（DB + Redis 累计） | 新建 |
| `backend/auction/service/antifraud/rules.go` | DefaultRules() 装配 | 新建 |
| `backend/auction/service/antifraud/*_test.go` | 单元测试（U1-U11） | 新建 |
| `backend/auction/model/risk_event.go` | RiskEvent gorm model | 新建 |
| `backend/auction/dao/risk_event.go` | RiskEventDAO | 新建 |
| `backend/auction/dao/risk_event_test.go` | DAO 单测 | 新建 |
| `backend/auction/service/bid.go` | 修改：注入 antifraudEngine + 第 0.2 步调用 | 修改 |
| `backend/auction/service/bid_test.go` | 修改：补集成测试 I1/I3 | 修改 |
| `backend/auction/handler/bid.go` | 修改：错误码映射 + Confirmed 字段透传 | 修改 |
| `backend/auction/main.go` | 修改：装配 antifraud.Engine + AutoMigrate RiskEvent | 修改 |
| `backend/auction/pkg/metrics/antifraud_metrics.go` | Prometheus 指标 | 新建 |
| `backend/auction/go.mod` | 新增 miniredis + go-sqlmock | 修改 |

---

## Task 1: antifraud 包 — 类型定义

**Files:**
- Create: `backend/auction/service/antifraud/types.go`
- Create: `backend/auction/service/antifraud/types_test.go`

- [ ] **Step 1: 写失败测试**

```go
// backend/auction/service/antifraud/types_test.go
package antifraud

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestBidEvent_Fields(t *testing.T) {
	evt := &BidEvent{
		UserID:    100,
		AuctionID: 200,
		Amount:    decimal.NewFromInt(500),
		IP:        "1.2.3.4",
		UA:        "go-test",
		Confirmed: true,
	}
	assert.Equal(t, int64(100), evt.UserID)
	assert.True(t, evt.Confirmed)
}

func TestRiskDecision_Defaults(t *testing.T) {
	dec := PassDecision()
	assert.Equal(t, ActionPass, dec.Action)
	assert.Equal(t, LevelLow, dec.Level)
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
cd backend/auction && go test ./service/antifraud/... -run TestBidEvent_Fields -v
```

Expected: FAIL，提示 `undefined: BidEvent` 等。

- [ ] **Step 3: 最小实现**

```go
// backend/auction/service/antifraud/types.go
package antifraud

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// 风险等级
const (
	LevelLow      = "low"
	LevelMedium   = "medium"
	LevelHigh     = "high"
	LevelCritical = "critical"
)

// 处置动作
const (
	ActionPass      = "pass"
	ActionMark      = "mark"
	ActionChallenge = "challenge"
	ActionBlock     = "block"
)

// 规则 ID
const (
	RuleRapidFire     = "R1_rapid_fire"
	RuleAbnormalJump  = "R4_abnormal_jump"
	RuleFreshAccount  = "R5_fresh_account_sniping"
	RuleBanned        = "banned"
)

// BidEvent 风控判定输入
type BidEvent struct {
	UserID    int64
	AuctionID int64
	Amount    decimal.Decimal
	IP        string
	UA        string
	Timestamp time.Time
	Confirmed bool // 用户在 R4 challenge 后二次确认；为 true 时 R4 自动放行
}

// RiskDecision 风控判定输出
type RiskDecision struct {
	Level    string
	Action   string
	Rules    []string
	Features map[string]any
	Reason   string
}

// PassDecision 默认放行决策
func PassDecision() *RiskDecision {
	return &RiskDecision{Level: LevelLow, Action: ActionPass}
}

// Rule 规则接口；每条规则独立判定
type Rule interface {
	ID() string
	Check(ctx context.Context, evt *BidEvent) (*RiskDecision, error)
}

// RiskExplainer LLM 解释器接口（v1.1 接入；MVP 不注入实现）
type RiskExplainer interface {
	Explain(ctx context.Context, event *BidEvent, decision *RiskDecision) (string, error)
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
cd backend/auction && go test ./service/antifraud/... -run "TestBidEvent_Fields|TestRiskDecision_Defaults" -v
```

Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/service/antifraud/types.go backend/auction/service/antifraud/types_test.go
git commit -m "feat(antifraud): 定义反作弊核心类型 BidEvent/RiskDecision/Rule/RiskExplainer"
```

---

## Task 2: 引入 miniredis + go-sqlmock 测试依赖

**Files:**
- Modify: `backend/auction/go.mod`
- Modify: `backend/auction/go.sum`

- [ ] **Step 1: 拉取依赖**

```bash
cd backend/auction
go get github.com/alicebob/miniredis/v2@latest
go get github.com/DATA-DOG/go-sqlmock@latest
go mod tidy
```

- [ ] **Step 2: 写探针测试**

```go
// backend/auction/service/antifraud/probe_test.go
package antifraud

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
)

func TestMiniredisAvailable(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()
	assert.NotEmpty(t, mr.Addr())
}
```

- [ ] **Step 3: 验证可编译**

```bash
cd backend/auction && go test ./service/antifraud/... -run TestMiniredisAvailable -v
```

Expected: PASS。

- [ ] **Step 4: Commit**

```bash
git add backend/auction/go.mod backend/auction/go.sum backend/auction/service/antifraud/probe_test.go
git commit -m "chore(antifraud): 引入 miniredis 与 go-sqlmock 作为测试依赖"
```

---

## Task 3: R1 规则 — 高频出价（RapidFireRule）

**Files:**
- Create: `backend/auction/service/antifraud/rule_rapid_fire.go`
- Create: `backend/auction/service/antifraud/rule_rapid_fire_test.go`

- [ ] **Step 1: 写失败测试（U1/U2/U3）**

```go
// backend/auction/service/antifraud/rule_rapid_fire_test.go
package antifraud

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func newTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { mr.Close(); _ = cli.Close() })
	return cli, mr
}

func TestRapidFire_U1_Hit_8thBidIn5s(t *testing.T) {
	cli, _ := newTestRedis(t)
	rule := NewRapidFireRule(cli, RapidFireConfig{
		WindowSec: 5, Threshold: 8, BanThreshold: 3, BanTTL: 600 * time.Second,
	})
	ctx := context.Background()
	evt := &BidEvent{UserID: 1, Timestamp: time.Now()}

	for i := 0; i < 7; i++ {
		dec, err := rule.Check(ctx, evt)
		assert.NoError(t, err)
		assert.Equal(t, ActionPass, dec.Action, "i=%d", i)
	}
	dec, err := rule.Check(ctx, evt)
	assert.NoError(t, err)
	assert.Equal(t, ActionBlock, dec.Action)
	assert.Equal(t, LevelHigh, dec.Level)
	assert.Contains(t, dec.Rules, RuleRapidFire)
}

func TestRapidFire_U2_Pass_7thBid(t *testing.T) {
	cli, _ := newTestRedis(t)
	rule := NewRapidFireRule(cli, RapidFireConfig{
		WindowSec: 5, Threshold: 8, BanThreshold: 3, BanTTL: 600 * time.Second,
	})
	ctx := context.Background()
	evt := &BidEvent{UserID: 2, Timestamp: time.Now()}

	for i := 0; i < 7; i++ {
		dec, err := rule.Check(ctx, evt)
		assert.NoError(t, err)
		assert.Equal(t, ActionPass, dec.Action)
	}
}

func TestRapidFire_U3_Pass_AfterWindow(t *testing.T) {
	cli, mr := newTestRedis(t)
	rule := NewRapidFireRule(cli, RapidFireConfig{
		WindowSec: 5, Threshold: 8, BanThreshold: 3, BanTTL: 600 * time.Second,
	})
	ctx := context.Background()
	evt := &BidEvent{UserID: 3, Timestamp: time.Now()}

	for i := 0; i < 7; i++ {
		_, _ = rule.Check(ctx, evt)
	}
	mr.FastForward(6 * time.Second) // 跳过窗口
	evt.Timestamp = time.Now()
	dec, err := rule.Check(ctx, evt)
	assert.NoError(t, err)
	assert.Equal(t, ActionPass, dec.Action)
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
cd backend/auction && go test ./service/antifraud/... -run TestRapidFire -v
```

Expected: FAIL，`undefined: NewRapidFireRule`。

- [ ] **Step 3: 实现规则**

```go
// backend/auction/service/antifraud/rule_rapid_fire.go
package antifraud

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RapidFireConfig 高频出价规则配置
type RapidFireConfig struct {
	WindowSec    int           // 滑动窗口秒数（默认 5）
	Threshold    int           // 命中阈值（默认 8）
	BanThreshold int           // 连续命中后封禁的次数阈值（默认 3）
	BanTTL       time.Duration // 封禁 TTL（默认 600s）
	KeyTTL       time.Duration // ZSET key TTL（默认 60s）
}

func DefaultRapidFireConfig() RapidFireConfig {
	return RapidFireConfig{
		WindowSec:    5,
		Threshold:    8,
		BanThreshold: 3,
		BanTTL:       600 * time.Second,
		KeyTTL:       60 * time.Second,
	}
}

// RapidFireRule R1 高频出价规则
type RapidFireRule struct {
	rdb *redis.Client
	cfg RapidFireConfig
}

func NewRapidFireRule(rdb *redis.Client, cfg RapidFireConfig) *RapidFireRule {
	if cfg.WindowSec == 0 {
		cfg = DefaultRapidFireConfig()
	}
	return &RapidFireRule{rdb: rdb, cfg: cfg}
}

func (r *RapidFireRule) ID() string { return RuleRapidFire }

func (r *RapidFireRule) Check(ctx context.Context, evt *BidEvent) (*RiskDecision, error) {
	now := evt.Timestamp
	if now.IsZero() {
		now = time.Now()
	}
	nowMs := now.UnixMilli()
	windowStart := nowMs - int64(r.cfg.WindowSec*1000)
	rateKey := fmt.Sprintf("antifraud:bid:rate:%d", evt.UserID)

	pipe := r.rdb.Pipeline()
	pipe.ZAdd(ctx, rateKey, redis.Z{Score: float64(nowMs), Member: nowMs})
	pipe.ZRemRangeByScore(ctx, rateKey, "-inf", fmt.Sprintf("(%d", windowStart))
	cardCmd := pipe.ZCard(ctx, rateKey)
	pipe.Expire(ctx, rateKey, r.cfg.KeyTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("rapid_fire pipeline: %w", err)
	}

	count := cardCmd.Val()
	if count < int64(r.cfg.Threshold) {
		return PassDecision(), nil
	}

	// 命中：累计封禁计数器
	hitKey := fmt.Sprintf("antifraud:bid:hits:%d", evt.UserID)
	hits, err := r.rdb.Incr(ctx, hitKey).Result()
	if err != nil {
		return nil, fmt.Errorf("rapid_fire incr hits: %w", err)
	}
	r.rdb.Expire(ctx, hitKey, 10*time.Minute)

	if hits >= int64(r.cfg.BanThreshold) {
		banKey := fmt.Sprintf("antifraud:ban:%d", evt.UserID)
		r.rdb.Set(ctx, banKey, "1", r.cfg.BanTTL)
	}

	return &RiskDecision{
		Level:  LevelHigh,
		Action: ActionBlock,
		Rules:  []string{RuleRapidFire},
		Reason: "出价过于频繁，请稍后再试",
		Features: map[string]any{
			"window_sec":  r.cfg.WindowSec,
			"count":       count,
			"hits":        hits,
		},
	}, nil
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
cd backend/auction && go test ./service/antifraud/... -run TestRapidFire -v
```

Expected: PASS（3 个用例）。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/service/antifraud/rule_rapid_fire.go backend/auction/service/antifraud/rule_rapid_fire_test.go
git commit -m "feat(antifraud): R1 高频出价规则（5s/8 次 + 封禁累计）"
```

---

## Task 4: R4 规则 — 异常加价（AbnormalJumpRule）

**Files:**
- Create: `backend/auction/service/antifraud/rule_abnormal_jump.go`
- Create: `backend/auction/service/antifraud/rule_abnormal_jump_test.go`

- [ ] **Step 1: 写失败测试（U5/U6/U7）**

```go
// backend/auction/service/antifraud/rule_abnormal_jump_test.go
package antifraud

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// 测试用桩：实现 AuctionPriceLoader
type fakeLoader struct {
	current   decimal.Decimal
	increment decimal.Decimal
	err       error
}

func (f *fakeLoader) Load(ctx context.Context, auctionID int64) (decimal.Decimal, decimal.Decimal, error) {
	return f.current, f.increment, f.err
}

func TestAbnormalJump_U5_Challenge_10x(t *testing.T) {
	loader := &fakeLoader{
		current:   decimal.NewFromInt(100),
		increment: decimal.NewFromInt(10),
	}
	rule := NewAbnormalJumpRule(loader, AbnormalJumpConfig{Multiplier: 10, ZeroIncrementMultiplier: 100})
	dec, err := rule.Check(context.Background(), &BidEvent{
		AuctionID: 1, Amount: decimal.NewFromInt(1100),
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionChallenge, dec.Action)
	assert.Equal(t, LevelMedium, dec.Level)
	assert.Contains(t, dec.Rules, RuleAbnormalJump)
}

func TestAbnormalJump_U6_Pass_WhenConfirmed(t *testing.T) {
	loader := &fakeLoader{
		current:   decimal.NewFromInt(100),
		increment: decimal.NewFromInt(10),
	}
	rule := NewAbnormalJumpRule(loader, AbnormalJumpConfig{Multiplier: 10, ZeroIncrementMultiplier: 100})
	dec, err := rule.Check(context.Background(), &BidEvent{
		AuctionID: 1, Amount: decimal.NewFromInt(1100), Confirmed: true,
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionPass, dec.Action)
}

func TestAbnormalJump_U7_Challenge_OnZeroCurrent(t *testing.T) {
	loader := &fakeLoader{
		current:   decimal.Zero,
		increment: decimal.NewFromInt(10),
	}
	rule := NewAbnormalJumpRule(loader, AbnormalJumpConfig{Multiplier: 10, ZeroIncrementMultiplier: 100})
	dec, err := rule.Check(context.Background(), &BidEvent{
		AuctionID: 1, Amount: decimal.NewFromInt(1000), // 10 * 100
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionChallenge, dec.Action)
}

func TestAbnormalJump_Pass_NormalIncrement(t *testing.T) {
	loader := &fakeLoader{
		current:   decimal.NewFromInt(100),
		increment: decimal.NewFromInt(10),
	}
	rule := NewAbnormalJumpRule(loader, AbnormalJumpConfig{Multiplier: 10, ZeroIncrementMultiplier: 100})
	dec, err := rule.Check(context.Background(), &BidEvent{
		AuctionID: 1, Amount: decimal.NewFromInt(150),
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionPass, dec.Action)
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
cd backend/auction && go test ./service/antifraud/... -run TestAbnormalJump -v
```

Expected: FAIL。

- [ ] **Step 3: 实现规则**

```go
// backend/auction/service/antifraud/rule_abnormal_jump.go
package antifraud

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
)

// AuctionPriceLoader 加载竞拍当前价 + 加价幅度
type AuctionPriceLoader interface {
	Load(ctx context.Context, auctionID int64) (current decimal.Decimal, increment decimal.Decimal, err error)
}

// AbnormalJumpConfig R4 配置
type AbnormalJumpConfig struct {
	Multiplier              int64 // 单笔加价幅度倍数阈值（默认 10）
	ZeroIncrementMultiplier int64 // 起拍前的兜底倍数（默认 100）
}

func DefaultAbnormalJumpConfig() AbnormalJumpConfig {
	return AbnormalJumpConfig{Multiplier: 10, ZeroIncrementMultiplier: 100}
}

type AbnormalJumpRule struct {
	loader AuctionPriceLoader
	cfg    AbnormalJumpConfig
}

func NewAbnormalJumpRule(loader AuctionPriceLoader, cfg AbnormalJumpConfig) *AbnormalJumpRule {
	if cfg.Multiplier == 0 {
		cfg = DefaultAbnormalJumpConfig()
	}
	return &AbnormalJumpRule{loader: loader, cfg: cfg}
}

func (r *AbnormalJumpRule) ID() string { return RuleAbnormalJump }

func (r *AbnormalJumpRule) Check(ctx context.Context, evt *BidEvent) (*RiskDecision, error) {
	if evt.Confirmed {
		return PassDecision(), nil
	}
	current, increment, err := r.loader.Load(ctx, evt.AuctionID)
	if err != nil {
		return nil, fmt.Errorf("abnormal_jump loader: %w", err)
	}

	hit := false
	if current.IsZero() {
		// 起拍前：Amount >= increment * ZeroIncrementMultiplier
		threshold := increment.Mul(decimal.NewFromInt(r.cfg.ZeroIncrementMultiplier))
		hit = evt.Amount.GreaterThanOrEqual(threshold) && threshold.GreaterThan(decimal.Zero)
	} else {
		// 加价幅度 = Amount - current；命中：>= current * Multiplier
		jump := evt.Amount.Sub(current)
		threshold := current.Mul(decimal.NewFromInt(r.cfg.Multiplier))
		hit = jump.GreaterThanOrEqual(threshold)
	}

	if !hit {
		return PassDecision(), nil
	}
	return &RiskDecision{
		Level:  LevelMedium,
		Action: ActionChallenge,
		Rules:  []string{RuleAbnormalJump},
		Reason: "出价金额异常，请确认后再次提交",
		Features: map[string]any{
			"current":   current.String(),
			"increment": increment.String(),
			"amount":    evt.Amount.String(),
		},
	}, nil
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
cd backend/auction && go test ./service/antifraud/... -run TestAbnormalJump -v
```

Expected: PASS（4 个用例）。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/service/antifraud/rule_abnormal_jump.go backend/auction/service/antifraud/rule_abnormal_jump_test.go
git commit -m "feat(antifraud): R4 异常加价规则（10x 阈值 + Confirmed 放行 + 起拍前兜底）"
```

---

## Task 5: R5 规则 — 新账号秒拍（FreshAccountRule）

**Files:**
- Create: `backend/auction/service/antifraud/rule_fresh_account.go`
- Create: `backend/auction/service/antifraud/rule_fresh_account_test.go`

- [ ] **Step 1: 写失败测试（U8/U9/U10）**

```go
// backend/auction/service/antifraud/rule_fresh_account_test.go
package antifraud

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

type fakeUserInfo struct {
	createdAt time.Time
	kyc       bool
	err       error
}

type fakeUserLoader struct {
	users map[int64]fakeUserInfo
}

func (f *fakeUserLoader) Load(ctx context.Context, userID int64) (time.Time, bool, error) {
	u, ok := f.users[userID]
	if !ok {
		return time.Time{}, false, nil
	}
	return u.createdAt, u.kyc, u.err
}

func TestFreshAccount_U8_Block_FreshAndOver10000(t *testing.T) {
	cli, _ := newTestRedis(t)
	loader := &fakeUserLoader{users: map[int64]fakeUserInfo{
		1: {createdAt: time.Now().Add(-23 * time.Hour), kyc: false},
	}}
	rule := NewFreshAccountRule(cli, loader, FreshAccountConfig{
		FreshDuration: 24 * time.Hour, AmountThreshold: decimal.NewFromInt(10000),
	})
	// 累计 9000 + 本次 1001 = 10001
	cli.IncrByFloat(context.Background(), "antifraud:bid:total:1", 9000)
	dec, err := rule.Check(context.Background(), &BidEvent{
		UserID: 1, Amount: decimal.NewFromInt(1001),
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionBlock, dec.Action)
	assert.Equal(t, LevelHigh, dec.Level)
	assert.Contains(t, dec.Rules, RuleFreshAccount)
}

func TestFreshAccount_U9_Pass_OldAccount(t *testing.T) {
	cli, _ := newTestRedis(t)
	loader := &fakeUserLoader{users: map[int64]fakeUserInfo{
		2: {createdAt: time.Now().Add(-25 * time.Hour), kyc: false},
	}}
	rule := NewFreshAccountRule(cli, loader, FreshAccountConfig{
		FreshDuration: 24 * time.Hour, AmountThreshold: decimal.NewFromInt(10000),
	})
	cli.IncrByFloat(context.Background(), "antifraud:bid:total:2", 50000)
	dec, err := rule.Check(context.Background(), &BidEvent{
		UserID: 2, Amount: decimal.NewFromInt(1),
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionPass, dec.Action)
}

func TestFreshAccount_U10_Pass_KYC(t *testing.T) {
	cli, _ := newTestRedis(t)
	loader := &fakeUserLoader{users: map[int64]fakeUserInfo{
		3: {createdAt: time.Now().Add(-1 * time.Hour), kyc: true},
	}}
	rule := NewFreshAccountRule(cli, loader, FreshAccountConfig{
		FreshDuration: 24 * time.Hour, AmountThreshold: decimal.NewFromInt(10000),
	})
	dec, err := rule.Check(context.Background(), &BidEvent{
		UserID: 3, Amount: decimal.NewFromInt(99999),
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionPass, dec.Action)
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
cd backend/auction && go test ./service/antifraud/... -run TestFreshAccount -v
```

Expected: FAIL。

- [ ] **Step 3: 实现规则**

```go
// backend/auction/service/antifraud/rule_fresh_account.go
package antifraud

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

// UserInfoLoader 加载用户注册时间 + KYC 状态
type UserInfoLoader interface {
	Load(ctx context.Context, userID int64) (createdAt time.Time, kycVerified bool, err error)
}

// FreshAccountConfig R5 配置
type FreshAccountConfig struct {
	FreshDuration   time.Duration   // 新账号窗口（默认 24h）
	AmountThreshold decimal.Decimal // 累计出价金额阈值（默认 10000）
	KeyTTL          time.Duration   // 累计 key TTL（默认 24h）
}

func DefaultFreshAccountConfig() FreshAccountConfig {
	return FreshAccountConfig{
		FreshDuration:   24 * time.Hour,
		AmountThreshold: decimal.NewFromInt(10000),
		KeyTTL:          24 * time.Hour,
	}
}

type FreshAccountRule struct {
	rdb    *redis.Client
	loader UserInfoLoader
	cfg    FreshAccountConfig
}

func NewFreshAccountRule(rdb *redis.Client, loader UserInfoLoader, cfg FreshAccountConfig) *FreshAccountRule {
	if cfg.FreshDuration == 0 {
		cfg = DefaultFreshAccountConfig()
	}
	if cfg.KeyTTL == 0 {
		cfg.KeyTTL = 24 * time.Hour
	}
	return &FreshAccountRule{rdb: rdb, loader: loader, cfg: cfg}
}

func (r *FreshAccountRule) ID() string { return RuleFreshAccount }

func (r *FreshAccountRule) Check(ctx context.Context, evt *BidEvent) (*RiskDecision, error) {
	createdAt, kyc, err := r.loader.Load(ctx, evt.UserID)
	if err != nil {
		return nil, fmt.Errorf("fresh_account loader: %w", err)
	}
	if kyc {
		return PassDecision(), nil
	}
	if time.Since(createdAt) >= r.cfg.FreshDuration {
		return PassDecision(), nil
	}

	totalKey := fmt.Sprintf("antifraud:bid:total:%d", evt.UserID)
	totalStr, err := r.rdb.Get(ctx, totalKey).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("fresh_account get total: %w", err)
	}
	total, _ := decimal.NewFromString(totalStr) // 空字符串解析为 0
	projected := total.Add(evt.Amount)

	if projected.LessThanOrEqual(r.cfg.AmountThreshold) {
		// 未超阈值：累加并放行
		amtF, _ := evt.Amount.Float64()
		r.rdb.IncrByFloat(ctx, totalKey, amtF)
		r.rdb.Expire(ctx, totalKey, r.cfg.KeyTTL)
		return PassDecision(), nil
	}

	return &RiskDecision{
		Level:  LevelHigh,
		Action: ActionBlock,
		Rules:  []string{RuleFreshAccount},
		Reason: "新账号需完成实名认证后才能高额出价",
		Features: map[string]any{
			"created_at":   createdAt,
			"total_amount": total.String(),
			"this_amount":  evt.Amount.String(),
		},
	}, nil
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
cd backend/auction && go test ./service/antifraud/... -run TestFreshAccount -v
```

Expected: PASS（3 个用例）。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/service/antifraud/rule_fresh_account.go backend/auction/service/antifraud/rule_fresh_account_test.go
git commit -m "feat(antifraud): R5 新账号秒拍规则（24h + 1 万累计 + KYC 放行）"
```

---

## Task 6: 引擎 Engine + 封禁 fast-path + fail-open

**Files:**
- Create: `backend/auction/service/antifraud/engine.go`
- Create: `backend/auction/service/antifraud/engine_test.go`

- [ ] **Step 1: 写失败测试（U4/U11 + 短路）**

```go
// backend/auction/service/antifraud/engine_test.go
package antifraud

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// 桩：始终命中 block
type stubBlockRule struct{ id string }

func (s *stubBlockRule) ID() string { return s.id }
func (s *stubBlockRule) Check(ctx context.Context, evt *BidEvent) (*RiskDecision, error) {
	return &RiskDecision{Level: LevelHigh, Action: ActionBlock, Rules: []string{s.id}}, nil
}

type stubPassRule struct{ id string; called *int }

func (s *stubPassRule) ID() string { return s.id }
func (s *stubPassRule) Check(ctx context.Context, evt *BidEvent) (*RiskDecision, error) {
	*s.called++
	return PassDecision(), nil
}

type stubErrRule struct{}

func (s *stubErrRule) ID() string { return "err" }
func (s *stubErrRule) Check(ctx context.Context, evt *BidEvent) (*RiskDecision, error) {
	return nil, errors.New("boom")
}

func TestEngine_ShortCircuit_OnBlock(t *testing.T) {
	cli, _ := newTestRedis(t)
	called := 0
	engine := NewEngine(cli, EngineOptions{
		Rules: []Rule{&stubBlockRule{id: "B"}, &stubPassRule{id: "P", called: &called}},
	})
	dec, err := engine.Evaluate(context.Background(), &BidEvent{UserID: 1, Amount: decimal.NewFromInt(1)})
	assert.NoError(t, err)
	assert.Equal(t, ActionBlock, dec.Action)
	assert.Equal(t, 0, called, "短路后第二条规则不应被调用")
}

func TestEngine_U4_BannedFastPath(t *testing.T) {
	cli, _ := newTestRedis(t)
	called := 0
	engine := NewEngine(cli, EngineOptions{
		Rules: []Rule{&stubPassRule{id: "P", called: &called}},
	})
	cli.Set(context.Background(), "antifraud:ban:42", "1", 10*time.Minute)
	dec, err := engine.Evaluate(context.Background(), &BidEvent{UserID: 42, Amount: decimal.NewFromInt(1)})
	assert.NoError(t, err)
	assert.Equal(t, ActionBlock, dec.Action)
	assert.Equal(t, LevelCritical, dec.Level)
	assert.Contains(t, dec.Rules, RuleBanned)
	assert.Equal(t, 0, called, "封禁 fast-path 不应进入规则链")
}

func TestEngine_U11_FailOpen_OnRuleError(t *testing.T) {
	cli, _ := newTestRedis(t)
	engine := NewEngine(cli, EngineOptions{
		Rules: []Rule{&stubErrRule{}},
	})
	dec, err := engine.Evaluate(context.Background(), &BidEvent{UserID: 1, Amount: decimal.NewFromInt(1)})
	assert.NoError(t, err, "fail-open 不向上抛错")
	assert.Equal(t, ActionPass, dec.Action)
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
cd backend/auction && go test ./service/antifraud/... -run TestEngine -v
```

Expected: FAIL。

- [ ] **Step 3: 实现引擎**

```go
// backend/auction/service/antifraud/engine.go
package antifraud

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

// EngineOptions Engine 装配选项
type EngineOptions struct {
	Rules     []Rule
	Explainer RiskExplainer // 可选
	OnError   func(stage string, err error)
}

// Engine 风控引擎
type Engine struct {
	rdb       *redis.Client
	rules     []Rule
	explainer RiskExplainer
	onError   func(stage string, err error)
}

func NewEngine(rdb *redis.Client, opts EngineOptions) *Engine {
	onErr := opts.OnError
	if onErr == nil {
		onErr = func(stage string, err error) {
			log.Printf("[antifraud] %s error: %v", stage, err)
		}
	}
	return &Engine{
		rdb:       rdb,
		rules:     opts.Rules,
		explainer: opts.Explainer,
		onError:   onErr,
	}
}

// Evaluate 执行规则链；任何错误都 fail-open（返回 pass）
func (e *Engine) Evaluate(ctx context.Context, evt *BidEvent) (*RiskDecision, error) {
	// 0. 封禁 fast-path
	if e.isBanned(ctx, evt.UserID) {
		return &RiskDecision{
			Level:  LevelCritical,
			Action: ActionBlock,
			Rules:  []string{RuleBanned},
			Reason: "账号已临时封禁，请稍后再试",
		}, nil
	}
	// 1. 规则链短路
	for _, rule := range e.rules {
		dec, err := rule.Check(ctx, evt)
		if err != nil {
			e.onError(rule.ID(), err)
			continue // fail-open
		}
		if dec.Action != ActionPass {
			return dec, nil
		}
	}
	return PassDecision(), nil
}

func (e *Engine) isBanned(ctx context.Context, userID int64) bool {
	key := fmt.Sprintf("antifraud:ban:%d", userID)
	v, err := e.rdb.Get(ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			e.onError("isBanned", err)
		}
		return false
	}
	return v == "1"
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
cd backend/auction && go test ./service/antifraud/... -run TestEngine -v
```

Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/service/antifraud/engine.go backend/auction/service/antifraud/engine_test.go
git commit -m "feat(antifraud): Engine 编排（封禁 fast-path + 短路 + fail-open）"
```

---

## Task 7: RiskEvent 模型 + DAO

**Files:**
- Create: `backend/auction/model/risk_event.go`
- Create: `backend/auction/dao/risk_event.go`
- Create: `backend/auction/dao/risk_event_test.go`

- [ ] **Step 1: 写失败测试**

```go
// backend/auction/dao/risk_event_test.go
package dao

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"auction-service/model"
)

func newMockGorm(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	assert.NoError(t, err)
	return gdb, mock
}

func TestRiskEventDAO_Create(t *testing.T) {
	gdb, mock := newMockGorm(t)
	dao := NewRiskEventDAO(gdb)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `risk_events` (`user_id`,`auction_id`,`bid_id`,`rules`,`level`,`action`,`features`,`explanation`,`created_at`) VALUES (?,?,?,?,?,?,?,?,?)").
		WithArgs(int64(1), int64(2), nil, "R1_rapid_fire", "high", "block", "{}", "", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(10, 1))
	mock.ExpectCommit()

	evt := &model.RiskEvent{
		UserID:    1,
		AuctionID: 2,
		Rules:     "R1_rapid_fire",
		Level:     "high",
		Action:    "block",
		Features:  "{}",
		CreatedAt: time.Now(),
	}
	err := dao.Create(context.Background(), evt)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
cd backend/auction && go test ./dao/... -run TestRiskEventDAO -v
```

Expected: FAIL，`undefined: model.RiskEvent`。

- [ ] **Step 3: 实现 model + DAO**

```go
// backend/auction/model/risk_event.go
package model

import "time"

// RiskEvent 风控事件
type RiskEvent struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID      int64     `gorm:"index;not null" json:"user_id"`
	AuctionID   int64     `gorm:"index;not null" json:"auction_id"`
	BidID       *int64    `gorm:"index" json:"bid_id,omitempty"`
	Rules       string    `gorm:"type:varchar(255);not null" json:"rules"`
	Level       string    `gorm:"type:varchar(16);not null;index" json:"level"`
	Action      string    `gorm:"type:varchar(16);not null" json:"action"`
	Features    string    `gorm:"type:json" json:"features"`
	Explanation string    `gorm:"type:text" json:"explanation"`
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}

func (RiskEvent) TableName() string { return "risk_events" }
```

```go
// backend/auction/dao/risk_event.go
package dao

import (
	"context"

	"gorm.io/gorm"

	"auction-service/model"
)

// RiskEventDAO 风控事件 DAO
type RiskEventDAO struct {
	db *gorm.DB
}

func NewRiskEventDAO(db *gorm.DB) *RiskEventDAO {
	return &RiskEventDAO{db: db}
}

// Create 写入一条风控事件
func (d *RiskEventDAO) Create(ctx context.Context, evt *model.RiskEvent) error {
	return d.db.WithContext(ctx).Create(evt).Error
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
cd backend/auction && go test ./dao/... -run TestRiskEventDAO -v
```

Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/model/risk_event.go backend/auction/dao/risk_event.go backend/auction/dao/risk_event_test.go
git commit -m "feat(antifraud): RiskEvent 模型与 DAO 层（GORM + sqlmock 单测）"
```

---

## Task 8: Engine 持久化 RiskEvent + 写入策略

**Files:**
- Modify: `backend/auction/service/antifraud/engine.go`
- Modify: `backend/auction/service/antifraud/engine_test.go`

- [ ] **Step 1: 写失败测试**

```go
// 追加到 engine_test.go
type fakeRiskEventSink struct {
	calls []*RiskEventLog
}

func (f *fakeRiskEventSink) Persist(ctx context.Context, log *RiskEventLog) error {
	f.calls = append(f.calls, log)
	return nil
}

func TestEngine_PersistOnBlock(t *testing.T) {
	cli, _ := newTestRedis(t)
	sink := &fakeRiskEventSink{}
	engine := NewEngine(cli, EngineOptions{
		Rules: []Rule{&stubBlockRule{id: "R1_rapid_fire"}},
		Sink:  sink,
	})
	_, err := engine.Evaluate(context.Background(), &BidEvent{UserID: 7, AuctionID: 8, Amount: decimal.NewFromInt(1)})
	assert.NoError(t, err)
	assert.Len(t, sink.calls, 1)
	assert.Equal(t, "block", sink.calls[0].Action)
	assert.Equal(t, int64(7), sink.calls[0].UserID)
}

func TestEngine_NoPersistOnPass(t *testing.T) {
	cli, _ := newTestRedis(t)
	sink := &fakeRiskEventSink{}
	called := 0
	engine := NewEngine(cli, EngineOptions{
		Rules: []Rule{&stubPassRule{id: "P", called: &called}},
		Sink:  sink,
	})
	_, _ = engine.Evaluate(context.Background(), &BidEvent{UserID: 1, Amount: decimal.NewFromInt(1)})
	assert.Empty(t, sink.calls, "pass 不持久化")
}
```

- [ ] **Step 2: 运行测试验证失败**

Expected: FAIL，`undefined: RiskEventSink`。

- [ ] **Step 3: 扩展 Engine**

在 `types.go` 末尾追加：

```go
// RiskEventLog 持久化结构（与 model.RiskEvent 解耦）
type RiskEventLog struct {
	UserID    int64
	AuctionID int64
	BidID     *int64
	Rules     []string
	Level     string
	Action    string
	Features  map[string]any
}

// RiskEventSink 持久化接口（由 DAO 适配实现）
type RiskEventSink interface {
	Persist(ctx context.Context, log *RiskEventLog) error
}
```

修改 `engine.go`：

```go
// 在 EngineOptions 中追加：
type EngineOptions struct {
	Rules     []Rule
	Explainer RiskExplainer
	Sink      RiskEventSink
	OnError   func(stage string, err error)
}

// Engine 字段追加 sink RiskEventSink

// Evaluate 末尾在返回前增加持久化
// （在 isBanned 命中后 与 命中规则后 都调用 e.persist）
```

完整修改 `engine.go`：

```go
package antifraud

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

type EngineOptions struct {
	Rules     []Rule
	Explainer RiskExplainer
	Sink      RiskEventSink
	OnError   func(stage string, err error)
}

type Engine struct {
	rdb       *redis.Client
	rules     []Rule
	explainer RiskExplainer
	sink      RiskEventSink
	onError   func(stage string, err error)
}

func NewEngine(rdb *redis.Client, opts EngineOptions) *Engine {
	onErr := opts.OnError
	if onErr == nil {
		onErr = func(stage string, err error) {
			log.Printf("[antifraud] %s error: %v", stage, err)
		}
	}
	return &Engine{
		rdb:       rdb,
		rules:     opts.Rules,
		explainer: opts.Explainer,
		sink:      opts.Sink,
		onError:   onErr,
	}
}

func (e *Engine) Evaluate(ctx context.Context, evt *BidEvent) (*RiskDecision, error) {
	if e.isBanned(ctx, evt.UserID) {
		dec := &RiskDecision{
			Level:  LevelCritical,
			Action: ActionBlock,
			Rules:  []string{RuleBanned},
			Reason: "账号已临时封禁，请稍后再试",
		}
		e.persist(ctx, evt, dec, true)
		return dec, nil
	}
	for _, rule := range e.rules {
		dec, err := rule.Check(ctx, evt)
		if err != nil {
			e.onError(rule.ID(), err)
			continue
		}
		if dec.Action != ActionPass {
			e.persist(ctx, evt, dec, dec.Action == ActionMark) // mark 异步；challenge/block 同步
			return dec, nil
		}
	}
	return PassDecision(), nil
}

func (e *Engine) isBanned(ctx context.Context, userID int64) bool {
	key := fmt.Sprintf("antifraud:ban:%d", userID)
	v, err := e.rdb.Get(ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			e.onError("isBanned", err)
		}
		return false
	}
	return v == "1"
}

// persist 持久化风控事件；async=true 时异步写
func (e *Engine) persist(ctx context.Context, evt *BidEvent, dec *RiskDecision, async bool) {
	if e.sink == nil {
		return
	}
	logEntry := &RiskEventLog{
		UserID:    evt.UserID,
		AuctionID: evt.AuctionID,
		Rules:     dec.Rules,
		Level:     dec.Level,
		Action:    dec.Action,
		Features:  dec.Features,
	}
	if async {
		go func() {
			if err := e.sink.Persist(context.Background(), logEntry); err != nil {
				e.onError("sink.Persist", err)
			}
		}()
		return
	}
	if err := e.sink.Persist(ctx, logEntry); err != nil {
		e.onError("sink.Persist", err)
	}
}
```

注意：block/challenge 的 `async=false` 因为 `dec.Action == ActionMark` 才为 true，但 Mark 不会出现在当前 R1/R4/R5 规则集中——保留参数语义供后续 mark 类规则使用。Banned fast-path 用 sync 写入。

- [ ] **Step 4: 运行测试验证通过**

```bash
cd backend/auction && go test ./service/antifraud/... -v
```

Expected: PASS（所有用例）。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/service/antifraud/engine.go backend/auction/service/antifraud/engine_test.go backend/auction/service/antifraud/types.go
git commit -m "feat(antifraud): Engine 增加 RiskEventSink，按 action 选择同步/异步写入"
```

---

## Task 9: DefaultRules 装配 + DAO Sink 适配

**Files:**
- Create: `backend/auction/service/antifraud/rules.go`
- Create: `backend/auction/service/antifraud/sink_dao.go`
- Create: `backend/auction/service/antifraud/rules_test.go`

- [ ] **Step 1: 写失败测试**

```go
// backend/auction/service/antifraud/rules_test.go
package antifraud

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultRules_HasThreeRules(t *testing.T) {
	cli, _ := newTestRedis(t)
	loader := &fakeUserLoader{}
	priceLoader := &fakeLoader{}
	rules := DefaultRules(cli, priceLoader, loader)
	ids := make(map[string]bool)
	for _, r := range rules {
		ids[r.ID()] = true
	}
	assert.True(t, ids[RuleRapidFire])
	assert.True(t, ids[RuleAbnormalJump])
	assert.True(t, ids[RuleFreshAccount])
	assert.Len(t, rules, 3)
}
```

- [ ] **Step 2: 运行测试验证失败**

Expected: FAIL，`undefined: DefaultRules`。

- [ ] **Step 3: 实现装配**

```go
// backend/auction/service/antifraud/rules.go
package antifraud

import "github.com/redis/go-redis/v9"

// DefaultRules 装配 R1/R4/R5 三条规则（按短路顺序）
func DefaultRules(rdb *redis.Client, priceLoader AuctionPriceLoader, userLoader UserInfoLoader) []Rule {
	return []Rule{
		NewRapidFireRule(rdb, DefaultRapidFireConfig()),
		NewAbnormalJumpRule(priceLoader, DefaultAbnormalJumpConfig()),
		NewFreshAccountRule(rdb, userLoader, DefaultFreshAccountConfig()),
	}
}
```

```go
// backend/auction/service/antifraud/sink_dao.go
package antifraud

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"auction-service/dao"
	"auction-service/model"
)

// DAOSink 把 RiskEventLog 写入 RiskEventDAO
type DAOSink struct {
	dao *dao.RiskEventDAO
}

func NewDAOSink(d *dao.RiskEventDAO) *DAOSink {
	return &DAOSink{dao: d}
}

func (s *DAOSink) Persist(ctx context.Context, log *RiskEventLog) error {
	featuresBytes, _ := json.Marshal(log.Features)
	return s.dao.Create(ctx, &model.RiskEvent{
		UserID:    log.UserID,
		AuctionID: log.AuctionID,
		BidID:     log.BidID,
		Rules:     strings.Join(log.Rules, ","),
		Level:     log.Level,
		Action:    log.Action,
		Features:  string(featuresBytes),
		CreatedAt: time.Now(),
	})
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
cd backend/auction && go test ./service/antifraud/... -v
```

Expected: PASS（所有用例）。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/service/antifraud/rules.go backend/auction/service/antifraud/sink_dao.go backend/auction/service/antifraud/rules_test.go
git commit -m "feat(antifraud): DefaultRules 装配 + DAOSink 适配 RiskEventDAO"
```

---

## Task 10: Prometheus 指标

**Files:**
- Create: `backend/auction/pkg/metrics/antifraud_metrics.go`
- Modify: `backend/auction/service/antifraud/engine.go`
- Modify: `backend/auction/service/antifraud/engine_test.go`

- [ ] **Step 1: 写失败测试**

```go
// 追加到 engine_test.go
type fakeMetrics struct {
	evaluations map[string]int
	hits        map[string]int
	errors      map[string]int
}

func newFakeMetrics() *fakeMetrics {
	return &fakeMetrics{
		evaluations: map[string]int{},
		hits:        map[string]int{},
		errors:      map[string]int{},
	}
}
func (f *fakeMetrics) IncEvaluation(result string)     { f.evaluations[result]++ }
func (f *fakeMetrics) IncRuleHit(ruleID string)        { f.hits[ruleID]++ }
func (f *fakeMetrics) ObserveDuration(d time.Duration) {}
func (f *fakeMetrics) IncError(stage string)           { f.errors[stage]++ }

func TestEngine_Metrics_OnBlock(t *testing.T) {
	cli, _ := newTestRedis(t)
	m := newFakeMetrics()
	engine := NewEngine(cli, EngineOptions{
		Rules:   []Rule{&stubBlockRule{id: "R1_rapid_fire"}},
		Metrics: m,
	})
	_, _ = engine.Evaluate(context.Background(), &BidEvent{UserID: 1, Amount: decimal.NewFromInt(1)})
	assert.Equal(t, 1, m.evaluations["block"])
	assert.Equal(t, 1, m.hits["R1_rapid_fire"])
}

func TestEngine_Metrics_OnRuleError(t *testing.T) {
	cli, _ := newTestRedis(t)
	m := newFakeMetrics()
	engine := NewEngine(cli, EngineOptions{
		Rules:   []Rule{&stubErrRule{}},
		Metrics: m,
	})
	_, _ = engine.Evaluate(context.Background(), &BidEvent{UserID: 1, Amount: decimal.NewFromInt(1)})
	assert.Equal(t, 1, m.errors["err"])
	assert.Equal(t, 1, m.evaluations["pass"])
}
```

- [ ] **Step 2: 运行测试验证失败**

Expected: FAIL，`undefined: Metrics`。

- [ ] **Step 3: 实现接口与 Prometheus 实现**

```go
// 追加到 backend/auction/service/antifraud/types.go
type Metrics interface {
	IncEvaluation(result string)
	IncRuleHit(ruleID string)
	ObserveDuration(d time.Duration)
	IncError(stage string)
}
```

修改 `engine.go`：

```go
// EngineOptions 增加字段：
type EngineOptions struct {
	Rules     []Rule
	Explainer RiskExplainer
	Sink      RiskEventSink
	Metrics   Metrics
	OnError   func(stage string, err error)
}

// Engine 字段增加 metrics Metrics

// 改写 onError 默认值，支持 Metrics.IncError；改写 Evaluate 入口记录起始时间，结束记录耗时；
// 在 isBanned 命中后调用 m.IncEvaluation("block") + IncRuleHit("banned")；
// 在 rule 命中后调用 IncEvaluation(action) + IncRuleHit(rule.ID())；
// 在 pass 路径调用 IncEvaluation("pass")。
```

完整新版 `engine.go`：

```go
package antifraud

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type EngineOptions struct {
	Rules     []Rule
	Explainer RiskExplainer
	Sink      RiskEventSink
	Metrics   Metrics
	OnError   func(stage string, err error)
}

type Engine struct {
	rdb       *redis.Client
	rules     []Rule
	explainer RiskExplainer
	sink      RiskEventSink
	metrics   Metrics
	onError   func(stage string, err error)
}

func NewEngine(rdb *redis.Client, opts EngineOptions) *Engine {
	m := opts.Metrics
	onErr := opts.OnError
	if onErr == nil {
		onErr = func(stage string, err error) {
			log.Printf("[antifraud] %s error: %v", stage, err)
			if m != nil {
				m.IncError(stage)
			}
		}
	}
	return &Engine{
		rdb: rdb, rules: opts.Rules, explainer: opts.Explainer,
		sink: opts.Sink, metrics: m, onError: onErr,
	}
}

func (e *Engine) Evaluate(ctx context.Context, evt *BidEvent) (*RiskDecision, error) {
	start := time.Now()
	defer func() {
		if e.metrics != nil {
			e.metrics.ObserveDuration(time.Since(start))
		}
	}()

	if e.isBanned(ctx, evt.UserID) {
		dec := &RiskDecision{Level: LevelCritical, Action: ActionBlock, Rules: []string{RuleBanned}, Reason: "账号已临时封禁，请稍后再试"}
		e.recordMetrics(dec)
		e.persist(ctx, evt, dec, true)
		return dec, nil
	}
	for _, rule := range e.rules {
		dec, err := rule.Check(ctx, evt)
		if err != nil {
			e.onError(rule.ID(), err)
			continue
		}
		if dec.Action != ActionPass {
			e.recordMetrics(dec)
			e.persist(ctx, evt, dec, dec.Action == ActionMark)
			return dec, nil
		}
	}
	if e.metrics != nil {
		e.metrics.IncEvaluation(ActionPass)
	}
	return PassDecision(), nil
}

func (e *Engine) recordMetrics(dec *RiskDecision) {
	if e.metrics == nil {
		return
	}
	e.metrics.IncEvaluation(dec.Action)
	for _, id := range dec.Rules {
		e.metrics.IncRuleHit(id)
	}
}

func (e *Engine) isBanned(ctx context.Context, userID int64) bool {
	key := fmt.Sprintf("antifraud:ban:%d", userID)
	v, err := e.rdb.Get(ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			e.onError("isBanned", err)
		}
		return false
	}
	return v == "1"
}

func (e *Engine) persist(ctx context.Context, evt *BidEvent, dec *RiskDecision, async bool) {
	if e.sink == nil {
		return
	}
	logEntry := &RiskEventLog{
		UserID: evt.UserID, AuctionID: evt.AuctionID,
		Rules: dec.Rules, Level: dec.Level, Action: dec.Action, Features: dec.Features,
	}
	if async {
		go func() {
			if err := e.sink.Persist(context.Background(), logEntry); err != nil {
				e.onError("sink.Persist", err)
			}
		}()
		return
	}
	if err := e.sink.Persist(ctx, logEntry); err != nil {
		e.onError("sink.Persist", err)
	}
}
```

```go
// backend/auction/pkg/metrics/antifraud_metrics.go
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// AntifraudMetrics Prometheus 实现
type AntifraudMetrics struct {
	evaluations *prometheus.CounterVec
	hits        *prometheus.CounterVec
	duration    prometheus.Histogram
	errors      *prometheus.CounterVec
}

func NewAntifraudMetrics() *AntifraudMetrics {
	return &AntifraudMetrics{
		evaluations: promauto.With(GetRegistry()).NewCounterVec(prometheus.CounterOpts{
			Name: "antifraud_evaluations_total",
			Help: "Total antifraud evaluations by result",
		}, []string{"result"}),
		hits: promauto.With(GetRegistry()).NewCounterVec(prometheus.CounterOpts{
			Name: "antifraud_rule_hits_total",
			Help: "Hits per antifraud rule",
		}, []string{"rule_id"}),
		duration: promauto.With(GetRegistry()).NewHistogram(prometheus.HistogramOpts{
			Name: "antifraud_eval_duration_seconds", Help: "Evaluation latency",
			Buckets: prometheus.DefBuckets,
		}),
		errors: promauto.With(GetRegistry()).NewCounterVec(prometheus.CounterOpts{
			Name: "antifraud_engine_errors_total",
			Help: "Engine internal errors by stage",
		}, []string{"stage"}),
	}
}

func (m *AntifraudMetrics) IncEvaluation(result string)     { m.evaluations.WithLabelValues(result).Inc() }
func (m *AntifraudMetrics) IncRuleHit(ruleID string)        { m.hits.WithLabelValues(ruleID).Inc() }
func (m *AntifraudMetrics) ObserveDuration(d time.Duration) { m.duration.Observe(d.Seconds()) }
func (m *AntifraudMetrics) IncError(stage string)           { m.errors.WithLabelValues(stage).Inc() }
```

> **注**：`GetRegistry()` 已在现有 `backend/auction/pkg/metrics/` 包中存在（Task 实施时若发现命名不同，按现有命名调整）；如不存在，使用 `prometheus.NewRegistry()` 包级单例。

- [ ] **Step 4: 运行测试验证通过**

```bash
cd backend/auction && go test ./service/antifraud/... ./pkg/metrics/... -v
```

Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/service/antifraud/engine.go backend/auction/service/antifraud/engine_test.go backend/auction/service/antifraud/types.go backend/auction/pkg/metrics/antifraud_metrics.go
git commit -m "feat(antifraud): 增加 Metrics 接口 + Prometheus 4 个指标"
```

---

## Task 11: BidService 集成 antifraud + Confirmed 字段

**Files:**
- Modify: `backend/auction/service/bid.go`
- Modify: `backend/auction/handler/bid.go`
- Modify: `backend/auction/service/bid_test.go`

- [ ] **Step 1: 写失败测试（I3 + 集成）**

```go
// 追加到 backend/auction/service/bid_test.go
package service

import (
	// ... 既有 import
	"auction-service/service/antifraud"
)

// 桩：始终 block 的引擎
type stubAntifraudEngine struct {
	called int
	dec    *antifraud.RiskDecision
}

func (s *stubAntifraudEngine) Evaluate(ctx context.Context, evt *antifraud.BidEvent) (*antifraud.RiskDecision, error) {
	s.called++
	return s.dec, nil
}

func TestPlaceBidRequest_HasConfirmedAndSkipFields(t *testing.T) {
	req := PlaceBidRequest{Confirmed: true, SkipAntifraud: true}
	assert.True(t, req.Confirmed)
	assert.True(t, req.SkipAntifraud)
}

func TestPlaceBidResult_HasRiskCode(t *testing.T) {
	res := PlaceBidResult{RiskCode: "risk_rapid_fire"}
	assert.Equal(t, "risk_rapid_fire", res.RiskCode)
}

func TestMapRiskCode(t *testing.T) {
	assert.Equal(t, "risk_rapid_fire", mapRiskCode(&antifraud.RiskDecision{
		Action: antifraud.ActionBlock, Rules: []string{antifraud.RuleRapidFire},
	}))
	assert.Equal(t, "risk_confirm_required", mapRiskCode(&antifraud.RiskDecision{
		Action: antifraud.ActionChallenge, Rules: []string{antifraud.RuleAbnormalJump},
	}))
	assert.Equal(t, "risk_kyc_required", mapRiskCode(&antifraud.RiskDecision{
		Action: antifraud.ActionBlock, Rules: []string{antifraud.RuleFreshAccount},
	}))
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
cd backend/auction && go test ./service/... -run "TestPlaceBidRequest_HasConfirmed|TestPlaceBidResult_HasRiskCode|TestMapRiskCode" -v
```

Expected: FAIL（字段未定义）。

- [ ] **Step 3: 修改 BidService**

修改 [bid.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/service/bid.go) — 在 `BidService` struct、`PlaceBidRequest`、`PlaceBidResult` 与 `PlaceBid` 主链路上加入 antifraud：

```go
// 文件顶部 import 增加：
import (
	// ... 既有
	"auction-service/service/antifraud"
)

// AntifraudEngine 出价反作弊引擎（接口，便于测试桩）
type AntifraudEngine interface {
	Evaluate(ctx context.Context, evt *antifraud.BidEvent) (*antifraud.RiskDecision, error)
}

// BidService struct 增加：
type BidService struct {
	// ... 既有字段
	antifraudEngine AntifraudEngine
}

// 增加 setter
func (s *BidService) SetAntifraudEngine(e AntifraudEngine) {
	s.antifraudEngine = e
}

// PlaceBidRequest 增加字段
type PlaceBidRequest struct {
	AuctionID          int64
	UserID             int64
	Amount             decimal.Decimal
	SkipSkyLampTrigger bool
	Confirmed          bool // 用户在 R4 challenge 后二次确认
	SkipAntifraud      bool // 内部场景（如点天灯自动跟价）跳过风控
}

// PlaceBidResult 增加字段
type PlaceBidResult struct {
	Success      bool            `json:"success"`
	Message      string          `json:"message"`
	CurrentPrice decimal.Decimal `json:"current_price"`
	Rank         int             `json:"rank"`
	WinnerID     int64           `json:"winner_id"`
	RiskCode     string          `json:"risk_code,omitempty"`
}

// mapRiskCode 把 RiskDecision 映射成前端可识别的错误码
func mapRiskCode(dec *antifraud.RiskDecision) string {
	for _, id := range dec.Rules {
		switch id {
		case antifraud.RuleRapidFire, antifraud.RuleBanned:
			return "risk_rapid_fire"
		case antifraud.RuleAbnormalJump:
			return "risk_confirm_required"
		case antifraud.RuleFreshAccount:
			return "risk_kyc_required"
		}
	}
	return "risk_unknown"
}
```

在 `PlaceBid` 函数 step 1（用户校验之后、step 2 GetByID 之前）插入：

```go
	// 1.5 反作弊判定（新增）
	if s.antifraudEngine != nil && !req.SkipAntifraud {
		evt := &antifraud.BidEvent{
			UserID:    req.UserID,
			AuctionID: req.AuctionID,
			Amount:    req.Amount,
			Timestamp: time.Now(),
			Confirmed: req.Confirmed,
		}
		dec, err := s.antifraudEngine.Evaluate(ctx, evt)
		if err == nil && (dec.Action == antifraud.ActionBlock || dec.Action == antifraud.ActionChallenge) {
			return &PlaceBidResult{
				Success:  false,
				Message:  dec.Reason,
				RiskCode: mapRiskCode(dec),
			}, nil
		}
	}
```

修改 `handler/bid.go` — `PlaceBidRequest` 增加 `Confirmed` 字段并透传：

```go
type PlaceBidRequest struct {
	Amount    decimal.Decimal `json:"amount" binding:"required"`
	UserID    int64           `json:"user_id,omitempty"`
	Confirmed bool            `json:"confirmed,omitempty"`
}
```

在 `PlaceBid` handler 调用 `bidService.PlaceBid` 时把 `req.Confirmed` 透传给 `service.PlaceBidRequest.Confirmed`。

`PlaceBidResult` 在 handler 序列化时已自动带上 `risk_code`。命中 `block/challenge` 时 handler 当前返回 200 + `success=false`，本 spec 4.3 描述了 HTTP 429/400/403。修改 handler 的响应分支：

```go
// handler/bid.go PlaceBid 末尾：
result, err := h.bidService.PlaceBid(ctx, &service.PlaceBidRequest{
	AuctionID: auctionID, UserID: userID, Amount: req.Amount, Confirmed: req.Confirmed,
})
if err != nil {
	c.JSON(500, map[string]interface{}{"code": 500, "message": err.Error()})
	return
}
if !result.Success && result.RiskCode != "" {
	status := 200
	switch result.RiskCode {
	case "risk_rapid_fire":
		status = 429
	case "risk_confirm_required":
		status = 400
	case "risk_kyc_required":
		status = 403
	}
	c.JSON(status, map[string]interface{}{
		"code":      status,
		"message":   result.Message,
		"risk_code": result.RiskCode,
		"success":   false,
	})
	return
}
c.JSON(200, result)
```

- [ ] **Step 4: 运行测试验证通过**

```bash
cd backend/auction && go test ./service/... ./handler/... -v
```

Expected: PASS（包括既有 `TestBidService_PlaceBidRequest_Validation` 仍然通过）。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/service/bid.go backend/auction/service/bid_test.go backend/auction/handler/bid.go
git commit -m "feat(antifraud): BidService 接入 antifraud.Engine + handler 错误码映射"
```

---

## Task 12: main.go 装配 + AutoMigrate

**Files:**
- Modify: `backend/auction/main.go`

- [ ] **Step 1: 编译型校验（手动）**

main.go 是装配代码，主要靠编译 + 现有集成测试覆盖。无需为 main 写专用测试，但需确保改后整体可编译。

- [ ] **Step 2: 修改 main.go**

修改 [main.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/main.go)：

1. import 增加：

```go
import (
	// ... 既有
	"auction-service/service/antifraud"
)
```

2. `AutoMigrate` 列表增加 `&model.RiskEvent{}`：

```go
if err := db.AutoMigrate(
	&model.Auction{},
	&model.Bid{},
	&model.Notification{},
	&model.UserLiveStreamFollow{},
	&model.UserProductReminder{},
	&model.SkyLampSubscription{},
	&model.UserBalance{},
	&model.UserAddress{},
	&model.RiskEvent{}, // 新增
); err != nil {
	log.Printf("Warning: AutoMigrate failed (tables may already exist): %v", err)
}
```

3. DAO 装配处增加：

```go
riskEventDAO := dao.NewRiskEventDAO(db)
```

4. 在 `bidService` 装配后增加 antifraud 引擎装配（紧跟 `bidService.SetMetrics(metrics.GetMetrics())`）：

```go
// 反作弊引擎装配
antifraudPriceLoader := antifraud.NewAuctionPriceLoaderFromDAO(auctionDAO, ruleDAO)
antifraudUserLoader := antifraud.NewUserInfoLoaderFromDAO(userDAO)
antifraudEngine := antifraud.NewEngine(dao.GetRedis(), antifraud.EngineOptions{
	Rules:   antifraud.DefaultRules(dao.GetRedis(), antifraudPriceLoader, antifraudUserLoader),
	Sink:    antifraud.NewDAOSink(riskEventDAO),
	Metrics: metrics.NewAntifraudMetrics(),
})
bidService.SetAntifraudEngine(antifraudEngine)
log.Println("Antifraud engine initialized")
```

5. 配套创建两个 DAO 适配器（在 antifraud 包内补一个文件）：

```go
// backend/auction/service/antifraud/loaders.go
package antifraud

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"auction-service/dao"
)

// AuctionPriceLoaderFromDAO 适配 AuctionDAO + AuctionRuleDAO
type AuctionPriceLoaderFromDAO struct {
	auctionDAO *dao.AuctionDAO
	ruleDAO    *dao.AuctionRuleDAO
}

func NewAuctionPriceLoaderFromDAO(a *dao.AuctionDAO, r *dao.AuctionRuleDAO) *AuctionPriceLoaderFromDAO {
	return &AuctionPriceLoaderFromDAO{auctionDAO: a, ruleDAO: r}
}

func (l *AuctionPriceLoaderFromDAO) Load(ctx context.Context, auctionID int64) (decimal.Decimal, decimal.Decimal, error) {
	auc, err := l.auctionDAO.GetByID(ctx, auctionID)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	rule, err := l.ruleDAO.GetByProductID(ctx, auc.ProductID)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	return auc.CurrentPrice, rule.Increment, nil
}

// UserInfoLoaderFromDAO 适配 UserDAO；MVP 阶段 KYC 始终返回 false
type UserInfoLoaderFromDAO struct {
	userDAO *dao.UserDAO
}

func NewUserInfoLoaderFromDAO(u *dao.UserDAO) *UserInfoLoaderFromDAO {
	return &UserInfoLoaderFromDAO{userDAO: u}
}

func (l *UserInfoLoaderFromDAO) Load(ctx context.Context, userID int64) (time.Time, bool, error) {
	u, err := l.userDAO.GetByID(ctx, userID)
	if err != nil {
		return time.Time{}, false, err
	}
	return u.CreatedAt, false, nil
}
```

- [ ] **Step 3: 编译验证**

```bash
cd backend/auction && go build ./...
```

Expected: 0 错误。

- [ ] **Step 4: 全量测试**

```bash
cd backend/auction && go test ./... -count=1
```

Expected: 全部 PASS。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/main.go backend/auction/service/antifraud/loaders.go
git commit -m "feat(antifraud): main.go 装配 antifraud.Engine + DAO Loader 适配器"
```

---

## Task 13: 端到端集成测试 I1/I2

**Files:**
- Create: `backend/auction/service/antifraud/integration_test.go`

- [ ] **Step 1: 写集成测试**

```go
// backend/auction/service/antifraud/integration_test.go
package antifraud

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// I1: R4 challenge → confirmed=true 重试 → 放行
func TestIntegration_I1_R4ChallengeThenConfirmed(t *testing.T) {
	cli, _ := newTestRedis(t)
	priceLoader := &fakeLoader{
		current:   decimal.NewFromInt(100),
		increment: decimal.NewFromInt(10),
	}
	userLoader := &fakeUserLoader{users: map[int64]fakeUserInfo{
		1: {createdAt: time.Now().Add(-72 * time.Hour), kyc: false},
	}}
	sink := &fakeRiskEventSink{}
	engine := NewEngine(cli, EngineOptions{
		Rules: DefaultRules(cli, priceLoader, userLoader),
		Sink:  sink,
	})

	// 第 1 次：异常加价 → challenge
	dec1, err := engine.Evaluate(context.Background(), &BidEvent{
		UserID: 1, AuctionID: 99, Amount: decimal.NewFromInt(2000), Timestamp: time.Now(),
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionChallenge, dec1.Action)

	// 第 2 次：confirmed=true → pass
	dec2, err := engine.Evaluate(context.Background(), &BidEvent{
		UserID: 1, AuctionID: 99, Amount: decimal.NewFromInt(2000), Confirmed: true, Timestamp: time.Now(),
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionPass, dec2.Action)
}

// I2: R1 block → 持久化 1 条 risk_event
func TestIntegration_I2_R1BlockPersisted(t *testing.T) {
	cli, _ := newTestRedis(t)
	priceLoader := &fakeLoader{current: decimal.NewFromInt(100), increment: decimal.NewFromInt(10)}
	userLoader := &fakeUserLoader{users: map[int64]fakeUserInfo{
		1: {createdAt: time.Now().Add(-72 * time.Hour), kyc: false},
	}}
	sink := &fakeRiskEventSink{}
	engine := NewEngine(cli, EngineOptions{
		Rules: DefaultRules(cli, priceLoader, userLoader),
		Sink:  sink,
	})

	// 8 次连续出价 → R1 block
	var lastDec *RiskDecision
	for i := 0; i < 8; i++ {
		lastDec, _ = engine.Evaluate(context.Background(), &BidEvent{
			UserID: 1, AuctionID: 1, Amount: decimal.NewFromInt(110), Timestamp: time.Now(),
		})
	}
	assert.Equal(t, ActionBlock, lastDec.Action)
	// 等异步 Mark 写入；当前是 block，sync 写入
	assert.GreaterOrEqual(t, len(sink.calls), 1)
	found := false
	for _, c := range sink.calls {
		if c.Action == ActionBlock {
			for _, r := range c.Rules {
				if r == RuleRapidFire {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "应包含 1 条 R1 block 持久化记录")
}
```

- [ ] **Step 2: 运行测试验证通过**

```bash
cd backend/auction && go test ./service/antifraud/... -run TestIntegration -v
```

Expected: PASS。

- [ ] **Step 3: 全量回归**

```bash
cd backend/auction && go test ./... -count=1 -race
```

Expected: 全部 PASS，无 race 警告。

- [ ] **Step 4: Commit**

```bash
git add backend/auction/service/antifraud/integration_test.go
git commit -m "test(antifraud): 端到端集成测试 I1/I2（R4 challenge 流转 + R1 block 持久化）"
```

---

## 自审清单

| 项 | 检查 |
|---|---|
| Spec §4.1 risk_event 表 | Task 7 实现 ✅ |
| Spec §4.2 BidEvent / RiskDecision / RiskExplainer | Task 1 实现 ✅ |
| Spec §4.3 错误码映射 | Task 11 mapRiskCode ✅ |
| Spec §5.1 R1 RapidFire | Task 3 ✅ |
| Spec §5.2 R4 AbnormalJump | Task 4 ✅ |
| Spec §5.3 R5 FreshAccount | Task 5 ✅ |
| Spec §5.4 短路执行 | Task 6 ✅ |
| Spec §6 LLM 接入位（接口预留，不注入实现） | Task 1 RiskExplainer ✅ |
| Spec §7.1 bid.go 第 0.2 步改造 | Task 11 ✅ |
| Spec §7.2 封禁 fast-path | Task 6 ✅ |
| Spec §7.3 写入策略（pass 不写 / mark 异步 / challenge+block 同步） | Task 8 ✅ |
| Spec §8 Prometheus 指标 | Task 10 ✅ |
| Spec §9.1 单元测试 U1-U11 | Task 3/4/5/6 共 11 用例 ✅ |
| Spec §9.2 集成测试 I1/I2/I3 | Task 13（I1/I2）+ Task 11（I3 由 SkipAntifraud 字段间接验证） |
| Spec §10 里程碑 M1-M5 | Task 1/6（M1）→ Task 3-5/9（M2）→ Task 7/8（M3）→ Task 11/12（M4）→ Task 10（M5）|

**类型一致性检查**：所有 Task 引用的类型/常量（`BidEvent`/`RiskDecision`/`RuleRapidFire`/`ActionBlock` 等）均在 Task 1/8 定义；`RiskEventSink`/`RiskEventLog` 在 Task 8 定义；`Metrics` 在 Task 10 定义。

---

## 关键依赖顺序

```
Task 1 (types) ──┬─► Task 3 (R1) ──┐
                 ├─► Task 4 (R4) ──┤
                 ├─► Task 5 (R5) ──┤
Task 2 (deps) ───┘                 │
                                   ▼
                                 Task 6 (engine)
                                   │
                                   ▼
                                 Task 7 (risk_event)
                                   │
                                   ▼
                                 Task 8 (sink in engine) ──► Task 9 (DefaultRules+DAOSink)
                                                                │
                                                                ▼
                                                              Task 10 (metrics)
                                                                │
                                                                ▼
                                                              Task 11 (bid.go 接入)
                                                                │
                                                                ▼
                                                              Task 12 (main.go)
                                                                │
                                                                ▼
                                                              Task 13 (端到端)
```

Task 3/4/5 互相独立，可并行；其余严格顺序。

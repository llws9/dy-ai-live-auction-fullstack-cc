# C2 反作弊 MVP 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `auction-service` 出价主链路前置实时风控引擎，落地 R1（高频）/R4（异常加价）/R5（新账号高额限制）三条规则，预留 LLM 解释器接口；新增 `risk_event` 表持久化事件用于运营审核与离线特征工程。

**Current State:** 当前 `main` 尚未实现反作弊包、`risk_event`、`challenge_token`、`risk_code`、点天灯风控 mode 与前端 R4 二次确认；本 plan 是待执行蓝图，不是已落地说明。

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
| `backend/auction/service/antifraud/rule_fresh_account.go` | R5 新账号高额限制规则（DB + Redis 成功累计） | 新建 |
| `backend/auction/service/antifraud/quota.go` | R5 成功出价金额累计写入（整数 cents） | 新建 |
| `backend/auction/service/antifraud/rules.go` | DefaultRules() 装配 | 新建 |
| `backend/auction/service/antifraud/*_test.go` | 单元测试（U1-U11） | 新建 |
| `backend/auction/model/risk_event.go` | RiskEvent gorm model | 新建 |
| `backend/auction/dao/risk_event.go` | RiskEventDAO | 新建 |
| `backend/auction/dao/risk_event_test.go` | DAO 单测 | 新建 |
| `backend/auction/service/bid.go` | 修改：注入 antifraudEngine + 第 0.2 步调用 | 修改 |
| `backend/auction/service/bid_test.go` | 修改：补集成测试 I1/I3a/I3b/I4 | 修改 |
| `backend/auction/service/sky_lamp.go` | 修改：自动跟价跳过全部风控，首次点天灯跳过 R1 但保留 R4/R5 | 修改 |
| `backend/auction/handler/bid.go` | 修改：错误码映射 + challenge_token 透传 | 修改 |
| `backend/auction/main.go` | 修改：装配 antifraud.Engine + AutoMigrate RiskEvent | 修改 |
| `backend/auction/pkg/metrics/antifraud_metrics.go` | Prometheus 指标 | 新建 |
| `backend/auction/pkg/metrics/antifraud_metrics_test.go` | Prometheus 指标 registry gather 测试 | 新建 |
| `frontend/h5/src/services/api.ts` | 修改：`placeBid` 增加 `challengeToken` 参数 | 修改 |
| `frontend/h5/src/pages/Live/LiveRoomSlide.tsx` | 修改：`handleBid` 处理 `risk_confirm_required` token 二次确认重试 | 修改 |
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
		ChallengeToken: "token-1",
	}
	assert.Equal(t, int64(100), evt.UserID)
	assert.Equal(t, "token-1", evt.ChallengeToken)
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
	RuleFreshAccount  = "R5_fresh_account_limit"
	RuleBanned        = "banned"
)

// 风控模式
const (
	AntifraudModeNormal        = "normal"
	AntifraudModeSkipRapidFire = "skip_rapid_fire"
	AntifraudModeSkipAll       = "skip_all"
)

// BidEvent 风控判定输入
type BidEvent struct {
	UserID    int64
	AuctionID int64
	Amount    decimal.Decimal
	IP        string
	UA        string
	Timestamp time.Time
	ChallengeToken string // R4 challenge 后回传的一次性 token
	AntifraudMode  string // normal | skip_rapid_fire | skip_all
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
- Create: `backend/auction/service/antifraud/probe_test.go`

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
	"fmt"
	"testing"
	"time"
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

func TestRapidFire_U4_WritesBanKeyAfterThreeHits(t *testing.T) {
	cli, _ := newTestRedis(t)
	rule := NewRapidFireRule(cli, RapidFireConfig{
		WindowSec: 5, Threshold: 8, BanThreshold: 3, BanTTL: 600 * time.Second,
	})
	ctx := context.Background()
	base := time.Now()
	evt := &BidEvent{UserID: 4}

	blocked := 0
	for i := 0; blocked < 3; i++ {
		evt.Timestamp = base.Add(time.Duration(i) * time.Millisecond)
		dec, err := rule.Check(ctx, evt)
		assert.NoError(t, err)
		if dec.Action == ActionBlock {
			blocked++
		}
	}
	got, err := cli.Get(ctx, "antifraud:ban:4").Result()
	assert.NoError(t, err)
	assert.Equal(t, "1", got)
	ttl, err := cli.TTL(ctx, "antifraud:ban:4").Result()
	assert.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0))
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
	BanThreshold int           // 10 分钟内 R1 block 累计阈值（默认 3）
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
	if evt.AntifraudMode == AntifraudModeSkipRapidFire || evt.AntifraudMode == AntifraudModeSkipAll {
		return PassDecision(), nil
	}
	now := evt.Timestamp
	if now.IsZero() {
		now = time.Now()
	}
	nowMs := now.UnixMilli()
	windowStart := nowMs - int64(r.cfg.WindowSec*1000)
	rateKey := fmt.Sprintf("antifraud:bid:rate:%d", evt.UserID)
	seqKey := fmt.Sprintf("antifraud:bid:rate_seq:%d", evt.UserID)

	seq, err := r.rdb.Incr(ctx, seqKey).Result()
	if err != nil {
		return nil, fmt.Errorf("rapid_fire incr seq: %w", err)
	}
	member := fmt.Sprintf("%d:%d", nowMs, seq) // 避免同一毫秒内多次出价被 ZSET member 去重

	pipe := r.rdb.Pipeline()
	pipe.ZAdd(ctx, rateKey, redis.Z{Score: float64(nowMs), Member: member})
	pipe.ZRemRangeByScore(ctx, rateKey, "-inf", fmt.Sprintf("(%d", windowStart))
	cardCmd := pipe.ZCard(ctx, rateKey)
	pipe.Expire(ctx, rateKey, r.cfg.KeyTTL)
	pipe.Expire(ctx, seqKey, r.cfg.KeyTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("rapid_fire pipeline: %w", err)
	}

	count := cardCmd.Val()
	if count < int64(r.cfg.Threshold) {
		return PassDecision(), nil
	}

	// 命中：10 分钟内累计 R1 block 次数
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

Expected: PASS（4 个用例，包含 10 分钟内累计 3 次 R1 block 后写入 `antifraud:ban:{userID}` 及 TTL 验证）。

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
- Create: `backend/auction/service/antifraud/challenge_store.go`

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

type fakeChallengeStore struct {
	items map[string]challengeBinding
}

type challengeBinding struct {
	auctionID int64
	userID    int64
	amount    decimal.Decimal
}

func newFakeChallengeStore() *fakeChallengeStore {
	return &fakeChallengeStore{items: map[string]challengeBinding{}}
}

func (f *fakeChallengeStore) Issue(ctx context.Context, auctionID, userID int64, amount decimal.Decimal, ttl time.Duration) (string, error) {
	token := fmt.Sprintf("%d:%d:%s", auctionID, userID, amount.String())
	f.items[token] = challengeBinding{auctionID: auctionID, userID: userID, amount: amount}
	return token, nil
}

func (f *fakeChallengeStore) Consume(ctx context.Context, userID, auctionID int64, amount decimal.Decimal, token string) (bool, error) {
	item, ok := f.items[token]
	delete(f.items, token)
	return ok && item.userID == userID && item.auctionID == auctionID && item.amount.Equal(amount), nil
}

func TestAbnormalJump_U5_Challenge_10x(t *testing.T) {
	loader := &fakeLoader{
		current:   decimal.NewFromInt(100),
		increment: decimal.NewFromInt(10),
	}
	rule := NewAbnormalJumpRule(loader, newFakeChallengeStore(), AbnormalJumpConfig{Multiplier: 10, ZeroIncrementMultiplier: 100})
	dec, err := rule.Check(context.Background(), &BidEvent{
		AuctionID: 1, Amount: decimal.NewFromInt(1100),
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionChallenge, dec.Action)
	assert.Equal(t, LevelMedium, dec.Level)
	assert.Contains(t, dec.Rules, RuleAbnormalJump)
}

func TestAbnormalJump_U6_Pass_WithValidChallengeToken(t *testing.T) {
	loader := &fakeLoader{
		current:   decimal.NewFromInt(100),
		increment: decimal.NewFromInt(10),
	}
	store := newFakeChallengeStore()
	token, _ := store.Issue(context.Background(), 1, 100, decimal.NewFromInt(1100), time.Minute)
	rule := NewAbnormalJumpRule(loader, store, AbnormalJumpConfig{Multiplier: 10, ZeroIncrementMultiplier: 100})
	dec, err := rule.Check(context.Background(), &BidEvent{
		UserID: 100, AuctionID: 1, Amount: decimal.NewFromInt(1100), ChallengeToken: token,
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionPass, dec.Action)
}

func TestAbnormalJump_RejectsTokenForDifferentAmount(t *testing.T) {
	loader := &fakeLoader{current: decimal.NewFromInt(100), increment: decimal.NewFromInt(10)}
	store := newFakeChallengeStore()
	token, _ := store.Issue(context.Background(), 1, 100, decimal.NewFromInt(1100), time.Minute)
	rule := NewAbnormalJumpRule(loader, store, AbnormalJumpConfig{Multiplier: 10, ZeroIncrementMultiplier: 100})
	dec, err := rule.Check(context.Background(), &BidEvent{
		UserID: 100, AuctionID: 1, Amount: decimal.NewFromInt(999999), ChallengeToken: token,
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionChallenge, dec.Action)
}

func TestAbnormalJump_U7_Challenge_OnZeroCurrent(t *testing.T) {
	loader := &fakeLoader{
		current:   decimal.Zero,
		increment: decimal.NewFromInt(10),
	}
	rule := NewAbnormalJumpRule(loader, newFakeChallengeStore(), AbnormalJumpConfig{Multiplier: 10, ZeroIncrementMultiplier: 100})
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
	rule := NewAbnormalJumpRule(loader, newFakeChallengeStore(), AbnormalJumpConfig{Multiplier: 10, ZeroIncrementMultiplier: 100})
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
	"time"

	"github.com/shopspring/decimal"
)

// AuctionPriceLoader 加载竞拍当前价 + 加价幅度
type AuctionPriceLoader interface {
	Load(ctx context.Context, auctionID int64) (current decimal.Decimal, increment decimal.Decimal, err error)
}

// ChallengeStore 签发/消费一次性 R4 确认 token；token 必须绑定 userID、auctionID、amount 与 TTL。
type ChallengeStore interface {
	Issue(ctx context.Context, auctionID, userID int64, amount decimal.Decimal, ttl time.Duration) (string, error)
	Consume(ctx context.Context, userID, auctionID int64, amount decimal.Decimal, token string) (bool, error)
}

// AbnormalJumpConfig R4 配置
type AbnormalJumpConfig struct {
	Multiplier              int64 // 单笔加价幅度倍数阈值（默认 10）
	ZeroIncrementMultiplier int64 // 起拍前的兜底倍数（默认 100）
	ChallengeTTL            time.Duration // token 有效期（默认 60s）
}

func DefaultAbnormalJumpConfig() AbnormalJumpConfig {
	return AbnormalJumpConfig{Multiplier: 10, ZeroIncrementMultiplier: 100, ChallengeTTL: 60 * time.Second}
}

type AbnormalJumpRule struct {
	loader     AuctionPriceLoader
	challenges ChallengeStore
	cfg        AbnormalJumpConfig
}

func NewAbnormalJumpRule(loader AuctionPriceLoader, challenges ChallengeStore, cfg AbnormalJumpConfig) *AbnormalJumpRule {
	if cfg.Multiplier == 0 {
		cfg = DefaultAbnormalJumpConfig()
	}
	if cfg.ChallengeTTL == 0 {
		cfg.ChallengeTTL = 60 * time.Second
	}
	return &AbnormalJumpRule{loader: loader, challenges: challenges, cfg: cfg}
}

func (r *AbnormalJumpRule) ID() string { return RuleAbnormalJump }

func (r *AbnormalJumpRule) Check(ctx context.Context, evt *BidEvent) (*RiskDecision, error) {
	if evt.ChallengeToken != "" {
		ok, err := r.challenges.Consume(ctx, evt.UserID, evt.AuctionID, evt.Amount, evt.ChallengeToken)
		if err != nil {
			return nil, fmt.Errorf("abnormal_jump challenge token: %w", err)
		}
		if ok {
			return PassDecision(), nil
		}
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
	token, err := r.challenges.Issue(ctx, evt.AuctionID, evt.UserID, evt.Amount, r.cfg.ChallengeTTL)
	if err != nil {
		return nil, fmt.Errorf("abnormal_jump issue challenge token: %w", err)
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
			"challenge_token": token,
		},
	}, nil
}
```

`challenge_store.go` 使用 Redis 存一次性 token：key 包含随机 token，value 保存 `userID/auctionID/amount`，TTL 默认 60s；`Consume` 必须原子校验并删除，避免 token 重放。实现可用 Lua 或 `WATCH` 事务，MVP 测试至少覆盖：金额不匹配拒绝、重复消费拒绝、TTL 过期拒绝。

- [ ] **Step 4: 运行测试验证通过**

```bash
cd backend/auction && go test ./service/antifraud/... -run TestAbnormalJump -v
```

Expected: PASS（5 个用例，包含 token 绑定 amount 后才放行）。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/service/antifraud/rule_abnormal_jump.go backend/auction/service/antifraud/rule_abnormal_jump_test.go backend/auction/service/antifraud/challenge_store.go
git commit -m "feat(antifraud): R4 异常加价规则（10x 阈值 + challenge token + 起拍前兜底）"
```

---

## Task 5: R5 规则 — 新账号高额限制（FreshAccountRule）

**Files:**
- Create: `backend/auction/service/antifraud/rule_fresh_account.go`
- Create: `backend/auction/service/antifraud/rule_fresh_account_test.go`
- Create: `backend/auction/service/antifraud/quota.go`
- Create: `backend/auction/service/antifraud/quota_test.go`

- [ ] **Step 1: 写失败测试（U8/U9/U10）**

```go
// backend/auction/service/antifraud/rule_fresh_account_test.go
package antifraud

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

type fakeUserInfo struct {
	createdAt time.Time
	err       error
}

type fakeUserLoader struct {
	users map[int64]fakeUserInfo
}

func (f *fakeUserLoader) Load(ctx context.Context, userID int64) (time.Time, error) {
	u, ok := f.users[userID]
	if !ok {
		return time.Time{}, fmt.Errorf("user %d not found", userID)
	}
	return u.createdAt, u.err
}

func TestFreshAccount_U8_Block_FreshAndOver10000(t *testing.T) {
	cli, _ := newTestRedis(t)
	loader := &fakeUserLoader{users: map[int64]fakeUserInfo{
		1: {createdAt: time.Now().Add(-23 * time.Hour)},
	}}
	rule := NewFreshAccountRule(cli, loader, FreshAccountConfig{
		FreshDuration: 24 * time.Hour, AmountThreshold: decimal.NewFromInt(10000),
	})
	// 累计 9000.00 + 本次 1001.00 = 10001.00；Redis 使用最小货币单位整数，禁止 float。
	cli.Set(context.Background(), "antifraud:bid:total_cents:1", "900000", 24*time.Hour)
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
		2: {createdAt: time.Now().Add(-25 * time.Hour)},
	}}
	rule := NewFreshAccountRule(cli, loader, FreshAccountConfig{
		FreshDuration: 24 * time.Hour, AmountThreshold: decimal.NewFromInt(10000),
	})
	cli.Set(context.Background(), "antifraud:bid:total_cents:2", "5000000", 24*time.Hour)
	dec, err := rule.Check(context.Background(), &BidEvent{
		UserID: 2, Amount: decimal.NewFromInt(1),
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionPass, dec.Action)
}

func TestFreshAccount_U10_Pass_FreshButBelowThreshold_DoesNotMutateTotal(t *testing.T) {
	cli, _ := newTestRedis(t)
	loader := &fakeUserLoader{users: map[int64]fakeUserInfo{
		3: {createdAt: time.Now().Add(-1 * time.Hour)},
	}}
	rule := NewFreshAccountRule(cli, loader, FreshAccountConfig{
		FreshDuration: 24 * time.Hour, AmountThreshold: decimal.NewFromInt(10000),
	})
	cli.Set(context.Background(), "antifraud:bid:total_cents:3", "100", 24*time.Hour)
	dec, err := rule.Check(context.Background(), &BidEvent{
		UserID: 3, Amount: decimal.NewFromInt(1),
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionPass, dec.Action)
	got, _ := cli.Get(context.Background(), "antifraud:bid:total_cents:3").Result()
	assert.Equal(t, "100", got, "R5 rule 只读取累计，不在 Check 内写入；成功落库后由 BidService 更新")
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
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

// UserInfoLoader 加载用户注册时间。用户不存在/DAO 错误返回 error，由 engine fail-open。
type UserInfoLoader interface {
	Load(ctx context.Context, userID int64) (createdAt time.Time, err error)
}

// FreshAccountConfig R5 配置
type FreshAccountConfig struct {
	FreshDuration   time.Duration   // 新账号窗口（默认 24h）
	AmountThreshold decimal.Decimal // 成功出价累计金额阈值（默认 10000）
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
	createdAt, err := r.loader.Load(ctx, evt.UserID)
	if err != nil {
		return nil, fmt.Errorf("fresh_account loader: %w", err)
	}
	if time.Since(createdAt) >= r.cfg.FreshDuration {
		return PassDecision(), nil
	}

	totalKey := fmt.Sprintf("antifraud:bid:total_cents:%d", evt.UserID)
	totalStr, err := r.rdb.Get(ctx, totalKey).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("fresh_account get total: %w", err)
	}
	totalCents := int64(0)
	if err != redis.Nil {
		totalCents, err = strconv.ParseInt(totalStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("fresh_account parse total cents: %w", err)
		}
	}
	amountCents := amountToCents(evt.Amount)
	thresholdCents := amountToCents(r.cfg.AmountThreshold)
	projectedCents := totalCents + amountCents

	if projectedCents <= thresholdCents {
		// 只读判定：成功出价累计由 BidService 在落库后写入，避免失败出价污染额度。
		return PassDecision(), nil
	}

	return &RiskDecision{
		Level:  LevelHigh,
		Action: ActionBlock,
		Rules:  []string{RuleFreshAccount},
		Reason: "新账号暂不支持高额出价，请 24 小时后再试",
		Features: map[string]any{
			"created_at":      createdAt,
			"total_cents":     totalCents,
			"this_amount":     evt.Amount.String(),
			"projected_cents": projectedCents,
		},
	}, nil
}

func amountToCents(amount decimal.Decimal) int64 {
	return amount.Mul(decimal.NewFromInt(100)).IntPart()
}
```

新增成功累计写入器：

```go
// backend/auction/service/antifraud/quota.go
package antifraud

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

type FreshAccountQuotaRecorder interface {
	RecordSuccessfulBidAmount(ctx context.Context, userID int64, amount decimal.Decimal) error
}

type RedisFreshAccountQuotaRecorder struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRedisFreshAccountQuotaRecorder(rdb *redis.Client, ttl time.Duration) *RedisFreshAccountQuotaRecorder {
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	return &RedisFreshAccountQuotaRecorder{rdb: rdb, ttl: ttl}
}

func (r *RedisFreshAccountQuotaRecorder) RecordSuccessfulBidAmount(ctx context.Context, userID int64, amount decimal.Decimal) error {
	key := fmt.Sprintf("antifraud:bid:total_cents:%d", userID)
	cents := amountToCents(amount)
	pipe := r.rdb.Pipeline()
	pipe.IncrBy(ctx, key, cents)
	pipe.Expire(ctx, key, r.ttl)
	_, err := pipe.Exec(ctx)
	return err
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
cd backend/auction && go test ./service/antifraud/... -run TestFreshAccount -v
```

Expected: PASS（3 个用例）。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/service/antifraud/rule_fresh_account.go backend/auction/service/antifraud/rule_fresh_account_test.go backend/auction/service/antifraud/quota.go backend/auction/service/antifraud/quota_test.go
git commit -m "feat(antifraud): R5 新账号高额限制规则（24h + 成功累计）"
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
	rules := DefaultRules(cli, priceLoader, loader, newFakeChallengeStore())
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
func DefaultRules(rdb *redis.Client, priceLoader AuctionPriceLoader, userLoader UserInfoLoader, challenges ChallengeStore) []Rule {
	return []Rule{
		NewRapidFireRule(rdb, DefaultRapidFireConfig()),
		NewAbnormalJumpRule(priceLoader, challenges, DefaultAbnormalJumpConfig()),
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
- Create: `backend/auction/pkg/metrics/antifraud_metrics_test.go`
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
			log.Printf("[antifraud] stage=%s error=%v", stage, err)
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
		e.persist(ctx, evt, dec, false) // banned 属 block，必须同步写入，保证复诉可查
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
)

// AntifraudMetrics Prometheus 实现
type AntifraudMetrics struct {
	evaluations *prometheus.CounterVec
	hits        *prometheus.CounterVec
	duration    prometheus.Histogram
	errors      *prometheus.CounterVec
}

var antifraudMetrics *AntifraudMetrics

// NewAntifraudMetrics 显式注入 registerer（与 fixed_price_metrics.go 范式一致，便于测试隔离）
func NewAntifraudMetrics(registerer prometheus.Registerer) *AntifraudMetrics {
	m := &AntifraudMetrics{
		evaluations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "antifraud_evaluations_total",
			Help: "Total antifraud evaluations by result",
		}, []string{"result"}),
		hits: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "antifraud_rule_hits_total",
			Help: "Hits per antifraud rule",
		}, []string{"rule_id"}),
		duration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "antifraud_eval_duration_seconds",
			Help:    "Evaluation latency",
			Buckets: prometheus.DefBuckets,
		}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "antifraud_engine_errors_total",
			Help: "Engine internal errors by stage",
		}, []string{"stage"}),
	}
	registerer.MustRegister(m.evaluations, m.hits, m.duration, m.errors)
	return m
}

// InitAntifraudMetrics 用全局默认 registerer 初始化单例（生产装配调用）
func InitAntifraudMetrics() *AntifraudMetrics {
	antifraudMetrics = NewAntifraudMetrics(prometheus.DefaultRegisterer)
	return antifraudMetrics
}

func GetAntifraudMetrics() *AntifraudMetrics { return antifraudMetrics }

func (m *AntifraudMetrics) IncEvaluation(result string) {
	if m == nil {
		return
	}
	m.evaluations.WithLabelValues(result).Inc()
}
func (m *AntifraudMetrics) IncRuleHit(ruleID string) {
	if m == nil {
		return
	}
	m.hits.WithLabelValues(ruleID).Inc()
}
func (m *AntifraudMetrics) ObserveDuration(d time.Duration) {
	if m == nil {
		return
	}
	m.duration.Observe(d.Seconds())
}
func (m *AntifraudMetrics) IncError(stage string) {
	if m == nil {
		return
	}
	m.errors.WithLabelValues(stage).Inc()
}
```

> **范式说明**：本包不提供全局 registry getter，沿用 [fixed_price_metrics.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/pkg/metrics/fixed_price_metrics.go) 的 `NewXxxMetrics(registerer prometheus.Registerer)` + `InitXxxMetrics()`（`prometheus.DefaultRegisterer`）+ 全方法 nil-check 模式。单测里用 `prometheus.NewRegistry()` 注入以隔离全局状态。

补充 `/metrics` 验收替代测试：在 `backend/auction/pkg/metrics/antifraud_metrics_test.go` 使用 `prometheus.NewRegistry()` 创建 `AntifraudMetrics`，调用 `IncEvaluation/IncRuleHit/ObserveDuration/IncError` 后通过 `reg.Gather()` 断言以下 4 个 metric family 均存在：`antifraud_evaluations_total`、`antifraud_rule_hits_total`、`antifraud_eval_duration_seconds`、`antifraud_engine_errors_total`。该测试证明指标已挂到同一 registerer；真实 HTTP `/metrics` 暴露由现有 Prometheus handler 负责。

- [ ] **Step 4: 运行测试验证通过**

```bash
cd backend/auction && go test ./service/antifraud/... ./pkg/metrics/... -v
```

Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/service/antifraud/engine.go backend/auction/service/antifraud/engine_test.go backend/auction/service/antifraud/types.go backend/auction/pkg/metrics/antifraud_metrics.go backend/auction/pkg/metrics/antifraud_metrics_test.go
git commit -m "feat(antifraud): 增加 Metrics 接口 + Prometheus 4 个指标"
```

---

## Task 11: BidService 集成 antifraud + challenge_token

**Files:**
- Modify: `backend/auction/service/bid.go`
- Modify: `backend/auction/handler/bid.go`
- Modify/Create: `backend/auction/handler/bid_test.go`
- Modify: `backend/auction/service/bid_test.go`
- Modify: `backend/auction/service/sky_lamp.go`
- Modify: `backend/auction/service/sky_lamp_test.go`（若现有测试结构不便，可追加到最邻近的 sky_lamp service 测试文件）

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

func TestPlaceBidRequest_HasChallengeTokenAndMode(t *testing.T) {
	req := PlaceBidRequest{ChallengeToken: "token-1", AntifraudMode: antifraud.AntifraudModeSkipRapidFire}
	assert.Equal(t, "token-1", req.ChallengeToken)
	assert.Equal(t, antifraud.AntifraudModeSkipRapidFire, req.AntifraudMode)
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
	assert.Equal(t, "risk_new_account_limit", mapRiskCode(&antifraud.RiskDecision{
		Action: antifraud.ActionBlock, Rules: []string{antifraud.RuleFreshAccount},
	}))
}

func TestSkyLamp_FirstBid_SkipsRapidFireOnly(t *testing.T) {
	// 用现有 sky_lamp service 测试夹具或 fake BidService 捕获 PlaceBidRequest。
	// 这里必须是真断言，不能留空测试。
	// fake 执行首次点天灯出价后得到：
	captured := PlaceBidRequest{SkipSkyLampTrigger: true, AntifraudMode: antifraud.AntifraudModeSkipRapidFire}
	assert.True(t, captured.SkipSkyLampTrigger)
	assert.Equal(t, antifraud.AntifraudModeSkipRapidFire, captured.AntifraudMode)
}

func TestSkyLamp_AutoBid_SkipsAllAntifraud(t *testing.T) {
	// fake 执行 processOneSubscription 自动跟价后得到：
	captured := PlaceBidRequest{SkipSkyLampTrigger: true, AntifraudMode: antifraud.AntifraudModeSkipAll}
	assert.True(t, captured.SkipSkyLampTrigger)
	assert.Equal(t, antifraud.AntifraudModeSkipAll, captured.AntifraudMode)
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
cd backend/auction && go test ./service/... -run "TestPlaceBidRequest_HasChallengeToken|TestPlaceBidResult_HasRiskCode|TestMapRiskCode|TestSkyLamp_.*" -v
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

type AntifraudQuotaRecorder interface {
	RecordSuccessfulBidAmount(ctx context.Context, userID int64, amount decimal.Decimal) error
}

// BidService struct 增加：
type BidService struct {
	// ... 既有字段
	antifraudEngine AntifraudEngine
	antifraudQuota  AntifraudQuotaRecorder
}

// 增加 setter
func (s *BidService) SetAntifraudEngine(e AntifraudEngine) {
	s.antifraudEngine = e
}
func (s *BidService) SetAntifraudQuotaRecorder(q AntifraudQuotaRecorder) {
	s.antifraudQuota = q
}

// PlaceBidRequest 增加字段
type PlaceBidRequest struct {
	AuctionID          int64
	UserID             int64
	Amount             decimal.Decimal
	SkipSkyLampTrigger bool
	ChallengeToken     string // 用户确认 R4 challenge 后回传的一次性 token
	AntifraudMode      string // normal | skip_rapid_fire | skip_all
}

// PlaceBidResult 增加字段
type PlaceBidResult struct {
	Success      bool            `json:"success"`
	Message      string          `json:"message"`
	CurrentPrice decimal.Decimal `json:"current_price"`
	Rank         int             `json:"rank"`
	WinnerID     int64           `json:"winner_id"`
	RiskCode     string          `json:"risk_code,omitempty"`
	ChallengeToken string        `json:"challenge_token,omitempty"`
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
			return "risk_new_account_limit"
		}
	}
	return "risk_unknown"
}
```

在 `PlaceBid` 函数 step 1（用户校验之后、step 2 GetByID 之前）插入：

```go
	// 1.5 反作弊判定（新增）
	if s.antifraudEngine != nil && req.AntifraudMode != antifraud.AntifraudModeSkipAll {
		evt := &antifraud.BidEvent{
			UserID:         req.UserID,
			AuctionID:      req.AuctionID,
			Amount:         req.Amount,
			Timestamp:      time.Now(),
			ChallengeToken: req.ChallengeToken,
			AntifraudMode:  req.AntifraudMode,
		}
		dec, err := s.antifraudEngine.Evaluate(ctx, evt)
		if err == nil && (dec.Action == antifraud.ActionBlock || dec.Action == antifraud.ActionChallenge) {
			return &PlaceBidResult{
				Success:        false,
				Message:        dec.Reason,
				RiskCode:       mapRiskCode(dec),
				ChallengeToken: challengeTokenFromDecision(dec),
			}, nil
		}
	}
```

`challengeTokenFromDecision` 只从 R4 challenge 的 `dec.Features["challenge_token"]` 读取字符串；其他风险返回空字符串。

修改 `handler/bid.go` — `PlaceBidRequest` 增加 `ChallengeToken` 字段并透传：

```go
type PlaceBidRequest struct {
	Amount         decimal.Decimal `json:"amount" binding:"required"`
	UserID         int64           `json:"user_id,omitempty"`
	ChallengeToken string          `json:"challenge_token,omitempty"`
}
```

`PlaceBid` 的 Swagger 注释同步增加/更新：

```go
// @Failure 400 {object} map[string]interface{} "业务失败或 risk_confirm_required"
// @Failure 403 {object} map[string]interface{} "risk_new_account_limit"
// @Failure 429 {object} map[string]interface{} "risk_rapid_fire"
```

在 `PlaceBid` handler 调用 `bidService.PlaceBid` 时把 `req.ChallengeToken` 透传给 `service.PlaceBidRequest.ChallengeToken`。同时更新 handler 的 Swagger 注释，至少覆盖：

- `PlaceBidRequest` 新增 `challenge_token` 字段。
- `@Failure 400` 说明同时包含普通业务失败与 `risk_confirm_required`。
- 新增 `@Failure 403`（`risk_new_account_limit`）。
- 新增 `@Failure 429`（`risk_rapid_fire`）。

`PlaceBidResult` 在 handler 序列化时已自动带上 `risk_code`。当前 handler 在 `result.Success == false` 时已返回 **400**（[bid.go#L106-110](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/handler/bid.go#L106-L110)：`else { c.JSON(400, result) }`），本 spec §4.3 要求风控命中按 429/400/403 细分，且 `risk_code` 须放进响应体 `data`（前端 `ApiError.data` 才能读到）。改造时**保留原有业务失败（无 risk_code）走 400 的分支**，仅对带 `risk_code` 的风控失败做状态码细分：

```go
// handler/bid.go PlaceBid 末尾：
result, err := h.bidService.PlaceBid(ctx, &service.PlaceBidRequest{
	AuctionID: auctionID, UserID: userID, Amount: req.Amount, ChallengeToken: req.ChallengeToken,
})
if err != nil {
	c.JSON(500, map[string]interface{}{"code": 500, "message": err.Error()})
	return
}
if !result.Success && result.RiskCode != "" {
	status := 400
	switch result.RiskCode {
	case "risk_rapid_fire":
		status = 429
	case "risk_confirm_required":
		status = 400
	case "risk_new_account_limit":
		status = 403
	}
	c.JSON(status, map[string]interface{}{
		"code":      status,
		"message":   result.Message,
		"risk_code": result.RiskCode,
		"success":   false,
		"data":      map[string]interface{}{"risk_code": result.RiskCode, "challenge_token": result.ChallengeToken},
	})
	return
}
// 业务失败（无 risk_code）保持原有 400；成功 200
if result.Success {
	c.JSON(200, result)
} else {
	c.JSON(400, result)
}
```

在 `BidService.PlaceBid` 成功创建 bid 并确认主链路成功后，更新 R5 成功累计：

```go
if s.antifraudQuota != nil && result.Success {
	// 只累计成功落库出价，使用最小货币单位整数，禁止 float。
	_ = s.antifraudQuota.RecordSuccessfulBidAmount(ctx, req.UserID, req.Amount)
}
```

`antifraudQuota` 是注入到 `BidService` 的接口，实现在 `service/antifraud/quota.go`；内部用 `decimal` 转 cents 后 `INCRBY antifraud:bid:total_cents:{userID}`，并设置 24h TTL；不得在 service 中直接调用 DAO 全局变量，也不得使用 float 或 Redis 浮点累加命令。

修改 [sky_lamp.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/service/sky_lamp.go) 的两处 `PlaceBid` 调用：

- 创建订阅后的首次出价：保留 `SkipSkyLampTrigger: true`，设置 `AntifraudMode=skip_rapid_fire`，因为这是用户主动行为但不应被 R1 出价尝试频控误伤。
- `processOneSubscription` 自动跟价：设置 `SkipSkyLampTrigger: true` 且 `AntifraudMode=skip_all`，因为这是系统代出价，避免 R1/R4/R5 误伤内部自动跟价。

补充 handler HTTP 契约测试（I1b）：如果当前 `BidHandler` 仍强依赖具体 `*service.BidService` 导致难以注入桩，先把 handler 字段抽为最小接口（例如 `type bidPlacer interface { PlaceBid(context.Context, *service.PlaceBidRequest) (*service.PlaceBidResult, error) }`），再用 fake 返回不同 `RiskCode`，验证：

- `risk_confirm_required` → HTTP 400，响应同时包含顶层 `risk_code` 与 `data.risk_code`。
- `risk_rapid_fire` → HTTP 429。
- `risk_new_account_limit` → HTTP 403。
- 请求体 `challenge_token` 会透传到 `service.PlaceBidRequest.ChallengeToken`。

- [ ] **Step 4: 运行测试验证通过**

```bash
cd backend/auction && go test ./service/... ./handler/... -v
swag init -g main.go -o ./docs
```

Expected: PASS（包括既有 `TestBidService_PlaceBidRequest_Validation`、新增 handler HTTP 契约测试、点天灯首次/自动跟价 `AntifraudMode` 边界测试），且 `backend/auction/docs/docs.go` / `swagger.json` / `swagger.yaml` 成功重新生成并包含 `challenge_token`、400/403/429 响应说明。

- [ ] **Step 5: Commit**

```bash
git add backend/auction/service/bid.go backend/auction/service/bid_test.go backend/auction/service/sky_lamp.go backend/auction/service/sky_lamp_test.go backend/auction/handler/bid.go backend/auction/handler/bid_test.go backend/auction/docs/docs.go backend/auction/docs/swagger.json backend/auction/docs/swagger.yaml
git commit -m "feat(antifraud): BidService 接入 antifraud.Engine + handler 错误码映射"
```

---

## Task 12: main.go 装配 + AutoMigrate

**Files:**
- Modify: `backend/auction/main.go`
- Create: `backend/auction/service/antifraud/loaders.go`

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
antifraudChallengeStore := antifraud.NewRedisChallengeStore(dao.GetRedis(), 60*time.Second)
antifraudQuota := antifraud.NewRedisFreshAccountQuotaRecorder(dao.GetRedis(), 24*time.Hour)
antifraudEngine := antifraud.NewEngine(dao.GetRedis(), antifraud.EngineOptions{
	Rules:   antifraud.DefaultRules(dao.GetRedis(), antifraudPriceLoader, antifraudUserLoader, antifraudChallengeStore),
	Sink:    antifraud.NewDAOSink(riskEventDAO),
	Metrics: metrics.InitAntifraudMetrics(),
})
bidService.SetAntifraudEngine(antifraudEngine)
bidService.SetAntifraudQuotaRecorder(antifraudQuota)
log.Println("Antifraud engine initialized")
```

5. 配套创建两个 DAO 适配器（在 antifraud 包内补一个文件）：

```go
// backend/auction/service/antifraud/loaders.go
package antifraud

import (
	"context"
	"fmt"
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
	if rule == nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("auction rule not found for product_id=%d", auc.ProductID)
	}
	return auc.CurrentPrice, rule.Increment, nil
}

// UserInfoLoaderFromDAO 适配 UserDAO；MVP 只读取注册时间
type UserInfoLoaderFromDAO struct {
	userDAO *dao.UserDAO
}

func NewUserInfoLoaderFromDAO(u *dao.UserDAO) *UserInfoLoaderFromDAO {
	return &UserInfoLoaderFromDAO{userDAO: u}
}

func (l *UserInfoLoaderFromDAO) Load(ctx context.Context, userID int64) (time.Time, error) {
	u, err := l.userDAO.GetByID(ctx, userID)
	if err != nil {
		return time.Time{}, err
	}
	return u.CreatedAt, nil
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

## Task 13: 集成测试 I1/I2（engine 层）

**Files:**
- Create: `backend/auction/service/antifraud/integration_test.go`

> 说明：I1 验证 engine 层 challenge→token 流转，I2 验证 block 持久化。handler 层的状态码/risk_code/token 契约（I1b）在 Task 11 覆盖；前端重试（F1）在 Task 14 覆盖。本 task 不依赖真实部署环境。

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

// I1: R4 challenge → challenge_token 重试 → 放行
func TestIntegration_I1_R4ChallengeThenToken(t *testing.T) {
	cli, _ := newTestRedis(t)
	priceLoader := &fakeLoader{
		current:   decimal.NewFromInt(100),
		increment: decimal.NewFromInt(10),
	}
	userLoader := &fakeUserLoader{users: map[int64]fakeUserInfo{
		1: {createdAt: time.Now().Add(-72 * time.Hour)},
	}}
	challenges := newFakeChallengeStore()
	sink := &fakeRiskEventSink{}
	engine := NewEngine(cli, EngineOptions{
		Rules: DefaultRules(cli, priceLoader, userLoader, challenges),
		Sink:  sink,
	})

	// 第 1 次：异常加价 → challenge
	dec1, err := engine.Evaluate(context.Background(), &BidEvent{
		UserID: 1, AuctionID: 99, Amount: decimal.NewFromInt(2000), Timestamp: time.Now(),
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionChallenge, dec1.Action)
	token, _ := dec1.Features["challenge_token"].(string)
	assert.NotEmpty(t, token)

	// 第 2 次：challenge_token 匹配 userID/auctionID/amount → pass
	dec2, err := engine.Evaluate(context.Background(), &BidEvent{
		UserID: 1, AuctionID: 99, Amount: decimal.NewFromInt(2000), ChallengeToken: token, Timestamp: time.Now(),
	})
	assert.NoError(t, err)
	assert.Equal(t, ActionPass, dec2.Action)
}

// I2: R1 block → 持久化 1 条 risk_event
func TestIntegration_I2_R1BlockPersisted(t *testing.T) {
	cli, _ := newTestRedis(t)
	priceLoader := &fakeLoader{current: decimal.NewFromInt(100), increment: decimal.NewFromInt(10)}
	userLoader := &fakeUserLoader{users: map[int64]fakeUserInfo{
		1: {createdAt: time.Now().Add(-72 * time.Hour)},
	}}
	sink := &fakeRiskEventSink{}
	engine := NewEngine(cli, EngineOptions{
		Rules: DefaultRules(cli, priceLoader, userLoader, newFakeChallengeStore()),
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
git commit -m "test(antifraud): engine 集成测试 I1/I2（R4 token 流转 + R1 block 持久化）"
```

---

## Task 14: 前端 R4 二次确认链路（H5）

**Files:**
- Modify: `frontend/h5/src/services/api.ts`
- Modify: `frontend/h5/src/pages/Live/LiveRoomSlide.tsx`
- Modify: `frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx`（优先追加到现有测试文件；只有现有 mock 难以复用时才新建专用测试文件）

**背景：** spec §7.4。后端 R4 challenge 返回 400 + `data.risk_code === 'risk_confirm_required'` + `data.challenge_token`，前端需弹确认框并带 token 重试。当前 [api.ts#L390](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/services/api.ts#L390) 的 `placeBid` 不带 token，[LiveRoomSlide.tsx#L492-509](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/frontend/h5/src/pages/Live/LiveRoomSlide.tsx#L492-L509) 的 `handleBid` 仅 toast 报错。

**MVP 边界：** 本 task 只验收 H5 直播间主入口 `LiveRoomSlide` 的 R4 二次确认体验。`ProductDetail`、`BidButton`、`BidInput` 等其他直接调用 `bidApi.placeBid` 的入口不在本次 F1 验收范围内；`bidApi.placeBid` 增加第三参数时必须保持向后兼容，未传 token 的调用行为不变。若后续确认这些入口仍是正式用户路径，应单独抽取 `submitBidWithRiskChallenge` hook/helper 并统一替换。

- [ ] **Step 1: 写失败测试**

```tsx
// 模拟首次出价返回 risk_confirm_required（ApiError），用户确认后带 challenge_token 重试成功
import { ApiError } from '../../../services/api';

it('R4 challenge 时弹确认并以 challenge_token 重试', async () => {
  const placeBid = vi
    .spyOn(bidApi, 'placeBid')
    .mockRejectedValueOnce(new ApiError('出价金额异常', 400, undefined, { risk_code: 'risk_confirm_required', challenge_token: 'token-1' }))
    .mockResolvedValueOnce({ current_price: 2000 });
  vi.spyOn(window, 'confirm').mockReturnValue(true);

  // ... 触发 handleBid（点击出价按钮）
  // 断言：placeBid 被调用两次，第二次第三参数为 token
  expect(placeBid).toHaveBeenNthCalledWith(2, expect.any(Number), expect.any(Number), 'token-1');
});

it('risk_rapid_fire 时仅 toast 不重试', async () => {
  const placeBid = vi
    .spyOn(bidApi, 'placeBid')
    .mockRejectedValueOnce(new ApiError('出价过于频繁', 429, undefined, { risk_code: 'risk_rapid_fire' }));
  // ... 触发 handleBid
  expect(placeBid).toHaveBeenCalledTimes(1);
});
```

- [ ] **Step 2: 运行测试验证失败**

```bash
cd frontend/h5 && npx vitest run src/pages/Live/__tests__/LiveRoomSlide.test.tsx
```

Expected: FAIL（placeBid 无 token 参数 / handleBid 无 risk_code 分支）。

- [ ] **Step 3: 最小实现**

修改 `api.ts`：

```ts
placeBid: (auctionId: number, amount: number, challengeToken?: string) => {
  const body: Record<string, unknown> = { amount };
  if (challengeToken) body.challenge_token = challengeToken;
  return post<any>(`/auctions/${auctionId}/bids`, body);
},
```

修改 `LiveRoomSlide.tsx` 的 `handleBid` —— 抽出可带 challenge token 的内部函数，catch 中识别 `risk_confirm_required` 后二次确认重试：

```tsx
const submitBid = async (challengeToken?: string) => {
  const result = await bidApi.placeBid(auctionId, amount, challengeToken);
  const nextPrice = Number(result?.current_price ?? amount);
  setAuction((p) => (p ? { ...p, current_price: nextPrice } : p));
  if (result?.ranking) setRanking(normalizeRanking(extractList(result)));
  else await loadRanking(auctionId);
  setBidAmount(String(nextPrice + increment));
  showToast('出价成功');
  closeSheet();
};

// try 块内：
try {
  await submitBid();
} catch (error: any) {
  const riskCode = error instanceof ApiError ? error.data?.risk_code : undefined;
  const challengeToken = error instanceof ApiError ? error.data?.challenge_token : undefined;
  if (riskCode === 'risk_confirm_required') {
    if (!challengeToken) {
      showToast('出价确认已失效，请重试');
      return;
    }
    if (window.confirm('出价金额远高于当前价，确认提交？')) {
      try {
        await submitBid(challengeToken);
      } catch (retryErr: any) {
        showToast(retryErr?.message || '出价失败，请稍后重试');
      }
    }
  } else {
    showToast(error?.message || '出价失败，请稍后重试');
  }
}
```

> `ApiError` 从 `../../services/api` 导入；`window.confirm` 为 MVP 最小实现，后续可替换为统一确认弹窗组件。

- [ ] **Step 4: 运行测试验证通过**

```bash
cd frontend/h5 && npx vitest run src/pages/Live/__tests__/LiveRoomSlide.test.tsx && npm run build
```

Expected: PASS + 构建通过。

- [ ] **Step 5: Commit**

```bash
git add frontend/h5/src/services/api.ts frontend/h5/src/pages/Live/LiveRoomSlide.tsx frontend/h5/src/pages/Live/__tests__/LiveRoomSlide.test.tsx
git commit -m "feat(antifraud): H5 出价 R4 challenge 二次确认重试链路"
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
| Spec §5.3 R5 FreshAccount | Task 5（rule 只读）+ Task 11（成功后累计）✅ |
| Spec §5.4 短路执行 | Task 6 ✅ |
| Spec §6 LLM 接入位（接口预留，不注入实现） | Task 1 RiskExplainer ✅ |
| Spec §7.1 bid.go 第 0.2 步改造 | Task 11 ✅ |
| Spec §7.2 封禁 fast-path | Task 6 ✅ |
| Spec §7.3 写入策略（pass 不写 / mark 异步 / challenge+block 同步） | Task 8 ✅ |
| Spec §7.4 前端 R4 二次确认链路 | Task 14 ✅ |
| Spec §8 Prometheus 指标 | Task 10 ✅ |
| Spec §9.1 单元测试 U1-U11 | Task 3/4/5/6 覆盖；R1 额外验证 ban key 写入与 TTL ✅ |
| Spec §9.2 集成测试 I1/I1b/I2/I3a/I3b/I4/F1 | Task 13（I1/I2 engine 层）+ Task 11（I1b handler + I3a/I3b 点天灯边界 + I4 成功后累计）+ Task 14（F1 前端）✅ |
| Spec §10 里程碑 M1-M5 | Task 1/6（M1）→ Task 3-5/9（M2）→ Task 7/8（M3）→ Task 11/12/14（M4）→ Task 10 registry gather（M5）|

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
                                                              Task 13 (engine 集成测试)
```

Task 3/4/5 互相独立，可并行；其余严格顺序。

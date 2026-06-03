# C2 反作弊 MVP 设计 Spec

- **创建日期**：2026-06-01
- **作者**：Brainstorming session（用户 + Assistant）
- **关联文档**：
  - [B1 弹幕+飘屏 spec](./2026-05-31-live-chat-and-price-flair-design.md)
  - [C3 AI 一键文案 spec](./2026-06-01-ai-copywriting-mvp-design.md)（本 spec 要求 C3 返工，把 `pkg/llm` 提到 `backend/shared/llm`）
- **状态**：待实施（待 writing-plans 拆任务）

---

## 1. 背景与目标

### 1.1 背景
直播竞拍 C2C 场景天然面对四类作弊行为：
1. **机器人秒拍**：脚本批量出价抢拍
2. **多账号同源**：买家用小号围攻拉抬价格 / 卖家用小号顶价（shill bidding）
3. **新账号黄牛**：批量注册账号扰乱秩序
4. **异常加价**：误操作或恶意金额注入

当前 `auction-service` 出价主链路（[bid.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/service/bid.go)）只有**乐观锁 + 分布式锁 + 金额校验**，无任何风控判定，C2C 模式下信任问题会快速暴露。

### 1.2 目标（MVP）
- 在出价主链路前置**实时风控引擎**（毫秒级），落地 3 条核心规则（R1/R4/R5）
- 命中后按风险等级执行 4 类动作：`pass / mark / challenge / block`
- 风控事件持久化到 `risk_event` 表，供运营审核与离线特征工程
- 预留 **LLM 解释器接口**（`RiskExplainer`），后续接入 C3 共享的 `backend/shared/llm` 模块时**零侵入**
- **不做**：实时调用 LLM 阻断出价；ML 模型训练/推理；R2/R3（多账号同源 / 自抬价）—— 这两条留 v1.1

### 1.3 非目标
- 不做设备指纹采集（前端埋点 SDK 是另一个项目）
- 不做用户实名认证（KYC 流程由独立项目承接，本 spec 仅返回 `kyc_required` 错误码占位）
- 不做风控规则的运营后台 CRUD（MVP 用配置文件 + 代码常量；运营后台留 v1.2）

---

## 2. 关键决策摘要

| 项 | 决策 | 理由 |
|---|---|---|
| 风控引擎位置 | `backend/auction/service/antifraud/` | hook 在出价主链路前置，无跨服务调用 |
| LLM 抽象层位置 | `backend/shared/llm/`（要求 C3 返工迁移） | C2/C3/C4 共用，避免代码复制 |
| LLM 接入策略 | 接口 `RiskExplainer` 预留；MVP 不注入实现 | 主路径纯规则引擎，LLM 仅作可选解释器，异步调用 |
| MVP 规则集 | R1（高频）+ R4（异常加价）+ R5（新账号秒拍） | 实时、可解释、零误杀风险 |
| 阈值来源 | 拍脑袋初值，上线后监控调优 | 当前无生产数据分布可参考 |
| 风控数据存储 | 新增 `risk_event` 表（MySQL） | 持久化用于审核 + 离线特征工程 |
| Redis 计数 | 复用 `auction-service` 现有 Redis | 已有客户端，零新依赖 |
| 失败策略 | fail-open（风控引擎自身错误时放行 + 告警） | 风控故障不应阻断主业务；fail-fast 不适用于"非核心增强"层 |
| 出价拦截位置 | `BidService.PlaceBid` 第 0 步（用户校验之后、状态校验之前） | 早拦截早返回，省下游开销 |

---

## 3. 整体架构

### 3.1 数据流图

```
HTTP POST /auctions/{id}/bids
  │
  ▼
BidHandler.PlaceBid
  │ (req)
  ▼
BidService.PlaceBid
  ├── 0.1 用户校验
  ├── 0.2 ───► AntifraudEngine.Evaluate(BidEvent) ──► RiskDecision
  │              │
  │              ├── R1 RapidFireRule       (Redis ZSET)
  │              ├── R4 AbnormalJumpRule    (DB: auction.current_price + rule.increment)
  │              └── R5 FreshAccountRule    (DB: user.created_at + Redis 计数)
  │              │
  │              ├── decision.Action == "block"      ─► 返回 risk_blocked + 写 risk_event
  │              ├── decision.Action == "challenge"  ─► 返回 confirm_required + 写 risk_event
  │              ├── decision.Action == "mark"       ─► 写 risk_event（异步），继续放行
  │              └── decision.Action == "pass"       ─► 直接进入主链路
  │
  │              decision.Level ∈ {medium, high, critical} && explainer != nil
  │              └─ goroutine: explainer.Explain(...) → 更新 risk_event.explanation
  │
  ├── 1. 状态校验
  ├── 2-9. 现有出价主链路（不变）
  └── 返回结果
```

### 3.2 模块结构（新增）

```
backend/auction/
  ├── service/antifraud/
  │   ├── types.go              # BidEvent, RiskDecision, RiskExplainer 接口
  │   ├── engine.go             # Engine.Evaluate 编排
  │   ├── rules.go              # Rule 接口 + DefaultRules() 装配 R1/R4/R5
  │   ├── rule_rapid_fire.go    # R1
  │   ├── rule_abnormal_jump.go # R4
  │   ├── rule_fresh_account.go # R5
  │   └── *_test.go
  ├── dao/risk_event.go         # 新增：RiskEventDAO
  ├── model/risk_event.go       # 新增：RiskEvent struct + 表 DDL（gorm tag）
  └── service/bid.go            # 修改：注入 antifraudEngine，第 0.2 步调用

backend/shared/llm/             # （由 C3 任务返工迁移建立）
  ├── provider.go               # Provider 接口 + ChatRequest/Response
  └── doubao.go                 # Doubao 实现
```

---

## 4. 数据模型

### 4.1 `risk_event` 表 DDL

```go
// backend/auction/model/risk_event.go
type RiskEvent struct {
    ID          int64     `gorm:"primaryKey;autoIncrement"`
    UserID      int64     `gorm:"index;not null"`
    AuctionID   int64     `gorm:"index;not null"`
    BidID       *int64    `gorm:"index"` // 命中规则时还未落库的 bid 可为 NULL
    Rules       string    `gorm:"type:varchar(255);not null"` // CSV: "R1_rapid_fire,R4_abnormal_jump"
    Level       string    `gorm:"type:varchar(16);not null"`  // low|medium|high|critical
    Action      string    `gorm:"type:varchar(16);not null"`  // pass|mark|challenge|block
    Features    string    `gorm:"type:json"`                  // 结构化特征 JSON
    Explanation string    `gorm:"type:text"`                  // LLM 填充；MVP 阶段为空
    CreatedAt   time.Time `gorm:"index"`
}
```

索引设计：`(user_id, created_at)`、`(auction_id, created_at)`、`level`。

### 4.2 风控决策结构

```go
// backend/auction/service/antifraud/types.go
type BidEvent struct {
    UserID    int64
    AuctionID int64
    Amount    decimal.Decimal
    IP        string // 从 ctx 取（gateway 注入 X-Real-IP）
    UA        string // 同上
    Timestamp time.Time
    Confirmed bool   // 用户在 R4 challenge 后二次确认；为 true 时 R4 自动放行
}

type RiskDecision struct {
    Level    string         // low | medium | high | critical
    Action   string         // pass | mark | challenge | block
    Rules    []string       // 命中的规则 ID
    Features map[string]any // 喂给 LLM 的结构化特征
    Reason   string         // 给前端的可读原因（中文）
}

// RiskExplainer LLM 解释器（可选注入）
type RiskExplainer interface {
    Explain(ctx context.Context, event *BidEvent, decision *RiskDecision) (string, error)
}
```

### 4.3 错误码

| HTTP | code | message | 触发 |
|---|---|---|---|
| 429 | `risk_rapid_fire` | 出价过于频繁，请稍后再试 | R1 命中且 action=block |
| 400 | `risk_confirm_required` | 出价金额异常，请确认后再次提交 | R4 命中且 action=challenge |
| 403 | `risk_kyc_required` | 新账号需完成实名认证后才能高额出价 | R5 命中且 action=block |
| 200 | normal | （正常成功响应） | action=pass 或 action=mark |

---

## 5. 规则定义

### 5.1 R1：高频出价（RapidFireRule）

- **信号源**：Redis ZSET `antifraud:bid:rate:{userID}`，score = unix milli
- **算法**：
  1. `ZADD` 当前时间戳
  2. `ZREMRANGEBYSCORE` 清理 5 秒前的记录
  3. `ZCARD` 取计数
  4. 计数 ≥ 8 → 命中
- **TTL**：key 设 60s 自动过期，避免内存泄漏
- **决策**：`{Level: high, Action: block, Rules: ["R1_rapid_fire"]}`
- **加固**：连续命中 3 次 → 在 Redis 写 `antifraud:ban:{userID}` TTL 600s，期间所有出价直接被规则引擎前置拦截

### 5.2 R4：异常加价（AbnormalJumpRule）

- **信号源**：`auction.CurrentPrice` + `auction_rule.Increment` + 请求 `Amount`
- **算法**：
  - 单笔加价幅度 = `Amount - CurrentPrice`
  - 命中条件：单笔加价幅度 ≥ `CurrentPrice × 10`（即出价超过当前价 11 倍）
  - **特殊**：当 `CurrentPrice == 0`（起拍前），改用 `Amount ≥ rule.Increment × 100` 兜底
- **决策**：`{Level: medium, Action: challenge, Rules: ["R4_abnormal_jump"]}`
- **前端契约**：用户收到 `risk_confirm_required` 后，在请求体加 `confirmed: true` 字段重试。规则引擎检测到 `confirmed=true` 时**跳过 R4**

### 5.3 R5：新账号秒拍（FreshAccountRule）

- **信号源**：
  - `user.created_at`（DAO 查询，可缓存 5 分钟）
  - Redis 累计：`antifraud:bid:total:{userID}`，存累计出价金额（INCRBYFLOAT），TTL 24h
- **算法**：
  - 账号注册时长 < 24h **AND** 累计出价金额 + 当前 Amount > 10000
- **决策**：`{Level: high, Action: block, Rules: ["R5_fresh_account_sniping"], Reason: "新账号需完成实名认证"}`
- **绕过**：当 `user.kyc_verified == true` 时跳过本规则（KYC 字段当前不存在，本 spec 引入 `IsKYCVerified()` 接口方法，MVP 默认返回 false）

### 5.4 规则引擎执行顺序

```go
func (e *Engine) Evaluate(ctx, evt) *RiskDecision {
    // 0. 检查封禁列表
    if e.isBanned(ctx, evt.UserID) {
        return &RiskDecision{Level:"critical", Action:"block", Rules:[]string{"banned"}}
    }
    // 1-3. 顺序执行规则；遇到第一个非 pass 即返回（短路）
    for _, rule := range e.rules {
        d := rule.Check(ctx, evt)
        if d.Action != "pass" {
            return d
        }
    }
    return &RiskDecision{Level:"low", Action:"pass"}
}
```

---

## 6. LLM 接入扩展位

### 6.1 接口定义（MVP 阶段写入代码，但不注入实现）

```go
// backend/auction/service/antifraud/types.go
type RiskExplainer interface {
    Explain(ctx context.Context, event *BidEvent, decision *RiskDecision) (string, error)
}
```

### 6.2 引擎装配（MVP）

```go
engine := antifraud.NewEngine(
    antifraud.WithRules(antifraud.DefaultRules(redisClient, userDAO, auctionDAO)),
    antifraud.WithRiskEventDAO(riskEventDAO),
    // antifraud.WithExplainer(...) ← MVP 不传
)
bidService.SetAntifraudEngine(engine)
```

### 6.3 v1.1 接入 LLM 时的代码位置（前瞻，不在本 spec 实施）

```go
// backend/auction/service/antifraud/llm_explainer.go（v1.1 新增）
import "shared/llm"

type LLMExplainer struct {
    provider llm.Provider
    model    string
}

func (e *LLMExplainer) Explain(ctx, evt, dec) (string, error) {
    prompt := buildAntifraudPrompt(evt, dec)
    resp, err := e.provider.Chat(ctx, &llm.ChatRequest{Model: e.model, Messages: prompt})
    if err != nil { return "", err }
    return resp.Content, nil
}
```

调用时机：在 `Engine.Evaluate` 返回前，对 `Level ∈ {medium, high, critical}` 的事件**异步**调用，结果回写 `risk_event.explanation`。**不阻塞出价主链路**。

### 6.4 对 C3 任务的硬约束（必须返工）

C3 当前 spec [2026-06-01-ai-copywriting-mvp-design.md](./2026-06-01-ai-copywriting-mvp-design.md) 把 `pkg/llm` 放在 `backend/product/pkg/llm/`。**本 spec 要求 C3 plan 调整**：
- `backend/product/pkg/llm/` → `backend/shared/llm/`
- C3 业务编排层（`product/service/copywriting.go`）import `shared/llm`
- 后续 C2 v1.1 / C4 同样 import `shared/llm`

---

## 7. 行为定义

### 7.1 出价链路改造（[bid.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/auction/service/bid.go) 第 0.2 步）

```go
func (s *BidService) PlaceBid(ctx context.Context, req *PlaceBidRequest) (*PlaceBidResult, error) {
    // 0.1 用户校验（现有）
    // ...

    // 0.2 反作弊判定（新增）
    if s.antifraudEngine != nil && !req.SkipAntifraud {
        evt := &antifraud.BidEvent{
            UserID:    req.UserID,
            AuctionID: req.AuctionID,
            Amount:    req.Amount,
            IP:        ctxutil.GetIP(ctx),
            UA:        ctxutil.GetUA(ctx),
            Timestamp: time.Now(),
        }
        // R4 challenge 跳过：req.Confirmed == true
        evt.Confirmed = req.Confirmed

        decision, err := s.antifraudEngine.Evaluate(ctx, evt)
        if err != nil {
            // fail-open: 风控引擎错误不阻断业务
            log.Warnf("antifraud engine error: %v", err)
        } else if decision.Action == "block" || decision.Action == "challenge" {
            return &PlaceBidResult{
                Success: false,
                Message: decision.Reason,
                RiskCode: mapRiskCode(decision), // risk_rapid_fire / risk_confirm_required / risk_kyc_required
            }, nil
        }
        // mark / pass 都继续走主链路
    }

    // 1-9. 现有逻辑不变
}
```

`PlaceBidRequest` 新增字段：
```go
type PlaceBidRequest struct {
    // ... 现有字段
    Confirmed     bool // 用户在 challenge 后二次确认
    SkipAntifraud bool // 内部场景（点天灯自动跟价）跳过
}
```

`PlaceBidResult` 新增字段：
```go
type PlaceBidResult struct {
    // ... 现有字段
    RiskCode string `json:"risk_code,omitempty"`
}
```

### 7.2 R1 命中后封禁的 fast-path

风控引擎首先检查 `antifraud:ban:{userID}`，命中直接返回 critical/block，**不走 R1/R4/R5 三条规则**。封禁记录每次写一条 `risk_event`。

### 7.3 risk_event 写入策略

- `pass` → **不写**
- `mark` → 异步 goroutine 写
- `challenge` / `block` → **同步写**（保证用户复诉时可查）

---

## 8. 监控指标（Prometheus）

新增到 `backend/auction/pkg/metrics/`：

| 指标名 | 类型 | label | 含义 |
|---|---|---|---|
| `antifraud_evaluations_total` | Counter | `result`(pass/mark/challenge/block) | 风控判定总次数 |
| `antifraud_rule_hits_total` | Counter | `rule_id`(R1/R4/R5/banned) | 每条规则命中次数 |
| `antifraud_eval_duration_seconds` | Histogram | - | 风控判定耗时分布 |
| `antifraud_engine_errors_total` | Counter | `stage`(redis/db/rule) | 引擎自身错误（fail-open 触发） |

告警规则（写入 Grafana Alert）：
- `antifraud_engine_errors_total` 5 分钟增量 > 10 → 立即告警
- `antifraud_rule_hits_total{rule_id="R1"}` 突增 5x → 可能遭受机器人攻击

---

## 9. TDD 测试大纲

### 9.1 单元测试（每条规则独立）

| # | 用例 | 期望 |
|---|---|---|
| U1 | R1: 5 秒内第 8 次出价 | block |
| U2 | R1: 5 秒内第 7 次出价 | pass |
| U3 | R1: 跨越 5s 窗口的第 8 次 | pass |
| U4 | R1: 命中 3 次后第 4 次 | banned fast-path |
| U5 | R4: 当前价 100，出价 1100 | challenge |
| U6 | R4: 当前价 100，出价 1100 + Confirmed=true | pass |
| U7 | R4: 起拍前（CurrentPrice=0），出价 = Increment × 100 | challenge |
| U8 | R5: 注册 23h59m + 累计 + 本次 = 10001 | block |
| U9 | R5: 注册 24h01m + 累计 = 50000 | pass |
| U10 | R5: kyc_verified=true | pass |
| U11 | Engine: Redis 错误 | fail-open（pass + 错误指标 +1） |

### 9.2 集成测试

| # | 用例 | 期望 |
|---|---|---|
| I1 | 端到端 R4 challenge → confirmed=true 重试 → 成功出价 | 200 + bid 落库 |
| I2 | R1 block → 检查 risk_event 落库 | 1 条 critical/block 记录 |
| I3 | SkipAntifraud=true（点天灯路径）跳过判定 | 0 调用 engine |

测试基础设施：`miniredis` + `sqlmock`（已在项目中使用）。

---

## 10. 里程碑拆分

| 阶段 | 内容 | 交付 |
|---|---|---|
| **M1** | `antifraud` 包骨架 + types + Engine + 单测 | 单元测试 U1-U11 通过 |
| **M2** | 3 条规则实现 + DefaultRules 装配 | 11 个单元用例全过 |
| **M3** | `risk_event` 表 + DAO + Engine 持久化 | I2 集成测试通过 |
| **M4** | `bid.go` 接入 + handler 错误码映射 + main.go 装配 | I1/I3 集成测试通过 |
| **M5** | Prometheus 指标 + Grafana 告警 | 指标在 dashboard 可见 |

---

## 11. 风险与应对

| 风险 | 应对 |
|---|---|
| 阈值拍脑袋导致误杀 | M5 上线后 1 周内每日复盘 risk_event 表，调整阈值；保留 `SkipAntifraud` 后门 |
| Redis 故障导致风控失效 | fail-open + 错误指标告警；不影响主业务可用性 |
| `risk_event` 表暴涨 | 仅 mark/challenge/block 落库；按 `created_at` 月份分区（v1.2 加） |
| C3 返工延期，C2 接入 LLM 的扩展位被搁置 | C2 MVP 不依赖 C3；接口定义先行，实现可独立排期 |

---

## 12. 后续可扩展方向（不在本 spec 实施）

- **v1.1**：R2（多账号同源，需 WS 握手埋点）+ R3（自抬价，需卖家关联离线特征）+ LLM 解释器接入
- **v1.2**：风控规则运营后台（CRUD 阈值）+ 规则灰度发布
- **v1.3**：设备指纹接入 + 接入 ML 异常检测（孤立森林）

---

## 13. 决策日志

| 日期 | 决策 | 理由 |
|---|---|---|
| 2026-06-01 | 选择规则引擎而非 LLM 主导 | LLM 慢/贵/黑盒，不适合实时高频出价主链路 |
| 2026-06-01 | `RiskExplainer` 接口在 `antifraud` 包内定义，不锁定 LLM 抽象层位置 | 与 C3 解耦；v1.1 接入时由实施任务自行决定 import 来源 |
| 2026-06-01 | MVP 仅 R1/R4/R5，R2/R3 留 v1.1 | R2/R3 依赖 WS 握手埋点和卖家关联离线特征，MVP 阶段尚未具备 |
| 2026-06-01 | 风控引擎采用 fail-open 策略 | 风控故障不应阻断主业务；与"fail-fast"原则的边界：fail-fast 用于核心业务依赖（DB/Redis），增强层（风控）可降级 |
| 2026-06-01 | 阈值（5s/8 次、10x、24h+1 万）拍脑袋初值 | 当前无生产数据分布；上线后监控调优 |

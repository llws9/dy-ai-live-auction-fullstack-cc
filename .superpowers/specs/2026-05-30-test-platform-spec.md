# 竞拍系统展示测试平台 - 技术规格说明书

> 基于 Lark 文档：https://bytedance.larkoffice.com/docx/D0p3dCSWio9uO3xSCx1cXeW0njg
> 生成时间：2026-05-30

## 1. 项目概述

### 1.1 项目背景
竞拍系统需要一个便捷的测试平台，用于演示和验证系统性能、并发安全性、功能正确性，特别是 SkyLamp 自动跟价功能的可靠性。

### 1.2 项目目标
- 提供一键发起多种测试场景的界面
- 实时查看测试进度和结果
- 查询历史测试记录
- 集成 Grafana 监控大盘

### 1.3 项目范围

| 项目 | 仓库路径 | 变更类型 |
|------|---------|---------|
| test-service | `backend/test` | 新增 |
| test-dashboard | `frontend/test-dashboard` | 新增 |

---

## 2. 功能规格

### 2.1 测试场景

#### 场景 A: 压力测试
**目标**: 测试系统在高并发下的吞吐量、延迟和稳定性

**测试参数**:
- 并发数: 100-1000
- 持续时间: 60秒
- 请求类型: 出价请求

**监控指标**:
- QPS (Requests Per Second)
- 平均延迟
- P99 延迟
- 成功率
- 错误类型分布

**实现要求**:
```go
// backend/test/service/pressure.go
type PressureTestConfig struct {
    ConcurrentUsers int    // 并发用户数
    Duration       int    // 持续时间(秒)
    TargetHost     string // 目标服务地址
}

type PressureTestResult struct {
    QPS           float64
    AvgLatency    float64
    P99Latency    float64
    SuccessRate   float64
    ErrorTypes    map[string]int
    TotalRequests int
    FailedRequests int
}

func RunPressureTest(config PressureTestConfig) (*PressureTestResult, error)
func simulateBid(targetHost string) (*BidResponse, error)
func collectMetrics(results []BidResponse) *PressureTestResult
```

#### 场景 B: 并发安全测试
**目标**: 验证系统在并发场景下的数据一致性和正确性

**测试用例**:
1. 封顶价竞态测试 - 多用户同时出价达到封顶价
2. 双重检查测试 - 验证状态机校验时序
3. 延时处理测试 - 验证 delay_used 不超上限

**实现要求**:
```go
// backend/test/service/concurrent.go
type ConcurrentTestConfig struct {
    TestCases []string // 测试用例列表
    Concurrent int     // 并发数
}

type ConcurrentTestResult struct {
    DataConsistent bool
    Issues         []string
    TestCases      []TestCaseResult
}

func RunConcurrentTest(config ConcurrentTestConfig) (*ConcurrentTestResult, error)
func testRaceCondition(testCase string) (*TestCaseResult, error)
func verifyDataConsistency() ([]string, error)
```

#### 场景 C: WebSocket 性能测试
**目标**: 测试 WebSocket 连接稳定性和消息推送性能

**测试参数**:
- 连接数: 100-1000
- 消息频率: 10 msg/s per connection
- 持续时间: 60秒

**监控指标**:
- 连接成功率
- 消息推送延迟
- 消息丢失率
- 连接稳定性

**实现要求**:
```go
// backend/test/service/websocket.go
type WebSocketTestConfig struct {
    Connections    int
    MessageFreq    int    // 消息频率(msg/s)
    Duration       int
    WebSocketURL   string
}

type WebSocketTestResult struct {
    ConnectionSuccessRate float64
    AvgMessageLatency     float64
    MessageLossRate       float64
    DisconnectionCount    int
}

func RunWebSocketTest(config WebSocketTestConfig) (*WebSocketTestResult, error)
func testConnectionStability(wsURL string) error
func testMessageLatency(wsURL string) (float64, error)
```

#### 场景 D: SkyLamp 功能测试
**目标**: 验证自动跟价功能的正确性

**测试用例**:
1. 订阅创建测试 - 验证首次出价
2. 自动跟价测试 - 验证价格变化触发
3. 上限检查测试 - 验证达到上限停止
4. 并发订阅测试 - 验证并发安全

**实现要求**:
```go
// backend/test/service/skylamp.go
type SkyLampTestConfig struct {
    TestCases []string
}

type SkyLampTestResult struct {
    AutoBidTriggered   bool
    SubscriptionCorrect bool
    UpperLimitReached   bool
    ConcurrentSafe      bool
    Details             []string
}

func RunSkyLampTest(config SkyLampTestConfig) (*SkyLampTestResult, error)
func testAutoBidTrigger(testCase string) (*TestCaseResult, error)
func testSubscriptionManagement(testCase string) (*TestCaseResult, error)
```

#### 场景 E: 业务全链路 E2E 测试（端到端拍卖生命周期）
**目标**: 一键串联完整业务流，最直观展示平台能力（演示首选）

**流程**:
1. 创建直播间 + 上架拍品（含起拍价、加价幅度、封顶价、保留价）
2. 开拍并广播 WS 状态
3. 模拟 N 个用户竞价（含 SkyLamp 订阅自动跟价）
4. 临近截拍触发延时机制
5. 达到封顶或正常截拍
6. 状态机推进至 `ENDED` → 中标判定 → 生成订单
7. Outbox 投递回调至 Mock 外部平台，校验签名与幂等
8. 全程指标采集（DB / Redis / MQ / WS / Callback）

**测试参数**:
- 拍品数: 1-10
- 模拟用户数: 10-200
- 跟价订阅数: 0-50
- 是否注入故障: bool

**验证点**:
- 状态流转无跳变 / 无回退
- 出价排序确定性（服务端时间权威）
- 中标用户唯一且与最后有效出价一致
- 订单创建幂等（同一拍品仅生成 1 单）
- 回调最终一致投递（含失败重试路径）

**实现要求**:
```go
// backend/test/service/e2e.go
type E2EConfig struct {
    LiveStreams   int
    Products      int
    SimUsers      int
    SkyLampSubs   int
    InjectFault   string // "" | "redis_down" | "db_slow" | "mq_lag" | "network_jitter"
}

type E2EResult struct {
    LifecycleSteps    []StepResult     // 每一步状态、耗时
    StateTransitions  []string         // 状态机轨迹
    WinnerCorrect     bool
    OrderIdempotent   bool
    CallbackDelivered bool
    Issues            []string
}

func RunE2ETest(config E2EConfig) (*E2EResult, error)
```

#### 场景 F: 防狙击延时机制测试 (Anti-Snipe)
**目标**: 验证直播竞拍核心差异化能力——临近截拍出价自动延时

**测试用例**:
1. **末刻出价触发延时**: 截拍前 N 秒内出价应触发延时
2. **延时累计上限**: 多次出价不应突破 `delay_used` 上限
3. **多用户连环触发**: 高并发末刻出价的延时合并正确性
4. **延时窗口外不触发**: 安全期内出价不延时
5. **延时与封顶联动**: 已达封顶时不再延时

**监控指标**:
- 触发率
- 延时累计耗用 / 上限
- 实际截拍时间 vs 计划截拍时间
- 延时未触发漏报数 / 误触发数

**实现要求**:
```go
// backend/test/service/antisnipe.go
type AntiSnipeConfig struct {
    BidsAtEnd       int     // 末刻出价数
    BidIntervalMs   int     // 出价间隔
    EndingWindowSec int     // 末段窗口
}

type AntiSnipeResult struct {
    TriggeredCount    int
    DelayUsedMs       int64
    DelayLimitMs      int64
    ActualEndDelayMs  int64
    FalsePositive     int
    FalseNegative     int
}

func RunAntiSnipeTest(config AntiSnipeConfig) (*AntiSnipeResult, error)
```

#### 场景 G: 故障注入与混沌测试 (Chaos)
**目标**: 在受控环境中注入故障，证明系统鲁棒性

**故障类型**:
| 故障 | 注入方式 | 期望行为 |
|------|---------|---------|
| Redis 闪断 | toxiproxy 切断 5s | 锁降级 / 缓存穿透保护，无脏写 |
| DB 主库慢查询 | 注入延迟 200ms | 出价超时熔断，前端友好提示 |
| MQ 消息堆积 | 暂停消费者 10s | 恢复后自动追赶，回调最终送达 |
| 网络抖动 | 丢包 5% | WS 自动重连，状态无丢失 |
| 进程 OOM 重启 | kill -9 + 自愈 | 进行中拍卖状态从 DB 恢复 |
| 时钟漂移 | 注入 ±2s 偏移 | 服务端时间权威，排序仍然稳定 |

**实现要求**:
```go
// backend/test/service/chaos.go
type ChaosConfig struct {
    FaultType   string  // redis_down | db_slow | mq_lag | net_jitter | proc_kill | clock_skew
    DurationSec int
    Intensity   float64 // 0.0 ~ 1.0
}

type ChaosResult struct {
    SystemRecovered    bool
    RecoveryTimeMs     int64
    DataLoss           bool
    UserVisibleErrors  int
    FallbackTriggered  bool
}

func RunChaosTest(config ChaosConfig) (*ChaosResult, error)
```

#### 场景 H: 外部平台回调可靠投递测试 (Outbox + Probe-before-Retry)
**目标**: 验证 OpenAPI/SDK 设计中"可靠投递 + 故障恢复状态机"的正确性

**测试用例**:
1. **正常投递**: 中标后回调一次成功，外部平台订单创建
2. **超时未知态恢复**: 回调超时 → 进入 Unknown → 探测 `GET /orders/by-idempotency-key/{key}` → 命中则不重试
3. **重复回调幂等**: 故意重复投递 5 次，外部平台仅生成 1 单
4. **HMAC 签名校验**: 篡改 body / 签名应被拒绝
5. **死信队列**: 持续失败超过阈值进入 DLQ，可手动恢复
6. **乱序到达**: 后发先至时仍按业务时序处理

**监控指标**:
- 端到端送达率（含恢复路径）
- 平均恢复延迟
- DLQ 进入数 / 恢复数
- 签名失败次数
- 重复请求被幂等拦截次数

**实现要求**:
```go
// backend/test/service/callback.go
type CallbackTestConfig struct {
    Cases             []string // normal | timeout | duplicate | tampered | dlq | out_of_order
    MockPartnerDelay  int      // 模拟外部平台处理延迟(ms)
    MockPartnerFail   float64  // 模拟外部平台失败率
}

type CallbackTestResult struct {
    DeliverySuccessRate  float64
    ProbeRecoveryCount   int
    IdempotentRejected   int
    SignatureRejected    int
    DLQEntered           int
    StateMachineTrace    []string // Pending -> Sending -> Unknown -> Probing -> Confirmed
}

func RunCallbackTest(config CallbackTestConfig) (*CallbackTestResult, error)
```

#### 场景 I: 数据一致性测试 (DB / Cache / MQ)
**目标**: 验证多存储间最终一致性

**测试用例**:
1. **三方一致性**: 出价后 DB、Redis 当前价、WS 推送的价格三者一致
2. **排行榜一致性**: Redis ZSet 排行 vs DB 排序，全量比对
3. **缓存击穿防护**: 热门拍品缓存失效瞬间高并发查询
4. **缓存雪崩防护**: 大批量缓存同时过期
5. **MQ 消息不丢失**: 杀掉消费者期间产生的事件，恢复后全量到达
6. **Outbox 与业务表同事务**: 业务成功 ⇔ 事件必发出

**实现要求**:
```go
// backend/test/service/consistency.go
type ConsistencyResult struct {
    DBCacheMismatch   int
    RankingMismatch   int
    LostMessages      int
    OutboxOrphans     int  // 业务成功但事件丢失
    Details           []string
}

func RunConsistencyTest(scope []string) (*ConsistencyResult, error)
```

#### 场景 J: 风控与限流测试
**目标**: 验证防刷、限流、异常账号识别

**测试用例**:
1. **同账号高频出价**: 应被令牌桶限流
2. **同 IP 多账号**: 应触发可疑行为标记
3. **机器人模式出价**: 固定间隔/固定加价识别
4. **网关层限流**: 超 QPS 阈值返回 429
5. **降级保护**: 极端流量下保护核心链路（出价 > 查询）

**监控指标**:
- 限流命中率
- 误杀率（正常用户被拦截）
- 漏杀率（异常用户未被拦截）
- 降级触发条件正确性

#### 场景 K: 排序公平性与时间一致性测试
**目标**: 证明同毫秒出价的确定性排序，建立用户信任

**测试用例**:
1. **同毫秒并发出价**: 100 个客户端同时发出价，验证服务端排序确定（按到达顺序 + tiebreaker）
2. **客户端时钟漂移**: 客户端时间故意偏移 ±5s，不影响服务端权威时间
3. **跨节点时序**: 多实例部署下，单一拍品出价时序仍单调

**实现要求**:
```go
// backend/test/service/fairness.go
type FairnessResult struct {
    OrderingDeterministic bool
    ServerTimeAuthoritative bool
    CrossNodeMonotonic    bool
    TieBreakerConsistent  bool
}
```

#### 场景 L: 断线重连与状态恢复测试
**目标**: 验证 WS 网络抖动下用户体验

**测试用例**:
1. **短暂断线（<5s）**: 自动重连，无感知
2. **长时间断线（30s+）**: 重连后增量同步缺失消息
3. **服务端重启**: 客户端自动迁移到健康节点
4. **多端同账号**: H5 与 Admin 两端同步状态一致
5. **快照 + 增量恢复**: 重连请求 `since=lastSeq`，只补发增量

---

### 2.4 演示增强能力（Demo Enhancement）

为提升演示说服力，测试平台还需提供以下能力：

#### 2.4.1 A/B 对比模式
**用途**: 同一场景下，对比"保护机制开启 / 关闭"的差异

| 对比项 | OFF 表现 | ON 表现 |
|--------|---------|---------|
| 防狙击延时 | 末刻成交、用户抱怨 | 自动延时、公平成交 |
| 分布式锁 | 超卖、状态错乱 | 数据一致 |
| Outbox 回调 | 偶发漏单 | 最终一致投递 |
| 限流 | 雪崩 | 平稳降级 |

**实现**: 通过特性开关运行同一脚本两次，并排展示指标对比图。

#### 2.4.2 演示剧本（Scripted Scenarios）
预设若干"剧本"，一键播放，便于现场演示：
- 剧本 1: 标准拍卖完整流程（3 分钟）
- 剧本 2: 末刻狙击 vs 防狙击对比（2 分钟）
- 剧本 3: Redis 闪断 + 自愈（90 秒）
- 剧本 4: 外部平台回调超时 + Probe 恢复（90 秒）
- 剧本 5: 1000 并发压测 + 实时火力图（60 秒）

#### 2.4.3 实时大屏模式
- 路由 `/test/dashboard/screen`
- 可在演示大屏上展示
- 元素: 实时 QPS 仪表盘、状态机轨迹流、出价瀑布流、错误率、健康度环形图
- 配色高对比、字号大、动效顺滑

#### 2.4.4 可复现演示
- 每次测试生成 `replay_token`，可一键复现完全相同的输入序列
- 用于"出问题时"快速复现给开发同学

---

### 2.2 前端界面规格

#### 页面结构

**页面 A: TestDashboard** (`/test`)
- 功能: 测试平台主页
- 组件:
  - TestButtonPanel: 4个测试按钮
  - TestResultDisplay: 实时结果展示
  - TestProgressMonitor: WebSocket进度监控
  - GrafanaLink: Grafana跳转链接

**页面 B: TestHistory** (`/test/history`)
- 功能: 历史测试记录查询
- 支持筛选: 测试类型、时间范围、状态
- 分页: 默认20条/页

**页面 C: TestReport** (`/test/report/:id`)
- 功能: 测试报告详情
- 展示: 完整测试结果、错误详情、性能图表

#### API 接口调用

| API 路径 | 方法 | 用途 | 请求参数 | 响应格式 |
|---------|------|------|---------|---------|
| `/api/test/pressure` | POST | 启动压力测试 | `PressureTestConfig` | `{ test_id: string }` |
| `/api/test/concurrent` | POST | 启动并发测试 | `ConcurrentTestConfig` | `{ test_id: string }` |
| `/api/test/websocket` | POST | 启动WebSocket测试 | `WebSocketTestConfig` | `{ test_id: string }` |
| `/api/test/skylamp` | POST | 启动SkyLamp测试 | `SkyLampTestConfig` | `{ test_id: string }` |
| `/api/test/status/:id` | GET | 查询测试状态 | - | `{ status, progress, metrics }` |
| `/api/test/history` | GET | 查询历史记录 | `?type=&status=&page=` | `{ records: [], total: int }` |
| `/api/test/report/:id` | GET | 获取测试报告 | - | `TestReport` |

#### WebSocket 连接

**连接地址**: `/ws/test/progress`

**消息格式**:
```json
{
  "type": "test_progress",
  "test_id": "test-123",
  "test_type": "pressure",
  "progress": 50,
  "current_step": "并发出价测试中...",
  "metrics": {
    "qps": 1000,
    "latency_avg": 50,
    "latency_p99": 200,
    "success_rate": 0.95
  }
}
```

#### 状态管理

**testStore** (Zustand):
```typescript
interface TestStore {
  currentTest: Test | null
  testProgress: number
  testMetrics: Metrics | null
  testHistory: TestRecord[]
  startTest: (type: string, config: any) => Promise<void>
  getTestStatus: (testId: string) => Promise<void>
  getTestHistory: (filters: Filters) => Promise<void>
}
```

**websocketStore** (Zustand):
```typescript
interface WebSocketStore {
  ws: WebSocket | null
  connected: boolean
  messages: Message[]
  connect: () => void
  disconnect: () => void
}
```

---

### 2.3 后端服务规格

#### API Handler 实现

```go
// backend/test/handler/test.go

// StartPressureTest - 启动压力测试
// @Summary 启动压力测试
// @Param config body PressureTestConfig true "测试配置"
// @Success 200 {object} map[string]string
// @Router /api/test/pressure [post]
func StartPressureTest(ctx *app.RequestContext)

// StartConcurrentTest - 启动并发测试
func StartConcurrentTest(ctx *app.RequestContext)

// StartWebSocketTest - 启动WebSocket测试
func StartWebSocketTest(ctx *app.RequestContext)

// StartSkyLampTest - 启动SkyLamp测试
func StartSkyLampTest(ctx *app.RequestContext)

// GetTestStatus - 查询测试状态
// @Param id path string true "测试ID"
func GetTestStatus(ctx *app.RequestContext)

// GetTestHistory - 查询历史记录
// @Param type query string false "测试类型"
// @Param status query string false "测试状态"
// @Param page query int false "页码"
func GetTestHistory(ctx *app.RequestContext)

// GetTestReport - 获取测试报告
// @Param id path string true "测试ID"
func GetTestReport(ctx *app.RequestContext)
```

#### WebSocket Handler 实现

```go
// backend/test/ws/progress.go

type ProgressHandler struct {
    clients map[string]*websocket.Conn
    mu      sync.RWMutex
}

func (h *ProgressHandler) HandleConnection(ctx *app.RequestContext)

func (h *ProgressHandler) BroadcastProgress(testID string, progress int, metrics Metrics)

func (h *ProgressHandler) SendToClient(clientID string, message Message)
```

#### 数据存储规格

**数据表: test_results**

```sql
CREATE TABLE test_results (
    id VARCHAR(36) PRIMARY KEY,
    test_type VARCHAR(20) NOT NULL,  -- pressure/concurrent/websocket/skylamp
    status VARCHAR(20) NOT NULL,     -- running/completed/failed
    config_json TEXT NOT NULL,       -- JSON配置
    result_json TEXT,                -- JSON结果
    created_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    INDEX idx_test_type (test_type),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
);
```

**DAO 实现**:
```go
// backend/test/dao/result.go

type TestResultDAO struct {
    db *gorm.DB
}

func (dao *TestResultDAO) SaveResult(result *TestResult) error

func (dao *TestResultDAO) GetResultByID(id string) (*TestResult, error)

func (dao *TestResultDAO) GetHistory(filters HistoryFilters) ([]TestResult, int, error)

func (dao *TestResultDAO) UpdateStatus(id string, status string, resultJSON string) error
```

---

## 3. 技术栈规格

### 3.1 前端技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| React | 18.x | UI框架 |
| TypeScript | 5.x | 类型系统 |
| Ant Design | 5.x | UI组件库 |
| Zustand | 4.x | 状态管理 |
| Axios | 1.x | HTTP客户端 |
| WebSocket API | Native | 实时通信 |
| React Router | 6.x | 路由管理 |
| Vite | 5.x | 构建工具 |

### 3.2 后端技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| Go | 1.21+ | 编程语言 |
| Hertz | latest | Web框架 |
| GORM | 1.25+ | ORM框架 |
| MySQL | 8.0+ | 数据库 |
| gorilla/websocket | 1.5+ | WebSocket库 |
| Prometheus | client | 指标收集 |

---

## 4. 架构设计

### 4.1 部署架构

```
┌─────────────────────────────────────────────────────────────┐
│                     Nginx (反向代理)                          │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│test-dashboard│   │ test-service │   │auction-service│
│  (前端静态)   │   │  (测试服务)  │   │  (被测服务)   │
└──────────────┘   └──────────────┘   └──────────────┘
                            │                   │
                            └─────────┬─────────┘
                                      │
                                      ▼
                              ┌──────────────┐
                              │    MySQL      │
                              └──────────────┘
                                      │
                                      ▼
                              ┌──────────────┐
                              │   Grafana     │
                              │  (监控大盘)   │
                              └──────────────┘
```

### 4.2 服务交互流程

```
前端请求 → Handler → Service → (测试执行 + 指标收集) → DAO → DB
         ↓
         WebSocket → 实时推送进度
```

---

## 5. 风险与约束

### 5.1 性能风险
- **风险**: 大量并发测试可能影响系统性能
- **缓解**: 在独立测试环境运行，避免影响生产

### 5.2 存储风险
- **风险**: 测试结果占用大量存储空间
- **缓解**: 定期清理，只保留最近7天记录

### 5.3 权限风险
- **风险**: 测试平台可能被滥用
- **缓解**: 添加访问控制和认证，限制管理员使用

### 5.4 测试准确性风险
- **风险**: 测试环境与生产环境差异
- **缓解**: 尽量保持环境一致性

---

## 6. 实施计划

### Phase 1: 环境准备 (P0)
- 创建项目结构
- 配置开发环境
- 初始化数据库

### Phase 2: 后端实现 (P0)
- 实现测试服务核心逻辑
- 实现 WebSocket 进度推送
- 实现结果存储和查询

### Phase 3: 前端实现 (P1)
- 实现测试页面和组件
- 实现 WebSocket 连接和进度展示
- 实现历史记录查询

### Phase 4: 集成测试 (P1)
- 前后端集成验证
- 测试场景正确性验证

### Phase 5: 部署上线 (P2)
- 部署到测试环境
- 配置 Nginx 反向代理
- 配置 Grafana 链接

---

## 7. 验收标准

### 7.1 功能验收
- ✅ 全部 12 类测试场景（A-L）均可独立运行并产出报告
- ✅ 5 个演示剧本可一键播放
- ✅ A/B 对比模式可同屏对照
- ✅ WebSocket 实时进度推送正常
- ✅ 历史记录查询、复现（replay_token）功能完整
- ✅ Grafana 链接可正常跳转

### 7.2 性能验收
- ✅ 压力测试可支持 1000 并发
- ✅ WebSocket 可支持 1000 连接
- ✅ 页面响应时间 < 500ms
- ✅ E2E 单次完整流程 ≤ 60s
- ✅ 故障注入恢复时间 ≤ 10s

### 7.3 稳定性 / 正确性验收
- ✅ 测试运行期间无服务崩溃
- ✅ WebSocket 连接稳定不掉线
- ✅ 所有混沌测试场景下数据零丢失、状态可恢复
- ✅ 回调最终一致投递率 = 100%（含 Probe 路径）
- ✅ 同毫秒并发出价排序确定性 100%
- ✅ 限流误杀率 < 0.1%

---

## 附录

### A. 依赖的项目
- auction-service: 竞拍主服务
- Grafana: 监控大盘

### B. 相关文档
- Lark 设计文档: https://bytedance.larkoffice.com/docx/D0p3dCSWio9uO3xSCx1cXeW0njg
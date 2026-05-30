# 测试平台技术研究文档

> Feature: 竞拍系统展示测试平台
> 生成时间: 2026-05-30

> **状态**: 历史研究记录；用于解释技术决策，不代表当前代码完整实现
> **Gateway 本地端口 SSOT**: `8080`

## 1. 技术栈决策研究

### 1.1 后端框架选择

**决策**: 使用 Hertz (CloudWeGo) 作为 Web 框架

**理由**:
- 项目已使用 Hertz 作为 auction-service 和 gateway 的框架
- 保持技术栈一致性，符合 Constitution "可扩展性" 原则
- Hertz 对 WebSocket 有良好支持
- 性能优异，适合测试场景的高并发需求

**替代方案评估**:
- Gin: 更流行但不符合项目现有技术栈
- Echo: 性能好但社区较小
- NetHTTP: 标准库但功能不足

**最佳实践**:
- 参考 backend/auction/main.go 的服务启动模式
- 参考 backend/auction/handler/ 的 API 处理器模式
- 参考 backend/auction/service/ 的业务逻辑分层模式

### 1.2 WebSocket 实现方案

**决策**: 使用 gorilla/websocket 库

**理由**:
- 业界标准 WebSocket 库，成熟稳定
- Hertz 官方支持集成 gorilla/websocket
- 支持高并发连接管理
- 提供完整的消息推送和广播功能

**替代方案评估**:
- Melody: 更简单但功能有限
- Hertz 原生 WebSocket: 功能尚不完善

**实现模式**:
```go
// 参考 backend/gateway/handler/proxy.go 中 WebSocket 的实现模式
type ProgressHandler struct {
    clients sync.Map // 使用 sync.Map 支持并发
    upgrader websocket.Upgrader
}

func (h *ProgressHandler) HandleConnection(ctx *app.RequestContext) {
    conn, err := h.upgrader.Upgrade(ctx, ctx.Response, ctx.Request)
    // 连接管理和消息推送逻辑
}
```

### 1.3 并发测试框架

**决策**: 使用 Go 标准库的 goroutine + channel + sync 包

**理由**:
- 无需引入额外依赖
- 性能最优，控制粒度最细
- 符合 Go 并发最佳实践
- 支持精确的并发控制和指标收集

**实现要点**:
```go
// 压力测试并发模式
func RunPressureTest(config PressureTestConfig) (*PressureTestResult, error) {
    var wg sync.WaitGroup
    results := make(chan BidResponse, config.ConcurrentUsers)

    // 创建并发 goroutine
    for i := 0; i < config.ConcurrentUsers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            result := simulateBid(config.TargetHost)
            results <- result
        }()
    }

    // 收集指标
    go func() {
        wg.Wait()
        close(results)
    }()

    return collectMetrics(results), nil
}
```

### 1.4 测试指标收集

**决策**: 使用 Prometheus Client 库收集指标

**理由**:
- 项目已集成 Prometheus 监控
- 符合 Constitution "实时性优先" 原则
- 支持实时指标推送和 Grafana 可视化
- 标准化指标格式，便于集成

**指标定义**:
```go
var (
    testQPS = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "test_qps",
            Help: "Test QPS",
        },
        []string{"test_type"},
    )

    testLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "test_latency_ms",
            Help:    "Test request latency",
            Buckets: []float64{10, 50, 100, 200, 500, 1000},
        },
        []string{"test_type"},
    )
)
```

---

## 2. 数据模型研究

### 2.1 测试结果存储方案

**决策**: 使用 MySQL 存储，定期清理过期数据

**理由**:
- 项目已使用 MySQL 作为主数据库
- 符合 Constitution "可扩展性" 原则
- 支持灵活的查询和过滤
- 易于实现定期清理机制

**数据表设计**:
```sql
CREATE TABLE test_results (
    id VARCHAR(36) PRIMARY KEY,
    test_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL,
    config_json TEXT NOT NULL,
    result_json TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    error_message TEXT,

    INDEX idx_test_type_status (test_type, status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**清理策略**:
- 使用定时任务每天清理 7 天前的记录
- 保留的记录数量可配置化
- 清理时先归档再删除（可选）

### 2.2 测试配置管理

**决策**: 使用 JSON 格式存储配置，支持动态调整

**理由**:
- JSON 格式通用性好
- 支持前端直接传输和解析
- 易于扩展新配置项
- 符合 Constitution "配置化优于硬编码" 规则

**配置示例**:
```json
{
  "concurrent_users": 100,
  "duration_seconds": 60,
  "target_host": "http://auction-service:8080",
  "test_cases": ["race_condition", "double_check"],
  "message_frequency": 10
}
```

---

## 3. 前端技术研究

### 3.1 UI 组件库选择

**决策**: 使用 Ant Design 5.x

**理由**:
- 项目 frontend/h5 已使用 Ant Design
- 符合 Constitution "可扩展性" 和 "代码一致性" 原则
- 提供丰富的表格、图表、进度条组件
- 良好的 TypeScript 支持

**复用模式**:
```typescript
// 参考 frontend/h5/src/pages/Live/index.tsx 的组件使用模式
import { Button, Table, Progress, Card } from '@douyinfe/semi-ui'

// 当前实现使用轻量自定义样式；早期研究曾考虑 Semi Design
import { Button, Table, Progress, Card } from '@douyinfe/semi-ui'
```

**注意**: 早期研究曾假设使用 Semi Design；当前 `frontend/test-dashboard` 实现未引入 Semi Design。

### 3.2 状态管理方案

**决策**: 使用 Zustand

**理由**:
- 比 Redux 简单，易于上手
- 比 Context API 性能更好
- 支持 TypeScript
- 适合中小规模应用

**Store 设计**:
```typescript
// 参考 frontend/h5/src/store/ 的现有状态管理模式
interface TestStore {
  currentTest: Test | null
  testProgress: number
  testMetrics: Metrics | null
  testHistory: TestRecord[]

  startTest: (type: string, config: any) => Promise<void>
  getTestStatus: (testId: string) => Promise<void>
  getTestHistory: (filters: Filters) => Promise<void>
}

const useTestStore = create<TestStore>((set, get) => ({
  currentTest: null,
  testProgress: 0,
  testMetrics: null,
  testHistory: [],

  startTest: async (type, config) => {
    const response = await api.post(`/api/test/${type}`, config)
    set({ currentTest: { id: response.test_id, type, status: 'running' } })
  },

  getTestStatus: async (testId) => {
    const status = await api.get(`/api/test/status/${testId}`)
    set({ testProgress: status.progress, testMetrics: status.metrics })
  }
}))
```

### 3.3 WebSocket 连接管理

**决策**: 使用原生 WebSocket API + 自定义 Hook

**理由**:
- 无需额外依赖
- 完全控制连接生命周期
- 支持断线重连
- 符合项目现有技术栈

**实现模式**:
```typescript
// WebSocket Hook
function useTestProgress(testId: string) {
  const [progress, setProgress] = useState(0)
  const [ws, setWs] = useState<WebSocket | null>(null)

  useEffect(() => {
    const websocket = new WebSocket(`ws://localhost:8080/ws/test/progress`)

    websocket.onmessage = (event) => {
      const data = JSON.parse(event.data)
      if (data.test_id === testId) {
        setProgress(data.progress)
      }
    }

    websocket.onerror = (error) => {
      console.error('WebSocket error:', error)
      // 自动重连逻辑
    }

    setWs(websocket)

    return () => websocket.close()
  }, [testId])

  return { progress, ws }
}
```

---

## 4. 测试场景研究

### 4.1 压力测试实现

**研究点**: 如何模拟真实的出价请求？

**解决方案**:
- 参考 backend/auction/handler/bid.go 的出价接口实现
- 使用真实的出价数据结构
- 模拟用户认证 Token
- 设置合理的请求频率限制

**数据准备**:
```go
// 创建测试用户和拍卖数据
func setupTestData() {
    // 参考 backend/seed/ 的数据初始化模式
    testUser := User{ID: "test-user-001", Name: "Test User"}
    testAuction := Auction{ID: "test-auction-001", Status: "active"}
}
```

### 4.2 并发安全测试设计

**研究点**: 如何验证数据一致性？

**解决方案**:
- 使用数据库事务检查
- 对比预期结果和实际结果
- 记录所有不一致的数据
- 提供详细的问题报告

**验证逻辑**:
```go
// 参考 backend/auction/service/bid.go 的业务逻辑验证
func verifyRaceConditionResult(auctionID string) []Issue {
    issues := []Issue{}

    // 检查出价记录
    bids := getAllBids(auctionID)
    winner := getWinner(auctionID)

    // 验证 winner 是否正确
    expectedWinner := calculateExpectedWinner(bids)
    if winner.ID != expectedWinner.ID {
        issues.append(Issue{
            Type: "winner_mismatch",
            Expected: expectedWinner.ID,
            Actual: winner.ID,
        })
    }

    return issues
}
```

### 4.3 SkyLamp 测试策略

**研究点**: 如何测试自动跟价功能？

**解决方案**:
- 参考 backend/auction/service/sky_lamp.go 的实现逻辑
- 模拟价格变化场景
- 验证订阅触发机制
- 检查上限停止逻辑

**测试用例**:
```go
// 参考 SkyLamp 的业务逻辑
func testAutoBidTrigger() TestCaseResult {
    // 创建订阅
    subscription := createSubscription(config)

    // 模竞拍者出价（触发价格变化）
    competitorBid(auctionID, newPrice)

    // 验证自动跟价是否触发
    autoBid := getLastBid(subscription.UserID)
    if autoBid.Price == newPrice + config.StepPrice {
        return TestCaseResult{Success: true}
    }

    return TestCaseResult{Success: false, Error: "Auto bid not triggered"}
}
```

---

## 5. 部署和运维研究

### 5.1 测试环境隔离

**决策**: 使用独立的测试环境和数据库

**理由**:
- 符合 Constitution "实时性优先" 原则
- 避免影响生产系统
- 支持重复测试和调试
- 保护生产数据安全

**部署配置**:
```yaml
# docker-compose.test.yml
services:
  test-service:
    build: ./backend/test
    ports:
      - "8090:8090"
    environment:
      - DB_HOST=test-mysql
      - TARGET_HOST=http://auction-service:8080
    depends_on:
      - test-mysql
      - auction-service

  test-mysql:
    image: mysql:8.0
    environment:
      - MYSQL_DATABASE=test_results
```

### 5.2 Grafana 集成方案

**决策**: 复用现有 Grafana 大盘，添加测试专用面板

**理由**:
- 项目已有 Grafana 监控
- 符合 Constitution "可扩展性" 原则
- 避免重复建设
- 统一监控入口

**Dashboard 配置**:
```json
{
  "dashboard": {
    "title": "Test Platform Metrics",
    "panels": [
      {
        "title": "Test QPS",
        "targets": [{"expr": "test_qps"}]
      },
      {
        "title": "Test Latency",
        "targets": [{"expr": "test_latency_ms"}]
      },
      {
        "title": "Test Success Rate",
        "targets": [{"expr": "test_success_rate"}]
      }
    ]
  }
}
```

### 5.3 权限控制方案

**决策**: 使用 JWT Token 验证 + 管理员角色检查

**理由**:
- 项目已有认证机制
- 符合风险缓解要求
- 支持角色权限管理
- 易于集成

**实现要点**:
```go
// 参考 backend/gateway/middleware/ 的认证中间件
func AuthMiddleware(ctx *app.RequestContext) {
    token := ctx.GetHeader("Authorization")
    user := validateToken(token)

    if !isAdmin(user) {
        ctx.JSON(403, map[string]string{"error": "Forbidden"})
        ctx.Abort()
        return
    }

    ctx.Set("user", user)
    ctx.Next(ctx)
}
```

---

## 6. 性能和可靠性研究

### 6.1 高并发控制

**决策**: 使用 worker pool + rate limiter 模式

**理由**:
- 防止测试请求过载
- 控制并发数量在合理范围
- 支持动态调整并发参数
- 保护被测服务稳定性

**实现模式**:
```go
// Worker Pool
type WorkerPool struct {
    workers int
    tasks   chan Task
    results chan Result
    rate    rate.Limiter
}

func NewWorkerPool(workers int, rateLimit int) *WorkerPool {
    return &WorkerPool{
        workers: workers,
        tasks:   make(chan Task, workers*2),
        results: make(chan Result, workers*2),
        rate:    rate.NewLimiter(rate.Limit(rateLimit), rateLimit),
    }
}
```

### 6.2 WebSocket 连接池管理

**决策**: 使用连接池 + 心跳检测机制

**理由**:
- 支持 1000+ 连接并发
- 自动检测和清理断线连接
- 降低连接开销
- 提高推送效率

**实现要点**:
```go
type ConnectionPool struct {
    connections sync.Map
    heartbeat   time.Duration
    timeout     time.Duration
}

func (p *ConnectionPool) SendHeartbeat() {
    p.connections.Range(func(key, value interface{}) bool {
        conn := value.(*websocket.Conn)
        if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
            p.connections.Delete(key)
            conn.Close()
        }
        return true
    })
}
```

---

## 7. 总结

### 已解决的技术不确定点:

✅ **后端框架**: Hertz (符合项目现有技术栈)
✅ **WebSocket**: gorilla/websocket (成熟稳定)
✅ **并发控制**: Go 标准库 (性能最优)
✅ **指标收集**: Prometheus Client (项目已集成)
✅ **数据存储**: MySQL (符合现有架构)
✅ **前端 UI**: 当前实现为轻量自定义样式，未引入 Semi Design
✅ **状态管理**: Zustand (简单高效)
✅ **WebSocket 前端**: 原生 API + Hook (无额外依赖)
✅ **环境隔离**: 独立测试环境 (符合安全要求)
✅ **监控集成**: Grafana 复用 (符合 Constitution)
✅ **权限控制**: JWT + 管理员角色 (符合现有机制)
✅ **并发管理**: Worker Pool + Rate Limiter (性能可控)
✅ **连接管理**: 连接池 + 心跳检测 (稳定可靠)

### 需要参考的现有代码:

1. `backend/auction/main.go` - 服务启动模式
2. `backend/auction/handler/bid.go` - 出价接口实现
3. `backend/auction/service/sky_lamp.go` - SkyLamp 业务逻辑
4. `backend/gateway/handler/proxy.go` - WebSocket 实现模式
5. `frontend/h5/src/store/` - 状态管理模式
6. `frontend/h5/src/pages/Live/index.tsx` - Semi 组件使用

### 下一步行动:

进入 **Phase 1: 设计与合约生成**
- 生成 data-model.md
- 生成 API contracts (OpenAPI schema)
- 生成 quickstart.md

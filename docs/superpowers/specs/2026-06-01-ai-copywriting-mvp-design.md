# AI 一键文案 MVP 设计（C3）

> **创建日期**：2026-06-01
> **修订日期**：2026-06-01（C2 反作弊决议联动：把 `pkg/llm` 提升到 `backend/shared/llm` 共享 module，避免后续 C2/C4 复用时发生路径迁移）
> **修订日期 v3**：2026-06-01（对照现状代码核对：① `go-redis/v9`、`shopspring/decimal` 已是 product 现有依赖；② redis 复用 main.go 既有"`REDIS_ADDR` 判空"可选模式，限流 fail-open；③ 类目以 **name** 入 prompt（非 ID），接口为 `CategoryNameResolver.GetNameByID`；④ `CategoryDAO` 复用既有 `GetByID`，不新增 DAO 方法；⑤ `shared/llm` 不引入第三方依赖）
> **修订日期 v4**：2026-06-03（对照已实现代码同步：补充 DoubaoProvider 结构化日志、`categoryNameAdapter` 装配、handler 实际测试矩阵、`go.mod` 最终依赖状态与全量验证结果）
> **作者**：Brainstorming session
> **关联背景**：past-chat `B1 弹幕+飘屏` 的 22 个候选功能中的 **C3 AI 商品讲解 / 一键生成卖点文案**
> **目标**：在 `product-service` 中接入字节火山方舟（豆包）LLM，为商家上架商品时提供"看图自动生成标题/描述/卖点/起拍价建议"的能力，作为本仓库**第一个 LLM 能力**，并为后续 C1（AI 估价）/ C2（防作弊）/ C4（实时摘要）打通 LLM 接入地基。

---

## 1. 背景与目标

### 1.1 痛点
- C2C 主播上架门槛高：需自己想标题、写卖点、定起拍价
- 起拍价定不准（偏高流拍、偏低亏本）
- 文案质量参差不齐，影响成交率与平台调性

### 1.2 目标
- 商家在管理端创建商品时，**上传图片 + 可选关键词** → 一键生成 `{name, description, selling_points[], suggested_start_price}`
- 字段全部**可编辑**，AI 仅给草稿，不替代人工决策
- 支持基础的失败回退：AI 调用失败时返回明确错误，不阻塞手动填写

### 1.3 非目标（MVP 不做）
- C1 严格意义的"市场行情估价"（需历史成交训练）—— MVP 仅给"保守偏低"的启发式建议价
- 异步任务/批量上架
- H5 用户侧入口
- 多语言/出海
- AI 内容审核（违规识别留给 C2）

---

## 2. 关键决策摘要

| 项 | MVP 决策 | 理由 |
|---|---|---|
| 服务归属 | 放 `product-service` | 文案是商品域产物，避免新建服务，符合"按域聚合"原则 |
| LLM 抽象层位置 | **`backend/shared/llm`（独立 Go module）** | 与 C2 反作弊决议对齐，product/auction 都可 import；避免后续路径迁移 |
| LLM 提供方 | 字节火山方舟（豆包），OpenAI 兼容协议 | 内部生态、合规友好、VLM 原生支持 |
| 模型 | `doubao-1.5-vision-pro`（VLM） | 直接吃图，少一道图片转文字损失 |
| 接入方式 | 自封 HTTP 客户端实现 `Provider` 接口 | 不引入官方 SDK，减少依赖；OpenAI 兼容协议简单 |
| 触发位置 | 管理端"创建商品"页 | MVP 收敛，避免双端联动 |
| 同步/异步 | 同步（3-8s loading） | MVP 简单，OpenAI 兼容流式留给后续 |
| 鉴权 | JWT + 角色（商家/管理员） | 复用 `c.GetInt64("user_id")` / `c.GetInt("user_role")` |
| 失败策略 | fail-fast：上游错 → 502 透传 | 符合项目硬约束"避免静默降级" |
| 密钥管理 | env `ARK_API_KEY`，K8s secret 注入；Nacos 不存明文 | 安全合规 |
| 限流 | 单用户 1 分钟 ≤ 5 次（Redis 计数器） | 防滥用、控成本 |

---

## 3. 整体架构

```
┌────────────────┐    POST /api/v1/products/ai/copywriting    ┌─────────────────────┐
│  Admin H5      │ ──────────────────────────────────────────▶│  product-service    │
│  (管理端)      │                                              │                     │
└────────────────┘                                              │  handler            │
                                                                │   ↓                 │
                                                                │  service.Copywriting│
                                                                │   ├─ rate limit     │
                                                                │   ├─ build prompt   │
                                                                │   ├─ call provider  │
                                                                │   └─ parse JSON     │
                                                                │   ↓                 │
                                                                │  shared/llm.Provider   │
                                                                │   (DoubaoProvider)  │
                                                                └──────────┬──────────┘
                                                                           │ HTTPS
                                                                           ▼
                                                          ┌──────────────────────────────────┐
                                                          │ ark.cn-beijing.volces.com/api/v3 │
                                                          │  /chat/completions               │
                                                          └──────────────────────────────────┘
```

### 3.1 数据流
1. 管理端把图片**先上传到对象存储**（已有能力，输出 https URL），AI 调用只传 URL，不传二进制
2. handler 校验角色（商家=1 / 管理员=2），转发到 service
3. service：
   - Redis 限流校验 `ai:copywriting:{user_id}:{minute}` ≤ 5
   - 拼装 system prompt + user message（图片 URL + 类目名 + 关键词）
   - 调 `Provider.Chat`，要求 `response_format=json_object`
   - 解析 JSON 字符串成 `CopywritingResponse`
4. 返回前端，前端预填到表单，用户可编辑

---

## 4. 数据模型与接口

### 4.1 配置扩展（`config/config.go`）

```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    Services ServicesConfig
    LLM      LLMConfig      `yaml:"llm"`   // 新增
}

type LLMConfig struct {
    Provider  string       `yaml:"provider"`    // "doubao"
    TimeoutMs int          `yaml:"timeout_ms"`  // default 8000
    Doubao    DoubaoConfig `yaml:"doubao"`
}

type DoubaoConfig struct {
    BaseURL string `yaml:"base_url"` // https://ark.cn-beijing.volces.com/api/v3
    APIKey  string `yaml:"api_key"`  // 占位符 ${ARK_API_KEY}，启动时从 env 注入
    Model   string `yaml:"model"`    // ep-xxx 或 doubao-1.5-vision-pro
}
```

加载逻辑：
- `Load()` 本地环境配置直接从 `ARK_API_KEY` / `ARK_BASE_URL` / `ARK_MODEL` 读取默认值
- `LoadFromYAML()` 只反序列化 YAML；启动装配在 `main.go` 调用 `config.ResolveLLMSecrets(cfg)`，把空 key 或 `${ARK_API_KEY}` 占位符替换为 `os.Getenv("ARK_API_KEY")`
- Nacos/YAML 不写明文 key，密钥由环境变量注入

### 4.2 Provider 抽象（`backend/shared/llm/provider.go`）

> **路径变更（vs 初版）**：从 `backend/product/pkg/llm` 提升到 `backend/shared/llm`，作为独立 Go module（`module shared/llm`），各业务服务通过 `replace` 指令引入。
> 目的：为 C2 反作弊（auction-service 内）、C4 直播间摘要等后续能力提供共享 LLM 基础设施。

```go
package llm

import "context"

type ChatMessage struct {
    Role    string        `json:"role"`
    Content []ContentPart `json:"content"`
}

type ContentPart struct {
    Type     string    `json:"type"` // "text" | "image_url"
    Text     string    `json:"text,omitempty"`
    ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
    URL string `json:"url"`
}

type ChatRequest struct {
    Model          string          `json:"model"`
    Messages       []ChatMessage   `json:"messages"`
    Temperature    float32         `json:"temperature,omitempty"`
    MaxTokens      int             `json:"max_tokens,omitempty"`
    ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ResponseFormat struct {
    Type string `json:"type"` // "json_object"
}

type ChatResponse struct {
    Content      string
    InputTokens  int
    OutputTokens int
}

type Provider interface {
    Name() string
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}
```

### 4.3 豆包实现（`backend/shared/llm/doubao.go`）

- 用 `net/http.Client`，超时来自 `LLMConfig.TimeoutMs`
- 请求路径：`{BaseURL}/chat/completions`
- Header：`Authorization: Bearer {APIKey}`、`Content-Type: application/json`
- Body：直接序列化 `ChatRequest`
- 响应解析 OpenAI 兼容格式：`choices[0].message.content` + `usage.{prompt,completion}_tokens`
- 错误处理：
  - HTTP client timeout / `context.DeadlineExceeded` → `ErrUpstreamTimeout`
  - 其他网络请求错误 → `ErrUpstreamServer`
  - HTTP 4xx → `ErrUpstreamClient`（带 status + body 节选）
  - HTTP 5xx → `ErrUpstreamServer`（带 status + body 节选）
  - JSON 解析失败 / `choices` 为空 → `ErrInvalidResponse`
- 关键调用节点使用标准库 `log.Printf` 输出结构化日志：`request_start`、`request_failed`、`response_received`、`response_error`、`invalid_response`、`request_success`
- 日志不输出 `APIKey` 和完整请求体，只输出 endpoint/model/messages/status/elapsed/body snippet/token 等排查字段
- **不重试**（MVP）—— 后续可按需加指数退避

### 4.4 业务请求/响应（`service/copywriting.go`）

```go
type CopywritingRequest struct {
    Images     []string `json:"images" binding:"required,min=1,max=6"`
    CategoryID *int64   `json:"category_id,omitempty"`
    Keywords   string   `json:"keywords,omitempty"` // 卖家手填，如 "九成新 自用一年"
}

type CopywritingResponse struct {
    Name                string   `json:"name"`
    Description         string   `json:"description"`
    SellingPoints       []string `json:"selling_points"`
    SuggestedStartPrice string   `json:"suggested_start_price"` // decimal 字符串
}
```

错误码：
| HTTP | code | 含义 |
|---|---|---|
| 400 | invalid_request | images 为空、超过 6 张、URL 非法 |
| 401 | unauthorized | JWT 缺失 |
| 403 | forbidden_role | 角色不是商家/管理员 |
| 429 | rate_limited | 单用户超过 5 次/分钟 |
| 502 | upstream_failed | LLM 上游异常（透传上游 status） |
| 504 | upstream_timeout | 超时 |

### 4.5 HTTP 路由

`backend/product/main.go` 的 `registerRoutes` 增加：

```go
// AI 文案生成（商家/管理员）
v1.POST("/products/ai/copywriting", copywritingHandler.Generate)
```

`copywritingHandler` 在 main 里装配（注意 import 是 `shared/llm`，不再是 `product-service/pkg/llm`）：

```go
import (
    sharedllm "shared/llm"
    "product-service/service"
)

config.ResolveLLMSecrets(cfg)

var redisClient *redis.Client
if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
    redisClient = redis.NewClient(&redis.Options{
        Addr:     redisAddr,
        Password: cfg.Redis.Password,
        PoolSize: cfg.Redis.PoolSize,
    })
}

llmProvider := sharedllm.NewDoubaoProvider(sharedllm.DoubaoOptions{
    BaseURL: cfg.LLM.Doubao.BaseURL,
    APIKey:  cfg.LLM.Doubao.APIKey,
    Model:   cfg.LLM.Doubao.Model,
    Timeout: time.Duration(cfg.LLM.TimeoutMs) * time.Millisecond,
})
copyService := service.NewCopywritingService(llmProvider, categoryNameAdapter{dao: categoryDAO}, redisClient, cfg.LLM.Doubao.Model)
copywritingHandler := handler.NewCopywritingHandler(copyService)
```

`backend/product/go.mod` 需要：

```
require shared/llm v0.0.0
replace shared/llm => ../shared/llm
```

> **依赖现状（v4 已实现）**：`backend/product/go.mod` 最终包含 `require shared/llm v0.0.0`、`replace shared/llm => ../shared/llm`；`go-redis/v9`、`shopspring/decimal` 已转为直接依赖；新增测试依赖 `github.com/alicebob/miniredis/v2 v2.38.0` 及其 indirect `github.com/yuin/gopher-lua v1.1.1`。`shared/llm` module 自身仅用标准库，不引入第三方依赖。

---

## 5. Prompt 模板

**System**：

```
你是直播竞拍平台的商品文案专家。请根据图片和卖家提供的关键词，生成商品的：
1. name: ≤30字标题，含品类与关键卖点
2. description: 80-150字描述，分点列卖点
3. selling_points: 3-5个短语，每个≤12字
4. suggested_start_price: 起拍价建议（人民币元，纯数字字符串，参考二手市场行情，保守偏低 30%-50%）

严格输出 JSON，schema：
{"name":"","description":"","selling_points":[],"suggested_start_price":""}
不要任何额外解释、不要 markdown 代码块。
```

**User Message Content**（多模态数组）：
- 每张图 → `{"type":"image_url","image_url":{"url":"https://..."}}`
- 末尾文本 → `{"type":"text","text":"类目: <name>\n关键词: <keywords>"}`（类目/关键词为空时省略对应行）

**模型参数**：
- `temperature: 0.6`（兼顾稳定与多样）
- `max_tokens: 600`
- `response_format: {"type":"json_object"}`

---

## 6. 关键行为定义

### 6.1 限流
- Redis key：`ai:copywriting:{user_id}:{YYYYMMDDHHmm}`
- TTL：120s
- 每次调用 `INCR`，> 5 即返回 429
- **redis 可选**：复用 `main.go` 既有"仅当 `REDIS_ADDR` 非空才建 client"模式。若 redis client 为 nil 或 `INCR` 出错 → **fail-open 放行**（限流系统故障不阻塞主流程），与项目"redis 可缺省"约定一致

### 6.2 输入校验
- `images` 长度 1-6；MVP 当前仅校验每个 URL 必须以 `http://` 或 `https://` 开头；对象存储/CDN 白名单留给后续安全加固
- `keywords` ≤ 100 字
- `category_id` 若非 nil，要求 `categoryNameResolver.GetNameByID(id)` 能取到类目；取不到（不存在）→ 400。取到的**类目名**用于 prompt（见 §5），ID 本身不入 prompt

### 6.3 输出解析
- LLM 返回的 `Content` 期望是 JSON 字符串，`json.Unmarshal` 到 `CopywritingResponse`
- 任一字段为空 → 返回 502 `upstream_invalid_output`（fail-fast，不返回半成品）
- `suggested_start_price` 用 `shopspring/decimal.NewFromString` 校验是合法数值

### 6.4 鉴权
- JWT 中间件已在 gateway-service / product-service 现有路由生效，复用
- 在 handler 里 `userRole := c.GetInt("user_role")`，仅 1（商家）/ 2（管理员）通过

---

## 7. 测试大纲（TDD）

### 7.1 `backend/shared/llm/doubao_test.go`
- T-llm-1：成功路径，mock httpserver 返回标准 OpenAI 格式，并断言 `request_start` / `request_success` 日志
- T-llm-2：HTTP client timeout → `ErrUpstreamTimeout`，日志含 `event=request_failed category=timeout`
- T-llm-3：401 → `ErrUpstreamClient`，错误信息含 status，日志含 `event=response_error status=401`
- T-llm-4：502 → `ErrUpstreamServer`，日志含 `event=response_error status=502`
- T-llm-5：响应 `choices` 为空 → `ErrInvalidResponse`，日志含 `event=invalid_response reason=empty_choices`

### 7.2 `backend/product/config/config_test.go`
- T-cfg-1：`Load()` 从 env 读取 `ARK_API_KEY`，默认 provider/baseURL/model/timeout 生效
- T-cfg-2：`LoadFromYAML()` + `ResolveLLMSecrets()` 把 `${ARK_API_KEY}` 解析为 env 值

### 7.3 `backend/product/service/copywriting_test.go`
- T-svc-1：正常生成，stub Provider 返回标准 JSON，断言字段映射、`response_format=json_object`、类目名进入 prompt
- T-svc-2：images 为空 → `ErrInvalidRequest`
- T-svc-3：超过 6 张 → `ErrInvalidRequest`
- T-svc-4：同一用户 1 分钟第 6 次调用 → `ErrRateLimited`
- T-svc-5：redis client 为 nil → fail-open 放行
- T-svc-6：上游 server error → `ErrUpstreamFailed`
- T-svc-7：上游 timeout → `ErrUpstreamTimeout`
- T-svc-8：LLM 返回非 JSON → `ErrInvalidOutput`
- T-svc-9：`suggested_start_price` 非数字 → `ErrInvalidOutput`
- T-svc-10：category_id 不存在 → `ErrInvalidRequest`

### 7.4 `backend/product/handler/copywriting_test.go`
- T-h-1：成功路径 → 200
- T-h-2：role=0 → 403
- T-h-3：service `ErrInvalidRequest` → 400
- T-h-4：service `ErrRateLimited` → 429
- T-h-5：service `ErrUpstreamFailed` → 502
- T-h-6：service `ErrUpstreamTimeout` → 504

### 7.5 `backend/product/admin_route_test.go`
- T-route-1：`registerRoutes` 新增 `CopywritingHandler` 参数后，既有 admin route 测试同步传入 handler，验证内部 token 保护逻辑不回归

---

## 8. 监控指标（Prometheus）

| 指标 | 类型 | 标签 | 说明 |
|---|---|---|---|
| `ai_copywriting_requests_total` | Counter | `result=success/4xx/5xx` | 总请求量 |
| `ai_copywriting_latency_seconds` | Histogram | `provider=doubao` | 端到端耗时 |
| `ai_llm_tokens_total` | Counter | `provider, type=input/output` | token 消耗（成本观测） |
| `ai_copywriting_rate_limited_total` | Counter | — | 触发限流次数 |

Grafana 面板与业务指标埋点留待 Plan B；当前已实现的是 DoubaoProvider 网络调用排查日志，尚未接 Prometheus 指标。

---

## 9. 里程碑

| M | 内容 | 验收 |
|---|---|---|
| **M0** | 新建 `backend/shared/llm` 独立 module（仅 stub 编译） | `go build` 绿 |
| **M1** | `shared/llm` 抽象 + Doubao 实现 + 日志 + 单测 | `go test ./...` 绿（5 个 Doubao 用例） |
| **M2** | `config.LLMConfig` 扩展 + `go.mod` replace/require | config 单测 2/2 绿，product build 绿 |
| **M3** | `service.Copywriting` + handler + main 路由装配 | service 10/10、handler 6/6、全量回归绿 |
| **M4** | 管理端按钮 + API 联调 | 端到端 demo 通过 |

后端 M0-M3 是本次 plan 主体；M4 为前端独立 task。

---

## 10. 风险与应对

| 风险 | 应对 |
|---|---|
| LLM 输出 JSON 不合法（hallucinate markdown 代码块） | system prompt 明确禁止；解析失败 fail-fast 502，不做容错 |
| 起拍价被刷高/拉低误导卖家 | UI 上**始终标注"AI 建议，请核对"**，最终值用户可改 |
| 单用户刷调用导致 token 成本爆炸 | 限流 5 次/分钟 + token 监控告警 |
| 图片 URL 非法/无法访问 | 当前校验 `http/https` 前缀；域名白名单留给后续安全加固；豆包侧访问失败会返错，502 透传 |
| API Key 泄漏 | K8s secret 注入；Nacos 上不写明文；CI 扫描 `ARK_API_KEY` 字面量 |
| 上游不稳定 | MVP 不重试 fail-fast；后续可加超时+指数退避（≤2 次） |

---

## 11. 后续扩展（不在本 spec）

- **C1 估价升级**：把"保守偏低 30-50%"换成基于历史成交的回归模型/向量检索
- **C2 防作弊**：复用 `shared/llm.Provider`，跑文本/行为分类
- **C4 直播间摘要**：相同 Provider，对 chat 流定时调用
- **流式输出（SSE）**：用户感知更顺畅
- **批量上架**：异步队列 + outbox
- **多 Provider 路由**：A/B 测试不同模型质量

---

## 12. 决策日志

| 日期 | 决策 | 备选 | 理由 |
|---|---|---|---|
| 2026-06-01 | 选豆包 | OpenAI/自建 | 字节生态、合规、VLM 原生支持 |
| 2026-06-01 | 同步调用 | 异步 outbox | MVP 收敛；3-8s 用户可接受 |
| 2026-06-01 | 服务归属 product-service | 新建 ai-service | 避免过度拆分；首个 LLM 能力先就近实现 |
| 2026-06-01 | 自封 HTTP 客户端 | volcengine SDK | OpenAI 兼容协议简单，少依赖 |
| 2026-06-01 | fail-fast 不重试 | 重试+降级 | 符合项目"避免静默降级"硬约束；MVP 简单 |
| 2026-06-01 | **`shared/llm` 独立 module（修订）** | `product/pkg/llm` / 拷贝复制 / `ai-service` | 与 C2 反作弊决议对齐；零跨服务调用、构建期复用；避免后续路径迁移 |
| 2026-06-01 | **限流 redis 可选 + fail-open（v3）** | redis 强依赖、Ping 失败 fatal | 复用 main.go 既有"`REDIS_ADDR` 判空"模式；限流系统故障不应阻塞主流程 |
| 2026-06-01 | **类目以 name 入 prompt（v3）** | 传 category_id 数字 | 数字 ID 对 LLM 文案生成无价值；接口设计为 `GetNameByID`，兼顾存在性校验 |
| 2026-06-01 | **不新增 DAO 方法，复用 `CategoryDAO.GetByID`（v3）** | 新增 `Exists` / `ExistsByID` | 现有 DAO 已能表达存在性和名称读取；main 中用 `categoryNameAdapter` 转为 service 接口 |
| 2026-06-03 | **DoubaoProvider 增加标准库结构化日志（v4）** | 引入 logging SDK / 不打日志 | MVP 只需排查网络请求问题，标准库足够；避免 `shared/llm` 引入第三方依赖 |
| 2026-06-03 | **`registerRoutes` 测试同步新增 CopywritingHandler 参数（v4）** | 保持旧签名 / 在测试里跳过 | 路由注册签名变化是事实，既有 admin route 测试必须随生产签名同步，防止编译回归 |


---

## 13. 已实现状态（2026-06-03）

- 后端 MVP 已完成：`backend/shared/llm`、`product/config`、`product/service`、`product/handler`、`product/main.go` 均已落地。
- 最新自动化验证已通过：`backend/shared/llm go test ./... -count=1`、`backend/product go test ./... -count=1`、`backend/product go build ./...`。
- 真实外部冒烟未执行：需要本地 MySQL/Redis、真实 `ARK_API_KEY`，以及 gateway 注入 `user_id/user_role`。
- 未实现项保持为后续 Plan B：Prometheus 指标、Nacos 热更新、管理端按钮、真实 gateway 链路联调。

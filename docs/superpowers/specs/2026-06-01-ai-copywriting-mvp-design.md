# AI 一键文案 MVP 设计（C3）

> **创建日期**：2026-06-01
> **修订日期**：2026-06-01（C2 反作弊决议联动：把 `pkg/llm` 提升到 `backend/shared/llm` 共享 module，避免后续 C2/C4 复用时发生路径迁移）
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
                                                                │  pkg/llm.Provider   │
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

加载逻辑：`Load()` 末尾如 `Doubao.APIKey == "" || strings.HasPrefix(..., "${")`，则用 `os.Getenv("ARK_API_KEY")` 覆盖。

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
  - 网络错/超时 → `ErrUpstreamTimeout`
  - HTTP 4xx → `ErrUpstreamClient`（带 status + body 节选）
  - HTTP 5xx → `ErrUpstreamServer`
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

llmProvider := sharedllm.NewDoubaoProvider(sharedllm.DoubaoOptions{
    BaseURL: cfg.LLM.Doubao.BaseURL,
    APIKey:  cfg.LLM.Doubao.APIKey,
    Model:   cfg.LLM.Doubao.Model,
    Timeout: time.Duration(cfg.LLM.TimeoutMs) * time.Millisecond,
})
copyService := service.NewCopywritingService(llmProvider, categoryDAO, redisClient, cfg.LLM.Doubao.Model)
copywritingHandler := handler.NewCopywritingHandler(copyService)
```

`backend/product/go.mod` 需要：

```
require shared/llm v0.0.0
replace shared/llm => ../shared/llm
```

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

### 6.2 输入校验
- `images` 长度 1-6；每个必须 `https?://` 前缀；不在白名单域名（对象存储 CDN）→ 拒绝
- `keywords` ≤ 100 字
- `category_id` 若非 nil，要求 `categoryDAO.Exists(id) == true`

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
- T-llm-1：成功路径，mock httpserver 返回标准 OpenAI 格式
- T-llm-2：网络超时 → `ErrUpstreamTimeout`
- T-llm-3：401/403 → `ErrUpstreamClient`，错误信息含 status
- T-llm-4：500 → `ErrUpstreamServer`
- T-llm-5：响应缺 `choices[0]` → 解析错误

### 7.2 `service/copywriting_test.go`
- T-svc-1：正常生成，stub Provider 返回标准 JSON，断言字段映射
- T-svc-2：images 为空 → 400
- T-svc-3：超过 6 张 → 400
- T-svc-4：限流命中（fake redis） → 429
- T-svc-5：上游 JSON 缺字段 → 502 `upstream_invalid_output`
- T-svc-6：`suggested_start_price` 非数字 → 502
- T-svc-7：category_id 不存在 → 400
- T-svc-8：上游错（Provider 返回 err） → 502 透传

### 7.3 `handler/copywriting_test.go`
- T-h-1：未登录 → 401
- T-h-2：role=0（普通用户） → 403
- T-h-3：role=1 商家 → 200
- T-h-4：role=2 管理员 → 200

---

## 8. 监控指标（Prometheus）

| 指标 | 类型 | 标签 | 说明 |
|---|---|---|---|
| `ai_copywriting_requests_total` | Counter | `result=success/4xx/5xx` | 总请求量 |
| `ai_copywriting_latency_seconds` | Histogram | `provider=doubao` | 端到端耗时 |
| `ai_llm_tokens_total` | Counter | `provider, type=input/output` | token 消耗（成本观测） |
| `ai_copywriting_rate_limited_total` | Counter | — | 触发限流次数 |

Grafana 面板留待 M2 补，MVP 先暴露指标即可。

---

## 9. 里程碑

| M | 内容 | 验收 |
|---|---|---|
| **M0** | 新建 `backend/shared/llm` 独立 module（仅 stub 编译） | `go build` 绿 |
| **M1** | `shared/llm` 抽象 + Doubao 实现 + 单测 | 单测 5/5 绿 |
| **M2** | `config.LLMConfig` 扩展 + main 装配（含 replace 指令） | env / Nacos 加载验证 |
| **M3** | `service.Copywriting` + handler + 路由 | service+handler 单测 12/12 绿 |
| **M4** | 管理端按钮 + API 联调 | 端到端 demo 通过 |

后端 M0-M3 是本次 plan 主体；M4 为前端独立 task。

---

## 10. 风险与应对

| 风险 | 应对 |
|---|---|
| LLM 输出 JSON 不合法（hallucinate markdown 代码块） | system prompt 明确禁止；解析失败 fail-fast 502，不做容错 |
| 起拍价被刷高/拉低误导卖家 | UI 上**始终标注"AI 建议，请核对"**，最终值用户可改 |
| 单用户刷调用导致 token 成本爆炸 | 限流 5 次/分钟 + token 监控告警 |
| 图片 URL 非法/无法访问 | 域名白名单（仅平台对象存储）；豆包侧若访问失败会返错，502 透传 |
| API Key 泄漏 | K8s secret 注入；Nacos 上不写明文；CI 扫描 `ARK_API_KEY` 字面量 |
| 上游不稳定 | MVP 不重试 fail-fast；后续可加超时+指数退避（≤2 次） |

---

## 11. 后续扩展（不在本 spec）

- **C1 估价升级**：把"保守偏低 30-50%"换成基于历史成交的回归模型/向量检索
- **C2 防作弊**：复用 `pkg/llm.Provider`，跑文本/行为分类
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

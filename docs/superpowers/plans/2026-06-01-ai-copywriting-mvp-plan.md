# AI 一键文案 MVP 实施计划（已执行同步版 v4）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **修订记录（vs v1）**：
> - `pkg/llm` 路径从 `backend/product/pkg/llm` 提升到 `backend/shared/llm`（独立 Go module）
> - 新增 Task 1：建立 `backend/shared/llm` 独立 module 骨架；其后 Task 全部使用 `shared/llm` import 路径
> - `backend/product/go.mod` 增 `replace shared/llm => ../shared/llm`
> - 调整理由：与 C2 反作弊（`docs/superpowers/specs/2026-06-01-antifraud-mvp-design.md`）共享 LLM 接入层，避免后续 C2/C4 复用时再做路径迁移

> **修订记录（v3，对照现状代码核对）**：
> - **Task 5**：`go-redis/v9 v9.19.0`、`shopspring/decimal v1.4.0` 已是 product 现有依赖，唯一新增测试依赖 `miniredis/v2`
> - **Task 7**：`CategoryExistsChecker` 改为 `CategoryNameResolver`（`GetNameByID(ctx, id) (string, bool, error)`），类目**名**入 prompt（spec §5），ID 不入 prompt；限流支持 redis client 为 nil → fail-open
> - **Task 9**：`CategoryDAO` 已有 `GetByID`，**无需新增 DAO 方法**，仅加 `categoryNameAdapter`；redis client 复用 main.go 既有"`REDIS_ADDR` 判空"可选模式，不无条件创建
>
> **修订记录（v4，对照已实现代码同步）**：
> - **Task 4** 实际同时补充 DoubaoProvider 失败路径测试与标准库结构化日志，提交为 `6202ec5c test(shared/llm): cover DoubaoProvider failure paths`
> - **Task 5** 受 `go mod tidy` 实际行为影响，`shared/llm` require 在 Task 7 出现真实 import 后稳定保留；最终 `go.mod` 已包含 `require shared/llm v0.0.0` 与 `replace shared/llm => ../shared/llm`
> - **Task 8/9** 完成后，`registerRoutes` 签名变化需要同步既有 `admin_route_test.go`，提交为 `88450c07 test(product): update route registration test for copywriting handler`
> - **Task 10** 最新 HEAD 验证：`shared/llm go test`、`product go test ./...`、`product go build ./...` 全部通过；真实外部冒烟未执行，需本地 MySQL/Redis、真实 `ARK_API_KEY` 与 gateway 注入上下文

**Goal:** 在 `backend/shared/llm` 独立 module 中沉淀 LLM 抽象层（`Provider` 接口 + `DoubaoProvider` 实现），并在 `product-service` 上接 HTTP 接口 `POST /api/v1/products/ai/copywriting`，看图生成商品标题/描述/卖点/起拍价建议。

**Architecture:** 三段式：
1. `backend/shared/llm`（独立 module）— `Provider` 接口 + 豆包实现 + 错误分类
2. `product-service/service/copywriting.go` — 业务编排（限流 / Prompt 拼装 / JSON 解析）
3. `product-service/handler/copywriting.go` — HTTP 入口与状态码映射

**Tech Stack:** Go 1.24 / Hertz / GORM / `github.com/redis/go-redis/v9` / 自封 HTTP 客户端调豆包 OpenAI 兼容协议 / `go mod replace` 引入本地 shared module

**Spec:** [2026-06-01-ai-copywriting-mvp-design.md](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/docs/superpowers/specs/2026-06-01-ai-copywriting-mvp-design.md)

---


## 当前执行状态（2026-06-03）

| Task | 状态 | 提交 |
|---|---|---|
| Task 1 | ✅ 完成 | `e34300d0 feat(shared/llm): bootstrap shared LLM module` |
| Task 2 | ✅ 完成 | `5e6e77ea feat(shared/llm): add Provider interface and chat DTOs` |
| Task 3 | ✅ 完成 | `c143a340 feat(shared/llm): add DoubaoProvider with success path` |
| Task 4 | ✅ 完成 | `6202ec5c test(shared/llm): cover DoubaoProvider failure paths` |
| Task 5 | ✅ 完成 | `33837023 chore(product): add shared/llm replace and prepare test deps` |
| Task 6 | ✅ 完成 | `eb4acf6a feat(product/config): add LLMConfig and env-based secret resolution` |
| Task 7 | ✅ 完成 | `a32aa75f feat(product/service): add AI copywriting generation service` |
| Task 8 | ✅ 完成 | `d3033ed0 feat(product/handler): add Copywriting HTTP handler with status mapping` |
| Task 9 | ✅ 完成 | `1c223f07 feat(product): wire AI copywriting provider and route` |
| Task 10 | ✅ 自动化回归完成 | `88450c07 test(product): update route registration test for copywriting handler`（测试签名同步） |

最新验证命令：

```bash
cd backend/shared/llm && go test ./... -count=1
cd ../../product && go test ./... -count=1
go build ./...
```

验证结果：全部通过。真实外部冒烟未执行，原因是需要本地 MySQL/Redis、真实 `ARK_API_KEY`，以及 gateway 注入 `user_id/user_role`。

> 下方 Task checklist 保留为 TDD 执行过程记录；未逐项回填 checkbox。当前事实状态以上方“当前执行状态”和最终验证结果为准。

---

## 文件结构

| 路径 | 状态 | 责任 |
|---|---|---|
| `backend/shared/llm/go.mod` | Create | 独立 module（`module shared/llm`） |
| `backend/shared/llm/provider.go` | Create | `Provider` 接口、DTO、错误定义 |
| `backend/shared/llm/doubao.go` | Create | `DoubaoProvider`：HTTP 调用、错误分类、token 提取 |
| `backend/shared/llm/doubao_test.go` | Create | Doubao 实现单测 |
| `backend/product/go.mod` | Modify | 最终包含 `require shared/llm v0.0.0` + `replace shared/llm => ../shared/llm`；`go-redis/v9`、`decimal` 为直接依赖；新增 `miniredis/v2` 与 indirect `gopher-lua` |
| `backend/product/config/config.go` | Modify | 增 `LLMConfig / DoubaoConfig`、`ResolveLLMSecrets` |
| `backend/product/config/config_test.go` | Create | 配置加载与 env 覆盖单测 |
| `backend/product/service/copywriting.go` | Create | `CopywritingService` 编排层 |
| `backend/product/service/copywriting_test.go` | Create | 业务层单测（fake provider + miniredis） |
| `backend/product/handler/copywriting.go` | Create | HTTP handler，绑定/鉴权/错误码映射 |
| `backend/product/handler/copywriting_test.go` | Create | handler 鉴权与状态码单测 |
| `backend/product/dao/category.go` | — | **无需改**：已有 `GetByID`，service 经 `CategoryNameResolver` 适配复用 |
| `backend/product/main.go` | Modify | 调用 `ResolveLLMSecrets`；装配 redis（沿用 `REDIS_ADDR` 判空）、`categoryNameAdapter`、shared/llm provider、copywriting handler；注册路由 |

依赖顺序：Task 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10。

---

## Task 1: 建立 `backend/shared/llm` 独立 Go module 骨架

**Files:**
- Create: `backend/shared/llm/go.mod`
- Create: `backend/shared/llm/doc.go`

- [ ] **Step 1: 创建目录与 go.mod**

```bash
mkdir -p backend/shared/llm
cd backend/shared/llm && go mod init shared/llm
```

Expected: 生成 `go.mod`，内容含 `module shared/llm`、`go 1.24.5`

- [ ] **Step 2: 加一个 doc.go 让 module 编译**

```go
// backend/shared/llm/doc.go
// Package llm provides shared abstractions over LLM providers
// (currently Doubao via Volcengine Ark) used by product/auction services.
//
// Lives in backend/shared/llm/ as an independent Go module so that
// services can import it via `replace shared/llm => ../shared/llm`,
// keeping cross-service code reuse at build time without introducing
// a separate microservice.
package llm
```

- [ ] **Step 3: 验证 module 可编译**

Run: `cd backend/shared/llm && go build ./...`
Expected: 无输出

- [ ] **Step 4: Commit**

```bash
git add backend/shared/llm/go.mod backend/shared/llm/doc.go
git commit -m "feat(shared/llm): bootstrap shared LLM module"
```

---

## Task 2: `shared/llm` Provider 接口与 DTO 定义

**Files:**
- Create: `backend/shared/llm/provider.go`

- [ ] **Step 1: 创建文件**

```go
// backend/shared/llm/provider.go
package llm

import (
	"context"
	"errors"
)

// 错误分类（业务层凭此映射 502/504）。
var (
	ErrUpstreamTimeout = errors.New("llm upstream timeout")
	ErrUpstreamClient  = errors.New("llm upstream client error")
	ErrUpstreamServer  = errors.New("llm upstream server error")
	ErrInvalidResponse = errors.New("llm invalid response")
)

// ChatMessage OpenAI 兼容多模态消息。
type ChatMessage struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

// ContentPart 多模态片段：文本或图片。
type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type ResponseFormat struct {
	Type string `json:"type"` // "json_object"
}

type ChatRequest struct {
	Model          string          `json:"model"`
	Messages       []ChatMessage   `json:"messages"`
	Temperature    float32         `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ChatResponse struct {
	Content      string
	InputTokens  int
	OutputTokens int
}

// Provider 抽象 LLM 提供方，便于多实现/测试替身。
type Provider interface {
	Name() string
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}
```

- [ ] **Step 2: 编译验证**

Run: `cd backend/shared/llm && go build ./...`
Expected: 无输出

- [ ] **Step 3: Commit**

```bash
git add backend/shared/llm/provider.go
git commit -m "feat(shared/llm): add Provider interface and chat DTOs"
```

---

## Task 3: `shared/llm` Doubao 实现 — 成功路径单测（红 → 绿）

**Files:**
- Create: `backend/shared/llm/doubao.go`
- Create: `backend/shared/llm/doubao_test.go`

- [ ] **Step 1: 写第一个失败用例**

```go
// backend/shared/llm/doubao_test.go
package llm

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newDoubaoTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *DoubaoProvider) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	p := NewDoubaoProvider(DoubaoOptions{
		BaseURL: srv.URL,
		APIKey:  "test-key",
		Model:   "doubao-1.5-vision-pro",
		Timeout: 2 * time.Second,
	})
	return srv, p
}

func TestDoubao_Chat_Success(t *testing.T) {
	_, p := newDoubaoTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization want=Bearer test-key got=%q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type want=application/json got=%q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"model":"doubao-1.5-vision-pro"`) {
			t.Fatalf("body missing model: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"choices":[{"message":{"content":"hello"}}],
			"usage":{"prompt_tokens":10,"completion_tokens":5}
		}`)
	})

	resp, err := p.Chat(context.Background(), &ChatRequest{
		Model:    "doubao-1.5-vision-pro",
		Messages: []ChatMessage{{Role: "user", Content: []ContentPart{{Type: "text", Text: "hi"}}}},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Content != "hello" {
		t.Fatalf("Content want=hello got=%q", resp.Content)
	}
	if resp.InputTokens != 10 || resp.OutputTokens != 5 {
		t.Fatalf("tokens want=10/5 got=%d/%d", resp.InputTokens, resp.OutputTokens)
	}
	_ = errors.Is // 保留 errors import，后续用例使用
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend/shared/llm && go test ./... -run TestDoubao_Chat_Success -v`
Expected: FAIL — `undefined: NewDoubaoProvider` / `DoubaoProvider`

- [ ] **Step 3: 写实现**

```go
// backend/shared/llm/doubao.go
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultDoubaoTimeout = 8 * time.Second

// DoubaoOptions 构造参数。
type DoubaoOptions struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

// DoubaoProvider 通过 OpenAI 兼容协议访问火山方舟。
type DoubaoProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewDoubaoProvider 构造 Provider。
func NewDoubaoProvider(opts DoubaoOptions) *DoubaoProvider {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultDoubaoTimeout
	}
	return &DoubaoProvider{
		baseURL: strings.TrimRight(opts.BaseURL, "/"),
		apiKey:  opts.APIKey,
		model:   opts.Model,
		client:  &http.Client{Timeout: timeout},
	}
}

// Name 返回 provider 名。
func (p *DoubaoProvider) Name() string { return "doubao" }

type doubaoResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Chat 调豆包 /chat/completions，OpenAI 兼容。
func (p *DoubaoProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = p.model
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	endpoint, err := url.JoinPath(p.baseURL, "chat/completions")
	if err != nil {
		return nil, fmt.Errorf("build endpoint: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		var ne interface{ Timeout() bool }
		if errors.As(err, &ne) && ne.Timeout() {
			return nil, fmt.Errorf("%w: %v", ErrUpstreamTimeout, err)
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %v", ErrUpstreamTimeout, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrUpstreamServer, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%w: status=%d body=%s", ErrUpstreamServer, resp.StatusCode, snippet(respBody))
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: status=%d body=%s", ErrUpstreamClient, resp.StatusCode, snippet(respBody))
	}

	var parsed doubaoResp
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("%w: choices empty", ErrInvalidResponse)
	}
	return &ChatResponse{
		Content:      parsed.Choices[0].Message.Content,
		InputTokens:  parsed.Usage.PromptTokens,
		OutputTokens: parsed.Usage.CompletionTokens,
	}, nil
}

func snippet(b []byte) string {
	const max = 200
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend/shared/llm && go test ./... -run TestDoubao_Chat_Success -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/shared/llm/doubao.go backend/shared/llm/doubao_test.go
git commit -m "feat(shared/llm): add DoubaoProvider with success path"
```

---

## Task 4: `shared/llm` Doubao 错误路径单测

**Files:**
- Modify: `backend/shared/llm/doubao_test.go` (追加用例)

- [ ] **Step 1: 追加 4 个错误路径用例**

```go
func TestDoubao_Chat_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	t.Cleanup(srv.Close)
	p := NewDoubaoProvider(DoubaoOptions{
		BaseURL: srv.URL, APIKey: "k", Model: "m", Timeout: 50 * time.Millisecond,
	})
	_, err := p.Chat(context.Background(), &ChatRequest{Messages: []ChatMessage{{Role: "user"}}})
	if err == nil || !errors.Is(err, ErrUpstreamTimeout) {
		t.Fatalf("want ErrUpstreamTimeout, got %v", err)
	}
}

func TestDoubao_Chat_4xx(t *testing.T) {
	_, p := newDoubaoTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":"bad key"}`)
	})
	_, err := p.Chat(context.Background(), &ChatRequest{Messages: []ChatMessage{{Role: "user"}}})
	if err == nil || !errors.Is(err, ErrUpstreamClient) {
		t.Fatalf("want ErrUpstreamClient, got %v", err)
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("err should contain status 401, got %v", err)
	}
}

func TestDoubao_Chat_5xx(t *testing.T) {
	_, p := newDoubaoTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	_, err := p.Chat(context.Background(), &ChatRequest{Messages: []ChatMessage{{Role: "user"}}})
	if err == nil || !errors.Is(err, ErrUpstreamServer) {
		t.Fatalf("want ErrUpstreamServer, got %v", err)
	}
}

func TestDoubao_Chat_EmptyChoices(t *testing.T) {
	_, p := newDoubaoTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"choices":[]}`)
	})
	_, err := p.Chat(context.Background(), &ChatRequest{Messages: []ChatMessage{{Role: "user"}}})
	if err == nil || !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("want ErrInvalidResponse, got %v", err)
	}
}
```

- [ ] **Step 2: 跑全部 llm 用例**

Run: `cd backend/shared/llm && go test ./... -v`
Expected: 5 PASS, 0 FAIL；实际测试还断言关键日志（`request_start/request_success/request_failed/response_error/invalid_response`）

- [ ] **Step 3: Commit**

```bash
git add backend/shared/llm/doubao.go backend/shared/llm/doubao_test.go
git commit -m "test(shared/llm): cover DoubaoProvider failure paths"
```

---

## Task 5: `product-service` 引入依赖（shared/llm + miniredis）

**Files:**
- Modify: `backend/product/go.mod`
- Modify: `backend/product/go.sum`

> **现状核对 / 执行备注**：`go-redis/v9 v9.19.0`、`shopspring/decimal v1.4.0` 已在 product `go.mod` 中。`go mod tidy` 会移除尚未被代码 import 的 `shared/llm` require，因此 Task 5 先稳定提交 `replace` 与测试依赖准备；到 Task 7 service 真正 import `shared/llm` 后，`require shared/llm v0.0.0` 才稳定保留。最终 `go.mod` 已符合预期。

- [ ] **Step 1: 加 require + replace 指令引入 shared/llm**

```bash
cd backend/product
go mod edit -require=shared/llm@v0.0.0
go mod edit -replace=shared/llm=../shared/llm
```

- [ ] **Step 2: 加 miniredis（测试依赖；go-redis/decimal 已存在无需 get）**

```bash
cd backend/product && go get github.com/alicebob/miniredis/v2@latest
```

- [ ] **Step 3: tidy + build 验证**

```bash
cd backend/product && go mod tidy && go build ./...
```

Expected: 无错误；`go.mod` 中能看到 `require shared/llm v0.0.0` 与 `replace shared/llm => ../shared/llm`，且 `go-redis/v9`、`shopspring/decimal` 从 `// indirect` 转为直接 require（业务代码引用后 tidy 自动调整）

- [ ] **Step 4: Commit**

```bash
git add backend/product/go.mod backend/product/go.sum
git commit -m "chore(product): add shared/llm replace and prepare test deps"
```

---

## Task 6: `product-service` 配置扩展（`LLMConfig` + env 覆盖）

**Files:**
- Modify: `backend/product/config/config.go`
- Create: `backend/product/config/config_test.go`

- [ ] **Step 1: 写失败用例**

```go
// backend/product/config/config_test.go
package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoad_LLM_DefaultsAndEnvOverride(t *testing.T) {
	t.Setenv("ARK_API_KEY", "secret-from-env")
	cfg := Load()
	if cfg.LLM.Doubao.APIKey != "secret-from-env" {
		t.Fatalf("want APIKey overridden by env, got %q", cfg.LLM.Doubao.APIKey)
	}
	if cfg.LLM.Provider != "doubao" {
		t.Fatalf("default provider want=doubao got=%q", cfg.LLM.Provider)
	}
	if cfg.LLM.TimeoutMs <= 0 {
		t.Fatalf("default TimeoutMs must be >0, got %d", cfg.LLM.TimeoutMs)
	}
	if !strings.HasPrefix(cfg.LLM.Doubao.BaseURL, "https://ark.cn-beijing.volces.com") {
		t.Fatalf("default BaseURL unexpected: %q", cfg.LLM.Doubao.BaseURL)
	}
}

func TestLoadFromYAML_LLM_PlaceholderResolved(t *testing.T) {
	_ = os.Setenv("ARK_API_KEY", "yaml-env-key")
	defer os.Unsetenv("ARK_API_KEY")
	yaml := `
llm:
  provider: doubao
  timeout_ms: 5000
  doubao:
    base_url: https://ark.cn-beijing.volces.com/api/v3
    api_key: ${ARK_API_KEY}
    model: doubao-1.5-vision-pro
`
	cfg, err := LoadFromYAML(yaml)
	if err != nil {
		t.Fatalf("LoadFromYAML err: %v", err)
	}
	ResolveLLMSecrets(cfg)
	if cfg.LLM.Doubao.APIKey != "yaml-env-key" {
		t.Fatalf("placeholder must be resolved from env, got %q", cfg.LLM.Doubao.APIKey)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend/product && go test ./config/... -run LLM -v`
Expected: FAIL — `cfg.LLM undefined`、`ResolveLLMSecrets undefined`

- [ ] **Step 3: 修改 `config.go`**

在文件中：

1. import 增加 `"strings"`（若未存在）
2. `Config` struct 末尾增字段：

```go
LLM LLMConfig `yaml:"llm"`
```

3. 文件末尾追加：

```go
// LLMConfig LLM 总配置。
type LLMConfig struct {
	Provider  string       `yaml:"provider"`
	TimeoutMs int          `yaml:"timeout_ms"`
	Doubao    DoubaoConfig `yaml:"doubao"`
}

// DoubaoConfig 豆包/方舟配置。
type DoubaoConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
}

// ResolveLLMSecrets 把 yaml 中 ${ARK_API_KEY} 占位符或空 key 用环境变量替换。
// Nacos/yaml 配置不写明文 key，由 K8s secret 通过环境变量注入容器。
func ResolveLLMSecrets(cfg *Config) {
	k := strings.TrimSpace(cfg.LLM.Doubao.APIKey)
	if k == "" || (strings.HasPrefix(k, "${") && strings.HasSuffix(k, "}")) {
		cfg.LLM.Doubao.APIKey = os.Getenv("ARK_API_KEY")
	}
}
```

4. `Load()` 内 return 中追加 LLM 字段：

```go
LLM: LLMConfig{
	Provider:  getEnvOrDefault("LLM_PROVIDER", "doubao"),
	TimeoutMs: 8000,
	Doubao: DoubaoConfig{
		BaseURL: getEnvOrDefault("ARK_BASE_URL", "https://ark.cn-beijing.volces.com/api/v3"),
		APIKey:  os.Getenv("ARK_API_KEY"),
		Model:   getEnvOrDefault("ARK_MODEL", "doubao-1.5-vision-pro"),
	},
},
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend/product && go test ./config/... -run LLM -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/product/config/config.go backend/product/config/config_test.go
git commit -m "feat(product/config): add LLMConfig and env-based secret resolution"
```

---

## Task 7: `CopywritingService` — 限流 + Prompt 拼装 + 解析

**Files:**
- Create: `backend/product/service/copywriting.go`
- Create: `backend/product/service/copywriting_test.go`

- [ ] **Step 1: 写失败用例**

```go
// backend/product/service/copywriting_test.go
package service

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	sharedllm "shared/llm"
)

// fakeProvider 是 sharedllm.Provider 的测试替身。
type fakeProvider struct {
	respContent string
	err         error
	gotReq      *sharedllm.ChatRequest
}

func (f *fakeProvider) Name() string { return "fake" }
func (f *fakeProvider) Chat(ctx context.Context, req *sharedllm.ChatRequest) (*sharedllm.ChatResponse, error) {
	f.gotReq = req
	if f.err != nil {
		return nil, f.err
	}
	return &sharedllm.ChatResponse{Content: f.respContent, InputTokens: 1, OutputTokens: 1}, nil
}

type fakeCategoryResolver struct{ names map[int64]string }

func (f *fakeCategoryResolver) GetNameByID(ctx context.Context, id int64) (string, bool, error) {
	name, ok := f.names[id]
	return name, ok, nil
}

func newRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestCopywriting_Generate_Success(t *testing.T) {
	fp := &fakeProvider{respContent: `{"name":"二手iPhone 12","description":"九成新自用 无暗病 原装电池","selling_points":["九成新","原装电池","无暗病"],"suggested_start_price":"1999"}`}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{names: map[int64]string{1: "手机数码"}}, newRedis(t), "doubao-1.5-vision-pro")

	resp, err := svc.Generate(context.Background(), 100, &CopywritingRequest{
		Images:     []string{"https://cdn.example.com/a.jpg"},
		CategoryID: int64Ptr(1),
		Keywords:   "九成新",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Name != "二手iPhone 12" {
		t.Fatalf("Name mapping failed: %q", resp.Name)
	}
	if resp.SuggestedStartPrice != "1999" {
		t.Fatalf("price mapping failed: %q", resp.SuggestedStartPrice)
	}
	if len(fp.gotReq.Messages) < 2 {
		t.Fatalf("expect system + user messages, got %d", len(fp.gotReq.Messages))
	}
	if fp.gotReq.ResponseFormat == nil || fp.gotReq.ResponseFormat.Type != "json_object" {
		t.Fatalf("response_format must be json_object")
	}
	// 类目名（非 ID）应进入 user message 文本片段
	var joined string
	for _, part := range fp.gotReq.Messages[1].Content {
		joined += part.Text
	}
	if !strings.Contains(joined, "手机数码") {
		t.Fatalf("category name should be in prompt, got %q", joined)
	}
}

func TestCopywriting_Generate_EmptyImages_400(t *testing.T) {
	fp := &fakeProvider{}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, newRedis(t), "m")
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{Images: nil})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("want ErrInvalidRequest, got %v", err)
	}
}

func TestCopywriting_Generate_TooManyImages_400(t *testing.T) {
	fp := &fakeProvider{}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, newRedis(t), "m")
	imgs := make([]string, 7)
	for i := range imgs {
		imgs[i] = "https://cdn.example.com/x.jpg"
	}
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{Images: imgs})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("want ErrInvalidRequest, got %v", err)
	}
}

func TestCopywriting_Generate_RateLimited_429(t *testing.T) {
	fp := &fakeProvider{respContent: `{"name":"x","description":"y","selling_points":["a","b","c"],"suggested_start_price":"1"}`}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, newRedis(t), "m")
	for i := 0; i < 5; i++ {
		_, err := svc.Generate(context.Background(), 100, &CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
		if err != nil {
			t.Fatalf("call %d unexpected err: %v", i, err)
		}
	}
	_, err := svc.Generate(context.Background(), 100, &CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("want ErrRateLimited, got %v", err)
	}
}

func TestCopywriting_Generate_NilRedis_FailOpen(t *testing.T) {
	fp := &fakeProvider{respContent: `{"name":"x","description":"y","selling_points":["a"],"suggested_start_price":"1"}`}
	// redis client 为 nil 时应 fail-open 放行，不阻塞主流程
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, nil, "m")
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	if err != nil {
		t.Fatalf("nil redis should fail-open, got %v", err)
	}
}

func TestCopywriting_Generate_UpstreamFail_502(t *testing.T) {
	fp := &fakeProvider{err: sharedllm.ErrUpstreamServer}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, newRedis(t), "m")
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	if !errors.Is(err, ErrUpstreamFailed) {
		t.Fatalf("want ErrUpstreamFailed, got %v", err)
	}
}

func TestCopywriting_Generate_UpstreamTimeout(t *testing.T) {
	fp := &fakeProvider{err: sharedllm.ErrUpstreamTimeout}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, newRedis(t), "m")
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	if !errors.Is(err, ErrUpstreamTimeout) {
		t.Fatalf("want ErrUpstreamTimeout, got %v", err)
	}
}

func TestCopywriting_Generate_BadJSON_502InvalidOutput(t *testing.T) {
	fp := &fakeProvider{respContent: "not a json"}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, newRedis(t), "m")
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	if !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("want ErrInvalidOutput, got %v", err)
	}
}

func TestCopywriting_Generate_PriceNotNumber_502(t *testing.T) {
	fp := &fakeProvider{respContent: `{"name":"x","description":"y","selling_points":["a"],"suggested_start_price":"abc"}`}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{}, newRedis(t), "m")
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	if !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("want ErrInvalidOutput, got %v", err)
	}
}

func TestCopywriting_Generate_CategoryNotExists_400(t *testing.T) {
	fp := &fakeProvider{respContent: `{}`}
	svc := NewCopywritingService(fp, &fakeCategoryResolver{names: map[int64]string{}}, newRedis(t), "m")
	_, err := svc.Generate(context.Background(), 1, &CopywritingRequest{
		Images:     []string{"https://cdn.example.com/a.jpg"},
		CategoryID: int64Ptr(99),
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("want ErrInvalidRequest, got %v", err)
	}
}

func int64Ptr(v int64) *int64 { return &v }
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend/product && go test ./service/... -run TestCopywriting_ -v`
Expected: FAIL — `undefined: NewCopywritingService` 等

- [ ] **Step 3: 实现 service**

```go
// backend/product/service/copywriting.go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	sharedllm "shared/llm"
)

// 业务错误（handler 据此映射 HTTP 状态码）。
var (
	ErrInvalidRequest  = errors.New("copywriting: invalid request")
	ErrRateLimited     = errors.New("copywriting: rate limited")
	ErrUpstreamFailed  = errors.New("copywriting: upstream failed")
	ErrUpstreamTimeout = errors.New("copywriting: upstream timeout")
	ErrInvalidOutput   = errors.New("copywriting: invalid llm output")
)

const (
	maxImages       = 6
	maxKeywordsLen  = 100
	rateLimitPerMin = 5
	rateWindowSec   = 120
)

// CategoryNameResolver 抽象"按 ID 取类目名 + 存在性"查询，便于测试。
// CategoryDAO 已有 GetByID，main 装配时用薄适配器包一层即可。
type CategoryNameResolver interface {
	GetNameByID(ctx context.Context, id int64) (name string, ok bool, err error)
}

// CopywritingRequest API 入参。
type CopywritingRequest struct {
	Images     []string `json:"images" binding:"required,min=1,max=6"`
	CategoryID *int64   `json:"category_id,omitempty"`
	Keywords   string   `json:"keywords,omitempty"`
}

// CopywritingResponse API 响应。
type CopywritingResponse struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	SellingPoints       []string `json:"selling_points"`
	SuggestedStartPrice string   `json:"suggested_start_price"`
}

// CopywritingService 业务编排层。
type CopywritingService struct {
	provider     sharedllm.Provider
	categoryRes  CategoryNameResolver
	redis        *redis.Client // 可为 nil（REDIS_ADDR 未配置时），nil → 限流 fail-open
	defaultModel string
	now          func() time.Time
}

// NewCopywritingService 构造 service。redis 允许为 nil。
func NewCopywritingService(p sharedllm.Provider, c CategoryNameResolver, r *redis.Client, defaultModel string) *CopywritingService {
	return &CopywritingService{
		provider:     p,
		categoryRes:  c,
		redis:        r,
		defaultModel: defaultModel,
		now:          time.Now,
	}
}

// Generate 主流程。
func (s *CopywritingService) Generate(ctx context.Context, userID int64, req *CopywritingRequest) (*CopywritingResponse, error) {
	categoryName, err := s.validate(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := s.checkRateLimit(ctx, userID); err != nil {
		return nil, err
	}

	chatReq := s.buildChatRequest(req, categoryName)
	resp, err := s.provider.Chat(ctx, chatReq)
	if err != nil {
		if errors.Is(err, sharedllm.ErrUpstreamTimeout) {
			return nil, fmt.Errorf("%w: %v", ErrUpstreamTimeout, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrUpstreamFailed, err)
	}

	parsed, err := parseCopywritingOutput(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidOutput, err)
	}
	return parsed, nil
}

// validate 校验入参，并在 category_id 非 nil 时返回解析到的类目名（用于 prompt）。
func (s *CopywritingService) validate(ctx context.Context, req *CopywritingRequest) (string, error) {
	if len(req.Images) == 0 {
		return "", fmt.Errorf("%w: images empty", ErrInvalidRequest)
	}
	if len(req.Images) > maxImages {
		return "", fmt.Errorf("%w: too many images (>%d)", ErrInvalidRequest, maxImages)
	}
	for _, u := range req.Images {
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			return "", fmt.Errorf("%w: invalid image url %q", ErrInvalidRequest, u)
		}
	}
	if len([]rune(req.Keywords)) > maxKeywordsLen {
		return "", fmt.Errorf("%w: keywords too long", ErrInvalidRequest)
	}
	var categoryName string
	if req.CategoryID != nil {
		name, ok, err := s.categoryRes.GetNameByID(ctx, *req.CategoryID)
		if err != nil {
			return "", fmt.Errorf("%w: category lookup: %v", ErrInvalidRequest, err)
		}
		if !ok {
			return "", fmt.Errorf("%w: category %d not exists", ErrInvalidRequest, *req.CategoryID)
		}
		categoryName = name
	}
	return categoryName, nil
}

func (s *CopywritingService) checkRateLimit(ctx context.Context, userID int64) error {
	if s.redis == nil {
		// redis 未配置：fail-open 放行（与 main.go REDIS_ADDR 判空模式一致）
		return nil
	}
	key := fmt.Sprintf("ai:copywriting:%d:%s", userID, s.now().UTC().Format("200601021504"))
	n, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		// fail-open：限流系统故障不阻塞主流程
		return nil
	}
	if n == 1 {
		_ = s.redis.Expire(ctx, key, rateWindowSec*time.Second).Err()
	}
	if n > int64(rateLimitPerMin) {
		return fmt.Errorf("%w: %d/min exceeded", ErrRateLimited, rateLimitPerMin)
	}
	return nil
}

const systemPrompt = `你是直播竞拍平台的商品文案专家。请根据图片和卖家提供的关键词，生成商品的：
1. name: ≤30字标题，含品类与关键卖点
2. description: 80-150字描述，分点列卖点
3. selling_points: 3-5个短语，每个≤12字
4. suggested_start_price: 起拍价建议（人民币元，纯数字字符串，参考二手市场行情，保守偏低 30%-50%）

严格输出 JSON，schema：
{"name":"","description":"","selling_points":[],"suggested_start_price":""}
不要任何额外解释、不要 markdown 代码块。`

func (s *CopywritingService) buildChatRequest(req *CopywritingRequest, categoryName string) *sharedllm.ChatRequest {
	parts := make([]sharedllm.ContentPart, 0, len(req.Images)+1)
	for _, u := range req.Images {
		parts = append(parts, sharedllm.ContentPart{Type: "image_url", ImageURL: &sharedllm.ImageURL{URL: u}})
	}
	var info []string
	if categoryName != "" {
		info = append(info, "类目: "+categoryName)
	}
	if strings.TrimSpace(req.Keywords) != "" {
		info = append(info, "关键词: "+req.Keywords)
	}
	if len(info) > 0 {
		parts = append(parts, sharedllm.ContentPart{Type: "text", Text: strings.Join(info, "\n")})
	} else {
		parts = append(parts, sharedllm.ContentPart{Type: "text", Text: "请基于图片生成商品文案。"})
	}
	return &sharedllm.ChatRequest{
		Model: s.defaultModel,
		Messages: []sharedllm.ChatMessage{
			{Role: "system", Content: []sharedllm.ContentPart{{Type: "text", Text: systemPrompt}}},
			{Role: "user", Content: parts},
		},
		Temperature:    0.6,
		MaxTokens:      600,
		ResponseFormat: &sharedllm.ResponseFormat{Type: "json_object"},
	}
}

func parseCopywritingOutput(raw string) (*CopywritingResponse, error) {
	raw = strings.TrimSpace(raw)
	// 防御：剥离可能的 markdown 代码块
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var out CopywritingResponse
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	if out.Name == "" || out.Description == "" || len(out.SellingPoints) == 0 || out.SuggestedStartPrice == "" {
		return nil, fmt.Errorf("missing required field")
	}
	if _, err := decimal.NewFromString(out.SuggestedStartPrice); err != nil {
		return nil, fmt.Errorf("price not a number: %w", err)
	}
	return &out, nil
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend/product && go test ./service/... -run TestCopywriting_ -v`
Expected: 10 PASS

- [ ] **Step 5: Commit**

```bash
git add backend/product/service/copywriting.go backend/product/service/copywriting_test.go
git commit -m "feat(product/service): add AI copywriting generation service"
```

---

## Task 8: HTTP Handler — 鉴权与状态码映射

**Files:**
- Create: `backend/product/handler/copywriting.go`
- Create: `backend/product/handler/copywriting_test.go`

- [ ] **Step 1: 写失败用例**

```go
// backend/product/handler/copywriting_test.go
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"

	"product-service/service"
)

// stubCopySvc 用于隔离 handler 测试。
type stubCopySvc struct {
	resp *service.CopywritingResponse
	err  error
}

func (s *stubCopySvc) Generate(ctx context.Context, userID int64, req *service.CopywritingRequest) (*service.CopywritingResponse, error) {
	return s.resp, s.err
}

func setupCopyRouter(t *testing.T, svc CopywritingServiceAPI, role int, userID int64) *server.Hertz {
	t.Helper()
	h := server.New(server.WithExitWaitTime(0))
	hh := NewCopywritingHandler(svc)
	h.POST("/api/v1/products/ai/copywriting", func(c context.Context, ctx *app.RequestContext) {
		ctx.Set("user_id", userID)
		ctx.Set("user_role", role)
		hh.Generate(c, ctx)
	})
	return h
}

func mustBody(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func TestCopyHandler_Success_200(t *testing.T) {
	svc := &stubCopySvc{resp: &service.CopywritingResponse{Name: "x", Description: "y", SellingPoints: []string{"a"}, SuggestedStartPrice: "1"}}
	h := setupCopyRouter(t, svc, 1, 100)

	body := mustBody(t, service.CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/products/ai/copywriting",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	if w.Result().StatusCode() != http.StatusOK {
		t.Fatalf("status want=200 got=%d body=%s", w.Result().StatusCode(), w.Result().Body())
	}
}

func TestCopyHandler_ForbiddenRole_403(t *testing.T) {
	svc := &stubCopySvc{}
	h := setupCopyRouter(t, svc, 0, 100)
	body := mustBody(t, service.CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/products/ai/copywriting",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	if w.Result().StatusCode() != http.StatusForbidden {
		t.Fatalf("status want=403 got=%d", w.Result().StatusCode())
	}
}

func TestCopyHandler_BadRequest_400(t *testing.T) {
	svc := &stubCopySvc{err: service.ErrInvalidRequest}
	h := setupCopyRouter(t, svc, 1, 100)
	body := mustBody(t, service.CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/products/ai/copywriting",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	if w.Result().StatusCode() != http.StatusBadRequest {
		t.Fatalf("status want=400 got=%d", w.Result().StatusCode())
	}
}

func TestCopyHandler_RateLimited_429(t *testing.T) {
	svc := &stubCopySvc{err: service.ErrRateLimited}
	h := setupCopyRouter(t, svc, 1, 100)
	body := mustBody(t, service.CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/products/ai/copywriting",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	if w.Result().StatusCode() != http.StatusTooManyRequests {
		t.Fatalf("status want=429 got=%d", w.Result().StatusCode())
	}
}

func TestCopyHandler_Upstream_502(t *testing.T) {
	svc := &stubCopySvc{err: service.ErrUpstreamFailed}
	h := setupCopyRouter(t, svc, 2, 100)
	body := mustBody(t, service.CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/products/ai/copywriting",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	if w.Result().StatusCode() != http.StatusBadGateway {
		t.Fatalf("status want=502 got=%d", w.Result().StatusCode())
	}
}

func TestCopyHandler_Timeout_504(t *testing.T) {
	svc := &stubCopySvc{err: service.ErrUpstreamTimeout}
	h := setupCopyRouter(t, svc, 1, 100)
	body := mustBody(t, service.CopywritingRequest{Images: []string{"https://cdn.example.com/a.jpg"}})
	w := ut.PerformRequest(h.Engine, http.MethodPost, "/api/v1/products/ai/copywriting",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	if w.Result().StatusCode() != http.StatusGatewayTimeout {
		t.Fatalf("status want=504 got=%d", w.Result().StatusCode())
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend/product && go test ./handler/... -run TestCopyHandler -v`
Expected: FAIL — `undefined: NewCopywritingHandler` / `CopywritingServiceAPI`

- [ ] **Step 3: 实现 handler**

```go
// backend/product/handler/copywriting.go
package handler

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"

	"product-service/service"
)

// CopywritingServiceAPI 抽象 service 行为，便于 handler 测试替身。
type CopywritingServiceAPI interface {
	Generate(ctx context.Context, userID int64, req *service.CopywritingRequest) (*service.CopywritingResponse, error)
}

// CopywritingHandler HTTP 入口。
type CopywritingHandler struct {
	svc CopywritingServiceAPI
}

// NewCopywritingHandler 构造 handler。
func NewCopywritingHandler(svc CopywritingServiceAPI) *CopywritingHandler {
	return &CopywritingHandler{svc: svc}
}

// Generate POST /api/v1/products/ai/copywriting
func (h *CopywritingHandler) Generate(ctx context.Context, c *app.RequestContext) {
	role := c.GetInt("user_role")
	if role != 1 && role != 2 {
		c.JSON(403, map[string]interface{}{"code": "forbidden_role", "message": "需要商家或管理员权限"})
		return
	}
	userID := c.GetInt64("user_id")
	if userID == 0 {
		c.JSON(401, map[string]interface{}{"code": "unauthorized", "message": "未登录"})
		return
	}

	var req service.CopywritingRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]interface{}{"code": "invalid_request", "message": err.Error()})
		return
	}

	resp, err := h.svc.Generate(ctx, userID, &req)
	if err != nil {
		mapCopywritingError(c, err)
		return
	}
	c.JSON(200, resp)
}

func mapCopywritingError(c *app.RequestContext, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidRequest):
		c.JSON(400, map[string]interface{}{"code": "invalid_request", "message": err.Error()})
	case errors.Is(err, service.ErrRateLimited):
		c.JSON(429, map[string]interface{}{"code": "rate_limited", "message": err.Error()})
	case errors.Is(err, service.ErrUpstreamTimeout):
		c.JSON(504, map[string]interface{}{"code": "upstream_timeout", "message": err.Error()})
	case errors.Is(err, service.ErrInvalidOutput):
		c.JSON(502, map[string]interface{}{"code": "upstream_invalid_output", "message": err.Error()})
	case errors.Is(err, service.ErrUpstreamFailed):
		c.JSON(502, map[string]interface{}{"code": "upstream_failed", "message": err.Error()})
	default:
		c.JSON(500, map[string]interface{}{"code": "internal_error", "message": err.Error()})
	}
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend/product && go test ./handler/... -run TestCopyHandler -v`
Expected: 6 PASS

- [ ] **Step 5: Commit**

```bash
git add backend/product/handler/copywriting.go backend/product/handler/copywriting_test.go
git commit -m "feat(product/handler): add Copywriting HTTP handler with status mapping"
```

---

## Task 9: `main.go` 装配 + 路由注册

**Files:**
- Modify: `backend/product/main.go`
- （`backend/product/dao/category.go` **无需改**：已有 `GetByID`）

> **现状核对**：`CategoryDAO` 已有 `GetByID(ctx, id) (*model.Category, error)`（找不到返回 `gorm.ErrRecordNotFound`），以及 `RedisConfig`。因此不新增 DAO 方法，改为在 main 包内定义一个薄适配器把 `GetByID` 转换成 `service.CategoryNameResolver.GetNameByID(ctx, id) (name, ok, err)`；redis 沿用 `main.go` 既有"`REDIS_ADDR` 判空才建 client"模式，service 端 nil → fail-open。

- [ ] **Step 1: 在 main 包内定义 CategoryNameResolver 适配器**

在 [main.go](file:///Users/bytedance/myself/coding/dy-ai-live-auction-fullstack-cc/backend/product/main.go) 中追加一个适配类型（复用既有 `GetByID`，不碰 DAO）：

```go
// categoryNameAdapter 把 dao.CategoryDAO.GetByID 适配成 service.CategoryNameResolver。
type categoryNameAdapter struct{ dao *dao.CategoryDAO }

func (a categoryNameAdapter) GetNameByID(ctx context.Context, id int64) (string, bool, error) {
	cat, err := a.dao.GetByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return cat.Name, true, nil
}
```

> 注：import 需加 `"errors"` 与 `"gorm.io/gorm"`（product 已依赖 gorm）。

- [ ] **Step 2: 修改 `main.go` 装配逻辑**

import 增加：

```go
"context"
"errors"
sharedllm "shared/llm"
"gorm.io/gorm"
// 注意：redis 已在现有 import 中（main.go 已使用 github.com/redis/go-redis/v9）
```

在 `main()` 中：

1. `cfg, nacosLoader := config.LoadFromNacosWithFallback()` 之后增加：

```go
config.ResolveLLMSecrets(cfg)
```

2. 复用既有"`REDIS_ADDR` 判空"模式初始化 redis（可为 nil）。现有 main.go 已有 `viewerCounter` 用到 `os.Getenv("REDIS_ADDR")`，这里**抽出一个共享 client**避免重复创建：

```go
var redisClient *redis.Client
if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		PoolSize: cfg.Redis.PoolSize,
	})
}
```

> 现有 `viewerCounter` 分支可改为复用该 `redisClient`（若非 nil）：`viewerCounter = service.NewRedisLiveViewerCounter(redisClient)`，避免建两个连接池。

3. 在 `categoryHandler := handler.NewCategoryHandler(categoryService)` 之后追加：

```go
llmProvider := sharedllm.NewDoubaoProvider(sharedllm.DoubaoOptions{
	BaseURL: cfg.LLM.Doubao.BaseURL,
	APIKey:  cfg.LLM.Doubao.APIKey,
	Model:   cfg.LLM.Doubao.Model,
	Timeout: time.Duration(cfg.LLM.TimeoutMs) * time.Millisecond,
})
copyService := service.NewCopywritingService(llmProvider, categoryNameAdapter{dao: categoryDAO}, redisClient, cfg.LLM.Doubao.Model)
copywritingHandler := handler.NewCopywritingHandler(copyService)
```

4. 修改 `registerRoutes` 签名增加 `copywritingHandler *handler.CopywritingHandler` 参数；并在 `v1` 组中加：

```go
// AI 文案生成（商家/管理员）
v1.POST("/products/ai/copywriting", copywritingHandler.Generate)
```

5. 在 `main()` 调用 `registerRoutes` 时把新 handler 传进去。

- [ ] **Step 3: 编译验证**

Run: `cd backend/product && go build ./...`
Expected: 无错误

- [ ] **Step 4: 整体跑全部测试**

Run: `cd backend/product && go test ./... && cd ../shared/llm && go test ./...`
Expected: 全部 PASS

- [ ] **Step 5: Commit**

```bash
git add backend/product/main.go
git commit -m "feat(product): wire AI copywriting provider and route"
```

---

## Task 10: 端到端冒烟（手工验证 + 全量回归）

**Files:** 无新增

- [ ] **Step 1: 启动服务（需本地 mysql、redis）**

```bash
export ARK_API_KEY=<真实 key>
export DB_HOST=localhost DB_USER=root DB_PASSWORD=<...> DB_NAME=auction
export REDIS_ADDR=localhost:6379
cd backend/product && go run .
```

Expected: 看到 `Product service starting on :8081`

- [ ] **Step 2: 用 curl 触发**

```bash
curl -X POST http://localhost:8081/api/v1/products/ai/copywriting \
  -H 'Content-Type: application/json' \
  -d '{
    "images": ["https://cdn.example.com/iphone-12.jpg"],
    "keywords": "九成新 自用一年"
  }'
```

> 注：直连 product-service 时 `user_id`/`user_role` 默认为 0，会返回 403。完整链路需经 gateway-service 注入 JWT 上下文，本步骤主要验证 LLM 链路；开发阶段可临时在 handler 入口手动 set 角色快速验证。

Expected: 通过 gateway 时 200 + JSON 含 `name / description / selling_points / suggested_start_price`

- [x] **Step 3: 全量回归**

```bash
cd backend/shared/llm && go test ./... -count=1 && \
cd ../../product && go test ./... -count=1
```

Expected: ALL PASS

- [x] **Step 4: 修复回归测试签名同步并提交**

`registerRoutes` 新增 `copywritingHandler *handler.CopywritingHandler` 参数后，既有 `backend/product/admin_route_test.go` 需要同步传入 `handler.NewCopywritingHandler(nil)`，否则 product 包测试编译失败。

```bash
git add backend/product/admin_route_test.go
git commit -m "test(product): update route registration test for copywriting handler"
```

---

## 自审清单

- ✅ **Spec §2 关键决策（含 shared/llm 路径）** → Task 1 + Task 5
- ✅ **Spec §4.1 配置扩展** → Task 6 + Task 9
- ✅ **Spec §4.2 Provider 接口** → Task 2
- ✅ **Spec §4.3 Doubao 实现** → Task 3 + Task 4
- ✅ **Spec §4.4 业务 DTO** → Task 7
- ✅ **Spec §4.5 路由 + replace 指令** → Task 5 + Task 9
- ✅ **Spec §5 Prompt 模板** → Task 7 systemPrompt
- ✅ **Spec §6.1 限流（含 redis 可选 fail-open）** → Task 7 checkRateLimit
- ✅ **Spec §6.2 输入校验（含类目名解析）** → Task 7 validate + Task 9 categoryNameAdapter
- ✅ **Spec §6.3 输出解析** → Task 7 parseCopywritingOutput
- ✅ **Spec §6.4 鉴权** → Task 8 mapCopywritingError + role check
- ✅ **Spec §7 测试大纲** → Task 3/4/6/7/8 + admin route 同步（共 ≥24 用例，含 nil-redis fail-open）
- ⏳ **Spec §8 监控指标** → 留给 Plan B（监控 + Nacos 热更新）
- ⏳ **Spec §11 后续扩展** → 留给后续 spec

类型一致性：
- `service.CategoryNameResolver.GetNameByID` ↔ `main.categoryNameAdapter.GetNameByID`（包 `dao.CategoryDAO.GetByID`）一致
- `CopywritingServiceAPI.Generate` ↔ `CopywritingService.Generate` 签名一致
- `sharedllm.Provider` 在 service/main 中均使用同名 alias 引用
- `redisClient` 可为 nil：`NewCopywritingService` 第三参允许 nil，service 内 fail-open

---

## 执行状态

本计划已按 Inline Execution 执行完成。后续如继续推进 Plan B（Prometheus 指标、Nacos 热更新、前端管理端按钮、真实 gateway 链路冒烟），应另起 plan，避免把 MVP 已完成范围和后续增强混在一起。

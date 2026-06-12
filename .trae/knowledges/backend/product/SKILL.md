---
name: knowledge-backend-product
description: >
  Covers product-service 的商品管理、AI 文案生成、LLM 供应商集成、配置管理和 Nacos 配置规范。
  Navigate when: modifying backend/product handlers, services, configs, or LLM integrations.
  Excludes: auction-service, gateway-service, test-service.
  Keywords: backend/product, product-service, copywriting, LLM, Doubao, Ark, Nacos, config
---

## Module Structure

product-service 是商品域服务，负责商品 CRUD、类目管理、AI 文案生成和直播间管理；核心风险集中在 LLM 供应商集成、配置管理和 Nacos 配置同步。

### Directory Layout
- `backend/product/handler/` — HTTP 处理器，包括商品、类目、AI 文案接口。
- `backend/product/service/` — 业务逻辑层，包括文案生成编排。
- `backend/product/config/` — 配置加载、Nacos 集成和环境变量解析。
- `backend/product/pkg/llm/` — LLM 抽象层（已迁移至 `backend/shared/llm`）。
- `backend/product/model/` — 数据模型定义。

### Key Entry Points
- `backend/product/handler/copywriting.go` — AI 文案生成 HTTP 入口。
- `backend/product/service/copywriting.go` — 文案生成业务编排。
- `backend/product/config/config.go` — 配置结构和加载逻辑。
- `configs/nacos/product-config.yaml` — Nacos 配置模板。

## Gotchas

### LLM 供应商集成
- **模型选型必须匹配接口协议**：当前代码使用 OpenAI 兼容接口 `/api/v3/chat/completions`，需选用支持 chat/vision 的模型，而非视频生成模型（如 Seedance）或纯文本模型
- **Endpoint ID 与模型名称区别**：火山方舟控制台展示的模型名称（如 `Doubao-1.5-lite-32k`）不一定等于实际可用的 Endpoint ID，需从控制台获取准确的接入点标识
- **Vision 模型选择**：C3 AI 文案功能需要处理图片输入，应选用支持 `image_url` 的多模态模型（如 `doubao-seed-1-6-vision-250815`），而非纯文本模型
- **超时配置**：LLM 调用需设置合理超时（推荐 30s），避免阻塞业务请求；配置项为 `llm.timeout_ms`

### API 密钥管理
- **环境变量注入**：生产环境 `ARK_API_KEY` 必须通过服务器环境文件（如 `/srv/auction/env/.env.demo`）配置，由 `product-service` 容器读取
- **配置占位符**：Nacos 配置中 `api_key` 应使用 `${ARK_API_KEY}` 占位符，不写明文密钥
- **安全规范**：严禁将 API Key 提交至 Git 仓库或在对话中明文传输；更新密钥需修改服务器 `.env` 文件并重启对应服务
- **Gateway 职责**：Gateway 仅负责请求转发和鉴权，不直接调用 AI 接口，也不持有 AI 密钥

### Nacos 配置管理
- **配置段完整性**：`product-config.yaml` 必须包含完整的 `llm` 配置段，包括 `provider`、`timeout_ms`、`doubao.base_url`、`doubao.api_key`、`doubao.model`
- **默认值归一化**：代码层需实现 `ApplyDefaults` 逻辑，确保即使 Nacos 配置缺少 `llm` 段也能获得合法默认值
- **配置热更新**：LLM 配置变更后需重启 `product-service` 才能生效

### 商品分类数据治理与测试 SDK 默认分类 (Product Category Data Governance)

**问题背景**：管理端存在大量「未分类」商品（`category_id IS NULL`），且 H5 控制台和独立测试平台创建商品时未传入 `category_id`，持续产生脏数据。

**根因分析**：
- 历史数据：`products.category_id IS NULL`，本地有 292 条未分类记录
- 造数入口：`backend/test/client/auction/client.go` 的 `CreateProductReq` 无 `category_id` 字段
- H5 控制台和独立测试平台都经测试 SDK 创建商品，均未传分类

**修复方案**：
1. **测试 SDK 自动填充默认分类** (`backend/test/client/auction/client.go`)
   - `CreateProductAs` 方法中，当 `req.Images` 为空时填充默认图片
   - 同步补充 `category_id` 字段，未传入时自动填充默认分类（如「其他」分类 ID）

2. **数据清洗** (`scripts/backfill_product_category.sql`)
   - 基于商品名称关键词自动映射（如含「玉」「镯」→ 珠宝首饰）
   - 无法推断的保留 `NULL` 并显示「未分类」
   - 删除测试残留数据（名称含 "E2E 测试拍品"、UUID 等无业务语义数据）

3. **后端契约扩展**
   - `POST /api/v1/products`：请求体新增 `category_id?: number | null`
   - `PUT /api/v1/products/:id`：支持编辑商品分类
   - 非法分类返回 `400` 错误

**关键代码模式**：
```go
// 测试 SDK 自动填充默认分类
func (c *Client) CreateProductAs(ctx context.Context, merchantID int64, req CreateProductReq) (*Product, error) {
    // 自动填充默认图片
    if len(req.Images) == 0 {
        req.Images = DefaultProductImages
    }
    // 自动填充默认分类
    if req.CategoryID == 0 {
        req.CategoryID = DefaultCategoryID // 如 1 = "其他"
    }
    // ... 继续创建逻辑
}
```

**测试残留识别标准**：
- 名称包含 "E2E 测试拍品"、"Fixed Price Demo"
- 描述包含 "orchestrator auto-generated"、随机 UUID
- 无业务语义，无法映射到真实分类

**来源**：session:6a25bce00bfcee1b04fb15bd

---

### 商品发布状态管理
- **状态枚举**：商品 `status` 字段通常包含 `0=草稿/未发布`、`1=已发布` 等状态，需确保 Admin 端与后端定义一致
- **列表过滤**：用户端商品列表接口（如 `GET /api/v1/products`）必须过滤 `status=1`（已发布），避免未发布商品对用户可见
- **管理端权限**：商家/管理员在 Admin 端可查看全部状态商品，但需通过角色权限控制，而非简单暴露所有数据
- **状态变更**：商品从"未发布"到"已发布"的状态变更需同步更新数据库并确保缓存一致性

### 身份透传
- **Gateway 透传 Header**：`product-service` 应从 HTTP Header 读取 `X-User-ID` 和 `X-User-Role`，而非从 JWT 直接解析
- **角色兼容**：文案生成接口允许 `streamer`、`merchant`、`admin` 角色访问，需正确解析 Gateway 透传的角色标识

## Architecture

### AI 文案生成链路
```
Admin 前端 → Gateway → product-service → DoubaoProvider → 火山方舟 API
                ↓              ↓
           X-User-ID      LLMConfig
           X-User-Role    (Nacos + Env)
```

### 配置加载优先级
1. Nacos 配置（`product-config.yaml`）
2. 环境变量覆盖（`${ARK_API_KEY}` 解析）
3. 代码默认值（`ApplyDefaults`）

## Patterns

### LLM 调用模式
```go
provider := llm.NewProvider(cfg)
response, err := provider.Chat(ctx, &llm.ChatRequest{
    Model: cfg.Model,
    Messages: []llm.ChatMessage{
        {Role: "system", Content: []llm.ContentPart{{Type: "text", Text: systemPrompt}}},
        {Role: "user", Content: multimodalContent},
    },
    ResponseFormat: &llm.ResponseFormat{Type: "json_object"},
})
```

### 配置默认值应用
```go
func ApplyDefaults(cfg *Config) {
    if cfg.LLM.Provider == "" {
        cfg.LLM.Provider = "doubao"
    }
    if cfg.LLM.TimeoutMs == 0 {
        cfg.LLM.TimeoutMs = 30000
    }
    // ...
}
```

## Conventions
- 技术方案、接口契约优先使用中文注释，代码标识保留 canonical 写法
- LLM 相关测试需使用 httpmock 模拟上游响应，避免真实调用
- 错误处理需区分网络错误、配额超限、内容审核等不同类型

### 内部订单创建接口 (Internal Order Creation API)

**功能概述**：为 `auction-service` 提供内部 API，在竞拍结束时自动创建待支付订单，补全"中标事实 -> 订单"的闭环。

**接口契约**：
- `POST /internal/orders/from-auction-result`
- 请求：`{ auction_id, product_id, winner_id, final_price }`
- 响应：`{ order_id, status, created_at }` 或幂等返回已有订单

**幂等设计**：
- 数据库层：`orders.auction_id` 唯一索引保证同一竞拍只创建一个订单
- 冲突处理：重复调用时返回已有订单（`200 OK`），而非报错

**实现要点**：
```go
// Handler 层
func (h *OrderHandler) CreateFromAuctionResult(c *gin.Context) {
    var req CreateFromAuctionReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse{Code: "INVALID_PARAM"})
        return
    }
    
    order, err := h.orderService.CreateFromAuction(ctx, req)
    if err != nil {
        // 检查是否是唯一索引冲突（幂等返回）
        if isDuplicateKeyError(err) {
            existing, _ := h.orderService.GetByAuctionID(ctx, req.AuctionID)
            c.JSON(200, existing) // 返回已有订单
            return
        }
        c.JSON(500, ErrorResponse{Code: "INTERNAL_ERROR"})
        return
    }
    c.JSON(201, order)
}
```

**调用方 (auction-service) 责任**：
- 在 `EndAuction` 确定中标者后调用
- 订单创建成功后再发送中标通知
- 处理网络超时等异常情况（可配合重试或记录告警）

**来源**：session:6a23e4a22ec60aa1a73a5f31

### 商家订单列表内部接口 (Admin Order List Internal API)

**功能概述**：为 Admin 商家订单管理页面提供后端支持，包括订单列表查询、搜索、状态统计和买家信息回填。

**接口契约**：
- `GET /admin/orders` — 商家订单列表
  - Query: `search`（关键词模糊搜索）、`status`（状态筛选）、`page`、`page_size`
  - 响应：`{ list: Order[], total, summary: { pending_payment, pending_shipment, shipped, completed } }`

**买家信息回填模式**：
```go
// 1. 查询订单列表（仅含 winner_id）
orders, total, err := h.orderRepo.SearchOrders(ctx, sellerID, search, status, page, pageSize)

// 2. 批量提取 winner IDs
winnerIDs := extractWinnerIDs(orders)

// 3. 通过内部接口批量获取用户摘要
summaryMap, err := auctionClient.BatchGetUserSummaries(ctx, winnerIDs)
if err != nil {
    // 降级：记录 WARN，继续返回不含买家信息的订单
    log.Printf("[WARN] batch get user summary failed: %v", err)
} else {
    // 回填买家信息
    for _, order := range orders {
        if summary, ok := summaryMap[order.WinnerID]; ok {
            order.BuyerUsername = summary.Username
            order.BuyerAvatar = summary.Avatar
        }
    }
}
```

**跨服务调用规范**：
- 内部接口：`POST /internal/users/batch`（`auction-service` 提供）
- 请求：`{ user_ids: number[] }`
- 响应：`map[user_id]{ username, avatar }`
- 降级策略：调用失败时静默跳过，不阻断主流程

**HTTP Client 连接复用**：
```go
// 必须 drain 响应体以保证连接复用
resp, err := httpClient.Do(req)
if err != nil {
    return nil, err
}
defer func() {
    io.Copy(io.Discard, resp.Body)
    resp.Body.Close()
}()
```

**商家隔离原则**：
- 所有查询必须带 `seller_id` 过滤
- 后端从 JWT 解析当前用户 ID，作为 `seller_id` 查询条件
- 禁止返回非当前商家的订单数据

**测试覆盖**：
- Handler 层：搜索、状态筛选、分页、商家隔离
- Service 层：买家信息回填成功与降级路径
- Client 层：HTTP 调用、错误处理、连接复用

**来源**：session:6a2419153eefb8c530aa7658

### 用户订单列表接口扩展 (User Order List API Enhancement)

**问题背景**：H5 用户订单列表最初只返回订单基础字段（`id/auction_id/product_id/final_price/status/created_at`），前端只能显示「商品 #id」和「竞拍场次 #id」的 fallback 文案，用户体验不佳。

**解决方案**：扩展用户订单列表接口，返回商品展示信息。

**接口契约变更**：
- `GET /api/v1/orders` — 用户订单列表
  - 新增响应字段：
    - `product_name`: string — 商品名称
    - `product_image`: string — 商品首图 URL
    - `seller_name`: string — 商家名称

**实现模式**：
```go
// DAO 层：订单 + 商品 + 商家 JOIN 查询
func (dao *OrderDAO) ListWithProductDisplay(ctx context.Context, userID int64, page, pageSize int) ([]OrderWithDisplay, int64, error) {
    // SELECT orders.*, products.name as product_name, products.images as product_images, users.name as seller_name
    // FROM orders
    // LEFT JOIN products ON products.id = orders.product_id
    // LEFT JOIN users ON users.id = products.owner_id
    // WHERE orders.winner_id = ?
}
```

**数据边界原则**：
- 只 join product-service 自己的表（`products`、`users`）
- 不跨服务查 auction 表或 live_streams 表
- 直播间名称不属于 product-service 数据所有权，不应在订单列表返回

**测试要点**：
- 验证返回字段包含 `product_name`、`product_image`、`seller_name`
- 验证商品图片不存在时返回空字符串（而非报错）
- 验证商家名称为空时返回「商家 #seller_id」兜底

**来源**：session:6a2416b73eefb8c530aa74a2

---

### 竞拍历史记录接口契约修复 (Auction History API Contract Fix)

**问题背景**：H5「我的竞拍记录」页面需要区分「待处理中标」和「已处理中标」，但后端 `/orders/history` 接口未返回订单 `status` 字段，导致前端无法判断。

**修复方案**：

**1. 数据模型扩展** (`backend/product/dao/history.go`)
```go
type UserHistoryItem struct {
    AuctionID   int64   `json:"auction_id"`
    ProductName string  `json:"product_name"`
    FinalPrice  float64 `json:"final_price"`
    IsWinner    bool    `json:"is_winner"`
    BidCount    int     `json:"bid_count"`
    Status      int     `json:"status"`        // 新增：订单状态
    CreatedAt   string  `json:"created_at"`
}
```

**2. SQL 查询调整**
- 从 `orders` 表获取 `status` 字段
- 修复 SQLite 兼容性：将 MySQL 特有的 `DATE_FORMAT` 改为标准 `created_at as created_at`
- 以 `orders` 表为 SSOT（单一事实源），不再依赖 product-service 本地 `auctions/bids` 镜像

**3. 查询逻辑优化**
```go
// 以 orders 表为驱动，LEFT JOIN 其他表
SELECT 
    o.auction_id,
    p.name as product_name,
    o.final_price,
    o.winner_id IS NOT NULL as is_winner,
    o.status,
    a.created_at
FROM orders o
LEFT JOIN products p ON p.id = o.product_id
LEFT JOIN auctions a ON a.id = o.auction_id
WHERE o.winner_id = ?
```

**关键决策**：
- **SSOT 原则**：历史记录从 `orders` 表获取，而非 `auctions` + `bids` 组合查询
- **状态语义**：`status=0` 表示待支付（未读），`status=1` 表示已支付/完结（已读）
- **兼容性**：SQL 语法需同时兼容 MySQL（生产）和 SQLite（测试）

**测试验证**：
- 单元测试断言响应包含 `status` 字段
- 验证 SQLite 环境下查询正常执行
- 验证 `status` 值与订单实际状态一致

**来源**：session:6a2464ce00057ea64ca286e5

---

### 历史中标订单数据回填 (Historical Auction Result Backfill)

**问题背景**：竞拍结束产生中标事实后，没有自动创建待支付订单，导致「我的竞拍」页面显示竞拍成功数量为 0，但消息通知中能看到中标记录，两个数据源不一致。

**解决方案**：通过内部 API 对已有中标竞拍做幂等 backfill，补齐历史订单。

**执行流程**：
1. **重启后端服务**：让最新的结算建单链路和数据库迁移生效
2. **查询缺失订单**：找出 `winner_id IS NOT NULL` 但 `orders` 表中没有对应记录的竞拍
3. **幂等补单**：通过 `POST /internal/orders/from-auction-result` 内部接口逐个创建订单

**补单脚本要点**：
```bash
# 查询需要补单的中标竞拍
SELECT a.id, a.product_id, a.winner_id, a.current_price 
FROM auctions a 
LEFT JOIN orders o ON o.auction_id = a.id 
WHERE a.winner_id IS NOT NULL AND o.id IS NULL

# 循环调用内部接口创建订单
curl -X POST http://localhost:8081/internal/orders/from-auction-result \
  -H "Content-Type: application/json" \
  -H "X-Internal-Token: dev" \
  -d '{"auction_id":$auction_id,"product_id":$product_id,"winner_id":$winner_id,"final_price":"$final_price"}'
```

**脏数据处理**：
- 部分历史商品可能 `owner_id=NULL`，导致订单创建失败（订单服务 fail-closed）
- 需先修正这些商品的 `owner_id`，再重试补单

**验证要点**：
- 补单后 `orders` 表记录数应与中标竞拍数一致
- 当前用户通过 `/api/v1/orders` 能正确返回订单列表
- 订单数据与中标通知数据一致

**来源**：session:6a2416b73eefb8c530aa74a2

---

## Feature Knowledge

### AI 文案生成 (C3 Copywriting)

**功能概述**：商家输入商品图片和关键词，后端调用 Doubao/Ark 大模型生成营销文案（标题、描述、卖点、起拍价建议）。

**接口契约**：
- `POST /api/v1/products/ai/copywriting`
- 请求：`{ images: string[], category_id?: number, keywords?: string }`
- 响应：`{ name, description, selling_points: string[], suggested_start_price }`

**模型选型经验**：
- **初始尝试**：`Doubao-1.5-lite-32k` — 纯文本模型，不支持图片输入
- **最终选择**：`doubao-seed-1-6-vision-250815` — 支持 vision 的多模态模型，可处理 `image_url` 输入
- **排除选项**：`doubao-seedance-2-0-fast-260128` — Seedance 视频生成模型，不适配 chat/completions 协议

**关键设计决策**：
1. **服务归属**：直接放在 `product-service`，避免新建 `ai-service`（MVP 阶段）
2. **同步调用**：MVP 采用同步调用（3-5s 内返回），后续可扩展为 outbox+轮询
3. **图片输入**：支持最多 6 张图片 URL，通过 `image_url` 类型 ContentPart 传入
4. **JSON 输出**：使用 `response_format: {type: "json_object"}` 强制结构化输出

**Prompt 模板要点**：
- 系统角色：直播竞拍平台商品文案专家
- 输出约束：name ≤30字，description 80-150字，selling_points 3-5条每条≤12字
- 价格建议：保守偏低 30%-50%，以人民币元为单位

**测试策略**：
- 单元测试覆盖 Provider 封装（成功/超时/4xx/5xx 路径）
- 单元测试覆盖文案生成业务逻辑（Prompt 组装、JSON 解析、空类目处理）
- 单元测试覆盖 Handler 层（鉴权、参数校验、错误码映射）

**来源**：session:6a1c6af6959156a8dfc85954

### LLM 供应商抽象层 (Shared LLM Provider)

**目录结构**：
- `backend/shared/llm/provider.go` — Provider 接口定义
- `backend/shared/llm/doubao.go` — Doubao/Ark 实现
- `backend/shared/llm/factory.go` — Provider 工厂

**关键约束**：
- 环境变量注入 API Key，禁止硬编码
- 超时控制默认 30s
- 错误分类：网络错误、配额超限、内容审核

**使用模式**：
```go
provider := llm.NewProvider(cfg)
response, err := provider.Generate(ctx, prompt)
```

**来源**：session:6a1c6af6959156a8dfc85954

### 竞拍规则模板管理 (Auction Rule Template Management)

**功能概述**：支持商家在 Admin 后台创建、管理竞拍规则模板，并一键应用到商品。

**数据模型**：
- `auction_rule_templates` 表：模板元数据（名称、描述、商家归属）
- `auction_rules` 表：商品实际规则（由模板应用生成或独立创建）

**核心接口**：
- `GET /api/v1/admin/auction-rule-templates` — 商家模板列表
- `POST /api/v1/admin/auction-rule-templates` — 创建模板
- `PUT /api/v1/admin/auction-rule-templates/:id` — 编辑模板
- `DELETE /api/v1/admin/auction-rule-templates/:id` — 删除模板
- `POST /api/v1/admin/products/:id/apply-rule-template` — 应用模板到商品

**Upsert 语义关键修复**：
- 问题：GORM `Assign(rule).FirstOrCreate(rule)` 无法覆盖 nil/zero 字段
- 场景：旧规则有 `cap_price=1000`，应用无封顶价模板后 `cap_price` 仍为 1000
- 修复：显式字段覆盖模式
```go
// 存在则 Updates（map 可传 nil/zero），不存在则 Create
if exists {
    db.Model(&rule).Updates(map[string]interface{}{
        "start_price": req.StartPrice,
        "cap_price": req.CapPrice,  // nil 会正确覆盖为 NULL
        // ...
    })
} else {
    db.Create(&rule)
}
```

**与独立测试平台的关系**：
- 测试平台不依赖模板功能，直接调用 `POST /api/v1/products/:id/rules`
- 模板功能仅服务于 Admin 商家后台的「配置复用」场景
- 两者底层共享 `auction_rules` 表，但入口不同

**来源**：session:6a241e023eefb8c530aa78a6

---

## Deployment Notes

### 上线前检查清单
- [ ] `product-service` 运行环境已设置 `ARK_API_KEY`
- [ ] Nacos `product-config.yaml` 包含完整 `llm` 配置段
- [ ] 实例出网放行 `ark.cn-beijing.volces.com:443`
- [ ] 商品图片 URL 必须能被 Ark 公网下载（不能是内网地址）

### 配置验证命令
```bash
# 验证 LLM 配置加载
curl -s http://localhost:18081/health | grep -i llm

# 查看 provider 初始化日志
docker logs product-service | grep "provider=doubao"
```

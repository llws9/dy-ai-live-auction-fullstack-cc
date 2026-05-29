# 直播竞拍可复用仓库划分设计

## 1. 背景与目标

本设计用于回答一个工程化问题：如何把当前直播竞拍全栈平台中的后端能力沉淀为可复用、可二次开发、可被主流直播平台快速接入的代码资产。

现有设计文档 `docs/superpowers/specs/2026-05-30-live-auction-openapi-sdk-design.md` 已经定义了平台接入契约、OpenAPI 能力、Client SDK、回调可靠性、订单探测和幂等机制。本设计不重复定义 API 细节，而是基于该契约继续定义仓库边界。

核心目标：

- 将“稳定竞拍事实”和“直播平台差异”分离。
- 将“对外接入契约”和“内部服务实现”分离。
- 将“可复用基础能力”和“参考部署工程”分离。
- 支持平台快速接入，同时允许平台在 Adapter、Host Service、策略配置层做二次开发。
- 避免把当前 `gateway`、`auction`、`product`、DAO、数据库模型和基础设施配置直接变成外部依赖。

非目标：

- 不把现有后端整体打包成一个 SDK。
- 不要求直播平台替换已有用户、直播间、商品、订单、支付和履约系统。
- 不把平台差异写进竞拍核心。
- 不开放平台自定义赢家判定、出价排序等会影响公平性的核心规则，除非未来单独设计规则引擎。

## 2. 第一性原理

仓库划分不是为了“拆得多”，而是为了让每个仓库拥有清晰、稳定、可替换的责任边界。

直播竞拍平台对外复用时，本质上有四类变化：

```text
稳定不变：
  竞拍事实、出价规则、排名、延时、结果确认、可靠回调、幂等。

相对稳定：
  OpenAPI 契约、错误码、事件 Schema、SDK 方法命名。

经常变化：
  不同直播平台的用户、直播间、商品、订单、签名、限流、回调格式。

交付变化：
  平台方选择 SaaS 接入、私有化部署、混合部署、二次开发部署。
```

因此仓库边界应按变化原因划分，而不是按当前项目目录或微服务名称划分。

## 3. 推荐方案

采用“契约 + Client SDK + 竞拍核心 + 平台适配器 + 参考宿主服务”的多仓库方案。

```text
live-auction-contracts
  -> live-auction-client-sdk
  -> live-auction-core
  -> live-auction-adapters
  -> reference-host-service
```

该方案对应方案 B：

- `contracts` 是唯一对外契约来源。
- `client-sdk` 是平台后端接入工具。
- `core` 是直播竞拍最大基础能力。
- `adapters` 隔离平台差异。
- `reference-host-service` 提供可运行参考实现，但不作为所有接入方的强制依赖。

## 4. 仓库边界

### 4.1 live-auction-contracts

职责：

- 维护 OpenAPI 规范。
- 维护 JSON Schema、事件 Schema、错误码、状态机枚举。
- 维护 HMAC 签名规范、幂等规范、回调响应规范。
- 维护 Callback、Order Probe、Realtime Token 等对外契约。
- 生成多语言 SDK 所需的契约输入。

应包含：

```text
openapi/
  live-auction-openapi.yaml
schemas/
  auction-result-confirmed.schema.json
  callback-response.schema.json
  order-probe-response.schema.json
errors/
  error-codes.md
auth/
  hmac-signature.md
idempotency/
  idempotency.md
examples/
  callback-payload.json
  order-probe-found.json
```

不应包含：

- 业务服务实现。
- 数据库迁移。
- DAO、GORM Model、内部 Service。
- 某个直播平台的私有字段逻辑。

版本规则：

- 契约使用 SemVer。
- 兼容性新增字段走 minor。
- 删除字段、修改字段含义、修改签名规则走 major。
- 错误码只能新增，不能静默改变语义。

### 4.2 live-auction-client-sdk

职责：

- 为平台后端提供轻量 Client SDK。
- 封装 HTTP 调用、HMAC 签名、`X-Request-Id`、幂等键、超时、重试、错误码映射。
- 提供回调验签工具。
- 提供 Go、Java、Node 等语言实现或生成入口。

应包含：

```text
go/
java/
node/
shared/
  conformance-tests/
  fixtures/
docs/
  quick-start.md
  callback-verification.md
```

SDK 模块：

```text
LiveAuctionClient
  AuctionAPI
  BidAPI
  MappingAPI
  RealtimeAPI
  CallbackEventAPI

Middleware
  SigningMiddleware
  IdempotencyMiddleware
  RetryMiddleware
  TimeoutMiddleware
  LoggingMiddleware
```

不应包含：

- 出价排序、赢家判定、延时竞拍等业务规则。
- 关键竞拍状态缓存作为事实源。
- 平台订单创建逻辑。
- 当前后端内部数据库模型。

二次开发点：

- 自定义 HTTP Transport。
- 自定义日志与 Metrics。
- 自定义 RetryPolicy，但必须遵守 `Probe-before-Retry` 的安全边界。
- 自定义签名算法仅能在平台配置允许时启用。

### 4.3 live-auction-core

职责：

- 承载直播竞拍最大基础能力。
- 维护竞拍生命周期、出价校验、排名、延时、结果确认、快照、实时事件、可靠回调 Outbox。
- 提供平台无关的核心应用服务和扩展端口。

应包含：

```text
domain/
  auction/
  bid/
  ranking/
  result/
application/
  auction-service/
  bid-service/
  result-service/
  callback-dispatcher/
ports/
  order-callback-port
  order-probe-port
  realtime-push-port
  notification-port
  audit-port
  metrics-port
outbox/
  callback-event
  delivery-attempt
realtime/
  room-state
  event-types
```

核心规则：

- `live-auction-core` 是竞拍事实源。
- `auction.result_confirmed` 是订单创建唯一可信事件。
- 回调事件必须先落库再投递。
- Timeout、响应丢失、平台异步接收必须进入 Unknown/Probe 流程。
- 后续重试必须复用同一个 `event_id` 和 `idempotency_key`。

不应包含：

- 抖音、快手、淘宝直播等平台的具体 API 调用。
- 平台订单系统实现。
- 当前项目的完整 `gateway` 实现。
- 平台 UI 或管理台实现。
- 任意会让平台绕过竞拍公平性的扩展点。

二次开发点：

- 插拔式 `OrderCallbackPort`。
- 插拔式 `OrderProbePort`。
- 插拔式 `RealtimePushPort`。
- 平台级 `RetryPolicyStrategy`、`CallbackTimeoutStrategy`、`RateLimitStrategy`。
- 审计、通知、指标、告警 Handler。

### 4.4 live-auction-adapters

职责：

- 隔离不同直播平台的身份、直播间、商品、订单、回调、探测差异。
- 将平台特有 API 映射到 `live-auction-core` 的 Port。
- 提供主流平台和自定义平台的适配模板。

应包含：

```text
adapters/
  douyin/
  kuaishou/
  taobao-live/
  custom-template/
ports/
  user-mapping-port
  live-stream-mapping-port
  product-mapping-port
  order-callback-port
  order-probe-port
policies/
  signature-strategy
  retry-policy
  rate-limit-policy
```

不应包含：

- 竞拍状态机。
- 出价排序和赢家判定。
- 直接访问 `live-auction-core` 的数据库表。
- 改写已确认竞拍结果的逻辑。

二次开发点：

- 新增平台 Adapter。
- 替换平台签名策略。
- 调整平台回调响应解析。
- 调整订单探测字段映射。
- 接入平台内部风控或商品校验，但不能改变竞拍核心事实。

### 4.5 reference-host-service

职责：

- 提供一套可运行的参考宿主服务。
- 演示如何组合 `contracts`、`client-sdk`、`core`、`adapters`。
- 提供网关、配置、部署、监控、健康检查、管理端 API 的参考实现。

应包含：

```text
cmd/
  live-auction-host/
internal/
  api-gateway/
  platform-admin/
  config/
  observability/
deploy/
  docker-compose/
  kubernetes/
docs/
  deployment.md
  operations.md
```

不应包含：

- 不应成为平台接入的唯一部署方式。
- 不应把所有平台差异写死在 host service。
- 不应让平台必须复制当前主项目的完整目录结构。

适用场景：

- 平台希望快速本地部署验证。
- 私有化交付需要参考工程。
- 内部团队需要端到端 Demo。
- 第三方二开团队需要知道推荐组合方式。

## 5. 依赖方向

推荐依赖方向：

```text
live-auction-contracts
  <- live-auction-client-sdk
  <- reference-host-service

live-auction-core
  <- live-auction-adapters
  <- reference-host-service

live-auction-core
  -> Extension Ports
  <- Infrastructure Adapters
```

约束：

- `contracts` 不依赖任何业务实现。
- `client-sdk` 只依赖 `contracts`，不依赖 `core`。
- `core` 不依赖具体平台 Adapter。
- `adapters` 依赖 `core` 的 Port，不反向污染核心。
- `reference-host-service` 负责组装依赖，不沉淀不可替换业务规则。

禁止依赖：

```text
client-sdk -> core
contracts -> client-sdk
core -> douyin-adapter
core -> current-gateway
adapter -> core database table
platform frontend -> app_secret
```

## 6. 与当前项目的关系

当前项目可以继续作为主产品工程演进，但不能直接等同于可复用仓库。

当前目录与未来仓库的映射建议：

```text
docs/superpowers/specs/2026-05-30-live-auction-openapi-sdk-design.md
  -> live-auction-contracts 的契约来源

backend/gateway
  -> reference-host-service 的网关参考实现

backend/auction/service
  -> live-auction-core 的候选来源

backend/auction/websocket
  -> live-auction-core realtime 模块的候选来源

backend/product/order
  -> 不直接进入 core；仅抽象为平台订单 Adapter 或示例订单系统

backend/pkg
  -> 可拆为 host service 基础设施参考，不作为 core 必需依赖
```

迁移原则：

- 先定义契约，再提取代码。
- 先抽 Port，再迁移 Adapter。
- 先保证行为测试，再替换内部实现。
- 不把当前数据库表结构作为外部稳定契约。

## 7. 平台接入路径

标准接入流程：

```text
1. 平台注册 app_id、app_secret、scopes、callback_url。
2. 平台后端安装 live-auction-client-sdk。
3. 平台完成用户、直播间、商品映射。
4. 平台后端调用 OpenAPI 创建竞拍。
5. 平台前端通过平台后端换取短期 realtime token。
6. 用户出价请求经平台后端或授权入口进入竞拍服务。
7. 竞拍核心确认结果并生成 auction.result_confirmed。
8. Callback Outbox 投递结果到平台订单系统。
9. 平台用 idempotency_key 幂等建单并返回 external_order_id。
10. 如果回调超时或响应丢失，我们先按 idempotency_key 探测订单，再决定是否重试。
```

平台二次开发位置：

```text
推荐：
  Adapter
  Host Service
  Policy Strategy
  Observability
  Admin Console

谨慎：
  Client SDK RetryPolicy
  Callback Timeout
  RateLimit Policy

不推荐：
  Auction Core State Machine
  Winner Decision
  Bid Ranking
  Idempotency Semantics
```

## 8. 备选方案对比

### 8.1 单仓库产品化

形态：

```text
live-auction-platform
  gateway
  auction
  product
  db
  infra
  sdk
```

优点：

- 私有化交付简单。
- 本地端到端运行成本低。
- 短期改造少。

问题：

- 粒度过大，平台接入成本高。
- 容易暴露内部微服务、数据库和 DAO 结构。
- 平台二开容易直接改核心，后续难升级。
- 难以区分“必须复用”和“只是当前实现”。

结论：

- 可作为私有化交付包或参考部署，不应作为默认复用边界。

### 8.2 SDK 优先

形态：

```text
live-auction-sdk
  go
  java
  node
  docs
```

优点：

- 最快验证平台接入。
- 对现有后端改造少。
- 适合 SaaS 模式。

问题：

- 复用的是接入能力，不是直播竞拍基础能力本身。
- 如果后续要私有化或二开，仍然缺少 core/adapters 边界。
- SDK 容易被误用为承载业务规则的地方。

结论：

- 可作为第一阶段落地方式，但不足以回答“最大基础能力仓库化”的目标。

### 8.3 推荐方案 B

形态：

```text
contracts + client-sdk + core + adapters + reference-host-service
```

优点：

- 接入契约稳定。
- 核心能力可复用。
- 平台差异可扩展。
- 参考部署可复制但不强绑定。
- 支持 SaaS、私有化、混合部署三种交付模式。

代价：

- 初始仓库治理成本更高。
- 需要维护版本兼容矩阵。
- 需要补充跨仓库测试和契约测试。

结论：

- 最符合“快速接入 + 最大基础能力 + 可二次开发”的目标。

## 9. 版本与发布策略

版本来源：

- `live-auction-contracts` 是兼容性判断的源头。
- `client-sdk` 版本应声明兼容的 `contracts` 版本范围。
- `core` 版本应声明支持的事件 Schema 和状态机版本。
- `adapters` 版本应声明兼容的平台 API 版本和 `core` Port 版本。
- `reference-host-service` 版本锁定一组经过验证的组合。

示例：

```text
contracts v1.2.0
client-sdk-go v1.2.3 supports contracts >=1.2 <2.0
core v1.4.0 supports event schema v1
adapters-douyin v0.3.0 supports core ports v1
reference-host-service v0.5.0 locks:
  contracts v1.2.0
  core v1.4.0
  adapters-douyin v0.3.0
```

发布原则：

- 先发布 `contracts`，再发布 SDK。
- `core` 的内部重构不应破坏对外契约。
- Adapter 变更必须通过契约测试和平台沙箱测试。
- 任何影响幂等、签名、状态机、回调语义的变化都必须显式升级版本。

## 10. 测试策略

必须具备：

- `contracts` 的 OpenAPI lint、Schema validation、breaking change check。
- `client-sdk` 的签名一致性测试、错误码映射测试、回调验签测试。
- `core` 的竞拍状态机测试、出价并发测试、延时规则测试、Outbox 状态机测试。
- `adapters` 的平台响应解析测试、订单探测测试、幂等冲突测试。
- `reference-host-service` 的端到端接入测试。

跨仓库契约测试：

```text
contracts fixtures
  -> client-sdk conformance tests
  -> core callback payload tests
  -> adapters probe response tests
  -> reference-host-service e2e tests
```

关键验收：

- 平台重复回调不会重复建单。
- Timeout 不会直接重试导致重复订单。
- `FOUND` 但缺少 `external_order_id` 不会标记成功。
- SDK 不包含竞拍核心规则。
- Adapter 无法直接改写竞拍结果。

## 11. 分阶段落地

### 阶段 1：契约仓库优先

目标：

- 从当前设计文档中提取 OpenAPI、Schema、错误码、签名规范、幂等规范。
- 建立 `live-auction-contracts`。
- 形成 breaking change 检查。

产物：

- `live-auction-openapi.yaml`
- Callback Schema
- Order Probe Schema
- Error Codes
- HMAC Signature Spec
- Idempotency Spec

### 阶段 2：Client SDK

目标：

- 建立 `live-auction-client-sdk`。
- 优先实现 Go SDK。
- 验证平台后端接入体验。

产物：

- `LiveAuctionClient`
- Signing Middleware
- Retry/Timeout Middleware
- Callback Verifier
- Quick Start

### 阶段 3：Core 边界收敛

目标：

- 从当前 `backend/auction` 提取平台无关的竞拍核心。
- 定义 Core Port。
- 引入 Outbox 和回调状态机的稳定实现。

产物：

- Auction Domain
- Bid Service
- Ranking Service
- Result Service
- Callback Dispatcher
- Realtime Port

### 阶段 4：Adapters 与参考宿主服务

目标：

- 建立 `live-auction-adapters`。
- 建立 `reference-host-service`。
- 支持至少一个主流直播平台或 mock platform 端到端接入。

产物：

- Custom Platform Adapter Template
- Mock Platform Adapter
- Reference Host Service
- Docker Compose Demo
- E2E 接入样例

## 12. 风险与约束

主要风险：

- 过早拆仓导致开发效率下降。
- `core` 边界不清会让平台差异反向污染竞拍事实。
- `client-sdk` 过重会变成业务逻辑载体。
- Adapter 如果能访问核心数据库，会破坏封装和可升级性。
- 缺少契约测试会导致多仓库版本漂移。

控制措施：

- 第一阶段只拆 `contracts` 和 `client-sdk`，不要立刻大规模搬迁服务代码。
- `core` 只暴露 Port 和 Application Service，不暴露 DAO。
- Adapter 只能通过 Port 与 Core 交互。
- `reference-host-service` 只做组装和参考部署。
- 每个仓库必须有明确的“不做什么”清单。

## 13. 最终结论

推荐采用方案 B：

```text
live-auction-contracts
live-auction-client-sdk
live-auction-core
live-auction-adapters
reference-host-service
```

该方案与现有 `OpenAPI + 轻量 Client SDK` 设计兼容。现有文档回答“平台如何接入”，本设计回答“代码资产如何仓库化、复用化、二次开发友好化”。

最短落地路径不是先拆现有服务，而是：

```text
先沉淀 contracts
再交付 client-sdk
再收敛 core
最后建设 adapters 与 reference-host-service
```

这样可以先让平台快速接入，同时避免把当前主项目的内部实现误当成长期稳定的外部契约。

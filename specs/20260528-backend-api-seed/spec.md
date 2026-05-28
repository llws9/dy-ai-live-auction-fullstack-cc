# Feature Specification: 后端API补充与测试数据生成

**Feature**: `20260528-backend-api-seed`
**Created**: 2026-05-28
**Status**: Draft
**Input**: Brainstorm文档: `.adk-mobile/specs/backend-api-seed/brainstorm-output.md`

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Gateway路由补充 (Priority: P1)

补充Gateway服务缺失的路由注册，使前端能够正常调用订单发货、订单历史、直播间管理等接口。

**目标用户**: 管理员、商家、主播

**触发条件**: 
- Admin订单管理页点击"标记发货"按钮
- H5用户查看订单历史
- Admin直播间列表页加载

**业务规则**:
- 订单发货需商家或管理员权限
- 订单历史查询需用户JWT认证
- 管理端直播间列表需管理员权限

**Why this priority**: P1 - 前端页面依赖这些接口才能正常工作，是系统可用性的基础

**Technical Implementation**:

#### Gateway路由变更 (`backend/gateway/router/router.go`)

| 接口 | 方法 | 目标服务 | 权限要求 |
|------|------|----------|----------|
| `/api/v1/orders/:id/ship` | PUT | Product Proxy | 需商家/管理员权限 |
| `/api/v1/orders/history` | GET | Product Proxy | 需JWT认证 |
| `/api/v1/admin/live-streams` | GET | Product Proxy | 需管理员权限 |
| `/api/v1/live-streams/:id` | GET | Product Proxy | 无权限要求 |

#### Product Service Handler补充 (`backend/product/handler/live_stream.go`)

| 方法 | 说明 |
|------|------|
| `ListAdminLiveStreams` | 管理端直播间列表，返回所有直播间含状态筛选 |
| `GetLiveStreamDetail` | 直播间详情，含关联商品和竞拍场次 |

#### 调用链路

```
Request → Gateway JWT验证 → 权限中间件 → Product Proxy → Product Service Handler → Service → DAO → MySQL
```

**Independent Test**: 可通过Swagger测试新增接口，验证路由注册和权限控制正确

**Acceptance Scenarios**:

1. **Given** 管理员已登录, **When** 调用PUT `/orders/:id/ship`, **Then** 订单状态更新为已发货
2. **Given** 用户已登录, **When** 调用GET `/orders/history`, **Then** 返回用户订单历史列表
3. **Given** 管理员已登录, **When** 调用GET `/admin/live-streams`, **Then** 返回直播间列表
4. **Given** 未登录用户, **When** 调用GET `/live-streams/:id`, **Then** 返回直播间详情

---

### User Story 2 - 动态商品类别系统 (Priority: P2)

新增独立的商品类别表，支持类别的动态增删改，商品通过逻辑外键关联类别。

**目标用户**: 管理员

**触发条件**: 
- 管理员在后台新增商品类别
- 管理员编辑或删除现有类别
- 前端查询商品时需显示类别信息

**业务规则**:
- 类别代码(code)唯一，不可重复
- 删除类别时需校验是否有关联商品，有商品则禁止删除
- 类别状态可启用/禁用，禁用类别不显示在商品选择列表

**Why this priority**: P2 - 类别系统是商品管理的基础设施，支持业务扩展

**Technical Implementation**:

#### 新增Category模型 (`backend/product/model/category.go`)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INT64 | 主键，自增 |
| name | VARCHAR(64) | 类别名称，如"艺术收藏" |
| code | VARCHAR(32) | 类别代码，唯一索引，如"art" |
| description | TEXT | 类别描述 |
| sort_order | INT | 排序顺序，默认0 |
| status | TINYINT | 状态：1=启用, 0=禁用 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

#### Product模型变更 (`backend/product/model/product.go`)

新增字段: `CategoryID *int64` - 逻辑外键关联Category.id

> **设计决策**: 采用逻辑外键而非物理外键，避免高并发场景下的外键约束检查开销

#### 新增API接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/v1/categories` | GET | 类别列表 |
| `/api/v1/categories` | POST | 新增类别（需管理员权限） |
| `/api/v1/categories/:id` | PUT | 更新类别（需管理员权限） |
| `/api/v1/categories/:id` | DELETE | 删除类别（需管理员权限+无商品关联校验） |

#### 代码文件变更

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `model/category.go` | 新增 | Category模型定义 |
| `dao/category.go` | 新增 | CategoryDAO数据访问层 |
| `service/category.go` | 新增 | CategoryService业务逻辑 |
| `handler/category.go` | 新增 | CategoryHandler CRUD Handler |
| `model/product.go` | 修改 | 添加CategoryID字段 |
| `main.go` | 修改 | 注册Category路由 |

#### 调用链路

```
Request → Gateway → Product Service → Category Handler → Category Service → Category DAO → MySQL
                                                    ↓
                                          (删除时检查商品关联)
```

**Independent Test**: 可通过API创建、查询、更新、删除类别，验证CRUD功能和删除保护逻辑

**Acceptance Scenarios**:

1. **Given** 管理员已登录, **When** POST `/categories` 创建新类别, **Then** 类别成功创建并返回ID
2. **Given** 类别已创建, **When** GET `/categories`, **Then** 返回类别列表包含新类别
3. **Given** 类别关联了商品, **When** DELETE `/categories/:id`, **Then** 返回错误"该类别下有商品，无法删除"
4. **Given** 类别无商品关联, **When** DELETE `/categories/:id`, **Then** 类别成功删除

---

### User Story 3 - 测试数据生成脚本 (Priority: P3)

编写Go Seed脚本生成多样性测试数据，支持前端开发测试和系统演示。

**目标用户**: 开发人员

**触发条件**: 执行 `go run backend/seed/main.go`

**业务规则**:
- 数据按顺序生成：先父表后子表，确保关联正确
- 每次执行先清空现有数据，再生成新数据
- 数据分布符合真实业务场景

**Why this priority**: P3 - 测试数据是开发辅助工具，不影响核心功能

**Technical Implementation**:

#### Seed脚本结构

| 文件 | 说明 |
|------|------|
| `backend/seed/main.go` | 主程序入口，连接数据库，协调生成顺序 |
| `backend/seed/generators.go` | 各类型数据生成函数 |
| `backend/seed/config.go` | 数据配置（数量、分布比例） |

#### 数据生成顺序

```
1. GenerateCategories (6个类别)
2. GenerateUsers (53个用户：买家40、商家8、主播3、管理员2)
3. GenerateProducts (30+商品，分布各类别)
4. GenerateLiveStreams (10+直播间，关联商家)
5. GenerateAuctionRules (30+规则，关联商品)
6. GenerateAuctions (20+场次，多状态分布)
7. GenerateBids (100+出价记录)
8. GenerateOrders (15+订单，多状态分布)
9. GenerateNotifications (30+通知)
```

#### 数据多样性设计

| 类型 | 数量 | 多样性维度 |
|------|------|------------|
| 类别 | 6 | 艺术收藏、珠宝名表、数码电子、奢侈品、时尚服饰、家居生活 |
| 用户 | 53 | 角色分布；邮箱/手机号；名称多样性 |
| 商品 | 30+ | 类别均匀分布；价格段：低价20%、中价50%、高价30%；状态分布 |
| 直播间 | 10+ | 关联商家；状态：活跃80%、禁用20% |
| 竞拍规则 | 30+ | 起拍价100-10000；加价幅度10-500；时长60-600秒 |
| 竞拍场次 | 20+ | 状态：待开始5、进行中3、延时中2、已结束8、已取消2 |
| 出价 | 100+ | 多用户参与；金额递增；时间分布 |
| 订单 | 15+ | 状态：待支付3、已支付4、已发货3、已完成5 |
| 通知 | 30+ | 类型分布；已读/未读状态 |

**Independent Test**: 执行Seed脚本后检查数据库，验证数据数量和关联关系

**Acceptance Scenarios**:

1. **Given** 数据库已连接, **When** 执行Seed脚本, **Then** 所有数据成功生成
2. **Given** 数据已生成, **When** 查询商品表, **Then** 商品类别ID正确关联
3. **Given** 数据已生成, **When** 查询竞拍表, **Then** 竞拍状态分布符合配置

---

### Edge Cases

- **类别删除保护**: 尝试删除有商品的类别时，系统返回错误并阻止删除
- **重复类别代码**: 创建类别时使用已存在的code，返回唯一约束错误
- **数据关联完整性**: Seed脚本生成失败时，数据库应回滚或保持一致状态
- **并发场景**: 多用户同时出价时，出价金额和时间顺序正确记录

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Gateway必须注册 `/orders/:id/ship` 路由并转发到Product Service
- **FR-002**: Gateway必须注册 `/orders/history` 路由并转发到Product Service
- **FR-003**: Gateway必须注册 `/admin/live-streams` 路由并转发到Product Service
- **FR-004**: Gateway必须注册 `/live-streams/:id` 路由并转发到Product Service
- **FR-005**: Product Service必须实现 ListAdminLiveStreams Handler
- **FR-006**: Product Service必须实现 GetLiveStreamDetail Handler
- **FR-007**: 系统必须支持商品类别的CRUD操作
- **FR-008**: 类别删除时必须校验商品关联，有商品则禁止删除
- **FR-009**: Product模型必须添加CategoryID字段（逻辑外键）
- **FR-010**: Seed脚本必须按正确顺序生成数据确保关联完整
- **FR-011**: Seed脚本必须生成多样性的测试数据

### Key Entities

- **Category**: 商品类别实体，包含名称、代码、描述、排序、状态
- **Product**: 商品实体，新增category_id字段关联类别
- **User**: 用户实体，角色分布（买家、商家、主播、管理员）
- **LiveStream**: 直播间实体，关联商家creator_id
- **Auction**: 竞拍场次实体，状态分布（待开始、进行中、延时中、已结束、已取消）
- **Order**: 订单实体，状态分布（待支付、已支付、已发货、已完成）

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 所有新增API通过Swagger测试验证
- **SC-002**: Seed脚本生成数据关联关系100%正确
- **SC-003**: 前端Admin/H5页面正常加载真实数据
- **SC-004**: 类别管理功能完整可用（增删改查）
- **SC-005**: 测试数据覆盖所有业务场景（用户角色、商品状态、竞拍状态、订单状态）
- **SC-006**: 数据库迁移自动完成，无数据丢失

## Assumptions

- [INFERRED] 现有Order Ship Handler已在Product Service实现
- [INFERRED] 现有Order History Handler已在Product Service实现
- [INFERRED] Gateway权限中间件已支持RequireMerchant和RequireAdmin

## Dependencies

- Product Service数据库连接正常
- Gateway JWT认证中间件正常工作
- GORM AutoMigrate支持新表和字段迁移

## Risk Mitigation

| 风险 | 处理方案 |
|------|----------|
| 数据库迁移影响现有数据 | GORM AutoMigrate自动处理，Seed先清空再生成 |
| 数据关联错误 | Seed按顺序生成，先父表后子表 |
| 类别删除导致数据问题 | 应用层校验，有商品禁止删除 |

## Observability *(metrics instrumentation)*

### Gateway 指标采集

Gateway 服务通过 Prometheus 中间件自动采集 HTTP 请求指标：

- **指标端点**: `http://localhost:9090/metrics`（独立端口）
- **采集方式**: MetricsMiddleware 自动记录每个请求

### HTTP 指标

| 指标 | 类型 | 标签 | 用途 |
|------|------|------|------|
| `http_requests_total` | Counter | service, method, path, status | QPS 计算、错误率统计 |
| `http_request_duration_seconds` | Histogram | service, method, path | 响应时间分布分析 |

### QPS 计算方式

使用 PromQL 的 `rate()` 函数：

```promql
# 全局 QPS
rate(http_requests_total[1m])

# 按路径分组的 QPS
sum by (path) (rate(http_requests_total[1m]))
```

### 监控集成

详见 `docs/operations_manual.md` 的"监控告警"章节，包含：
- Prometheus scrape 配置
- Grafana Dashboard 面板配置
- 告警规则（高错误率、响应时间过长）
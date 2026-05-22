# Feature Specification: MVP阶段功能完善

**Feature**: `20260522-mvp-notification-report`
**Created**: 2026-05-22
**Status**: Draft
**Input**: 本地技术提案文档: [mvp-priority-tasks_brainstorm.md](../brainstorm/mvp-priority-tasks_brainstorm.md)

## 需求背景

直播竞拍系统已完成核心功能实现，但MVP阶段仍需完善以下功能：

1. **消息通知系统缺失**：用户无法及时收到出价提醒、中标通知、订单状态变更等关键信息
2. **数据分析能力不足**：管理后台缺少数据统计和可视化报表
3. **测试覆盖不足**：核心业务逻辑缺乏测试保护，仅8个测试文件
4. **API文档缺失**：无Swagger文档，影响前后端协作效率

**设计原则**：
- 采用并行开发策略（方案A），API文档+测试优先，通知+报表并行开发
- 订单通知链路预留接口，二期接入真实订单系统时无需修改通知核心代码
- 每次新增/修改业务功能必须同步更新API文档（Swagger注解）

---

## User Scenarios & Testing

### User Story 1 - 消息通知系统 (Priority: P1)

**描述**：作为用户，希望在出价被超越、竞拍中标/未中标、订单状态变更时收到实时通知，以便及时了解竞拍动态和订单进度。

**Why this priority**: 通知是用户体验的核心组成部分，直接影响用户参与度和留存率。

**当前限制**：
- 订单和发货功能为Mock实现，订单状态变更通知仅由Mock触发
- 二期接入真实订单系统后，通过预留接口自动接入

**Technical Implementation**:

**1. 通知渠道**
- WebSocket实时推送（复用现有连接）
- 站内信存储（数据库持久化）

**2. 通知类型**
| 类型 | 触发条件 | 接收者 | 当前状态 |
| --- | --- | --- | --- |
| 出价被超越 | 有人出更高价 | 前出价者 | ✅ 完整 |
| 竞拍中标 | 竞拍结束且中标 | 中标用户 | ✅ 完整 |
| 竞拍未中标 | 竞拍结束未中标 | 参与用户 | ✅ 完整 |
| 订单已支付 | 支付成功 | 订单所有者 | ⚠️ Mock触发 |
| 订单已发货 | 发货操作 | 订单所有者 | ⚠️ Mock触发 |
| 订单已完成 | 确认收货 | 订单所有者 | ⚠️ Mock触发 |

**3. 接口预留设计**

```go
// NotificationSender 通知发送接口
type NotificationSender interface {
    SendNotification(ctx context.Context, req *NotificationRequest) error
    SendBatchNotifications(ctx context.Context, reqs []*NotificationRequest) error
}

// OrderEventPublisher 订单事件发布接口（二期实现）
type OrderEventPublisher interface {
    PublishOrderEvent(ctx context.Context, event *OrderEvent) error
}

// EventSubscriber 事件订阅器
type EventSubscriber interface {
    Subscribe(eventType string, handler EventHandler) error
}
```

**4. 数据库变更**
| Table | Field | Type | Description |
| --- | --- | --- | --- |
| notifications | id | BIGINT | 主键 |
| notifications | user_id | BIGINT | 接收用户ID |
| notifications | type | VARCHAR(32) | 通知类型 |
| notifications | title | VARCHAR(128) | 通知标题 |
| notifications | content | TEXT | 通知内容 |
| notifications | data | JSON | 扩展数据 |
| notifications | read_at | DATETIME | 已读时间 |
| notifications | created_at | DATETIME | 创建时间 |

**5. 代码变更**
| File | Change | Description |
| --- | --- | --- |
| `auction/service/notification.go` | New | 通知服务（含接口定义） |
| `auction/dao/notification.go` | New | 通知DAO |
| `auction/handler/notification.go` | New | 通知API |
| `auction/websocket/message.go` | Modified | 新增通知消息类型 |
| `auction/service/bid.go#PlaceBid` | Modified | 出价时发送通知 |
| `auction/service/auction.go#EndAuction` | Modified | 结束时发送中标通知 |
| `product/service/order.go` | Modified | Mock订单操作调用通知接口 |
| `frontend/h5/src/components/Notification/` | New | 通知组件 |
| `frontend/h5/src/hooks/useNotification.ts` | New | 通知Hook |

**Independent Test**: 可通过出价测试验证通知发送，通过WebSocket连接测试验证实时推送。

**Acceptance Scenarios**:

1. **Given** 用户A已出价100元，**When** 用户B出价120元，**Then** 用户A收到"出价被超越"通知
2. **Given** 竞拍结束，用户A为最高出价者，**When** 系统处理竞拍结果，**Then** 用户A收到"竞拍中标"通知
3. **Given** 竞拍结束，用户B未中标，**When** 系统处理竞拍结果，**Then** 用户B收到"竞拍未中标"通知
4. **Given** 用户A订单已支付，**When** Mock发货操作，**Then** 用户A收到"订单已发货"通知
5. **Given** 用户收到通知，**When** 用户刷新页面或重连WebSocket，**Then** 未读通知仍然可查看（数据库持久化）

---

### User Story 2 - 数据分析报表 (Priority: P2)

**描述**：作为管理员，希望查看竞拍统计、收入分析、用户活跃度等数据报表，以便了解平台运营状况并做出决策。

**Why this priority**: 数据分析是运营决策的基础，但不阻塞核心竞拍流程。

**Technical Implementation**:

**1. 统计维度**
- 竞拍统计：总场次、成功率、平均出价次数
- 收入统计：总成交额、日均收入、商品类目分布
- 用户统计：活跃用户数、新用户注册、出价分布

**2. API端点**
| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/v1/statistics/overview` | 总览数据 |
| GET | `/api/v1/statistics/auctions` | 竞拍统计 |
| GET | `/api/v1/statistics/revenue` | 收入统计 |
| GET | `/api/v1/statistics/users` | 用户统计 |

**3. 代码变更**
| File | Change | Description |
| --- | --- | --- |
| `product/service/statistics.go` | New | 统计服务 |
| `product/handler/statistics.go` | New | 统计API |
| `product/dao/statistics.go` | New | 统计DAO |
| `frontend/admin/src/pages/Dashboard/` | New | 数据大屏 |
| `frontend/admin/src/pages/Statistics/` | New | 统计报表页 |
| `frontend/admin/src/components/Charts/` | New | 图表组件 |

**Independent Test**: 可通过API调用验证统计数据正确性，通过管理后台验证图表渲染。

**Acceptance Scenarios**:

1. **Given** 平台有历史竞拍数据，**When** 管理员访问数据大屏，**Then** 显示总览数据（总场次、成交额、用户数）
2. **Given** 管理员查看竞拍统计，**When** 选择时间范围，**Then** 显示该时间段内的竞拍成功率和平均出价次数
3. **Given** 管理员查看收入统计，**When** 选择商品类目，**Then** 显示该类目的收入分布

---

### User Story 3 - 测试覆盖增强 (Priority: P1)

**描述**：作为开发者，希望核心业务逻辑有充分的测试覆盖，以确保代码质量和变更安全。

**Why this priority**: 测试是质量保障的基础，直接影响系统稳定性和开发效率。

**Technical Implementation**:

**1. 单元测试目标**
| 模块 | 当前覆盖 | 目标覆盖 |
| --- | --- | --- |
| auction/service | ~20% | >80% |
| product/service | ~15% | >80% |
| auction/websocket | ~10% | >60% |

**2. E2E测试场景**
| 场景 | 描述 |
| --- | --- |
| 用户注册登录 | 完整认证流程 |
| 创建竞拍 | 主播创建并启动竞拍 |
| 出价竞拍 | 用户出价、实时排名 |
| 竞拍结束 | 自动结束、订单生成 |
| 订单流程 | 支付、发货、完成 |

**3. 代码变更**
| File | Change | Description |
| --- | --- | --- |
| `auction/service/auction_test.go` | New | 竞拍服务测试 |
| `auction/service/bid_test.go` | New | 出价服务测试 |
| `product/service/statistics_test.go` | New | 统计服务测试 |
| `e2e/auction.spec.ts` | Modified | E2E测试增强 |

**Independent Test**: 运行 `go test ./...` 验证单元测试，运行 `npx playwright test` 验证E2E测试。

**Acceptance Scenarios**:

1. **Given** 运行单元测试，**When** 执行 `go test ./auction/service/...`，**Then** 覆盖率 >80%
2. **Given** 运行E2E测试，**When** 执行完整竞拍流程，**Then** 所有断言通过

---

### User Story 4 - API文档生成 (Priority: P1)

**描述**：作为开发者，希望有自动生成的API文档，以便前后端高效协作。

**Why this priority**: API文档是协作的基础设施，应最先完成。

**Technical Implementation**:

**1. 技术方案**
- 使用 `swaggo/swag` 生成Swagger文档
- 在Handler方法上添加Swagger注解
- Gateway暴露 `/swagger/*` 路由

**2. Swagger注解示例**

```go
// @Summary 创建竞拍
// @Description 创建新的竞拍场次
// @Tags auction
// @Accept json
// @Produce json
// @Param body body CreateAuctionRequest true "竞拍信息"
// @Success 200 {object} Auction
// @Router /auctions [post]
```

**3. 代码变更**
| File | Change | Description |
| --- | --- | --- |
| `docs/swagger.json` | New | Swagger文档 |
| `docs/swagger.yaml` | New | Swagger文档 |
| `gateway/main.go` | Modified | 集成Swagger中间件 |
| `gateway/router/router.go` | Modified | 添加Swagger路由 |
| 所有Handler | Modified | 添加Swagger注解 |

**Independent Test**: 访问 `/swagger/index.html` 验证文档渲染。

**Acceptance Scenarios**:

1. **Given** Swagger已集成，**When** 访问 `/swagger/index.html`，**Then** 显示完整API文档
2. **Given** 新增API，**When** 运行 `swag init`，**Then** Swagger文档自动更新
3. **Given** API参数变更，**When** 更新Swagger注解，**Then** 文档显示最新参数定义

---

### Edge Cases

1. **WebSocket断连**：用户WebSocket连接断开时，通知仍保存到数据库，重连后可拉取未读通知
2. **大数据量统计**：统计查询可能影响性能，使用缓存和定时预计算缓解
3. **Swagger注解不一致**：CI检查注解与代码一致性，防止文档过时
4. **通知批量发送**：竞拍结束时大量用户需要通知，使用批量发送接口优化性能

---

## Requirements

### Functional Requirements

**消息通知系统**：
- **FR-001**: System MUST 在出价被超越时向前出价者发送实时通知
- **FR-002**: System MUST 在竞拍结束时向中标者发送中标通知
- **FR-003**: System MUST 在竞拍结束时向未中标参与者发送未中标通知
- **FR-004**: System MUST 在订单状态变更时向订单所有者发送通知（MVP阶段为Mock触发）
- **FR-005**: System MUST 将通知持久化到数据库，支持离线后重新拉取
- **FR-006**: System MUST 预留OrderEventPublisher接口，支持二期订单系统接入

**数据分析报表**：
- **FR-007**: System MUST 提供竞拍统计API，返回总场次、成功率、平均出价次数
- **FR-008**: System MUST 提供收入统计API，返回总成交额、日均收入、类目分布
- **FR-009**: System MUST 提供用户统计API，返回活跃用户数、新用户注册数
- **FR-010**: System MUST 在管理后台展示可视化图表

**测试覆盖增强**：
- **FR-011**: System MUST 核心业务单元测试覆盖率 >80%
- **FR-012**: System MUST 包含完整E2E测试场景（注册登录、创建竞拍、出价、结束、订单）
- **FR-013**: System MUST 所有测试可通过标准命令运行（go test、playwright test）

**API文档生成**：
- **FR-014**: System MUST 集成Swagger UI，暴露 /swagger/index.html
- **FR-015**: System MUST 所有API Handler包含Swagger注解
- **FR-016**: System MUST 支持通过 swag init 自动生成文档
- **FR-017**: System MUST 每次新增/修改业务功能时同步更新API文档

### Key Entities

- **Notification**: 通知消息实体，包含user_id、type、title、content、data、read_at等属性
- **NotificationRequest**: 通知请求实体，用于通知发送接口
- **OrderEvent**: 订单事件实体，用于二期订单系统事件发布
- **StatisticsOverview**: 统计总览数据，包含总场次、成交额、用户数
- **AuctionStatistics**: 竞拍统计数据，包含成功率、平均出价次数
- **RevenueStatistics**: 收入统计数据，包含总成交额、日均收入、类目分布

---

## Success Criteria

### Measurable Outcomes

- **SC-001**: 用户在出价被超越后1秒内收到通知
- **SC-002**: WebSocket断连后重连，用户可查看断连期间的所有通知
- **SC-003**: 管理员可在3秒内查看数据大屏完整信息
- **SC-004**: 单元测试覆盖率 auction/service >80%，product/service >80%
- **SC-005**: E2E测试覆盖5个核心场景，全部通过
- **SC-006**: Swagger UI可访问，显示所有API文档
- **SC-007**: 新增API后，运行 swag init 后文档自动更新

---

## Involved Projects

| Service (PSM) | Project Path | Change Type |
| --- | --- | --- |
| auction-service | backend/auction | Modified |
| product-service | backend/product | Modified |
| gateway-service | backend/gateway | Modified |
| frontend-h5 | frontend/h5 | Modified |
| frontend-admin | frontend/admin | Modified |

---

## Risk Points

1. **通知实时性**: WebSocket连接不稳定可能导致通知丢失。缓解：通知持久化到数据库，支持重新拉取
2. **统计性能**: 大数据量统计查询可能影响性能。缓解：添加缓存，使用定时任务预计算
3. **测试维护成本**: 测试代码维护成本高。缓解：合理设计测试用例，避免过度mock
4. **Swagger注解维护**: API变更时需同步更新注解。缓解：CI检查注解与代码一致性
5. **订单通知链路不完整**: 当前订单系统为Mock实现。缓解：预留接口，二期无缝接入

---

## Development Constraints

> **强制要求**: 每次新增/修改业务功能必须同步更新API文档（Swagger注解）

**检查清单**:
- [ ] 新增API时添加Swagger注解
- [ ] 修改API参数时更新注解
- [ ] 修改响应结构时更新注解
- [ ] PR提交前运行 `swag init` 验证文档生成

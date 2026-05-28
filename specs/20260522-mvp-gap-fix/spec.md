# Feature Specification: MVP版本不符合预期项修复

**Feature**: `20260522-mvp-gap-fix`
**Created**: 2026-05-22
**Status**: Draft
**Input**: MVP版本规格核对报告

## 需求背景

经过对MVP版本三个spec的核对,发现13个不符合预期的项(高优先级7个,中优先级5个,低优先级1个)。本spec旨在修复这些问题,确保MVP版本完整交付。

**核对发现的差距**:
- 管理后台核心功能缺失(商品编辑、竞拍详情、竞拍取消)
- 数据统计报表系统不完整(页面缺失、图表组件缺失)
- 后端测试覆盖不足(Gateway 0%、Product-service 20%)
- WebSocket消息节流未实现
- 用户中心页面缺失

---

## User Scenarios & Testing

### User Story 1 - 管理后台核心功能补齐 (Priority: P0)

**描述**: 补齐管理后台缺失的核心功能,确保CRUD操作完整,主播能够正常管理商品和竞拍。

**为什么这个优先级**: 管理后台是主播运营的核心工具,功能缺失直接影响业务运营。

**当前状态**:
- ❌ 商品编辑功能缺失
- ❌ 竞拍详情页面仅占位符
- ❌ 竞拍取消功能缺失

**Technical Implementation**:

**前端变更**:
| 变更类型 | 文件/组件 | 描述 |
| --- | --- | --- |
| 新增 | `frontend/admin/src/pages/Product/Edit.tsx` | 商品编辑页面 |
| 修改 | `frontend/admin/src/pages/Product/List.tsx` | 添加编辑按钮 |
| 实现 | `frontend/admin/src/pages/Auction/Detail.tsx` | 实现竞拍详情页(当前仅占位) |
| 新增 | `frontend/admin/src/components/AuctionInfo/index.tsx` | 竞拍信息展示组件 |
| 新增 | `frontend/admin/src/components/BidHistory/index.tsx` | 出价记录列表组件 |
| 修改 | `frontend/admin/src/pages/Auction/List.tsx` | 添加取消按钮和确认弹窗 |

**后端支持**:
- ✅ 商品更新API已实现 (`PUT /api/v1/products/:id`)
- ✅ 竞拍取消API已实现 (`PUT /api/v1/auctions/:id/cancel`)
- ✅ 竞拍详情API已实现 (`GET /api/v1/auctions/:id`)
- ⚠️ 需添加出价记录查询API (`GET /api/v1/auctions/:id/bids`)

**验收场景**:
1. **Given** 商品已创建, **When** 主播点击编辑, **Then** 进入编辑页面,可修改商品信息并保存
2. **Given** 竞拍正在进行, **When** 主播查看竞拍详情, **Then** 显示竞拍信息、实时价格、出价记录
3. **Given** 竞拍正在进行, **When** 主播点击取消, **Then** 确认后竞拍状态变为cancelled,所有参与者收到通知

---

### User Story 2 - 数据统计报表系统 (Priority: P0)

**描述**: 实现完整的数据统计报表系统,包括数据大屏、统计页面和图表组件,支持数据可视化。

**为什么这个优先级**: 数据分析是运营决策的基础,无数据报表无法评估业务状况。

**当前状态**:
- ❌ Dashboard页面不存在
- ❌ Statistics页面不存在
- ❌ 图表组件未实现
- ✅ 后端统计API已实现

**Technical Implementation**:

**前端变更**:
| 变更类型 | 文件/组件 | 描述 |
| --- | --- | --- |
| 新增 | `frontend/admin/src/pages/Dashboard/index.tsx` | 数据大屏页面 |
| 新增 | `frontend/admin/src/pages/Statistics/Index.tsx` | 统计报表主页 |
| 新增 | `frontend/admin/src/pages/Statistics/Auction.tsx` | 竞拍统计页 |
| 新增 | `frontend/admin/src/pages/Statistics/Revenue.tsx` | 收入统计页 |
| 新增 | `frontend/admin/src/pages/Statistics/User.tsx` | 用户统计页 |
| 新增 | `frontend/admin/src/components/Charts/LineChart.tsx` | 趋势图组件 |
| 新增 | `frontend/admin/src/components/Charts/BarChart.tsx` | 柱状图组件 |
| 新增 | `frontend/admin/src/components/Charts/PieChart.tsx` | 饼图组件 |
| 新增 | `frontend/admin/src/components/Charts/StatCard.tsx` | 统计卡片组件 |

**后端API (已实现)**:
- ✅ `GET /api/v1/statistics/overview` - 统计总览
- ✅ `GET /api/v1/statistics/auctions` - 竞拍统计
- ✅ `GET /api/v1/statistics/revenue` - 收入统计
- ✅ `GET /api/v1/statistics/users` - 用户统计

**技术选型**: Recharts (React图表库)

**验收场景**:
1. **Given** 管理员登录后台, **When** 访问Dashboard, **Then** 显示总览数据(总场次、成交额、用户数)
2. **Given** 管理员查看竞拍统计, **When** 选择时间范围, **Then** 显示成功率、平均出价次数的图表
3. **Given** 管理员查看收入统计, **When** 选择类目, **Then** 显示收入分布饼图和趋势折线图

---

### User Story 3 - 后端测试覆盖提升 (Priority: P1)

**描述**: 提升后端测试覆盖率,确保系统稳定性: Gateway 0%→60%, Product-service 20%→80%。

**为什么这个优先级**: 测试是质量保障的基础,覆盖率不足影响系统稳定性。

**当前状态**:
- ❌ Gateway: 0%测试覆盖
- ⚠️ Product-service: ~20%测试覆盖(仅订单测试)
- ⚠️ Auction-service: ~70%测试覆盖

**Technical Implementation**:

**Gateway测试 (目标60%)**:
| 文件 | 覆盖内容 |
| --- | --- |
| `backend/gateway/handler/auth_test.go` | 认证接口测试 |
| `backend/gateway/middleware/jwt_test.go` | JWT中间件测试 |
| `backend/gateway/middleware/ratelimit_test.go` | 限流中间件测试 |
| `backend/gateway/middleware/rbac_test.go` | 权限控制测试 |

**Product-service测试 (目标80%)**:
| 文件 | 覆盖内容 |
| --- | --- |
| `backend/product/service/product_test.go` | 商品服务测试 |
| `backend/product/service/statistics_test.go` | 统计服务测试 |
| `backend/product/handler/product_test.go` | 商品Handler测试 |

**验收场景**:
1. **Given** 运行Gateway测试, **When** `go test ./...`, **Then** 覆盖率 > 60%
2. **Given** 运行Product-service测试, **When** `go test ./...`, **Then** 覆盖率 > 80%
3. **Given** 运行所有测试, **When** `go test ./...`, **Then** 全部通过,无失败用例

---

### User Story 4 - WebSocket消息节流 (Priority: P1)

**描述**: 实现WebSocket消息节流机制,防止消息洪泛,提升前端性能。

**为什么这个优先级**: 高并发场景下消息洪泛会导致前端卡顿,影响用户体验。

**当前状态**:
- ❌ 前端未实现消息节流
- ✅ 后端已实现排名推送节流(200ms)

**Technical Implementation**:

**前端变更**:
| 变更类型 | 文件/方法 | 描述 |
| --- | --- | --- |
| 修改 | `frontend/h5/src/services/websocket.ts` | 添加消息队列和节流逻辑 |
| 新增 | `frontend/h5/src/utils/throttle.ts` | 通用节流工具函数 |

**实现策略**:
- 排名更新节流: 200ms内只处理最新一条
- 价格更新节流: 100ms合并窗口
- 使用requestAnimationFrame优化渲染

**验收场景**:
1. **Given** 用户正在竞拍页面, **When** 1秒内收到100条排名更新, **Then** 只处理最新5条(每200ms一条)
2. **Given** 用户快速出价, **When** 连续发送出价请求, **Then** 防抖机制生效,500ms内只发送一次

---

### User Story 5 - 用户中心页面 (Priority: P2)

**描述**: 实现独立的用户中心页面,展示用户个人信息、参与竞拍统计、中标记录等。

**为什么这个优先级**: 用户中心提升用户体验,但不阻塞核心竞拍流程。

**当前状态**:
- ❌ 无独立用户中心页面
- ⚠️ 部分功能分散在History页面

**Technical Implementation**:

**前端变更**:
| 变更类型 | 文件/组件 | 描述 |
| --- | --- | --- |
| 新增 | `frontend/h5/src/pages/User/Index.tsx` | 用户中心主页 |
| 新增 | `frontend/h5/src/components/UserInfo/index.tsx` | 用户信息组件 |
| 新增 | `frontend/h5/src/components/UserStats/index.tsx` | 用户统计组件 |

**展示内容**:
- 用户基本信息(头像、昵称)
- 竞拍统计(参与数、中标数、成功率)
- 最近竞拍记录
- 账户余额(预留)

**验收场景**:
1. **Given** 用户登录H5端, **When** 进入用户中心, **Then** 显示用户信息和竞拍统计
2. **Given** 用户查看最近竞拍, **When** 点击记录, **Then** 跳转到竞拍详情页

---

## Requirements

### Functional Requirements

**管理后台核心功能**:
- **FR-001**: System MUST 提供商品编辑功能,支持修改商品名称、描述、图片
- **FR-002**: System MUST 提供竞拍详情页,展示竞拍信息、实时价格、出价记录
- **FR-003**: System MUST 提供竞拍取消功能,主播可取消异常竞拍
- **FR-004**: System MUST 提供出价记录查询API

**数据统计报表**:
- **FR-005**: System MUST 提供数据大屏页面,展示总览数据
- **FR-006**: System MUST 提供统计报表页面,支持竞拍、收入、用户统计
- **FR-007**: System MUST 提供图表组件,支持折线图、柱状图、饼图
- **FR-008**: System MUST 使用Recharts作为图表库

**测试覆盖**:
- **FR-009**: System MUST Gateway测试覆盖率 > 60%
- **FR-010**: System MUST Product-service测试覆盖率 > 80%
- **FR-011**: System MUST 所有测试用例通过

**WebSocket消息节流**:
- **FR-012**: System MUST 实现排名更新节流(200ms)
- **FR-013**: System MUST 实现价格更新节流(100ms)
- **FR-014**: System MUST 使用requestAnimationFrame优化渲染

**用户中心**:
- **FR-015**: System MUST 提供用户中心页面
- **FR-016**: System MUST 展示用户信息和竞拍统计

---

## Success Criteria

### Measurable Outcomes

**功能完整性**:
- **SC-001**: 管理后台CRUD操作完整(商品、竞拍、订单)
- **SC-002**: 数据统计报表系统可用
- **SC-003**: 图表组件正常渲染

**质量指标**:
- **SC-004**: Gateway测试覆盖率 > 60%
- **SC-005**: Product-service测试覆盖率 > 80%
- **SC-006**: 所有测试用例通过

**性能指标**:
- **SC-007**: WebSocket消息处理不卡顿(>30fps)
- **SC-008**: 页面加载时间 < 2秒

---

## Technical Architecture

### 涉及项目

| 服务 (PSM) | 项目路径 | 变更类型 |
| --- | --- | --- |
| auction-service | backend/auction | Modified |
| product-service | backend/product | Modified |
| gateway-service | backend/gateway | Modified |
| frontend-h5 | frontend/h5 | Modified |
| frontend-admin | frontend/admin | Modified |

---

## Risk Points

1. **前端图表库学习成本**: Recharts需要学习时间
   - **缓解**: 提供示例代码,使用官方文档

2. **测试覆盖提升工作量大**: 需要编写大量测试代码
   - **缓解**: 优先覆盖核心流程,逐步提升

3. **前后端联调**: 新功能需要联调
   - **缓解**: 使用Swagger文档,提前Mock数据

---

## Development Priority

### P0 - 核心功能补齐 (第一周)

1. 商品编辑功能
2. 竞拍详情页面
3. 竞拍取消功能
4. 数据大屏页面
5. 统计报表页面
6. 图表组件

### P1 - 质量保障 (第二周)

1. Gateway测试
2. Product-service测试
3. WebSocket消息节流

### P2 - 体验优化 (第三周)

1. 用户中心页面
2. 性能优化

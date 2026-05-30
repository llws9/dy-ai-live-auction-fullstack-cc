# 竞拍系统展示测试平台 - 实施计划

> Feature: test-platform
> 基于: specs/2026-05-30-test-platform/spec.md, research.md
> 生成时间: 2026-05-30

> **状态**: 目标态/部分实现对照文档；M1 已部分落地，Docker Compose 与后续场景仍待实现
> **Gateway 本地端口 SSOT**: `8080`

## 一、技术上下文

### 1.1 项目范围

| 项目 | 仓库路径 | 技术栈 | 状态 |
|------|---------|--------|------|
| test-service | `backend/test` | Go + Hertz + GORM + gorilla/websocket | 新增 |
| test-dashboard | `frontend/test-dashboard` | React + TypeScript + Zustand + Axios | 新增 |

### 1.2 技术栈决策

**后端技术栈**（已确认）:
- Go 1.21+ - 编程语言
- Hertz - Web框架（与 auction-service 保持一致）
- GORM 1.25+ - ORM框架（与现有架构一致）
- MySQL 8.0+ - 数据库（复用现有 MySQL 实例）
- gorilla/websocket 1.5+ - WebSocket库（成熟稳定）
- Prometheus Client - 指标收集（复用现有监控）

**前端技术栈**（已确认）:
- React 18.x - UI框架
- TypeScript 5.x - 类型系统
- Zustand 4.x - 状态管理
- Axios 1.x - HTTP客户端
- WebSocket API (Native) - 实时通信
- React Router 6.x - 路由管理
- Vite 5.x - 构建工具

### 1.3 依赖的项目

- **auction-service**: 竞拍主服务（被测目标）
- **Grafana**: 监控大盘（复用现有）
- **gateway**: API网关（可选，用于认证）

### 1.4 待明确事项

✅ 所有技术不确定点已通过 research.md 解决

---

## 二、宪法检查

### 2.1 核心原则一致性检查

**✅ I. 全栈一体化 (Full-Stack Integration)**
- API 变更将同步更新前后端代码
- 前后端类型定义共享（使用 TypeScript 生成 Go 类型）
- 测试平台涉及前后端联动，已同步设计

**✅ II. 实时性优先 (Real-Time Priority)**
- WebSocket 实时进度推送采用高效实现
- 关键测试操作有超时和重试机制
- 状态同步保证最终一致性

**✅ III. 质量保障 (Quality Assurance)**
- 所有代码变更将通过 CI 检查
- 关键测试逻辑有单元测试覆盖
- 发布前通过 Code Review

**✅ IV. 可扩展性 (Scalability)**
- 测试服务遵循现有架构规范
- 测试参数配置化（支持动态调整）
- 复用现有监控和基础设施

### 2.2 固定规则检查

**✅ Commit Workflow**: 将使用 `/adk:commit` 提交代码
**✅ Code Consistency**: 优先复用现有代码模式（参考 auction-service）
**✅ Real-Time Changes**: WebSocket 变已有延迟影响评估和回滚策略
**✅ API First**: API 定义已先于实现完成（见 spec.md）

### 2.3 代码约定检查

**✅ Naming Conventions**:
- Go: CamelCase (导出) / camelCase (私有)
- TypeScript: camelCase
- Database: snake_case
- API: kebab-case

**✅ Error Handling**:
- 错误包含错误码和描述
- 错误记录足够上下文
- 用户友好的错误信息
- API 统一响应格式

**✅ Logging Standards**:
- 结构化日志（时间戳、级别、追踪ID）
- 不记录敏感信息
- 关键业务操作记录

### 2.4 门槛评估

**✅ 无宪法违规项**
**✅ 无未解决的待明确事项**
**✅ 技术决策符合项目规范**

---

## 三、分阶段实施计划

### Phase 0: 环境准备 (P0 - 必需)

**目标**: 创建项目基础结构，配置开发环境

**任务清单**:
1. 创建 `backend/test` 项目结构
   - 参考 `backend/auction/main.go` 的目录结构
   - 创建 handler/service/dao/ws/ 子目录
   - 配置 go.mod 和依赖

2. 创建 `frontend/test-dashboard` 项目结构
   - 使用 Vite + React + TypeScript 初始化
   - 安装 Zustand、Axios、React Router
   - 配置 TypeScript 和 ESLint

3. 数据库准备
   - 创建 `test_results` 表
   - 配置数据库连接
   - 编写数据迁移脚本

4. Docker Compose 配置
   - 添加 test-service 和 test-mysql 服务
   - 配置网络和依赖关系
   - 配置环境变量

**交付物**:
- ✅ 项目目录结构
- ✅ 基础配置文件（go.mod, package.json, tsconfig.json）
- ✅ Docker Compose 配置
- ✅ 数据库 schema

**预计耗时**: 1-2天

---

### Phase 1: 后端核心功能实现 (P0 - 必需)

**目标**: 实现测试服务核心逻辑和 WebSocket 推送

#### 1.1 API Handler 实现

**任务清单**:
1. 实现 `handler/test.go`
   - StartPressureTest - 启动压力测试
   - StartConcurrentTest - 启动并发测试
   - StartWebSocketTest - 启动WebSocket测试
   - StartSkyLampTest - 启动SkyLamp测试
   - GetTestStatus - 查询测试状态
   - GetTestHistory - 查询历史记录
   - GetTestReport - 获取测试报告

2. 参考 `backend/auction/handler/` 的实现模式
   - 使用统一的请求/响应格式
   - 错误处理遵循项目约定
   - 日志记录遵循项目标准

#### 1.2 测试服务层实现

**任务清单**:
1. 实现 `service/pressure.go`
   - RunPressureTest - 压力测试核心逻辑
   - simulateBid - 模拟出价请求
   - collectMetrics - 收集性能指标

2. 实现 `service/concurrent.go`
   - RunConcurrentTest - 并发测试核心逻辑
   - testRaceCondition - 测试竞态条件
   - verifyDataConsistency - 验证数据一致性

3. 实现 `service/websocket.go`
   - RunWebSocketTest - WebSocket测试核心逻辑
   - testConnectionStability - 测试连接稳定性
   - testMessageLatency - 测试消息延迟

4. 实现 `service/skylamp.go`
   - RunSkyLampTest - SkyLamp测试核心逻辑
   - testAutoBidTrigger - 测试自动跟价触发
   - testSubscriptionManagement - 测试订阅管理

5. 参考 `backend/auction/service/sky_lamp.go` 的业务逻辑
   - 复用订阅管理机制
   - 复用自动跟价触发逻辑

#### 1.3 WebSocket 实现

**任务清单**:
1. 实现 `ws/progress.go`
   - ProgressHandler - WebSocket处理器
   - HandleConnection - 连接管理
   - BroadcastProgress - 进度广播
   - SendToClient - 单客户端推送

2. 参考 `backend/gateway/handler/proxy.go` 的 WebSocket 模式
   - 使用 gorilla/websocket
   - 连接池管理
   - 心跳检测机制

#### 1.4 DAO 实现

**任务清单**:
1. 实现 `dao/result.go`
   - SaveResult - 保存测试结果
   - GetResultByID - 查询测试结果
   - GetHistory - 查询历史记录
   - UpdateStatus - 更新测试状态

2. 参考 `backend/auction/dao/` 的 GORM 使用模式
   - 使用 GORM 标准方法
   - 错误处理规范
   - 事务管理

**交付物**:
- ✅ 7个 API Handler
- ✅ 4个测试服务实现
- ✅ WebSocket 进度推送
- ✅ DAO 数据访问层
- ✅ 单元测试覆盖

**预计耗时**: 3-5天

---

### Phase 2: 前端界面实现 (P1 - 重要)

**目标**: 实现测试平台前端界面和交互

#### 2.1 页面开发

**任务清单**:
1. 实现 TestDashboard (`/test`)
   - TestButtonPanel - 4个测试按钮
   - TestResultDisplay - 实时结果展示
   - TestProgressMonitor - WebSocket进度监控
   - GrafanaLink - Grafana跳转链接

2. 实现 TestHistory (`/test/history`)
   - 历史记录列表
   - 筛选器（测试类型、时间、状态）
   - 分页组件

3. 实现 TestReport (`/test/report/:id`)
   - 测试报告详情展示
   - 性能图表（使用 Semi Charts）
   - 错误详情列表

#### 2.2 组件开发

**任务清单**:
1. TestButtonPanel 组件
   - 4个测试按钮（压力、并发、WebSocket、SkyLamp）
   - 参数配置面板
   - 启动/停止控制

2. TestProgressMonitor 组件
   - 进度条显示
   - 实时指标更新
   - WebSocket 连接管理

3. TestResultDisplay 组件
   - 测试结果表格
   - 性能指标展示
   - 成功/失败状态

4. GrafanaLink 组件
   - Grafana 大盘链接
   - 链接参数配置

#### 2.3 状态管理

**任务清单**:
1. 实现 testStore (Zustand)
   - currentTest - 当前测试状态
   - testProgress - 测试进度
   - testMetrics - 性能指标
   - testHistory - 历史记录
   - API 调用方法

2. 实现 websocketStore (Zustand)
   - ws - WebSocket 连接
   - connected - 连接状态
   - messages - 消息列表
   - 连接/断开方法

3. 参考 `frontend/h5/src/store/` 的状态管理模式

#### 2.4 API 服务

**任务清单**:
1. 实现 API 服务层
   - 使用 Axios 封装
   - 统一请求/响应处理
   - 错误处理

2. 实现 WebSocket Hook
   - useTestProgress - 测试进度 Hook
   - 自动重连机制
   - 心跳检测

**交付物**:
- ✅ 3个页面实现
- ✅ 4个组件实现
- ✅ 2个状态管理 Store
- ✅ API 服务层
- ✅ WebSocket Hook

**预计耗时**: 2-3天

---

### Phase 3: 集成测试 (P1 - 重要)

**目标**: 验证前后端集成和测试场景正确性

#### 3.1 前后端集成验证

**任务清单**:
1. API 集成测试
   - 测试所有 API 接口连通性
   - 验证请求/响应格式
   - 错误处理测试

2. WebSocket 集成测试
   - 连接建立和断开
   - 消息推送和接收
   - 断线重连测试

3. 数据流测试
   - 测试启动 → 进度推送 → 结果保存
   - 历史记录查询
   - 测试报告生成

#### 3.2 测试场景验证

**任务清单**:
1. 压力测试场景验证
   - 100并发测试
   - 指标收集准确性
   - 性能表现

2. 并发安全测试验证
   - 竞态条件测试
   - 数据一致性验证
   - 问题检测准确性

3. WebSocket 性能测试验证
   - 100连接测试
   - 消息延迟测试
   - 连接稳定性

4. SkyLamp 功能测试验证
   - 自动跟价触发验证
   - 订阅管理测试
   - 上限停止验证

**交付物**:
- ✅ 集成测试报告
- ✅ 测试场景验证报告
- ✅ Bug修复清单

**预计耗时**: 1-2天

---

### Phase 4: 部署和监控集成 (P2 - 可选)

**目标**: 部署到测试环境，集成 Grafana 监控

#### 4.1 环境部署

**任务清单**:
1. 配置测试环境
   - 部署 test-service
   - 部署 test-dashboard
   - 配置数据库

2. Nginx 反向代理配置
   - 配置 test-service 路径
   - 配置 WebSocket 转发
   - 配置静态文件服务

3. 权限控制配置
   - 添加 JWT 认证
   - 管理员角色检查
   - 访问限制

#### 4.2 Grafana 集成

**任务清单**:
1. Prometheus 指标注册
   - 注册测试指标
   - 配置指标采集

2. Grafana Dashboard 配置
   - 创建测试平台面板
   - 配置可视化图表
   - 链接集成

**交付物**:
- ✅ 测试环境部署
- ✅ Nginx 配置
- ✅ Grafana Dashboard
- ✅ 权限控制配置

**预计耗时**: 1天

---

## 四、关键里程碑

| 里程碑 | 预计完成时间 | 交付物 | 验收标准 |
|--------|-------------|--------|---------|
| M1: 环境准备完成 | Day 2 | 项目结构、配置文件 | ✅ 项目可编译运行 |
| M2: 后端核心完成 | Day 7 | API、测试服务、WebSocket | ✅ 所有 API 可调用 |
| M3: 前端界面完成 | Day 10 | 页面、组件、状态管理 | ✅ 界面可交互 |
| M4: 集成测试完成 | Day 12 | 测试报告、Bug修复 | ✅ 四种测试场景正常 |
| M5: 部署上线 | Day 13 | 部署环境、Grafana | ✅ 测试平台可用 |

---

## 五、风险和缓解措施

### 5.1 技术风险

**风险**: WebSocket 高并发连接可能不稳定

**缓解措施**:
- 使用连接池管理
- 心跳检测机制
- 断线自动重连
- 连接数限制（1000）

### 5.2 性能风险

**风险**: 压力测试可能影响被测服务

**缓解措施**:
- 独立测试环境
- Worker Pool 控制并发
- Rate Limiter 限制请求频率
- 监控被测服务状态

### 5.3 数据风险

**风险**: 测试数据占用大量存储

**缓解措施**:
- 定期清理（7天）
- 数据归档机制
- 存储监控
- 数据库容量规划

### 5.4 权限风险

**风险**: 测试平台可能被滥用

**缓解措施**:
- JWT 认证
- 管理员角色限制
- 访问日志记录
- 异常访问告警

---

## 六、资源需求

### 6.1 人力资源

- 后端开发: 1人 (5-7天)
- 前端开发: 1人 (2-3天)
- 测试工程师: 0.5人 (1-2天)

### 6.2 环境资源

- 测试环境服务器: 1台
- MySQL 数据库: 1个实例（可复用）
- Grafana: 1个实例（已存在）

### 6.3 外部依赖

- auction-service: 竞拍主服务（已存在）
- gateway: API网关（可选，已存在）
- Grafana: 监控大盘（已存在）

---

## 七、验收标准

### 7.1 功能验收

- ✅ 四种测试场景均可正常运行
- ✅ WebSocket 实时进度推送正常
- ✅ 历史记录查询功能完整
- ✅ 测试报告展示准确
- ✅ Grafana 链接可正常跳转

### 7.2 性能验收

- ✅ 压力测试支持 1000 并发
- ✅ WebSocket 支持 1000 连接
- ✅ 页面响应时间 < 500ms
- ✅ 测试结果准确可靠

### 7.3 稳定性验收

- ✅ 测试运行期间无服务崩溃
- ✅ WebSocket 连接稳定不掉线
- ✅ 数据一致性验证通过
- ✅ 错误处理正常

### 7.4 代码质量验收

- ✅ 所有代码通过 CI 检查
- ✅ 关键逻辑有单元测试
- ✅ 代码符合项目约定
- ✅ 通过 Code Review

---

## 八、下一步行动

### 8.1 确认计划

请验证本实施计划的正确性和可行性：

1. **技术栈选择**是否合理？
2. **里程碑时间**是否可行？
3. **资源需求**是否充足？
4. **风险缓解**是否有效？

### 8.2 开始实施

确认计划后，执行下一步：

1. **创建任务清单**: 使用 `/adk:sdd:tasks` 生成详细任务分解
2. **开始开发**: 使用 `/adk:sdd:implement` 开始逐步实施
3. **Code Review**: 使用 `/adk:sdd:codereview` 进行代码审查
4. **提交代码**: 使用 `/adk:commit` 提交变更

---

## 附录

### A. 参考资料

- Spec: `.superpowers/specs/2026-05-30-test-platform-spec.md`
- Research: `specs/2026-05-30-test-platform/research.md`
- Constitution: `docs/CONSTITUTION.md`
- Coding Standards: `docs/CODING.md`

### B. 相关代码参考

1. `backend/auction/main.go` - 服务启动模式
2. `backend/auction/handler/bid.go` - API Handler 模式
3. `backend/auction/service/sky_lamp.go` - SkyLamp 业务逻辑
4. `backend/gateway/handler/proxy.go` - WebSocket 实现
5. `frontend/h5/src/store/` - 状态管理模式
6. `frontend/h5/src/pages/Live/index.tsx` - Semi 组件使用

### C. 技术文档

- Hertz: https://www.cloudwego.io/docs/hertz/
- gorilla/websocket: https://pkg.go.dev/github.com/gorilla/websocket
- UI 说明: 当前 `frontend/test-dashboard` 未引入 Semi Design；如需组件库需另行评估
- Zustand: https://zustand-demo.pmnd.rs/
- GORM: https://gorm.io/docs/

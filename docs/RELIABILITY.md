<!--
Sync Impact Report:
Version: 1.0.0 (Initial)
Modified practices: N/A (new document)
Added sections: Core Practices (4), Fixed Rules, Governance
Removed sections: N/A
Cross-validation results: ✅ Consistent with CONSTITUTION.md "实时性优先" principle
Templates requiring updates: ✅ No updates needed (initial creation)
Follow-up TODOs: None
-->

# dy-ai-live-auction-fullstack-cc Reliability

## Core Practices

### I. Fault Tolerance

设计变更时确保故障被隔离，恢复路径清晰。实时拍卖系统对可用性要求极高，任何单点故障都可能导致交易损失。

**实践要求：**
- 关键服务必须有降级方案
- 外部依赖必须有超时和熔断机制
- 异常状态必须有明确的恢复流程

### II. Degradation Strategy

优先采用优雅降级和便于回滚的变更，避免全有或全无的行为。拍卖进行中的服务中断必须有状态恢复机制。

**实践要求：**
- 部署变更必须支持灰度发布
- 数据库迁移必须有回滚脚本
- 功能开关优先于硬编码切换

### III. Monitoring Patterns

关键工作流应暴露有助于检测故障和确认健康运行的信号。实时拍卖、出价、结算等核心流程必须有完整的可观测性。

**实践要求：**
- 关键指标必须有监控告警
- 日志必须包含追踪 ID
- 异常必须有明确的错误码和上下文

### IV. Graceful Startup and Shutdown

初始化和关闭路径应避免使系统处于部分或不一致状态。服务启动必须完成依赖检查，关闭必须处理进行中的请求。

**实践要求：**
- 服务启动必须有健康检查
- 关闭信号必须优雅处理进行中的连接
- WebSocket 连接必须有明确的断开重连机制

## Fixed Rules

- **Availability Target**: 可靠性决策应支持 99.9% 的服务可用性目标。
- **Rollback Readiness**: 部署必须保持实用的回滚路径。
- **Incident Response**: 可靠性事件必须被检测、分类、缓解和复盘。
- **Circuit Breaker**: 外部服务调用必须有熔断保护。
- **Data Consistency**: 拍卖状态变更必须保证最终一致性。

## Governance

- 可靠性指导应在设计、实现、部署和事件跟进期间考虑。
- 降低可观测性或回滚信心的变更需要明确审查。
- 事件后经验应反馈到工具、自动化或文档中。

**Version**: 1.0.0 | **Ratified**: 2026-05-21 | **Last Amended**: 2026-05-21

<!--
Sync Impact Report:
Version: 1.0.0 (Initial)
Modified principles: N/A (new document)
Added sections: Core Principles (4), Fixed Rules, Governance
Removed sections: N/A
Cross-validation results: ✅ Consistent with QUALITY.md, RELIABILITY.md, SECURITY.md, CODING.md
Templates requiring updates: ✅ No updates needed (initial creation)
Follow-up TODOs: None
-->

# dy-ai-live-auction-fullstack-cc Constitution

## Core Principles

### I. 全栈一体化 (Full-Stack Integration)

前后端代码统一管理于同一仓库，确保接口契约、数据模型和业务逻辑的一致性。任何涉及前后端联动的功能必须同步设计与实现，避免接口不匹配和集成问题。

**核心规则：**
- API 变更必须同步更新前后端代码
- 共享类型定义优先使用共享模块
- 跨端功能需求必须包含前后端实现方案

### II. 实时性优先 (Real-Time Priority)

AI 直播拍卖系统对延迟极其敏感，所有实时交互功能（拍卖出价、弹幕、状态同步）必须优先保证低延迟和高可用。

**核心规则：**
- 实时通道路径不得引入不必要的中间层
- 关键实时操作必须有超时和重试机制
- 状态同步必须保证最终一致性

### III. 质量保障 (Quality Assurance)

代码质量、测试覆盖和 CI/CD 是项目可持续发展的基础。所有变更必须通过质量门禁后方可合并。

**核心规则：**
- 所有代码变更必须通过 CI 检查
- 关键业务逻辑必须有单元测试覆盖
- 发布前必须通过 Code Review

### IV. 可扩展性 (Scalability)

系统采用模块化设计，支持快速迭代和功能扩展。新功能应尽量复用现有基础设施，避免重复建设。

**核心规则：**
- 新模块必须遵循现有架构规范
- 配置化优于硬编码
- 服务发现和配置中心优先使用统一方案

## Fixed Rules

- **Commit Workflow**: 当用户通过自然语言请求提交代码时，执行 `/adk:commit`。
- **Code Consistency**: 优先复用现有代码模式和既定约定，再引入新结构。
- **Knowledge Lookup**: 内部文档优先使用 `tiksearch`，飞书文档使用 `lark_docs`。
- **Real-Time Changes**: 涉及实时通信的变更必须评估延迟影响和回滚策略。
- **API First**: 接口定义变更必须先于实现，确保前后端对齐。

## Governance

- 本宪法指导仓库内的实现、审查和文档决策。
- 变更应审查是否与核心原则保持一致。
- 重大偏离或修正应记录并附理由。

**Version**: 1.0.0 | **Ratified**: 2026-05-21 | **Last Amended**: 2026-05-21

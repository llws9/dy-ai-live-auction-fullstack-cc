<!--
Sync Impact Report:
Version: 1.0.0 (Initial)
Modified conventions: N/A (new document)
Added sections: Core Conventions (3), Fixed Rules, Governance
Removed sections: N/A
Cross-validation results: ✅ Consistent with CONSTITUTION.md "全栈一体化" principle
Templates requiring updates: ✅ No updates needed (initial creation)
Follow-up TODOs: None
-->

# dy-ai-live-auction-fullstack-cc Coding Standards

## Core Conventions

### I. Naming Conventions

使用清晰、一致的命名，与周围语言和仓库约定相匹配。全栈项目需注意前后端命名风格的一致性。

**约定要求：**
- Go 后端使用 CamelCase（导出）和 camelCase（私有）
- TypeScript 前端使用 camelCase
- 数据库字段使用 snake_case
- API 路径使用 kebab-case

### II. Error Handling

以保留可调试性和保持故障路径明确的方式处理错误。拍卖系统的错误处理必须包含足够的上下文用于问题定位。

**约定要求：**
- 错误必须包含错误码和描述信息
- 错误必须记录足够的上下文
- 用户面向的错误信息必须友好
- API 错误必须遵循统一的响应格式

### III. Logging Standards

使用结构化、有目的的日志记录，帮助运维人员和开发人员理解系统行为。实时系统的日志必须支持问题追踪和性能分析。

**约定要求：**
- 日志必须包含时间戳、级别、追踪 ID
- 结构化日志优先于文本日志
- 敏感信息不得记录到日志
- 关键业务操作必须记录

## Fixed Rules

- **Style Guides**: 遵循每个服务已使用的语言特定风格指南。
- **Formatting Tools**: 一致使用 Linter 和格式化工具。
- **Atomic Commits**: 提交应聚焦并限定为单个一致的变更。
- **Shared Types**: 前后端共享的类型定义应放在共享模块中。
- **API Documentation**: API 变更必须更新对应的文档。

## Governance

- 在引入新抽象之前，优先使用仓库中已有的模式。
- 代码审查应检查正确性、样式和变更的充分验证。
- 分支和提交实践应保持 `main` 可部署，变更易于审查。

**Version**: 1.0.0 | **Ratified**: 2026-05-21 | **Last Amended**: 2026-05-21

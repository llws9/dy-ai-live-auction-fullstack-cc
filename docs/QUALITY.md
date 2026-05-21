<!--
Sync Impact Report:
Version: 1.0.0 (Initial)
Modified standards: N/A (new document)
Added sections: Core Standards (2), Fixed Rules, Governance
Removed sections: N/A
Cross-validation results: ✅ Consistent with CONSTITUTION.md principles
Templates requiring updates: ✅ No updates needed (initial creation)
Follow-up TODOs: None
-->

# dy-ai-live-auction-fullstack-cc Quality

## Core Standards

### I. Quality Gates

所有变更必须通过构建、审查和验证门禁后方可合并。质量门禁包括但不限于：代码检查、单元测试、集成测试、安全扫描。

**标准要求：**
- 代码必须通过 Linter 检查
- 单元测试覆盖率不得低于项目基线
- 安全扫描无高危漏洞

### II. Testing Strategy

测试应覆盖变更的预期行为，并保护关键工作流免受回归影响。测试策略分层：单元测试、集成测试、端到端测试。

**标准要求：**
- 业务逻辑必须有对应的单元测试
- API 接口变更必须有集成测试
- 关键用户流程必须有端到端测试

## Fixed Rules

- **CI Required**: 所有服务必须通过 CI 后方可合并。
- **Code Review Required**: 每个变更在合并前必须经过审查。
- **Coverage Guardrail**: 测试覆盖率不得因 PR 而降低，除非有明确理由。
- **Real-Time Testing**: 涉及实时通信的功能必须有并发测试和压力测试。
- **Regression Prevention**: 修复 Bug 时必须添加回归测试用例。

## Governance

- 质量标准适用于代码、文档和生成的项目资产。
- 当质量门禁失败时，应修复根本问题而非绕过检查。
- 团队应定期审查质量目标，确保其与仓库需求保持一致。

**Version**: 1.0.0 | **Ratified**: 2026-05-21 | **Last Amended**: 2026-05-21

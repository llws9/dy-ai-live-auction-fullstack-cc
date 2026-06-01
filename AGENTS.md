# Agent Rules

本文件是仓库级 agent 规则。主 agent 与所有被派发的 subagent 在处理本仓库任务时都必须遵守。

## Mandatory First Line

每一个主 agent 或 subagent 在完成任务后的最终回答，第一句必须展示当前分支和 worktree。

固定格式：

```text
当前分支/worktree：<branch> @ <absolute-worktree-path>
```

示例：

```text
当前分支/worktree：feat/align-admin-api @ /Users/bytedance/.config/superpowers/worktrees/dy-ai-live-auction-fullstack-cc/feat-align-admin-api
```

如果无法读取 Git 分支，必须明确说明：

```text
当前分支/worktree：unknown @ <absolute-worktree-path>
```

## SDD Execution Rules

- 多任务开发优先使用 `docs/superpowers/sdd/RUNBOOK.md` 作为执行协议。
- 每次 SDD 执行必须维护一个状态文件，推荐从 `docs/superpowers/sdd/state-template.md` 复制生成。
- 状态文件是任务执行 SSOT，聊天上下文不能作为唯一状态来源。
- 每个子任务必须记录范围、依赖、测试证据、验证命令、风险和交付结论。
- 子任务完成后必须先更新状态文件，再汇报完成情况。

## Engineering Constraints

- 前端流量必须经 `gateway-service` 的 `/api/v1` 入口，不允许前端直连后端子服务。
- 跨服务访问必须走 RPC/API，不允许跨服务直接查库。
- 身份使用 JWT 派生的 `X-User-ID`，不得硬编码用户身份或内部 Token。
- 金额字段必须使用 `shopspring/decimal`，不得使用 float 表达业务金额。
- 接口契约变更必须同步更新前端、后端与文档。
- 默认遵循 TDD：先写失败测试，再最小实现，再验证通过。

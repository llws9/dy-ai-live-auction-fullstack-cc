# SDD Run State Template

> 使用方式：复制本文件到 `docs/superpowers/sdd/runs/YYYY-MM-DD-<topic>-state.md`，再把尖括号占位替换为本次执行的真实信息。

## Run Metadata

| Key | Value |
| --- | --- |
| Run ID | `<YYYY-MM-DD-topic>` |
| Topic | `<topic>` |
| Goal | `<one-sentence-goal>` |
| Mode | `subagent-driven` |
| Branch | `<branch>` |
| Worktree | `<absolute-worktree-path>` |
| Base Branch | `<base-branch>` |
| Started At | `<YYYY-MM-DD HH:mm>` |
| Owner | `<main-agent-or-user>` |
| Status | `active` |

## Input Documents

| Type | Path | Required | Loaded |
| --- | --- | --- | --- |
| Agent Rules | `AGENTS.md` | yes | no |
| Constitution | `docs/CONSTITUTION.md` | yes | no |
| Coding Standards | `docs/CODING.md` | yes | no |
| Spec | `<spec-path>` | yes | no |
| Tasks | `<tasks-path>` | yes | no |
| Checklist | `<checklist-path>` | yes | no |
| Audit / Source Doc | `<audit-or-requirement-path>` | no | no |

## Execution Summary

| Metric | Value |
| --- | --- |
| Total Tasks | `0` |
| Done | `0` |
| Blocked | `0` |
| In Progress | `0` |
| Pending | `0` |
| Last Updated | `<YYYY-MM-DD HH:mm>` |

## Status Legend

| Status | Meaning |
| --- | --- |
| `pending` | 尚未派发 |
| `assigned` | 已派发，subagent 尚未开始 |
| `in_progress` | 正在实现 |
| `verifying` | 正在测试或构建验证 |
| `review` | 等待主 agent 复核 |
| `changes_requested` | 复核要求修改 |
| `blocked` | 被外部条件或设计问题阻塞 |
| `done` | 已完成并通过验证 |

## Task Matrix

| Task ID | Title | Status | Owner | Parallel Group | Depends On | Scope | Allowed Files |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `T001` | `<task-title>` | `pending` | `<agent>` | `P1` | `-` | `<scope>` | `<paths>` |

## Wave Plan

| Wave | Goal | Tasks | Start Condition | Completion Condition |
| --- | --- | --- | --- | --- |
| `W1` | `<wave-goal>` | `T001,T002` | `<condition>` | `<condition>` |

## Task Records

### T001 - `<task-title>`

| Key | Value |
| --- | --- |
| Status | `pending` |
| Owner | `<agent>` |
| Started At | `<YYYY-MM-DD HH:mm>` |
| Completed At | `<YYYY-MM-DD HH:mm>` |
| Branch | `<branch>` |
| Worktree | `<absolute-worktree-path>` |
| Depends On | `-` |
| Parallel Group | `P1` |

**Scope**

- `<scope-item-1>`
- `<scope-item-2>`

**Allowed Files**

- `<path-1>`
- `<path-2>`

**TDD Plan**

- Failing test: `<test-name-or-file>`
- Expected failure: `<expected-failure-message>`
- Minimal implementation: `<implementation-summary>`
- Regression scope: `<affected-module-or-command>`

**Verification Evidence**

| Command | Expected | Actual | Result |
| --- | --- | --- | --- |
| `<command>` | `<expected-output>` | `<actual-output>` | `not_run` |

**Modified Files**

- `<path>`

**Commits**

- `<commit-sha-or-not-committed>`

**Review Notes**

- `<review-note>`

**Risks / Blockers**

- `<risk-or-blocker>`

**Handoff**

- Completion summary: `<summary>`
- Remaining work: `<remaining-work-or-none>`
- First response line used: `当前分支/worktree：<branch> @ <absolute-worktree-path>`

## Cross-Task Decisions

| Time | Decision | Reason | Impact | Owner |
| --- | --- | --- | --- | --- |
| `<YYYY-MM-DD HH:mm>` | `<decision>` | `<reason>` | `<impact>` | `<owner>` |

## API Contract Changes

| API / Field | Change | Frontend Impact | Backend Impact | Docs Updated |
| --- | --- | --- | --- | --- |
| `<api-or-field>` | `<change>` | `<impact>` | `<impact>` | `no` |

## Test Commands

| Area | Command | Required | Last Result | Notes |
| --- | --- | --- | --- | --- |
| Backend Gateway | `cd backend/gateway && go test ./...` | no | `not_run` | `<notes>` |
| Backend Product | `cd backend/product && go test ./...` | no | `not_run` | `<notes>` |
| Backend Auction | `cd backend/auction && go test ./...` | no | `not_run` | `<notes>` |
| Frontend Admin | `cd frontend/admin && npm test -- --runInBand` | no | `not_run` | `<notes>` |
| Frontend Admin Build | `cd frontend/admin && npm run build` | no | `not_run` | `<notes>` |
| Frontend H5 | `cd frontend/h5 && npm test -- --runInBand` | no | `not_run` | `<notes>` |
| Frontend H5 Build | `cd frontend/h5 && npm run build` | no | `not_run` | `<notes>` |

## Final Review Checklist

- [ ] 所有任务状态已更新。
- [ ] 没有未解释的 `blocked` 任务。
- [ ] 每个 `done` 任务都有测试或替代验证证据。
- [ ] 每个实现型任务都遵循 TDD 或写明无法 TDD 的原因。
- [ ] API 契约变更已同步文档。
- [ ] 最终回答第一句展示当前分支/worktree。
- [ ] 用户已获得下一步选项：继续下一波、发起 review、提交 PR、归档。

## Final Handoff

当前分支/worktree：`<branch> @ <absolute-worktree-path>`

**完成项**

- `<done-item>`

**未完成项**

- `<remaining-item-or-none>`

**验证结果**

- `<verification-summary>`

**建议下一步**

- `<next-action>`

---
name: sdd-run
description: Use this skill whenever the user wants to execute an already-created development plan and tasks through SDD/TDD, especially prompts like "/sdd-run", "/sdd-run 这是本次开发的 plan：... task：...，开始执行", "按这个 plan/tasks 开始执行", "派发 subagent 执行这些 tasks", or "继续 SDD 执行". It safely infers state/plan/tasks when omitted, enforces project runbook usage, state-file SSOT, isolated worktree checks, subagent prompts, TDD verification, and the mandatory first response line showing current branch/worktree.
---

# SDD Run

This skill executes an existing `plan` and `tasks` pair through the project SDD workflow. It is not for writing the plan. Use it after the user has already generated or approved plan/tasks and now wants agents to execute.

## Core Contract

Always follow the project files first:

- `AGENTS.md`
- `docs/superpowers/sdd/RUNBOOK.md`
- `docs/superpowers/sdd/state-template.md`
- `docs/CONSTITUTION.md`
- `docs/CODING.md`

Every final response from the main agent and every dispatched subagent starts with:

```text
当前分支/worktree：<branch> @ <absolute-worktree-path>
```

If branch detection fails:

```text
当前分支/worktree：unknown @ <absolute-worktree-path>
```

## Expected User Input

The user may provide:

```text
/sdd-run
```

When no arguments are provided, the bootstrap script safely infers context.

```text
/sdd-run 这是本次开发的plan：<plan-path-or-text> 和 task：<tasks-path-or-text>，开始执行
```

Also support variants:

- `plan: <path>` and `tasks: <path>`
- `方案：<path>` and `任务：<path>`
- `plan：<path> 和 task：<path>`
- `scope: T001-T003`
- `state: docs/superpowers/sdd/runs/...-state.md`
- `继续执行 state: ...`

Input validation:

- New run: explicit `plan` and `tasks` are preferred but not required when safe inference succeeds.
- New run without `state`: automatically create a new state file before any implementation or subagent dispatch.
- Resume run: providing `state:<path>` is sufficient to resume; the script will load the state file and recover plan/tasks/scope from it. Words like `继续`/`continue`/`resume` are accepted but not required.
- Empty `/sdd-run`: infer in this order:
  - exactly one active state with pending work -> resume it
  - otherwise exactly one plan/tasks candidate pair -> create a new state
  - otherwise stop with `needs_selection` and show candidates
- If required inputs are still missing after inference, ask for only the missing inputs and stop.

## Execution Steps

1. Announce that you are using `sdd-run` to execute the prepared plan/tasks.
2. Detect current branch and worktree.
3. Read required project context.
4. Parse user input into:
   - `plan`
   - `tasks`
   - `scope`
   - `state`
   - `mode`
5. Load plan/tasks from file paths when paths are provided.
6. Run the state bootstrap script before any implementation or subagent dispatch:

   ```bash
   python3 docs/superpowers/sdd/scripts/sdd_run.py --repo-root . --input "<user /sdd-run input>"
   ```

7. Parse the script result.
   - If it exits `0`, use the returned `state_path`, `branch`, and `worktree` for the rest of the run.
   - If it exits `3`, parse stdout as JSON. If JSON contains `needs_selection: true`, show candidate states/plans/tasks and stop.
   - If it exits with any other non-zero code, stop and report stderr instead of dispatching subagents.
8. Build a task matrix:
   - task id
   - title
   - status
   - owner
   - dependencies
   - allowed files
   - expected tests
9. Build execution waves:
   - same file means sequential
   - different services/pages may be parallel
   - tests before implementation for every implementation task
10. Dispatch subagents only with explicit bounded scope.
11. Review every subagent result:
   - first line contains current branch/worktree
   - state file updated
   - tests or verification evidence recorded
   - no scope creep
12. Run final verification.
13. Return final handoff.

## Worktree Behavior

For non-readonly work:

- If already in an isolated worktree, continue.
- If on `main`, create or ask to create an isolated worktree before changing business code.
- If there are unrelated dirty changes, do not overwrite them.
- Record branch and worktree in the state file.

Docs-only workflow setup may be done in the current worktree if the user explicitly asks to create or update workflow files.

## Subagent Prompt

When dispatching a subagent, include this block:

```text
你正在仓库 <repo-path> 的 worktree <worktree-path> 中执行 SDD 子任务。

必须遵守：
- 先读取并遵守 AGENTS.md。
- 最终回答第一句必须是：当前分支/worktree：<branch> @ <absolute-worktree-path>
- 遵循 TDD：实现型任务必须先写失败测试或契约测试，确认失败原因符合预期后，才能写实现。
- 如果任务无法自动化测试，必须先在 state 中记录原因、替代验证方式和剩余风险，再继续。
- 不要修改任务范围外文件。
- 不要回滚用户或其他 agent 的改动。
- 完成后先更新状态文件，再汇报。

任务输入：
- state: <state-file-path>
- plan: <plan-path-or-summary>
- tasks: <tasks-path-or-summary>
- task_id: <task-id>
- scope: <scope>
- files: <allowed-files>
- dependencies: <dependency-task-ids>
- expected_tests: <test-commands>
- expected_output: <acceptance-criteria>

交付要求：
- 列出修改文件。
- 列出测试命令和结果。
- 列出未解决风险。
- 如果未完成，必须说明阻塞根因和下一步。
```

## State File Rules

The state file is the SSOT. Use:

```text
docs/superpowers/sdd/runs/YYYY-MM-DD-<topic>-state.md
```

If the user does not provide `state`, create a new file automatically before execution.

This is enforced by the script:

```bash
python3 docs/superpowers/sdd/scripts/sdd_run.py --repo-root . --input "<user /sdd-run input>"
```

Use the script's JSON output as authoritative:

- `state_path`: state file for the run
- `created`: whether a new state file was generated
- `branch`: current branch
- `worktree`: current worktree
- `plan`: parsed plan path
- `tasks`: parsed tasks path
- `scope`: parsed scope
- `mode`: execution mode
- `inferred`: whether context was inferred
- `inference_source`: `active_state` or `single_plan_tasks_pair`
- `needs_selection`: safe inference failed because context was ambiguous

Script fail-fast behavior:

- If a new run omits `plan` or `tasks`, the script first tries safe inference.
- If empty `/sdd-run` has multiple plausible active states or plan/tasks candidates, the script exits with candidate JSON and the agent must ask for selection.
- If `plan` or `tasks` looks like a file path but does not exist, the script exits non-zero.
- If the script exits with code `3`, parse stdout candidate JSON. For other non-zero codes, report stderr and do not dispatch subagents.

Creation rules:

- Derive `<topic>` from the plan filename, tasks directory name, or scope.
- Copy the structure from `docs/superpowers/sdd/state-template.md`.
- Fill run metadata, branch/worktree, input documents, task matrix, wave plan, and summary counts.
- Mark the initial execution tasks as `pending`; mark tasks as `assigned` only when actually dispatched.

Update state before saying a task is complete. Required evidence:

- task status
- branch/worktree
- modified files
- tests and actual result
- risks/blockers
- handoff first line

## Completion Criteria

Only mark a task `done` when:

- implementation matches plan/tasks scope
- TDD evidence exists: failing test or contract test before implementation, then passing verification after implementation
- if automated tests are impossible, alternative verification evidence and residual risk are recorded before implementation
- state file is updated
- docs/contracts are updated when relevant
- subagent first line follows the required branch/worktree format

## Final Response

Use this structure:

```text
当前分支/worktree：<branch> @ <absolute-worktree-path>

**执行结果**
- ...

**状态文件**
- ...

**验证**
- ...

**下一步**
- ...
```

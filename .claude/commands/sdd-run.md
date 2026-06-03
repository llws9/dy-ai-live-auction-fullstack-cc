---
argument-hint: [plan path/text] [tasks path/text] [scope] [state]
description: Execute SDD/TDD tasks through the project Runbook, auto-inferring state/plan/tasks when /sdd-run has no arguments
model: inherit
---

## User Input

```text
$ARGUMENTS
```

You MUST consider the user input before proceeding. Examples:

```text
/sdd-run 这是本次开发的 plan：docs/superpowers/plans/example.md task：.trae/specs/example/tasks.md，开始执行
/sdd-run state: docs/superpowers/sdd/runs/2026-06-02-example-state.md
/sdd-run
```

## Mission

Execute a prepared development plan and task list using the project SDD protocol.

This command is used AFTER the user has already generated or reviewed `plan.md` and `tasks.md`. When arguments are omitted, the bootstrap script infers context safely; do not regenerate the plan.

## Authoritative Behavior

The full execution contract — input parsing, bootstrap script invocation, state file rules, worktree behavior, subagent prompt template, review gate, completion criteria, and final response format — is defined in:

- `.agents/skills/sdd-run/SKILL.md`

This command file is a thin wrapper. **You MUST follow `.agents/skills/sdd-run/SKILL.md` as the source of truth.** Any divergence between this file and the skill is a bug; the skill wins.

## Required Context (from SKILL.md)

Before execution, read:

1. `AGENTS.md`
2. `docs/superpowers/sdd/RUNBOOK.md`
3. `docs/superpowers/sdd/state-template.md`
4. `docs/CONSTITUTION.md`
5. `docs/CODING.md`
6. The explicit or inferred plan file
7. The explicit or inferred tasks file
8. Any checklist/spec/audit referenced by the plan or tasks

## Bootstrap

Before any implementation work or subagent dispatch, run from repo root:

```bash
python3 docs/superpowers/sdd/scripts/sdd_run.py --repo-root . --input "$ARGUMENTS"
```

Exit codes:

- `0`: success — use returned `state_path`, `branch`, `worktree`.
- `2`: ValueError (missing inputs or path does not exist) — STOP and report stderr.
- `3`: `needs_selection: true` — show candidate states/plans/tasks and ask the user to pick. Do not guess.

## Mandatory First Response Line

Every final response from the main agent and every dispatched subagent MUST start with:

```text
当前分支/worktree：<branch> @ <absolute-worktree-path>
```

If branch detection fails, use `unknown` instead of guessing.

## Final Output

Defer to `.agents/skills/sdd-run/SKILL.md` for the final response structure and completion criteria.

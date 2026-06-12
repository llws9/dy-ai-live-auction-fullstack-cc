---
name: fullstack-feature-workflow
description: Use this skill when the user wants to take a full-stack feature from idea to implementation in this repository, especially prompts like "开发一个新功能", "按全栈 workflow 执行", "从需求到前后端落地", "开始一个跨前后端功能", or when a request needs coordinated spec, UI, API contract, writing-plans, sdd-run, and knowledge update. Do not use for already-created plan/tasks execution; use sdd-run for that. Do not use for one-off Q&A, pure review, tiny copy edits, or tasks that do not create code or docs.
---

# Fullstack Feature Workflow

This skill orchestrates the repository's end-to-end development workflow for full-stack features. It is a coordinator: it routes work through existing skills and project protocols instead of replacing them.

Authoritative references:

- `AGENTS.md`
- `docs/superpowers/specs/2026-06-11-fullstack-feature-workflow.md`
- `docs/superpowers/specs/2026-06-11-layered-dev-methodology.md`
- `docs/superpowers/sdd/RUNBOOK.md`
- `docs/superpowers/sdd/state-template.md`

Every final response must start with:

```text
当前分支/worktree：<branch> @ <absolute-worktree-path>
```

If branch detection fails:

```text
当前分支/worktree：unknown @ <absolute-worktree-path>
```

## Applicability Gate

Use this workflow only when the task needs at least one of:

- persisted design/code/docs
- multiple steps from requirement to implementation
- frontend/backend coordination
- API/RPC contract changes
- SDD/TDD execution

Use a lightweight path instead when the request is:

- one-off Q&A, pure analysis, or pure review
- simple copy change
- single-file no-behavior style tweak
- a task that does not need code or docs persisted

For lightweight tasks, record the decision in chat or commit message. Do not force spec/plan/knowledge updates.

Decision order when both could apply (e.g. a small frontend change that asks for "workflow"):

1. If the user explicitly asks for the workflow, use it but pick the minimal path (see `## Minimal Closure Table`).
2. Otherwise, if the change persists code/docs and needs more than one step, use the workflow.
3. Otherwise, use the lightweight path.

A task with no contract impact still uses the workflow; it just skips Stage 2 and any backend wave.

## Stage Protocol

Follow these stages in order. Skip only when the skip rule applies.

```text
[0] Requirement clarification  -> brainstorming -> spec.md
[1] UI design                  -> ui-design-trio or UI brief -> chosen UI direction
[2] Contract-first design      -> lightweight brainstorming -> Contract SSOT
[3] Plan split                 -> writing-plans -> plan.md + tasks.md (+ checklist)
[4] Implementation waves       -> sdd-run -> implementation + TDD evidence (per delivery domain)
[5] Knowledge update review    -> knowledges-update or no-op
```

Stage `[4]` covers all implementation waves. A wave maps to a delivery domain
(frontend app, backend service, shared contract, data migration, docs), not a
fixed frontend-then-backend order. See `## Scale Decision` for how many waves and
how to split them.

### Stage 0 - Requirement Clarification

Invoke `brainstorming` before implementation-oriented work.

Clarify:

- user goal and motivation
- feature scope and non-goals
- success criteria
- target surfaces: H5, Admin, Test Dashboard, backend services
- likely contract impact

Exit condition:

- a spec under `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md`
- if the user is still deciding, stop with focused questions instead of drafting implementation details

### Stage 1 - UI Design

If the feature has meaningful UI/UX decisions, invoke `ui-design-trio` to produce 2-3 alternatives.

Use a single UI brief instead of three variants when:

- the existing design system already determines the answer
- the change is a small UI fix
- the change only reuses existing components
- it is a small information hierarchy adjustment

Skip when there is no UI change.

### Stage 2 - Contract SSOT

Before frontend/backend implementation, establish the API/RPC contract as the single source of truth.

Minimum Contract SSOT fields:

- `path` + `method`: frontend traffic must go through `gateway-service` `/api/v1`
- `request` / `response`: field names, types, optionality, pagination
- `auth`: JWT requirement and downstream `X-User-ID`; no hardcoded user identity or internal tokens
- `error`: error code and semantics
- money fields: backend uses `shopspring/decimal`; no business float
- cross-service dependency: caller, callee, RPC/API path, degradation semantics; no direct cross-service DB query
- `owner` and contract path
- `contract_version`, `frozen_at`, and `owner` when frozen

Contract consistency rules:

- `grep -R "<api-path-or-field>" -n frontend backend docs` is only the minimum regression sentinel.
- Core or cross-service interfaces need at least one stronger check: OpenAPI validation, type check, API test, or frontend/backend mock parity.
- The freeze record and the change-tracking tables (`Cross-Task Decisions`, `API Contract Changes`, `Wave Plan`) live in the SDD state file created from `docs/superpowers/sdd/state-template.md`.
- Once frozen, record the freeze in the Contract SSOT and the SDD state `Cross-Task Decisions` table.
- If the contract changes during execution, increment `contract_version`, update `API Contract Changes`, update `Cross-Task Decisions`, and adjust affected `Wave Plan` start conditions before continuing.

Skip only when there is no contract impact.

### Stage 3 - Plan Split

Invoke `writing-plans` before `sdd-run`. Development-oriented brainstorming must not jump directly from spec to execution.

First decide plan shape via `## Scale Decision`: one plan with task groups, or
separate plans per delivery domain.

Plan outputs:

- `plan.md`
- `tasks.md`
- optional `checklist.md`

If no standalone `checklist.md` exists, embed acceptance criteria in tasks or state, and mark the SDD `Input Documents` entry as `embedded in tasks/state`.

Each task must declare:

- dependency
- write set / read set
- regression sentinel
- verification command
- whether it belongs to frontend, backend, contract, docs, or knowledge update

### Stage 4 - Implementation Waves

Invoke `sdd-run` only after plan/tasks exist.

Prefer:

```bash
/sdd-run 这是本次开发的 plan：<plan-path> 和 task：<tasks-path>，开始执行
```

Equivalent script bootstrap:

```bash
python3 docs/superpowers/sdd/scripts/sdd_run.py --repo-root . --input "<用户的 /sdd-run 输入>"
```

Use script output as authoritative:

- `state_path`
- `branch`
- `worktree`

Frontend and backend waves are not necessarily sequential, and a wave is not
necessarily one frontend plus one backend. Decide the number of waves and how to
split them via `## Scale Decision`. Use `Wave Plan`, `Parallel Group`,
dependencies, write sets, and local service ownership to decide parallelism.

Implementation requirements:

- TDD by default: failing test first, minimal implementation, passing verification
- state file updated before reporting completion
- runtime source recorded for local/browser validation
- no workaround that changes trunk config for local `localhost` / IPv6 / stale process issues

### Stage 5 - Knowledge Update Review

After the feature closes, evaluate whether durable knowledge was created.

Update `.trae/knowledges/**/SKILL.md` only for:

- new constraints
- non-obvious decisions
- reusable workflow lessons
- recurring gotchas

Record no-op when there is no durable knowledge. Do not write temporary execution logs or tool chatter into long-term knowledge.

## Scale Decision

Default: one plan with frontend/backend task groups.

Use separate `writing-plans -> sdd-run` loops by affected delivery domain when any of these is true:

- cross-service call or data contract
- data model or DB migration
- multiple subagents needed
- overlapping write sets require isolation
- many interfaces/pages
- complex state machine
- high compatibility risk
- one plan would overload context

Delivery domain means the actual affected surface: frontend app, backend service, shared contract, data migration, or docs. Do not create an empty frontend/backend loop just to fit the template. For a backend-only cross-service feature, split by backend service or contract boundary.

Do not introduce a scoring system; any clear hit is enough to upgrade.

## Minimal Closure Table

| Task type | Minimal path | Implementation waves at [4] |
|---|---|---|
| Cross-stack feature with UI + contract change | [0][1][2][3][4][5] | frontend + backend |
| Frontend-only, no contract change | [0][1][3][4][5] | frontend only |
| Frontend feature with contract change | [0][1][2][3][4][5] | frontend + backend |
| Backend-only capability | [0][2][3][4][5] | backend only |
| Internal refactor / tiny style change with no behavior change | lightweight path | — |

Judge contract impact before frontend/backend impact. Stage `[4]` always exists
for implementation work; what differs is which delivery-domain waves run inside it.

## Output To User

At each stage, be explicit about:

- current stage
- artifact produced or reused
- skipped stages and why
- next skill to invoke
- blockers or user decisions required

When stopping for user input, ask only the smallest decision needed to continue.

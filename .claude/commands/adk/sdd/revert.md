---
argument-hint: [optional: spec name] [target phase: requirement | design | task | implementation]
description: Revert a spec to the start of a target phase — delete downstream artifacts, clean up approval state, and log the rollback
model: inherit
---

## Mission

Roll back a spec's workflow to the beginning of a specified phase, removing all intermediate artifacts from that phase onward, cleaning up associated approval records, and recording the revert event for traceability. Requires explicit user confirmation before any destructive action.

## Implementation

**CRITICAL: This uses MCP tool call, NOT bash command**

When invoked:

### Step 1 — Resolve Target Spec

`spec name` is OPTIONAL. Resolution order:

1. **Argument provided** → use it directly.
2. **No argument** → infer from current conversation context (the spec being actively worked on in this session).
3. **Cannot infer** → ask the user which spec to revert.

### Step 2 — Determine Target Phase

Parse the `target phase` argument. Accept both singular and plural forms: `requirement`/`requirements`, `design`, `task`/`tasks`, `implementation`.

- If not provided or invalid: ask the user which phase to revert to.
- **Normalize for logging**: When calling the `log` tool in Step 6, always use the internal phase names: `requirements`, `design`, `tasks`, `implementation`. This applies to both the `phase` field and `data.revertedTo`.

### Step 3 — Assess Current State

1. **Find and call** the available MCP tool ending with `spec-status` to retrieve the current workflow state for the resolved spec.
2. Read existing artifact files in `{workflow-dir}/specs/{spec-name}/` (check `.mcp.json` for the `--workflow-dir` argument to determine the actual workflow directory) to confirm what currently exists:
   - `requirements.md`
   - `design.md`
   - `tasks.md`
   - `explore.md`
   - `spec.md` (from prd-to-spec flow)
3. Determine what will be affected based on the target phase:

   | Revert to | Preserved | Deleted / Reset |
   |-----------|-----------|-----------------|
   | `requirement` | _(nothing)_ | `requirements.md`, `design.md`, `explore.md`, `tasks.md`, task implementation progress |
   | `design` | `requirements.md` | `design.md`, `explore.md`, `tasks.md`, task implementation progress |
   | `task` | `requirements.md`, `design.md`, `explore.md` | `tasks.md`, task implementation progress |
   | `implementation` | `requirements.md`, `design.md`, `explore.md`, `tasks.md` | Reset all task checkboxes in `tasks.md` from `[x]`/`[-]` back to `[ ]` |

   - `spec.md` is NEVER deleted (it originates from prd-to-spec and serves as the upstream source of truth).
   - `events.jsonl` is NEVER deleted (append-only audit log).

4. If the target phase has no existing artifact (e.g., reverting to design but `design.md` doesn't exist yet), inform the user there is nothing to revert and STOP.

### Step 4 — Confirmation (REQUIRED)

Present a clear, explicit summary of what will happen and **wait for user confirmation**. Format:

```
⚠️ Revert spec "{spec-name}" to the start of "{target phase}" phase.

Will be DELETED:
  - {workflow-dir}/specs/{spec-name}/design.md
  - {workflow-dir}/specs/{spec-name}/tasks.md
  - {workflow-dir}/specs/{spec-name}/explore.md
  - Related approval records for the above files
  - Implementation progress (task checkboxes reset)

Will be PRESERVED:
  - {workflow-dir}/specs/{spec-name}/requirements.md
  - {workflow-dir}/specs/{spec-name}/spec.md
  - {workflow-dir}/specs/{spec-name}/events.jsonl

This action cannot be undone. Type "confirm" to proceed or "cancel" to abort.
```

- Only proceed if user explicitly replies "confirm" (or clear equivalent like "yes", "go ahead").
- Any other response (including silence, ambiguity, or "wait") → abort and inform the user nothing was changed.

### Step 5 — Log Revert Event (MUST succeed before Step 6)

**BLOCKING**: Call `log` first. Do NOT proceed to delete until log returns success. If log fails (e.g. missing data.revertedTo), fix the parameters and retry — do NOT skip.

**Find and call** the MCP tool ending with `log`:

- `specName`: `{spec-name}`
- `event`: `phase_revert`
- `phase`: `{current phase}` — from Step 3 spec-status (e.g., implementation if reverting from impl to design)
- `data`: **REQUIRED**. `{ "revertedTo": "{target phase}", "deletedArtifacts": ["design.md", "tasks.md", ...], "reason": "<user-stated reason or 'user-initiated revert'>" }`
  - `revertedTo` = target phase; status derivation uses this only

### Step 6 — Execute Revert

Only after Step 5 log succeeds, perform **in order**:

1. **Clean up approvals**: Call the MCP tool ending with `approvals` with action `delete` for each artifact file that will be removed.
2. **Delete artifact files**: Remove each file identified in Step 3 from `{workflow-dir}/specs/{spec-name}/`.
3. **Reset implementation progress** (when revert target is `implementation`): Reset all task checkboxes in `tasks.md` from `[x]` or `[-]` to `[ ]`.

### Step 7 — Completion Report

After successful revert:

1. **Confirm** what was deleted and what was preserved.
2. **Current state**: The spec is now at the start of `{target phase}` with no artifacts for that phase or beyond.
3. **Next step**: Suggest the user resume the workflow:
   - `/adk:sdd:continue {spec-name}` to re-enter the target phase and regenerate artifacts from scratch.

## Examples

- `/adk:sdd:revert design` — Revert the current spec to the start of design phase (deletes design.md, explore.md, tasks.md)
- `/adk:sdd:revert 001-user-auth requirement` — Revert user-auth spec all the way back to requirement phase
- `/adk:sdd:revert task` — Revert to task phase (deletes only tasks.md and implementation progress, preserves requirements + design)
- `/adk:sdd:revert implementation` — Redo implementation from scratch (resets all task checkboxes to `[ ]`, keeps all spec documents intact)

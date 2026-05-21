---
argument-hint: [spec name] [path]
description: Save spec from workflow directory to codebase (git-trackable)
model: inherit
---

## Mission

Persist a spec from `{workflow-dir}/specs/` into the codebase so it can be version-controlled with the code. Copies spec.md, requirements.md, design.md, tasks.md, explore.md to the target module's specs directory.

**Note**: Generally save after all tasks are completed. If tasks are incomplete, the tool will ask for confirmation.

## Implementation

**CRITICAL: This uses MCP tool call, NOT bash command**

**Workflow directory**: Determined by `--workflow-dir` arg in `.mcp.json` (e.g., `.ttadk/.adk-mobile`). Source specs are read from `{workflow-dir}/specs/{spec-name}/`.

When invoked:

1. **Resolve spec name and targetDir** from argument:
   - If argument has two parts (e.g. `story-archive src/feature-name/specs`): first is specName, second is targetDir.
   - If argument has one part: use as specName, targetDir omitted (auto-infer).
   - If argument missing: call `spec-list` first to get active specs, then ask user which one to save (or use current context spec if unambiguous).

2. **Find and call** the MCP tool ending with `spec-save`:
   - Tool name pattern: `mcp__*__spec-save`
   - Top-level `success: true` includes "needs confirmation / inference" flows (not an error). Use **`data.saved === true`** to confirm files were written; if `saved` is false, use the `require*` flags below.
   - Parameters:
     - `specName`: (required) the spec to save
     - `targetDir`: (optional) target directory; omit to auto-infer from File Manifest
     - `confirmOverwrite`: (optional) only pass on retry after user confirmed overwrite; do NOT pass on first call
   - `confirmTaskIncomplete`: (optional) only pass on retry after user confirmed save despite incomplete tasks; do NOT pass on first call

3. **Handle tool responses**:
   - **requirePathConfirmation** (data.requirePathConfirmation): Code inference succeeded. Use AskUserQuestion: "Save to {data.inferredTargetDir}? Confirm or correct the path." If user confirms, call spec-save again with targetDir = data.inferredTargetDir. If user corrects, use the corrected path.
   - **requireModelInference** (data.requireModelInference): Code inference failed. Follow data.modelInferencePrompt: read spec, infer targetDir, confirm with user, then call spec-save with targetDir. If still unclear, use AskUserQuestion to ask user for targetDir.
   - **requireConfirmation** (data.requireConfirmation): Target already exists. Use AskUserQuestion to confirm overwrite. If user confirms, call spec-save again with confirmOverwrite: true.
   - **requireTaskConfirmation** (data.requireTaskConfirmation): Tasks not all completed (data.taskProgress e.g. "3/8"). Use AskUserQuestion: "Tasks not all completed (current {taskProgress}). Save anyway?". If user confirms, call spec-save again with confirmTaskIncomplete: true.
   - **success**: Report the saved path.

## Examples

- `/adk:sdd:save` — Save current/only spec (or prompt to choose)
- `/adk:sdd:save story-archive` — Save spec named story-archive
- `/adk:sdd:save 1-story-archive` — Save spec with numeric prefix
- `/adk:sdd:save story-archive src/feature-name/specs` — Save to specified directory

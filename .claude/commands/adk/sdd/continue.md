---
argument-hint: [spec name]
description: Resume an existing Mobile SDD workflow session
model: inherit
disable-model-invocation: true
---

## Mission
Resume an existing SDD workflow, restoring context and checking status to continue where left off.

## Implementation
**CRITICAL: This uses MCP tool call, NOT bash command**

When invoked:
1. Resolve spec name:
    - `spec name` is REQUIRED. If missing: STOP immediately. Do NOT call any MCP tools. Tell the user to re-run `/adk:sdd:continue <spec name>`.
    - **Spec lookup**: Call the MCP tool ending with `spec-list` (pattern: `mcp__*__spec-list`) to get all specs. Then match user input against the returned `data.specs[].name` list:
      - **Exact match**: user input matches a name directly (e.g., `001-user-auth`).
      - **Prefix/number match**: if user provides only a number or prefix (e.g., `1`, `001`), find the name that starts with that prefix (e.g., `001-user-auth`).
      - **Partial name match**: if user provides a keyword (e.g., `user-auth`), find the name that contains it.
      - **Multiple matches**: list all matching specs and ask the user to pick one.
      - **No match**: inform the user no spec was found and show the full list from `spec-list`.
2. **Find and call** the available MCP tool ending with `sdd-guide` to load workflow context.
    - Tool name pattern: `mcp__*__sdd-guide`
3. Call `spec-status` tool with the resolved spec name.
4. Follow the next steps returned by `spec-status` to resume the workflow.

## Examples
- `/adk:sdd:continue user-auth` (Resumes user-auth spec)
- `/adk:sdd:continue 001` (Finds spec starting with 001, e.g., 001-user-auth)
- `/adk:sdd:continue 1` (Same as above, matches prefix 001)

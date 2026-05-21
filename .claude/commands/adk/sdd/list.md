---
argument-hint:
description: List all specs in the current project with their phase status and progress
model: inherit
---

## Mission

Display a concise overview of all existing specs with their current phase, status, and task progress.

## Implementation

**CRITICAL: This uses MCP tool call, NOT bash command**

When invoked:

1. **Find and call** the available MCP tool ending with `spec-list`.
    - Tool name pattern: `mcp__*__spec-list`
    - No parameters required.
2. Present the returned specs as a Markdown table:

   ```
   | Spec | Phase | Status | Tasks | Last Modified |
   |------|-------|--------|-------|---------------|
   | 001-user-auth | design | approved | — | 2026-02-27 |
   | 002-payment-flow | task | pending | 2/8 | 2026-02-28 |
   ```

3. If no specs returned, respond: **"No specs found. Run `/adk:sdd:new` to create one."**

## Examples

- `/adk:sdd:list` — Show all specs and their current status

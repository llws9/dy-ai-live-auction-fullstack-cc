---
argument-hint: [feature description | doc path | doc url]
description: Start Mobile SDD workflow for feature development (Requirements → Design → Tasks → Implementation)
model: inherit
---

## Mission
Initialize Mobile SDD workflow to systematically implement features with planning, architecture design, and documentation.

## Implementation
**CRITICAL: This uses MCP tool call, NOT bash command**

When invoked:
1. Parse goal from argument and any referenced files
2. **Find and call** the available MCP tool ending with `sdd-guide`
    - Tool name pattern: `mcp__*__sdd-guide`
    - Pass goal as parameter (use MCP tool interface, not shell execution)
3. Follow the workflow steps returned by the MCP tool

## Examples
- `/adk:sdd:new Create user authentication system`
- `/adk:sdd:new Add dark mode support`
- `/adk:sdd:new @prd_file.md Implement feature per PRD`

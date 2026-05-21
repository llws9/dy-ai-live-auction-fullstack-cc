---
argument-hint: [prd or tech design url]
description: Generate initial technical spec from PRD or Tech Design via adk-mobile MCP (prd-to-spec)
model: inherit
---

## Mission
Generate an initial technical spec from a PRD or Tech Design Lark doc URL using the adk-mobile MCP tool. Output is a client-side spec; if the doc covers both client and server, the tool will identify client-side scope (clarify with user if unclear).

## Implementation
**CRITICAL: This uses MCP tool call, NOT bash command**

When invoked:
1. Validate argument:
    - Doc URL (PRD or Tech Design) is REQUIRED.
    - If missing: STOP immediately. Do NOT call any MCP tools. Ask the user to provide a PRD or Tech Design Lark doc URL and re-run `/adk:sdd:spec <url>`.
2. **Find and call** the MCP tool from `adk-mobile` named `prd-to-spec`.
    - Tool name: `mcp__adk-mobile__prd-to-spec`
    - Parameters:
      - `larkDocUrl`: <doc url>
3. Follow the workflow steps returned by the MCP tool (includes **auto-reading linked Lark docs** when relevant to template sections, e.g. tracking doc → Event Tracking; use `lark-docs` MCP as instructed).

## Examples
- `/adk:sdd:spec https://bytedance.larkoffice.com/docx/xxxxxxxxxxxx`
- `/adk:sdd:spec <tech-design-url>` (PRD or Tech Design both supported)

---
name: kb-init-docs
description: Generate/refresh documentation (CLAUDE.md, interface.md, workflow.md, domain.md, rule.md) for given directory path or analyzed directory paths
allowed-tools: Bash(python3 scripts/kb_tool.py*), Read, AskUserQuestion, Edit
---

> Note: All `python3 scripts/...` commands below must be executed with the current skill directory as the working directory (the directory containing this SKILL.md). The scripts resolve their own dependencies relative to that location.

## Intention

Generate or refresh documentation for:

- The given directory path
- The analyzed doc-root paths when `--analysis` is used

## Inputs

- Directory path(s) (required): repo-relative or absolute path; accepts multiple paths separated by spaces and/or newlines
- Optional flag: `--analysis`
- Optional flag: `--lang <auto|en|zh>` (default: `auto`)

## Examples

```bash
/kb-init-docs --analysis path 
/kb-init-docs path --lang en
/kb-init-docs path --analysis --lang zh
/kb-init-docs pathA pathB --analysis
```

## Workflow

Before generating `docs/rule.md`, read and follow the shared rule contract:

- `../kb-docs-validator/references/kb-rule-quality-contract.md`

### 1 Resolve directory path(s) (stable roots)

Interpret the user’s input as:

- If the input contains `--analysis`, remove that flag.
- If the input contains `--lang <auto|en|zh>`, remove that flag pair and record the language preference.
- Split the remaining text by whitespace/newlines into one or more directory paths.
- Treat each directory path independently in the following steps.

### 2 Decide scope

2.1 - If `--analysis` is NOT set: use each given directory path as a doc root and proceed to Step 3 for each path

2.2 - If `--analysis` IS set:
   2.2.1 analyze all given directory paths first, and collect suggested doc roots from the following script:

   !python3 scripts/kb_tool.py suggest-doc-roots "<module_path1>" "<module_path2>" ...
   || (printf '\a' >&2; echo "kb-init-docs error: suggest-doc-roots failed" >&2; exit 1)

   2.2.2 Use AskUserQuestion to ask the user how to choose doc roots. 
     1) Paste newline-separated doc root paths copied from `suggested_doc_roots`
     2) Reply `ALL` (or `全部`) to select all `suggested_doc_roots`
     3) Reply `USE_ARGUMENT_PATHS` (or `使用参数路径`) to use doc root paths already included in the original input argument (newline-separated)

### 3 Generate docs (writes, after explicit user approval)

What to generate:
Generate ONLY the following files and content:

Language:

- If `--lang en`, write all generated content in English.
- If `--lang zh`, write all generated content in Chinese.
- If `--lang auto`, follow the user's language in the conversation.
- **CLAUDE.md** - Main documentation with EXACTLY these three sections:
  - **Module Overview**: Brief description of the directory's primary purpose, responsibilities, and compact module structure.
    - Include a short structure summary when useful (for example: public contracts, implementations, adapters, resources, or other local layers that actually exist in this doc root).
    - Keep structure guidance at summary level. Do not paste long directory trees or code templates in `CLAUDE.md`.
  - **Key Classes**:
    step 1: find important classes/interfaces with one-line concise and clear descriptions by
    !python3 scripts/kb_tool.py select-key-classes <path> --top 20
    || (printf '\a' >&2; echo "kb-init-docs error: select-key-classes failed" >&2; exit 1)
    step 2: display key classes in the following table format
    <example-output>
       | Type | Kind | Des |
       |------|------|-------|
       | `ClassA` | class | brief description |
       | `ClassB` | class | brief description  |
    </example-output>
  - **References**: Fixed references to the following docs
    ```
    - [docs/interface.md](./docs/interface.md) - External interfaces and public APIs exposed by this doc root
    - [docs/workflow.md](./docs/workflow.md) - Business process flows
    - [docs/domain.md](./docs/domain.md) - Business terminology and strategy documentation,
    - [docs/rule.md](./docs/rule.md) - module-specific rules and constraints for future changes in this doc root
    ```
- **docs/interface.md** - External interfaces and public APIs exposed by this doc root (less than 100 lines)
- **docs/workflow.md** - Main business process flows with UML diagrams if needed
- **docs/domain.md** - Business knowledge template for developers, terms and concepts used in the business
- **docs/rule.md** - Module-specific rules for this doc root only.
  - Follow `../kb-docs-validator/references/kb-rule-quality-contract.md`.
  - Before writing rules, inspect ancestor `docs/rule.md` / `rule.md` files and do not duplicate parent/global rules.
  - Generate rules only when they are specific to this doc root, evidence-backed, useful for future code changes, and shaped as `WHEN` + `MUST` / `MUST NOT` scenario blocks.
  - Put compact module structure summaries in `CLAUDE.md`; use `docs/rule.md` only for enforceable constraints derived from that structure.
  - Keep 1-3 high-value compact code examples when they materially improve future coding accuracy; attach each example to a nearby local rule.

#### Document Generation Rules

- CLAUDE.md top-level sections MUST be exactly: `## Module Overview`, `## Key Classes`, `## References`
- If files already exist, no any changes to them
- Place `interface.md`, `workflow.md`, `domain.md`, `rule.md` under a `docs/` folder next to `CLAUDE.md`
- Include README.md content in Module Overview if it exists
- Ensure `AGENTS.md` is a symbolic link pointing to `CLAUDE.md` in the same directory
- Focus on conceptual understanding, not implementation details
- DO NOT add generic development practices or tips

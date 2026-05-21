---
name: kb-update-docs
description: Use when you find outdated/wrong information in existing module docs (CLAUDE.md, workflow.md, domain.md, rule.md); apply minimal, evidence-based edits and preserve formatting
allowed-tools: Bash(python3 scripts/kb_tool.py*), Read, Edit
---

> Note: All `python3 scripts/...` commands below must be executed with the current skill directory as the working directory (the directory containing this SKILL.md). The scripts resolve their own dependencies relative to that location.

## Mission
Update existing module documentation to match the latest code. Only update outdated information. Do not add speculative content.

Target: $ARGUMENTS

Before updating `docs/rule.md`, read and follow the shared rule contract:

- `../kb-docs-validator/references/kb-rule-quality-contract.md`

## Use Scenarios

`$ARGUMENTS` format:
- Optional flags: `--p` and/or `--s` (space-separated, order-insensitive) or `--c`
- Then a target path (repo-relative or absolute)

Target path can be:
- A folder path: update existing docs under that folder (recursive)
- A doc root folder: update docs in that root
- A specific existing doc file path (`CLAUDE.md` or `docs/{workflow,domain,interface,rule}.md`): update only that file

Modes:
- Default: update docs (`CLAUDE.md` and/or `docs/{workflow,domain,interface,rule}.md`) under the given folder path.
- `--p` (CI mode): no user interaction; emit a concise, machine-friendly summary (edited roots + edited files).
- `--s` (scope-parents): starting from the given path, walk upward and update every parent directory that contains `CLAUDE.md` or `docs/{workflow,domain,interface,rule}.md`.
- `--c` (scope-children): starting from the given path, walk downward and update every child directory that contains `CLAUDE.md` or `docs/{workflow,domain,interface,rule}.md`.

## Examples

```bash
/kb-update-docs path 
/kb-update-docs --p path 
/kb-update-docs --s path 
/kb-update-docs --c path 
/kb-update-docs path/CLAUDE.md
/kb-update-docs path/docs/workflow.md
```


## Find Update Targets (no writes)
Resolve update targets under the given path:

!python3 scripts/kb_tool.py list-doc-targets --path "<target_path>"

Resolve update targets in parent scope (`--s`):

!python3 scripts/kb_tool.py list-doc-ancestors --path "<target_path>"

If no targets are found:
- Stop and report that no existing `CLAUDE.md` or `docs/{workflow,domain,interface,rule}.md` files were found under the target scope; do not edit files.

If one or more targets are found:
- Enumerate all doc roots to update
- For each doc root, list the exact files that will be edited and the evidence commands to run (e.g. `scan-types`)
- Execute updates doc-root-by-doc-root, ensuring every root completes before moving to the next

## Update Rules (strict)
- Only update directories that already contain `CLAUDE.md` or `docs/{workflow,domain,interface,rule}.md`.
- Only update existing doc files. Do not create new doc roots or new docs.
- Do not edit any content wrapped by `<nay-ai>...</nay-ai>`; treat it as human-authored and immutable.
- Only edit statements that are outdated according to the latest code structure.
- Do not introduce new “architecture patterns”, “principles”, or “best practices” unless they already exist in the codebase/docs under this module.
- Keep `CLAUDE.md` structure stable:
  - Keep existing H2 sections; remove H2 section only when the whole section is outdated.
- Keep formatting stable across all docs:
  - Do not reflow paragraphs, rewrap lines, or “clean up” punctuation/whitespace.
  - Do not reorder lists, headings, sections, or files.
  - Do not change code fences, diagram blocks, tables, or indentation unless fixing incorrect content inside them.
  - Prefer the smallest possible edit that makes the statement accurate.

## Evidence Requirement
Before changing any “Key Classes” line or API description:
1. Run the type scan for that doc root:
   !python3 scripts/kb_tool.py scan-types --path "<doc_root_relpath>"
2. Only add/remove/update class names if they appear in the scan output (or are clearly present in files under the same doc root).

## What to Update Per Doc Root
For each doc root:
1. `CLAUDE.md`:
   - Update **Module Overview** only when ownership/integration changed.
   - Keep compact module structure summaries here when they help agents navigate the module (for example: public contracts, implementations, adapters, resources, or other local layers that actually exist in this doc root).
   - Do not paste long directory trees or code templates; keep `CLAUDE.md` as the entry summary and index.
   - Update **Key Classes** by removing missing symbols and adding only obvious entry points.
   - Keep **References** exactly as-is.
2. `interface.md`:
   - Update public-facing protocols/routers/services that changed.
   - Keep it < 100 lines; prefer pruning over adding.
3. `workflow.md`, `domain.md`, `rule.md`:
   - Only update outdated named entities, flows, and in-repo examples.
4. `rule.md` quality upgrades for existing docs:
   - Treat low-quality existing rules as update targets when they violate `../kb-docs-validator/references/kb-rule-quality-contract.md`.
   - Compare against ancestor `docs/rule.md` / `rule.md` files and remove child rules that only restate parent/global rules.
   - Move or summarize module structure into `CLAUDE.md`; keep `rule.md` focused on enforceable future-change constraints.
   - Rewrite keep-worthy rules into `WHEN` + `MUST` / `MUST NOT` scenario blocks.
   - Keep only local, evidence-backed rules; preserve 1-3 high-value compact examples when they improve future coding accuracy, but remove standalone tutorials, full templates, generic style guides, product notes, test matrices, and inventories.
   - Apply the scoped length budgets from the shared contract.
   - After updating, run `kb-docs-validator` and continue fixing deterministic `rule.md` quality findings unless the remaining issue needs human judgment.
   - Preserve human-authored `<nay-ai>...</nay-ai>` blocks and ask before removing a large rule section.

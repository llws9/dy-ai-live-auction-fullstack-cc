---
name: kb-evals-creator
description: >
  Create and maintain structured evals.json files for kb-docs-benchmark.
  Use this skill when the user wants to create documentation evaluations,
  generate benchmark questions, set up Q&A tests for knowledge base docs,
  add assertions to evals, or work with kb-docs-benchmark evaluation files.
  Trigger when user mentions: "create evals.json", "kb-docs-benchmark",
  "doc evaluation", "benchmark questions", "Q&A assertions", "generate eval questions",
  "add eval to docs", or asks to validate/test documentation quality.
---

# KB Evals Creator

This skill is the single source of truth for creating and maintaining `evals.json` files used by `kb-docs-benchmark`. `kb-docs-benchmark` should delegate missing or expanded eval generation here instead of maintaining a separate generation workflow.

## Goal

Create realistic Q&A benchmark cases that measure whether a knowledge base helps an agent answer developer questions better than code-only exploration.

High-quality evals should:

- reflect real developer questions about the module
- be answerable from repository evidence
- distinguish docs-assisted answers from code-only answers
- avoid leaking the expected answer through docs file names or benchmark wording
- carry provenance so generated and human-authored questions can be analyzed separately

## Expected Documentation Layout

Prefer this layout:

```text
<doc-root>/
├── CLAUDE.md
└── docs/
    ├── interface.md
    ├── workflow.md
    ├── domain.md
    ├── rule.md
    └── evals/
        └── evals.json
```

Store evals at `<doc-root>/docs/evals/evals.json`.

## Schema Rules

Use this shape:

```json
{
  "suite_name": "kb-docs-benchmark",
  "evals": [
    {
      "id": 1,
      "name": "module-name-qa",
      "mode": "qa",
      "doc_root": "path/to/module-root",
      "doc_paths": [
        "CLAUDE.md",
        "docs/interface.md",
        "docs/workflow.md",
        "docs/domain.md",
        "docs/rule.md"
      ],
      "questions": [
        {
          "id": "q1",
          "question": "Where is the public entry point and what inputs does it require?",
          "provenance": {
            "type": "llm",
            "model": "claude",
            "prompt_source": "kb-evals-creator-docs-first-v1"
          },
          "topics": ["interface"],
          "source_files": ["path/to/source/file"],
          "assertions": [
            {
              "id": "mentions_public_entry",
              "type": "contains",
              "text": "PublicEntryPoint"
            }
          ]
        }
      ]
    }
  ]
}
```

Important rules:

- `suite_name` must be `kb-docs-benchmark`.
- Use `mode: "qa"` for compare benchmarking.
- For in-repo docs, prefer repo-root-relative `doc_root`, not an absolute path.
- `doc_root` must point to the module root that contains `CLAUDE.md`, not the `docs/` directory.
- `doc_paths` are relative to `doc_root`.
- `questions[].id` should be stable and unique, usually `q1`, `q2`, etc.
- Generated questions must include `provenance.type: "llm"` and `provenance.prompt_source`.
- Add `topics` and `source_files` when known. They are metadata for review and analysis, not answer hints.

## Assertion Rules

Assertions must check answer content:

- symbols, APIs, class names, methods, protocols, configuration points
- call-chain nodes, lifecycle/state transitions, fallback behavior
- business concepts, domain terms, invariants, constraints

Supported QA assertion types:

- `contains`: use for exact required symbols or terms. This is best for class names, method names, protocol names, AB keys, enum cases, and stable business terms.
- `regex`: use when the answer may phrase the fact in several valid ways, but must still express a specific relationship, ordering, or set of alternatives. This is best for call-chain ordering, "A or B" alternatives, optional prefixes/suffixes, or flexible wording around a required behavior.

Prefer a mix of `contains` and `regex` when it improves signal. Do not force regex for simple symbol checks; do use regex when multiple independent `contains` checks would be too weak to prove the answer describes the relationship correctly.

Do not assert docs path mentions. Avoid assertion text such as:

- `CLAUDE.md`
- `docs/interface.md`
- `docs/workflow.md`
- `docs/domain.md`
- `docs/rule.md`
- `docs/evals/**`

Bad:

```json
{
  "id": "references_interface_doc",
  "type": "contains",
  "text": "docs/interface.md"
}
```

Good:

```json
{
  "id": "mentions_reward_ad_service",
  "type": "contains",
  "text": "RewardADService"
}
```

Good regex examples:

```json
{
  "id": "describes_manager_to_adapter_chain",
  "type": "regex",
  "pattern": "GMTRewardADManagerAdapter[\\s\\S]{0,300}GMTRewardADAdapter[\\s\\S]{0,300}GMTRewardADHostDelegate"
}
```

```json
{
  "id": "mentions_success_or_fallback_callback",
  "type": "regex",
  "pattern": "(loadDataSuccess|loadDataFail|fallback)"
}
```

Avoid weak regex assertions that match almost anything, such as empty patterns, `.*`, or very broad words unrelated to the module.

## Generation Workflow

Use this workflow when creating a missing `evals.json` or expanding benchmark questions.

1. Resolve the repository root, docs root, evals path, and likely `doc_paths`.
2. Read the docs bundle first. Summarize what the docs claim to cover: public APIs, workflows, domain concepts, rules, constraints, and known edge cases.
3. Launch a targeted codebase exploration subagent using the docs-claims summary.
4. Synthesize or update QA entries from the docs summary, code findings, and existing evals if present.
5. Validate the final file before telling the user it is ready for benchmarking.

## Language Rules

Generate all eval prose in English for repository compatibility. Preserve code symbols, APIs, class names, methods, protocols, AB keys, file names, and assertion keywords in their original spelling.

## Quantity Budget

Do not generate an unbounded eval set.

Default limits:

- If the user gives an exact question count, honor that count unless it exceeds 8.
- If the user does not specify a count, create 5 active questions for a new eval.
- When expanding an existing eval without a requested count, add at most 3 new questions.
- Keep a single eval's active `questions` list at 8 or fewer by default.
- If the user asks for more than 8 questions, stop and confirm the larger benchmark size before writing the file.

When there are more good ideas than the active budget, prioritize questions that cover distinct high-value behavior and mention the remaining ideas in the response as a backlog. Do not put backlog ideas into `evals.json` unless the user explicitly asks to expand the active benchmark.

## Targeted Codebase Exploration

The exploration worker may inspect repository source code and should infer the module's language, framework, and file types from the repo. Do not hard-code Swift, Objective-C, or any specific language.

Exploration constraints:

- Time budget: 5 minutes.
- Scope budget: prefer the most relevant 20-30 files.
- Search strategy: start from symbols, flows, and concepts mentioned in the docs, then inspect key entry points, callers, configuration, and error handling.
- Stop when the budget is exhausted and return findings from evidence already collected.
- Do not pass this exploration result to future benchmark answer subagents. It is only for eval authoring.

Ask the exploration worker to return structured findings:

```json
{
  "documented_and_verified": ["claim supported by code evidence"],
  "documented_but_stale_or_unclear": ["claim that is stale, ambiguous, or unsupported"],
  "important_but_missing_from_docs": ["important code behavior not covered by docs"],
  "public_entries": ["PublicEntryPoint"],
  "key_call_chains": [
    {
      "name": "critical flow",
      "nodes": ["Caller", "Adapter", "NetworkEntry"],
      "source_files": ["path/to/file"]
    }
  ],
  "lifecycle_or_state": ["important lifecycle behavior"],
  "configuration_or_registration": ["registration point"],
  "error_or_fallback_behavior": ["fallback behavior"],
  "cross_file_invariants": ["invariant"],
  "likely_confusions": ["common misunderstanding"]
}
```

## Question Quality

Good questions are:

- specific: "What class handles X?" not "Tell me about X"
- realistic: something a developer would ask while using or changing the module
- answerable: supported by docs and/or code evidence
- discriminating: docs should make a strong answer easier, but code-only agents can still attempt it
- diverse: cover interfaces, workflows, domain concepts, rules, lifecycle, configuration, and fallback behavior

Avoid questions that:

- ask for line numbers or trivia
- reveal docs paths in the prompt
- only test whether an answer cites a document
- are unanswerable from the docs/code evidence
- duplicate existing questions with different wording

## Adding Questions

When extending an existing `evals.json`:

- Read the current file first.
- Preserve existing questions unless the user explicitly asks for cleanup.
- Continue the ID sequence.
- Fill coverage gaps from docs and targeted code exploration.
- Keep assertion IDs readable because they appear in reports.
- Prefer 2-4 meaningful assertions per question.

## Validation

Before finishing, validate JSON syntax and schema rules. If the `kb-docs-benchmark` skill is available in the same plugin checkout, use its validator:

```bash
python3 path/to/kb-docs-benchmark/scripts/validate_evals.py \
  --evals /abs/path/to/repo-root/path/to/module-root/docs/evals/evals.json \
  --base-dir /abs/path/to/repo-root
```

Validation should confirm:

- JSON is valid.
- `doc_root` resolves to the module root.
- `doc_paths` exist relative to `doc_root`.
- question IDs and eval IDs are unique.
- assertions are objective and meaningful.
- QA assertions do not check docs path mentions.

## Output To User

After creating or updating evals, summarize:

- eval file path
- number of questions created or added
- active question count versus the quantity budget
- major topics covered
- whether validation passed
- any stale docs or code/doc mismatches found during targeted exploration

Do not run `kb-docs-benchmark` unless the user asks for it after eval creation.
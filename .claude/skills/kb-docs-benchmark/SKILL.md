---
name: kb-docs-benchmark
description: Benchmark whether a docs root with CLAUDE.md and docs/{interface,workflow,domain,rule}.md improves multi-round Q&A quality, time, and token usage.
---

# KB Docs Benchmark

> Note: This skill runs the benchmark through Claude Code subagents. Python scripts are used only for grading, timing metadata helpers, and report/viewer generation; they must not call models for the benchmark.
>
> Benchmark answer generation uses foreground subagents: the parent agent launches each subagent, waits for its result, writes artifacts, and then continues. Do not use background subagents for benchmark answer runs because timing/usage attribution and missing-answer handling depend on the foreground result.
>
> All `python3 scripts/grade_existing_answers.py`, `python3 scripts/write_timing.py`, and `python3 eval-viewer/...` commands below must be executed with the current skill directory as the working directory (the directory containing this SKILL.md). The scripts resolve their own dependencies relative to that location.

Use this skill to measure whether a documentation root is actually useful for question answering.

## Evaluation Protocol

The goal is to measure documentation usefulness, not to maximize answer pass rate.

Treat each `(question, configuration)` as an evaluation trial with strict information boundaries. Do not help a worker answer better by giving it information that the configuration is supposed to withhold. A lower score for `without_docs` is a valid and useful result; do not "fix" it by leaking docs context.

Configuration meanings:

- `without_docs`: code-only baseline. The worker may inspect repository source code, but must not read, quote, summarize, or cite the docs bundle (`CLAUDE.md`, `docs/interface.md`, `docs/workflow.md`, `docs/domain.md`, `docs/rule.md`, or `docs/evals/**`).
- `with_docs`: docs-assisted condition. The worker may inspect repository source code and may use the docs bundle listed in `doc_paths`.

For a given question, both configurations must have the same repository/code access. The only controlled difference is docs-bundle access. Do not restrict, expand, or otherwise change the source files one configuration may inspect unless the eval is explicitly designed to measure code-access differences rather than documentation usefulness.

Both configurations must use isolated workspaces by default. Prepare them from the same repository revision. Remove the docs bundle from `without_docs`; keep the docs bundle but remove `docs/evals/**` from `with_docs`. If this isolation cannot be prepared, stop and report the run as blocked rather than silently falling back to prompt-only isolation.

When orchestrating subagents:

- Use foreground subagents for benchmark answer runs. Do not set `run_in_background: true` for the `with_docs` / `without_docs` answer workers.
- Do not share answers, evidence, or docs excerpts from one configuration with another configuration.
- Do not tell the `without_docs` worker what the docs contain.
- Do not ask the `without_docs` worker to cite docs paths.
- If `without_docs` cannot answer part of a question from code inspection, that uncertainty should remain in the answer.
- Preserve failures and uncertainty; they are benchmark signal.

The canonical workflow is:
1. Prepare a docs root containing:
   - `CLAUDE.md`
   - `docs/interface.md`
   - `docs/workflow.md`
   - `docs/domain.md`
   - `docs/rule.md`
2. Define QA questions in `<doc-root>/docs/evals/evals.json`.
3. Run Claude Code subagent orchestration to generate answers.
4. Review `benchmark.json`, `benchmark.md`, per-question outputs, and `review.html`.
5. Preserve the configured information boundaries when interpreting results.

## What this skill benchmarks

This skill is focused on docs usefulness, not generic skill creation.

It compares two configurations:
- `without_docs`: code-only baseline; answer from repository/source inspection without reading or citing the docs bundle.
- `with_docs`: docs-assisted condition; answer from repository/source inspection plus the docs bundle.

The benchmark captures:
- pass rate from answer assertions
- wall time
- token usage
- question provenance breakdowns such as human-authored vs LLM-authored questions
- per-question and per-configuration answer artifacts


## Inputs

### Docs root

Expected layout:

```text
<doc-root>/
├── CLAUDE.md
└── docs/
    ├── interface.md
    ├── workflow.md
    ├── domain.md
    └── rule.md
```

### Evals file

Prefer storing benchmark cases in `<doc-root>/docs/evals/evals.json`.

- The evals file should already exist before running this skill.
- If it is missing, ask the user whether to create it with `kb-evals-creator`.
- Use `mode: "qa"` for compare benchmarking.
- Use `mode: "docs"` for direct doc checks.
- For QA questions, add `provenance` when available.
- For generated QA questions, prefer `doc_root` relative to the repository root when the docs root lives in the repository.
- QA assertions must check answer content such as symbols, APIs, call-chain nodes, state transitions, error behavior, or business concepts. Do not assert that an answer contains docs paths such as `CLAUDE.md`, `docs/interface.md`, or `docs/evals/**`.

Example QA entry:

```json
{
  "id": 2,
  "name": "module-foo-qa",
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
        "type": "human",
        "author": "alice"
      },
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
```

## Eval Generation

`kb-docs-benchmark` does not own eval generation. If `<doc-root>/docs/evals/evals.json` is missing, stop and ask the user whether to create it with `kb-evals-creator`. Do not run benchmark answer subagents until `evals.json` exists and passes validation.

The `kb-evals-creator` skill is the single source of truth for:
- docs-first question generation
- targeted codebase exploration with a 5 minute budget
- repo-root-relative `doc_root` for in-repo docs
- QA assertions that check answer content rather than docs path mentions

After `kb-evals-creator` creates or updates the file, validate it locally before running the benchmark:

```bash
cd .claude/skills/kb-docs-benchmark
python3 scripts/validate_evals.py \
  --evals /abs/path/to/repo-root/path/to/module-root/docs/evals/evals.json \
  --base-dir /abs/path/to/repo-root
```

## Canonical command

### Claude Code mode (recommended inside Claude Code)

When this skill is invoked, orchestrate the answers from the current Claude Code session:

1. Resolve the docs root and evals file:
   - docs root: the user-provided path
   - evals: `<doc-root>/docs/evals/evals.json`
   - workspace: `/tmp/kb-docs-benchmark-<doc-root-name>`
2. If the workspace already exists, stop and ask the user for confirmation before deleting it. Do not delete previous benchmark results automatically.

```bash
if [ -d /tmp/kb-docs-benchmark-<doc-root-name> ]; then
  echo "Previous benchmark workspace exists: /tmp/kb-docs-benchmark-<doc-root-name>"
  echo "Ask the user before deleting it. Do not proceed until they confirm."
  exit 2
fi
mkdir -p /tmp/kb-docs-benchmark-<doc-root-name>
```

If the user confirms deletion, then run:

```bash
rm -rf /tmp/kb-docs-benchmark-<doc-root-name>
mkdir -p /tmp/kb-docs-benchmark-<doc-root-name>
```

If the user wants to preserve previous results, choose a new workspace path with a timestamp suffix and report that path.

3. Prepare isolated worktrees for both answer configurations:

```bash
cd .claude/skills/kb-docs-benchmark
python3 scripts/prepare_code_only_worktree.py \
  --repo-root /abs/path/to/repo-root \
  --doc-root path/to/module-root \
  --evals /abs/path/to/repo-root/path/to/module-root/docs/evals/evals.json \
  --workspace /tmp/kb-docs-benchmark-<doc-root-name>
```

This creates:

```text
<workspace>/code_only_worktree
<workspace>/with_docs_worktree
<workspace>/isolation_manifest.json
```

If valid isolated worktrees already exist for the same repo root, doc root, and source revision, the helper reuses them. If either worktree is stale or invalid, the helper recreates both and records `reused` / `recreated_reason` in `isolation_manifest.json`.

The `without_docs` worker must receive `<workspace>/code_only_worktree` as its repository root and must not be pointed at the original repo. This worktree removes `CLAUDE.md` and `docs/`.

The `with_docs` worker must receive `<workspace>/with_docs_worktree` as its repository root and must not be pointed at the original repo. This worktree keeps the docs bundle but removes `docs/evals/**`, so the worker cannot read questions or assertions from `evals.json`.

4. Materialize canonical docs bundles before launching any answer subagent:

```bash
python3 scripts/prepare_docs_bundles.py \
  --evals /abs/path/to/repo-root/path/to/module-root/docs/evals/evals.json \
  --workspace /tmp/kb-docs-benchmark-<doc-root-name> \
  --base-dir /abs/path/to/repo-root
```

This writes:

```text
<workspace>/docs_bundles/eval-<eval_id>.md
<workspace>/docs_bundles/manifest.json
```

The parent orchestrator must treat these files as the only canonical docs bundle source for the run. Do not ask subagents to reformat, translate, summarize, or reconstruct the bundle. For every `with_docs` batch from the same eval, provide the same `<workspace>/docs_bundles/eval-<eval_id>.md` content and copy that exact file to each `outputs/docs_bundle.md`.

5. Read `evals.json`. For a first smoke pass, use only the questions currently present in that file.
6. Instruct answer and analysis subagents to write prose in English while preserving code symbols and API names verbatim.
7. Use `session_batch` granularity by default:
   - Split questions into internal batches of at most 5 questions.
   - For each batch, launch one `without_docs` subagent and one `with_docs` subagent.
   - Do not expose batch IDs in the final report; batching is only an execution-stability strategy.
   - This measures session-level docs usefulness within each batch: the docs bundle is paid once and its cost is amortized across the questions in that batch.
   - If the user explicitly asks for cold-start isolation, use `cold_start_per_question`: one fresh subagent for each `(question, configuration)` pair.
8. Each batch subagent must return one answer per question:

```json
{
  "granularity": "session_batch",
  "configuration": "with_docs",
  "answers": [
    {
      "question_id": "q1",
      "answer": "Final answer text.",
      "evidence": ["Concise evidence or source symbols."]
    }
  ]
}
```

9. For each question, produce two answers:
   - `without_docs`: answer from `<workspace>/code_only_worktree` only. The docs bundle has been removed from that worktree; do not search/read the original repo or cite the docs bundle.
   - `with_docs`: answer from `<workspace>/with_docs_worktree` plus the docs bundle listed in `doc_paths`. The `docs/evals/**` files have been removed from that worktree; do not search/read the original repo.
10. Save each answer to the existing workspace layout:

```text
<workspace>/eval-<eval_id>-<question_id>/<configuration>/run-1/outputs/answer.md
<workspace>/eval-<eval_id>-<question_id>/<configuration>/run-1/outputs/question.md
<workspace>/eval-<eval_id>-<question_id>/<configuration>/run-1/outputs/prompt.md
```

If the Agent/Subagent tool result includes a `<usage>` block, the orchestrating parent agent must also write timing metadata. For `session_batch`, write per-question timing with even amortization by passing the batch question count:

```text
<workspace>/eval-<eval_id>-<question_id>/<configuration>/run-1/timing.json
```

Prefer using the helper:

```bash
python3 scripts/write_timing.py \
  --run-dir /tmp/kb-docs-benchmark-<doc-root-name>/eval-<eval_id>-<question_id>/<configuration>/run-1 \
  --docs-bundle /tmp/kb-docs-benchmark-<doc-root-name>/docs_bundles/eval-<eval_id>.md \
  --total-tokens <usage.total_tokens> \
  --duration-ms <usage.duration_ms> \
  --fallback-duration-ms <parent_wall_clock_duration_ms> \
  --tool-uses <usage.tool_uses> \
  --granularity session_batch \
  --batch-id batch-1 \
  --batch-index 0 \
  --batch-size 5 \
  --batch-question-count <number_of_questions_in_this_batch> \
  --allocation-count <number_of_questions_answered_by_this_subagent>
```

The helper writes this JSON shape:

```json
{
  "total_tokens": 39728,
  "duration_ms": 37301,
  "total_duration_seconds": 37.301,
  "tool_uses": 7,
  "granularity": "session_batch",
  "batch_id": "batch-1",
  "batch_index": 0,
  "batch_size": 5,
  "batch_question_count": 2,
  "allocation_strategy": "even",
  "allocation_count": 2,
  "source_total_tokens": 79455,
  "source_duration_ms": 74602,
  "usage_duration_ms": 74602,
  "fallback_duration_ms": 0,
  "duration_source": "subagent_usage",
  "source_tool_uses": 14,
  "docs_bundle_hash": "",
  "docs_bundle_chars": 0,
  "estimated_docs_tokens": 0,
  "docs_token_estimate_method": ""
}
```

Rules:
- `total_tokens` comes from the Agent/Subagent `<usage>.total_tokens`.
- `duration_ms` comes from `<usage>.duration_ms` when available. If it is missing or `0`, pass the parent-measured wall-clock duration via `--fallback-duration-ms`; the helper will use it and set `duration_source` to `parent_wall_clock`.
- `total_duration_seconds` is `duration_ms / 1000`.
- `tool_uses` comes from `<usage>.tool_uses`.
- If token/tool fields are unavailable, write `0` for those fields rather than guessing. Duration should use the parent wall-clock fallback when available.
- The worker/subagent itself cannot write these values; the parent orchestrator must write them after receiving the Agent result.
- For `session_batch`, report per-question token/time as amortized values and preserve source batch totals in `source_*` fields.
- The final summary should show both total source usage and amortized per-question usage. When aggregating totals, deduplicate repeated per-question `source_*` values from the same session batch.
- For multi-batch `with_docs` runs, the aggregator also estimates docs bundle tokens from canonical `outputs/docs_bundle.md` using `ceil(character_count / 4)` and deduplicates repeated docs bundles by SHA-256 hash. Reports must keep raw `Total Tokens` and show the docs-deduped value only as `Normalized Tokens (est.)`.
- In user-facing summaries, treat `Normalized Tokens (est.)` as the primary total-token metric when it is available. Do not add a prominent `Total Tokens` row in that case; mention that raw total usage is preserved in `benchmark.json`.
- `scripts/write_timing.py` automatically reads the `--docs-bundle` path when provided, otherwise `<run-dir>/outputs/docs_bundle.md` when present, and writes docs bundle hash/token estimate fields. If older timing files contain empty or non-canonical docs fields, rerun `scripts/grade_existing_answers.py` to canonicalize them before regenerating the report.
- Exact per-doc token usage is not available from the Subagent tool. Do not present the docs bundle estimate as precise model billing data.
- Batch metadata is for debugging only. Do not expose batch IDs or batch counts in user-facing summaries unless the user asks about execution details.
- Do not compare `session_batch` and `cold_start_per_question` metrics in the same run summary.

For `with_docs`, also write the canonical docs bundle that was given to the worker:

```text
<workspace>/eval-<eval_id>-<question_id>/with_docs/run-1/outputs/docs_bundle.md
```

11. After all answers are written, run grading only:

```bash
cd .claude/skills/kb-docs-benchmark
python3 scripts/grade_existing_answers.py \
  --evals /abs/path/to/repo-root/path/to/module-root/docs/evals/evals.json \
  --workspace /tmp/kb-docs-benchmark-<doc-root-name> \
  --base-dir /abs/path/to/repo-root \
  --configurations without_docs,with_docs \
  --run-number 1 \
  --granularity session_batch
```

If any expected `answer.md` is missing, grading fails by default and does not publish a partial `benchmark.json`. Use `--allow-partial` only when the user explicitly wants an incomplete diagnostic report.

12. Generate and open the HTML review page:

```bash
python3 eval-viewer/generate_review.py \
  /tmp/kb-docs-benchmark-<doc-root-name> \
  --skill-name kb-docs-benchmark \
  --benchmark /tmp/kb-docs-benchmark-<doc-root-name>/benchmark.json \
  --static /tmp/kb-docs-benchmark-<doc-root-name>/review.html
open /tmp/kb-docs-benchmark-<doc-root-name>/review.html
```

13. If the user wants documentation quality recommendations, run Docs Quality Analysis Mode below before regenerating the final report/viewer.
14. Report `benchmark.md`, `review.html`, `isolation_manifest.json`, and the workspace path. If `docs_quality_suggestions.json` exists, summarize the highest-priority suggestions and ask the user whether they want to optimize the docs based on this analysis. Do not modify docs automatically.

This mode preserves question/config isolation through subagents. Do not run model-calling Python scripts as part of this skill; only run grading, timing, and viewer scripts after subagent answers have been written.

## Docs Quality Analysis Mode

Use this mode after `benchmark.json` has been generated. It explains why the docs helped or failed to help and suggests concrete improvements to the docs root.

1. Launch one analysis subagent. Provide:
   - `benchmark.json`
   - `evals.json`
   - docs bundle paths
   - all available `answer.md`, `grading.json`, and `run_manifest.json` files for `with_docs` and `without_docs`
   - the output contract in `agents/docs_quality_analyzer.md`
2. The analysis worker may inspect the same repository/code access used by answer workers and may read the docs bundle. It should not rerun the benchmark. It must write summary/problem/evidence/recommendation prose in English while preserving category IDs and code symbols verbatim.
3. The worker must write or return this JSON shape:

```json
{
  "summary": "Short explanation of current docs quality.",
  "suggestions": [
    {
      "priority": "high",
      "category": "coverage_gap",
      "question_id": "q1",
      "doc_paths": ["docs/workflow.md"],
      "problem": "The docs do not describe the adapter handoff.",
      "evidence": "with_docs failed assertion mentions_manager_adapter.",
      "recommendation": "Add the manager adapter handoff and owning class to docs/workflow.md.",
      "expected_impact": "Should improve q1 with_docs correctness."
    }
  ]
}
```

Allowed categories:
- `coverage_gap`: `with_docs` still fails because the docs do not cover the behavior.
- `redundant_docs`: `without_docs` already passes consistently, so docs add little signal for this question.
- `misleading_docs`: `with_docs` performs worse than `without_docs`, suggesting stale or confusing docs.
- `navigation_gap`: `with_docs` is correct but materially slower or more token-heavy, suggesting poor docs navigation.
- `stale_or_inconsistent`: docs conflict with code evidence.
- `eval_issue`: the eval assertion is low quality or measures the wrong thing.

4. Save the outputs:

```text
<workspace>/docs_quality_suggestions.json
<workspace>/docs_quality_suggestions.md
```

5. Re-run local aggregation and viewer generation so the suggestions are embedded in `benchmark.json`, `benchmark.md`, and `review.html`.
6. After presenting the report, ask the user whether to apply the suggested docs improvements. Treat docs optimization as a separate user-approved step, not part of the benchmark run.

## Output artifacts

Per question and configuration:

```text
<workspace>/
└── eval-<id>-<question_id>/
    ├── eval_metadata.json
    ├── without_docs/
    │   └── run-<n>/
    │       ├── outputs/
    │       │   ├── question.md
    │       │   ├── prompt.md
    │       │   ├── answer.md
    │       │   └── missing_docs.json
    │       ├── grading.json
    │       ├── timing.json
    │       └── run_manifest.json
    └── with_docs/
        └── run-<n>/
            ├── outputs/
            │   ├── question.md
            │   ├── prompt.md
            │   ├── answer.md
            │   ├── docs_bundle.md
            │   └── missing_docs.json
            ├── grading.json
            ├── timing.json
            └── run_manifest.json
```

Workspace-level outputs:
- `benchmark.json`: viewer-compatible aggregated benchmark
- `benchmark.md`: human-readable summary
- `isolation_manifest.json`: code-only workspace preparation details for `without_docs`
- `docs_quality_suggestions.json`: structured doc-quality suggestions when Docs Quality Analysis Mode is run
- `docs_quality_suggestions.md`: human-readable doc-quality suggestions when Docs Quality Analysis Mode is run
- `review.html`: static review UI

## Viewer review

Render the benchmark with the existing viewer:

```bash
python3 eval-viewer/generate_review.py \
  /tmp/kb-docs-benchmark-real \
  --skill-name "kb-docs-benchmark" \
  --benchmark /tmp/kb-docs-benchmark-real/benchmark.json \
  --static /tmp/kb-docs-benchmark-real/review.html
```

In the viewer, confirm:
- outputs load by question, configuration, and run
- benchmark data groups under `with_docs` and `without_docs`
- pass rate, time, and tokens render correctly
- eval labels come from `eval_name`

## Notes for editing or extending the benchmark

- Prefer additive updates to the subagent orchestration, grading, aggregation, and viewer flow.
- Do not redesign the system when a naming normalization or metadata addition is enough.
- Keep `benchmark.json` viewer-compatible.
- If you need schema details, use `references/schemas.md`.

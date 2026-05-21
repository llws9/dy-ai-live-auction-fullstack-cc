---
description: "Execute or re-execute the env deploy and testing plan defined in test/task.md"

---

## User Input

```text
$ARGUMENTS
```
You **MUST** consider the user input before proceeding (if not empty).

If the given `$ARGUMENTS` contains a link, you need to read the content of the link. For lark/feishu doc URLs, export it via lark-docs MCP (`mcp__lark-docs__export_lark_doc_markdown`), then read the exported markdown content.

If `$ARGUMENTS` explicitly provides a `FEATURE_DIR` value, or clearly provides a feature-directory path, you **MUST**
treat it as the highest-priority target feature input. Reuse that exact value when invoking prerequisite scripts, and do
not ask the user to choose a different feature unless the provided path is invalid or ambiguous.

## Context
**Read context before Executing**:
1. Language Setting
   - Read `preferred_language` from `.ttadk/config.json` (default: 'en' if missing). **IMPORTANT** **Use the configured language for ALL outputs: 'en' → English, 'zh' → 中文. This applies to: generated documents (specs, plans, tasks), interactive prompts, confirmations, status messages, and error descriptions.**

## Outline

1. Run `node .ttadk/plugins/ttadk/core/resources/scripts/check-prerequisites.js --json --require-tasks --include-tasks` from repo root and parse FEATURE_DIR and AVAILABLE_DOCS list. All paths must be absolute.

Goal: Act as a re-entrant test execution dispatcher — automatically resolve the correct testing skills (user-invisible) and execute or re-execute the env deploy + test plan defined in `test/task.md` and `test/case.md`.

### Skill Profiles (domain-based, user-invisible)

SDT must keep skill-selection logic **invisible** to normal users. Do **not** ask users to choose skills.

1. Read SDT skill profiles from `plugins/ttadk/core/resources/templates/sdt-skill-profiles.json`.
2. Infer `DOMAIN` from `FEATURE_DIR/spec.md` and stable repo signals (PSM/product/system identifiers). If unsure, use `unknown`.
3. Resolve `PROFILE` from `domains[DOMAIN].profile`, otherwise use `defaultProfile`.
4. Choose mapping by platform:
   - Backend tasks → `profiles[PROFILE].backend`
   - Frontend tasks → `profiles[PROFILE].frontend`
5. Apply **fallback rule** per `sdt:ff`:
   - If a mapped skill is missing on this host, fall back to `profiles[defaultProfile]` for the same logical key.
6. Keep the final mapping as `SKILLS` (logical key → skill name). Do not display skill names to normal users unless debugging is explicitly requested.

### 0: Git Status Check

1. Run `git status --porcelain` from repo root to check for uncommitted changes.
2. If the output is **non-empty** (there are uncommitted changes):
   - Display the list of changed files to the user.
   - Ask the user (using the host-compatible structured question tool): "There are uncommitted changes in the repository. Would you like to commit them before proceeding?"
     - **Yes**: Execute the `/adk:commit` skill to stage, commit, and push changes. After the commit completes, continue to the next step.
     - **No**: Skip committing and continue directly to the next step.
3. If the output is **empty** (no uncommitted changes), proceed directly to the next step.

### 1: Pre-check

1. Read `test/case.md` under FEATURE_DIR.
   - If `test/case.md` does not exist, **STOP** and display error: "test/case.md not found. Please run `/adk:sdt:ff` first to generate test cases."
2. Read `test/task.md` under FEATURE_DIR.
   - If `test/task.md` does not exist, **STOP** and display error: "test/task.md not found. Please run `/adk:sdt:ff` first to generate test tasks."

Parse the test case file:
- Count total test cases.
- Identify platforms distribution (Backend / Frontend / Client).
- Identify priority distribution (P0 / P1 / P2 / P3).
4. Display summary:
   ```
   Loaded test/case.md:
   - Total test cases: N
   - Backend test cases: X
   - Frontend test cases: Y
   - Client test cases: Z
   ```

### 2: Read test/task.md File

- **Task phases**: Env Deploy, Tests, Report
- **Task dependencies**: Sequential vs parallel execution rules
- **Task details**: ID, description, file paths, parallel markers [P]
- **Execution flow**: Order and dependency requirements
- **Task status markers**: Treat both checklist status (`- [ ]`, `- [x]`) and task-table status rows (for example `| 状态 | \`[ ]\` |`) as authoritative task status.

### 2.5: Re-entry / Rerun State Normalization

This command is **re-entrant**. Re-running `/adk:sdt:implement` after a previous execution must not force test case generation again when reusable generated cases already exist.

1. Parse `test/task.md` status and classify the current run:
   - **Fresh run**: `test/task.md` has pending tasks and no completed test execution report. Continue normally.
   - **Interrupted resume**: some tasks are `[x]` and some are `[ ]`. Skip completed tasks and continue from the first pending task, subject to dependencies.
   - **Rerun after fix**: all tasks are `[x]`, or `test/report.md` exists from a previous run, or `$ARGUMENTS` indicates rerun/retest/after-fix/no-regenerate intent. In this mode, reuse existing cases and rerun execution tasks.
2. Detect reusable generated case artifacts:
   - Generic required source: `test/case.md` exists.
   - Backend/API generated cases: at least one `test/**/api_test_case.yaml` exists under `FEATURE_DIR`.
   - Frontend/WebE2E: `test/case.md` is sufficient; WebE2E execution uses platform cases and must not regenerate `test/case.md`.
3. In **Rerun after fix** mode:
   - **Do not regenerate cases** if reusable generated case artifacts exist.
   - Keep case-generation tasks completed. Examples: backend `T005: 生成 API 测试用例`, or any task whose title/description means "generate test cases".
   - Reset test execution and report/diagnosis tasks from `[x]` to `[ ]` before execution. Examples:
     - Backend: `T006` / `T007` / `T008` / `T009`
     - Frontend: `T002` / `T003`
   - For env deploy tasks, preserve existing completed status only if the user explicitly says the current environment is ready, or the task input indicates `run_env=local`. Otherwise ask whether deployment/check should be rerun; if the user chooses rerun, reset the relevant Env Deploy tasks to `[ ]`.
   - Write the normalized statuses back to `test/task.md` **before** Step 3 parallelization analysis.
4. If rerun intent is detected but generated case artifacts are missing:
   - Ask the user whether to regenerate cases or stop.
   - Do not silently regenerate cases when the user explicitly requested no regeneration.
5. Display a short re-entry summary before execution:

   ```text
   Re-entry mode: Rerun after fix
   Reused cases: yes/no, paths: [...]
   Reset tasks: [...]
   Preserved tasks: [...]
   ```

### 3: **⚠️ MANDATORY: Parallelization Analysis (MUST be done BEFORE any task execution)**

**You MUST NOT start executing any task until this analysis is complete.** Before running the first task, you are required to explicitly identify and plan all tasks that can be executed in parallel. Skipping this step is not allowed.

1. **Scan all pending tasks** (`[ ]` in checklist items or task-table status rows) in the normalized `test/task.md` phase by phase.
2. **Identify parallel groups**:
   - Tasks explicitly marked with `[P]` within the same phase are candidates for parallel execution.
   - Verify no file-path conflict between candidates — tasks touching the same files MUST run sequentially, even if marked `[P]`.
   - Verify no implicit dependency (e.g., one task's output is another's input).
3. **Produce a Parallelization Plan** and present it to the user before execution. The plan MUST include:
   - Phase name
   - Parallel groups: `[Group-N] → [TaskID-A, TaskID-B, ...]`
   - Sequential tasks (tasks that must run alone): `[TaskID-X]`
   - Reasoning for why non-`[P]` or conflicting tasks cannot be parallelized
4. **Only after the plan is produced**, proceed to Step 4 to execute tasks according to the plan.

### 4: Execute Tasks Based on Dependencies in test/task.md
- **Follow the Parallelization Plan from Step 3**: Dispatch parallel groups concurrently, run sequential tasks one by one
- **Phase-by-phase execution**: Complete each phase before moving to the next
- **Never skip pending tasks**: If a task is marked as `[ ]`, do not skip it, except when a documented task-level skip condition in `test/task.md` applies and you immediately mark it `[x]` with the skip reason recorded.
- **Reuse generated cases on rerun**: In Rerun after fix mode, do not execute case-generation tasks that were preserved as `[x]`; proceed directly to the first pending deploy/test task according to dependencies.
- **Respect dependencies**: Run sequential tasks in order, parallel tasks [P] can run together
- **Follow TDD approach**: Execute test tasks before their corresponding implementation tasks
- **File-based coordination**: Tasks affecting the same files must run sequentially
- **Validation checkpoints**: Verify each phase completion before proceeding

**Skill invocation rule (important):**

- When a task says "use `<some-skill>`", interpret it as "use the resolved skill for the corresponding logical key" from `SKILLS`.
- Backend tasks typically require: `SKILLS.TEST_KNOWLEDGE`, `SKILLS.API_TEST`.
- Frontend tasks typically require: `SKILLS.DEPLOY`, `SKILLS.WEBE2E_META`, `SKILLS.WEBE2E_RUN`.

### 4: Summary:
1. Read the report template: `plugins/ttadk/core/resources/templates/sdt-report-template.md`.
2. Fill in the template with execution results:
   - **Execution Overview**: execution time, skill name, scope, pass/fail/skip counts and rates.
   - **Result Details Table**: each test case's TC-ID, title, method, status, **Log ID** (extracted from each PSM's `test_report.md` — every case must have a Log ID, use N/A if unavailable), duration, failure reason.
   - **Failure Analysis**: for each failed case — error message, root cause, Argos log (backend), fix status.
3. Write the report to `test/report.md`.
4. Display the execution summary.


### 6: **⚠️ CRITICAL: Progress Tracking (MUST follow strictly)**

**Resume Rule**:
- If this is an **Interrupted resume**, skip tasks already marked as `[X]` or `[x]` and move to the next uncompleted task.
- If this is a **Rerun after fix**, first apply Step 2.5 normalization: preserve reusable case-generation tasks, reset execution/report tasks, then run from the first normalized pending task.
- If all tasks are already `[x]`, do **not** immediately declare completion. Treat it as a possible rerun after fix, inspect `$ARGUMENTS` and prior artifacts, and ask only when rerun intent is ambiguous.

**Immediate Update Rule**: As soon as a task is completed, you MUST immediately update its status in `test/task.md` from `[ ]` to `[x]` BEFORE moving to the next task. This applies to both checklist items and task-table status rows. Do NOT batch updates. Do NOT wait until the end.

**Workflow**: Complete task → Verify completion → Update `test/task.md` (`[ ]` → `[x]`) → Move to next task

Other error handling:
- Report progress after each completed task
- Halt execution if any non-parallel task fails
- For parallel tasks [P], continue with successful tasks, report failed ones
- Provide clear error messages with context for debugging

### 7. Completion validation:
- Verify all required tasks are completed

### 8. Lark Export
   - Use `mcp__lark-docs__import_markdown_to_lark` to import the generated report into Lark
   - Parameter settings:
     - `filePath`: Absolute path of the generated `test/report.md`
     - `title`: **IMPORTANT** - Generate a concise descriptive title that summarizes the core purpose of the feature:
       - Use the language matching the `preferred_language` setting (zh → Chinese, en → English)
       - Keep it short (preferably no more than 30 characters)
       - Do not use the folder name directly — summarize the actual content
   - Obtain the Lark document URL
   - **Directly use the `open` command to open the Lark document link**:
     ```bash
     open "https://feishu.cn/docx/xxxxx"
     ```
   - provide
     ```markdown
      ## Report Generated
      **Lark Document**: [Link] (opened in browser)
      **Local File**: `test/report.md`
     ```

### 9. **Final Task Completion Check**:
- Re-read `test/task.md` and verify no pending `[ ]` task status remains
- If incomplete tasks exist, complete them before proceeding



## Error Handling

| Scenario | Handling |
|----------|---------|
| test/case.md does not exist | Prompt user to run `/adk:sdt:ff` first |
| Skill not registered/does not exist | Display error "Testing Skill {name} not found" |
| Fix introduces new failures | Mark as "New failure introduced by fix" in report and alert |
| No test cases matching the platform | Display "No test cases found for {platform}" |
| Skill execution timeout | Mark test case as SKIP and continue to the next one |

## Next Step Guidance

After executing this command:

### Step 1 - Review Results
Review `test/report.md` to understand test results and any remaining failures.

### Step 2 - Next Step Recommendation
Once all tests pass or results are satisfactory:

**Commit Changes**: Execute `/adk:commit` to stage changes, generate commit message, and push to remote.

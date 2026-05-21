---
description: "Create or update the feature specification from a natural language feature description."

---

## User Input

```text
$ARGUMENTS
```
You **MUST** consider the user input before proceeding (if not empty).

If the given `$ARGUMENTS` contains a link, you need to read the content of the link. For lark/feishu doc URLs, export it via lark-docs MCP with `outputDir` set to `specs/` directory, then read the exported markdown content to understand the feature requirements.

**Preserve References**: If user input contains local references (images, files, diagrams, etc.), preserve these references in spec.md with paths adjusted to remain valid from spec.md location. Read referenced content to understand context, but maintain reference format rather than forcing content embedding.

## Context
**Read context before Executing**:
1. Language Setting
   - Read `preferred_language` from `.ttadk/config.json` (default: 'en' if missing). **IMPORTANT** **Use the configured language for ALL outputs: 'en' → English, 'zh' → 中文. This applies to: generated documents (specs, plans, tasks), interactive prompts, confirmations, status messages, and error descriptions.**

## Outline

The text the user typed after `/adk:sdd:specify` in the triggering message **is** the feature description. Assume you always have it available in this conversation even if `$ARGUMENTS` appears literally below. Do not ask the user to repeat it unless they provided an empty command.

Given that feature description, do this:

1. **Input Quality Assessment**: Evaluate the completeness of user input before proceeding.

    a. **Document Detection & Business Type Classification**:

    First, determine whether the user input contains a structured document or document link (lark/feishu doc URL, local `.md` file path, or content that contains sections, tables, code blocks, mermaid diagrams — i.e., goes beyond a brief natural-language description).

    - **If a document/link is present**: Read the full document content (using lark-docs MCP export for lark/feishu URLs, or direct file read for local paths). Then classify the business type by scanning for signal keywords:

        | Business Type | Signal Keywords & Patterns (match ANY) |
        |---------------|----------------------------------------|
        | **backend** | PSM names (e.g., `local_alliance_*`), IDL references (`Request`/`Response` structs, protobuf), DB table/field changes (`新增字段`/`修改字段`), Go file paths (`.go`), RPC/gRPC calls, `domain/`/`action/`/`dal/` code paths, mermaid flowcharts with service nodes, TCC config |
        | **client** | Platform markers (`Android`, `iOS`, `TikTok`, `TikTok-Lite`), TUX components (`TUXIntroPanel`), `ARouter` / `Coordinator`, `.kt`/`.swift` file paths, Figma links with mobile UI, `R.color.*`/`R.drawable.*`, `Glide`, AB test config with traffic splits |
        | **lynx** | `ReactLynx`, Spark JSB (`x.request`, JSBridge), `lynx-figma-to-code`, `lynxbase-mcp`, `spark-mcp`, `hdt-mcp`, `jotai`, container APIs, `.tsx` files in lynx project structure |
        | **web** | `Valtio`/`Redux`/`proxy()`, `Semi-UI`/`gne-base-components`, route config (`path:`/`component: lazy()`), `bam.config`, `pages/` directory structure, `apps/op-tool/`, `searchParams`, BAM IDL |

    - **Classification rules**:
        - Count signal matches per type. Select the type with the highest match count.
        - If two types tie, prefer the type whose signals appear in structural positions (headings, table headers) over those appearing only in body text.
        - If no type gets >= 2 signal matches, classify as `unknown` and skip template linting — fall through to generic scoring only (sub-step **d**).

    - **If NO document/link is present** (user provided only a brief natural-language description): Skip directly to sub-step **(d)** for generic 5-dimension scoring only.

    b. **Template-Based Linter Check** (only when business type is identified in step a):

    Load the corresponding ERD template from `.ttadk/plugins/ttadk/core/resources/templates/specify-input-template/`:

    | Business Type | Template File |
    |---------------|---------------|
    | backend | `backend-erd-template.md` |
    | client | `client-erd-template.md` |
    | lynx | `lynx-erd-template.md` |
    | web | `web-erd-template.md` |

    Compare the user's document content against the template's required sections. For each section, assess coverage status as one of: ✅ adequate / ⚠️ incomplete / ❌ missing.

    **Backend linter sections**:

    | Section | Level | ✅ Adequate | ⚠️ Incomplete | ❌ Missing |
    |---------|-------|-------------|---------------|-----------|
    | 需求背景 | mandatory | Clear problem statement with design principles (>50 chars) | Has background but vague on goals | Section absent |
    | 涉及工程 (PSM + paths) | mandatory | Table with PSM, repo paths, and change types | Some services listed but no paths or change types | Section absent |
    | 功能模块 (≥1 module with 功能概述) | mandatory | At least 1 module with 功能概述 + IDL + DB + 代码改动 + 调用链路 | Module(s) present but missing ≥2 sub-sections | No modules defined |
    | IDL 变更 (per module) | mandatory | Table with change type, object, description | Mentioned but no structured table | Absent |
    | DB 变更 (per module) | mandatory | Table with change type, table/field, description | Mentioned but incomplete | Absent |
    | 代码改动 (per module) | mandatory | Table with change type, file/method, description | Some files listed but no detail | Absent |
    | 调用链路 (per module) | mandatory | Mermaid diagram or clear textual call chain | Partial description | Absent |
    | 行业差异化逻辑汇总 | conditional | Comparison table across dimensions | Partial comparison | Absent when multiple industries exist |
    | 风险点 | recommended | Numbered risk items with mitigation hints | Risks mentioned but vague | Absent |

    **Client linter sections**:

    | Section | Level | ✅ Adequate | ⚠️ Incomplete | ❌ Missing |
    |---------|-------|-------------|---------------|-----------|
    | Basic Info (Platform, App, Goal) | mandatory | All three fields specified | 1-2 fields present | Section absent |
    | Editable Scope & File Manifest | mandatory | File list with `[NEW]`/`[MOD]`/`[REF]` tags and descriptions | Files listed but no tags or descriptions | Section absent |
    | UI/UX Structure + Figma | optional | Figma links per UI block with component mapping | Partial Figma links | N/A |
    | Data Models & API | optional | Pseudo-code models + API endpoints with error handling | Partial models or missing error handling | N/A |
    | Business Logic (When→Then) | mandatory | Init + Interaction flows in When→Then structure | Some logic described but not in When→Then format | Section absent |
    | Event Tracking | optional | Event table + parameter definitions | Events listed, no parameters | N/A |
    | AB Testing | optional | Experiment groups with traffic splits | Mentioned but no config | N/A |
    | Constraints | recommended | Navigation, images, theme rules specified | Partial constraints | Absent |

    **Lynx linter sections**:

    | Section | Level | ✅ Adequate | ⚠️ Incomplete | ❌ Missing |
    |---------|-------|-------------|---------------|-----------|
    | 功能简述 | mandatory | Checklist of functional points | Vague single-line description | Section absent |
    | 方案设计: 交互流程 | mandatory | Per-page/component interaction flows | Some flows but not per-page | Section absent |
    | 方案设计: 状态管理 | mandatory | State management approach specified (e.g., jotai) | Mentioned but no detail | Absent |
    | 方案设计: 组件通信 | mandatory | Data flow between components described | Vague | Absent |
    | 设计稿 (Figma) | recommended | Per-page Figma node URLs with notes | Single link, no per-page breakdown | Absent |
    | 组件依赖分析 | mandatory | Component list with source/implementation/figma per component | Components listed but missing source | Section absent |
    | 能力依赖: 容器 | mandatory | JSBridge calls with platform and API details | Mentioned but no API structure | Section absent |
    | 能力依赖: 现有能力 | mandatory | Existing utilities/methods with usage examples | Listed but no usage detail | Section absent |
    | 能力依赖: 接口 | mandatory | API endpoints with type definitions or IDL config | Some endpoints but no types | Section absent |

    **Web linter sections**:

    | Section | Level | ✅ Adequate | ⚠️ Incomplete | ❌ Missing |
    |---------|-------|-------------|---------------|-----------|
    | 功能简述 | mandatory | Checklist of functional points | Vague description | Section absent |
    | 整体架构 | conditional | Page structure diagram + data flow | Partial structure | Absent when multi-page |
    | 目录规划 | conditional | Directory tree with file placement | Partial | Absent when multi-page |
    | 技术选型 | conditional | Selection with rationale | Selection without rationale | Absent when new deps |
    | 全局状态管理 | conditional | Store design with types + state ownership table | Mentioned but no design | Absent when cross-page state |
    | 公共组件封装 | conditional | Props interface + placement + usage scenarios | Listed but no interface | Absent when cross-page reuse |
    | 页面方案 (≥1 page) | mandatory | At least 1 page with: 设计稿 + 交互流程 + 异常处理 + 状态管理 + 组件拆分 + 接口设计 | Page present but missing ≥3 sub-sections | No pages defined |
    | 埋点设计 | optional | Event table with parameters | Events listed, no params | N/A |
    | 权限设计 | optional | Permission table with keys and scope | Partial | N/A |
    | 路由 & 菜单配置 | recommended | Route config code + menu hierarchy | Partial config | Absent |

    **Linter scoring rules**:
    - `mandatory`: ✅=2, ⚠️=1, ❌=0
    - `conditional` (condition met): ✅=2, ⚠️=1, ❌=0 — if condition NOT met, exclude from scoring
    - `recommended`: ✅=1, ⚠️=0.5, ❌=0
    - `optional`: ✅=0.5, ⚠️=0.25, ❌=0 (bonus only, does not reduce score)
    - **Template Coverage Score** = earned points / max possible points (mandatory + conditional + recommended) × 100%

    c. **Merge Linter + Generic Scoring → Unified Score**:

    Always run the generic 5-dimension scoring from sub-step **(d)** regardless of whether template linting was performed.

    - **When template linting WAS performed** (document with identified business type):
      **Unified Score** = Generic Score (0-10) × 40% + Template Coverage (0-100%) × 0.1 × 60%, mapped to 0-10 scale
    - **When template linting was NOT performed** (no document, or `unknown` type):
      **Unified Score** = Generic Score (0-10) as-is

    **Tier determination** (applied to Unified Score):

    | Unified Score | Tier | Action |
    |---------------|------|--------|
    | **8-10** | Sufficient | Proceed directly with specification generation |
    | **5-7** | Workable | Proceed with notes on weak areas; AI will make reasonable inferences marked as `[INFERRED]` |
    | **0-4** | Early-stage | Recommend enriching input or using `/adk:sdd:brainstorm` (if available). **Stop and wait for user confirmation before proceeding.** |

    d. **Generic 5-Dimension Scoring** (always applied, score each 0-2: Low/Medium/High):

    | Dimension | High (2) | Medium (1) | Low (0) |
    |-----------|----------|------------|---------|
    | **Goal Clarity** | Explicit problem/goal statement | Has direction but vague | Only a topic keyword |
    | **Actor Identification** | Named roles with scenarios | Roles inferable from context | Cannot determine users |
    | **Functional Behavior** | Step-by-step interaction flow | Some behaviors described | No concrete actions |
    | **Scope & Constraints** | Explicit boundaries and constraints | Partial constraints mentioned | No scope information |
    | **Acceptance Hint** | Measurable success criteria | Implicit expectations | No completion criteria |

    e. **Present Unified Assessment to user** (use configured language):

    **Format A — When template linting was performed:**

        ```markdown
        ## Input Assessment

        **Detected Business Type**: [backend | client | lynx | web]
        **Template Used**: [template filename]
        **Maturity: [TIER]** (Unified: [SCORE]/10 — Generic: [G]/10, Template Coverage: [T]%)

        ### Generic Quality Score ([G]/10)

        | Dimension            | Score | Note                        |
        |----------------------|-------|-----------------------------|
        | Goal Clarity         | [X]/2 | [brief note]                |
        | Actor Identification | [X]/2 | [brief note]                |
        | Functional Behavior  | [X]/2 | [brief note]                |
        | Scope & Constraints  | [X]/2 | [brief note]                |
        | Acceptance Hint      | [X]/2 | [brief note]                |

        ### Template Coverage Report ([T]%)

        | Section | Level | Status | Feedback |
        |---------|-------|--------|----------|
        | [section name] | mandatory | ✅ adequate / ⚠️ incomplete / ❌ missing | [specific actionable feedback] |
        | ... | ... | ... | ... |

        ### Recommendation
        - **Sufficient**: "Input is well-structured and covers key template sections. Proceeding with specification generation."
        - **Workable**: "Proceeding. The following template sections need attention: [list missing/incomplete mandatory sections]. AI will infer where possible (marked [INFERRED]). Refine later via `/adk:sdd:clarify`."
        - **Early-stage**: "The document is missing critical sections: [list missing mandatory sections]. Consider enriching the ERD document, or proceed — the spec can be refined later via `/adk:sdd:clarify`."
        ```

    **Format B — When template linting was NOT performed (generic scoring only):**

        ```markdown
        ## Input Assessment

        **Maturity: [TIER]** ([SCORE]/10)

        | Dimension            | Score | Note                        |
        |----------------------|-------|-----------------------------|
        | Goal Clarity         | [X]/2 | [brief note]                |
        | Actor Identification | [X]/2 | [brief note]                |
        | Functional Behavior  | [X]/2 | [brief note]                |
        | Scope & Constraints  | [X]/2 | [brief note]                |
        | Acceptance Hint      | [X]/2 | [brief note]                |

        ### Recommendation
        - **Sufficient**: "Input is well-formed. Proceeding with specification generation."
        - **Workable**: "Proceeding. Weak areas will rely on AI inference (marked [INFERRED]). Refine later via `/adk:sdd:clarify`."
        - **Early-stage**: "Input is brief. Consider enriching it for better results, or proceed — the spec can be refined later via `/adk:sdd:clarify`."
        ```

    f. **Routing**:
    - **Sufficient / Workable**: Proceed to step 2 automatically.
    - **Early-stage**: Present the assessment, then **stop and ask the user whether to proceed or enrich input first**. Do NOT continue to step 2 until the user explicitly confirms. If user confirms, the generated spec will contain more `[INFERRED]` content.

2. Get feature name and setup feature directory:
- Analyze the feature description and extract key concepts (actors, actions, outcomes)
- Create a concise 3-part name in format: "part1-part2-part3"
  - Each part should be 2-10 characters, lowercase, alphanumeric with hyphens
  - Focus on: domain-action-outcome or subject-verb-object pattern
  - Examples: "user-auth-login", "payment-refund-process", "data-export-csv"
- Run `node .ttadk/plugins/ttadk/core/resources/scripts/create-new-feature.js "<generated-name>" --json` from repo root, replacing `<generated-name>` with the name you created above.
- This script generates a feature name (YYYYMMDD-description format), creates `specs/{feature-name}/` directory, and copies spec.md template.
- Parse the JSON output to get FEATURE_DIR and SPEC_FILE path.

3. **Load guiding principles**: Read `docs/CONSTITUTION.md` (fallback: `.ttadk/memory/constitution.md` for legacy projects) and apply these principles when generating the specification.

4. **Read compound knowledge assets** (if available):
   - **IF EXISTS**: Scan `docs/` for available knowledge assets (e.g., check `docs/arch/`, `docs/references/`, root-level `docs/*.md`). Load whichever files are relevant to the current task based on their filenames and contents — do not rely on a hardcoded list.
   - Use these assets to inform requirement analysis and avoid duplicating established patterns

5. Load `.ttadk/plugins/ttadk/core/resources/templates/spec-template.md` to understand required sections.

6. Follow this execution flow:

    1. Parse user description from Input
        If empty: ERROR "No feature description provided"
    2. Extract key concepts from description
        - **CRITICAL - Completeness Guarantee**: The final spec.md MUST be a superset of user input (spec.md >= user input). Every line of information from user input must be traceable in spec.md. If it exists in user input, it MUST exist in spec.md. This includes all references to local resources (code blocks, images, files, diagrams) - preserve these references so they remain findable in spec.md.
        - Identify: actors, actions, data, constraints, technical implementation details, etc.
    3. For unclear aspects:
        - Make informed guesses based on context and industry standards
        - Only mark with [NEEDS CLARIFICATION: specific question] if:
          - The choice significantly impacts feature scope or user experience
          - Multiple reasonable interpretations exist with different implications
          - No reasonable default exists
        - **LIMIT: Maximum 10 [NEEDS CLARIFICATION] markers total**
        - Prioritize clarifications by impact: technical correctness > business logic > edge cases
    4. Fill User Scenarios & Testing section
        - If no clear user flow: ERROR "Cannot determine user scenarios"
        - For each user story, fill in all fields including **Technical Implementation** section
        - **Technical Implementation**: Extract complete implementation details from user input (implementation flow, interface design, model design, data tables, pseudo code, code modifications, configuration, diagrams, local file references, etc.)
    5. Generate Functional Requirements
        Each requirement must be testable
        Use reasonable defaults for unspecified details (document assumptions in Assumptions section)
    6. Define Success Criteria
        Create measurable, technology-agnostic outcomes
        Include both quantitative metrics (time, performance, volume) and qualitative measures (user satisfaction, task completion)
        Each criterion must be verifiable without implementation details
    7. Identify Key Entities (if data involved)
    8. Return: SUCCESS (spec ready for planning)

7. Write the specification to SPEC_FILE using the template structure:
   - **CRITICAL**: Preserve ALL information from source document - nothing should be lost
   - Replace placeholders with concrete details from user input
   - Preserve section order and headings
   - **Write-size guardrail**: First write ≤ 150 lines; each subsequent update ≤ 100 lines (measured by newline-delimited line count). If a single write would exceed the limit, split by complete lines into multiple writes, preserving order; do not overwrite existing content and do not fail.
   - **If lark doc was exported**: When filling the `Input` field in spec.md, include both the original URL and the local file path. Calculate the correct relative path based on feature directory depth:
     - For simple feature (e.g., `20250107-feature`): `specs/20250107-feature/spec.md` → `../doc_export/file.md`
     - The feature directory is always a single level under `specs/`
     - Example: `**Input**: https://bytedance.larkoffice.com/wiki/xxx (local copy: [filename.md](../doc_export/file.md))`

8. **Specification Quality Validation**: After writing the initial spec, validate it against quality criteria:

    a. **Create Spec Quality Checklist**: Generate a checklist file at `FEATURE_DIR/checklists/requirements.md` using the checklist template structure with these validation items:

    ```markdown
    # Specification Quality Checklist: [FEATURE NAME]

    **Purpose**: Validate specification completeness and quality before proceeding to planning
    **Created**: [DATE]
    **Feature**: [Link to spec.md]

    ## Content Quality

    - [ ] All user stories from source document are captured
    - [ ] Technical implementation details are preserved for each story
    - [ ] All mandatory sections completed
    - [ ] No information lost from source document
    - [ ] **Completeness check (CRITICAL)**: spec.md >= user input. For every line in user input, verify it has a corresponding entry in spec.md. All references (code blocks, images, local files) from user input must be findable in spec.md

    ## Requirement Completeness

    - [ ] No [NEEDS CLARIFICATION] markers remain
    - [ ] Requirements are testable and unambiguous
    - [ ] Success criteria are measurable
    - [ ] All acceptance scenarios are defined
    - [ ] Edge cases are identified
    - [ ] Scope is clearly bounded
    - [ ] Dependencies and assumptions identified

    ## Feature Readiness

    - [ ] All functional requirements have clear acceptance criteria
    - [ ] User scenarios cover primary flows
    - [ ] Success criteria are defined

    ## Notes

    - Items marked incomplete require spec updates before `/adk:sdd:clarify` or `/adk:sdd:plan`
    ```

    b. **Run Validation Check**: Review the spec against each checklist item:
    - For each item, determine if it passes or fails
    - **For completeness check**: Review user input ($ARGUMENTS) side by side with spec.md, verify every piece of information appears in spec (descriptions, requirements, technical details, all references, code examples, etc.)
    - Document specific issues found (quote relevant spec sections)

    c. **Handle Validation Results**:

    - **If all items pass**: Mark checklist complete and proceed to step 8

    - **If items fail (excluding [NEEDS CLARIFICATION])**:
        1. List the failing items and specific issues
        2. Update the spec to address each issue
        3. Re-run validation until all items pass (max 3 iterations)
        4. If still failing after 3 iterations, document remaining issues in checklist notes and warn user

    - **If [NEEDS CLARIFICATION] markers remain**:
        1. Extract all [NEEDS CLARIFICATION: ...] markers from the spec
        2. **LIMIT CHECK**: If more than 10 markers exist, keep only the 10 most critical (by technical correctness/business logic/edge case impact) and make reasonable inferences for the rest
        3. For each clarification needed (max 10), present options to user in this format:

            ```markdown
            ## Question [N]: [Topic]

            **Context**: [Quote relevant spec section]

            **What we need to know**: [Specific question from NEEDS CLARIFICATION marker]

            **Suggested Answers**:

            | Option | Answer | Implications |
            |--------|--------|--------------|
            | A      | [First suggested answer] | [What this means for the feature] |
            | B      | [Second suggested answer] | [What this means for the feature] |
            | C      | [Third suggested answer] | [What this means for the feature] |
            | Custom | Provide your own answer | [Explain how to provide custom input] |

            **Your choice**: _[Wait for user response]_
            ```

        4. **CRITICAL - Table Formatting**: Ensure markdown tables are properly formatted:
            - Use consistent spacing with pipes aligned
            - Each cell should have spaces around content: `| Content |` not `|Content|`
            - Header separator must have at least 3 dashes: `|--------|`
            - Test that the table renders correctly in markdown preview
        5. Number questions sequentially (Q1, Q2, ... up to Q10 max)
        6. Present all questions together before waiting for responses
        7. Wait for user to respond with their choices for all questions (e.g., "Q1: A, Q2: Custom - [details], Q3: B")
        8. Update the spec by replacing each [NEEDS CLARIFICATION] marker with the user's selected or provided answer
        9. Re-run validation after all clarifications are resolved

    d. **Update Checklist**: After each validation iteration, update the checklist file with current pass/fail status

8. **Test Task Generation Workflow Execution**:
   - First, check if user input contains any test-related descriptions.
   - If test-related descriptions are found, you **MUST** run the `/adk:sdt:ff` command.
   - Do not pause or ask the user for permission before doing this.
   - Do not show **Next Step Guidance** that mentioned in `/adk:sdt:ff`.
   - If no test-related descriptions are found in user input, skip this step.

9. Report completion with feature name, spec file path, checklist results, and readiness for the next phase (`/adk:sdd:clarify` or `/adk:sdd:plan`).

**NOTE:** The script generates a feature name in YYYYMMDD-description format and initializes the spec file before writing.

## General Guidelines

- Focus on **extracting and preserving** ALL details from user input
- **CRITICAL - Completeness Guarantee**: spec.md MUST be a superset of user input. If any information exists in user input, it MUST be findable in spec.md. This applies to every line, every reference (code blocks, images, local files), and every detail. After generation, verify: for any content in user input, you can locate its corresponding entry in spec.md.
- Technical implementation details are REQUIRED in each user story
- **Local references**: Preserve references from user input (images, files, etc.) with adjusted paths valid from spec.md location. Read referenced content to understand context, but keep reference format rather than forcing inline embedding.
- **Write-size guardrail**: The limit applies per write/update to spec.md (measured by line count), not to the document's total length. The first write must be ≤ 150 lines and establish the skeleton plus initial section(s). Each subsequent update must be ≤ 100 lines and add content by section or subsection until the document is complete. If a single operation would exceed the limit, split it into multiple writes by complete lines, preserving original order; do not overwrite existing content, do not fail, and do not omit or over-summarize required content.

### Section Requirements

- **Mandatory sections**: Must be completed for every feature
- **Optional sections**: Include only when relevant to the feature
- When a section doesn't apply, remove it entirely (don't leave as "N/A")

### For AI Generation

When creating this spec from a user prompt:

1. **Preserve all details**: Extract and preserve ALL information from source document - nothing should be lost
2. **Organize into template**: Format the extracted information into the template structure
3. **Technical Implementation is key**: Each user story's Technical Implementation section should contain complete implementation details
4. **Limit clarifications**: Maximum 10 [NEEDS CLARIFICATION] markers - use only for genuinely unclear points:
    - Ambiguous technical specifications
    - Missing key information
    - Conflicting requirements
5. **Prioritize clarifications**: technical correctness > business logic > edge cases
6. **Think like a tester**: Every vague requirement should fail the "testable and unambiguous" checklist item
7. **Common areas needing clarification** (only if no reasonable default exists):
    - Feature scope and boundaries (include/exclude specific use cases)
    - Technical implementation conflicts or ambiguities
    - Data model and interface design decisions
8. **Local references**: Preserve references from source (images, code files, diagrams) with paths adjusted for spec.md location. Keep reference format; read content for understanding but don't force embedding.

### Success Criteria Guidelines

Success criteria must be:

1. **Measurable**: Include specific metrics (time, percentage, count, rate)
2. **User-focused**: Describe outcomes from user/business perspective
3. **Verifiable**: Can be tested/validated

**Good examples**:

- "Users can complete checkout in under 3 minutes"
- "System supports 10,000 concurrent users"
- "95% of searches return results in under 1 second"
- "Task completion rate improves by 40%"

**Bad examples** (implementation-focused):

- "API response time is under 200ms" (too technical, use "Users see results instantly")
- "Database can handle 1000 TPS" (implementation detail, use user-facing metric)
- "React components render efficiently" (framework-specific)
- "Redis cache hit rate above 80%" (technology-specific)

## Next Step Guidance

After executing this command, provide next-step guidance to user:

### Step 1 - Confirmation
Guide user to verify the generated spec.md is correct and captures all requirements.

**If needs adjustment**: Run `/adk:sdd:clarify [feedback]` to refine the specification.

### Step 2 - Next Step Recommendation
Once spec is confirmed and satisfactory:

**Create Implementation Plan**: Execute `/adk:sdd:plan` to generate the technical implementation plan based on the specification.

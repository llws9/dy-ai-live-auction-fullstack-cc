# TTADK Mobile 命令参考

## SDD 工作流命令

### /adk:sdd:spec

从 PRD 或技术方案 Lark 文档生成初始技术 Spec。输出是后续流程的核心输入，需研发理解需求并仔细补充。

**对应 MCP 工具**：`prd-to-spec`

**用法**：`/adk:sdd:spec <lark-doc-url>`

**示例**：
- `/adk:sdd:spec https://bytedance.larkoffice.com/docx/xxxxxxxxxxxx`

**前置条件**：必须提供 Lark 文档 URL（PRD 或技术方案均可）。

**行为**：
1. 验证参数中包含文档 URL；缺失则提示用户补充。
2. 调用 `mcp__adk-mobile__prd-to-spec`，传入 `larkDocUrl`。
3. 如果文档涉及客户端和服务端，工具会识别客户端范围（不清时与用户确认）。
4. 按返回步骤推进，包括自动读取关联的 Lark 文档（如埋点文档 → Event Tracking 部分）。

**注意**：Spec.md 是后续流程核心，`/adk:sdd:spec` 的作用是辅助生成粗版草稿，仍需研发理解需求并仔细补充。

---

### /adk:sdd:new

启动 Mobile SDD 工作流，从目标描述或 Spec 文档开始，依次经过 Requirements → Design → Tasks → Implementation 四个阶段。每个阶段需 Review & Approve 后自动进入下一阶段。

**对应 MCP 工具**：`sdd-guide`

**用法**：`/adk:sdd:new <功能描述 | 本地文件路径 | 飞书 URL>`

**示例**：
- `/adk:sdd:new Create user authentication system`
- `/adk:sdd:new Add dark mode support`
- `/adk:sdd:new @spec_file.md Implement feature per Spec`

**推荐输入**：使用 AI Technical Spec 模版对需求进行系统化描述（参见 Spec 输入模版章节）。前期描述越清楚，需求设计生成质量越高。

---

### /adk:sdd:continue

恢复已有的 SDD 工作流会话，从断点继续推进。

**对应 MCP 工具**：`sdd-guide` + `spec-list` + `spec-status`

**用法**：`/adk:sdd:continue <spec-name>`

**示例**：
- `/adk:sdd:continue user-auth`（精确匹配）
- `/adk:sdd:continue 001`（前缀匹配 `001-user-auth`）
- `/adk:sdd:continue 1`（同上）

**行为**：
1. 调用 `spec-list` 获取所有 Spec，匹配用户输入（精确 / 前缀 / 关键词）。
2. 多个匹配时让用户选择；无匹配时展示完整列表。
3. 调用 `sdd-guide` 加载工作流上下文，再调用 `spec-status` 获取进度。
4. 按 `spec-status` 返回的下一步继续推进。

**常用场景**：模型降智或服务中断后，`/clear` + `/adk:sdd:continue <spec-name>` 可安全恢复。

---

### /adk:sdd:clarify

通过逆序扫描制品、交互式问答（最多 5 个问题）和级联更新，解决 Spec 制品中的歧义和不一致。

**用法**：`/adk:sdd:clarify [spec-name] [可选：聚焦问题]`

**示例**：
- `/adk:sdd:clarify`（自动检测当前 Spec）
- `/adk:sdd:clarify user-auth`
- `/adk:sdd:clarify 001-payment-flow the retry logic seems unclear`

**使用场景**：在需求开发过程中，发现需求或技术设计需要修改时，通过 clarify 保证修改在不同阶段的一致性。小改动用 clarify，大改动用 revert。

**行为**：
1. 解析目标 Spec（参数 / 上下文推断 / 询问用户）。
2. 调用 `spec-status` 确定当前阶段，构建制品链。
3. 按逆序（最新阶段优先）扫描制品，检测歧义、不一致、覆盖缺失、过时引用。
4. 生成最多 5 个按影响排序的澄清问题，逐一交互。
5. 每次回答后立即级联更新所有受影响制品。

---

### /adk:sdd:revert

将 Spec 回退到指定阶段的起始状态，删除该阶段及之后的制品。

**对应 MCP 工具**：`spec-status` + `log` + `approvals`

**用法**：`/adk:sdd:revert [spec-name] <target-phase>`

**可用的 target-phase**：`requirement`、`design`、`task`、`implementation`

**示例**：
- `/adk:sdd:revert design`（回退到 design 阶段起点）
- `/adk:sdd:revert 001-user-auth requirement`（回退到 requirement 阶段起点）
- `/adk:sdd:revert implementation`（重置所有任务进度）

**受影响范围**：

| 回退到 | 保留 | 删除/重置 |
|--------|------|-----------|
| `requirement` | （无） | requirements.md, design.md, explore.md, tasks.md, 实现进度 |
| `design` | requirements.md | design.md, explore.md, tasks.md, 实现进度 |
| `task` | requirements.md, design.md, explore.md | tasks.md, 实现进度 |
| `implementation` | 所有文档 | 重置 tasks.md 中的任务 checkbox |

**永不删除**：`spec.md`（上游真相来源）、`events.jsonl`（审计日志）。

---

### /adk:sdd:save

将 Spec 从 MCP 工作流目录保存到代码仓库，便于纳入 git 版本管理，随 MR 一起合入并沉淀。

**对应 MCP 工具**：`spec-save`

**用法**：`/adk:sdd:save [spec-name] [target-dir]`

**示例**：
- `/adk:sdd:save`（保存当前/唯一 Spec）
- `/adk:sdd:save story-archive`（保存指定 Spec）
- `/adk:sdd:save story-archive src/feature-name/specs`（保存到指定目录）

**行为**：
1. 解析参数确定 `specName` 和 `targetDir`。
2. 调用 `spec-save`；会自动推断目标路径，也可手动指定。
3. 处理需要确认的场景：路径确认、已存在覆盖、任务未完成。

**保存的文件**：`spec.md`、`requirements.md`、`design.md`、`tasks.md`、`explore.md`。

---

### /adk:sdd:list

列出所有 Spec 及其当前阶段、状态和任务进度。

**对应 MCP 工具**：`spec-list`

**用法**：`/adk:sdd:list`

**输出格式**：

```text
| Spec | Phase | Status | Tasks | Last Modified |
|------|-------|--------|-------|---------------|
| 001-user-auth | design | approved | — | 2026-02-27 |
| 002-payment-flow | task | pending | 2/8 | 2026-02-28 |
```

---

## 基础公共命令

### /adk:help

回答 Mobile SDD 工作流、命令用法、KB Skills、当前状态等问题。

**用法**：`/adk:help [你的问题]`

**示例**：
- `/adk:help 下一步该做什么`
- `/adk:help /adk:sdd:new 怎么用`
- `/adk:help 怎么为模块生成文档`

---

### /adk:readiness

评估移动端仓库或特定模块的 AI Coding 就绪度，生成多维度成熟度报告。

**用法**：`/adk:readiness [模块路径 | --all-modules | 其他选项]`

**示例**：
- `/adk:readiness`（仓库级扫描）
- `/adk:readiness Modules/Search`（模块级扫描）
- `/adk:readiness --all-modules`（批量扫描所有模块）

**评估维度**：

仓库级（5 个维度）：Context Engineering、Build & Dependencies、Style & Validation、Security & Governance、SDD Readiness

模块级（3 个维度）：Module Documentation、Module Testing、Module Code Organization

**综合评分**（模块扫描时）：40% 仓库基础设施 + 60% 模块评分。成熟度等级 L1 → L4。

---

### /adk:commit

提交当前工作区改动，自动生成规范化 commit message 并 push。支持多仓库/子模块。

**用法**：`/adk:commit`

**单仓库流程**：`git add -A` → 生成 conventional commit → `git commit` + `git push`。

**多仓库/子模块流程**：先逐个 commit 有变更的子模块，最后 commit 主仓库。

---

## 知识库 Skills

### kb-init-docs

为模块目录首次生成结构化文档（CLAUDE.md + docs/）。

**用法**：
- `kb-init-docs <module-path>`（直接为指定路径生成文档）
- `kb-init-docs --analysis <path>`（分析路径下的子目录，推荐 doc roots；适合大仓库）
- `kb-init-docs <path> --lang en|zh`（指定语言）

**生成的文件**：CLAUDE.md、AGENTS.md（符号链接）、docs/interface.md、docs/workflow.md、docs/domain.md、docs/rule.md

**注意**：如果不满意 LLM 生成的文件质量，可以自行修改文档内容。

---

### kb-update-docs

根据最新代码增量更新已有模块文档，仅修改过时内容。

**用法**：
- `kb-update-docs <path>`（默认模式）
- `kb-update-docs --p <path>`（CI 模式，无交互）
- `kb-update-docs --s <path>`（向上遍历更新父级文档）
- `kb-update-docs --c <path>`（向下遍历更新子级文档）

**更新规则**：仅更新已存在的文档，`<nay-ai>...</nay-ai>` 包裹的内容不可修改。

---

### kb-docs-validator

校验知识文档质量，检测跨文件冲突和不一致，对齐代码与文档。

**用法**：`kb-docs-validator <doc-root1> [doc-root2 ...]`

---

### kb-evals-creator

为 kb-docs-benchmark 创建和维护 `evals.json` 评测文件。

**输出位置**：`<doc-root>/docs/evals/evals.json`

---

### kb-docs-benchmark

通过 Q&A 对比测试（有/无文档）衡量文档对回答质量的影响。

**对比维度**：通过率、响应时间、Token 用量、问题来源分析。

---

## Spec 输入模版

推荐使用以下模版结构化描述需求，作为 `/adk:sdd:new` 的输入：

```markdown
# AI Technical Spec: [任务名称]

## 1. Basic Info (基本信息)
- Platform: [Android | iOS]
- App: [TikTok | TikTok-Lite | TikTok-TV]
- Goal: [一句话描述需求与目标]

## 2. Editable Scope & File Manifest (改动范围 & 文件清单)
- Modification Boundary: 只能修改 <path> 内文件
- Target Files:
  - [NEW] /path/to/NewFile.kt: [新建说明]
  - [MOD] /path/to/ExistingFile.kt: [修改说明]
  - [REF] /path/to/ReferenceFile.kt: [参考说明]

## 3. UI/UX Structure (界面交互结构描述) (Optional)
- Figma 设计稿链接（不同 UI 区块贴独立 Figma node 链接）
- TUX 组件约束
- 相对位置（Anchor）

## 4. Data Models & API (数据模型与接口) (Optional)
- Models (Pseudo-code / JSON)
- API Interactions（含 Loading/Error 处理）

## 5. Business Logic (业务逻辑流程)
使用 "When → Then" 格式描述交互

## 6. Event Tracking (埋点设计) (Optional)
- 新增埋点表格
- 已有埋点参数变更

## 7. AB Testing Setup (AB 测试配置) (Optional)
- 实验组配置表格

## 8. Constraints (约束与规范)
- Navigation、Images、Theme 等约束
```

---

## 工作流个性化配置

`.ttadk/.adk-mobile/config.toml` 支持以下配置项：

```toml
port = 49972                    # Dashboard 端口
lang = "zh"                     # 输出语言 "en" | "zh"
approvalMode = "dashboard"      # 审批模式 "dashboard" | "cli"
codeReview = false              # 是否分阶段 Review 代码
userCustomDir = "."             # 业务定制目录（相对于 .ttadk/.adk-mobile/）
useRemoteD2C4Android = false    # Android 是否开启 Remote D2C
```

- **approvalMode**：`dashboard` 模式在 Web 网页上进行 Review 和审批；`cli` 模式在命令行中弹出 Review 提示。
- **codeReview**：开启后，每个 task 完成时 SDE 会暂停等待人工 Review 代码。
- **userCustomDir**：业务定制目录，包含 `user-templates/`（模版）、`user-hooks/`（阶段钩子）、`user-knowledge/`（业务知识）。

---

## MCP 配置

adk-mobile MCP 通过 `.mcp.json` 配置：

```json
{
  "mcpServers": {
    "adk-mobile": {
      "command": "npx",
      "args": [
        "-y", "--registry=https://bnpm.byted.org",
        "@byted-tiktok/adk-mobile-mcp@latest",
        "--workflow-dir", ".ttadk/.adk-mobile"
      ],
      "env": { "platform": "Android" }
    }
  }
}
```

- `platform`：`Android` 或 `iOS`。
- `--workflow-dir`：工作流目录（`.ttadk/.adk-mobile`）。
- Lemon8 项目可额外传 `--user-custom-dir`。

## Preset 配置

| Preset | 包含插件 |
|--------|---------|
| `ttadk/android` | `ttadk/mobile-core` + `ttadk/android` |
| `ttadk/ios` | `ttadk/mobile-core` + `ttadk/ios` |
| `lemon8/android` | `ttadk/mobile-core` + `lemon8/android` |
| `lemon8/ios` | `ttadk/mobile-core` + `lemon8/ios` |

平台插件（`ttadk/android`、`ttadk/ios`）包含各平台特有的 Skills（如 Android 的 smartrouter、tux、ab-test、network、applog 等；iOS 的 Serena、Xcode Build、TTKC Swift 等），已集成到各自仓库中。

# Mobile SDD 工作流详细指南

## 为什么需要 SDD？

Vibe Coding（直接让 AI 写代码，不做结构化规范）在移动端大型项目中会导致：

| 问题 | 表现 | SDD 如何解决 |
|------|------|-------------|
| 架构腐化 | 命名/结构混乱，模块边界模糊 | Spec 预先定义架构；Requirements 强制规范 |
| 知识蒸发 | 只有 AI 知道代码怎么跑 | requirements.md + design.md 形成活的文档 |
| 速度悖论 | 起步快但后续全是填坑 | Spec 保持匀速前进，减少返工 |
| 上下文丢失 | AI 在长对话中忘记上下文 | 每个命令从 Spec 文件读取，不依赖对话历史 |
| 质量问题 | 没测试、没文档、代码像黑箱 | 先产出设计制品，分阶段 Review 再实现 |

**SDD 核心理念**：在让 AI 写代码之前，用结构化的方式描述清楚要做什么。Spec 成为真相来源，代码只是它的"编译产物"。

## 工作流概览

### 标准模式（推荐）

```text
readiness → sdd:spec <prd-url> → sdd:new <spec> → [sdd:clarify] → sdd:save → commit
```

或直接：

```text
readiness → sdd:new <goal> → [sdd:clarify] → sdd:save → commit
```

`/adk:sdd:new` 启动后，工作流自动流转四个阶段，每个阶段需 Review & Approve 后进入下一阶段。

### 知识库建设流程

```text
readiness → kb-init-docs --analysis <path> → kb-docs-validator → kb-evals-creator → kb-docs-benchmark → kb-update-docs
```

## SDD 四个阶段

### Phase 1：Requirements（需求分析）

**目的**：通过问答做需求澄清分析，明确模糊点（交互方式、策略、UI、埋点等）。

**产出**：`.ttadk/.adk-mobile/specs/{spec-name}/requirements.md`

**人机交互**：AI 通过对答做需求澄清，用户**务必仔细 Review & Approve** 后进入 Phase 2。

### Phase 2：Design（技术设计）

**目的**：通过问答扫清设计歧义，生成技术设计方案。

**产出**：`.ttadk/.adk-mobile/specs/{spec-name}/design.md`、`explore.md`（可选）

**人机交互**：AI 提出设计方案和探索性问题，用户 **Review & Approve** 后进入 Phase 3。

### Phase 3：Tasks（任务拆分）

**目的**：将设计方案拆解为可执行的原子化任务。

**产出**：`.ttadk/.adk-mobile/specs/{spec-name}/tasks.md`

**任务特征**：包含编号、checkbox、依赖关系、文件路径。审批通过后进入 Phase 4。

### Phase 4：Implementation（实施）

**目的**：按 tasks.md 逐项落地代码。

**D2C 集成**：在涉及 UI 的 task 中，工作流会自动触发 Design-to-Code：
- **Android**：支持 Remote D2C（云端 Figma → 代码流水线）和 Local D2C
- **iOS**：使用 Local D2C MCP

**Code Review**：如果 `config.toml` 中 `codeReview = true`，每个 task 完成后暂停等待人工 Review。

## 审批模式

`.ttadk/.adk-mobile/config.toml` 中 `approvalMode` 支持两种模式：

### Dashboard 模式（默认）

流程自动打开 Web 网页，在网页上进行 Review 操作：
- 实时查看工作流进度和日志 Timeline
- 选中内容 → 评论 → 请求修订
- 审批通过后自动流转到下一阶段
- Mac 系统通知提醒，自动定位到浏览器审批 Tab

### CLI 模式

在命令行中弹出文档 Review 提示：
- 点击链接在 IDE 中查看
- 选中需要修改的内容 → 发送给 Claude → 提交修改意见

## 工作流状态流转

```text
not-started ──sdd:new / sdd:spec──► requirements (pending)
requirements (pending) ──Approve──► design (pending)
design (pending) ──Approve──► tasks (pending)
tasks (pending) ──Approve──► implementation (in_progress)
implementation ──全部完成──► sdd:save → commit
```

任何阶段都可以使用 `/adk:sdd:clarify` 来完善制品并级联更新已有下游制品。

使用 `/adk:sdd:revert <phase>` 可以回退到指定阶段重新开始。

## 中断与恢复

工作流设计足够健壮，支持随时中断：

| 场景 | 操作 |
|------|------|
| 模型跑偏或没有继续运行 | 手动 interrupt 后输入"继续/continue" |
| 模型降智幻觉严重 | `/clear` + `/adk:sdd:continue {spec-name}` |
| 模型服务挂了 | 等待恢复后 `/adk:sdd:continue {spec-name}` |
| 想重启 session | `/clear` + `/adk:sdd:continue {spec-name}`，工作状态不会丢失 |

所有工作流状态持久化在 MCP 工作流目录中，不依赖对话上下文。

## Spec 制品目录结构

MCP 工作流目录：

```text
.ttadk/.adk-mobile/
├── config.toml                     # 个性化配置
├── specs/
│   └── <spec-name>/
│       ├── spec.md                 # 从 PRD 生成的初始 Spec
│       ├── requirements.md         # Phase 1 产出
│       ├── design.md               # Phase 2 产出
│       ├── explore.md              # Phase 2 探索性调研（可选）
│       ├── tasks.md                # Phase 3 产出
│       └── events.jsonl            # 事件审计日志（append-only）
├── user-templates/                 # 业务定制模版
├── user-hooks/                     # 阶段钩子
└── user-knowledge/                 # 业务知识
```

通过 `/adk:sdd:save` 保存到代码仓库后：

```text
<module-path>/specs/<spec-name>/
├── spec.md
├── requirements.md
├── design.md
├── tasks.md
└── explore.md
```

## 业务定制

业务可以通过以下目录（相对于 `.ttadk/.adk-mobile/`，由 `config.toml` 中 `userCustomDir` 配置）进行定制：

### user-templates/

定制中间产物模版（requirements/design/tasks），覆盖内置模版。

### user-hooks/

在不同阶段（requirements、design、tasks、implementation）的开始和结束节点插入执行业务流程。

### user-knowledge/

加载业务特有的知识库文档，补充通用知识之外的领域知识。

## 知识库建设

### 为什么需要知识库

移动端 Monorepo 通常有几十甚至上百个模块。结构化的模块文档帮助 LLM：
- 从业务术语定位相关代码（domain.md → PRD/TRD 生成）
- 理解模块间接口关系（interface.md → 多模块改动时提供横向视角）
- 理解上下游节点和影响范围（workflow.md → TRD/代码生成）
- 保证生成代码的质量和一致性（rule.md → 代码生成）

### 建设路径

```text
1. 评估就绪度    /adk:readiness [module-path]
2. 分析结构      kb-init-docs --analysis <top-level-path>
3. 生成文档      kb-init-docs <doc-root>
4. 人工 Review   按 CLAUDE.md 文件规范审查并调整
5. 校验质量      kb-docs-validator <doc-root>
6. 创建评测      kb-evals-creator
7. 基准测试      kb-docs-benchmark
8. 持续维护      kb-update-docs <doc-root>
```

### 模块文档标准结构

```text
<Module>/
├── CLAUDE.md               # 模块概览、关键类、文档引用（入口文件）
├── AGENTS.md               # → CLAUDE.md 的符号链接
└── docs/
    ├── interface.md         # 外部接口和公共 API（< 100 行）
    ├── workflow.md          # 业务流程（可含 UML 图）
    ├── domain.md            # 业务术语和领域知识
    ├── rule.md              # 代码规范、设计模式、约定
    └── evals/
        └── evals.json       # Q&A 评测用例
```

## 最佳实践

### Spec 质量

- **前期投入 Spec 描述**：描述越清楚，需求设计生成质量越高，交互时间省下来。
- **推荐使用 AI Technical Spec 模版**：系统化描述需求，避免大量时间花在对话纠偏上。
- **文件清单要准确**：`[NEW]`/`[MOD]`/`[REF]` 标记帮助 AI 理解改动范围。

### 审批流程

- **务必仔细 Review 每个阶段的产出**：Requirements 和 Design 是后续质量的基础。
- **有问题及时用 clarify 修补**：比等到 Implementation 阶段再改成本低得多。
- **开启 codeReview**：对于复杂需求，建议开启分 task Review 代码。

### 上下文管理

- **命令之间使用 `/clear`**：工作流命令可重入，从 Spec 文件读取而非对话历史。
- **降智时 `/clear` + `/adk:sdd:continue`**：清理上下文后继续，工作状态不丢失。

### 知识库建设

- 从 readiness 最低分的模块开始建设文档。
- `kb-init-docs --analysis` 自动发现需要文档的模块。
- 定期 `kb-update-docs` 保持文档与代码同步。
- `<nay-ai>...</nay-ai>` 标记保护人工编写的内容不被自动更新覆盖。
- 用 `kb-docs-benchmark` 量化文档改进效果。

### 版本管理

| 路径 | 是否提交 | 说明 |
|------|---------|------|
| `.ttadk/` | 是 | 团队共享配置 |
| `CLAUDE.md` / `AGENTS.md` | 是 | AI 工具规范 |
| `docs/` | 是 | 模块知识库文档 |
| `.mcp.json` | 建议 | MCP 配置 |
| `.ttadk/.adk-mobile/specs/` | 否 | MCP 工作流临时目录 |
| 已 save 的 specs/ | 建议 | 随 MR 合入沉淀 |

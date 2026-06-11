# 全栈功能开发 Workflow（开发域）

- 日期：2026-06-11
- 目标：把"一个功能从想法到落地"的端到端编排顺序固化为标准流程，回答"哪个 skill 在什么阶段触发、产出什么交付物"
- 输入边界：基于本仓库 60+ 个 `spec / plan / tasks` 三件套的真实历史沉淀；**仅覆盖开发域，不含部署**
- 与既有文档的关系：
  - 向上承接 [layered-dev-methodology](./2026-06-11-layered-dev-methodology.md)（载体分层：一个动作该用哪类载体）
  - 向下调用 [RUNBOOK](../sdd/RUNBOOK.md)（多任务并行执行协议）
  - 本文解决的是第三层：**端到端的阶段编排顺序**，是前两者之间缺失的一层

---

## 0. 适用范围

本 workflow 面向**需要落库、多步骤实现、跨前后端协作或需要 SDD 的开发任务**。

**不适用**（与 [RUNBOOK](../sdd/RUNBOOK.md) 适用范围一致）：
- 一次性问答、纯分析、纯评审
- 简单文案、单文件无行为变更的样式微调
- 无需代码或文档落库的小任务

对不适用任务走**轻量通道**：可跳过 spec/plan/知识沉淀，只需在提交信息或对话里记录决策即可。下文“不可跳”约束仅在本 workflow 适用范围内生效。

---

## 1. 为什么需要这一层

`layered-dev-methodology` 回答"一个动作用 Knowledge/Skill/Script/Workflow/MCP 哪种载体"，`RUNBOOK` 回答"多任务并行怎么保证状态一致与防回退"。但两者都没回答一个更上层的问题：

> 一个新功能进来，从需求到闭环，**各个 skill 应该按什么顺序触发、每步产出什么、什么时候能跳过**？

没有这层编排，反复出现的浪费是：
- `brainstorming` 出了 spec 直接跳进开发，漏掉 `writing-plans`，导致没有 tasks 拆分和 write set 声明，sdd-run 失去防回退基础
- 前端先写死再补后端契约，导致 mock 与真实后端语义漂移、联调返工
- 功能闭环后经验不沉淀，下次重复踩同样的坑

---

## 2. 第一性原理：契约是前后端的单一事实源

整个 workflow 的主干形状由一条原则决定：**接口契约优先于实现，且契约是前后端共同派生的单一事实源（SSOT）**。

依据：
- `AGENTS.md` 硬约束："接口契约变更必须同步更新前端、后端与文档"
- `RUNBOOK` 任务拆分规则："数据模型和接口契约优先于实现"

由此推出：需求语义/UI 稳定后**先定契约**，前端按契约对 mock、后端按契约实现，两端并行而不返工。契约不一致的根因是"契约被前后端各自解读产生漂移"，因此解法是源头治理（契约 SSOT + 冻结闸门 + 分档校验），而非末端加联调关口。

---

## 3. 标准流程（7 阶段）

```text
阶段  名称        触发载体              交付物                         可跳过
[0]  需求澄清     brainstorming        spec.md                        适用范围内不可跳
[1]  UI 设计      ui-design-trio (等)   选定 UI 稿                      无 UI 改动时跳过
[2]  契约先行     brainstorming(轻)     契约 SSOT (见 3.2 必填项)        无接口变更时跳过
[3]  计划拆分     writing-plans        plan.md + tasks.md (+checklist)  适用范围内不可跳
[4]  前端波次     sdd-run              前端实现 + TDD 证据 (按契约)      无前端改动时跳过
[5]  后端波次     sdd-run              后端实现 + TDD 证据 (按契约)      无后端改动时跳过
[6]  知识沉淀评估  knowledges-update    更新知识树或记录 no-op           适用范围内评估不可跳
```

> 阶段 4、5 默认对应前端/后端两类实现波次（task groups），但**不强制先前端后后端**；具体串并行由 [RUNBOOK](../sdd/RUNBOOK.md) 的 `Wave Plan` 与 `Parallel Group` 依据契约冻结状态、依赖、write set、本地服务占用决定。

### 3.0 阶段 0 — 需求澄清（brainstorming）

- 探明动机、约束、成功标准；目标不清晰时停下来讨论
- 终态是产出 `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md` 并提交
- **适用范围内不可跳过**：再"简单"的开发任务也要有一份可被评审的设计（轻量通道任务除外）

### 3.1 阶段 1 — UI 设计（ui-design-trio 及其他 UI skill）

- 基于 spec 产出 2-3 个可对比的 UI 变体，用户选定
- 纯后端功能、无界面改动时跳过

### 3.2 阶段 2 — 契约先行（轻量 brainstorming）

- 触发时机：需求语义或 UI 交互稳定后即可，**无 UI 的纯后端能力同样适用**
- 基于需求/UI 推导出接口契约，写成单一事实源（SSOT），前端 mock 与后端实现都引用同一份

**Definition of Contract SSOT（最小合格项，缺一不算 SSOT）：**
- `path` + `method`：必须经 `gateway-service` 的 `/api/v1` 入口，禁止前端直连子服务
- `request` / `response` 字段：名称、类型、可选性、`pagination` 约定
- `auth`：是否需要 JWT，下游身份统一用网关派生的 `X-User-ID`，禁止硬编码用户身份或内部 Token
- `error`：错误码与语义
- 金额字段：后端类型必须为 `shopspring/decimal`，JSON 表达需明确字符串/数值策略；禁止 float
- 跨服务依赖：声明调用方、被调用方、RPC/API 路径与降级语义；禁止跨服务直接查库
- `owner` + 引用位置：契约文件路径，供前后端 sdd-run 引用

**落地档位：**
- **默认档（小功能）**：结构化 markdown 契约片段，落在 spec 内或独立契约文件
- **进阶档（核心/跨服务）**：升级为 OpenAPI 片段或共享 types，前端 mock 从契约派生，从机制上消除漂移（参考 [openapi-sdk-design](./2026-05-30-live-auction-openapi-sdk-design.md)）

**一致性保证（分档，不可只靠 grep）：**
- `grep -R "<api-path-or-field>" -n frontend backend docs` 是 regression sentinel 的**最低档**，只能证明字符串存在，**不能**证明字段可选性、错误码语义、鉴权、金额精度、序列化兼容
- 核心/跨服务接口必须额外配至少一种：OpenAPI 校验 / type check / API 测试 / 前后端 mock parity
- 无接口变更时跳过

### 3.3 阶段 3 — 计划拆分（writing-plans）

- **开发型 brainstorming 进入 SDD 前的前置条件是 writing-plans**，不允许从 spec 直接跳到 sdd-run（纯讨论/纯文档类 brainstorming 不受此约束）
- 产出 `plan.md` + `tasks.md`；`checklist.md` 是可选独立产物，无独立 checklist 时可把验收清单内嵌进 tasks 或 state 文件
- 任务按依赖拆分、声明 write set / read set / regression sentinel
- 拆分形态由需求规模决定（见第 4 节）

### 3.4 / 3.5 阶段 4、5 — 实现（sdd-run）

- 阶段 4/5 默认对应**前端波次 / 后端波次**两类 task groups，但不固定先后；串并行由 [RUNBOOK](../sdd/RUNBOOK.md) 的 `Wave Plan` + `Parallel Group` 依据契约冻结状态、依赖、write set、本地服务占用决定
- **进入 sdd-run 前必须先创建或续用 state 文件**：运行 `python3 docs/superpowers/sdd/scripts/sdd_run.py`，无既有 state 时从 `docs/superpowers/sdd/state-template.md` 创建；以 state 的 branch/worktree/commit 为执行事实源
- 各波次按 RUNBOOK 协议执行：隔离 worktree → **先写失败测试 → 最小实现 → 验证通过** → 回归 sentinel → review；TDD red-green 证据写入 state
- 前端按契约对 mock；后端按契约实现，并跑契约一致性检查（见阶段 2 分档）
- 本地排障遇到 `localhost`/IPv6/旧进程/端口占用时，先清理旧进程、核对运行事实，**不得为绕过本机问题修改主干配置**（如把统一 `localhost` 改成 `127.0.0.1`）
- 对应端无改动时跳过；后端逻辑直观时无需额外设计讨论，但 [3] writing-plans 拆 tasks 不可跳

### 3.6 阶段 6 — 知识沉淀评估（knowledges-update）

- 功能闭环后**适用范围内必须评估是否需要沉淀**（评估动作不可跳，轻量通道任务除外）
- 有新增约束 / 经验 / 决策 → 更新 `.trae/knowledges/**/SKILL.md`，必要时更新 memory topics
- 无 durable knowledge（一次性实现细节）→ 记录 no-op 结论，不写入知识库，避免召回噪音
- 持久化主目标是 `.trae/knowledges/**/SKILL.md`；memory 仅在需要跨会话续接时更新

---

## 4. 规模驱动的两种拆分形态

阶段 3-5 的组织形态由需求规模决定：

### 4.1 默认形态（小/中功能）：一份计划拆两组 tasks

```text
brainstorming → ui-design-trio → 契约(默认档) → writing-plans(前+后两组 tasks)
                                              ├─ sdd-run 前端波次
                                              └─ sdd-run 后端波次
```

- 前后端共享同一份 plan 与契约，上下文最连贯
- 适用：单服务内、字段/页面级改动

### 4.2 大需求形态：前后端各自独立走一轮

```text
brainstorming → ui-design-trio → 契约(进阶档)
   ├─ 前端: writing-plans → sdd-run
   └─ 后端: writing-plans → sdd-run
```

- 前端、后端各自独立完成 writing-plans + sdd-run，隔离更彻底
- 适用：跨服务、涉及数据模型变更、需要并行多 subagent 的大功能
- 代价：上下文需各自重述，但换来更强的隔离与并行度

判据（清单式，命中任一即走大需求形态，否则默认形态）：
- 涉及**跨服务**调用或数据契约
- 涉及**数据模型 / DB migration**变更
- 需要**并行多 subagent** 或存在共享 write set 需隔离
- 接口/页面数量多、状态机复杂、兼容性风险高，单份 plan 难以承载

> 不引入打分制：清单命中即升级，保持判定最省路径。

### 4.3 契约冻结闸门（两种形态共用）

前后端并行的前提是**契约已冻结**。规则：
- 阶段 2 契约确认后标记为 frozen，前后端波次才能并行进入 sdd-run
- 开发中若必须改契约：先更新契约 SSOT → 在 state 文件的 `API Contract Changes` 与 `Cross-Task Decisions` 表登记变更 → 受影响波次在 `Wave Plan` 中重新对齐其 Start Condition → 再继续
- 禁止任一端私自改契约后继续，否则产生二次漂移

---

## 5. 最小闭环表（按任务类型）

判定顺序：**先判 contract impact，再判 UI / frontend / backend impact。**

| 任务类型 | 必经阶段（最小闭环） | 可跳过 |
|---|---|---|
| 跨端功能（有 UI + 接口变更） | [0][1][2][3][4][5][6评估] | — |
| 纯前端功能，无接口变更 | [0][1][3][4][6评估] | [2] 契约、[5] 后端 |
| 纯前端功能，但有接口变更 | [0][1][2][3][4][5][6评估] | — |
| 纯后端能力（无 UI） | [0][2][3][5][6评估] | [1] UI、[4] 前端 |
| 内部重构 / 纯样式，无接口与行为变更 | 走**轻量通道**（见第 0 节） | spec/plan/知识沉淀 |

注：后端逻辑直观时无需额外设计讨论，但 [3] writing-plans 拆 tasks 不可跳。

---

## 6. 反模式

| 反模式 | 后果 | 正确做法 |
|---|---|---|
| 开发型 brainstorming 后直接 sdd-run | 无 tasks 拆分与 write set，sdd-run 失去防回退基础 | 必经 writing-plans |
| 前端先写死再补后端契约 | mock 与真实后端语义漂移、联调返工 | 契约先行，前后端共享 SSOT |
| 契约只写在文档里靠人/LLM 各自解读 | 契约漂移 | 契约 SSOT，在 plan/tasks 中声明契约文件路径与 sentinel |
| 只用 grep 当契约一致性保证 | 假安全感，漏掉语义/鉴权/精度问题 | 核心接口配 OpenAPI / type check / API 测试 / mock parity |
| 末端加重型联调关口救漂移 | 末端挽救、治标 | 源头用契约 SSOT + 冻结闸门消除漂移 |
| 并行中任一端私改契约 | 二次漂移 | 改契约必须先更新 SSOT 并广播阻塞 |
| 无 durable knowledge 也强写知识库 | 召回噪音 | 阶段6 先评估，无新增则记 no-op |
| 把大需求塞进一份 plan | 上下文过载、并行度低 | 大需求前后端各自独立走一轮 |

---

## 7. 与现有资产的关系映射

| 阶段 | 触发载体 | 对应仓库资产 |
|---|---|---|
| [0] 需求澄清 | brainstorming | `docs/superpowers/specs/*-design.md` |
| [1] UI 设计 | ui-design-trio | `docs/superpowers/specs/ui/*`、`2026-06-09-ui-design-trio-skill-design.md` |
| [2] 契约先行 | brainstorming(轻) | `2026-05-30-live-auction-openapi-sdk-design.md`、契约对齐类 spec |
| [3] 计划拆分 | writing-plans | `docs/superpowers/plans/*.md`、`*-tasks.md` |
| [4][5] 实现波次 | sdd-run | `docs/superpowers/sdd/RUNBOOK.md` + `runs/*-state.md` |
| [6] 知识沉淀评估 | knowledges-update | `.trae/knowledges/**/SKILL.md`（主）+ memory topics（按需） |

---

## 8. 一句话结论

全栈功能 workflow 的本质编排：**brainstorming 出 spec → UI 定稿 → 契约先行（带必填项的 SSOT）→ writing-plans 拆 tasks（开发型 brainstorming 进 SDD 的前置条件）→ sdd-run 实现波次（TDD）→ knowledges-update 评估沉淀。** 默认一份计划拆前后端两组 tasks；命中跨服务/数据模型/多 subagent 等清单项的大需求则前后端各自独立走一轮。契约一致性靠源头 SSOT + 冻结闸门 + 分档 sentinel 治理，而非只靠 grep 或末端联调。轻量任务走轻量通道，不被完整流程绑架。

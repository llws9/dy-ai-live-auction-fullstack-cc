# 全栈开发分层方法论

- 日期：2026-06-11
- 目标：把本仓库已散落验证的 `Knowledge / Skill / Script / Workflow / MCP` 抽象成一套统一的"载体分层操作系统"，回答"哪类问题该由哪类载体承接"
- 输入边界：基于本项目历史会话沉淀的真实资产，结论优先服务跨项目复用
- 与既有文档的关系：本文是"上层方法论"，向下统辖 `docs/superpowers/specs/2026-06-11-cross-project-mcp-opportunity-map-design.md` 的 MCP 边界判据，以及 `docs/superpowers/sdd/RUNBOOK.md` 的执行协议

---

## 1. 为什么需要这一层

本仓库已经积累了大量有效资产，但它们分散在不同位置、用不同方式触发：

- `.trae/knowledges/**/SKILL.md`：模块知识
- `.agents/skills/**/SKILL.md`：可触发技能
- `docs/superpowers/sdd/scripts/*.py`、`docs/superpowers/runtime-facts/*.py`：确定性脚本
- `docs/superpowers/sdd/RUNBOOK.md`：多任务执行协议
- `mcp_GitHub`、`integrated_browser`：已装 MCP

问题不在于"缺资产"，而在于**缺一层统一判据**：一个新动作进来，应该写成知识、技能、脚本、纳入 workflow，还是产品化成 MCP？没有这层判据，就会反复出现两种浪费：

- 把"该编排的推理"硬塞进脚本或 MCP，导致僵化
- 把"该固化的确定性事实"留在 prompt 里反复让 LLM 临场解析，导致翻车

本方法论就是把这条判据显式化、可复用。

---

## 2. 第一性原理：按"能力形态"分层，而不是按技术域

一个动作的本质形态，决定了它该由哪类载体承接。判断维度只有两条根问题：

1. **它的核心价值是"理解/决策/解释"，还是"取事实/执行确定路径"？**
   - 前者偏推理，归 `Skill`
   - 后者偏确定性，归 `Script` / `MCP`
2. **它的输入输出是否稳定、是否需要被多方稳定复用、是否需要屏蔽跨平台差异？**
   - 否：留在 `Script`
   - 是：上移为 `MCP`

横切于两者之上的是 `Workflow`（编排多个载体的协议）和 `Knowledge`（提供背景约束）。

---

## 3. 五层载体模型

### 3.1 Knowledge（知识层）

- **承接什么**：模块背景、约束、领域规则、"这块代码为什么这样"
- **本质**：被动召回的上下文，不主动执行动作
- **本仓库实例**：`.trae/knowledges/backend/auction/SKILL.md`、`.trae/knowledges/frontend/h5/SKILL.md` 等渐进式知识树
- **典型关键词**：`context`、`constraint`、`convention`、`domain rule`
- **判据**：当价值是"让任何后续动作不犯已知错误"，而不是"执行某个动作"

### 3.2 Skill（技能层）

- **承接什么**：理解意图、做取舍、组织步骤、解释结果、归因
- **本质**：一段 prompt，**指挥 agent 用已有工具**完成任务；不提供新原子能力，只编排能力
- **本仓库实例**：`brainstorming`、`writing-plans`、`sdd-run`、`ui-design-trio`、`runtime-facts`
- **典型关键词**：`plan`、`review`、`decide`、`orchestrate`、`handoff`
- **判据**：核心价值是推理与编排；底层工具已存在，只缺调用顺序和判断逻辑

### 3.3 Script / Command（脚本层）

- **承接什么**：单项目内某条已知确定路径的执行
- **本质**：确定性代码，承载环境细节与命令实现，输入输出明确但**未必需要跨项目稳定接口**
- **本仓库实例**：`docs/superpowers/sdd/scripts/sdd_run.py`、`docs/superpowers/runtime-facts/runtime_facts.py`
- **典型关键词**：`restart`、`deploy`、`seed`、`bootstrap`、`collect`
- **判据**：路径确定、可测试，但复用面还局限在本仓库；是 MCP 的"前身验证形态"

### 3.4 Workflow（编排协议层）

- **承接什么**：把多个 Skill / Script / 载体按固定协议串成可复用的执行流程
- **本质**：跨载体的执行契约 + 状态 SSOT，降低"手动调用多个 skill"的心智负担
- **本仓库实例**：`docs/superpowers/sdd/RUNBOOK.md`（`Spec -> Tasks -> Worktree -> Subagent -> TDD -> Verify -> Review -> Handoff`）+ 状态文件机制
- **典型关键词**：`runbook`、`state SSOT`、`wave`、`definition of done`
- **判据**：单个 Skill 解决不了"多任务并行 + 状态一致性 + 防回退"，需要协议而非更长的 prompt

### 3.5 MCP（工具接口层）

- **承接什么**：把高价值、强结构化、可复用的能力暴露成稳定工具接口
- **本质**：提供新的原子工具，稳定入参 + 结构化返回，任何 agent/skill 可直接调
- **本仓库实例**：`mcp_GitHub`、`integrated_browser`
- **典型关键词**：`query`、`verify`、`inspect`、`diff`、`bridge`、`probe`
- **判据**（须同时满足，源自 MCP 机会图 spec）：
  1. 输入结构稳定，可参数化
  2. 输出是事实或受控动作结果，而非主观结论
  3. 需要被多个 Skill / Agent 复用
  4. 手工解析自由文本易错
  5. 需要屏蔽跨 OS / 跨平台差异

---

## 4. 载体选择决策树

一个新动作进来，按以下顺序判断落点：

```text
这个动作的核心价值是什么？
├─ 提供背景约束，不执行动作        -> Knowledge
├─ 理解 / 取舍 / 解释 / 归因 / 编排 -> Skill
└─ 取事实 / 执行确定路径
   ├─ 仅本项目、路径确定、暂不需稳定接口 -> Script
   ├─ 需串联多个载体 + 状态一致性 + 防回退 -> Workflow
   └─ 同时满足 MCP 五条判据
      ├─ 现有 MCP/Skill 已覆盖 ≥70%   -> 不新建，复用或做编排 Skill
      └─ 现有能力覆盖不足              -> MCP 候选（仍建议 Script 先验证）
```

关键护栏（来自二次核实的教训）：

- **识别 MCP 机会前，必须先核对环境已有能力**（`trace-query`、`argos-log`、`mcp_GitHub` 等），防止重复造轮子。
- **能用 Skill/Script 低成本验证的能力，先跑出复用证据，再承担 MCP 开发维护成本。**

---

## 5. 层间升降级：载体不是静态归属

载体分层不是一次性归类，而是随"复用证据"流动。最重要的一条通道是 `Script -> MCP`：

```text
Skill 临场拼命令(高翻车)
      │ 发现确定性子动作
      ▼
Script(可测试、确定性)  ← 当前 runtime_facts.py 所处位置
      │ 出现以下任一信号
      │  1. 跨项目复用(≥2 个项目)
      │  2. ≥2 类调用者依赖(如部署验证 + 排障)
      │  3. LLM 解析 shell 文本出过误判
      │  4. 输出 schema 已稳定
      ▼
MCP(稳定接口、结构化返回、跨平台封装)
```

反向降级同样合法：如果一个 MCP 长期只服务单项目、只有一个调用者，说明它本不该上移，应回退为 Script。

**核心论断**：MCP 不是"更高级的 Skill"，而是"被验证过、值得固化成稳定接口的 Script"。

---

## 6. 反模式

| 反模式 | 后果 | 正确做法 |
|---|---|---|
| 把架构取舍、code review 结论塞进 MCP | 把推理伪装成工具，僵化且失真 | 留在 Skill |
| 把确定性事实采集留在 prompt 里反复解析 | 高翻车，结论不可复现 | 下沉为 Script，达标后 MCP 化 |
| 新动作直接上 MCP，跳过验证 | 为未证明高频的能力先付维护税 | Script 先行，复用达标再上移 |
| 不核对现有能力就立项 MCP | 与 `mcp_GitHub`/`trace-query` 等重叠，能力碎片化 | 先盘点现有载体 |
| 用更长的 prompt 解决多任务一致性 | 状态丢失、任务重复/遗漏 | 上升到 Workflow + 状态 SSOT |

---

## 7. 本仓库资产的分层映射

| 载体 | 本仓库实例 | 当前状态 |
|---|---|---|
| Knowledge | `.trae/knowledges/**/SKILL.md` | 已建渐进式知识树 |
| Skill | `brainstorming` / `writing-plans` / `sdd-run` / `ui-design-trio` / `runtime-facts` | 成熟 |
| Script | `sdd_run.py` / `runtime_facts.py` | 成熟；`runtime_facts.py` 是待验证的 MCP 前身 |
| Workflow | `docs/superpowers/sdd/RUNBOOK.md` + 状态文件 | 成熟，已是默认执行协议 |
| MCP | `mcp_GitHub` / `integrated_browser` | 已装；新增 MCP 须过第 4 节决策树 |

---

## 8. 一句话结论

分层方法论的本质是一条判据：**按"推理 vs 确定性"决定 Skill 还是 Script/MCP，按"复用证据"决定 Script 是否上移为 MCP，按"多任务一致性需求"决定是否上升为 Workflow，并在立项任何新载体前先盘点现有能力。** 这套判据已被 `环境事实查询`（Script 先行、达标再 MCP 化）这一真实案例验证。

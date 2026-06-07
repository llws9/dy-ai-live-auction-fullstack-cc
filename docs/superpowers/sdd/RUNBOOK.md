# Project SDD Runbook

本 Runbook 用于把 `Spec -> Tasks -> Worktree -> Subagent -> TDD -> Verify -> Review -> Handoff` 固化为本项目的标准执行流程。

## 目标

- 降低手动调用多个 skill 的心智负担。
- 让多 agent 并行执行时仍有统一的任务状态 SSOT。
- 保证每个子任务都有一致的 Definition of Done。
- 避免聊天上下文丢失导致任务重复、遗漏或验证证据缺失。

## 适用范围

- 适用于有 `spec.md`、`tasks.md`、`checklist.md` 或明确任务清单的开发工作。
- 适用于需要 SDD、TDD、并行 subagent、阶段性 code review 的任务。
- 不适用于一次性问答、简单文案修改、无需代码或文档落库的小任务。

## 标准入口

优先使用项目级 slash command：

```text
/sdd-run 这是本次开发的plan：<plan-path> 和 task：<tasks-path>，开始执行
```

也支持空输入，由脚本安全推断上下文：

```text
/sdd-run
```

可选参数：

- `scope: <task-id-or-wave>`：只执行指定任务或波次。
- `state: <state-file-path>`：继续已有 SDD 状态文件。
- `target branch: <branch>`：本次最终集成目标，默认 `main`。
- `mode: subagent-driven`：默认执行模式。

如果当前环境不支持 slash command，使用以下等价启动指令：

```text
请按 docs/superpowers/sdd/RUNBOOK.md 执行 SDD。
输入：
- plan: <absolute-or-repo-relative-plan-path>
- spec: <absolute-or-repo-relative-spec-path>
- tasks: <absolute-or-repo-relative-tasks-path>
- checklist: <absolute-or-repo-relative-checklist-path>
- scope: <本次执行范围>
- target branch: <本次最终集成目标，默认 main>
- mode: subagent-driven
- state: <state-file-path，如果没有则从 state-template.md 创建>
```

## 必读上下文

执行前必须读取：

- `AGENTS.md`
- `docs/CONSTITUTION.md`
- `docs/CODING.md`
- 当前任务关联的 `spec.md`
- 当前任务关联的 `tasks.md`
- 当前任务关联的 `checklist.md`
- 当前任务关联的审计或需求文档，例如 `docs/admin-api-audit.md`
- 已存在的状态文件；如果不存在，从 `docs/superpowers/sdd/state-template.md` 创建

## 分支与 Worktree

默认规则：

- 非只读任务必须在隔离 worktree 中执行。
- worktree 路径推荐：`/Users/bytedance/.config/superpowers/worktrees/dy-ai-live-auction-fullstack-cc/<branch-name>`
- 分支命名推荐：`feat/<short-topic>`、`fix/<short-topic>`、`docs/<short-topic>`
- 禁止在未确认的脏工作区上直接改动业务代码。

每次最终回答第一句必须使用固定格式：

```text
当前分支/worktree：<branch> @ <absolute-worktree-path>
```

## 状态文件

状态文件是 SDD 执行的唯一事实来源。

执行 `/sdd-run` 时，必须先运行脚本创建或续用状态文件：

```bash
python3 docs/superpowers/sdd/scripts/sdd_run.py --repo-root . --input "<用户的 /sdd-run 输入>"
```

脚本输出 JSON，后续执行必须使用其中的 `state_path`、`branch`、`worktree`。如果脚本以 code `3` 返回 `needs_selection`，展示候选并停止；其他非零退出才按错误处理，不派发 subagent。

空 `/sdd-run` 的安全推断顺序：

- 若只有一个 active 且仍有 pending 的状态文件，续用该 state。
- 否则若只有一个 plan/tasks 候选对，自动创建新 state。
- 否则输出 `needs_selection` 和候选列表，停止执行，禁止猜测。

推荐路径：

```text
docs/superpowers/sdd/runs/YYYY-MM-DD-<topic>-state.md
```

状态文件必须记录：

- Run 元信息：目标、分支、worktree、输入文档、执行模式。
- 任务矩阵：任务 ID、状态、owner、依赖、可并行性、write set、read set、regression sentinel。
- 子任务证据：测试命令、验证结果、修改文件、提交信息、运行环境来源。
- 集成证据：base commit、target branch、分支 ahead/behind、旧分支 diff review。
- 风险与阻塞：根因、影响、处理决策。
- 汇总结论：完成项、未完成项、后续动作。

状态流转：

```text
pending -> assigned -> in_progress -> verifying -> review -> done
pending -> blocked
review -> changes_requested -> in_progress
```

## 任务拆分规则

任务必须按依赖拆分，而不是按文件数量拆分。

拆分原则：

- 同一文件强相关任务串行。
- write set 有重叠的任务必须串行，即使它们修改的是同一文件的不同行。
- 同一个本地服务或 dev server 被多个任务验证使用时，必须串行或显式隔离端口/进程。
- 不同服务、不同页面、不同测试边界可并行。
- 数据模型和接口契约优先于实现。
- 测试任务必须位于对应实现任务之前。
- 每个任务都必须能独立验证。

禁止拆分方式：

- 让多个 subagent 同时修改同一文件。
- 让多个 subagent 同时拥有同一个 write set。
- 把“修所有 bug”作为一个子任务。
- 只给自然语言目标，不给文件范围、接口范围和验证命令。
- 让旧分支整文件覆盖当前目标分支来“恢复功能”。

## 防回退控制

每个实现型任务必须在状态文件中声明：

- `write set`：允许修改的文件或 glob。
- `read set`：允许读取但默认不得修改的上下文文件。
- `regression sentinel`：能抓住语义回退的测试或确定性检查。
- `runtime source`：如果使用本地服务或前端 dev server 验证，记录实际运行的 branch/worktree/commit/dirty status。

防回退规则：

- bugfix、UI 行为、接口契约、演示链路修复必须有 regression sentinel。
- 如果无法自动化 sentinel，必须先在状态文件写明原因、手动检查步骤和剩余风险。
- Git 无冲突不代表无回退；同文件或同契约的任务必须串行 review。
- 旧分支超过目标分支时效后，不允许整分支直接合入；优先 rebase 到最新目标分支，并 cherry-pick 仍需要的 commit。
- 合入前必须执行 `git diff <target>...HEAD --name-only` 或等价检查，确认没有把目标分支已有修复覆盖掉。

冲突处理规则：

- `main` 或目标分支是当前已验证基线，但不是冲突中的无条件胜者。
- 发生冲突时，默认以目标分支为基线做语义合并，再把任务分支仍然有效的优化迁移到最新代码结构上。
- 禁止在冲突文件上整文件选择 `ours` 或 `theirs`，除非在状态文件记录原因、被丢弃行为、替代方案和验证证据。
- 冲突解决后必须同时运行目标分支已有 regression sentinel 和任务分支新增 regression sentinel。
- 若任务分支没有 sentinel，合入前必须补充或记录无法自动化的确定性验证；否则无法证明优化没有被吞掉。

## 运行环境来源

浏览器、日志和本地接口验证必须绑定到明确代码来源。每次启动或复用服务时，在状态文件记录：

- 服务名。
- 启动命令。
- branch。
- worktree。
- commit。
- dirty status。
- 端口。

如果浏览器或日志指向的 worktree/commit 与当前任务 worktree 不一致，本次验证无效，必须先对齐运行环境。

原则：worktree 是任务沙盒，演示事实源应是干净的集成分支或目标分支，不应长期停留在旧任务分支。

## Subagent 派发模板

给每个 subagent 的任务说明必须包含：

```text
你正在仓库 <repo-path> 的 worktree <worktree-path> 中执行 SDD 子任务。

必须遵守：
- 读取并遵守 AGENTS.md。
- 最终回答第一句必须是：当前分支/worktree：<branch> @ <absolute-worktree-path>
- 遵循 TDD：先测试，后实现，再验证。
- 不要修改任务范围外文件。
- 不要回滚用户或其他 agent 的改动。
- 完成后先更新状态文件，再汇报。

任务输入：
- state: <state-file-path>
- task_id: <task-id>
- scope: <scope>
- files: <allowed-files>
- write_set: <files-or-globs-the-task-may-modify>
- read_set: <files-or-globs-the-task-may-read-only>
- dependencies: <dependency-task-ids>
- expected_tests: <test-commands>
- regression_sentinels: <tests-or-manual-checks-that-catch-rollback>
- runtime_services: <services/dev-servers-owned-by-this-task>
- expected_output: <acceptance-criteria>

交付要求：
- 列出修改文件。
- 列出测试命令和结果。
- 列出 regression sentinel 或替代验证证据。
- 如果使用本地服务，列出 branch/worktree/commit/dirty status。
- 列出未解决风险。
- 如果未完成，必须说明阻塞根因和下一步。
```

## TDD 子任务协议

每个实现型子任务按以下顺序执行：

1. 阅读状态文件和任务输入。
2. 阅读相关 spec、tasks、checklist 和代码。
3. 写失败测试或补充契约测试。
4. 运行目标测试，确认失败原因匹配预期。
5. 最小实现使测试通过。
6. 运行目标测试。
7. 运行受影响模块的回归测试。
8. 更新状态文件的证据区。
9. 汇报结果，第一句展示分支/worktree。

如果任务不适合写自动化测试，必须在状态文件写明原因，并提供替代验证证据。
如果任务需要修改 write set 之外的文件，必须停止并回报，由主 agent 更新任务拆分或串行顺序后再继续。

## Review 协议

主 agent 在每个波次结束后执行 review：

- 检查状态文件是否完整。
- 检查任务是否越界修改。
- 检查 write set 重叠任务是否已串行。
- 检查 regression sentinel 是否能抓住本次修复的回退。
- 检查运行环境来源是否与验证结论一致。
- 检查测试证据是否可复现。
- 检查接口契约、文档和实现是否一致。
- 对未完成项标记 `blocked` 或重新派发。

建议在以下节点触发 code review：

- 一个独立 user story 完成后。
- 一个服务或页面改动闭环后。
- 所有 P0/P1 任务完成后。
- 合并或提交 PR 前。

## 验证协议

验证命令必须按任务影响范围选择。

后端 Go：

```bash
go test ./...
```

前端 Admin：

```bash
npm test -- --runInBand
npm run build
```

前端 H5：

```bash
npm test -- --runInBand
npm run build
```

跨端接口契约：

```bash
grep -R "<api-path-or-field>" -n frontend backend docs
```

如果全量验证成本过高，必须记录：

- 已执行的最小验证命令。
- 未执行的验证命令。
- 未执行原因。
- 剩余风险。

## 完成定义

一个子任务只有同时满足以下条件才可标记为 `done`：

- 需求范围已实现或明确判定无需实现。
- 对应测试已新增或已有测试覆盖。
- 对应 regression sentinel 已新增、复用或明确记录无法自动化的替代验证。
- 修改文件没有越过 write set，或状态文件记录了批准的 scope expansion。
- 验证命令已执行并记录结果。
- 使用本地服务或 dev server 验证时，已记录 branch/worktree/commit/dirty status。
- 文档或接口契约已同步更新。
- 状态文件已更新。
- 最终回答第一句展示当前分支/worktree。

一个 SDD Run 只有同时满足以下条件才可关闭：

- 状态文件中没有 `pending`、`assigned`、`in_progress`、`verifying`、`review` 状态任务。
- 所有 `blocked` 任务都有根因、影响和下一步。
- 所有测试证据可复现。
- 主 agent 完成最终 review。
- 用户确认合并、继续下一波或归档。

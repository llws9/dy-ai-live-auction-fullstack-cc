# Runtime Facts 增强设计：从「版本对齐」到「改了不生效」诊断闭环

- 日期：2026-06-11
- 目标：把 `runtime-facts` 从「只回答运行 commit 是否等于目标 commit」增强为「回答改了为什么不生效 / 这次验证可不可信」的诊断闭环
- 输入边界：基于本仓库 06/04-06/11 共 8 天约 150 个 session 的真实痛点审计
- 与既有文档的关系：本文将 `runtime-facts` 路线统一为 **deterministic Script + project-level Skill**，并增强 `docs/superpowers/plans/2026-06-11-runtime-facts-skill-script.md`（尚未执行的 plan）；落地后需修正 `docs/superpowers/specs/2026-06-11-layered-dev-methodology.md` 中「runtime-facts 已成熟」的超前表述

---

## 1. 为什么要做这次增强

### 1.1 痛点审计结论（证据驱动，非臆测）

对最近 8 天历史会话做频次审计，本地全栈「改了不生效 / 验证无效」是**绝对高频**痛点，几乎每天复现。其根因分布如下：

| 根因 | 频次 | 现有 runtime-facts plan 能否检出 |
|---|---|---|
| A. 构建产物陈旧：Vite/Docker 缓存吃旧码，进程没重启/没 rebuild | 最高 | 否。commit 对 ≠ 跑的是新构建 |
| B. 进程 cwd 在别的 worktree | 高 | 部分（仅当 commit 不同才报） |
| C. 端口无监听 / test-service 没起 | 高 | 否，无结构化结论 |
| D. 旧进程没杀干净（同端口多监听） | 中 | 否 |
| E. 工作区 dirty 但进程跑旧 commit | 中 | 否 |

典型实例（均来自历史会话）：

- 06/07、06/09 多次：「修改不生效」根因是 Docker H5 容器缓存旧构建，需 `--build` + 浏览器强刷
- 06/09：`deploy-dev.sh` 误杀 Docker 端口代理进程
- 06/07：多 worktree 导致 gateway/前端跑旧分支代码，「服务版本不一致」
- 06/08、06/05：test-service（18090）没起导致 Demo Console 报错

### 1.2 现有 plan 的能力边界

现有 plan 的脚本只产出 2 条诊断码：

- `HEAD_DIFFERS_FROM_TARGET`：仓库 HEAD 与目标 ref 不一致
- `PROCESS_HEAD_DIFFERS_FROM_REPO`：端口进程 cwd 的 HEAD 与仓库 HEAD 不一致

二者本质都在回答**「代码版本对不对」**。但审计中最高频的 A 类痛点是**「进程是否反映了最新代码」**——commit 对、源码对，跑的还是旧构建。这是现有 plan 的真空白。

### 1.3 第一性结论

> runtime-facts 当前回答的是「运行的代码版本」，而用户被坑最多的是「运行的进程是否已吃进最新改动」。前者是版本一致性，后者是**新鲜度（freshness）**。增强的核心就是补上新鲜度这一维度。

---

## 2. 分层定位（遵循本仓库分层方法论）

严格按 `layered-dev-methodology.md` 的判据切分，避免把推理塞进脚本、把确定性留在 prompt：

- **Script 层**：只产出**确定性事实与可计算的 finding code**。所有新增码必须能从采集到的事实直接推导，不做主观判断。
- **Skill 层**：把 finding code 映射为**「根因在哪一层 + 下一步动作」**的诊断决策。这是推理，留在 skill。
- **边界不变**：脚本与 skill 都**只采集与解释，不重启、不杀进程、不 build、不改 git 状态**。给出结论后由 `/dp-dev` 接手执行。

---

## 3. Script 层增强：新增 finding codes

在现有 `consistency.findings` 基础上新增 5 条码。除 `STALE_PROCESS_BEFORE_CHANGE` 需补一条 `ps -o lstart=` 命令外，其余均可由现有已采集事实计算。

### 3.1 `NO_LISTENER_ON_PORT`（info）

- 触发：请求了 `--port`，但该端口 `processes` 为空
- 含义：服务未在该端口运行
- 事实来源：现有 `collect_port` 输出，无需新命令

### 3.2 `MULTIPLE_LISTENERS_ON_PORT`（warning）

- 触发：同一端口 `processes` 长度 > 1（去重 PID 后）
- 含义：可能存在旧进程残留 / 端口被多进程占用
- 事实来源：现有 `parse_lsof_listeners`，无需新命令

### 3.3 `PROCESS_CWD_OUTSIDE_REPO`（warning）

- 触发：进程 `cwd_inside_repo` 为 false
- 含义：该端口进程可能属于另一个项目或另一个 worktree，本次验证可能指向错误代码源
- 事实来源：现有 `cwd_inside_repo` 字段，无需新命令
- 说明：现有 plan 已采集该字段但未生成诊断码，本次补齐

### 3.4 `DIRTY_TREE_NOT_DEPLOYED`（warning）

- 触发：`repo.dirty == true` 且存在端口进程且进程 `runtime_source.dirty == false`
- 含义：工作区有未提交改动，但进程跑的是干净的已提交代码，改动大概率没生效
- 事实来源：现有 `repo.dirty` 与 `runtime_source.dirty`，无需新命令

### 3.5 ⭐ `STALE_PROCESS_BEFORE_CHANGE`（warning，本次灵魂）

- 触发：进程启动时间 **早于** 代码最近变更时间
- 含义：进程在代码改动之前就启动了，跑的是旧构建——直接命中 A 类痛点
- 判定公式：

  ```text
  stale_threshold = code_changed_at_epoch - stale_tolerance_seconds
  process_started_at_epoch < stale_threshold  ->  STALE
  其中：
    process_started_at_epoch = ps -p <pid> -o lstart= 解析后统一转换的 epoch seconds
    code_changed_at_epoch    = max(HEAD commit time, 现存 dirty runtime-input 文件的最新 mtime)
    stale_tolerance_seconds  = 默认 5 秒，可通过 CLI 参数覆盖
  ```

- 新增事实采集：
  - 进程启动时间：`ps -p <pid> -o lstart=`
  - HEAD 提交时间：`git -C <root> log -1 --format=%cI`
  - dirty runtime-input 文件最新 mtime：对 `git status --porcelain` 列出的**现存运行输入文件**取 `os.path.getmtime` 最大值；删除文件、不存在文件和非运行输入文件不参与 mtime 计算
- JSON 字段扩展：
  - 进程级新增 `started_at`（ISO8601 字符串，解析失败为 null）
  - repo 级新增 `code_changed_at`（ISO8601）
  - repo 级新增 `runtime_input_changed_at`（ISO8601 或 null），用于区分「运行输入变更」和纯文档/测试产物变更
- 运行输入文件（runtime-input files）白名单：
  - `backend/**`
  - `frontend/**/src/**`
  - `frontend/**/public/**`
  - `frontend/**/package*.json`
  - `frontend/**/vite.config.*`
  - `docker-compose*.yml`
  - `deploy/**`
  - `scripts/deploy*.sh`
  - `scripts/start*.sh`
  - `nginx*.conf`
  - `*.env.example`
- 误报控制（必须实现）：
  - 时间戳任一无法获取 → **不产出该 finding**（缺证据不臆断）
  - 容差默认 `STALE_TOLERANCE_SECONDS = 5`，CLI 暴露 `--stale-tolerance-seconds` 便于不同机器/CI 调整；测试必须显式传入固定值
  - 比较前必须把 `ps -o lstart`、`git log --format=%cI` 和文件 mtime 统一转换为 timezone-aware epoch seconds；禁止直接比较字符串或 naive datetime
  - dirty 文件只看 runtime-input 白名单，纯 `docs/**`、测试报告、构建缓存等变更不得触发 `STALE_PROCESS_BEFORE_CHANGE`
  - 跨平台：`ps -o lstart` 在 macOS / Linux 输出格式不同，解析需兼容；解析失败按「无法获取」处理，降级为不报，绝不误报

### 3.6 退出码契约（不变）

沿用现有：`0` 成功（即使 mismatch）、`2` 输入非法、`3` 依赖命令失败导致无法生成基础快照。新增码不改变退出码语义。

---

## 4. Skill 层增强：「改了不生效」诊断决策树

在现有 SKILL.md 的 `Decision Rules` 基础上，新增一段诊断决策树，把 finding code 翻译成根因层级与下一步动作。

```text
用户报「改了不生效 / 这次验证可不可信」时，采集快照后按以下顺序判定：

1. NO_LISTENER_ON_PORT
   -> 根因层：服务未启动
   -> 结论：该端口没有服务，先用 /dp-dev 启动，不是业务 bug

2. PROCESS_CWD_OUTSIDE_REPO
   -> 根因层：运行环境来源错误
   -> 结论：端口进程跑的是仓库外/别的 worktree 代码，本次验证无效，先对齐运行环境

3. STALE_PROCESS_BEFORE_CHANGE 或 DIRTY_TREE_NOT_DEPLOYED
   -> 根因层：构建/进程未刷新
   -> 结论：进程早于代码改动，跑的是旧构建；本次验证不可信。
            需重新 build/重启（前端清 Vite 缓存、Docker 加 --build、浏览器强刷），再验证。
            这不是业务逻辑 bug。

4. MULTIPLE_LISTENERS_ON_PORT
   -> 根因层：旧进程残留
   -> 结论：同端口多监听，可能命中旧进程，需先清理残留再验证

5. PROCESS_HEAD_DIFFERS_FROM_REPO / HEAD_DIFFERS_FROM_TARGET（现有）
   -> 根因层：代码版本不一致
   -> 结论：按现有规则处理

6. consistency.status == ok 且无以上 finding
   -> 结论：运行环境新鲜且对齐，验证可信，可继续排查业务逻辑
```

核心护栏：**只要命中 1-4 任意一条，先不要把现象当业务 bug 调试**，必须先消除运行环境/新鲜度问题，否则一切验证结论不可信。

### Skill 触发词扩展

在现有 `description` 触发词基础上增加：`改了不生效`、`修改没生效`、`重启了还是旧的`、`验证可不可信`、`是不是缓存`、`stale build`、`why is my change not showing`。

---

## 5. 测试策略（确定性，不依赖真实端口/Docker/网络）

沿用现有 `FakeRunner` 注入式测试。新增用例：

- `test_no_listener_emits_finding`：请求端口无进程 → 产出 `NO_LISTENER_ON_PORT`
- `test_multiple_listeners_emits_finding`：lsof 返回两个 PID → 产出 `MULTIPLE_LISTENERS_ON_PORT`
- `test_cwd_outside_repo_emits_finding`：进程 cwd 在 repo 外 → 产出 `PROCESS_CWD_OUTSIDE_REPO`
- `test_dirty_tree_not_deployed`：repo dirty + 进程 clean → 产出 `DIRTY_TREE_NOT_DEPLOYED`
- `test_stale_process_before_change`：进程 lstart 早于 code_changed_at 超过容差 → 产出 `STALE_PROCESS_BEFORE_CHANGE`
- `test_stale_not_emitted_when_timestamp_missing`：lstart 解析失败 → **不产出** stale finding（误报控制）
- `test_stale_within_tolerance_not_emitted`：进程晚于或在容差内启动 → 不产出
- `test_docs_only_dirty_change_does_not_emit_stale`：仅 `docs/**` dirty 且 mtime 晚于进程启动 → 不产出 `STALE_PROCESS_BEFORE_CHANGE`
- `test_stale_tolerance_is_configurable`：显式传入 `--stale-tolerance-seconds` 后，按传入值而非默认值判定 stale

为支持时间戳测试，`FakeRunner` 需能返回 `ps -o lstart=` 与 `git log -1 --format=%cI` 的伪输出；mtime 计算需可注入，测试中显式构造 dirty 文件路径、mtime 与 runtime-input 白名单命中情况。

---

## 6. 落地范围与文件变更

以现有未执行 plan 为落地入口，正式新增 runtime-facts script 与 project-level skill：

- 修改 `docs/superpowers/plans/2026-06-11-runtime-facts-skill-script.md`：
  - Task 1/2 的脚本与测试代码块升级为增强版（新增 5 码 + 时间戳采集 + 误报控制）
  - Task 3 的 SKILL.md 增加诊断决策树与触发词
- 新增 `docs/superpowers/runtime-facts/runtime_facts.py` 与 `docs/superpowers/runtime-facts/test_runtime_facts.py`，作为只读事实采集与确定性测试产物
- 新增 `.agents/skills/runtime-facts/SKILL.md`，作为面向 agent 的运行事实诊断协议；Skill 只解释脚本输出，不重启、不 kill、不 build、不改 git
- 落地后修正 `docs/superpowers/specs/2026-06-11-layered-dev-methodology.md`：把「runtime-facts 已成熟」改为「已设计/落地中」，消除超前表述
- 不触碰当前工作区已有的其他未提交改动（H5、知识库等）

---

## 7. 非目标（YAGNI）

- 不做 Docker 容器内构建产物 hash 比对（成本高，时间戳推断已覆盖绝大多数场景）
- 不做自动修复（重启/清缓存/build）——严守诊断与执行分离边界
- 不做 MCP 化——继续走 Script/Skill 先行验证路径，复用证据达标后再议
- 不引入第三方依赖，纯 Python stdlib

---

## 8. 一句话结论

这次增强把 runtime-facts 从「代码版本对不对」升级为「**进程是否吃进了最新改动、这次验证可不可信**」，用 `STALE_PROCESS_BEFORE_CHANGE` 等 5 条确定性 finding 在 Script 层固化新鲜度事实，用诊断决策树在 Skill 层把事实翻译成根因层级与下一步动作，直接命中 8 天审计中最高频的「改了不生效」痛点，且不越过「只诊断不执行」的边界。

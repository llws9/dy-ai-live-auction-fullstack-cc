---
name: api-mind/go_driver
description: |
  api-mind 的 Go 语言测试用例驱动。负责：
  1. Generate: 解析 case.md，生成或修改基于 apitest 运行时的 *_test.go 测试代码。
  2. Execute: 执行 go test，并在失败时根据日志自动修复代码。
  3. Report: 分析测试日志和服务端错误，生成 test_report.md。
  本模块作为底层执行器被 api-mind 中枢调度，不直接处理用户的上层意图与任务组装。
---

# Go Driver: `apitest` + `go test`

作为 `api-mind` 的 Go 语言执行驱动，本模块严格实现 Driver 接口约束，将原有的单体执行流拆解为 `generate`、`execute` 和 `report` 三个标准生命周期阶段，并针对业务仓库内的 `apitest` 运行时进行深度集成。

## 1. 接口约定与执行流 (Driver Interface)

### 1.1 Generate（用例生成）
负责前置检查、上下文收集、用例分流决策及生成可编译的测试代码。

**输入**:
- `case_file`: `case.md` 文件路径
- `output_dir`: 输出目录（生成代码和元数据的存放地），默认为`FEATURE_DIR/test/`

**执行逻辑**:
1. **§0 运行时同步**: 检查项目根目录是否为 Go Module（执行 `go list -m` 获取当前模块路径），并比对 `tests/integration/apitest/` 与基准 `runtime/` 目录的清单。必要时自动同步 `apitest` 运行时代码，并确保集成目录通过基础编译检查。
2. **§2 上下文解析**: 
   - 读取 `case_file`，关联获取 `spec.md` 与 `test/task.md`。
   - 生成本地环境文件（`.env`），来源为`test/task.md`，参考`SKILL_DIR/resources/env_template.md`。
3. **§3 用例分流 (Triage)**:
   - 智能判定每一个测试目标的决策结果：**复用 (Reuse)**（已有且无需修改）、**修改 (Amend)**（针对增量修改现有测试），还是 **新建 (New)**（从零生成），参考`SKILL_DIR/resources/reuse_amend_guide.md`。
   - 探查当前工作区改动与基线分支，生成/更新 `triage.yaml`，参考`SKILL_DIR/resources/reuse_amend_guide.md`。
4. **§4 代码生成与校验**:
   - **载荷构造**: 解析目标 IDL（本地或在线）与契约，严格区分字段类型（显式传参、资源引用、防重名动态构造、环境配置）。
   - **Mock注入**: 如果用例包含 Bytemock 配置，则执行 `mock.md` 中的 §1（读取并解析 `case.md` 的 `### Mock Setup`）+ §3（生成 dry-run 计划：`runtime_metainfo_by_case` + `skipped_cases`），并将 `runtime_metainfo_by_case` 注入到每个生成的 `Step.RpcContext` 中
   - **代码落地**: 根据分流决策，强制遵循 `SKILL_DIR/resources/go_test_template.md` 规范生成 `*_test.go`。
   - **编译检查**: 必须执行 `GOWORK=off go test -run '^NoMatch$' -count=1 <package>`。如果编译报错，允许尝试修复不超过 3 次。

**输出**:
- `code_file`: 生成或修改的测试代码文件及包路径信息。
- `status`: `success` / `failed`。
- `error`: 如果失败，提供的具体链路错误信息。

---

### 1.2 Execute（用例执行）
负责运行前安全检查、Mock 注入、实测发包及失败后的安全自愈修复。

**输入**:
- `code_file`: 需执行的代码路径或包配置（由 Generate 提供）
- `work_dir`: 工作目录
- `auto_fix`: 是否开启自动修复 (默认 true)
- `max_retries`: 最大重试次数 (默认 3)

**执行逻辑**:
1. **§5 风险拦截 (Risk Gate)**:
   - 检查写操作类型接口（POST/PUT/DELETE 等）是否运行在安全的隔离环境（如 BOE / localhost）。如处于高危环境且无用户白名单豁免，终止执行。
2. **§6 Mock 协调**: 
   - 如果用例包含 Bytemock 配置，执行 `mock.md` 中的 §2（安装 api-mock 技能）+ §4（准备 Bytemock 前置资源）+ §5（协调 permanent rules）。
3. **§7 触发测试**:
   - 处理依赖鉴权（如通过 `user_jwt` 技能获取 PaaS GW Token并写入本次运行变量）。
   - 清理旧版日志，执行命令：`GOWORK=off go test -v -count=1 -run ...`。
   - 确保生成的日志定向写入 `$APITEST_LOG_DIR/apitest_<case_id>.log`。
4. **§8 安全自愈修复 (Safe Repair)**: 
   - 针对非 0 状态退出的测试，且 `auto_fix=true`，解析当前测试日志归类错误。
   - **仅限修复用例级别的契约错误**（如断言 JSONPath 写错、响应 Wrapper 取层错误、IDL 结构更新导致的字段拼写错误）。禁止放宽或删除业务预期断言、禁止篡改源逻辑代码。
   - 修复后自动重跑，并更新重试次数。

**输出**:
- `status`: `success` / `failed`。
- `results`: 包含日志目录绝对路径和执行状态摘要。
- `fix_history`: 所有的修复历史记录与核心改动点。
- `error`: 最终失败原因（如有）。

---

### 1.3 Report（报告生成）
负责聚拢各项证据（测试用例元数据、运行时日志、服务端错误），输出 Markdown 格式的综合测试报告。

**输入**:
- `execute_results`: Execute 阶段返回的结果信息和日志目录
- `output_dir`: 报告输出存放目录

**执行逻辑**:
1. **§9 分析与组装**:
   - 读取代码 `WithCaseID(...)` 元信息、`apitest_<case_id>.log` 证据，以及 `triage.yaml` 中的分类状态。
   - **服务端日志深度分析**: 对于失败/Error 等级的用例，提取请求中的 `LogID`，尝试调用调试工具（如 `bam-cli api-test ...`）抓取服务端分析结论。抓取成功时，仅提取【问题分析结论】的精要内容补充至报告中，以提供研发视角的修复建议。
   - 按 `resources/test_report_guide.md` 标准，将内容渲染至 `test_report.md`（覆写模式）。

**输出**:
- `report_file`: 最终的 `test_report.md` 生成路径。
- `status`: `success` / `failed`。
- `error`: 生成阶段错误（如有）。

---

## 2. 核心决策契约 (Decision Contracts)

### 2.1 IDL 与接口形态获取
- **全新接口 (New API)**: 优先读取 `erd.md` -> 本地特性分支 `conf/.idl/` -> 线上平台查询（`bam-api` skill）。
- **存量接口 (Existing API)**: 优先查找本地 `conf/.idl/` -> 线上平台查询（`bam-api` skill）。
- **注意**：未找到任何定义时，触发审计记录并请求用户确认，**严禁使用假想或推测的字段拼凑接口模型**。

### 2.2 请求数据构造策略 (Payload Construction)
区分字段类型并应用专用填充策略：
- **明确字段 (explicit)**: 严格以 `case.md` 为准，如遇未赋值字段，触发审计记录并请求用户确认。
- **资源引用 (resource_ref)**: (如需传入真实有效的 `policy_id`)，必须调用提供方的接口查询提取，或遵循知识库 (KB) 规范。严禁随意传递 `[0]` 或写死假 ID。
- **动态构造 (dynamic_construct)**: 依赖每次运行刷新，防止唯一约束冲突的字段（如命名名称、创建时间戳等）。**必须**调用 `apitest` Helpers 函数（如 `UniqueName` / `NowMilli` / `RandString`）。
- **环境隔离业务参数 (env_business_sample)**: 必须使用包级别的 `envSamples` 字典和 `apitest.Sample` 方法来动态加载，以便不同运行环境平滑切换。

### 2.3 复用/修改/新建 决策 (Reuse / Amend / New)
- **New (新建)**: 增量业务且无基准覆盖，使用模板文件全新生成。
- **Amend (修改)**: 有基准覆盖且需因需求变更调整。采用**最小文本 Diff 修改策略**，只变更目标范围，保留与当前需求无关的用例上下文。
- **Reuse (复用)**: 目标代码已完全涵盖测试诉求无需更新，仅记录标记并一同参与运行。

---

## 3. 严格底线约束 (Strict Constraints)

1. **唯一合法执行路径**: `*_test.go` 结合 `apitest` 是本 Driver 唯一接受的执行载体。严禁使用 `curl` 命令、外部脚本或第三方测试客户端来执行与验证。所有的报告证据均应来自 `apitest_<case_id>.log`。
2. **规范及模板强校验**: 新建或修改后的 Go 代码必须严格符合三段式结构规范（协议常量区、样本数据区、测试用例区）。参数初始化必须通过 `apitest.EnvFromFile()` 加载环境。**模板合规性审查是硬性拦截门槛**，不合规时即使通过了 Go 语法编译，也不允许进入 Execute 阶段。
3. **安全凭证绝对隔离 (No Secrets Included)**: 
   - `.env` 配置文件只允许存放路由信息。
   - 所有敏感鉴权（如 PaaS-GW Token、用户登陆 Cookie）只允许通过环境变量的内存注入（如 `$APITEST_TOKEN`）作用于当前进程，**严禁硬编码到 `*_test.go` 代码中，也严禁明文保存在 `.env` 中**。执行失败时不得使用 `t.Skip` 掩盖鉴权阻断错误。
4. **先查后问 (Source-first, ask-last)**:
   - 遇到业务未决数据时排查路径必须是：看 `case.md` -> 查 IDL 契约 -> 搜 知识库(KB) -> 检索 同级目录现有用例。
   - 所有手段穷尽后方可向用户提问。且提问必须提供上述链路的排查轨迹（例：“经过查询文档和代码均未发现 XXX 字段的预期格式...”），禁止采用“XX参数是什么？”等无脑提问。
5. **依赖环境阻断**: 针对所有需鉴权的 HTTP API 请求，如果在 Execute 阶段启动前检测到依赖的鉴权信息缺失，且不属于跳过白名单，严禁发起无用请求，必须立刻标记为 `SKIPPED/BLOCKED` 并向用户抛出需补充鉴权的前置拦截提示。
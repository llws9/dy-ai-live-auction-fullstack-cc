# Test Report 断言指南

目标：给定生成的 Go 测试文件（`tests/integration/<method_snake>/<method_snake>_test.go`，scenario 用例位于 `tests/integration/scenario/<scenario_snake>/<scenario_snake>_test.go`）与执行日志（`apitest_{case_id}.log`），生成每个 case 的 Pass/Fail/Skipped/Error 断言结论，并输出可复核的失败原因与证据。

## 1. 输入与映射

- **用例文件**：Go 测试中的每个 case id 对应一条用例。优先从 `WithCaseID("<id>")` 提取；为了兼容存量生成代码，也必须支持本地 helper 包装形态（例如 `newContext(t, "<id>")`，详见 `reuse_amend_guide.md §2.3`）。至少包含：
  - case id：用例 ID（例如 `TC-G01-06`）
  - `apitest.Assert(...)`：断言表达式列表（可为空）
  - `CallHTTP` / `CallRPC`：该用例内的执行步骤
- **日志根目录**：`FEATURE_DIR/test/api_test_logs/`（spec 驱动；由 SKILL 在执行 `go test` 前 `export APITEST_LOG_DIR` 注入）。**不在** `tests/integration/<method>/api_test_logs/` —— 后者只是开发者本地手跑 `go test` 时的兜底位置，正式跑批不再使用。
- **日志文件**：`apitest_{case_id}.log`（method 与 scenario 共用同一个根目录，case_id 在仓库范围内必须唯一）
- **映射规则**：对每个 `case_id`，在日志根目录下匹配同名日志文件：
  - 找不到日志文件 → `SKIPPED`（用例未执行）
  - 找到 → 进入解析与断言

## 2. 日志结构（必须严格按字段来源理解）

日志按分隔段落组织，断言只依赖下列段落：

- `--- Response: Business (JSON) ---`
  - **业务响应体**，是所有 `jsonpath(...)` 断言的唯一数据源
  - 若内容为空、或为 `N/A` → 视为“业务响应为空”（见 4.5）
- `--- Metadata: Business ---`
  - `Business.StatusCode`：业务请求的 HTTP 状态码（用于 `status_code` 断言）
  - `Business.LogID`：业务日志 ID（如存在）
- `--- Metadata: Gateway ---`
  - `Gateway.HTTPStatusCode`：调用网关的 HTTP 状态码
  - `Gateway.HasPermission`：是否有权限（True/False）
  - `Gateway.ErrorCode`：网关错误码（0 表示无错）
  - `Gateway.LogID`：网关日志 ID（如存在）

以下段落仅用于排障上下文，不参与状态判定与断言数据源：

- `--- Runtime: Gateway Request (Curl) ---`
- `--- Runtime: Gateway Response ---`

## 3. 断言表达式（最小语法约定）

断言表达式来自 `apitest.Assert(...)`，每条表达式独立求值，全部通过才算断言通过。

### 3.1 LHS（左值）

- `status_code`
  - 取值来源：`Metadata: Business` 的 `Business.StatusCode`
- `jsonpath('$.path')`
  - 取值来源：`Response: Business (JSON)` 解析后的 JSON
- `len(jsonpath('$.path'))`
  - 返回数组或字符串的长度，值为非负整数
  - 取值来源：对 `Response: Business (JSON)` 解析后提取 `$.path`，计算其长度
- `typeof(jsonpath('$.path'))`
  - 返回类型字符串：`'int' | 'float' | 'string' | 'boolean' | 'list' | 'dict' | 'null'`

### 3.2 操作符（仅支持这些）

- `==`, `!=`, `>`, `>=`, `<`, `<=`
- `in`：右值为列表（例如 `in [1, 2, 3]`），检查左值是否在列表中
- `contains`：左值为字符串，检查是否包含右值字符串

### 3.3 RHS（右值）

- 数字：`200`
- 字符串：单引号（`'InvalidArgument'`）
- 布尔：`true` / `false`
- 列表：`[1, 2, 3]`
- 空：`null`

## 4. Case 状态判定（强制 6 步短路，顺序不可变）

对每个 case，必须严格按以下顺序检查；任一步失败立刻返回，不再执行后续步骤：

### 4.1 网关 HTTP 检查
- 若 `Gateway.HTTPStatusCode != 200` → `FAIL`

### 4.2 权限检查
- 若 `Gateway.HasPermission != True` → `FAIL`

### 4.3 网关错误码检查
- 若 `Gateway.ErrorCode != 0` → `FAIL`

### 4.4 业务状态码规则（与 status_code 断言绑定）

**判断依据**：根据 Go 测试中的调用函数区分：
- `apitest.CallRPC(...)` → **RPC 接口**
- `apitest.CallHTTP(...)` → **HTTP 接口**

**RPC 接口**：**直接跳过此步检查，直接进入 4.5**（RPC 无 HTTP 状态码概念）

**HTTP 接口**：
- 若 **存在** `status_code ...` 断言（例如 `status_code == 400`）：
  - 只在此步检查该 `status_code` 断言是否通过
  - 不通过 → `FAIL`
  - 通过 → 继续后续步骤（即使 `Business.StatusCode != 200` 也不默认失败）
- 若 **不存在** `status_code` 断言：
  - 若 `Business.StatusCode != 200` → `FAIL`

### 4.5 业务响应为空
- 若 `Response: Business (JSON)` 为空或为 `N/A` → `FAIL`

### 4.6 字段级断言
- 对除 `status_code` 外的所有断言（例如 `jsonpath('$.code') == 'InvalidArgument'`）逐条求值
- **字段不存在视为失败**：若 `jsonpath('$.field')` 提取的字段在 JSON 中不存在 → 该断言 `FAIL`
- 任一不通过 → `FAIL`
- 全部通过 → `PASS`

## 5. 解析错误（ERROR）

以下属于技术错误，返回 `ERROR`（不是 PASS/FAIL）：

- 日志文件无法读取
- `Response: Business (JSON)` 非空且非 `N/A`，但 JSON 解析失败
- `Metadata: Business` 或 `Metadata: Gateway` 缺少必需字段（无法完成 4.1~4.4）

## 6. 输出报告格式（必须严格按此格式生成 test_report.md）

### 6.0 Coverage Summary（来自 `triage.yaml` × 执行结果）

> 数据源：`FEATURE_DIR/test/triage.yaml`（每个方法的 `decision` 与 `classification`）+ `apitest_*.log`（执行结果）。本节是「需求维护视角」的核心，必须出现在 §6.1 Summary 之前。

**6.0.1 三向覆盖矩阵**（按 method 维度统计 case）

```markdown
## 用例来源汇总

| 接口 | 分类 | 复用 | 修改 | 新增 | 老用例失败（待修） |
| :--- | :--- | :---: | :---: | :---: | :---: |
| GetAllPolicyGroupMeta | case-only | 1 | 0 | 0 | 0 |
| SearchBank | idl-changed | 4 | 1 | 1 | 1 ← TC-G02-04 失败，根因：IDL 改名（详见 §6.4） |
| ListEnforcementRule | new-method | 0 | 0 | 3 | - |
| **合计** | — | **5** | **1** | **4** | **1** |
```

字段口径：

- **接口**：method 名称（PascalCase RPC func / HTTP path 末段），对应 `triage.yaml.<method>` 节点。
- **分类**：直接拷贝自 `triage.yaml.<method>.classification`。
- **复用 / 修改 / 新增** 列计数规则：
  - `复用`（REUSE）：method 的 `decision == reuse` 时，统计该 method 文件里**所有 existing case** 的数量（来自 `existing_case_ids`）。这些 case 在本次仍然被执行了。
  - `修改`（AMEND）：method 的 `decision == amend` 时，统计 `triage.yaml.changes` 里 `target_case_id` 去重后的数量（被打补丁的 case 数）。`add_case` 不计入此列。
  - `新增`（NEW）：统计本次产出的新 case 数 = `decision == new` 文件里全部 case + `decision == amend` 中 `kind == add_case` 的数量。
- **老用例失败（待修）**：仅对 `decision ∈ {reuse, amend}` 的方法统计 —— 执行结果 FAIL 且 case id 在 `existing_case_ids` 中（即不是本次新增的 case）。这些是本需求顺便暴露的存量缺陷，必须在 §6.4 单独列条目。

**6.0.2 老用例失败处置原则**

- `decision == reuse` 但执行 FAIL：**对本需求不构成阻塞**（功能改动无关），但报告必须明确建议起单独 issue 修复 —— 否则下次 reuse 还会踩同一个雷。
- `decision == amend` 中**未在 `triage.yaml.changes` 内**的 case 失败：**视为本需求引入的回归**，直接计入 §6.1 Failed，不算"老用例待修"。
- `decision == amend` 中**在 `triage.yaml.changes` 内**的 case 失败：按常规失败处理（§6.4），但根因要标注为"补丁不充分"或"补丁错误"，不算"老用例待修"。

**6.0.3 新增 §6.4 老用例失败清单（仅当"老用例失败"列 > 0 时输出）**

```markdown
## 老用例失败（待修）

| 接口 | 用例ID | 根因猜测 | 建议修复 |
| :--- | :--- | :--- | :--- |
| SearchBank | TC-G02-04 | IDL 字段 `Bizlines` 改名为 `BusinessLines` | 单独提 issue 更新该 case Body，或将 method 从 reuse 升级到 amend |
```

根因猜测要按"IDL 改名 / handler 行为变 / 数据漂移 / 环境差异"四类粗分；不能确定时填"待人工排查"，不要乱猜。

### 6.1 Summary Section

必须包含日期和汇总表格：

```markdown
# 测试执行报告
**执行时间**：YYYY-MM-DD HH:MM:SS

## 执行概况

| 总数 | 通过 | 失败 | 跳过 | 错误 |
| :---: | :---: | :---: | :---: | :---: |
| {X} | {Y} | {Z} | {S} | {E} |
```

### 6.2 结果详情（All Cases）

列出**所有** case（包括 PASS / FAIL / SKIPPED / ERROR），使用表格格式：

```markdown
## 结果详情

| 状态 | 用例ID | 标题 | 接口 | 用例位置 | 来源 | 依赖Mock | 日志ID | 失败原因 |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| ✅ **PASS** | {case_id} | {case_title} | {METHOD} {path} 或 RPC {method_name} | {repo_relative_test_path} | {source_cell} | {mock_cell} | `{log_id}` 或 `N/A` | - 或 多行失败详情 |
```

> **图标说明**：✅ PASS / ❌ FAIL / ⏭️ SKIP / ⚠️ ERROR
> **来源取值**：`new` = 本次新生成；`reuse·base` / `reuse·worktree` = 复用 base 分支或工作区已有；`amend·base` / `amend·worktree` = 在 base 或工作区基础上增量修改

**字段说明**：

- **状态**：图标 + 加粗文本，`✅ **PASS**` / `❌ **FAIL**` / `⏭️ **SKIP**` / `⚠️ **ERROR**`，对应 §4 短路判定结果。
- **用例ID**：从 `WithCaseID("<id>")` 提取，与 `case.md` 一致。
- **标题**：取 `case.md` 表格"测试场景"列原文。多步骤 scenario 用例填 scenario 名。
- **接口**：HTTP 写 `<METHOD> <path>`（如 `POST /api/v2/infa/sds/bank/search`）；RPC 写 `RPC <method_name>`（如 `RPC GetAllPolicyGroupMeta`）。从 `triage.yaml.targets[].method` + `case.md` 接口标题取。
- **用例位置**：相对仓库根目录的 `*_test.go` 路径（如 `tests/integration/search_bank/search_bank_test.go`），来自 `triage.yaml.targets[].target_path` / `existing_path`。
- **来源**：单列合并 `triage.yaml.targets[].decision` 与 `coverage_source`，编码规则：
  - `decision=new` → `new`（generated 是隐含值，不再赘述）
  - `decision=reuse` + `coverage_source=base` → `reuse·base`
  - `decision=reuse` + `coverage_source=worktree` → `reuse·worktree`
  - `decision=amend` + `coverage_source=base` → `amend·base`
  - `decision=amend` + `coverage_source=worktree` → `amend·worktree`
  - 若一个 `amend` target 中包含 `changes[].kind == add_case`，该新增 case 的来源填 `new`，其余被修改 case 用 `amend·*`。
- **依赖Mock**：
  - `否`：`case.md` 中该 case 不在任何 `### Mock Setup` 行
  - `是 [<rule_id>](<mock_rule_url>)`：mock-required 且 Bytemock reconcile 成功；rule_id 来自 BAM 返回的 `data.id`，URL 按 `mock.md §3.6` 规则拼接（`<cloud_host>/bam/mock/service/detail?psm=&namespace=&mock_env=&x-bc-region-id=bytedance&api_branch=&endpoint_id=`）
  - `跳过（<reason>）`：mock-required 但 reconcile 失败 / 工具缺失（同时该 case 整体状态为 SKIPPED）
- **日志ID**：**每个用例都必须填写**。从对应的 `apitest_{case_id}.log` 中提取，优先取 `Business.LogID`，其次 `Gateway.LogID`，都没有则填 `N/A`。**PASS 用例同样需要提取，不可省略。**
- **失败原因**：
  - PASS 用例填 `-`（mock 命中可附 `mock 命中 → {code:1, message:"..."}` 这种对照说明，方便 review，不强制）
  - FAIL / ERROR 用例用 `<br>` 换行，包含：
    - 失败类型和关键证据（如 `Gateway.HasPermission: False`）
    - 断言失败时：期望值 vs 实际值
    - `失败分类`：`test_contract_error`（测试契约，可自动修复）/ `env_data_error`（环境数据）/ `product_behavior_error`（产品行为）/ `unknown`。仅 `test_contract_error` 满足自动修复条件（`SKILL.md` §1.2 Execute 第 4 步（安全自愈修复）已规定），其他类别一律不自动改动；无需在报告中重复声明修复状态
    - `服务日志分析`：优先用 `Business.LogID`，其次 `Gateway.LogID` 调 `bam-cli api-test --act analyze-result` 后提取的"问题分析结论"；未查询或失败时写 `skipped (<reason>)`
  - SKIPPED 用例填 `日志文件未找到` 或 mock-required 跳过的具体原因

### 6.3 内部判定字段（用于 thought 过程，不出现在最终报告中）

在生成报告前，模型应内部记录：
- `FailStep`：命中短路的步骤编号（4.1~4.6）
- `Evidence`：直接引用日志中的关键字段
- `AssertionDetails`：断言失败的详细对比
- `FailureCategory`：`test_contract_error`（测试契约错误，可修）/ `env_data_error`（环境或数据问题）/ `product_behavior_error`（疑似被测代码问题）/ `unknown`。仅 `test_contract_error` 准入 `SKILL.md` §1.2 Execute 第 4 步（安全自愈修复）自动修复，其他类别由上层人为决策
- `ServerLogAnalysis`：基于 LogID 的 `bam-cli api-test --act analyze-result` 摘要；仅作失败归因证据，不参与断言判定

自动修复只允许作用于 `test_contract_error`，且只修生成/补丁过的 `*_test.go`。严禁为通过测试而削弱业务断言、删除失败断言、把 expected 改成 actual，或因疑似产品行为问题修改用例。

### 6.2.1 Mock Fixtures（条件输出）

当本次执行包含 `### Mock Setup` 且 Bytemock reconcile 实际运行时，输出 Mock 规则审计章节：

```markdown
## Mock 配置

| 用例ID | 协议 | 下游PSM | 下游方法 | 规则 | 过滤器 | 动作 |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| {case_id} | {RPC/HTTP} | `{callee_psm}` | `{method}` | [{rule_name}]({mock_rule_url}) | `{filter}` | {created/updated/reused/skipped} |
```

规则：
- `规则` 列必须用文本 + 超链接展示，避免在报告中直接展开长 URL。
- `mock_rule_url` 来自 `mock.md` §3.6 的 BAM Mock console URL 拼接规则；§6.2 表中"依赖Mock"列必须用同一个 URL，二者一致。
- 如果某条 mock setup 被跳过，仍在 `Mock Setup Skipped` 中写明原因和修复建议。

### 6.5 报告章节最终顺序（强制）

```
# 测试执行报告
**执行时间**：...

## 用例来源汇总              ← §6.0（含三向矩阵；存在老用例失败时附 §6.0.3）
## 执行概况                  ← §6.1
## 结果详情                  ← §6.2
## Mock 配置                 ← §6.2.1（条件输出：仅当 Bytemock reconcile 运行）
## 老用例失败（待修）          ← §6.4（条件输出：仅当 §6.0 「老用例失败」列 > 0）
```

## 7. 强约束（必须遵守）

- 不允许猜测缺失字段；缺失就按规则给出 `ERROR` 或 `FAIL`
- 不允许把 `Runtime: Gateway Response` 当作断言数据源
- `status_code` 断言只对比 `Business.StatusCode`
- §6.0 Coverage Summary 中 复用/修改/新增 列**必须等于** `triage.yaml` 推导值；任何偏差都视为生成 bug
- 「老用例失败」与 §6.1 的 `失败` 计数**互不重叠**：一条 case 要么算 §6.0「老用例失败（待修）」，要么算 §6.1 失败，按 §6.0.2 区分
- §6.2「依赖Mock」列与 §6.2.1「Mock 配置」表的 `mock_rule_url` **必须一致**（同一 case 在两处的链接指向同一 BAM rule，便于 review 跳转）

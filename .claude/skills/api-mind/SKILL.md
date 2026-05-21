---
name: api-mind
description: API 测试用例自动化 SKILL。核心能力：1. 将 case.md 内容转为可执行的用例代码（支持 Go / Python） 2. 执行用例，并根据执行结果自动修复用例 3. 根据执行结果生成测试报告.支持任务组合调度（仅生成 / 仅执行 / 生成并执行 / 执行并报告 等），支持可重入。当用户提到：用例生成、case转代码、API测试、测试执行、自动修复用例、测试报告 等场景时触发。
---

# api-mind

API 测试用例自动化 SKILL —— 负责任务分类、分发与状态维护，本身不直接执行任何功能逻辑。

## 架构概览

```
api-mind/
├── SKILL.md              # 主入口：任务分类、分发、状态维护
├── mock.md               # ByteMock 使用说明
├── go_driver/            # Go 语言用例生成与执行驱动
│   ├── SKILL.md           # Go driver 调度逻辑
│   ├── apitest.md         # API 测试用例编写规范
│   ├── resources/         # 模板与参考文档
│   │   ├── env_template.md       # 环境变量配置模板
│   │   ├── go_test_template.md   # Go 测试代码模板
│   │   ├── reuse_amend_guide.md  # 用例复用与修正指南
│   │   ├── test_report_guide.md  # 测试报告生成指南
│   │   └── zone_mapping.md       # 可用区映射参考
│   └── runtime/           # 运行时库（Go 源码）
│       ├── runner.go             # 测试执行引擎
│       ├── facade.go             # API 请求门面
│       ├── gateway.go            # 网关路由
│       ├── assert.go             # 断言辅助
│       ├── env.go                # 环境管理
│       ├── extract.go            # 数据提取（JSONPath）
│       ├── fixture.go            # 测试 fixtures
│       ├── header_inject.go      # 请求头注入
│       ├── jsonpath.go           # JSONPath 解析
│       ├── vars.go               # 变量管理
│       ├── log.go                # 日志记录
│       ├── models.go             # 数据模型定义
│       ├── version.go            # 版本信息
│       ├── manifest.json         # 运行时配置清单
│       └── README.md             # 运行时库说明
└── py_driver/            # Python 语言用例生成与执行驱动
    └── SKILL.md           # Python 用例的具体实现（待实现）
```

## 核心职责

SKILL.md 是**调度中枢**，仅负责：

1. **任务分类** — 解析用户意图，识别任务类型与组合
2. **任务分发** — 将子任务路由至对应 driver（go_driver / py_driver）
3. **状态维护** — 跟踪任务执行状态，支持可重入
4. **结果汇总** — 汇总各子任务结果，向用户呈现

**SKILL.md 不直接执行**：用例代码生成、用例执行、报告生成等具体逻辑均由对应 driver 实现。

## 任务类型

| 任务类型 | 标识 | 说明 | 执行方 |
|---------|------|------|--------|
| 用例生成 | `generate` | 将 case.md 转为可执行用例代码 | go_driver 或 py_driver |
| 用例执行 | `execute` | 执行已有用例代码，失败时自动修复 | go_driver 或 py_driver |
| 报告生成 | `report` | 根据执行结果生成测试报告 | go_driver 或 py_driver |

## 用例语言路由规则

根据以下优先级确定用例语言（Go / Python）：

1. **用户显式指定**：用户在输入中明确指定语言（如 "生成 go 用例"、"python 测试代码"）
2. **配置文件读取**：
   - 优先读取当前仓库根目录下的 `.test_config.ini`
   - 其次读取当前仓库根目录下的 `agent.md`
3. **默认值**：若均未指定，默认使用 **Go**

路由逻辑：

- Go 语言 → 任务分发至 `go_driver/`
- Python 语言 → 任务分发至 `py_driver/`

## 配置文件格式参考

### .test_config.ini（示例）

```ini
[api-mind]
language = go
```

### agent.md（示例）

```markdown
# Agent Config

## api-mind
- language: python
```

## 任务组合调度

支持灵活组合，常见模式：

| 用户意图 | 任务组合 | 说明 |
|---------|---------|------|
| "生成用例" | `generate` | 仅生成用例代码 |
| "执行用例" | `execute` | 仅执行已有用例 |
| "生成并执行" | `generate → execute` | 生成后立即执行 |
| "执行并出报告" | `execute → report` | 执行后生成报告 |
| "全流程" | `generate → execute → report` | 完整流水线 |
| "修复用例" | `execute`（含自动修复） | 执行失败时自动修复并重试 |

## 可重入机制

### 状态持久化

每个任务实例以**工作目录**为唯一标识，状态文件存于：

```
<工作目录>/.api-mind/state.json
```

### 状态文件格式

```json
{
  "task_id": "uuid",
  "language": "go",
  "tasks": {
    "generate": {
      "status": "completed",
      "input": "case.md",
      "output": "generated_test.go",
      "timestamp": "2026-05-11T10:00:00+08:00"
    },
    "execute": {
      "status": "failed",
      "input": "generated_test.go",
      "output": null,
      "error": "exit code 1: ...",
      "fix_attempts": 2,
      "timestamp": "2026-05-11T10:05:00+08:00"
    },
    "report": {
      "status": "pending",
      "input": null,
      "output": null,
      "timestamp": null
    }
  },
  "created_at": "2026-05-11T10:00:00+08:00",
  "updated_at": "2026-05-11T10:05:00+08:00"
}
```

### 状态值定义

| 状态 | 含义 |
|------|------|
| `pending` | 未开始 |
| `running` | 执行中 |
| `completed` | 已完成 |
| `failed` | 执行失败（可重试） |
| `skipped` | 已跳过 |

### 可重入流程

1. 收到请求后，先读取 `state.json`
2. 如果状态文件存在且任务未完成，从**上次中断处**继续
3. 如果状态文件不存在，创建新任务实例
4. 已 `completed` 的任务不重复执行，除非用户显式要求
5. `failed` 的任务可重试，修复次数累加

## 工作流程

### 步骤 1：解析用户意图

从用户输入中提取：

- **任务类型**：generate / execute / report 或其组合
- **用例语言**：Go / Python（按路由规则确定）
- **输入文件**：case.md 路径（如未指定，默认为当前目录下的 `case.md`）
- **工作目录**：用例代码和状态文件的存放位置，默认为`FEATURE_DIR/test/`

### 步骤 2：加载状态

读取 `<工作目录>/.api-mind/state.json`，判断：

- 是否为可重入场景
- 各子任务的当前状态
- 是否需要跳过已完成的步骤

### 步骤 3：按序执行任务链

根据任务组合，按依赖顺序执行：

```
generate → execute → report
```

每个子任务的执行逻辑：

1. 更新状态为 `running`
2. **分发**至对应 driver（go_driver 或 py_driver）
3. 等待 driver 返回结果
4. 根据结果更新状态为 `completed` 或 `failed`
5. 若 `execute` 失败，触发自动修复流程（由 driver 内部实现）

### 步骤 4：自动修复（execute 阶段）

当用例执行失败时：

1. driver 收集错误信息
2. driver 基于错误信息修复用例代码
3. 重新执行修复后的用例
4. 最多重试 **3 次**
5. 超过重试次数仍失败，标记为 `failed` 并记录错误详情

### 步骤 5：结果汇总

所有任务完成后，向用户呈现：

- 各子任务的执行状态
- 生成的文件路径
- 执行结果摘要
- 报告路径（如适用）
- 失败原因及修复历史（如适用）

## Driver 接口约定

go_driver 和 py_driver 需遵循统一接口：

### generate

**输入**：
- `case_file`：case.md 文件路径
- `output_dir`：输出目录

**输出**：
- `code_file`：生成的用例代码文件路径
- `status`：success / failed
- `error`：错误信息（如失败）

### execute

**输入**：
- `code_file`：用例代码文件路径
- `work_dir`：工作目录
- `auto_fix`：是否自动修复（默认 true）
- `max_retries`：最大重试次数（默认 3）

**输出**：
- `status`：success / failed
- `results`：执行结果详情
- `error`：错误信息（如失败）
- `fix_history`：修复历史（如有修复）

### report

**输入**：
- `execute_results`：执行结果
- `output_dir`：报告输出目录

**输出**：
- `report_file`：报告文件路径
- `status`：success / failed
- `error`：错误信息（如失败）

## 注意事项

1. **工作目录规范**：所有中间产物（代码、状态、报告）均存于工作目录下，不污染项目源码
2. **幂等性**：相同输入多次执行，结果一致（已完成的步骤不重复执行）
3. **日志记录**：每个子任务的关键操作和错误信息应记录到状态文件中
4. **错误处理**：任一子任务失败时，停止后续任务，但保留已完成步骤的状态
5. **用户覆盖**：用户显式指令优先级高于配置文件和默认值

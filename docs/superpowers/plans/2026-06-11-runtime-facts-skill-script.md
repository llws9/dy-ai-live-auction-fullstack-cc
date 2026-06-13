# Runtime Facts Skill/Script Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `runtime-facts` as a deterministic script plus agent-facing skill that answers both "what code is this service running" and "why did my change not take effect".

**Architecture:** A Python stdlib script collects repo, port, process, URL, timestamp, and consistency facts as structured JSON. The skill interprets those findings with a fixed diagnosis tree, especially for stale processes and invalid validation environments. Documentation links the implementation back to the MCP opportunity analysis and corrects any method-documentation statements that described runtime-facts as already mature before implementation.

**Tech Stack:** Python 3 stdlib (`argparse`, `json`, `subprocess`, `dataclasses`, `datetime`, `fnmatch`, `os`, `pathlib`, `unittest`, `urllib.parse`), macOS/Linux commands (`git`, `lsof`, `ps`), project Skill markdown under `.agents/skills/`.

---

## Source Spec

Implement from:

- `docs/superpowers/specs/2026-06-11-runtime-facts-staleness-diagnosis-design.md`

Key requirements:

- Preserve the `runtime-facts/v1` JSON schema shape while adding freshness fields.
- Add finding codes: `NO_LISTENER_ON_PORT`, `MULTIPLE_LISTENERS_ON_PORT`, `PROCESS_CWD_OUTSIDE_REPO`, `DIRTY_TREE_NOT_DEPLOYED`, `STALE_PROCESS_BEFORE_CHANGE`.
- Compare timestamps only after converting to timezone-aware epoch seconds.
- Default stale tolerance is `5` seconds and must be configurable via `--stale-tolerance-seconds`.
- Dirty file mtime only considers runtime-input files; pure docs/test-report/cache changes must not trigger stale.
- Skill must diagnose "改了不生效 / 验证可不可信" before treating symptoms as business bugs.
- Script and skill are read-only: no restart, no kill, no build, no git mutation.

## File Structure

- Create: `docs/superpowers/runtime-facts/runtime_facts.py`
  - CLI and library for collecting runtime facts.
  - Owns deterministic finding generation.
  - Does not mutate repo, processes, services, or Docker.
- Create: `docs/superpowers/runtime-facts/test_runtime_facts.py`
  - Unit tests with fake command runner and injectable mtime provider.
  - Tests do not require real ports, Docker, or network.
- Create: `.agents/skills/runtime-facts/SKILL.md`
  - Skill trigger metadata, invocation protocol, JSON interpretation rules, diagnosis tree, response format.
- Modify: `docs/superpowers/specs/2026-06-11-cross-project-mcp-opportunity-map-design.md`
  - Link to the script/skill validation artifact and command.
- Modify: `docs/superpowers/specs/2026-06-11-layered-dev-methodology.md`
  - Correct premature wording that described runtime-facts as mature before implementation.

## Command Contract

Primary command:

```bash
python3 docs/superpowers/runtime-facts/runtime_facts.py \
  --repo-root . \
  --target-ref origin/main \
  --port 5173 \
  --url http://localhost:5173
```

Optional stale tolerance override:

```bash
python3 docs/superpowers/runtime-facts/runtime_facts.py \
  --repo-root . \
  --target-ref origin/main \
  --port 5173 \
  --stale-tolerance-seconds 10
```

Exit behavior:

- Exit `0`: JSON snapshot generated successfully, even when `consistency.status` is `mismatch`.
- Exit `2`: invalid input, such as an invalid port or invalid stale tolerance.
- Exit `3`: command dependency failure prevents basic repo snapshot.

Expected JSON shape includes:

```json
{
  "schema_version": "runtime-facts/v1",
  "repo": {
    "root": "/abs/path",
    "branch": "main",
    "head": "abc123",
    "dirty": true,
    "target_ref": "origin/main",
    "target_head": "def456",
    "head_matches_target": false,
    "head_committed_at": "2026-06-11T10:00:00+08:00",
    "runtime_input_changed_at": "2026-06-11T10:01:00+08:00",
    "code_changed_at": "2026-06-11T10:01:00+08:00"
  },
  "ports": [
    {
      "port": 5173,
      "processes": [
        {
          "pid": 12345,
          "command": "node",
          "cwd": "/abs/path/frontend/h5",
          "cwd_inside_repo": true,
          "started_at": "2026-06-11T09:59:00+08:00",
          "runtime_source": {
            "branch": "main",
            "head": "abc123",
            "dirty": false
          }
        }
      ]
    }
  ],
  "urls": [
    {
      "url": "http://localhost:5173",
      "host": "localhost",
      "port": 5173,
      "scheme": "http"
    }
  ],
  "consistency": {
    "status": "mismatch",
    "findings": [
      {
        "severity": "warning",
        "code": "STALE_PROCESS_BEFORE_CHANGE",
        "message": "Port 5173 process 12345 started before the latest runtime-input change."
      }
    ]
  }
}
```

---

### Task 1: Add Runtime Facts Test Suite

**Files:**
- Create: `docs/superpowers/runtime-facts/test_runtime_facts.py`
- Create directory: `docs/superpowers/runtime-facts/`

- [ ] **Step 1: Create the test directory**

Run:

```bash
mkdir -p docs/superpowers/runtime-facts
```

Expected: directory exists.

- [ ] **Step 2: Write failing tests**

Create `docs/superpowers/runtime-facts/test_runtime_facts.py` with the complete tests below. These tests intentionally import `runtime_facts` before the implementation exists, so the first run must fail.

```python
import unittest

import runtime_facts


class FakeRunner:
    def __init__(self, outputs):
        self.outputs = outputs
        self.calls = []

    def run(self, args, cwd=None, check=False):
        key = tuple(args)
        self.calls.append((key, cwd))
        if key not in self.outputs:
            return runtime_facts.CommandResult(returncode=1, stdout="", stderr=f"missing fake output: {key}")
        return self.outputs[key]


def result(stdout="", stderr="", returncode=0):
    return runtime_facts.CommandResult(returncode=returncode, stdout=stdout, stderr=stderr)


def base_outputs(extra=None):
    outputs = {
        ("git", "rev-parse", "--show-toplevel"): result("/repo\n"),
        ("git", "branch", "--show-current"): result("main\n"),
        ("git", "rev-parse", "--short", "HEAD"): result("abc123\n"),
        ("git", "status", "--porcelain"): result(""),
        ("git", "rev-parse", "--short", "origin/main"): result("abc123\n"),
        ("git", "log", "-1", "--format=%cI"): result("2026-06-11T10:00:00+08:00\n"),
    }
    if extra:
        outputs.update(extra)
    return outputs


def codes(snapshot):
    return [finding["code"] for finding in snapshot["consistency"]["findings"]]


def listener_outputs(pid="12345", cwd="/repo/frontend/h5", process_head="abc123", lstart="Thu Jun 11 10:00:10 2026"):
    return {
        ("lsof", "-nP", "-iTCP:5173", "-sTCP:LISTEN"): result(
            "COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME\n"
            f"node    {pid} user   20u  IPv6  12345      0t0  TCP *:5173 (LISTEN)\n"
        ),
        ("ps", "-p", pid, "-o", "comm="): result("node\n"),
        ("ps", "-p", pid, "-o", "lstart="): result(f"{lstart}\n"),
        ("lsof", "-a", "-p", pid, "-d", "cwd", "-Fn"): result(f"p{pid}\nn{cwd}\n"),
        ("git", "-C", cwd, "branch", "--show-current"): result("main\n"),
        ("git", "-C", cwd, "rev-parse", "--short", "HEAD"): result(f"{process_head}\n"),
        ("git", "-C", cwd, "status", "--porcelain"): result(""),
    }


class RuntimeFactsTest(unittest.TestCase):
    def test_collects_repo_snapshot_and_detects_target_mismatch(self):
        runner = FakeRunner(base_outputs({
            ("git", "status", "--porcelain"): result(" M frontend/h5/src/App.tsx\n"),
            ("git", "rev-parse", "--short", "origin/main"): result("def456\n"),
        }))

        snapshot = runtime_facts.collect_snapshot(
            repo_root="/repo",
            target_ref="origin/main",
            ports=[],
            urls=[],
            stale_tolerance_seconds=5,
            runner=runner,
            mtime_provider=lambda path: 1781157660.0,
        )

        self.assertEqual(snapshot["repo"]["branch"], "main")
        self.assertEqual(snapshot["repo"]["head"], "abc123")
        self.assertTrue(snapshot["repo"]["dirty"])
        self.assertFalse(snapshot["repo"]["head_matches_target"])
        self.assertIn("HEAD_DIFFERS_FROM_TARGET", codes(snapshot))

    def test_parses_lsof_and_attaches_runtime_source_for_process_cwd(self):
        runner = FakeRunner(base_outputs(listener_outputs()))

        snapshot = runtime_facts.collect_snapshot(
            repo_root="/repo",
            target_ref="origin/main",
            ports=[5173],
            urls=[],
            stale_tolerance_seconds=5,
            runner=runner,
        )

        process = snapshot["ports"][0]["processes"][0]
        self.assertEqual(process["pid"], 12345)
        self.assertEqual(process["command"], "node")
        self.assertEqual(process["cwd"], "/repo/frontend/h5")
        self.assertTrue(process["cwd_inside_repo"])
        self.assertEqual(process["runtime_source"]["head"], "abc123")
        self.assertNotIn("PROCESS_HEAD_DIFFERS_FROM_REPO", codes(snapshot))

    def test_url_parsing_maps_default_ports(self):
        runner = FakeRunner(base_outputs())

        snapshot = runtime_facts.collect_snapshot(
            repo_root="/repo",
            target_ref="origin/main",
            ports=[],
            urls=["http://localhost/app", "https://example.com/admin"],
            stale_tolerance_seconds=5,
            runner=runner,
        )

        self.assertEqual(snapshot["urls"][0]["port"], 80)
        self.assertEqual(snapshot["urls"][1]["port"], 443)

    def test_no_listener_emits_finding(self):
        runner = FakeRunner(base_outputs({
            ("lsof", "-nP", "-iTCP:5173", "-sTCP:LISTEN"): result(""),
        }))

        snapshot = runtime_facts.collect_snapshot(
            repo_root="/repo",
            target_ref="origin/main",
            ports=[5173],
            urls=[],
            stale_tolerance_seconds=5,
            runner=runner,
        )

        self.assertIn("NO_LISTENER_ON_PORT", codes(snapshot))

    def test_multiple_listeners_emits_finding(self):
        multi_lsof = (
            "COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME\n"
            "node    12345 user   20u  IPv6  12345      0t0  TCP *:5173 (LISTEN)\n"
            "node    67890 user   20u  IPv6  12345      0t0  TCP *:5173 (LISTEN)\n"
        )
        outputs = listener_outputs(pid="12345", cwd="/repo/frontend/h5")
        outputs.update(listener_outputs(pid="67890", cwd="/repo/frontend/admin"))
        outputs[("lsof", "-nP", "-iTCP:5173", "-sTCP:LISTEN")] = result(multi_lsof)
        runner = FakeRunner(base_outputs(outputs))

        snapshot = runtime_facts.collect_snapshot(
            repo_root="/repo",
            target_ref="origin/main",
            ports=[5173],
            urls=[],
            stale_tolerance_seconds=5,
            runner=runner,
        )

        self.assertIn("MULTIPLE_LISTENERS_ON_PORT", codes(snapshot))

    def test_cwd_outside_repo_emits_finding(self):
        runner = FakeRunner(base_outputs(listener_outputs(cwd="/tmp/other-project", process_head="zzz999")))

        snapshot = runtime_facts.collect_snapshot(
            repo_root="/repo",
            target_ref="origin/main",
            ports=[5173],
            urls=[],
            stale_tolerance_seconds=5,
            runner=runner,
        )

        self.assertIn("PROCESS_CWD_OUTSIDE_REPO", codes(snapshot))

    def test_dirty_tree_not_deployed(self):
        runner = FakeRunner(base_outputs({
            ("git", "status", "--porcelain"): result(" M frontend/h5/src/App.tsx\n"),
            **listener_outputs(),
        }))

        snapshot = runtime_facts.collect_snapshot(
            repo_root="/repo",
            target_ref="origin/main",
            ports=[5173],
            urls=[],
            stale_tolerance_seconds=5,
            runner=runner,
            mtime_provider=lambda path: 1781157660.0,
        )

        self.assertIn("DIRTY_TREE_NOT_DEPLOYED", codes(snapshot))

    def test_stale_process_before_change(self):
        runner = FakeRunner(base_outputs({
            ("git", "status", "--porcelain"): result(" M frontend/h5/src/App.tsx\n"),
            **listener_outputs(lstart="Thu Jun 11 09:59:00 2026"),
        }))

        snapshot = runtime_facts.collect_snapshot(
            repo_root="/repo",
            target_ref="origin/main",
            ports=[5173],
            urls=[],
            stale_tolerance_seconds=5,
            runner=runner,
            mtime_provider=lambda path: 1781157660.0,
        )

        self.assertIn("STALE_PROCESS_BEFORE_CHANGE", codes(snapshot))

    def test_stale_not_emitted_when_timestamp_missing(self):
        runner = FakeRunner(base_outputs({
            ("git", "status", "--porcelain"): result(" M frontend/h5/src/App.tsx\n"),
            **listener_outputs(lstart="not-a-date"),
        }))

        snapshot = runtime_facts.collect_snapshot(
            repo_root="/repo",
            target_ref="origin/main",
            ports=[5173],
            urls=[],
            stale_tolerance_seconds=5,
            runner=runner,
            mtime_provider=lambda path: 1781157660.0,
        )

        self.assertNotIn("STALE_PROCESS_BEFORE_CHANGE", codes(snapshot))

    def test_stale_within_tolerance_not_emitted(self):
        runner = FakeRunner(base_outputs({
            ("git", "status", "--porcelain"): result(" M frontend/h5/src/App.tsx\n"),
            **listener_outputs(lstart="Thu Jun 11 10:00:57 2026"),
        }))

        snapshot = runtime_facts.collect_snapshot(
            repo_root="/repo",
            target_ref="origin/main",
            ports=[5173],
            urls=[],
            stale_tolerance_seconds=5,
            runner=runner,
            mtime_provider=lambda path: 1781157660.0,
        )

        self.assertNotIn("STALE_PROCESS_BEFORE_CHANGE", codes(snapshot))

    def test_docs_only_dirty_change_does_not_emit_stale(self):
        runner = FakeRunner(base_outputs({
            ("git", "status", "--porcelain"): result(" M docs/superpowers/specs/example.md\n"),
            **listener_outputs(lstart="Thu Jun 11 09:59:00 2026"),
        }))

        snapshot = runtime_facts.collect_snapshot(
            repo_root="/repo",
            target_ref="origin/main",
            ports=[5173],
            urls=[],
            stale_tolerance_seconds=5,
            runner=runner,
            mtime_provider=lambda path: 1781157660.0,
        )

        self.assertNotIn("STALE_PROCESS_BEFORE_CHANGE", codes(snapshot))

    def test_stale_tolerance_is_configurable(self):
        runner = FakeRunner(base_outputs({
            ("git", "status", "--porcelain"): result(" M frontend/h5/src/App.tsx\n"),
            **listener_outputs(lstart="Thu Jun 11 10:00:50 2026"),
        }))

        snapshot = runtime_facts.collect_snapshot(
            repo_root="/repo",
            target_ref="origin/main",
            ports=[5173],
            urls=[],
            stale_tolerance_seconds=15,
            runner=runner,
            mtime_provider=lambda path: 1781157660.0,
        )

        self.assertNotIn("STALE_PROCESS_BEFORE_CHANGE", codes(snapshot))


if __name__ == "__main__":
    unittest.main()
```

- [ ] **Step 3: Run tests to verify they fail**

Run:

```bash
python3 -m unittest docs/superpowers/runtime-facts/test_runtime_facts.py
```

Expected: FAIL with `ModuleNotFoundError: No module named 'runtime_facts'`.

- [ ] **Step 4: Commit tests only**

```bash
git add docs/superpowers/runtime-facts/test_runtime_facts.py
git commit -m "test: add runtime facts collector tests"
```

Expected: commit succeeds in an isolated worktree. If not in an isolated clean worktree, update the SDD state and delay commit.

---

### Task 2: Implement Runtime Facts Collector Core

**Files:**
- Create: `docs/superpowers/runtime-facts/runtime_facts.py`
- Test: `docs/superpowers/runtime-facts/test_runtime_facts.py`

- [ ] **Step 1: Create the implementation file**

Create `docs/superpowers/runtime-facts/runtime_facts.py` with a Python stdlib implementation that exposes these exact public names and signatures:

- `SCHEMA_VERSION = "runtime-facts/v1"`
- `DEFAULT_STALE_TOLERANCE_SECONDS = 5`
- `CommandResult(returncode: int, stdout: str, stderr: str)` as a `@dataclass`
- `SubprocessRunner.run(args: list[str], cwd: str | None = None, check: bool = False) -> CommandResult`
- `collect_snapshot(repo_root: str, target_ref: str, ports: list[int], urls: list[str], stale_tolerance_seconds: int = DEFAULT_STALE_TOLERANCE_SECONDS, runner=None, mtime_provider=os.path.getmtime) -> dict[str, object]`
- `parse_ports(values: list[str]) -> list[int]`
- `parse_tolerance(value: str) -> int`
- `main(argv: list[str] | None = None) -> int`

Implementation requirements:

- `git_snapshot()` must run:
  - `git rev-parse --show-toplevel`
  - `git branch --show-current`
  - `git rev-parse --short HEAD`
  - `git status --porcelain`
  - `git rev-parse --short <target-ref>`
  - `git log -1 --format=%cI`
- `collect_port()` must run:
  - `lsof -nP -iTCP:<port> -sTCP:LISTEN`
  - `ps -p <pid> -o comm=`
  - `ps -p <pid> -o lstart=`
  - `lsof -a -p <pid> -d cwd -Fn`
  - `git -C <cwd> branch --show-current`
  - `git -C <cwd> rev-parse --short HEAD`
  - `git -C <cwd> status --porcelain`
- Timestamp comparison must convert `git %cI`, `ps lstart`, and file mtime to epoch seconds before comparison.
- `runtime_input_changed_at` must only consider existing files matching:
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
- Private helper keys such as `_code_changed_epoch` and `_started_epoch` may be used internally but must not appear in JSON output.
- `main()` must parse `--stale-tolerance-seconds`, reject negative values with exit `2`, and print valid JSON on success.

- [ ] **Step 2: Run targeted tests**

Run:

```bash
python3 -m unittest docs/superpowers/runtime-facts/test_runtime_facts.py
```

Expected: PASS.

- [ ] **Step 3: Run JSON smoke test**

Run:

```bash
python3 docs/superpowers/runtime-facts/runtime_facts.py --repo-root . --target-ref origin/main --port 5173 | python3 -m json.tool >/tmp/runtime-facts.json
```

Expected: exits `0`; `/tmp/runtime-facts.json` contains `schema_version` equal to `runtime-facts/v1`.

- [ ] **Step 4: Commit script implementation**

```bash
git add docs/superpowers/runtime-facts/runtime_facts.py docs/superpowers/runtime-facts/test_runtime_facts.py
git commit -m "feat: add runtime facts collector"
```

Expected: commit succeeds in an isolated worktree. If not in an isolated clean worktree, update the SDD state and delay commit.

---

### Task 3: Add Runtime Facts Skill

**Files:**
- Create: `.agents/skills/runtime-facts/SKILL.md`

- [ ] **Step 1: Create skill directory**

Run:

```bash
mkdir -p .agents/skills/runtime-facts
```

Expected: directory exists.

- [ ] **Step 2: Write Skill definition**

Create `.agents/skills/runtime-facts/SKILL.md` with this content:

```markdown
---
name: runtime-facts
description: Use this skill when the user needs to verify what code a local service, URL, port, process, or browser-visible app is actually running, or when changes appear not to take effect. It collects structured runtime facts including git branch, commit, dirty status, port listeners, process cwd, process start time, stale-process findings, and consistency with a target ref. Triggers include "运行事实", "环境事实", "查端口对应哪个 worktree", "服务是不是最新代码", "浏览器看到的是哪个分支", "改了不生效", "修改没生效", "重启了还是旧的", "验证可不可信", "是不是缓存", "stale build", "runtime facts", "what is this port running", "why is my change not showing".
---

# Runtime Facts

This skill verifies the runtime source and freshness of local services before debugging, deployment verification, or browser-based validation.

## Core Rule

Do not guess from memory or previous terminal output. Always collect a fresh snapshot with:

```bash
python3 docs/superpowers/runtime-facts/runtime_facts.py --repo-root . --target-ref origin/main
```

Add one or more `--port <port>` values when checking a known local service:

```bash
python3 docs/superpowers/runtime-facts/runtime_facts.py --repo-root . --target-ref origin/main --port 5173 --port 8080
```

Add `--url <url>` when the user provides a browser URL:

```bash
python3 docs/superpowers/runtime-facts/runtime_facts.py --repo-root . --target-ref origin/main --url http://localhost:5173
```

Use `--stale-tolerance-seconds <n>` only when a machine or CI environment needs a wider or narrower freshness window:

```bash
python3 docs/superpowers/runtime-facts/runtime_facts.py --repo-root . --target-ref origin/main --port 5173 --stale-tolerance-seconds 10
```

## Finding Codes

- `NO_LISTENER_ON_PORT`: requested port has no listener.
- `MULTIPLE_LISTENERS_ON_PORT`: requested port has multiple listener processes.
- `PROCESS_CWD_OUTSIDE_REPO`: process cwd is outside the current repository.
- `DIRTY_TREE_NOT_DEPLOYED`: repository has local changes but the port process runs a clean tree.
- `STALE_PROCESS_BEFORE_CHANGE`: process started before the latest runtime-input code change, outside tolerance.
- `HEAD_DIFFERS_FROM_TARGET`: repository HEAD does not match target ref.
- `PROCESS_HEAD_DIFFERS_FROM_REPO`: port process git HEAD does not match repository HEAD.

## Diagnosis Rules

For "改了不生效 / 修改没生效 / 验证可不可信" reports, interpret findings in this order:

1. If `NO_LISTENER_ON_PORT` appears, the service is not running on that port. Do not debug the business symptom yet.
2. If `PROCESS_CWD_OUTSIDE_REPO` appears, the port belongs to another repo or worktree. Treat current validation as invalid until runtime source is aligned.
3. If `STALE_PROCESS_BEFORE_CHANGE` or `DIRTY_TREE_NOT_DEPLOYED` appears, the process likely has not consumed the latest change. Recommend rebuild/restart through the project deploy flow before business debugging.
4. If `MULTIPLE_LISTENERS_ON_PORT` appears, old process residue may be present. Recommend process cleanup through the project deploy flow before validation.
5. If `PROCESS_HEAD_DIFFERS_FROM_REPO` or `HEAD_DIFFERS_FROM_TARGET` appears, report source-version mismatch and do not claim target-ref validation.
6. If `consistency.status` is `ok` and no freshness findings exist, runtime-source evidence is sufficient for local validation.

## Boundary

This skill does not restart services, kill processes, deploy code, clean caches, or mutate git state. It only collects and interprets runtime facts.

For local deployment or restart, use the project deployment flow after presenting the runtime mismatch evidence.

## Response Format

Start with the required project first line:

```text
当前分支/worktree：<branch> @ <absolute-worktree-path>
```

Then report:

- `事实快照`: repo branch, HEAD, target ref, dirty status, code change time.
- `端口/URL`: requested ports and URLs with listener details.
- `一致性结论`: `ok` or `mismatch`.
- `根因层`: service not running, wrong worktree, stale process/build, old process residue, source-version mismatch, or runtime aligned.
- `下一步`: exact recommended next action, without automatically mutating state.
```

- [ ] **Step 3: Smoke-check skill file exists**

Run:

```bash
test -f .agents/skills/runtime-facts/SKILL.md
```

Expected: command exits `0`.

- [ ] **Step 4: Commit skill**

```bash
git add .agents/skills/runtime-facts/SKILL.md
git commit -m "feat: add runtime facts skill"
```

Expected: commit succeeds in an isolated worktree. If not in an isolated clean worktree, update the SDD state and delay commit.

---

### Task 4: Link Runtime Facts Validation Artifacts

**Files:**
- Modify: `docs/superpowers/specs/2026-06-11-cross-project-mcp-opportunity-map-design.md`

- [ ] **Step 1: Insert validation artifact section**

In `docs/superpowers/specs/2026-06-11-cross-project-mcp-opportunity-map-design.md`, insert this section before the final one-sentence conclusion section:

```markdown
## Skill/Script 验证产物

`环境事实查询` 不直接进入 MCP 实现，先通过以下产物验证复用价值：

- `docs/superpowers/runtime-facts/runtime_facts.py`：结构化运行事实与 stale 进程诊断采集脚本
- `.agents/skills/runtime-facts/SKILL.md`：面向 agent 的调用、解释与「改了不生效」诊断协议
- `docs/superpowers/runtime-facts/test_runtime_facts.py`：确定性单元测试，覆盖端口监听、worktree 来源、dirty tree、stale 容差与纯文档变更不误报

验证命令：

```bash
python3 docs/superpowers/runtime-facts/runtime_facts.py --repo-root . --target-ref origin/main --port 5173 --port 8080
```

达标后再 MCP 化的证据标准：

- 在两个以上项目复用。
- 至少服务两个调用场景，例如部署验证和排障诊断。
- 发生过 shell 文本解析误判，且结构化 JSON 能避免。
- 输出 schema 保持稳定。
```

If the document uses numbered headings, renumber subsequent headings so there is no duplicate number.

- [ ] **Step 2: Check headings**

Run:

```bash
grep -n "^## " docs/superpowers/specs/2026-06-11-cross-project-mcp-opportunity-map-design.md
```

Expected: headings are not duplicated or out of order.

- [ ] **Step 3: Commit spec link**

```bash
git add docs/superpowers/specs/2026-06-11-cross-project-mcp-opportunity-map-design.md
git commit -m "docs: link runtime facts validation path"
```

Expected: commit succeeds in an isolated worktree. If not in an isolated clean worktree, update the SDD state and delay commit.

---

### Task 5: Correct Layered Methodology Runtime-Facts Status

**Files:**
- Modify: `docs/superpowers/specs/2026-06-11-layered-dev-methodology.md`

- [ ] **Step 1: Replace premature maturity wording**

Edit `docs/superpowers/specs/2026-06-11-layered-dev-methodology.md` so runtime-facts is described as a validation-path asset, not as already mature before implementation.

Use these replacements:

```text
Before: docs/superpowers/runtime-facts/*.py
After: docs/superpowers/runtime-facts/runtime_facts.py（Skill/Script 验证路径产物）
```

```text
Before: `runtime_facts.py` 是待验证的 MCP 前身
After: `runtime_facts.py` 是环境事实查询的 Script 前身，用于先验证复用价值与 schema 稳定性
```

If exact text differs, preserve the section intent and make the status accurate: runtime-facts is a designed/implemented validation artifact, not proof that MCP is already warranted.

- [ ] **Step 2: Verify no stale wording remains**

Run:

```bash
grep -n "runtime-facts\|runtime_facts" docs/superpowers/specs/2026-06-11-layered-dev-methodology.md
```

Expected: references are accurate and no line claims runtime-facts is a long-running mature production asset without qualification.

- [ ] **Step 3: Commit method correction**

```bash
git add docs/superpowers/specs/2026-06-11-layered-dev-methodology.md
git commit -m "docs: clarify runtime facts validation status"
```

Expected: commit succeeds in an isolated worktree. If not in an isolated clean worktree, update the SDD state and delay commit.

---

### Task 6: Final Verification

**Files:**
- Verify: `docs/superpowers/runtime-facts/runtime_facts.py`
- Verify: `docs/superpowers/runtime-facts/test_runtime_facts.py`
- Verify: `.agents/skills/runtime-facts/SKILL.md`
- Verify: `docs/superpowers/specs/2026-06-11-cross-project-mcp-opportunity-map-design.md`
- Verify: `docs/superpowers/specs/2026-06-11-layered-dev-methodology.md`

- [ ] **Step 1: Run unit tests**

```bash
python3 -m unittest docs/superpowers/runtime-facts/test_runtime_facts.py
```

Expected: PASS.

- [ ] **Step 2: Run live snapshot command**

```bash
python3 docs/superpowers/runtime-facts/runtime_facts.py --repo-root . --target-ref origin/main --port 5173 --port 8080
```

Expected: exits `0` and prints valid JSON. `consistency.status` may be `ok` or `mismatch`; mismatch is a factual result, not a script failure.

- [ ] **Step 3: Validate JSON shape**

```bash
python3 docs/superpowers/runtime-facts/runtime_facts.py --repo-root . --target-ref origin/main --port 5173 | python3 -m json.tool >/tmp/runtime-facts.json
```

Expected: exits `0`, proving output is valid JSON.

- [ ] **Step 4: Verify stale tolerance CLI validation**

```bash
python3 docs/superpowers/runtime-facts/runtime_facts.py --repo-root . --stale-tolerance-seconds -1 >/tmp/runtime-facts-invalid.json
```

Expected: exits `2` and prints JSON containing `invalid stale tolerance` to stderr.

- [ ] **Step 5: Verify write set**

```bash
git diff --name-only
```

Expected: changed files for this run are limited to:

```text
.agents/skills/runtime-facts/SKILL.md
docs/superpowers/runtime-facts/runtime_facts.py
docs/superpowers/runtime-facts/test_runtime_facts.py
docs/superpowers/specs/2026-06-11-cross-project-mcp-opportunity-map-design.md
docs/superpowers/specs/2026-06-11-layered-dev-methodology.md
```

If other files appear, stop and inspect whether they were pre-existing dirty changes or scope creep.

---

## Self-Review

Spec coverage:

- Structured runtime snapshot is covered by Tasks 1-2.
- New finding codes are covered by Task 1 tests and Task 2 implementation.
- Stale process freshness with epoch comparison, runtime-input whitelist, and configurable tolerance is covered by Task 1 tests and Task 2 implementation.
- Skill diagnosis tree is covered by Task 3.
- MCP validation linkage is covered by Task 4.
- Premature method-document wording is corrected by Task 5.
- Final command and JSON validation are covered by Task 6.

Placeholder scan:

- No `TBD`, `TODO`, or unspecified implementation steps remain.
- Each code-producing task includes concrete file content or exact required public interfaces and commands.
- Each verification step includes exact command and expected result.

Type consistency:

- Tests call `collect_snapshot(..., stale_tolerance_seconds=..., runner=..., mtime_provider=...)` and Task 2 defines that exact signature.
- Tests use `CommandResult`, `schema_version`, and finding code strings defined by Task 2.
- Skill finding codes match script finding codes exactly.
EOF; __tr_native_ec=$?; pwd -P >| '/var/folders/r4/sz9jkd_s4vx_yhrz1yn42cfm0000gn/T/agent-toolhost/jobs/job-1fc0f7cf05474812b836dbc3dc3206ef/cwd.txt'; exit "$__tr_native_ec"

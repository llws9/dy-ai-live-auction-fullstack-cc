#!/usr/bin/env python3
"""Bootstrap SDD state for /sdd-run.

This script is intentionally small and deterministic: it does not execute the
development task. It only parses the user's /sdd-run input and ensures a state
file exists before an agent dispatches implementation subagents.
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import re
import subprocess
import sys
from pathlib import Path
from typing import Iterable


LABELS = {
    "plan": ("plan：", "plan:", "开发的plan：", "开发的plan:", "方案：", "方案:"),
    "tasks": ("tasks：", "tasks:", "task：", "task:", "任务：", "任务:"),
    "state": ("state：", "state:"),
    "scope": ("scope：", "scope:", "范围：", "范围:"),
}


class NeedsSelection(Exception):
    def __init__(self, payload: dict[str, object]):
        super().__init__("multiple or missing /sdd-run context candidates")
        self.payload = payload


def run_git(repo_root: Path, args: list[str], fallback: str) -> str:
    try:
        result = subprocess.run(
            ["git", *args],
            cwd=repo_root,
            text=True,
            capture_output=True,
            check=True,
        )
        return result.stdout.strip() or fallback
    except Exception:
        return fallback


def kebab(value: str) -> str:
    value = value.strip()
    value = re.sub(r"[^\w\u4e00-\u9fff.-]+", "-", value)
    value = re.sub(r"-+", "-", value).strip("-._")
    return value.lower() or "sdd-run"


def normalize_path(path: str | None) -> str | None:
    if not path:
        return None
    return path.strip().strip("`'\"，,。")


def extract_labeled(raw: str, labels: Iterable[str]) -> str | None:
    escaped = "|".join(re.escape(label) for label in sorted(labels, key=len, reverse=True))
    next_labels = []
    for values in LABELS.values():
        next_labels.extend(values)
    next_escaped = "|".join(re.escape(label) for label in sorted(next_labels, key=len, reverse=True))
    connector = r"(?:和|以及|and|with)"
    # 行尾命令词：用户自然语言里常见的"开始执行 / 执行 / 启动 / run / go / 开始"
    imperative = r"(?:开始执行|开始|执行|启动|run|go|start)"
    pattern = (
        rf"(?:{escaped})\s*(.+?)(?="
        rf"\s+(?:{connector}\s+)?(?:{next_escaped})"
        rf"|\s+{imperative}(?:\s|，|,|。|$)"
        rf"|\s*，|\s*,|$)"
    )
    match = re.search(pattern, raw, flags=re.IGNORECASE)
    if not match:
        return None
    return normalize_path(match.group(1))


def parse_user_input(raw: str) -> dict[str, str | None | bool]:
    return {
        "plan": extract_labeled(raw, LABELS["plan"]),
        "tasks": extract_labeled(raw, LABELS["tasks"]),
        "state": extract_labeled(raw, LABELS["state"]),
        "scope": extract_labeled(raw, LABELS["scope"]),
        "mode": "subagent-driven",
        "resume": bool(re.search(r"\bcontinue\b|\bresume\b|继续|续跑|恢复", raw, re.IGNORECASE)),
    }


def derive_topic(plan: str | None, tasks: str | None, scope: str | None) -> str:
    if plan:
        stem = Path(plan).stem
        if stem:
            return kebab(stem.replace("-plan", "").replace("_plan", ""))
    if tasks:
        path = Path(tasks)
        candidate = path.parent.name if path.name == "tasks.md" else path.stem
        if candidate:
            return kebab(candidate)
    if scope:
        return kebab(scope)
    return "sdd-run"


def relative_or_original(repo_root: Path, value: str | None) -> str | None:
    if not value:
        return None
    path = Path(value)
    if path.is_absolute():
        try:
            return str(path.resolve().relative_to(repo_root.resolve()))
        except ValueError:
            return str(path)
    return str(path)


def looks_like_path(value: str) -> bool:
    return "/" in value or "\\" in value or value.endswith((".md", ".txt", ".json", ".yaml", ".yml"))


def require_existing_path(repo_root: Path, value: str, label: str) -> None:
    path = Path(value)
    absolute = path if path.is_absolute() else repo_root / path
    if looks_like_path(value) and not absolute.exists():
        raise ValueError(f"{label} file does not exist: {value}")


def read_tasks(tasks_path: Path) -> list[tuple[str, str, bool]]:
    if not tasks_path.exists():
        return []
    rows: list[tuple[str, str, bool]] = []
    in_fence = False
    checklist_re = re.compile(
        r"^-\s+\[([ xX])\]\s+(T\d+(?:\.\d+)*)(?:[（(][^)）]+[)）])?[:：]?\s*(.*)"
    )
    heading_re = re.compile(
        r"^#{2,6}\s+(T\d+(?:\.\d+)*)(?:[（(][^)）]+[)）])?[:：]?\s*(.*)"
    )
    for line in tasks_path.read_text(encoding="utf-8").splitlines():
        stripped = line.strip()
        if stripped.startswith("```"):
            in_fence = not in_fence
            continue
        if in_fence:
            continue
        checklist = checklist_re.match(stripped)
        if checklist:
            box, task_id, title = checklist.groups()
            rows.append((task_id, title.strip(" -[]xX.") or "Imported task", box.lower() == "x"))
            continue
        heading = heading_re.match(stripped)
        if heading:
            task_id, title = heading.groups()
            rows.append((task_id, title.strip(" -[]xX.") or "Imported task", False))
    deduped: dict[str, tuple[str, bool]] = {}
    for task_id, title, done in rows:
        if task_id in deduped:
            existing_title, existing_done = deduped[task_id]
            # 任意一处标记完成则视为完成；保留首次出现的标题
            deduped[task_id] = (existing_title, existing_done or done)
        else:
            deduped[task_id] = (title, done)
    return [(task_id, title, done) for task_id, (title, done) in deduped.items()]


def filter_tasks_by_scope(
    task_items: list[tuple[str, str, bool]], scope: str | None
) -> list[tuple[str, str, bool]]:
    if not scope:
        return task_items
    scopes = re.findall(r"T\d+(?:\.\d+)*", scope)
    if not scopes:
        return task_items

    def included(task_id: str) -> bool:
        return any(task_id == scope_id or task_id.startswith(f"{scope_id}.") for scope_id in scopes)

    return [item for item in task_items if included(item[0])]


def sorted_existing_files(repo_root: Path, patterns: Iterable[str]) -> list[Path]:
    files: list[Path] = []
    for pattern in patterns:
        files.extend(path for path in repo_root.glob(pattern) if path.is_file())
    deduped = {path.resolve(): path for path in files}
    return sorted(deduped.values(), key=lambda path: path.stat().st_mtime, reverse=True)


def relative_candidates(repo_root: Path, paths: Iterable[Path], limit: int = 8) -> list[str]:
    return [str(path.resolve().relative_to(repo_root.resolve())) for path in list(paths)[:limit]]


def state_is_active(path: Path) -> bool:
    try:
        text = path.read_text(encoding="utf-8")
    except OSError:
        return False
    has_active_status = re.search(r"\|\s*Status\s*\|\s*`?active`?\s*\|", text) is not None
    has_pending_work = re.search(r"\|\s*Pending\s*\|\s*`?0`?\s*\|", text) is None
    return has_active_status and has_pending_work


def recover_inputs_from_state(path: Path) -> dict[str, str | None]:
    try:
        text = path.read_text(encoding="utf-8")
    except OSError:
        return {"plan": None, "tasks": None, "scope": None}

    def extract_row(label: str) -> str | None:
        pattern = rf"^\|\s*{re.escape(label)}\s*\|\s*`([^`-][^`]*)`\s*\|"
        match = re.search(pattern, text, flags=re.MULTILINE)
        return match.group(1) if match else None

    return {
        "plan": extract_row("Plan"),
        "tasks": extract_row("Tasks"),
        "scope": extract_row("Scope"),
    }


def infer_context(repo_root: Path, scope: str | None, mode: str | None) -> dict[str, object]:
    state_candidates = [
        path for path in sorted_existing_files(repo_root, ["docs/superpowers/sdd/runs/*state.md"]) if state_is_active(path)
    ]
    if len(state_candidates) == 1:
        state_rel = str(state_candidates[0].resolve().relative_to(repo_root.resolve()))
        return {
            "kind": "state",
            "state": state_rel,
            "inference_source": "active_state",
        }

    plan_candidates = sorted_existing_files(repo_root, ["docs/superpowers/plans/*.md"])
    task_candidates = sorted_existing_files(
        repo_root,
        [
            ".trae/specs/*/tasks.md",
            "specs/*/tasks.md",
        ],
    )

    if len(plan_candidates) == 1 and len(task_candidates) == 1:
        return {
            "kind": "plan_tasks",
            "plan": str(plan_candidates[0].resolve().relative_to(repo_root.resolve())),
            "tasks": str(task_candidates[0].resolve().relative_to(repo_root.resolve())),
            "scope": scope,
            "mode": mode or "subagent-driven",
            "inference_source": "single_plan_tasks_pair",
        }

    raise NeedsSelection(
        {
            "needs_selection": True,
            "reason": "No explicit state/plan/tasks were provided and context inference was not unique.",
            "state_candidates": relative_candidates(repo_root, state_candidates),
            "plan_candidates": relative_candidates(repo_root, plan_candidates),
            "task_candidates": relative_candidates(repo_root, task_candidates),
            "hint": "Run /sdd-run with state:<path>, or plan:<path> tasks:<path>.",
        }
    )


def input_doc_row(doc_type: str, path: str | None, required: str = "yes") -> str:
    loaded = "yes" if path else "no"
    return f"| {doc_type} | `{path or '-'}` | {required} | {loaded} |"


def build_state(
    *,
    repo_root: Path,
    topic: str,
    plan: str,
    tasks: str,
    scope: str | None,
    mode: str,
    branch: str,
    worktree: str,
) -> str:
    now = dt.datetime.now().strftime("%Y-%m-%d %H:%M")
    today = dt.date.today().isoformat()
    task_items = filter_tasks_by_scope(read_tasks(repo_root / tasks), scope)
    if not task_items:
        task_items = [("T001", "Imported execution task", False)]

    done_count = sum(1 for _, _, done in task_items if done)
    pending_count = len(task_items) - done_count

    def status_for(done: bool) -> str:
        return "done" if done else "pending"

    task_rows = "\n".join(
        f"| `{task_id}` | `{title}` | `{status_for(done)}` | `unassigned` | `W1` | `-` | `{scope or 'full plan'}` | `from tasks` |"
        for task_id, title, done in task_items
    )
    record_blocks = "\n\n".join(
        f"""### {task_id} - `{title}`

| Key | Value |
| --- | --- |
| Status | `{status_for(done)}` |
| Owner | `unassigned` |
| Started At | `-` |
| Completed At | `{'recovered' if done else '-'}` |
| Branch | `{branch}` |
| Worktree | `{worktree}` |
| Depends On | `-` |
| Parallel Group | `W1` |

**TDD Plan**

- Red: write or identify a failing test/contract check before implementation.
- Green: implement the minimum change that makes the test pass.
- Verify: run targeted and affected regression checks.

**Verification Evidence**

| Command | Expected | Actual | Result |
| --- | --- | --- | --- |
| `not_run` | `TDD Red -> Green -> Verify evidence` | `not_run` | `{status_for(done)}` |

**Handoff**

- First response line used: `{status_for(done)}`
"""
        for task_id, title, done in task_items
    )

    return f"""# SDD Run State - {topic}

> Auto-generated by `docs/superpowers/sdd/scripts/sdd_run.py` before `/sdd-run` execution.

## Run Metadata

| Key | Value |
| --- | --- |
| Run ID | `{today}-{topic}` |
| Topic | `{topic}` |
| Goal | `Execute plan/tasks through SDD/TDD` |
| Mode | `{mode}` |
| Branch | `{branch}` |
| Worktree | `{worktree}` |
| Base Branch | `main` |
| Started At | `{now}` |
| Owner | `main-agent` |
| Status | `active` |

## Input Documents

| Type | Path | Required | Loaded |
| --- | --- | --- | --- |
{input_doc_row("Agent Rules", "AGENTS.md")}
{input_doc_row("SDD Runbook", "docs/superpowers/sdd/RUNBOOK.md")}
{input_doc_row("State Template", "docs/superpowers/sdd/state-template.md")}
{input_doc_row("Plan", plan)}
{input_doc_row("Tasks", tasks)}
{input_doc_row("Scope", scope, "no")}

## Execution Summary

| Metric | Value |
| --- | --- |
| Total Tasks | `{len(task_items)}` |
| Done | `{done_count}` |
| Blocked | `0` |
| In Progress | `0` |
| Pending | `{pending_count}` |
| Last Updated | `{now}` |

## Task Matrix

| Task ID | Title | Status | Owner | Parallel Group | Depends On | Scope | Allowed Files |
| --- | --- | --- | --- | --- | --- | --- | --- |
{task_rows}

## Wave Plan

| Wave | Goal | Tasks | Start Condition | Completion Condition |
| --- | --- | --- | --- | --- |
| `W1` | `Execute imported tasks with TDD evidence` | `{','.join(task_id for task_id, _, _ in task_items)}` | `state file initialized` | `all tasks done or blocked with reason` |

## Task Records

{record_blocks}

## Final Review Checklist

- [ ] State file was created before subagent dispatch.
- [ ] Every implementation task records TDD Red -> Green -> Verify evidence.
- [ ] Every completed subagent response starts with `当前分支/worktree：`.
- [ ] Verification commands and results are recorded.

## Final Handoff

当前分支/worktree：{branch} @ {worktree}

**状态**

- `initialized`
"""


def ensure_state(args: argparse.Namespace) -> dict[str, object]:
    repo_root = Path(args.repo_root).resolve()
    parsed = parse_user_input(args.input or "")

    plan = normalize_path(args.plan) or parsed["plan"]
    tasks = normalize_path(args.tasks) or parsed["tasks"]
    state = normalize_path(args.state) or parsed["state"]
    scope = normalize_path(args.scope) or parsed["scope"]
    mode = normalize_path(args.mode) or parsed["mode"] or "subagent-driven"
    resume = bool(args.resume or parsed["resume"])
    inferred = False
    inference_source = None

    if not state and (not plan or not tasks):
        context = infer_context(repo_root, scope, str(mode))
        inferred = True
        inference_source = str(context["inference_source"])
        if context["kind"] == "state":
            state = str(context["state"])
            resume = True
        else:
            plan = str(context["plan"])
            tasks = str(context["tasks"])
            scope = str(context["scope"]) if context.get("scope") else scope
            mode = str(context["mode"])

    if state:
        state_path = Path(state)
        state_rel = relative_or_original(repo_root, str(state_path))
        absolute_state = state_path if state_path.is_absolute() else repo_root / state_path
        if absolute_state.exists():
            recovered = recover_inputs_from_state(absolute_state)
            plan = plan or recovered["plan"]
            tasks = tasks or recovered["tasks"]
            scope = scope or recovered["scope"]
            return {
                "created": False,
                "state_path": state_rel,
                "branch": run_git(repo_root, ["branch", "--show-current"], "unknown"),
                "worktree": str(repo_root),
                "plan": relative_or_original(repo_root, str(plan)) if plan else None,
                "tasks": relative_or_original(repo_root, str(tasks)) if tasks else None,
                "scope": scope,
                "mode": mode,
                "resume": True,
                "inferred": inferred,
                "inference_source": inference_source,
            }
        if resume:
            raise ValueError(f"state file does not exist: {state}")
    elif not plan or not tasks:
        raise ValueError("plan and tasks are required for a new /sdd-run without state")

    if not plan or not tasks:
        raise ValueError("plan and tasks are required for a new /sdd-run without state")

    require_existing_path(repo_root, str(plan), "plan")
    require_existing_path(repo_root, str(tasks), "tasks")

    plan_rel = relative_or_original(repo_root, str(plan))
    tasks_rel = relative_or_original(repo_root, str(tasks))
    topic = kebab(args.topic) if args.topic else derive_topic(plan_rel, tasks_rel, scope)
    date_prefix = dt.date.today().isoformat()
    state_rel = state or f"docs/superpowers/sdd/runs/{date_prefix}-{topic}-state.md"
    absolute_state = Path(state_rel)
    if not absolute_state.is_absolute():
        absolute_state = repo_root / absolute_state

    branch = run_git(repo_root, ["branch", "--show-current"], "unknown")
    worktree = run_git(repo_root, ["rev-parse", "--show-toplevel"], str(repo_root))
    absolute_state.parent.mkdir(parents=True, exist_ok=True)
    if absolute_state.exists() and not args.force:
        return {
            "created": False,
            "state_path": relative_or_original(repo_root, str(absolute_state)),
            "branch": branch,
            "worktree": worktree,
            "plan": plan_rel,
            "tasks": tasks_rel,
            "scope": scope,
            "mode": mode,
            "resume": resume,
            "inferred": inferred,
            "inference_source": inference_source,
        }

    content = build_state(
        repo_root=repo_root,
        topic=topic,
        plan=plan_rel or str(plan),
        tasks=tasks_rel or str(tasks),
        scope=scope,
        mode=str(mode),
        branch=branch,
        worktree=worktree,
    )
    absolute_state.write_text(content, encoding="utf-8")
    return {
        "created": True,
        "state_path": relative_or_original(repo_root, str(absolute_state)),
        "branch": branch,
        "worktree": worktree,
        "plan": plan_rel,
        "tasks": tasks_rel,
        "scope": scope,
        "mode": mode,
        "resume": resume,
        "inferred": inferred,
        "inference_source": inference_source,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Bootstrap /sdd-run state file")
    parser.add_argument("--repo-root", default=".")
    parser.add_argument("--input", default="")
    parser.add_argument("--plan")
    parser.add_argument("--tasks")
    parser.add_argument("--state")
    parser.add_argument("--scope")
    parser.add_argument("--mode")
    parser.add_argument("--topic")
    parser.add_argument("--resume", action="store_true")
    parser.add_argument("--force", action="store_true")
    parsed_args = parser.parse_args()

    try:
        payload = ensure_state(parsed_args)
    except NeedsSelection as exc:
        print(json.dumps(exc.payload, ensure_ascii=False, indent=2))
        return 3
    except ValueError as exc:
        print(str(exc), file=sys.stderr)
        return 2

    print(json.dumps(payload, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

import json
import subprocess
import sys
import tempfile
import unittest
import datetime as dt
from pathlib import Path


SCRIPT = Path(__file__).with_name("sdd_run.py")


class SddRunBootstrapTest(unittest.TestCase):
    def make_repo(self) -> Path:
        root = Path(tempfile.mkdtemp(prefix="sdd-run-test-"))
        (root / "docs/superpowers/sdd").mkdir(parents=True)
        (root / "docs/superpowers/plans").mkdir(parents=True)
        (root / ".trae/specs/example").mkdir(parents=True)
        (root / "docs/superpowers/sdd/state-template.md").write_text(
            "# SDD Run State Template\n\n## Run Metadata\n\n"
            "| Key | Value |\n| --- | --- |\n| Run ID | `<YYYY-MM-DD-topic>` |\n",
            encoding="utf-8",
        )
        (root / "docs/superpowers/plans/example-plan.md").write_text(
            "# Example Plan\n\nGoal: test state bootstrap.\n",
            encoding="utf-8",
        )
        (root / ".trae/specs/example/tasks.md").write_text(
            "# Tasks\n\n"
            "## T1 分页响应统一 `list`（P0）\n\n"
            "- [ ] T1.1 后端：product List 响应字段 `items` -> `list`\n"
            "- [ ] T1.2 测试：handler 层断言 JSON 结构含 `list`\n"
            "- T9 这是任务总览，不应被解析\n\n"
            "```text\nT10 ── dependency graph should be ignored\n```\n\n"
            "## T2 订单管理端语义\n\n"
            "- [ ] T2 订单管理端语义\n",
            encoding="utf-8",
        )
        return root

    def make_repo_with_single_context(self) -> Path:
        root = Path(tempfile.mkdtemp(prefix="sdd-run-test-"))
        (root / "docs/superpowers/sdd").mkdir(parents=True)
        (root / "docs/superpowers/plans").mkdir(parents=True)
        (root / ".trae/specs/only").mkdir(parents=True)
        (root / "docs/superpowers/sdd/state-template.md").write_text("# Template\n", encoding="utf-8")
        (root / "docs/superpowers/plans/only-plan.md").write_text("# Only Plan\n", encoding="utf-8")
        (root / ".trae/specs/only/tasks.md").write_text(
            "# Tasks\n\n## T1 Only task\n\n- [ ] T1.1 Verify inference\n",
            encoding="utf-8",
        )
        return root

    def run_script(self, repo: Path, user_input: str) -> subprocess.CompletedProcess:
        return subprocess.run(
            [
                sys.executable,
                str(SCRIPT),
                "--repo-root",
                str(repo),
                "--input",
                user_input,
            ],
            text=True,
            capture_output=True,
            check=False,
        )

    def test_creates_state_when_state_argument_is_missing(self):
        repo = self.make_repo()
        result = self.run_script(
            repo,
            "/sdd-run plan: docs/superpowers/plans/example-plan.md "
            "tasks: .trae/specs/example/tasks.md scope: 第一波",
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(result.stdout)
        state_path = repo / payload["state_path"]

        self.assertTrue(payload["created"])
        self.assertTrue(state_path.exists())
        text = state_path.read_text(encoding="utf-8")
        self.assertIn("docs/superpowers/plans/example-plan.md", text)
        self.assertIn(".trae/specs/example/tasks.md", text)
        self.assertIn("T1.1", text)
        self.assertIn("product List 响应字段", text)
        self.assertIn("当前分支/worktree：", text)

    def test_parses_project_task_ids(self):
        repo = self.make_repo()
        result = self.run_script(
            repo,
            "/sdd-run plan: docs/superpowers/plans/example-plan.md "
            "tasks: .trae/specs/example/tasks.md",
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(result.stdout)
        text = (repo / payload["state_path"]).read_text(encoding="utf-8")
        self.assertIn("`T1`", text)
        self.assertIn("`T1.1`", text)
        self.assertIn("`T1.2`", text)
        self.assertIn("`T2`", text)
        self.assertNotIn("`T9`", text)
        self.assertNotIn("`T10`", text)

    def test_scope_filters_task_ids_without_matching_t10_for_t1(self):
        repo = self.make_repo()
        result = self.run_script(
            repo,
            "/sdd-run plan: docs/superpowers/plans/example-plan.md "
            "tasks: .trae/specs/example/tasks.md scope: T1",
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(result.stdout)
        text = (repo / payload["state_path"]).read_text(encoding="utf-8")
        self.assertIn("`T1`", text)
        self.assertIn("`T1.1`", text)
        self.assertIn("`T1.2`", text)
        self.assertNotIn("`T2`", text)
        self.assertNotIn("`T10`", text)

    def test_parses_chinese_connector_between_plan_and_task(self):
        repo = self.make_repo()
        result = self.run_script(
            repo,
            "/sdd-run 这是本次开发的plan：docs/superpowers/plans/example-plan.md "
            "和 task：.trae/specs/example/tasks.md，开始执行",
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(result.stdout)
        self.assertEqual(payload["plan"], "docs/superpowers/plans/example-plan.md")
        self.assertEqual(payload["tasks"], ".trae/specs/example/tasks.md")

    def test_reuses_existing_state_for_resume(self):
        repo = self.make_repo()
        existing = repo / "docs/superpowers/sdd/runs/existing-state.md"
        existing.parent.mkdir(parents=True, exist_ok=True)
        existing.write_text("# Existing State\n", encoding="utf-8")

        result = self.run_script(
            repo,
            f"/sdd-run 继续执行 state: {existing.relative_to(repo)}",
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(result.stdout)
        self.assertFalse(payload["created"])
        self.assertEqual(payload["state_path"], str(existing.relative_to(repo)))
        self.assertEqual(existing.read_text(encoding="utf-8"), "# Existing State\n")

    def test_recovers_plan_tasks_scope_from_existing_state(self):
        repo = self.make_repo()
        existing = repo / "docs/superpowers/sdd/runs/existing-state.md"
        existing.parent.mkdir(parents=True, exist_ok=True)
        existing.write_text(
            "# Existing State\n\n"
            "## Input Documents\n\n"
            "| Type | Path | Required | Loaded |\n"
            "| --- | --- | --- | --- |\n"
            "| Plan | `docs/superpowers/plans/example-plan.md` | yes | yes |\n"
            "| Tasks | `.trae/specs/example/tasks.md` | yes | yes |\n"
            "| Scope | `T1` | no | yes |\n",
            encoding="utf-8",
        )

        result = self.run_script(
            repo,
            f"/sdd-run state: {existing.relative_to(repo)}",
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(result.stdout)
        self.assertFalse(payload["created"])
        self.assertTrue(payload["resume"])
        self.assertEqual(payload["plan"], "docs/superpowers/plans/example-plan.md")
        self.assertEqual(payload["tasks"], ".trae/specs/example/tasks.md")
        self.assertEqual(payload["scope"], "T1")

    def test_empty_sdd_run_does_not_reuse_single_active_state_without_resume_intent(self):
        repo = self.make_repo()
        existing = repo / "docs/superpowers/sdd/runs/active-state.md"
        existing.parent.mkdir(parents=True, exist_ok=True)
        existing.write_text(
            "# Active State\n\n| Status | `active` |\n| Pending | `1` |\n",
            encoding="utf-8",
        )

        result = self.run_script(repo, "/sdd-run")

        self.assertEqual(result.returncode, 3)
        payload = json.loads(result.stdout)
        self.assertTrue(payload["needs_selection"])
        self.assertEqual(payload["reason"], "active_state_requires_resume_intent")
        self.assertIn(str(existing.relative_to(repo)), payload["state_candidates"])

    def test_resume_intent_reuses_single_active_state(self):
        repo = self.make_repo()
        existing = repo / "docs/superpowers/sdd/runs/active-state.md"
        existing.parent.mkdir(parents=True, exist_ok=True)
        existing.write_text(
            "# Active State\n\n| Status | `active` |\n| Pending | `1` |\n",
            encoding="utf-8",
        )

        result = self.run_script(repo, "/sdd-run 继续")

        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(result.stdout)
        self.assertFalse(payload["created"])
        self.assertTrue(payload["inferred"])
        self.assertEqual(payload["inference_source"], "active_state")
        self.assertEqual(payload["state_path"], str(existing.relative_to(repo)))

    def test_negated_resume_words_do_not_enable_active_state_resume(self):
        repo = self.make_repo()
        existing = repo / "docs/superpowers/sdd/runs/active-state.md"
        existing.parent.mkdir(parents=True, exist_ok=True)
        existing.write_text(
            "# Active State\n\n| Status | `active` |\n| Pending | `1` |\n",
            encoding="utf-8",
        )

        result = self.run_script(repo, "/sdd-run 不要恢复旧 state")

        self.assertEqual(result.returncode, 3)
        payload = json.loads(result.stdout)
        self.assertTrue(payload["needs_selection"])
        self.assertEqual(payload["reason"], "active_state_requires_resume_intent")
        self.assertIn(str(existing.relative_to(repo)), payload["state_candidates"])

    def test_resume_intent_with_multiple_active_states_requires_selection(self):
        repo = self.make_repo()
        first = repo / "docs/superpowers/sdd/runs/active-a-state.md"
        second = repo / "docs/superpowers/sdd/runs/active-b-state.md"
        first.parent.mkdir(parents=True, exist_ok=True)
        first.write_text(
            "# Active A\n\n| Status | `active` |\n| Pending | `1` |\n",
            encoding="utf-8",
        )
        second.write_text(
            "# Active B\n\n| Status | `active` |\n| Pending | `2` |\n",
            encoding="utf-8",
        )

        result = self.run_script(repo, "/sdd-run 继续")

        self.assertEqual(result.returncode, 3)
        payload = json.loads(result.stdout)
        self.assertTrue(payload["needs_selection"])
        self.assertEqual(payload["reason"], "multiple_active_states_require_selection")
        self.assertIn(str(first.relative_to(repo)), payload["state_candidates"])
        self.assertIn(str(second.relative_to(repo)), payload["state_candidates"])

    def test_empty_sdd_run_creates_state_from_single_plan_and_tasks_pair(self):
        repo = self.make_repo_with_single_context()

        result = self.run_script(repo, "/sdd-run")

        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(result.stdout)
        self.assertTrue(payload["created"])
        self.assertTrue(payload["inferred"])
        self.assertEqual(payload["inference_source"], "single_plan_tasks_pair")
        self.assertEqual(payload["plan"], "docs/superpowers/plans/only-plan.md")
        self.assertEqual(payload["tasks"], ".trae/specs/only/tasks.md")
        self.assertTrue((repo / payload["state_path"]).exists())

    def test_empty_sdd_run_fails_with_multiple_candidates(self):
        repo = self.make_repo()
        (repo / "docs/superpowers/plans/another-plan.md").write_text("# Another\n", encoding="utf-8")

        result = self.run_script(repo, "/sdd-run")

        self.assertEqual(result.returncode, 3)
        payload = json.loads(result.stdout)
        self.assertTrue(payload["needs_selection"])
        self.assertIn("plan_candidates", payload)
        self.assertIn("task_candidates", payload)

    def test_scope_only_uses_unique_context_pair(self):
        repo = self.make_repo()
        result = self.run_script(repo, "/sdd-run scope: 第一波")

        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(result.stdout)
        self.assertTrue(payload["created"])
        self.assertTrue(payload["inferred"])
        self.assertEqual(payload["scope"], "第一波")

    def test_completed_tasks_are_marked_done_not_pending(self):
        repo = self.make_repo_with_single_context()
        (repo / ".trae/specs/only/tasks.md").write_text(
            "# Tasks\n\n"
            "- [x] T1 done task\n"
            "- [ ] T2 todo task\n",
            encoding="utf-8",
        )
        result = self.run_script(repo, "/sdd-run")

        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(result.stdout)
        text = (repo / payload["state_path"]).read_text(encoding="utf-8")
        self.assertRegex(text, r"\|\s*`T1`\s*\|\s*`done task`\s*\|\s*`done`")
        self.assertRegex(text, r"\|\s*`T2`\s*\|\s*`todo task`\s*\|\s*`pending`")
        self.assertRegex(text, r"\|\s*Done\s*\|\s*`1`\s*\|")
        self.assertRegex(text, r"\|\s*Pending\s*\|\s*`1`\s*\|")

    def test_recovers_inputs_from_build_state_real_output(self):
        repo = self.make_repo_with_single_context()
        # 第一次：让脚本用 build_state 生成一份真实 state
        first = self.run_script(repo, "/sdd-run")
        self.assertEqual(first.returncode, 0, first.stderr)
        state_rel = json.loads(first.stdout)["state_path"]

        # 第二次：仅传 state 路径，期望从 build_state 4 列输出中恢复 plan/tasks
        second = self.run_script(repo, f"/sdd-run state: {state_rel}")
        self.assertEqual(second.returncode, 0, second.stderr)
        payload = json.loads(second.stdout)
        self.assertFalse(payload["created"])
        self.assertTrue(payload["resume"])
        self.assertEqual(payload["plan"], "docs/superpowers/plans/only-plan.md")
        self.assertEqual(payload["tasks"], ".trae/specs/only/tasks.md")

    def test_state_with_pending_zero_is_not_treated_as_active(self):
        repo = self.make_repo_with_single_context()
        finished = repo / "docs/superpowers/sdd/runs/finished-state.md"
        finished.parent.mkdir(parents=True, exist_ok=True)
        finished.write_text(
            "# Finished\n\n| Status | `active` |\n| Pending | `0` |\n",
            encoding="utf-8",
        )
        result = self.run_script(repo, "/sdd-run")

        # 不应把 Pending=0 的 state 当作 active 候选；应回退到唯一 plan/tasks 推断
        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(result.stdout)
        self.assertEqual(payload["inference_source"], "single_plan_tasks_pair")

    def test_state_missing_pending_metric_is_not_treated_as_active(self):
        repo = self.make_repo_with_single_context()
        legacy = repo / "docs/superpowers/sdd/runs/legacy-active-state.md"
        legacy.parent.mkdir(parents=True, exist_ok=True)
        legacy.write_text(
            "# Legacy Active State\n\n| Status | `active` |\n",
            encoding="utf-8",
        )

        result = self.run_script(repo, "/sdd-run")

        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(result.stdout)
        self.assertEqual(payload["inference_source"], "single_plan_tasks_pair")
        self.assertNotEqual(payload["state_path"], str(legacy.relative_to(repo)))

    def test_explicit_plan_tasks_does_not_silently_reuse_existing_default_state(self):
        repo = self.make_repo()
        existing = repo / (
            "docs/superpowers/sdd/runs/"
            f"{dt.date.today().isoformat()}-example-state.md"
        )
        existing.parent.mkdir(parents=True, exist_ok=True)
        existing.write_text("# Existing default topic state\n", encoding="utf-8")

        result = self.run_script(
            repo,
            "/sdd-run plan: docs/superpowers/plans/example-plan.md "
            "tasks: .trae/specs/example/tasks.md",
        )

        self.assertEqual(result.returncode, 3)
        payload = json.loads(result.stdout)
        self.assertTrue(payload["needs_selection"])
        self.assertEqual(payload["reason"], "existing_state_path_collision")
        self.assertEqual(payload["state_path"], str(existing.relative_to(repo)))
        self.assertFalse(payload["created"])

    def test_label_trailing_imperative_does_not_swallow_action_words(self):
        repo = self.make_repo()
        result = self.run_script(
            repo,
            "/sdd-run plan: docs/superpowers/plans/example-plan.md "
            "tasks: .trae/specs/example/tasks.md 开始执行",
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        payload = json.loads(result.stdout)
        # 行尾命令词不应被吞进 tasks 路径
        self.assertEqual(payload["tasks"], ".trae/specs/example/tasks.md")
        self.assertEqual(payload["plan"], "docs/superpowers/plans/example-plan.md")

    def test_bare_label_words_in_prose_are_not_extracted(self):
        repo = self.make_repo()
        # 句子里出现裸词 "plan"、"task" 但无冒号，不应被当作标签匹配
        result = self.run_script(
            repo,
            "/sdd-run state: docs/superpowers/sdd/runs/missing.md 我有一个 plan 和 task 想恢复",
        )
        # 关键断言：不应把 "plan 和 task" 等自然语言当成 plan/tasks 标签提取
        # 由于 state 文件不存在，应当 fail-fast 并提示 state file does not exist
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("state file does not exist", result.stderr)

    def test_new_run_rejects_missing_plan_or_tasks_path(self):
        repo = self.make_repo()
        result = self.run_script(
            repo,
            "/sdd-run plan: docs/superpowers/plans/missing.md "
            "tasks: .trae/specs/example/tasks.md",
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("plan file does not exist", result.stderr)


if __name__ == "__main__":
    unittest.main()

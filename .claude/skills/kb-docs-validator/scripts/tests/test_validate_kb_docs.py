#!/usr/bin/env python3
from __future__ import annotations

import importlib.util
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT_PATH = Path(__file__).resolve().parents[1] / "validate_kb_docs.py"
SPEC = importlib.util.spec_from_file_location("validate_kb_docs", SCRIPT_PATH)
assert SPEC is not None and SPEC.loader is not None
validator = importlib.util.module_from_spec(SPEC)
sys.modules["validate_kb_docs"] = validator
SPEC.loader.exec_module(validator)


class RuleDocQualityTests(unittest.TestCase):
    def test_medium_rule_doc_over_500_lines_is_hard_error(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "CLAUDE.md").write_text("# Module\n", encoding="utf-8")
            (root / "Interface").mkdir()
            docs = root / "docs"
            docs.mkdir()
            rule = docs / "rule.md"
            rule.write_text(
                "# Rules\n"
                + "\n".join(f"line {idx}" for idx in range(501)),
                encoding="utf-8",
            )

            issues = validator._check_rule_doc_quality(rule, "docs/rule.md", root)

            titles = [issue.title for issue in issues]
            self.assertIn("docs/rule.md exceeds hard length limit", titles)
            self.assertTrue(any("scope=medium" in issue.details for issue in issues))

    def test_table_pseudo_rules_are_flagged(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "CLAUDE.md").write_text("# Module\n", encoding="utf-8")
            docs = root / "docs"
            docs.mkdir()
            rule = docs / "rule.md"
            rule.write_text(
                "\n".join(
                    [
                        "# Rules",
                        "| Rule | Rationale |",
                        "|------|-----------|",
                        "| Show success feedback | User knows it worked |",
                    ]
                ),
                encoding="utf-8",
            )

            issues = validator._check_rule_doc_quality(rule, "docs/rule.md", root)

            titles = [issue.title for issue in issues]
            self.assertIn("docs/rule.md contains table-based pseudo rules", titles)
            self.assertIn("docs/rule.md has no scenario trigger", titles)

    def test_compact_local_rule_with_code_example_is_allowed(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "CLAUDE.md").write_text("# Module\n", encoding="utf-8")
            (root / "GuideCamera.m").write_text(
                "[registry addFragment:GBLGuideCameraFragment.class];\n",
                encoding="utf-8",
            )
            docs = root / "docs"
            docs.mkdir()
            rule = docs / "rule.md"
            rule.write_text(
                "\n".join(
                    [
                        "# Rules",
                        "## Register Fragment",
                        "WHEN adding a camera guide fragment:",
                        "- MUST register it through `GuideCamera.plist`.",
                        "- MUST NOT instantiate the fragment from another component.",
                        "Example from GuideCamera.m:",
                        "```objc",
                        "[registry addFragment:GBLGuideCameraFragment.class];",
                        "```",
                    ]
                ),
                encoding="utf-8",
            )

            issues = validator._check_rule_doc_quality(rule, "docs/rule.md", root)

            titles = [issue.title for issue in issues]
            self.assertNotIn("docs/rule.md contains many code examples", titles)
            self.assertNotIn("docs/rule.md has code examples without source paths", titles)
            self.assertNotIn("docs/rule.md has code examples with missing source paths", titles)
            self.assertNotIn("docs/rule.md has no scenario trigger", titles)
            self.assertNotIn("docs/rule.md has no MUST/MUST NOT actions", titles)

    def test_code_example_without_source_path_is_flagged(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "CLAUDE.md").write_text("# Module\n", encoding="utf-8")
            docs = root / "docs"
            docs.mkdir()
            rule = docs / "rule.md"
            rule.write_text(
                "\n".join(
                    [
                        "# Rules",
                        "## Register Fragment",
                        "WHEN adding a camera guide fragment:",
                        "- MUST register it through `GuideCamera.plist`.",
                        "- MUST NOT instantiate the fragment from another component.",
                        "Example:",
                        "```objc",
                        "[registry addFragment:GBLGuideCameraFragment.class];",
                        "```",
                    ]
                ),
                encoding="utf-8",
            )

            issues = validator._check_rule_doc_quality(rule, "docs/rule.md", root)

            titles = [issue.title for issue in issues]
            self.assertIn("docs/rule.md has code examples without source paths", titles)

    def test_generic_performance_stability_terms_are_flagged(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "CLAUDE.md").write_text("# Module\n", encoding="utf-8")
            docs = root / "docs"
            docs.mkdir()
            rule = docs / "rule.md"
            rule.write_text(
                "\n".join(
                    [
                        "# Rules",
                        "## Reliability",
                        "WHEN changing loader behavior:",
                        "- MUST improve performance and reliability.",
                        "- MUST NOT reduce stability.",
                    ]
                ),
                encoding="utf-8",
            )

            issues = validator._check_rule_doc_quality(rule, "docs/rule.md", root)

            titles = [issue.title for issue in issues]
            self.assertIn("docs/rule.md contains generic rule text", titles)

    def test_duplicate_parent_rule_scenarios_are_flagged(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            parent = Path(tmp) / "Parent"
            child = parent / "Child"
            (parent / "docs").mkdir(parents=True)
            (child / "docs").mkdir(parents=True)
            (child / "CLAUDE.md").write_text("# Child\n", encoding="utf-8")

            (parent / "docs" / "rule.md").write_text(
                "\n".join(
                    [
                        "# Rules",
                        "## Register Fragment",
                        "WHEN adding a camera guide fragment:",
                        "- MUST register it through the owning plist.",
                    ]
                ),
                encoding="utf-8",
            )
            rule = child / "docs" / "rule.md"
            rule.write_text(
                "\n".join(
                    [
                        "# Rules",
                        "## Register Fragment",
                        "WHEN adding a camera guide fragment:",
                        "- MUST register it through `GuideCamera.plist`.",
                        "- MUST NOT bypass parent registration rules.",
                    ]
                ),
                encoding="utf-8",
            )

            issues = validator._check_rule_doc_quality(rule, "docs/rule.md", child)

            titles = [issue.title for issue in issues]
            self.assertIn("docs/rule.md duplicates ancestor rule scenarios", titles)

    def test_kotlin_source_files_promote_scope_to_medium(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "CLAUDE.md").write_text("# Module\n", encoding="utf-8")
            for idx in range(100):
                (root / f"Feature{idx}.kt").write_text(
                    f"class Feature{idx}\n",
                    encoding="utf-8",
                )

            scope = validator._detect_rule_scope(root)

            self.assertEqual(scope.name, "medium")
            self.assertEqual(scope.code_files, 100)


if __name__ == "__main__":
    unittest.main()

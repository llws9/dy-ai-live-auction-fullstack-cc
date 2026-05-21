#!/usr/bin/env python3
"""kb-docs-validator

Validate and reconcile kb-init-docs knowledge docs across one or multiple doc roots.

Outputs a Markdown report to stdout.

Design goals:
- Fast: do filesystem checks locally; avoid expensive repo-wide scans.
- Deterministic: prefer code-derived signals (via kb_tool.py) over doc content.
- Non-destructive: report by default; do not auto-edit files.

Usage:
  python3 validate_kb_docs.py <doc_root1> [doc_root2 ...]
"""

from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Optional, Sequence, Tuple


DEFAULT_KB_INIT_SCRIPTS_DIR = Path(__file__).resolve().parent.parent.parent / "kb-init-docs" / "scripts"
DEFAULT_KB_TOOL = str(DEFAULT_KB_INIT_SCRIPTS_DIR / "kb_tool.py")
DEFAULT_TOP = 20

if str(DEFAULT_KB_INIT_SCRIPTS_DIR) not in sys.path:
    sys.path.insert(0, str(DEFAULT_KB_INIT_SCRIPTS_DIR))

try:
    from kb_common import SOURCE_EXTS as KB_SOURCE_EXTS
except Exception:  # noqa: BLE001
    KB_SOURCE_EXTS = {
        ".c",
        ".cc",
        ".cpp",
        ".h",
        ".hpp",
        ".java",
        ".kt",
        ".kts",
        ".m",
        ".mm",
        ".swift",
    }


@dataclass
class Issue:
    severity: str  # INFO|WARN|ERROR
    title: str
    details: str


@dataclass
class RuleScope:
    name: str  # small|medium|large
    target_lines: int
    warn_lines: int
    strong_lines: int
    child_doc_roots: int
    code_files: int
    direct_subdirs: int
    reasons: List[str]


CODE_EXTENSIONS = set(KB_SOURCE_EXTS)

RULE_SCOPE_THRESHOLDS: Dict[str, Tuple[int, int, int]] = {
    "small": (80, 120, 200),
    "medium": (120, 180, 300),
    "large": (180, 250, 500),
}

AGGREGATE_DOC_ROOT_NAMES = {
    "Component",
    "Components",
    "Foundation",
    "Foundations",
    "LiveRoom",
    "Service",
    "Services",
}


def _read_text(path: Path) -> Optional[str]:
    try:
        return path.read_text(encoding="utf-8")
    except FileNotFoundError:
        return None


def _is_symlink_to(path: Path, target_name: str) -> bool:
    if not path.exists() and not path.is_symlink():
        return False
    if not path.is_symlink():
        return False
    try:
        return Path(os.readlink(path)).name == target_name
    except OSError:
        return False


def _normalize_md(s: str) -> str:
    # normalize line endings + trim trailing spaces
    return "\n".join(line.rstrip() for line in s.replace("\r\n", "\n").split("\n")).strip()


def _extract_markdown_table_names(md: str) -> List[str]:
    # Extract names in markdown table cells like: | `Name` | ... |
    names: List[str] = []
    for line in md.splitlines():
        m = re.search(r"\|\s*`([^`]+)`\s*\|", line)
        if m:
            names.append(m.group(1).strip())
    return names


def _extract_claude_key_classes(md: str) -> List[str]:
    # Best-effort: try to locate the Key Classes section; fall back to table extraction.
    # Supports both the new 3-section format and older variants.
    section_pat = re.compile(r"^##\s+Key\s+Classes\s*$", re.MULTILINE)
    m = section_pat.search(md)
    if not m:
        return _extract_markdown_table_names(md)

    start = m.end()
    # Find next H2
    next_h2 = re.search(r"^##\s+", md[start:], re.MULTILINE)
    body = md[start : start + next_h2.start()] if next_h2 else md[start:]
    return _extract_markdown_table_names(body)


def _run_kb_tool_select_key_classes(doc_root: Path, top: int) -> Tuple[List[str], Optional[str]]:
    if not Path(DEFAULT_KB_TOOL).exists():
        return [], f"kb_tool.py not found at: {DEFAULT_KB_TOOL}"

    cmd = [sys.executable, DEFAULT_KB_TOOL, "select-key-classes", str(doc_root), "--top", str(top)]
    try:
        p = subprocess.run(cmd, check=False, capture_output=True, text=True)
    except Exception as e:  # noqa: BLE001
        return [], f"failed to run kb_tool.py: {e}"

    if p.returncode != 0:
        msg = p.stderr.strip() or p.stdout.strip() or f"exit={p.returncode}"
        return [], f"kb_tool.py failed: {msg}"

    # Parse lines like:
    # - `TTKFriendsRootViewController` (objc_class) — ...
    key_classes: List[str] = []
    for line in p.stdout.splitlines():
        line = line.strip()
        if not line.startswith("- `"):
            continue
        m = re.match(r"-\s+`([^`]+)`\s+\(", line)
        if m:
            key_classes.append(m.group(1))

    return key_classes, None


def _find_entry_headers(doc_root: Path) -> List[Path]:
    # Code-as-truth heuristic: the module entry commonly includes *ModuleService.h
    # Keep it shallow by limiting to a couple expected locations.
    candidates: List[Path] = []
    for pat in [
        "**/ModuleInterface/Interface/*ModuleService.h",
        "**/ModuleInterface/Interface/*Service.h",
    ]:
        candidates.extend(doc_root.glob(pat))
    # Deduplicate by resolved path if possible
    uniq: List[Path] = []
    seen = set()
    for p in candidates:
        key = str(p)
        if key in seen:
            continue
        seen.add(key)
        uniq.append(p)
    return uniq[:20]


def _is_code_file(path: Path) -> bool:
    return path.suffix in CODE_EXTENSIONS


def _count_child_doc_roots(doc_root: Path) -> int:
    count = 0
    for claude in doc_root.rglob("CLAUDE.md"):
        if claude.parent == doc_root:
            continue
        count += 1
    return count


def _count_code_files(doc_root: Path) -> int:
    return sum(1 for path in doc_root.rglob("*") if path.is_file() and _is_code_file(path))


def _count_direct_subdirs(doc_root: Path) -> int:
    try:
        return sum(1 for path in doc_root.iterdir() if path.is_dir())
    except OSError:
        return 0


def _detect_rule_scope(doc_root: Path) -> RuleScope:
    child_doc_roots = _count_child_doc_roots(doc_root)
    code_files = _count_code_files(doc_root)
    direct_subdirs = _count_direct_subdirs(doc_root)
    direct_names = {path.name for path in doc_root.iterdir() if path.is_dir()}
    reasons: List[str] = []

    # Prefer explicit knowledge-doc hierarchy over raw code size: large feature
    # modules can have many files but should still keep feature-level rule docs concise.
    if child_doc_roots >= 3:
        name = "large"
        reasons.append(f"{child_doc_roots} child doc roots")
    elif doc_root.name in AGGREGATE_DOC_ROOT_NAMES:
        name = "large"
        reasons.append(f"aggregate doc-root name `{doc_root.name}`")
    elif child_doc_roots >= 1:
        name = "medium"
        reasons.append(f"{child_doc_roots} child doc root")
    elif {"Interface", "Module", "Fragments"} & direct_names:
        name = "medium"
        reasons.append("known local module-layer layout")
    elif code_files >= 80:
        name = "medium"
        reasons.append(f"{code_files} code files")
    elif direct_subdirs >= 8:
        name = "medium"
        reasons.append(f"{direct_subdirs} direct subdirectories")
    else:
        name = "small"
        reasons.append("leaf doc root")

    target, warn, strong = RULE_SCOPE_THRESHOLDS[name]
    return RuleScope(
        name=name,
        target_lines=target,
        warn_lines=warn,
        strong_lines=strong,
        child_doc_roots=child_doc_roots,
        code_files=code_files,
        direct_subdirs=direct_subdirs,
        reasons=reasons,
    )


def _normalize_rule_signature(text: str) -> str:
    text = re.sub(r"`([^`]+)`", r"\1", text)
    text = re.sub(r"[^0-9A-Za-z_\u4e00-\u9fff]+", " ", text)
    return " ".join(text.lower().split())


def _extract_rule_signatures(txt: str) -> Dict[str, List[str]]:
    headings: List[str] = []
    whens: List[str] = []
    for line in txt.splitlines():
        heading = re.match(r"^##\s+(.+?)\s*$", line)
        if heading:
            normalized = _normalize_rule_signature(heading.group(1))
            if normalized:
                headings.append(normalized)
            continue
        when = re.match(r"^WHEN\s+(.+?):\s*$", line)
        if when:
            normalized = _normalize_rule_signature(when.group(1))
            if normalized:
                whens.append(normalized)
    return {"headings": headings, "whens": whens}


def _find_ancestor_rule_docs(doc_root: Path) -> List[Path]:
    docs: List[Path] = []
    seen = set()
    for ancestor in doc_root.parents:
        for candidate in (ancestor / "docs" / "rule.md", ancestor / "rule.md"):
            if not candidate.exists() or not candidate.is_file():
                continue
            key = str(candidate.resolve())
            if key in seen:
                continue
            seen.add(key)
            docs.append(candidate)
    return docs


def _resolve_example_path(raw_path: str, doc_root: Path) -> Path:
    raw_path = raw_path.strip().strip("`").strip()
    candidate = Path(raw_path).expanduser()
    if candidate.is_absolute():
        return candidate

    doc_root_candidate = (doc_root / candidate).resolve()
    if doc_root_candidate.exists():
        return doc_root_candidate

    for ancestor in (doc_root, *doc_root.parents):
        repo_candidate = (ancestor / candidate).resolve()
        if repo_candidate.exists():
            return repo_candidate
    return doc_root_candidate


def _check_code_example_sources(txt: str, doc_root: Path, label: str) -> List[Issue]:
    lines = txt.splitlines()
    issues: List[Issue] = []
    missing_sources = 0
    missing_paths: List[str] = []
    in_code_block = False

    for idx, line in enumerate(lines):
        if not line.strip().startswith("```"):
            continue
        if in_code_block:
            in_code_block = False
            continue
        in_code_block = True
        previous = lines[idx - 1].strip() if idx > 0 else ""
        previous2 = lines[idx - 2].strip() if idx > 1 else ""
        source_line = previous if previous else previous2
        match = re.match(r"(?i)^(?:bad\s+)?example\s+from\s+(.+?):\s*$", source_line)
        if not match:
            missing_sources += 1
            continue
        source_path = match.group(1).strip()
        if not _resolve_example_path(source_path, doc_root).exists():
            missing_paths.append(source_path)

    if missing_sources:
        issues.append(
            Issue(
                "WARN",
                f"{label} has code examples without source paths",
                f"Found {missing_sources} code fence(s) without `Example from <path>:` or `Bad example from <path>:` immediately before them.",
            )
        )
    if missing_paths:
        issues.append(
            Issue(
                "WARN",
                f"{label} has code examples with missing source paths",
                "Example source paths not found: "
                + ", ".join(f"`{path}`" for path in missing_paths[:5])
                + ". Use real repo files for examples.",
            )
        )

    return issues


def _check_rule_doc_quality(rule_path: Path, label: str, doc_root: Path) -> List[Issue]:
    issues: List[Issue] = []
    txt = _read_text(rule_path)
    if txt is None or not txt.strip():
        return issues

    lines = txt.splitlines()
    line_count = len(lines)
    scope = _detect_rule_scope(doc_root)
    when_count = sum(1 for line in lines if re.match(r"^WHEN\s+.+:", line))
    must_count = sum(1 for line in lines if re.match(r"^\s*-\s+MUST\s+", line))
    must_not_count = sum(1 for line in lines if re.match(r"^\s*-\s+MUST\s+NOT\s+", line))
    table_lines = sum(1 for line in lines if line.strip().startswith("|"))
    code_fences = sum(1 for line in lines if line.strip().startswith("```"))

    has_when = re.search(r"(?m)^WHEN\s+.+:", txt) is not None
    has_must = must_count > 0
    has_must_not = must_not_count > 0

    scope_details = (
        f"Detected scope={scope.name} ({', '.join(scope.reasons)}; "
        f"child_doc_roots={scope.child_doc_roots}, code_files={scope.code_files}, "
        f"direct_subdirs={scope.direct_subdirs}). "
        f"Thresholds: target<={scope.target_lines}, warn>{scope.warn_lines}, "
        f"strong>{scope.strong_lines}, hard_error>500."
    )

    if line_count > 500:
        issues.append(
            Issue(
                "ERROR",
                f"{label} exceeds hard length limit",
                f"Lines={line_count}. {scope_details}",
            )
        )
    elif line_count > scope.strong_lines:
        issues.append(
            Issue(
                "WARN",
                f"{label} is strongly oversized for {scope.name} doc root",
                f"Lines={line_count}. {scope_details}",
            )
        )
    elif line_count > scope.warn_lines:
        issues.append(
            Issue(
                "WARN",
                f"{label} is oversized for {scope.name} doc root",
                f"Lines={line_count}. {scope_details}",
            )
        )

    if not has_when:
        issues.append(
            Issue(
                "WARN",
                f"{label} has no scenario trigger",
                "Expected at least one `WHEN <specific local scenario>:` block.",
            )
        )
    if not has_must and not has_must_not:
        issues.append(
            Issue(
                "WARN",
                f"{label} has no MUST/MUST NOT actions",
                "Expected actionable bullets starting with `MUST` or `MUST NOT`.",
            )
        )

    own_signatures = _extract_rule_signatures(txt)
    parent_duplicates: List[str] = []
    for parent_rule in _find_ancestor_rule_docs(doc_root):
        parent_txt = _read_text(parent_rule) or ""
        parent_signatures = _extract_rule_signatures(parent_txt)
        for kind, label_name in (("headings", "heading"), ("whens", "WHEN")):
            overlap = sorted(set(own_signatures[kind]) & set(parent_signatures[kind]))
            for signature in overlap[:3]:
                parent_duplicates.append(f"{label_name} `{signature}` already exists in {parent_rule}")
    if parent_duplicates:
        issues.append(
            Issue(
                "WARN",
                f"{label} duplicates ancestor rule scenarios",
                "Child rule docs should not restate parent/global rules. "
                + "; ".join(parent_duplicates[:5]),
            )
        )

    misplaced_section_patterns = [
        r"(?mi)^#{2,3}\s*(模块结构|目录结构|代码组织|文件结构|命名约定|文档编写)\s*$",
        r"(?mi)^#{2,3}\s*(Module Structure|Directory Structure|Code Organization|File Structure|Naming Conventions|Documentation)\s*$",
        r"(?mi)^#{2,3}\s*(Testing Requirements|Dependencies)\s*$",
    ]
    found_misplaced = [
        pat
        for pat in misplaced_section_patterns
        if re.search(pat, txt)
    ]
    if found_misplaced:
        issues.append(
            Issue(
                "WARN",
                f"{label} contains misplaced structure or inventory sections",
                "Module structure belongs in `CLAUDE.md` summary; dependency/test/name inventories "
                "should not live in rule.md unless narrowed into enforceable local constraints.",
            )
        )

    pseudo_rule_table_patterns = [
        r"(?mi)^\|\s*Rule\s*\|\s*(Rationale|Implementation|Details)\s*\|",
        r"(?mi)^\|\s*Feature Area\s*\|\s*Test Focus\s*\|",
        r"(?mi)^\|\s*Dependency\s*\|\s*Used For\s*\|",
    ]
    found_pseudo_tables = [
        pat
        for pat in pseudo_rule_table_patterns
        if re.search(pat, txt)
    ]
    if found_pseudo_tables:
        issues.append(
            Issue(
                "WARN",
                f"{label} contains table-based pseudo rules",
                "Convert keep-worthy rows into `WHEN` scenario blocks with local evidence; "
                "remove product notes, test matrices, and dependency inventories.",
            )
        )

    generic_patterns = [
        r"write clean code",
        r"follow best practices",
        r"handle errors",
        r"pay attention to performance",
        r"ensure code quality",
        r"be careful",
        r"good UX",
        r"user guidance",
        r"user knows",
        r"memory management",
        r"complete coverage",
        r"smooth switching",
        r"show success feedback",
        r"\bperformance\b",
        r"\bstability\b",
        r"\brobustness\b",
        r"\breliability\b",
        r"性能",
        r"稳定性",
        r"健壮性",
        r"可靠性",
        r"最佳实践",
        r"代码质量",
        r"用户体验",
    ]
    found_generic = [
        pat
        for pat in generic_patterns
        if re.search(pat, txt, flags=re.IGNORECASE)
    ]
    if found_generic:
        issues.append(
            Issue(
                "INFO",
                f"{label} contains generic rule text",
                "Generic phrases found: "
                + ", ".join(f"`{p}`" for p in found_generic)
                + ". Prefer local, evidence-backed rules or remove them.",
            )
        )

    if code_fences:
        issues.extend(_check_code_example_sources(txt, doc_root, label))

    if line_count > scope.warn_lines and when_count == 0:
        issues.append(
            Issue(
                "WARN",
                f"{label} has low scenario-rule density",
                f"Lines={line_count}, WHEN blocks=0, code fences={code_fences}, table lines={table_lines}. "
                "Long rule docs need explicit scenario blocks; code examples are allowed only as compact "
                "evidence for nearby local constraints.",
            )
        )
    elif when_count > 0:
        lines_per_when = line_count / when_count
        if line_count > scope.warn_lines and lines_per_when > 60:
            issues.append(
                Issue(
                    "INFO",
                    f"{label} may have low scenario-rule density",
                    f"Lines={line_count}, WHEN blocks={when_count}, approx lines per scenario={lines_per_when:.1f}. "
                    "Consider pruning examples or splitting broad scenarios.",
                )
            )

    if code_fences > max(6, when_count * 2 + 2) and line_count > scope.warn_lines:
        issues.append(
            Issue(
                "WARN",
                f"{label} contains many code examples",
                f"Code fence count={code_fences}. Keep short positive/negative examples only when they support "
                "nearby local `MUST` / `MUST NOT` constraints.",
            )
        )

    return issues


def _check_doc_root(doc_root: Path) -> List[Issue]:
    issues: List[Issue] = []

    claude = doc_root / "CLAUDE.md"
    agent = doc_root / "AGENTS.md"

    docs_dir = doc_root / "docs"
    docs_files = {
        "interface": docs_dir / "interface.md",
        "workflow": docs_dir / "workflow.md",
        "domain": docs_dir / "domain.md",
        "rule": docs_dir / "rule.md",
    }

    legacy_files = {
        "interface": doc_root / "interface.md",
        "workflow": doc_root / "workflow.md",
        "domain": doc_root / "domain.md",
        "rule": doc_root / "rule.md",
    }

    # Basic presence checks
    if not claude.exists():
        issues.append(Issue("ERROR", "Missing CLAUDE.md", "Expected CLAUDE.md at doc root."))
    if agent.exists() or agent.is_symlink():
        if not _is_symlink_to(agent, "CLAUDE.md"):
            issues.append(
                Issue(
                    "WARN",
                    "AGENTS.md is not a symlink to CLAUDE.md",
                    "Expected AGENTS.md -> CLAUDE.md for consistency.",
                )
            )
    else:
        issues.append(Issue("INFO", "AGENTS.md missing", "Optional but recommended: AGENTS.md -> CLAUDE.md"))

    has_docs_dir = docs_dir.exists() and docs_dir.is_dir()
    has_any_docs = any(p.exists() for p in docs_files.values())
    has_any_legacy = any(p.exists() for p in legacy_files.values())

    if has_any_docs and not has_docs_dir:
        issues.append(Issue("ERROR", "docs/* exists but docs/ is not a directory", "Check filesystem state."))

    if has_any_docs and has_any_legacy:
        # Compare each pair
        for k in ["interface", "workflow", "domain", "rule"]:
            d = docs_files[k]
            l = legacy_files[k]
            if not d.exists() or not l.exists():
                continue
            d_txt = _read_text(d)
            l_txt = _read_text(l)
            if d_txt is None or l_txt is None:
                continue
            if _normalize_md(d_txt) != _normalize_md(l_txt):
                issues.append(
                    Issue(
                        "WARN",
                        f"Conflict: docs/{k}.md differs from root {k}.md",
                        "Two versions exist. Prefer code as truth; ask human which doc should be canonical.",
                    )
                )
            else:
                issues.append(
                    Issue(
                        "INFO",
                        f"Duplication: docs/{k}.md identical to root {k}.md",
                        "Consider keeping a single canonical copy to avoid drift.",
                    )
                )

    # Reference checks in CLAUDE.md/README.md
    claude_txt = _read_text(claude) or ""
    readme_txt = _read_text(doc_root / "README.md") or ""

    def _check_link(md: str, label: str, rel: str, must_exist: bool = True) -> None:
        if rel not in md:
            return
        target = (doc_root / rel).resolve() if not rel.startswith("./") else (doc_root / rel[2:]).resolve()
        if must_exist and not target.exists():
            issues.append(Issue("WARN", f"Broken reference in {label}", f"Link target not found: {rel}"))

    for rel in [
        "./docs/interface.md",
        "./docs/workflow.md",
        "./docs/domain.md",
        "./docs/rule.md",
        "./interface.md",
        "./workflow.md",
        "./domain.md",
        "./rule.md",
    ]:
        _check_link(claude_txt, "CLAUDE.md", rel)
        _check_link(readme_txt, "README.md", rel)

    for label, rule_path in [
        ("docs/rule.md", docs_files["rule"]),
        ("root rule.md", legacy_files["rule"]),
    ]:
        if rule_path.exists():
            issues.extend(_check_rule_doc_quality(rule_path, label, doc_root))

    # Code-as-truth validation: key classes overlap
    if claude_txt:
        claude_keys = set(_extract_claude_key_classes(claude_txt))
        kb_keys, kb_err = _run_kb_tool_select_key_classes(doc_root, DEFAULT_TOP)
        if kb_err:
            issues.append(Issue("WARN", "kb_tool.py key class extraction failed", kb_err))
        else:
            kb_set = set(kb_keys)
            overlap = len(claude_keys & kb_set)
            if claude_keys and overlap == 0:
                issues.append(
                    Issue(
                        "WARN",
                        "Key Classes mismatch vs code",
                        "No overlap between CLAUDE.md key-class table and code-derived key classes. "
                        "Docs may be stale or using a different module boundary.",
                    )
                )
            elif claude_keys and overlap < min(5, len(claude_keys)):
                issues.append(
                    Issue(
                        "INFO",
                        "Key Classes low overlap vs code",
                        f"Overlap={overlap}. Consider regenerating or updating the key-class list from code.",
                    )
                )

    # Code-as-truth heuristic: entry headers exist
    entry_headers = _find_entry_headers(doc_root)
    if not entry_headers:
        issues.append(
            Issue(
                "INFO",
                "No obvious ModuleService header found",
                "Could be Swift-only or different module layout; validate the entry point manually.",
            )
        )
    else:
        # If interface docs exist, check whether any entry header path is mentioned.
        interface_doc = _read_text(docs_files["interface"]) or _read_text(legacy_files["interface"]) or ""
        if interface_doc:
            mentioned = any(p.name in interface_doc or str(p).replace(str(doc_root) + "/", "") in interface_doc for p in entry_headers)
            if not mentioned:
                issues.append(
                    Issue(
                        "INFO",
                        "Interface doc may be missing the real entry header",
                        "Found candidate entry headers in code, but none appear to be referenced in interface docs.",
                    )
                )

    return issues


def _format_report(results: List[Tuple[Path, List[Issue]]]) -> str:
    out: List[str] = []
    out.append("# kb-docs-validator report")
    out.append("")

    for doc_root, issues in results:
        out.append(f"## {doc_root}")
        out.append("")
        if not issues:
            out.append("No issues found.")
            out.append("")
            continue

        for sev in ["ERROR", "WARN", "INFO"]:
            group = [i for i in issues if i.severity == sev]
            if not group:
                continue
            out.append(f"### {sev}")
            out.append("")
            for i in group:
                out.append(f"- **{i.title}** — {i.details}")
            out.append("")

    return "\n".join(out).rstrip() + "\n"


def main(argv: Sequence[str]) -> int:
    ap = argparse.ArgumentParser(prog="validate_kb_docs.py")
    ap.add_argument("doc_roots", nargs="+", help="One or more doc root directories")
    args = ap.parse_args(argv)

    results: List[Tuple[Path, List[Issue]]] = []
    for p in args.doc_roots:
        doc_root = Path(p).expanduser().resolve()
        if not doc_root.exists() or not doc_root.is_dir():
            results.append((doc_root, [Issue("ERROR", "Invalid doc root", "Path does not exist or is not a directory")]))
            continue
        results.append((doc_root, _check_doc_root(doc_root)))

    sys.stdout.write(_format_report(results))
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))

#!/usr/bin/env python3
"""Materialize canonical docs bundles for kb-docs-benchmark runs."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

_SKILL_ROOT = Path(__file__).resolve().parent.parent
if str(_SKILL_ROOT) not in sys.path:
    sys.path.insert(0, str(_SKILL_ROOT))

from scripts.eval_helpers import docs_bundle_stats, load_docs_bundle, resolve_doc_root


def _load_json(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


def main() -> None:
    parser = argparse.ArgumentParser(description="Prepare canonical docs_bundle.md files")
    parser.add_argument("--evals", required=True, type=Path, help="Path to docs/evals/evals.json")
    parser.add_argument("--workspace", required=True, type=Path, help="Benchmark workspace")
    parser.add_argument("--base-dir", required=True, type=Path, help="Repo root for relative doc_root values")
    args = parser.parse_args()

    evals_path = args.evals.expanduser().resolve()
    workspace = args.workspace.expanduser().resolve()
    base_dir = args.base_dir.expanduser().resolve()
    bundle_dir = workspace / "docs_bundles"
    bundle_dir.mkdir(parents=True, exist_ok=True)

    payload = _load_json(evals_path)
    eval_items = payload.get("evals", [])
    manifest: list[dict] = []

    for eval_item in eval_items:
        eval_id = int(eval_item["id"])
        doc_root = resolve_doc_root(str(eval_item["doc_root"]), base_dir)
        doc_paths = list(eval_item.get("doc_paths") or [])
        docs_bundle, missing_docs = load_docs_bundle(doc_root, doc_paths)
        bundle_path = bundle_dir / f"eval-{eval_id}.md"
        bundle_path.write_text(docs_bundle + "\n", encoding="utf-8")
        stats = docs_bundle_stats(docs_bundle)
        manifest.append(
            {
                "eval_id": eval_id,
                "doc_root": str(doc_root),
                "doc_paths": doc_paths,
                "bundle_path": str(bundle_path),
                "missing_docs": missing_docs,
                **stats,
            }
        )

    manifest_path = bundle_dir / "manifest.json"
    manifest_path.write_text(json.dumps(manifest, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(f"[kb-docs-benchmark] wrote {len(manifest)} docs bundle(s) to {bundle_dir}")


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Write timing metadata for a benchmark run.

Useful in Claude Code subagent mode: the parent agent can read the Agent tool's
usage block and call this script to persist timing.json before grading.
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

_SKILL_ROOT = Path(__file__).resolve().parent.parent
if str(_SKILL_ROOT) not in sys.path:
    sys.path.insert(0, str(_SKILL_ROOT))

from scripts.eval_helpers import docs_bundle_stats, read_text


def main() -> None:
    parser = argparse.ArgumentParser(description="Write timing.json for one benchmark run")
    parser.add_argument("--run-dir", required=True, type=Path, help="Path like <workspace>/eval-1-q1/with_docs/run-1")
    parser.add_argument("--total-tokens", type=int, default=0)
    parser.add_argument("--duration-ms", type=int, default=0)
    parser.add_argument(
        "--fallback-duration-ms",
        type=int,
        default=0,
        help="Parent-measured wall-clock duration to use when subagent usage.duration_ms is unavailable or zero.",
    )
    parser.add_argument("--tool-uses", type=int, default=0)
    parser.add_argument("--granularity", choices=["session_batch", "cold_start_per_question"], default="session_batch")
    parser.add_argument("--batch-id", default="", help="Internal batch identifier for debugging, not shown in reports.")
    parser.add_argument("--batch-index", type=int, default=0, help="Zero-based internal batch index.")
    parser.add_argument("--batch-size", type=int, default=0, help="Configured max batch size for this run.")
    parser.add_argument("--batch-question-count", type=int, default=0, help="Actual number of questions answered by this batch.")
    parser.add_argument("--docs-bundle-hash", default="", help="SHA-256 hash for the docs bundle used by this run.")
    parser.add_argument("--docs-bundle-chars", type=int, default=0, help="Character count for the docs bundle used by this run.")
    parser.add_argument("--estimated-docs-tokens", type=int, default=0, help="Estimated token count for the docs bundle used by this run.")
    parser.add_argument("--docs-token-estimate-method", default="", help="Method used for estimated docs tokens.")
    parser.add_argument(
        "--docs-bundle",
        type=Path,
        default=None,
        help="Optional docs bundle path. Defaults to <run-dir>/outputs/docs_bundle.md when present.",
    )
    parser.add_argument(
        "--allocation-count",
        type=int,
        default=1,
        help="For session_batch, number of answers sharing this subagent usage. Values are evenly amortized.",
    )
    args = parser.parse_args()

    run_dir = args.run_dir.expanduser()
    run_dir.mkdir(parents=True, exist_ok=True)
    source_total_tokens = max(0, int(args.total_tokens))
    usage_duration_ms = max(0, int(args.duration_ms))
    fallback_duration_ms = max(0, int(args.fallback_duration_ms))
    if usage_duration_ms > 0:
        source_duration_ms = usage_duration_ms
        duration_source = "subagent_usage"
    elif fallback_duration_ms > 0:
        source_duration_ms = fallback_duration_ms
        duration_source = "parent_wall_clock"
    else:
        source_duration_ms = 0
        duration_source = "missing"
    source_tool_uses = max(0, int(args.tool_uses))
    allocation_count = max(1, int(args.allocation_count))
    batch_index = max(0, int(args.batch_index))
    batch_size = max(0, int(args.batch_size))
    batch_question_count = max(0, int(args.batch_question_count))

    docs_bundle_path = args.docs_bundle.expanduser() if args.docs_bundle else run_dir / "outputs" / "docs_bundle.md"
    if docs_bundle_path.is_file():
        inferred_docs_stats = docs_bundle_stats(read_text(docs_bundle_path))
    else:
        inferred_docs_stats = docs_bundle_stats("")
    docs_bundle_hash = str(args.docs_bundle_hash or inferred_docs_stats["docs_bundle_hash"])
    docs_bundle_chars = max(0, int(args.docs_bundle_chars or inferred_docs_stats["docs_bundle_chars"]))
    estimated_docs_tokens = max(0, int(args.estimated_docs_tokens or inferred_docs_stats["estimated_docs_tokens"]))
    docs_token_estimate_method = str(args.docs_token_estimate_method or inferred_docs_stats["docs_token_estimate_method"])

    if args.granularity == "session_batch":
        total_tokens = round(source_total_tokens / allocation_count)
        duration_ms = round(source_duration_ms / allocation_count)
        tool_uses = round(source_tool_uses / allocation_count)
        allocation_strategy = "even"
    else:
        total_tokens = source_total_tokens
        duration_ms = source_duration_ms
        tool_uses = source_tool_uses
        allocation_strategy = "none"

    payload = {
        "total_tokens": total_tokens,
        "duration_ms": duration_ms,
        "total_duration_seconds": round(duration_ms / 1000.0, 3) if duration_ms else 0.0,
        "tool_uses": tool_uses,
        "granularity": args.granularity,
        "batch_id": str(args.batch_id or ""),
        "batch_index": batch_index,
        "batch_size": batch_size,
        "batch_question_count": batch_question_count or allocation_count,
        "allocation_strategy": allocation_strategy,
        "allocation_count": allocation_count,
        "source_total_tokens": source_total_tokens,
        "source_duration_ms": source_duration_ms,
        "usage_duration_ms": usage_duration_ms,
        "fallback_duration_ms": fallback_duration_ms,
        "duration_source": duration_source,
        "source_tool_uses": source_tool_uses,
        "docs_bundle_hash": docs_bundle_hash,
        "docs_bundle_chars": docs_bundle_chars,
        "estimated_docs_tokens": estimated_docs_tokens,
        "docs_token_estimate_method": docs_token_estimate_method,
    }
    timing_path = run_dir / "timing.json"
    timing_path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
    print(f"[kb-docs-benchmark] wrote {timing_path}")


if __name__ == "__main__":
    main()

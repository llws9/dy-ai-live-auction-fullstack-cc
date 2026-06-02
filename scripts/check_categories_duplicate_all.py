#!/usr/bin/env python3
"""Check whether GET /api/v1/categories returns duplicated "全部" categories."""

from __future__ import annotations

import argparse
import json
import sys
import urllib.error
import urllib.request
from collections import Counter
from typing import Any


TARGET_NAME = "全部"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description='Call GET /api/v1/categories and detect duplicated "全部" categories.'
    )
    parser.add_argument(
        "--base-url",
        default="http://localhost:8080",
        help="Backend base URL, default: http://localhost:8080",
    )
    parser.add_argument(
        "--path",
        default="/api/v1/categories",
        help="Categories endpoint path, default: /api/v1/categories",
    )
    return parser.parse_args()


def build_url(base_url: str, path: str) -> str:
    return f"{base_url.rstrip('/')}/{path.lstrip('/')}"


def fetch_json(url: str) -> Any:
    request = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(request, timeout=10) as response:
        status = response.status
        body = response.read().decode("utf-8")
    if status < 200 or status >= 300:
        raise RuntimeError(f"HTTP {status}: {body[:500]}")
    return json.loads(body)


def extract_categories(payload: Any) -> list[dict[str, Any]]:
    candidates = [
        payload,
        payload.get("list") if isinstance(payload, dict) else None,
        payload.get("items") if isinstance(payload, dict) else None,
        payload.get("categories") if isinstance(payload, dict) else None,
        payload.get("data") if isinstance(payload, dict) else None,
    ]

    if isinstance(payload, dict) and isinstance(payload.get("data"), dict):
        data = payload["data"]
        candidates.extend([data.get("list"), data.get("items"), data.get("categories")])

    for candidate in candidates:
        if isinstance(candidate, list):
            return [item for item in candidate if isinstance(item, dict)]
    return []


def main() -> int:
    args = parse_args()
    url = build_url(args.base_url, args.path)
    print(f"Requesting: {url}")

    try:
        payload = fetch_json(url)
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError, RuntimeError) as exc:
        print(f"ERROR: failed to fetch or parse categories: {exc}", file=sys.stderr)
        return 1

    categories = extract_categories(payload)
    names = [item.get("name") for item in categories if isinstance(item.get("name"), str)]
    name_counter = Counter(names)
    all_items = [item for item in categories if item.get("name") == TARGET_NAME]

    print(f"Parsed categories: {len(categories)}")
    print(f'Found "{TARGET_NAME}" categories: {len(all_items)}')

    if all_items:
        print(f'\n"{TARGET_NAME}" category rows:')
        for item in all_items:
            print(json.dumps(item, ensure_ascii=False, sort_keys=True))

    duplicated_names = {name: count for name, count in name_counter.items() if count > 1}
    if duplicated_names:
        print("\nDuplicated category names:")
        for name, count in sorted(duplicated_names.items()):
            print(f"- {name}: {count}")

    if len(all_items) > 1:
        print(f'\nRESULT: duplicated "{TARGET_NAME}" categories found.')
        return 2
    if len(all_items) == 1:
        print(f'\nRESULT: one "{TARGET_NAME}" category exists in backend response.')
        return 0

    print(f'\nRESULT: no "{TARGET_NAME}" category in backend response.')
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

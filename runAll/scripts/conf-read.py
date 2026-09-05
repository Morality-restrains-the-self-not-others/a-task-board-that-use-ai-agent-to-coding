#!/usr/bin/env python3
"""Read monorepo conf/<app>/ YAML for shell, Vite, and Playwright.

Examples:
  conf-read.py sms --json    # OPT-20260806-057: django 键退役，改读语义化目录
  conf-read.py vue apiBaseUrl
  conf-read.py domain-events transport
  conf-read.py snapshot-json   # tooling: same shape as former port_config.json
"""
from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parent.parent  # runAll/scripts/ → parent = runAll/, parent.parent = monorepo root
sys.path.insert(0, str(Path(__file__).resolve().parent))

from conf_loader import (  # noqa: E402
    JSON_KEY_TO_APP_DIR,
    build_runtime_snapshot,
    load_app_config,
    load_domain_events_config,
    load_git_oauth_catalog,
    monorepo_root,
)


def _get_by_path(data: dict[str, Any], dot_path: str) -> Any:
    cur: Any = data
    for part in dot_path.split("."):
        if not isinstance(cur, dict) or part not in cur:
            raise KeyError(dot_path)
        cur = cur[part]
    return cur


def build_snapshot() -> dict[str, Any]:
    return build_runtime_snapshot()


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Read monorepo conf/")
    parser.add_argument("app", nargs="?", help="app dir name, e.g. sms, vue, snapshot-json")
    parser.add_argument("path", nargs="?", help="dot.path key inside app config")
    parser.add_argument("--json", action="store_true", help="print JSON object for app config")
    args = parser.parse_args(argv)

    if not args.app:
        parser.print_help()
        return 2

    if args.app == "snapshot-json":
        print(json.dumps(build_snapshot(), ensure_ascii=False))
        return 0

    if args.app == "git-oauth" and args.path:
        # git-oauth catalog is keyed by website URL
        catalog = load_git_oauth_catalog()
        val = _get_by_path(catalog, args.path) if "." in args.path else catalog.get(args.path)
        if val is None:
            raise SystemExit(f"git-oauth key not found: {args.path}")
        if isinstance(val, (dict, list)):
            print(json.dumps(val, ensure_ascii=False))
        else:
            print(val)
        return 0

    app_dir = JSON_KEY_TO_APP_DIR.get(args.app) or args.app.replace("_", "-")
    data = load_app_config(app_dir)
    if args.app == "domain-events" and not data:
        data = load_domain_events_config()

    if args.json or not args.path:
        print(json.dumps(data, ensure_ascii=False))
        return 0

    val = _get_by_path(data, args.path)
    if isinstance(val, (dict, list)):
        print(json.dumps(val, ensure_ascii=False))
    else:
        print(val)
    return 0


if __name__ == "__main__":
    try:
        monorepo_root()
        raise SystemExit(main())
    except FileNotFoundError as e:
        print(e, file=sys.stderr)
        raise SystemExit(2) from e
    except KeyError as e:
        print(f"key not found: {e}", file=sys.stderr)
        raise SystemExit(1) from e

#!/usr/bin/env python3
"""Shim → db/scripts/ci/check_go_openapi_routes.py --service taskBill。"""

from __future__ import annotations

import runpy
import sys
from pathlib import Path

SCRIPT = (
    Path(__file__).resolve().parents[2]
    / "db"
    / "scripts"
    / "ci"
    / "check_go_openapi_routes.py"
)

if not SCRIPT.is_file():
    print(f"ERROR: missing {SCRIPT}", file=sys.stderr)
    raise SystemExit(2)

sys.argv = [str(SCRIPT), "--service", "taskBill", *sys.argv[1:]]
runpy.run_path(str(SCRIPT), run_name="__main__")

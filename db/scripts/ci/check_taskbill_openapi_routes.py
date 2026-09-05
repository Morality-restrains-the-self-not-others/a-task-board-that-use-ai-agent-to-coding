#!/usr/bin/env python3
"""兼容入口 → check_go_openapi_routes.py --service taskBill。"""

from __future__ import annotations

import sys
from pathlib import Path

SCRIPT = Path(__file__).resolve().parent / "check_go_openapi_routes.py"
sys.argv = [str(SCRIPT), "--service", "taskBill", *sys.argv[1:]]
import runpy

runpy.run_path(str(SCRIPT), run_name="__main__")

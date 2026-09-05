#!/usr/bin/env python3
"""ai-provider 缺 SPA dist 必须在启动前 Fatal（OPT-20260831-019）。

runAll conf/runAll.yaml 的 ai-provider 使用 `./bin/taskAiProvider` 直启，绕过
taskAiProvider/run.sh start 对 frontend/dist/index.html 的硬失败；seed 部署若缺
SPA 包，进程仍起来、handleSPA 对 / 静默 404。本门禁断言：
  - taskAiProvider/src/main.go 在缺 dist 时走 tracelog.Fatal（禁止回归为 Emit warn）；
  - taskAiProvider/run.sh start 分支保留 frontend/dist/index.html 硬失败（兜底）。

用法:
  python3 db/scripts/ci/check_ai_provider_dist_fatal.py
"""
from __future__ import annotations

import argparse
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]


def collect_violations(root: Path) -> list[str]:
    hits: list[str] = []
    main_go = root / "taskAiProvider" / "src" / "main.go"
    if not main_go.is_file():
        hits.append("taskAiProvider/src/main.go: missing")
    else:
        src = main_go.read_text(encoding="utf-8")
        if "FrontendDistDir" not in src:
            hits.append("taskAiProvider/src/main.go: missing FrontendDistDir dist check")
        if "index.html" not in src:
            hits.append("taskAiProvider/src/main.go: missing index.html dist guard")
        if "frontend dist missing" in src:
            if "tracelog.Fatal" not in src:
                hits.append("taskAiProvider/src/main.go: 缺 dist 未走 tracelog.Fatal")
            if 'tracelog.Emit("warn"' in src and "frontend dist missing" in src:
                hits.append("taskAiProvider/src/main.go: 缺 dist 仍 Emit warn（应 Fatal）")
        else:
            hits.append("taskAiProvider/src/main.go: 缺少缺 dist 的失败分支")

    run_sh = root / "taskAiProvider" / "run.sh"
    if not run_sh.is_file():
        hits.append("taskAiProvider/run.sh: missing（run.sh start dist 门闩缺失）")
    elif "frontend/dist/index.html" not in run_sh.read_text(encoding="utf-8"):
        hits.append("taskAiProvider/run.sh: start 分支缺少 frontend/dist/index.html 硬失败")
    return hits


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT)
    args = parser.parse_args(argv)
    root = args.root.resolve()
    hits = collect_violations(root)
    if hits:
        print("VIOLATION (OPT-20260831-019: ai-provider 缺 SPA dist 必须启动前 Fatal):")
        for h in hits:
            print(f"  - {h}")
        return 1
    print("ok: ai-provider 缺 dist 启动前 Fatal（runAll 直启不绕过）")
    return 0


if __name__ == "__main__":
    sys.exit(main())

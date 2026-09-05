#!/usr/bin/env python3
"""Self-test for check_run_sh_start_no_implicit_build (no pytest required).

Run:  python3 db/scripts/ci/test_check_run_sh_start_no_implicit_build.py
"""

from __future__ import annotations

import importlib.util
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_run_sh_start_no_implicit_build.py"

# 将 ROOT 加入 path 以便 checker import yaml 无需额外处理
sys.path.insert(0, str(ROOT / "db" / "scripts" / "ci"))

def _load():
    spec = importlib.util.spec_from_file_location(
        "check_run_sh_start_no_implicit_build", CHECKER
    )
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def test_extract_start_branch_clean(tmp_path):
    mod = _load()
    src = """
case "$cmd" in
  build)
    build_all
    ;;
  start)
    start_all_intents
    ;;
  stop)
    stop_all_intents
    ;;
esac
"""
    branch = mod.extract_start_branch(src)
    assert "start_all_intents" in branch
    assert "build)" not in branch
    assert not mod.branch_compiles(branch)


def test_extract_start_branch_violating(tmp_path):
    mod = _load()
    src = """
case "$cmd" in
  start)
    go build -o bin/app .
    ;;
  build)
    go build -o bin/app .
    ;;
esac
"""
    branch = mod.extract_start_branch(src)
    assert mod.branch_compiles(branch)


def test_start_branch_calling_build_fn_is_flagged(tmp_path):
    mod = _load()
    src = """
case "$cmd" in
  start)
    build_intent "$target"
    ;;
esac
"""
    branch = mod.extract_start_branch(src)
    assert "build_intent" in branch
    assert mod.branch_compiles(branch)


def test_build_branch_allowed_even_with_go_build(tmp_path):
    mod = _load()
    src = """
case "$cmd" in
  build)
    go build -o bin/app .
    ;;
esac
"""
    # 只有 build) 分支：没有 start) 分支 → 不触发
    branch = mod.extract_start_branch(src)
    assert branch == ""


def test_scan_one_clean(tmp_path: Path):
    mod = _load()
    sh = tmp_path / "app" / "run.sh"
    sh.parent.mkdir(parents=True)
    sh.write_text(
        """#!/usr/bin/env bash
case "$cmd" in
  start)
    exec ./bin/app "$@"
    ;;
  build)
    go build -o bin/app .
    ;;
esac
""",
        encoding="utf-8",
    )
    assert mod.scan_one(sh, "app") == []


def test_scan_one_violating(tmp_path: Path):
    mod = _load()
    sh = tmp_path / "app" / "run.sh"
    sh.parent.mkdir(parents=True)
    sh.write_text(
        """#!/usr/bin/env bash
case "$cmd" in
  start)
    ./build.sh
    exec ./bin/app "$@"
    ;;
esac
""",
        encoding="utf-8",
    )
    violations = mod.scan_one(sh, "app")
    assert len(violations) == 1
    assert "start" in violations[0]


def test_runall_resolution(tmp_path: Path):
    mod = _load()
    sh = tmp_path / "taskEvents" / "run.sh"
    sh.parent.mkdir(parents=True)
    sh.write_text(
        """#!/usr/bin/env bash
case "$cmd" in
  start)
    exec ./bin/task-events "$@"
    ;;
esac
""",
        encoding="utf-8",
    )
    cfg = {
        "groups": [
            {
                "name": "platform",
                "services": [
                    {
                        "name": "task-events-billing-transaction-created-1-process-billing-transaction",
                        "working_dir": str(tmp_path),
                        "start_command": f"bash taskEvents/run.sh start billing_transaction_created/1_process_billing_transaction",
                    }
                ],
            }
        ]
    }
    paths = mod.run_sh_paths_from_runall(cfg)
    assert len(paths) == 1
    assert paths[0][1].resolve() == sh.resolve()


def main() -> int:
    import tempfile as _tf

    failures = 0
    tests = [
        fn
        for name, fn in sorted(globals().items())
        if name.startswith("test_") and callable(fn)
    ]
    for fn in tests:
        try:
            with _tf.TemporaryDirectory() as tmp:
                fn(Path(tmp))
            print(f"PASS {fn.__name__}")
        except Exception as exc:  # noqa: BLE001
            failures += 1
            print(f"FAIL {fn.__name__}: {exc}", file=sys.stderr)
    print(f"\n{len(tests) - failures}/{len(tests)} tests passed.")
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())

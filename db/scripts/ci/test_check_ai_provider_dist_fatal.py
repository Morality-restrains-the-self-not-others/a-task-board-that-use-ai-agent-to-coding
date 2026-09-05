#!/usr/bin/env python3
"""Self-test for check_ai_provider_dist_fatal (no pytest required)."""

from __future__ import annotations

import importlib.util
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_ai_provider_dist_fatal.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_ai_provider_dist_fatal", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def _write_good(tmp: Path) -> None:
    main_go = tmp / "taskAiProvider" / "src" / "main.go"
    main_go.parent.mkdir(parents=True)
    main_go.write_text(
        'if _, err := os.Stat(filepath.Join(cfg.FrontendDistDir, "index.html")); err != nil {\n'
        '    tracelog.Fatal(service, "frontend dist missing", err)\n'
        "}\n",
        encoding="utf-8",
    )
    run_sh = tmp / "taskAiProvider" / "run.sh"
    run_sh.parent.mkdir(parents=True, exist_ok=True)
    run_sh.write_text(
        "if [ ! -f frontend/dist/index.html ]; then\n"
        '    echo "ERROR: frontend/dist/index.html missing" >&2\n'
        "    exit 1\n"
        "fi\n",
        encoding="utf-8",
    )


def test_ok_when_fatal_guard_present(tmp_path) -> None:
    mod = _load()
    _write_good(tmp_path)
    hits = mod.collect_violations(tmp_path)
    assert hits == [], hits


def test_fails_when_emit_warn(tmp_path) -> None:
    mod = _load()
    _write_good(tmp_path)
    main_go = tmp_path / "taskAiProvider" / "src" / "main.go"
    main_go.write_text(
        'if _, err := os.Stat(filepath.Join(cfg.FrontendDistDir, "index.html")); err != nil {\n'
        '    tracelog.Emit("warn", "frontend dist missing ...", "", nil)\n'
        "}\n",
        encoding="utf-8",
    )
    hits = mod.collect_violations(tmp_path)
    assert any("Emit warn" in h for h in hits), hits


def test_fails_when_guard_missing(tmp_path) -> None:
    mod = _load()
    _write_good(tmp_path)
    (tmp_path / "taskAiProvider" / "src" / "main.go").write_text(
        "package main\n", encoding="utf-8"
    )
    hits = mod.collect_violations(tmp_path)
    assert hits, "expected violations when main.go lacks dist Fatal guard"


def test_fails_when_run_sh_latch_missing(tmp_path) -> None:
    mod = _load()
    _write_good(tmp_path)
    (tmp_path / "taskAiProvider" / "run.sh").write_text(
        "exec ./bin/taskAiProvider\n", encoding="utf-8"
    )
    hits = mod.collect_violations(tmp_path)
    assert any("run.sh" in h for h in hits), hits


if __name__ == "__main__":
    import tempfile

    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for t in tests:
        with tempfile.TemporaryDirectory() as td:
            t(Path(td))
        print(f"PASS {t.__name__}")
    print(f"ok ({len(tests)} tests)")

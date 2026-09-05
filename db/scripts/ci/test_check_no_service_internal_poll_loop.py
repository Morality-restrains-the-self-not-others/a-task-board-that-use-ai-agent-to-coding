#!/usr/bin/env python3
"""Self-test for check_no_service_internal_poll_loop (no pytest required)."""

from __future__ import annotations

import importlib.util
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_no_service_internal_poll_loop.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_no_service_internal_poll_loop", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def test_ticker_regex_detects_newticker_and_tick() -> None:
    mod = _load()
    assert mod.TICKER_RE.search("ticker := time.NewTicker(time.Second)")
    assert mod.TICKER_RE.search("for range time.Tick(30 * time.Second) {")
    assert not mod.TICKER_RE.search("func Ticketing() {}")
    assert not mod.TICKER_RE.search("time.After(time.Second)")


def test_classify_business_service_ticker_is_violation() -> None:
    mod = _load()
    src = "package main\nfunc loop() { t := time.NewTicker(time.Hour) }\n"
    kind, _ = mod.classify("taskBill/src/foo.go", src)
    assert kind == "violation"


def test_classify_dedicated_timer_handler_is_ok() -> None:
    mod = _load()
    src = "package x\nfunc Run() { t := time.NewTicker(time.Hour) }\n"
    kind, _ = mod.classify(
        "taskEvents/internal/handlers/taskpostexpiryscan/runner.go", src
    )
    assert kind == "ok"


def test_classify_protocol_broker_is_ok() -> None:
    mod = _load()
    src = "package broker\nfunc recover() { t := time.NewTicker(time.Second) }\n"
    kind, _ = mod.classify("taskEvents/broker/redis.go", src)
    assert kind == "ok"


def test_classify_worktree_dir_is_ok() -> None:
    mod = _load()
    src = "package main\nfunc loop() { t := time.NewTicker(time.Hour) }\n"
    kind, _ = mod.classify("taskCloudService-wt/src/server_orphan_reconcile.go", src)
    assert kind == "ok"


def test_classify_test_file_is_ok() -> None:
    mod = _load()
    src = "package main\nfunc TestX() { t := time.NewTicker(time.Millisecond) }\n"
    kind, _ = mod.classify("taskBill/src/foo_test.go", src)
    assert kind == "ok"


def test_classify_legacy_allowlist_is_legacy() -> None:
    mod = _load()
    src = "package main\nfunc loop() { t := time.NewTicker(time.Hour) }\n"
    if not mod.LEGACY_INTERNAL_TICKERS:
        # 存量 ticker 已迁出后 allowlist 为空；业务路径仍须判 violation。
        kind, _ = mod.classify("taskBill/src/loop.go", src)
        assert kind == "violation"
        return
    rel = next(iter(mod.LEGACY_INTERNAL_TICKERS))
    kind, _ = mod.classify(rel, src)
    assert kind == "legacy"


def test_classify_scheduler_ok_comment_outside_allow_path_still_violation() -> None:
    """Business HTTP services cannot self-exempt with a comment."""
    mod = _load()
    src = (
        "// Scheduler-OK: dedicated-timer — pretend\n"
        "package main\nfunc loop() { t := time.NewTicker(time.Hour) }\n"
    )
    kind, _ = mod.classify("taskCloudService/src/new_reconcile.go", src)
    assert kind == "violation"


def test_scan_tmp_tree_flags_new_business_ticker() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        tmp = Path(d)
        billed = tmp / "taskBill" / "src"
        billed.mkdir(parents=True)
        (billed / "loop.go").write_text(
            "package main\nfunc f() { _ = time.NewTicker(time.Second) }\n",
            encoding="utf-8",
        )
        handlers = tmp / "taskEvents" / "internal" / "handlers" / "scan"
        handlers.mkdir(parents=True)
        (handlers / "runner.go").write_text(
            "package scan\nfunc f() { _ = time.NewTicker(time.Hour) }\n",
            encoding="utf-8",
        )
        old_root = mod.ROOT
        mod.ROOT = tmp
        try:
            hits = mod.scan()
        finally:
            mod.ROOT = old_root
        rels = {h.rel: h.kind for h in hits}
        assert rels.get("taskBill/src/loop.go") == "violation"
        assert "taskEvents/internal/handlers/scan/runner.go" not in rels


def test_main_returns_nonzero_on_violation() -> None:
    mod = _load()
    with tempfile.TemporaryDirectory() as d:
        tmp = Path(d)
        src_dir = tmp / "taskBill" / "src"
        src_dir.mkdir(parents=True)
        (src_dir / "loop.go").write_text(
            "package main\nfunc f() { _ = time.NewTicker(time.Second) }\n",
            encoding="utf-8",
        )
        old_root = mod.ROOT
        mod.ROOT = tmp
        try:
            from io import StringIO
            from contextlib import redirect_stdout

            buf = StringIO()
            with redirect_stdout(buf):
                rc = mod.main([])
        finally:
            mod.ROOT = old_root
        assert rc == 1
        assert "taskBill/src/loop.go" in buf.getvalue()


if __name__ == "__main__":
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for t in tests:
        t()
        print(f"PASS {t.__name__}")
    print(f"ok: {len(tests)} tests")

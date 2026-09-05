#!/usr/bin/env python3
"""Smoke tests for check_logic_rollback_approval (no pytest required)."""
from __future__ import annotations

import importlib.util
import os
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
CHECKER = ROOT / "db" / "scripts" / "ci" / "check_logic_rollback_approval.py"


def _load():
    spec = importlib.util.spec_from_file_location("check_logic_rollback_approval", CHECKER)
    assert spec and spec.loader
    mod = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = mod
    spec.loader.exec_module(mod)
    return mod


def _diff(path: str, *added: str) -> str:
    body = "\n".join("+" + line for line in added)
    return f"--- a/{path}\n+++ b/{path}\n@@\n{body}\n"


def test_clean_added_code_passes() -> None:
    mod = _load()
    diff = _diff("app/order.go", "func Total(n int) int { return n }")
    rc = mod.run_gate(ROOT, "feat: add total", diff, ["app/order.go"], skip_artifacts=True)
    assert rc == 0


def test_keep_old_marker_blocked() -> None:
    mod = _load()
    diff = _diff("app/pay.go", "func fallbackToOld() {}")
    rc = mod.run_gate(ROOT, "feat: pay", diff, ["app/pay.go"], skip_artifacts=True)
    assert rc == 1


def test_waiver_allows_marker() -> None:
    mod = _load()
    diff = _diff("app/pay.go", "func fallbackToOld() {}")
    msg = "feat: pay\n\nLogic-Rollback-OK: gateway still signs v1; remove-by 2026-09-01"
    rc = mod.run_gate(ROOT, msg, diff, ["app/pay.go"], skip_artifacts=True)
    assert rc == 0


def test_cn_keep_old_blocked() -> None:
    mod = _load()
    diff = _diff("app/x.go", "// 先留着这套旧实现")
    rc = mod.run_gate(ROOT, "fix: x", diff, ["app/x.go"], skip_artifacts=True)
    assert rc == 1


def test_legacy_filename_blocked() -> None:
    mod = _load()
    diff = _diff("app/handler_legacy.go", "package app")
    rc = mod.run_gate(ROOT, "feat: split", diff, ["app/handler_legacy.go"], skip_artifacts=True)
    assert rc == 1


def test_markdown_ignored() -> None:
    mod = _load()
    diff = _diff("docs/note.md", "先留着旧逻辑说明")
    rc = mod.run_gate(ROOT, "docs: note", diff, ["docs/note.md"], skip_artifacts=True)
    assert rc == 0


def test_commented_out_block_blocked() -> None:
    mod = _load()
    lines = [f"// func old{i}() {{ return }}" for i in range(6)]
    diff = _diff("app/dead.go", *lines)
    rc = mod.run_gate(ROOT, "refactor: move", diff, ["app/dead.go"], skip_artifacts=True)
    assert rc == 1


def test_revert_source_blocked() -> None:
    mod = _load()
    rc = mod.run_gate(
        ROOT,
        "revert: undo pay change",
        "",
        ["app/pay.go"],
        skip_artifacts=True,
    )
    assert rc == 1


def test_revert_docs_only_passes() -> None:
    mod = _load()
    rc = mod.run_gate(
        ROOT,
        "revert: undo readme typo",
        "",
        ["README.md"],
        skip_artifacts=True,
    )
    assert rc == 0


def test_checker_self_ignored() -> None:
    mod = _load()
    diff = _diff(
        "db/scripts/ci/check_logic_rollback_approval.py",
        "fallbackToOld = 1",
    )
    rc = mod.run_gate(
        ROOT,
        "chore: gate",
        diff,
        ["db/scripts/ci/check_logic_rollback_approval.py"],
        skip_artifacts=True,
    )
    assert rc == 0


def test_repo_artifacts_present() -> None:
    mod = _load()
    hits = mod.missing_artifacts(ROOT)
    assert hits == [], hits


def _git(tmp: Path, *args: str) -> str:
    # git hooks inject GIT_DIR / GIT_WORK_TREE / GIT_INDEX_FILE. Inheriting them
    # makes `git init` in a temp dir rewrite the host repo (core.bare=true) and
    # then `git add` fails with "core.bare and core.worktree do not make sense".
    clean_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}
    proc = subprocess.run(
        ["git", *args],
        cwd=tmp,
        env=clean_env,
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        raise AssertionError(f"git {' '.join(args)} failed: {proc.stderr}")
    return proc.stdout


def _diff_text(path: str, *added: str) -> str:
    body = "\n".join("+" + line for line in added)
    return f"--- a/{path}\n+++ b/{path}\n@@\n{body}\n"


def test_history_rollback_re_added_symbol_blocked() -> None:
    """OPT-20260819-043: 从 git 历史拷回已删函数（无关键字）应被拦截。"""
    mod = _load()
    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        _git(tmp, "init", "-q")
        _git(tmp, "config", "user.email", "t@t")
        _git(tmp, "config", "user.name", "t")
        pay = tmp / "app"
        pay.mkdir()
        f = pay / "pay.go"
        f.write_text("package app\n\nfunc oldTotal() {}\n", encoding="utf-8")
        _git(tmp, "add", "app/pay.go")
        _git(tmp, "commit", "-q", "-m", "feat: add total")
        # 删除 oldTotal
        f.write_text("package app\n\nfunc other() {}\n", encoding="utf-8")
        _git(tmp, "commit", "-q", "-am", "refactor: drop oldTotal")
        # 静默拷回 oldTotal（未提交）
        f.write_text("package app\n\nfunc oldTotal() {}\n", encoding="utf-8")
        _git(tmp, "add", "app/pay.go")
        staged = _git(tmp, "diff", "--cached", "-U0", "--")
        rc = mod.run_gate(
            tmp,
            "feat: reimplement",
            staged,
            ["app/pay.go"],
            skip_artifacts=True,
        )
        assert rc == 1, "silent re-add of a deleted symbol must be blocked"


def test_history_rollback_new_symbol_passes() -> None:
    """全新符号不触发历史回退检测。"""
    mod = _load()
    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        _git(tmp, "init", "-q")
        _git(tmp, "config", "user.email", "t@t")
        _git(tmp, "config", "user.name", "t")
        f = tmp / "pay.go"
        f.write_text("package app\n\ntype T struct{}\n", encoding="utf-8")
        _git(tmp, "add", "pay.go")
        _git(tmp, "commit", "-q", "-m", "feat: base")
        f.write_text("package app\n\ntype T struct{}\n\nfunc brandNewTotal() {}\n", encoding="utf-8")
        _git(tmp, "add", "pay.go")
        staged = _git(tmp, "diff", "--cached", "-U0", "--")
        rc = mod.run_gate(
            tmp,
            "feat: new total",
            staged,
            ["pay.go"],
            skip_artifacts=True,
        )
        assert rc == 0, "brand-new symbol must pass history-rollback gate"


def test_history_rollback_waiver_allows() -> None:
    """Logic-Rollback-OK 豁免历史回退检测。"""
    mod = _load()
    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        _git(tmp, "init", "-q")
        _git(tmp, "config", "user.email", "t@t")
        _git(tmp, "config", "user.name", "t")
        f = tmp / "pay.go"
        f.write_text("package app\n\nfunc oldTotal() {}\n", encoding="utf-8")
        _git(tmp, "add", "pay.go")
        _git(tmp, "commit", "-q", "-m", "feat: add")
        f.write_text("package app\n\nfunc other() {}\n", encoding="utf-8")
        _git(tmp, "commit", "-q", "-am", "refactor: drop")
        f.write_text("package app\n\nfunc oldTotal() {}\n", encoding="utf-8")
        _git(tmp, "add", "pay.go")
        staged = _git(tmp, "diff", "--cached", "-U0", "--")
        msg = "feat: reimplement\n\nLogic-Rollback-OK: keep v1 fallback; remove-by 2026-10-01"
        rc = mod.run_gate(
            tmp,
            msg,
            staged,
            ["pay.go"],
            skip_artifacts=True,
        )
        assert rc == 0, "Logic-Rollback-OK must exempt history rollback"


def test_history_rollback_git_isolated_from_hook_env() -> None:
    """Hooks inject GIT_DIR; temp-repo git must not inherit it (core.bare clash)."""
    old = os.environ.get("GIT_DIR")
    os.environ["GIT_DIR"] = "/nonexistent/gitdir-from-hook"
    try:
        test_history_rollback_new_symbol_passes()
    finally:
        if old is None:
            os.environ.pop("GIT_DIR", None)
        else:
            os.environ["GIT_DIR"] = old


def test_extract_added_symbols_recognizes_decls() -> None:
    mod = _load()
    diff = _diff_text(
        "app/x.go",
        "func NewTotal() {}",
        "def old_compat(): pass",
        "class LegacySolver:",
        "const arrow = () => 1",
    )
    added = mod.parse_unified_added(diff)
    syms = mod.extract_added_symbols(added)
    assert syms == {"app/x.go": ["NewTotal", "old_compat", "LegacySolver"]}


def test_main_ok_on_clean_repo() -> None:
    mod = _load()
    assert mod.main(["--root", str(ROOT), "--diff", "", "--staged-files", "", "--message", "chore: n"]) == 0
    assert mod.main(["--root", str(ROOT), "--diff", "", "--staged-files", "", "--message", "chore: n", "--no-history-rollback"]) == 0


def main() -> int:
    tests = [
        test_clean_added_code_passes,
        test_keep_old_marker_blocked,
        test_waiver_allows_marker,
        test_cn_keep_old_blocked,
        test_legacy_filename_blocked,
        test_markdown_ignored,
        test_commented_out_block_blocked,
        test_revert_source_blocked,
        test_revert_docs_only_passes,
        test_checker_self_ignored,
        test_repo_artifacts_present,
        test_main_ok_on_clean_repo,
        test_history_rollback_re_added_symbol_blocked,
        test_history_rollback_new_symbol_passes,
        test_history_rollback_waiver_allows,
        test_history_rollback_git_isolated_from_hook_env,
        test_extract_added_symbols_recognizes_decls,
    ]
    failed = 0
    for fn in tests:
        try:
            fn()
            print(f"PASS {fn.__name__}")
        except AssertionError as exc:
            failed += 1
            print(f"FAIL {fn.__name__}: {exc}")
    if failed:
        print(f"{failed} failed")
        return 1
    print(f"{len(tests)} passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

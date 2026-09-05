#!/usr/bin/env python3
"""Smoke tests for check_bug_fix_unit_tests (no pytest required).

Rule: .ai/01_project_constraints/41_bug_fix_unit_test_required.md
Run:  python3 db/scripts/ci/test_check_bug_fix_unit_tests.py
"""

from __future__ import annotations

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / "db" / "scripts" / "ci"))

from check_bug_fix_unit_tests import main as gate_main


def run_gate(message: str, staged_files: list[str]) -> int:
    return gate_main(
        [
            "--root",
            str(ROOT),
            "--message",
            message,
            "--staged-files",
            "\n".join(staged_files),
        ]
    )


def test_feat_commit_without_tests_passes() -> None:
    assert run_gate("feat: add user profile endpoint", ["app/profile.py"]) == 0


def test_fix_commit_without_tests_fails() -> None:
    code = run_gate("fix: NPE on empty input", ["app/parser.go"])
    assert code == 1


def test_fix_commit_with_matching_go_test_passes() -> None:
    code = run_gate("fix(parser): NPE on empty input", ["app/parser.go", "app/parser_test.go"])
    assert code == 0


def test_fix_commit_with_unrelated_test_fails() -> None:
    code = run_gate("fix(parser): NPE on empty input", ["app/parser.go", "other/thing_test.go"])
    assert code == 1


def test_fix_docs_only_passes() -> None:
    assert run_gate("fix: typo in README", ["README.md"]) == 0


def test_fix_with_no_test_waiver_passes() -> None:
    code = run_gate(
        "fix: infra config\n\nno-test: pure CI config, no business logic",
        ["app/parser.go"],
    )
    assert code == 0


def test_cn_fix_detection() -> None:
    assert run_gate("修复：订单金额计算错误", ["app/order.py"]) == 1


def test_cn_fix_with_test_passes() -> None:
    assert run_gate("修复：订单金额计算错误", ["app/order.py", "app/tests/test_order.py"]) == 0


def test_hotfix_prefix_with_scope() -> None:
    assert run_gate("hotfix(api): 400 on GET", ["api/handler.go"]) == 1


def test_tests_only_commit_passes() -> None:
    assert run_gate("test: add regression for parser", ["app/parser_test.go"]) == 0


def test_python_tests_mirror_dir_passes() -> None:
    code = run_gate("fix: validate missing required field", ["app/models.py", "app/tests/test_models.py"])
    assert code == 0


def test_python_stem_overlap_passes() -> None:
    code = run_gate("fix: order total rounding", ["app/order.py", "tests/test_order_flow.py"])
    assert code == 0


def test_js_unit_test_passes() -> None:
    code = run_gate("fix: button disabled state", ["fe/src/Button.tsx", "fe/src/Button.unit.test.tsx"])
    assert code == 0


def test_fix_mjs_without_test_fails() -> None:
    code = run_gate("fix: backfill PR timing", ["onlineServiceJS/src/autoRunPrBackfill.mjs"])
    assert code == 1


def test_fix_mjs_with_matching_test_passes() -> None:
    code = run_gate(
        "fix: backfill PR timing",
        ["onlineServiceJS/src/autoRunPrBackfill.mjs", "onlineServiceJS/src/autoRunPrBackfill.test.mjs"],
    )
    assert code == 0


def test_fix_cjs_without_test_fails() -> None:
    code = run_gate("fix: config load order", ["lib/shared.cjs"])
    assert code == 1


def test_playwright_excluded_from_unit() -> None:
    code = run_gate(
        "fix: button disabled state",
        ["fe/src/Button.tsx", "e2e/Button.playwright.test.ts"],
    )
    assert code == 1


def test_revert_excluded() -> None:
    assert run_gate("revert: revert broken change", ["app/parser.go"]) == 0


def test_empty_staged_passes() -> None:
    assert run_gate("fix: something", []) == 0


def test_fix_keyword_false_positive_docs_only() -> None:
    assert run_gate("chore: fix dev env docs", ["docs/dev.md", ".github/workflows/x.yml"]) == 0


def test_fix_keyword_detection() -> None:
    assert run_gate("fix typo in auth flow", ["auth/login.py"]) == 1


def test_force_bugfix_flag() -> None:
    code = gate_main(
        [
            "--root",
            str(ROOT),
            "--message",
            "feat: sneaky",
            "--staged-files",
            "app/parser.go",
            "--force-bugfix",
        ]
    )
    assert code == 1


def test_fix_case_insensitive_prefix() -> None:
    assert run_gate("FIX: uppercase prefix", ["app/parser.go"]) == 1


def test_integration_real_commit_hooks() -> None:
    """Full git commit path with .githooks/commit-msg installed (E2E wiring guard)."""
    import os
    import shutil
    import subprocess
    import tempfile

    # git 运行 hooks 时会注入 GIT_DIR 等环境变量（gitlink 子仓时为
    # .git/modules/<name>，见 `git help hooks` 环境变量一节）。该变量会被子进程
    # 继承：若直接透传，本测试在 tmp 内执行的 `git init` 不会初始化 tmp，而是
    # 重新初始化宿主仓库的 gitdir（把 core.bare 写成 true 并污染其 config），
    # 随后 `git add` 报 "core.bare and core.worktree do not make sense"。
    # 剥离全部 GIT_* 变量，确保临时仓库完全隔离。
    clean_env = {k: v for k, v in os.environ.items() if not k.startswith("GIT_")}

    def git(*args: str, **kwargs: object) -> subprocess.CompletedProcess[str]:
        check = bool(kwargs.pop("check", False))
        return subprocess.run(
            ["git", *args], cwd=tmp, env=clean_env, **kwargs, check=check
        )

    hooks_dir = ROOT / ".githooks"
    with tempfile.TemporaryDirectory() as tmp:
        repo = Path(tmp)
        git("init", "-q", check=True)
        git("config", "user.email", "t@t.local", check=True)
        git("config", "user.name", "Gate Test", check=True)
        # Keep the installed pre-commit framework from erroring on a missing config.
        (repo / ".pre-commit-config.yaml").write_text("repos: []\n", encoding="utf-8")
        # The hooks resolve ROOT from the temp repo's git dir; the checker must
        # live inside it (same layout as the real monorepo).
        checker_dst = repo / "db" / "scripts" / "ci"
        checker_dst.mkdir(parents=True)
        shutil.copy2(ROOT / "db" / "scripts" / "ci" / "check_bug_fix_unit_tests.py",
                     checker_dst / "check_bug_fix_unit_tests.py")
        shutil.copy2(ROOT / "db" / "scripts" / "ci" / "check_logic_rollback_approval.py",
                     checker_dst / "check_logic_rollback_approval.py")
        for rel in (
            ".ai/01_project_constraints/58_logic_rollback_requires_approval.md",
            ".cursor/rules/logic-rollback-requires-approval.mdc",
            "docs/adr/0021-logic-rollback-requires-approval.md",
            ".ai/01_project_constraints/00_project_constraints.md",
        ):
            src = ROOT / rel
            dst = repo / rel
            dst.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(src, dst)
        # v13 模型: .githooks/ 入库真源 + core.hooksPath 激活（.git/hooks/ 复制模式已退役）
        (repo / ".githooks").mkdir(exist_ok=True)
        for hook in ("pre-commit", "commit-msg"):
            shutil.copy2(hooks_dir / hook, repo / ".githooks" / hook)
            Path(repo / ".githooks" / hook).chmod(0o755)
        git("config", "core.hooksPath", ".githooks", check=True)
        (repo / "parser.go").write_text(
            "package app\n\nfunc Parse(s string) string { return s }\n", encoding="utf-8"
        )
        git("add", "parser.go", check=True)
        blocked = git(
            "commit", "-m", "fix(parser): NPE on empty input",
            capture_output=True, text=True,
        )
        assert blocked.returncode != 0, blocked.stdout + blocked.stderr
        (repo / "parser_test.go").write_text(
            "package app\n\nimport \"testing\"\n"
            "func TestParse(t *testing.T) { if Parse(\"\") != \"\" { t.Fatal(\"NPE\") } }\n",
            encoding="utf-8",
        )
        git("add", "parser.go", "parser_test.go", check=True)
        passed = git(
            "commit", "-m", "fix(parser): NPE on empty input",
            capture_output=True, text=True,
        )
        assert passed.returncode == 0, passed.stdout + passed.stderr


def test_commit_msg_file_arg() -> None:
    import tempfile

    with tempfile.TemporaryDirectory() as tmp:
        msg_file = Path(tmp) / "COMMIT_EDITMSG"
        msg_file.write_text("fix(parser): NPE on empty input\n", encoding="utf-8")
        code = gate_main(
            [
                "--root",
                str(ROOT),
                "--commit-msg-file",
                str(msg_file),
                "--staged-files",
                "app/parser.go",
            ]
        )
        assert code == 1


def test_fix_css_style_file_is_not_source() -> None:
    # .css is not business source in this gate; docs/config-style exemption applies.
    assert run_gate("fix: widget glitch", ["fe/src/widget.css"]) == 0


def main() -> int:
    failures = 0
    tests = [
        fn
        for name, fn in sorted(globals().items())
        if name.startswith("test_") and callable(fn)
    ]
    for fn in tests:
        try:
            fn()
            print(f"PASS {fn.__name__}")
        except AssertionError as exc:
            failures += 1
            print(f"FAIL {fn.__name__}: {exc}", file=sys.stderr)
    print(f"\n{len(tests) - failures}/{len(tests)} tests passed.")
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())

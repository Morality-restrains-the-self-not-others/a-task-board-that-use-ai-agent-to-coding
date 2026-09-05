#!/usr/bin/env python3
"""子仓库优先提交 — 自动检测有变更的子仓库，按顺序先提交推送子仓库再提交推送主仓库。

细则：.ai/01_project_constraints/32_submodule_commit_order.md

用法:
  # 检测变更（dry-run）
  python3 runAll/scripts/commit_with_submodules.py --dry-run

  # 自动提交全部有变更的子仓库 + 主仓库（生成 commit message）
  python3 runAll/scripts/commit_with_submodules.py --apply

  # 指定 commit message
  python3 runAll/scripts/commit_with_submodules.py --apply -m "feat: add new feature"

  # 仅提交子仓库，不提交主仓库
  python3 runAll/scripts/commit_with_submodules.py --apply --no-root

  # 仅提交指定子仓库
  python3 runAll/scripts/commit_with_submodules.py --apply --repo taskAuth --repo taskEvents

  # 跳过推送（仅本地提交）
  python3 runAll/scripts/commit_with_submodules.py --apply --no-push

  # 检查子仓库 pre-commit 钩子部署状态
  python3 runAll/scripts/commit_with_submodules.py --check-hooks

  # 自动部署缺失的子仓库 pre-commit 钩子（含根仓 .githooks 同步）
  python3 runAll/scripts/commit_with_submodules.py --deploy-hooks

  # 仅同步根仓 .githooks/* → .git/hooks/*（幂等）
  python3 runAll/scripts/commit_with_submodules.py --install-root-hooks
"""

from __future__ import annotations

import argparse
import subprocess
import sys
import time
from pathlib import Path


DEFAULT_ROOT_MSG_PREFIX = "chore: update submodule pointers"


def run(cwd: Path, *args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
    """Run a command in cwd, return completed process."""
    return subprocess.run(
        list(args),
        cwd=cwd,
        capture_output=True,
        text=True,
        check=check,
    )


def monorepo_root() -> Path:
    """Infer monorepo root from this script's location."""
    here = Path(__file__).resolve()
    # runAll/scripts/this.py → parents[2] = monorepo root
    return here.parents[2]


def load_submodules(root: Path) -> list[tuple[str, Path]]:
    """Parse .gitmodules and return list of (name, absolute_path)."""
    gitmodules = root / ".gitmodules"
    if not gitmodules.is_file():
        return []

    out = subprocess.check_output(
        ["git", "config", "-f", str(gitmodules), "--get-regexp", r"^submodule\..*\.path$"],
        text=True,
    )

    subs: list[tuple[str, Path]] = []
    for line in out.splitlines():
        parts = line.split(None, 1)
        if len(parts) != 2:
            continue
        key = parts[0].strip()
        rel_path = parts[1].strip()
        # key format: submodule.<name>.path
        name = key.split(".")[1] if key.count(".") >= 2 else rel_path
        abs_path = (root / rel_path).resolve()
        if abs_path.is_dir():
            subs.append((name, abs_path))
    return sorted(subs, key=lambda x: x[0])


def has_git(repo: Path) -> bool:
    """Check if repo has a .git directory or file (submodule)."""
    git = repo / ".git"
    return git.is_dir() or git.is_file()


def has_uncommitted(repo: Path) -> bool:
    """Check if repo has uncommitted changes (staged or unstaged)."""
    r = run(repo, "git", "status", "--porcelain", check=False)
    return bool((r.stdout or "").strip())


def has_unpushed(repo: Path) -> bool:
    """Check if repo has commits not yet pushed to remote."""
    # Check if there's an upstream tracking branch
    r = run(repo, "git", "rev-parse", "--abbrev-ref", "@{u}", check=False)
    if r.returncode != 0:
        # No upstream configured — treat as needing push
        r2 = run(repo, "git", "rev-list", "--count", "HEAD", check=False)
        try:
            return int((r2.stdout or "0").strip()) > 0
        except ValueError:
            return True
    # Compare with upstream
    r3 = run(repo, "git", "rev-list", "--count", "@{u}..HEAD", check=False)
    try:
        return int((r3.stdout or "0").strip()) > 0
    except ValueError:
        return False


def hooks_path_ok(repo: Path) -> bool:
    """v13: core.hooksPath == .githooks（Git 原生激活，零复制）。"""
    r = run(repo, "git", "config", "core.hooksPath", check=False)
    return r.returncode == 0 and (r.stdout or "").strip() == ".githooks"


def has_precommit_hook(repo: Path) -> bool:
    """v15: .githooks/pre-commit 入库真源 + hooksPath 激活 + v1.1.0 锁校验存在。

    git-hooks-version-control：.githooks/ 入库真源 + core.hooksPath 激活。
    v1.1.0: 旧版（无 session_hub_lock_check）视为未部署 → --deploy-hooks 会升级。
    """
    if (repo / ".githooks" / "pre-commit").is_file() and hooks_path_ok(repo):
        text = (repo / ".githooks" / "pre-commit").read_text(errors="ignore")
        if "session_hub_lock_check" in text:
            return True
    # 兼容旧布局（迁移期未完成时仍可识别）
    if (repo / "scripts" / "hooks" / "pre-commit").is_file():
        return True
    return False


def deploy_hooks_for_repos(root: Path, repos: list[str]) -> tuple[bool, str]:
    """Deploy random unit test pre-commit hooks to specified repos."""
    deploy_script = root / "scripts" / "deploy_repo_random_precommit.sh"
    if not deploy_script.is_file():
        return False, f"Deploy script not found: {deploy_script}"

    args = ["bash", str(deploy_script)] + repos
    r = subprocess.run(args, cwd=root, capture_output=True, text=True, check=False)
    if r.returncode != 0:
        return False, f"Deploy failed:\n{r.stderr or ''}\n{r.stdout or ''}"
    return True, r.stdout


def install_root_hooks(root: Path) -> tuple[bool, str]:
    """v13: 校验根仓 hooksPath 激活 + .git/hooks 无残留副本（复制模式已退役）。

    git-hooks-version-control：.githooks/ 入库真源 + core.hooksPath 激活。
    """
    script = root / "runAll" / "scripts" / "install_root_hooks.sh"
    if not script.is_file():
        return False, f"Root hook install script not found: {script}"
    r = subprocess.run(["bash", str(script)], cwd=root, capture_output=True, text=True, check=False)
    if r.returncode != 0:
        return False, (r.stderr or r.stdout or "").strip()
    return True, (r.stdout or "").strip()


def generate_commit_message(repo_name: str, repo_path: Path, include_untracked: bool = False) -> str:
    """Auto-generate a commit message based on changed files.

    OPT-20260815-010: 用 `git diff --name-only` + `git ls-files -o` 生成说明，
    勿手切 porcelain（porcelain 的 ` M foo.go` 行去掉前 3 字符会得到 `oo.go`）。
    """
    r = run(repo_path, "git", "diff", "--name-only", check=False)
    changed_files = [l.strip() for l in (r.stdout or "").splitlines() if l.strip()]
    if include_untracked:
        r2 = run(repo_path, "git", "ls-files", "-o", "--exclude-standard", check=False)
        changed_files += [l.strip() for l in (r2.stdout or "").splitlines() if l.strip()]
    # 去重保序（diff 与 ls-files 可能有交集）
    changed_files = list(dict.fromkeys(changed_files))
    # 黑名单过滤（防 message 里出现将被排除的敏感/缓存路径）
    changed_files = [f for f in changed_files if not _is_excluded(f)]
    if not changed_files:
        return f"chore: update {repo_name}"

    # Build a simple summary
    if len(changed_files) == 1:
        fname = Path(changed_files[0]).name
        return f"chore: update {repo_name} — {fname}"
    elif len(changed_files) <= 5:
        fnames = ", ".join(Path(f).name for f in changed_files)
        return f"chore: update {repo_name} — {fnames}"
    else:
        return f"chore: update {repo_name} — {len(changed_files)} files changed"


def status_emoji(repo: Path) -> str:
    """Return a status indicator for a repo."""
    uc = has_uncommitted(repo)
    up = has_unpushed(repo)
    if uc and up:
        return "🔴"  # both uncommitted and unpushed
    elif uc:
        return "🟡"  # uncommitted only
    elif up:
        return "🟠"  # unpushed only
    return "🟢"  # clean


def dirty_repos(root: Path, subs: list[tuple[str, Path]]) -> list[tuple[str, Path]]:
    """Return list of submodules with uncommitted or unpushed changes, plus root if dirty."""
    dirty: list[tuple[str, Path]] = []
    for name, path in subs:
        if not has_git(path):
            continue
        if has_uncommitted(path) or has_unpushed(path):
            dirty.append((name, path))
    # Check root repo
    if has_uncommitted(root) or has_unpushed(root):
        dirty.append(("(root)", root))
    return dirty


# 敏感/缓存路径黑名单 — 无论 add 模式都必须排除（OPT-20260815-010）
STAGING_EXCLUDES = ("config.local.yaml", "__pycache__", "*.pyc")


def _is_excluded(rel_path: str) -> bool:
    """判断相对路径是否命中黑名单（文件名或路径段匹配）。"""
    import fnmatch
    for pattern in STAGING_EXCLUDES:
        if fnmatch.fnmatch(rel_path, pattern) or fnmatch.fnmatch(Path(rel_path).name, pattern):
            return True
    return "__pycache__" in Path(rel_path).parts


def _exclude_staged_paths(path: Path) -> None:
    """取消暂存黑名单路径，防止 --include-untracked 时误收密钥/缓存。"""
    for pattern in STAGING_EXCLUDES:
        run(path, "git", "reset", "-q", "--", f":(glob)**/{pattern}", check=False)


def commit_repo(
    name: str,
    path: Path,
    message: str,
    *,
    apply: bool,
    push: bool,
    include_untracked: bool = False,
) -> bool:
    """Commit and optionally push changes in a repo. Returns True on success."""
    if not has_uncommitted(path):
        print(f"  {name}: no uncommitted changes, skip commit")
        if push and has_unpushed(path):
            print(f"  {name}: has unpushed commits, pushing...")
            if apply:
                r = run(path, "git", "push", check=False)
                if r.returncode != 0:
                    err = (r.stderr or r.stdout or "").strip()
                    print(f"  {name}: PUSH FAILED — {err}")
                    return False
                print(f"  {name}: pushed ✓")
        return True

    if not apply:
        staged_desc = "git add -A" if include_untracked else "git add -u (仅已跟踪)"
        print(f"  [dry-run] {staged_desc} && git commit -m '{message}'")
        if push and has_unpushed(path):
            changed_count = _count_changes(path)
            print(f"  [dry-run] git push ({changed_count} commits ahead)")
        return True

    # OPT-20260815-010: 默认只 add 已跟踪变更（-u）；untracked 需显式允许
    if include_untracked:
        r = run(path, "git", "add", "-A", check=False)
    else:
        r = run(path, "git", "add", "-u", check=False)
    if r.returncode != 0:
        print(f"  {name}: git add FAILED — {(r.stderr or '').strip()}")
        return False
    # 无论 add 模式，显式排除敏感/缓存路径
    _exclude_staged_paths(path)

    # Check if there's anything to commit after staging
    r2 = run(path, "git", "diff", "--cached", "--quiet", check=False)
    if r2.returncode == 0:
        print(f"  {name}: nothing to commit after staging")
        return True

    # Commit
    r3 = run(path, "git", "commit", "-m", message, check=False)
    if r3.returncode != 0:
        err = (r3.stderr or r3.stdout or "").strip()
        print(f"  {name}: COMMIT FAILED — {err}")
        return False
    print(f"  {name}: committed ✓")

    # Push
    if push:
        r4 = run(path, "git", "push", check=False)
        if r4.returncode != 0:
            err = (r4.stderr or r4.stdout or "").strip()
            print(f"  {name}: PUSH FAILED — {err}")
            return False
        # Also push tags if any
        run(path, "git", "push", "--tags", check=False)
        print(f"  {name}: pushed ✓")

    return True


def _count_changes(path: Path) -> int:
    """Count commits ahead of upstream."""
    r = run(path, "git", "rev-list", "--count", "@{u}..HEAD", check=False)
    try:
        return int((r.stdout or "0").strip())
    except ValueError:
        return 0


def session_settings_ok(repo: Path) -> bool:
    """v15: 仓级 .claude/settings.json 会话钩子接线（SessionStart/SessionEnd/sessionctl.sh）。

    OPT-20260806-001: 会话钩子分发到全部仓 — 无论 Claude Code 在哪个仓目录启动，
    session-hub 互知/心跳/提交前探测均生效。机器本地配置（不入库），部署器负责同步。
    """
    settings = repo / ".claude" / "settings.json"
    if not settings.is_file():
        return False
    text = settings.read_text(errors="ignore")
    return all(marker in text for marker in ("SessionStart", "SessionEnd", "sessionctl.sh"))


def session_hooks_ok(root: Path) -> tuple[bool, list[str]]:
    """v14: Session Hub 互知接线完整性（agent-session-coordination）。
    - meta root .claude/settings.json 含 SessionStart/SessionEnd/PreToolUse 钩子
    - scripts/lib/sessionctl.sh 薄封装存在
    - .githooks/HOOK_VERSION 含 session-hooks 条目（或 pre-commit >= 1.1.0）
    - pre-commit 含 session_hub_lock_check 调用
    """
    problems: list[str] = []
    settings = root / ".claude" / "settings.json"
    if not settings.is_file():
        problems.append("meta .claude/settings.json 缺失（SessionStart 钩子未接线）")
    else:
        text = settings.read_text(errors="ignore")
        for marker in ("SessionStart", "SessionEnd", "sessionctl.sh"):
            if marker not in text:
                problems.append(f".claude/settings.json 缺 {marker} 接线")
    if not (root / "scripts" / "lib" / "sessionctl.sh").is_file():
        problems.append("scripts/lib/sessionctl.sh 缺失")
    if not (root / "scripts" / "lib" / "session_lock_check.sh").is_file():
        problems.append("scripts/lib/session_lock_check.sh 缺失")
    precommit = root / ".githooks" / "pre-commit"
    if precommit.is_file():
        text = precommit.read_text(errors="ignore")
        if "session_hub_lock_check" not in text:
            problems.append(".githooks/pre-commit 未含提交锁校验（需 v1.1.0）")
    return len(problems) == 0, problems


def check_and_report_hooks(root: Path, subs: list[tuple[str, Path]]) -> tuple[list[str], list[str]]:
    """Check pre-commit hook status for all submodules.
    Returns (ok_repos, missing_repos).
    """
    ok: list[str] = []
    missing: list[str] = []
    settings_ok: list[str] = []
    settings_missing: list[str] = []
    for name, path in subs:
        if not has_git(path):
            continue
        if has_precommit_hook(path):
            ok.append(name)
        else:
            missing.append(name)
        if session_settings_ok(path):
            settings_ok.append(name)
        else:
            settings_missing.append(name)

    # v14: Session Hub 互知接线（meta root 维度）
    sh_ok, sh_problems = session_hooks_ok(root)
    print(f"\n{'='*60}")
    print("Session Hub 互知接线 (v14 agent-session-coordination)")
    print(f"{'='*60}")
    if sh_ok:
        print("  ✅ 完整（settings.json hooks / sessionctl.sh / pre-commit v1.1.0 锁校验）")
    else:
        print("  ⚠ 不完整:")
        for p in sh_problems:
            print(f"    - {p}")
    print()

    # v15 (OPT-20260806-001): 每仓会话钩子 settings.json presence
    print(f"\n{'='*60}")
    print("Session 钩子 settings.json 分发 (OPT-20260806-001)")
    print(f"{'='*60}")
    root_ok = session_settings_ok(root)
    print(f"  meta root:      {'✅' if root_ok else '❌ 缺失'}")
    print(f"  子仓已分发:     {len(settings_ok)} repos")
    print(f"  子仓未分发:     {len(settings_missing)} repos")
    if settings_missing:
        print(f"\n  Missing ({len(settings_missing)}):")
        for name in settings_missing:
            print(f"    - {name}")
    print()
    return ok, missing


def print_hook_status(ok: list[str], missing: list[str]) -> None:
    """Print a formatted hook status report."""
    print(f"\n{'='*60}")
    print("Pre-commit Hook Status")
    print(f"{'='*60}")
    print(f"  With hook:    {len(ok)} repos")
    print(f"  Missing hook: {len(missing)} repos")
    if missing:
        print(f"\n  Missing ({len(missing)}):")
        for name in missing:
            print(f"    - {name}")
    print()


def main() -> int:
    parser = argparse.ArgumentParser(
        description="子仓库优先提交 — 先提交推送子仓库再提交推送主仓库"
    )
    parser.add_argument("--apply", action="store_true", help="Actually commit/push (default: dry-run)")
    parser.add_argument("--dry-run", action="store_true", help="Preview only (default)")
    parser.add_argument("-m", "--message", default="", help="Commit message for root repo (auto-generated for subs)")
    parser.add_argument("--no-root", action="store_true", help="Skip root repo commit")
    parser.add_argument("--no-push", action="store_true", help="Skip pushing to remote")
    parser.add_argument("--repo", action="append", default=[], help="Limit to specific repos (can repeat)")
    parser.add_argument("--include-untracked", action="store_true",
                        help="OPT-20260815-010: 同时暂存未跟踪文件（默认仅已跟踪变更）")
    parser.add_argument("--allow-feature-branch", action="store_true",
                        help="OPT-20260815-010: 允许在非 main 分支提交（默认拒绝）")
    parser.add_argument("--check-hooks", action="store_true", help="Check pre-commit hook deployment status and exit")
    parser.add_argument("--deploy-hooks", action="store_true", help="Auto-deploy missing pre-commit hooks and exit")
    parser.add_argument("--install-root-hooks", action="store_true",
                        help="校验根仓 hooksPath 激活并退出（v13，幂等）")
    parser.add_argument(
        "--root",
        default="",
        help="Monorepo root (default: inferred from script path)",
    )
    args = parser.parse_args()

    apply_mode = bool(args.apply) and not bool(args.dry_run)
    push = not bool(args.no_push)
    root = Path(args.root).resolve() if args.root else monorepo_root()

    if not root.is_dir():
        print(f"ERROR: monorepo root not found: {root}", file=sys.stderr)
        return 1

    # Load all submodules
    subs = load_submodules(root)
    if not subs:
        print("No submodules found in .gitmodules", file=sys.stderr)
        return 1

    # --- Root-hook-only mode ---
    if args.install_root_hooks:
        ok, output = install_root_hooks(root)
        if not ok:
            print(f"ERROR: root hooks install failed:\n{output}", file=sys.stderr)
            return 1
        print(output)
        return 0

    # --- Hook-only modes ---
    if args.check_hooks or args.deploy_hooks:
        # 根仓门禁（commit-msg / pre-commit / pre-push）与子仓钩子一并保持最新
        if args.deploy_hooks:
            ok, output = install_root_hooks(root)
            print(f"Root hooks: {output}")

        ok, missing = check_and_report_hooks(root, subs)
        print_hook_status(ok, missing)

        if args.deploy_hooks:
            # v15 (OPT-20260806-001): 全量同步（幂等）— 钩子模板升级 + 每仓 settings.json 会话钩子
            # 即使钩子已全绿也运行：settings.json 为机器本地配置，需随 SSOT 同步
            all_subs = [n for n, _ in subs]
            print(f"Deploying hooks + session settings.json to {len(all_subs)} repos...")
            success, output = deploy_hooks_for_repos(root, all_subs)
            if success:
                print("Deployment completed.")
                # Re-check
                ok2, missing2 = check_and_report_hooks(root, subs)
                if missing2:
                    print(f"WARNING: {len(missing2)} repos still missing hooks after deploy:")
                    for name in missing2:
                        print(f"  - {name}")
                else:
                    print("All repos now have pre-commit hooks ✓")
            else:
                print(f"Hook deployment failed:\n{output}")
                return 1
        elif args.check_hooks and missing:
            print("Run with --deploy-hooks to auto-deploy missing hooks.")
            return 1 if missing else 0

        return 0

    # --- Main commit flow ---
    # Filter repos if --repo specified
    wanted_repos = set(args.repo) if args.repo else None

    # Find dirty repos (submodules first, root last)
    dirty_subs: list[tuple[str, Path]] = []
    dirty_root = False
    for name, path in subs:
        if not has_git(path):
            continue
        if wanted_repos and name not in wanted_repos:
            continue
        if has_uncommitted(path) or (push and has_unpushed(path)):
            dirty_subs.append((name, path))

    if not args.no_root:
        root_dirty = has_uncommitted(root) or (push and has_unpushed(root))
        if root_dirty and (not wanted_repos or "(root)" in wanted_repos or "root" in wanted_repos):
            dirty_root = True

    if not dirty_subs and not dirty_root:
        print("No changes to commit — all repos clean ✓")
        # Still check hooks
        ok, missing = check_and_report_hooks(root, subs)
        if missing:
            print(f"\n⚠  {len(missing)} repo(s) missing pre-commit hooks:")
            for name in missing:
                print(f"  - {name}")
            print("  Run with --deploy-hooks to fix.")
        return 0

    mode = "APPLY" if apply_mode else "DRY-RUN"
    print(f"[{mode}] root={root} push={push}")
    print()

    # Display status table
    print(f"{'Repo':<35} {'Status':<10} {'Hook'}")
    print("-" * 55)
    for name, path in dirty_subs:
        emoji = status_emoji(path)
        hook = "✓" if has_precommit_hook(path) else "✗"
        print(f"  {name:<33} {emoji:<10} {hook}")
    if dirty_root:
        emoji = status_emoji(root)
        hook = "✓" if has_precommit_hook(root) else "✗"
        print(f"  (root){' '*28} {emoji:<10} {hook}")
    print()

    # Warn about missing hooks before proceeding
    missing_hooks: list[str] = []
    for name, path in dirty_subs:
        if not has_precommit_hook(path):
            missing_hooks.append(name)
    if dirty_root and not has_precommit_hook(root):
        missing_hooks.append("(root)")

    if missing_hooks:
        print(f"⚠  {len(missing_hooks)} repo(s) missing pre-commit hooks:")
        for name in missing_hooks:
            print(f"  - {name}")
        if apply_mode:
            print("  Auto-deploying missing hooks before commit...")
            # Filter to only actual submodule names (not root)
            deploy_targets = [n for n in missing_hooks if n != "(root)"]
            if deploy_targets:
                success, output = deploy_hooks_for_repos(root, deploy_targets)
                if not success:
                    print(f"WARNING: Hook deployment failed. Continuing anyway...\n{output}")
                else:
                    print("  Hooks deployed ✓")
            # Re-check
            still_missing = []
            for name, path in dirty_subs:
                if not has_precommit_hook(path):
                    still_missing.append(name)
            if still_missing:
                print(f"  WARNING: {len(still_missing)} repos still missing hooks, but proceeding...")
        else:
            print("  Run with --deploy-hooks to fix, or --apply to auto-deploy.")
        print()

    if not apply_mode:
        print("Run with --apply to execute.\n")
        return 0

    # === APPLY MODE ===

    # OPT-20260815-010: 默认只提交 main 分支；feature 分支须显式放行
    if not args.allow_feature_branch:
        for name, path in dirty_subs + ([( "(root)", root)] if dirty_root else []):
            br = run(path, "git", "branch", "--show-current", check=False)
            branch = (br.stdout or "").strip()
            if branch and branch != "main":
                print(f"⛔ {name} 当前在分支 '{branch}'（非 main）— 拒绝提交。")
                print("   先切回 main（或提交前合并），或显式用 --allow-feature-branch 放行。")
                return 1

    # 提交前同步根仓权威门禁（commit-msg / pre-commit / pre-push），
    # 确保新门禁在既有开发机生效（OPT-20260805-003）
    print("=" * 60)
    print("Phase 0: Sync root git hooks (.githooks -> .git/hooks)")
    print("=" * 60)
    ok_hooks, hook_output = install_root_hooks(root)
    if not ok_hooks:
        print(f"⚠ Root hooks sync failed:\n{hook_output}")
    else:
        print(f"  {hook_output}")
    print()

    # Phase 1: Commit and push all dirty submodules
    print("=" * 60)
    print("Phase 1: Submodule commits")
    print("=" * 60)

    failed: list[str] = []
    for name, path in dirty_subs:
        print(f"\n--- {name} ---")
        # Show what changed
        r = run(path, "git", "status", "--short", check=False)
        if (r.stdout or "").strip():
            print((r.stdout or "").strip())

        msg = generate_commit_message(name, path, include_untracked=args.include_untracked)
        print(f"  Commit message: {msg}")

        if not commit_repo(name, path, msg, apply=True, push=push,
                           include_untracked=args.include_untracked):
            failed.append(name)
            print(f"  ⛔ FAILED to commit {name}")

    if failed:
        print(f"\n⚠  {len(failed)} submodule(s) failed: {', '.join(failed)}")
        print("Fix the issues above and re-run. Root commit skipped.")
        return 1

    # Phase 2: Commit and push root repo
    if not dirty_root:
        print("\n✅ All submodules committed. Root repo has no changes.")
        return 0

    print(f"\n{'='*60}")
    print("Phase 2: Root repo commit")
    print("=" * 60)

    root_msg = args.message
    if not root_msg:
        # Auto-generate root message listing updated submodules
        sub_names = [name for name, _ in dirty_subs]
        if len(sub_names) <= 3:
            root_msg = f"{DEFAULT_ROOT_MSG_PREFIX}: {', '.join(sub_names)}"
        else:
            root_msg = f"{DEFAULT_ROOT_MSG_PREFIX}: {len(sub_names)} submodules"

    print(f"\n--- (root) ---")
    r = run(root, "git", "status", "--short", check=False)
    if (r.stdout or "").strip():
        print((r.stdout or "").strip())
    print(f"  Commit message: {root_msg}")

    if not commit_repo("(root)", root, root_msg, apply=True, push=push,
                       include_untracked=args.include_untracked):
        print("⛔ FAILED to commit root repo")
        return 1

    print(f"\n{'='*60}")
    print("✅ All done!")
    print(f"   Submodules committed: {len(dirty_subs)}")
    print(f"   Root committed:       {1 if dirty_root else 0}")
    print(f"   Failed:               {len(failed)}")
    print(f"{'='*60}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())

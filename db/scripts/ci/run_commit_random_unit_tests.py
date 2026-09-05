#!/usr/bin/env python3
"""Commit-time random unit tests + legacy debt 10% fix quota.

SSOT rule: .ai/01_project_constraints/28_commit_random_unit_test_debt_fix.md
Config: db/scripts/ci/commit_random_unit_tests.yaml

Exit codes:
  0 — sample passed, or legacy quota met and remaining debt recorded
  1 — failures pending (quota not met) or staged-related hard fail
  2 — configuration / IO error
"""

from __future__ import annotations

import argparse
import json
import math
import os
import random
import re
import subprocess
import sys
import time
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path, PurePosixPath
from typing import Any

try:
    import yaml
except ImportError:  # pragma: no cover
    print("ERROR: PyYAML required (import yaml)", file=sys.stderr)
    raise SystemExit(2)

SCRIPT_DIR = Path(__file__).resolve().parent
DEFAULT_CONFIG = SCRIPT_DIR / "commit_random_unit_tests.yaml"
REPORT_REL = Path("commitResult/random_unit_tests/last_run.json")
DEBT_REL = Path(".learnings/UNIT_TEST_DEBT.md")


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError("db/registry.yaml not found above checker")


def load_config(path: Path) -> dict[str, Any]:
    data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    if not isinstance(data, dict):
        raise ValueError(f"{path}: root must be a mapping")
    return data


def required_fix_count(n_failures: int, ratio: float = 0.10) -> int:
    """At least 10% of discovered failures, ceil, minimum 1 when N>=1."""
    if n_failures <= 0:
        return 0
    return max(1, math.ceil(n_failures * ratio))


def _norm(rel: str) -> str:
    return rel.replace("\\", "/").lstrip("./")


def _match_glob(rel: str, pattern: str) -> bool:
    """Path.match plus **/…/** prefix/segment excludes (stdlib match misses mid dirs)."""
    norm = _norm(rel)
    p = PurePosixPath(norm)
    pat = pattern.replace("\\", "/")
    if p.match(pat):
        return True
    # **/foo/** or **/a/b/** → path under that prefix or containing that segment
    if pat.startswith("**/") and pat.endswith("/**"):
        mid = pat[3:-3]
        if not mid:
            return False
        if "/" in mid:
            if norm == mid or norm.startswith(mid + "/"):
                return True
            if f"/{mid}/" in f"/{norm}/":
                return True
        elif mid in p.parts:
            return True
    # **/name.ext → basename match
    if pat.startswith("**/") and "/" not in pat[3:]:
        if PurePosixPath(p.name).match(pat[3:]):
            return True
    return False


def _match_any(rel: str, globs: list[str]) -> bool:
    return any(_match_glob(rel, g) for g in globs)


def discover_unit_tests(root: Path, cfg: dict[str, Any]) -> list[str]:
    includes = list(cfg.get("include_globs") or [])
    excludes = list(cfg.get("exclude_globs") or [])
    hard_prune = {
        "node_modules",
        ".git",
        "vendor",
        "venv",
        ".venv",
        "site-packages",
        "__pycache__",
        "gitlab_home",
    }
    found: list[str] = []
    for dirpath, dirnames, filenames in os.walk(root):
        rel_dir = Path(dirpath).relative_to(root).as_posix()
        pruned = []
        for d in list(dirnames):
            cand = f"{rel_dir}/{d}".lstrip("/")
            if d in hard_prune or _match_any(cand, excludes) or _match_any(cand + "/", excludes):
                pruned.append(d)
        for d in pruned:
            dirnames.remove(d)
        for name in filenames:
            rel = f"{rel_dir}/{name}".lstrip("/")
            if not _match_any(rel, includes):
                continue
            if _match_any(rel, excludes):
                continue
            found.append(rel)
    return sorted(set(found))


def sample_tests(
    pool: list[str],
    *,
    ratio: float,
    min_sample: int,
    max_sample: int,
    rng: random.Random,
    prefer_globs: list[str] | None = None,
) -> list[str]:
    if not pool:
        return []
    n = max(min_sample, math.ceil(len(pool) * ratio))
    n = min(n, max_sample, len(pool))

    # OPT-20260719-031: layered sampling — prefer fast JS/Python unit tests first,
    # fill remaining slots with Go tests (avoid expensive Go package compilation).
    if prefer_globs:
        tiers: list[list[str]] = []
        remaining = set(pool)
        for pat in prefer_globs:
            tier = sorted([p for p in remaining if _match_glob(p, pat)])
            if tier:
                remaining -= set(tier)
                tiers.append(tier)
        # Any unmatched go to a final "rest" tier
        if remaining:
            tiers.append(sorted(remaining))

        sampled: list[str] = []
        # Distribute: at least 1 from each non-empty tier, then round-robin fill
        for tier in tiers:
            if not tier or len(sampled) >= n:
                break
            sampled.append(tier.pop(rng.randint(0, len(tier) - 1)))

        # Fill remaining from all tiers (flatten remaining)
        flat: list[str] = []
        for tier in tiers:
            flat.extend(tier)
        rng.shuffle(flat)
        needed = n - len(sampled)
        if needed > 0 and flat:
            sampled.extend(flat[:needed])

        return sorted(sampled)

    return sorted(rng.sample(pool, n))


def staged_files(root: Path) -> set[str]:
    try:
        out = subprocess.check_output(
            ["git", "diff", "--cached", "--name-only", "--diff-filter=ACMR"],
            cwd=root,
            text=True,
            stderr=subprocess.DEVNULL,
        )
    except (subprocess.CalledProcessError, FileNotFoundError):
        return set()
    return {line.strip() for line in out.splitlines() if line.strip()}


@dataclass
class RunResult:
    path: str
    ok: bool
    detail: str


def run_one(root: Path, rel: str, timeout: int) -> RunResult:
    path = root / rel
    if not path.is_file():
        return RunResult(rel, False, "missing file")
    try:
        if rel.endswith((".unit.test.js", ".unit.test.ts")):
            return _run_js_unit(root, rel, timeout)
        if rel.endswith("_test.go"):
            return _run_go_test(root, rel, timeout)
        if rel.endswith(".py"):
            return _run_pytest(root, rel, timeout)
        return RunResult(rel, True, "skipped unknown type")
    except subprocess.TimeoutExpired:
        return RunResult(rel, False, f"timeout>{timeout}s")
    except Exception as exc:  # noqa: BLE001 — surface runner errors as fail
        return RunResult(rel, False, f"{type(exc).__name__}: {exc}")


def _run_js_unit(root: Path, rel: str, timeout: int) -> RunResult:
    # Vitest roots — tests importing .vue / using vi.* must run via vitest, not bare node.
    vitest_roots = [
        ("task2app/front_project/app/", root / "task2app" / "front_project" / "app"),
        ("taskFE/app/", root / "taskFE" / "app"),
    ]
    for prefix, app_dir in vitest_roots:
        if rel.startswith(prefix) and (app_dir / "package.json").is_file():
            inner = rel[len(prefix):]
            proc = subprocess.run(
                ["npx", "vitest", "run", inner],
                cwd=app_dir,
                capture_output=True,
                text=True,
                timeout=timeout,
                env={**os.environ, "CI": "1"},
            )
            ok = proc.returncode == 0
            detail = (proc.stdout + proc.stderr)[-2000:]
            return RunResult(rel, ok, detail if not ok else "ok")
    # Generic: node file (may no-op skip)
    proc = subprocess.run(
        ["node", str(root / rel)],
        cwd=root,
        capture_output=True,
        text=True,
        timeout=timeout,
    )
    return RunResult(rel, proc.returncode == 0, (proc.stdout + proc.stderr)[-2000:])


_GO_TEST_FUNC_RE = re.compile(r"^func (Test[A-Za-z0-9_]+)\(")


def go_test_func_names(source: str) -> list[str]:
    """Extract `func TestXxx(` names from a Go test file (declaration lines only)."""
    names: list[str] = []
    seen: set[str] = set()
    for line in source.splitlines():
        m = _GO_TEST_FUNC_RE.match(line.rstrip())
        if not m:
            continue
        name = m.group(1)
        if name not in seen:
            seen.add(name)
            names.append(name)
    return names


def go_test_run_pattern(names: list[str]) -> str:
    """`-run` regexp that selects only the given top-level Test functions."""
    if not names:
        return ""
    return "^(" + "|".join(re.escape(n) for n in names) + ")$"


def go_test_is_helper_file(source: str) -> bool:
    """True when a `_test.go` has no `func TestXxx` (shared fixture / helpers)."""
    return not go_test_func_names(source)


def _run_go_test(root: Path, rel: str, timeout: int) -> RunResult:
    pkg_dir = (root / rel).parent
    src = (root / rel).read_text(encoding="utf-8", errors="replace")
    names = go_test_func_names(src)
    if go_test_is_helper_file(src):
        # Shared helpers have no TestXxx. Running `.` without -run executes the
        # whole package (taskCloudService/src often >300s) and false-fails this file.
        return RunResult(rel, True, "no Test functions (package helper)")
    cmd = ["go", "test", "-count=1", f"-timeout={timeout}s"]
    # File-level sampling must not execute the rest of a large package:
    # taskCloudService/src 全包常 >300s，会把抽到的单文件误判为失败。
    pattern = go_test_run_pattern(names)
    cmd.extend(["-run", pattern])
    cmd.append(".")
    proc = subprocess.run(
        cmd,
        cwd=pkg_dir,
        capture_output=True,
        text=True,
        timeout=timeout + 5,
        check=False,
    )
    return RunResult(rel, proc.returncode == 0, (proc.stdout + proc.stderr)[-2000:])


def _run_pytest(root: Path, rel: str, timeout: int) -> RunResult:
    env = {**os.environ, "USE_IN_MEMORY_SERVICES": "true"}
    if rel.startswith("task2app/Saas_project/"):
        env["DJANGO_SETTINGS_MODULE"] = "saas_project.settings_test"
    proc = subprocess.run(
        [sys.executable, "-m", "pytest", rel, "-q", "--tb=line"],
        cwd=root,
        capture_output=True,
        text=True,
        timeout=timeout,
        env=env,
    )
    return RunResult(rel, proc.returncode == 0, (proc.stdout + proc.stderr)[-2000:])


def load_report(path: Path) -> dict[str, Any] | None:
    if not path.is_file():
        return None
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError:
        return None
    return data if isinstance(data, dict) else None


def save_report(path: Path, data: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")


def append_debt(root: Path, remaining: list[str], note: str) -> None:
    if not remaining:
        return
    debt_path = root / DEBT_REL
    debt_path.parent.mkdir(parents=True, exist_ok=True)
    ts = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    lines = [
        "",
        f"## {ts}",
        "",
        f"- Note: {note}",
        "- Remaining failures (deferred after 10% quota):",
    ]
    for p in remaining:
        lines.append(f"  - `{p}`")
    lines.append("")
    if not debt_path.is_file():
        header = (
            "# Unit Test Debt\n\n"
            "遗留单元测试失败清单（由提交随机单测门禁追加）。"
            "细则见 `.ai/01_project_constraints/28_commit_random_unit_test_debt_fix.md`。\n"
        )
        debt_path.write_text(header + "\n".join(lines), encoding="utf-8")
    else:
        with debt_path.open("a", encoding="utf-8") as fh:
            fh.write("\n".join(lines))


def _print_quota_help(failed: list[str], required: int) -> None:
    print("", file=sys.stderr)
    print(
        f"LEGACY UNIT TEST DEBT: discovered {len(failed)} failing file(s); "
        f"must fix at least {required} (10% ceil, min 1).",
        file=sys.stderr,
    )
    for f in failed:
        print(f"  FAIL {f}", file=sys.stderr)
    print(
        "Fix >= quota, stage changes, re-run this gate "
        "(it re-checks previous failures before a new sample).",
        file=sys.stderr,
    )


def try_clear_blocked_report(
    root: Path,
    report_path: Path,
    cfg: dict[str, Any],
    timeout: int,
) -> bool:
    """If a blocked report exists, verify 10% fix quota. Returns True if cleared."""
    report = load_report(report_path)
    if not report or report.get("status") != "blocked":
        return True
    failed = [str(x) for x in (report.get("failed") or [])]
    required = int(report.get("required_fix_count") or required_fix_count(len(failed)))
    if not failed:
        report_path.unlink(missing_ok=True)
        return True
    print(f"Re-checking {len(failed)} previously failed unit test file(s) for debt quota...")
    now_pass: list[str] = []
    still_fail: list[str] = []
    for rel in failed:
        res = run_one(root, rel, timeout)
        (now_pass if res.ok else still_fail).append(rel)
        print(f"  {'PASS' if res.ok else 'FAIL'} {rel}")
    if len(now_pass) < required:
        print(
            f"Quota not met: fixed {len(now_pass)}/{required} required "
            f"(still failing: {len(still_fail)}).",
            file=sys.stderr,
        )
        _print_quota_help(still_fail, required - len(now_pass))
        return False
    append_debt(
        root,
        still_fail,
        f"quota met: fixed {len(now_pass)}/{len(failed)} (required {required})",
    )
    print(
        f"Debt quota met: fixed {len(now_pass)} (>= {required}); "
        f"deferred {len(still_fail)} to {DEBT_REL}."
    )
    report_path.unlink(missing_ok=True)
    return True


def run_sample(
    root: Path,
    cfg: dict[str, Any],
    *,
    seed: int | None,
    staged: set[str],
) -> int:
    timeout = int(cfg.get("timeout_sec") or 120)
    ratio = float(cfg.get("sample_ratio") or 0.10)
    debt_ratio = float(cfg.get("debt_fix_ratio") or 0.10)
    min_sample = int(cfg.get("min_sample") or 1)
    max_sample = int(cfg.get("max_sample") or 8)

    pool = discover_unit_tests(root, cfg)
    if not pool:
        print("No unit tests in pool; gate skipped.")
        return 0

    rng_seed = seed if seed is not None else int(time.time()) ^ os.getpid()
    rng = random.Random(rng_seed)
    prefer_globs = list(cfg.get("prefer_globs") or [])
    sampled = sample_tests(
        pool, ratio=ratio, min_sample=min_sample, max_sample=max_sample,
        rng=rng, prefer_globs=prefer_globs,
    )
    print(
        f"Random unit tests: pool={len(pool)} sample={len(sampled)} "
        f"seed={rng_seed} ratio={ratio}"
    )

    results: list[RunResult] = []
    for rel in sampled:
        print(f"Running {rel} ...")
        results.append(run_one(root, rel, timeout))

    failed = [r.path for r in results if not r.ok]
    passed = [r.path for r in results if r.ok]
    staged_failed = [p for p in failed if p in staged]
    legacy_failed = [p for p in failed if p not in staged]

    report = {
        "status": "ok" if not failed else "blocked",
        "seed": rng_seed,
        "sampled": sampled,
        "passed": passed,
        "failed": failed,
        "staged_failed": staged_failed,
        "legacy_failed": legacy_failed,
        "required_fix_count": required_fix_count(len(legacy_failed), debt_ratio),
        "created_at": datetime.now(timezone.utc).isoformat(),
    }
    save_report(root / REPORT_REL, report)

    if staged_failed:
        print("HARD FAIL: staged-related unit tests must all pass:", file=sys.stderr)
        for p in staged_failed:
            print(f"  {p}", file=sys.stderr)
        return 1

    if legacy_failed:
        req = required_fix_count(len(legacy_failed), debt_ratio)
        report["required_fix_count"] = req
        report["failed"] = legacy_failed
        save_report(root / REPORT_REL, report)
        _print_quota_help(legacy_failed, req)
        return 1

    print("Random unit tests passed.")
    (root / REPORT_REL).unlink(missing_ok=True)
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--config", type=Path, default=DEFAULT_CONFIG)
    parser.add_argument("--discover-only", action="store_true")
    parser.add_argument("--seed", type=int, default=None)
    parser.add_argument(
        "--skip-quota-resume",
        action="store_true",
        help="Do not resume/clear a previous blocked report",
    )
    args = parser.parse_args(argv)

    try:
        root = monorepo_root()
        cfg = load_config(args.config if args.config.is_absolute() else root / args.config)
    except (OSError, ValueError, FileNotFoundError) as e:
        print(f"ERROR: {e}", file=sys.stderr)
        return 2

    if args.discover_only:
        pool = discover_unit_tests(root, cfg)
        print(f"pool size: {len(pool)}")
        for p in pool[:50]:
            print(p)
        if len(pool) > 50:
            print(f"... and {len(pool) - 50} more")
        return 0

    report_path = root / REPORT_REL
    timeout = int(cfg.get("timeout_sec") or 120)
    if not args.skip_quota_resume:
        if not try_clear_blocked_report(root, report_path, cfg, timeout):
            return 1

    return run_sample(root, cfg, seed=args.seed, staged=staged_files(root))


if __name__ == "__main__":
    sys.exit(main())

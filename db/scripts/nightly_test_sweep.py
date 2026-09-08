#!/usr/bin/env python3
# ============================================================================
# nightly_test_sweep.py — 夜间随机单测巡检与自愈编排器 (架构 v65 nightly-test-sweep)
# ============================================================================
# 每晚 00:00-08:00 窗口（cron 00:20 由 nightly-test-sweep.sh 触发）：
#   1. 枚举 .gitmodules 全部子仓（SSOT），跳过脏工作区 / 无 remote / 无单测 / 排除仓
#   2. 用共享抽测库 scripts/lib/random_test_runner.sh（SSOT）随机抽测单测（默认 30%）
#   3. 失败 → claude-agent 无头修复（重跑复现 → 修复 → 复跑验证 → commit → push）
#   4. 未解失败 → 追加 .learnings/UNIT_TEST_DEBT.md 债台账（7 天去重）
#   5. 每日报告 logs/nightly-test-sweep-report-<date>.md
#
# 用法:
#   python3 scripts/nightly_test_sweep.py [--repo X] [--ratio 30] [--no-fix]
#       [--exclude a,b] [--dry-run] [--simulate-hour H] [--log-dir logs]
# 设计: docs/superpowers/specs/2026-08-05-nightly-test-sweep-design.md
# ============================================================================
from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
LIB = ROOT / "scripts" / "lib" / "random_test_runner.sh"
DEBT_LEDGER = ROOT / ".learnings" / "UNIT_TEST_DEBT.md"

WINDOW_START_HOUR = 0        # 00:00
WINDOW_END_HOUR = 8          # 08:00（hour < 8 视为窗口内）
GLOBAL_DEADLINE = "07:30"    # 到点不再接新仓
REPO_TEST_BUDGET_SEC = 15 * 60    # 每仓抽测预算 15 min
SINGLE_RUN_TIMEOUT_SEC = 8 * 60   # 单个测例单元超时 8 min
FIX_TIMEOUT_SEC = 25 * 60         # claude-agent 修复限时 25 min

DEFAULT_EXCLUDE = {
    "DaydaymoneyGrafana", "AiMonitor", "docs", "db", "valueStream", "taskChromePlugin",
    "trae-agent", "claude-agent", "dataMigrate", "e2e-tests", "playwright",
}

SECRET_RE = re.compile(
    r"(?i)(password|passwd|secret|token|apikey|api_key|access_key|private_key)"
    r"\s*[=:]\s*[^\s,;\"']{6,}"
)


def redact(text: str) -> str:
    return SECRET_RE.sub(r"\1=***", text)


def now() -> dt.datetime:
    return dt.datetime.now()


def run(cmd, cwd=None, timeout=None, input_text=None):
    return subprocess.run(
        cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout, input=input_text,
    )


def submodule_paths() -> list[str]:
    out = subprocess.check_output(
        ["git", "config", "-f", str(ROOT / ".gitmodules"), "--get-regexp", r"^submodule\..*\.path$"],
        text=True, stderr=subprocess.DEVNULL,
    )
    paths = []
    for line in out.splitlines():
        parts = line.split(None, 1)
        if len(parts) == 2:
            paths.append(parts[1].strip())
    return sorted(set(paths))


def collect_hook_versions(root: Path) -> list[tuple[str, str, str]]:
    """v13 (git-hooks-version-control): 收集各仓 .githooks/HOOK_VERSION 版本戳 + hooksPath 激活态。"""
    rows: list[tuple[str, str, str]] = []
    for p in submodule_paths():
        repo = root / p
        vf = repo / ".githooks" / "HOOK_VERSION"
        if not vf.is_file():
            rows.append((p, "—", "⚠ missing .githooks/HOOK_VERSION"))
            continue
        hp = run(["git", "-C", str(repo), "config", "core.hooksPath"], timeout=10)
        active = "✅" if hp.returncode == 0 and hp.stdout.strip() == ".githooks" else "⚠"
        vers = [ln.strip() for ln in vf.read_text(errors="ignore").splitlines()
                if ln.strip() and not ln.startswith("#") and "=" in ln.strip()]
        rows.append((p, "; ".join(vers) if vers else "—", active))
    return rows


def lib(args: list[str], cwd: Path | None = None, timeout: int | None = None, input_text: str | None = None):
    return run(["bash", str(LIB)] + args, cwd=cwd, timeout=timeout, input_text=input_text)


def rel_of(repo: Path) -> str:
    try:
        return str(repo.relative_to(ROOT))
    except ValueError:
        return str(repo)


def repo_type(repo: Path) -> str:
    r = lib(["type", rel_of(repo)], timeout=60)
    return r.stdout.strip() if r.returncode == 0 else "none"


def repo_kinds(rtype: str) -> list[str]:
    """复合类型 → ['go','js','py'] 子集"""
    return [k for k in ("go", "js", "py") if k in rtype.split("_")]


def collect(repo: Path, kind: str) -> list[str]:
    rel = rel_of(repo)
    r = lib([f"collect-{kind}", rel], timeout=120)
    return [ln for ln in r.stdout.splitlines() if ln.strip()]


def pick(targets: list[str], ratio: int) -> list[str]:
    r = lib(["pick", str(ratio)], timeout=30, input_text="\n".join(targets) + "\n")
    return [ln for ln in r.stdout.splitlines() if ln.strip()]


def run_test(repo: Path, kind: str, target: str) -> tuple[bool, str]:
    rel = rel_of(repo)
    r = lib([f"run-{kind}", rel, target], timeout=SINGLE_RUN_TIMEOUT_SEC)
    return r.returncode == 0, redact(r.stdout + r.stderr)


# 仅未跟踪、且文件名为下列之一时视为「许可/说明噪音」，不阻断抽测。
# 任意已跟踪改动、或其他未跟踪路径 → 仍记 dirty。
IGNORABLE_UNTRACKED_NAMES = frozenset({
    "LICENSE",
    "README.md",
    "README copy.md",
})


def _porcelain_path(line: str) -> str:
    """Extract path from a git status --porcelain line (handles quoted paths)."""
    raw = line[3:] if len(line) >= 3 else line
    if len(raw) >= 2 and raw[0] == '"' and raw[-1] == '"':
        # git quotes with C-style escapes when needed
        return bytes(raw[1:-1], "utf-8").decode("unicode_escape")
    return raw


def _untracked_tree_only_ignorable(repo: Path, rel: str) -> bool:
    """True if untracked path is a file with ignorable name, or a dir whose
    files are all ignorable names (git often reports ``?? vendor/`` for new trees)."""
    rel = rel.rstrip("/")
    target = repo / rel
    if target.is_file() or (not target.exists() and Path(rel).name in IGNORABLE_UNTRACKED_NAMES):
        return Path(rel).name in IGNORABLE_UNTRACKED_NAMES
    if not target.is_dir():
        return Path(rel).name in IGNORABLE_UNTRACKED_NAMES
    saw_file = False
    for p in target.rglob("*"):
        if p.is_file():
            saw_file = True
            if p.name not in IGNORABLE_UNTRACKED_NAMES:
                return False
    return saw_file  # empty dir alone still counts as noise-free enough to ignore


def _is_ignorable_untracked(repo: Path, line: str) -> bool:
    if not line.startswith("??"):
        return False
    return _untracked_tree_only_ignorable(repo, _porcelain_path(line))


def repo_clean(repo: Path) -> tuple[bool, str]:
    r = run(["git", "-C", str(repo), "status", "--porcelain"], timeout=30)
    if r.returncode != 0:
        return False, "git status failed"
    lines = [ln for ln in r.stdout.splitlines() if ln.strip()]
    if not lines:
        return True, ""
    real = [ln for ln in lines if not _is_ignorable_untracked(repo, ln)]
    if not real:
        return True, ""
    samples = [_porcelain_path(ln) for ln in real[:3]]
    note = f"dirty worktree ({len(real)} paths; e.g. {', '.join(samples)})"
    return False, note


def repo_has_remote(repo: Path) -> bool:
    r = run(["git", "-C", str(repo), "remote", "get-url", "origin"], timeout=30)
    return r.returncode == 0 and bool(r.stdout.strip())


def session_hub_register_sweep() -> str:
    """v66: 注册巡检会话（kind=sweep）。失败返回空串（不阻塞巡检）。"""
    bin_agent = ROOT / "claude-agent" / "bin" / "claude-agent"
    if not bin_agent.is_file():
        return ""
    try:
        r = run([str(bin_agent), "session", "register", "--kind", "sweep", "--pid", str(os.getpid()),
                 "--note", "nightly test sweep (00:00-08:00)"], timeout=15)
        return r.stdout.strip() if r.returncode == 0 else ""
    except Exception:
        return ""


def session_hub_unregister(sid: str) -> None:
    if not sid:
        return
    bin_agent = ROOT / "claude-agent" / "bin" / "claude-agent"
    if not bin_agent.is_file():
        return
    try:
        run([str(bin_agent), "session", "release", "--all", "--sid", sid], timeout=15)
    except Exception:
        pass


def session_hub_active_repos() -> dict[str, str]:
    """v66: 查询 Session Hub 活跃会话，返回 {repo: 会话摘要}。
    活跃会话 = pid 存活且（持有该仓锁 或 工作目录位于该仓下）。
    二进制/会话机制不可用时返回空 dict（不阻塞巡检）。
    """
    bin_agent = ROOT / "claude-agent" / "bin" / "claude-agent"
    if not bin_agent.is_file():
        return {}
    try:
        r = run([str(bin_agent), "session", "list", "--active", "--json"], timeout=15)
    except Exception:
        return {}
    if r.returncode != 0 or not r.stdout.strip():
        return {}
    try:
        sessions = json.loads(r.stdout)
    except json.JSONDecodeError:
        return {}
    active: dict[str, str] = {}
    root = str(ROOT)
    for s in sessions:
        note = s.get("note", "") or s.get("kind", "")
        for lock in s.get("locks") or []:
            active.setdefault(lock, f"{s.get('session_id','?')} kind={s.get('kind','?')} {note}")
        cwd = s.get("start_cwd", "")
        if cwd.startswith(root + "/"):
            rel = cwd[len(root) + 1:].split("/", 1)[0]
            if rel:
                active.setdefault(rel, f"{s.get('session_id','?')} kind={s.get('kind','?')} {note}")
    return active


def in_window(hour: int) -> bool:
    return WINDOW_START_HOUR <= hour < WINDOW_END_HOUR


def past_global_deadline(clock: dt.datetime) -> bool:
    return clock.strftime("%H:%M") >= GLOBAL_DEADLINE


def fix_with_agent(repo: Path, rtype: str, targets: list[str], fail_log: Path) -> tuple[bool, str]:
    """调用 claude-agent 无头修复。返回 (ok, detail)。"""
    task = (
        "「夜间单测巡检」修复任务 — 仓库: %s (类型: %s)\n"
        "失败明细见: %s\n"
        "抽测目标: %s\n"
        "1. 重跑失败测例复现失败\n"
        "2. 修复：允许修改产品代码（若测例揭示真实回归）；测试断言过时则只改测试\n"
        "   （遵守规则 41：fix 提交必须携带与被修源码对应的回归单测）\n"
        "3. 复跑该测例 + 相关测例直到通过（禁止 --no-verify，遵守规则 6/28）\n"
        "4. git commit（message 前缀 fix:/test: 如实反映）\n"
        "5. git push 到当前分支 upstream\n"
        "完成后输出一行 JSON：{\"ok\": true, \"commit\": \"<hash>\"} 或 "
        "{\"ok\": false, \"reason\": \"<原因>\"}"
    ) % (repo.name, rtype, fail_log, ", ".join(targets))
    import shutil
    import time
    # 优先 Go 版 claude-agent 二进制（--config-file 显式指定配置；run.sh 仅做代理清理/构建，
    # 而 claude CLI 需要网络访问，故直接用二进制保留调用方代理环境）
    bin_agent = ROOT / "claude-agent" / "bin" / "claude-agent"
    # 配置优先真实文件（gitignored，本地生成），回退示例（入库，model 可能过期但可运行）
    cfg = ROOT / "claude-agent" / "claude_config.yaml"
    if not cfg.is_file():
        cfg = ROOT / "claude-agent" / "claude_config.yaml.example"
    # trajectory 记录到 logs/trajectories/（避免在仓库内生成 .trajectories/ 弄脏工作区）
    traj_dir = ROOT / "logs" / "trajectories"
    traj_dir.mkdir(parents=True, exist_ok=True)
    traj_file = traj_dir / f"{repo.name.replace('/', '_')}-{int(time.time())}.jsonl"
    if bin_agent.is_file() and cfg.is_file():
        agent_cmd = [str(bin_agent), "run", "--config-file", str(cfg),
                     "--trajectory-file", str(traj_file), task]
    elif shutil.which("claude-agent") is not None:
        agent_cmd = ["claude-agent", "run", task]
    else:
        return False, "claude-agent not found (bin nor PATH)"
    try:
        r = run(
            agent_cmd,
            cwd=str(repo), timeout=FIX_TIMEOUT_SEC,
        )
    except subprocess.TimeoutExpired:
        return False, "fix timeout (25min)"
    except OSError as e:
        return False, f"agent invocation failed: {e}"
    out = redact(r.stdout + r.stderr)
    # 解析最后一行 JSON
    for line in reversed(out.strip().splitlines()):
        line = line.strip()
        if line.startswith("{") and line.endswith("}"):
            try:
                data = json.loads(line)
            except json.JSONDecodeError:
                continue
            if data.get("ok"):
                return True, f"commit={data.get('commit', '?')}"
            return False, f"agent: {data.get('reason', 'no reason')}"
    return False, f"agent exit={r.returncode}, no JSON result (tail: {out[-200:]})"


def ledger_append(repo_name: str, paths: list[str], summary: str, reason: str) -> bool:
    """追加债台账条目。7 天去重：同仓库同路径在 7 天内已有登记 → 跳过。"""
    import datetime
    now_utc = dt.datetime.now(dt.timezone.utc)
    content = DEBT_LEDGER.read_text() if DEBT_LEDGER.exists() else ""
    # 解析既有章节: ## <ISO时间戳> … 条目行（- 仓库: X / - 抽测失败文件（未修复）: …）
    for section in re.split(r"\n## ", content):
        m = re.match(r"(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z)", section)
        if not m:
            continue
        try:
            ts = datetime.datetime.fromisoformat(m.group(1).replace("Z", "+00:00"))
        except ValueError:
            continue
        if (now_utc - ts).days > 7:
            continue
        if repo_name in section and any(p in section for p in paths):
            return False
    section = (
        f"\n## {now_utc.strftime('%Y-%m-%dT%H:%M:%SZ')} — nightly sweep\n\n"
        f"- 仓库: {repo_name}\n"
        f"- 抽测失败文件（未修复）: {' '.join(paths)}\n"
        f"- 失败摘要: {summary}\n"
        f"- 修复尝试: {reason}\n"
    )
    with DEBT_LEDGER.open("a") as f:
        f.write(section)
    return True


def main() -> int:
    ap = argparse.ArgumentParser(description="Nightly random unit-test sweep + auto-fix (v65)")
    ap.add_argument("--repo", action="append", default=[], help="只巡检指定子仓名或路径（可多次）")
    ap.add_argument("--ratio", type=int, default=30, help="随机抽测比例 %% (默认 30)")
    ap.add_argument("--no-fix", action="store_true", help="发现失败不调用 claude-agent 修复")
    ap.add_argument("--exclude", help="逗号分隔排除仓（覆盖默认排除表）")
    ap.add_argument("--dry-run", action="store_true", help="只列出将巡检的仓与抽测池规模")
    ap.add_argument("--simulate-hour", type=int, help="模拟当前小时（窗口/截止测试用）")
    ap.add_argument("--log-dir", default=str(ROOT / "logs"))
    ap.add_argument("--max-rounds", type=int, default=3, help="窗口内最大巡检轮数 (默认 3)")
    ap.add_argument("--round-cutoff", default="07:00", help="不在此时间后开始新轮 (默认 07:00)")
    args = ap.parse_args()
    if args.dry_run:
        args.max_rounds = 1

    clock = now()
    hour = args.simulate_hour if args.simulate_hour is not None else clock.hour
    if not in_window(hour):
        print(f"[sweep] outside window (hour={hour}), exit.", file=sys.stderr)
        return 0

    log_dir = Path(args.log_dir)
    log_dir.mkdir(parents=True, exist_ok=True)
    date_str = clock.strftime("%Y%m%d")
    detail_log = log_dir / f"nightly-test-sweep-{date_str}.log"
    report_file = log_dir / f"nightly-test-sweep-report-{date_str}.md"
    fail_root = ROOT / "tmp" / "nightly"
    fail_root.mkdir(parents=True, exist_ok=True)

    excludes = set(DEFAULT_EXCLUDE)
    if args.exclude:
        excludes = set(x.strip() for x in args.exclude.split(",") if x.strip())

    # 枚举仓库（--repo 可多次；值为路径或子仓名）
    if args.repo:
        repos = []
        for r in args.repo:
            cand = Path(r)
            if not cand.is_absolute():
                cand = ROOT / cand
            if cand.is_dir():
                repos.append((r, cand))
    else:
        repos = [(p, ROOT / p) for p in submodule_paths() if p not in excludes]

    # 预算与状态（多轮循环：每轮重新随机抽样；跑满 max_rounds 或到 round_cutoff 停止）
    deadline_reached = False
    ledger_written = False
    started_at = clock.strftime("%Y-%m-%d %H:%M:%S")

    # v66: 注册巡检会话（sweep）— 使交互/无头会话互知
    sweep_sid = session_hub_register_sweep()

    all_results: list[dict] = []
    rounds_run = 0
    while rounds_run < args.max_rounds:
        if (now() if args.simulate_hour is None else clock).strftime("%H:%M") >= args.round_cutoff:
            print(f"[sweep] round {rounds_run + 1}: past round cutoff ({args.round_cutoff}), stop.")
            break
        rounds_run += 1
        deadline_reached = False
        results: list[dict] = []
        print(f"[sweep] === Round {rounds_run}/{args.max_rounds} ===")
        detail_log.open("a").write(f"=== Round {rounds_run}/{args.max_rounds} ===\n")
        for name, repo in repos:
            if deadline_reached:
                results.append({"repo": name, "result": "SKIP", "note": "global deadline 07:30"})
                continue
            if past_global_deadline(now() if args.simulate_hour is None else clock.replace(hour=hour)):
                deadline_reached = True
                results.append({"repo": name, "result": "SKIP", "note": "global deadline 07:30"})
                continue
            if not repo.is_dir():
                results.append({"repo": name, "result": "SKIP", "note": "dir missing"})
                continue
            if not (repo / ".git").exists():
                results.append({"repo": name, "result": "SKIP", "note": "not a git repo"})
                continue
            if not repo_has_remote(repo):
                results.append({"repo": name, "result": "SKIP", "note": "no origin remote"})
                continue
            clean, why = repo_clean(repo)
            if not clean:
                results.append({"repo": name, "result": "SKIP", "note": why})
                continue
            # v66: Session Hub 互知 — 该仓有活跃会话时跳过（避免与交互/无头会话冲突）
            active = session_hub_active_repos()
            if name in active:
                results.append({"repo": name, "result": "SKIP", "note": f"active session in repo ({active[name]})"})
                continue

            rtype = repo_type(repo)
            kinds = repo_kinds(rtype)
            # 每种类型独立收集 + 抽测（复合仓每类保证至少 1 个）
            picked: list[tuple[str, str]] = []
            pool_n = 0
            for k in kinds:
                targets = collect(repo, k)
                pool_n += len(targets)
                picked.extend((k, t) for t in pick(targets, args.ratio))
            if rtype == "none" or not picked:
                results.append({"repo": name, "result": "SKIP", "note": f"no unit tests (type={rtype})"})
                continue
            line = f"[{name}] type={rtype} pool={pool_n} picked={len(picked)}"
            print(line)
            detail_log.open("a").write(line + "\n")
            if args.dry_run:
                results.append({"repo": name, "result": "DRY", "note": f"type={rtype} pool={pool_n} picked={[t for _, t in picked]}"})
                continue

            # 执行抽测
            failures: list[str] = []
            outputs: list[str] = []
            for k, t in picked:
                ok, out = run_test(repo, k, t)
                detail_log.open("a").write(f"  [{k}] {t}: {'PASS' if ok else 'FAIL'}\n")
                if ok:
                    outputs.append(f"  ✅ [{k}] {t}\n{out}")
                else:
                    failures.append(t)
                    outputs.append(f"  ❌ [{k}] {t}\n{out}")

            if not failures:
                results.append({"repo": name, "result": "PASS", "note": f"picked={len(picked)}"})
                continue

            # 失败 → 落盘 fail.log
            fail_log = fail_root / f"{name.replace('/', '_')}-{clock.strftime('%H%M%S')}.log"
            fail_log.write_text(
                f"# {name} nightly sweep failures ({clock})\n"
                f"type={rtype} targets={' '.join(t for _, t in picked)}\n\n" + "\n".join(outputs)
            )

            if args.no_fix:
                results.append({"repo": name, "result": "UNFIXED", "note": f"failures={failures} (--no-fix)", "fail_log": str(fail_log)})
                continue

            # 修复闭环（每仓 1 次尝试）
            ok_fix, detail = fix_with_agent(repo, rtype, failures, fail_log)
            detail_log.open("a").write(f"  fix: ok={ok_fix} {detail}\n")
            if ok_fix:
                # 复跑验证
                still_fail = []
                for k, t in picked:
                    ok, out = run_test(repo, k, t)
                    if not ok:
                        still_fail.append(t)
                if still_fail:
                    results.append({"repo": name, "result": "UNFIXED", "note": f"fix claimed ok but still failing: {still_fail}", "fail_log": str(fail_log)})
                    continue
                # push 兜底：agent 漏推时由巡检补推（本地提交保留，报告标注）
                r = run(["git", "-C", str(repo), "rev-list", "@{u}..HEAD", "--count"], timeout=30)
                unpushed = r.stdout.strip() if r.returncode == 0 else "?"
                if unpushed not in ("0", ""):
                    p = run(["git", "-C", str(repo), "push"], timeout=120)
                    detail += f" | sweep-push={'ok' if p.returncode == 0 else 'FAILED'}"
                results.append({"repo": name, "result": "FIXED", "note": detail, "fail_log": str(fail_log)})
            else:
                summary = outputs[0][:200] if outputs else "unknown"
                ledger_written = ledger_append(name, failures, summary, detail) or ledger_written
                results.append({"repo": name, "result": "UNFIXED", "note": f"fix failed: {detail}", "fail_log": str(fail_log)})
        # ── 轮次收尾 ──
        for r in results:
            r["round"] = rounds_run
        all_results.extend(results)
        print(f"[sweep] round {rounds_run} done: " + " ".join(f"{k}={v}" for k, v in
              __import__("collections").Counter(r["result"] for r in results).items()))

    if rounds_run == 0:
        print("[sweep] no rounds ran (cutoff); keep existing report.")
        session_hub_unregister(sweep_sid)
        return 0
    # 报告（多轮：按轮分区 + 汇总）
    counts = {}
    for r in all_results:
        counts[r["result"]] = counts.get(r["result"], 0) + 1
    finished_at = dt.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    md = [
        f"# Nightly Test Sweep Report — {date_str}",
        "",
        f"- window: 00:00-08:00 | started: {started_at} | finished: {finished_at} | ratio: {args.ratio}% | rounds: {rounds_run}/{args.max_rounds} (cutoff {args.round_cutoff})",
        f"- summary: " + " | ".join(f"{k}={v}" for k, v in sorted(counts.items())),
        "",
        "| round | repo | result | note |",
        "|-------|------|--------|------|",
    ]
    for r in all_results:
        md.append(f"| R{r.get('round', '?')} | {r['repo']} | {r['result']} | {r.get('note', '')} |")
    md.append("")
    if any(r["result"] == "UNFIXED" for r in all_results):
        md.append("### 未解失败" + ("（已记入 .learnings/UNIT_TEST_DEBT.md）" if ledger_written else "（--no-fix 未记台账）"))
        for r in all_results:
            if r["result"] == "UNFIXED":
                md.append(f"- R{r.get('round', '?')} {r['repo']}: {r.get('note', '')} (fail log: {r.get('fail_log', '-')})")
        md.append("")

    # Git Hooks 版本统计（git-hooks-version-control v13）
    md.append("### Git Hooks 版本（v13）")
    md.append("")
    md.append("| repo | 钩子版本 (HOOK_VERSION) | hooksPath |")
    md.append("|------|------------------------|-----------|")
    for repo_name, ver, active in collect_hook_versions(ROOT):
        md.append(f"| {repo_name} | {ver} | {active} |")
    md.append("")
    report_file.write_text("\n".join(md))
    print(f"[sweep] report: {report_file}")

    # v66: 巡检会话注销（进程死亡时注册表会自行回收，此处显式清理）
    session_hub_unregister(sweep_sid)
    if any(r.get("result") == "UNFIXED" for r in all_results):
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

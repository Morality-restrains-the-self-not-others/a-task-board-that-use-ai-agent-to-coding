#!/usr/bin/env python3
"""禁止业务服务进程内 ticker/轮询循环（元规则 46 / ADR-0011）。

业务 HTTP/RPC 进程不得用 time.NewTicker / time.Tick 自唤醒做周期工作。
周期工作须落在专门的定时服务（taskEvents timer worker），或由 HTTP / Kafka /
Webhook 等外界触发。

用法:
  python3 db/scripts/ci/check_no_service_internal_poll_loop.py
  python3 db/scripts/ci/check_no_service_internal_poll_loop.py --list-legacy
"""
from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]

TICKER_RE = re.compile(r"\btime\.(?:NewTicker|Tick)\s*\(")

# 存量业务进程内 ticker，禁止再新增；迁移到 taskEvents timer 后从本表删除。
# OPT-20260816-025：taskCloudService 空闲回收 ticker 已迁到 taskEvents
# workspace_machine_idle timer（18044），条目已删除。
LEGACY_INTERNAL_TICKERS: frozenset[str] = frozenset()

SKIP_DIR_PARTS = frozenset(
    {
        "third_party",
        "vendor",
        "node_modules",
        "gitlab-ce",
        "gitlab_home",
        ".git",
    }
)

# 专门定时服务、协议恢复循环、编排器：允许 ticker。
SKIP_PREFIXES = (
    "runAll/",
    "gitService/",
    "AiMonitor/",
    "scripts/",
    "sdk/",
    "trae-agent/",
    "taskEvents/internal/handlers/",
    "taskEvents/broker/",
)

SERVICE_TOP_EXACT = frozenset({"go_relayToTrae", "go_run_container"})


@dataclass(frozen=True)
class Hit:
    rel: str
    kind: str  # violation | legacy
    line: int


def _is_service_rel(rel: str) -> bool:
    top = rel.split("/", 1)[0]
    # git worktrees 本地目录名为 `{service}-wt/`（已 gitignore），不得当业务仓扫描。
    if top.endswith("-wt"):
        return False
    if top in SERVICE_TOP_EXACT:
        return True
    return top.startswith("task")


def _skipped_path(rel: str, parts: tuple[str, ...]) -> bool:
    if rel.endswith("_test.go"):
        return True
    if any(p in SKIP_DIR_PARTS for p in parts):
        return True
    return any(rel.startswith(prefix) for prefix in SKIP_PREFIXES)


def first_ticker_line(text: str) -> int:
    for i, line in enumerate(text.splitlines(), 1):
        if TICKER_RE.search(line):
            return i
    return 0


def classify(rel: str, text: str) -> tuple[str, int]:
    """Return (kind, line). kind is ok | legacy | violation."""
    posix = rel.replace("\\", "/")
    parts = tuple(Path(posix).parts)
    if _skipped_path(posix, parts):
        return "ok", 0
    if not _is_service_rel(posix):
        return "ok", 0
    line = first_ticker_line(text)
    if line <= 0:
        return "ok", 0
    if posix in LEGACY_INTERNAL_TICKERS:
        return "legacy", line
    return "violation", line


def iter_go_files(root: Path):
    for path in sorted(root.rglob("*.go")):
        try:
            rel = path.relative_to(root).as_posix()
        except ValueError:
            continue
        yield path, rel, path.parts


def scan(root: Path | None = None) -> list[Hit]:
    base = root if root is not None else ROOT
    hits: list[Hit] = []
    for path, rel, parts in iter_go_files(base):
        if _skipped_path(rel, tuple(Path(rel).parts)):
            continue
        if not _is_service_rel(rel):
            continue
        text = path.read_text(encoding="utf-8", errors="replace")
        kind, line = classify(rel, text)
        if kind in ("violation", "legacy"):
            hits.append(Hit(rel=rel, kind=kind, line=line))
    return hits


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "--list-legacy",
        action="store_true",
        help="只打印存量 allowlist 命中，退出码仍按 violation 计算",
    )
    args = ap.parse_args(argv)

    hits = scan()
    violations = [h for h in hits if h.kind == "violation"]
    legacies = [h for h in hits if h.kind == "legacy"]

    if args.list_legacy or legacies:
        for h in legacies:
            print(f"LEGACY {h.rel}:{h.line} time.NewTicker/Tick（须迁到 taskEvents timer）")
        print(f"legacy-internal-ticker: {len(legacies)} file(s)")

    for h in violations:
        print(
            f"ERROR {h.rel}:{h.line} 业务服务禁止进程内 ticker/轮询；"
            "改为 taskEvents 定时 worker 或 HTTP/Kafka/Webhook 触发"
            "（元规则 46 / ADR-0011）"
        )

    if violations:
        print(f"no-service-internal-poll-loop: {len(violations)} violation(s)")
        return 1

    print(
        f"ok: no new business-service ticker "
        f"({len(legacies)} legacy allowlisted)"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())

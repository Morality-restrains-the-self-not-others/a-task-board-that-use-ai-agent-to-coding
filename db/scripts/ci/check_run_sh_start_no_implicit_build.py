#!/usr/bin/env python3
"""Fail if a run.sh start branch implicitly compiles.

ADR-0027: 「全部重启」空窗的根因是 start 隐式编脏树——run.sh 的 start 分支
直接 `go build` / `./build.sh` 会在每次拉起时从工作树编译，restart-all 期间
上游服务还在构建、下游拿不到 last-good 二进制。

本门禁：
  - 解析 conf/runAll.yaml，对每个 start_command 指向 `run.sh start` 的服务，
    解析 run.sh 的 start) 分支，断言不含编译调用；
  - 同时扫描仓库顶层服务子仓的 `**/run.sh`（上限防大爆炸）；
  - 编译只允许出现在 build)/dev) 分支或显式 build 函数（如 build_intent）；
  - 白名单：runAll/run.sh 编排器自举（`RUNALL_SKIP_BUILD` 守卫后的 ./build.sh）。

用法：python3 db/scripts/ci/check_run_sh_start_no_implicit_build.py
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

try:
    import yaml
except ImportError:  # pragma: no cover
    print("SKIP: PyYAML not installed", file=sys.stderr)
    raise SystemExit(0)

ROOT = Path(__file__).resolve().parents[3]

# 编排器自举白名单：runAll/run.sh 顶层用 RUNALL_SKIP_BUILD 守卫后 ./build.sh。
WHITELIST = {ROOT / "runAll" / "run.sh"}

# 直接编译标记（命令行/子脚本）
_COMPILE_DIRECT_RE = re.compile(
    r"(^|\s|\|)(go\s+build|go\s+install|go\s+vet\b|make\s+build|\./build\.sh|build\.sh)(\s|$)"
)
# 调用 *build* 编译函数（build_intent / build_all / build_event ...）：只匹配
# 命令位置（行首或 ;/&&/||/| 后），避免把 echo 文案里的 `run.sh build` 误报。
# 前缀 `[a-zA-Z_0-9]*` 允许空（build_intent 自身）或短前缀（rebuild_all 等）。
_CALL_BUILD_FN_RE = re.compile(
    r"(?:^|[;&|]\s*)[a-zA-Z_0-9]*build[a-zA-Z0-9_]*\b"
)

# case "$cmd" in 块
_CASE_RE = re.compile(r'case\s+"?\$[A-Za-z_][A-Za-z0-9_]*"?\s+in')


def extract_start_branch(src: str) -> str:
    """返回 run.sh 中 start) 分支的正文（不含外层 case 头）。找不到返回 ''。"""
    case = _CASE_RE.search(src)
    if not case:
        return ""
    start_idx = src.find("start)", case.end())
    if start_idx < 0:
        return ""
    # 分支结束于该 start) 后的第一个行首 `;;`
    tail = src[start_idx:]
    term = re.search(r"\n[ \t]*;;", tail)
    if term:
        return tail[: term.start()]
    # 兜底：到下一个平级 case 分支或文件尾
    nxt = re.search(r"\n[ \t]*[a-z_][a-z0-9_]*\)", tail[len("start)") :])
    if nxt:
        return tail[: len("start)") + nxt.start()]
    return tail


def branch_compiles(branch: str) -> bool:
    """start 分支是否直接/经 build 函数隐式编译。"""
    for line in branch.splitlines():
        stripped = line.lstrip()
        if stripped.startswith("#"):
            continue
        # echo/printf 只是给运维的提示文案，不当作编译调用
        if re.match(r"(?:echo|printf)\b", stripped):
            continue
        # 变量赋值（BUILD_DIR=...）不是编译调用
        if re.match(r"[A-Za-z_][A-Za-z0-9_]*=", stripped):
            continue
        if _COMPILE_DIRECT_RE.search(stripped):
            return True
        if _CALL_BUILD_FN_RE.search(stripped):
            return True
    return False


def run_sh_paths_from_runall(cfg: dict) -> list[tuple[str, Path]]:
    """从 conf/runAll.yaml 解析每个 start_command 指向的 run.sh。"""
    out: list[tuple[str, Path]] = []
    if not isinstance(cfg, dict):
        return out
    for group in cfg.get("groups") or []:
        for svc in group.get("services") or []:
            cmd = str(svc.get("start_command") or "").strip()
            if "run.sh" not in cmd:
                continue
            working_dir = str(svc.get("working_dir") or ".").strip()
            base = (ROOT / working_dir).resolve() if working_dir else ROOT
            # start_command 形如 `bash <path>/run.sh start ...`
            m = re.search(r"(?:^|\s)([^\s]*run\.sh)(\s|$)", cmd)
            if not m:
                continue
            rel = m.group(1)
            p = (base / rel).resolve() if not rel.startswith("/") else Path(rel)
            out.append((str(svc.get("name") or ""), p))
    return out


def scan_one(rel: Path, name: str) -> list[str]:
    if rel in WHITELIST:
        return []
    if not rel.is_file():
        return []
    try:
        src = rel.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return []
    branch = extract_start_branch(src)
    if branch and branch_compiles(branch):
        return [f"{rel}: start) 分支隐式编译（ADR-0027；编译只允许在 build)/dev) 或显式 build 函数）"]
    return []


def main() -> int:
    violations: list[str] = []

    runall = ROOT / "conf" / "runAll.yaml"
    if runall.is_file():
        cfg = yaml.safe_load(runall.read_text(encoding="utf-8")) or {}
        for name, p in run_sh_paths_from_runall(cfg):
            violations.extend(scan_one(p, name))

    # 顶层服务子仓的 run.sh（白名单内的 runAll/run.sh 跳过）
    for p in sorted(ROOT.rglob("run.sh")):
        if p.resolve() in WHITELIST:
            continue
        if ".git" in p.parts or "node_modules" in p.parts:
            continue
        violations.extend(scan_one(p, ""))

    if violations:
        print(
            f"FAIL: run.sh start 分支不得隐式编译（{len(violations)} hit(s)；ADR-0027）:",
            file=sys.stderr,
        )
        for v in violations[:50]:
            print(f"  {v}", file=sys.stderr)
        if len(violations) > 50:
            print(f"  ... +{len(violations) - 50} more", file=sys.stderr)
        return 1

    print("OK: run.sh start 分支均不隐式编译（ADR-0027）")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

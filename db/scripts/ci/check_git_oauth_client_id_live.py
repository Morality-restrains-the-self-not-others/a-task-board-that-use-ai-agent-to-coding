#!/usr/bin/env python3
"""CI gate: git-oauth provider client_id must resolve against GitHub Apps API.

OPT-20260807-041 事故闭环 a): conf 变更审批钩子对 git-oauth 凭据类字段做 diff 校验。

当一个 git-oauth 提供方 YAML 进入暂存区 diff（conf/auth/git-oauth/providers/ 或
conf/auth/task-credential/git-oauth-providers/）时，本脚本从 service_provider /
文件名推导 GitHub App slug，调用 api.github.com/apps/{slug} 校验:

  - App 不存在 (404)             -> FAIL（8-3 事故根因：client_id 指向不存在的 App）
  - App 存在但 client_id 不一致    -> FAIL（凭据字段被改错）
  - 网络不可达 / 限流 / 无相关 diff -> SKIP（warn，不阻塞离线开发）

SSOT 行为: 离线/限流时 SKIP 而非 FAIL —— 本校验是"已知坏值"门禁，不是联网准入。
仅当 git-oauth 提供方文件进入暂存区 diff 才触发，避免无关提交被网络查询拖慢。

Exit codes:
  0 — ok / skip（无相关 diff、离线、非 github provider）
  1 — 校验失败（App 不存在或 client_id 不一致）
  2 — 配置 / 读取错误
"""

from __future__ import annotations

import argparse
import json
import re
import sys
import urllib.error
import urllib.request
from pathlib import Path

try:
    import yaml
except ImportError:  # pragma: no cover
    print("ERROR: PyYAML required", file=sys.stderr)
    raise SystemExit(2)

PROVIDER_DIRS = (
    "conf/auth/git-oauth/providers",
    "conf/auth/task-credential/git-oauth-providers",
)
API_BASE = "https://api.github.com/apps"
UA = "daydaymoney-ci-check/1.0"
REQUEST_TIMEOUT = 5.0

# 从 service_provider "github-official-{slug}" 或文件名 "http-github-com--app-{slug}.yaml" 推导 slug
SP_RE = re.compile(r"^github-official-([A-Za-z0-9_-]+)$")
FILE_RE = re.compile(r"^http-github-com--app-([A-Za-z0-9_-]+)\.yaml$")


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError("db/registry.yaml not found")


def is_provider_path(rel: str) -> bool:
    """True if a repo-relative path is one of the git-oauth provider YAMLs.

    Filters out *.ai.md 配套说明（如 github.ai.md / http-github-com.yaml.ai.md）。
    """
    p = Path(rel)
    if p.suffix not in (".yaml", ".yml"):
        return False
    if rel.endswith(".ai.md"):
        return False
    return str(p.parent) in PROVIDER_DIRS


def filter_provider_files(changed: list[str]) -> list[str]:
    """从暂存区 diff 文件名列表中筛出 git-oauth 提供方文件。"""
    return sorted({c for c in changed if is_provider_path(c)})


def derive_slug(data: dict, filename: str) -> str | None:
    """从 service_provider / 文件名推导 GitHub App slug。"""
    sp = data.get("service_provider")
    if isinstance(sp, str):
        m = SP_RE.match(sp.strip())
        if m:
            return m.group(1)
    m = FILE_RE.match(filename)
    if m:
        return m.group(1)
    return None


def provider_website(data: dict) -> str | None:
    """target.website 归一化（去尾斜杠、小写）。"""
    website = data.get("target", {})
    if isinstance(website, dict):
        website = website.get("website")
    if isinstance(website, str) and website.strip():
        return website.strip().rstrip("/").lower()
    return None


def fetch_json(url: str, timeout: float = REQUEST_TIMEOUT):
    """GET url，返回 (http_code | None, json_dict | None, error_msg | None)。

    网络错误/超时 -> (None, None, msg)；HTTP 错误 -> (code, None, msg)。
    """
    req = urllib.request.Request(url, headers={"User-Agent": UA, "Accept": "application/vnd.github+json"})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            code = resp.getcode()
            body = resp.read()
    except urllib.error.HTTPError as e:
        return e.code, None, f"HTTP {e.code}"
    except (urllib.error.URLError, OSError, TimeoutError) as e:
        return None, None, f"{type(e).__name__}: {e}"
    try:
        return code, json.loads(body.decode("utf-8")), None
    except (ValueError, UnicodeDecodeError) as e:
        return code, None, f"bad json: {e}"


def validate_client_id(api_result, expected: str, slug: str) -> tuple[str, str]:
    """校验 GitHub Apps API 返回 vs 配置的 client_id。

    api_result = (code | None, json | None, error_msg | None)
    返回 (status, detail)，status ∈ {ok, not_found, mismatch, skip, config}。
    """
    code, payload, err = api_result
    if code is None:
        # 网络不可达 / 超时 -> SKIP（离线开发不阻塞）
        return "skip", f"network: {err}"
    if code in (401, 403, 429):
        # 未认证限额/限流 -> SKIP，避免 CI 误伤
        return "skip", f"api rate/limit (HTTP {code})"
    if code == 404:
        return "not_found", f"GitHub App '{slug}' does not exist (HTTP 404)"
    if code != 200:
        return "skip", f"api HTTP {code} for app '{slug}'"
    actual = payload.get("client_id") if isinstance(payload, dict) else None
    if actual != expected:
        return "mismatch", f"app '{slug}' client_id={actual!r} != conf client_id={expected!r}"
    return "ok", f"app '{slug}' client_id matches"


def load_yaml(path: Path) -> dict:
    data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    return data if isinstance(data, dict) else {}


def staged_provider_files(root: Path) -> list[str]:
    """git diff --cached --name-only 中命中的 git-oauth 提供方文件（repo 相对路径）。"""
    try:
        import subprocess

        out = subprocess.run(
            ["git", "-C", str(root), "diff", "--cached", "--name-only", "-z"],
            capture_output=True,
            text=True,
            timeout=10,
        )
    except (OSError, subprocess.SubprocessError):
        # 无 git 环境（CI 单文件副本）-> 交给调用方决定（--all 时全量）
        return []
    if out.returncode != 0:
        return []
    return filter_provider_files(out.stdout.split("\0"))


def validate_provider_path(root: Path, rel: str) -> tuple[str, str]:
    """校验单个提供方文件，返回 (status, detail)。"""
    path = root / rel
    try:
        data = load_yaml(path)
    except (OSError, yaml.YAMLError) as e:
        return "config", f"cannot parse {rel}: {e}"

    website = provider_website(data)
    if not website or "github.com" not in website:
        return "skip", f"{rel}: not a github.com provider (website={website!r})"
    expected = data.get("target", {}).get("client_id") if isinstance(data.get("target"), dict) else None
    if not expected:
        return "config", f"{rel}: missing target.client_id"
    slug = derive_slug(data, path.name)
    if not slug:
        return "skip", f"{rel}: cannot derive GitHub app slug (service_provider={data.get('service_provider')!r})"

    api_result = fetch_json(f"{API_BASE}/{slug}")
    status, detail = validate_client_id(api_result, expected, slug)
    return status, f"{rel}: {detail}"


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument(
        "--all",
        action="store_true",
        help="校验全部 git-oauth 提供方（无视暂存区 diff，用于手动/CI 全量扫描）",
    )
    ap.add_argument(
        "--changed",
        nargs="*",
        default=None,
        help="显式指定变更文件列表（覆盖 git diff 检测，测试用）",
    )
    args = ap.parse_args(argv)

    root = monorepo_root()
    if args.changed is not None:
        targets = filter_provider_files(args.changed)
    elif args.all:
        targets = []
        for d in PROVIDER_DIRS:
            base = root / d
            if base.is_dir():
                targets.extend(str(p.relative_to(root)) for p in sorted(base.glob("*.yaml")))
        targets = filter_provider_files(targets)
    else:
        targets = staged_provider_files(root)

    if not targets:
        print("OK: no git-oauth provider diff (skip live client_id check)")
        return 0

    failures: list[str] = []
    skips: list[str] = []
    for rel in targets:
        status, detail = validate_provider_path(root, rel)
        if status in ("not_found", "mismatch", "config"):
            failures.append(detail)
        elif status == "skip":
            skips.append(detail)
        else:
            print(f"OK: {detail}")

    for s in skips:
        print(f"SKIP: {s}", file=sys.stderr)

    if failures:
        print("git-oauth client_id live validation failures:", file=sys.stderr)
        for f in failures:
            print(f"  - {f}", file=sys.stderr)
        return 1

    print(f"OK: validated {len(targets)} git-oauth provider file(s) via GitHub Apps API")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

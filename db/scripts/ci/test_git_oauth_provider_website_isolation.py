#!/usr/bin/env python3
"""git-oauth provider website 隔离回归（OPT-20260818-043）。

多实例 GitLab token 按 host 隔离（catalog/换票以 target.website 匹配实例）。
若 gitlab-local / synology-gitlab 与默认实例（gitlab.daydaymoney.com）抢同一
website，独立实例会被绑到默认 CE，token 隔离形同虚设。

本测解析两棵 conf 树（git-oauth/providers + task-credential/git-oauth-providers）
中全部 gitlab provider 的 website，按 base.yaml 模板解析后断言 host 互不相同。
"""
from __future__ import annotations

import re
from pathlib import Path

import yaml

REPO = Path(__file__).resolve().parents[3]
BASE = REPO / "conf" / "base.yaml"
TREES = (
    REPO / "conf" / "auth" / "git-oauth" / "providers",
    REPO / "conf" / "auth" / "task-credential" / "git-oauth-providers",
)
DEFAULT_RE = re.compile(r"^\$\{([A-Za-z_][A-Za-z0-9_]*):-(.*)\}$")


def _default_of(value: object) -> str:
    """Extract the default from `${VAR:-default}`; leave literal values intact."""
    s = str(value)
    m = DEFAULT_RE.match(s.strip())
    return m.group(2) if m else s


def _resolve_website(raw: str, vars_map: dict[str, str]) -> str:
    s = raw
    # 迭代展开，处理嵌套引用（如 subdomains.gitlab = gitlab.${baseDomain}）。
    for _ in range(6):
        prev = s
        for key, val in vars_map.items():
            s = s.replace("${" + key + "}", val)
        if s == prev:
            break
    return s.rstrip("/").lower()


def test_gitlab_provider_websites_are_host_distinct() -> None:
    base = yaml.safe_load(BASE.read_text(encoding="utf-8"))
    assert isinstance(base, dict)
    subs = base.get("subdomains") or {}
    # 模板变量表：scheme / baseDomain / subdomains.* 全部取默认值。
    vars_map: dict[str, str] = {
        "scheme": _default_of(base.get("scheme", "https")),
        "baseDomain": _default_of(base.get("baseDomain", "daydaymoney.com")),
    }
    for k, v in subs.items():
        vars_map["subdomains." + k] = _default_of(v)

    seen: dict[str, str] = {}
    collisions: list[str] = []
    checked = 0
    for tree in TREES:
        if not tree.is_dir():
            continue
        for path in sorted(tree.glob("*.yaml")):
            data = yaml.safe_load(path.read_text(encoding="utf-8"))
            if not isinstance(data, dict) or data.get("provider") != "gitlab":
                continue
            sp = data.get("service_provider") or path.stem
            website = (data.get("target") or {}).get("website")
            if not website:
                continue
            host = _resolve_website(str(website), vars_map)
            checked += 1
            if host in seen and seen[host] != sp:
                collisions.append(f"{sp} 与 {seen[host]} 都解析到 {host}（{path.name}）")
            seen.setdefault(host, sp)
    assert checked >= 2, "未扫描到 gitlab provider，测试无意义"
    assert not collisions, "gitlab provider website 冲突:\n" + "\n".join(collisions)


if __name__ == "__main__":
    test_gitlab_provider_websites_are_host_distinct()
    print("ok")

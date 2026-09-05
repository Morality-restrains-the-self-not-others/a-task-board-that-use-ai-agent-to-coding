#!/usr/bin/env python3
"""Fail if conf/** (except base.yaml) or edge nginx templates hardcode domains.

Domain literals belong only in conf/base.yaml; other YAML / nginx examples must use
${baseDomain} / ${subdomains.*} templates.
"""
from __future__ import annotations

import os
import re
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("SKIP: PyYAML not installed", file=sys.stderr)
    raise SystemExit(0)

ROOT = Path(__file__).resolve().parents[3]
BASE = ROOT / "conf" / "base.yaml"
_ENV_DEFAULT_RE = re.compile(r"\$\{(\w+):-([^}]*)\}")
# Any registrable-looking FQDN (after stripping ${...} placeholders)
_FQDN_RE = re.compile(
    r"\b[a-zA-Z0-9_-]+(?:\.[a-zA-Z0-9_-]+)+\.(?:com|net|org|io|cn)\b"
)

NGINX_TEMPLATES = [
    ROOT / "task2app/scripts/daydaymoney.sh.nginx.example",
    ROOT / "task2app/scripts/daydaymoney.sh.nginx.http.conf",
    ROOT / "task2app/scripts/daydaymoney.hk.nginx.example",
    ROOT / "task2app/scripts/daydaymoney.hk.nginx.example",
]


def resolve_base_domain() -> str:
    raw = yaml.safe_load(BASE.read_text(encoding="utf-8")) or {}
    value = str(raw.get("baseDomain", ""))

    def repl(m: re.Match[str]) -> str:
        return os.environ.get(m.group(1), m.group(2) or "")

    return _ENV_DEFAULT_RE.sub(repl, value).strip()


def main() -> int:
    if not BASE.is_file():
        print(f"FAIL: missing {BASE}", file=sys.stderr)
        return 1
    base_domain = resolve_base_domain()
    if not base_domain or "." not in base_domain:
        print(f"FAIL: invalid baseDomain={base_domain!r}", file=sys.stderr)
        return 1

    needle = re.compile(
        rf"(?<!\$\{{)(?<![.\w])(?:[a-zA-Z0-9_-]+\.)*{re.escape(base_domain)}\b"
    )
    bare = re.compile(rf"(?<!\$\{{)(?<![.\w/-]){re.escape(base_domain)}\b")

    violations: list[str] = []

    def scan_conf_yaml(path: Path) -> None:
        text = path.read_text(encoding="utf-8")
        for i, line in enumerate(text.splitlines(), 1):
            stripped = line.lstrip()
            if stripped.startswith("#"):
                continue
            if needle.search(line) or bare.search(line):
                rel = path.relative_to(ROOT)
                violations.append(f"{rel}:{i}:{line.strip()[:160]}")

    def scan_nginx_template(path: Path) -> None:
        """Nginx templates must not contain any FQDN literal (even in comments)."""
        text = path.read_text(encoding="utf-8")
        for i, line in enumerate(text.splitlines(), 1):
            stripped_tpl = re.sub(r"\$\{[^}]+\}", "", line)
            if _FQDN_RE.search(stripped_tpl):
                rel = path.relative_to(ROOT)
                violations.append(f"{rel}:{i}:{line.strip()[:160]}")

    for path in sorted((ROOT / "conf").rglob("*.yaml")):
        if path.resolve() == BASE.resolve():
            continue
        scan_conf_yaml(path)

    for path in NGINX_TEMPLATES:
        if path.is_file():
            scan_nginx_template(path)

    if violations:
        print(
            f"FAIL: hardcoded domain outside conf/base.yaml "
            f"({len(violations)} hit(s); active baseDomain={base_domain!r}):",
            file=sys.stderr,
        )
        for v in violations[:50]:
            print(f"  {v}", file=sys.stderr)
        if len(violations) > 50:
            print(f"  ... +{len(violations) - 50} more", file=sys.stderr)
        return 1

    print(
        f"OK: no hardcoded domains in conf/** (except base.yaml) "
        f"or edge nginx templates (active baseDomain={base_domain!r})"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

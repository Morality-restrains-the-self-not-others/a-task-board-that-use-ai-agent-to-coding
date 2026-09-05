#!/usr/bin/env python3
"""Require cloud SDK call sites to use direct_network helpers.

Covers:
- Aliyun Tea: RuntimeOptions → see check_aliyun_tea_direct_runtime.py
- AWS boto3: boto3.client/resource must import direct_botocore_config
- Huawei: HttpConfig.get_default_config must import direct_http_config / apply_direct_http_config
- Tencent: SmsClient / HttpProfile construction should import apply_direct_client / direct_http_profile
  (best-effort string scan)

SSOT: .ai/01_project_constraints/23_app_startup_no_env_proxy.md

Exit codes:
  0 — ok
  1 — violations
  2 — IO error
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

SKIP_DIR_NAMES = {
    ".git",
    "node_modules",
    "vendor",
    "__pycache__",
    ".venv",
    "venv",
    "dist",
    "build",
    ".tox",
}

ALLOWLIST_SUFFIXES = (
    "/cloud/providers/aws/direct_network.py",
    "/cloud/providers/huawei/direct_network.py",
    "/cloud/providers/tencent/direct_network.py",
    "/cloud/providers/aliyun/direct_network.py",
    "/db/scripts/ci/check_cloud_sdk_direct_network.py",
    "/db/scripts/ci/test_check_cloud_sdk_direct_network.py",
)

RULES = (
    {
        "name": "aws-boto3",
        "trigger": re.compile(r"\bboto3\.(client|resource)\s*\("),
        "require_any": (
            "direct_botocore_config",
            "cloud.providers.aws.direct_network",
        ),
        "hint": "use cloud.providers.aws.direct_network.direct_botocore_config",
    },
    {
        "name": "huawei-http-config",
        "trigger": re.compile(r"\bHttpConfig\.get_default_config\s*\("),
        "require_any": (
            "direct_http_config",
            "apply_direct_http_config",
            "cloud.providers.huawei.direct_network",
        ),
        "hint": "use cloud.providers.huawei.direct_network.direct_http_config",
    },
    {
        "name": "tencent-http-profile",
        "trigger": re.compile(r"\bHttpProfile\s*\("),
        "require_any": (
            "direct_http_profile",
            "cloud.providers.tencent.direct_network",
        ),
        "hint": "use cloud.providers.tencent.direct_network.direct_http_profile",
    },
)


def monorepo_root() -> Path:
    here = Path(__file__).resolve()
    for parent in here.parents:
        if (parent / "db" / "registry.yaml").is_file():
            return parent
    raise FileNotFoundError("db/registry.yaml not found above checker")


def is_allowlisted(path: Path, root: Path) -> bool:
    rel = "/" + path.resolve().relative_to(root).as_posix()
    if any(rel.endswith(suf) for suf in ALLOWLIST_SUFFIXES):
        return True
    if "/test_" in rel or rel.endswith("_test.py"):
        return True
    return False


def iter_python_files(root: Path) -> list[Path]:
    base = root / "task2app" / "Saas_project"
    if not base.is_dir():
        return []
    out: list[Path] = []
    for path in base.rglob("*.py"):
        if any(part in SKIP_DIR_NAMES for part in path.parts):
            continue
        out.append(path)
    return sorted(out)


def find_violations(root: Path) -> list[str]:
    violations: list[str] = []
    for path in iter_python_files(root):
        if is_allowlisted(path, root):
            continue
        try:
            text = path.read_text(encoding="utf-8")
        except OSError as exc:
            raise RuntimeError(f"read failed: {path}: {exc}") from exc
        rel = path.relative_to(root).as_posix()
        for rule in RULES:
            if not rule["trigger"].search(text):
                continue
            if any(tok in text for tok in rule["require_any"]):
                continue
            # Report first trigger line
            for lineno, line in enumerate(text.splitlines(), start=1):
                if rule["trigger"].search(line):
                    violations.append(
                        f"{rel}:{lineno}: [{rule['name']}] missing direct_network helper — "
                        f"{rule['hint']}\n  {line.strip()}"
                    )
                    break
    return violations


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=None)
    args = parser.parse_args(argv)
    try:
        root = (args.root or monorepo_root()).resolve()
    except FileNotFoundError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 2
    try:
        violations = find_violations(root)
    except RuntimeError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 2
    if violations:
        print("VIOLATION: cloud SDK call sites must use direct_network helpers")
        for item in violations:
            print(item)
        return 1
    print("OK: cloud SDK call sites use direct_network helpers (or none present)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

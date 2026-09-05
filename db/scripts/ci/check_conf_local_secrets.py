#!/usr/bin/env python3
"""conf 已跟踪 YAML 禁止非空机密键；机密只放 conf-local（ADR-0054）。"""
from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parents[3]
SECRET_KEYS = frozenset(
    {
        "accesskeyid",
        "accesskeysecret",
        "apiv3key",
        "appsecret",
        "clientsecret",
        "encodingaeskey",
        "gatewayinternalsecret",
        "merchantserialno",
        "secretid",
        "secretkey",
        "ssojwtsecret",
        "taskauthinternalsecret",
        "token",
    }
)
SKIP_KEYS = frozenset({"passwordauthweb", "passwordauthgit", "publishablekey", "tokenurl"})
SKIP_SUBSTR = ("ttl", "expiryseconds", "maxtokens", "tokencount")
PLACEHOLDER = re.compile(r"(?i)^(replace|change.?me|your_|example|placeholder|dummy|\*|xxx)?$")
META_FILES = (
    ".ai/01_project_constraints/63_conf_local_secrets_only.md",
    ".cursor/rules/conf-local-secrets.mdc",
    "docs/adr/0054-conf-local-secrets-only.md",
)
GONE = (
    "runAll/scripts/daydaymoney-host-secrets.md",
    ".ai/01_project_constraints/63_host_secrets_catalog.md",
    ".cursor/rules/host-secrets-catalog.mdc",
    "db/scripts/ci/check_host_secrets_catalog.py",
)


def _collapse(key: str) -> str:
    return str(key).lower().replace("-", "").replace("_", "")


def _is_secret_key(key: str) -> bool:
    n = _collapse(key)
    if any(s in n for s in SKIP_KEYS) or any(s in n for s in SKIP_SUBSTR):
        return False
    if n in SECRET_KEYS or n in {"secret", "password", "passwd", "token"}:
        return True
    if n.endswith(("secret", "password", "passwd")):
        return True
    return "secretkey" in n or "secretid" in n or "accesskey" in n


def _is_secret_value(val) -> bool:
    if not isinstance(val, str):
        return False
    text = val.strip().strip("'\"")
    if not text or text.startswith("${"):
        return False
    return PLACEHOLDER.match(text) is None


def _is_forbidden_secret_doc(rel: str) -> bool:
    name = Path(rel).name.lower()
    return name.endswith("pwd.md") or name.endswith("password.md")


def _secret_hits(obj, path: str = "") -> list[str]:
    hits: list[str] = []
    if isinstance(obj, dict):
        for key, val in obj.items():
            cur = f"{path}.{key}" if path else str(key)
            if _is_secret_key(key) and _is_secret_value(val):
                hits.append(cur)
            hits.extend(_secret_hits(val, cur))
    elif isinstance(obj, list):
        for i, item in enumerate(obj):
            hits.extend(_secret_hits(item, f"{path}[{i}]"))
    return hits


def collect_violations(root: Path) -> list[str]:
    hits: list[str] = []
    for rel in META_FILES:
        if not (root / rel).is_file():
            hits.append(f"missing meta {rel}")
    for rel in GONE:
        if (root / rel).is_file():
            hits.append(f"HOST_SECRETS catalog must be removed: {rel}")
    gi = (root / ".gitignore").read_text(encoding="utf-8") if (root / ".gitignore").is_file() else ""
    if "/conf-local/" not in gi:
        hits.append(".gitignore must ignore /conf-local/")
    git_env = {k: v for k, v in os.environ.items() if k not in {"GIT_DIR", "GIT_INDEX_FILE", "GIT_WORK_TREE", "GIT_PREFIX"}}
    proc = subprocess.run(
        ["git", "-C", str(root / "conf"), "ls-files", "-z"],
        capture_output=True,
        check=False,
        env=git_env,
    )
    if proc.returncode != 0:
        hits.append("cannot list conf git files")
        return hits
    for raw in proc.stdout.split(b"\0"):
        if not raw:
            continue
        rel = raw.decode("utf-8", "replace")
        path = root / "conf" / rel
        if _is_forbidden_secret_doc(rel):
            hits.append(f"conf tracked secret doc must be removed: {rel}")
            continue
        if path.suffix not in {".yaml", ".yml"} or not path.is_file():
            continue
        try:
            data = yaml.safe_load(path.read_text(encoding="utf-8"))
        except Exception:
            continue
        keys = _secret_hits(data)
        if keys:
            hits.append(f"conf tracked secret keys must be empty: {rel} ({', '.join(keys)})")
    registry = root / "db" / "registry.yaml"
    if registry.is_file():
        try:
            data = yaml.safe_load(registry.read_text(encoding="utf-8"))
        except Exception:
            data = None
        if data is not None:
            keys = _secret_hits(data)
            if keys:
                hits.append(f"db/registry.yaml tracked secret keys must be empty ({', '.join(keys)})")
    return hits


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", default=str(ROOT))
    args = parser.parse_args()
    hits = collect_violations(Path(args.root))
    if hits:
        print("VIOLATION (rule 63_conf_local_secrets_only.md)")
        for h in hits:
            print(h)
        return 1
    print("conf-local secrets ok")
    return 0


if __name__ == "__main__":
    sys.exit(main())

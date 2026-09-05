#!/usr/bin/env python3
"""Move confidential YAML keys and PEM files from conf/ into conf-local/ (ADR-0054).

Does not print secret values. Idempotent: empty keys stay as skeleton.
"""
from __future__ import annotations

import re
import shutil
import sys
from pathlib import Path
from typing import Any

import yaml

sys.path.insert(0, str(Path(__file__).resolve().parent))
from conf_lib import deep_merge

ROOT = Path(__file__).resolve().parents[2]
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
KEY_LINE = re.compile(r"^(\s*)([A-Za-z0-9_]+):\s*(.*)$")


def _collapse(key: str) -> str:
    return str(key).lower().replace("-", "").replace("_", "")


def is_secret_key(key: str) -> bool:
    n = _collapse(key)
    if any(s in n for s in SKIP_KEYS) or any(s in n for s in SKIP_SUBSTR):
        return False
    if n in SECRET_KEYS or n in {"secret", "password", "passwd", "token"}:
        return True
    if n.endswith(("secret", "password", "passwd")):
        return True
    return "secretkey" in n or "secretid" in n or "accesskey" in n


def is_secret_value(val: Any) -> bool:
    if not isinstance(val, str):
        return False
    text = val.strip().strip("'\"")
    if not text or text.startswith("${"):
        return False
    return PLACEHOLDER.match(text) is None


def split_secrets(obj: Any) -> Any:
    if isinstance(obj, dict):
        out: dict[str, Any] = {}
        for key, val in obj.items():
            if is_secret_key(str(key)) and is_secret_value(val):
                out[key] = val
                continue
            nested = split_secrets(val)
            if nested:
                out[key] = nested
        return out
    if isinstance(obj, list):
        parts = []
        found = False
        for item in obj:
            nested = split_secrets(item)
            if nested:
                found = True
                parts.append(nested)
            elif isinstance(item, dict):
                parts.append({})
            else:
                parts.append(item)
        return parts if found else {}
    return {}


def blank_secret_lines(text: str) -> str:
    lines = []
    for line in text.splitlines(True):
        match = KEY_LINE.match(line.rstrip("\n"))
        if match and is_secret_key(match.group(2)):
            raw = match.group(3).strip().strip("'\"")
            if is_secret_value(raw):
                nl = "\n" if line.endswith("\n") else ""
                lines.append(f'{match.group(1)}{match.group(2)}: ""{nl}')
                continue
        lines.append(line)
    return "".join(lines)


def write_local_overlay(rel: str, secrets: dict[str, Any]) -> None:
    dest = ROOT / "conf-local" / Path(rel)
    dest.parent.mkdir(parents=True, exist_ok=True)
    existing: dict[str, Any] = {}
    if dest.is_file():
        loaded = yaml.safe_load(dest.read_text(encoding="utf-8")) or {}
        if isinstance(loaded, dict):
            existing = loaded
    dest.write_text(
        yaml.dump(deep_merge(existing, secrets), allow_unicode=True, default_flow_style=False, sort_keys=False),
        encoding="utf-8",
    )


def warn_tree_outside_pems() -> None:
    hints = [
        (
            ROOT / "taskGateway" / "certs" / "dev-gateway.pem",
            "conf-local/gateway/task-gateway/dev-gateway.pem",
        ),
        (
            ROOT / "taskGateway" / "certs" / "dev-gateway-key.pem",
            "conf-local/gateway/task-gateway/dev-gateway-key.pem",
        ),
        (
            ROOT / "db" / "task-auth" / "oidc_signing_key.pem",
            "conf-local/auth/task-auth/oidc_signing_key.pem",
        ),
        (
            ROOT / "taskAuth" / "oidc_signing_key.pem",
            "conf-local/auth/task-auth/oidc_signing_key.pem",
        ),
    ]
    for src, dest in hints:
        if src.is_file() and not (ROOT / dest).is_file():
            print(f"hint: copy {src.relative_to(ROOT)} -> {dest}")


def main() -> int:
    conf = ROOT / "conf"
    if not conf.is_dir():
        print("extract_conf_secrets: missing conf/", file=sys.stderr)
        return 1
    moved = 0
    for path in sorted(conf.rglob("*")):
        if not path.is_file() or ".git" in path.parts:
            continue
        rel = path.relative_to(conf).as_posix()
        if path.suffix.lower() in {".pem", ".key"}:
            dest = ROOT / "conf-local" / Path(rel)
            dest.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(path, dest)
            path.unlink()
            print(f"moved pem {rel}")
            moved += 1
            continue
        if path.suffix not in {".yaml", ".yml"}:
            continue
        try:
            data = yaml.safe_load(path.read_text(encoding="utf-8"))
        except Exception:
            continue
        secrets = split_secrets(data)
        if not secrets:
            if path.name.endswith(".local.yaml"):
                path.unlink(missing_ok=True)
                print(f"removed leftover {rel}")
            continue
        local_rel = rel.replace(".local.yaml", ".yaml") if path.name.endswith(".local.yaml") else rel
        write_local_overlay(local_rel, secrets)
        if path.name.endswith(".local.yaml"):
            path.unlink(missing_ok=True)
        else:
            path.write_text(blank_secret_lines(path.read_text(encoding="utf-8")), encoding="utf-8")
        print(f"extracted {rel} -> conf-local/{local_rel} keys={list(secrets)}")
        moved += 1
    print(f"extract_conf_secrets: {moved} files")
    warn_tree_outside_pems()
    return 0


if __name__ == "__main__":
    sys.exit(main())

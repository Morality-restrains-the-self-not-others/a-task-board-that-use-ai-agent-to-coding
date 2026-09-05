#!/usr/bin/env python3
"""Validate daydaymoney.yaml files listed in daydaymoney_yaml_manifest.yaml."""
from __future__ import annotations

import re
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("PyYAML required", file=sys.stderr)
    sys.exit(2)

ROOT = Path(__file__).resolve().parents[3]
MANIFEST = Path(__file__).resolve().parent / "daydaymoney_yaml_manifest.yaml"
SERVICE_ID_RE = re.compile(r"^[A-Za-z][A-Za-z0-9_-]{1,63}$")
FORBIDDEN = {"workspace_id", "workspace_ids", "project_id", "project_ids", "company_id", "tenant_id"}


def main() -> int:
    data = yaml.safe_load(MANIFEST.read_text(encoding="utf-8")) or {}
    paths = data.get("files") or []
    errors: list[str] = []
    for rel in paths:
        path = ROOT / rel
        if not path.is_file():
            errors.append(f"missing: {rel}")
            continue
        doc = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
        if not isinstance(doc, dict):
            errors.append(f"{rel}: root must be mapping")
            continue
        for k in FORBIDDEN:
            if k in doc:
                errors.append(f"{rel}: forbidden key {k}")
        if doc.get("version") != 1:
            errors.append(f"{rel}: version must be 1")
        sid = str(doc.get("service_id") or "").strip()
        if not SERVICE_ID_RE.match(sid):
            errors.append(f"{rel}: invalid service_id {sid!r}")
        tags = doc.get("tags") or []
        if not isinstance(tags, list) or not tags:
            errors.append(f"{rel}: tags required")
            continue
        svc = f"svc:{sid}"
        if not any(str(t).lower() == svc.lower() for t in tags):
            errors.append(f"{rel}: tags must include {svc}")
    if errors:
        print("\n".join(errors))
        return 1
    print(f"ok: {len(paths)} daydaymoney.yaml file(s)")
    return 0


if __name__ == "__main__":
    sys.exit(main())

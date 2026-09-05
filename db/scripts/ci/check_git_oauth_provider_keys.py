#!/usr/bin/env python3
"""Cross-check git-oauth provider service_provider/client_id across conf trees."""
from __future__ import annotations
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("skip: PyYAML not installed")
    sys.exit(0)

ROOT = Path(__file__).resolve().parents[3]


def load_providers(dir_path: Path) -> dict[str, dict]:
    out = {}
    if not dir_path.is_dir():
        return out
    for p in sorted(dir_path.glob("*.yaml")):
        if p.name.endswith(".ai.md"):
            continue
        data = yaml.safe_load(p.read_text()) or {}
        website = str(data.get("website") or data.get("host") or p.stem)
        out[website] = {
            "path": str(p.relative_to(ROOT)),
            "service_provider": data.get("service_provider") or data.get("provider"),
            "client_id": data.get("client_id") or (data.get("oauth") or {}).get("client_id"),
        }
    return out


def main() -> int:
    # OPT-20260806-057: django 目录退役，副本对比改为 auth vs task-credential
    auth = load_providers(ROOT / "conf/auth/git-oauth/providers")
    django = load_providers(ROOT / "conf/auth/task-credential/git-oauth-providers")
    bad = []
    for website, a in auth.items():
        # match by website substring
        matches = [d for w, d in django.items() if website in w or w in website or Path(a["path"]).stem.split("--")[0] in w]
        if not matches:
            continue
        d = matches[0]
        if a.get("client_id") and d.get("client_id") and str(a["client_id"]) != str(d["client_id"]):
            bad.append(f"{website}: client_id mismatch {a['path']} vs {d['path']}")
        if a.get("service_provider") and d.get("service_provider") and str(a["service_provider"]) != str(d["service_provider"]):
            bad.append(f"{website}: service_provider mismatch {a['path']} vs {d['path']}")
    if bad:
        print("git-oauth provider key check FAILED:")
        for b in bad:
            print(" ", b)
        return 1
    print(f"ok: compared {len(auth)} auth providers with {len(django)} credential providers")
    return 0


if __name__ == "__main__":
    sys.exit(main())

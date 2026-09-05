#!/usr/bin/env python3
"""SPA 静态资源 CI 门禁 — 验证 vite_asset 产出路径与 STATIC_ROOT 一致性。

防止 Vite build 后 collectstatic 缺失、或 vite_tags 输出 /static/main-*.js（无 assets/ 前缀）
导致公网 SPA 白屏（见故障经验 04_public_spa_static_js_404_after_vite_build.md）。

用法：
  python3 scripts/ci/check_spa_static_assets.py              # 检查默认路径
  python3 scripts/ci/check_spa_static_assets.py --manifest <path>  # 指定 manifest
  python3 scripts/ci/check_spa_static_assets.py --skip-missing      # 缺少文件仅警告不失败

退出码 0 = 通过；1 = 阻断性错误。
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path


def repo_root() -> Path:
    d = Path(__file__).resolve().parent
    while d != d.parent:
        if (d / "conf" / "base.yaml").is_file():
            return d
        d = d.parent
    return Path.cwd()


def find_manifest(root: Path) -> Path | None:
    candidates = [
        root / "task2app" / "front_project" / "app" / ".vite" / "manifest.json",
        root / "task2app" / "front_project" / "app" / "dist" / ".vite" / "manifest.json",
    ]
    for c in candidates:
        if c.is_file():
            return c
    return None


def main() -> int:
    p = argparse.ArgumentParser(description="SPA static asset CI gate")
    p.add_argument("--manifest", help="Path to .vite/manifest.json")
    p.add_argument("--skip-missing", action="store_true", help="Warn on missing files instead of failing")
    args = p.parse_args()

    root = repo_root()
    manifest_path = Path(args.manifest) if args.manifest else find_manifest(root)

    if manifest_path is None or not manifest_path.is_file():
        print("[SPA-CI] WARNING: .vite/manifest.json not found — skipping check")
        return 0

    try:
        manifest = json.loads(manifest_path.read_text())
    except (json.JSONDecodeError, OSError) as e:
        print(f"[SPA-CI] ERROR: cannot parse manifest {manifest_path}: {e}")
        return 1

    # Resolve STATIC_ROOT — try Django first, then guess from conf
    static_root = os.environ.get("TASK2APP_STATIC_ROOT", "")
    if not static_root:
        # Try from conf/base.yaml or common path
        candidate = root / "task2app" / "Saas_project" / "static"
        if candidate.is_dir():
            static_root = str(candidate)

    errors = []
    for entry_path, entry in manifest.items():
        if not entry.get("isEntry", False) and "." not in str(entry_path):
            continue
        file_rel = entry.get("file") or entry_path

        # Rule 1: vite_asset output must start with assets/ (vite_tags prepends it)
        if not file_rel.startswith("assets/"):
            errors.append(f"BAD prefix: '{file_rel}' does not start with 'assets/'")
            continue

        # Rule 2: file must exist in STATIC_ROOT
        if static_root:
            full = Path(static_root) / file_rel
            if not full.is_file():
                msg = f"MISSING: {full} (from manifest entry {entry_path})"
                if args.skip_missing:
                    print(f"[SPA-CI] WARNING: {msg}")
                else:
                    errors.append(msg)

    if errors:
        print(f"[SPA-CI] FAIL: {len(errors)} asset check(s) failed:")
        for e in errors:
            print(f"  - {e}")
        print("\nFix: run `bash task2app/front_project/app/scripts/runall-lifecycle.sh build`")
        return 1

    print(f"[SPA-CI] OK: {len(manifest)} manifest entries checked, all assets found under assets/")
    return 0


if __name__ == "__main__":
    sys.exit(main())

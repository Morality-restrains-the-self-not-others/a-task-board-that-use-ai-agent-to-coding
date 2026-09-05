#!/usr/bin/env python3
"""禁止把生产 conf / 本机密钥提交进源码仓（ADR-0052 P3）。

运行时 SSOT 是 daydaymoney-deploy（本机双写仍可读根目录 conf/ 子仓）。
源码仓只允许 conf.example/ schema；禁止提交 config.local.yaml。
根目录 conf/ 子仓视为双写运行时树，本门禁跳过。
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]

SKIP_DIR_PARTS = frozenset({".git", "node_modules", "vendor", "third_party", ".daydaymoney-deploy-seed"})

SECRET_KEYS = re.compile(
    r"(?im)^[ \t]*(password|secret|api_key|access_key|private_key|token)\s*:\s*(.+?)\s*$"
)
PLACEHOLDER = re.compile(
    r"(?i)^(replace|change.?me|your_|example|placeholder|dummy|\*|\$\{|xxx)"
)


def _under_root_conf(root: Path, path: Path) -> bool:
    try:
        path.resolve().relative_to((root / "conf").resolve())
        return True
    except (ValueError, OSError):
        return False


def _is_skipped_dir(path: Path) -> bool:
    return any(part in SKIP_DIR_PARTS for part in path.parts)


def scan_example_yaml(path: Path) -> list[str]:
    hits: list[str] = []
    try:
        text = path.read_text(encoding="utf-8", errors="replace")
    except OSError as exc:
        return [f"{path}: read error {exc}"]
    for m in SECRET_KEYS.finditer(text):
        raw = m.group(2).strip().strip("'\"")
        if PLACEHOLDER.search(raw):
            continue
        if len(raw) < 4:
            continue
        hits.append(f"{path}: conf.example secret-like key {m.group(1)} is not a placeholder")
    return hits


def _iter_files(root: Path):
    git_dir = root / ".git"
    if git_dir.exists() or git_dir.is_file():
        import subprocess

        proc = subprocess.run(
            ["git", "-C", str(root), "ls-files", "-z"],
            check=False,
            capture_output=True,
        )
        if proc.returncode == 0 and proc.stdout:
            for rel in proc.stdout.split(b"\0"):
                if not rel:
                    continue
                yield root / rel.decode("utf-8", "replace")
            return
    yield from (p for p in root.rglob("*") if p.is_file())


def collect_violations(root: Path) -> list[str]:
    root = root.resolve()
    hits: list[str] = []
    for path in _iter_files(root):
        if not path.is_file() or _is_skipped_dir(path):
            continue
        if _under_root_conf(root, path):
            continue
        name = path.name
        if name == "config.local.yaml" or name.endswith(".local.yaml"):
            try:
                rel = path.relative_to(root)
            except ValueError:
                rel = path
            hits.append(f"{rel}: production local overlay must not be committed to source")
            continue
        if "conf.example" in path.parts and path.suffix in {".yaml", ".yml"}:
            hits.extend(scan_example_yaml(path))
    return hits


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT)
    args = parser.parse_args(argv)
    hits = collect_violations(args.root)
    if hits:
        print("VIOLATION (ADR-0052 P3 / no prod conf in source):")
        for hit in hits[:80]:
            print(f"  - {hit}")
        if len(hits) > 80:
            print(f"  ... +{len(hits) - 80} more")
        return 1
    print("ok: no production conf overlays in source (root conf/ dual-write skipped)")
    return 0


if __name__ == "__main__":
    sys.exit(main())

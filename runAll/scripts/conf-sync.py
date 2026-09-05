#!/usr/bin/env python3
"""Apply conf/<app>/sync.manifest.yaml — write GENERATED fragment YAML files."""
from __future__ import annotations

import sys
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parent))
from conf_lib import generated_header, load_yaml, repo_root, resolve_conf_app_dir


def _pick_keys(source: dict[str, Any], pick: list[Any]) -> dict[str, Any]:
    out: dict[str, Any] = {}
    for item in pick:
        if isinstance(item, str):
            if item in source:
                out[item] = source[item]
            continue
        if isinstance(item, dict):
            src_key = item.get("from") or item.get("path")
            dst_key = item.get("as") or src_key
            if src_key and src_key in source:
                out[dst_key] = source[src_key]
    return out


def sync_app(app_dir: Path) -> None:
    manifest_path = app_dir / "sync.manifest.yaml"
    if not manifest_path.is_file():
        return
    manifest = load_yaml(manifest_path)
    fragments = manifest.get("fragments") or []
    if not isinstance(fragments, list):
        raise ValueError(f"{manifest_path}: fragments must be a list")
    app_rel = app_dir.relative_to(repo_root() / "conf").as_posix()
    for entry in fragments:
        if not isinstance(entry, dict):
            continue
        from_rel = str(entry.get("from") or "").strip()
        to_name = str(entry.get("to") or "").strip()
        pick = entry.get("pick") or []
        if not from_rel or not to_name:
            continue
        src_path = (app_dir / from_rel).resolve()
        if not src_path.is_file():
            raise FileNotFoundError(f"{manifest_path}: missing source {src_path}")
        source_data = load_yaml(src_path)
        picked = _pick_keys(source_data, pick if isinstance(pick, list) else [])
        rel_source = src_path.relative_to(repo_root()).as_posix()
        header = generated_header(app_rel, rel_source)
        import yaml

        yaml_body = yaml.dump(
            picked, allow_unicode=True, default_flow_style=False, sort_keys=False
        )
        out_path = app_dir / to_name
        new_text = header + yaml_body
        existing_body = ""
        if out_path.is_file():
            existing = out_path.read_text(encoding="utf-8")
            existing_body = existing.split("\n", 3)[-1] if existing.startswith("# GENERATED") else existing
        if existing_body != yaml_body:
            out_path.write_text(new_text, encoding="utf-8")
            print(f"synced {out_path.relative_to(repo_root())}")
        _sync_conf_local_overlay(
            repo_root(), app_dir, src_path, to_name, pick if isinstance(pick, list) else []
        )


def _sync_conf_local_overlay(
    root: Path, app_dir: Path, src_path: Path, to_name: str, pick: list[Any]
) -> None:
    from conf_local import conf_local_root

    try:
        src_rel = src_path.relative_to(root / "conf")
    except ValueError:
        return
    local_src = conf_local_root(root) / src_rel
    if not local_src.is_file():
        return
    local_picked = _pick_keys(load_yaml(local_src), pick)
    if not local_picked:
        return
    dest = conf_local_root(root) / app_dir.relative_to(root / "conf") / to_name
    dest.parent.mkdir(parents=True, exist_ok=True)
    import yaml

    dest.write_text(
        yaml.dump(local_picked, allow_unicode=True, default_flow_style=False, sort_keys=False),
        encoding="utf-8",
    )


def main(argv: list[str]) -> None:
    root = repo_root()
    if len(argv) > 1:
        app = argv[1].strip()
        sync_app(resolve_conf_app_dir(root, app))
        return
    for func_dir in sorted((root / "conf").iterdir()):
        if not func_dir.is_dir() or func_dir.name.startswith("."):
            continue
        # 一级目录也可持有 sync.manifest.yaml（如 conf/task-referral/）
        if (func_dir / "sync.manifest.yaml").is_file():
            sync_app(func_dir)
        for app_dir in sorted(func_dir.iterdir()):
            if app_dir.is_dir():
                sync_app(app_dir)


if __name__ == "__main__":
    main(sys.argv)

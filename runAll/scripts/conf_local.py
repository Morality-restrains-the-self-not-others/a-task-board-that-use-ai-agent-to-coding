#!/usr/bin/env python3
"""Merge conf-local/<rel> over a conf YAML mapping (ADR-0054)."""
from __future__ import annotations

from pathlib import Path
from typing import Any

from conf_lib import deep_merge, load_yaml


def conf_local_root(root: Path) -> Path:
    return root / "conf-local"


def merge_conf_local(root: Path, rel_under_conf: str, data: dict[str, Any]) -> dict[str, Any]:
    rel = (rel_under_conf or "").replace("\\", "/").lstrip("/")
    if not rel or ".." in Path(rel).parts:
        return data
    path = conf_local_root(root) / Path(rel)
    if not path.is_file():
        return data
    return deep_merge(data, load_yaml(path))


def overlay_conf_file(conf_path: Path) -> dict[str, Any]:
    """Load a file under .../conf/... and overlay conf-local/<same rel> (ADR-0054)."""
    conf_path = Path(conf_path).resolve()
    data: dict[str, Any] = {}
    if conf_path.is_file():
        loaded = load_yaml(conf_path)
        if isinstance(loaded, dict):
            data = loaded
    for parent in conf_path.parents:
        if parent.name == "conf" and (parent / "base.yaml").is_file():
            rel = conf_path.relative_to(parent).as_posix()
            return merge_conf_local(parent.parent, rel, data)
    return data

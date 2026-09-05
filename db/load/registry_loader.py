"""Resolve SQLite paths from db/registry.yaml (monorepo single source of truth)."""

from __future__ import annotations

import os
from functools import lru_cache
from pathlib import Path
from typing import Any

try:
    import yaml
except ImportError:  # pragma: no cover - PyYAML expected in dev venv
    yaml = None  # type: ignore

_REGISTRY_NAME = "registry.yaml"
_ENV_BY_KEY = {
    "task-auth": "TASKAUTH_DATABASE_PATH",
    "task-bill": "TASKBILL_DATABASE_PATH",
    "task-budget": "TASK_BUDGET_DATABASE_PATH",
    "git-oauth": "GITOAUTH_DATABASE_PATH",
    "email": "EMAIL_DATABASE_PATH",
    "ai-provider": "AI_PROVIDER_DATABASE_PATH",
    "task-project": "TASK_PROJECT_DATABASE_PATH",
    "task-task": "TASK_TASK_DATABASE_PATH",
    "task-cloud": "TASK_CLOUD_DATABASE_PATH",
    "task-ai-comment": "TASK_AI_COMMENT_DATABASE_PATH",
    "container": "CONTAINER_DATABASE_PATH",
}


def find_monorepo_root(start: Path | None = None) -> Path:
    """Walk upward until db/registry.yaml exists."""
    seeds: list[Path] = []
    if start is not None:
        seeds.append(start if start.is_dir() else start.parent)
    seeds.append(Path.cwd())
    seeds.append(Path(__file__).resolve().parent.parent.parent)

    seen: set[Path] = set()
    for seed in seeds:
        current = seed.resolve()
        for _ in range(10):
            if current in seen:
                break
            seen.add(current)
            registry = current / "db" / _REGISTRY_NAME
            if registry.is_file():
                return current
            parent = current.parent
            if parent == current:
                break
            current = parent
    raise FileNotFoundError(
        "未找到 db/registry.yaml；请确认位于 monorepo 根目录下。"
    )


@lru_cache(maxsize=1)
def _load_registry(monorepo_root: str) -> dict[str, Any]:
    registry_path = Path(monorepo_root) / "db" / _REGISTRY_NAME
    raw = registry_path.read_text(encoding="utf-8")
    if yaml is None:
        raise RuntimeError("PyYAML 未安装，无法解析 db/registry.yaml")
    data = yaml.safe_load(raw) or {}
    databases = data.get("databases") or {}
    if not isinstance(databases, dict):
        raise ValueError("registry.yaml: databases 必须为 mapping")
    return databases


def resolve_database_path(key: str, *, monorepo_root: Path | None = None, mkdir: bool = True) -> Path:
    """Return absolute SQLite path for registry key."""
    key = str(key or "").strip()
    if not key:
        raise KeyError("database key required")

    env_name = _ENV_BY_KEY.get(key)
    if env_name:
        override = os.environ.get(env_name, "").strip()
        if override:
            path = Path(override).expanduser()
            if not path.is_absolute() and monorepo_root is not None:
                path = (monorepo_root / path).resolve()
            elif not path.is_absolute():
                path = path.resolve()
            if mkdir:
                path.parent.mkdir(parents=True, exist_ok=True)
            return path

    root = monorepo_root or find_monorepo_root()
    databases = _load_registry(str(root.resolve()))
    block = databases.get(key)
    if not isinstance(block, dict):
        raise KeyError(f"registry.yaml 未定义 database key: {key}")

    rel = str(block.get("path") or "").strip()
    if not rel:
        raise ValueError(f"registry.yaml databases.{key}.path 为空")

    path = (root / rel).resolve()
    if mkdir:
        path.parent.mkdir(parents=True, exist_ok=True)
    return path

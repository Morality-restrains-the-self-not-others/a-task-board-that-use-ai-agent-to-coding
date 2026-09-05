#!/usr/bin/env python3
"""所有进程加载 conf YAML 必须叠加 conf-local（约束索引第 59 条 / ADR-0054）。

生产源码若 os.ReadFile / yaml.safe_load / readFileSync 打开 conf/**/*.yaml，
必须走 SSOT 深合并（confload / overlay_conf_file 等），禁止只读 tracked conf。
禁止加载 config.local.yaml。
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]

META_FILES = (
    ".ai/01_project_constraints/64_conf_local_overlay_all_processes.md",
    ".cursor/rules/conf-local-overlay-all-processes.mdc",
    ".ai/01_project_constraints/00_project_constraints.md",
)

INDEX_NEEDLE = "64_conf_local_overlay_all_processes.md"

OVERLAY_TOKENS = (
    "ReadAppConfig",
    "ReadAppFragment",
    "UnmarshalYAMLMerged",
    "ReadOverlaidYAML",
    "ReadYAMLMerged",
    "MergeYAMLAtPath",
    "MergeConfLocal",
    "overlay_conf_file",
    "merge_conf_local",
    "overlayConfLocal",
    "overlayConfLocalRel",
    "overlayWechatLocalConfig",
    "_yaml_merged",
    "_conf_local_overlay",
    "applyMySQLPasswordOverlay",
    "ResolveBaseYaml",
    "Conf-Local-Overlay-OK:",
)

READ_YAML_CALL_RE = re.compile(
    r"""(?x)
    (?:os\.ReadFile|ioutil\.ReadFile|readFileSync|readYamlFile|fs\.readFile(?:Sync)?)
    \s*\([^)]*(?:conf/|\.ya?ml)
    """
)
PY_LOAD_RE = re.compile(r"\byaml\.safe_load\b|\byaml\.load\s*\(")
YAML_RE = re.compile(r"\.(?:ya?ml)\b")
CONF_DIR_RE = re.compile(
    r"""(?x)
    ["']conf/
    | ["']conf["']
    """
)
QUOTED_CONF_LOCAL_RE = re.compile(r"""["']conf-local["']""")
CONFIG_LOCAL_USE_RE = re.compile(
    r"(?:ReadFile|readFileSync|existsSync|Join|join|open|safe_load)\s*\([^)\n]{0,200}config\.local\.ya?ml",
    re.IGNORECASE,
)
WAIVER_RE = re.compile(r"Conf-Local-Overlay-OK:")

SKIP_DIR_PARTS = frozenset(
    {
        ".git",
        "node_modules",
        "vendor",
        "third_party",
        "gitlab-ce",
        "gitlab_home",
        "sdk",
        "__pycache__",
        ".venv",
        "venv",
        "dist",
        "build",
        ".tox",
        ".pytest_cache",
        "coverage",
        "tmp",
        "logs",
        ".daydaymoney-deploy-seed",
    }
)
SOURCE_SUFFIXES = frozenset({".go", ".py", ".js", ".mjs", ".cjs", ".ts", ".tsx", ".sh"})
MAX_FILE_BYTES = 1_000_000
# CI/lint 只检查 tracked conf；抽取脚本以 tracked 为源；load_yaml 是 overlay 原语。
EXEMPT_PREFIXES = (
    "shareLib/confload/",
    "db/scripts/ci/",
    "playwright/",
)
EXEMPT_RELS = frozenset(
    {
        "runAll/scripts/conf_local.py",
        "runAll/scripts/conf_lib.py",
        "runAll/scripts/extract_conf_secrets_to_local.py",
        "db/scripts/ci/check_conf_local_overlay.py",
        "db/scripts/ci/test_check_conf_local_overlay.py",
    }
)


def posix_rel(path: Path, root: Path) -> str:
    return path.relative_to(root).as_posix()


def should_skip_dir(path: Path, root: Path) -> bool:
    try:
        rel = path.relative_to(root)
    except ValueError:
        return True
    return any(part in SKIP_DIR_PARTS for part in rel.parts)


def is_test_path(rel: str) -> bool:
    name = Path(rel).name
    if name.endswith("_test.go") or name.startswith(("test_", "test-")):
        return True
    if ".test." in name or ".spec." in name:
        return True
    parts = rel.split("/")
    return any(p in {"testdata", "fixtures", "tests", "test"} for p in parts)


def is_exempt(rel: str) -> bool:
    if rel in EXEMPT_RELS:
        return True
    if any(rel.startswith(p) for p in EXEMPT_PREFIXES):
        return True
    top = rel.split("/", 1)[0]
    return top.endswith("-wt")


def has_overlay(text: str) -> bool:
    if WAIVER_RE.search(text) or QUOTED_CONF_LOCAL_RE.search(text):
        return True
    return any(tok in text for tok in OVERLAY_TOKENS)


def looks_like_conf_yaml_read(rel: str, text: str) -> bool:
    if not YAML_RE.search(text) or not CONF_DIR_RE.search(text):
        return False
    suffix = Path(rel).suffix.lower()
    if suffix in {".py", ".sh"}:
        return bool(PY_LOAD_RE.search(text))
    return bool(READ_YAML_CALL_RE.search(text))


def classify(rel: str, text: str) -> tuple[str, str]:
    """Return (kind, reason). kind is ok | violation."""
    if is_test_path(rel) or is_exempt(rel):
        return "ok", ""
    if CONFIG_LOCAL_USE_RE.search(text) and not has_overlay(text):
        return "violation", "loads config.local.yaml; use conf-local overlay"
    if looks_like_conf_yaml_read(rel, text) and not has_overlay(text):
        return "violation", "reads conf YAML without conf-local overlay"
    return "ok", ""


def collect_meta_violations(root: Path) -> list[str]:
    hits: list[str] = []
    for rel in META_FILES:
        path = root / rel
        if not path.is_file():
            hits.append(f"missing meta {rel}")
            continue
        if rel.endswith("00_project_constraints.md"):
            body = path.read_text(encoding="utf-8")
            if INDEX_NEEDLE not in body:
                hits.append(f"00_project_constraints.md must index {INDEX_NEEDLE}")
    return hits


def iter_scan_files(root: Path):
    for path in root.rglob("*"):
        if not path.is_file():
            continue
        if should_skip_dir(path, root):
            continue
        try:
            if path.stat().st_size > MAX_FILE_BYTES:
                continue
        except OSError:
            continue
        if path.suffix.lower() not in SOURCE_SUFFIXES:
            continue
        try:
            yield posix_rel(path, root), path
        except ValueError:
            continue


def collect_source_violations(root: Path) -> list[str]:
    hits: list[str] = []
    for rel, path in iter_scan_files(root):
        try:
            text = path.read_text(encoding="utf-8")
        except (OSError, UnicodeDecodeError):
            continue
        kind, reason = classify(rel, text)
        if kind == "violation":
            hits.append(f"{rel}: {reason}")
    return hits


def collect_violations(root: Path) -> list[str]:
    return collect_meta_violations(root) + collect_source_violations(root)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=ROOT)
    args = parser.parse_args()
    hits = collect_violations(args.root.resolve())
    if hits:
        print("VIOLATION (rule 64_conf_local_overlay_all_processes.md / constraint 59):")
        for h in hits:
            print(f"  {h}")
        return 1
    print("check_conf_local_overlay: ok")
    return 0


if __name__ == "__main__":
    sys.exit(main())

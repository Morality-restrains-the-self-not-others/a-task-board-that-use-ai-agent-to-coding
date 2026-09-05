#!/usr/bin/env python3
"""
Migration integrity checker — scans Django migration files for dependencies
on apps not listed in INSTALLED_APPS.

Usage:
  python3 dataMigrate/check_migration_integrity.py          # scan Saas_project
  python3 dataMigrate/check_migration_integrity.py --json   # machine-readable output

Exit code: 1 if stale dependencies found, 0 otherwise.
"""

from __future__ import annotations

import argparse
import ast
import re
import sys
from pathlib import Path
from typing import Dict, List, Set, Tuple


# ── known-safe built-in / third-party apps (always available) ──────────
DJANGO_CONTRIB_APPS = frozenset({
    "admin", "auth", "contenttypes", "sessions", "messages", "staticfiles",
})
THIRD_PARTY_APPS = frozenset({
    "rest_framework", "rest_framework.authtoken", "corsheaders", "drf_yasg",
})
ALWAYS_AVAILABLE = DJANGO_CONTRIB_APPS | THIRD_PARTY_APPS


def _parse_dependencies(source: str) -> List[List[str]]:
    """Extract the `dependencies` list-of-lists from a migration file AST."""
    tree = ast.parse(source)
    for node in ast.walk(tree):
        if isinstance(node, ast.ClassDef) and node.name == "Migration":
            for item in node.body:
                if isinstance(item, ast.Assign):
                    for target in item.targets:
                        if isinstance(target, ast.Name) and target.id == "dependencies":
                            deps_node = item.value
                            if isinstance(deps_node, ast.List):
                                result: List[List[str]] = []
                                for elt in deps_node.elts:
                                    # Django migration dependencies can be lists or tuples:
                                    #   dependencies = [('app', '0001_initial'), ...]
                                    if isinstance(elt, (ast.List, ast.Tuple)):
                                        result.append([
                                            e.value if isinstance(e, ast.Constant) else ""
                                            for e in elt.elts
                                        ])
                                return result
    return []


def _extract_app_label(dep: List[str]) -> str:
    """Extract the app label from a dependency pair like ('appname', '0001_initial')."""
    if len(dep) >= 1 and dep[0]:
        return dep[0]
    return ""


def _resolve_saas_settings_path() -> Path | None:
    """Locate Saas_project settings.py from known monorepo locations."""
    candidates = [
        Path.cwd() / "task2app" / "Saas_project" / "saas_project" / "settings.py",
        Path.cwd() / "Saas_project" / "saas_project" / "settings.py",
    ]
    # also search upward from cwd
    for p in Path.cwd().parents:
        cand = p / "task2app" / "Saas_project" / "saas_project" / "settings.py"
        if cand not in candidates:
            candidates.append(cand)
    candidates.extend([
        Path("/tmp/ram-work/task2app/Saas_project/saas_project/settings.py"),
    ])
    for cand in candidates:
        if cand.is_file():
            return cand
    return None


def get_installed_app_labels(settings_path: Path) -> Set[str]:
    """Parse INSTALLED_APPS from Django settings.py to get local app labels."""
    source = settings_path.read_text()
    tree = ast.parse(source)

    apps_raw: List[str] = []
    for node in ast.walk(tree):
        if isinstance(node, ast.Assign):
            for target in node.targets:
                if isinstance(target, ast.Name) and target.id == "INSTALLED_APPS":
                    if isinstance(node.value, ast.List):
                        for elt in node.value.elts:
                            if isinstance(elt, ast.Constant):
                                apps_raw.append(elt.value)

    labels: Set[str] = set()
    for app in apps_raw:
        # "accounts" / "projects" / "cloud" etc.
        if "." not in app:
            labels.add(app)
        else:
            # "core.apps.CoreConfig" -> "core"
            labels.add(app.split(".")[0])

    return labels


def scan_migrations(migration_root: Path) -> Dict[str, List[Tuple[Path, List[str]]]]:
    """
    Walk migration_root and collect every migration file's dependency app labels.

    Returns {migration_file_path: [(dependency_app_label, ...), ...]}
    -only for files that have non-empty dependencies.
    """
    results: Dict[str, List[Tuple[Path, List[str]]]] = {}
    seen: Set[str] = set()

    for mig_file in sorted(migration_root.rglob("migrations/*.py")):
        if mig_file.name == "__init__.py":
            continue
        try:
            source = mig_file.read_text()
        except Exception:
            continue
        deps = _parse_dependencies(source)
        if not deps:
            continue
        # deps is List[List[str]] — each inner list is one dependency tuple
        results[str(mig_file)] = [(mig_file, dep) for dep in deps]
    return results


def check(settings_path: Path | None = None) -> Tuple[bool, List[str]]:
    """Main entry point. Returns (ok, report_lines)."""
    if settings_path is None:
        settings_path = _resolve_saas_settings_path()
    if settings_path is None or not settings_path.is_file():
        return False, ["ERROR: Cannot locate Django settings.py"]

    installed = get_installed_app_labels(settings_path) | ALWAYS_AVAILABLE
    migration_root = settings_path.parents[1]  # saas_project/ -> Saas_project/

    all_deps = scan_migrations(migration_root)

    issues: List[str] = []
    for filepath, dep_list in all_deps.items():
        rel = Path(filepath).relative_to(migration_root)
        for mig_file, dep in dep_list:
            app_label = _extract_app_label(dep)
            if app_label and app_label not in installed:
                issues.append(
                    f"  {rel}: depends on '{app_label}' (from {dep}) — NOT in INSTALLED_APPS"
                )

    if issues:
        report = ["❌ Stale migration dependencies found:"]
        report.extend(issues)
        report.append(f"\n  Total: {len(issues)} stale dependencies")
        report.append(f"  Installed apps: {sorted(installed - ALWAYS_AVAILABLE)}")
        return False, report

    return True, [f"✅ All migration dependencies reference installed apps ({len(all_deps)} migration files scanned)"]


def main() -> int:
    parser = argparse.ArgumentParser(description="Check Django migration dependency integrity")
    parser.add_argument("--json", action="store_true", help="Machine-readable JSON output")
    parser.add_argument("--settings", type=str, help="Path to settings.py")
    args = parser.parse_args()

    settings_path = Path(args.settings) if args.settings else None
    ok, lines = check(settings_path)

    if args.json:
        import json
        print(json.dumps({"ok": ok, "report": "\n".join(lines)}))
    else:
        for line in lines:
            print(line)

    return 0 if ok else 1


if __name__ == "__main__":
    raise SystemExit(main())

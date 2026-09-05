#!/usr/bin/env python3
"""
Go DDD compliance checker for valueStream domain layer.

Checks:
1) domain/**/*.go must not import infrastructure/application packages.
2) aggregates must not depend on services/repositories.
3) repositories package must define at least one interface.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[3]

FORBIDDEN_IMPORT_PREFIXES = (
    "gopkg.in/yaml.v3",
    "database/sql",
    "gorm.io",
    "net/http",
    "os/exec",
)

IMPORT_BLOCK_RE = re.compile(r"(?ms)^\s*import\s*\((.*?)\)")
IMPORT_SINGLE_RE = re.compile(r'(?m)^\s*import\s+"([^"]+)"')
QUOTED_PATH_RE = re.compile(r'"([^"]+)"')
INTERFACE_RE = re.compile(r"\btype\s+\w+\s+interface\s*\{")


def parse_imports(source: str) -> set[str]:
    imports: set[str] = set()
    for block in IMPORT_BLOCK_RE.findall(source):
        for line in block.splitlines():
            line = line.strip()
            if not line or line.startswith("//"):
                continue
            match = QUOTED_PATH_RE.search(line)
            if match:
                imports.add(match.group(1))
    for single in IMPORT_SINGLE_RE.findall(source):
        imports.add(single)
    return imports


def is_external_import(imp: str) -> bool:
    # External imports usually have dot in first segment: github.com/..., gopkg.in/...
    first_segment = imp.split("/", 1)[0]
    return "." in first_segment


def check_domain_imports(go_files: list[Path], module_root: str, forbid_external: bool) -> list[str]:
    errors: list[str] = []
    local_forbidden = (f"{module_root}/src", *FORBIDDEN_IMPORT_PREFIXES)
    for path in go_files:
        rel = path.relative_to(REPO_ROOT)
        source = path.read_text(encoding="utf-8")
        imports = parse_imports(source)
        for imp in sorted(imports):
            for forbidden in local_forbidden:
                if imp == forbidden or imp.startswith(forbidden + "/"):
                    errors.append(f"{rel}: forbidden import `{imp}` (matched `{forbidden}`)")
            if forbid_external and is_external_import(imp):
                errors.append(f"{rel}: external dependency not allowed in domain layer `{imp}`")
    return errors


def check_aggregate_boundaries(go_files: list[Path], module_root: str) -> list[str]:
    errors: list[str] = []
    service_prefix = f"{module_root}/domain/services"
    repository_prefix = f"{module_root}/domain/repositories"
    for path in go_files:
        rel = path.relative_to(REPO_ROOT)
        source = path.read_text(encoding="utf-8")
        imports = parse_imports(source)
        if "aggregates" in rel.parts:
            for imp in imports:
                if imp.startswith(service_prefix):
                    errors.append(f"{rel}: aggregate must not import services `{imp}`")
                if imp.startswith(repository_prefix):
                    errors.append(f"{rel}: aggregate must not import repositories `{imp}`")
    return errors


def check_repository_interfaces(repo_files: list[Path], module_root: str) -> list[str]:
    errors: list[str] = []
    if not repo_files:
        return [f"{module_root}/domain/repositories: no repository files found"]
    has_interface = False
    for path in repo_files:
        source = path.read_text(encoding="utf-8")
        if INTERFACE_RE.search(source):
            has_interface = True
    if not has_interface:
        errors.append(f"{module_root}/domain/repositories: no interface definitions found")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description="Go DDD compliance checker")
    parser.add_argument(
        "--module-root",
        default="valueStream",
        help="module root folder under workspace (default: valueStream)",
    )
    parser.add_argument(
        "--forbid-external",
        action="store_true",
        help="forbid all external dependencies in domain imports",
    )
    parser.add_argument(
        "--strict-module-root",
        action="store_true",
        help="fail when <module-root>/domain does not exist",
    )
    args, _ = parser.parse_known_args()  # Keep compatibility with forwarded args.

    module_root = args.module_root.strip() or "valueStream"
    domain_root = REPO_ROOT / module_root / "domain"

    if not domain_root.is_dir():
        if args.strict_module_root or args.module_root != "valueStream":
            print(f"=== Go DDD compliance check failed ===", file=sys.stderr)
            print(f"{module_root}/domain not found under workspace root", file=sys.stderr)
            return 2
        print(f"Go DDD check skipped: {module_root}/domain not found.")
        return 0

    go_files = sorted(domain_root.glob("**/*.go"))
    repo_files = sorted((domain_root / "repositories").glob("*.go"))

    errors: list[str] = []
    errors.extend(check_domain_imports(go_files, module_root, args.forbid_external))
    errors.extend(check_aggregate_boundaries(go_files, module_root))
    errors.extend(check_repository_interfaces(repo_files, module_root))

    if errors:
        print("=== Go DDD compliance check failed ===", file=sys.stderr)
        for err in errors:
            print(err, file=sys.stderr)
        return 1

    print("Go DDD compliance check passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())


#!/usr/bin/env python3
"""CI: business openDB/OpenDB/openBudgetDB must not run dataMigrate on connect.

Exit 0 = clean, 1 = violations found.
Allowed: migrate CLI, tests, bootstrap CLIs, migrate.go definitions.
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

# Functions that establish DB connections for long-running business processes.
OPEN_FUNCS = (
    "openDB",
    "OpenDB",
    "openBudgetDB",
)

MIGRATE_CALLS = re.compile(
    r"\b(runDataMigrate|runDataMigrateFromDir|runBudgetDataMigrate|EnsureSchema|RunGoDataMigrate)\b"
)

FUNC_START = re.compile(
    r"^func\s+(?:\([^)]+\)\s+)?(" + "|".join(OPEN_FUNCS) + r")\s*\("
)


def _iter_go_files() -> list[Path]:
    skip_parts = {".git", "vendor", "node_modules", "bin"}
    out: list[Path] = []
    for p in ROOT.rglob("*.go"):
        if any(part in skip_parts for part in p.parts):
            continue
        if p.name.endswith("_test.go"):
            continue
        out.append(p)
    return out


def find_open_func_migrate_calls(text: str, path: Path) -> list[str]:
    """Return violation messages for migrate calls inside open* function bodies."""
    lines = text.splitlines()
    violations: list[str] = []
    i = 0
    while i < len(lines):
        m = FUNC_START.match(lines[i])
        if not m:
            i += 1
            continue
        func = m.group(1)
        # find opening brace
        brace_line = i
        while brace_line < len(lines) and "{" not in lines[brace_line]:
            brace_line += 1
        if brace_line >= len(lines):
            break
        depth = 0
        started = False
        j = brace_line
        while j < len(lines):
            for ch in lines[j]:
                if ch == "{":
                    depth += 1
                    started = True
                elif ch == "}":
                    depth -= 1
            if started and depth == 0:
                for k in range(i, j + 1):
                    code = lines[k].split("//", 1)[0]
                    cm = MIGRATE_CALLS.search(code)
                    if cm:
                        violations.append(
                            f"{path.relative_to(ROOT)}:{k + 1}: {func}() calls {cm.group(1)} — "
                            "business connect path must not migrate (use 9999 / migrate.sh)"
                        )
                break
            j += 1
        i = j + 1 if j > i else i + 1
    return violations


def find_server_main_migrate_calls(text: str, path: Path) -> list[str]:
    """Flag runDataMigrate* in main.go that are not under migrate/bootstrap CLI args."""
    if path.name != "main.go":
        return []
    violations: list[str] = []
    lines = text.splitlines()
    in_migrate_cli = False
    depth = 0
    migrate_cli_depth = -1
    for idx, line in enumerate(lines, 1):
        stripped = line.strip()
        if re.search(r'os\.Args\[1\]\s*==\s*"(migrate|bootstrap-[^"]+)"', line):
            in_migrate_cli = True
            migrate_cli_depth = depth
        depth += line.count("{") - line.count("}")
        if in_migrate_cli and depth <= migrate_cli_depth:
            in_migrate_cli = False
            migrate_cli_depth = -1
        if in_migrate_cli:
            continue
        if MIGRATE_CALLS.search(line) and not stripped.startswith("//"):
            # definitions elsewhere are not in main.go typically
            violations.append(
                f"{path.relative_to(ROOT)}:{idx}: server main path calls migrate — "
                "move to `migrate` CLI / 9999 init"
            )
    return violations


def main() -> int:
    violations: list[str] = []
    for path in sorted(_iter_go_files()):
        text = path.read_text(encoding="utf-8", errors="replace")
        violations.extend(find_open_func_migrate_calls(text, path))
        violations.extend(find_server_main_migrate_calls(text, path))

    if violations:
        print("FAIL: business process must not run DB migrations on startup\n", file=sys.stderr)
        for v in violations:
            print(f"  {v}", file=sys.stderr)
        print(
            "\nSee .ai/01_project_constraints/40_app_process_independent_of_db_migrate.md",
            file=sys.stderr,
        )
        return 1
    print("PASS: no startup migrate calls in openDB/OpenDB/openBudgetDB or server main")
    return 0


if __name__ == "__main__":
    sys.exit(main())

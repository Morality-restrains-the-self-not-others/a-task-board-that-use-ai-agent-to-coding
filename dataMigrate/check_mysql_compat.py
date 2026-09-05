#!/usr/bin/env python3
"""
MySQL compatibility checker for dataMigrate/ SQL files.

Scans all .sql files under dataMigrate/ for SQLite-specific syntax that
is incompatible with MySQL. Designed for pre-commit/CI integration.

Usage:
    python3 dataMigrate/check_mysql_compat.py           # scan all
    python3 dataMigrate/check_mysql_compat.py --fix      # dry run (reports only)
    python3 dataMigrate/check_mysql_compat.py path/...   # scan specific paths

Exit code: 0 = clean, 1 = issues found.

Related: OPT-20260727-027, .learnings/OPT-20260727-027.md
"""
import argparse
import re
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
DATAMIGRATE_DIR = REPO_ROOT / "dataMigrate"

# Patterns that are SQLite-only and invalid in MySQL.
# Each tuple: (regex, severity, description)
SQLITE_PATTERNS = [
    (re.compile(r'CREATE\s+INDEX\s+IF\s+NOT\s+EXISTS', re.IGNORECASE),
     'error', 'CREATE INDEX IF NOT EXISTS — MySQL does not support IF NOT EXISTS on CREATE INDEX'),
    (re.compile(r'\bATTACH\s+DATABASE\b', re.IGNORECASE),
     'error', 'ATTACH DATABASE — MySQL has no equivalent'),
    (re.compile(r'\bDETACH\s+DATABASE\b', re.IGNORECASE),
     'error', 'DETACH DATABASE — MySQL has no equivalent'),
    (re.compile(r'\bTEXT\s+PRIMARY\s+KEY\b', re.IGNORECASE),
     'error', 'TEXT PRIMARY KEY — MySQL InnoDB requires prefix length for TEXT index'),
    # DATETIME(fsp) 列类型（MySQL 合法，fsp∈0..6）不是 SQLite datetime() 函数调用；
    # 负向前瞻排除小整数参数，保留 datetime('now') / datetime(created_at) 等真调用（OPT-20260811-080 顺带）。
    (re.compile(r'\bdatetime\s*\(\s*(?!\d{1,2}\s*\))', re.IGNORECASE),
     'error', 'datetime() function — SQLite-only; use NOW() or DATE_ADD in MySQL'),
    (re.compile(r'\bstrftime\s*\(', re.IGNORECASE),
     'error', 'strftime() function — SQLite-only; use DATE_FORMAT() in MySQL'),
    (re.compile(r"\|\|\s*'", re.IGNORECASE),
     'warn', '|| string concatenation — SQLite-only; use CONCAT() in MySQL (check context)'),
    (re.compile(r'julianday\s*\(', re.IGNORECASE),
     'error', 'julianday() function — SQLite-only'),
    (re.compile(r'\bAUTOINCREMENT\b', re.IGNORECASE),
     'warn', 'AUTOINCREMENT keyword — MySQL uses AUTO_INCREMENT (underscore)'),
    (re.compile(r'PRAGMA\s+\w+', re.IGNORECASE),
     'warn', 'PRAGMA statement — SQLite-only'),
]


def check_file(filepath: Path) -> list[dict]:
    """Check a single SQL file for MySQL incompatibilities."""
    issues = []
    try:
        content = filepath.read_text(encoding='utf-8')
    except Exception as e:
        issues.append({
            'file': str(filepath.relative_to(REPO_ROOT)),
            'line': 0,
            'severity': 'error',
            'message': f'Cannot read file: {e}',
        })
        return issues

    return check_sql_text(content, filepath.relative_to(REPO_ROOT))


def check_sql_text(content: str, rel_path) -> list[dict]:
    """Check raw SQL text for MySQL incompatibilities (testable without temp files)."""
    issues = []
    lines = content.split('\n')
    for lineno, line in enumerate(lines, 1):
        stripped = line.strip()
        if stripped.startswith('--'):
            continue  # skip comment lines

        for pattern, severity, description in SQLITE_PATTERNS:
            if pattern.search(stripped):
                # For || operator, skip if line already has CONCAT (false positive)
                if "|| '" in description and 'CONCAT' in stripped.upper():
                    continue
                issues.append({
                    'file': str(rel_path),
                    'line': lineno,
                    'severity': severity,
                    'message': f'{description}: {stripped[:120]}',
                })

    return issues


def find_sql_files(root: Path) -> list[Path]:
    """Find all .sql files under the given directory."""
    return sorted(root.rglob('*.sql'))


def main():
    parser = argparse.ArgumentParser(description='Check SQLite→MySQL compatibility')
    parser.add_argument('paths', nargs='*', help='Specific files/directories to scan')
    parser.add_argument('--fix', action='store_true', help='Remove IF NOT EXISTS from CREATE INDEX statements')
    args = parser.parse_args()

    if args.paths:
        files = []
        for p in args.paths:
            path = Path(p)
            if path.is_dir():
                files.extend(find_sql_files(path))
            elif path.is_file():
                files.append(path)
    else:
        if not DATAMIGRATE_DIR.is_dir():
            print(f"[check_mysql_compat] dataMigrate/ not found — nothing to check")
            sys.exit(0)
        files = find_sql_files(DATAMIGRATE_DIR)

    all_issues = []
    for f in files:
        all_issues.extend(check_file(f))

    # --fix: remove IF NOT EXISTS from CREATE INDEX statements
    if args.fix:
        import re as fix_re
        idx_if_not_exists_re = fix_re.compile(
            r'^(CREATE\s+(?:UNIQUE\s+)?INDEX)\s+IF\s+NOT\s+EXISTS\s+',
            fix_re.IGNORECASE
        )
        fixed_files = 0
        fixed_count = 0
        for f in files:
            try:
                content = f.read_text(encoding='utf-8')
            except Exception:
                continue
            new_lines = []
            file_modified = False
            for line in content.split('\n'):
                m = idx_if_not_exists_re.match(line.strip())
                if m:
                    new_line = line.replace(' IF NOT EXISTS', '', 1)
                    new_lines.append(new_line)
                    file_modified = True
                    fixed_count += 1
                else:
                    new_lines.append(line)
            if file_modified:
                f.write_text('\n'.join(new_lines), encoding='utf-8')
                relative = str(f.relative_to(REPO_ROOT))
                print(f"  Fixed {fixed_count} indexes in: {relative}")
                fixed_files += 1
                fixed_count = 0  # reset per-file counter for display
        total_fixed = sum(1 for issue in all_issues if 'CREATE INDEX IF NOT EXISTS' in issue['message'])
        print(f"\n[check_mysql_compat] --fix: removed IF NOT EXISTS in {fixed_files} files")
        # Re-scan after fix
        return main_wrapper()

    if not all_issues:
        print(f"[check_mysql_compat] ✅ {len(files)} SQL files — no issues found")
        sys.exit(0)

    errors = [i for i in all_issues if i['severity'] == 'error']
    warns = [i for i in all_issues if i['severity'] == 'warn']

    for issue in all_issues:
        tag = '❌' if issue['severity'] == 'error' else '⚠️ '
        print(f"{tag} {issue['file']}:{issue['line']} — {issue['message']}")

    print(f"\n[check_mysql_compat] {len(errors)} error(s), {len(warns)} warning(s) in {len(files)} files")

    # Fail on errors; warnings alone don't fail.
    sys.exit(1 if errors else 0)


def main_wrapper():
    """Re-run main() for --fix rescan."""
    # Save and restore sys.argv
    saved_argv = sys.argv[:]
    sys.argv = [a for a in sys.argv if a != '--fix']
    # Re-parse
    main()


if __name__ == '__main__':
    main()

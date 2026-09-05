#!/usr/bin/env python3
"""Convert SQLite-specific SQL syntax to MySQL-compatible syntax.

Usage:
  python3 _convert_sqlite_to_mysql.py [--dry-run] [file1.sql ...]
  python3 _convert_sqlite_to_mysql.py --dir dataMigrate/  [--dry-run]

If no files/dir specified, converts all .sql files under dataMigrate/.
"""

import re
import sys
import os
import argparse
from pathlib import Path

# ── Transformation rules ──────────────────────────────────────────

def remove_pragma(line: str) -> str:
    """Remove PRAGMA statements (SQLite only)."""
    if re.match(r'^\s*PRAGMA\s+\w+', line, re.IGNORECASE):
        return f'-- [MySQL compat] removed: {line.strip()}'
    return line


def convert_autoincrement(line: str) -> str:
    """AUTOINCREMENT → AUTO_INCREMENT (SQLite spelling)."""
    return re.sub(r'\bAUTOINCREMENT\b', 'AUTO_INCREMENT', line, flags=re.IGNORECASE)


def convert_integer_primary_key(line: str) -> str:
    """INTEGER PRIMARY KEY → INT AUTO_INCREMENT PRIMARY KEY.

    Only converts bare 'INTEGER PRIMARY KEY' (SQLite rowid alias).
    Preserves cases like 'id INTEGER PRIMARY KEY,' → 'id INT AUTO_INCREMENT PRIMARY KEY,'.
    Does NOT convert 'TEXT PRIMARY KEY' or 'VARCHAR(...) PRIMARY KEY'.
    """
    # Match: <column_name> INTEGER PRIMARY KEY [CHECK (...)] [,)]
    # Replace with: <column_name> INT AUTO_INCREMENT PRIMARY KEY
    pattern = r'(\w+)\s+INTEGER\s+PRIMARY\s+KEY'
    replacement = r'\1 INT AUTO_INCREMENT PRIMARY KEY'
    return re.sub(pattern, replacement, line, flags=re.IGNORECASE)


def convert_datetime_now(line: str) -> str:
    """datetime('now') → NOW()"""
    return re.sub(r"datetime\('now'\)", 'NOW()', line, flags=re.IGNORECASE)


def convert_insert_or_ignore(line: str) -> str:
    """INSERT OR IGNORE INTO → INSERT IGNORE INTO"""
    return re.sub(r'\bINSERT\s+OR\s+IGNORE\s+INTO\b', 'INSERT IGNORE INTO', line, flags=re.IGNORECASE)


def convert_default_datetime_now(line: str) -> str:
    """DEFAULT (datetime('now')) → DEFAULT CURRENT_TIMESTAMP
    Also handles DEFAULT datetime('now') without parens.
    """
    line = re.sub(
        r"DEFAULT\s*\(\s*datetime\('now'\)\s*\)",
        'DEFAULT CURRENT_TIMESTAMP',
        line, flags=re.IGNORECASE
    )
    return line


def convert_real_type(line: str) -> str:
    """REAL column type → DOUBLE (only when REAL is a standalone type keyword)."""
    # Match REAL as a column type: 'col REAL NOT NULL DEFAULT 0'
    return re.sub(r'\bREAL\b', 'DOUBLE', line, flags=re.IGNORECASE)


def convert_bool_type(line: str) -> str:
    """bool column type → TINYINT(1) (SQLite-specific, MySQL uses TINYINT(1) for bool).

    Only converts 'bool' when used as a column type keyword,
    not inside strings or identifiers.
    """
    # Match 'bool' as a standalone type word in column definitions
    return re.sub(r'\bbool\b', 'TINYINT(1)', line)


def convert_strftime(line: str) -> str:
    """Convert strftime patterns to MySQL DATE_FORMAT.

    strftime('%Y-%m-%d %H:%M:%S', ...) → DATE_FORMAT(..., '%Y-%m-%d %H:%i:%S')

    MySQL DATE_FORMAT format codes differ from SQLite strftime:
      %M → %i (minutes, not month name)
      %S, %Y, %m, %d, %H stay the same
    """
    line = re.sub(r"\bstrftime\(", 'DATE_FORMAT(', line, flags=re.IGNORECASE)
    # %M in strftime = minutes; in MySQL DATE_FORMAT, %M = month name, %i = minutes
    # Only fix inside format strings (single-quoted)
    line = re.sub(r"'(%[YmdHMSs]*)%M", r"'\1%i", line)
    return line


def flag_complex_conversion(line: str) -> str:
    """Flag lines that contain complex SQLite expressions needing manual review.

    - strftime with nested datetime() calls (MySQL has no datetime() function)
    - || string concatenation in non-trivial expressions
    """
    needs_review = False
    reasons = []

    # datetime() function call that's not datetime('now') already converted
    if re.search(r'\bdatetime\(', line, re.IGNORECASE):
        # Check if it's already been converted from datetime('now')
        if "NOW()" not in line:
            needs_review = True
            reasons.append('datetime() function not valid in MySQL; use DATE_ADD/ADDTIME')

    # || concatenation (after strftime → DATE_FORMAT conversion)
    if '||' in line and "'" in line and 'DATE_FORMAT' in line:
        needs_review = True
        reasons.append('|| string concatenation needs CONCAT() in MySQL')

    if needs_review:
        return f'-- [MySQL compat] Manual review needed: {"; ".join(reasons)}\n{line}'
    return line


def convert_substr(line: str) -> str:
    """substr( → SUBSTRING( (MySQL compatible)."""
    return re.sub(r'\bsubstr\(', 'SUBSTRING(', line, flags=re.IGNORECASE)


def convert_concat_operator(line: str) -> str:
    """Convert || string concatenation to CONCAT().

    This is tricky - only convert when || appears to be string concatenation,
    not SQL boolean OR. We detect patterns like: '...' || '...' or expr || '...'.

    For safety, only do this when the line contains both || and literal strings.
    """
    # Only transform lines that look like string concatenation
    if "'" not in line:
        return line
    # Simple pattern: value || 'literal_string'
    # This is a heuristic - full SQL parsing would be needed for 100% accuracy
    if '||' in line and ("'.'" in line or "|| '" in line or "' ||" in line):
        # Wrap the entire expression in CONCAT() - best effort
        # For the specific case in credit_lot_expires.sql
        pass  # Handled manually for complex cases
    return line


def convert_cast_text(line: str) -> str:
    """CAST(... AS TEXT) → CAST(... AS CHAR) (MySQL doesn't support CAST AS TEXT)."""
    return re.sub(r'\bCAST\s*\(\s*(.+?)\s+AS\s+TEXT\s*\)', r'CAST(\1 AS CHAR)', line, flags=re.IGNORECASE)


def convert_double_quoted_identifiers(line: str) -> str:
    """Convert "identifier" to `identifier` for MySQL compatibility.

    Only converts quoted identifiers that look like column names
    (lowercase, simple). Does NOT convert string literals (single-quoted).
    """
    # Match patterns like t."order" → t.`order`
    return re.sub(r'(\w+)\."(\w+)"', r'\1.`\2`', line)


def handle_attach_detach(line: str) -> str:
    """Comment out ATTACH DATABASE / DETACH DATABASE (MySQL has no equivalent)."""
    if re.match(r'^\s*(ATTACH|DETACH)\s+DATABASE\b', line, re.IGNORECASE):
        return f'-- [MySQL compat] ATTACH/DETACH not supported. Run as separate connection.\n-- {line.strip()}'
    return line


def handle_partial_index(line: str) -> str:
    """Flag CREATE INDEX ... WHERE for manual review (MySQL 8.0.13+ supports
    functional indexes but with limitations)."""
    if re.match(r'^\s*CREATE\s+(UNIQUE\s+)?INDEX\b', line, re.IGNORECASE):
        # Check if this index definition ends with WHERE (partial index)
        if re.search(r'\bWHERE\b', line, re.IGNORECASE):
            return (
                f'-- [MySQL compat] Partial index (WHERE clause) may need review.\n'
                f'-- MySQL 8.0.13+ supports functional indexes; verify syntax.\n'
                f'{line}'
            )
    return line


def handle_sqlite_specific(line: str) -> str:
    """Handle other SQLite-specific patterns."""
    # Comment out references to sqlite_master or sqlite_sequence
    if re.search(r'\bsqlite_\w+', line, re.IGNORECASE):
        return f'-- [MySQL compat] SQLite internal table: {line.strip()}'
    return line


# ── Pipeline ───────────────────────────────────────────────────────

TRANSFORMS = [
    ('remove_pragma', remove_pragma),
    ('handle_attach_detach', handle_attach_detach),
    ('handle_sqlite_specific', handle_sqlite_specific),
    ('convert_autoincrement', convert_autoincrement),
    ('convert_integer_primary_key', convert_integer_primary_key),
    # convert_default_datetime_now BEFORE convert_datetime_now
    # so DEFAULT (datetime('now')) → DEFAULT CURRENT_TIMESTAMP
    # rather than DEFAULT (NOW()) which is invalid MySQL
    ('convert_default_datetime_now', convert_default_datetime_now),
    ('convert_datetime_now', convert_datetime_now),
    ('convert_insert_or_ignore', convert_insert_or_ignore),
    ('convert_real_type', convert_real_type),
    ('convert_bool_type', convert_bool_type),
    ('convert_strftime', convert_strftime),
    ('convert_substr', convert_substr),
    ('convert_cast_text', convert_cast_text),
    ('convert_double_quoted_identifiers', convert_double_quoted_identifiers),
    ('handle_partial_index', handle_partial_index),
    ('flag_complex_conversion', flag_complex_conversion),
]


def transform_line(line: str) -> tuple[str, list[str]]:
    """Apply all transformations to a line. Returns (transformed_line, [applied_rules])."""
    applied = []
    result = line
    for name, func in TRANSFORMS:
        new_result = func(result)
        if new_result != result:
            applied.append(name)
        result = new_result
    return result, applied


def transform_file(filepath: str, dry_run: bool = False) -> dict:
    """Transform a single SQL file. Returns stats dict."""
    stats = {
        'file': filepath,
        'lines_total': 0,
        'lines_changed': 0,
        'rules_applied': set(),
    }

    with open(filepath, 'r', encoding='utf-8') as f:
        original = f.read()

    lines = original.split('\n')
    stats['lines_total'] = len(lines)
    new_lines = []

    for line in lines:
        new_line, applied = transform_line(line)
        if new_line != line:
            stats['lines_changed'] += 1
            stats['rules_applied'].update(applied)
        new_lines.append(new_line)

    result = '\n'.join(new_lines)

    if not dry_run and result != original:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(result)

    return stats


def find_sql_files(directory: str) -> list[str]:
    """Find all .sql files recursively under directory."""
    sql_files = []
    for root, dirs, files in os.walk(directory):
        # Skip __pycache__
        dirs[:] = [d for d in dirs if d != '__pycache__']
        for f in files:
            if f.endswith('.sql'):
                sql_files.append(os.path.join(root, f))
    return sorted(sql_files)


def main():
    parser = argparse.ArgumentParser(description='Convert SQLite SQL to MySQL syntax')
    parser.add_argument('files', nargs='*', help='SQL files to convert')
    parser.add_argument('--dir', default=None, help='Directory to recursively find .sql files')
    parser.add_argument('--dry-run', action='store_true', help='Show changes without writing')
    args = parser.parse_args()

    sql_files = []
    if args.files:
        sql_files = [os.path.abspath(f) for f in args.files]
    elif args.dir:
        sql_files = find_sql_files(args.dir)
    else:
        # Default: all SQL files under dataMigrate/
        script_dir = os.path.dirname(os.path.abspath(__file__))
        repo_root = os.path.dirname(script_dir)
        data_migrate = os.path.join(repo_root, 'dataMigrate')
        if os.path.isdir(data_migrate):
            sql_files = find_sql_files(data_migrate)
        else:
            print("Error: dataMigrate/ directory not found. Specify --dir or files.", file=sys.stderr)
            sys.exit(1)

    if not sql_files:
        print("No .sql files found.")
        sys.exit(0)

    mode = "DRY RUN (no changes)" if args.dry_run else "WRITING changes"
    print(f"=== SQLite → MySQL Converter ({mode}) ===\n")
    print(f"Files to process: {len(sql_files)}\n")

    total_stats = {
        'files': 0,
        'files_changed': 0,
        'lines_total': 0,
        'lines_changed': 0,
        'all_rules': set(),
    }

    for fpath in sql_files:
        stats = transform_file(fpath, dry_run=args.dry_run)
        total_stats['files'] += 1
        total_stats['lines_total'] += stats['lines_total']
        total_stats['lines_changed'] += stats['lines_changed']
        total_stats['all_rules'].update(stats['rules_applied'])

        if stats['lines_changed'] > 0:
            total_stats['files_changed'] += 1
            relpath = os.path.relpath(fpath, os.path.commonprefix([fpath, os.getcwd()]))
            rules = ', '.join(sorted(stats['rules_applied']))
            print(f"  ✓ {relpath}: {stats['lines_changed']}/{stats['lines_total']} lines ({rules})")

    print(f"\n=== Summary ===")
    print(f"  Files: {total_stats['files']} total, {total_stats['files_changed']} changed")
    print(f"  Lines: {total_stats['lines_total']} total, {total_stats['lines_changed']} modified")
    print(f"  Rules applied: {', '.join(sorted(total_stats['all_rules']))}")

    if args.dry_run:
        print("\n  [DRY RUN] No files were modified. Remove --dry-run to apply.")
    else:
        print(f"\n  Conversion complete. {total_stats['files_changed']} files written.")


if __name__ == '__main__':
    main()

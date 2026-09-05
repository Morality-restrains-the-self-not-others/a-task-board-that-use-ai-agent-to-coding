#!/usr/bin/env python3
"""Add DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci to CREATE TABLE statements missing CHARSET.

Usage: python3 fix_missing_charset.py [--dry-run]
"""

import os
import re
import sys

ROOT = os.path.dirname(os.path.abspath(__file__))
CHARSET_CLAUSE = ' ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci'

def has_charset(statement):
    """Check if a CREATE TABLE statement already has a CHARSET/CHARACTER SET clause."""
    upper = statement.upper()
    return 'CHARSET' in upper or 'CHARACTER SET' in upper

def fix_create_table(sql):
    """Add CHARSET clause to CREATE TABLE statements that are missing it."""
    # Match CREATE TABLE [IF NOT EXISTS] `table_name` (...) ... ;
    # We need to find the closing ); of the column/index definitions
    pattern = re.compile(
        r'(CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?`?\w+`?\s*\()',
        re.IGNORECASE
    )

    lines = sql.split('\n')
    result = []
    i = 0
    modified = False

    while i < len(lines):
        line = lines[i]
        m = pattern.search(line)
        if not m:
            result.append(line)
            i += 1
            continue

        # Found a CREATE TABLE - collect the full statement
        stmt_lines = [line]
        paren_depth = 0
        j = i
        # Count open parens
        for ch in line[m.end()-1:]:  # the opening ( is at m.end()-1
            if ch == '(':
                paren_depth += 1
            elif ch == ')':
                paren_depth -= 1

        while paren_depth > 0 and j + 1 < len(lines):
            j += 1
            stmt_lines.append(lines[j])
            for ch in lines[j]:
                if ch == '(':
                    paren_depth += 1
                elif ch == ')':
                    paren_depth -= 1

        full_stmt = '\n'.join(stmt_lines)

        if not has_charset(full_stmt):
            # Replace the last ); with CHARSET clause
            # Find the last ');' or just ';' in the statement
            # The statement ends with );  (closing paren then semicolon)
            # But there might be whitespace and maybe ENGINE before the ;
            last_line = stmt_lines[-1]
            # Check if the closing ); is on the last line
            if re.search(r'\)\s*;', last_line):
                new_last = re.sub(r'\)(\s*);', r')' + CHARSET_CLAUSE + r'\1;', last_line, count=1)
                stmt_lines[-1] = new_last
                result.extend(stmt_lines)
                modified = True
            else:
                # The ); is split across lines, handle edge case
                result.extend(stmt_lines)
        else:
            result.extend(stmt_lines)

        i = j + 1

    return '\n'.join(result), modified

def process_file(filepath, dry_run=False):
    with open(filepath, 'r') as f:
        original = f.read()

    fixed, modified = fix_create_table(original)

    if not modified:
        return False

    if dry_run:
        print(f"  Would fix: {filepath}")
        return True

    with open(filepath, 'w') as f:
        f.write(fixed)
    print(f"  Fixed: {filepath}")
    return True

def main():
    dry_run = '--dry-run' in sys.argv
    total = 0

    for service in sorted(os.listdir(ROOT)):
        svc_dir = os.path.join(ROOT, service)
        if not os.path.isdir(svc_dir):
            continue
        for fname in sorted(os.listdir(svc_dir)):
            if not fname.endswith('.sql'):
                continue
            fpath = os.path.join(svc_dir, fname)
            if process_file(fpath, dry_run=dry_run):
                total += 1

    if dry_run:
        print(f"\nWould fix {total} files.")
    else:
        print(f"\nFixed {total} files.")
    return 0

if __name__ == '__main__':
    sys.exit(main())

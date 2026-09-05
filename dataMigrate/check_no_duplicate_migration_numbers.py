#!/usr/bin/env python3
"""CI check: no duplicate numeric prefixes in any dataMigrate service directory.

Each SQL file in dataMigrate/<service>/ must have a unique 3-digit numeric prefix.
This is part of the human-readable ordering contract — duplicate prefixes cause
ambiguity during review, operations, and debugging.

Usage: python3 check_no_duplicate_migration_numbers.py
Exit 0 if all prefixes unique, exit 1 with conflict report otherwise.
"""

import os
import sys
from collections import defaultdict

ROOT = os.path.dirname(os.path.abspath(__file__))

def find_conflicts():
    conflicts = {}
    for service in sorted(os.listdir(ROOT)):
        svc_dir = os.path.join(ROOT, service)
        if not os.path.isdir(svc_dir):
            continue
        sql_files = [f for f in os.listdir(svc_dir) if f.endswith('.sql')]
        if not sql_files:
            continue
        prefix_map = defaultdict(list)
        for f in sorted(sql_files):
            # Extract 3-digit prefix: "NNN_..." or "NNN-..."
            prefix = f[:3]
            if prefix.isdigit():
                prefix_map[prefix].append(f)
        dups = {p: files for p, files in prefix_map.items() if len(files) > 1}
        if dups:
            conflicts[service] = dups
    return conflicts

def main():
    conflicts = find_conflicts()
    if not conflicts:
        print("OK: All dataMigrate migration prefixes are unique.")
        return 0
    print("ERROR: Duplicate numeric prefixes found in dataMigrate directories:")
    for service, dups in sorted(conflicts.items()):
        for prefix, files in sorted(dups.items()):
            print(f"  dataMigrate/{service}/: prefix {prefix} used by:")
            for f in files:
                print(f"    - {f}")
    print("\nAction: Rename files so each has a unique 3-digit prefix (preserving relative order).")
    print("Then run fix_renumber_data_migrate_log.sql against deployed databases.")
    return 1

if __name__ == '__main__':
    sys.exit(main())

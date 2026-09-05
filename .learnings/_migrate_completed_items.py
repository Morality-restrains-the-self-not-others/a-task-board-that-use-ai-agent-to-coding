#!/usr/bin/env python3
"""
Migrate completed [x] checklist items from OPTIMIZATION_TODOS.md to OPTIMIZATION_TODOS_COMPLETED.md.

Handles:
- ID collisions between TODOS and COMPLETED (different items, same number)
- Duplicate IDs within TODOS file
- Proper metadata extraction
"""

import re
import sys
from datetime import datetime, timezone, timedelta
from pathlib import Path

TZ = timezone(timedelta(hours=8))
LEARNINGS = Path('/tmp/ram-work/.learnings')
TODOS = LEARNINGS / 'OPTIMIZATION_TODOS.md'
COMPLETED = LEARNINGS / 'OPTIMIZATION_TODOS_COMPLETED.md'


def extract_items(content):
    """Parse line by line, return (new_content, [(opt_id, block_lines), ...])."""
    lines = content.split('\n')
    out_lines = []
    completed_items = []

    i = 0
    while i < len(lines):
        line = lines[i]

        if line.startswith('- [x] '):
            opt_match = re.search(r'\*\*(OPT-\d{8}-\d+)\*\*', line)
            opt_id = opt_match.group(1) if opt_match else f'UNKNOWN-{i}'

            block_lines = [line]
            i += 1
            while i < len(lines):
                nxt = lines[i]
                if nxt.startswith('- [x] ') or nxt.startswith('- [ ] '):
                    break
                if nxt.startswith('## ') or nxt.startswith('---'):
                    break
                block_lines.append(nxt)
                i += 1

            completed_items.append((opt_id, block_lines))

        elif line.startswith('- [ ] '):
            out_lines.append(line)
            i += 1
            while i < len(lines):
                nxt = lines[i]
                if nxt.startswith('- [x] ') or nxt.startswith('- [ ] '):
                    break
                if nxt.startswith('## ') or nxt.startswith('---'):
                    break
                out_lines.append(nxt)
                i += 1
            if i < len(lines):
                out_lines.append('')
        else:
            out_lines.append(line)
            i += 1

    new_content = '\n'.join(out_lines)
    new_content = re.sub(r'\n{4,}', '\n\n\n', new_content)
    return new_content, completed_items


def extract_metadata(block_text):
    """Extract metadata from checklist item block."""
    meta = {'completed_date': None, 'completion_note': None, 'area': None}

    m = re.search(r'\*\*Completed[：:]\s*(\S+)', block_text)
    if m:
        d = m.group(1).strip()
        date_m = re.match(r'(\d{4}-\d{2}-\d{2})', d)
        if date_m:
            meta['completed_date'] = date_m.group(1)

    m = re.search(r'\*\*Completion-Note[：:]\s*(.+?)\*\*', block_text)
    if m:
        meta['completion_note'] = m.group(1).strip()

    m = re.search(r'\*\*Area[：:]\s*(.+?)\*\*', block_text)
    if m:
        meta['area'] = m.group(1).strip()

    if not meta['completion_note']:
        m = re.search(r'\*\*Why[：:]\s*(.+?)\*\*', block_text)
        if m:
            meta['completion_note'] = m.group(1).strip()

    return meta


def resolve_id(opt_id, seen_in_completed, used_in_migration):
    """
    Resolve ID conflicts.
    Returns (final_id, note) where note is the reason for renaming if any.
    """
    # Check if this ID already exists in COMPLETED file (from previous proper migrations)
    if opt_id in seen_in_completed:
        # Try -b suffix
        candidate = opt_id + '-b'
        if candidate not in seen_in_completed and candidate not in used_in_migration:
            return candidate, f"ID冲突: {opt_id} 已被 COMPLETED 文件中的另一条目使用，重命名为 {candidate}"
        # Try -c, -d...
        for suffix in ['-c', '-d', '-e', '-f']:
            candidate = opt_id + suffix
            if candidate not in seen_in_completed and candidate not in used_in_migration:
                return candidate, f"ID冲突: {opt_id} 已被占用，重命名为 {candidate}"

    # Check if already used in this migration run (duplicate within TODOS)
    if opt_id in used_in_migration:
        candidate = opt_id + '-dup'
        if candidate not in seen_in_completed and candidate not in used_in_migration:
            return candidate, f"TODOS内重复ID: {opt_id} 出现多次，冲突副本重命名为 {candidate}"
        for suffix in ['-dup2', '-dup3']:
            candidate = opt_id + suffix
            if candidate not in seen_in_completed and candidate not in used_in_migration:
                return candidate, f"TODOS内重复ID: {opt_id} 出现多次，冲突副本重命名为 {candidate}"

    return opt_id, None


def format_for_completed(opt_id, block_lines, today_str, rename_note=None):
    """Format checklist block as COMPLETED file entry."""
    block_text = '\n'.join(block_lines).strip()
    meta = extract_metadata(block_text)

    completed_date = meta['completed_date'] or today_str
    completion_note = meta['completion_note'] or '从 OPTIMIZATION_TODOS.md 迁移（详见原始条目）'
    area = meta['area'] or ''

    if rename_note:
        completion_note = f"[{rename_note}] {completion_note}"

    now_ts = datetime.now(TZ).strftime('%Y-%m-%dT%H:%M:%S+08:00')

    entry = f"## [{opt_id}] completed\n\n"
    entry += f"**Logged**: {now_ts}\n"
    entry += f"**Completed**: {completed_date}\n"
    entry += f"**Completion-Note**: {completion_note}\n"
    entry += f"**Status**: completed\n"
    if area:
        entry += f"**Area**: {area}\n"
    entry += "\n"
    entry += block_text + "\n"

    return entry


def main():
    dry_run = '--dry-run' in sys.argv
    today = datetime.now(TZ).strftime('%Y-%m-%d')

    print(f"=== OPTIMIZATION_TODOS.md completed item migration ===")
    print(f"Today: {today}")
    print(f"Mode: {'DRY-RUN' if dry_run else 'LIVE'}")
    print()

    content = TODOS.read_text()

    if not dry_run:
        backup = TODOS.with_suffix('.md.bak')
        backup.write_text(content)
        print(f"✓ Backup: {backup}")

    new_content, completed_items = extract_items(content)
    print(f"Completed items to process: {len(completed_items)}")

    # Load existing COMPLETED IDs
    completed_content = COMPLETED.read_text() if COMPLETED.exists() else ""
    existing_ids = set(re.findall(r'^## \[(OPT-\d{8}-\d+(?:-[a-z]+)?)\]', completed_content, re.MULTILINE))
    print(f"Existing IDs in COMPLETED: {len(existing_ids)}")

    # Resolve all IDs
    resolved = []  # (final_id, block_lines, rename_note)
    used_in_migration = set()
    stats = {'ok': 0, 'conflict_renamed': 0, 'dup_renamed': 0}

    for opt_id, block_lines in completed_items:
        final_id, note = resolve_id(opt_id, existing_ids, used_in_migration)
        used_in_migration.add(final_id)

        if note:
            if '已被占用' in note or '已被 COMPLETED' in note:
                stats['conflict_renamed'] += 1
                print(f"  ⚠ COLLISION: {opt_id} → {final_id}")
            else:
                stats['dup_renamed'] += 1
                print(f"  ⚠ DUPLICATE: {opt_id} → {final_id}")
        else:
            stats['ok'] += 1

        resolved.append((final_id, block_lines, note))

    print(f"\nResolution: {stats['ok']} unchanged, "
          f"{stats['conflict_renamed']} cross-file collisions renamed, "
          f"{stats['dup_renamed']} within-file duplicates renamed")

    if dry_run:
        for final_id, block_lines, note in resolved:
            first_line = block_lines[0][:120]
            tag = f" [{note.split(':', 1)[0]}]" if note else ""
            print(f"  {final_id}{tag}: {first_line}...")

        print(f"\n[Dry-run] Would move {len(resolved)} items.")
        print(f"[Dry-run] New TODOS: {len(new_content)} chars.")
        print(f"[Dry-run] No changes made. Run without --dry-run to execute.")
        return

    # Write cleaned TODOS
    TODOS.write_text(new_content)
    print(f"\n✓ Updated {TODOS}")

    # Append to COMPLETED
    if not completed_content.endswith('\n'):
        completed_content += '\n'

    added = 0
    for final_id, block_lines, note in resolved:
        entry = format_for_completed(final_id, block_lines, today, note)
        completed_content += '\n' + entry
        added += 1

    COMPLETED.write_text(completed_content)
    print(f"✓ Appended {added} entries to {COMPLETED}")

    print(f"\n=== Migration complete ===")
    print(f"Moved: {len(resolved)} items")
    print(f"  - {stats['ok']} with original ID")
    print(f"  - {stats['conflict_renamed']} renamed (COMPLETED collision)")
    print(f"  - {stats['dup_renamed']} renamed (TODOS internal duplicate)")


if __name__ == '__main__':
    main()

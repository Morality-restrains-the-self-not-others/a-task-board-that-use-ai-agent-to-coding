#!/usr/bin/env python3
"""
将 OPTIMIZATION_TODOS_COMPLETED.md 中非当日的条目按天归档到
archive/completed/OPT_COMPLETED_YYYY-MM-DD.md。

用法:
  python .learnings/_archive_completed_opts.py --dry-run   # 仅预览，不实际修改
  python .learnings/_archive_completed_opts.py              # 执行归档
"""

import os
import re
import sys
from datetime import datetime, timezone, timedelta
from collections import defaultdict

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
LEARNINGS_DIR = SCRIPT_DIR
COMPLETED_FILE = os.path.join(LEARNINGS_DIR, "OPTIMIZATION_TODOS_COMPLETED.md")
ARCHIVE_DIR = os.path.join(LEARNINGS_DIR, "archive", "completed")

# 上海时区（简化：UTC+8）
TZ_SH = timezone(timedelta(hours=8))


def parse_date(s: str):
    """解析 ISO-8601 日期/日期时间字符串，返回 date 对象。"""
    s = s.strip()
    try:
        if "T" in s:
            dt = datetime.fromisoformat(s)
            return dt.date()
        return datetime.strptime(s[:10], "%Y-%m-%d").date()
    except Exception:
        return None


def extract_entry_date(entry_text: str):
    """
    从条目文本中提取归档判定日期。
    优先 **Completed**: 其次 **Logged**: 否则 None。
    """
    m = re.search(r"\*\*Completed\*\*:\s*(\S+)", entry_text)
    if m:
        d = parse_date(m.group(1))
        if d:
            return d
    m = re.search(r"\*\*Logged\*\*:\s*(\S+)", entry_text)
    if m:
        d = parse_date(m.group(1))
        if d:
            return d
    m = re.search(r"## \[OPT-(\d{8})-", entry_text)
    if m:
        try:
            return datetime.strptime(m.group(1), "%Y%m%d").date()
        except Exception:
            pass
    return None


def get_day_key(d):
    """返回 YYYY-MM-DD 字符串。"""
    return d.strftime("%Y-%m-%d")


def split_entries(content: str):
    """
    将文件内容拆分为：(header, entries_list)
    header: 第一个 ## [OPT- 之前的所有内容
    entries_list: [(entry_text, start_pos), ...]
    """
    first_entry = re.search(r"^## \[OPT-", content, re.MULTILINE)
    if not first_entry:
        return content, []

    header = content[:first_entry.start()]
    body = content[first_entry.start():]

    entries = []
    for m in re.finditer(r"^## \[OPT-", body, re.MULTILINE):
        start = m.start()
        if entries:
            prev_start, _ = entries[-1]
            entries[-1] = (body[prev_start:start], prev_start)
        entries.append((start, start))
    if entries:
        prev_start, _ = entries[-1]
        entries[-1] = (body[prev_start:], prev_start)

    return header, entries


def write_daily_archive(day_key: str, entries: list):
    """将条目列表追加入按天归档文件。跳过已存在的编号。"""
    archive_path = os.path.join(ARCHIVE_DIR, f"OPT_COMPLETED_{day_key}.md")

    # 宽松正则：匹配标准编号 OPT-YYYYMMDD-NNN 及带后缀的变体（如 -refund-provider）
    OPT_ID_RE = r"## \[(OPT-[\w-]+)\]"

    existing_ids = set()
    if os.path.exists(archive_path):
        with open(archive_path) as f:
            existing = f.read()
        existing_ids = set(re.findall(OPT_ID_RE, existing))

    new_entries = []
    for e in entries:
        m = re.search(OPT_ID_RE, e)
        if m and m.group(1) not in existing_ids:
            new_entries.append(e)

    if not new_entries:
        return 0

    os.makedirs(ARCHIVE_DIR, exist_ok=True)

    if not os.path.exists(archive_path):
        header = f"""# Completed OPT Archive — {day_key}

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 __COUNT__ 条。
> 归档执行时间：{datetime.now(TZ_SH).strftime('%Y-%m-%dT%H:%M:%S+08:00')}

"""
        with open(archive_path, "w") as f:
            f.write(header)
        with open(archive_path, "a") as f:
            for e in new_entries:
                f.write(e.rstrip() + "\n\n")
        with open(archive_path) as f:
            content = f.read()
        content = content.replace("__COUNT__", str(len(new_entries)))
        with open(archive_path, "w") as f:
            f.write(content)
    else:
        with open(archive_path, "a") as f:
            f.write("\n")
            for e in new_entries:
                f.write(e.rstrip() + "\n\n")
        with open(archive_path) as f:
            content = f.read()
        total = len(re.findall(r"^## \[OPT-", content, re.MULTILINE))
        content = re.sub(r"共 \d+ 条", f"共 {total} 条", content)
        with open(archive_path, "w") as f:
            f.write(content)

    return len(new_entries)


def rebuild_completed_file(header: str, retained_entries: list, archive_map: dict):
    """重建主文件，仅保留当日条目并更新归档索引。"""
    index_lines = []
    for day_key in sorted(archive_map.keys(), reverse=True):
        count = archive_map[day_key]
        path = f"./archive/completed/OPT_COMPLETED_{day_key}.md"
        index_lines.append(
            f"<!-- {day_key}: {count} 条 → [{path}]({path}) -->"
        )

    existing = header
    if "<!-- 归档索引" in existing:
        existing = re.sub(
            r"<!-- 归档索引：.*?(?=\n\n|\n##|\Z)",
            "",
            existing,
            flags=re.DOTALL,
        )
        existing = re.sub(r"\n{3,}", "\n\n", existing)

    if index_lines:
        index_block = (
            "<!-- 归档索引：非当日的条目已按天归档至 archive/completed/ -->\n"
            + "\n".join(index_lines)
            + "\n\n"
        )
    else:
        index_block = ""

    lines = existing.splitlines()
    insert_after = 0
    for i, line in enumerate(lines):
        if line.startswith("行为约束见"):
            insert_after = i + 1
            break

    result_lines = lines[:insert_after + 1]
    while insert_after + 1 < len(lines) and lines[insert_after + 1].strip() == "":
        insert_after += 1

    if index_block:
        result_lines.append("")
        result_lines.extend(index_block.strip().splitlines())
        result_lines.append("")

    result_lines.extend(lines[insert_after + 1:])

    if insert_after == 0 and index_block:
        result_lines = lines[:3] + [""] + index_block.strip().splitlines() + [""] + lines[3:]

    result = "\n".join(result_lines)
    result = re.sub(r"\n{3,}", "\n\n", result)

    result = result.rstrip() + "\n"
    for e in retained_entries:
        result += "\n" + e.rstrip() + "\n"

    return result


def main():
    dry_run = "--dry-run" in sys.argv

    if not os.path.exists(COMPLETED_FILE):
        print(f"错误：文件不存在 {COMPLETED_FILE}")
        sys.exit(1)

    with open(COMPLETED_FILE) as f:
        content = f.read()

    header, entries = split_entries(content)
    today = datetime.now(TZ_SH).date()

    print(f"今日日期：{today}")
    print(f"留存规则：仅保留 Completed 日期 = {today} 的条目")
    print(f"文件条目总数：{len(entries)}")
    print()

    to_archive = defaultdict(list)
    retained = []
    stats = {"archived": 0, "retained": 0, "no_date": 0}

    for entry_text, _ in entries:
        if not re.match(r"^## \[OPT-", entry_text):
            header += entry_text
            continue

        date = extract_entry_date(entry_text)
        if date is None:
            retained.append(entry_text)
            stats["no_date"] += 1
            print(f"  ⚠ 无法判定日期，保留：{entry_text.split(chr(10))[0]}")
            continue

        if date != today:
            day_key = get_day_key(date)
            to_archive[day_key].append(entry_text)
            stats["archived"] += 1
            opt_id = re.search(r"## \[(OPT-[\w-]+)\]", entry_text)
            opt_str = opt_id.group(1) if opt_id else "?"
            print(f"  → 归档 [{day_key}] {opt_str}")
        else:
            retained.append(entry_text)
            stats["retained"] += 1

    print()
    print(f"统计：归档 {stats['archived']} | 保留 {stats['retained']} | 无日期 {stats['no_date']}")

    if stats["archived"] == 0:
        print("没有需要归档的条目（全部为当日）。")
        return

    if dry_run:
        print("\n[Dry-run] 以下日期文件将被创建/更新：")
        for day_key, entry_list in sorted(to_archive.items()):
            print(f"  {day_key}: {len(entry_list)} 条")
        print("\n[Dry-run] 未实际修改文件。使用不带 --dry-run 执行。")
        return

    # 写入按天归档文件
    archive_map = {}
    for day_key, entry_list in sorted(to_archive.items()):
        count = write_daily_archive(day_key, entry_list)
        archive_map[day_key] = count
        print(f"已写入 {day_key}: {count} 条")

    # 收集已有的归档信息
    if os.path.exists(ARCHIVE_DIR):
        for fname in sorted(os.listdir(ARCHIVE_DIR)):
            m = re.match(r"OPT_COMPLETED_(\d{4}-\d{2}-\d{2})\.md", fname)
            if m:
                dk = m.group(1)
                if dk not in archive_map:
                    with open(os.path.join(ARCHIVE_DIR, fname)) as f:
                        cnt = len(re.findall(r"^## \[OPT-", f.read(), re.MULTILINE))
                    archive_map[dk] = cnt

    new_content = rebuild_completed_file(header, retained, archive_map)

    with open(COMPLETED_FILE, "w") as f:
        f.write(new_content)

    print(f"\n✓ 主文件已更新：{COMPLETED_FILE}")
    print(f"  保留 {len(retained)} 个当日条目")


if __name__ == "__main__":
    main()

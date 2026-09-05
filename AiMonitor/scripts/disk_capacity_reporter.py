#!/usr/bin/env python3
"""
disk_capacity_reporter.py — 导出仓库数据目录/承载文件系统的容量 Prometheus 指标

背景（OPT-20260823-066）：手动开具发票文件落 <repoRoot>/data/taskbill_invoice_files/，
属租户票面业务数据。node-exporter 看不到宿主 /tmp/ram-work tmpfs 挂载点（容器内仅 /rootfs），
因此仓库承载文件系统（RAM tmpfs）与发票目录增长无容量指标 → 磁盘/RAM 打满会导致上传静默失败。

输出格式：Prometheus textfile（供 node_exporter textfile collector 采集）

指标：
  repo_tmpfs_size_bytes{path="..."}   — 承载路径所在文件系统总容量
  repo_tmpfs_used_bytes{path="..."}   — 已用字节（total - free，含 root 保留块）
  repo_tmpfs_avail_bytes{path="..."}  — 非 root 可用字节（写入实际可用的上限）
  repo_data_dir_size_bytes{path="..."}— 指定数据目录下所有文件字节之和（含子目录）

使用方式（宿主 cron，每 5 分钟）：
  python3 /tmp/ram-work/AiMonitor/scripts/disk_capacity_reporter.py \
      --output /home/ljy/ramwork-recovery/node-exporter-textfiles/disk_capacity.prom

依赖：纯标准库（os/argparse），无第三方包。
"""

from __future__ import annotations

import argparse
import os
import sys
from pathlib import Path

# 默认承载路径与数据目录（可被 --path/--data-dir 覆盖）
DEFAULT_PATHS = ["/tmp/ram-work"]
DEFAULT_DATA_DIRS = ["/tmp/ram-work/data/taskbill_invoice_files"]
# 持久化盘（ramsync 目标）也纳入告警视野——node-exporter 看得到 /rootfs，
# 但这里给出语义更明确、直接对应业务数据承载盘的指标。
DEFAULT_PERSIST_PATHS = ["/home/ljy/gitClone/ramDisk/ram-mount"]


def fs_metrics(path: str) -> list[str]:
    """返回承载 path 的文件系统容量指标行。文件系统不可访问时返回空列表。"""
    try:
        v = os.statvfs(path)
    except OSError as exc:
        print(f"WARNING: statvfs failed for {path}: {exc}", file=sys.stderr)
        return []

    total = v.f_bsize * v.f_blocks
    used = v.f_bsize * (v.f_blocks - v.f_bfree)
    avail = v.f_bsize * v.f_bavail
    return [
        f'repo_tmpfs_size_bytes{{path="{path}"}} {total}',
        f'repo_tmpfs_used_bytes{{path="{path}"}} {used}',
        f'repo_tmpfs_avail_bytes{{path="{path}"}} {avail}',
    ]


def dir_size_bytes(path: str) -> int:
    """递归求目录内所有普通文件字节之和；目录不存在/不可读返回 0 并告警。"""
    root = Path(path)
    if not root.exists():
        print(f"WARNING: data dir not found: {path}", file=sys.stderr)
        return 0
    total = 0
    try:
        for entry in root.rglob("*"):
            if entry.is_file():
                try:
                    total += entry.stat().st_size
                except OSError:
                    pass
    except OSError as exc:
        print(f"WARNING: failed to walk {path}: {exc}", file=sys.stderr)
    return total


def collect_metrics(paths: list[str], data_dirs: list[str]) -> list[str]:
    lines: list[str] = []
    lines.append("# HELP repo_tmpfs_size_bytes Total bytes of the filesystem backing a repo path")
    lines.append("# TYPE repo_tmpfs_size_bytes gauge")
    lines.append("# HELP repo_tmpfs_used_bytes Used bytes of the filesystem backing a repo path")
    lines.append("# TYPE repo_tmpfs_used_bytes gauge")
    lines.append("# HELP repo_tmpfs_avail_bytes Available (non-root) bytes of the filesystem backing a repo path")
    lines.append("# TYPE repo_tmpfs_avail_bytes gauge")
    lines.append("# HELP repo_data_dir_size_bytes Total file bytes under a data directory")
    lines.append("# TYPE repo_data_dir_size_bytes gauge")

    for path in paths:
        lines.extend(fs_metrics(path))

    for data_dir in data_dirs:
        lines.append(f'repo_data_dir_size_bytes{{path="{data_dir}"}} {dir_size_bytes(data_dir)}')

    return lines


def main() -> None:
    parser = argparse.ArgumentParser(description="Export repo data-dir / backing-filesystem capacity metrics")
    parser.add_argument("--output", "-o", help="Write textfile to this path instead of stdout")
    parser.add_argument("--path", action="append", default=DEFAULT_PATHS, help="Filesystem backing path to measure")
    parser.add_argument("--data-dir", action="append", default=DEFAULT_DATA_DIRS, help="Data directory to sum")
    parser.add_argument("--persist-path", action="append", default=DEFAULT_PERSIST_PATHS,
                        help="Persistent disk path to also report (default ramsync target)")
    parser.add_argument("--self-test", action="store_true", help="Collect and print metrics to stdout, exit 0")
    args = parser.parse_args()

    # 保留用户显式 --path 语义；持久盘仅在未显式覆盖时附加。
    if "--path" in sys.argv:
        paths = args.path
    else:
        paths = args.path + args.persist_path

    lines = collect_metrics(paths, args.data_dir)

    if args.output:
        tmp = args.output + ".tmp"
        os.makedirs(os.path.dirname(args.output), exist_ok=True)
        with open(tmp, "w") as f:
            f.write("\n".join(lines) + "\n")
        os.rename(tmp, args.output)
        print(f"Written to {args.output} ({len(lines)} lines)", file=sys.stderr)
    else:
        sys.stdout.write("\n".join(lines) + "\n")

    if args.self_test:
        # 自测：指标行必须可被 node_exporter textfile 解析（# HELP/# TYPE 之后每行 `name{labels} value`）
        for line in lines:
            if line.startswith("#") or not line.strip():
                continue
            name, _, rest = line.partition("{")
            if not name.startswith("repo_"):
                print(f"SELF-TEST FAIL: unexpected metric name: {line}", file=sys.stderr)
                sys.exit(1)
            if "} " not in rest:
                print(f"SELF-TEST FAIL: malformed label/value: {line}", file=sys.stderr)
                sys.exit(1)
        print(f"SELF-TEST OK ({len(lines)} lines, {sum(1 for l in lines if l and not l.startswith('#'))} metrics)")


if __name__ == "__main__":
    main()

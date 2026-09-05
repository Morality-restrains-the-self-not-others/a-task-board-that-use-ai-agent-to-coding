#!/usr/bin/env bash
# 检测 MySQL 数据目录是否为「损坏空壳」：非空但缺 InnoDB 系统表空间。
# tmpfs（/tmp/ram-work）断电/异常中断后常见：mysqld 报
# 'Failed to find valid data directory' 并 exit 1，级联全站 3306 拒绝。
#
# 退出码：0 = 损坏（需挪走重建）；1 = 正常 / 完全空目录（首次初始化） / 不存在。
set -u

dir="${1:-}"
if [[ -z "$dir" || ! -d "$dir" ]]; then
  exit 1
fi
if [[ -z "$(ls -A "$dir" 2>/dev/null)" ]]; then
  # 完全空目录：视为未初始化，MySQL entrypoint 会正常建系统表。
  exit 1
fi
if [[ ! -f "$dir/ibdata1" ]] || [[ ! -d "$dir/mysql" ]]; then
  # 非空但缺 InnoDB 系统表空间（ibdata1）或 mysql 系统库 → 损坏空壳。
  exit 0
fi
exit 1

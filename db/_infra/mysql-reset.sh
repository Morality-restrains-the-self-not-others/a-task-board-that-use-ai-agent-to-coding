#!/usr/bin/env bash
# 清空全部 MySQL 数据库 — DROP 所有非系统库（含测试残留）+ CREATE registry 库 + 清除 binlog。
# 由 runAll /api/dev/clear-databases 调用，不替代 migrate/init。
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PYTHON="${PYTHON:-python3}"
"$PYTHON" "$ROOT/db/_infra/mysql_reset.py"

#!/usr/bin/env bash
# 损坏空壳 datadir：默认拒绝启动，避免 P4/切流后静默 mkdir 空库。
# 退出码：0 = 可启动（非损坏，或 MYSQL_ALLOW_EMPTY_REINIT=1 已挪走重建）；1 = 拒绝。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
dir="${1:-}"
DETECT="$SCRIPT_DIR/detect_corrupt_data.sh"
DUMP="${MYSQL_BACKUP_DIR:-/home/ljy/ramwork-recovery/mysql-dumps}/mysql-dump-latest.sql.gz"

if [[ -z "$dir" ]]; then
  echo "[docker-mysql] handle_corrupt_datadir: missing data dir argument" >&2
  exit 1
fi

if ! "$DETECT" "$dir"; then
  exit 0
fi

real="$(readlink -f "$dir" 2>/dev/null || printf '%s' "$dir")"
echo "[docker-mysql] REFUSE: datadir is a corrupt empty shell (missing ibdata1 or mysql/): $dir" >&2
if [[ -L "$dir" ]]; then
  echo "[docker-mysql] datadir is a symlink -> $real" >&2
fi
echo "[docker-mysql] restore from dump: $DUMP" >&2
echo "[docker-mysql] empty reinit only with MYSQL_ALLOW_EMPTY_REINIT=1" >&2

if [[ "${MYSQL_ALLOW_EMPTY_REINIT:-}" != "1" ]]; then
  exit 1
fi

ts="$(date +%Y%m%d_%H%M%S)"
parent="$(dirname "$dir")"
base="$(basename "$dir")"
# 符号链接：只拆链接，禁止 mv 到真实目标（可能仍是 ram-work 生产库）。
if [[ -L "$dir" ]]; then
  rm -f "$dir"
else
  mv "$dir" "$parent/${base}.corrupt.$ts"
fi
mkdir -p "$dir"
echo "[docker-mysql] MYSQL_ALLOW_EMPTY_REINIT=1: created empty $dir (backup suffix .corrupt.$ts if it was a real dir)" >&2
exit 0

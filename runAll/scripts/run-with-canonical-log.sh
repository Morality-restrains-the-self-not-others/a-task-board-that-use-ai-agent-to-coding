#!/usr/bin/env bash
# 将任意命令的 stdout/stderr tee 到 runAll 规范路径 logs/<service>.log。
# 禁止再使用 *.restart.log / *-run.log 旁路（Promtail 主路径不保证采集）。
#
# 用法:
#   bash runAll/scripts/run-with-canonical-log.sh <runAll-service-name> -- <command> [args...]
# 例:
#   bash runAll/scripts/run-with-canonical-log.sh go-relay -- ./bin/go_relayToTrae
#   bash runAll/scripts/run-with-canonical-log.sh task-cloud-service -- ./bin/taskCloudService
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

usage() {
  sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//'
}

SERVICE_NAME="${1:-}"
if [[ -z "$SERVICE_NAME" || "$SERVICE_NAME" == "-h" || "$SERVICE_NAME" == "--help" ]]; then
  usage
  exit 0
fi
shift
if [[ "${1:-}" != "--" ]]; then
  echo "错误: 服务名后须为 -- 再跟命令" >&2
  usage
  exit 1
fi
shift
if [[ "$#" -lt 1 ]]; then
  echo "错误: 缺少要执行的命令" >&2
  usage
  exit 1
fi

if [[ -n "${RUNALL_LOG_ROOT:-}" ]]; then
  LOG_ROOT="$RUNALL_LOG_ROOT"
else
  LOG_ROOT="$(python3 -c "
import yaml, os
p='${ROOT}/conf/runAll.yaml'
root='${ROOT}/logs'
if os.path.isfile(p):
    d=yaml.safe_load(open(p)) or {}
    raw=(d.get('logging') or {}).get('file_root') or ''
    if raw:
        root=raw if raw.startswith('/') else os.path.realpath(os.path.join('${ROOT}', raw))
print(root)
" 2>/dev/null || echo "${ROOT}/logs")"
fi

mkdir -p "$LOG_ROOT"
LOG_FILE="${LOG_ROOT}/${SERVICE_NAME}.log"

# 拒绝明显旁路文件名，强制规范路径
case "$LOG_FILE" in
  *.restart.log|*-run.log)
    echo "错误: 禁止写入旁路日志 $LOG_FILE" >&2
    exit 1
    ;;
esac

echo "==> canonical log: $LOG_FILE"
echo "==> exec: $*"
# 追加写入，与 runAll tee 共存；行首不加 runAll 前缀（Promtail pipeline 兼容裸 JSON）
exec "$@" >>"$LOG_FILE" 2>&1

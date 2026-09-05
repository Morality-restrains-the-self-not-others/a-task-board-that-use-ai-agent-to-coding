#!/usr/bin/env bash
# 截断 ram-work tmpfs 下各服务日志，避免 10G 挂载打满。
#
# 对 runAll tee 目录（默认 <repo>/logs）：优先调用 runAll truncate API
#   POST /api/logs/clear-all
# 禁止对 logs/*.log 使用 rm（会导致 runAll 持有 deleted inode，Promtail 漏采）。
#
# taskGateway/logs 由 APISIX 容器（uid 636）以 0644 创建：优先 docker exec
# 在容器内 in-place truncate，并 chmod a+rw 以便宿主后续可写。
#
# 用法:
#   bash runAll/scripts/truncate-ram-work-logs.sh          # 截断常见日志
#   bash runAll/scripts/truncate-ram-work-logs.sh --all    # 含 taskEvents/logs 全量
# 环境变量:
#   RUNALL_UI_PORT                    runAll Web UI 端口，默认 9999
#   TASKGATEWAY_APISIX_CONTAINER      默认 taskgateway-apisix-1
# 建议 cron（每小时）:
#   0 * * * * cd /path/to/ram-work && bash runAll/scripts/truncate-ram-work-logs.sh --all >> logs/log-truncate-cron.log 2>&1
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ALL=false
if [[ "${1:-}" == "--all" ]]; then
  ALL=true
fi

RUNALL_UI_PORT="${RUNALL_UI_PORT:-9999}"
RUNALL_LOG_ROOT="${RUNALL_LOG_ROOT:-${ROOT}/logs}"
# OPT-20260905-005: live tail retained in runall-console.log after shell fallback truncate.
RUNALL_CONSOLE_KEEP_TAIL_BYTES="${RUNALL_CONSOLE_KEEP_TAIL_BYTES:-262144}"
# OPT-20260901-005: clone-run 部署根容器前缀为 taskgateway-deploy，源码仓为 taskgateway。
if [[ -z "${TASKGATEWAY_APISIX_CONTAINER:-}" ]]; then
  if [[ -n "${COMPOSE_PROJECT_NAME:-}" ]]; then
    TASKGATEWAY_APISIX_CONTAINER="${COMPOSE_PROJECT_NAME}-apisix-1"
  elif [[ "${DEPLOY_MODE:-}" == "1" || -n "${DEPLOY_ROOT:-}" ]]; then
    TASKGATEWAY_APISIX_CONTAINER="taskgateway-deploy-apisix-1"
  else
    TASKGATEWAY_APISIX_CONTAINER="taskgateway-apisix-1"
  fi
fi
export TASKGATEWAY_APISIX_CONTAINER

# 短窗归档（OPT-20260823-037）：截断前保留各服务日志尾部，事故 stdout 不被抹掉。
# 归档根默认 logs/archive——runAll /api/logs/clear-all 与 shell 截断都只碰 *.log 顶层、不递归子目录。
TRUNCATE_ARCHIVE_ENABLED="${TRUNCATE_ARCHIVE_ENABLED:-1}"             # 0 关闭短窗归档
TRUNCATE_ARCHIVE_ROOT="${TRUNCATE_ARCHIVE_ROOT:-${ROOT}/logs/archive}" # 归档根
TRUNCATE_ARCHIVE_KEEP="${TRUNCATE_ARCHIVE_KEEP:-7}"                   # 保留最近 N 批
TRUNCATE_ARCHIVE_TAIL_BYTES="${TRUNCATE_ARCHIVE_TAIL_BYTES:-8388608}" # 每文件尾窗上限字节
# shellcheck disable=SC2034  # ARCHIVE_BATCH_TS 由 lib/log-archive.sh 的 archive_short_window 消费
ARCHIVE_BATCH_TS="$(date +%Y%m%dT%H%M%S)"
# shellcheck source=lib/log-archive.sh
source "${ROOT}/runAll/scripts/lib/log-archive.sh"

truncate_dir_in_place() {
  # In-place truncate (: >) keeps the inode when the path exists — never rm.
  local dir="$1"
  [[ -d "$dir" ]] || return 0
  local count=0
  local skipped=0
  local f
  for f in "$dir"/*.log; do
    [[ -f "$f" ]] || continue
    # 夜间巡检日志运行中截断会产生 NUL 填充（进程 fd 偏移 > 新 size）——跳过
    case "$(basename "$f")" in
      nightly-test-sweep-*.log) continue ;;
      runall-console.log)
        if truncate_keep_tail "$f" "${RUNALL_CONSOLE_KEEP_TAIL_BYTES}"; then
          count=$((count + 1))
        else
          echo "[truncate] skip (keep-tail failed): $f"
          skipped=$((skipped + 1))
        fi
        continue
        ;;
    esac
    if [[ ! -w "$f" ]]; then
      echo "[truncate] skip (not writable): $f"
      skipped=$((skipped + 1))
      continue
    fi
    if ! : >"$f" 2>/dev/null; then
      echo "[truncate] skip (truncate failed): $f"
      skipped=$((skipped + 1))
      continue
    fi
    count=$((count + 1))
  done
  echo "[truncate] $dir: $count file(s) (in-place, no rm); skipped=$skipped"
}

# Keep last N bytes of a live log (same path). Best-effort for open FDs.
truncate_keep_tail() {
  local f="$1"
  local keep="${2:-262144}"
  [[ -f "$f" ]] || return 1
  [[ -w "$f" ]] || return 1
  local tmp
  tmp="$(mktemp)"
  if ! tail -c "$keep" "$f" >"$tmp" 2>/dev/null; then
    rm -f -- "$tmp"
    return 1
  fi
  if ! cat "$tmp" >"$f" 2>/dev/null; then
    rm -f -- "$tmp"
    return 1
  fi
  rm -f -- "$tmp"
  return 0
}

clear_runall_tee_logs() {
  local url="http://127.0.0.1:${RUNALL_UI_PORT}/api/logs/clear-all"
  local body code
  body="$(mktemp)"
  code="$(curl -sS -m 15 -o "$body" -w '%{http_code}' -X POST "$url" \
    -H 'Content-Type: application/json' 2>/dev/null || echo 000)"
  if [[ "$code" == "200" ]]; then
    echo "[truncate] runAll $url ok (HTTP $code): $(tr -d '\n' <"$body")"
    rm -f "$body"
    return 0
  fi
  echo "[truncate] runAll truncate API unavailable (HTTP ${code} on :${RUNALL_UI_PORT})"
  rm -f "$body"
  return 1
}

# APISIX writes logs as uid 636 mode 0644; truncate as that user inside the container.
truncate_taskgateway_logs() {
  local dir="$ROOT/taskGateway/logs"
  local container="$TASKGATEWAY_APISIX_CONTAINER"
  local count

  if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -qx "$container"; then
    echo "[truncate] $container not running — fallback host in-place for $dir"
    truncate_dir_in_place "$dir"
    return 0
  fi

  count="$(docker exec "$container" sh -c '
    count=0
    for f in /usr/local/apisix/logs/*.log; do
      [ -f "$f" ] || continue
      if : >"$f" 2>/dev/null; then
        count=$((count + 1))
      fi
    done
    # Host cron/user (e.g. ljy) needs write for future in-place truncate without docker.
    chmod a+rw /usr/local/apisix/logs/*.log 2>/dev/null || true
    echo "$count"
  ')"
  count="$(echo "$count" | tr -d '[:space:]')"
  if [[ -z "$count" || ! "$count" =~ ^[0-9]+$ ]]; then
    echo "[truncate] WARNING: docker exec truncate failed for $container — fallback host in-place"
    truncate_dir_in_place "$dir"
    return 0
  fi
  echo "[truncate] $dir: $count file(s) via docker exec $container (in-place, no rm; chmod a+rw)"
}

echo "[truncate] root=$ROOT started at $(date -Iseconds)"

# 0) 短窗归档（best-effort）— 截断前保留尾部，即使 Loki/promtail 未及时 ingest 也不丢事故 stdout
archive_short_window "$RUNALL_LOG_ROOT"

# 1) runAll tee logs — API first (keeps open FDs on the same inode)
if clear_runall_tee_logs; then
  echo "[truncate] skipped shell truncate of ${RUNALL_LOG_ROOT} (handled by FileServiceLogSink.TruncateAll)"
else
  echo "[truncate] WARNING: falling back to in-place truncate of ${RUNALL_LOG_ROOT}; prefer starting runAll UI on :${RUNALL_UI_PORT}"
  truncate_dir_in_place "$RUNALL_LOG_ROOT"
fi

# 2) taskGateway APISIX logs (container-owned) — docker exec truncate
archive_short_window "$ROOT/taskGateway/logs"
truncate_taskgateway_logs

# 3) Other non-runAll log dirs — host in-place
archive_short_window "$ROOT/trae-agent/onlineProject_state/reqLogs"
truncate_dir_in_place "$ROOT/trae-agent/onlineProject_state/reqLogs"

if $ALL; then
  archive_short_window "$ROOT/taskEvents/logs"
  truncate_dir_in_place "$ROOT/taskEvents/logs"
fi

# 4) 清理超出保留数的旧归档批
prune_archive_batches

echo "[truncate] done; df:"
df -h "$ROOT" | tail -1

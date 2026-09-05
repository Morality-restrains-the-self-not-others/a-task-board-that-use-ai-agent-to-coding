#!/usr/bin/env bash
# 短窗归档：整点 truncate 前保留各服务日志尾部，事故 stdout 不被抹掉。
#
# 背景（OPT-20260823-037）：Loki 无 ingest 时整点 `truncate-ram-work-logs.sh --all`
# 会清空 `logs/*.log`，排障只剩网关审计。promtail 恢复后 JSON/裸行都会进 Loki，
# 但 promtail 进程不在、或尾部 last-write 尚未读走时，truncate 仍会丢数据。
# 本库在截断前把每个非空日志的尾部窗口（默认 8MiB）落一份 gzip 到归档根，
# 并按批保留最近 N 份；任何失败只告警不阻断截断（best-effort）。
#
# 由 truncate-ram-work-logs.sh 与 log-archive_selftest.sh 共同 source。
# 全局变量（均可在调用方覆盖）:
#   TRUNCATE_ARCHIVE_ENABLED        0 关闭归档（默认 1）
#   TRUNCATE_ARCHIVE_ROOT           归档根（默认 <ROOT>/logs/archive）
#   TRUNCATE_ARCHIVE_KEEP           保留最近 N 批（默认 7）
#   TRUNCATE_ARCHIVE_TAIL_BYTES     每文件尾窗上限字节（默认 8388608）
#   ARCHIVE_BATCH_TS                归档批时间戳（默认当前时刻，可注入便于测试）

# 归档单个目录下 *.log 的尾部。调用方负责在截断前调用。
archive_short_window() {
  [[ "${TRUNCATE_ARCHIVE_ENABLED:-1}" == "1" ]] || return 0
  local dir="$1"
  [[ -d "$dir" ]] || return 0
  local batch_ts="${ARCHIVE_BATCH_TS:-$(date +%Y%m%dT%H%M%S)}"
  local batch_dir="${TRUNCATE_ARCHIVE_ROOT}/${batch_ts}"
  local f base dest count=0
  for f in "$dir"/*.log; do
    [[ -f "$f" ]] || continue
    # 夜间巡检日志运行中截断会产生 NUL 填充——与 truncate 同一跳过规则
    case "$(basename "$f")" in
      nightly-test-sweep-*.log) continue ;;
    esac
    [[ -s "$f" ]] || continue                 # 空文件不归档
    [[ -r "$f" ]] || { echo "[truncate] archive skip (not readable): $f"; continue; }
    mkdir -p "$batch_dir" 2>/dev/null || { echo "[truncate] archive skip (mkdir fail): $batch_dir"; continue; }
    base="$(basename "$f")"
    dest="$batch_dir/${base}.gz"
    if ! tail -c "${TRUNCATE_ARCHIVE_TAIL_BYTES:-8388608}" "$f" 2>/dev/null | gzip -c >"$dest" 2>/dev/null; then
      echo "[truncate] archive warn (tail/gzip failed): $f"
      rm -f -- "$dest" 2>/dev/null || true
      continue
    fi
    count=$((count + 1))
  done
  if (( count > 0 )); then
    echo "[truncate] archive: $dir -> $batch_dir ($count file(s) tail window)"
  fi
}

# 清理超出保留数的旧归档批（批名 ISO 时间戳，字典序即时间序，删最旧）。
prune_archive_batches() {
  [[ "${TRUNCATE_ARCHIVE_ENABLED:-1}" == "1" ]] || return 0
  local arch_root="${TRUNCATE_ARCHIVE_ROOT}"
  local keep="${TRUNCATE_ARCHIVE_KEEP:-7}"
  [[ -d "$arch_root" ]] || return 0
  local d n=0
  while IFS= read -r d; do
    n=$((n + 1))
    if (( n > keep )); then
      rm -rf -- "$d" 2>/dev/null || true
    fi
  done < <(find "$arch_root" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | sort -r)
}

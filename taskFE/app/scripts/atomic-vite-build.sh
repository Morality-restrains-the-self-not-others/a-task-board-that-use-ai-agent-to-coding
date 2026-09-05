#!/usr/bin/env bash
# taskFE 原子 Vite 生产构建：先写临时目录 public/.next，校验成功后再 mv 进 public/releases/<id>
# 并 ln -sfn 切换 public/html。失败时不切 symlink，nginx 继续旧 release。
# 构建不 recreate SPA 容器；切 public/html symlink 即可。
# OPT-20260812-026：构建前内存预检 + OOM 重试。
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
app_dir="$(cd "$script_dir/.." && pwd)"
cd "$app_dir"

STAGING="public/.next"
RELEASES="public/releases"
LIVE_LINK="public/html"

# ── 内存预检（OPT-20260812-026）─────────────────────────────
# MEM_MIN_MB：低于此值告警并回收 page cache；MEM_HARD_MB：低于此值自动停非关键容器 kafka-ui 腾内存。
# 可用内存来源可用 TASKFE_BUILD_MEM_AVAIL_MB 覆盖（便于单测），不设则读 /proc/meminfo。
MEM_MIN_MB="${TASKFE_BUILD_MIN_MEM_MB:-6144}"
MEM_HARD_MB="${TASKFE_BUILD_HARD_MEM_MB:-3584}"
AUTO_FREE="${TASKFE_BUILD_AUTO_FREE:-1}"   # 1=允许自动停 kafka-ui（纯 UI 面板，安全）；GitLab 保留仅提示
STOPPED_CONTAINERS=""

mem_available_mb() {
  if [ -n "${TASKFE_BUILD_MEM_AVAIL_MB:-}" ]; then
    echo "$TASKFE_BUILD_MEM_AVAIL_MB"
    return 0
  fi
  awk '/MemAvailable/ {printf "%d", $2/1024}' /proc/meminfo 2>/dev/null || echo 0
}

free_page_cache() {
  # 尽力回收 page cache（drop_caches 仅 root 生效，失败静默）
  sync 2>/dev/null || true
  if [ "$(id -u)" = "0" ]; then
    echo 1 > /proc/sys/vm/drop_caches 2>/dev/null || true
    echo 3 > /proc/sys/vm/drop_caches 2>/dev/null || true
  fi
}

stop_noncritical_containers() {
  if [ "$AUTO_FREE" != "1" ] || ! command -v docker >/dev/null 2>&1; then
    return 0
  fi
  # 仅自动停 kafka-ui（无状态 UI 面板，重启无副作用）；GitLab 影响 git 鉴权，保留只提示。
  for c in kafka-kafka-ui-1; do
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx "$c"; then
      echo "内存不足：自动停止非关键容器 $c 以释放构建内存…" >&2
      if docker stop "$c" >/dev/null 2>&1; then
        STOPPED_CONTAINERS="$STOPPED_CONTAINERS $c"
      fi
    fi
  done
}

restart_stopped_containers() {
  if [ -n "$STOPPED_CONTAINERS" ]; then
    for c in $STOPPED_CONTAINERS; do
      echo "构建完成：恢复容器 $c …" >&2
      docker start "$c" >/dev/null 2>&1 || echo "警告：$c 重启失败，请手动 docker start $c" >&2
    done
    STOPPED_CONTAINERS=""
  fi
}
trap restart_stopped_containers EXIT

# 首次预检
AVAIL_MB="$(mem_available_mb)"
if [ "$AVAIL_MB" -lt "$MEM_HARD_MB" ]; then
  echo "警告：构建前可用内存 ${AVAIL_MB}MiB（<${MEM_HARD_MB}MiB），释放内存后继续…" >&2
  free_page_cache
  stop_noncritical_containers
elif [ "$AVAIL_MB" -lt "$MEM_MIN_MB" ]; then
  echo "警告：构建前可用内存 ${AVAIL_MB}MiB（<${MEM_MIN_MB}MiB），构建可能 OOM；已尽力回收 page cache" >&2
  free_page_cache
fi

if [ ! -x node_modules/.bin/vite ]; then
  # OPT-20260812-015: tmpfs/清依赖后「全部重新编译」不应因缺 node_modules 单点失败
  echo "node_modules/.bin/vite 缺失：自动执行 npm ci 重建依赖…" >&2
  NPM_CI_LOG="$(mktemp /tmp/taskfe-npm-ci.XXXXXX.log)"
  if ! npm ci > "$NPM_CI_LOG" 2>&1; then
    echo "npm ci 失败，完整日志：$NPM_CI_LOG" >&2
    tail -40 "$NPM_CI_LOG" >&2
    exit 127
  fi
  if [ ! -x node_modules/.bin/vite ]; then
    echo "npm ci 后 vite 仍不可执行（依赖未含 vite？），日志：$NPM_CI_LOG" >&2
    exit 127
  fi
  echo "npm ci 完成，继续构建" >&2
fi

rm -rf "$STAGING"
mkdir -p public
# outDir 相对 vite root(./src)；../public/.next → app/public/.next
BUILD_LOG="$(mktemp /tmp/taskfe-vite-build.XXXXXX.log)"
build_vite() {
  npx vite build --outDir "../${STAGING}"
}

if build_vite >"$BUILD_LOG" 2>&1; then
  rm -f "$BUILD_LOG"
else
  # 注意：不能写成 `if ! build_vite`，否则 $? 是取反后的 0，拿不到真实退出码
  rc=$?
  if [ "$rc" -eq 137 ] || grep -qE 'heap out of memory|JavaScript heap|ENOMEM|Killed' "$BUILD_LOG"; then
    echo "vite build 疑似 OOM（rc=$rc）：释放内存后重试一次…" >&2
    free_page_cache
    stop_noncritical_containers
    sleep 3
    if build_vite >"$BUILD_LOG" 2>&1; then
      rm -f "$BUILD_LOG"
    else
      rc2=$?
      cat "$BUILD_LOG" >&2
      rm -rf "$STAGING"
      rm -f "$BUILD_LOG"
      exit "$rc2"
    fi
  else
    cat "$BUILD_LOG" >&2
    rm -rf "$STAGING"
    rm -f "$BUILD_LOG"
    exit "$rc"
  fi
fi

if [ ! -f "${STAGING}/index.html" ]; then
  echo "build 未产出 ${STAGING}/index.html，保留现有 html symlink / 当前 release 不变" >&2
  rm -rf "$STAGING"
  exit 1
fi

mkdir -p "$RELEASES"
release_id="$(date +%Y%m%d%H%M%S)-$$"
dest="${RELEASES}/${release_id}"
mv "$STAGING" "$dest"

ln -sfn "releases/${release_id}" "${LIVE_LINK}.tmp"
mv -T "${LIVE_LINK}.tmp" "$LIVE_LINK"

keep="${TASKFE_RELEASE_KEEP:-2}"
if [ "$keep" -gt 0 ] 2>/dev/null; then
  live_rel=""
  if [ -L "$LIVE_LINK" ]; then
    live_rel="$(readlink "$LIVE_LINK")"
  fi
  n=0
  shopt -s nullglob
  for d in $(ls -1dt "$RELEASES"/*/ 2>/dev/null); do
    n=$((n + 1))
    base="${d%/}"
    rel="releases/$(basename "$base")"
    if [ "$rel" = "$live_rel" ]; then
      continue
    fi
    if [ "$n" -gt "$keep" ]; then
      rm -rf "$base"
    fi
  done
  shopt -u nullglob
fi

echo "atomic-vite-build: ${LIVE_LINK} -> releases/${release_id} ($(wc -c < "${LIVE_LINK}/index.html" | tr -d ' ') bytes index.html)"

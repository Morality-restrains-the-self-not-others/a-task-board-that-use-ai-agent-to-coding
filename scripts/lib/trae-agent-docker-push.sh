#!/usr/bin/env bash
# ============================================================================
# trae-agent-docker-push.sh — 约束 46：在 onlineServiceJS 执行 DOCKER_PUSH=1 ./buildDocker.sh
# ============================================================================
# 用法:
#   bash scripts/lib/trae-agent-docker-push.sh              # 立即前台推送
#   bash scripts/lib/trae-agent-docker-push.sh --if-pending # 仅当 pending 存在时推送
#   bash scripts/lib/trae-agent-docker-push.sh --background # nohup 后台（SessionEnd 兜底）
#   组合: --if-pending --background
#
# 环境变量:
#   TRAE_AGENT_SKIP_DOCKER_PUSH=1     跳过
#   TRAE_AGENT_DOCKER_PUSH_DRY_RUN=1  不跑 docker，仅更新水位线（自测）
#   TRAE_AGENT_DOCKER_PUSH_STATE_DIR  覆盖 .runall 状态目录
# ============================================================================
set -u

IF_PENDING=0
BACKGROUND=0
while [ $# -gt 0 ]; do
  case "$1" in
    --if-pending) IF_PENDING=1; shift ;;
    --background) BACKGROUND=1; shift ;;
    -h|--help)
      sed -n '2,20p' "$0"
      exit 0
      ;;
    *)
      echo "trae-agent-docker-push: unknown arg: $1" >&2
      exit 2
      ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel 2>/dev/null)" || exit 0
cd "$REPO_ROOT" || exit 0

SUB="trae-agent"
ONLINE_DIR="$REPO_ROOT/$SUB/onlineServiceJS"
BUILD_SH="$ONLINE_DIR/buildDocker.sh"
RUNALL_DIR="${TRAE_AGENT_DOCKER_PUSH_STATE_DIR:-$REPO_ROOT/.runall}"
PENDING_FILE="$RUNALL_DIR/trae_agent_docker_push_pending"
SHA_FILE="$RUNALL_DIR/trae_agent_docker_push_sha"
LOG_FILE="$RUNALL_DIR/trae_agent_docker_push.log"
LOCK_FILE="$RUNALL_DIR/trae_agent_docker_push.lock"

if [ "${TRAE_AGENT_SKIP_DOCKER_PUSH:-0}" = "1" ] || [ "${TRAE_AGENT_SKIP_DOCKER_PUSH:-}" = "true" ]; then
  echo "trae-agent-docker-push: skipped (TRAE_AGENT_SKIP_DOCKER_PUSH=1)" >&2
  exit 0
fi

if [ "$IF_PENDING" -eq 1 ] && [ ! -f "$PENDING_FILE" ]; then
  exit 0
fi

if [ ! -x "$BUILD_SH" ] && [ ! -f "$BUILD_SH" ]; then
  echo "trae-agent-docker-push: missing $BUILD_SH" >&2
  exit 0
fi

mkdir -p "$RUNALL_DIR" || true

run_push() {
  local head_sha=""
  if [ -d "$REPO_ROOT/$SUB/.git" ] || [ -f "$REPO_ROOT/$SUB/.git" ]; then
    head_sha="$(git -C "$REPO_ROOT/$SUB" rev-parse HEAD 2>/dev/null || true)"
  fi

  echo "trae-agent-docker-push: cwd=$ONLINE_DIR cmd='DOCKER_PUSH=1 ./buildDocker.sh'" >&2

  if [ "${TRAE_AGENT_DOCKER_PUSH_DRY_RUN:-0}" = "1" ]; then
    echo "trae-agent-docker-push: DRY_RUN success" >&2
  else
    (
      cd "$ONLINE_DIR" || exit 1
      # shellcheck disable=SC2091
      DOCKER_PUSH=1 ./buildDocker.sh
    )
    local rc=$?
    if [ "$rc" -ne 0 ]; then
      echo "trae-agent-docker-push: FAILED exit=$rc（见日志/终端；pending 保留以便重试）" >&2
      return "$rc"
    fi
  fi

  if [ -n "$head_sha" ]; then
    printf '%s\n' "$head_sha" > "$SHA_FILE" || true
  fi
  rm -f "$PENDING_FILE" || true
  echo "trae-agent-docker-push: OK watermark=${head_sha:-none}" >&2
  return 0
}

if [ "$BACKGROUND" -eq 1 ]; then
  # 简单锁：已有推送在跑则跳过
  if [ -f "$LOCK_FILE" ]; then
    old_pid="$(tr -d '[:space:]' < "$LOCK_FILE" 2>/dev/null || true)"
    if [ -n "$old_pid" ] && kill -0 "$old_pid" 2>/dev/null; then
      echo "trae-agent-docker-push: already running pid=$old_pid，跳过后台启动" >&2
      exit 0
    fi
  fi
  (
    echo $$ > "$LOCK_FILE"
    trap 'rm -f "$LOCK_FILE"' EXIT
    {
      echo "==== $(date -Iseconds 2>/dev/null || date) background push start ===="
      run_push
      ec=$?
      echo "==== $(date -Iseconds 2>/dev/null || date) background push end exit=$ec ===="
      exit "$ec"
    } >>"$LOG_FILE" 2>&1
  ) &
  echo "trae-agent-docker-push: background pid=$! log=$LOG_FILE" >&2
  exit 0
fi

run_push
exit $?

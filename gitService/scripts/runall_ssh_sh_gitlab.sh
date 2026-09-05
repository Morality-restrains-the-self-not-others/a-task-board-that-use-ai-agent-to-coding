#!/usr/bin/env bash
# 从 INFRA 机 9999 远程启停 Host sh 上的上海 GitLab（compose 已部署在 REMOTE_DIR）。
# 禁止在 INFRA 本机执行 deploy_tencent_sh_1.sh：本机 :8014 是 task-container-gateway。
set -euo pipefail
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

ACTION="${1:-}"
SSH_HOST="${GITSERVICE_SH_SSH_HOST:-sh}"
REMOTE_DIR="${GITSERVICE_SH_COMPOSE_DIR:-/opt/daydaymoney/gitservice-tencent-sh-1}"
SSH_BIN="${GITSERVICE_SH_SSH_BIN:-ssh}"
CONNECT_TIMEOUT="${GITSERVICE_SH_SSH_CONNECT_TIMEOUT:-15}"

usage() {
  echo "usage: $0 start|stop" >&2
  echo "  GITSERVICE_SH_SSH_HOST (default sh)" >&2
  echo "  GITSERVICE_SH_COMPOSE_DIR (default /opt/daydaymoney/gitservice-tencent-sh-1)" >&2
  exit 2
}

case "$ACTION" in
  start) REMOTE_CMD="docker compose up -d" ;;
  stop) REMOTE_CMD="docker compose stop" ;;
  *) usage ;;
esac

if [[ -z "$SSH_HOST" || -z "$REMOTE_DIR" ]]; then
  echo "error: SSH_HOST and REMOTE_DIR must be non-empty" >&2
  exit 2
fi

# Remote path is operator-controlled; reject metacharacters so the ssh payload stays one cd+compose.
case "$REMOTE_DIR" in
  *[!A-Za-z0-9._/-]* | "")
    echo "error: GITSERVICE_SH_COMPOSE_DIR has unsafe characters: $REMOTE_DIR" >&2
    exit 2
    ;;
esac
case "$SSH_HOST" in
  *[!A-Za-z0-9._-]* | "")
    echo "error: GITSERVICE_SH_SSH_HOST has unsafe characters: $SSH_HOST" >&2
    exit 2
    ;;
esac

REMOTE_SH="cd ${REMOTE_DIR} && ${REMOTE_CMD}"
echo "runall_ssh_sh_gitlab: action=${ACTION} host=${SSH_HOST} dir=${REMOTE_DIR}" >&2

if [[ "${GITSERVICE_SH_SSH_DRY_RUN:-}" == "1" ]]; then
  printf '%s -o BatchMode=yes -o ConnectTimeout=%s %s %q\n' \
    "$SSH_BIN" "$CONNECT_TIMEOUT" "$SSH_HOST" "$REMOTE_SH"
  exit 0
fi

exec "$SSH_BIN" -o BatchMode=yes -o ConnectTimeout="$CONNECT_TIMEOUT" "$SSH_HOST" "$REMOTE_SH"

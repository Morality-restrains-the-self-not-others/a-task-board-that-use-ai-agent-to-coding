#!/usr/bin/env bash
# 从 INFRA 机 9999 远程启停 Host sh 上的 wechat-mp-egress（systemd 单元，ADR-0059）。
# 部署/升级走 scripts/deploy_wechat_mp_egress_sh.sh（build + scp + 写 unit + restart）。
# 健康探活由 conf/runAll.yaml health_check.url 负责（http://1.117.67.121:8030/healthz）。
set -euo pipefail
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

ACTION="${1:-}"
SSH_HOST="${WECHAT_MP_EGRESS_SSH_HOST:-sh}"
SSH_BIN="${WECHAT_MP_EGRESS_SSH_BIN:-ssh}"
CONNECT_TIMEOUT="${WECHAT_MP_EGRESS_SSH_CONNECT_TIMEOUT:-15}"
UNIT="wechat-mp-egress"

usage() {
  echo "usage: $0 start|stop" >&2
  echo "  WECHAT_MP_EGRESS_SSH_HOST (default sh)" >&2
  echo "  WECHAT_MP_EGRESS_SSH_BIN   (default ssh)" >&2
  exit 2
}

case "$ACTION" in
  start) REMOTE_CMD="systemctl start ${UNIT}" ;;
  stop) REMOTE_CMD="systemctl stop ${UNIT}" ;;
  *) usage ;;
esac

case "$SSH_HOST" in
  *[!A-Za-z0-9._-]* | "")
    echo "error: WECHAT_MP_EGRESS_SSH_HOST has unsafe characters: $SSH_HOST" >&2
    exit 2
    ;;
esac

echo "runall_ssh_sh_wechat_mp_egress: action=${ACTION} host=${SSH_HOST} unit=${UNIT}" >&2

if [[ "${WECHAT_MP_EGRESS_SSH_DRY_RUN:-}" == "1" ]]; then
  printf '%s -o BatchMode=yes -o ConnectTimeout=%s %s %q\n' \
    "$SSH_BIN" "$CONNECT_TIMEOUT" "$SSH_HOST" "$REMOTE_CMD"
  exit 0
fi

exec "$SSH_BIN" -o BatchMode=yes -o ConnectTimeout="$CONNECT_TIMEOUT" "$SSH_HOST" "$REMOTE_CMD"

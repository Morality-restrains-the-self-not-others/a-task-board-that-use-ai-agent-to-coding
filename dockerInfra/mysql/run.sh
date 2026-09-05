#!/usr/bin/env bash
set -euo pipefail
# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=../scripts/resolve_infra_conf.sh
source "$SCRIPT_DIR/../scripts/resolve_infra_conf.sh"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.yml"
PROJECT_NAME="docker-mysql"

# ADR-0052: CONF_ROOT may be the conf dir or deploy root; data dir stays at SCRIPT_DIR.
MYSQL_CONF="$(resolve_infra_conf mysql)"
echo "[docker-mysql] conf=${MYSQL_CONF}"

load_mysql_conf() {
  # SSOT: conf/infra/mysql/config.yaml → MYSQL_* for compose
  if [[ ! -f "$MYSQL_CONF" ]]; then
    echo "[docker-mysql] missing $MYSQL_CONF" >&2
    exit 1
  fi
  # shellcheck disable=SC1090
  eval "$(python3 "$SCRIPT_DIR/load_mysql_conf.py" "$MYSQL_CONF")"
}

usage() {
  echo "Usage: $0 {start|stop|restart|status|logs}" >&2
  exit 1
}

start() {
  echo "[docker-mysql] Starting MySQL..."
  load_mysql_conf
  cd "$SCRIPT_DIR"

  # OPT-20260830-025：空壳默认拒绝启动，禁止 mv 符号链接后再 mkdir 空库。
  if ! "$SCRIPT_DIR/handle_corrupt_datadir.sh" "$SCRIPT_DIR/data"; then
    exit 1
  fi

  docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" up -d --wait
  echo "[docker-mysql] MySQL is ready (port 3306)"
}

stop() {
  echo "[docker-mysql] Stopping MySQL..."
  cd "$SCRIPT_DIR"
  docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down --volumes=false
  echo "[docker-mysql] Stopped"
}

restart() {
  stop
  start
}

status_cmd() {
  cd "$SCRIPT_DIR"
  docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps
}

logs_cmd() {
  cd "$SCRIPT_DIR"
  docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" logs -f --tail=100
}

case "${1:-}" in
  start)   start ;;
  stop)    stop ;;
  restart) restart ;;
  status)  status_cmd ;;
  logs)    logs_cmd ;;
  managed)
    # runAll compatibility: 'managed' = idempotent start
    load_mysql_conf
    if docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps --status running 2>/dev/null | grep -q 'Up'; then
      echo "[docker-mysql] Already running (durability flags from conf apply on next recreate)"
    else
      start
    fi
    ;;
  *)       usage ;;
esac

#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=docker-env.sh
source "$ROOT/docker-env.sh"

usage() {
  cat <<EOF
用法: $(basename "$0") {start|stop|status} [--core]

远程 Docker 模式下，将远程主机 127.0.0.1 上的容器端口转发到本机。

  start [--core]   后台启动 SSH 端口转发（--core 仅 Redis/Kafka，不含 Grafana/OTEL）
  stop             停止转发
  status           查看隧道状态
EOF
}

TUNNEL_PORTS=()

resolve_tunnel_ports() {
  local mode="${1:-full}"
  TUNNEL_PORTS=()
  if [[ "$mode" == "core" ]]; then
    TUNNEL_PORTS=("${DOCKER_REMOTE_TUNNEL_PORTS_CORE[@]}")
  else
    TUNNEL_PORTS=("${DOCKER_REMOTE_TUNNEL_PORTS[@]}")
  fi
}

tunnel_running() {
  [[ -f "$DOCKER_TUNNEL_PID_FILE" ]] && kill -0 "$(cat "$DOCKER_TUNNEL_PID_FILE")" 2>/dev/null
}

local_port_in_use() {
  local port="$1"
  if command -v lsof &>/dev/null; then
    lsof -iTCP:"$port" -sTCP:LISTEN -P -n &>/dev/null
    return $?
  fi
  nc -z 127.0.0.1 "$port" &>/dev/null
}

check_local_port_conflicts() {
  local port conflicts=0
  for port in "${TUNNEL_PORTS[@]}"; do
    if local_port_in_use "$port"; then
      echo "  127.0.0.1:${port} 已被占用" >&2
      conflicts=1
    fi
  done
  if (( conflicts )); then
    echo "" >&2
    echo "提示: 端口冲突时无法启动隧道。可尝试:" >&2
    echo "  - 停止本地 Docker Desktop / 占用端口的进程" >&2
    echo "  - runAll/scripts/docker-tunnel-remote.sh start --core   # 仅转发 Redis/Kafka" >&2
    return 1
  fi
  return 0
}

cmd_start() {
  if tunnel_running; then
    echo "SSH 隧道已在运行 (pid $(cat "$DOCKER_TUNNEL_PID_FILE"))"
    return 0
  fi

  if ! check_local_port_conflicts; then
    exit 1
  fi

  local -a forwards=()
  local port
  for port in "${TUNNEL_PORTS[@]}"; do
    forwards+=(-L "${port}:127.0.0.1:${port}")
  done

  echo "启动 SSH 隧道 → ${DOCKER_SSH_USER}@${DOCKER_SSH_ADDR} (${DOCKER_SSH_HOST})"
  echo "转发端口: ${TUNNEL_PORTS[*]}"

  ssh -N "${forwards[@]}" -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 "$DOCKER_SSH_HOST" &
  local pid=$!
  echo "$pid" >"$DOCKER_TUNNEL_PID_FILE"
  printf '%s\n' "${TUNNEL_PORTS[@]}" >"${DOCKER_TUNNEL_PID_FILE}.ports"
  sleep 0.5
  if kill -0 "$pid" 2>/dev/null; then
    echo "隧道已启动 (pid $pid)"
  else
    rm -f "$DOCKER_TUNNEL_PID_FILE" "${DOCKER_TUNNEL_PID_FILE}.ports"
    echo "错误: SSH 隧道启动失败" >&2
    exit 1
  fi
}

cmd_stop() {
  if ! tunnel_running; then
    rm -f "$DOCKER_TUNNEL_PID_FILE" "${DOCKER_TUNNEL_PID_FILE}.ports"
    echo "SSH 隧道未运行"
    return 0
  fi
  local pid
  pid="$(cat "$DOCKER_TUNNEL_PID_FILE")"
  kill "$pid" 2>/dev/null || true
  rm -f "$DOCKER_TUNNEL_PID_FILE" "${DOCKER_TUNNEL_PID_FILE}.ports"
  echo "已停止 SSH 隧道 (pid $pid)"
}

cmd_status() {
  if tunnel_running; then
    echo "SSH 隧道: 运行中 (pid $(cat "$DOCKER_TUNNEL_PID_FILE"))"
    if [[ -f "${DOCKER_TUNNEL_PID_FILE}.ports" ]]; then
      echo "转发端口: $(tr '\n' ' ' <"${DOCKER_TUNNEL_PID_FILE}.ports")"
    else
      echo "转发端口: ${DOCKER_REMOTE_TUNNEL_PORTS[*]}"
    fi
  else
    echo "SSH 隧道: 未运行"
    rm -f "$DOCKER_TUNNEL_PID_FILE" "${DOCKER_TUNNEL_PID_FILE}.ports"
  fi
}

main() {
  local action="${1:-status}"
  shift || true
  local tunnel_mode="full"
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --core) tunnel_mode="core"; shift ;;
      *) echo "未知参数: $1" >&2; usage >&2; exit 1 ;;
    esac
  done
  resolve_tunnel_ports "$tunnel_mode"
  case "$action" in
    start) cmd_start ;;
    stop) cmd_stop ;;
    status) cmd_status ;;
    -h|--help|help) usage ;;
    *)
      usage >&2
      exit 1
      ;;
  esac
}

main "$@"
